package security

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// TLSConfig represents the TLS configuration for gRPC services
type TLSConfig struct {
	// CertFile is the path to the client certificate file
	CertFile string
	// KeyFile is the path to the client private key file
	KeyFile string
	// CAFile is the path to the CA certificate file
	CAFile string
	// ServerName is the expected server name in the certificate
	ServerName string
	// InsecureSkipVerify determines whether to skip server certificate verification
	InsecureSkipVerify bool
}

// NewTLSConfig creates a new TLS configuration
func NewTLSConfig(cfg *TLSConfig) (*tls.Config, error) {
	if cfg == nil {
		return nil, fmt.Errorf("TLS configuration is required")
	}

	// Load client certificate
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %v", err)
	}

	// Load CA certificate
	caCert, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %v", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certificate")
	}

	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caCertPool,
		ServerName:         cfg.ServerName,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}, nil
}

// NewInsecureTLSConfig creates an insecure TLS configuration for testing
func NewInsecureTLSConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true,
	}
}
