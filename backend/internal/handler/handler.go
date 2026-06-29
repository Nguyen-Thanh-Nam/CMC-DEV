package handler

type Handlers struct {
	Asset  *AssetHandler
	Health *HealthHandler
	Scan   *ScanHandler
}

func NewHandlers(asset *AssetHandler, health *HealthHandler, scan *ScanHandler) *Handlers {
	return &Handlers{
		Asset:  asset,
		Health: health,
		Scan:   scan,
	}
}
