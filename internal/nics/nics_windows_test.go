//go:build windows

package nics

import (
	"net/netip"
	"syscall"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

// rawSockaddrInet4 构造 AF_INET 的 RawSockaddrAny。
func rawSockaddrInet4(ip [4]byte) windows.RawSockaddrAny {
	var any windows.RawSockaddrAny
	any.Addr.Family = windows.AF_INET
	data := (*[14]byte)(unsafe.Pointer(&any.Addr.Data))
	// Data 自偏移 2 起,SockaddrInet4.Addr 在 any 偏移 4-7 => data[2:6]
	copy(data[2:6], ip[:])
	return any
}

// rawSockaddrInet6 构造 AF_INET6 的 RawSockaddrAny。
func rawSockaddrInet6(ip [16]byte) windows.RawSockaddrAny {
	var any windows.RawSockaddrAny
	any.Addr.Family = windows.AF_INET6
	data := (*[14]byte)(unsafe.Pointer(&any.Addr.Data))
	pad := (*[96]byte)(unsafe.Pointer(&any.Pad))
	// SockaddrInet6.Addr 在 any 偏移 8-23 => Data[6:14] + Pad[0:10]
	copy(data[6:14], ip[:8])
	copy(pad[0:10], ip[8:])
	return any
}

func toSocketAddress(any *windows.RawSockaddrAny) windows.SocketAddress {
	return windows.SocketAddress{
		Sockaddr:       (*syscall.RawSockaddrAny)(unsafe.Pointer(any)),
		SockaddrLength: int32(unsafe.Sizeof(*any)),
	}
}

func TestSockaddrToAddrIPv4(t *testing.T) {
	any := rawSockaddrInet4([4]byte{192, 168, 1, 1})
	sa := toSocketAddress(&any)
	addr, ok := sockaddrToAddr(sa)
	if !ok {
		t.Fatal("IPv4 解析应成功")
	}
	if addr != netip.MustParseAddr("192.168.1.1") {
		t.Fatalf("解析结果不符: %v", addr)
	}
}

func TestSockaddrToAddrIPv6(t *testing.T) {
	any := rawSockaddrInet6([16]byte{0xfe, 0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	sa := toSocketAddress(&any)
	addr, ok := sockaddrToAddr(sa)
	if !ok {
		t.Fatal("IPv6 解析应成功")
	}
	if addr != netip.MustParseAddr("fe80::1") {
		t.Fatalf("解析结果不符: %v", addr)
	}
}

func TestSockaddrToAddrUnsupportedFamily(t *testing.T) {
	var any windows.RawSockaddrAny
	any.Addr.Family = windows.AF_UNSPEC
	if _, ok := sockaddrToAddr(toSocketAddress(&any)); ok {
		t.Fatal("AF_UNSPEC 应解析失败")
	}
}
