package server

import (
	"bytes"
	"context"
	"errors"
	"log"
	"testing"

	"go-loan-service-v3/internal/credit_enquiry/entity"
	cerrors "go-loan-service-v3/internal/credit_enquiry/errors"
	creditEnquiryProto "go-loan-service-v3/proto/credit_enquiry"
	productAssessmentPb "go-loan-service-v3/proto/product_assessment"
	sopPb "go-loan-service-v3/proto/sop"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations
type mockValidator struct {
	mock.Mock
}

func (m *mockValidator) ValidateRequest(request *creditEnquiryProto.CreditEnquiryRequest) (*cerrors.ValidationResponse, error) {
	args := m.Called(request)
	return args.Get(0).(*cerrors.ValidationResponse), args.Error(1)
}

type mockRedisRepository struct {
	mock.Mock
}

func (m *mockRedisRepository) SaveCreditEnquiry(ctx context.Context, request *creditEnquiryProto.CreditEnquiryRequest) error {
	args := m.Called(ctx, request)
	return args.Error(0)
}

func (m *mockRedisRepository) GetCreditEnquiry(ctx context.Context, requestID string, version string) (*creditEnquiryProto.CreditEnquiryRequest, error) {
	args := m.Called(ctx, requestID, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*creditEnquiryProto.CreditEnquiryRequest), args.Error(1)
}

type mockSpannerRepository struct {
	mock.Mock
}

func (m *mockSpannerRepository) SaveCreditEnquiry(ctx context.Context, request *creditEnquiryProto.CreditEnquiryRequest) error {
	args := m.Called(ctx, request)
	return args.Error(0)
}

func (m *mockSpannerRepository) GetCreditEnquiry(ctx context.Context, requestID string, version string) (*creditEnquiryProto.CreditEnquiryRequest, error) {
	args := m.Called(ctx, requestID, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*creditEnquiryProto.CreditEnquiryRequest), args.Error(1)
}

func (m *mockSpannerRepository) SaveRequest(req *creditEnquiryProto.CreditEnquiryRequest) error {
	args := m.Called(req)
	return args.Error(0)
}

type mockSopRepository struct {
	mock.Mock
}

func (m *mockSopRepository) SaveSop(ctx context.Context, sop *entity.Sop) error {
	args := m.Called(ctx, sop)
	return args.Error(0)
}

func (m *mockSopRepository) GetSop(ctx context.Context, creditEnquiryID string, version string) (*entity.Sop, error) {
	args := m.Called(ctx, creditEnquiryID, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Sop), args.Error(1)
}

type mockPublisher struct {
	mock.Mock
}

func (m *mockPublisher) PublishCreditEnquiryEvent(ctx context.Context, request *creditEnquiryProto.CreditEnquiryRequest) error {
	args := m.Called(ctx, request)
	return args.Error(0)
}

type mockSOPService struct {
	mock.Mock
}

func (m *mockSOPService) GetConsolidatedSOP(ctx context.Context, request *sopPb.SOPRequest) (*sopPb.SOPResponse, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sopPb.SOPResponse), args.Error(1)
}

type mockProductAssessmentService struct {
	mock.Mock
}

func (m *mockProductAssessmentService) GetProductRate(ctx context.Context, request *productAssessmentPb.ProductRateRequest) (*productAssessmentPb.ProductRateResponse, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*productAssessmentPb.ProductRateResponse), args.Error(1)
}

type mockProductAssessmentRepository struct {
	mock.Mock
}

func (m *mockProductAssessmentRepository) SaveProductAssessment(ctx context.Context, assessment *entity.ProductAssessment) error {
	args := m.Called(ctx, assessment)
	return args.Error(0)
}

func (m *mockProductAssessmentRepository) GetProductAssessment(ctx context.Context, creditEnquiryID string, version string) (*entity.ProductAssessment, error) {
	args := m.Called(ctx, creditEnquiryID, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.ProductAssessment), args.Error(1)
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
		request     *creditEnquiryProto.CreditEnquiryRequest
		setupMocks  func() (*mockValidator, *mockRedisRepository, *mockSpannerRepository, *mockPublisher, *mockSopRepository, *mockSOPService, *mockProductAssessmentService, *mockProductAssessmentRepository)
		wantErr     bool
		wantSuccess bool
		wantCode    string
		wantMessage string
	}{
		{
			name: "valid request - successful processing",
			request: &creditEnquiryProto.CreditEnquiryRequest{
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
				ProductCode:                      "PROD001",
				ProductName:                      "Test Product",
			},
			setupMocks: func() (*mockValidator, *mockRedisRepository, *mockSpannerRepository, *mockPublisher, *mockSopRepository, *mockSOPService, *mockProductAssessmentService, *mockProductAssessmentRepository) {
				validator := new(mockValidator)
				redisRepo := new(mockRedisRepository)
				spannerRepo := new(mockSpannerRepository)
				publisher := new(mockPublisher)
				sopRepo := new(mockSopRepository)
				sopService := new(mockSOPService)
				productAssessmentService := new(mockProductAssessmentService)
				productAssessmentRepo := new(mockProductAssessmentRepository)

				// Setup validator mock
				validator.On("ValidateRequest", mock.Anything).Return(&cerrors.ValidationResponse{Success: true}, nil)

				// Setup Redis mock
				redisRepo.On("SaveCreditEnquiry", mock.Anything, mock.Anything).Return(nil)

				// Setup Spanner mock
				spannerRepo.On("SaveCreditEnquiry", mock.Anything, mock.Anything).Return(nil)

				// Setup SOP service mock
				sopService.On("GetConsolidatedSOP", mock.Anything, mock.Anything).Return(&sopPb.SOPResponse{
					SopAssessmentId:            "test-sop-id",
					ServiceabilityAssessmentId: "test-serviceability-id",
				}, nil)

				// Setup SOP repository mock
				sopRepo.On("SaveSop", mock.Anything, mock.Anything).Return(nil)

				// Setup product assessment service mock
				productAssessmentService.On("GetProductRate", mock.Anything, mock.Anything).Return(&productAssessmentPb.ProductRateResponse{
					InitialStructureIndexRate: 5,
				}, nil)

				// Setup product assessment repository mock
				productAssessmentRepo.On("SaveProductAssessment", mock.Anything, mock.Anything).Return(nil)

				// Setup publisher mock
				publisher.On("PublishCreditEnquiryEvent", mock.Anything, mock.Anything).Return(nil)

				return validator, redisRepo, spannerRepo, publisher, sopRepo, sopService, productAssessmentService, productAssessmentRepo
			},
			wantErr:     false,
			wantSuccess: true,
			wantCode:    "",
			wantMessage: "Credit enquiry processed successfully",
		},
		{
			name: "empty request ID",
			request: &creditEnquiryProto.CreditEnquiryRequest{
				RequestId: []byte{},
			},
			setupMocks: func() (*mockValidator, *mockRedisRepository, *mockSpannerRepository, *mockPublisher, *mockSopRepository, *mockSOPService, *mockProductAssessmentService, *mockProductAssessmentRepository) {
				return &mockValidator{}, &mockRedisRepository{}, &mockSpannerRepository{}, &mockPublisher{}, &mockSopRepository{}, &mockSOPService{}, &mockProductAssessmentService{}, &mockProductAssessmentRepository{}
			},
			wantErr:     false,
			wantSuccess: false,
			wantCode:    cerrors.ErrBadRequest,
			wantMessage: "request_id is required",
		},
		{
			name: "invalid UUID length",
			request: &creditEnquiryProto.CreditEnquiryRequest{
				RequestId: []byte("invalid-uuid"),
			},
			setupMocks: func() (*mockValidator, *mockRedisRepository, *mockSpannerRepository, *mockPublisher, *mockSopRepository, *mockSOPService, *mockProductAssessmentService, *mockProductAssessmentRepository) {
				return &mockValidator{}, &mockRedisRepository{}, &mockSpannerRepository{}, &mockPublisher{}, &mockSopRepository{}, &mockSOPService{}, &mockProductAssessmentService{}, &mockProductAssessmentRepository{}
			},
			wantErr:     false,
			wantSuccess: false,
			wantCode:    cerrors.ErrBadRequest,
			wantMessage: "invalid UUID length",
		},
		{
			name: "validation failure",
			request: &creditEnquiryProto.CreditEnquiryRequest{
				RequestId: validUUIDBytes,
			},
			setupMocks: func() (*mockValidator, *mockRedisRepository, *mockSpannerRepository, *mockPublisher, *mockSopRepository, *mockSOPService, *mockProductAssessmentService, *mockProductAssessmentRepository) {
				validator := &mockValidator{}
				validator.On("ValidateRequest", mock.Anything).Return(&cerrors.ValidationResponse{Success: false, Error: cerrors.NewValidationError(cerrors.ErrBadRequest, "validation failed", nil)}, nil)
				return validator, &mockRedisRepository{}, &mockSpannerRepository{}, &mockPublisher{}, &mockSopRepository{}, &mockSOPService{}, &mockProductAssessmentService{}, &mockProductAssessmentRepository{}
			},
			wantErr:     false,
			wantSuccess: false,
			wantCode:    cerrors.ErrBadRequest,
			wantMessage: "validation failed",
		},
		{
			name: "validation internal error",
			request: &creditEnquiryProto.CreditEnquiryRequest{
				RequestId: validUUIDBytes,
			},
			setupMocks: func() (*mockValidator, *mockRedisRepository, *mockSpannerRepository, *mockPublisher, *mockSopRepository, *mockSOPService, *mockProductAssessmentService, *mockProductAssessmentRepository) {
				validator := &mockValidator{}
				validator.On("ValidateRequest", mock.Anything).Return((*cerrors.ValidationResponse)(nil), errors.New("internal error"))
				return validator, &mockRedisRepository{}, &mockSpannerRepository{}, &mockPublisher{}, &mockSopRepository{}, &mockSOPService{}, &mockProductAssessmentService{}, &mockProductAssessmentRepository{}
			},
			wantErr:     false,
			wantSuccess: false,
			wantCode:    cerrors.ErrInternalError,
			wantMessage: "Internal validation error",
		},
		{
			name: "redis storage failure",
			request: &creditEnquiryProto.CreditEnquiryRequest{
				RequestId: validUUIDBytes,
			},
			setupMocks: func() (*mockValidator, *mockRedisRepository, *mockSpannerRepository, *mockPublisher, *mockSopRepository, *mockSOPService, *mockProductAssessmentService, *mockProductAssessmentRepository) {
				validator := &mockValidator{}
				validator.On("ValidateRequest", mock.Anything).Return(&cerrors.ValidationResponse{Success: true}, nil)
				redisRepo := &mockRedisRepository{}
				redisRepo.On("SaveCreditEnquiry", mock.Anything, mock.Anything).Return(errors.New("redis error"))
				return validator, redisRepo, &mockSpannerRepository{}, &mockPublisher{}, &mockSopRepository{}, &mockSOPService{}, &mockProductAssessmentService{}, &mockProductAssessmentRepository{}
			},
			wantErr:     false,
			wantSuccess: false,
			wantCode:    cerrors.ErrStorageError,
			wantMessage: "Failed to save request to cache",
		},
		{
			name: "spanner storage failure",
			request: &creditEnquiryProto.CreditEnquiryRequest{
				RequestId: validUUIDBytes,
			},
			setupMocks: func() (*mockValidator, *mockRedisRepository, *mockSpannerRepository, *mockPublisher, *mockSopRepository, *mockSOPService, *mockProductAssessmentService, *mockProductAssessmentRepository) {
				validator := &mockValidator{}
				validator.On("ValidateRequest", mock.Anything).Return(&cerrors.ValidationResponse{Success: true}, nil)
				redisRepo := &mockRedisRepository{}
				redisRepo.On("SaveCreditEnquiry", mock.Anything, mock.Anything).Return(nil)
				spannerRepo := &mockSpannerRepository{}
				spannerRepo.On("SaveCreditEnquiry", mock.Anything, mock.Anything).Return(errors.New("spanner error"))
				return validator, redisRepo, spannerRepo, &mockPublisher{}, &mockSopRepository{}, &mockSOPService{}, &mockProductAssessmentService{}, &mockProductAssessmentRepository{}
			},
			wantErr:     false,
			wantSuccess: false,
			wantCode:    cerrors.ErrStorageError,
			wantMessage: "Failed to save request to database",
		},
		{
			name: "publisher failure - request still succeeds",
			request: &creditEnquiryProto.CreditEnquiryRequest{
				RequestId: validUUIDBytes,
			},
			setupMocks: func() (*mockValidator, *mockRedisRepository, *mockSpannerRepository, *mockPublisher, *mockSopRepository, *mockSOPService, *mockProductAssessmentService, *mockProductAssessmentRepository) {
				validator := &mockValidator{}
				validator.On("ValidateRequest", mock.Anything).Return(&cerrors.ValidationResponse{Success: true}, nil)
				redisRepo := &mockRedisRepository{}
				redisRepo.On("SaveCreditEnquiry", mock.Anything, mock.Anything).Return(nil)
				spannerRepo := &mockSpannerRepository{}
				spannerRepo.On("SaveCreditEnquiry", mock.Anything, mock.Anything).Return(nil)
				publisher := &mockPublisher{}
				publisher.On("PublishCreditEnquiryEvent", mock.Anything, mock.Anything).Return(errors.New("publisher error"))
				sopRepo := &mockSopRepository{}
				sopRepo.On("SaveSop", mock.Anything, mock.Anything).Return(nil)
				sopService := &mockSOPService{}
				sopService.On("GetConsolidatedSOP", mock.Anything, mock.Anything).Return(&sopPb.SOPResponse{
					SopAssessmentId:            "test-sop-id",
					ServiceabilityAssessmentId: "test-serviceability-id",
				}, nil)
				productAssessmentService := &mockProductAssessmentService{}
				productAssessmentService.On("GetProductRate", mock.Anything, mock.Anything).Return(&productAssessmentPb.ProductRateResponse{
					InitialStructureIndexRate: 5,
				}, nil)
				productAssessmentRepo := &mockProductAssessmentRepository{}
				productAssessmentRepo.On("SaveProductAssessment", mock.Anything, mock.Anything).Return(nil)
				return validator, redisRepo, spannerRepo, publisher, sopRepo, sopService, productAssessmentService, productAssessmentRepo
			},
			wantErr:     false,
			wantSuccess: true,
			wantCode:    cerrors.ErrPublishError,
			wantMessage: "Credit enquiry request processed successfully, but event publishing failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, redisRepo, spannerRepo, publisher, sopRepo, sopService, productAssessmentService, productAssessmentRepo := tt.setupMocks()

			server := NewServer(
				logger,
				validator,
				redisRepo,
				spannerRepo,
				sopRepo,
				publisher,
				sopService,
				productAssessmentService,
				productAssessmentRepo,
			)

			response, err := server.ProcessCreditEnquiry(context.Background(), tt.request)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wantSuccess, response.Success)
			assert.Equal(t, tt.wantCode, response.Code)
			assert.Equal(t, tt.wantMessage, response.Message)

			// Verify all mock expectations
			validator.AssertExpectations(t)
			redisRepo.AssertExpectations(t)
			spannerRepo.AssertExpectations(t)
			publisher.AssertExpectations(t)
			sopRepo.AssertExpectations(t)
			sopService.AssertExpectations(t)
			productAssessmentService.AssertExpectations(t)
			productAssessmentRepo.AssertExpectations(t)
		})
	}
}

func TestNewServer(t *testing.T) {
	// Create a test logger
	logger := log.Default()

	// Create mock dependencies
	validator := new(mockValidator)
	redisRepo := new(mockRedisRepository)
	spannerRepo := new(mockSpannerRepository)
	publisher := new(mockPublisher)
	sopRepo := new(mockSopRepository)
	sopService := new(mockSOPService)
	productAssessmentService := new(mockProductAssessmentService)
	productAssessmentRepo := new(mockProductAssessmentRepository)

	// Create server
	server := NewServer(
		logger,
		validator,
		redisRepo,
		spannerRepo,
		sopRepo,
		publisher,
		sopService,
		productAssessmentService,
		productAssessmentRepo,
	)

	// Verify server fields
	assert.NotNil(t, server)
	assert.Equal(t, logger, server.logger)
	assert.Equal(t, validator, server.validator)
	assert.Equal(t, redisRepo, server.redisRepo)
	assert.Equal(t, spannerRepo, server.spannerRepo)
	assert.Equal(t, sopRepo, server.sopRepo)
	assert.Equal(t, publisher, server.publisher)
	assert.Equal(t, sopService, server.sopService)
	assert.Equal(t, productAssessmentService, server.productAssessmentService)
	assert.Equal(t, productAssessmentRepo, server.productAssessmentRepo)
}

func TestStartServer(t *testing.T) {
	// Create mock dependencies
	validator := new(mockValidator)
	redisRepo := new(mockRedisRepository)
	spannerRepo := new(mockSpannerRepository)
	publisher := new(mockPublisher)
	sopRepo := new(mockSopRepository)
	sopService := new(mockSOPService)
	productAssessmentService := new(mockProductAssessmentService)
	productAssessmentRepo := new(mockProductAssessmentRepository)

	// Test invalid port
	err := StartServer(
		"invalid",
		validator,
		redisRepo,
		spannerRepo,
		sopRepo,
		publisher,
		sopService,
		productAssessmentService,
		productAssessmentRepo,
	)
	assert.Error(t, err)
}
