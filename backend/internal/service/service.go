package service

import (
	"context"

	"asset-api/internal/model"
)

type AssetServiceInterface interface {
	Create(ctx context.Context, input model.CreateAssetInput) (*model.Asset, error)
	BatchCreate(ctx context.Context, req model.BatchCreateRequest) (*model.BatchCreateResponse, error)
	GetByID(ctx context.Context, id string) (*model.Asset, error)
	BatchDelete(ctx context.Context, ids []string) (*model.BatchDeleteResponse, error)
	GetStats(ctx context.Context) (*model.AssetStats, error)
	Count(ctx context.Context, filter model.AssetFilter) (*model.CountResult, error)
	List(ctx context.Context, filter model.AssetFilter, page, limit int) (*model.PaginatedAssetsResponse, error)
	Search(ctx context.Context, query string) ([]model.Asset, error)
}

type HealthServiceInterface interface {
	Check(ctx context.Context) *model.HealthResponse
}

type ScanServiceInterface interface {
	StartScan(ctx context.Context, assetID, scanType string) (*model.ScanJob, error)
	GetScanJob(ctx context.Context, jobID string) (*model.ScanJob, error)
	GetScanJobResults(ctx context.Context, jobID string) ([]model.ScanResult, error)
	ListScanJobsForAsset(ctx context.Context, assetID string) ([]model.ScanJob, error)
	GetScanResultsForAsset(ctx context.Context, assetID string) ([]model.ScanResult, error)
}
