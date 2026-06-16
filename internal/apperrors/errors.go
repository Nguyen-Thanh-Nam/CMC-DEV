package apperrors

import "errors"

var (
	ErrNotFound          = errors.New("asset not found")
	ErrInvalidInput      = errors.New("invalid input")
	ErrInvalidAssetType  = errors.New("invalid asset type")
	ErrInvalidAssetStatus = errors.New("invalid asset status")
	ErrEmptyName         = errors.New("name is required")
	ErrEmptyAssets       = errors.New("assets list is required")
	ErrBatchLimit        = errors.New("batch limit exceeded: maximum 100 assets per request")
	ErrMissingIDs        = errors.New("ids parameter required")
	ErrMissingQuery      = errors.New("q parameter required")
)

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func NewValidationError(msg string) error {
	return &ValidationError{Message: msg}
}
