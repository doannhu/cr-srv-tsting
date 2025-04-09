package server

import (
	"context"
	"testing"

	pb "go-loan-service-v3/proto"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// mockValidator implements the Validator interface for testing
type mockValidator struct {
	validateFunc func(*pb.CreditEnquiryRequest) error
}

func (m *mockValidator) ValidateRequest(req *pb.CreditEnquiryRequest) error {
	return m.validateFunc(req)
}

// mockRepository implements the RequestCacheRepository interface for testing
type mockRepository struct {
	saveFunc func(*pb.CreditEnquiryRequest) error
}

func (m *mockRepository) SaveRequest(req *pb.CreditEnquiryRequest) error {
	return m.saveFunc(req)
}

func (m *mockRepository) GetRequest(requestID string) (*pb.CreditEnquiryRequest, error) {
	return nil, nil
}

func TestCreditEnquiryServer_ProcessCreditEnquiry(t *testing.T) {
	// Generate a valid UUID for testing
	validUUID := uuid.New()
	validUUIDBytes, err := validUUID.MarshalBinary()
	assert.NoError(t, err)

	// Create a base valid request
	validRequest := &pb.CreditEnquiryRequest{
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
	}

	tests := []struct {
		name           string
		request        *pb.CreditEnquiryRequest
		validatorError error
		repoError      error
		wantSuccess    bool
		wantMessage    string
	}{
		{
			name:           "valid request",
			request:        validRequest,
			validatorError: nil,
			repoError:      nil,
			wantSuccess:    true,
			wantMessage:    "Credit enquiry request processed successfully",
		},
		{
			name:           "validation failed",
			request:        validRequest,
			validatorError: assert.AnError,
			repoError:      nil,
			wantSuccess:    false,
			wantMessage:    assert.AnError.Error(),
		},
		{
			name:           "repository error",
			request:        validRequest,
			validatorError: nil,
			repoError:      assert.AnError,
			wantSuccess:    false,
			wantMessage:    assert.AnError.Error(),
		},
		{
			name: "invalid UUID",
			request: &pb.CreditEnquiryRequest{
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
			validatorError: nil,
			repoError:      nil,
			wantSuccess:    false,
			wantMessage:    "invalid UUID length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock validator
			validator := &mockValidator{
				validateFunc: func(req *pb.CreditEnquiryRequest) error {
					return tt.validatorError
				},
			}

			// Create mock repository
			repo := &mockRepository{
				saveFunc: func(req *pb.CreditEnquiryRequest) error {
					return tt.repoError
				},
			}

			// Create server instance
			server := NewCreditEnquiryServer(validator, repo)

			// Process the request
			response, err := server.ProcessCreditEnquiry(context.Background(), tt.request)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantSuccess, response.Success)
			assert.Equal(t, tt.wantMessage, response.Message)
		})
	}
}
