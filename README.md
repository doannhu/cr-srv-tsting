# Loan Service

A Go service that processes loan requests via gRPC, validates them, stores them in Redis for caching, persists them in Google Cloud Spanner, and publishes events to Google Cloud Pub/Sub.

## Requirements

### Core Functionality
- Receives loan request messages in Protocol Buffers format via gRPC
- Validates loan request data
- Stores validated requests in Redis for caching
- Persists validated requests in Google Cloud Spanner
- Publishes credit enquiry events to Google Cloud Pub/Sub
- Returns processing results via gRPC response

### Infrastructure Requirements
- Google Cloud Project with Spanner and Pub/Sub enabled
- Spanner Instance and Database
- Pub/Sub Topic for credit enquiry events
- Redis instance for caching
- Service Account with appropriate permissions

## Infrastructure Setup (Terraform)

### Prerequisites
- Terraform installed
- Google Cloud SDK installed
- Appropriate Google Cloud permissions
- Service account with required roles

### Required Roles
- Spanner Admin
- Pub/Sub Publisher
- Service Account Admin
- IAM Admin

### Setup Steps
1. Initialize Terraform:
```bash
terraform init
```

2. Review the plan:
```bash
terraform plan
```

3. Apply the configuration:
```bash
terraform apply
```

### Terraform Configuration
The Terraform configuration creates:
1. **Service Accounts**
   - `credit-enquiry-sa`: Service account for the application
   - `spanner-sa`: Service account for Spanner access
   - `pubsub-sa`: Service account for Pub/Sub access

2. **IAM Bindings**
   - Spanner Admin role for the application service account
   - Pub/Sub Publisher role for the application service account
   - Required permissions for service account management

3. **Spanner Resources**
   - Instance configuration
   - Database setup
   - Table schema for credit enquiries

4. **Pub/Sub Resources**
   - Topic for credit enquiry events
   - Subscription for event processing
   - Required permissions

### Variables
Required variables in `variables.tf`:
- `project_id`: Google Cloud Project ID
- `region`: Region for resources
- `spanner_instance_name`: Name for Spanner instance
- `spanner_database_name`: Name for Spanner database
- `service_account_name`: Name for the service account

### Processing Flow
1. **Request Validation**
   - Validates UUID format and length
   - Validates request fields using protoc-gen-validate
   - Returns error response if validation fails

2. **Cache Storage**
   - Stores validated request in Redis
   - Enables quick access for duplicate detection
   - Returns error if cache storage fails

3. **Persistent Storage**
   - Asynchronously stores request in Google Cloud Spanner
   - Maintains historical record of all requests
   - Continues processing even if Spanner storage fails

4. **Event Publishing**
   - Publishes credit enquiry event to Pub/Sub topic
   - Includes request details and processing status
   - Continues processing even if event publishing fails
   - Logs any publishing errors for monitoring

### Validations
1. **UUID Validation**
   - Request ID must be a valid UUID v4
   - Stored as bytes in protocol buffer message
   - Validates length (16 bytes), format, and version (must be v4)
   - Uses protoc-gen-validate for proto-level validation

2. **String Length Validations**
   - Enquiry State: maximum 40 characters
   - Application Number: maximum 37 characters
   - Loan Purpose: maximum 50 characters

3. **Numeric Field Validations**
   - Loan Amount: must be positive and less than 999,999,999
   - Initial Structure Term: must be positive
   - Monthly Net Income: must be positive
   - Annual Gross Income: must be positive
   - Savings Amount: must be non-negative
   - Number of Continuing Home Loans: must be non-negative

4. **Duplicate Request Handling**
   - Checks if request ID exists in Redis
   - If exists, compares all fields of existing and new request using proto.Equal
   - Returns error if payloads differ (prevents request tampering)
   - Returns success if payloads match (idempotent operation)
   - Detailed logging of validation and comparison results

### Technical Requirements
- Comprehensive logging using standard log package
- Unit tests for service, repository, and validation layers
- Protocol buffer message definitions with validation rules
- Redis integration for data caching
- Google Cloud Spanner integration for persistent storage
- gRPC server implementation

## Project Structure

```
go-loan-service-v3/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── credit_enquiry/
│   │   ├── entity/
│   │   │   ├── request_cache.go # Request cache entity definition
│   │   │   └── credit_enquiry.go # Credit enquiry entity definition
│   │   ├── repository/
│   │   │   ├── redis_repository.go  # Redis data caching
│   │   │   └── spanner_repository.go # Spanner data persistence
│   │   ├── publisher/
│   │   │   └── pubsub_publisher.go  # Pub/Sub event publishing
│   │   ├── server/
│   │   │   └── server.go       # gRPC server implementation
│   │   └── validator.go        # Business logic and validations
│   └── config/
│       └── config.go           # Configuration management
├── terraform/
│   ├── main.tf                 # Main Terraform configuration
│   ├── variables.tf            # Terraform variables
│   └── outputs.tf              # Terraform outputs
├── proto/
│   └── credit_enquiry.proto    # Protocol buffer definitions
├── go.mod                      # Go module definition
└── README.md                   # Project documentation
```

## Components

### Protocol Buffers (`proto/`)
- `CreditEnquiryRequest` message definition
- UUID stored as bytes with validation rules
- Numeric fields as appropriate types (double, int64)
- gRPC service definition
- Event message definitions for Pub/Sub

### Repository Layer (`internal/credit_enquiry/repository`)
- Redis connection management for caching
- Spanner connection management for persistence
- CRUD operations for loan requests
- Key management using UUID strings
- Schema-based storage with request metadata
- Request comparison using proto.Equal

### Publisher Layer (`internal/credit_enquiry/publisher`)
- Pub/Sub client management
- Event publishing functionality
- Error handling and retry logic
- Message formatting and serialization

### Service Layer (`internal/credit_enquiry`)
- Business logic implementation
- Field validations (UUID, strings, numeric)
- Duplicate request detection with payload comparison
- Detailed error handling and logging
- Request comparison functionality

### Server Layer (`internal/credit_enquiry/server`)
- gRPC server implementation
- Request processing
- Error handling and response formatting
- Service initialization

### Main Application (`cmd/server`)
- Service initialization
- Configuration loading
- Dependency injection
- Server startup

## Testing

### Unit Tests
1. **UUID Validation Tests**
   - Valid UUID v4
   - Invalid UUID length
   - Invalid UUID format
   - Invalid UUID version

2. **String Length Tests**
   - Maximum length validation for all string fields
   - Error messages for exceeded lengths

3. **Numeric Field Tests**
   - Zero values
   - Negative values
   - Excessive values
   - Boundary conditions

4. **Duplicate Request Tests**
   - New unique requests
   - Duplicate requests with matching payloads
   - Duplicate requests with different payloads
   - Redis error handling

5. **Server Tests**
   - Request validation
   - Repository interaction
   - Error handling
   - Response formatting

### Test Coverage
   - Mock repository for Redis operations
   - Error case coverage
   - Logging verification
   - Comprehensive assertion checks

## Dependencies

- Redis (github.com/go-redis/redis/v8)
- Protocol Buffers (google.golang.org/protobuf)
- gRPC (google.golang.org/grpc)
- UUID (github.com/google/uuid)
- Google Cloud Spanner (cloud.google.com/go/spanner)
- Testify (github.com/stretchr/testify)
- protoc-gen-validate (github.com/envoyproxy/protoc-gen-validate)
- Terraform (for infrastructure setup)

## Docker Setup

### Prerequisites
- Docker installed
- Docker Compose installed
- Google Cloud credentials (key.json)

### Building the Docker Image
1. Using the build script:
```bash
./scripts/build-docker.sh [--registry=<registry>] [--tag=<tag>]
```

2. Using Docker directly:
```bash
docker build -t go-loan-service:latest .
```

### Running with Docker Compose
1. Set up environment variables:
```bash
export SPANNER_PROJECT="your-project-id"
export SPANNER_INSTANCE="your-instance"
export SPANNER_DATABASE="your-database"
export PUBSUB_PROJECT="your-project-id"
export PUBSUB_TOPIC="your-topic"
```

2. Start the services:
```bash
docker-compose up
```

### Environment Variables
- `REDIS_HOST`: Redis server host (default: localhost)
- `REDIS_PORT`: Redis server port (default: 6379)
- `SERVER_PORT`: gRPC server port (default: 50051)
- `SPANNER_PROJECT`: Google Cloud project ID
- `SPANNER_INSTANCE`: Spanner instance name
- `SPANNER_DATABASE`: Spanner database name
- `PUBSUB_PROJECT`: Google Cloud project ID for Pub/Sub
- `PUBSUB_TOPIC`: Pub/Sub topic name (default: credit-enquiry-events)

### Volumes
- Redis data is persisted in a Docker volume
- Google Cloud credentials are mounted from `key.json`

### Networks
- Services communicate through a bridge network
- Redis is accessible to the server at hostname `redis` 