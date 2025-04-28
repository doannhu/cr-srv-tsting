package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"
)

func TestNewTLSConfig(t *testing.T) {
	// Create temporary directory for test certificates
	tempDir, err := os.MkdirTemp("", "tls-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Generate test certificates
	certFile, keyFile, caFile, err := generateTestCertificates(tempDir)
	if err != nil {
		t.Fatalf("Failed to generate test certificates: %v", err)
	}

	tests := []struct {
		name    string
		config  *TLSConfig
		wantErr bool
	}{
		{
			name: "valid configuration",
			config: &TLSConfig{
				CertFile:   certFile,
				KeyFile:    keyFile,
				CAFile:     caFile,
				ServerName: "test-server",
			},
			wantErr: false,
		},
		{
			name:    "nil configuration",
			config:  nil,
			wantErr: true,
		},
		{
			name: "missing certificate file",
			config: &TLSConfig{
				CertFile:   "nonexistent.crt",
				KeyFile:    keyFile,
				CAFile:     caFile,
				ServerName: "test-server",
			},
			wantErr: true,
		},
		{
			name: "missing key file",
			config: &TLSConfig{
				CertFile:   certFile,
				KeyFile:    "nonexistent.key",
				CAFile:     caFile,
				ServerName: "test-server",
			},
			wantErr: true,
		},
		{
			name: "missing CA file",
			config: &TLSConfig{
				CertFile:   certFile,
				KeyFile:    keyFile,
				CAFile:     "nonexistent.ca",
				ServerName: "test-server",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewTLSConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTLSConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && config == nil {
				t.Error("NewTLSConfig() returned nil config when no error was expected")
			}
		})
	}
}

func TestNewInsecureTLSConfig(t *testing.T) {
	config := NewInsecureTLSConfig()
	if config == nil {
		t.Fatal("NewInsecureTLSConfig() returned nil")
	}
	if !config.InsecureSkipVerify {
		t.Error("NewInsecureTLSConfig() InsecureSkipVerify is false")
	}
}

// Helper function to generate test certificates
func generateTestCertificates(dir string) (string, string, string, error) {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", "", err
	}

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
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	caCert, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &privateKey.PublicKey, privateKey)
	if err != nil {
		return "", "", "", err
	}

	// Create client certificate
	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Test Client"},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(24 * time.Hour),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}

	clientCert, err := x509.CreateCertificate(rand.Reader, clientTemplate, caTemplate, &privateKey.PublicKey, privateKey)
	if err != nil {
		return "", "", "", err
	}

	// Write certificates to files
	caFile := dir + "/ca.crt"
	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCert})
	if err := os.WriteFile(caFile, caCertPEM, 0644); err != nil {
		return "", "", "", err
	}

	certFile := dir + "/client.crt"
	clientCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientCert})
	if err := os.WriteFile(certFile, clientCertPEM, 0644); err != nil {
		return "", "", "", err
	}

	keyFile := dir + "/client.key"
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	if err := os.WriteFile(keyFile, privateKeyPEM, 0600); err != nil {
		return "", "", "", err
	}

	return certFile, keyFile, caFile, nil
}
