package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go-loan-service-v3/internal/credit_enquiry/entity"
	creditEnquiryProto "go-loan-service-v3/proto"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

const (
	requestType = "CREDIT_ENQUIRY"
	keyPrefix   = "request_cache:"
)

type redisRepository struct {
	client *redis.Client
}

// NewRedisRepository creates a new Redis repository instance
func NewRedisRepository(client *redis.Client) entity.RequestCacheRepository {
	return &redisRepository{
		client: client,
	}
}

// SaveRequest implements RequestCacheRepository
func (r *redisRepository) SaveRequest(request *creditEnquiryProto.CreditEnquiryRequest) error {
	ctx := context.Background()

	// Convert request ID bytes to UUID string
	requestUUID, err := uuid.FromBytes(request.RequestId)
	if err != nil {
		return fmt.Errorf("invalid request ID: %w", err)
	}
	requestID := requestUUID.String()

	// Check if request already exists
	key := keyPrefix + requestID
	val, err := r.client.Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to check request existence: %w", err)
	}

	// If request exists, compare with incoming request
	if err != redis.Nil {
		var cacheEntry entity.RequestCache
		if err := json.Unmarshal([]byte(val), &cacheEntry); err != nil {
			return fmt.Errorf("failed to unmarshal existing request: %w", err)
		}

		// Compare the requests
		if !proto.Equal(cacheEntry.RequestDataPayload, request) {
			return errors.New("request already exists with different data")
		}
		return nil // Request exists and is identical, no need to save
	}

	// Create cache entry
	cacheEntry := entity.RequestCache{
		RequestID:           requestID,
		RequestType:         requestType,
		RequestDataPayload:  request,
		ResponseDataPayload: "",
		UpdatedTime:         time.Now(),
	}

	// Convert cache entry to JSON
	cacheJSON, err := json.Marshal(cacheEntry)
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	// Save to Redis
	err = r.client.Set(ctx, key, cacheJSON, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to save to Redis: %w", err)
	}

	return nil
}

// GetRequest implements RequestCacheRepository
func (r *redisRepository) GetRequest(requestID string) (*creditEnquiryProto.CreditEnquiryRequest, error) {
	ctx := context.Background()

	// Get from Redis
	key := keyPrefix + requestID
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, errors.New("request not found")
		}
		return nil, fmt.Errorf("failed to get from Redis: %w", err)
	}

	// Unmarshal cache entry
	var cacheEntry entity.RequestCache
	err = json.Unmarshal([]byte(val), &cacheEntry)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache entry: %w", err)
	}

	return cacheEntry.RequestDataPayload, nil
}
