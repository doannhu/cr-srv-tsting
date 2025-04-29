package product_assessment

import (
	"context"
	"fmt"
	"time"

	tlsSecurityConfig "go-loan-service-v3/internal/config/security"
	"go-loan-service-v3/internal/credit_enquiry/interfaces"
	"go-loan-service-v3/internal/credit_enquiry/utils"
	pb "go-loan-service-v3/proto/product_assessment"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
)

const (
	defaultProductCode = "HLV"
)

// productAssessmentService implements interfaces.ProductAssessmentService
type productAssessmentService struct {
	retryConfig *utils.RetryConfig
	client      pb.ProductAssessmentServiceClient
}

// NewProductAssessmentClient creates a new ProductAssessmentService client with TLS
func NewProductAssessmentClient(ctx context.Context, addr string, tlsConfig *tlsSecurityConfig.TLSConfig, retryConfig *utils.RetryConfig) (interfaces.ProductAssessmentService, error) {
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

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(creds),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Product Assessment service: %v", err)
	}

	if conn.GetState() != connectivity.Ready {
		conn.Close()
		return nil, fmt.Errorf("connection not ready: %v", conn.GetState())
	}

	return &productAssessmentService{
		retryConfig: retryConfig,
		client:      pb.NewProductAssessmentServiceClient(conn),
	}, nil
}

// NewProductAssessmentService creates a new instance of interfaces.ProductAssessmentService from an existing connection
func NewProductAssessmentService(retryConfig *utils.RetryConfig, conn *grpc.ClientConn) interfaces.ProductAssessmentService {
	return &productAssessmentService{
		retryConfig: retryConfig,
		client:      pb.NewProductAssessmentServiceClient(conn),
	}
}

// GetProductRate implements interfaces.ProductAssessmentService
func (s *productAssessmentService) GetProductRate(ctx context.Context, request *pb.ProductRateRequest) (*pb.ProductRateResponse, error) {
	if request == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	// Use default product code if empty
	if request.ProductCode == "" {
		request.ProductCode = defaultProductCode
	}

	var response *pb.ProductRateResponse
	var err error

	operation := func() error {
		response, err = s.client.GetProductRate(ctx, request)
		return err
	}

	if err := utils.Retry(ctx, s.retryConfig, "get product rate", operation); err != nil {
		return nil, fmt.Errorf("failed to get product rate: %w", err)
	}

	return response, nil
}
