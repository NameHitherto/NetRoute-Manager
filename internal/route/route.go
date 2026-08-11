// Package route 提供 Windows 路由表管理:为指定 IP 添加/删除临时主机路由。
//
// 路由通过 IP Helper API(CreateIpForwardEntry(2)/DeleteIpForwardEntry(2))管理,
// 协议标记为 MIB_IPPROTO_NETMGMT,属临时路由(重启系统自动消失,非持久化变更);
// 采用网关转发语义:next-hop 为网卡网关(而非 on-link 直连),保证广播型物理网卡上
// 目标 IP 的流量经网关正确转发。创建时返回路由行快照,删除时复用快照精确匹配,
// 避免接口 metric 变化导致删除失配。
// 非 Windows 平台提供返回错误的 stub 实现。
package route

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
)

// MIB_IPPROTO_NETMGMT 标识由网络管理应用(本服务)添加的路由。
const mibIPProtoNetMgmt = 3

// mibIPRouteTypeIndirect 路由类型:网关转发(indirect)。
const mibIPRouteTypeIndirect = 4

// ipv4HostMask IPv4 主机路由掩码(/32)。
const ipv4HostMask = 0xFFFFFFFF

// ErrRouteConflict 表示目标 /32(/128) 已存在于路由表且指向其他网卡/网关。
// 此时创建将静默"成功"但流量不会引到目标网卡,故显式报错。
var ErrRouteConflict = errors.New("路由冲突:目标 IP 已存在指向其他网卡/网关的路由")

// Manager 路由管理器接口(平台无关)。
type Manager interface {
	// AddHostRoute 在指定网卡(网卡 GUID,即 AdapterName)上添加一条临时主机路由:
	// IPv4 /32、IPv6 /128,next-hop 为该网卡网关。
	// 返回创建快照(snapshot),供 DeleteHostRoute 精确删除。
	// 目标 IP 已存在且指向本网卡网关时幂等成功;指向其他网卡/网关时返回 ErrRouteConflict。
	AddHostRoute(ip netip.Addr, nicID string) (string, error)
	// DeleteHostRoute 删除指定网卡上的临时主机路由。
	// snapshot 为创建时快照(可为空,为空时按当前状态重新构造);
	// 目标路由不存在时视为成功(幂等清理)。
	DeleteHostRoute(ip netip.Addr, nicID string, snapshot string) error
	// Validate 校验网卡可用于本服务:存在、具备 IPv4 网关(网关转发前提)。
	Validate(nicID string) error
	// NicUp 报告网卡当前是否处于连接状态(OperStatus==Up)。
	NicUp(nicID string) (bool, error)
}

// NewManager 返回当前平台的 Manager 实现。
func NewManager() Manager {
	return newManager()
}

// routeSnapshot 路由行快照,持久化与精确删除用。
// 删除时复用创建时的 ifIndex/metric/next-hop,避免运行期接口 metric
// 变化导致 DeleteIpForwardEntry 匹配不到而残留。
type routeSnapshot struct {
	Family  int    `json:"family"` // 2=IPv4, 23=IPv6
	IfIndex uint32 `json:"ifIndex"`
	Metric  uint32 `json:"metric"`
	NextHop string `json:"nextHop"`
	Type    uint32 `json:"type"`
	Proto   uint32 `json:"proto"`
}

// marshalSnapshot 序列化快照(内部构造,不应失败)。
func marshalSnapshot(s routeSnapshot) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// unmarshalSnapshot 反序列化快照;非法输入返回错误(调用方回退按当前状态构造)。
func unmarshalSnapshot(s string) (routeSnapshot, error) {
	var snap routeSnapshot
	if err := json.Unmarshal([]byte(s), &snap); err != nil {
		return routeSnapshot{}, err
	}
	return snap, nil
}

// mibIPForwardRow MIB_IPFORWARDROW(IPv4 路由表项)。
// 与 Windows ipmib.h 布局一致:14 个 DWORD,共 56 字节。
type mibIPForwardRow struct {
	Dest, Mask, Policy, NextHop, IfIndex uint32
	Type, Proto, Age, NextHopAS          uint32
	Metric1, Metric2, Metric3            uint32
	Metric4, Metric5                     uint32
}

// sockaddrFor 将 IP 编码为 SOCKADDR_INET 的 28 字节布局:
// IPv4 使用前 16 字节的 sockaddr_in(family=AF_INET),IPv6 使用完整 sockaddr_in6。
// 供构造 MIB_IPFORWARD_ROW2 的 DestinationPrefix / NextHop 使用。
func sockaddrFor(ip netip.Addr) [28]byte {
	var sa [28]byte
	if ip.Is4() {
		b := ip.As4()
		sa[0], sa[1] = 2, 0 // AF_INET
		copy(sa[4:8], b[:])
		return sa
	}
	b := ip.As16()
	sa[0], sa[1] = 23, 0 // AF_INET6
	copy(sa[8:24], b[:])
	return sa
}

// sameSockaddrInetExceptScopeID 比较两个 28 字节 SOCKADDR_INET 是否等价,
// 忽略 IPv6 scope_id(最后 4 字节 sin6_scope_id)。
// 路由表(GetIpForwardTable2)返回的 link-local 下一跳会带 scope_id(接口索引),
// 而本地构造的 sockaddrFor 恒为 0,逐字节比较会把同一网关误判为冲突。
func sameSockaddrInetExceptScopeID(a, b [28]byte) bool {
	if a[0] != b[0] { // family(低字节)不同则不同
		return false
	}
	if a[0] == 2 { // AF_INET:sockaddr_in 有效内容为前 16 字节
		return bytes.Equal(a[:16], b[:16])
	}
	// AF_INET6:比较前 24 字节(family/port/flowinfo/addr),忽略 scope_id
	return bytes.Equal(a[:24], b[:24])
}

// ipv4AsDWORD 将 IPv4 地址转为 MIB_IPFORWARDROW 所需的 DWORD 值。
// Windows 路由表以"内存字节序 = IP 顺序"存储(见 GetIpForwardTable 返回),
// 即 DWORD 值为 IP 字节的逆序拼装,与 binary.LittleEndian 读取一致。
func ipv4AsDWORD(ip netip.Addr) (uint32, error) {
	if !ip.Is4() {
		return 0, fmt.Errorf("不是 IPv4 地址: %s", ip)
	}
	b := ip.As4()
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24, nil
}
