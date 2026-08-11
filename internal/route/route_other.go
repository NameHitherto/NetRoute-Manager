//go:build !windows

package route

import (
	"errors"
	"net/netip"
)

// unsupportedManager 非 Windows 平台的占位实现,所有操作返回错误。
type unsupportedManager struct{}

func newManager() Manager { return &unsupportedManager{} }

func (m *unsupportedManager) AddHostRoute(netip.Addr, string) (string, error) {
	return "", errors.New("路由管理仅支持 Windows")
}

func (m *unsupportedManager) DeleteHostRoute(netip.Addr, string, string) error {
	return errors.New("路由管理仅支持 Windows")
}

func (m *unsupportedManager) Validate(string) error {
	return errors.New("路由管理仅支持 Windows")
}

func (m *unsupportedManager) NicUp(string) (bool, error) {
	return false, errors.New("路由管理仅支持 Windows")
}
