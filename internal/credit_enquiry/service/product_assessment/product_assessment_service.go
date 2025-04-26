package product_assessment

import (
	"context"
	"fmt"

	"go-loan-service-v3/internal/credit_enquiry/interfaces"
	"go-loan-service-v3/internal/credit_enquiry/utils"
	pb "go-loan-service-v3/proto/product_assessment"

	"google.golang.org/grpc"
)

const (
	defaultProductCode = "HLV"
)

// productAssessmentService implements interfaces.ProductAssessmentService
type productAssessmentService struct {
	retryConfig *utils.RetryConfig
	client      pb.ProductAssessmentServiceClient
}

// NewProductAssessmentService creates a new instance of interfaces.ProductAssessmentService
func NewProductAssessmentService(retryConfig *utils.RetryConfig, conn *grpc.ClientConn) interfaces.ProductAssessmentService {
	return &productAssessmentService{
		retryConfig: retryConfig,
		client:      pb.NewProductAssessmentServiceClient(conn),
	}
}

// GetProductRate implements interfaces.ProductAssessmentService
func (s *productAssessmentService) GetProductRate(ctx context.Context, request *pb.ProductRateRequest) (*pb.ProductRateResponse, error) {
	if request == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	// Use default product code if empty
	if request.ProductCode == "" {
		request.ProductCode = defaultProductCode
	}

	var response *pb.ProductRateResponse
	var err error

	operation := func() error {
		response, err = s.client.GetProductRate(ctx, request)
		return err
	}

	if err := utils.Retry(ctx, s.retryConfig, "get product rate", operation); err != nil {
		return nil, fmt.Errorf("failed to get product rate: %w", err)
	}

	return response, nil
}
