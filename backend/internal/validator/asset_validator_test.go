package validator

import (
	"strings"
	"testing"

	"asset-api/internal/model"
)

func TestValidateName(t *testing.T) {
	validator := NewAssetValidator()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid name",
			input:   "example.com",
			wantErr: false,
		},
		{
			name:    "empty name",
			input:   "",
			wantErr: true,
		},
		{
			name:    "name too long",
			input:   strings.Repeat("a", 256),
			wantErr: true,
		},
		{
			name:    "name with null byte",
			input:   "test\x00name",
			wantErr: true,
		},
		{
			name:    "name at max length",
			input:   strings.Repeat("a", 255),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateType(t *testing.T) {
	validator := NewAssetValidator()

	tests := []struct {
		name      string
		assetType string
		wantErr   bool
	}{
		{
			name:      "valid domain type",
			assetType: model.TypeDomain,
			wantErr:   false,
		},
		{
			name:      "valid ip type",
			assetType: model.TypeIP,
			wantErr:   false,
		},
		{
			name:      "valid service type",
			assetType: model.TypeService,
			wantErr:   false,
		},
		{
			name:      "invalid type",
			assetType: "invalid",
			wantErr:   true,
		},
		{
			name:      "empty type",
			assetType: "",
			wantErr:   true,
		},
		{
			name:      "case sensitive - DOMAIN",
			assetType: "DOMAIN",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateType(tt.assetType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateType(%q) error = %v, wantErr %v", tt.assetType, err, tt.wantErr)
			}
		})
	}
}

func TestValidateStatus(t *testing.T) {
	validator := NewAssetValidator()

	tests := []struct {
		name    string
		status  string
		wantErr bool
	}{
		{
			name:    "valid active status",
			status:  model.StatusActive,
			wantErr: false,
		},
		{
			name:    "valid inactive status",
			status:  model.StatusInactive,
			wantErr: false,
		},
		{
			name:    "invalid status",
			status:  "pending",
			wantErr: true,
		},
		{
			name:    "empty status",
			status:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateStatus(tt.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStatus(%q) error = %v, wantErr %v", tt.status, err, tt.wantErr)
			}
		})
	}
}

func TestValidateDomain(t *testing.T) {
	validator := NewAssetValidator()

	tests := []struct {
		name    string
		domain  string
		wantErr bool
	}{
		{
			name:    "valid simple domain",
			domain:  "example.com",
			wantErr: false,
		},
		{
			name:    "valid subdomain",
			domain:  "api.example.com",
			wantErr: false,
		},
		{
			name:    "valid deep subdomain",
			domain:  "api.v2.staging.example.com",
			wantErr: false,
		},
		{
			name:    "valid domain with hyphen",
			domain:  "my-site.example.com",
			wantErr: false,
		},
		{
			name:    "valid single label",
			domain:  "localhost",
			wantErr: false,
		},
		{
			name:    "empty domain",
			domain:  "",
			wantErr: true,
		},
		{
			name:    "domain too long",
			domain:  strings.Repeat("a", 254),
			wantErr: true,
		},
		{
			name:    "domain with leading dot",
			domain:  ".example.com",
			wantErr: true,
		},
		{
			name:    "domain with trailing dot",
			domain:  "example.com.",
			wantErr: true,
		},
		{
			name:    "domain with leading hyphen",
			domain:  "-example.com",
			wantErr: true,
		},
		{
			name:    "domain with trailing hyphen",
			domain:  "example.com-",
			wantErr: true,
		},
		{
			name:    "domain with spaces",
			domain:  "example .com",
			wantErr: true,
		},
		{
			name:    "domain with special chars",
			domain:  "example@com",
			wantErr: true,
		},
		{
			name:    "domain with double dots",
			domain:  "example..com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateDomain(tt.domain)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDomain(%q) error = %v, wantErr %v", tt.domain, err, tt.wantErr)
			}
		})
	}
}

func TestValidateIP(t *testing.T) {
	validator := NewAssetValidator()

	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{
			name:    "valid IPv4",
			ip:      "192.168.1.1",
			wantErr: false,
		},
		{
			name:    "valid IPv4 - zeros",
			ip:      "0.0.0.0",
			wantErr: false,
		},
		{
			name:    "valid IPv4 - max",
			ip:      "255.255.255.255",
			wantErr: false,
		},
		{
			name:    "valid IPv6",
			ip:      "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			wantErr: false,
		},
		{
			name:    "valid IPv6 - short",
			ip:      "2001:db8::1",
			wantErr: false,
		},
		{
			name:    "valid IPv6 - loopback",
			ip:      "::1",
			wantErr: false,
		},
		{
			name:    "invalid - empty",
			ip:      "",
			wantErr: true,
		},
		{
			name:    "invalid - not an IP",
			ip:      "example.com",
			wantErr: true,
		},
		{
			name:    "invalid - IPv4 out of range",
			ip:      "256.1.1.1",
			wantErr: true,
		},
		{
			name:    "invalid - IPv4 incomplete",
			ip:      "192.168.1",
			wantErr: true,
		},
		{
			name:    "invalid - IPv6 malformed",
			ip:      "2001:db8::xyz",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateIP(tt.ip)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIP(%q) error = %v, wantErr %v", tt.ip, err, tt.wantErr)
			}
		})
	}
}

func TestValidateService(t *testing.T) {
	validator := NewAssetValidator()

	tests := []struct {
		name    string
		service string
		wantErr bool
	}{
		{
			name:    "valid URL",
			service: "http://example.com",
			wantErr: false,
		},
		{
			name:    "valid HTTPS URL",
			service: "https://example.com",
			wantErr: false,
		},
		{
			name:    "valid URL with port",
			service: "https://example.com:443",
			wantErr: false,
		},
		{
			name:    "valid URL with path",
			service: "https://example.com/api",
			wantErr: false,
		},
		{
			name:    "valid service name",
			service: "ssh",
			wantErr: false,
		},
		{
			name:    "valid service name with hyphen",
			service: "my-service",
			wantErr: false,
		},
		{
			name:    "empty service",
			service: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateService(tt.service)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateService(%q) error = %v, wantErr %v", tt.service, err, tt.wantErr)
			}
		})
	}
}

func TestValidateCreate(t *testing.T) {
	validator := NewAssetValidator()

	tests := []struct {
		name      string
		assetName string
		assetType string
		wantErr   bool
	}{
		{
			name:      "valid domain asset",
			assetName: "example.com",
			assetType: model.TypeDomain,
			wantErr:   false,
		},
		{
			name:      "valid IP asset",
			assetName: "192.168.1.1",
			assetType: model.TypeIP,
			wantErr:   false,
		},
		{
			name:      "valid service asset",
			assetName: "https://example.com",
			assetType: model.TypeService,
			wantErr:   false,
		},
		{
			name:      "domain with invalid type",
			assetName: "example.com",
			assetType: "invalid",
			wantErr:   true,
		},
		{
			name:      "IP as domain type - should fail",
			assetName: "192.168.1.1",
			assetType: model.TypeDomain,
			wantErr:   true,
		},
		{
			name:      "domain as IP type - should fail",
			assetName: "example.com",
			assetType: model.TypeIP,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateCreate(tt.assetName, tt.assetType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCreate(%q, %q) error = %v, wantErr %v",
					tt.assetName, tt.assetType, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePaginationParams(t *testing.T) {
	validator := NewAssetValidator()

	tests := []struct {
		name     string
		page     int
		pageSize int
		wantErr  bool
	}{
		{
			name:     "valid pagination",
			page:     1,
			pageSize: 20,
			wantErr:  false,
		},
		{
			name:     "page zero",
			page:     0,
			pageSize: 20,
			wantErr:  true,
		},
		{
			name:     "negative page",
			page:     -1,
			pageSize: 20,
			wantErr:  true,
		},
		{
			name:     "page size zero",
			page:     1,
			pageSize: 0,
			wantErr:  true,
		},
		{
			name:     "page size too large",
			page:     1,
			pageSize: 101,
			wantErr:  true,
		},
		{
			name:     "page size at max",
			page:     1,
			pageSize: 100,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePaginationParams(tt.page, tt.pageSize)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePaginationParams(%d, %d) error = %v, wantErr %v",
					tt.page, tt.pageSize, err, tt.wantErr)
			}
		})
	}
}

func TestValidateSortParams(t *testing.T) {
	validator := NewAssetValidator()

	tests := []struct {
		name      string
		sortBy    string
		sortOrder string
		wantErr   bool
	}{
		{
			name:      "valid sort by name asc",
			sortBy:    "name",
			sortOrder: "asc",
			wantErr:   false,
		},
		{
			name:      "valid sort by created_at desc",
			sortBy:    "created_at",
			sortOrder: "desc",
			wantErr:   false,
		},
		{
			name:      "empty sort params",
			sortBy:    "",
			sortOrder: "",
			wantErr:   false,
		},
		{
			name:      "invalid sort field",
			sortBy:    "invalid",
			sortOrder: "asc",
			wantErr:   true,
		},
		{
			name:      "invalid sort order",
			sortBy:    "name",
			sortOrder: "random",
			wantErr:   true,
		},
		{
			name:      "SQL injection attempt in sort field",
			sortBy:    "name; DROP TABLE",
			sortOrder: "asc",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateSortParams(tt.sortBy, tt.sortOrder)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSortParams(%q, %q) error = %v, wantErr %v",
					tt.sortBy, tt.sortOrder, err, tt.wantErr)
			}
		})
	}
}

func TestValidateSearchQuery(t *testing.T) {
	validator := NewAssetValidator()

	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "valid search",
			query:   "example",
			wantErr: false,
		},
		{
			name:    "empty search",
			query:   "",
			wantErr: false,
		},
		{
			name:    "query too long",
			query:   strings.Repeat("a", 256),
			wantErr: true,
		},
		{
			name:    "SQL injection - single quote",
			query:   "' OR '1'='1",
			wantErr: true,
		},
		{
			name:    "SQL injection - comment",
			query:   "example--",
			wantErr: true,
		},
		{
			name:    "SQL injection - semicolon",
			query:   "example; DROP TABLE",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateSearchQuery(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSearchQuery(%q) error = %v, wantErr %v", tt.query, err, tt.wantErr)
			}
		})
	}
}

func BenchmarkValidateDomain(b *testing.B) {
	validator := NewAssetValidator()
	for i := 0; i < b.N; i++ {
		validator.ValidateDomain("api.example.com")
	}
}

func BenchmarkValidateIP(b *testing.B) {
	validator := NewAssetValidator()
	for i := 0; i < b.N; i++ {
		validator.ValidateIP("192.168.1.1")
	}
}
