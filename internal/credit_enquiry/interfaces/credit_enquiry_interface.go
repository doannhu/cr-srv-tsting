package interfaces

import (
	"context"

	"go-loan-service-v3/proto"
)

// CreditEnquiryRepository interface defines the contract for credit enquiry storage
type CreditEnquiryRepository interface {
	SaveRequest(request *proto.CreditEnquiryRequest) error
	// SaveCreditEnquiry saves a credit enquiry request to the repository
	SaveCreditEnquiry(ctx context.Context, request *proto.CreditEnquiryRequest) error
	// GetCreditEnquiry retrieves a credit enquiry request from the repository
	GetCreditEnquiry(ctx context.Context, requestID string, version string) (*proto.CreditEnquiryRequest, error)
}
