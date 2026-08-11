//go:build !windows

package nics

import (
	"NetRoute-Manager/internal/models"
)

// ActivePhysicalInterfaces 非 Windows 平台暂不支持网卡枚举,返回空列表。
func ActivePhysicalInterfaces() ([]models.NetworkInterface, error) {
	return []models.NetworkInterface{}, nil
}
