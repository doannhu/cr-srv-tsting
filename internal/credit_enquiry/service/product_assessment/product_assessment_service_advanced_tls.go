package product_assessment

import (
	"context"
	"fmt"
	"os"

	tlsSecurityConfig "go-loan-service-v3/internal/config/security"
	"go-loan-service-v3/internal/credit_enquiry/interfaces"
	"go-loan-service-v3/internal/credit_enquiry/utils"
	pb "go-loan-service-v3/proto/product_assessment"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/tls/certprovider/pemfile"
	"google.golang.org/grpc/security/advancedtls"
)

// productAssessmentService embeds BaseService for common functionality
type productAssessmentService struct {
	*BaseService
}

// NewProductAssessmentClientAT creates a client using **advanced-TLS**.
func NewProductAssessmentClientAT(
	ctx context.Context,
	addr string,
	tlsCfg *tlsSecurityConfig.TLSConfig,
	retryCfg *utils.RetryConfig,
) (interfaces.ProductAssessmentService, error) {

	if tlsCfg == nil {
		return nil, fmt.Errorf("TLS configuration is required")
	}

	// Initialize credential struct using reloading API.
	idProvider, err := pemfile.NewProvider(pemfile.Options{
		CertFile:        tlsCfg.CertFile,
		KeyFile:         tlsCfg.KeyFile,
		RefreshDuration: CredRefreshingInterval,
	})
	if err != nil {
		return nil, fmt.Errorf("pemfile identity provider: %w", err)
	}
	rootOptions := pemfile.Options{
		RootFile:        tlsCfg.CAFile,
		RefreshDuration: CredRefreshingInterval,
	}
	// Validate CA file path before creating root provider.
	if _, err := os.Stat(tlsCfg.CAFile); err != nil {
		return nil, fmt.Errorf("pemfile root provider: %w", err)
	}
	rootProvider, err := pemfile.NewProvider(rootOptions)
	if err != nil {
		return nil, fmt.Errorf("pemfile root provider: %w", err)
	}

	atOpts := &advancedtls.Options{
		IdentityOptions: advancedtls.IdentityCertificateOptions{
			IdentityProvider: idProvider,
		},
		RootOptions: advancedtls.RootCertificateOptions{
			RootProvider: rootProvider,
		},
		VerificationType: advancedtls.CertVerification,
	}

	creds, err := advancedtls.NewClientCreds(atOpts)
	if err != nil {
		return nil, fmt.Errorf("advancedtls creds: %w", err)
	}

	// Make a connection using the credentials.
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("NewClient %s: %w", addr, err)
	}

	// Wait for the connection to be ready
	// state := conn.GetState()
	// if state != connectivity.Ready {
	// 	conn.Close()
	// 	return nil, fmt.Errorf("connection not ready: %s", state)
	// }

	client := pb.NewProductAssessmentServiceClient(conn)
	base := NewBaseService(retryCfg, client)
	return &productAssessmentService{BaseService: base}, nil
}
