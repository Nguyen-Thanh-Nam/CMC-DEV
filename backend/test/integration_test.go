package test

import (
	"context"
	"testing"
	"time"

	"asset-api/internal/model"
	"asset-api/internal/service"
	"asset-api/internal/storage/memory"
)

func TestE2E_ScanWorkflow(t *testing.T) {
	// 1. Initialize memory dependencies
	store := memory.NewMemoryStorage()
	assetSvc := service.NewAssetService(store)
	scanSvc := service.NewScanService(store)

	ctx := context.Background()

	// 2. Create Domain Asset
	asset, err := assetSvc.Create(ctx, model.CreateAssetInput{
		Name: "localhost",
		Type: "domain",
		Tags: "test,local",
	})
	if err != nil {
		t.Fatalf("Asset creation failed: %v", err)
	}

	// 3. Trigger DNS scan job
	job, err := scanSvc.StartScan(ctx, asset.ID, "dns")
	if err != nil {
		t.Fatalf("Failed to trigger scan job: %v", err)
	}

	if job.Status != model.ScanStatusPending {
		t.Errorf("Expected job status pending, got %s", job.Status)
	}

	// 4. Poll for job completion (up to 3 seconds)
	deadline := time.Now().Add(3 * time.Second)
	var finalJob *model.ScanJob
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		finalJob, err = scanSvc.GetScanJob(ctx, job.ID)
		if err != nil {
			t.Fatalf("Failed to check job: %v", err)
		}
		if finalJob.Status == model.ScanStatusCompleted || finalJob.Status == model.ScanStatusFailed {
			break
		}
	}

	if finalJob.Status != model.ScanStatusCompleted {
		t.Fatalf("Expected job status completed, got %s (Error: %s)", finalJob.Status, finalJob.Error)
	}

	// 5. Fetch and verify results
	results, err := scanSvc.GetScanJobResults(ctx, job.ID)
	if err != nil {
		t.Fatalf("Failed to fetch job results: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected scan results to be saved, got 0")
	}

	// Verify that all is stored properly
	allResults, err := scanSvc.GetScanResultsForAsset(ctx, asset.ID)
	if err != nil {
		t.Fatalf("Failed to fetch asset results: %v", err)
	}
	if len(allResults) == 0 {
		t.Fatal("Expected asset results to be saved, got 0")
	}
}
