package model

import "time"

// ScanCategory represents the reconnaissance category of a scan
type ScanCategory string

const (
	ScanCategoryPassive ScanCategory = "passive"
	ScanCategoryActive  ScanCategory = "active"
)

// ScanType represents the type of scan being performed
type ScanType string

const (
	ScanTypeAll       ScanType = "all"
	ScanTypeDNS       ScanType = "dns"
	ScanTypeWHOIS     ScanType = "whois"
	ScanTypeSubdomain ScanType = "subdomain"
	ScanTypePort      ScanType = "port"
	ScanTypeSSL       ScanType = "ssl"
	ScanTypeASN       ScanType = "asn"
	ScanTypeCertTrans ScanType = "cert_trans"
	ScanTypeIP        ScanType = "ip"
	ScanTypeTech      ScanType = "tech"
)

// ScanStatus represents the status of a scan
type ScanStatus string

const (
	ScanStatusPending   ScanStatus = "pending"
	ScanStatusRunning   ScanStatus = "running"
	ScanStatusCompleted ScanStatus = "completed"
	ScanStatusFailed    ScanStatus = "failed"
	ScanStatusPartial   ScanStatus = "partial"
)

// ScanJob represents a scan task
type ScanJob struct {
	ID           string     `json:"id"`
	AssetID      string     `json:"asset_id"`
	ScanType     ScanType   `json:"scan_type"`
	Status       ScanStatus `json:"status"`
	StartedAt    *time.Time `json:"started_at"`
	EndedAt      *time.Time `json:"ended_at"`
	Error        string     `json:"error"`
	ResultsCount int        `json:"results_count"`
	CreatedAt    time.Time  `json:"created_at"`
}

// ScanResult represents the raw scan result data
type ScanResult struct {
	ID         string    `json:"id"`
	JobID      string    `json:"job_id"`
	ResultData any       `json:"result_data"`
	CreatedAt  time.Time `json:"created_at"`
}

// StartScanInput represents the input for starting a scan
type StartScanInput struct {
	ScanType string `json:"scan_type"`
}

// IsValidScanType checks if the given scan type is valid
func IsValidScanType(t ScanType) bool {
	switch t {
	case ScanTypeAll, ScanTypeSubdomain, ScanTypeDNS, ScanTypeWHOIS, ScanTypeCertTrans,
		ScanTypePort, ScanTypeASN, ScanTypeSSL, ScanTypeIP, ScanTypeTech:
		return true
	}
	return false
}

// Category returns the reconnaissance category for this scan type
func (st ScanType) Category() ScanCategory {
	switch st {
	case ScanTypeAll, ScanTypeDNS, ScanTypeWHOIS, ScanTypeSubdomain, ScanTypeCertTrans, ScanTypeASN, ScanTypeIP:
		return ScanCategoryPassive
	case ScanTypePort, ScanTypeSSL, ScanTypeTech:
		return ScanCategoryActive
	default:
		return ScanCategoryPassive
	}
}

// RequiresPermission returns true if this scan type requires authorization
func (st ScanType) RequiresPermission() bool {
	return st.Category() == ScanCategoryActive
}

// IsPassive returns true if this is a passive/OSINT scan
func (st ScanType) IsPassive() bool {
	return st.Category() == ScanCategoryPassive
}

// IsActive returns true if this is an active/intrusive scan
func (st ScanType) IsActive() bool {
	return st.Category() == ScanCategoryActive
}

// Description returns a human-readable description of the scan type
func (st ScanType) Description() string {
	switch st {
	case ScanTypeAll:
		return "Run all available passive scans (passive)"
	case ScanTypeDNS:
		return "DNS queries for A, AAAA, MX, NS, TXT, CNAME records (passive)"
	case ScanTypeWHOIS:
		return "WHOIS registration lookup (passive)"
	case ScanTypeSubdomain:
		return "Subdomain enumeration via DNS bruteforce (passive)"
	case ScanTypeCertTrans:
		return "Certificate Transparency log search (passive)"
	case ScanTypeASN:
		return "ASN and IP range lookup (passive)"
	case ScanTypeIP:
		return "IP geolocation and reverse DNS lookup (passive)"
	case ScanTypePort:
		return "Port scanning for open services (active)"
	case ScanTypeSSL:
		return "SSL/TLS certificate probing (active)"
	case ScanTypeTech:
		return "Technology fingerprinting (active)"
	default:
		return string(st)
	}
}

// IsValidScanStatus checks if the given scan status is valid
func IsValidScanStatus(s ScanStatus) bool {
	switch s {
	case ScanStatusPending, ScanStatusRunning, ScanStatusCompleted, ScanStatusFailed, ScanStatusPartial:
		return true
	}
	return false
}
