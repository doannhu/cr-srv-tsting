package sop

import (
	"context"
	"fmt"

	"go-loan-service-v3/internal/credit_enquiry/interfaces"
	"go-loan-service-v3/internal/credit_enquiry/utils"
	pb "go-loan-service-v3/proto/sop"
)

type BaseService struct {
	RetryConfig *utils.RetryConfig
	Client      pb.SOPServiceClient
}

func NewBaseService(retryConfig *utils.RetryConfig, client pb.SOPServiceClient) *BaseService {
	return &BaseService{
		RetryConfig: retryConfig,
		Client:      client,
	}
}

func (s *BaseService) GetConsolidatedSOP(ctx context.Context, request *pb.SOPRequest) (*pb.SOPResponse, error) {
	if request == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	var response *pb.SOPResponse
	var err error

	operation := func() error {
		response, err = s.Client.GetConsolidatedSOP(ctx, request)
		return err
	}

	if err := utils.Retry(ctx, s.RetryConfig, "get consolidated SOP", operation); err != nil {
		return nil, fmt.Errorf("failed to get consolidated SOP: %w", err)
	}

	return response, nil
}

var _ interfaces.SOPService = (*BaseService)(nil)
