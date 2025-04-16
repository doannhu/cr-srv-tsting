package credit_enquiry

import (
	"context"

	"go-loan-service-v3/proto"
)

// CreditEnquiryPublisher defines the interface for publishing credit enquiry events
type CreditEnquiryPublisher interface {
	// PublishCreditEnquiryEvent publishes a credit enquiry event
	PublishCreditEnquiryEvent(ctx context.Context, request *proto.CreditEnquiryRequest) error
}
