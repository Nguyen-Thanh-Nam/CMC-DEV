package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"asset-api/internal/model"
	"asset-api/internal/scanner"
	"asset-api/internal/storage"
)

// ScanService chịu trách nhiệm quản lý nghiệp vụ quét tài sản
type ScanService struct {
	storage  storage.AssetStorage          // Giao diện kết nối database (MySQL hoặc Memory)
	scanners map[string]scanner.Scanner    // Map chứa các bộ quét (DNS, SSL, Port,...)
	jobsChan chan string                   // Channel (Hàng đợi) chứa ID tác vụ để xử lý chạy nền
}

// NewScanService khởi tạo dịch vụ quét và kích hoạt các Worker chạy ngầm
func NewScanService(s storage.AssetStorage) *ScanService {
	svc := &ScanService{
		storage:  s,
		scanners: make(map[string]scanner.Scanner),
		jobsChan: make(chan string, 100),
	}

	svc.scanners["dns"] = scanner.NewDNSScanner()
	svc.scanners["whois"] = scanner.NewWHOISScanner()
	svc.scanners["ip"] = scanner.NewIPScanner()
	svc.scanners["asn"] = scanner.NewIPScanner()
	svc.scanners["port"] = scanner.NewPortScanner()
	svc.scanners["ssl"] = scanner.NewSSLScanner()
	svc.scanners["tech"] = scanner.NewTechScanner()

	for i := 0; i < 3; i++ {
		go svc.worker()
	}

	return svc
}

// StartScan bắt đầu một tác vụ quét: kiểm tra hợp lệ, tạo bản ghi lưu trữ, và đẩy vào hàng đợi
func (s *ScanService) StartScan(ctx context.Context, assetID, scanType string) (*model.ScanJob, error) {
	asset, err := s.storage.GetByID(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if asset.Status == model.AssetStatusInactive {
		return nil, errors.New("cannot scan inactive asset")
	}

	if scanType != "all" && s.scanners[scanType] == nil {
		return nil, errors.New("invalid scan type: " + scanType)
	}

	job := &model.ScanJob{
		ID:           uuid.NewString(),
		AssetID:      assetID,
		ScanType:     model.ScanType(scanType),
		Status:       model.ScanStatusPending,
		ResultsCount: 0,
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.storage.CreateScanJob(ctx, job); err != nil {
		return nil, err
	}

	s.jobsChan <- job.ID

	return job, nil
}

// GetScanJob lấy thông tin chi tiết một tác vụ quét theo ID
func (s *ScanService) GetScanJob(ctx context.Context, jobID string) (*model.ScanJob, error) {
	return s.storage.GetScanJobByID(ctx, jobID)
}

// GetScanJobResults lấy danh sách kết quả quét của một tác vụ cụ thể
func (s *ScanService) GetScanJobResults(ctx context.Context, jobID string) ([]model.ScanResult, error) {
	return s.storage.GetScanResultsForJob(ctx, jobID)
}

// ListScanJobsForAsset liệt kê toàn bộ lịch sử các lượt quét của một tài sản
func (s *ScanService) ListScanJobsForAsset(ctx context.Context, assetID string) ([]model.ScanJob, error) {
	return s.storage.ListScanJobsForAsset(ctx, assetID)
}

// ListAllScanJobs liệt kê tất cả các tiến trình quét trên toàn bộ hệ thống (giới hạn số lượng).
func (s *ScanService) ListAllScanJobs(ctx context.Context, limit int) ([]model.ScanJob, error) {
	return s.storage.ListAllScanJobs(ctx, limit)
}

// GetScanResultsForAsset lấy toàn bộ kết quả quét (mới nhất và cũ) của một tài sản
func (s *ScanService) GetScanResultsForAsset(ctx context.Context, assetID string) ([]model.ScanResult, error) {
	return s.storage.GetScanResultsForAsset(ctx, assetID)
}

// worker là hàm chạy ngầm (chạy trong Goroutine) liên tục lắng nghe hàng đợi tác vụ
func (s *ScanService) worker() {
	for jobID := range s.jobsChan {
		ctx := context.Background()

		job, err := s.storage.GetScanJobByID(ctx, jobID)
		if err != nil {
			log.Printf("Worker error retrieving job %s: %v", jobID, err)
			continue
		}

		now := time.Now().UTC()
		job.Status = model.ScanStatusRunning
		job.StartedAt = &now
		if err := s.storage.UpdateScanJob(ctx, job); err != nil {
			log.Printf("Worker error updating job status %s: %v", jobID, err)
			continue
		}

		asset, err := s.storage.GetByID(ctx, job.AssetID)
		if err != nil {
			s.failJob(ctx, job, fmt.Sprintf("failed to get asset: %v", err))
			continue
		}

		var results []any
		var scanErr error

		if job.ScanType == model.ScanTypeAll {
			var typesToRun []string
			if asset.Type == model.AssetTypeDomain {
				typesToRun = []string{"dns", "whois", "ssl", "tech"}
			} else if asset.Type == model.AssetTypeIP {
				typesToRun = []string{"ip"}
			}

			for _, t := range typesToRun {
				scn := s.scanners[t]
				if scn == nil {
					continue
				}
				res, err := scn.Scan(ctx, asset.Name)
				if err != nil {
					log.Printf("Sub-scan %s failed for job %s: %v", t, job.ID, err)
					continue
				}
				results = append(results, res)
			}
		} else {
			scn := s.scanners[string(job.ScanType)]
			if scn == nil {
				scanErr = fmt.Errorf("scanner %s not found", job.ScanType)
			} else {
				var res any
				res, scanErr = scn.Scan(ctx, asset.Name)
				if scanErr == nil {
					results = append(results, res)
				}
			}
		}

		if scanErr != nil {
			s.failJob(ctx, job, scanErr.Error())
			continue
		}

		for _, resData := range results {
			resRecord := &model.ScanResult{
				ID:         uuid.NewString(),
				JobID:      job.ID,
				ResultData: resData,
				CreatedAt:  time.Now().UTC(),
			}
			if err := s.storage.CreateScanResult(ctx, resRecord); err != nil {
				log.Printf("Worker error saving result for job %s: %v", job.ID, err)
			}
		}

		endTime := time.Now().UTC()
		job.Status = model.ScanStatusCompleted
		job.EndedAt = &endTime
		job.ResultsCount = len(results)
		if err := s.storage.UpdateScanJob(ctx, job); err != nil {
			log.Printf("Worker error completing job %s: %v", job.ID, err)
		}
	}
}

// failJob là hàm phụ trợ để đánh dấu tác vụ thất bại và lưu lỗi vào database
func (s *ScanService) failJob(ctx context.Context, job *model.ScanJob, errMsg string) {
	endTime := time.Now().UTC()
	job.Status = model.ScanStatusFailed
	job.EndedAt = &endTime
	job.Error = errMsg
	_ = s.storage.UpdateScanJob(ctx, job)
}

// StartScheduledScans khởi chạy chu kỳ tự động quét (Scheduled Scans) các tài sản Active
func (s *ScanService) StartScheduledScans(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			ctx := context.Background()
			log.Println("Starting scheduled passive scan for active assets...")

			paginated, _, err := s.storage.List(ctx, model.AssetFilter{Status: string(model.AssetStatusActive)}, 1, 100)
			if err != nil {
				log.Printf("Scheduled scan error listing assets: %v", err)
				continue
			}

			for _, asset := range paginated {
				scanType := "dns"
				if asset.Type == model.AssetTypeIP {
					scanType = "ip"
				}

				jobs, err := s.storage.ListScanJobsForAsset(ctx, asset.ID)
				if err == nil && len(jobs) > 0 {
					latest := jobs[0]
					if latest.Status == model.ScanStatusPending || latest.Status == model.ScanStatusRunning {
						continue
					}
				}

				_, err = s.StartScan(ctx, asset.ID, scanType)
				if err != nil {
					log.Printf("Scheduled scan failed to trigger for asset %s (%s): %v", asset.Name, asset.ID, err)
				} else {
					log.Printf("Scheduled scan triggered successfully for asset %s (%s)", asset.Name, scanType)
				}
			}
		}
	}()
}

