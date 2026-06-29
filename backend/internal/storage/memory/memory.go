package memory

import (
	"context"
	"strings"
	"sync"
	"time"

	"asset-api/internal/model"
)

// MemoryStorage là một triển khai của interface storage.AssetStorage lưu trữ dữ liệu hoàn toàn trong bộ nhớ (RAM).
// Nó sử dụng sync.RWMutex để đảm bảo an toàn luồng (thread-safety), tránh tình trạng data race khi nhiều goroutine
// truy cập đồng thời vào các cấu trúc dữ liệu bản đồ (map).
type MemoryStorage struct {
	mu      sync.RWMutex                // Mutex khóa đọc-ghi để đồng bộ hóa quyền truy cập tài nguyên dùng chung.
	assets  map[string]model.Asset      // Bản đồ lưu trữ danh sách tài sản với mã định danh (ID) làm khóa.
	jobs    map[string]model.ScanJob    // Bản đồ lưu trữ thông tin các tiến trình quét (scan jobs) theo ID.
	results map[string]model.ScanResult // Bản đồ lưu trữ kết quả quét (scan results) theo ID.
}

// NewMemoryStorage khởi tạo một đối tượng MemoryStorage mới cùng với các bản đồ được cấp phát bộ nhớ trống.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		assets:  make(map[string]model.Asset),
		jobs:    make(map[string]model.ScanJob),
		results: make(map[string]model.ScanResult),
	}
}

// Create thêm một tài sản mới vào bộ nhớ. Phương thức này yêu cầu khóa ghi (Write Lock).
func (s *MemoryStorage) Create(_ context.Context, asset *model.Asset) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.assets[asset.ID] = *asset
	return nil
}

// BatchCreate thực hiện lưu trữ hàng loạt tài sản vào bộ nhớ dưới dạng một giao dịch ghi duy nhất.
func (s *MemoryStorage) BatchCreate(_ context.Context, assets []model.Asset) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, asset := range assets {
		s.assets[asset.ID] = asset
	}
	return nil
}

// GetByID truy xuất một tài sản dựa trên ID. Sử dụng khóa đọc (Read Lock) để cho phép nhiều goroutine đọc đồng thời.
func (s *MemoryStorage) GetByID(_ context.Context, id string) (*model.Asset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	asset, ok := s.assets[id]
	if !ok || asset.DeletedAt != nil {
		return nil, model.ErrNotFound
	}
	copy := asset
	return &copy, nil
}

// Delete thực hiện xóa mềm một tài sản bằng cách gắn nhãn thời gian hiện tại vào thuộc tính DeletedAt.
func (s *MemoryStorage) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	asset, ok := s.assets[id]
	if !ok || asset.DeletedAt != nil {
		return model.ErrNotFound
	}
	
	now := time.Now().UTC()
	asset.DeletedAt = &now
	s.assets[id] = asset
	return nil
}

// BatchDelete thực hiện xóa mềm hàng loạt tài sản trong danh sách ID được cung cấp.
// Trả về số tài sản đã được xóa mềm, số tài sản không tồn tại (hoặc đã xóa từ trước) và lỗi nếu có.
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

// Count đếm số lượng tài sản đang hoạt động (chưa xóa mềm) khớp với các tiêu chí lọc.
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

// GetStats tổng hợp số liệu thống kê về tài sản: tổng số, số lượng theo phân loại và số lượng theo trạng thái.
func (s *MemoryStorage) GetStats(_ context.Context) (*model.AssetStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &model.AssetStats{
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

// List trả về danh sách các tài sản đã được lọc và hỗ trợ phân trang (pagination).
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

// Search tìm kiếm tài sản theo tên (không phân biệt hoa thường) và có giới hạn số lượng kết quả trả về.
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

// CountAll trả về tổng số lượng tài sản chưa bị xóa mềm trong bộ nhớ.
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

// matchAsset là hàm bổ trợ kiểm tra xem một tài sản có thỏa mãn các tiêu chí lọc trong AssetFilter hay không.
func matchAsset(a model.Asset, filter model.AssetFilter) bool {
	if filter.Type != "" && string(a.Type) != filter.Type {
		return false
	}
	if filter.Status != "" && string(a.Status) != filter.Status {
		return false
	}
	if filter.Tag != "" && !strings.Contains(strings.ToLower(a.Tags), strings.ToLower(filter.Tag)) {
		return false
	}
	return true
}

// CreateScanJob khởi tạo và lưu một tiến trình quét mới vào bộ nhớ.
func (s *MemoryStorage) CreateScanJob(_ context.Context, job *model.ScanJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = *job
	return nil
}

// UpdateScanJob cập nhật thông tin tiến trình quét hiện có (ví dụ: trạng thái, thời gian chạy, kết quả).
func (s *MemoryStorage) UpdateScanJob(_ context.Context, job *model.ScanJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = *job
	return nil
}

// GetScanJobByID lấy thông tin tiến trình quét dựa trên ID tiến trình.
func (s *MemoryStorage) GetScanJobByID(_ context.Context, id string) (*model.ScanJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return &job, nil
}

// ListScanJobsForAsset liệt kê tất cả các tiến trình quét đã thực hiện đối với một tài sản cụ thể.
func (s *MemoryStorage) ListScanJobsForAsset(_ context.Context, assetID string) ([]model.ScanJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []model.ScanJob
	for _, j := range s.jobs {
		if j.AssetID == assetID {
			list = append(list, j)
		}
	}
	return list, nil
}

// CreateScanResult lưu trữ kết quả quét mới vào bộ nhớ.
func (s *MemoryStorage) CreateScanResult(_ context.Context, res *model.ScanResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results[res.ID] = *res
	return nil
}

// GetScanResultsForJob lấy danh sách tất cả các kết quả quét thuộc về một tiến trình quét cụ thể.
func (s *MemoryStorage) GetScanResultsForJob(_ context.Context, jobID string) ([]model.ScanResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []model.ScanResult
	for _, r := range s.results {
		if r.JobID == jobID {
			list = append(list, r)
		}
	}
	return list, nil
}

// GetScanResultsForAsset tìm kiếm tất cả các kết quả quét liên quan tới một tài sản cụ thể.
// Nó thực hiện tìm các công việc quét (scan jobs) của tài sản đó trước, sau đó lấy các kết quả tương ứng.
func (s *MemoryStorage) GetScanResultsForAsset(_ context.Context, assetID string) ([]model.ScanResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	jobIDs := make(map[string]bool)
	for _, j := range s.jobs {
		if j.AssetID == assetID {
			jobIDs[j.ID] = true
		}
	}

	var list []model.ScanResult
	for _, r := range s.results {
		if jobIDs[r.JobID] {
			list = append(list, r)
		}
	}
	return list, nil
}

// ListAllScanJobs liệt kê tất cả các tiến trình quét trên hệ thống (giới hạn số lượng).
func (s *MemoryStorage) ListAllScanJobs(_ context.Context, limit int) ([]model.ScanJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []model.ScanJob
	for _, j := range s.jobs {
		list = append(list, j)
	}

	// Sắp xếp giảm dần theo thời gian tạo (mới nhất lên đầu)
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[i].CreatedAt.Before(list[j].CreatedAt) {
				list[i], list[j] = list[j], list[i]
			}
		}
	}

	if len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}
