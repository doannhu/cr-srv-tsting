package main

import (
	"log"
	"os"

	"go-loan-service-v3/internal/credit_enquiry"
	"go-loan-service-v3/internal/credit_enquiry/repository"
	"go-loan-service-v3/internal/credit_enquiry/server"

	"github.com/go-redis/redis/v8"
)

func main() {
	// Get Redis connection details from environment variables
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	serverPort := getEnv("SERVER_PORT", "50051")

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisHost + ":" + redisPort,
	})

	// Initialize validator and repository
	validator := credit_enquiry.NewValidator(log.New(os.Stdout, "", log.LstdFlags))
	repo := repository.NewRedisRepository(redisClient)

	// Start the gRPC server
	if err := server.StartServer(serverPort, validator, repo); err != nil {
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
