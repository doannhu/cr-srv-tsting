#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Loan Service Setup Script ===${NC}"

# Check if gcloud is installed
if ! command -v gcloud &> /dev/null; then
    echo -e "${RED}Error: gcloud CLI is not installed.${NC}"
    echo "Please install Google Cloud SDK from: https://cloud.google.com/sdk/docs/install"
    exit 1
fi

# Check if user is authenticated with gcloud
if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" &> /dev/null; then
    echo -e "${YELLOW}You need to authenticate with Google Cloud.${NC}"
    echo "Running: gcloud auth application-default login"
    gcloud auth application-default login
fi

# Get project ID
echo -e "${GREEN}Enter your Google Cloud Project ID:${NC}"
read -r SPANNER_PROJECT

# Get Spanner instance ID
echo -e "${GREEN}Enter your Spanner Instance ID:${NC}"
read -r SPANNER_INSTANCE

# Get Spanner database ID
echo -e "${GREEN}Enter your Spanner Database ID:${NC}"
read -r SPANNER_DATABASE

# Set environment variables
echo -e "${GREEN}Setting environment variables...${NC}"
export SPANNER_PROJECT="$SPANNER_PROJECT"
export SPANNER_INSTANCE="$SPANNER_INSTANCE"
export SPANNER_DATABASE="$SPANNER_DATABASE"
export REDIS_HOST="localhost"
export REDIS_PORT="6379"
export SERVER_PORT="50051"

# Verify Spanner connection
echo -e "${GREEN}Verifying Spanner connection...${NC}"

# First, check if the instance exists
echo -e "${YELLOW}Checking Spanner instance...${NC}"
if ! gcloud spanner instances describe "$SPANNER_INSTANCE" --project="$SPANNER_PROJECT" &> /dev/null; then
    echo -e "${RED}Error: Spanner instance '$SPANNER_INSTANCE' not found in project '$SPANNER_PROJECT'${NC}"
    echo "Please verify:"
    echo "1. The instance name is correct"
    echo "2. You have access to the project"
    echo "3. The instance exists in the specified region"
    echo "You can list available instances with:"
    echo "  gcloud spanner instances list --project=$SPANNER_PROJECT"
    exit 1
fi

# Then, check if the database exists
echo -e "${YELLOW}Checking Spanner database...${NC}"
if ! gcloud spanner databases describe "$SPANNER_DATABASE" \
    --instance="$SPANNER_INSTANCE" \
    --project="$SPANNER_PROJECT" &> /dev/null; then
    echo -e "${RED}Error: Database '$SPANNER_DATABASE' not found in instance '$SPANNER_INSTANCE'${NC}"
    echo "Please verify:"
    echo "1. The database name is correct"
    echo "2. The database exists in the instance"
    echo "You can list available databases with:"
    echo "  gcloud spanner databases list --instance=$SPANNER_INSTANCE --project=$SPANNER_PROJECT"
    exit 1
fi

# Finally, try to execute a simple query
echo -e "${YELLOW}Testing database connection...${NC}"
if ! gcloud spanner databases execute-sql "$SPANNER_DATABASE" \
    --instance="$SPANNER_INSTANCE" \
    --project="$SPANNER_PROJECT" \
    --sql="SELECT 1" &> /dev/null; then
    echo -e "${RED}Error: Could not execute query on the database${NC}"
    echo "Please verify:"
    echo "1. Your authentication is valid"
    echo "2. You have the necessary permissions"
    echo "3. The database is accessible"
    echo "Try running:"
    echo "  gcloud auth application-default login"
    echo "  gcloud config set project $SPANNER_PROJECT"
    exit 1
fi

echo -e "${GREEN}Spanner connection verified successfully!${NC}"

# Check if Redis is running
echo -e "${GREEN}Checking Redis connection...${NC}"
if ! redis-cli ping &> /dev/null; then
    echo -e "${YELLOW}Redis is not running. Starting Redis...${NC}"
    if ! command -v redis-server &> /dev/null; then
        echo -e "${RED}Error: Redis is not installed.${NC}"
        echo "Please install Redis using:"
        echo "  - macOS: brew install redis"
        echo "  - Ubuntu: sudo apt-get install redis-server"
        exit 1
    fi
    redis-server --daemonize yes
fi

# Build the service
echo -e "${GREEN}Building the service...${NC}"
go build -o loan-service cmd/server/main.go

# Start the service
echo -e "${GREEN}Starting the service...${NC}"
./loan-service

# Instructions for next steps
echo -e "\n${GREEN}=== Next Steps ===${NC}"
echo "1. The service is now running on port 50051"
echo "2. You can test the service using a gRPC client"
echo "3. To stop the service, press Ctrl+C"
echo "4. To run tests: go test ./..."
echo "5. To check logs: tail -f /var/log/loan-service.log"

# Cleanup function
cleanup() {
    echo -e "\n${GREEN}Cleaning up...${NC}"
    pkill -f loan-service
    redis-cli shutdown
    echo -e "${GREEN}Done!${NC}"
}

# Set up trap for cleanup
trap cleanup EXIT 