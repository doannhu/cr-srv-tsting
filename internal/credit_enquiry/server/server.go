package server

import (
	"context"
	"log"
	"net"

	"go-loan-service-v3/internal/credit_enquiry"
	"go-loan-service-v3/internal/credit_enquiry/entity"
	pb "go-loan-service-v3/proto"

	"google.golang.org/grpc"
)

// CreditEnquiryServer implements the gRPC service for credit enquiries
type CreditEnquiryServer struct {
	pb.UnimplementedCreditEnquiryServiceServer
	validator credit_enquiry.Validator
	repo      entity.RequestCacheRepository
}

// NewCreditEnquiryServer creates a new instance of the credit enquiry server
func NewCreditEnquiryServer(validator credit_enquiry.Validator, repo entity.RequestCacheRepository) *CreditEnquiryServer {
	return &CreditEnquiryServer{
		validator: validator,
		repo:      repo,
	}
}

// ProcessCreditEnquiry handles incoming credit enquiry requests
func (s *CreditEnquiryServer) ProcessCreditEnquiry(ctx context.Context, req *pb.CreditEnquiryRequest) (*pb.CreditEnquiryResponse, error) {
	// Validate the request
	if err := s.validator.ValidateRequest(req); err != nil {
		return &pb.CreditEnquiryResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// Save the request to cache
	if err := s.repo.SaveRequest(req); err != nil {
		return &pb.CreditEnquiryResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.CreditEnquiryResponse{
		Success: true,
		Message: "Credit enquiry request processed successfully",
	}, nil
}

// StartServer starts the gRPC server
func StartServer(port string, validator credit_enquiry.Validator, repo entity.RequestCacheRepository) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	server := NewCreditEnquiryServer(validator, repo)
	pb.RegisterCreditEnquiryServiceServer(grpcServer, server)

	log.Printf("Starting gRPC server on port %s", port)
	return grpcServer.Serve(lis)
}
