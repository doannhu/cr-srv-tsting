package sop

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-loan-service-v3/internal/credit_enquiry/utils"
	pb "go-loan-service-v3/proto/sop"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockSOPClient struct {
	attempts       int
	permanentError bool
}

func (m *mockSOPClient) GetConsolidatedSOP(ctx context.Context, in *pb.SOPRequest, opts ...grpc.CallOption) (*pb.SOPResponse, error) {
	m.attempts++

	// For the permanent error test case, always return an error
	if m.permanentError {
		return nil, errors.New("permanent error")
	}

	// For the transient error test case, succeed on the third attempt
	if m.attempts == 3 {
		return &pb.SOPResponse{
			SopAssessmentId:                  "sop-123",
			ServiceabilityAssessmentId:       "sa-123",
			TotalMonthlyNetIncomeAmount:      5000.0,
			TotalAnnualGrossIncome:           60000.0,
			TotalSavingsAmount:               10000.0,
			TotalNumberOfContinuingHomeLoans: 1,
		}, nil
	}

	// Return a transient error for the first two attempts
	return nil, status.Error(codes.Unavailable, "temporary error")
}

func TestGetConsolidatedSOP(t *testing.T) {
	tests := []struct {
		name             string
		client           *mockSOPClient
		expectedError    error
		expectedResponse *pb.SOPResponse
	}{
		{
			name:   "successful response",
			client: &mockSOPClient{},
			expectedResponse: &pb.SOPResponse{
				SopAssessmentId:                  "sop-123",
				ServiceabilityAssessmentId:       "sa-123",
				TotalMonthlyNetIncomeAmount:      5000.0,
				TotalAnnualGrossIncome:           60000.0,
				TotalSavingsAmount:               10000.0,
				TotalNumberOfContinuingHomeLoans: 1,
			},
		},
		{
			name:   "transient error with retry success",
			client: &mockSOPClient{},
			expectedResponse: &pb.SOPResponse{
				SopAssessmentId:                  "sop-123",
				ServiceabilityAssessmentId:       "sa-123",
				TotalMonthlyNetIncomeAmount:      5000.0,
				TotalAnnualGrossIncome:           60000.0,
				TotalSavingsAmount:               10000.0,
				TotalNumberOfContinuingHomeLoans: 1,
			},
		},
		{
			name:          "permanent error",
			client:        &mockSOPClient{permanentError: true},
			expectedError: errors.New("permanent error"),
		},
	}

	retryConfig := &utils.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewSOPServiceClient(nil, retryConfig)
			client.(*sopServiceClient).client = tt.client
			response, err := client.GetConsolidatedSOP(context.Background(), &pb.SOPRequest{
				CreditEnquiryId:      "test-id",
				CreditEnquiryVersion: "1.0",
			})

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Nil(t, response)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, response)
				assert.Equal(t, tt.expectedResponse.SopAssessmentId, response.SopAssessmentId)
				assert.Equal(t, tt.expectedResponse.ServiceabilityAssessmentId, response.ServiceabilityAssessmentId)
				assert.Equal(t, tt.expectedResponse.TotalMonthlyNetIncomeAmount, response.TotalMonthlyNetIncomeAmount)
				assert.Equal(t, tt.expectedResponse.TotalAnnualGrossIncome, response.TotalAnnualGrossIncome)
				assert.Equal(t, tt.expectedResponse.TotalSavingsAmount, response.TotalSavingsAmount)
				assert.Equal(t, tt.expectedResponse.TotalNumberOfContinuingHomeLoans, response.TotalNumberOfContinuingHomeLoans)
			}
		})
	}
}
