package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"asset-api/internal/handler"
	"asset-api/internal/model"
	"asset-api/internal/service"
	"asset-api/internal/storage/memory"
)

func TestCreateAssetHandler(t *testing.T) {
	// Setup in-memory components
	store := memory.NewMemoryStorage()
	svc := service.NewAssetService(store)
	h := handler.NewAssetHandler(svc)

	// JSON request payload
	body := `{"name":"test-domain.com","type":"domain","tags":"prod,web"}`
	req := httptest.NewRequest("POST", "/assets", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	// Invoke handler
	h.Create(rr, req)

	// Check status code
	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, rr.Code)
	}

	// Parse response
	var res model.Asset
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if res.Name != "test-domain.com" {
		t.Errorf("Expected asset name 'test-domain.com', got '%s'", res.Name)
	}
	if res.Tags != "prod,web" {
		t.Errorf("Expected tags 'prod,web', got '%s'", res.Tags)
	}
}

func TestListAssetsHandler(t *testing.T) {
	store := memory.NewMemoryStorage()
	svc := service.NewAssetService(store)
	h := handler.NewAssetHandler(svc)

	// Pre-create some assets
	ctx := context.Background()
	_, _ = svc.Create(ctx, model.CreateAssetInput{Name: "domain1.com", Type: "domain", Tags: "tag1"})
	_, _ = svc.Create(ctx, model.CreateAssetInput{Name: "domain2.com", Type: "domain", Tags: "tag2"})

	req := httptest.NewRequest("GET", "/assets?limit=5", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	var resp model.PaginatedAssetsResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Pagination.Total != 2 {
		t.Errorf("Expected 2 total assets, got %d", resp.Pagination.Total)
	}
}
