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
	"google.golang.org/grpc/credentials/insecure"

	"go-loan-service-v3/internal/credit_enquiry/repository"
	sopRepo "go-loan-service-v3/internal/credit_enquiry/repository/sop"
	"go-loan-service-v3/internal/credit_enquiry/server"
	sopService "go-loan-service-v3/internal/credit_enquiry/service/sop"
	creditUtils "go-loan-service-v3/internal/credit_enquiry/utils"
	pb "go-loan-service-v3/proto"

	"cloud.google.com/go/spanner"
)

func main() {
	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	// Initialize Redis repository
	redisRepo := repository.NewRedisRepository(redisClient)

	// Initialize Spanner client
	spannerClient, err := spanner.NewClient(context.Background(),
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
	spannerRepo := repository.NewSpannerRepository(spannerClient)

	// Initialize SOP repository
	sopRepository := sopRepo.NewSpannerSopRepository(spannerClient)

	// Initialize SOP service client
	sopConn, err := grpc.Dial(
		os.Getenv("SOP_SERVICE_ADDR"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to connect to SOP service: %v", err)
	}
	defer sopConn.Close()

	retryConfig := &creditUtils.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
	}

	sopSvc := sopService.NewSOPServiceClient(sopConn, retryConfig)

	// Start gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", os.Getenv("PORT")))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	creditEnquiryServer := server.NewServer(
		log.Default(),
		nil, // validator
		redisRepo,
		spannerRepo,
		sopRepository,
		nil, // publisher
		sopSvc,
	)

	pb.RegisterCreditEnquiryServiceServer(s, creditEnquiryServer)
	log.Printf("Starting gRPC server on port %s", os.Getenv("PORT"))
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
