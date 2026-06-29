package model

import "time"

type AssetType string

const (
	AssetTypeDomain  AssetType = "domain"
	AssetTypeIP      AssetType = "ip"
	AssetTypeService AssetType = "service"
)

type AssetStatus string

const (
	AssetStatusActive   AssetStatus = "active"
	AssetStatusInactive AssetStatus = "inactive"
)

const (
	TypeDomain  = "domain"
	TypeIP      = "ip"
	TypeService = "service"
)

const (
	StatusActive   = "active"
	StatusInactive = "inactive"
)

func IsValidType(t string) bool {
	return t == TypeDomain || t == TypeIP || t == TypeService
}

func IsValidStatus(s string) bool {
	return s == StatusActive || s == StatusInactive
}

type Asset struct {
	ID        string      `json:"id" csv:"id"`
	Name      string      `json:"name" csv:"name"`
	Type      AssetType   `json:"type" csv:"type"`
	Status    AssetStatus `json:"status" csv:"status"`
	Tags      string      `json:"tags" csv:"tags"`
	CreatedAt time.Time   `json:"created_at" csv:"created_at"`
	DeletedAt *time.Time  `json:"deleted_at,omitempty" csv:"-"`
}

type AssetFilter struct {
	Type   string
	Status string
	Tag    string
}

type AssetStats struct {
	Total    int            `json:"total"`
	ByType   map[string]int `json:"by_type"`
	ByStatus map[string]int `json:"by_status"`
}

type CountResult struct {
	Count   int               `json:"count"`
	Filters map[string]string `json:"filters"`
}

type BatchCreateRequest struct {
	Assets []CreateAssetInput `json:"assets"`
}

type CreateAssetInput struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status,omitempty"`
	Tags   string `json:"tags,omitempty"`
}

type BatchCreateResponse struct {
	Created int      `json:"created"`
	IDs     []string `json:"ids"`
}

type BatchDeleteResponse struct {
	Deleted  int `json:"deleted"`
	NotFound int `json:"not_found"`
}

type PaginationMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type PaginatedAssetsResponse struct {
	Data       []Asset        `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

type HealthResponse struct {
	Status        string        `json:"status"`
	Storage       StorageHealth `json:"storage"`
	UptimeSeconds int64         `json:"uptime_seconds"`
	Timestamp     string        `json:"timestamp"`
}

type StorageHealth struct {
	Type       string `json:"type"`
	AssetCount int    `json:"asset_count"`
}

func ValidAssetType(t string) bool {
	switch AssetType(t) {
	case AssetTypeDomain, AssetTypeIP, AssetTypeService:
		return true
	default:
		return false
	}
}

func ValidAssetStatus(s string) bool {
	switch AssetStatus(s) {
	case AssetStatusActive, AssetStatusInactive:
		return true
	default:
		return false
	}
}
