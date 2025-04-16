package interfaces

import (
	"go-loan-service-v3/proto"
)

// RequestCacheRepository defines the interface for request cache operations
type RequestCacheRepository interface {
	// SaveRequest saves a credit enquiry request to Redis
	SaveRequest(request *proto.CreditEnquiryRequest) error
	// GetRequest retrieves a credit enquiry request from Redis by request ID
	GetRequest(requestID string) (*proto.CreditEnquiryRequest, error)
}
