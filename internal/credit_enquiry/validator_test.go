package credit_enquiry

import (
	"bytes"
	"log"
	"testing"

	"go-loan-service-v3/internal/credit_enquiry/errors"
	"go-loan-service-v3/proto"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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
		name           string
		request        *proto.CreditEnquiryRequest
		wantErr        bool
		expectedErrMsg string
		expectedCode   string
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
			wantErr:        true,
			expectedErrMsg: "invalid UUID length",
			expectedCode:   errors.ErrInvalidUUID,
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
			wantErr:        true,
			expectedErrMsg: "enquiry state exceeds maximum length of 40",
			expectedCode:   errors.ErrInvalidLength,
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
			wantErr:        true,
			expectedErrMsg: "loan amount must be positive and less than 999999999",
			expectedCode:   errors.ErrInvalidNumeric,
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
			wantErr:        true,
			expectedErrMsg: "loan amount must be positive and less than 999999999",
			expectedCode:   errors.ErrInvalidNumeric,
		},
		{
			name: "empty application number",
			request: &proto.CreditEnquiryRequest{
				RequestId:                        validUUIDBytes,
				EnquiryState:                     "NEW",
				ApplicationNumber:                "",
				LoanAmount:                       100000,
				LoanPurpose:                      "Home Purchase",
				InitialStructureTermMonth:        360,
				TotalMonthlyNetIncomeAmount:      5000,
				TotalAnnualGrossIncome:           60000,
				TotalSavingsAmount:               10000,
				TotalNumberOfContinuingHomeLoans: 0,
			},
			wantErr:        true,
			expectedErrMsg: "application number is required",
			expectedCode:   errors.ErrBadRequest,
		},
		{
			name: "empty loan purpose",
			request: &proto.CreditEnquiryRequest{
				RequestId:                        validUUIDBytes,
				EnquiryState:                     "NEW",
				ApplicationNumber:                "APP123",
				LoanAmount:                       100000,
				LoanPurpose:                      "",
				InitialStructureTermMonth:        360,
				TotalMonthlyNetIncomeAmount:      5000,
				TotalAnnualGrossIncome:           60000,
				TotalSavingsAmount:               10000,
				TotalNumberOfContinuingHomeLoans: 0,
			},
			wantErr:        true,
			expectedErrMsg: "loan purpose is required",
			expectedCode:   errors.ErrBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := validator.ValidateRequest(tt.request)
			assert.NoError(t, err)

			if tt.wantErr {
				assert.NotNil(t, resp)
				assert.False(t, resp.Success)
				assert.NotNil(t, resp.Error)
				assert.Equal(t, tt.expectedErrMsg, resp.Error.Message)
				assert.Equal(t, tt.expectedCode, resp.Error.Code)
			} else {
				assert.NotNil(t, resp)
				assert.True(t, resp.Success)
				assert.Nil(t, resp.Error)
			}
		})
	}
}
