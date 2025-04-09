# Loan Service

A Go service that processes loan requests via gRPC, validates them, and stores them in Redis.

## Requirements

### Core Functionality
- Receives loan request messages in Protocol Buffers format via gRPC
- Validates loan request data
- Stores validated requests in Redis
- Returns processing results via gRPC response

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
- Redis integration for data persistence
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
│   │   │   └── request_cache.go # Request cache entity definition
│   │   ├── repository/
│   │   │   └── redis_repository.go # Redis data persistence
│   │   ├── server/
│   │   │   └── server.go       # gRPC server implementation
│   │   └── validator.go        # Business logic and validations
│   └── config/
│       └── config.go           # Configuration management
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

### Repository Layer (`internal/credit_enquiry/repository`)
- Redis connection management
- CRUD operations for loan requests
- Key management using UUID strings
- Schema-based storage with request metadata
- Request comparison using proto.Equal

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
- Testify (github.com/stretchr/testify)
- protoc-gen-validate (github.com/envoyproxy/protoc-gen-validate) 