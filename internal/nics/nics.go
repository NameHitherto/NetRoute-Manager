// Package nics 提供本机物理网卡识别。
//
// 仅识别真实物理有线/无线网卡,排除虚拟、隧道与软件适配器;
// 默认仅返回处于活动(连接)状态的物理网卡。
package nics

import (
	"strings"

	"NetRoute-Manager/internal/models"
)

// 网卡类型常量:与 Windows IP Helper 的 ifType 取值对应。
const (
	ifTypeEthernetCSMACD = 6   // 有线以太网
	ifTypeIEEE80211      = 71  // 无线 802.11
	ifTypeWWANPP         = 243 // 蜂窝网卡(PP 封装)
	ifTypeWWANPP2        = 244 // 蜂窝网卡(PP2 封装)
)

// virtualAdapterKeywords 虚拟/软件适配器名称关键词黑名单。
// 名称(含描述)命中任一关键词即视为非物理网卡,予以排除。
var virtualAdapterKeywords = []string{
	"virtual", "vmware", "hyper-v", "hyperv", "virtualbox", "wsl", "loopback",
	"bluetooth", "tap", "tun", "wan miniport", "wi-fi direct", "tailscale",
	"zerotier", "wireguard", "npcap", "nordvpn", "openvpn", "vpn",
	"microsoft kernel debug", "kernel debug", "usb ncm", "remote ndis",
	"vethernet", "default switch", // Hyper-V 虚拟交换机(vEthernet)
}

// classifyNicType 根据 ifType 映射为前端契约的 wired/wireless 类型。
func classifyNicType(ifType uint32) (models.NicType, bool) {
	switch ifType {
	case ifTypeEthernetCSMACD:
		return models.NicTypeWired, true
	case ifTypeIEEE80211:
		return models.NicTypeWireless, true
	default:
		return "", false
	}
}

// isPhysicalAdapterName 判断网卡名称/描述是否为物理网卡(非虚拟/软件)。
func isPhysicalAdapterName(name string) bool {
	if name == "" {
		return false
	}
	lower := strings.ToLower(name)
	for _, kw := range virtualAdapterKeywords {
		if strings.Contains(lower, kw) {
			return false
		}
	}
	return true
}

// classify 根据 ifType 与名称综合判断网卡是否为可用的物理网卡,
// 返回对应的 NicType(wired/wireless)。非有线/无线物理网卡返回 ok=false。
func classify(ifType uint32, name string) (models.NicType, bool) {
	nicType, ok := classifyNicType(ifType)
	if !ok {
		return "", false
	}
	if !isPhysicalAdapterName(name) {
		return "", false
	}
	return nicType, true
}

// DisplayName 生成类似 ipconfig 标题的网卡显示名:
// 有线 -> "Ethernet adapter <名称>",无线 -> "Wireless LAN adapter <名称>"。
// 用于 GUI 下拉框中展示,使网卡名更具辨识度。
func DisplayName(nicType models.NicType, name string) string {
	switch nicType {
	case models.NicTypeWired:
		return "Ethernet adapter " + name
	case models.NicTypeWireless:
		return "Wireless LAN adapter " + name
	default:
		return name
	}
}
