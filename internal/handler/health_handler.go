package handler

import (
	"net/http"

	"asset-api/internal/service"
)

type HealthHandler struct {
	svc *service.HealthService
}

func NewHealthHandler(svc *service.HealthService) *HealthHandler {
	return &HealthHandler{svc: svc}
}

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	resp := h.svc.Check(r.Context())
	writeJSON(w, http.StatusOK, resp)
}
