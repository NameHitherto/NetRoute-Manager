package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"NetRoute-Manager/internal/dns"
	"NetRoute-Manager/internal/models"
)

// fakeResult 一次解析的预置结果。
type fakeResult struct {
	ips []netip.Addr
	err error
}

// fakeResolver 可编程解析器:按域名返回预置结果并统计调用次数。
type fakeResolver struct {
	mu      sync.Mutex
	results map[string]fakeResult
	calls   int
}

func newFakeResolver(results map[string]fakeResult) *fakeResolver {
	return &fakeResolver{results: results}
}

func (f *fakeResolver) Resolve(_ context.Context, domain string, _ []string, _ bool) ([]netip.Addr, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if r, ok := f.results[domain]; ok {
		return r.ips, r.err
	}
	return nil, errors.New("未配置 " + domain)
}

func (f *fakeResolver) set(domain string, ips []netip.Addr, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.results[domain] = fakeResult{ips: ips, err: err}
}

func (f *fakeResolver) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// gatedResolver 可阻塞解析器:首次解析进入时通知 entered,随后阻塞
// 直到 release 关闭或 ctx 取消,用于测试 Start 首轮解析期间的并发 Stop。
type gatedResolver struct {
	mu      sync.Mutex
	results map[string]fakeResult
	entered chan struct{}
	release chan struct{}
}

func (g *gatedResolver) Resolve(ctx context.Context, domain string, _ []string, _ bool) ([]netip.Addr, error) {
	select {
	case g.entered <- struct{}{}:
	default:
	}
	select {
	case <-g.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if r, ok := g.results[domain]; ok {
		return r.ips, r.err
	}
	return nil, errors.New("未配置 " + domain)
}

// TestStartStoppedDuringResolve 首轮解析期间并发 Stop:
// Start 应返回错误、服务应停止、不残留任何路由。
func TestStartStoppedDuringResolve(t *testing.T) {
	g := &gatedResolver{
		results: map[string]fakeResult{"a.com": {ips: []netip.Addr{netip.MustParseAddr("1.1.1.1")}}},
		entered: make(chan struct{}),
		release: make(chan struct{}), // 不关闭,依赖 Stop 取消 ctx 放行解析
	}
	router := newFakeRouter()
	e, _ := newTestEngine(t, g, router)
	settings := models.DefaultSettings()
	rules := []models.RouteRule{{ID: "r1", Domain: "a.com", Checked: true}}

	startDone := make(chan struct{})
	var startErr error
	go func() {
		startErr = e.Start("nic", rules, settings)
		close(startDone)
	}()

	<-g.entered // Start 已进入首轮解析并阻塞

	stopDone := make(chan struct{})
	go func() {
		_ = e.Stop()
		close(stopDone)
	}()

	<-startDone
	select {
	case <-stopDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() 超时:WaitGroup 未正确收尾")
	}

	if startErr == nil {
		t.Fatal("首轮解析期间被 Stop,Start() 应返回错误")
	}
	if e.IsRunning() {
		t.Fatal("停止后服务不应处于运行状态")
	}
	if got := router.snapshot(); len(got) != 0 {
		t.Fatalf("停止后不应残留路由: %v", got)
	}

	// 中止后引擎应可复用:WaitGroup 计数未破坏,可重新启动再停止。
	close(g.release)
	if err := e.Start("nic", rules, settings); err != nil {
		t.Fatalf("停止后重新 Start() 出错: %v", err)
	}
	if err := e.Stop(); err != nil {
		t.Fatalf("重新启动后 Stop() 出错: %v", err)
	}
	if e.IsRunning() {
		t.Fatal("重新启动并停止后服务不应处于运行状态")
	}
}

// fakeRouter 记录路由操作的假路由管理器。
type fakeRouter struct {
	mu          sync.Mutex
	applied     map[string]struct{}
	snapshots   map[string]string
	added       int
	deleted     int
	failAdd     map[string]bool
	up          bool
	validateErr error
}

func newFakeRouter() *fakeRouter {
	return &fakeRouter{
		applied:   map[string]struct{}{},
		snapshots: map[string]string{},
		failAdd:   map[string]bool{},
		up:        true,
	}
}

func (f *fakeRouter) AddHostRoute(ip netip.Addr, _ string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failAdd[ip.String()] {
		return "", errors.New("注入的添加失败")
	}
	f.applied[ip.String()] = struct{}{}
	snap := "snapshot:" + ip.String()
	f.snapshots[ip.String()] = snap
	f.added++
	return snap, nil
}

func (f *fakeRouter) DeleteHostRoute(ip netip.Addr, _ string, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.applied[ip.String()]; ok {
		delete(f.applied, ip.String())
		delete(f.snapshots, ip.String())
		f.deleted++
	}
	return nil
}

func (f *fakeRouter) Validate(string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.validateErr
}

func (f *fakeRouter) NicUp(string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.up, nil
}

func (f *fakeRouter) setUp(up bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.up = up
}

func (f *fakeRouter) setValidateErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.validateErr = err
}

func (f *fakeRouter) snapshot() map[string]struct{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[string]struct{}, len(f.applied))
	for ip := range f.applied {
		out[ip] = struct{}{}
	}
	return out
}

func (f *fakeRouter) addCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.added
}

// preseed 预置已应用路由(模拟上次会话残留)。
func (f *fakeRouter) preseed(ips ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, ip := range ips {
		f.applied[ip] = struct{}{}
	}
}

// newTestEngine 创建基于临时目录的引擎。
func newTestEngine(t *testing.T, resolver ipResolver, router *fakeRouter) (*Engine, string) {
	t.Helper()
	dir := t.TempDir()
	return New(resolver, router, dir), dir
}

func mapsEqual(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

func TestStartAppliesRoutesAndStatus(t *testing.T) {
	resolver := newFakeResolver(map[string]fakeResult{
		"api.example.com": {ips: []netip.Addr{netip.MustParseAddr("1.2.3.4"), netip.MustParseAddr("1.2.3.5")}},
		"cdn.example.com": {ips: []netip.Addr{netip.MustParseAddr("9.9.9.9")}},
	})
	router := newFakeRouter()
	e, dir := newTestEngine(t, resolver, router)

	settings := models.DefaultSettings()
	rules := []models.RouteRule{
		{ID: "r1", Domain: "api.example.com", Checked: true},
		{ID: "r2", Domain: "cdn.example.com", Checked: true},
		{ID: "r3", Domain: "disabled.example.com", Checked: false},
	}
	if err := e.Start("nic-guid-1", rules, settings); err != nil {
		t.Fatalf("Start() 出错: %v", err)
	}
	if !e.IsRunning() {
		t.Fatal("启动后应处于运行状态")
	}

	want := map[string]struct{}{"1.2.3.4": {}, "1.2.3.5": {}, "9.9.9.9": {}}
	if got := router.snapshot(); !mapsEqual(got, want) {
		t.Fatalf("启动后路由不符: got %v want %v", got, want)
	}

	st := e.Status()
	if !st.Running || st.NicID != "nic-guid-1" {
		t.Fatalf("状态快照不符: %#v", st)
	}
	if len(st.Rules) != 2 {
		t.Fatalf("运行快照应只含 2 条勾选规则,got %d", len(st.Rules))
	}
	if st.Rules[0].ResolvedIP != "1.2.3.4" {
		t.Fatalf("首规则 resolvedIp 应为 1.2.3.4,got %q", st.Rules[0].ResolvedIP)
	}

	// 已应用清单已落盘
	data, err := os.ReadFile(filepath.Join(dir, appliedRoutesFileName))
	if err != nil {
		t.Fatalf("已应用清单未落盘: %v", err)
	}
	var f appliedRoutesFile
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("清单解析失败: %v", err)
	}
	if f.NicID != "nic-guid-1" || len(f.Routes) != 3 {
		t.Fatalf("清单内容不符: %#v", f)
	}
	for _, r := range f.Routes {
		if r.Snapshot == "" {
			t.Fatalf("清单条目应含快照: %#v", r)
		}
	}
}

func TestStartTwiceFails(t *testing.T) {
	resolver := newFakeResolver(map[string]fakeResult{"a.com": {ips: []netip.Addr{netip.MustParseAddr("1.1.1.1")}}})
	router := newFakeRouter()
	e, _ := newTestEngine(t, resolver, router)
	settings := models.DefaultSettings()
	rules := []models.RouteRule{{ID: "r1", Domain: "a.com", Checked: true}}

	if err := e.Start("nic", rules, settings); err != nil {
		t.Fatalf("首次 Start() 出错: %v", err)
	}
	if err := e.Start("nic", rules, settings); err == nil {
		t.Fatal("重复 Start() 应返回错误")
	}
}

func TestConcurrentStartOnlyOneSucceeds(t *testing.T) {
	resolver := newFakeResolver(map[string]fakeResult{"a.com": {ips: []netip.Addr{netip.MustParseAddr("1.1.1.1")}}})
	router := newFakeRouter()
	e, _ := newTestEngine(t, resolver, router)
	settings := models.DefaultSettings()
	rules := []models.RouteRule{{ID: "r1", Domain: "a.com", Checked: true}}

	const n = 8
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = e.Start("nic", rules, settings)
		}(i)
	}
	wg.Wait()

	success := 0
	for _, err := range errs {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("并发 Start() 应恰好 1 次成功,got %d (errs=%v)", success, errs)
	}
	if !e.IsRunning() {
		t.Fatal("并发启动后应处于运行状态")
	}
	if err := e.Stop(); err != nil {
		t.Fatalf("Stop() 出错: %v", err)
	}
}

func TestStopCleansRoutes(t *testing.T) {
	resolver := newFakeResolver(map[string]fakeResult{"a.com": {ips: []netip.Addr{netip.MustParseAddr("1.1.1.1")}}})
	router := newFakeRouter()
	e, dir := newTestEngine(t, resolver, router)
	settings := models.DefaultSettings()
	rules := []models.RouteRule{{ID: "r1", Domain: "a.com", Checked: true}}

	if err := e.Start("nic", rules, settings); err != nil {
		t.Fatalf("Start() 出错: %v", err)
	}
	if err := e.Stop(); err != nil {
		t.Fatalf("Stop() 出错: %v", err)
	}
	if e.IsRunning() {
		t.Fatal("Stop() 后不应处于运行状态")
	}
	if got := router.snapshot(); len(got) != 0 {
		t.Fatalf("Stop() 后应清理全部路由,got %v", got)
	}
	if _, err := os.Stat(filepath.Join(dir, appliedRoutesFileName)); !os.IsNotExist(err) {
		t.Fatalf("Stop() 后清单文件应被删除,err=%v", err)
	}
	// 幂等:重复 Stop 不报错
	if err := e.Stop(); err != nil {
		t.Fatalf("重复 Stop() 应幂等,got %v", err)
	}
}

func TestResolveChangeUpdatesRoutes(t *testing.T) {
	resolver := newFakeResolver(map[string]fakeResult{
		"api.example.com": {ips: []netip.Addr{netip.MustParseAddr("1.2.3.4"), netip.MustParseAddr("1.2.3.5")}},
	})
	router := newFakeRouter()
	e, _ := newTestEngine(t, resolver, router)
	settings := models.DefaultSettings()
	rules := []models.RouteRule{{ID: "r1", Domain: "api.example.com", Checked: true}}

	if err := e.Start("nic", rules, settings); err != nil {
		t.Fatalf("Start() 出错: %v", err)
	}

	// 解析结果变化:1.2.3.5 消失,2.2.2.2 出现
	resolver.set("api.example.com", []netip.Addr{netip.MustParseAddr("1.2.3.4"), netip.MustParseAddr("2.2.2.2")}, nil)
	e.resolveAll(context.Background())

	want := map[string]struct{}{"1.2.3.4": {}, "2.2.2.2": {}}
	if got := router.snapshot(); !mapsEqual(got, want) {
		t.Fatalf("diff 更新后路由不符: got %v want %v", got, want)
	}
	st := e.Status()
	if len(st.Rules) != 1 || st.Rules[0].ResolvedIP != "1.2.3.4" {
		t.Fatalf("状态快照不符: %#v", st.Rules)
	}
}

func TestStartCleansLeftoverRoutes(t *testing.T) {
	resolver := newFakeResolver(map[string]fakeResult{"a.com": {ips: []netip.Addr{netip.MustParseAddr("1.1.1.1")}}})
	router := newFakeRouter()
	e, dir := newTestEngine(t, resolver, router)

	// 模拟上次异常退出:残留清单 + 系统里仍存在的旧路由
	leftover := appliedRoutesFile{NicID: "old-nic", Routes: []appliedRouteEntry{
		{IP: "10.0.0.1", Snapshot: "snapshot:10.0.0.1"},
		{IP: "2001:db8::1", Snapshot: "snapshot:2001:db8::1"},
	}}
	data, _ := json.Marshal(leftover)
	if err := os.WriteFile(filepath.Join(dir, appliedRoutesFileName), data, 0o644); err != nil {
		t.Fatalf("写入残留清单失败: %v", err)
	}
	router.preseed("10.0.0.1")

	settings := models.DefaultSettings()
	rules := []models.RouteRule{{ID: "r1", Domain: "a.com", Checked: true}}
	if err := e.Start("new-nic", rules, settings); err != nil {
		t.Fatalf("Start() 出错: %v", err)
	}

	got := router.snapshot()
	if _, ok := got["10.0.0.1"]; ok {
		t.Fatalf("残留路由 10.0.0.1 应被清理: %v", got)
	}
	if _, ok := got["1.1.1.1"]; !ok {
		t.Fatalf("新会话路由 1.1.1.1 应已添加: %v", got)
	}
	// 启动后清单文件应被新会话覆盖(含新路由,不含残留)
	data, err := os.ReadFile(filepath.Join(dir, appliedRoutesFileName))
	if err != nil {
		t.Fatalf("新会话清单应已落盘: %v", err)
	}
	var f appliedRoutesFile
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("清单解析失败: %v", err)
	}
	if f.NicID != "new-nic" || len(f.Routes) != 1 || f.Routes[0].IP != "1.1.1.1" {
		t.Fatalf("新会话清单内容不符: %#v", f)
	}
}

func TestResolveNetworkErrorKeepsRoutes(t *testing.T) {
	resolver := newFakeResolver(map[string]fakeResult{
		"api.example.com": {ips: []netip.Addr{netip.MustParseAddr("1.2.3.4")}},
	})
	router := newFakeRouter()
	e, _ := newTestEngine(t, resolver, router)
	settings := models.DefaultSettings()
	rules := []models.RouteRule{{ID: "r1", Domain: "api.example.com", Checked: true}}

	if err := e.Start("nic", rules, settings); err != nil {
		t.Fatalf("Start() 出错: %v", err)
	}

	// 网络类失败(DNS 超时):应保留旧路由
	resolver.set("api.example.com", nil, errors.New("i/o timeout"))
	e.resolveAll(context.Background())

	if got := router.snapshot(); len(got) != 1 {
		t.Fatalf("解析失败应保留旧路由,got %v", got)
	}
	st := e.Status()
	if len(st.Rules) != 1 || st.Rules[0].ResolvedIP != "1.2.3.4" {
		t.Fatalf("解析失败应保留旧解析结果: %#v", st.Rules)
	}
}

func TestErrNoResultRemovesRoutes(t *testing.T) {
	resolver := newFakeResolver(map[string]fakeResult{
		"api.example.com": {ips: []netip.Addr{netip.MustParseAddr("1.2.3.4")}},
	})
	router := newFakeRouter()
	e, _ := newTestEngine(t, resolver, router)
	settings := models.DefaultSettings()
	rules := []models.RouteRule{{ID: "r1", Domain: "api.example.com", Checked: true}}

	if err := e.Start("nic", rules, settings); err != nil {
		t.Fatalf("Start() 出错: %v", err)
	}

	// 域名确定性无记录:应清理该规则路由
	resolver.set("api.example.com", nil, dns.ErrNoResult)
	e.resolveAll(context.Background())

	if got := router.snapshot(); len(got) != 0 {
		t.Fatalf("ErrNoResult 应清理该规则路由,got %v", got)
	}
	st := e.Status()
	if len(st.Rules) != 1 || st.Rules[0].ResolvedIP != "" {
		t.Fatalf("ErrNoResult 后 resolvedIp 应为空: %#v", st.Rules)
	}
}

func TestDuplicateIPDeduplicated(t *testing.T) {
	resolver := newFakeResolver(map[string]fakeResult{
		"a.com": {ips: []netip.Addr{netip.MustParseAddr("9.9.9.9")}},
		"b.com": {ips: []netip.Addr{netip.MustParseAddr("9.9.9.9")}},
	})
	router := newFakeRouter()
	e, _ := newTestEngine(t, resolver, router)
	settings := models.DefaultSettings()
	rules := []models.RouteRule{
		{ID: "r1", Domain: "a.com", Checked: true},
		{ID: "r2", Domain: "b.com", Checked: true},
	}

	if err := e.Start("nic", rules, settings); err != nil {
		t.Fatalf("Start() 出错: %v", err)
	}
	if router.addCount() != 1 {
		t.Fatalf("重复 IP 应只添加一次路由,got %d 次", router.addCount())
	}
}

func TestStartValidateFails(t *testing.T) {
	resolver := newFakeResolver(map[string]fakeResult{"a.com": {ips: []netip.Addr{netip.MustParseAddr("1.1.1.1")}}})
	router := newFakeRouter()
	router.setValidateErr(errors.New("网卡无 IPv4 网关"))
	e, _ := newTestEngine(t, resolver, router)
	settings := models.DefaultSettings()
	rules := []models.RouteRule{{ID: "r1", Domain: "a.com", Checked: true}}

	if err := e.Start("nic", rules, settings); err == nil {
		t.Fatal("前置校验失败时 Start() 应返回错误")
	}
	if e.IsRunning() {
		t.Fatal("校验失败后不应处于运行状态")
	}
	if got := router.snapshot(); len(got) != 0 {
		t.Fatalf("校验失败不应添加路由,got %v", got)
	}
}

func TestNicDownPausesSyncAndResumes(t *testing.T) {
	resolver := newFakeResolver(map[string]fakeResult{"a.com": {ips: []netip.Addr{netip.MustParseAddr("1.1.1.1")}}})
	router := newFakeRouter()
	e, _ := newTestEngine(t, resolver, router)
	settings := models.DefaultSettings()
	rules := []models.RouteRule{{ID: "r1", Domain: "a.com", Checked: true}}

	if err := e.Start("nic", rules, settings); err != nil {
		t.Fatalf("Start() 出错: %v", err)
	}
	if got := router.snapshot(); len(got) != 1 {
		t.Fatalf("启动后应已添加路由,got %v", got)
	}

	// 网卡断开:后续轮次应暂停同步(不新增/不删除路由)
	router.setUp(false)
	resolver.set("a.com", []netip.Addr{netip.MustParseAddr("2.2.2.2")}, nil)
	e.resolveAll(context.Background())
	if got := router.snapshot(); len(got) != 1 {
		t.Fatalf("网卡断开时不应变更路由,got %v", got)
	}

	// 网卡恢复:自动继续同步(diff 到新 IP)
	router.setUp(true)
	e.resolveAll(context.Background())
	want := map[string]struct{}{"2.2.2.2": {}}
	if got := router.snapshot(); !mapsEqual(got, want) {
		t.Fatalf("恢复后应同步到新路由,got %v want %v", got, want)
	}
}

func TestStartCleansLegacyLeftoverFormat(t *testing.T) {
	resolver := newFakeResolver(map[string]fakeResult{"a.com": {ips: []netip.Addr{netip.MustParseAddr("1.1.1.1")}}})
	router := newFakeRouter()
	e, dir := newTestEngine(t, resolver, router)

	// 旧版清单格式 {nicId, ips}:无快照,清理时应按当前状态回退删除
	leftover := `{"nicId":"old-nic","ips":["10.0.0.1"]}`
	if err := os.WriteFile(filepath.Join(dir, appliedRoutesFileName), []byte(leftover), 0o644); err != nil {
		t.Fatalf("写入旧版清单失败: %v", err)
	}
	router.preseed("10.0.0.1")

	settings := models.DefaultSettings()
	rules := []models.RouteRule{{ID: "r1", Domain: "a.com", Checked: true}}
	if err := e.Start("new-nic", rules, settings); err != nil {
		t.Fatalf("Start() 出错: %v", err)
	}

	got := router.snapshot()
	if _, ok := got["10.0.0.1"]; ok {
		t.Fatalf("旧版清单残留路由 10.0.0.1 应被清理: %v", got)
	}
	if _, ok := got["1.1.1.1"]; !ok {
		t.Fatalf("新会话路由 1.1.1.1 应已添加: %v", got)
	}
}

func TestLoopReResolves(t *testing.T) {
	resolver := newFakeResolver(map[string]fakeResult{
		"a.com": {ips: []netip.Addr{netip.MustParseAddr("1.1.1.1")}},
	})
	router := newFakeRouter()
	e, _ := newTestEngine(t, resolver, router)
	settings := models.DefaultSettings()
	settings.QueryInterval = 1 // 1 秒
	rules := []models.RouteRule{{ID: "r1", Domain: "a.com", Checked: true}}

	if err := e.Start("nic", rules, settings); err != nil {
		t.Fatalf("Start() 出错: %v", err)
	}
	defer func() { _ = e.Stop() }()

	// 首轮解析 1 次,等待 ticker 触发第二轮
	deadline := time.Now().Add(3 * time.Second)
	for resolver.count() < 2 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if resolver.count() < 2 {
		t.Fatalf("定时循环未按间隔触发第二轮解析,calls=%d", resolver.count())
	}
}
