// Package models 定义 NetRoute-Manager 的领域数据结构。
//
// 所有结构体与前端契约(frontend/src/types/index.ts)保持 1:1 对应,
// JSON tag 严格使用 camelCase,与前端字段名一致。
package models

// RouteRule 路由规则条目,与前端 RouteRule 类型对应。
type RouteRule struct {
	ID              string `json:"id"`
	Domain          string `json:"domain"`
	Port            string `json:"port"`
	Alias           string `json:"alias"`
	Checked         bool   `json:"checked"`
	ResolvedIP      string `json:"resolvedIp"`
	LastResolvedSec int    `json:"lastResolvedSec"`
}

// RouteRuleInput 新建/编辑路由规则时提交的表单数据。
type RouteRuleInput struct {
	Domain string `json:"domain"`
	Port   string `json:"port"`
	Alias  string `json:"alias"`
}

// NicType 物理网卡类型。
type NicType string

const (
	NicTypeWired    NicType = "wired"
	NicTypeWireless NicType = "wireless"
)

// NetworkInterface 物理网卡。
// IPv4Gateway/IPv6Gateway 为网关转发所需:AddHostRoute 将解析出的 IP
// 路由到该网卡的网关(非 on-link 直连),故无 IPv4 网关的网卡不可用于本服务。
type NetworkInterface struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Type        NicType `json:"type"`
	Active      bool    `json:"active"`
	IPv4Gateway string  `json:"ipv4Gateway"`
	IPv6Gateway string  `json:"ipv6Gateway"`
}

// DnsMode DNS 解析模式。
// 本期仅支持 UDP 直连查询(设置中的主备 DNS 服务器);DoH/DoT 暂未实现。
type DnsMode string

const (
	DnsModeUDP DnsMode = "UDP"
)

// Valid 判断 DNS 模式是否为合法取值。
func (m DnsMode) Valid() bool {
	return m == DnsModeUDP
}

// AppSettings 全局设置,与前端 AppSettings 类型对应。
type AppSettings struct {
	PrimaryDNS    string  `json:"primaryDns"`
	SecondaryDNS  string  `json:"secondaryDns"`
	QueryInterval int     `json:"queryInterval"` // 秒
	EnableIPv6    bool    `json:"enableIpv6"`
	AutoStart     bool    `json:"autoStart"`
	MinToTray     bool    `json:"minToTray"`
	DNSMode       DnsMode `json:"dnsMode"`
}

// LogLevel 日志级别。
type LogLevel string

const (
	LogLevelInfo    LogLevel = "info"
	LogLevelWarn    LogLevel = "warn"
	LogLevelError   LogLevel = "error"
	LogLevelSuccess LogLevel = "success"
)

// Valid 判断日志级别是否为合法取值。
func (l LogLevel) Valid() bool {
	switch l {
	case LogLevelInfo, LogLevelWarn, LogLevelError, LogLevelSuccess:
		return true
	}
	return false
}

// LogEntry 运行日志条目,与前端 LogEntry 类型对应。
type LogEntry struct {
	ID   int      `json:"id"`
	Time string   `json:"time"`
	Type LogLevel `json:"type"`
	Text string   `json:"text"`
}

// ServiceStartResult 服务启动结果,与前端 ServiceStartResult 类型对应。
type ServiceStartResult struct {
	Running bool        `json:"running"`
	NicID   string      `json:"nicId"`
	Rules   []RouteRule `json:"rules"`
}

// DefaultSettings 返回与前端 mock 一致的默认全局设置。
func DefaultSettings() AppSettings {
	return AppSettings{
		PrimaryDNS:    "223.5.5.5",
		SecondaryDNS:  "114.114.114.114",
		QueryInterval: 30,
		EnableIPv6:    false,
		AutoStart:     true,
		MinToTray:     true,
		DNSMode:       DnsModeUDP,
	}
}
