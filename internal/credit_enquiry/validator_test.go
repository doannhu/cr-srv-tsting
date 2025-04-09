package credit_enquiry

import (
	"bytes"
	"log"
	"testing"

	"go-loan-service-v3/proto"

	"github.com/google/uuid"
)

func TestValidator_ValidateRequest(t *testing.T) {
	// Create a test logger that writes to a buffer
	var logBuffer bytes.Buffer
	logger := log.New(&logBuffer, "", log.LstdFlags)
	validator := NewValidator(logger)

	// Generate a valid UUID v4
	validUUID := uuid.New()
	validUUIDBytes, _ := validUUID.MarshalBinary()

	tests := []struct {
		name    string
		request *proto.CreditEnquiryRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: &proto.CreditEnquiryRequest{
				RequestId:                        validUUIDBytes,
				EnquiryState:                     "NEW",
				ApplicationNumber:                "APP123",
				LoanAmount:                       100000,
				LoanPurpose:                      "Home Purchase",
				InitialStructureTermMonth:        360,
				TotalMonthlyNetIncomeAmount:      5000,
				TotalAnnualGrossIncome:           60000,
				TotalSavingsAmount:               10000,
				TotalNumberOfContinuingHomeLoans: 0,
			},
			wantErr: false,
		},
		{
			name: "invalid UUID",
			request: &proto.CreditEnquiryRequest{
				RequestId:                        []byte("invalid-uuid"),
				EnquiryState:                     "NEW",
				ApplicationNumber:                "APP123",
				LoanAmount:                       100000,
				LoanPurpose:                      "Home Purchase",
				InitialStructureTermMonth:        360,
				TotalMonthlyNetIncomeAmount:      5000,
				TotalAnnualGrossIncome:           60000,
				TotalSavingsAmount:               10000,
				TotalNumberOfContinuingHomeLoans: 0,
			},
			wantErr: true,
		},
		{
			name: "enquiry state too long",
			request: &proto.CreditEnquiryRequest{
				RequestId:                        validUUIDBytes,
				EnquiryState:                     "This is a very long enquiry state that exceeds the maximum length of 40 characters",
				ApplicationNumber:                "APP123",
				LoanAmount:                       100000,
				LoanPurpose:                      "Home Purchase",
				InitialStructureTermMonth:        360,
				TotalMonthlyNetIncomeAmount:      5000,
				TotalAnnualGrossIncome:           60000,
				TotalSavingsAmount:               10000,
				TotalNumberOfContinuingHomeLoans: 0,
			},
			wantErr: true,
		},
		{
			name: "negative loan amount",
			request: &proto.CreditEnquiryRequest{
				RequestId:                        validUUIDBytes,
				EnquiryState:                     "NEW",
				ApplicationNumber:                "APP123",
				LoanAmount:                       -100000,
				LoanPurpose:                      "Home Purchase",
				InitialStructureTermMonth:        360,
				TotalMonthlyNetIncomeAmount:      5000,
				TotalAnnualGrossIncome:           60000,
				TotalSavingsAmount:               10000,
				TotalNumberOfContinuingHomeLoans: 0,
			},
			wantErr: true,
		},
		{
			name: "loan amount too large",
			request: &proto.CreditEnquiryRequest{
				RequestId:                        validUUIDBytes,
				EnquiryState:                     "NEW",
				ApplicationNumber:                "APP123",
				LoanAmount:                       1000000000,
				LoanPurpose:                      "Home Purchase",
				InitialStructureTermMonth:        360,
				TotalMonthlyNetIncomeAmount:      5000,
				TotalAnnualGrossIncome:           60000,
				TotalSavingsAmount:               10000,
				TotalNumberOfContinuingHomeLoans: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateRequest(tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validator.ValidateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
