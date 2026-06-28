package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"asset-api/internal/apperrors"
	"asset-api/internal/model"
	"asset-api/internal/storage"
)

const (
	maxBatchSize  = 100
	maxSearchSize = 100
	defaultLimit  = 20
	maxLimit      = 100
)

type cacheEntry struct {
	data *model.PaginatedAssetsResponse
	exp  time.Time
}

type AssetService struct {
	storage storage.AssetStorage
	audit   *log.Logger
	cache   map[string]cacheEntry
	cacheMu sync.RWMutex
}

func NewAssetService(s storage.AssetStorage) *AssetService {
	// Fallback to stderr if audit.log cannot be opened (e.g. permission issues)
	f, err := os.OpenFile("audit.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		f = os.Stderr
	}
	logger := log.New(f, "AUDIT: ", log.Ldate|log.Ltime)
	return &AssetService{
		storage: s,
		audit:   logger,
		cache:   make(map[string]cacheEntry),
	}
}

func (svc *AssetService) clearCache() {
	svc.cacheMu.Lock()
	defer svc.cacheMu.Unlock()
	svc.cache = make(map[string]cacheEntry)
}

func triggerWebhook(asset *model.Asset) {
	payload, _ := json.Marshal(asset)
	go http.Post("http://localhost:8081/webhook-test", "application/json", bytes.NewBuffer(payload))
}

func (svc *AssetService) Create(ctx context.Context, input model.CreateAssetInput) (*model.Asset, error) {
	if err := validateCreateInput(input); err != nil {
		return nil, err
	}

	asset := toAsset(input)
	if err := svc.storage.Create(ctx, asset); err != nil {
		return nil, err
	}
	
	svc.audit.Printf("Action: CREATE, AssetID: %s, Name: %s", asset.ID, asset.Name)
	triggerWebhook(asset)
	svc.clearCache()
	
	return asset, nil
}

func (svc *AssetService) BatchCreate(ctx context.Context, req model.BatchCreateRequest) (*model.BatchCreateResponse, error) {
	if len(req.Assets) == 0 {
		return nil, apperrors.ErrEmptyAssets
	}
	if len(req.Assets) > maxBatchSize {
		return nil, apperrors.ErrBatchLimit
	}

	for i, input := range req.Assets {
		if err := validateCreateInput(input); err != nil {
			return nil, apperrors.NewValidationError("assets[" + strconv.Itoa(i) + "]: " + err.Error())
		}
	}

	created := make([]model.Asset, 0, len(req.Assets))
	ids := make([]string, 0, len(req.Assets))
	for _, input := range req.Assets {
		asset := toAsset(input)
		created = append(created, *asset)
		ids = append(ids, asset.ID)
	}

	if err := svc.storage.BatchCreate(ctx, created); err != nil {
		return nil, err
	}

	svc.audit.Printf("Action: BATCH_CREATE, Count: %d", len(ids))
	for i := range created {
		triggerWebhook(&created[i])
	}
	svc.clearCache()

	return &model.BatchCreateResponse{
		Created: len(ids),
		IDs:     ids,
	}, nil
}

func (svc *AssetService) GetByID(ctx context.Context, id string) (*model.Asset, error) {
	return svc.storage.GetByID(ctx, id)
}

func (svc *AssetService) BatchDelete(ctx context.Context, ids []string) (*model.BatchDeleteResponse, error) {
	if len(ids) == 0 {
		return nil, apperrors.ErrMissingIDs
	}

	deleted, notFound, err := svc.storage.BatchDelete(ctx, ids)
	if err != nil {
		return nil, err
	}

	svc.audit.Printf("Action: BATCH_DELETE, Deleted: %d, NotFound: %d", deleted, notFound)
	if deleted > 0 {
		svc.clearCache()
	}

	return &model.BatchDeleteResponse{
		Deleted:  deleted,
		NotFound: notFound,
	}, nil
}

func (svc *AssetService) GetStats(ctx context.Context) (*model.AssetStats, error) {
	return svc.storage.GetStats(ctx)
}

func (svc *AssetService) Count(ctx context.Context, filter model.AssetFilter) (*model.CountResult, error) {
	if filter.Type != "" && !model.ValidAssetType(filter.Type) {
		return nil, apperrors.ErrInvalidAssetType
	}
	if filter.Status != "" && !model.ValidAssetStatus(filter.Status) {
		return nil, apperrors.ErrInvalidAssetStatus
	}

	count, err := svc.storage.Count(ctx, filter)
	if err != nil {
		return nil, err
	}

	filters := make(map[string]string)
	if filter.Type != "" {
		filters["type"] = filter.Type
	}
	if filter.Status != "" {
		filters["status"] = filter.Status
	}

	return &model.CountResult{
		Count:   count,
		Filters: filters,
	}, nil
}

func (svc *AssetService) List(ctx context.Context, filter model.AssetFilter, page, limit int) (*model.PaginatedAssetsResponse, error) {
	if filter.Type != "" && !model.ValidAssetType(filter.Type) {
		return nil, apperrors.ErrInvalidAssetType
	}
	if filter.Status != "" && !model.ValidAssetStatus(filter.Status) {
		return nil, apperrors.ErrInvalidAssetStatus
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	cacheKey := fmt.Sprintf("%s_%s_%d_%d", filter.Type, filter.Status, page, limit)
	svc.cacheMu.RLock()
	if entry, ok := svc.cache[cacheKey]; ok && time.Now().Before(entry.exp) {
		svc.cacheMu.RUnlock()
		return entry.data, nil
	}
	svc.cacheMu.RUnlock()

	data, total, err := svc.storage.List(ctx, filter, page, limit)
	if err != nil {
		return nil, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	if data == nil {
		data = []model.Asset{}
	}

	resp := &model.PaginatedAssetsResponse{
		Data: data,
		Pagination: model.PaginationMeta{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}
	
	svc.cacheMu.Lock()
	svc.cache[cacheKey] = cacheEntry{
		data: resp,
		exp:  time.Now().Add(5 * time.Minute),
	}
	svc.cacheMu.Unlock()
	
	return resp, nil
}

func (svc *AssetService) Search(ctx context.Context, query string) ([]model.Asset, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, apperrors.ErrMissingQuery
	}

	results, err := svc.storage.Search(ctx, query, maxSearchSize)
	if err != nil {
		return nil, err
	}
	if results == nil {
		results = []model.Asset{}
	}
	return results, nil
}

func validateCreateInput(input model.CreateAssetInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return apperrors.ErrEmptyName
	}
	if !model.ValidAssetType(input.Type) {
		return apperrors.ErrInvalidAssetType
	}
	if input.Status != "" && !model.ValidAssetStatus(input.Status) {
		return apperrors.ErrInvalidAssetStatus
	}
	return nil
}

func toAsset(input model.CreateAssetInput) *model.Asset {
	status := model.AssetStatusActive
	if input.Status != "" {
		status = model.AssetStatus(input.Status)
	}

	return &model.Asset{
		ID:        uuid.NewString(),
		Name:      strings.TrimSpace(input.Name),
		Type:      model.AssetType(input.Type),
		Status:    status,
		CreatedAt: time.Now().UTC(),
	}
}
