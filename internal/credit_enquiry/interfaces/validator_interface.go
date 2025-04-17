package interfaces

import (
	"go-loan-service-v3/internal/credit_enquiry/errors"
	"go-loan-service-v3/proto"
)

// Validator interface defines the contract for request validation
type Validator interface {
	ValidateRequest(request *proto.CreditEnquiryRequest) (*errors.ValidationResponse, error)
}
