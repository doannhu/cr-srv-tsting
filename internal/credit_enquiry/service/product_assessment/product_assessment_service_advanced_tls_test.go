package product_assessment

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	tlsSecurityConfig "go-loan-service-v3/internal/config/security"
	"go-loan-service-v3/internal/credit_enquiry/utils"
	pb "go-loan-service-v3/proto/product_assessment"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestNewProductAssessmentClientAT(t *testing.T) {
	certFile, keyFile, caFile := setupTestCertificates(t)
	defer os.RemoveAll("testdata")

	// Start a test server with TLS
	serverTLS, err := tlsSecurityConfig.NewTLSConfig(&tlsSecurityConfig.TLSConfig{
		CertFile:   certFile,
		KeyFile:    keyFile,
		CAFile:     caFile,
		ServerName: "test-server",
	})
	require.NoError(t, err)

	server := grpc.NewServer(grpc.Creds(credentials.NewTLS(serverTLS)))
	pb.RegisterProductAssessmentServiceServer(server, &mockProductAssessmentServer{})

	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	errChan := make(chan error)
	go func() {
		if err := server.Serve(lis); err != nil {
			errChan <- err
		}
	}()
	defer server.Stop()

	// Wait a bit for the server to start
	time.Sleep(100 * time.Millisecond)

	retryConfig := &utils.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
	}

	t.Run("successful connection with advanced TLS", func(t *testing.T) {
		tlsConfig := &tlsSecurityConfig.TLSConfig{
			CertFile:   certFile,
			KeyFile:    keyFile,
			CAFile:     caFile,
			ServerName: "test-server",
		}

		client, err := NewProductAssessmentClientAT(context.Background(), lis.Addr().String(), tlsConfig, retryConfig)
		require.NoError(t, err)
		require.NotNil(t, client)

		// Test the client works
		resp, err := client.GetProductRate(context.Background(), &pb.ProductRateRequest{
			ProductCode: "TEST",
			ProductName: "Test Product",
		})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, int64(123), resp.InitialStructureIndexRate)
	})

	t.Run("nil TLS config", func(t *testing.T) {
		client, err := NewProductAssessmentClientAT(context.Background(), lis.Addr().String(), nil, retryConfig)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "TLS configuration is required")
	})

	t.Run("invalid certificate path", func(t *testing.T) {
		tlsConfig := &tlsSecurityConfig.TLSConfig{
			CertFile:   "nonexistent.crt",
			KeyFile:    keyFile,
			CAFile:     caFile,
			ServerName: "test-server",
		}

		client, err := NewProductAssessmentClientAT(context.Background(), lis.Addr().String(), tlsConfig, retryConfig)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "pemfile identity provider")
	})

	t.Run("invalid key path", func(t *testing.T) {
		tlsConfig := &tlsSecurityConfig.TLSConfig{
			CertFile:   certFile,
			KeyFile:    "nonexistent.key",
			CAFile:     caFile,
			ServerName: "test-server",
		}

		client, err := NewProductAssessmentClientAT(context.Background(), lis.Addr().String(), tlsConfig, retryConfig)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "pemfile identity provider")
	})

	t.Run("invalid CA path", func(t *testing.T) {
		tlsConfig := &tlsSecurityConfig.TLSConfig{
			CertFile:   certFile,
			KeyFile:    keyFile,
			CAFile:     "nonExistingCAFilePath",
			ServerName: "test-server",
		}

		client, err := NewProductAssessmentClientAT(context.Background(), lis.Addr().String(), tlsConfig, retryConfig)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "pemfile root provider")
	})

	t.Run("connection to invalid address", func(t *testing.T) {
		tlsConfig := &tlsSecurityConfig.TLSConfig{
			CertFile:   certFile,
			KeyFile:    keyFile,
			CAFile:     caFile,
			ServerName: "test-server",
		}

		client, err := NewProductAssessmentClientAT(context.Background(), "invalid-addr:1234", tlsConfig, retryConfig)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "dial invalid-addr:1234")
	})

	t.Run("certificate refresh test", func(t *testing.T) {
		// Create a new server for this test to avoid interference
		newLis, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		newServer := grpc.NewServer(grpc.Creds(credentials.NewTLS(serverTLS)))
		pb.RegisterProductAssessmentServiceServer(newServer, &mockProductAssessmentServer{})
		go newServer.Serve(newLis)
		defer newServer.Stop()

		// Generate new certificates in a temporary location
		tempDir := t.TempDir()
		certFile := filepath.Join(tempDir, "client.crt")
		keyFile := filepath.Join(tempDir, "client.key")
		caFile := filepath.Join(tempDir, "ca.crt")

		// Generate and write initial certificates
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		template := &x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject:      pkix.Name{Organization: []string{"Test Org"}},
			NotBefore:    time.Now(),
			NotAfter:     time.Now().Add(24 * time.Hour),
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		}

		certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
		require.NoError(t, err)

		// Write the certificate and key files
		err = os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes}), 0644)
		require.NoError(t, err)
		err = os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}), 0600)
		require.NoError(t, err)
		err = os.WriteFile(caFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes}), 0644)
		require.NoError(t, err)

		tlsConfig := &tlsSecurityConfig.TLSConfig{
			CertFile:   certFile,
			KeyFile:    keyFile,
			CAFile:     caFile,
			ServerName: "test-server",
		}

		// Create client with short refresh interval for testing
		client, err := NewProductAssessmentClientAT(context.Background(), newLis.Addr().String(), tlsConfig, retryConfig)
		require.NoError(t, err)
		require.NotNil(t, client)

		// Verify initial connection works
		resp, err := client.GetProductRate(context.Background(), &pb.ProductRateRequest{ProductCode: "TEST"})
		assert.NoError(t, err)
		assert.NotNil(t, resp)

		// Generate and write new certificates (simulating rotation)
		newPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		newTemplate := &x509.Certificate{
			SerialNumber: big.NewInt(2),
			Subject:      pkix.Name{Organization: []string{"Test Org New"}},
			NotBefore:    time.Now(),
			NotAfter:     time.Now().Add(24 * time.Hour),
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		}

		newCertBytes, err := x509.CreateCertificate(rand.Reader, newTemplate, newTemplate, &newPrivateKey.PublicKey, newPrivateKey)
		require.NoError(t, err)

		// Write the new certificate and key files
		err = os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: newCertBytes}), 0644)
		require.NoError(t, err)
		err = os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(newPrivateKey)}), 0600)
		require.NoError(t, err)

		// Wait for refresh interval
		time.Sleep(CredRefreshingInterval + time.Second)

		// Verify connection still works with new certificates
		resp, err = client.GetProductRate(context.Background(), &pb.ProductRateRequest{ProductCode: "TEST"})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
	})
}
