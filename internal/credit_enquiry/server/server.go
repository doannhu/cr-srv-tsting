package server

import (
	"context"
	"log"
	"net"
	"time"

	"go-loan-service-v3/internal/credit_enquiry/entity"
	"go-loan-service-v3/internal/credit_enquiry/errors"
	"go-loan-service-v3/internal/credit_enquiry/interfaces"
	"go-loan-service-v3/internal/credit_enquiry/utils"
	pb "go-loan-service-v3/proto"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// CreditEnquiryServer implements the gRPC service for credit enquiries
type CreditEnquiryServer struct {
	pb.UnimplementedCreditEnquiryServiceServer
	logger      *log.Logger
	validator   interfaces.Validator
	redisRepo   interfaces.RequestCacheRepository
	spannerRepo interfaces.CreditEnquiryRepository
	sopRepo     interfaces.SopRepository
	publisher   interfaces.CreditEnquiryPublisher
}

// NewServer creates a new instance of the credit enquiry server
func NewServer(
	logger *log.Logger,
	validator interfaces.Validator,
	redisRepo interfaces.RequestCacheRepository,
	spannerRepo interfaces.CreditEnquiryRepository,
	sopRepo interfaces.SopRepository,
	publisher interfaces.CreditEnquiryPublisher,
) *CreditEnquiryServer {
	return &CreditEnquiryServer{
		logger:      logger,
		validator:   validator,
		redisRepo:   redisRepo,
		spannerRepo: spannerRepo,
		sopRepo:     sopRepo,
		publisher:   publisher,
	}
}

// ProcessCreditEnquiry handles incoming credit enquiry requests
func (s *CreditEnquiryServer) ProcessCreditEnquiry(ctx context.Context, req *pb.CreditEnquiryRequest) (*pb.CreditEnquiryResponse, error) {
	s.logger.Printf("Processing credit enquiry request: %+v", req)

	// Validate UUID
	if len(req.RequestId) == 0 {
		s.logger.Printf("Validation error: request_id is required")
		return &pb.CreditEnquiryResponse{
			Success: false,
			Message: "request_id is required",
			Code:    errors.ErrBadRequest,
		}, nil
	}
	if len(req.RequestId) != 16 {
		s.logger.Printf("Validation error: invalid UUID length")
		return &pb.CreditEnquiryResponse{
			Success: false,
			Message: "invalid UUID length",
			Code:    errors.ErrBadRequest,
		}, nil
	}
	if _, err := uuid.FromBytes(req.RequestId); err != nil {
		s.logger.Printf("Validation error: invalid UUID format")
		return &pb.CreditEnquiryResponse{
			Success: false,
			Message: "invalid UUID format",
			Code:    errors.ErrBadRequest,
		}, nil
	}

	// Validate the request
	validationResp, err := s.validator.ValidateRequest(req)
	if err != nil {
		s.logger.Printf("Validation error: %v", err)
		return &pb.CreditEnquiryResponse{
			Success: false,
			Message: "Internal validation error",
			Code:    errors.ErrInternalError,
		}, nil
	}

	if !validationResp.Success {
		s.logger.Printf("Validation failed: %+v", validationResp.Error)
		return &pb.CreditEnquiryResponse{
			Success: false,
			Message: validationResp.Error.Message,
			Code:    validationResp.Error.Code,
		}, nil
	}

	// Save the request to cache
	if err := s.redisRepo.SaveCreditEnquiry(ctx, req); err != nil {
		s.logger.Printf("Failed to save to Redis: %v", err)
		return &pb.CreditEnquiryResponse{
			Success: false,
			Message: "Failed to save request to cache",
			Code:    errors.ErrStorageError,
		}, nil
	}

	// Save the request to Spanner with retry
	spannerOperation := func() error {
		return s.spannerRepo.SaveCreditEnquiry(ctx, req)
	}

	// Configure retry for Spanner operations
	retryConfig := utils.DefaultRetryConfig()
	retryConfig.MaxAttempts = 3
	retryConfig.BaseDelay = 100 * time.Millisecond
	retryConfig.MaxDelay = 1 * time.Second

	if err := utils.Retry(ctx, retryConfig, "save credit enquiry to spanner", spannerOperation); err != nil {
		s.logger.Printf("Failed to save to Spanner: %v", err)
		return &pb.CreditEnquiryResponse{
			Success: false,
			Message: "Failed to save request to database",
			Code:    errors.ErrStorageError,
		}, nil
	}

	// Create and save SOP with retry
	sop := &entity.Sop{
		CreditEnquiryID:             string(req.RequestId),
		CreditEnquiryVersion:        "1.0",
		TotalMonthlyNetIncomeAmount: &req.TotalMonthlyNetIncomeAmount,
		TotalAnnualGrossIncome:      &req.TotalAnnualGrossIncome,
		TotalSavingsAmount:          &req.TotalSavingsAmount,
		TotalNumberOfContinuingHomeLoans: func() *int32 {
			val := int32(req.TotalNumberOfContinuingHomeLoans)
			return &val
		}(),
	}

	sopOperation := func() error {
		return s.sopRepo.SaveSop(ctx, sop)
	}

	if err := utils.Retry(ctx, retryConfig, "save sop to spanner", sopOperation); err != nil {
		s.logger.Printf("Failed to save SOP to Spanner: %v", err)
		return &pb.CreditEnquiryResponse{
			Success: false,
			Message: "Failed to save SOP to database",
			Code:    errors.ErrStorageError,
		}, nil
	}

	// Publish the event
	if err := s.publisher.PublishCreditEnquiryEvent(ctx, req); err != nil {
		s.logger.Printf("Failed to publish event: %v", err)
		return &pb.CreditEnquiryResponse{
			Success: true,
			Message: "Credit enquiry request processed successfully, but event publishing failed",
			Code:    errors.ErrPublishError,
		}, nil
	}

	return &pb.CreditEnquiryResponse{
		Success: true,
		Message: "Credit enquiry processed successfully",
		Code:    "",
	}, nil
}

// StartServer starts the gRPC server
func StartServer(port string, validator interfaces.Validator, redisRepo interfaces.RequestCacheRepository, spannerRepo interfaces.CreditEnquiryRepository, sopRepo interfaces.SopRepository, publisher interfaces.CreditEnquiryPublisher) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	logger := log.Default()
	grpcServer := grpc.NewServer()
	server := NewServer(logger, validator, redisRepo, spannerRepo, sopRepo, publisher)
	pb.RegisterCreditEnquiryServiceServer(grpcServer, server)

	log.Printf("Starting gRPC server on port %s", port)
	return grpcServer.Serve(lis)
}
