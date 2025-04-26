package server

import (
	"context"
	"log"
	"net"
	"time"

	"go-loan-service-v3/internal/credit_enquiry/entity"
	"go-loan-service-v3/internal/credit_enquiry/errors"
	"go-loan-service-v3/internal/credit_enquiry/interfaces"
	"go-loan-service-v3/internal/credit_enquiry/service/sop"
	"go-loan-service-v3/internal/credit_enquiry/utils"
	creditEnquiryProto "go-loan-service-v3/proto/credit_enquiry"
	productAssessmentPb "go-loan-service-v3/proto/product_assessment"
	sopPb "go-loan-service-v3/proto/sop"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// CreditEnquiryServer implements the gRPC service for credit enquiries
type CreditEnquiryServer struct {
	creditEnquiryProto.UnimplementedCreditEnquiryServiceServer
	logger                   *log.Logger
	validator                interfaces.Validator
	redisRepo                interfaces.RequestCacheRepository
	spannerRepo              interfaces.CreditEnquiryRepository
	sopRepo                  interfaces.SopRepository
	publisher                interfaces.CreditEnquiryPublisher
	sopService               sop.SOPService
	productAssessmentService interfaces.ProductAssessmentService
	productAssessmentRepo    interfaces.ProductAssessmentRepository
}

// NewServer creates a new instance of the credit enquiry server
func NewServer(
	logger *log.Logger,
	validator interfaces.Validator,
	redisRepo interfaces.RequestCacheRepository,
	spannerRepo interfaces.CreditEnquiryRepository,
	sopRepo interfaces.SopRepository,
	publisher interfaces.CreditEnquiryPublisher,
	sopService sop.SOPService,
	productAssessmentService interfaces.ProductAssessmentService,
	productAssessmentRepo interfaces.ProductAssessmentRepository,
) *CreditEnquiryServer {
	return &CreditEnquiryServer{
		logger:                   logger,
		validator:                validator,
		redisRepo:                redisRepo,
		spannerRepo:              spannerRepo,
		sopRepo:                  sopRepo,
		publisher:                publisher,
		sopService:               sopService,
		productAssessmentService: productAssessmentService,
		productAssessmentRepo:    productAssessmentRepo,
	}
}

// ProcessCreditEnquiry handles incoming credit enquiry requests
func (s *CreditEnquiryServer) ProcessCreditEnquiry(ctx context.Context, req *creditEnquiryProto.CreditEnquiryRequest) (*creditEnquiryProto.CreditEnquiryResponse, error) {
	s.logger.Printf("Processing credit enquiry request: %+v", req)

	// Validate UUID
	if len(req.RequestId) == 0 {
		s.logger.Printf("Validation error: request_id is required")
		return &creditEnquiryProto.CreditEnquiryResponse{
			Success: false,
			Message: "request_id is required",
			Code:    errors.ErrBadRequest,
		}, nil
	}
	if len(req.RequestId) != 16 {
		s.logger.Printf("Validation error: invalid UUID length")
		return &creditEnquiryProto.CreditEnquiryResponse{
			Success: false,
			Message: "invalid UUID length",
			Code:    errors.ErrBadRequest,
		}, nil
	}
	if _, err := uuid.FromBytes(req.RequestId); err != nil {
		s.logger.Printf("Validation error: invalid UUID format")
		return &creditEnquiryProto.CreditEnquiryResponse{
			Success: false,
			Message: "invalid UUID format",
			Code:    errors.ErrBadRequest,
		}, nil
	}

	// Validate the request
	validationResp, err := s.validator.ValidateRequest(req)
	if err != nil {
		s.logger.Printf("Validation error: %v", err)
		return &creditEnquiryProto.CreditEnquiryResponse{
			Success: false,
			Message: "Internal validation error",
			Code:    errors.ErrInternalError,
		}, nil
	}

	if !validationResp.Success {
		s.logger.Printf("Validation failed: %+v", validationResp.Error)
		return &creditEnquiryProto.CreditEnquiryResponse{
			Success: false,
			Message: validationResp.Error.Message,
			Code:    validationResp.Error.Code,
		}, nil
	}

	// Save the request to cache
	if err := s.redisRepo.SaveCreditEnquiry(ctx, req); err != nil {
		s.logger.Printf("Failed to save to Redis: %v", err)
		return &creditEnquiryProto.CreditEnquiryResponse{
			Success: false,
			Message: "Failed to save request to cache",
			Code:    errors.ErrStorageError,
		}, nil
	}

	// Configure retry for Spanner operations
	retryConfig := utils.DefaultRetryConfig()
	retryConfig.MaxAttempts = 3
	retryConfig.BaseDelay = 100 * time.Millisecond
	retryConfig.MaxDelay = 1 * time.Second

	// Save the request to Spanner with retry
	spannerOperation := func() error {
		return s.spannerRepo.SaveCreditEnquiry(ctx, req)
	}

	if err := utils.Retry(ctx, retryConfig, "save credit enquiry to spanner", spannerOperation); err != nil {
		s.logger.Printf("Failed to save to Spanner: %v", err)
		return &creditEnquiryProto.CreditEnquiryResponse{
			Success: false,
			Message: "Failed to save request to database",
			Code:    errors.ErrStorageError,
		}, nil
	}

	// Call SOP service to get consolidated SOP data
	sopRequest := &sopPb.SOPRequest{
		CreditEnquiryId:      string(req.RequestId),
		CreditEnquiryVersion: "1.0",
	}

	sopResponse, err := s.sopService.GetConsolidatedSOP(ctx, sopRequest)
	if err != nil {
		s.logger.Printf("Failed to get SOP data: %v", err)
		return &creditEnquiryProto.CreditEnquiryResponse{
			Success: false,
			Message: "Failed to get SOP data",
			Code:    errors.ErrServiceError,
		}, nil
	}

	// Create and save SOP with retry
	sopData := &entity.Sop{
		CreditEnquiryID:                  string(req.RequestId),
		CreditEnquiryVersion:             "1.0",
		SopAssessmentID:                  sopResponse.SopAssessmentId,
		ServiceabilityAssessmentID:       sopResponse.ServiceabilityAssessmentId,
		TotalMonthlyNetIncomeAmount:      &sopResponse.TotalMonthlyNetIncomeAmount,
		TotalAnnualGrossIncome:           &sopResponse.TotalAnnualGrossIncome,
		TotalSavingsAmount:               &sopResponse.TotalSavingsAmount,
		TotalNumberOfContinuingHomeLoans: &sopResponse.TotalNumberOfContinuingHomeLoans,
	}

	sopOperation := func() error {
		return s.sopRepo.SaveSop(ctx, sopData)
	}

	if err := utils.Retry(ctx, retryConfig, "save sop to spanner", sopOperation); err != nil {
		s.logger.Printf("Failed to save SOP to Spanner: %v", err)
		return &creditEnquiryProto.CreditEnquiryResponse{
			Success: false,
			Message: "Failed to save SOP to database",
			Code:    errors.ErrStorageError,
		}, nil
	}

	// Create and save product assessment
	productRateRequest := &productAssessmentPb.ProductRateRequest{
		ProductCode: req.ProductCode,
		ProductName: req.ProductName,
	}

	productRateResponse, err := s.productAssessmentService.GetProductRate(ctx, productRateRequest)
	if err != nil {
		s.logger.Printf("Failed to get product rate: %v", err)
		return &creditEnquiryProto.CreditEnquiryResponse{
			Success: false,
			Message: "Failed to get product rate",
			Code:    errors.ErrServiceError,
		}, nil
	}

	productAssessment := &entity.ProductAssessment{
		ProductAssessmentID:        uuid.New().String(),
		ServiceabilityAssessmentID: sopResponse.ServiceabilityAssessmentId,
		CreditEnquiryID:            string(req.RequestId),
		CreditEnquiryVersion:       "1.0",
		ProductCode:                req.ProductCode,
		ProductName:                req.ProductName,
		LoanAmount:                 req.LoanAmount,
		LoanPurpose:                req.LoanPurpose,
		InitialStructureTermMonth:  req.InitialStructureTermMonth,
		InitialStructureIndexRate:  productRateResponse.InitialStructureIndexRate,
	}

	if err := s.productAssessmentRepo.SaveProductAssessment(ctx, productAssessment); err != nil {
		s.logger.Printf("Failed to save product assessment: %v", err)
		return &creditEnquiryProto.CreditEnquiryResponse{
			Success: false,
			Message: "Failed to save product assessment",
			Code:    errors.ErrStorageError,
		}, nil
	}

	// Publish the event
	if err := s.publisher.PublishCreditEnquiryEvent(ctx, req); err != nil {
		s.logger.Printf("Failed to publish event: %v", err)
		return &creditEnquiryProto.CreditEnquiryResponse{
			Success: true,
			Message: "Credit enquiry request processed successfully, but event publishing failed",
			Code:    errors.ErrPublishError,
		}, nil
	}

	return &creditEnquiryProto.CreditEnquiryResponse{
		Success: true,
		Message: "Credit enquiry processed successfully",
		Code:    "",
	}, nil
}

// StartServer starts the gRPC server
func StartServer(
	port string,
	validator interfaces.Validator,
	redisRepo interfaces.RequestCacheRepository,
	spannerRepo interfaces.CreditEnquiryRepository,
	sopRepo interfaces.SopRepository,
	publisher interfaces.CreditEnquiryPublisher,
	sopService sop.SOPService,
	productAssessmentService interfaces.ProductAssessmentService,
	productAssessmentRepo interfaces.ProductAssessmentRepository,
) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	logger := log.Default()
	grpcServer := grpc.NewServer()
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
	creditEnquiryProto.RegisterCreditEnquiryServiceServer(grpcServer, server)

	log.Printf("Starting gRPC server on port %s", port)
	return grpcServer.Serve(lis)
}
