package main

import (
	"context"
	"log"
	"os"

	"go-loan-service-v3/internal/credit_enquiry"
	"go-loan-service-v3/internal/credit_enquiry/repository"
	"go-loan-service-v3/internal/credit_enquiry/server"

	"cloud.google.com/go/spanner"

	"github.com/go-redis/redis/v8"
)

func main() {
	// Get environment variables
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	serverPort := getEnv("SERVER_PORT", "50051")
	spannerProject := getEnv("SPANNER_PROJECT", "")
	spannerInstance := getEnv("SPANNER_INSTANCE", "")
	spannerDatabase := getEnv("SPANNER_DATABASE", "")

	if spannerProject == "" || spannerInstance == "" || spannerDatabase == "" {
		log.Fatal("Spanner configuration is required. Please set SPANNER_PROJECT, SPANNER_INSTANCE, and SPANNER_DATABASE environment variables")
	}

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisHost + ":" + redisPort,
	})

	// Initialize Spanner client
	spannerClient, err := spanner.NewClient(context.Background(),
		"projects/"+spannerProject+"/instances/"+spannerInstance+"/databases/"+spannerDatabase)
	if err != nil {
		log.Fatalf("Failed to create Spanner client: %v", err)
	}
	defer spannerClient.Close()

	// Initialize validator and repositories
	validator := credit_enquiry.NewValidator(log.New(os.Stdout, "", log.LstdFlags))
	redisRepo := repository.NewRedisRepository(redisClient)
	spannerRepo := repository.NewSpannerRepository(spannerClient)

	// Start the gRPC server
	if err := server.StartServer(serverPort, validator, redisRepo, spannerRepo); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
