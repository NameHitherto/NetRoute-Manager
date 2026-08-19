package main

import (
	"errors"
	"strings"
	"testing"

	"NetRoute-Manager/internal/models"
	"NetRoute-Manager/internal/store"
)

// newTestRoutesService 创建基于临时目录的 RoutesService,避免污染真实用户数据目录。
func newTestRoutesService(t *testing.T) *RoutesService {
	t.Helper()
	return NewRoutesService(store.NewWithDir(t.TempDir()))
}

func TestCreateRoute(t *testing.T) {
	svc := newTestRoutesService(t)
	rule, err := svc.CreateRoute(models.RouteRuleInput{Domain: "api.openai.com", Port: "443", Alias: "AI API"})
	if err != nil {
		t.Fatalf("CreateRoute() 出错: %v", err)
	}
	if rule.ID == "" {
		t.Fatal("CreateRoute() 应生成非空 ID")
	}
	if !rule.Checked {
		t.Fatal("CreateRoute() 新建条目应默认 checked=true")
	}
	if rule.Domain != "api.openai.com" || rule.Port != "443" || rule.Alias != "AI API" {
		t.Fatalf("字段未正确写入: %#v", rule)
	}
}

func TestListRoutes(t *testing.T) {
	svc := newTestRoutesService(t)
	if _, err := svc.CreateRoute(models.RouteRuleInput{Domain: "github.com", Port: "443", Alias: "GitHub"}); err != nil {
		t.Fatalf("CreateRoute() 出错: %v", err)
	}
	routes, err := svc.ListRoutes()
	if err != nil {
		t.Fatalf("ListRoutes() 出错: %v", err)
	}
	if len(routes) != 1 {
		t.Fatalf("ListRoutes() 应返回 1 条,got %d", len(routes))
	}
}

func TestUpdateRoute(t *testing.T) {
	svc := newTestRoutesService(t)
	rule, err := svc.CreateRoute(models.RouteRuleInput{Domain: "github.com", Port: "443", Alias: "GitHub"})
	if err != nil {
		t.Fatalf("CreateRoute() 出错: %v", err)
	}
	// 模拟运行期字段(直接写 store 绕过 API 层,仅为构造 UpdateRoute 的输入前提)
	rule.Checked = false
	rule.ResolvedIP = "1.2.3.4"
	rule.LastResolvedSec = 12
	routes, _ := svc.ListRoutes()
	routes[0] = rule
	if err := svc.store.SaveRoutes(routes); err != nil {
		t.Fatalf("SaveRoutes() 出错: %v", err)
	}

	updated, err := svc.UpdateRoute(rule.ID, models.RouteRuleInput{Domain: "raw.githubusercontent.com", Port: "443", Alias: "Raw 加速"})
	if err != nil {
		t.Fatalf("UpdateRoute() 出错: %v", err)
	}
	if updated.Domain != "raw.githubusercontent.com" || updated.Alias != "Raw 加速" {
		t.Fatalf("更新字段未生效: %#v", updated)
	}
	if updated.Checked != false || updated.ResolvedIP != "1.2.3.4" || updated.LastResolvedSec != 12 {
		t.Fatalf("运行期字段不应被更新清空: %#v", updated)
	}
}

func TestUpdateRouteNotFound(t *testing.T) {
	svc := newTestRoutesService(t)
	_, err := svc.UpdateRoute("no-such-id", models.RouteRuleInput{Domain: "a.com", Port: "443"})
	if !errors.Is(err, ErrRouteNotFound) {
		t.Fatalf("UpdateRoute() 不存在应返回 ErrRouteNotFound,got %v", err)
	}
}

func TestDeleteRoute(t *testing.T) {
	svc := newTestRoutesService(t)
	rule, err := svc.CreateRoute(models.RouteRuleInput{Domain: "github.com", Port: "443", Alias: "GitHub"})
	if err != nil {
		t.Fatalf("CreateRoute() 出错: %v", err)
	}
	if err := svc.DeleteRoute(rule.ID); err != nil {
		t.Fatalf("DeleteRoute() 出错: %v", err)
	}
	routes, _ := svc.ListRoutes()
	if len(routes) != 0 {
		t.Fatalf("删除后应无残留,got %d 条", len(routes))
	}
}

func TestDeleteRouteNotFound(t *testing.T) {
	svc := newTestRoutesService(t)
	if err := svc.DeleteRoute("no-such-id"); !errors.Is(err, ErrRouteNotFound) {
		t.Fatalf("DeleteRoute() 不存在应返回 ErrRouteNotFound,got %v", err)
	}
}

// TestPersistenceAcrossStores 验证持久化闭环:数据落盘后,新实例仍可读取。
func TestPersistenceAcrossStores(t *testing.T) {
	dir := t.TempDir()
	svc1 := NewRoutesService(store.NewWithDir(dir))
	rule, err := svc1.CreateRoute(models.RouteRuleInput{Domain: "api.openai.com", Port: "443", Alias: "AI API"})
	if err != nil {
		t.Fatalf("CreateRoute() 出错: %v", err)
	}

	svc2 := NewRoutesService(store.NewWithDir(dir))
	routes, err := svc2.ListRoutes()
	if err != nil {
		t.Fatalf("新实例 ListRoutes() 出错: %v", err)
	}
	if len(routes) != 1 || routes[0].ID != rule.ID {
		t.Fatalf("持久化闭环失败: got %#v", routes)
	}
}

func TestCreateRouteValidation(t *testing.T) {
	svc := newTestRoutesService(t)
	cases := []struct {
		name  string
		input models.RouteRuleInput
	}{
		{"空域名", models.RouteRuleInput{Domain: "  ", Port: "443"}},
		{"非法端口-非数字", models.RouteRuleInput{Domain: "a.com", Port: "abc"}},
		{"非法端口-越界", models.RouteRuleInput{Domain: "a.com", Port: "70000"}},
	}
	for _, tc := range cases {
		if _, err := svc.CreateRoute(tc.input); err == nil {
			t.Errorf("%s: 应校验失败", tc.name)
		}
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	svc := NewSettingsService(store.NewWithDir(t.TempDir()))
	want := models.AppSettings{
		PrimaryDNS:    "8.8.8.8",
		SecondaryDNS:  "8.8.4.4",
		QueryInterval: 60,
		EnableIPv6:    true,
		AutoStart:     false,
		MinToTray:     false,
		DNSMode:       models.DnsModeUDP,
	}
	if err := svc.SaveSettings(want); err != nil {
		t.Fatalf("SaveSettings() 出错: %v", err)
	}
	got, err := svc.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() 出错: %v", err)
	}
	if got != want {
		t.Fatalf("round-trip 不一致: got %#v want %#v", got, want)
	}
}

func TestSaveSettingsValidation(t *testing.T) {
	svc := NewSettingsService(store.NewWithDir(t.TempDir()))
	base := models.DefaultSettings()
	cases := []struct {
		name       string
		mutate     func(*models.AppSettings)
		wantSubstr string
	}{
		{"空主 DNS", func(s *models.AppSettings) { s.PrimaryDNS = " " }, "DNS"},
		{"空备 DNS", func(s *models.AppSettings) { s.SecondaryDNS = "" }, "DNS"},
		{"非正间隔", func(s *models.AppSettings) { s.QueryInterval = 0 }, "间隔"},
		{"非法模式", func(s *models.AppSettings) { s.DNSMode = "QUIC" }, "模式"},
	}
	for _, tc := range cases {
		s := base
		tc.mutate(&s)
		err := svc.SaveSettings(s)
		if err == nil {
			t.Errorf("%s: 应校验失败", tc.name)
			continue
		}
		if !strings.Contains(err.Error(), tc.wantSubstr) {
			t.Errorf("%s: 错误信息应包含 %q,got %q", tc.name, tc.wantSubstr, err.Error())
		}
	}
}
