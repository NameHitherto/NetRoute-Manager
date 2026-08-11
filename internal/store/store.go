// Package store 提供路由规则与全局设置的 JSON 本地持久化。
//
// 数据保存在当前用户文档目录下的 NetRoute-Manager 文件夹中,
// 按域拆分为 routes.json 与 settings.json 两个结构化 JSON 文件。
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"NetRoute-Manager/internal/models"
)

const (
	// routesFileName 路由规则数据文件名。
	routesFileName = "routes.json"
	// settingsFileName 全局设置数据文件名。
	settingsFileName = "settings.json"
)

// Store 负责路由规则与全局设置的本地持久化。
// 内部使用互斥锁串行化所有读写,保证并发安全。
type Store struct {
	mu      sync.Mutex
	rootDir string
}

// New 创建使用系统默认数据目录(用户文档目录/NetRoute-Manager)的 Store。
func New() (*Store, error) {
	dir, err := DefaultDataDir()
	if err != nil {
		return nil, err
	}
	return NewWithDir(dir), nil
}

// NewOrPanic 同 New,但创建失败时直接 panic。
// 供应用装配使用:数据目录解析失败属于无法恢复的启动错误。
func NewOrPanic() *Store {
	s, err := New()
	if err != nil {
		panic(err)
	}
	return s
}

// NewWithDir 创建使用指定根目录的 Store,便于测试注入临时目录。
func NewWithDir(dir string) *Store {
	return &Store{rootDir: dir}
}

// RootDir 返回数据根目录。
func (s *Store) RootDir() string {
	return s.rootDir
}

// DefaultDataDir 返回默认数据目录:用户文档目录下的 NetRoute-Manager。
func DefaultDataDir() (string, error) {
	docs, err := documentsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(docs, "NetRoute-Manager"), nil
}

// LoadRoutes 加载全部路由规则。
// 文件不存在时自动创建空的 routes.json 并返回空切片;
// 文件损坏时自动备份为 .corrupt-<时间戳> 后重置为空列表继续运行。
func (s *Store) LoadRoutes() ([]models.RouteRule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureDir(); err != nil {
		return nil, err
	}
	path := filepath.Join(s.rootDir, routesFileName)
	var routes []models.RouteRule
	ok, err := s.load(path, &routes)
	if err != nil {
		return nil, err
	}
	if !ok {
		routes = []models.RouteRule{}
		if err := s.writeJSON(path, routes); err != nil {
			return nil, err
		}
	}
	return routes, nil
}

// SaveRoutes 持久化全部路由规则。
func (s *Store) SaveRoutes(routes []models.RouteRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureDir(); err != nil {
		return err
	}
	return s.writeJSON(filepath.Join(s.rootDir, routesFileName), routes)
}

// LoadSettings 加载全局设置。
// 文件不存在时写入默认设置并返回;文件损坏时备份后重置为默认设置;
// 字段不合法时(如非法 dnsMode、非正 queryInterval)自动回退默认值。
func (s *Store) LoadSettings() (models.AppSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureDir(); err != nil {
		return models.DefaultSettings(), err
	}
	path := filepath.Join(s.rootDir, settingsFileName)
	var settings models.AppSettings
	ok, err := s.load(path, &settings)
	if err != nil {
		return models.DefaultSettings(), err
	}
	if !ok {
		settings = models.DefaultSettings()
		if err := s.writeJSON(path, settings); err != nil {
			return models.DefaultSettings(), err
		}
		return settings, nil
	}
	// 兜底:非法字段回退默认值并写回,保证返回的数据始终可用。
	defaults := models.DefaultSettings()
	if !settings.DNSMode.Valid() {
		settings.DNSMode = defaults.DNSMode
	}
	if settings.QueryInterval <= 0 {
		settings.QueryInterval = defaults.QueryInterval
	}
	if settings.PrimaryDNS == "" {
		settings.PrimaryDNS = defaults.PrimaryDNS
	}
	if settings.SecondaryDNS == "" {
		settings.SecondaryDNS = defaults.SecondaryDNS
	}
	_ = s.writeJSON(path, settings) // 写回失败不影响读取结果
	return settings, nil
}

// SaveSettings 持久化全局设置。
func (s *Store) SaveSettings(settings models.AppSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureDir(); err != nil {
		return err
	}
	return s.writeJSON(filepath.Join(s.rootDir, settingsFileName), settings)
}

// ensureDir 确保数据根目录存在,不存在则创建。
func (s *Store) ensureDir() error {
	return os.MkdirAll(s.rootDir, 0o755)
}

// load 读取并解析 JSON 文件到 v。
// 返回 ok=false 表示文件不存在(或损坏且已备份为 .corrupt-*),
// 此时调用方应以默认数据重建文件。
func (s *Store) load(path string, v any) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if err := json.Unmarshal(data, v); err != nil {
		if berr := s.backupCorrupt(path); berr != nil {
			return false, fmt.Errorf("解析 %s 失败(%v);备份损坏文件失败: %w", path, err, berr)
		}
		return false, nil
	}
	return true, nil
}

// backupCorrupt 将损坏的 JSON 文件重命名为 <原名>.corrupt-<时间戳> 以保留现场。
func (s *Store) backupCorrupt(path string) error {
	backup := fmt.Sprintf("%s.corrupt-%s", path, time.Now().Format("20060102-150405.000"))
	return os.Rename(path, backup)
}

// writeJSON 原子写入:先写入同目录临时文件,再重命名替换目标文件,
// 避免写入中途崩溃导致目标文件损坏。
func (s *Store) writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(s.rootDir, ".tmp-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // 出错时清理临时文件

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
