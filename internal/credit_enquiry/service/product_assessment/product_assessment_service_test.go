package product_assessment

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-loan-service-v3/internal/credit_enquiry/utils"
	pb "go-loan-service-v3/proto/product_assessment"

	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	tlsSecurityConfig "go-loan-service-v3/internal/config/security"
	"math/big"
	"net"
	"os"
	"path/filepath"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// MockProductAssessmentServiceClient is a mock implementation of pb.ProductAssessmentServiceClient
type MockProductAssessmentServiceClient struct {
	mock.Mock
}

func (m *MockProductAssessmentServiceClient) GetProductRate(ctx context.Context, request *pb.ProductRateRequest, opts ...grpc.CallOption) (*pb.ProductRateResponse, error) {
	args := m.Called(ctx, request, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.ProductRateResponse), args.Error(1)
}

func TestProductAssessmentService_GetProductRate(t *testing.T) {
	retryConfig := &utils.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    time.Millisecond * 10,
	}

	ctx := context.Background()
	request := &pb.ProductRateRequest{
		ProductCode: "PROD001",
		ProductName: "Test Product",
	}

	t.Run("successful get", func(t *testing.T) {
		mockClient := new(MockProductAssessmentServiceClient)
		expectedResponse := &pb.ProductRateResponse{
			InitialStructureIndexRate: 500,
		}
		mockClient.On("GetProductRate", ctx, request, mock.Anything).Return(expectedResponse, nil)

		service := NewProductAssessmentService(retryConfig, nil).(*productAssessmentService)
		service.client = mockClient

		response, err := service.GetProductRate(ctx, request)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, expectedResponse.InitialStructureIndexRate, response.InitialStructureIndexRate)
		mockClient.AssertExpectations(t)
	})

	t.Run("nil request", func(t *testing.T) {
		service := NewProductAssessmentService(retryConfig, nil)
		response, err := service.GetProductRate(ctx, nil)

		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "request cannot be nil")
	})

	t.Run("service error", func(t *testing.T) {
		mockClient := new(MockProductAssessmentServiceClient)
		expectedErr := errors.New("service error")
		mockClient.On("GetProductRate", ctx, request, mock.Anything).Return(nil, expectedErr)

		service := NewProductAssessmentService(retryConfig, nil).(*productAssessmentService)
		service.client = mockClient

		response, err := service.GetProductRate(ctx, request)

		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "failed to get product rate")
		mockClient.AssertExpectations(t)
	})

	t.Run("retry on error", func(t *testing.T) {
		mockClient := new(MockProductAssessmentServiceClient)
		expectedResponse := &pb.ProductRateResponse{
			InitialStructureIndexRate: 500,
		}
		// First call fails, second call succeeds
		mockClient.On("GetProductRate", ctx, request, mock.Anything).Return(nil, errors.New("temporary error")).Once()
		mockClient.On("GetProductRate", ctx, request, mock.Anything).Return(expectedResponse, nil).Once()

		service := NewProductAssessmentService(retryConfig, nil).(*productAssessmentService)
		service.client = mockClient

		response, err := service.GetProductRate(ctx, request)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, expectedResponse.InitialStructureIndexRate, response.InitialStructureIndexRate)
		mockClient.AssertExpectations(t)
	})

	t.Run("empty product code uses default", func(t *testing.T) {
		mockClient := new(MockProductAssessmentServiceClient)
		emptyRequest := &pb.ProductRateRequest{
			ProductCode: "",
			ProductName: "Test Product",
		}
		expectedRequest := &pb.ProductRateRequest{
			ProductCode: defaultProductCode,
			ProductName: "Test Product",
		}
		expectedResponse := &pb.ProductRateResponse{
			InitialStructureIndexRate: 500,
		}
		mockClient.On("GetProductRate", ctx, expectedRequest, mock.Anything).Return(expectedResponse, nil)

		service := NewProductAssessmentService(retryConfig, nil).(*productAssessmentService)
		service.client = mockClient

		response, err := service.GetProductRate(ctx, emptyRequest)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, expectedResponse.InitialStructureIndexRate, response.InitialStructureIndexRate)
		mockClient.AssertExpectations(t)
	})
}

// mockProductAssessmentServer implements the ProductAssessmentServiceServer interface for testing
type mockProductAssessmentServer struct {
	pb.UnimplementedProductAssessmentServiceServer
}

func (s *mockProductAssessmentServer) GetProductRate(ctx context.Context, req *pb.ProductRateRequest) (*pb.ProductRateResponse, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	return &pb.ProductRateResponse{
		InitialStructureIndexRate: 123,
	}, nil
}

func setupTestCertificates(t *testing.T) (string, string, string) {
	testDir := filepath.Join("testdata")
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create testdata dir: %v", err)
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{"Test CA"}},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
	}
	caCert, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("Failed to create CA cert: %v", err)
	}

	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{Organization: []string{"Test Server"}, CommonName: "test-server"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		DNSNames:     []string{"test-server"},
	}
	serverCert, err := x509.CreateCertificate(rand.Reader, serverTemplate, caTemplate, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("Failed to create server cert: %v", err)
	}

	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{Organization: []string{"Test Client"}, CommonName: "test-client"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}
	clientCert, err := x509.CreateCertificate(rand.Reader, clientTemplate, caTemplate, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("Failed to create client cert: %v", err)
	}

	caFile := filepath.Join(testDir, "ca.crt")
	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCert})
	if err := os.WriteFile(caFile, caCertPEM, 0644); err != nil {
		t.Fatalf("Failed to write CA cert: %v", err)
	}

	serverCertFile := filepath.Join(testDir, "server.crt")
	serverCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverCert})
	if err := os.WriteFile(serverCertFile, serverCertPEM, 0644); err != nil {
		t.Fatalf("Failed to write server cert: %v", err)
	}

	clientCertFile := filepath.Join(testDir, "client.crt")
	clientCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientCert})
	if err := os.WriteFile(clientCertFile, clientCertPEM, 0644); err != nil {
		t.Fatalf("Failed to write client cert: %v", err)
	}

	keyFile := filepath.Join(testDir, "client.key")
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	if err := os.WriteFile(keyFile, privateKeyPEM, 0600); err != nil {
		t.Fatalf("Failed to write private key: %v", err)
	}

	return serverCertFile, keyFile, caFile
}

func TestNewProductAssessmentClient(t *testing.T) {
	certFile, keyFile, caFile := setupTestCertificates(t)
	defer os.RemoveAll("testdata")

	serverTLS, err := tlsSecurityConfig.NewTLSConfig(&tlsSecurityConfig.TLSConfig{
		CertFile:   certFile,
		KeyFile:    keyFile,
		CAFile:     caFile,
		ServerName: "test-server",
	})
	if err != nil {
		t.Fatalf("Failed to create server TLS config: %v", err)
	}

	server := grpc.NewServer(grpc.Creds(credentials.NewTLS(serverTLS)))
	pb.RegisterProductAssessmentServiceServer(server, &mockProductAssessmentServer{})

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	go server.Serve(lis)
	defer server.Stop()

	tlsConfig := &tlsSecurityConfig.TLSConfig{
		CertFile:   certFile,
		KeyFile:    keyFile,
		CAFile:     caFile,
		ServerName: "test-server",
	}

	retryConfig := &utils.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
	}

	client, err := NewProductAssessmentClient(context.Background(), lis.Addr().String(), tlsConfig, retryConfig)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	resp, err := client.GetProductRate(context.Background(), &pb.ProductRateRequest{ProductCode: "P", ProductName: "N"})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(123), resp.InitialStructureIndexRate)
}
