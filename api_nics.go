package main

import (
	"NetRoute-Manager/internal/models"
	"NetRoute-Manager/internal/nics"
)

// GetNetworkInterfaces 返回本机当前活动的物理网卡列表(有线/无线)。
// 非 Windows 平台返回空列表。
func (a *App) GetNetworkInterfaces() ([]models.NetworkInterface, error) {
	return nics.ActivePhysicalInterfaces()
}
