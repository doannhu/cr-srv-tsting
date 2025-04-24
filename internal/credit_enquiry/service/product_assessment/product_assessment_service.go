package product_assessment

import (
	"context"
	"fmt"

	"go-loan-service-v3/internal/credit_enquiry/interfaces"
	"go-loan-service-v3/internal/credit_enquiry/utils"
	"go-loan-service-v3/proto"
)

// productAssessmentService implements interfaces.ProductAssessmentService
type productAssessmentService struct {
	retryConfig *utils.RetryConfig
}

// NewProductAssessmentService creates a new instance of interfaces.ProductAssessmentService
func NewProductAssessmentService(retryConfig *utils.RetryConfig) interfaces.ProductAssessmentService {
	return &productAssessmentService{
		retryConfig: retryConfig,
	}
}

// GetProductRate implements interfaces.ProductAssessmentService
func (s *productAssessmentService) GetProductRate(ctx context.Context, request *proto.ProductRateRequest) (*proto.ProductRateResponse, error) {
	var response *proto.ProductRateResponse

	operation := func() error {
		// TODO: Implement the actual call to product service
		response = &proto.ProductRateResponse{
			InitialStructureIndexRate: 5, // Default value for now
		}
		return nil
	}

	if err := utils.Retry(ctx, s.retryConfig, "get product rate", operation); err != nil {
		return nil, fmt.Errorf("failed to get product rate: %w", err)
	}

	return response, nil
}
