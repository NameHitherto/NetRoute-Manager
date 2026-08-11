package main

import (
	"NetRoute-Manager/internal/models"
)

// StartService 启动核心路由服务:对勾选的规则做 DNS 解析,
// 并将解析出的 IP 以临时主机路由绑定到指定网卡。
// 返回启动后的服务状态快照(含已解析结果)。
// 重复启动返回错误;DNS 服务器与解析间隔取自当前持久化设置。
func (a *App) StartService(nicId string, rules []models.RouteRule) (models.ServiceStartResult, error) {
	settings, err := a.store.LoadSettings()
	if err != nil {
		return models.ServiceStartResult{}, err
	}
	if err := a.engine.Start(nicId, rules, settings); err != nil {
		return models.ServiceStartResult{}, err
	}
	return a.engine.Status(), nil
}

// StopService 停止核心路由服务,并清理全部已添加的临时路由。
// 服务未运行时返回 nil(幂等)。
func (a *App) StopService() error {
	return a.engine.Stop()
}

// GetServiceStatus 返回当前服务状态快照,供前端轮询刷新实时解析结果。
func (a *App) GetServiceStatus() (models.ServiceStartResult, error) {
	return a.engine.Status(), nil
}
