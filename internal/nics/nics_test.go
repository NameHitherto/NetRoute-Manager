package nics

import (
	"testing"

	"NetRoute-Manager/internal/models"
)

func TestClassifyNicType(t *testing.T) {
	cases := []struct {
		name     string
		ifType   uint32
		wantType models.NicType
		wantOK   bool
	}{
		{"有线以太网", ifTypeEthernetCSMACD, models.NicTypeWired, true},
		{"无线 802.11", ifTypeIEEE80211, models.NicTypeWireless, true},
		{"蜂窝 WWAN PP", ifTypeWWANPP, "", false},
		{"蜂窝 WWAN PP2", ifTypeWWANPP2, "", false},
		{"软件回环", 24, "", false}, // IF_TYPE_SOFTWARE_LOOPBACK
		{"隧道", 131, "", false},   // IF_TYPE_TUNNEL
	}
	for _, tc := range cases {
		got, ok := classifyNicType(tc.ifType)
		if ok != tc.wantOK || got != tc.wantType {
			t.Errorf("%s: classifyNicType(%d) = (%q,%v), want (%q,%v)",
				tc.name, tc.ifType, got, ok, tc.wantType, tc.wantOK)
		}
	}
}

func TestIsPhysicalAdapterName(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"Realtek PCIe GbE Family Controller", true},
		{"Intel(R) Wi-Fi 6E AX211 160MHz", true},
		{"Ethernet", true},
		{"", false},
		{"VMware Virtual Ethernet Adapter", false},
		{"Hyper-V Virtual Ethernet Adapter", false},
		{"vEthernet (Default Switch)", false},
		{"vEthernet (WSL)", false},
		{"Microsoft Wi-Fi Direct Virtual Adapter", false},
		{"Bluetooth Device (Personal Area Network)", false},
		{"TAP-Windows Adapter V9", false},
		{"Tailscale Tunnel", false},
		{"OpenVPN Data Channel Offload", false},
		{"Wintun Userspace Tunnel", false},
		{"Loopback Pseudo-Interface 1", false},
	}
	for _, tc := range cases {
		if got := isPhysicalAdapterName(tc.name); got != tc.want {
			t.Errorf("isPhysicalAdapterName(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestDisplayName(t *testing.T) {
	cases := []struct {
		name     string
		nicType  models.NicType
		nicName  string
		want     string
	}{
		{"有线", models.NicTypeWired, "以太网", "Ethernet adapter 以太网"},
		{"无线", models.NicTypeWireless, "WLAN", "Wireless LAN adapter WLAN"},
		{"有线英文名", models.NicTypeWired, "Ethernet", "Ethernet adapter Ethernet"},
		{"未知类型原样返回", models.NicType("other"), "X", "X"},
	}
	for _, tc := range cases {
		if got := DisplayName(tc.nicType, tc.nicName); got != tc.want {
			t.Errorf("%s: DisplayName(%q,%q) = %q, want %q", tc.name, tc.nicType, tc.nicName, got, tc.want)
		}
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		name     string
		ifType   uint32
		nicName  string
		wantType models.NicType
		wantOK   bool
	}{
		{"真实有线", ifTypeEthernetCSMACD, "Realtek PCIe GbE Family Controller", models.NicTypeWired, true},
		{"真实无线", ifTypeIEEE80211, "Intel(R) Wi-Fi 6E AX211 160MHz", models.NicTypeWireless, true},
		{"虚拟有线应排除", ifTypeEthernetCSMACD, "VMware Virtual Ethernet Adapter", "", false},
		{"回环类型应排除", 24, "Loopback Pseudo-Interface 1", "", false},
		{"蜂窝类型应排除", ifTypeWWANPP, "Mobile Broadband", "", false},
	}
	for _, tc := range cases {
		got, ok := classify(tc.ifType, tc.nicName)
		if ok != tc.wantOK || got != tc.wantType {
			t.Errorf("%s: classify(%d,%q) = (%q,%v), want (%q,%v)",
				tc.name, tc.ifType, tc.nicName, got, ok, tc.wantType, tc.wantOK)
		}
	}
}
