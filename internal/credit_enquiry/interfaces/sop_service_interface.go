package interfaces

import (
	"context"

	pb "go-loan-service-v3/proto/sop"
)

// SOPService defines the interface for SOP service operations
type SOPService interface {
	GetConsolidatedSOP(ctx context.Context, request *pb.SOPRequest) (*pb.SOPResponse, error)
}
