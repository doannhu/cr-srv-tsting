package interfaces

import (
	"context"
	"go-loan-service-v3/proto"
)

// RequestCacheRepository defines the interface for request cache operations
type RequestCacheRepository interface {
	// SaveCreditEnquiry saves a credit enquiry request to Redis
	SaveCreditEnquiry(ctx context.Context, request *proto.CreditEnquiryRequest) error
	// GetCreditEnquiry retrieves a credit enquiry request from Redis by request ID
	GetCreditEnquiry(ctx context.Context, requestID string, version string) (*proto.CreditEnquiryRequest, error)
}
