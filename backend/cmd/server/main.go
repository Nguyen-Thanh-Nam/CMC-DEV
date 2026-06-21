package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"asset-api/internal/handler"
	"asset-api/internal/middleware"
	"asset-api/internal/service"
	"asset-api/internal/storage"
	"asset-api/internal/storage/memory"
	"asset-api/internal/storage/mysql"
)

// loadEnv đọc file cấu hình môi trường (.env) thủ công và thiết lập các biến môi trường
func loadEnv(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
}

// getEnv lấy giá trị biến môi trường theo Key, nếu không tìm thấy sẽ trả về giá trị mặc định (defaultVal)
func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func main() {
	loadEnv(".env")
	startTime := time.Now().UTC()
	log.Println("Server is starting...")

	var store storage.AssetStorage
	useDB := getEnv("USE_DB", "true")

	if useDB == "true" {
		dbHost := getEnv("DB_HOST", "localhost")
		dbPort := getEnv("DB_PORT", "3306")
		dbUser := getEnv("DB_USER", "root")
		dbPassword := getEnv("DB_PASSWORD", "password")
		dbName := getEnv("DB_NAME", "mini_asm")

		config := mysql.NewDBConfig(dbHost, dbPort, dbUser, dbPassword, dbName)
		log.Printf("Connecting to MySQL at %s:%s...", dbHost, dbPort)
		
		mysqlStore, err := mysql.NewMySQLStorage(config.DSN())
		if err != nil {
			log.Printf("Database connection failed, falling back to Memory storage: %v", err)
			store = memory.NewMemoryStorage()
		} else {
			log.Println("Database connection established successfully!")
			store = mysqlStore
			defer mysqlStore.Close()
		}
	} else {
		log.Println("Using Memory storage as specified by environment variables.")
		store = memory.NewMemoryStorage()
	}

	assetSvc := service.NewAssetService(store)
	healthSvc := service.NewHealthService(store, startTime)
	scanSvc := service.NewScanService(store)

	scanSvc.StartScheduledScans(5 * time.Minute)

	assetHandler := handler.NewAssetHandler(assetSvc)
	healthHandler := handler.NewHealthHandler(healthSvc)
	scanHandler := handler.NewScanHandler(scanSvc)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler.Check)

	mux.HandleFunc("GET /assets/stats", assetHandler.GetStats)
	mux.HandleFunc("GET /assets/count", assetHandler.Count)
	mux.HandleFunc("GET /assets/search", assetHandler.Search)
	mux.HandleFunc("POST /assets/batch", assetHandler.BatchCreate)
	mux.HandleFunc("DELETE /assets/batch", assetHandler.BatchDelete)
	mux.HandleFunc("GET /assets/{id}", assetHandler.GetByID)
	mux.HandleFunc("POST /assets", assetHandler.Create)
	mux.HandleFunc("GET /assets", assetHandler.List)
	mux.HandleFunc("GET /assets/csv/export", assetHandler.ExportCSV)
	mux.HandleFunc("POST /assets/csv/import", assetHandler.ImportCSV)

	mux.HandleFunc("POST /assets/{id}/scan", scanHandler.StartScan)
	mux.HandleFunc("GET /scan-jobs", scanHandler.ListAll)
	mux.HandleFunc("GET /scan-jobs/{id}", scanHandler.GetJob)
	mux.HandleFunc("GET /scan-jobs/{id}/results", scanHandler.GetJobResults)
	mux.HandleFunc("GET /assets/{id}/scans", scanHandler.ListJobsForAsset)
	mux.HandleFunc("GET /assets/{id}/results", scanHandler.GetResultsForAsset)
	mux.HandleFunc("GET /assets/{id}/dns", scanHandler.GetAssetDNS)
	mux.HandleFunc("GET /assets/{id}/whois", scanHandler.GetAssetWHOIS)
	mux.HandleFunc("GET /assets/{id}/subdomains", scanHandler.GetAssetSubdomains)

	addr := ":8080"
	log.Printf("Server listening on http://localhost%s", addr)

	handlerWithMiddleware := middleware.CORSMiddleware(
		http.HandlerFunc(middleware.RateLimit(mux.ServeHTTP)),
	)

	if err := http.ListenAndServe(addr, handlerWithMiddleware); err != nil {
		log.Fatal(err)
	}
}

