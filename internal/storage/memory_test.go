package storage_test

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"

	"asset-api/internal/model"
	"asset-api/internal/storage"
)

func TestMemoryStorage_ConcurrentCreate(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStorage()
	ctx := context.Background()
	const n = 50

	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			asset := &model.Asset{
				ID:     uuid.NewString(),
				Name:   "concurrent-test.com",
				Type:   model.AssetTypeDomain,
				Status: model.AssetStatusActive,
			}
			if err := store.Create(ctx, asset); err != nil {
				t.Errorf("create failed: %v", err)
			}
		}(i)
	}

	wg.Wait()

	if got := store.CountAll(ctx); got != n {
		t.Fatalf("expected %d assets, got %d", n, got)
	}
}

func TestMemoryStorage_BatchDelete(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStorage()
	ctx := context.Background()

	id1 := uuid.NewString()
	id2 := uuid.NewString()
	_ = store.BatchCreate(ctx, []model.Asset{
		{ID: id1, Name: "a.com", Type: model.AssetTypeDomain, Status: model.AssetStatusActive},
		{ID: id2, Name: "b.com", Type: model.AssetTypeDomain, Status: model.AssetStatusActive},
	})

	deleted, notFound, err := store.BatchDelete(ctx, []string{id1, id2, "missing-id"})
	if err != nil {
		t.Fatalf("batch delete failed: %v", err)
	}
	if deleted != 2 || notFound != 1 {
		t.Fatalf("expected deleted=2 notFound=1, got deleted=%d notFound=%d", deleted, notFound)
	}
}

func TestMemoryStorage_GetStats(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStorage()
	ctx := context.Background()

	_ = store.BatchCreate(ctx, []model.Asset{
		{ID: uuid.NewString(), Name: "d1.com", Type: model.AssetTypeDomain, Status: model.AssetStatusActive},
		{ID: uuid.NewString(), Name: "1.1.1.1", Type: model.AssetTypeIP, Status: model.AssetStatusInactive},
	})

	stats, err := store.GetStats(ctx)
	if err != nil {
		t.Fatalf("get stats failed: %v", err)
	}
	if stats.Total != 2 {
		t.Fatalf("expected total 2, got %d", stats.Total)
	}
	if stats.ByType["domain"] != 1 || stats.ByType["ip"] != 1 {
		t.Fatalf("unexpected by_type: %+v", stats.ByType)
	}
}
