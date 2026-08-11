package route

import (
	"errors"
	"net/netip"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestSockaddrForIPv4(t *testing.T) {
	sa := sockaddrFor(netip.MustParseAddr("1.2.3.4"))
	if sa[0] != 2 || sa[1] != 0 { // AF_INET
		t.Fatalf("family 应为 AF_INET(2): %v", sa[:2])
	}
	if sa[2] != 0 || sa[3] != 0 { // port
		t.Fatalf("port 应为 0: %v", sa[2:4])
	}
	if got := sa[4:8]; !(got[0] == 1 && got[1] == 2 && got[2] == 3 && got[3] == 4) {
		t.Fatalf("地址字节不符: %v", got)
	}
	for i := 8; i < len(sa); i++ {
		if sa[i] != 0 {
			t.Fatalf("偏移 %d 应为 0: %v", i, sa[i])
		}
	}
}

func TestSockaddrForIPv6(t *testing.T) {
	sa := sockaddrFor(netip.MustParseAddr("2001:db8::1"))
	if sa[0] != 23 || sa[1] != 0 { // AF_INET6
		t.Fatalf("family 应为 AF_INET6(23): %v", sa[:2])
	}
	want := [16]byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	for i := 0; i < 16; i++ {
		if sa[8+i] != want[i] {
			t.Fatalf("偏移 %d 地址字节不符: got %x want %x", 8+i, sa[8+i], want[i])
		}
	}
	for i := 24; i < len(sa); i++ {
		if sa[i] != 0 {
			t.Fatalf("偏移 %d 应为 0(scope id): %v", i, sa[i])
		}
	}
}

func TestIPv4AsDWORD(t *testing.T) {
	dw, err := ipv4AsDWORD(netip.MustParseAddr("1.2.3.4"))
	if err != nil {
		t.Fatalf("ipv4AsDWORD() 出错: %v", err)
	}
	// Windows 路由表 DWORD 为"内存字节序 = IP 顺序":1.2.3.4 -> 0x04030201
	if dw != 0x04030201 {
		t.Fatalf("DWORD 值不符: got %#x want 0x04030201", dw)
	}
	if _, err := ipv4AsDWORD(netip.MustParseAddr("::1")); err == nil {
		t.Fatal("IPv6 地址应返回错误")
	}
}

func TestSameSockaddrInetExceptScopeID(t *testing.T) {
	// 同地址同 scope_id 应等价(自身比较)
	a := sockaddrFor(netip.MustParseAddr("fe80::1"))
	if !sameSockaddrInetExceptScopeID(a, a) {
		t.Fatal("自身应等价")
	}
	// IPv6:同一地址、不同 scope_id 应视为等价(link-local 网关场景)
	b := a
	b[24], b[25], b[26], b[27] = 5, 0, 0, 0 // scope_id = 接口索引 5(little-endian)
	if !sameSockaddrInetExceptScopeID(a, b) {
		t.Fatal("IPv6 同地址不同 scope_id 应视为等价")
	}
	// IPv6:不同地址应不等
	c := sockaddrFor(netip.MustParseAddr("fe80::2"))
	if sameSockaddrInetExceptScopeID(a, c) {
		t.Fatal("IPv6 不同地址应不等价")
	}
	// IPv4:仅比较前 16 字节,填充位差异不影响
	d := sockaddrFor(netip.MustParseAddr("1.2.3.4"))
	e := d
	e[16] = 0xFF
	if !sameSockaddrInetExceptScopeID(d, e) {
		t.Fatal("IPv4 填充位差异应视为等价")
	}
	// 不同 family 应不等
	f := sockaddrFor(netip.MustParseAddr("2001:db8::1"))
	if sameSockaddrInetExceptScopeID(d, f) {
		t.Fatal("不同 family 应不等价")
	}
	// IPv4 不同地址应不等
	g := sockaddrFor(netip.MustParseAddr("1.2.3.5"))
	if sameSockaddrInetExceptScopeID(d, g) {
		t.Fatal("IPv4 不同地址应不等价")
	}
}

func TestRouteSnapshotRoundTrip(t *testing.T) {
	snap := routeSnapshot{
		Family: familyIPv4, IfIndex: 29, Metric: 25,
		NextHop: "10.136.1.1", Type: mibIPRouteTypeIndirect, Proto: mibIPProtoNetMgmt,
	}
	s := marshalSnapshot(snap)
	got, err := unmarshalSnapshot(s)
	if err != nil {
		t.Fatalf("unmarshalSnapshot() 出错: %v", err)
	}
	if got != snap {
		t.Fatalf("快照往返不一致: got %#v want %#v", got, snap)
	}
	if _, err := unmarshalSnapshot("not-json"); err == nil {
		t.Fatal("非法快照应返回错误")
	}
}

func TestErrRouteConflictDefined(t *testing.T) {
	if !errors.Is(ErrRouteConflict, ErrRouteConflict) {
		t.Fatal("ErrRouteConflict 应可被 errors.Is 匹配")
	}
}

// TestStructLayouts 校验 Windows 结构体布局与 SDK 定义一致,
// 防止平台迁移或字段调整破坏与 DLL 的 ABI 兼容。
func TestStructLayouts(t *testing.T) {
	if size := unsafe.Sizeof(mibIPForwardRow{}); size != 56 {
		t.Fatalf("mibIPForwardRow 大小应为 56,got %d", size)
	}
	// MIB_IPINTERFACE_ROW:224 字节,关键字段偏移(UseAutomaticMetric=44, Metric=148)。
	if size := unsafe.Sizeof(mibIPInterfaceRow{}); size != 224 {
		t.Fatalf("mibIPInterfaceRow 大小应为 224,got %d", size)
	}

	// MIB_IPFORWARD_ROW2 使用 x/sys 官方类型 windows.MibIpForwardRow2,
	// 布局须与 netioapi.h 一致:InterfaceLuid=0(NET_LUID 8 字节),
	// InterfaceIndex=8, DestinationPrefix=12, NextHop=44, SitePrefixLength=72,
	// ValidLifetime=76, PreferredLifetime=80, Metric=84, Protocol=88,
	// Loopback=92, Age=96, Origin=100, 总大小 104。
	if size := unsafe.Sizeof(windows.MibIpForwardRow2{}); size != 104 {
		t.Fatalf("windows.MibIpForwardRow2 大小应为 104,got %d", size)
	}
	row2Checks := []struct {
		name   string
		offset uintptr
	}{
		{"InterfaceIndex", unsafe.Offsetof(windows.MibIpForwardRow2{}.InterfaceIndex)},
		{"DestinationPrefix", unsafe.Offsetof(windows.MibIpForwardRow2{}.DestinationPrefix)},
		{"NextHop", unsafe.Offsetof(windows.MibIpForwardRow2{}.NextHop)},
		{"SitePrefixLength", unsafe.Offsetof(windows.MibIpForwardRow2{}.SitePrefixLength)},
		{"ValidLifetime", unsafe.Offsetof(windows.MibIpForwardRow2{}.ValidLifetime)},
		{"PreferredLifetime", unsafe.Offsetof(windows.MibIpForwardRow2{}.PreferredLifetime)},
		{"Metric", unsafe.Offsetof(windows.MibIpForwardRow2{}.Metric)},
		{"Protocol", unsafe.Offsetof(windows.MibIpForwardRow2{}.Protocol)},
		{"Loopback", unsafe.Offsetof(windows.MibIpForwardRow2{}.Loopback)},
		{"Age", unsafe.Offsetof(windows.MibIpForwardRow2{}.Age)},
		{"Origin", unsafe.Offsetof(windows.MibIpForwardRow2{}.Origin)},
	}
	row2Want := []uintptr{8, 12, 44, 72, 76, 80, 84, 88, 92, 96, 100}
	for i, c := range row2Checks {
		if c.offset != row2Want[i] {
			t.Fatalf("%s 偏移应为 %d,got %d", c.name, row2Want[i], c.offset)
		}
	}

	// MIB_IPINTERFACE_ROW 关键字段偏移(netioapi.h):
	// InterfaceLuid=8, InterfaceIndex=16, UseAutomaticMetric=44, Metric=148。
	interfaceChecks := []struct {
		name   string
		offset uintptr
	}{
		{"InterfaceLuid", unsafe.Offsetof(mibIPInterfaceRow{}.InterfaceLuid)},
		{"InterfaceIndex", unsafe.Offsetof(mibIPInterfaceRow{}.InterfaceIndex)},
		{"UseAutomaticMetric", unsafe.Offsetof(mibIPInterfaceRow{}.UseAutomaticMetric)},
		{"Metric", unsafe.Offsetof(mibIPInterfaceRow{}.Metric)},
	}
	interfaceWant := []uintptr{8, 16, 44, 148}
	for i, c := range interfaceChecks {
		if c.offset != interfaceWant[i] {
			t.Fatalf("%s 偏移应为 %d,got %d", c.name, interfaceWant[i], c.offset)
		}
	}
}
