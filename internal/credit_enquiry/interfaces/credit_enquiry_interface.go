package interfaces

import (
	"context"

	creditEnquiryProto "go-loan-service-v3/proto/credit_enquiry"
)

// CreditEnquiryRepository interface defines the contract for credit enquiry storage
type CreditEnquiryRepository interface {
	SaveRequest(request *creditEnquiryProto.CreditEnquiryRequest) error
	// SaveCreditEnquiry saves a credit enquiry request to the repository
	SaveCreditEnquiry(ctx context.Context, request *creditEnquiryProto.CreditEnquiryRequest) error
	// GetCreditEnquiry retrieves a credit enquiry request from the repository
	GetCreditEnquiry(ctx context.Context, requestID string, version string) (*creditEnquiryProto.CreditEnquiryRequest, error)
}
