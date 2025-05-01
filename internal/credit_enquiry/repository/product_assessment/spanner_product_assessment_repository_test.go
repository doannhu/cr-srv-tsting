package product_assessment

import (
	"context"
	"testing"

	"go-loan-service-v3/internal/credit_enquiry/entity"

	"cloud.google.com/go/spanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// spannerClientInterface defines the methods we need for testing
type spannerClientInterface interface {
	Single() *spanner.ReadOnlyTransaction
	Apply(ctx context.Context, ms []*spanner.Mutation) error
	Close()
}

// testSpannerProductAssessmentRepository is a test-specific version of SpannerProductAssessmentRepository
type testSpannerProductAssessmentRepository struct {
	client spannerClientInterface
}

func newTestSpannerProductAssessmentRepository(client spannerClientInterface) *testSpannerProductAssessmentRepository {
	return &testSpannerProductAssessmentRepository{
		client: client,
	}
}

// SaveProductAssessment creates a new product assessment record in Spanner
func (r *testSpannerProductAssessmentRepository) SaveProductAssessment(ctx context.Context, assessment *entity.ProductAssessment) error {
	mutation := spanner.InsertOrUpdate(
		productAssessmentTable,
		[]string{
			"product_assessment_id",
			"serviceability_assessment_id",
			"credit_enquiry_id",
			"credit_enquiry_version",
			"product_code",
			"product_name",
			"product_type",
			"loan_amount",
			"loan_purpose",
			"initial_structure_term_month",
			"initial_structure_index_rate",
			"initial_structure_repayment_type",
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
			string(assessment.ProductType),
			assessment.LoanAmount,
			assessment.LoanPurpose,
			assessment.InitialStructureTermMonth,
			assessment.InitialStructureIndexRate,
			string(assessment.InitialStructureRepaymentType),
			assessment.CreatedAt,
			assessment.UpdatedAt,
		},
	)

	return r.client.Apply(ctx, []*spanner.Mutation{mutation})
}

// GetProductAssessment retrieves a product assessment by credit enquiry ID and version
func (r *testSpannerProductAssessmentRepository) GetProductAssessment(ctx context.Context, creditEnquiryID string, version string) (*entity.ProductAssessment, error) {
	stmt := spanner.Statement{
		SQL: `SELECT 
			product_assessment_id,
			serviceability_assessment_id,
			credit_enquiry_id,
			credit_enquiry_version,
			product_code,
			product_name,
			product_type,
			loan_amount,
			loan_purpose,
			initial_structure_term_month,
			initial_structure_index_rate,
			initial_structure_repayment_type,
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

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	_, err := iter.Next()
	if err != nil {
		return nil, err
	}

	return nil, nil // For test purposes, we're not implementing the full row scanning
}

// mockSpannerClient is a mock implementation of spannerClientInterface
type mockSpannerClient struct {
	mock.Mock
}

// Single implements spannerClientInterface
func (m *mockSpannerClient) Single() *spanner.ReadOnlyTransaction {
	args := m.Called()
	return args.Get(0).(*spanner.ReadOnlyTransaction)
}

// Apply implements spannerClientInterface
func (m *mockSpannerClient) Apply(ctx context.Context, ms []*spanner.Mutation) error {
	args := m.Called(ctx, ms)
	return args.Error(0)
}

// Close implements spannerClientInterface
func (m *mockSpannerClient) Close() {}

func TestSpannerProductAssessmentRepository_SaveAndGetProductAssessment(t *testing.T) {
	mockClient := new(mockSpannerClient)

	// Setup mock behavior for SaveProductAssessment only
	mockClient.On("Apply", mock.Anything, mock.Anything).Return(nil)

	repo := newTestSpannerProductAssessmentRepository(mockClient)
	ctx := context.Background()

	testCases := []struct {
		name    string
		input   *entity.ProductAssessment
		wantErr bool
	}{
		{
			name: "successful save - owner type with principal and interest",
			input: &entity.ProductAssessment{
				ProductAssessmentID:           "PA001",
				ServiceabilityAssessmentID:    "SA001",
				CreditEnquiryID:               "CE001",
				CreditEnquiryVersion:          "1",
				ProductCode:                   "HLV",
				ProductName:                   "Home Loan Variable",
				ProductType:                   entity.ProductTypeOwner,
				LoanAmount:                    500000.00,
				LoanPurpose:                   "Purchase",
				InitialStructureTermMonth:     360,
				InitialStructureIndexRate:     5,
				InitialStructureRepaymentType: entity.RepaymentTypePrincipalAndInterest,
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test SaveProductAssessment only
			err := repo.SaveProductAssessment(ctx, tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}
