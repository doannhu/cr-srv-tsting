package sop

import (
	"context"
	"fmt"

	tlsSecurityConfig "go-loan-service-v3/internal/config/security"
	"go-loan-service-v3/internal/credit_enquiry/interfaces"
	"go-loan-service-v3/internal/credit_enquiry/utils"
	pb "go-loan-service-v3/proto/sop"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

// sopServiceClient implements the SOPService interface
type sopServiceClient struct {
	client      pb.SOPServiceClient
	retryConfig *utils.RetryConfig
}

// NewSOPClient creates a new SOP service client with connection management
func NewSOPClient(ctx context.Context, addr string, tlsConfig *tlsSecurityConfig.TLSConfig, retryConfig *utils.RetryConfig) (interfaces.SOPService, error) {
	var creds credentials.TransportCredentials
	var err error

	if tlsConfig != nil {
		tls, err := tlsSecurityConfig.NewTLSConfig(tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS config: %v", err)
		}
		creds = credentials.NewTLS(tls)
	} else {
		return nil, fmt.Errorf("TLS configuration is required")
	}

	// Create a timeout context for the connection
	// ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	// defer cancel()

	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SOP service: %v", err)
	}

	// Verify the connection state
	// if conn.GetState() != connectivity.Ready {
	// 	conn.Close()
	// 	return nil, fmt.Errorf("connection not ready: %v", conn.GetState())
	// }

	return &sopServiceClient{
		client:      pb.NewSOPServiceClient(conn),
		retryConfig: retryConfig,
	}, nil
}

// NewSOPServiceClient creates a new SOP service client from an existing connection
func NewSOPServiceClient(conn *grpc.ClientConn, retryConfig *utils.RetryConfig) interfaces.SOPService {
	return &sopServiceClient{
		client:      pb.NewSOPServiceClient(conn),
		retryConfig: retryConfig,
	}
}

// GetConsolidatedSOP retrieves consolidated SOP data with retry logic
func (s *sopServiceClient) GetConsolidatedSOP(ctx context.Context, request *pb.SOPRequest) (*pb.SOPResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	var response *pb.SOPResponse
	operation := func() error {
		var err error
		response, err = s.client.GetConsolidatedSOP(ctx, request)
		return err
	}

	if err := utils.Retry(ctx, s.retryConfig, "get consolidated SOP", operation); err != nil {
		return nil, err
	}

	return response, nil
}
