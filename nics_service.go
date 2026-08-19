package main

import (
	"NetRoute-Manager/internal/models"
	"NetRoute-Manager/internal/nics"
)

// NicsService 网卡查询服务,暴露给前端绑定调用。
type NicsService struct{}

// NewNicsService 创建网卡查询服务。
func NewNicsService() *NicsService {
	return &NicsService{}
}

// GetNetworkInterfaces 返回本机当前活动的物理网卡列表(有线/无线)。
// 非 Windows 平台返回空列表。
func (s *NicsService) GetNetworkInterfaces() ([]models.NetworkInterface, error) {
	return nics.ActivePhysicalInterfaces()
}
