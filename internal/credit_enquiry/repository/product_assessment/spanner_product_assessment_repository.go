package product_assessment

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"go-loan-service-v3/internal/credit_enquiry/entity"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	productAssessmentTable = "product_assessment"
)

// SpannerProductAssessmentRepository implements the ProductAssessmentRepository interface using Google Spanner
type SpannerProductAssessmentRepository struct {
	client *spanner.Client
}

// NewSpannerProductAssessmentRepository creates a new instance of SpannerProductAssessmentRepository
func NewSpannerProductAssessmentRepository(client *spanner.Client) *SpannerProductAssessmentRepository {
	return &SpannerProductAssessmentRepository{
		client: client,
	}
}

// SaveProductAssessment creates a new product assessment record in Spanner
func (r *SpannerProductAssessmentRepository) SaveProductAssessment(ctx context.Context, assessment *entity.ProductAssessment) error {
	now := time.Now()
	assessment.CreatedAt = now
	assessment.UpdatedAt = now

	// Convert float64 to *big.Rat for Spanner NUMERIC
	loanAmount := new(big.Rat).SetFloat64(assessment.LoanAmount)
	loanAmountNumeric := spanner.NumericString(loanAmount)

	mutation := spanner.InsertOrUpdate(
		productAssessmentTable,
		[]string{
			"product_assessment_id",
			"serviceability_assessment_id",
			"credit_enquiry_id",
			"credit_enquiry_version",
			"product_code",
			"product_name",
			"loan_amount",
			"loan_purpose",
			"initial_structure_term_month",
			"initial_structure_index_rate",
			"created_at",
			"updated_at",
		},
		[]interface{}{
			assessment.ProductAssessmentID,
			assessment.ServiceabilityAssessmentID,
			assessment.CreditEnquiryID,
			assessment.CreditEnquiryVersion,
			assessment.ProductCode,
			assessment.ProductName,
			loanAmountNumeric,
			assessment.LoanPurpose,
			assessment.InitialStructureTermMonth,
			assessment.InitialStructureIndexRate,
			assessment.CreatedAt,
			assessment.UpdatedAt,
		},
	)

	_, err := r.client.Apply(ctx, []*spanner.Mutation{mutation})
	if err != nil {
		return fmt.Errorf("failed to save product assessment: %w", err)
	}

	return nil
}

// GetProductAssessment retrieves a product assessment by credit enquiry ID and version
func (r *SpannerProductAssessmentRepository) GetProductAssessment(ctx context.Context, creditEnquiryID string, version string) (*entity.ProductAssessment, error) {
	stmt := spanner.Statement{
		SQL: `SELECT 
			product_assessment_id,
			serviceability_assessment_id,
			credit_enquiry_id,
			credit_enquiry_version,
			product_code,
			product_name,
			CAST(loan_amount AS FLOAT64) as loan_amount,
			loan_purpose,
			initial_structure_term_month,
			initial_structure_index_rate,
			created_at,
			updated_at
		FROM product_assessment
		WHERE credit_enquiry_id = @credit_enquiry_id
		AND credit_enquiry_version = @version
		LIMIT 1`,
		Params: map[string]interface{}{
			"credit_enquiry_id": creditEnquiryID,
			"version":           version,
		},
	}

	var assessment entity.ProductAssessment
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return nil, status.Errorf(codes.NotFound, "product assessment not found")
		}
		return nil, fmt.Errorf("failed to get product assessment: %w", err)
	}

	err = row.Columns(
		&assessment.ProductAssessmentID,
		&assessment.ServiceabilityAssessmentID,
		&assessment.CreditEnquiryID,
		&assessment.CreditEnquiryVersion,
		&assessment.ProductCode,
		&assessment.ProductName,
		&assessment.LoanAmount,
		&assessment.LoanPurpose,
		&assessment.InitialStructureTermMonth,
		&assessment.InitialStructureIndexRate,
		&assessment.CreatedAt,
		&assessment.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan product assessment: %w", err)
	}

	return &assessment, nil
}
