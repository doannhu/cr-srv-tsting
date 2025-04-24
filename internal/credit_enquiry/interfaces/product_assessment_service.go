package interfaces

import (
	"context"
	"go-loan-service-v3/proto"
)

// ProductAssessmentService defines the interface for product assessment operations
type ProductAssessmentService interface {
	// GetProductRate retrieves the product rate from product service
	GetProductRate(ctx context.Context, request *proto.ProductRateRequest) (*proto.ProductRateResponse, error)
}
