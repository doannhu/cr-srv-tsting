package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go-loan-service-v3/internal/credit_enquiry/entity"
	creditEnquiryProto "go-loan-service-v3/proto"

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

func TestRedisRepository_SaveRequest_CompareRequests(t *testing.T) {
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
	}

	// Create a deep copy of the base request for modifications
	differentLoanRequest := proto.Clone(baseRequest).(*creditEnquiryProto.CreditEnquiryRequest)
	differentLoanRequest.LoanAmount = 200000

	differentStateRequest := proto.Clone(baseRequest).(*creditEnquiryProto.CreditEnquiryRequest)
	differentStateRequest.EnquiryState = "PROCESSING"

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
			name:          "no existing request",
			existingReq:   nil,
			incomingReq:   baseRequest,
			setupCache:    false,
			expectedError: "",
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
			err := repo.SaveRequest(tt.incomingReq)

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
			}
		})
	}
}
