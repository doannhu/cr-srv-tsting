package request_cache

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go-loan-service-v3/internal/credit_enquiry/entity"
	creditEnquiryProto "go-loan-service-v3/proto/credit_enquiry"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func setupTestRedis(t *testing.T) (*redis.Client, func()) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to create miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return client, func() {
		client.Close()
		mr.Close()
	}
}

func TestRedisRepository_SaveCreditEnquiry_CompareRequests(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := NewRedisRepository(client)

	// Generate a valid UUID
	validUUID := uuid.New()
	validUUIDBytes, err := validUUID.MarshalBinary()
	assert.NoError(t, err)
	validUUIDStr := validUUID.String()

	// Create base request
	baseRequest := &creditEnquiryProto.CreditEnquiryRequest{
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
		ProductName:                      "Home Loan",
		ProductCode:                      "HL001",
	}

	// Create a deep copy of the base request for modifications
	differentLoanRequest := proto.Clone(baseRequest).(*creditEnquiryProto.CreditEnquiryRequest)
	differentLoanRequest.LoanAmount = 200000

	differentStateRequest := proto.Clone(baseRequest).(*creditEnquiryProto.CreditEnquiryRequest)
	differentStateRequest.EnquiryState = "PROCESSING"

	differentProductRequest := proto.Clone(baseRequest).(*creditEnquiryProto.CreditEnquiryRequest)
	differentProductRequest.ProductName = "Personal Loan"
	differentProductRequest.ProductCode = "PL001"

	tests := []struct {
		name          string
		existingReq   *creditEnquiryProto.CreditEnquiryRequest
		incomingReq   *creditEnquiryProto.CreditEnquiryRequest
		setupCache    bool
		expectedError string
	}{
		{
			name:          "identical requests",
			existingReq:   baseRequest,
			incomingReq:   baseRequest,
			setupCache:    true,
			expectedError: "",
		},
		{
			name:          "different loan amount",
			existingReq:   baseRequest,
			incomingReq:   differentLoanRequest,
			setupCache:    true,
			expectedError: "request already exists with different data",
		},
		{
			name:          "different enquiry state",
			existingReq:   baseRequest,
			incomingReq:   differentStateRequest,
			setupCache:    true,
			expectedError: "request already exists with different data",
		},
		{
			name:          "different product",
			existingReq:   baseRequest,
			incomingReq:   differentProductRequest,
			setupCache:    true,
			expectedError: "request already exists with different data",
		},
		{
			name:          "no existing request",
			existingReq:   nil,
			incomingReq:   baseRequest,
			setupCache:    false,
			expectedError: "",
		},
		{
			name:        "invalid UUID",
			existingReq: nil,
			incomingReq: &creditEnquiryProto.CreditEnquiryRequest{
				RequestId:    []byte("invalid-uuid"),
				EnquiryState: "NEW",
			},
			setupCache:    false,
			expectedError: "invalid request ID: invalid UUID (got 12 bytes)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear Redis before each test
			client.FlushAll(context.Background())

			if tt.setupCache && tt.existingReq != nil {
				// Setup existing request in cache
				cacheEntry := entity.RequestCache{
					RequestID:           validUUIDStr,
					RequestType:         "CREDIT_ENQUIRY",
					RequestDataPayload:  tt.existingReq,
					ResponseDataPayload: "",
					UpdatedTime:         time.Now(),
				}
				cacheJSON, err := json.Marshal(cacheEntry)
				assert.NoError(t, err)
				key := "request_cache:" + validUUIDStr
				err = client.Set(context.Background(), key, cacheJSON, 0).Err()
				assert.NoError(t, err, "Failed to setup test cache")
			}

			// Test saving the incoming request
			err := repo.SaveCreditEnquiry(context.Background(), tt.incomingReq)

			if tt.expectedError != "" {
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)

				// Verify the state in Redis
				key := "request_cache:" + validUUIDStr
				val, err := client.Get(context.Background(), key).Result()
				assert.NoError(t, err)

				var savedEntry entity.RequestCache
				err = json.Unmarshal([]byte(val), &savedEntry)
				assert.NoError(t, err)

				assert.Equal(t, validUUIDStr, savedEntry.RequestID)
				assert.Equal(t, tt.incomingReq.EnquiryState, savedEntry.RequestDataPayload.EnquiryState)
				assert.Equal(t, tt.incomingReq.LoanAmount, savedEntry.RequestDataPayload.LoanAmount)
				assert.Equal(t, tt.incomingReq.ProductName, savedEntry.RequestDataPayload.ProductName)
				assert.Equal(t, tt.incomingReq.ProductCode, savedEntry.RequestDataPayload.ProductCode)
			}
		})
	}
}

func TestRedisRepository_GetCreditEnquiry(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := NewRedisRepository(client)

	// Generate a valid UUID
	validUUID := uuid.New()
	validUUIDBytes, err := validUUID.MarshalBinary()
	assert.NoError(t, err)
	validUUIDStr := validUUID.String()

	// Create test request
	testRequest := &creditEnquiryProto.CreditEnquiryRequest{
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
		ProductName:                      "Home Loan",
		ProductCode:                      "HL001",
	}

	// Save test request
	err = repo.SaveCreditEnquiry(context.Background(), testRequest)
	assert.NoError(t, err)

	// Test cases
	tests := []struct {
		name          string
		requestID     string
		version       string
		expectedError string
	}{
		{
			name:          "valid request",
			requestID:     validUUIDStr,
			version:       "1.0",
			expectedError: "",
		},
		{
			name:          "non-existent request",
			requestID:     "non-existent",
			version:       "1.0",
			expectedError: "request not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := repo.GetCreditEnquiry(context.Background(), tt.requestID, tt.version)

			if tt.expectedError != "" {
				assert.EqualError(t, err, tt.expectedError)
				assert.Nil(t, req)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, req)
				assert.Equal(t, testRequest.EnquiryState, req.EnquiryState)
				assert.Equal(t, testRequest.LoanAmount, req.LoanAmount)
				assert.Equal(t, testRequest.ProductName, req.ProductName)
				assert.Equal(t, testRequest.ProductCode, req.ProductCode)
			}
		})
	}
}
