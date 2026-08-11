package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"NetRoute-Manager/internal/models"
)

// newTestStore 创建基于临时目录的 Store,避免污染真实用户目录。
func newTestStore(t *testing.T) *Store {
	t.Helper()
	return NewWithDir(t.TempDir())
}

// TestLoadRoutesCreatesEmptyFileOnFirstLoad 首次加载自动创建目录与空的 routes.json。
func TestLoadRoutesCreatesEmptyFileOnFirstLoad(t *testing.T) {
	s := newTestStore(t)

	routes, err := s.LoadRoutes()
	if err != nil {
		t.Fatalf("LoadRoutes() 出错: %v", err)
	}
	if routes == nil || len(routes) != 0 {
		t.Fatalf("首次加载应返回空切片,got %#v", routes)
	}
	if _, err := os.Stat(filepath.Join(s.RootDir(), routesFileName)); err != nil {
		t.Fatalf("routes.json 应被创建: %v", err)
	}
}

// TestLoadSettingsWritesDefaultsOnFirstLoad 首次加载自动写入默认设置。
func TestLoadSettingsWritesDefaultsOnFirstLoad(t *testing.T) {
	s := newTestStore(t)

	got, err := s.LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings() 出错: %v", err)
	}
	if got != models.DefaultSettings() {
		t.Fatalf("首次加载应返回默认设置,got %#v", got)
	}
	if _, err := os.Stat(filepath.Join(s.RootDir(), settingsFileName)); err != nil {
		t.Fatalf("settings.json 应被创建: %v", err)
	}
}

// TestRoutesRoundTrip 保存后重新加载,数据完全一致。
func TestRoutesRoundTrip(t *testing.T) {
	s := newTestStore(t)
	want := []models.RouteRule{
		{ID: "r1", Domain: "api.openai.com", Port: "443", Alias: "AI 服务 API 节点", Checked: true, ResolvedIP: "1.2.3.4", LastResolvedSec: 12},
		{ID: "r2", Domain: "github.com", Port: "443", Alias: "GitHub 主站", Checked: false, ResolvedIP: "", LastResolvedSec: 0},
	}
	if err := s.SaveRoutes(want); err != nil {
		t.Fatalf("SaveRoutes() 出错: %v", err)
	}
	got, err := s.LoadRoutes()
	if err != nil {
		t.Fatalf("LoadRoutes() 出错: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("条目数不一致: got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("round-trip 不一致: got %#v want %#v", got[i], want[i])
		}
	}
}

// TestSettingsRoundTrip 保存后重新加载,数据完全一致。
func TestSettingsRoundTrip(t *testing.T) {
	s := newTestStore(t)
	want := models.AppSettings{
		PrimaryDNS:    "8.8.8.8",
		SecondaryDNS:  "8.8.4.4",
		QueryInterval: 60,
		EnableIPv6:    true,
		AutoStart:     false,
		MinToTray:     false,
		DNSMode:       models.DnsModeDoH,
	}
	if err := s.SaveSettings(want); err != nil {
		t.Fatalf("SaveSettings() 出错: %v", err)
	}
	got, err := s.LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings() 出错: %v", err)
	}
	if got != want {
		t.Fatalf("round-trip 不一致: got %#v want %#v", got, want)
	}
}

// TestCorruptRoutesBackedUpAndReset 损坏的路由文件自动备份并重置为空列表。
func TestCorruptRoutesBackedUpAndReset(t *testing.T) {
	s := newTestStore(t)
	path := filepath.Join(s.RootDir(), routesFileName)
	if err := os.WriteFile(path, []byte("{not-valid-json"), 0o644); err != nil {
		t.Fatalf("写入损坏文件失败: %v", err)
	}

	routes, err := s.LoadRoutes()
	if err != nil {
		t.Fatalf("损坏文件应自动恢复而非报错: %v", err)
	}
	if len(routes) != 0 {
		t.Fatalf("损坏后应重置为空列表,got %#v", routes)
	}
	backups, _ := filepath.Glob(filepath.Join(s.RootDir(), routesFileName+".corrupt-*"))
	if len(backups) != 1 {
		t.Fatalf("应生成 1 个备份文件,got %v", backups)
	}
}

// TestCorruptSettingsBackedUpAndReset 损坏的设置文件自动备份并重置为默认设置。
func TestCorruptSettingsBackedUpAndReset(t *testing.T) {
	s := newTestStore(t)
	path := filepath.Join(s.RootDir(), settingsFileName)
	if err := os.WriteFile(path, []byte("not json at all"), 0o644); err != nil {
		t.Fatalf("写入损坏文件失败: %v", err)
	}

	got, err := s.LoadSettings()
	if err != nil {
		t.Fatalf("损坏文件应自动恢复而非报错: %v", err)
	}
	if got != models.DefaultSettings() {
		t.Fatalf("损坏后应重置为默认设置,got %#v", got)
	}
	backups, _ := filepath.Glob(filepath.Join(s.RootDir(), settingsFileName+".corrupt-*"))
	if len(backups) != 1 {
		t.Fatalf("应生成 1 个备份文件,got %v", backups)
	}
}

// TestFilesIndependent 按域拆分:保存某一域不会创建另一域的文件。
func TestFilesIndependent(t *testing.T) {
	// 保存 settings 不应创建 routes.json
	s1 := newTestStore(t)
	if err := s1.SaveSettings(models.DefaultSettings()); err != nil {
		t.Fatalf("SaveSettings() 出错: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s1.RootDir(), routesFileName)); !os.IsNotExist(err) {
		t.Fatalf("保存 settings 不应创建 routes.json")
	}

	// 保存 routes 不应创建 settings.json
	s2 := newTestStore(t)
	if err := s2.SaveRoutes([]models.RouteRule{}); err != nil {
		t.Fatalf("SaveRoutes() 出错: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s2.RootDir(), settingsFileName)); !os.IsNotExist(err) {
		t.Fatalf("保存 routes 不应创建 settings.json")
	}
}

// TestLoadCreatesMissingDir 数据目录不存在时自动创建。
func TestLoadCreatesMissingDir(t *testing.T) {
	root := filepath.Join(t.TempDir(), "nested", "NetRoute-Manager")
	s := NewWithDir(root)

	if _, err := s.LoadRoutes(); err != nil {
		t.Fatalf("LoadRoutes() 应自动创建目录: %v", err)
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		t.Fatalf("数据目录应已创建: %v", err)
	}
}

// TestSettingsSanitizeInvalidFields 非法字段自动回退默认值。
func TestSettingsSanitizeInvalidFields(t *testing.T) {
	s := newTestStore(t)
	path := filepath.Join(s.RootDir(), settingsFileName)
	bad := `{"primaryDns":"","secondaryDns":"","queryInterval":-5,"enableIpv6":false,"autoStart":true,"minToTray":true,"dnsMode":"QUIC"}`
	if err := os.WriteFile(path, []byte(bad), 0o644); err != nil {
		t.Fatalf("写入设置文件失败: %v", err)
	}

	got, err := s.LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings() 出错: %v", err)
	}
	if got != models.DefaultSettings() {
		t.Fatalf("非法字段应回退默认值: got %#v want %#v", got, models.DefaultSettings())
	}
}

// TestDefaultDataDir 默认数据目录以 NetRoute-Manager 结尾且非空。
func TestDefaultDataDir(t *testing.T) {
	dir, err := DefaultDataDir()
	if err != nil {
		t.Fatalf("DefaultDataDir() 出错: %v", err)
	}
	if dir == "" || !strings.HasSuffix(dir, "NetRoute-Manager") {
		t.Fatalf("默认数据目录异常: %q", dir)
	}
}
