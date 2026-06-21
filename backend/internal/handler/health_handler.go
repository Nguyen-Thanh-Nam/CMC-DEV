// Package handler chứa các HTTP handler để xử lý các yêu cầu (request) từ client.
package handler

import (
	"net/http"

	"asset-api/internal/service"
)

// HealthHandler quản lý các yêu cầu kiểm tra trạng thái hoạt động (health check) của hệ thống.
// Nó phụ thuộc vào HealthService để lấy thông tin trạng thái thực tế.
type HealthHandler struct {
	svc *service.HealthService
}

// NewHealthHandler là hàm khởi tạo (constructor) cho HealthHandler,
// nhận vào một con trỏ tới HealthService và trả về một con trỏ tới HealthHandler.
func NewHealthHandler(svc *service.HealthService) *HealthHandler {
	return &HealthHandler{svc: svc}
}

// Check xử lý yêu cầu HTTP GET để kiểm tra trạng thái sức khỏe của hệ thống.
// Hàm này gọi service.Check để lấy kết quả và ghi phản hồi dưới dạng JSON với mã trạng thái 200 OK.
func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	resp := h.svc.Check(r.Context())
	writeJSON(w, http.StatusOK, resp)
}

