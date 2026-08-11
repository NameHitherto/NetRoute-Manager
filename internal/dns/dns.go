// Package dns 提供定向到指定 DNS 服务器的域名解析。
//
// 查询通过 Go 内置解析器(net.Resolver, PreferGo)发出,并强制走 UDP 直连
// 配置的主/备 DNS 服务器,不依赖系统 DNS 配置;主服务器失败/超时自动回退备服务器。
package dns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"time"
)

// defaultTimeout 单次查询的超时上限。
const defaultTimeout = 5 * time.Second

// ErrNoResult 表示域名解析成功但无可用记录(如仅查 AAAA 而无记录)。
var ErrNoResult = errors.New("域名无可用解析记录")

// Resolver 负责域名解析,内部无状态,可并发使用。
type Resolver struct {
	dial    func(ctx context.Context, network, address string) (net.Conn, error)
	timeout time.Duration
}

// New 创建默认 Resolver。
func New() *Resolver {
	return &Resolver{timeout: defaultTimeout}
}

// SetDial 注入自定义拨号函数,用于测试将查询定向到本地 mock DNS 服务器。
func (r *Resolver) SetDial(dial func(ctx context.Context, network, address string) (net.Conn, error)) {
	r.dial = dial
}

// SetTimeout 覆盖单次查询超时(测试用)。
func (r *Resolver) SetTimeout(d time.Duration) {
	r.timeout = d
}

// Resolve 解析域名,返回去重后的 IP 列表。
//
//   - servers: 依次尝试的 DNS 服务器地址(IP 字符串,如 "223.5.5.5"),
//     前一个失败/超时自动回退下一个;
//   - enableIPv6: true 时优先查询 AAAA 记录,无结果回退 A 记录;
//     false 时仅查询 A 记录;
//   - 返回的 IP 已规范化为无区域/无映射的 netip.Addr,并按查询顺序返回。
func (r *Resolver) Resolve(ctx context.Context, domain string, servers []string, enableIPv6 bool) ([]netip.Addr, error) {
	families := []string{"ip4"}
	if enableIPv6 {
		families = []string{"ip6", "ip4"}
	}

	var lastErr error
	for _, family := range families {
		for _, srv := range servers {
			srv = strings.TrimSpace(srv)
			if srv == "" {
				continue
			}
			if _, err := netip.ParseAddr(srv); err != nil {
				lastErr = fmt.Errorf("无效 DNS 服务器地址 %q", srv)
				continue
			}
			ips, err := r.lookup(ctx, domain, family, srv)
			if err != nil {
				lastErr = err
				continue
			}
			if len(ips) > 0 {
				return ips, nil
			}
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("DNS 解析 %s 失败: %w", domain, lastErr)
	}
	return nil, fmt.Errorf("%w: %s", ErrNoResult, domain)
}

// lookup 通过指定服务器查询指定 IP 族(ip4/ip6)的记录。
// 每次查询独立超时,避免单个慢服务器拖垮整体。
func (r *Resolver) lookup(ctx context.Context, domain, family, server string) ([]netip.Addr, error) {
	dial := r.dial
	if dial == nil {
		dial = func(ctx context.Context, network, address string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, network, address)
		}
	}
	timeout := r.timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	res := &net.Resolver{
		PreferGo: true,
		// 强制将查询发往指定 DNS 服务器(覆盖默认的 "host:53")。
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			return dial(ctx, network, net.JoinHostPort(server, "53"))
		},
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ips, err := res.LookupIP(ctx, family, domain)
	if err != nil {
		return nil, err
	}
	out := make([]netip.Addr, 0, len(ips))
	seen := make(map[netip.Addr]struct{}, len(ips))
	for _, ip := range ips {
		a, ok := netip.AddrFromSlice(ip)
		if !ok {
			continue
		}
		a = a.Unmap()
		if _, dup := seen[a]; dup {
			continue
		}
		seen[a] = struct{}{}
		out = append(out, a)
	}
	return out, nil
}
