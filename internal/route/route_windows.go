//go:build windows

package route

import (
	"bytes"
	"fmt"
	"net/netip"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// GAA_FLAG_INCLUDE_ALL_INTERFACES 枚举包括禁用/未连接在内的全部接口。
const gaaFlagIncludeAllInterfaces = 0x00000080

// 路由 API 返回的错误码。
// 注意:Create/DeleteIpForwardEntry(2) 返回 iphlpapi 专属错误码范围(5001+),
// ERROR_ROUTE_ALREADY_EXISTS=5010、ERROR_ROUTE_NOT_FOUND=5011,
// 非通用 winerror 的 ERROR_OBJECT_ALREADY_EXISTS(315)/ERROR_ALREADY_EXISTS(183)。
const (
	errorRouteAlreadyExists = 0x1392 // ERROR_ROUTE_ALREADY_EXISTS (5010):路由已存在
	errorRouteNotFound      = 0x1393 // ERROR_ROUTE_NOT_FOUND (5011):路由不存在
	errorNotFound           = 0x490  // ERROR_NOT_FOUND (1168):对象不存在
)

// 最大有效期限/优先期限:临时路由的生存期为无限长(重启系统后消失)。
const infiniteLifetime = 0xFFFFFFFF

// IP 族常量(af_inet 与 sockaddrFor 中的 family 保持一致)。
const (
	familyIPv4 = 2
	familyIPv6 = 23
)

var (
	iphlpapi = windows.NewLazySystemDLL("iphlpapi.dll")

	procCreateIpForwardEntry  = iphlpapi.NewProc("CreateIpForwardEntry")
	procDeleteIpForwardEntry  = iphlpapi.NewProc("DeleteIpForwardEntry")
	procCreateIpForwardEntry2 = iphlpapi.NewProc("CreateIpForwardEntry2")
	procDeleteIpForwardEntry2 = iphlpapi.NewProc("DeleteIpForwardEntry2")
	procGetIpForwardTable     = iphlpapi.NewProc("GetIpForwardTable")
	procGetIpForwardTable2    = iphlpapi.NewProc("GetIpForwardTable2")
	procFreeMibTable          = iphlpapi.NewProc("FreeMibTable")
	procGetIpInterfaceEntry   = iphlpapi.NewProc("GetIpInterfaceEntry")
)

// mibIPForwardTable MIB_IPFORWARDTABLE:IPv4 路由表(条目数组)。
type mibIPForwardTable struct {
	NumEntries uint32
	Table      [1]mibIPForwardRow
}

// mibIPInterfaceRow MIB_IPINTERFACE_ROW(接口信息,224 字节)。
// 仅命名本服务所需字段:Family/InterfaceLuid/InterfaceIndex/
// UseAutomaticMetric/Metric,其余以对齐填充占位(偏移与 netioapi.h 一致)。
type mibIPInterfaceRow struct {
	Family             uint16    // 0-1
	_pad0              [6]byte   // 2-7
	InterfaceLuid      [8]byte   // 8-15
	InterfaceIndex     uint32    // 16-19
	_pad1              [24]byte  // 20-43
	UseAutomaticMetric uint8     // 44
	_pad2              [103]byte // 45-147
	Metric             uint32    // 148-151
	_pad3              [72]byte  // 152-223
}

// winManager 基于 Windows IP Helper API 的路由管理器。
type winManager struct{}

func newManager() Manager { return &winManager{} }

// nicInfo 网卡的接口索引与网关信息。
type nicInfo struct {
	ipv4   uint32
	ipv6   uint32
	ipv4GW netip.Addr
	ipv6GW netip.Addr
	operUp bool
}

// AddHostRoute 实现 Manager。
func (m *winManager) AddHostRoute(ip netip.Addr, nicID string) (string, error) {
	nic, err := m.nicInfo(nicID)
	if err != nil {
		return "", err
	}
	if ip.Is4() {
		return m.addV4(ip, nic)
	}
	return m.addV6(ip, nic)
}

// DeleteHostRoute 实现 Manager。
func (m *winManager) DeleteHostRoute(ip netip.Addr, nicID string, snapshot string) error {
	nic, err := m.nicInfo(nicID)
	if err != nil {
		return err
	}
	if ip.Is4() {
		return m.deleteV4(ip, nic, snapshot)
	}
	return m.deleteV6(ip, nic, snapshot)
}

// Validate 实现 Manager:网卡存在且具备 IPv4 网关。
func (m *winManager) Validate(nicID string) error {
	nic, err := m.nicInfo(nicID)
	if err != nil {
		return err
	}
	if !nic.ipv4GW.IsValid() {
		return fmt.Errorf("网卡 %s 无 IPv4 网关,无法承担网关转发出口", nicID)
	}
	return nil
}

// NicUp 实现 Manager:返回网卡当前连接状态。
func (m *winManager) NicUp(nicID string) (bool, error) {
	nic, err := m.nicInfo(nicID)
	if err != nil {
		return false, err
	}
	return nic.operUp, nil
}

// nicInfo 根据网卡 GUID(AdapterName)实时枚举接口索引与网关。
// 每次操作实时枚举,避免网卡热插拔导致信息过期;缓冲区不足时扩容重试。
func (m *winManager) nicInfo(nicID string) (nicInfo, error) {
	var size uint32
	for {
		err := windows.GetAdaptersAddresses(
			windows.AF_UNSPEC, gaaFlagIncludeAllInterfaces, 0, nil, &size,
		)
		if err != nil && err != windows.ERROR_BUFFER_OVERFLOW {
			return nicInfo{}, fmt.Errorf("GetAdaptersAddresses 获取缓冲区大小失败: %w", err)
		}
		if size == 0 {
			return nicInfo{}, fmt.Errorf("未找到网卡 %s", nicID)
		}

		buf := make([]byte, size)
		first := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0]))
		err = windows.GetAdaptersAddresses(
			windows.AF_UNSPEC, gaaFlagIncludeAllInterfaces, 0, first, &size,
		)
		if err == nil {
			for addr := first; addr != nil; addr = addr.Next {
				if bytePtrToString(addr.AdapterName) != nicID {
					continue
				}
				v4gw, v6gw := gatewayAddrs(addr.FirstGatewayAddress)
				return nicInfo{
					ipv4:   addr.IfIndex,
					ipv6:   addr.Ipv6IfIndex,
					ipv4GW: v4gw,
					ipv6GW: v6gw,
					operUp: addr.OperStatus == windows.IfOperStatusUp,
				}, nil
			}
			return nicInfo{}, fmt.Errorf("未找到网卡 %s", nicID)
		}
		if err != windows.ERROR_BUFFER_OVERFLOW {
			return nicInfo{}, fmt.Errorf("GetAdaptersAddresses 枚举网卡失败: %w", err)
		}
		// 缓冲区不足,size 已更新,扩容后重试
	}
}

// gatewayAddrs 从 FirstGatewayAddress 链表中提取 IPv4 与 IPv6 网关各一个。
func gatewayAddrs(first *windows.IpAdapterGatewayAddress) (v4, v6 netip.Addr) {
	for ga := first; ga != nil; ga = ga.Next {
		ip := ga.Address.IP()
		if ip == nil {
			continue
		}
		a, ok := netip.AddrFromSlice(ip)
		if !ok {
			continue
		}
		a = a.Unmap()
		if a.Is4() && !v4.IsValid() {
			v4 = a
		} else if a.Is6() && !v6.IsValid() {
			v6 = a
		}
		if v4.IsValid() && v6.IsValid() {
			break
		}
	}
	return v4, v6
}

// addV4 添加 IPv4 网关转发主机路由(/32,next-hop=网关)。
// 冲突检测:目标已存在且指向其他网卡/网关时报 ErrRouteConflict。
func (m *winManager) addV4(ip netip.Addr, nic nicInfo) (string, error) {
	if !nic.ipv4GW.IsValid() {
		return "", fmt.Errorf("网卡无 IPv4 网关,无法添加网关转发路由 %s", ip)
	}
	dest, err := ipv4AsDWORD(ip)
	if err != nil {
		return "", err
	}
	gw, err := ipv4AsDWORD(nic.ipv4GW)
	if err != nil {
		return "", err
	}

	// P0-3 冲突检测:已存在且指向本网卡网关 -> 幂等成功(直接返回快照);
	// 指向他处 -> ErrRouteConflict。
	exists, err := m.checkV4Conflict(ip, nic.ipv4, gw)
	if err != nil {
		return "", err
	}
	if exists {
		// 已存在且一致:幂等成功。快照用真实接口 metric 记录,保证删除精确匹配。
		metric, merr := m.interfaceMetric(nic.ipv4)
		if merr != nil {
			metric = 0
		}
		return marshalSnapshot(routeSnapshot{
			Family: familyIPv4, IfIndex: nic.ipv4, Metric: metric,
			NextHop: nic.ipv4GW.String(), Type: mibIPRouteTypeIndirect, Proto: mibIPProtoNetMgmt,
		}), nil
	}

	metric, err := m.interfaceMetric(nic.ipv4)
	if err != nil {
		return "", fmt.Errorf("添加 IPv4 路由 %s: %w", ip, err)
	}
	row := mibIPForwardRow{
		Dest:    dest,
		Mask:    ipv4HostMask,
		NextHop: gw,
		IfIndex: nic.ipv4,
		Type:    mibIPRouteTypeIndirect,
		Proto:   mibIPProtoNetMgmt,
		Metric1: metric,
	}
	if e := callEntry(procCreateIpForwardEntry, unsafe.Pointer(&row), ip.String()); e != nil {
		return "", e
	}
	// exists=true 已由冲突检测返回(无错误)时走到这里,同样返回快照(幂等)。
	return marshalSnapshot(routeSnapshot{
		Family: familyIPv4, IfIndex: nic.ipv4, Metric: metric,
		NextHop: nic.ipv4GW.String(), Type: mibIPRouteTypeIndirect, Proto: mibIPProtoNetMgmt,
	}), nil
}

// deleteV4 删除 IPv4 网关转发主机路由。
// 优先用创建快照精确删除;快照缺失/失效时按当前状态重新构造。
func (m *winManager) deleteV4(ip netip.Addr, nic nicInfo, snapshot string) error {
	dest, err := ipv4AsDWORD(ip)
	if err != nil {
		return err
	}
	if snap, err := unmarshalSnapshot(snapshot); err == nil && snap.Family == familyIPv4 {
		nh, ok := netip.ParseAddr(snap.NextHop)
		var gw uint32
		if ok == nil {
			gw, _ = ipv4AsDWORD(nh)
		}
		row := mibIPForwardRow{
			Dest:    dest,
			Mask:    ipv4HostMask,
			NextHop: gw,
			IfIndex: snap.IfIndex,
			Type:    snap.Type,
			Proto:   snap.Proto,
			Metric1: snap.Metric,
		}
		if ok == nil {
			if e := callEntry(procDeleteIpForwardEntry, unsafe.Pointer(&row), ip.String()); e == nil {
				return nil
			}
			// 快照删除失败(如接口已断开),回退按当前状态重试
		}
	}
	return m.deleteV4ByState(ip, dest, nic)
}

// deleteV4ByState 按当前网卡状态构造删除(回退路径)。
func (m *winManager) deleteV4ByState(ip netip.Addr, dest uint32, nic nicInfo) error {
	gw := uint32(0)
	if nic.ipv4GW.IsValid() {
		gw, _ = ipv4AsDWORD(nic.ipv4GW)
	}
	metric, err := m.interfaceMetric(nic.ipv4)
	if err != nil {
		// 接口可能已断开(路由随接口消失),以 0 构造删除,匹配不到时幂等成功。
		metric = 0
	}
	row := mibIPForwardRow{
		Dest:    dest,
		Mask:    ipv4HostMask,
		NextHop: gw,
		IfIndex: nic.ipv4,
		Type:    mibIPRouteTypeIndirect,
		Proto:   mibIPProtoNetMgmt,
		Metric1: metric,
	}
	return callEntry(procDeleteIpForwardEntry, unsafe.Pointer(&row), ip.String())
}

// addV6 添加 IPv6 网关转发主机路由(/128,next-hop=IPv6 网关)。
func (m *winManager) addV6(ip netip.Addr, nic nicInfo) (string, error) {
	if nic.ipv6 == 0 {
		return "", fmt.Errorf("网卡未启用 IPv6,无法添加 IPv6 主机路由 %s", ip)
	}
	if !nic.ipv6GW.IsValid() {
		return "", fmt.Errorf("网卡无 IPv6 网关,无法添加 IPv6 网关转发路由 %s", ip)
	}

	// P0-3 冲突检测:已存在且指向本网卡网关 -> 幂等成功;指向他处 -> 报错
	exists, err := m.checkV6Conflict(ip, nic.ipv6, nic.ipv6GW)
	if err != nil {
		return "", err
	}
	if exists {
		return marshalSnapshot(routeSnapshot{
			Family: familyIPv6, IfIndex: nic.ipv6, Metric: 0,
			NextHop: nic.ipv6GW.String(), Type: 0, Proto: mibIPProtoNetMgmt,
		}), nil
	}

	row := windows.MibIpForwardRow2{
		InterfaceIndex:    nic.ipv6,
		ValidLifetime:     infiniteLifetime,
		PreferredLifetime: infiniteLifetime,
		Metric:            0, // 0 = 自动 metric(使用接口 metric)
		Protocol:          mibIPProtoNetMgmt,
	}
	nh := sockaddrFor(nic.ipv6GW)
	copy((*[28]byte)(unsafe.Pointer(&row.NextHop))[:], nh[:])
	prefix := sockaddrFor(ip)
	copy((*[28]byte)(unsafe.Pointer(&row.DestinationPrefix.Prefix))[:], prefix[:])
	row.DestinationPrefix.PrefixLength = 128 // 前缀长度 /128
	if e := callEntry(procCreateIpForwardEntry2, unsafe.Pointer(&row), ip.String()); e != nil {
		return "", e
	}
	return marshalSnapshot(routeSnapshot{
		Family: familyIPv6, IfIndex: nic.ipv6, Metric: 0,
		NextHop: nic.ipv6GW.String(), Type: 0, Proto: mibIPProtoNetMgmt,
	}), nil
}

// deleteV6 删除 IPv6 网关转发主机路由。
func (m *winManager) deleteV6(ip netip.Addr, nic nicInfo, snapshot string) error {
	if nic.ipv6 == 0 {
		// 网卡已无 IPv6 索引,路由必然不存在,视为清理完成。
		return nil
	}
	if snap, err := unmarshalSnapshot(snapshot); err == nil && snap.Family == familyIPv6 {
		nh, ok := netip.ParseAddr(snap.NextHop)
		row := windows.MibIpForwardRow2{
			InterfaceIndex: snap.IfIndex,
			Metric:         snap.Metric,
			Protocol:       snap.Proto,
		}
		prefix := sockaddrFor(ip)
		copy((*[28]byte)(unsafe.Pointer(&row.DestinationPrefix.Prefix))[:], prefix[:])
		row.DestinationPrefix.PrefixLength = 128
		if ok == nil {
			nhSA := sockaddrFor(nh)
			copy((*[28]byte)(unsafe.Pointer(&row.NextHop))[:], nhSA[:])
			if e := callEntry(procDeleteIpForwardEntry2, unsafe.Pointer(&row), ip.String()); e == nil {
				return nil
			}
		}
	}
	// 回退:按当前状态构造
	row := windows.MibIpForwardRow2{
		InterfaceIndex: nic.ipv6,
		Metric:         0,
		Protocol:       mibIPProtoNetMgmt,
	}
	nh := sockaddrFor(nic.ipv6GW)
	copy((*[28]byte)(unsafe.Pointer(&row.NextHop))[:], nh[:])
	prefix := sockaddrFor(ip)
	copy((*[28]byte)(unsafe.Pointer(&row.DestinationPrefix.Prefix))[:], prefix[:])
	row.DestinationPrefix.PrefixLength = 128
	return callEntry(procDeleteIpForwardEntry2, unsafe.Pointer(&row), ip.String())
}

// interfaceMetric 查询指定接口的 IPv4 metric。
// 首选 GetIpInterfaceEntry 读取接口自身 Metric(仅当 UseAutomaticMetric=0 时可用);
// 否则回退扫描 IPv4 路由表取该接口现有条目的最小 Metric1
// (接口自身的默认/直连路由 metric 恒不小于接口 metric)。
func (m *winManager) interfaceMetric(ifIndex uint32) (uint32, error) {
	row := mibIPInterfaceRow{Family: familyIPv4, InterfaceIndex: ifIndex}
	if ret, _, _ := procGetIpInterfaceEntry.Call(uintptr(unsafe.Pointer(&row))); ret == 0 && row.UseAutomaticMetric == 0 {
		return row.Metric, nil
	}
	return m.interfaceMetricFromTable(ifIndex)
}

// interfaceMetricFromTable 扫描 IPv4 路由表取该接口最小 Metric1(回退路径)。
func (m *winManager) interfaceMetricFromTable(ifIndex uint32) (uint32, error) {
	var size uint32
	for {
		ret, _, _ := procGetIpForwardTable.Call(0, uintptr(unsafe.Pointer(&size)), 0)
		if ret == 0 {
			// NULL 指针调用即成功:表为空,不存在该接口路由。
			return 0, fmt.Errorf("接口 %d 无路由条目,无法确定接口 metric", ifIndex)
		}
		if ret != uintptr(windows.ERROR_INSUFFICIENT_BUFFER) {
			return 0, fmt.Errorf("GetIpForwardTable 获取缓冲区大小失败: %v", syscall.Errno(ret))
		}
		buf := make([]byte, size)
		table := (*mibIPForwardTable)(unsafe.Pointer(&buf[0]))
		ret, _, _ = procGetIpForwardTable.Call(
			uintptr(unsafe.Pointer(table)), uintptr(unsafe.Pointer(&size)), 0,
		)
		if ret == 0 {
			rows := unsafe.Slice(&table.Table[0], table.NumEntries)
			var minM uint32
			found := false
			for _, r := range rows {
				if r.IfIndex != ifIndex {
					continue
				}
				if !found || r.Metric1 < minM {
					minM = r.Metric1
					found = true
				}
			}
			if !found {
				return 0, fmt.Errorf("接口 %d 无路由条目,无法确定接口 metric", ifIndex)
			}
			return minM, nil
		}
		if ret != uintptr(windows.ERROR_INSUFFICIENT_BUFFER) {
			return 0, fmt.Errorf("GetIpForwardTable 枚举路由失败: %v", syscall.Errno(ret))
		}
		// 缓冲区不足,size 已更新,扩容后重试
	}
}

// checkV4Conflict 检查目标 /32 是否已存在于 IPv4 路由表:
// 不存在 -> (false, nil);存在且 IfIndex+NextHop 与本次一致 -> (true, nil);
// 存在但指向其他网卡/网关 -> (true, ErrRouteConflict)。
func (m *winManager) checkV4Conflict(ip netip.Addr, ifIndex, gw uint32) (bool, error) {
	dest, err := ipv4AsDWORD(ip)
	if err != nil {
		return false, err
	}
	var size uint32
	ret, _, _ := procGetIpForwardTable.Call(0, uintptr(unsafe.Pointer(&size)), 0)
	if ret != uintptr(windows.ERROR_INSUFFICIENT_BUFFER) {
		if ret == 0 {
			return false, nil // 表为空
		}
		return false, fmt.Errorf("GetIpForwardTable 获取缓冲区大小失败: %v", syscall.Errno(ret))
	}
	if size == 0 {
		return false, nil // 空表
	}
	buf := make([]byte, size)
	table := (*mibIPForwardTable)(unsafe.Pointer(&buf[0]))
	ret, _, _ = procGetIpForwardTable.Call(
		uintptr(unsafe.Pointer(table)), uintptr(unsafe.Pointer(&size)), 0,
	)
	if ret != 0 {
		if ret == uintptr(windows.ERROR_INSUFFICIENT_BUFFER) {
			return false, fmt.Errorf("GetIpForwardTable 缓冲区不足且重试失败")
		}
		return false, fmt.Errorf("GetIpForwardTable 枚举路由失败: %v", syscall.Errno(ret))
	}
	rows := unsafe.Slice(&table.Table[0], table.NumEntries)
	for _, r := range rows {
		if r.Dest != dest || r.Mask != ipv4HostMask {
			continue
		}
		if r.IfIndex == ifIndex && r.NextHop == gw {
			return true, nil
		}
		return true, ErrRouteConflict
	}
	return false, nil
}

// checkV6Conflict 检查目标 /128 是否已存在于 IPv6 路由表(GetIpForwardTable2)。
func (m *winManager) checkV6Conflict(ip netip.Addr, ifIndex uint32, gw netip.Addr) (bool, error) {
	var tbl *windows.MibIpForwardTable2
	ret, _, _ := procGetIpForwardTable2.Call(
		uintptr(windows.AF_INET6), uintptr(unsafe.Pointer(&tbl)),
	)
	if ret != 0 {
		return false, fmt.Errorf("GetIpForwardTable2 枚举 IPv6 路由失败: %v", syscall.Errno(ret))
	}
	defer procFreeMibTable.Call(uintptr(unsafe.Pointer(tbl)))

	// 目标前缀 = sockaddr(28 字节) + 前缀长度 128
	var want [29]byte
	sa := sockaddrFor(ip)
	copy(want[:28], sa[:])
	want[28] = 128
	gwSA := sockaddrFor(gw)
	for _, r := range tbl.Rows() {
		dp := (*[29]byte)(unsafe.Pointer(&r.DestinationPrefix))
		if !bytes.Equal(dp[:], want[:]) {
			continue
		}
		nh := (*[28]byte)(unsafe.Pointer(&r.NextHop))
		if r.InterfaceIndex == ifIndex && sameSockaddrInetExceptScopeID(*nh, gwSA) {
			return true, nil
		}
		return true, ErrRouteConflict
	}
	return false, nil
}

// callEntry 调用 Create/DeleteIpForwardEntry(2) 并归一化错误:
// 路由已存在(添加)与路由不存在(删除)均视为幂等成功。
func callEntry(proc *windows.LazyProc, row unsafe.Pointer, ip string) error {
	ret, _, _ := proc.Call(uintptr(row))
	if ret == 0 {
		return nil
	}
	errno := syscall.Errno(ret)
	switch errno {
	// iphlpapi 专属码(5001+)
	case errorRouteAlreadyExists, errorRouteNotFound:
		return nil
	// 通用码兜底:不同 Windows 版本/API 可能返回这些
	case 183, 315, errorNotFound: // ERROR_ALREADY_EXISTS / ERROR_OBJECT_ALREADY_EXISTS / ERROR_NOT_FOUND
		return nil
	}
	return fmt.Errorf("%s 操作路由 %s 失败: %w", proc.Name, ip, errno)
}

// bytePtrToString 将 NUL 结尾的 ANSI 字节串(*byte)转换为 string。
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
