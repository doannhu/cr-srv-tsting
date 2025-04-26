package interfaces

import (
	"context"
	creditEnquiryProto "go-loan-service-v3/proto/credit_enquiry"
)

// RequestCacheRepository defines the interface for request cache operations
type RequestCacheRepository interface {
	// SaveCreditEnquiry saves a credit enquiry request to Redis
	SaveCreditEnquiry(ctx context.Context, request *creditEnquiryProto.CreditEnquiryRequest) error
	// GetCreditEnquiry retrieves a credit enquiry request from Redis by request ID
	GetCreditEnquiry(ctx context.Context, requestID string, version string) (*creditEnquiryProto.CreditEnquiryRequest, error)
}
