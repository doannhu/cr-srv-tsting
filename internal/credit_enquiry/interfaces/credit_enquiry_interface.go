package interfaces

import (
	"context"

	"go-loan-service-v3/proto"
)

// CreditEnquiryRepository defines the interface for credit enquiry operations
type CreditEnquiryRepository interface {
	// SaveCreditEnquiry saves a credit enquiry request to the repository
	SaveCreditEnquiry(ctx context.Context, request *proto.CreditEnquiryRequest) error
	// GetCreditEnquiry retrieves a credit enquiry request from the repository
	GetCreditEnquiry(ctx context.Context, requestID string, version string) (*proto.CreditEnquiryRequest, error)
}
