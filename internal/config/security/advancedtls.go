package security

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

type ClientTLSConfig struct {
	CertFile   string
	KeyFile    string
	CAFile     string
	ServerName string
}

func initTLSClient(cfg *ClientTLSConfig) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client cert/key: %v", err)
	}

	caCert, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA cert: %v", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to add CA cert to pool")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		ServerName:   cfg.ServerName,
	}, nil
}

func NewProductAssessmentClient() error {
	tlsCfg := &ClientTLSConfig{
		CertFile:   "/etc/certs/client.pem",
		KeyFile:    "/etc/certs/client-key.pem",
		CAFile:     "/etc/certs/ca.pem",
		ServerName: "product-svc.internal", // must match SAN/CN
	}

	_, err := initTLSClient(tlsCfg)
	return err
}
