package interfaces

import (
	"context"
	"go-loan-service-v3/internal/credit_enquiry/entity"
)

// ProductAssessmentRepository defines the interface for product assessment persistence
type ProductAssessmentRepository interface {
	// SaveProductAssessment creates a new product assessment record
	SaveProductAssessment(ctx context.Context, assessment *entity.ProductAssessment) error

	// GetProductAssessment retrieves a product assessment by credit enquiry ID and version
	GetProductAssessment(ctx context.Context, creditEnquiryID string, version string) (*entity.ProductAssessment, error)
}
