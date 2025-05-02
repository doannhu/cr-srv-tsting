package sop

import (
	"context"
	"fmt"

	tlsSecurityConfig "go-loan-service-v3/internal/config/security"
	"go-loan-service-v3/internal/credit_enquiry/interfaces"
	"go-loan-service-v3/internal/credit_enquiry/utils"
	pb "go-loan-service-v3/proto/sop"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type StandardService struct {
	*BaseService
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

	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SOP service: %v", err)
	}

	client := pb.NewSOPServiceClient(conn)
	base := NewBaseService(retryConfig, client)
	return &StandardService{BaseService: base}, nil
}

// NewSOPServiceClient creates a new SOP service client from an existing connection
func NewSOPServiceClient(conn *grpc.ClientConn, retryConfig *utils.RetryConfig) interfaces.SOPService {
	client := pb.NewSOPServiceClient(conn)
	base := NewBaseService(retryConfig, client)
	return &StandardService{BaseService: base}
}
