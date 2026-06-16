package storage

import (
	"context"

	"asset-api/internal/model"
)

type AssetStorage interface {
	Create(ctx context.Context, asset *model.Asset) error
	BatchCreate(ctx context.Context, assets []model.Asset) error
	GetByID(ctx context.Context, id string) (*model.Asset, error)
	Delete(ctx context.Context, id string) error
	BatchDelete(ctx context.Context, ids []string) (deleted, notFound int, err error)
	Count(ctx context.Context, filter model.AssetFilter) (int, error)
	GetStats(ctx context.Context) (*model.AssetStats, error)
	List(ctx context.Context, filter model.AssetFilter, page, limit int) ([]model.Asset, int, error)
	Search(ctx context.Context, query string, limit int) ([]model.Asset, error)
	CountAll(ctx context.Context) int
}
