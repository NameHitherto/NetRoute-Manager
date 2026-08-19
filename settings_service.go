package main

import (
	"errors"
	"strings"

	"NetRoute-Manager/internal/models"
	"NetRoute-Manager/internal/store"
)

// SettingsService 全局设置读写服务,暴露给前端绑定调用。
type SettingsService struct {
	store *store.Store
}

// NewSettingsService 创建设置服务。
func NewSettingsService(s *store.Store) *SettingsService {
	return &SettingsService{store: s}
}

// GetSettings 返回当前全局设置(持久化数据,缺失字段自动回退默认值)。
func (s *SettingsService) GetSettings() (models.AppSettings, error) {
	return s.store.LoadSettings()
}

// SaveSettings 校验并持久化全局设置。
func (s *SettingsService) SaveSettings(settings models.AppSettings) error {
	if err := validateSettings(settings); err != nil {
		return err
	}
	return s.store.SaveSettings(settings)
}

// validateSettings 校验全局设置:主备 DNS 必填、查询间隔为正数、dnsMode 合法。
func validateSettings(s models.AppSettings) error {
	if strings.TrimSpace(s.PrimaryDNS) == "" || strings.TrimSpace(s.SecondaryDNS) == "" {
		return errors.New("主备 DNS 地址不能为空")
	}
	if s.QueryInterval <= 0 {
		return errors.New("查询间隔必须为正数")
	}
	if !s.DNSMode.Valid() {
		return errors.New("无效的 DNS 解析模式")
	}
	return nil
}
