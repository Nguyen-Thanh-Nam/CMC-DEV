package scanner

import (
	"context"
	"fmt"
	"net"
	"time"
)

// OpenPort mô tả thông tin chi tiết về một cổng (port) đang mở được phát hiện.
type OpenPort struct {
	Port     int    `json:"port"`     // Số hiệu cổng (ví dụ: 80, 443)
	Protocol string `json:"protocol"` // Giao thức truyền tải (ví dụ: tcp)
	State    string `json:"state"`    // Trạng thái cổng (thường là "open")
	Service  string `json:"service"`  // Tên dịch vụ dự kiến chạy trên cổng đó (ví dụ: http, ssh)
	Version  string `json:"version"`  // Phiên bản dịch vụ (nếu phát hiện được)
}

// PortScanResult chứa kết quả tổng hợp sau khi quét danh sách các cổng của một IP.
type PortScanResult struct {
	IPAddress      string     `json:"ip_address"`       // Địa chỉ IP đích được quét
	OpenPorts      []OpenPort `json:"open_ports"`        // Danh sách các cổng đang mở
	ClosedPorts    int        `json:"closed_ports"`      // Số lượng cổng đóng/không phản hồi
	TotalScanned   int        `json:"total_scanned"`     // Tổng số lượng cổng được quét
	ScanDurationMs int64      `json:"scan_duration_ms"`  // Thời gian quét tính bằng mili giây
	CreatedAt      string     `json:"created_at"`        // Thời điểm tạo bản ghi kết quả (UTC)
}

// PortScanner chịu trách nhiệm thực hiện quét các cổng TCP thông dụng của một máy chủ mục tiêu.
type PortScanner struct{}

// NewPortScanner khởi tạo một bộ quét PortScanner mới.
func NewPortScanner() *PortScanner {
	return &PortScanner{}
}

// Scan thực hiện quét danh sách các cổng TCP xác định trên IP mục tiêu.
// Vì lý do an toàn bảo mật, bộ quét này chỉ cho phép quét các địa chỉ IP nội bộ (private IPs).
func (s *PortScanner) Scan(ctx context.Context, target string) (any, error) {
	startTime := time.Now()

	ips, err := net.LookupIP(target)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve target %s: %w", target, err)
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("no IP address found for target %s", target)
	}

	ip := ips[0]
	
	if !isPrivateIP(ip) {
		return nil, fmt.Errorf("security error: port scanning public IPs (%s) is not allowed", ip.String())
	}

	ports := []int{21, 22, 23, 25, 53, 80, 110, 143, 443, 445, 1433, 1521, 3000, 3306, 5432, 6379, 8000, 8080, 8443, 9000, 27017}
	var openPorts []OpenPort
	closedCount := 0

	for _, port := range ports {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		address := fmt.Sprintf("%s:%d", ip.String(), port)
		conn, err := net.DialTimeout("tcp", address, 500*time.Millisecond)
		if err != nil {
			closedCount++
			continue
		}
		conn.Close()

		service := "unknown"
		switch port {
		case 21:
			service = "ftp"
		case 22:
			service = "ssh"
		case 23:
			service = "telnet"
		case 25:
			service = "smtp"
		case 53:
			service = "dns"
		case 80:
			service = "http"
		case 110:
			service = "pop3"
		case 143:
			service = "imap"
		case 443:
			service = "https"
		case 445:
			service = "smb"
		case 1433:
			service = "mssql"
		case 1521:
			service = "oracle"
		case 3000:
			service = "frontend/nginx"
		case 3306:
			service = "mysql"
		case 5432:
			service = "postgresql"
		case 6379:
			service = "redis"
		case 8000:
			service = "http-alt"
		case 8080:
			service = "http-alt/go-api"
		case 8443:
			service = "https-alt"
		case 9000:
			service = "portainer/php-fpm"
		case 27017:
			service = "mongodb"
		}

		openPorts = append(openPorts, OpenPort{
			Port:     port,
			Protocol: "tcp",
			State:    "open",
			Service:  service,
			Version:  "",
		})
	}

	result := PortScanResult{
		IPAddress:      ip.String(),
		OpenPorts:      openPorts,
		ClosedPorts:    closedCount,
		TotalScanned:   len(ports),
		ScanDurationMs: time.Since(startTime).Milliseconds(),
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}

	return result, nil
}

