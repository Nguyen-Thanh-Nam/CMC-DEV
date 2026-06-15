package service_test

import (
	"context"
	"errors"
	"testing"

	"asset-api/internal/apperrors"
	"asset-api/internal/model"
	"asset-api/internal/service"
	"asset-api/internal/storage"
)

func TestAssetService_BatchCreateAllOrNothing(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStorage()
	svc := service.NewAssetService(store)
	ctx := context.Background()

	_, err := svc.BatchCreate(ctx, model.BatchCreateRequest{
		Assets: []model.CreateAssetInput{
			{Name: "ok.com", Type: "domain"},
			{Name: "bad.com", Type: "invalid_type"},
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}

	count, err := store.Count(ctx, model.AssetFilter{})
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 assets after failed batch, got %d", count)
	}
}

func TestAssetService_BatchCreateSuccess(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStorage()
	svc := service.NewAssetService(store)
	ctx := context.Background()

	resp, err := svc.BatchCreate(ctx, model.BatchCreateRequest{
		Assets: []model.CreateAssetInput{
			{Name: "a.com", Type: "domain"},
			{Name: "b.com", Type: "domain"},
		},
	})
	if err != nil {
		t.Fatalf("batch create failed: %v", err)
	}
	if resp.Created != 2 || len(resp.IDs) != 2 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestAssetService_Search(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStorage()
	svc := service.NewAssetService(store)
	ctx := context.Background()

	_, _ = svc.BatchCreate(ctx, model.BatchCreateRequest{
		Assets: []model.CreateAssetInput{
			{Name: "example.com", Type: "domain"},
			{Name: "other.org", Type: "domain"},
		},
	})

	results, err := svc.Search(ctx, "EXAMPLE")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) != 1 || results[0].Name != "example.com" {
		t.Fatalf("unexpected search results: %+v", results)
	}

	_, err = svc.Search(ctx, "")
	if !errors.Is(err, apperrors.ErrMissingQuery) {
		t.Fatalf("expected missing query error, got %v", err)
	}
}
