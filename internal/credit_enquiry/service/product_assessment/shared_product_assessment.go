package product_assessment

import (
	"context"
	"fmt"
	"go-loan-service-v3/internal/credit_enquiry/interfaces"
	"go-loan-service-v3/internal/credit_enquiry/utils"
	pb "go-loan-service-v3/proto/product_assessment"
	"time"
)

const (
	// DefaultProductCode is the fallback product code if none is specified
	DefaultProductCode = "HLV"
	// CredRefreshingInterval is the interval for refreshing TLS credentials
	CredRefreshingInterval = 10 * time.Minute
	// DialTimeout is the timeout for establishing gRPC connections
	DialTimeout = 5 * time.Second
)

// BaseService contains common fields for product assessment service implementations
type BaseService struct {
	RetryConfig *utils.RetryConfig
	Client      pb.ProductAssessmentServiceClient
}

// NewBaseService creates a new BaseService instance
func NewBaseService(retryConfig *utils.RetryConfig, client pb.ProductAssessmentServiceClient) *BaseService {
	return &BaseService{
		RetryConfig: retryConfig,
		Client:      client,
	}
}

// GetProductRate implements interfaces.ProductAssessmentService
func (s *BaseService) GetProductRate(ctx context.Context, req *pb.ProductRateRequest) (*pb.ProductRateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.ProductCode == "" {
		req.ProductCode = DefaultProductCode
	}

	var resp *pb.ProductRateResponse
	var err error
	op := func() error {
		resp, err = s.Client.GetProductRate(ctx, req)
		return err
	}
	if err := utils.Retry(ctx, s.RetryConfig, "get product rate", op); err != nil {
		return nil, fmt.Errorf("get product rate: %w", err)
	}
	return resp, nil
}

// Ensure BaseService implements interfaces.ProductAssessmentService
var _ interfaces.ProductAssessmentService = (*BaseService)(nil)
