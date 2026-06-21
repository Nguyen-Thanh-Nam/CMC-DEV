package validator

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"

	"asset-api/internal/model"
)

type AssetValidator struct{}

func NewAssetValidator() *AssetValidator {
	return &AssetValidator{}
}

func (v *AssetValidator) ValidateCreate(name, assetType string) error {
	if err := v.ValidateName(name); err != nil {
		return err
	}

	if err := v.ValidateType(assetType); err != nil {
		return err
	}

	switch assetType {
	case model.TypeDomain:
		if err := v.ValidateDomain(name); err != nil {
			return fmt.Errorf("invalid domain: %w", err)
		}
	case model.TypeIP:
		if err := v.ValidateIP(name); err != nil {
			return fmt.Errorf("invalid IP address: %w", err)
		}
	case model.TypeService:
		if err := v.ValidateService(name); err != nil {
			return fmt.Errorf("invalid service: %w", err)
		}
	}

	return nil
}

func (v *AssetValidator) ValidateUpdate(name, assetType, status string) error {
	if name != "" {
		if err := v.ValidateName(name); err != nil {
			return err
		}

		switch assetType {
		case model.TypeDomain:
			if err := v.ValidateDomain(name); err != nil {
				return fmt.Errorf("invalid domain: %w", err)
			}
		case model.TypeIP:
			if err := v.ValidateIP(name); err != nil {
				return fmt.Errorf("invalid IP address: %w", err)
			}
		case model.TypeService:
			if err := v.ValidateService(name); err != nil {
				return fmt.Errorf("invalid service: %w", err)
			}
		}
	}

	if assetType != "" {
		if err := v.ValidateType(assetType); err != nil {
			return err
		}
	}

	if status != "" {
		if err := v.ValidateStatus(status); err != nil {
			return err
		}
	}

	return nil
}

func (v *AssetValidator) ValidateName(name string) error {
	if name == "" {
		return errors.New("name is required")
	}

	if len(name) > 255 {
		return errors.New("name too long (max 255 characters)")
	}

	if strings.Contains(name, "\x00") {
		return errors.New("name contains invalid characters")
	}

	return nil
}

func (v *AssetValidator) ValidateType(assetType string) error {
	if !model.IsValidType(assetType) {
		return fmt.Errorf("invalid asset type: must be %s, %s, or %s",
			model.TypeDomain, model.TypeIP, model.TypeService)
	}
	return nil
}

func (v *AssetValidator) ValidateStatus(status string) error {
	if !model.IsValidStatus(status) {
		return fmt.Errorf("invalid status: must be %s or %s",
			model.StatusActive, model.StatusInactive)
	}
	return nil
}

func (v *AssetValidator) ValidateDomain(domain string) error {
	if len(domain) < 1 || len(domain) > 253 {
		return errors.New("domain length must be between 1 and 253 characters")
	}

	if net.ParseIP(domain) != nil {
		return errors.New("IP address should use type 'ip' instead of 'domain'")
	}

	domainRegex := regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)*[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?$`)

	if !domainRegex.MatchString(domain) {
		return errors.New("invalid domain format (e.g., example.com)")
	}

	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return errors.New("domain cannot start or end with a dot")
	}

	if strings.HasPrefix(domain, "-") || strings.HasSuffix(domain, "-") {
		return errors.New("domain cannot start or end with a hyphen")
	}

	return nil
}

func (v *AssetValidator) ValidateIP(ip string) error {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return errors.New("invalid IP address format (e.g., 192.168.1.1 or 2001:db8::1)")
	}

	return nil
}

func (v *AssetValidator) ValidateService(service string) error {
	if len(service) < 1 {
		return errors.New("service name is required")
	}

	serviceRegex := regexp.MustCompile(`^([a-zA-Z0-9\-]+)(://[a-zA-Z0-9\-\.]+)?(:[\d]+)?(/.*)?$`)

	if !serviceRegex.MatchString(service) {
		return errors.New("invalid service format (e.g., http://example.com or ssh)")
	}

	return nil
}

func (v *AssetValidator) ValidatePaginationParams(page, pageSize int) error {
	if page < 1 {
		return errors.New("page must be >= 1")
	}

	if pageSize < 1 {
		return errors.New("page_size must be >= 1")
	}

	if pageSize > 100 {
		return errors.New("page_size too large (max 100)")
	}

	return nil
}

func (v *AssetValidator) ValidateSortParams(sortBy, sortOrder string) error {
	validSortFields := map[string]bool{
		"name":       true,
		"type":       true,
		"status":     true,
		"created_at": true,
		"updated_at": true,
	}

	if sortBy != "" && !validSortFields[sortBy] {
		return fmt.Errorf("invalid sort field: %s (allowed: name, type, status, created_at, updated_at)", sortBy)
	}

	if sortOrder != "" && sortOrder != "asc" && sortOrder != "desc" {
		return errors.New("sort order must be 'asc' or 'desc'")
	}

	return nil
}

func (v *AssetValidator) ValidateSearchQuery(query string) error {
	if len(query) > 255 {
		return errors.New("search query too long (max 255 characters)")
	}

	dangerousPatterns := []string{
		"'", "\"", ";", "--", "/*", "*/", "xp_", "sp_",
	}

	lowerQuery := strings.ToLower(query)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerQuery, pattern) {
			return fmt.Errorf("search query contains invalid characters: %s", pattern)
		}
	}

	return nil
}
