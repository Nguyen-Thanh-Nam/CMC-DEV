package storage

import (
	"context"
	"strings"
	"sync"
	"time"

	"asset-api/internal/apperrors"
	"asset-api/internal/model"
)

type MemoryStorage struct {
	mu     sync.RWMutex
	assets map[string]model.Asset
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		assets: make(map[string]model.Asset),
	}
}

func (s *MemoryStorage) Create(_ context.Context, asset *model.Asset) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.assets[asset.ID] = *asset
	return nil
}

func (s *MemoryStorage) BatchCreate(_ context.Context, assets []model.Asset) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, asset := range assets {
		s.assets[asset.ID] = asset
	}
	return nil
}

func (s *MemoryStorage) GetByID(_ context.Context, id string) (*model.Asset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	asset, ok := s.assets[id]
	if !ok || asset.DeletedAt != nil {
		return nil, apperrors.ErrNotFound
	}
	copy := asset
	return &copy, nil
}

func (s *MemoryStorage) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	asset, ok := s.assets[id]
	if !ok || asset.DeletedAt != nil {
		return apperrors.ErrNotFound
	}
	
	now := time.Now().UTC()
	asset.DeletedAt = &now
	s.assets[id] = asset
	return nil
}

func (s *MemoryStorage) BatchDelete(_ context.Context, ids []string) (deleted, notFound int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		asset, ok := s.assets[id]
		if ok && asset.DeletedAt == nil {
			now := time.Now().UTC()
			asset.DeletedAt = &now
			s.assets[id] = asset
			deleted++
		} else {
			notFound++
		}
	}
	return deleted, notFound, nil
}

func (s *MemoryStorage) Count(_ context.Context, filter model.AssetFilter) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, asset := range s.assets {
		if asset.DeletedAt == nil && matchAsset(asset, filter) {
			count++
		}
	}
	return count, nil
}

func (s *MemoryStorage) GetStats(_ context.Context) (*model.AssetStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &model.AssetStats{
		// Count only non-deleted assets to reflect actual active data
		Total:    0,
		ByType:   map[string]int{"domain": 0, "ip": 0, "service": 0},
		ByStatus: map[string]int{"active": 0, "inactive": 0},
	}

	for _, asset := range s.assets {
		if asset.DeletedAt != nil {
			continue
		}
		stats.Total++
		stats.ByType[string(asset.Type)]++
		stats.ByStatus[string(asset.Status)]++
	}
	return stats, nil
}

func (s *MemoryStorage) List(_ context.Context, filter model.AssetFilter, page, limit int) ([]model.Asset, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	matched := make([]model.Asset, 0)
	for _, asset := range s.assets {
		if asset.DeletedAt == nil && matchAsset(asset, filter) {
			matched = append(matched, asset)
		}
	}

	total := len(matched)
	if total == 0 {
		return []model.Asset{}, 0, nil
	}

	start := (page - 1) * limit
	if start >= total {
		return []model.Asset{}, total, nil
	}

	end := start + limit
	if end > total {
		end = total
	}

	return matched[start:end], total, nil
}

func (s *MemoryStorage) Search(_ context.Context, query string, limit int) ([]model.Asset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query = strings.ToLower(strings.TrimSpace(query))
	results := make([]model.Asset, 0, limit)

	for _, asset := range s.assets {
		if asset.DeletedAt != nil {
			continue
		}
		if strings.Contains(strings.ToLower(asset.Name), query) {
			results = append(results, asset)
			if len(results) >= limit {
				break
			}
		}
	}
	return results, nil
}

func (s *MemoryStorage) CountAll(_ context.Context) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, asset := range s.assets {
		if asset.DeletedAt == nil {
			count++
		}
	}
	return count
}

func matchAsset(a model.Asset, filter model.AssetFilter) bool {
	if filter.Type != "" && string(a.Type) != filter.Type {
		return false
	}
	if filter.Status != "" && string(a.Status) != filter.Status {
		return false
	}
	return true
}
