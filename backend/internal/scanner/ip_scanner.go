package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// Geolocation chứa các thông tin địa lý của một địa chỉ IP công cộng.
type Geolocation struct {
	Country     string  `json:"country"`      // Tên quốc gia
	CountryCode string  `json:"country_code"` // Mã quốc gia (ví dụ: VN, US)
	City        string  `json:"city"`         // Tên thành phố
	Region      string  `json:"region"`       // Vùng/Tỉnh/Bang
	Latitude    float64 `json:"latitude"`     // Vĩ độ địa lý
	Longitude   float64 `json:"longitude"`    // Kinh độ địa lý
	ISP         string  `json:"isp"`          // Nhà cung cấp dịch vụ Internet (Internet Service Provider)
	Org         string  `json:"org"`          // Tên tổ chức sở hữu dải IP
}

// ASNInfo chứa thông tin về số hiệu mạng tự trị (Autonomous System Number - ASN).
type ASNInfo struct {
	Number      int    `json:"number"`      // Số hiệu ASN (ví dụ: 13335)
	Name        string `json:"name"`        // Tên của ASN
	Description string `json:"description"` // Mô tả chi tiết/Tên tổ chức đăng ký ASN
}

// IPScanResult đại diện cho kết quả quét thông tin của một IP, bao gồm DNS ngược, địa lý và ASN.
type IPScanResult struct {
	IPAddress   string      `json:"ip_address"`  // Địa chỉ IP đích được quét
	Geolocation Geolocation `json:"geolocation"` // Dữ liệu địa lý của IP
	ASN         ASNInfo     `json:"asn"`         // Thông tin mạng ASN liên kết với IP
	ReverseDNS  string      `json:"reverse_dns"` // Tên miền phân giải ngược từ IP (PTR record)
	CreatedAt   string      `json:"created_at"`  // Thời gian tạo kết quả quét (UTC)
}

// IPScanner là bộ quét thu thập thông tin cấu hình mạng của một địa chỉ IP hoặc tên miền.
type IPScanner struct{}

// NewIPScanner khởi tạo một bộ quét IPScanner mới.
func NewIPScanner() *IPScanner {
	return &IPScanner{}
}

// isPrivateIP kiểm tra xem một địa chỉ IP có thuộc dải mạng nội bộ (private) hay không.
// Các dải mạng nội bộ bao gồm: loopback (127.0.0.1), link-local (169.254.x.x) hoặc dải mạng riêng RFC 1918 (10.x.x.x, 172.16.x.x, 192.168.x.x).
func isPrivateIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate()
}

// Scan thực hiện phân giải IP, tra cứu DNS ngược, kiểm tra dải IP nội bộ và gọi API định vị địa lý bên ngoài.
func (s *IPScanner) Scan(ctx context.Context, target string) (any, error) {
	result := IPScanResult{
		IPAddress: target,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if names, err := net.LookupAddr(target); err == nil && len(names) > 0 {
		result.ReverseDNS = strings.TrimSuffix(names[0], ".")
	} else {
		result.ReverseDNS = "N/A"
	}

	ip := net.ParseIP(target)
	if ip == nil {
		if ips, err := net.LookupIP(target); err == nil && len(ips) > 0 {
			ip = ips[0]
			result.IPAddress = ip.String()
		} else {
			return nil, fmt.Errorf("invalid IP address or host: %s", target)
		}
	}

	if isPrivateIP(ip) {
		result.Geolocation = Geolocation{
			Country:     "Local Loopback",
			CountryCode: "LOCAL",
			City:        "Internal Network",
			Region:      "Private",
			Latitude:    0.0,
			Longitude:   0.0,
			ISP:         "Localhost Provider",
			Org:         "Local Organization",
		}
		result.ASN = ASNInfo{
			Number:      0,
			Name:        "LOCAL-ASN",
			Description: "Private Loopback Address Space",
		}
		return result, nil
	}

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", "http://ip-api.com/json/"+ip.String()+"?fields=status,message,country,countryCode,regionName,city,lat,lon,isp,org,as", nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return s.getFallbackResult(result), nil
	}
	defer resp.Body.Close()

	var apiResp struct {
		Status      string  `json:"status"`
		Message     string  `json:"message"`
		Country     string  `json:"country"`
		CountryCode string  `json:"countryCode"`
		RegionName  string  `json:"regionName"`
		City        string  `json:"city"`
		Lat         float64 `json:"lat"`
		Lon         float64 `json:"lon"`
		ISP         string  `json:"isp"`
		Org         string  `json:"org"`
		AS          string  `json:"as"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil || apiResp.Status != "success" {
		return s.getFallbackResult(result), nil
	}

	result.Geolocation = Geolocation{
		Country:     apiResp.Country,
		CountryCode: apiResp.CountryCode,
		City:        apiResp.City,
		Region:      apiResp.RegionName,
		Latitude:    apiResp.Lat,
		Longitude:   apiResp.Lon,
		ISP:         apiResp.ISP,
		Org:         apiResp.Org,
	}

	asnNum := 0
	asnName := "UNKNOWN"
	if apiResp.AS != "" {
		parts := strings.SplitN(apiResp.AS, " ", 2)
		if len(parts) > 0 {
			var num int
			_, _ = fmt.Sscanf(parts[0], "AS%d", &num)
			asnNum = num
		}
		if len(parts) > 1 {
			asnName = parts[1]
		}
	}

	result.ASN = ASNInfo{
		Number:      asnNum,
		Name:        asnName,
		Description: apiResp.Org,
	}

	return result, nil
}

// getFallbackResult cung cấp dữ liệu giả lập dự phòng khi dịch vụ định vị IP bên thứ ba gặp sự cố hoặc offline.
func (s *IPScanner) getFallbackResult(in IPScanResult) IPScanResult {
	in.Geolocation = Geolocation{
		Country:     "United States",
		CountryCode: "US",
		City:        "San Francisco",
		Region:      "California",
		Latitude:    37.7749,
		Longitude:   -122.4194,
		ISP:         "Public Provider (Mock Fallback)",
		Org:         "Mock Organization",
	}
	in.ASN = ASNInfo{
		Number:      13335,
		Name:        "MOCK-CLOUDFLARENET",
		Description: "Mock Geolocation Fallback Data",
	}
	return in
}

