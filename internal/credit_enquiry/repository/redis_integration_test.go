package repository

import (
	"context"
	"testing"

	"go-loan-service-v3/internal/credit_enquiry/interfaces"
	pb "go-loan-service-v3/proto"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type RedisCacheTestSuite struct {
	suite.Suite
	RedisURL    string
	container   testcontainers.Container
	redisClient interfaces.RequestCacheRepository
	rdb         *redis.Client
}

func (r *RedisCacheTestSuite) SetupSuite() {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "redis:latest",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	r.Require().NoError(err)

	host, err := container.Host(ctx)
	r.Require().NoError(err)

	port, err := container.MappedPort(ctx, "6379")
	r.Require().NoError(err)

	r.RedisURL = host + ":" + port.Port()
	r.container = container

	// Initialize Redis client
	client := redis.NewClient(&redis.Options{
		Addr: r.RedisURL,
	})
	r.rdb = client
	r.redisClient = NewRedisRepository(client)
}

func (r *RedisCacheTestSuite) TearDownSuite() {
	ctx := context.Background()
	if r.container != nil {
		r.Require().NoError(r.container.Terminate(ctx))
	}
}

func (r *RedisCacheTestSuite) TestSaveAndGetCreditEnquiry() {
	ctx := context.Background()
	requestUUID := uuid.New()
	requestID := requestUUID.String()
	request := &pb.CreditEnquiryRequest{
		RequestId:                        requestUUID[:],
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

	// Test saving
	err := r.redisClient.SaveCreditEnquiry(ctx, request)
	r.Require().NoError(err)

	// Test retrieving
	retrieved, err := r.redisClient.GetCreditEnquiry(ctx, requestID, "")
	r.Require().NoError(err)
	r.Require().NotNil(retrieved)
	r.Equal(request.EnquiryState, retrieved.EnquiryState)
	r.Equal(request.ApplicationNumber, retrieved.ApplicationNumber)
}

func (r *RedisCacheTestSuite) TestCacheExpiration() {
	ctx := context.Background()
	requestUUID := uuid.New()
	requestID := requestUUID.String()
	request := &pb.CreditEnquiryRequest{
		RequestId:                        requestUUID[:],
		EnquiryState:                     "NEW",
		ApplicationNumber:                "APP456",
		LoanAmount:                       200000,
		LoanPurpose:                      "Home Purchase",
		InitialStructureTermMonth:        360,
		TotalMonthlyNetIncomeAmount:      6000,
		TotalAnnualGrossIncome:           72000,
		TotalSavingsAmount:               20000,
		TotalNumberOfContinuingHomeLoans: 0,
	}

	// Save request
	err := r.redisClient.SaveCreditEnquiry(ctx, request)
	r.Require().NoError(err)

	// Verify the request exists immediately after saving
	retrieved, err := r.redisClient.GetCreditEnquiry(ctx, requestID, "")
	r.Require().NoError(err)
	r.Require().NotNil(retrieved)

	// Manually delete the key to simulate expiration
	key := "request_cache:" + requestID
	err = r.rdb.Del(ctx, key).Err()
	r.Require().NoError(err)

	// Try to retrieve deleted data
	retrieved, err = r.redisClient.GetCreditEnquiry(ctx, requestID, "")
	r.Require().Error(err)
	r.Equal("request not found", err.Error())
	r.Nil(retrieved)
}

func TestRedisCacheSuite(t *testing.T) {
	suite.Run(t, new(RedisCacheTestSuite))
}
