package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"asset-api/internal/model"
	"asset-api/internal/storage"
)

// HealthService cung cấp dịch vụ kiểm tra sức khỏe (health check) của ứng dụng,
// giúp theo dõi trạng thái hoạt động của hệ thống, bộ lưu trữ và thời gian hoạt động (uptime).
type HealthService struct {
	storage   storage.AssetStorage // Đối tượng lưu trữ tài sản dùng để truy vấn tổng số lượng tài sản hiện có.
	startTime time.Time            // Thời điểm ứng dụng bắt đầu khởi chạy, dùng để tính toán thời gian uptime (thời gian hoạt động liên tục).
}

// NewHealthService khởi tạo và trả về một instance mới của HealthService.
func NewHealthService(s storage.AssetStorage, startTime time.Time) *HealthService {
	return &HealthService{
		storage:   s,
		startTime: startTime,
	}
}

// Check thực hiện việc kiểm tra trạng thái sức khỏe hiện tại của ứng dụng.
// Nó trả về một cấu trúc model.HealthResponse chứa thông tin về:
// - Trạng thái hệ thống chung (Status: "ok").
// - Sức khỏe của bộ lưu trữ (StorageHealth) bao gồm kiểu dữ liệu và tổng số lượng tài sản.
// - Thời gian hoạt động liên tục của ứng dụng tính bằng giây (UptimeSeconds).
// - Thời gian ghi nhận hiện tại theo chuẩn định dạng RFC3339.
func (svc *HealthService) Check(ctx context.Context) *model.HealthResponse {
	now := time.Now().UTC()
	
	uptime := int64(now.Sub(svc.startTime).Seconds())
	if uptime < 0 {
		uptime = 0
	}

	storageType := "unknown"
	tStr := fmt.Sprintf("%T", svc.storage)
	if strings.Contains(tStr, "mysql") {
		storageType = "mysql"
	} else if strings.Contains(tStr, "memory") {
		storageType = "in-memory"
	}

	return &model.HealthResponse{
		Status: "ok",
		Storage: model.StorageHealth{
			Type:       storageType,
			AssetCount: svc.storage.CountAll(ctx),
		},
		UptimeSeconds: uptime,
		Timestamp:     now.Format(time.RFC3339),
	}
}
