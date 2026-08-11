// Package service 提供核心路由服务引擎。
//
// 职责链路:定时对启用的路由规则做 DNS 解析,将解析出的 IP 通过路由管理器
// 以临时主机路由(/32、/128)绑定到选中的网卡;解析结果变化时做 diff 同步
// (新增建路由、消失删路由);服务停止时清理全部已添加路由恢复系统默认。
//
// 服务运行期间将"已应用路由清单"持久化到数据目录,供下次启动时清理
// 上次异常退出(进程被杀/崩溃)遗留的路由,保证系统路由表始终可恢复。
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"NetRoute-Manager/internal/dns"
	"NetRoute-Manager/internal/models"
	"NetRoute-Manager/internal/route"
)

// appliedRoutesFileName 已应用路由清单文件名(数据目录下)。
const appliedRoutesFileName = "applied_routes.json"

// LogFunc 引擎日志回调,由外部(App 层)接线到 UI/标准输出。
type LogFunc func(level models.LogLevel, text string)

// ipResolver 解析接口抽象,便于测试注入 fake(实现见 internal/dns.Resolver)。
type ipResolver interface {
	Resolve(ctx context.Context, domain string, servers []string, enableIPv6 bool) ([]netip.Addr, error)
}

// ruleState 单条路由规则的运行期状态。
type ruleState struct {
	models.RouteRule
	// ips 该规则当前解析出的全部 IP(可能多个,路由表按 IP 粒度全部生效);
	// RouteRule.ResolvedIP 仅展示首个 IP。
	ips []netip.Addr
}

// appliedRouteEntry 已应用路由清单条目(IP + 创建快照)。
type appliedRouteEntry struct {
	IP       string `json:"ip"`
	Snapshot string `json:"snapshot"`
}

// appliedRoutesFile 已应用路由清单的落盘结构。
// 快照随清单持久化,崩溃恢复时用快照精确删除,避免接口 metric 漂移导致失配。
type appliedRoutesFile struct {
	NicID  string              `json:"nicId"`
	Routes []appliedRouteEntry `json:"routes"`
}

// appliedRoute 已应用路由记录(创建快照供精确删除)。
type appliedRoute struct {
	nicID    string
	snapshot string
}

// Engine 核心路由服务引擎,内部互斥锁保护运行状态,可并发调用。
type Engine struct {
	mu       sync.Mutex
	resolver ipResolver
	router   route.Manager
	dataDir  string
	logFn    LogFunc

	running    bool
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	nicID      string
	enableIPv6 bool
	servers    []string
	interval   time.Duration
	rules      []ruleState
	applied    map[string]appliedRoute // 已应用路由,key 为 netip.Addr.String()
	nicDown    bool                    // 目标网卡是否处于断开状态(日志去重)
}

// New 创建引擎。resolver/router 为依赖注入(测试传 fake);
// dataDir 用于持久化已应用路由清单(残留清理),可为空字符串(跳过持久化)。
func New(resolver ipResolver, router route.Manager, dataDir string) *Engine {
	return &Engine{
		resolver: resolver,
		router:   router,
		dataDir:  dataDir,
		applied:  map[string]appliedRoute{},
	}
}

// SetLogFunc 注入日志回调(可为 nil)。
func (e *Engine) SetLogFunc(f LogFunc) {
	e.logFn = f
}

// Start 启动服务:清理残留 -> 初始化运行状态 -> 首轮解析建路由 -> 启动定时循环。
// 仅接受 checked=true 的规则;重复启动返回错误。
// 并发安全:首次加锁完成"检查+占位"原子操作,防止并发 Start 双通过。
func (e *Engine) Start(nicID string, rules []models.RouteRule, settings models.AppSettings) error {
	// P1-3 启动前置校验:网卡存在且具备 IPv4 网关(网关转发前提),缺失即拒绝启动
	if err := e.router.Validate(nicID); err != nil {
		return fmt.Errorf("启动前置校验失败: %w", err)
	}

	// 原子检查运行标志并占位,阻止并发 Start
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return errors.New("路由服务已在运行")
	}
	e.running = true
	e.mu.Unlock()

	if err := e.cleanupLeftover(); err != nil {
		e.logf(models.LogLevelWarn, fmt.Sprintf("清理上次会话残留路由失败: %v", err))
	}

	e.mu.Lock()
	if !e.running {
		// 占位期间被并发 Stop 中断,放弃本次启动
		e.mu.Unlock()
		return errors.New("路由服务启动被并发停止")
	}
	e.nicID = nicID
	e.enableIPv6 = settings.EnableIPv6
	e.servers = dnsServers(settings)
	e.interval = time.Duration(settings.QueryInterval) * time.Second
	if e.interval <= 0 {
		e.interval = 30 * time.Second
	}
	e.rules = e.rules[:0]
	for _, r := range rules {
		if r.Checked {
			e.rules = append(e.rules, ruleState{RouteRule: r})
		}
	}
	e.applied = map[string]appliedRoute{}
	e.nicDown = false
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	// 先登记 WaitGroup:保证并发 Stop 的 wg.Wait() 一定等待本次启动收尾
	// (loop 退出时 Done;若启动中途被 Stop 放弃,由下方 wg.Done() 归还)。
	e.wg.Add(1)
	e.mu.Unlock()

	e.logf(models.LogLevelInfo, fmt.Sprintf("路由服务启动,绑定网卡 %s,共 %d 条规则生效", nicID, len(e.rules)))

	// 首轮解析(并发,阻塞至完成,保证调用方拿到真实解析结果)
	e.resolveAll(ctx)

	e.mu.Lock()
	if ctx.Err() != nil || !e.running {
		// 解析期间被并发 Stop:归还计数并放弃启动,让 Stop 的 wg.Wait() 立即收尾。
		// ctx 可能尚未取消(Stop 在读到 cancel 前已完成),显式取消以释放资源。
		e.cancel()
		e.wg.Done()
		e.mu.Unlock()
		return errors.New("路由服务启动被并发停止")
	}
	go e.loop(ctx) // loop 退出时 wg.Done()
	e.mu.Unlock()
	return nil
}

// Stop 停止服务:停定时循环并清理全部已添加路由,恢复系统默认路由表。
// 未运行调用返回 nil(幂等)。
func (e *Engine) Stop() error {
	e.mu.Lock()
	if !e.running {
		e.mu.Unlock()
		return nil
	}
	if e.cancel != nil {
		e.cancel()
	}
	e.mu.Unlock()

	e.wg.Wait()

	e.mu.Lock()
	defer e.mu.Unlock()
	e.running = false

	for ip, ar := range e.applied {
		a, err := netip.ParseAddr(ip)
		if err != nil {
			continue
		}
		if err := e.router.DeleteHostRoute(a, ar.nicID, ar.snapshot); err != nil {
			e.logf(models.LogLevelError, fmt.Sprintf("停止服务清理路由 %s 失败: %v", ip, err))
			continue
		}
		delete(e.applied, ip)
		e.logf(models.LogLevelInfo, fmt.Sprintf("已清理路由: %s", ip))
	}
	_ = os.Remove(filepath.Join(e.dataDir, appliedRoutesFileName))
	e.rules = nil
	e.nicID = ""
	return nil
}

// IsRunning 返回服务是否运行中。
func (e *Engine) IsRunning() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.running
}

// Status 返回服务状态快照(运行标志、绑定网卡、含解析结果的规则列表)。
func (e *Engine) Status() models.ServiceStartResult {
	e.mu.Lock()
	defer e.mu.Unlock()
	rules := make([]models.RouteRule, 0, len(e.rules))
	for _, rs := range e.rules {
		rules = append(rules, rs.RouteRule)
	}
	return models.ServiceStartResult{
		Running: e.running,
		NicID:   e.nicID,
		Rules:   rules,
	}
}

// loop 按配置间隔周期解析,直到 ctx 取消。
func (e *Engine) loop(ctx context.Context) {
	defer e.wg.Done()
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.resolveAll(ctx)
		}
	}
}

// resolveAll 并发解析全部生效规则,更新运行期快照并 diff 同步路由表。
// 解析策略:
//   - 解析成功:更新规则 IP,纳入期望集合;
//   - 确定性无记录(ErrNoResult):清空该规则路由;
//   - 网络类失败(超时等):保留旧路由与旧 IP,避免 DNS 抖动误删。
func (e *Engine) resolveAll(ctx context.Context) {
	e.mu.Lock()
	rules := make([]ruleState, len(e.rules))
	copy(rules, e.rules)
	servers := e.servers
	enableIPv6 := e.enableIPv6
	nicID := e.nicID
	e.mu.Unlock()

	// P1-3 运行期网卡状态监测:断开时暂停同步(避免反复失败刷屏),恢复后自动继续
	if !e.checkNicState(nicID) {
		return
	}

	type result struct {
		idx int
		ips []netip.Addr
		err error
	}
	results := make([]result, len(rules))
	var wg sync.WaitGroup
	for i := range rules {
		domain := strings.TrimSpace(rules[i].Domain)
		if domain == "" {
			continue
		}
		wg.Add(1)
		go func(i int, domain string) {
			defer wg.Done()
			ips, err := e.resolver.Resolve(ctx, domain, servers, enableIPv6)
			results[i] = result{idx: i, ips: ips, err: err}
		}(i, domain)
	}
	wg.Wait()

	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.running {
		return
	}

	desired := make(map[string]struct{})
	for _, r := range results {
		if r.err != nil {
			if errors.Is(r.err, dns.ErrNoResult) {
				e.rules[r.idx].ips = nil
				e.rules[r.idx].ResolvedIP = ""
				e.rules[r.idx].LastResolvedSec = 0
			} else {
				e.logf(models.LogLevelWarn, fmt.Sprintf("域名 %s 解析失败,保留现有路由: %v", e.rules[r.idx].Domain, r.err))
				// 旧 IP 继续纳入期望集合,避免 diff 误删现有路由
				for _, ip := range e.rules[r.idx].ips {
					desired[ip.String()] = struct{}{}
				}
			}
			continue
		}
		e.rules[r.idx].ips = r.ips
		e.rules[r.idx].ResolvedIP = firstIP(r.ips)
		e.rules[r.idx].LastResolvedSec = 0
		for _, ip := range r.ips {
			desired[ip.String()] = struct{}{}
		}
	}

	e.applyDiffLocked(desired)
	e.persistAppliedLocked()
}

// checkNicState 检查目标网卡连接状态。
// 断开时返回 false 并输出一次警告(状态变化去重);恢复时输出一次恢复日志。
func (e *Engine) checkNicState(nicID string) bool {
	up, err := e.router.NicUp(nicID)
	if err != nil {
		e.logf(models.LogLevelWarn, fmt.Sprintf("查询网卡 %s 状态失败: %v", nicID, err))
		return false
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if !up {
		if !e.nicDown {
			e.nicDown = true
			e.logf(models.LogLevelWarn, fmt.Sprintf("网卡 %s 已断开,暂停路由同步,恢复后自动继续", nicID))
		}
		return false
	}
	if e.nicDown {
		e.nicDown = false
		e.logf(models.LogLevelInfo, fmt.Sprintf("网卡 %s 已恢复,继续路由同步", nicID))
	}
	return true
}

// applyDiffLocked 将路由表同步到期望集合:新增 IP 建路由,消失 IP 删路由。
// 创建时记录快照,删除复用快照精确匹配。调用方须持有 e.mu。
func (e *Engine) applyDiffLocked(desired map[string]struct{}) {
	for ip := range desired {
		if _, ok := e.applied[ip]; ok {
			continue
		}
		a, err := netip.ParseAddr(ip)
		if err != nil {
			continue
		}
		snapshot, err := e.router.AddHostRoute(a, e.nicID)
		if err != nil {
			e.logf(models.LogLevelError, fmt.Sprintf("添加路由 %s 失败: %v", ip, err))
			continue
		}
		e.applied[ip] = appliedRoute{nicID: e.nicID, snapshot: snapshot}
		e.logf(models.LogLevelInfo, fmt.Sprintf("已添加路由: %s -> %s", ip, e.nicID))
	}
	for ip, ar := range e.applied {
		if _, ok := desired[ip]; ok {
			continue
		}
		a, err := netip.ParseAddr(ip)
		if err != nil {
			delete(e.applied, ip)
			continue
		}
		if err := e.router.DeleteHostRoute(a, ar.nicID, ar.snapshot); err != nil {
			e.logf(models.LogLevelError, fmt.Sprintf("删除路由 %s 失败: %v", ip, err))
			continue
		}
		delete(e.applied, ip)
		e.logf(models.LogLevelInfo, fmt.Sprintf("已删除路由: %s", ip))
	}
}

// cleanupLeftover 启动前清理上次异常退出遗留的路由清单。
func (e *Engine) cleanupLeftover() error {
	path := filepath.Join(e.dataDir, appliedRoutesFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	// 兼容旧版清单格式 {nicId, ips}(无快照);新版为 {nicId, routes(ip+snapshot)}。
	// 旧格式路由删除时快照为空,由路由管理器按当前状态回退构造。
	var legacy struct {
		NicID string   `json:"nicId"`
		IPs   []string `json:"ips"`
	}
	_ = json.Unmarshal(data, &legacy)

	var f appliedRoutesFile
	if err := json.Unmarshal(data, &f); err != nil {
		// 清单损坏无法恢复:删除文件,残留路由依赖系统重启或手动清理。
		_ = os.Remove(path)
		return fmt.Errorf("已应用路由清单解析失败(%v),已删除该文件", err)
	}
	if len(f.Routes) == 0 && len(legacy.IPs) > 0 {
		for _, ip := range legacy.IPs {
			f.Routes = append(f.Routes, appliedRouteEntry{IP: ip})
		}
	}
	for _, r := range f.Routes {
		a, err := netip.ParseAddr(r.IP)
		if err != nil {
			continue
		}
		if err := e.router.DeleteHostRoute(a, f.NicID, r.Snapshot); err != nil {
			e.logf(models.LogLevelWarn, fmt.Sprintf("清理残留路由 %s 失败: %v", r.IP, err))
			continue
		}
		e.logf(models.LogLevelInfo, fmt.Sprintf("已清理上次会话残留路由: %s", r.IP))
	}
	return os.Remove(path)
}

// persistAppliedLocked 将当前已应用路由清单原子落盘,供下次启动清理残留。
// 调用方须持有 e.mu;dataDir 为空时跳过。
func (e *Engine) persistAppliedLocked() {
	if e.dataDir == "" {
		return
	}
	routes := make([]appliedRouteEntry, 0, len(e.applied))
	for ip, ar := range e.applied {
		routes = append(routes, appliedRouteEntry{IP: ip, Snapshot: ar.snapshot})
	}
	sort.Slice(routes, func(i, j int) bool { return routes[i].IP < routes[j].IP })
	data, err := json.MarshalIndent(appliedRoutesFile{NicID: e.nicID, Routes: routes}, "", "  ")
	if err != nil {
		return
	}
	path := filepath.Join(e.dataDir, appliedRoutesFileName)
	if err := os.MkdirAll(e.dataDir, 0o755); err != nil {
		e.logf(models.LogLevelWarn, fmt.Sprintf("持久化已应用路由清单失败: %v", err))
		return
	}
	tmp, err := os.CreateTemp(e.dataDir, ".tmp-applied-*.json")
	if err != nil {
		e.logf(models.LogLevelWarn, fmt.Sprintf("持久化已应用路由清单失败: %v", err))
		return
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		e.logf(models.LogLevelWarn, fmt.Sprintf("持久化已应用路由清单失败: %v", err))
		return
	}
	if err := tmp.Close(); err != nil {
		e.logf(models.LogLevelWarn, fmt.Sprintf("持久化已应用路由清单失败: %v", err))
		return
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		e.logf(models.LogLevelWarn, fmt.Sprintf("持久化已应用路由清单失败: %v", err))
	}
}

// dnsServers 从设置提取主/备 DNS 服务器地址列表。
func dnsServers(s models.AppSettings) []string {
	servers := make([]string, 0, 2)
	if p := strings.TrimSpace(s.PrimaryDNS); p != "" {
		servers = append(servers, p)
	}
	if sec := strings.TrimSpace(s.SecondaryDNS); sec != "" {
		servers = append(servers, sec)
	}
	return servers
}

// firstIP 返回解析结果中的首个 IP 的字符串(无结果返回空串)。
func firstIP(ips []netip.Addr) string {
	if len(ips) == 0 {
		return ""
	}
	return ips[0].String()
}

// logf 安全输出日志(回调未注入时忽略)。
func (e *Engine) logf(level models.LogLevel, text string) {
	if e.logFn != nil {
		e.logFn(level, text)
	}
}
