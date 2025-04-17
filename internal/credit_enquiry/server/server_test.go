package server

import (
	"bytes"
	"context"
	"log"
	"testing"

	"go-loan-service-v3/internal/credit_enquiry/errors"
	pb "go-loan-service-v3/proto"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// mockValidator implements the Validator interface for testing
type mockValidator struct {
	validateFunc func(*pb.CreditEnquiryRequest) (*errors.ValidationResponse, error)
}

func (m *mockValidator) ValidateRequest(req *pb.CreditEnquiryRequest) (*errors.ValidationResponse, error) {
	return m.validateFunc(req)
}

// mockRedisRepository implements the RequestCacheRepository interface for testing
type mockRedisRepository struct {
	saveCreditEnquiryFunc func(context.Context, *pb.CreditEnquiryRequest) error
	getCreditEnquiryFunc  func(context.Context, string, string) (*pb.CreditEnquiryRequest, error)
}

func (m *mockRedisRepository) SaveCreditEnquiry(ctx context.Context, req *pb.CreditEnquiryRequest) error {
	return m.saveCreditEnquiryFunc(ctx, req)
}

func (m *mockRedisRepository) GetCreditEnquiry(ctx context.Context, requestID string, version string) (*pb.CreditEnquiryRequest, error) {
	return m.getCreditEnquiryFunc(ctx, requestID, version)
}

// mockSpannerRepository implements the CreditEnquiryRepository interface for testing
type mockSpannerRepository struct {
	saveCreditEnquiryFunc func(context.Context, *pb.CreditEnquiryRequest) error
	getCreditEnquiryFunc  func(context.Context, string, string) (*pb.CreditEnquiryRequest, error)
}

func (m *mockSpannerRepository) SaveCreditEnquiry(ctx context.Context, req *pb.CreditEnquiryRequest) error {
	return m.saveCreditEnquiryFunc(ctx, req)
}

func (m *mockSpannerRepository) GetCreditEnquiry(ctx context.Context, requestID string, version string) (*pb.CreditEnquiryRequest, error) {
	return m.getCreditEnquiryFunc(ctx, requestID, version)
}

func (m *mockSpannerRepository) SaveRequest(req *pb.CreditEnquiryRequest) error {
	return m.saveCreditEnquiryFunc(context.Background(), req)
}

// mockPublisher implements the CreditEnquiryPublisher interface for testing
type mockPublisher struct {
	publishFunc func(context.Context, *pb.CreditEnquiryRequest) error
}

func (m *mockPublisher) PublishCreditEnquiryEvent(ctx context.Context, req *pb.CreditEnquiryRequest) error {
	return m.publishFunc(ctx, req)
}

func TestCreditEnquiryServer_ProcessCreditEnquiry(t *testing.T) {
	// Create a test logger that writes to a buffer
	var logBuffer bytes.Buffer
	logger := log.New(&logBuffer, "", log.LstdFlags)

	// Create mock validator
	validator := &mockValidator{
		validateFunc: func(req *pb.CreditEnquiryRequest) (*errors.ValidationResponse, error) {
			return errors.NewValidationResponse(true, nil, nil), nil
		},
	}

	// Create mock Redis repository
	redisRepo := &mockRedisRepository{
		saveCreditEnquiryFunc: func(ctx context.Context, req *pb.CreditEnquiryRequest) error {
			return nil
		},
		getCreditEnquiryFunc: func(ctx context.Context, requestID string, version string) (*pb.CreditEnquiryRequest, error) {
			return nil, nil
		},
	}

	// Create mock Spanner repository
	spannerRepo := &mockSpannerRepository{
		saveCreditEnquiryFunc: func(ctx context.Context, req *pb.CreditEnquiryRequest) error {
			return nil
		},
		getCreditEnquiryFunc: func(ctx context.Context, requestID string, version string) (*pb.CreditEnquiryRequest, error) {
			return nil, nil
		},
	}

	// Create mock publisher
	publisher := &mockPublisher{
		publishFunc: func(ctx context.Context, req *pb.CreditEnquiryRequest) error {
			return nil
		},
	}

	// Create the server
	server := NewServer(logger, validator, redisRepo, spannerRepo, publisher)

	tests := []struct {
		name    string
		request *pb.CreditEnquiryRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: &pb.CreditEnquiryRequest{
				RequestId:                        []byte(uuid.New().String()),
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := server.ProcessCreditEnquiry(ctx, tt.request)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
