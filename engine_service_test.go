package main

import (
	"context"
	"errors"
	"net/netip"
	"sync"
	"testing"

	"NetRoute-Manager/internal/models"
	"NetRoute-Manager/internal/service"
	"NetRoute-Manager/internal/store"
)

// apiFakeResolver 供 EngineService 层测试注入的最小可编程解析器。
type apiFakeResolver struct {
	mu      sync.Mutex
	results map[string][]netip.Addr
}

func (f *apiFakeResolver) Resolve(_ context.Context, domain string, _ []string, _ bool) ([]netip.Addr, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if ips, ok := f.results[domain]; ok {
		return ips, nil
	}
	return nil, errors.New("未配置 " + domain)
}

// apiFakeRouter 记录路由操作的假路由管理器。
type apiFakeRouter struct {
	mu      sync.Mutex
	applied map[string]struct{}
	up      bool
}

func newAPIFakeRouter() *apiFakeRouter {
	return &apiFakeRouter{applied: map[string]struct{}{}, up: true}
}

func (f *apiFakeRouter) AddHostRoute(ip netip.Addr, _ string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.applied[ip.String()] = struct{}{}
	return "snapshot:" + ip.String(), nil
}

func (f *apiFakeRouter) DeleteHostRoute(ip netip.Addr, _ string, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.applied, ip.String())
	return nil
}

func (f *apiFakeRouter) Validate(string) error { return nil }

func (f *apiFakeRouter) NicUp(string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.up {
		return false, nil
	}
	return true, nil
}

func (f *apiFakeRouter) snapshot() map[string]struct{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[string]struct{}, len(f.applied))
	for ip := range f.applied {
		out[ip] = struct{}{}
	}
	return out
}

// newTestEngineService 创建带真实引擎(注入 fake 解析器/路由管理器)的 EngineService。
func newTestEngineService(t *testing.T, resolver *apiFakeResolver, router *apiFakeRouter) *EngineService {
	t.Helper()
	dir := t.TempDir()
	engine := service.New(resolver, router, dir)
	return NewEngineService(store.NewWithDir(dir), engine)
}

func TestStartService(t *testing.T) {
	resolver := &apiFakeResolver{results: map[string][]netip.Addr{
		"api.example.com": {netip.MustParseAddr("1.2.3.4")},
	}}
	router := newAPIFakeRouter()
	svc := newTestEngineService(t, resolver, router)

	rules := []models.RouteRule{
		{ID: "r1", Domain: "api.example.com", Checked: true},
		{ID: "r2", Domain: "off.example.com", Checked: false},
	}
	result, err := svc.StartService("nic-guid", rules)
	if err != nil {
		t.Fatalf("StartService() 出错: %v", err)
	}
	if !result.Running || result.NicID != "nic-guid" {
		t.Fatalf("启动结果不符: %#v", result)
	}
	if len(result.Rules) != 1 || result.Rules[0].ResolvedIP != "1.2.3.4" {
		t.Fatalf("解析结果未写入快照: %#v", result.Rules)
	}
	if got := router.snapshot(); len(got) != 1 {
		t.Fatalf("路由应已添加,got %v", got)
	}
}

func TestStartServiceTwiceFails(t *testing.T) {
	resolver := &apiFakeResolver{results: map[string][]netip.Addr{"a.com": {netip.MustParseAddr("1.1.1.1")}}}
	svc := newTestEngineService(t, resolver, newAPIFakeRouter())
	rules := []models.RouteRule{{ID: "r1", Domain: "a.com", Checked: true}}

	if _, err := svc.StartService("nic", rules); err != nil {
		t.Fatalf("首次 StartService() 出错: %v", err)
	}
	if _, err := svc.StartService("nic", rules); err == nil {
		t.Fatal("重复 StartService() 应返回错误")
	}
}

func TestStopServiceCleansRoutes(t *testing.T) {
	resolver := &apiFakeResolver{results: map[string][]netip.Addr{"a.com": {netip.MustParseAddr("1.1.1.1")}}}
	router := newAPIFakeRouter()
	svc := newTestEngineService(t, resolver, router)
	rules := []models.RouteRule{{ID: "r1", Domain: "a.com", Checked: true}}

	if _, err := svc.StartService("nic", rules); err != nil {
		t.Fatalf("StartService() 出错: %v", err)
	}
	if err := svc.StopService(); err != nil {
		t.Fatalf("StopService() 出错: %v", err)
	}
	if got := router.snapshot(); len(got) != 0 {
		t.Fatalf("停止后应清理全部路由,got %v", got)
	}
}

func TestGetServiceStatus(t *testing.T) {
	resolver := &apiFakeResolver{results: map[string][]netip.Addr{"a.com": {netip.MustParseAddr("1.1.1.1")}}}
	svc := newTestEngineService(t, resolver, newAPIFakeRouter())

	st, err := svc.GetServiceStatus()
	if err != nil {
		t.Fatalf("GetServiceStatus() 出错: %v", err)
	}
	if st.Running {
		t.Fatal("未启动时应返回 running=false")
	}

	rules := []models.RouteRule{{ID: "r1", Domain: "a.com", Checked: true}}
	if _, err := svc.StartService("nic", rules); err != nil {
		t.Fatalf("StartService() 出错: %v", err)
	}
	st, err = svc.GetServiceStatus()
	if err != nil {
		t.Fatalf("GetServiceStatus() 出错: %v", err)
	}
	if !st.Running || len(st.Rules) != 1 {
		t.Fatalf("运行中状态不符: %#v", st)
	}
	_ = svc.StopService()
}
