package interfaces

import (
	"context"
	"go-loan-service-v3/internal/credit_enquiry/entity"
)

// SopRepository defines the interface for SOP-related database operations
type SopRepository interface {
	// SaveSop saves a Statement of Position to the database
	SaveSop(ctx context.Context, sop *entity.Sop) error

	// GetSop retrieves a Statement of Position by credit enquiry ID and version
	GetSop(ctx context.Context, creditEnquiryID, version string) (*entity.Sop, error)
}
