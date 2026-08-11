package main

import (
	"errors"
	"strings"

	"NetRoute-Manager/internal/models"
)

// GetSettings 返回当前全局设置(持久化数据,缺失字段自动回退默认值)。
func (a *App) GetSettings() (models.AppSettings, error) {
	return a.store.LoadSettings()
}

// SaveSettings 校验并持久化全局设置。
func (a *App) SaveSettings(settings models.AppSettings) error {
	if err := validateSettings(settings); err != nil {
		return err
	}
	return a.store.SaveSettings(settings)
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
