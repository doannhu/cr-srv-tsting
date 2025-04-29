package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"google.golang.org/grpc"

	creditEnquiryRepo "go-loan-service-v3/internal/credit_enquiry/repository/credit_enquiry"
	redisRepo "go-loan-service-v3/internal/credit_enquiry/repository/request_cache"
	sopRepo "go-loan-service-v3/internal/credit_enquiry/repository/sop"
	"go-loan-service-v3/internal/credit_enquiry/server"
	sopService "go-loan-service-v3/internal/credit_enquiry/service/sop"
	creditUtils "go-loan-service-v3/internal/credit_enquiry/utils"
	creditEnquiryProto "go-loan-service-v3/proto/credit_enquiry"

	tlsSecurityConfig "go-loan-service-v3/internal/config/security"

	productAssessmentRepo "go-loan-service-v3/internal/credit_enquiry/repository/product_assessment"
	productAssessmentService "go-loan-service-v3/internal/credit_enquiry/service/product_assessment"

	"cloud.google.com/go/spanner"
)

func main() {
	ctx := context.Background()

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	// Initialize Redis repository
	redisRepo := redisRepo.NewRedisRepository(redisClient)

	// Initialize Spanner client
	spannerClient, err := spanner.NewClient(ctx,
		fmt.Sprintf("projects/%s/instances/%s/databases/%s",
			os.Getenv("SPANNER_PROJECT_ID"),
			os.Getenv("SPANNER_INSTANCE_ID"),
			os.Getenv("SPANNER_DATABASE_ID"),
		),
	)
	if err != nil {
		log.Fatalf("Failed to create Spanner client: %v", err)
	}
	defer spannerClient.Close()

	// Initialize Spanner repository
	spannerRepo := creditEnquiryRepo.NewSpannerRepository(spannerClient)

	// Initialize SOP repository
	sopRepository := sopRepo.NewSpannerSopRepository(spannerClient)

	// Initialize SOP service client
	retryConfig := &creditUtils.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
	}

	// Initialize Product Assessment repository
	productAssessmentRepository := productAssessmentRepo.NewSpannerProductAssessmentRepository(spannerClient)

	// Create a gRPC connection to the product assessment service
	productAssessmentConn, err := grpc.Dial(
		os.Getenv("PRODUCT_ASSESSMENT_SERVICE_ADDR"),
		grpc.WithInsecure(), // Use TLS in production!
	)
	if err != nil {
		log.Fatalf("Failed to connect to Product Assessment service: %v", err)
	}
	defer productAssessmentConn.Close()

	// Initialize Product Assessment service
	productAssessmentSvc := productAssessmentService.NewProductAssessmentService(retryConfig, productAssessmentConn)

	tlsConfig := &tlsSecurityConfig.TLSConfig{
		CertFile:   os.Getenv("SOP_CLIENT_CERT_FILE"),
		KeyFile:    os.Getenv("SOP_CLIENT_KEY_FILE"),
		CAFile:     os.Getenv("SOP_CA_FILE"),
		ServerName: os.Getenv("SOP_SERVER_NAME"),
	}

	sopSvc, err := sopService.NewSOPClient(ctx, os.Getenv("SOP_SERVICE_ADDR"), tlsConfig, retryConfig)
	if err != nil {
		log.Fatalf("Failed to create SOP service client: %v", err)
	}

	// Start gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", os.Getenv("PORT")))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	creditEnquiryServer := server.NewServer(
		log.Default(),
		nil, // TODO: implement validator
		redisRepo,
		spannerRepo,
		sopRepository,
		nil, // TODO: implement publisher
		sopSvc,
		productAssessmentSvc,
		productAssessmentRepository,
	)

	creditEnquiryProto.RegisterCreditEnquiryServiceServer(s, creditEnquiryServer)
	log.Printf("Starting gRPC server on port %s", os.Getenv("PORT"))
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
