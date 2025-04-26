package entity

import (
	"time"

	creditEnquiryProto "go-loan-service-v3/proto/credit_enquiry"
)

// RequestCache represents the structure of a cached request in Redis
type RequestCache struct {
	// RequestID is the string representation of CreditEnquiryRequest.request_id
	RequestID           string                                   `json:"request_id"`
	RequestType         string                                   `json:"request_type"`
	RequestDataPayload  *creditEnquiryProto.CreditEnquiryRequest `json:"request_data_payload"`
	ResponseDataPayload string                                   `json:"response_data_payload"`
	UpdatedTime         time.Time                                `json:"updated_time"`
}
