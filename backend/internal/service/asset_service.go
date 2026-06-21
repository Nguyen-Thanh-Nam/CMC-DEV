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

	"asset-api/internal/model"
	"asset-api/internal/storage"
	"asset-api/internal/validator"
)

// Các hằng số quy định cấu hình giới hạn cho hoạt động của AssetService.
const (
	maxBatchSize  = 100 // Số lượng tài sản tối đa được phép tạo trong một yêu cầu xử lý hàng loạt (batch).
	maxSearchSize = 100 // Số lượng kết quả tìm kiếm tối đa trả về trong một truy vấn tìm kiếm.
	defaultLimit  = 20  // Số lượng phần tử mặc định trên mỗi trang kết quả phân trang.
	maxLimit      = 100 // Giới hạn số lượng tài sản tối đa được hiển thị trên một trang để tránh quá tải bộ nhớ.
)

// cacheEntry định nghĩa cấu trúc của một mục lưu trữ trong bộ nhớ đệm (cache),
// bao gồm dữ liệu danh sách tài sản đã phân trang và thời gian hết hạn (expiration time).
type cacheEntry struct {
	data *model.PaginatedAssetsResponse // Con trỏ tới kết quả phân trang được lưu đệm.
	exp  time.Time                      // Thời điểm mục cache này hết hạn.
}

// AssetService đảm nhận việc xử lý các nghiệp vụ logic liên quan đến tài sản (Assets).
// Nó đóng vai trò là cầu nối giữa API Controller và tầng lưu trữ dữ liệu (Storage).
// Service này tích hợp tính năng ghi nhật ký kiểm toán (audit log), gọi webhook, và quản lý bộ nhớ đệm (caching).
type AssetService struct {
	storage   storage.AssetStorage  // Interface của bộ lưu trữ dữ liệu (ví dụ: DB hoặc in-memory).
	audit     *log.Logger           // Đối tượng ghi nhận nhật ký kiểm toán phục vụ việc giám sát hoạt động hệ thống.
	cache     map[string]cacheEntry // Bản đồ lưu trữ bộ nhớ đệm dùng để cải thiện hiệu năng cho các truy vấn danh sách.
	cacheMu   sync.RWMutex          // Khóa đọc-ghi để đồng bộ hóa luồng (thread-safety) khi truy cập hoặc chỉnh sửa bản đồ cache.
	validator *validator.AssetValidator
}

// NewAssetService khởi tạo một thực thể mới của AssetService.
// Hàm này mở file "audit.log" để lưu vết kiểm toán dưới chế độ ghi thêm (append).
// Trong trường hợp lỗi không mở được file (như thiếu quyền truy cập), nó sẽ sử dụng đầu ra lỗi chuẩn (os.Stderr) để tiếp tục hoạt động.
func NewAssetService(s storage.AssetStorage) *AssetService {
	f, err := os.OpenFile("audit.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		f = os.Stderr
	}
	logger := log.New(f, "AUDIT: ", log.Ldate|log.Ltime)
	return &AssetService{
		storage:   s,
		audit:     logger,
		cache:     make(map[string]cacheEntry),
		validator: validator.NewAssetValidator(),
	}
}

// clearCache làm trống toàn bộ dữ liệu đang lưu trong bộ nhớ đệm cache.
// Cơ chế này được gọi mỗi khi có dữ liệu tài sản bị thay đổi (tạo mới hoặc xóa) nhằm tránh việc trả về dữ liệu cũ (stale data).
func (svc *AssetService) clearCache() {
	svc.cacheMu.Lock()
	defer svc.cacheMu.Unlock()
	svc.cache = make(map[string]cacheEntry)
}

// triggerWebhook thực hiện gửi yêu cầu POST chứa thông tin JSON của tài sản tới một dịch vụ webhook bên ngoài.
// Để tối ưu hiệu năng và không chặn tiến trình xử lý chính (main thread), yêu cầu POST được thực hiện bất đồng bộ trong một goroutine riêng.
func triggerWebhook(asset *model.Asset) {
	payload, _ := json.Marshal(asset)
	go http.Post("http://localhost:8081/webhook-test", "application/json", bytes.NewBuffer(payload))
}

// Create xử lý yêu cầu tạo mới một tài sản đơn lẻ.
// Logic thực hiện:
// 1. Xác thực định dạng dữ liệu đầu vào.
// 2. Chuyển đổi dữ liệu input sang cấu trúc lưu trữ chính thức (hàm toAsset tự sinh ID).
// 3. Gọi lưu trữ dữ liệu xuống Storage.
// 4. Ghi audit log, kích hoạt webhook báo tin, và dọn dẹp bộ nhớ đệm cache.
func (svc *AssetService) Create(ctx context.Context, input model.CreateAssetInput) (*model.Asset, error) {
	if err := svc.validator.ValidateCreate(input.Name, input.Type); err != nil {
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

// BatchCreate thực hiện tạo hàng loạt tài sản trong cùng một yêu cầu.
// Logic thực hiện:
// 1. Kiểm tra danh sách đầu vào không được rỗng và không vượt quá giới hạn maxBatchSize (100).
// 2. Duyệt qua từng phần tử để xác thực trước. Nếu có bất cứ lỗi xác thực nào, hàm sẽ dừng lại ngay để đảm bảo tính nguyên tử (atomic-like).
// 3. Khởi tạo danh sách đối tượng lưu trữ, tự sinh ID và chuyển đổi định dạng.
// 4. Lưu toàn bộ danh sách xuống Storage.
// 5. Ghi nhận audit log, gửi webhook cho từng tài sản thành công, và xóa cache.
func (svc *AssetService) BatchCreate(ctx context.Context, req model.BatchCreateRequest) (*model.BatchCreateResponse, error) {
	if len(req.Assets) == 0 {
		return nil, model.ErrEmptyAssets
	}
	if len(req.Assets) > maxBatchSize {
		return nil, model.ErrBatchLimit
	}

	for i, input := range req.Assets {
		if err := svc.validator.ValidateCreate(input.Name, input.Type); err != nil {
			return nil, model.NewValidationError("assets[" + strconv.Itoa(i) + "]: " + err.Error())
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

// GetByID tìm kiếm và trả về thông tin của một tài sản dựa trên ID được cung cấp.
func (svc *AssetService) GetByID(ctx context.Context, id string) (*model.Asset, error) {
	return svc.storage.GetByID(ctx, id)
}

// BatchDelete xử lý xóa đồng thời nhiều tài sản theo danh sách các ID.
// Nếu có ít nhất một tài sản được xóa thành công, bộ nhớ cache sẽ được giải phóng để cập nhật lại dữ liệu mới nhất.
func (svc *AssetService) BatchDelete(ctx context.Context, ids []string) (*model.BatchDeleteResponse, error) {
	if len(ids) == 0 {
		return nil, model.ErrMissingIDs
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

// GetStats trả về báo cáo thống kê tổng số lượng tài sản và thống kê theo từng trạng thái cụ thể.
func (svc *AssetService) GetStats(ctx context.Context) (*model.AssetStats, error) {
	return svc.storage.GetStats(ctx)
}

// Count đếm tổng số tài sản trong hệ thống có bộ lọc tùy chọn theo loại hoặc trạng thái.
func (svc *AssetService) Count(ctx context.Context, filter model.AssetFilter) (*model.CountResult, error) {
	if filter.Type != "" && !model.ValidAssetType(filter.Type) {
		return nil, model.ErrInvalidAssetType
	}
	if filter.Status != "" && !model.ValidAssetStatus(filter.Status) {
		return nil, model.ErrInvalidAssetStatus
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

// List lấy danh sách tài sản đã được phân trang và áp dụng bộ lọc (Type và Status nếu có).
// Cơ chế tối ưu hóa cache được áp dụng ở đây:
// 1. Tạo cache key độc nhất dựa trên bộ lọc, trang hiện tại, và giới hạn hiển thị.
// 2. Sử dụng khóa đọc RLock để kiểm tra tính hợp lệ và thời gian sống của cache. Nếu cache hợp lệ, trả về kết quả ngay lập tức.
// 3. Nếu không có cache, truy vấn dữ liệu từ tầng Storage.
// 4. Tính toán tổng số trang từ tổng số tài sản đếm được.
// 5. Khóa ghi Lock để ghi nhận dữ liệu mới vào cache với thời gian hết hạn là 5 phút trước khi trả về dữ liệu.
func (svc *AssetService) List(ctx context.Context, filter model.AssetFilter, page, limit int) (*model.PaginatedAssetsResponse, error) {
	if err := svc.validator.ValidatePaginationParams(page, limit); err != nil {
		return nil, err
	}
	if filter.Type != "" {
		if err := svc.validator.ValidateType(filter.Type); err != nil {
			return nil, err
		}
	}
	if filter.Status != "" {
		if err := svc.validator.ValidateStatus(filter.Status); err != nil {
			return nil, err
		}
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

// Search tìm kiếm danh sách tài sản theo một chuỗi từ khóa.
// Chuỗi tìm kiếm sẽ được cắt bỏ các khoảng trắng thừa trước khi xử lý. Kết quả trả về tối đa là maxSearchSize (100).
func (svc *AssetService) Search(ctx context.Context, query string) ([]model.Asset, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, model.ErrMissingQuery
	}
	if err := svc.validator.ValidateSearchQuery(query); err != nil {
		return nil, err
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

// toAsset chuyển dữ liệu thô đầu vào (CreateAssetInput) thành đối tượng dữ liệu Asset hoàn chỉnh.
// Bao gồm tự động tạo ID duy nhất (UUID), làm sạch khoảng trắng các trường ký tự,
// gán trạng thái hoạt động mặc định nếu không truyền, và đánh dấu mốc thời gian tạo theo múi giờ UTC.
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
		Tags:      strings.TrimSpace(input.Tags),
		CreatedAt: time.Now().UTC(),
	}
}
