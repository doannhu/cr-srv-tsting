package interfaces

import (
	"context"

	creditEnquiryProto "go-loan-service-v3/proto/credit_enquiry"
)

// CreditEnquiryPublisher defines the interface for publishing credit enquiry events
type CreditEnquiryPublisher interface {
	// PublishCreditEnquiryEvent publishes a credit enquiry event
	PublishCreditEnquiryEvent(ctx context.Context, request *creditEnquiryProto.CreditEnquiryRequest) error
}
