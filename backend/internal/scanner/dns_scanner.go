package scanner

import (
	"context"
	"net"
	"time"
)

// DNSResult định nghĩa cấu trúc dữ liệu trả về sau khi quét DNS của một tên miền.
// Cấu trúc này chứa thông tin về tên miền, các bản ghi A, MX, NS, TXT và thời điểm quét.
type DNSResult struct {
	Domain    string   `json:"domain"`         // Tên miền được quét
	A         []string `json:"a,omitempty"`    // Danh sách các địa chỉ IPv4 (bản ghi A)
	MX        []string `json:"mx,omitempty"`    // Danh sách máy chủ thư điện tử (bản ghi MX)
	NS        []string `json:"ns,omitempty"`    // Danh sách máy chủ tên miền quản lý (bản ghi NS)
	TXT       []string `json:"txt,omitempty"`   // Danh sách các bản ghi văn bản (bản ghi TXT)
	ScannedAt string   `json:"scanned_at"`     // Thời gian thực hiện quét (định dạng UTC RFC3339)
}

// DNSScanner là bộ quét thực hiện các truy vấn DNS để thu thập thông tin cấu hình của tên miền.
type DNSScanner struct{}

// NewDNSScanner khởi tạo một đối tượng DNSScanner mới.
func NewDNSScanner() *DNSScanner {
	return &DNSScanner{}
}

// Scan thực hiện quét và truy vấn các bản ghi DNS chính của tên miền được chỉ định (target).
// Hàm này sử dụng một bộ giải quyết tên miền (Resolver) được cấu hình riêng để tránh phụ thuộc vào DNS mặc định của hệ thống.
func (s *DNSScanner) Scan(ctx context.Context, target string) (any, error) {
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: 3 * time.Second,
			}
			return d.DialContext(ctx, network, "8.8.8.8:53")
		},
	}

	result := DNSResult{
		Domain:    target,
		ScannedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if ips, err := resolver.LookupHost(ctx, target); err == nil {
		result.A = ips
	}

	if mxs, err := resolver.LookupMX(ctx, target); err == nil {
		for _, mx := range mxs {
			result.MX = append(result.MX, mx.Host)
		}
	}

	if nss, err := resolver.LookupNS(ctx, target); err == nil {
		for _, ns := range nss {
			result.NS = append(result.NS, ns.Host)
		}
	}

	if txts, err := resolver.LookupTXT(ctx, target); err == nil {
		result.TXT = txts
	}

	return result, nil
}

