package credit_enquiry

import (
	"bytes"
	"log"
	"testing"

	"go-loan-service-v3/internal/credit_enquiry/errors"
	proto "go-loan-service-v3/proto/credit_enquiry"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewValidator(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := log.New(&logBuffer, "", log.LstdFlags)

	v := NewValidator(logger)
	assert.NotNil(t, v)
	assert.NotNil(t, logger)
}

func TestValidator_ValidateRequest(t *testing.T) {
	// Create a test logger that writes to a buffer
	var logBuffer bytes.Buffer
	logger := log.New(&logBuffer, "", log.LstdFlags)
	validator := NewValidator(logger)

	// Generate a valid UUID v4
	validUUID := uuid.New()
	validUUIDBytes, _ := validUUID.MarshalBinary()

	// Generate a UUID v3 for testing invalid version
	invalidVersionUUID := uuid.NewMD5(uuid.NameSpaceDNS, []byte("test"))
	invalidVersionUUIDBytes, _ := invalidVersionUUID.MarshalBinary()

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
			name: "empty UUID",
			request: &proto.CreditEnquiryRequest{
				RequestId:                        []byte{},
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
			expectedErrMsg: "request_id is required",
			expectedCode:   errors.ErrRequiredField,
		},
		{
			name: "invalid UUID length",
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
			name: "invalid UUID version",
			request: &proto.CreditEnquiryRequest{
				RequestId:                        invalidVersionUUIDBytes,
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
			expectedErrMsg: "UUID must be version 4",
			expectedCode:   errors.ErrInvalidUUID,
		},
		{
			name: "empty enquiry state",
			request: &proto.CreditEnquiryRequest{
				RequestId:                        validUUIDBytes,
				EnquiryState:                     "",
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
			expectedErrMsg: "enquiry state is required",
			expectedCode:   errors.ErrBadRequest,
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
			name: "application number too long",
			request: &proto.CreditEnquiryRequest{
				RequestId:                        validUUIDBytes,
				EnquiryState:                     "NEW",
				ApplicationNumber:                "This is a very long application number that exceeds the maximum length of 37 characters",
				LoanAmount:                       100000,
				LoanPurpose:                      "Home Purchase",
				InitialStructureTermMonth:        360,
				TotalMonthlyNetIncomeAmount:      5000,
				TotalAnnualGrossIncome:           60000,
				TotalSavingsAmount:               10000,
				TotalNumberOfContinuingHomeLoans: 0,
			},
			wantErr:        true,
			expectedErrMsg: "application number exceeds maximum length of 37",
			expectedCode:   errors.ErrInvalidLength,
		},
		{
			name: "loan purpose too long",
			request: &proto.CreditEnquiryRequest{
				RequestId:                        validUUIDBytes,
				EnquiryState:                     "NEW",
				ApplicationNumber:                "APP123",
				LoanAmount:                       100000,
				LoanPurpose:                      "This is a very long loan purpose that exceeds the maximum length of 50 characters",
				InitialStructureTermMonth:        360,
				TotalMonthlyNetIncomeAmount:      5000,
				TotalAnnualGrossIncome:           60000,
				TotalSavingsAmount:               10000,
				TotalNumberOfContinuingHomeLoans: 0,
			},
			wantErr:        true,
			expectedErrMsg: "loan purpose exceeds maximum length of 50",
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
			name: "zero initial structure term",
			request: &proto.CreditEnquiryRequest{
				RequestId:                        validUUIDBytes,
				EnquiryState:                     "NEW",
				ApplicationNumber:                "APP123",
				LoanAmount:                       100000,
				LoanPurpose:                      "Home Purchase",
				InitialStructureTermMonth:        0,
				TotalMonthlyNetIncomeAmount:      5000,
				TotalAnnualGrossIncome:           60000,
				TotalSavingsAmount:               10000,
				TotalNumberOfContinuingHomeLoans: 0,
			},
			wantErr:        true,
			expectedErrMsg: "initial structure term month must be positive",
			expectedCode:   errors.ErrInvalidNumeric,
		},
		{
			name: "negative monthly net income",
			request: &proto.CreditEnquiryRequest{
				RequestId:                        validUUIDBytes,
				EnquiryState:                     "NEW",
				ApplicationNumber:                "APP123",
				LoanAmount:                       100000,
				LoanPurpose:                      "Home Purchase",
				InitialStructureTermMonth:        360,
				TotalMonthlyNetIncomeAmount:      -5000,
				TotalAnnualGrossIncome:           60000,
				TotalSavingsAmount:               10000,
				TotalNumberOfContinuingHomeLoans: 0,
			},
			wantErr:        true,
			expectedErrMsg: "total monthly net income amount must be positive",
			expectedCode:   errors.ErrInvalidNumeric,
		},
		{
			name: "negative annual gross income",
			request: &proto.CreditEnquiryRequest{
				RequestId:                        validUUIDBytes,
				EnquiryState:                     "NEW",
				ApplicationNumber:                "APP123",
				LoanAmount:                       100000,
				LoanPurpose:                      "Home Purchase",
				InitialStructureTermMonth:        360,
				TotalMonthlyNetIncomeAmount:      5000,
				TotalAnnualGrossIncome:           -60000,
				TotalSavingsAmount:               10000,
				TotalNumberOfContinuingHomeLoans: 0,
			},
			wantErr:        true,
			expectedErrMsg: "total annual gross income must be positive",
			expectedCode:   errors.ErrInvalidNumeric,
		},
		{
			name: "negative savings amount",
			request: &proto.CreditEnquiryRequest{
				RequestId:                        validUUIDBytes,
				EnquiryState:                     "NEW",
				ApplicationNumber:                "APP123",
				LoanAmount:                       100000,
				LoanPurpose:                      "Home Purchase",
				InitialStructureTermMonth:        360,
				TotalMonthlyNetIncomeAmount:      5000,
				TotalAnnualGrossIncome:           60000,
				TotalSavingsAmount:               -10000,
				TotalNumberOfContinuingHomeLoans: 0,
			},
			wantErr:        true,
			expectedErrMsg: "total savings amount cannot be negative",
			expectedCode:   errors.ErrInvalidNumeric,
		},
		{
			name: "negative continuing home loans",
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
				TotalNumberOfContinuingHomeLoans: -1,
			},
			wantErr:        true,
			expectedErrMsg: "total number of continuing home loans cannot be negative",
			expectedCode:   errors.ErrInvalidNumeric,
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
