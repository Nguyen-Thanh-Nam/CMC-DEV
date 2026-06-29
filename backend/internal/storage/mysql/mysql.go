package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"asset-api/internal/model"

	_ "github.com/go-sql-driver/mysql"
)

// DBConfig chứa các thông số cấu hình kết nối tới cơ sở dữ liệu MySQL.
type DBConfig struct {
	Host     string // Địa chỉ máy chủ database (ví dụ: localhost hoặc db-container)
	Port     string // Cổng kết nối (mặc định là 3306)
	User     string // Tên người dùng kết nối (mặc định là root)
	Password string // Mật khẩu truy cập
	DBName   string // Tên cơ sở dữ liệu sẽ làm việc (mặc định là mini_asm)
}

// NewDBConfig khởi tạo cấu hình DBConfig và gán giá trị mặc định cho các trường nếu chúng bị bỏ trống.
func NewDBConfig(host, port, user, password, dbName string) DBConfig {
	if host == "" { host = "localhost" }
	if port == "" { port = "3306" }
	if user == "" { user = "root" }
	if dbName == "" { dbName = "mini_asm" }
	return DBConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		DBName:   dbName,
	}
}

// DSN (Data Source Name) sinh ra chuỗi kết nối (Connection String) phù hợp với thư viện driver MySQL của Go.
// Chuỗi này chỉ định thêm parseTime=true để tự động chuyển đổi kiểu dữ liệu DATETIME/TIMESTAMP của MySQL sang time.Time của Go,
// và thiết lập charset/collation hỗ trợ đầy đủ các ký tự UTF-8.
func (c DBConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci",
		c.User, c.Password, c.Host, c.Port, c.DBName)
}

// MySQLStorage triển khai interface storage.AssetStorage, chịu trách nhiệm tương tác trực tiếp với hệ quản trị CSDL MySQL.
type MySQLStorage struct {
	db *sql.DB // Đối tượng đại diện cho hồ chứa kết nối (connection pool) cơ sở dữ liệu.
}

// NewMySQLStorage khởi tạo một đối tượng MySQLStorage mới.
// Hàm này tích hợp cơ chế thử lại (retry) kết nối 5 lần với thời gian chờ giữa các lần là 3 giây để đảm bảo 
// ứng dụng không bị sập ngay lập tức nếu MySQL chưa khởi động xong (phổ biến khi chạy bằng docker-compose).
func NewMySQLStorage(dsn string) (*MySQLStorage, error) {
	var db *sql.DB
	var err error

	for i := 1; i <= 5; i++ {
		db, err = sql.Open("mysql", dsn)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		log.Printf("Connecting to MySQL failed (attempt %d/5): %v. Retrying in 3 seconds...", i, err)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("could not connect to database after retries: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	storage := &MySQLStorage{db: db}
	if err := storage.AutoMigrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run database auto-migrations: %w", err)
	}

	return storage, nil
}

// Close đóng toàn bộ các kết nối trong pool của cơ sở dữ liệu MySQL.
func (s *MySQLStorage) Close() error {
	return s.db.Close()
}

// AutoMigrate tạo các bảng cơ sở dữ liệu cần thiết (assets, scan_jobs, scan_results) nếu chúng chưa tồn tại.
func (s *MySQLStorage) AutoMigrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS assets (
			id VARCHAR(36) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			type VARCHAR(50) NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			tags VARCHAR(255) NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at TIMESTAMP NULL,
			INDEX idx_assets_name (name),
			INDEX idx_assets_type (type),
			INDEX idx_assets_status (status),
			INDEX idx_assets_deleted_at (deleted_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,

		`CREATE TABLE IF NOT EXISTS scan_jobs (
			id VARCHAR(36) PRIMARY KEY,
			asset_id VARCHAR(36) NOT NULL,
			scan_type VARCHAR(50) NOT NULL,
			status VARCHAR(50) NOT NULL,
			started_at TIMESTAMP NULL,
			ended_at TIMESTAMP NULL,
			error TEXT,
			results_count INT NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_scan_jobs_asset_id (asset_id),
			INDEX idx_scan_jobs_status (status),
			FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,

		`CREATE TABLE IF NOT EXISTS scan_results (
			id VARCHAR(36) PRIMARY KEY,
			job_id VARCHAR(36) NOT NULL,
			result_data JSON NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_scan_results_job_id (job_id),
			FOREIGN KEY (job_id) REFERENCES scan_jobs(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
	}

	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("migration query failed: %w", err)
		}
	}
	log.Println("Database auto-migration completed successfully!")
	return nil
}

// Create thêm một tài sản mới vào bảng assets trong cơ sở dữ liệu.
func (s *MySQLStorage) Create(ctx context.Context, asset *model.Asset) error {
	query := "INSERT INTO assets (id, name, type, status, tags, created_at, deleted_at) VALUES (?, ?, ?, ?, ?, ?, ?)"
	_, err := s.db.ExecContext(ctx, query, asset.ID, asset.Name, asset.Type, asset.Status, asset.Tags, asset.CreatedAt, asset.DeletedAt)
	return err
}

// BatchCreate thực hiện thêm nhiều tài sản cùng lúc bằng cách sử dụng Giao dịch (Transaction - ACID).
// Việc chuẩn bị trước câu lệnh (Prepared Statement) giúp tăng tốc độ thực thi và bảo mật hơn khi chạy vòng lặp INSERT.
func (s *MySQLStorage) BatchCreate(ctx context.Context, assets []model.Asset) error {
	if len(assets) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := "INSERT INTO assets (id, name, type, status, tags, created_at, deleted_at) VALUES (?, ?, ?, ?, ?, ?, ?)"
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, asset := range assets {
		_, err := stmt.ExecContext(ctx, asset.ID, asset.Name, asset.Type, asset.Status, asset.Tags, asset.CreatedAt, asset.DeletedAt)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetByID truy vấn thông tin chi tiết một tài sản theo ID, bỏ qua các bản ghi đã bị xóa mềm (deleted_at IS NULL).
func (s *MySQLStorage) GetByID(ctx context.Context, id string) (*model.Asset, error) {
	query := "SELECT id, name, type, status, tags, created_at, deleted_at FROM assets WHERE id = ? AND deleted_at IS NULL"
	row := s.db.QueryRowContext(ctx, query, id)
	
	var asset model.Asset
	var deletedAt sql.NullTime
	err := row.Scan(&asset.ID, &asset.Name, &asset.Type, &asset.Status, &asset.Tags, &asset.CreatedAt, &deletedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	if deletedAt.Valid {
		asset.DeletedAt = &deletedAt.Time
	}
	return &asset, nil
}

// Delete thực hiện xóa mềm một tài sản bằng cách cập nhật trường deleted_at thành thời gian hiện tại.
// Nếu không có dòng nào bị ảnh hưởng (RowsAffected = 0), trả về lỗi model.ErrNotFound.
func (s *MySQLStorage) Delete(ctx context.Context, id string) error {
	query := "UPDATE assets SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL"
	result, err := s.db.ExecContext(ctx, query, time.Now().UTC(), id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return model.ErrNotFound
	}
	return nil
}

// BatchDelete thực hiện xóa mềm hàng loạt tài sản dựa trên danh sách ID cung cấp, chạy trong một Transaction.
// Trả về số bản ghi đã xóa, số lượng bản ghi không tìm thấy hoặc đã xóa từ trước, và lỗi nếu có.
func (s *MySQLStorage) BatchDelete(ctx context.Context, ids []string) (deleted, notFound int, err error) {
	if len(ids) == 0 {
		return 0, 0, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	query := "UPDATE assets SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL"
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return 0, 0, err
	}
	defer stmt.Close()

	now := time.Now().UTC()
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		res, err := stmt.ExecContext(ctx, now, id)
		if err != nil {
			return 0, 0, err
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return 0, 0, err
		}
		if rows > 0 {
			deleted++
		} else {
			notFound++
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, 0, err
	}

	return deleted, notFound, nil
}

// Count đếm số lượng tài sản thỏa mãn bộ lọc tìm kiếm được cung cấp.
func (s *MySQLStorage) Count(ctx context.Context, filter model.AssetFilter) (int, error) {
	query := "SELECT COUNT(*) FROM assets WHERE deleted_at IS NULL"
	args := []any{}

	if filter.Type != "" {
		query += " AND type = ?"
		args = append(args, filter.Type)
	}
	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}
	if filter.Tag != "" {
		query += " AND (FIND_IN_SET(?, tags) OR tags LIKE ?)"
		args = append(args, filter.Tag, "%"+filter.Tag+"%")
	}

	var count int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

// GetStats thực hiện thống kê tổng quan về tài sản: tổng số, số lượng theo loại và trạng thái.
func (s *MySQLStorage) GetStats(ctx context.Context) (*model.AssetStats, error) {
	stats := &model.AssetStats{
		Total:    0,
		ByType:   map[string]int{"domain": 0, "ip": 0, "service": 0},
		ByStatus: map[string]int{"active": 0, "inactive": 0},
	}

	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM assets WHERE deleted_at IS NULL").Scan(&stats.Total)
	if err != nil {
		return nil, err
	}

	typeRows, err := s.db.QueryContext(ctx, "SELECT type, COUNT(*) FROM assets WHERE deleted_at IS NULL GROUP BY type")
	if err != nil {
		return nil, err
	}
	defer typeRows.Close()
	for typeRows.Next() {
		var t string
		var count int
		if err := typeRows.Scan(&t, &count); err == nil {
			stats.ByType[t] = count
		}
	}

	statusRows, err := s.db.QueryContext(ctx, "SELECT status, COUNT(*) FROM assets WHERE deleted_at IS NULL GROUP BY status")
	if err != nil {
		return nil, err
	}
	defer statusRows.Close()
	for statusRows.Next() {
		var st string
		var count int
		if err := statusRows.Scan(&st, &count); err == nil {
			stats.ByStatus[st] = count
		}
	}

	return stats, nil
}

// List truy vấn danh sách tài sản có phân trang (Pagination) và lọc theo các tiêu chí của AssetFilter.
// Thứ tự sắp xếp mặc định là tài sản mới nhất lên đầu (created_at DESC).
func (s *MySQLStorage) List(ctx context.Context, filter model.AssetFilter, page, limit int) ([]model.Asset, int, error) {
	queryCount := "SELECT COUNT(*) FROM assets WHERE deleted_at IS NULL"
	querySelect := "SELECT id, name, type, status, tags, created_at, deleted_at FROM assets WHERE deleted_at IS NULL"
	args := []any{}

	whereClause := ""
	if filter.Type != "" {
		whereClause += " AND type = ?"
		args = append(args, filter.Type)
	}
	if filter.Status != "" {
		whereClause += " AND status = ?"
		args = append(args, filter.Status)
	}
	if filter.Tag != "" {
		whereClause += " AND (FIND_IN_SET(?, tags) OR tags LIKE ?)"
		args = append(args, filter.Tag, "%"+filter.Tag+"%")
	}

	queryCount += whereClause
	var total int
	err := s.db.QueryRowContext(ctx, queryCount, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	querySelect += whereClause + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	offset := (page - 1) * limit
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, querySelect, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var assets []model.Asset
	for rows.Next() {
		var asset model.Asset
		var deletedAt sql.NullTime
		err := rows.Scan(&asset.ID, &asset.Name, &asset.Type, &asset.Status, &asset.Tags, &asset.CreatedAt, &deletedAt)
		if err != nil {
			return nil, 0, err
		}
		if deletedAt.Valid {
			asset.DeletedAt = &deletedAt.Time
		}
		assets = append(assets, asset)
	}

	if assets == nil {
		assets = []model.Asset{}
	}

	return assets, total, nil
}

// Search tìm kiếm các tài sản có tên gần đúng dựa trên từ khóa tìm kiếm (Sử dụng mệnh đề LIKE %keyword%).
func (s *MySQLStorage) Search(ctx context.Context, query string, limit int) ([]model.Asset, error) {
	dbQuery := "SELECT id, name, type, status, tags, created_at, deleted_at FROM assets WHERE name LIKE ? AND deleted_at IS NULL LIMIT ?"
	rows, err := s.db.QueryContext(ctx, dbQuery, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []model.Asset
	for rows.Next() {
		var asset model.Asset
		var deletedAt sql.NullTime
		err := rows.Scan(&asset.ID, &asset.Name, &asset.Type, &asset.Status, &asset.Tags, &asset.CreatedAt, &deletedAt)
		if err != nil {
			return nil, err
		}
		if deletedAt.Valid {
			asset.DeletedAt = &deletedAt.Time
		}
		assets = append(assets, asset)
	}

	if assets == nil {
		assets = []model.Asset{}
	}

	return assets, nil
}

// CountAll trả về tổng số lượng tài sản đang hoạt động trong bảng assets (chưa bị xóa mềm).
func (s *MySQLStorage) CountAll(ctx context.Context) int {
	var count int
	_ = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM assets WHERE deleted_at IS NULL").Scan(&count)
	return count
}

// CreateScanJob khởi tạo và lưu trữ thông tin một tiến trình quét mới vào bảng scan_jobs.
func (s *MySQLStorage) CreateScanJob(ctx context.Context, job *model.ScanJob) error {
	query := "INSERT INTO scan_jobs (id, asset_id, scan_type, status, started_at, ended_at, error, results_count, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)"
	_, err := s.db.ExecContext(ctx, query, job.ID, job.AssetID, job.ScanType, job.Status, job.StartedAt, job.EndedAt, job.Error, job.ResultsCount, job.CreatedAt)
	return err
}

// UpdateScanJob cập nhật các trường động của tiến trình quét (trạng thái, thời gian bắt đầu/kết thúc, lỗi, số lượng kết quả).
func (s *MySQLStorage) UpdateScanJob(ctx context.Context, job *model.ScanJob) error {
	query := "UPDATE scan_jobs SET status = ?, started_at = ?, ended_at = ?, error = ?, results_count = ? WHERE id = ?"
	_, err := s.db.ExecContext(ctx, query, job.Status, job.StartedAt, job.EndedAt, job.Error, job.ResultsCount, job.ID)
	return err
}

// GetScanJobByID lấy thông tin chi tiết của một tiến trình quét cụ thể dựa trên ID.
func (s *MySQLStorage) GetScanJobByID(ctx context.Context, id string) (*model.ScanJob, error) {
	query := "SELECT id, asset_id, scan_type, status, started_at, ended_at, error, results_count, created_at FROM scan_jobs WHERE id = ?"
	row := s.db.QueryRowContext(ctx, query, id)

	var job model.ScanJob
	var startedAt, endedAt sql.NullTime
	err := row.Scan(&job.ID, &job.AssetID, &job.ScanType, &job.Status, &startedAt, &endedAt, &job.Error, &job.ResultsCount, &job.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if endedAt.Valid {
		job.EndedAt = &endedAt.Time
	}
	return &job, nil
}

// ListScanJobsForAsset liệt kê danh sách tất cả các công việc quét liên quan tới một tài sản, sắp xếp từ mới nhất tới cũ nhất.
func (s *MySQLStorage) ListScanJobsForAsset(ctx context.Context, assetID string) ([]model.ScanJob, error) {
	query := "SELECT id, asset_id, scan_type, status, started_at, ended_at, error, results_count, created_at FROM scan_jobs WHERE asset_id = ? ORDER BY created_at DESC"
	rows, err := s.db.QueryContext(ctx, query, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.ScanJob
	for rows.Next() {
		var job model.ScanJob
		var startedAt, endedAt sql.NullTime
		err := rows.Scan(&job.ID, &job.AssetID, &job.ScanType, &job.Status, &startedAt, &endedAt, &job.Error, &job.ResultsCount, &job.CreatedAt)
		if err != nil {
			return nil, err
		}
		if startedAt.Valid {
			job.StartedAt = &startedAt.Time
		}
		if endedAt.Valid {
			job.EndedAt = &endedAt.Time
		}
		list = append(list, job)
	}
	return list, nil
}

// ListAllScanJobs liệt kê tất cả các tiến trình quét trên hệ thống (giới hạn số lượng).
func (s *MySQLStorage) ListAllScanJobs(ctx context.Context, limit int) ([]model.ScanJob, error) {
	query := "SELECT id, asset_id, scan_type, status, started_at, ended_at, error, results_count, created_at FROM scan_jobs ORDER BY created_at DESC LIMIT ?"
	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.ScanJob
	for rows.Next() {
		var job model.ScanJob
		var startedAt, endedAt sql.NullTime
		err := rows.Scan(&job.ID, &job.AssetID, &job.ScanType, &job.Status, &startedAt, &endedAt, &job.Error, &job.ResultsCount, &job.CreatedAt)
		if err != nil {
			return nil, err
		}
		if startedAt.Valid {
			job.StartedAt = &startedAt.Time
		}
		if endedAt.Valid {
			job.EndedAt = &endedAt.Time
		}
		list = append(list, job)
	}
	return list, nil
}

// CreateScanResult lưu trữ kết quả của một tiến trình quét. 
// Do trường ResultData có kiểu dữ liệu là `any` (interface{}), hàm sẽ tiến hành tuần tự hóa (marshal) dữ liệu này 
// sang định dạng chuỗi JSON trước khi lưu vào cột kiểu JSON trong MySQL.
func (s *MySQLStorage) CreateScanResult(ctx context.Context, res *model.ScanResult) error {
	dataBytes, err := json.Marshal(res.ResultData)
	if err != nil {
		return err
	}

	query := "INSERT INTO scan_results (id, job_id, result_data, created_at) VALUES (?, ?, ?, ?)"
	_, err = s.db.ExecContext(ctx, query, res.ID, res.JobID, string(dataBytes), res.CreatedAt)
	return err
}

// GetScanResultsForJob truy xuất tất cả các kết quả quét thuộc về một tiến trình quét (Scan Job) nhất định.
// Sau khi đọc chuỗi JSON từ cột dữ liệu, nó sẽ giải tuần tự hóa (unmarshal) ngược lại đối tượng Go.
func (s *MySQLStorage) GetScanResultsForJob(ctx context.Context, jobID string) ([]model.ScanResult, error) {
	query := "SELECT id, job_id, result_data, created_at FROM scan_results WHERE job_id = ?"
	rows, err := s.db.QueryContext(ctx, query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.ScanResult
	for rows.Next() {
		var res model.ScanResult
		var rawJSON string
		err := rows.Scan(&res.ID, &res.JobID, &rawJSON, &res.CreatedAt)
		if err != nil {
			return nil, err
		}
		var parsedData any
		_ = json.Unmarshal([]byte(rawJSON), &parsedData)
		res.ResultData = parsedData
		list = append(list, res)
	}
	return list, nil
}

// GetScanResultsForAsset tìm kiếm tất cả các kết quả quét liên quan tới một tài sản nhất định.
// Sử dụng liên kết INNER JOIN giữa hai bảng scan_results và scan_jobs qua cột job_id để lọc theo asset_id nhanh chóng.
func (s *MySQLStorage) GetScanResultsForAsset(ctx context.Context, assetID string) ([]model.ScanResult, error) {
	query := `
		SELECT r.id, r.job_id, r.result_data, r.created_at 
		FROM scan_results r 
		INNER JOIN scan_jobs j ON r.job_id = j.id 
		WHERE j.asset_id = ? 
		ORDER BY r.created_at DESC`
	rows, err := s.db.QueryContext(ctx, query, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.ScanResult
	for rows.Next() {
		var res model.ScanResult
		var rawJSON string
		err := rows.Scan(&res.ID, &res.JobID, &rawJSON, &res.CreatedAt)
		if err != nil {
			return nil, err
		}
		var parsedData any
		_ = json.Unmarshal([]byte(rawJSON), &parsedData)
		res.ResultData = parsedData
		list = append(list, res)
	}
	return list, nil
}
