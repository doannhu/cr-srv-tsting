package security

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitTLSClient(t *testing.T) {
	// Create temporary directory for test certificates
	tmpDir, err := os.MkdirTemp("", "tls-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Test paths
	certFile := filepath.Join(tmpDir, "client.pem")
	keyFile := filepath.Join(tmpDir, "client-key.pem")
	caFile := filepath.Join(tmpDir, "ca.pem")

	tests := []struct {
		name    string
		cfg     *ClientTLSConfig
		setup   func() error
		wantErr bool
	}{
		{
			name: "missing cert file",
			cfg: &ClientTLSConfig{
				CertFile:   certFile,
				KeyFile:    keyFile,
				CAFile:     caFile,
				ServerName: "product-svc.internal",
			},
			setup:   func() error { return nil },
			wantErr: true,
		},
		{
			name: "missing key file",
			cfg: &ClientTLSConfig{
				CertFile:   "nonexistent.pem",
				KeyFile:    keyFile,
				CAFile:     caFile,
				ServerName: "product-svc.internal",
			},
			setup:   func() error { return nil },
			wantErr: true,
		},
		{
			name: "missing CA file",
			cfg: &ClientTLSConfig{
				CertFile:   certFile,
				KeyFile:    "nonexistent-key.pem",
				CAFile:     caFile,
				ServerName: "product-svc.internal",
			},
			setup:   func() error { return nil },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setup()
			require.NoError(t, err)

			tlsConfig, err := initTLSClient(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, tlsConfig)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tlsConfig)
				if tlsConfig != nil {
					assert.Equal(t, tt.cfg.ServerName, tlsConfig.ServerName)
				}
			}
		})
	}
}
