package publisher

import (
	"context"

	"go-loan-service-v3/proto"
)

// Publisher defines the interface for publishing events
type Publisher interface {
	// PublishCreditEnquiryEvent publishes a credit enquiry event
	PublishCreditEnquiryEvent(ctx context.Context, request *proto.CreditEnquiryRequest) error
}
