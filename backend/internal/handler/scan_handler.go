// Package handler chứa các HTTP handler để xử lý các yêu cầu (request) từ client.
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"asset-api/internal/service"
	"asset-api/internal/model"
)

// ScanHandler chịu trách nhiệm tiếp nhận và xử lý các yêu cầu HTTP liên quan đến hoạt động quét tài sản (asset scanning).
// Nó đóng vai trò là tầng giao tiếp (controller/handler) điều hướng yêu cầu tới ScanService để thực hiện logic nghiệp vụ.
type ScanHandler struct {
	svc *service.ScanService
}

// NewScanHandler khởi tạo một thực thể mới của ScanHandler với một ScanService được tiêm (inject) vào.
func NewScanHandler(svc *service.ScanService) *ScanHandler {
	return &ScanHandler{svc: svc}
}

// StartScan xử lý yêu cầu kích hoạt một tiến trình quét (scan job) mới cho một tài sản cụ thể.
// Nó mong đợi ID của tài sản nằm trong path parameter (ví dụ: /assets/{id}/scan) và loại quét (scan type) trong JSON request body.
func (h *ScanHandler) StartScan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "asset id is required")
		return
	}

	var input model.StartScanInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.ScanType == "" {
		input.ScanType = "dns"
	}

	job, err := h.svc.StartScan(r.Context(), id, input.ScanType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, job)
}

// GetJob lấy thông tin chi tiết và trạng thái hiện tại của một công việc quét (scan job) cụ thể theo ID của job đó.
func (h *ScanHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "job id is required")
		return
	}

	job, err := h.svc.GetScanJob(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "scan job not found")
		return
	}

	writeJSON(w, http.StatusOK, job)
}

// GetJobResults lấy các kết quả thu được từ một công việc quét (scan job) cụ thể.
func (h *ScanHandler) GetJobResults(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "job id is required")
		return
	}

	results, err := h.svc.GetScanJobResults(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"job_id":  id,
		"results": results,
	})
}

// ListJobsForAsset liệt kê tất cả các công việc quét (cả đang chạy, đã xong hoặc bị lỗi) liên quan đến một tài sản cụ thể.
func (h *ScanHandler) ListJobsForAsset(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "asset id is required")
		return
	}

	jobs, err := h.svc.ListScanJobsForAsset(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if jobs == nil {
		jobs = []model.ScanJob{}
	}

	writeJSON(w, http.StatusOK, jobs)
}

// GetResultsForAsset lấy tất cả các kết quả quét (scan results) được ghi nhận cho một tài sản cụ thể qua mọi lần quét.
func (h *ScanHandler) GetResultsForAsset(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "asset id is required")
		return
	}

	results, err := h.svc.GetScanResultsForAsset(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if results == nil {
		results = []model.ScanResult{}
	}

	writeJSON(w, http.StatusOK, results)
}

// ListAll xử lý yêu cầu lấy danh sách tất cả các tiến trình quét trên hệ thống (giới hạn tối đa 100 tác vụ mới nhất).
func (h *ScanHandler) ListAll(w http.ResponseWriter, r *http.Request) {
	limit := 100
	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 && val <= 100 {
			limit = val
		}
	}

	jobs, err := h.svc.ListAllScanJobs(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if jobs == nil {
		jobs = []model.ScanJob{}
	}

	writeJSON(w, http.StatusOK, jobs)
}

// GetAssetDNS lấy các kết quả quét DNS của một tài sản cụ thể.
func (h *ScanHandler) GetAssetDNS(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "asset id is required")
		return
	}

	jobs, err := h.svc.ListScanJobsForAsset(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	results, err := h.svc.GetScanResultsForAsset(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	dnsJobs := make(map[string]bool)
	for _, j := range jobs {
		if j.ScanType == model.ScanTypeDNS || j.ScanType == model.ScanTypeAll {
			dnsJobs[j.ID] = true
		}
	}

	var filtered []model.ScanResult
	for _, res := range results {
		if dnsJobs[res.JobID] {
			filtered = append(filtered, res)
		}
	}

	if filtered == nil {
		filtered = []model.ScanResult{}
	}
	writeJSON(w, http.StatusOK, filtered)
}

// GetAssetWHOIS lấy các kết quả quét WHOIS của một tài sản cụ thể.
func (h *ScanHandler) GetAssetWHOIS(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "asset id is required")
		return
	}

	jobs, err := h.svc.ListScanJobsForAsset(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	results, err := h.svc.GetScanResultsForAsset(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	whoisJobs := make(map[string]bool)
	for _, j := range jobs {
		if j.ScanType == model.ScanTypeWHOIS || j.ScanType == model.ScanTypeAll {
			whoisJobs[j.ID] = true
		}
	}

	var filtered []model.ScanResult
	for _, res := range results {
		if whoisJobs[res.JobID] {
			filtered = append(filtered, res)
		}
	}

	if filtered == nil {
		filtered = []model.ScanResult{}
	}
	writeJSON(w, http.StatusOK, filtered)
}

// GetAssetSubdomains lấy các kết quả quét subdomain của một tài sản cụ thể.
func (h *ScanHandler) GetAssetSubdomains(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "asset id is required")
		return
	}

	jobs, err := h.svc.ListScanJobsForAsset(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	results, err := h.svc.GetScanResultsForAsset(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	subdomainJobs := make(map[string]bool)
	for _, j := range jobs {
		if j.ScanType == model.ScanTypeSubdomain || j.ScanType == model.ScanTypeAll {
			subdomainJobs[j.ID] = true
		}
	}

	var filtered []model.ScanResult
	for _, res := range results {
		if subdomainJobs[res.JobID] {
			filtered = append(filtered, res)
		}
	}

	if filtered == nil {
		filtered = []model.ScanResult{}
	}
	writeJSON(w, http.StatusOK, filtered)
}

