package entity

import (
	"time"

	"go-loan-service-v3/proto"
)

// RequestCache represents the structure of a cached request in Redis
type RequestCache struct {
	// RequestID is the string representation of CreditEnquiryRequest.request_id
	RequestID           string                      `json:"request_id"`
	RequestType         string                      `json:"request_type"`
	RequestDataPayload  *proto.CreditEnquiryRequest `json:"request_data_payload"`
	ResponseDataPayload string                      `json:"response_data_payload"`
	UpdatedTime         time.Time                   `json:"updated_time"`
}

// RequestCacheRepository defines the interface for request cache operations
type RequestCacheRepository interface {
	// SaveRequest saves a credit enquiry request to Redis
	SaveRequest(request *proto.CreditEnquiryRequest) error
	// GetRequest retrieves a credit enquiry request from Redis by request ID
	GetRequest(requestID string) (*proto.CreditEnquiryRequest, error)
}
