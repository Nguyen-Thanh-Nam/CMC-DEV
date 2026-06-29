// Package handler chứa các HTTP handler để xử lý các yêu cầu (request) từ client.
package handler

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"asset-api/internal/model"
	"asset-api/internal/service"
)

// AssetHandler quản lý tất cả các yêu cầu HTTP liên quan đến tài sản (assets).
// Handler này đóng vai trò cầu nối, tiếp nhận yêu cầu từ client, gọi tầng nghiệp vụ AssetService
// để xử lý và trả về phản hồi tương ứng cho client.
type AssetHandler struct {
	svc *service.AssetService
}

// NewAssetHandler là hàm khởi tạo (constructor) cho AssetHandler, nhận vào thực thể AssetService.
func NewAssetHandler(svc *service.AssetService) *AssetHandler {
	return &AssetHandler{svc: svc}
}

// Create xử lý yêu cầu tạo một tài sản mới từ dữ liệu JSON gửi lên.
func (h *AssetHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input model.CreateAssetInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	asset, err := h.svc.Create(r.Context(), input)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, asset)
}

// GetByID xử lý yêu cầu truy vấn thông tin chi tiết của một tài sản dựa trên ID được truyền qua URL path.
func (h *AssetHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	asset, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, asset)
}

// BatchCreate cho phép tạo đồng thời nhiều tài sản thông qua một yêu cầu JSON (dưới dạng mảng).
func (h *AssetHandler) BatchCreate(w http.ResponseWriter, r *http.Request) {
	var req model.BatchCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	resp, err := h.svc.BatchCreate(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// BatchDelete xử lý việc xóa hàng loạt tài sản dựa trên danh sách ID được phân tách bởi dấu phẩy (comma-separated IDs) qua query parameter.
func (h *AssetHandler) BatchDelete(w http.ResponseWriter, r *http.Request) {
	idsParam := r.URL.Query().Get("ids")
	if strings.TrimSpace(idsParam) == "" {
		writeError(w, http.StatusBadRequest, model.ErrMissingIDs.Error())
		return
	}

	ids := strings.Split(idsParam, ",")
	resp, err := h.svc.BatchDelete(r.Context(), ids)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetStats xử lý yêu cầu lấy thông tin thống kê chung về các tài sản trong hệ thống (ví dụ: tổng số lượng theo trạng thái, loại tài sản).
func (h *AssetHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GetStats(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// Count tính tổng số lượng tài sản khớp với các điều kiện lọc (filter) được truyền qua query parameters.
func (h *AssetHandler) Count(w http.ResponseWriter, r *http.Request) {
	filter := parseFilter(r)
	result, err := h.svc.Count(r.Context(), filter)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// List trả về danh sách tài sản có phân trang (pagination) và lọc (filtering) theo các tiêu chí từ query parameters.
func (h *AssetHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := parseFilter(r)
	page := parseIntDefault(r.URL.Query().Get("page"), 1)
	limit := parseIntDefault(r.URL.Query().Get("limit"), 20)

	resp, err := h.svc.List(r.Context(), filter, page, limit)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// Search xử lý việc tìm kiếm toàn văn hoặc khớp một phần các tài sản theo từ khóa tìm kiếm được truyền qua tham số query "q".
func (h *AssetHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	results, err := h.svc.Search(r.Context(), query)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, results)
}

// ExportCSV xuất danh sách tài sản ra định dạng tệp CSV. Nó tải tối đa 1000 tài sản khớp với điều kiện lọc và ghi trực tiếp vào phản hồi HTTP.
func (h *AssetHandler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	filter := parseFilter(r)
	resp, err := h.svc.List(r.Context(), filter, 1, 1000)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment;filename=assets.csv")
	
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"id", "name", "type", "status", "created_at"})
	
	for _, a := range resp.Data {
		_ = writer.Write([]string{
			a.ID, a.Name, string(a.Type), string(a.Status), a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	writer.Flush()
}

// ImportCSV tiếp nhận một tệp CSV được upload thông qua multipart/form-data, phân tích cú pháp tệp CSV này,
// và thực hiện tạo hàng loạt tài sản (batch create) từ dữ liệu đọc được.
func (h *AssetHandler) ImportCSV(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file in form data")
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid csv format")
		return
	}

	if len(records) <= 1 {
		writeError(w, http.StatusBadRequest, "empty csv or no data rows")
		return
	}

	var req model.BatchCreateRequest
	for i, row := range records {
		if i == 0 {
			continue
		}
		if len(row) < 2 {
			continue
		}
		status := ""
		if len(row) >= 3 {
			status = row[2]
		}
		req.Assets = append(req.Assets, model.CreateAssetInput{
			Name:   row[0],
			Type:   row[1],
			Status: status,
		})
	}

	resp, err := h.svc.BatchCreate(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// parseFilter là một hàm trợ giúp (helper) trích xuất các điều kiện lọc tài sản từ URL query parameters.
func parseFilter(r *http.Request) model.AssetFilter {
	return model.AssetFilter{
		Type:   r.URL.Query().Get("type"),
		Status: r.URL.Query().Get("status"),
		Tag:    r.URL.Query().Get("tag"),
	}
}

// writeServiceError phân tích lỗi trả về từ tầng nghiệp vụ (Service) để phản hồi mã trạng thái HTTP phù hợp.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, model.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, model.ErrInvalidAssetType),
		errors.Is(err, model.ErrInvalidAssetStatus),
		errors.Is(err, model.ErrEmptyName),
		errors.Is(err, model.ErrEmptyAssets),
		errors.Is(err, model.ErrBatchLimit),
		errors.Is(err, model.ErrMissingIDs),
		errors.Is(err, model.ErrMissingQuery):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		var validationErr *model.ValidationError
		if errors.As(err, &validationErr) {
			writeError(w, http.StatusBadRequest, validationErr.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

// writeJSON viết phản hồi dạng JSON về client với mã HTTP status tương ứng.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// writeError là hàm trợ giúp để định dạng và ghi lỗi dưới dạng JSON: {"error": "thông điệp lỗi"}.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// parseIntDefault chuyển đổi một chuỗi số nguyên sang kiểu dữ liệu int.
// Nếu chuỗi rỗng, chứa ký tự không phải số, hoặc giá trị bé hơn 1, nó sẽ trả về giá trị mặc định được cung cấp (defaultVal).
func parseIntDefault(raw string, defaultVal int) int {
	if raw == "" {
		return defaultVal
	}
	val := 0
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return defaultVal
		}
		val = val*10 + int(ch-'0')
	}
	if val < 1 {
		return defaultVal
	}
	return val
}

