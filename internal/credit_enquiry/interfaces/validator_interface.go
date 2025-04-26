package interfaces

import (
	"go-loan-service-v3/internal/credit_enquiry/errors"
	creditEnquiryProto "go-loan-service-v3/proto/credit_enquiry"
)

// Validator interface defines the contract for request validation
type Validator interface {
	ValidateRequest(request *creditEnquiryProto.CreditEnquiryRequest) (*errors.ValidationResponse, error)
}
