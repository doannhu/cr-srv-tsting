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

// mockRedisRepository implements the RequestCacheRepository interface for testing
type mockRedisRepository struct {
	saveFunc func(*pb.CreditEnquiryRequest) error
}

func (m *mockRedisRepository) SaveRequest(req *pb.CreditEnquiryRequest) error {
	return m.saveFunc(req)
}

func (m *mockRedisRepository) GetRequest(requestID string) (*pb.CreditEnquiryRequest, error) {
	return nil, nil
}

// mockSpannerRepository implements the CreditEnquiryRepository interface for testing
type mockSpannerRepository struct {
	saveFunc func(context.Context, *pb.CreditEnquiryRequest) error
}

func (m *mockSpannerRepository) SaveCreditEnquiry(ctx context.Context, req *pb.CreditEnquiryRequest) error {
	return m.saveFunc(ctx, req)
}

func (m *mockSpannerRepository) GetCreditEnquiry(ctx context.Context, requestID string, version string) (*pb.CreditEnquiryRequest, error) {
	return nil, nil
}

// mockPublisher implements the CreditEnquiryPublisher interface for testing
type mockPublisher struct {
	publishFunc func(context.Context, *pb.CreditEnquiryRequest) error
}

func (m *mockPublisher) PublishCreditEnquiryEvent(ctx context.Context, req *pb.CreditEnquiryRequest) error {
	return m.publishFunc(ctx, req)
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
		name             string
		request          *pb.CreditEnquiryRequest
		validatorError   error
		redisRepoError   error
		spannerRepoError error
		wantSuccess      bool
		wantMessage      string
	}{
		{
			name:             "valid request",
			request:          validRequest,
			validatorError:   nil,
			redisRepoError:   nil,
			spannerRepoError: nil,
			wantSuccess:      true,
			wantMessage:      "Credit enquiry request processed successfully",
		},
		{
			name:             "validation failed",
			request:          validRequest,
			validatorError:   assert.AnError,
			redisRepoError:   nil,
			spannerRepoError: nil,
			wantSuccess:      false,
			wantMessage:      assert.AnError.Error(),
		},
		{
			name:             "redis repository error",
			request:          validRequest,
			validatorError:   nil,
			redisRepoError:   assert.AnError,
			spannerRepoError: nil,
			wantSuccess:      false,
			wantMessage:      assert.AnError.Error(),
		},
		{
			name:             "spanner repository error",
			request:          validRequest,
			validatorError:   nil,
			redisRepoError:   nil,
			spannerRepoError: assert.AnError,
			wantSuccess:      false,
			wantMessage:      assert.AnError.Error(),
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
			validatorError:   nil,
			redisRepoError:   nil,
			spannerRepoError: nil,
			wantSuccess:      false,
			wantMessage:      "invalid UUID length",
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

			// Create mock Redis repository
			redisRepo := &mockRedisRepository{
				saveFunc: func(req *pb.CreditEnquiryRequest) error {
					return tt.redisRepoError
				},
			}

			// Create mock Spanner repository
			spannerRepo := &mockSpannerRepository{
				saveFunc: func(ctx context.Context, req *pb.CreditEnquiryRequest) error {
					return tt.spannerRepoError
				},
			}

			// Create mock publisher
			publisher := &mockPublisher{
				publishFunc: func(ctx context.Context, req *pb.CreditEnquiryRequest) error {
					return nil
				},
			}

			// Create server instance
			server := NewCreditEnquiryServer(validator, redisRepo, spannerRepo, publisher)

			// Process the request
			response, err := server.ProcessCreditEnquiry(context.Background(), tt.request)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantSuccess, response.Success)
			assert.Equal(t, tt.wantMessage, response.Message)
		})
	}
}
