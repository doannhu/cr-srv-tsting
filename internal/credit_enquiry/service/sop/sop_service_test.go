package sop

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	tlsSecurityConfig "go-loan-service-v3/internal/config/security"
	"go-loan-service-v3/internal/credit_enquiry/utils"
	pb "go-loan-service-v3/proto/sop"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

type mockSOPClient struct {
	attempts       int
	permanentError bool
}

func (m *mockSOPClient) GetConsolidatedSOP(ctx context.Context, in *pb.SOPRequest, opts ...grpc.CallOption) (*pb.SOPResponse, error) {
	m.attempts++

	// For the permanent error test case, always return an error
	if m.permanentError {
		return nil, errors.New("permanent error")
	}

	// For the transient error test case, succeed on the third attempt
	if m.attempts == 3 {
		return &pb.SOPResponse{
			SopAssessmentId:                  "sop-123",
			ServiceabilityAssessmentId:       "sa-123",
			TotalMonthlyNetIncomeAmount:      5000.0,
			TotalAnnualGrossIncome:           60000.0,
			TotalSavingsAmount:               10000.0,
			TotalNumberOfContinuingHomeLoans: 1,
		}, nil
	}

	// Return a transient error for the first two attempts
	return nil, status.Error(codes.Unavailable, "temporary error")
}

func TestGetConsolidatedSOP(t *testing.T) {
	tests := []struct {
		name             string
		client           *mockSOPClient
		expectedError    error
		expectedResponse *pb.SOPResponse
	}{
		{
			name:   "successful response",
			client: &mockSOPClient{},
			expectedResponse: &pb.SOPResponse{
				SopAssessmentId:                  "sop-123",
				ServiceabilityAssessmentId:       "sa-123",
				TotalMonthlyNetIncomeAmount:      5000.0,
				TotalAnnualGrossIncome:           60000.0,
				TotalSavingsAmount:               10000.0,
				TotalNumberOfContinuingHomeLoans: 1,
			},
		},
		{
			name:   "transient error with retry success",
			client: &mockSOPClient{},
			expectedResponse: &pb.SOPResponse{
				SopAssessmentId:                  "sop-123",
				ServiceabilityAssessmentId:       "sa-123",
				TotalMonthlyNetIncomeAmount:      5000.0,
				TotalAnnualGrossIncome:           60000.0,
				TotalSavingsAmount:               10000.0,
				TotalNumberOfContinuingHomeLoans: 1,
			},
		},
		{
			name:          "permanent error",
			client:        &mockSOPClient{permanentError: true},
			expectedError: errors.New("permanent error"),
		},
	}

	retryConfig := &utils.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewSOPServiceClient(nil, retryConfig)
			client.(*StandardService).Client = tt.client
			response, err := client.GetConsolidatedSOP(context.Background(), &pb.SOPRequest{
				CreditEnquiryId:      "test-id",
				CreditEnquiryVersion: "1.0",
			})

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Nil(t, response)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, response)
				assert.Equal(t, tt.expectedResponse.SopAssessmentId, response.SopAssessmentId)
				assert.Equal(t, tt.expectedResponse.ServiceabilityAssessmentId, response.ServiceabilityAssessmentId)
				assert.Equal(t, tt.expectedResponse.TotalMonthlyNetIncomeAmount, response.TotalMonthlyNetIncomeAmount)
				assert.Equal(t, tt.expectedResponse.TotalAnnualGrossIncome, response.TotalAnnualGrossIncome)
				assert.Equal(t, tt.expectedResponse.TotalSavingsAmount, response.TotalSavingsAmount)
				assert.Equal(t, tt.expectedResponse.TotalNumberOfContinuingHomeLoans, response.TotalNumberOfContinuingHomeLoans)
			}
		})
	}
}

func setupTestCertificates(t *testing.T) (string, string, string) {
	// Create testdata directory
	testDir := filepath.Join("testdata")
	err := os.MkdirAll(testDir, 0755)
	require.NoError(t, err)

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Create CA certificate
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test CA"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
	}

	caCert, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	// Create server certificate
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Test Server"},
			CommonName:   "test-server",
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(24 * time.Hour),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		DNSNames:    []string{"test-server"},
	}

	serverCert, err := x509.CreateCertificate(rand.Reader, serverTemplate, caTemplate, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	// Create client certificate
	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject: pkix.Name{
			Organization: []string{"Test Client"},
			CommonName:   "test-client",
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(24 * time.Hour),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}

	clientCert, err := x509.CreateCertificate(rand.Reader, clientTemplate, caTemplate, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	// Write certificates to files
	caFile := filepath.Join(testDir, "ca.crt")
	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCert})
	err = os.WriteFile(caFile, caCertPEM, 0644)
	require.NoError(t, err)

	serverCertFile := filepath.Join(testDir, "server.crt")
	serverCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverCert})
	err = os.WriteFile(serverCertFile, serverCertPEM, 0644)
	require.NoError(t, err)

	clientCertFile := filepath.Join(testDir, "client.crt")
	clientCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientCert})
	err = os.WriteFile(clientCertFile, clientCertPEM, 0644)
	require.NoError(t, err)

	keyFile := filepath.Join(testDir, "client.key")
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	err = os.WriteFile(keyFile, privateKeyPEM, 0600)
	require.NoError(t, err)

	return serverCertFile, keyFile, caFile
}

func TestNewSOPClient(t *testing.T) {
	// Setup test certificates
	certFile, keyFile, caFile := setupTestCertificates(t)
	defer os.RemoveAll("testdata")

	// Create TLS credentials for server
	serverTLS, err := tlsSecurityConfig.NewTLSConfig(&tlsSecurityConfig.TLSConfig{
		CertFile:   certFile,
		KeyFile:    keyFile,
		CAFile:     caFile,
		ServerName: "test-server",
	})
	require.NoError(t, err)

	// Create a test server with TLS
	server := grpc.NewServer(grpc.Creds(credentials.NewTLS(serverTLS)))
	pb.RegisterSOPServiceServer(server, &mockSOPServer{})

	// Start the server
	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	go server.Serve(lis)
	defer server.Stop()

	// Create a non-TLS server for testing non-TLS connections
	nonTLSServer := grpc.NewServer()
	pb.RegisterSOPServiceServer(nonTLSServer, &mockSOPServer{})
	nonTLSLis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	go nonTLSServer.Serve(nonTLSLis)
	defer nonTLSServer.Stop()

	tests := []struct {
		name        string
		addr        string
		tlsConfig   *tlsSecurityConfig.TLSConfig
		retryConfig *utils.RetryConfig
		wantErr     bool
	}{
		{
			name: "successful connection with TLS",
			addr: lis.Addr().String(),
			tlsConfig: &tlsSecurityConfig.TLSConfig{
				CertFile:   certFile,
				KeyFile:    keyFile,
				CAFile:     caFile,
				ServerName: "test-server",
			},
			retryConfig: &utils.RetryConfig{
				MaxAttempts: 3,
				BaseDelay:   100 * time.Millisecond,
				MaxDelay:    1 * time.Second,
			},
			wantErr: false,
		},
		{
			name:      "connection without TLS should fail",
			addr:      lis.Addr().String(), // Using TLS server address
			tlsConfig: nil,
			retryConfig: &utils.RetryConfig{
				MaxAttempts: 3,
				BaseDelay:   100 * time.Millisecond,
				MaxDelay:    1 * time.Second,
			},
			wantErr: true,
		},
		// {
		// 	name: "invalid address should fail",
		// 	addr: "invalid-address:1234",
		// 	tlsConfig: &tlsSecurityConfig.TLSConfig{
		// 		CertFile:   certFile,
		// 		KeyFile:    keyFile,
		// 		CAFile:     caFile,
		// 		ServerName: "test-server",
		// 	},
		// 	retryConfig: &utils.RetryConfig{
		// 		MaxAttempts: 3,
		// 		BaseDelay:   100 * time.Millisecond,
		// 		MaxDelay:    1 * time.Second,
		// 	},
		// 	wantErr: true,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewSOPClient(context.Background(), tt.addr, tt.tlsConfig, tt.retryConfig)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestSOPClient_GetConsolidatedSOP(t *testing.T) {
	// Setup test certificates
	certFile, keyFile, caFile := setupTestCertificates(t)
	defer os.RemoveAll("testdata")

	// Create TLS credentials for server
	serverTLS, err := tlsSecurityConfig.NewTLSConfig(&tlsSecurityConfig.TLSConfig{
		CertFile:   certFile,
		KeyFile:    keyFile,
		CAFile:     caFile,
		ServerName: "test-server",
	})
	require.NoError(t, err)

	// Create a test server with TLS
	server := grpc.NewServer(grpc.Creds(credentials.NewTLS(serverTLS)))
	pb.RegisterSOPServiceServer(server, &mockSOPServer{})

	// Start the server
	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	go server.Serve(lis)
	defer server.Stop()

	// Create client with TLS credentials
	client, err := NewSOPClient(context.Background(), lis.Addr().String(), &tlsSecurityConfig.TLSConfig{
		CertFile:   certFile,
		KeyFile:    keyFile,
		CAFile:     caFile,
		ServerName: "test-server",
	}, &utils.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
	})
	require.NoError(t, err)

	tests := []struct {
		name    string
		request *pb.SOPRequest
		want    *pb.SOPResponse
		wantErr bool
	}{
		{
			name: "successful request",
			request: &pb.SOPRequest{
				CreditEnquiryId:      "test-id",
				CreditEnquiryVersion: "1.0",
			},
			want: &pb.SOPResponse{
				SopAssessmentId:                  "test-sop-id",
				ServiceabilityAssessmentId:       "test-serviceability-id",
				TotalMonthlyNetIncomeAmount:      5000.0,
				TotalAnnualGrossIncome:           60000.0,
				TotalSavingsAmount:               10000.0,
				TotalNumberOfContinuingHomeLoans: 1,
			},
			wantErr: false,
		},
		{
			name:    "nil request",
			request: nil,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := client.GetConsolidatedSOP(context.Background(), tt.request)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Equal(t, tt.want.SopAssessmentId, response.SopAssessmentId)
				assert.Equal(t, tt.want.ServiceabilityAssessmentId, response.ServiceabilityAssessmentId)
				assert.Equal(t, tt.want.TotalMonthlyNetIncomeAmount, response.TotalMonthlyNetIncomeAmount)
				assert.Equal(t, tt.want.TotalAnnualGrossIncome, response.TotalAnnualGrossIncome)
				assert.Equal(t, tt.want.TotalSavingsAmount, response.TotalSavingsAmount)
				assert.Equal(t, tt.want.TotalNumberOfContinuingHomeLoans, response.TotalNumberOfContinuingHomeLoans)
			}
		})
	}
}

// mockSOPServer implements the SOPServiceServer interface for testing
type mockSOPServer struct {
	pb.UnimplementedSOPServiceServer
}

func (s *mockSOPServer) GetConsolidatedSOP(ctx context.Context, req *pb.SOPRequest) (*pb.SOPResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	return &pb.SOPResponse{
		SopAssessmentId:                  "test-sop-id",
		ServiceabilityAssessmentId:       "test-serviceability-id",
		TotalMonthlyNetIncomeAmount:      5000.0,
		TotalAnnualGrossIncome:           60000.0,
		TotalSavingsAmount:               10000.0,
		TotalNumberOfContinuingHomeLoans: 1,
	}, nil
}
