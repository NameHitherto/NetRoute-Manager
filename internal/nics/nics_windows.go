//go:build windows

package nics

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"

	"NetRoute-Manager/internal/models"
)

// GAA_FLAG_INCLUDE_ALL_INTERFACES 枚举包括禁用/未连接在内的全部接口。
const gaaFlagIncludeAllInterfaces = 0x00000080

// bytePtrToString 将 NUL 结尾的 ANSI 字节串(*byte)转换为 string。
// 上限取 AdapterName 的最大合法长度,避免越界读。
func bytePtrToString(p *byte) string {
	if p == nil {
		return ""
	}
	b := unsafe.Slice(p, windows.MAX_ADAPTER_NAME_LENGTH+4)
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return ""
}

// ActivePhysicalInterfaces 枚举本机当前活动的物理网卡(有线/无线)。
// 采用 GetAdaptersAddresses(IP Helper API):
//   - 仅保留 IF_TYPE_ETHERNET_CSMACD / IF_TYPE_IEEE80211 的真实物理网卡,
//     通过名称关键词黑名单排除虚拟/软件适配器;
//   - 仅返回 OperStatus==Up(已连接)的活动网卡;
//   - ID 使用稳定 GUID(AdapterName),Name 优先 FriendlyName 回退 Description。
func ActivePhysicalInterfaces() ([]models.NetworkInterface, error) {
	var size uint32
	// 第一次调用获取所需缓冲区大小
	err := windows.GetAdaptersAddresses(
		windows.AF_UNSPEC, gaaFlagIncludeAllInterfaces, 0, nil, &size,
	)
	if err != nil && err != windows.ERROR_BUFFER_OVERFLOW {
		return nil, fmt.Errorf("GetAdaptersAddresses 获取缓冲区大小失败: %w", err)
	}
	if size == 0 {
		return []models.NetworkInterface{}, nil
	}

	buf := make([]byte, size)
	first := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0]))
	err = windows.GetAdaptersAddresses(
		windows.AF_UNSPEC, gaaFlagIncludeAllInterfaces, 0, first, &size,
	)
	if err != nil {
		return nil, fmt.Errorf("GetAdaptersAddresses 枚举网卡失败: %w", err)
	}

	nics := make([]models.NetworkInterface, 0, 8)
	for addr := first; addr != nil; addr = addr.Next {
		// 仅活动(已连接)网卡
		if addr.OperStatus != windows.IfOperStatusUp {
			continue
		}

		// 名称:优先 FriendlyName,回退 Description;
		// 两者均需通过物理网卡黑名单检查(FriendlyName 本地化后
		// 可能丢失英文虚拟关键词,故同时核对 Description)。
		name := windows.UTF16PtrToString(addr.FriendlyName)
		desc := windows.UTF16PtrToString(addr.Description)
		if name == "" {
			name = desc
		}
		if name == "" {
			continue
		}

		nicType, ok := classify(addr.IfType, name)
		// 仅当 Description 非空时才做黑名单兜底检查,
		// 避免真实物理网卡因空描述被误排除。
		if !ok || (desc != "" && !isPhysicalAdapterName(desc)) {
			continue
		}

		nics = append(nics, models.NetworkInterface{
			ID:     bytePtrToString(addr.AdapterName),
			Name:   DisplayName(nicType, name), // 类似 ipconfig 标题的显示名
			Type:   nicType,
			Active: true, // 仅返回活动网卡,恒为 true
		})
	}
	return nics, nil
}
