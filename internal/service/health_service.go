package service

import (
	"context"
	"time"

	"asset-api/internal/model"
	"asset-api/internal/storage"
)

type HealthService struct {
	storage   storage.AssetStorage
	startTime time.Time
}

func NewHealthService(s storage.AssetStorage, startTime time.Time) *HealthService {
	return &HealthService{
		storage:   s,
		startTime: startTime,
	}
}

func (svc *HealthService) Check(ctx context.Context) *model.HealthResponse {
	now := time.Now().UTC()
	uptime := int64(now.Sub(svc.startTime).Seconds())
	if uptime < 0 {
		uptime = 0
	}

	return &model.HealthResponse{
		Status: "ok",
		Storage: model.StorageHealth{
			Type:       "in-memory",
			AssetCount: svc.storage.CountAll(ctx),
		},
		UptimeSeconds: uptime,
		Timestamp:     now.Format(time.RFC3339),
	}
}
