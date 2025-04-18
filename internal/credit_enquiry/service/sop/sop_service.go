package sop

import (
	"context"

	"go-loan-service-v3/internal/credit_enquiry/utils"
	pb "go-loan-service-v3/proto/sop"

	"google.golang.org/grpc"
)

// SOPService defines the interface for SOP service operations
type SOPService interface {
	GetConsolidatedSOP(ctx context.Context, request *pb.SOPRequest) (*pb.SOPResponse, error)
}

// sopServiceClient implements the SOPService interface
type sopServiceClient struct {
	client      pb.SOPServiceClient
	retryConfig *utils.RetryConfig
}

// NewSOPServiceClient creates a new SOP service client
func NewSOPServiceClient(conn *grpc.ClientConn, retryConfig *utils.RetryConfig) SOPService {
	return &sopServiceClient{
		client:      pb.NewSOPServiceClient(conn),
		retryConfig: retryConfig,
	}
}

// GetConsolidatedSOP retrieves consolidated SOP data with retry logic
func (s *sopServiceClient) GetConsolidatedSOP(ctx context.Context, request *pb.SOPRequest) (*pb.SOPResponse, error) {
	var response *pb.SOPResponse
	var err error

	operation := func() error {
		response, err = s.client.GetConsolidatedSOP(ctx, request)
		return err
	}

	if err := utils.Retry(ctx, s.retryConfig, "get consolidated SOP", operation); err != nil {
		return nil, err
	}

	return response, nil
}
