package interfaces

import (
	"context"
	pb "go-loan-service-v3/proto/product_assessment"
)

// ProductAssessmentService defines the interface for product assessment operations
type ProductAssessmentService interface {
	// GetProductRate retrieves the product rate from product service
	GetProductRate(ctx context.Context, request *pb.ProductRateRequest) (*pb.ProductRateResponse, error)
}
