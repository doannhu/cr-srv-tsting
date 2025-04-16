package server

import (
	"context"
	"log"
	"net"

	"go-loan-service-v3/internal/credit_enquiry"
	"go-loan-service-v3/internal/credit_enquiry/interfaces"
	pb "go-loan-service-v3/proto"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// CreditEnquiryServer implements the gRPC service for credit enquiries
type CreditEnquiryServer struct {
	pb.UnimplementedCreditEnquiryServiceServer
	validator   credit_enquiry.Validator
	redisRepo   interfaces.RequestCacheRepository
	spannerRepo interfaces.CreditEnquiryRepository
	publisher   interfaces.CreditEnquiryPublisher
}

// NewCreditEnquiryServer creates a new instance of the credit enquiry server
func NewCreditEnquiryServer(validator credit_enquiry.Validator, redisRepo interfaces.RequestCacheRepository, spannerRepo interfaces.CreditEnquiryRepository, publisher interfaces.CreditEnquiryPublisher) *CreditEnquiryServer {
	return &CreditEnquiryServer{
		validator:   validator,
		redisRepo:   redisRepo,
		spannerRepo: spannerRepo,
		publisher:   publisher,
	}
}

// ProcessCreditEnquiry handles incoming credit enquiry requests
func (s *CreditEnquiryServer) ProcessCreditEnquiry(ctx context.Context, req *pb.CreditEnquiryRequest) (*pb.CreditEnquiryResponse, error) {
	// Validate UUID
	if len(req.RequestId) == 0 {
		return &pb.CreditEnquiryResponse{
			Success: false,
			Message: "request_id is required",
		}, nil
	}
	if len(req.RequestId) != 16 {
		return &pb.CreditEnquiryResponse{
			Success: false,
			Message: "invalid UUID length",
		}, nil
	}
	if _, err := uuid.FromBytes(req.RequestId); err != nil {
		return &pb.CreditEnquiryResponse{
			Success: false,
			Message: "invalid UUID format",
		}, nil
	}

	// Validate the request
	if err := s.validator.ValidateRequest(req); err != nil {
		return &pb.CreditEnquiryResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// Save the request to cache
	if err := s.redisRepo.SaveRequest(req); err != nil {
		return &pb.CreditEnquiryResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// Save the request to Spanner
	if err := s.spannerRepo.SaveCreditEnquiry(ctx, req); err != nil {
		return &pb.CreditEnquiryResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// Publish event to Pub/Sub
	if err := s.publisher.PublishCreditEnquiryEvent(ctx, req); err != nil {
		// Log the error but don't fail the request since data is already saved
		log.Printf("Failed to publish event: %v", err)
	}

	return &pb.CreditEnquiryResponse{
		Success: true,
		Message: "Credit enquiry request processed successfully",
	}, nil
}

// StartServer starts the gRPC server
func StartServer(port string, validator credit_enquiry.Validator, redisRepo interfaces.RequestCacheRepository, spannerRepo interfaces.CreditEnquiryRepository, publisher interfaces.CreditEnquiryPublisher) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	server := NewCreditEnquiryServer(validator, redisRepo, spannerRepo, publisher)
	pb.RegisterCreditEnquiryServiceServer(grpcServer, server)

	log.Printf("Starting gRPC server on port %s", port)
	return grpcServer.Serve(lis)
}
