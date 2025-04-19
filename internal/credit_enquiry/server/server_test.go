package server

import (
	"bytes"
	"context"
	"errors"
	"log"
	"testing"
	"time"

	"go-loan-service-v3/internal/credit_enquiry/entity"
	cerrors "go-loan-service-v3/internal/credit_enquiry/errors"
	pb "go-loan-service-v3/proto"
	sopPb "go-loan-service-v3/proto/sop"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

// mockValidator implements the Validator interface for testing
type mockValidator struct {
	validateFunc func(*pb.CreditEnquiryRequest) (*cerrors.ValidationResponse, error)
}

func (m *mockValidator) ValidateRequest(req *pb.CreditEnquiryRequest) (*cerrors.ValidationResponse, error) {
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

// mockSopRepository implements the SopRepository interface for testing
type mockSopRepository struct {
	saveSopFunc func(context.Context, *entity.Sop) error
}

func (m *mockSopRepository) SaveSop(ctx context.Context, sop *entity.Sop) error {
	return m.saveSopFunc(ctx, sop)
}

func (m *mockSopRepository) GetSop(ctx context.Context, creditEnquiryID string, version string) (*entity.Sop, error) {
	return nil, nil
}

// mockSOPService implements the SOPService interface for testing
type mockSOPService struct {
	getConsolidatedSOPFunc func(context.Context, *sopPb.SOPRequest) (*sopPb.SOPResponse, error)
}

func (m *mockSOPService) GetConsolidatedSOP(ctx context.Context, req *sopPb.SOPRequest) (*sopPb.SOPResponse, error) {
	return m.getConsolidatedSOPFunc(ctx, req)
}

func TestCreditEnquiryServer_ProcessCreditEnquiry(t *testing.T) {
	// Create a test logger that writes to a buffer
	var logBuffer bytes.Buffer
	logger := log.New(&logBuffer, "", log.LstdFlags)

	// Generate a valid UUID for testing
	validUUID := uuid.New()
	validUUIDBytes, _ := validUUID.MarshalBinary()

	tests := []struct {
		name        string
		request     *pb.CreditEnquiryRequest
		setupMocks  func() (*mockValidator, *mockRedisRepository, *mockSpannerRepository, *mockPublisher, *mockSopRepository, *mockSOPService)
		wantErr     bool
		wantSuccess bool
		wantCode    string
		wantMessage string
	}{
		{
			name: "valid request - successful processing",
			request: &pb.CreditEnquiryRequest{
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
			setupMocks: func() (*mockValidator, *mockRedisRepository, *mockSpannerRepository, *mockPublisher, *mockSopRepository, *mockSOPService) {
				validator := &mockValidator{
					validateFunc: func(req *pb.CreditEnquiryRequest) (*cerrors.ValidationResponse, error) {
						return cerrors.NewValidationResponse(true, nil, nil), nil
					},
				}
				redisRepo := &mockRedisRepository{
					saveCreditEnquiryFunc: func(ctx context.Context, req *pb.CreditEnquiryRequest) error {
						return nil
					},
				}
				spannerRepo := &mockSpannerRepository{
					saveCreditEnquiryFunc: func(ctx context.Context, req *pb.CreditEnquiryRequest) error {
						return nil
					},
				}
				publisher := &mockPublisher{
					publishFunc: func(ctx context.Context, req *pb.CreditEnquiryRequest) error {
						return nil
					},
				}
				sopRepo := &mockSopRepository{
					saveSopFunc: func(ctx context.Context, sop *entity.Sop) error {
						return nil
					},
				}
				sopService := &mockSOPService{
					getConsolidatedSOPFunc: func(ctx context.Context, req *sopPb.SOPRequest) (*sopPb.SOPResponse, error) {
						return &sopPb.SOPResponse{
							SopAssessmentId:                  "test-sop-id",
							ServiceabilityAssessmentId:       "test-serviceability-id",
							TotalMonthlyNetIncomeAmount:      5000,
							TotalAnnualGrossIncome:           60000,
							TotalSavingsAmount:               10000,
							TotalNumberOfContinuingHomeLoans: 0,
						}, nil
					},
				}
				return validator, redisRepo, spannerRepo, publisher, sopRepo, sopService
			},
			wantErr:     false,
			wantSuccess: true,
			wantMessage: "Credit enquiry processed successfully",
		},
		{
			name: "empty request ID",
			request: &pb.CreditEnquiryRequest{
				RequestId: []byte{},
			},
			setupMocks: func() (*mockValidator, *mockRedisRepository, *mockSpannerRepository, *mockPublisher, *mockSopRepository, *mockSOPService) {
				return &mockValidator{}, &mockRedisRepository{}, &mockSpannerRepository{}, &mockPublisher{}, &mockSopRepository{}, &mockSOPService{}
			},
			wantErr:     false,
			wantSuccess: false,
			wantCode:    cerrors.ErrBadRequest,
			wantMessage: "request_id is required",
		},
		{
			name: "invalid UUID length",
			request: &pb.CreditEnquiryRequest{
				RequestId: []byte("invalid-uuid"),
			},
			setupMocks: func() (*mockValidator, *mockRedisRepository, *mockSpannerRepository, *mockPublisher, *mockSopRepository, *mockSOPService) {
				return &mockValidator{}, &mockRedisRepository{}, &mockSpannerRepository{}, &mockPublisher{}, &mockSopRepository{}, &mockSOPService{}
			},
			wantErr:     false,
			wantSuccess: false,
			wantCode:    cerrors.ErrBadRequest,
			wantMessage: "invalid UUID length",
		},
		{
			name: "validation failure",
			request: &pb.CreditEnquiryRequest{
				RequestId: validUUIDBytes,
			},
			setupMocks: func() (*mockValidator, *mockRedisRepository, *mockSpannerRepository, *mockPublisher, *mockSopRepository, *mockSOPService) {
				validator := &mockValidator{
					validateFunc: func(req *pb.CreditEnquiryRequest) (*cerrors.ValidationResponse, error) {
						return cerrors.NewValidationResponse(false, cerrors.NewValidationError(
							cerrors.ErrBadRequest,
							"validation failed",
							nil,
						), nil), nil
					},
				}
				return validator, &mockRedisRepository{}, &mockSpannerRepository{}, &mockPublisher{}, &mockSopRepository{}, &mockSOPService{}
			},
			wantErr:     false,
			wantSuccess: false,
			wantCode:    cerrors.ErrBadRequest,
			wantMessage: "validation failed",
		},
		{
			name: "validation internal error",
			request: &pb.CreditEnquiryRequest{
				RequestId: validUUIDBytes,
			},
			setupMocks: func() (*mockValidator, *mockRedisRepository, *mockSpannerRepository, *mockPublisher, *mockSopRepository, *mockSOPService) {
				validator := &mockValidator{
					validateFunc: func(req *pb.CreditEnquiryRequest) (*cerrors.ValidationResponse, error) {
						return nil, errors.New("internal error")
					},
				}
				return validator, &mockRedisRepository{}, &mockSpannerRepository{}, &mockPublisher{}, &mockSopRepository{}, &mockSOPService{}
			},
			wantErr:     false,
			wantSuccess: false,
			wantCode:    cerrors.ErrInternalError,
			wantMessage: "Internal validation error",
		},
		{
			name: "redis storage failure",
			request: &pb.CreditEnquiryRequest{
				RequestId: validUUIDBytes,
			},
			setupMocks: func() (*mockValidator, *mockRedisRepository, *mockSpannerRepository, *mockPublisher, *mockSopRepository, *mockSOPService) {
				validator := &mockValidator{
					validateFunc: func(req *pb.CreditEnquiryRequest) (*cerrors.ValidationResponse, error) {
						return cerrors.NewValidationResponse(true, nil, nil), nil
					},
				}
				redisRepo := &mockRedisRepository{
					saveCreditEnquiryFunc: func(ctx context.Context, req *pb.CreditEnquiryRequest) error {
						return errors.New("redis error")
					},
				}
				return validator, redisRepo, &mockSpannerRepository{}, &mockPublisher{}, &mockSopRepository{}, &mockSOPService{}
			},
			wantErr:     false,
			wantSuccess: false,
			wantCode:    cerrors.ErrStorageError,
			wantMessage: "Failed to save request to cache",
		},
		{
			name: "spanner storage failure",
			request: &pb.CreditEnquiryRequest{
				RequestId: validUUIDBytes,
			},
			setupMocks: func() (*mockValidator, *mockRedisRepository, *mockSpannerRepository, *mockPublisher, *mockSopRepository, *mockSOPService) {
				validator := &mockValidator{
					validateFunc: func(req *pb.CreditEnquiryRequest) (*cerrors.ValidationResponse, error) {
						return cerrors.NewValidationResponse(true, nil, nil), nil
					},
				}
				redisRepo := &mockRedisRepository{
					saveCreditEnquiryFunc: func(ctx context.Context, req *pb.CreditEnquiryRequest) error {
						return nil
					},
				}
				spannerRepo := &mockSpannerRepository{
					saveCreditEnquiryFunc: func(ctx context.Context, req *pb.CreditEnquiryRequest) error {
						return errors.New("spanner error")
					},
				}
				return validator, redisRepo, spannerRepo, &mockPublisher{}, &mockSopRepository{}, &mockSOPService{}
			},
			wantErr:     false,
			wantSuccess: false,
			wantCode:    cerrors.ErrStorageError,
			wantMessage: "Failed to save request to database",
		},
		{
			name: "publisher failure - request still succeeds",
			request: &pb.CreditEnquiryRequest{
				RequestId: validUUIDBytes,
			},
			setupMocks: func() (*mockValidator, *mockRedisRepository, *mockSpannerRepository, *mockPublisher, *mockSopRepository, *mockSOPService) {
				validator := &mockValidator{
					validateFunc: func(req *pb.CreditEnquiryRequest) (*cerrors.ValidationResponse, error) {
						return cerrors.NewValidationResponse(true, nil, nil), nil
					},
				}
				redisRepo := &mockRedisRepository{
					saveCreditEnquiryFunc: func(ctx context.Context, req *pb.CreditEnquiryRequest) error {
						return nil
					},
				}
				spannerRepo := &mockSpannerRepository{
					saveCreditEnquiryFunc: func(ctx context.Context, req *pb.CreditEnquiryRequest) error {
						return nil
					},
				}
				publisher := &mockPublisher{
					publishFunc: func(ctx context.Context, req *pb.CreditEnquiryRequest) error {
						return errors.New("publisher error")
					},
				}
				sopRepo := &mockSopRepository{
					saveSopFunc: func(ctx context.Context, sop *entity.Sop) error {
						return nil
					},
				}
				sopService := &mockSOPService{
					getConsolidatedSOPFunc: func(ctx context.Context, req *sopPb.SOPRequest) (*sopPb.SOPResponse, error) {
						return &sopPb.SOPResponse{
							SopAssessmentId:                  "test-sop-id",
							ServiceabilityAssessmentId:       "test-serviceability-id",
							TotalMonthlyNetIncomeAmount:      5000,
							TotalAnnualGrossIncome:           60000,
							TotalSavingsAmount:               10000,
							TotalNumberOfContinuingHomeLoans: 0,
						}, nil
					},
				}
				return validator, redisRepo, spannerRepo, publisher, sopRepo, sopService
			},
			wantErr:     false,
			wantSuccess: true,
			wantCode:    cerrors.ErrPublishError,
			wantMessage: "Credit enquiry request processed successfully, but event publishing failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			validator, redisRepo, spannerRepo, publisher, sopRepo, sopService := tt.setupMocks()

			// Create the server
			server := NewServer(logger, validator, redisRepo, spannerRepo, sopRepo, publisher, sopService)

			// Process the request
			ctx := context.Background()
			resp, err := server.ProcessCreditEnquiry(ctx, tt.request)

			// Check error
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// Check response
			assert.NotNil(t, resp)
			assert.Equal(t, tt.wantSuccess, resp.Success)
			assert.Equal(t, tt.wantMessage, resp.Message)
			if tt.wantCode != "" {
				assert.Equal(t, tt.wantCode, resp.Code)
			}
		})
	}
}

func TestNewServer(t *testing.T) {
	logger := log.New(&bytes.Buffer{}, "", log.LstdFlags)
	validator := &mockValidator{}
	redisRepo := &mockRedisRepository{}
	spannerRepo := &mockSpannerRepository{}
	publisher := &mockPublisher{}
	sopRepo := &mockSopRepository{}
	sopService := &mockSOPService{}

	server := NewServer(logger, validator, redisRepo, spannerRepo, sopRepo, publisher, sopService)

	assert.NotNil(t, server)
	assert.Equal(t, logger, server.logger)
	assert.Equal(t, validator, server.validator)
	assert.Equal(t, redisRepo, server.redisRepo)
	assert.Equal(t, spannerRepo, server.spannerRepo)
	assert.Equal(t, publisher, server.publisher)
	assert.Equal(t, sopRepo, server.sopRepo)
	assert.Equal(t, sopService, server.sopService)
}

func TestStartServer(t *testing.T) {
	// Create mock dependencies
	validator := &mockValidator{
		validateFunc: func(req *pb.CreditEnquiryRequest) (*cerrors.ValidationResponse, error) {
			return cerrors.NewValidationResponse(true, nil, nil), nil
		},
	}
	redisRepo := &mockRedisRepository{
		saveCreditEnquiryFunc: func(ctx context.Context, req *pb.CreditEnquiryRequest) error {
			return nil
		},
		getCreditEnquiryFunc: func(ctx context.Context, requestID string, version string) (*pb.CreditEnquiryRequest, error) {
			return nil, nil
		},
	}
	spannerRepo := &mockSpannerRepository{
		saveCreditEnquiryFunc: func(ctx context.Context, req *pb.CreditEnquiryRequest) error {
			return nil
		},
		getCreditEnquiryFunc: func(ctx context.Context, requestID string, version string) (*pb.CreditEnquiryRequest, error) {
			return nil, nil
		},
	}
	publisher := &mockPublisher{
		publishFunc: func(ctx context.Context, req *pb.CreditEnquiryRequest) error {
			return nil
		},
	}
	sopRepo := &mockSopRepository{
		saveSopFunc: func(ctx context.Context, sop *entity.Sop) error {
			return nil
		},
	}
	sopService := &mockSOPService{
		getConsolidatedSOPFunc: func(ctx context.Context, req *sopPb.SOPRequest) (*sopPb.SOPResponse, error) {
			return &sopPb.SOPResponse{
				SopAssessmentId:                  "test-sop-id",
				ServiceabilityAssessmentId:       "test-serviceability-id",
				TotalMonthlyNetIncomeAmount:      5000,
				TotalAnnualGrossIncome:           60000,
				TotalSavingsAmount:               10000,
				TotalNumberOfContinuingHomeLoans: 0,
			}, nil
		},
	}

	// Start server in a goroutine
	go func() {
		err := StartServer("50051", validator, redisRepo, spannerRepo, sopRepo, publisher, sopService)
		assert.NoError(t, err)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Try to connect to the server
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	assert.NoError(t, err)
	defer conn.Close()

	// Create a client
	client := pb.NewCreditEnquiryServiceClient(conn)

	// Test that the server is responding
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Create a valid request
	validUUID := uuid.New()
	validUUIDBytes, _ := validUUID.MarshalBinary()
	req := &pb.CreditEnquiryRequest{
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

	// Make a request
	resp, err := client.ProcessCreditEnquiry(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	// Test invalid port
	err = StartServer("invalid", validator, redisRepo, spannerRepo, sopRepo, publisher, sopService)
	assert.Error(t, err)
}
