package handler

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"asset-api/internal/apperrors"
	"asset-api/internal/model"
	"asset-api/internal/service"
)

type AssetHandler struct {
	svc *service.AssetService
}

func NewAssetHandler(svc *service.AssetService) *AssetHandler {
	return &AssetHandler{svc: svc}
}

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

func (h *AssetHandler) BatchDelete(w http.ResponseWriter, r *http.Request) {
	idsParam := r.URL.Query().Get("ids")
	if strings.TrimSpace(idsParam) == "" {
		writeError(w, http.StatusBadRequest, apperrors.ErrMissingIDs.Error())
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

func (h *AssetHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GetStats(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

func (h *AssetHandler) Count(w http.ResponseWriter, r *http.Request) {
	filter := parseFilter(r)
	result, err := h.svc.Count(r.Context(), filter)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

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

func (h *AssetHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	results, err := h.svc.Search(r.Context(), query)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, results)
}

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
			continue // skip header
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

func parseFilter(r *http.Request) model.AssetFilter {
	return model.AssetFilter{
		Type:   r.URL.Query().Get("type"),
		Status: r.URL.Query().Get("status"),
	}
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, apperrors.ErrInvalidAssetType),
		errors.Is(err, apperrors.ErrInvalidAssetStatus),
		errors.Is(err, apperrors.ErrEmptyName),
		errors.Is(err, apperrors.ErrEmptyAssets),
		errors.Is(err, apperrors.ErrBatchLimit),
		errors.Is(err, apperrors.ErrMissingIDs),
		errors.Is(err, apperrors.ErrMissingQuery):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		var validationErr *apperrors.ValidationError
		if errors.As(err, &validationErr) {
			writeError(w, http.StatusBadRequest, validationErr.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

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
