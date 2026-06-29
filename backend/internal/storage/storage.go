package storage

import (
	"context"

	"asset-api/internal/model"
)

// AssetStorage định nghĩa interface cho việc lưu trữ và truy xuất dữ liệu tài sản (asset),
// các công việc quét (scan jobs) và kết quả quét (scan results).
// Interface này giúp trừu tượng hóa tầng dữ liệu (data access layer), cho phép dễ dàng thay thế
// các triển khai lưu trữ khác nhau như lưu trữ trong bộ nhớ (MemoryStorage) hoặc cơ sở dữ liệu (MySQLStorage).
type AssetStorage interface {
	// Create lưu một tài sản mới vào hệ thống lưu trữ.
	Create(ctx context.Context, asset *model.Asset) error

	// BatchCreate lưu danh sách nhiều tài sản mới vào hệ thống lưu trữ dưới dạng một lô (batch) để tối ưu hiệu năng.
	BatchCreate(ctx context.Context, assets []model.Asset) error

	// GetByID tìm kiếm và trả về thông tin chi tiết của một tài sản dựa trên ID của nó.
	// Trả về lỗi model.ErrNotFound nếu tài sản không tồn tại hoặc đã bị xóa mềm.
	GetByID(ctx context.Context, id string) (*model.Asset, error)

	// Delete thực hiện xóa mềm (soft delete) một tài sản bằng cách đánh dấu thời gian xóa.
	Delete(ctx context.Context, id string) error

	// BatchDelete thực hiện xóa mềm hàng loạt tài sản dựa trên danh sách ID truyền vào.
	// Trả về số lượng tài sản đã xóa thành công, số lượng tài sản không tìm thấy và lỗi nếu có.
	BatchDelete(ctx context.Context, ids []string) (deleted, notFound int, err error)

	// Count đếm số lượng tài sản thỏa mãn các điều kiện lọc trong AssetFilter (không tính các tài sản đã bị xóa mềm).
	Count(ctx context.Context, filter model.AssetFilter) (int, error)

	// GetStats thống kê các số liệu về tài sản: tổng số lượng, số lượng theo loại (type) và theo trạng thái (status).
	GetStats(ctx context.Context) (*model.AssetStats, error)

	// List trả về danh sách các tài sản được phân trang và lọc theo các tiêu chí trong AssetFilter.
	// Kết quả trả về gồm danh sách tài sản, tổng số lượng tài sản khớp với bộ lọc (để phục vụ phân trang) và lỗi nếu có.
	List(ctx context.Context, filter model.AssetFilter, page, limit int) ([]model.Asset, int, error)

	// Search tìm kiếm tài sản theo tên (không phân biệt chữ hoa chữ thường) và giới hạn số lượng kết quả trả về.
	Search(ctx context.Context, query string, limit int) ([]model.Asset, error)

	// CountAll đếm tổng số lượng tất cả tài sản hiện có trong hệ thống (chưa bị xóa mềm), không áp dụng bộ lọc nào.
	CountAll(ctx context.Context) int

	// Scan methods (Các phương thức liên quan đến tiến trình quét bảo mật tài sản)

	// CreateScanJob khởi tạo một tiến trình quét mới cho tài sản.
	CreateScanJob(ctx context.Context, job *model.ScanJob) error

	// UpdateScanJob cập nhật thông tin trạng thái, thời gian bắt đầu/kết thúc hoặc lỗi của một tiến trình quét.
	UpdateScanJob(ctx context.Context, job *model.ScanJob) error

	// GetScanJobByID lấy thông tin chi tiết của một tiến trình quét cụ thể qua ID.
	GetScanJobByID(ctx context.Context, id string) (*model.ScanJob, error)

	// ListScanJobsForAsset liệt kê tất cả các tiến trình quét đã từng thực hiện trên một tài sản cụ thể.
	ListScanJobsForAsset(ctx context.Context, assetID string) ([]model.ScanJob, error)

	// CreateScanResult lưu lại kết quả thu được sau khi hoàn thành tiến trình quét.
	CreateScanResult(ctx context.Context, res *model.ScanResult) error

	// GetScanResultsForJob lấy danh sách tất cả kết quả quét thuộc về một tiến trình quét cụ thể.
	GetScanResultsForJob(ctx context.Context, jobID string) ([]model.ScanResult, error)

	// GetScanResultsForAsset lấy toàn bộ kết quả quét từ mọi tiến trình quét đã thực hiện đối với một tài sản cụ thể.
	GetScanResultsForAsset(ctx context.Context, assetID string) ([]model.ScanResult, error)

	// ListAllScanJobs liệt kê tất cả các tiến trình quét trên hệ thống (giới hạn số lượng).
	ListAllScanJobs(ctx context.Context, limit int) ([]model.ScanJob, error)
}
