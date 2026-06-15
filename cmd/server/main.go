package main

import (
	"log"
	"net/http"
	"time"

	"asset-api/internal/handler"
	"asset-api/internal/middleware"
	"asset-api/internal/service"
	"asset-api/internal/storage"
)

func main() {
	startTime := time.Now().UTC()

	memStorage := storage.NewMemoryStorage()
	assetSvc := service.NewAssetService(memStorage)
	healthSvc := service.NewHealthService(memStorage, startTime)

	assetHandler := handler.NewAssetHandler(assetSvc)
	healthHandler := handler.NewHealthHandler(healthSvc)

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

	addr := ":8080"
	log.Printf("server listening on http://localhost%s", addr)
	
	handlerWithMiddleware := middleware.RateLimit(mux.ServeHTTP)
	if err := http.ListenAndServe(addr, handlerWithMiddleware); err != nil {
		log.Fatal(err)
	}
}
