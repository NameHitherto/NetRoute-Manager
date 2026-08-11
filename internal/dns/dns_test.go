package dns

import (
	"context"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockDNSServer 是一个极简 UDP DNS 服务器,用于测试。
// records 以 "名称:qtype" 为键(A=1 / AAAA=28),命中则回复对应记录;
// silent 模式下收到请求不回复,用于模拟超时。
type mockDNSServer struct {
	ln      *net.UDPConn
	records map[string][]netip.Addr
	silent  bool

	mu      sync.Mutex
	stopped bool
}

func newMockDNSServer(t *testing.T, records map[string][]netip.Addr) *mockDNSServer {
	t.Helper()
	ln, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("ListenUDP 失败: %v", err)
	}
	s := &mockDNSServer{ln: ln, records: records}
	go s.serve(t)
	t.Cleanup(s.close)
	return s
}

// addr 返回监听地址字符串。
func (s *mockDNSServer) addr() string { return s.ln.LocalAddr().String() }

func (s *mockDNSServer) close() {
	s.mu.Lock()
	s.stopped = true
	s.mu.Unlock()
	_ = s.ln.Close()
}

func (s *mockDNSServer) serve(t *testing.T) {
	buf := make([]byte, 512)
	for {
		n, client, err := s.ln.ReadFromUDP(buf)
		if err != nil {
			return
		}
		s.mu.Lock()
		silent := s.silent
		s.mu.Unlock()
		if silent {
			continue
		}
		name, qtype, ok := parseQuestion(buf[:n])
		if !ok {
			continue
		}
		key := strings.ToLower(name) + ":" + strconv.Itoa(qtype)
		ips, ok := s.records[key]
		if !ok {
			continue // 未知查询不响应
		}
		resp := buildResponse(buf[:n], qtype, ips)
		_, _ = s.ln.WriteToUDP(resp, client)
	}
}

// parseQuestion 解析 DNS 查询的 question 段(假定无压缩指针),返回名称与查询类型。
func parseQuestion(msg []byte) (string, int, bool) {
	if len(msg) < 12 {
		return "", 0, false
	}
	// 跳过 header(12 字节),解析 QNAME
	off := 12
	var labels []string
	for {
		if off >= len(msg) {
			return "", 0, false
		}
		l := int(msg[off])
		if l == 0 {
			off++
			break
		}
		if off+1+l > len(msg) {
			return "", 0, false
		}
		labels = append(labels, string(msg[off+1:off+1+l]))
		off += 1 + l
	}
	if off+4 > len(msg) {
		return "", 0, false
	}
	qtype := int(msg[off])<<8 | int(msg[off+1])
	return strings.Join(labels, "."), qtype, true
}

// buildResponse 构造标准 DNS 响应:复制 header 与 question,追加 answer 记录。
func buildResponse(req []byte, qtype int, ips []netip.Addr) []byte {
	name, _, ok := parseQuestion(req)
	if !ok {
		return nil
	}
	// 计算 question 段长度:QNAME 各 label + 结尾 0 + QTYPE/QCLASS(4 字节)
	qnameLen := 1 // 结尾 0
	for _, part := range strings.Split(name, ".") {
		qnameLen += 1 + len(part)
	}
	qLen := 12 + qnameLen + 4

	resp := make([]byte, 0, qLen+len(ips)*24)
	resp = append(resp, req[0], req[1])         // 复制事务 ID
	resp = append(resp, 0x81, 0x80)             // QR=1 RA=1 RD=1 NOERROR
	resp = append(resp, 0x00, 0x01)             // QDCOUNT=1
	resp = append(resp, 0x00, byte(len(ips)))   // ANCOUNT
	resp = append(resp, 0x00, 0x00, 0x00, 0x00) // NSCOUNT/ARCOUNT
	resp = append(resp, req[12:qLen]...)        // 原样回显 question

	for _, ip := range ips {
		resp = append(resp, 0xC0, 0x0C) // 名称指针指向 question 中的 QNAME
		resp = append(resp, byte(qtype>>8), byte(qtype))
		resp = append(resp, 0x00, 0x01)             // CLASS=IN
		resp = append(resp, 0x00, 0x00, 0x00, 0x3C) // TTL=60
		b := ip.AsSlice()
		resp = append(resp, 0x00, byte(len(b))) // RDLENGTH
		resp = append(resp, b...)
	}
	return resp
}

// testResolver 构造绑定到 mock 服务器的 Resolver。
func testResolver(t *testing.T, s *mockDNSServer) *Resolver {
	t.Helper()
	r := New()
	addr := s.addr()
	r.SetDial(func(ctx context.Context, network, address string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, "udp", addr)
	})
	r.SetTimeout(500 * time.Millisecond)
	return r
}

func TestResolveA(t *testing.T) {
	server := newMockDNSServer(t, map[string][]netip.Addr{
		"example.com:1": {netip.MustParseAddr("93.184.216.34"), netip.MustParseAddr("93.184.216.35")},
	})
	r := testResolver(t, server)

	ips, err := r.Resolve(context.Background(), "example.com", []string{"127.0.0.1"}, false)
	if err != nil {
		t.Fatalf("Resolve() 出错: %v", err)
	}
	if len(ips) != 2 || ips[0] != netip.MustParseAddr("93.184.216.34") {
		t.Fatalf("A 记录解析结果不符: %v", ips)
	}
}

func TestResolveIPv6Preferred(t *testing.T) {
	server := newMockDNSServer(t, map[string][]netip.Addr{
		"dual.example:1":  {netip.MustParseAddr("1.2.3.4")},
		"dual.example:28": {netip.MustParseAddr("2606:4700::1111"), netip.MustParseAddr("2606:4700::2222")},
	})
	r := testResolver(t, server)

	ips, err := r.Resolve(context.Background(), "dual.example", []string{"127.0.0.1"}, true)
	if err != nil {
		t.Fatalf("Resolve() 出错: %v", err)
	}
	if len(ips) != 2 || ips[0].Is6() != true {
		t.Fatalf("enableIPv6 应优先返回 AAAA 记录: %v", ips)
	}
	if ips[0] != netip.MustParseAddr("2606:4700::1111") {
		t.Fatalf("AAAA 结果不符: %v", ips)
	}
}

func TestResolveIPv6FallbackToA(t *testing.T) {
	server := newMockDNSServer(t, map[string][]netip.Addr{
		"v4only.example:1":  {netip.MustParseAddr("9.9.9.9")},
		"v4only.example:28": {}, // 无 AAAA 记录:返回 NOERROR 空答案
	})
	r := testResolver(t, server)

	ips, err := r.Resolve(context.Background(), "v4only.example", []string{"127.0.0.1"}, true)
	if err != nil {
		t.Fatalf("Resolve() 出错: %v", err)
	}
	if len(ips) != 1 || ips[0] != netip.MustParseAddr("9.9.9.9") {
		t.Fatalf("无 AAAA 时应回退 A 记录: %v", ips)
	}
}

func TestResolveFallbackToSecondary(t *testing.T) {
	// 主服务器 silent(模拟超时),备服务器正常响应
	primary := newMockDNSServer(t, nil)
	primary.silent = true
	secondary := newMockDNSServer(t, map[string][]netip.Addr{
		"example.com:1": {netip.MustParseAddr("5.6.7.8")},
	})
	r := New()
	r.SetDial(func(ctx context.Context, network, address string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, "udp", secondary.addr())
	})
	r.SetTimeout(200 * time.Millisecond)

	ips, err := r.Resolve(context.Background(), "example.com", []string{"127.0.0.1", "127.0.0.1"}, false)
	if err != nil {
		t.Fatalf("主服务器超时后应回退备服务器: %v", err)
	}
	if len(ips) != 1 || ips[0] != netip.MustParseAddr("5.6.7.8") {
		t.Fatalf("回退结果不符: %v", ips)
	}
}

func TestResolveFailure(t *testing.T) {
	server := newMockDNSServer(t, nil) // 对任何查询都不响应
	r := testResolver(t, server)

	_, err := r.Resolve(context.Background(), "nowhere.example", []string{"127.0.0.1"}, false)
	if err == nil {
		t.Fatal("全部服务器失败应返回错误")
	}
	if !strings.Contains(err.Error(), "nowhere.example") {
		t.Fatalf("错误信息应包含域名: %v", err)
	}
}

func TestResolveInvalidServerSkipped(t *testing.T) {
	server := newMockDNSServer(t, map[string][]netip.Addr{
		"example.com:1": {netip.MustParseAddr("1.1.1.1")},
	})
	r := testResolver(t, server)

	ips, err := r.Resolve(context.Background(), "example.com", []string{"not-an-ip", "127.0.0.1"}, false)
	if err != nil {
		t.Fatalf("无效服务器应被跳过并回退合法服务器: %v", err)
	}
	if len(ips) != 1 {
		t.Fatalf("结果不符: %v", ips)
	}
}
