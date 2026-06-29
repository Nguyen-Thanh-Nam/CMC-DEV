package model_test

import (
	"testing"

	"asset-api/internal/model"
)

func TestValidAssetType(t *testing.T) {
	tests := []struct {
		name      string
		assetType string
		want      bool
	}{
		{"valid domain", "domain", true},
		{"valid ip", "ip", true},
		{"valid service", "service", true},
		{"invalid type empty", "", false},
		{"invalid type string", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := model.ValidAssetType(tt.assetType); got != tt.want {
				t.Errorf("ValidAssetType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidAssetStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"valid active", "active", true},
		{"valid inactive", "inactive", true},
		{"invalid status empty", "", false},
		{"invalid status string", "suspended", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := model.ValidAssetStatus(tt.status); got != tt.want {
				t.Errorf("ValidAssetStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}
