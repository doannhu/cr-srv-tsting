package sop

import (
	"context"
	"fmt"

	"go-loan-service-v3/internal/credit_enquiry/entity"
	"go-loan-service-v3/internal/credit_enquiry/interfaces"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// spannerSopRepository implements the SopRepository interface using Spanner
type spannerSopRepository struct {
	client *spanner.Client
}

// NewSpannerSopRepository creates a new instance of spannerSopRepository
func NewSpannerSopRepository(client *spanner.Client) interfaces.SopRepository {
	return &spannerSopRepository{
		client: client,
	}
}

// SaveSop saves a Statement of Position to Spanner
func (r *spannerSopRepository) SaveSop(ctx context.Context, sop *entity.Sop) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Check if SOP already exists
		stmt := spanner.Statement{
			SQL: `SELECT sop_assessment_id FROM sop WHERE credit_enquiry_id = @creditEnquiryID AND credit_enquiry_version = @version`,
			Params: map[string]interface{}{
				"creditEnquiryID": sop.CreditEnquiryID,
				"version":         sop.CreditEnquiryVersion,
			},
		}
		iter := txn.Query(ctx, stmt)
		defer iter.Stop()

		_, err := iter.Next()
		if err == nil {
			return fmt.Errorf("sop already exists")
		}
		if err != iterator.Done {
			return err
		}

		// Insert new SOP
		var continuingLoans *int64
		if sop.TotalNumberOfContinuingHomeLoans != nil {
			val := int64(*sop.TotalNumberOfContinuingHomeLoans)
			continuingLoans = &val
		}

		m := spanner.InsertOrUpdate(
			"sop",
			[]string{
				"sop_assessment_id",
				"credit_enquiry_id",
				"credit_enquiry_version",
				"serviceability_assessment_id",
				"total_monthly_net_income_amount",
				"total_annual_gross_income",
				"total_savings_amount",
				"total_number_of_continuing_home_loans",
				"created_at",
			},
			[]interface{}{
				sop.SopAssessmentID,
				sop.CreditEnquiryID,
				sop.CreditEnquiryVersion,
				sop.ServiceabilityAssessmentID,
				sop.TotalMonthlyNetIncomeAmount,
				sop.TotalAnnualGrossIncome,
				sop.TotalSavingsAmount,
				continuingLoans,
				spanner.CommitTimestamp,
			},
		)

		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	return err
}

// GetSop retrieves a Statement of Position from Spanner
func (r *spannerSopRepository) GetSop(ctx context.Context, creditEnquiryID string, version string) (*entity.Sop, error) {
	stmt := spanner.Statement{
		SQL: `SELECT 
			sop_assessment_id,
			credit_enquiry_id,
			credit_enquiry_version,
			serviceability_assessment_id,
			total_monthly_net_income_amount,
			total_annual_gross_income,
			total_savings_amount,
			total_number_of_continuing_home_loans
		FROM sop 
		WHERE credit_enquiry_id = @creditEnquiryID 
		AND credit_enquiry_version = @version`,
		Params: map[string]interface{}{
			"creditEnquiryID": creditEnquiryID,
			"version":         version,
		},
	}

	var sop entity.Sop
	var continuingLoans *int64

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("sop not found")
	}
	if err != nil {
		return nil, err
	}

	err = row.Columns(
		&sop.SopAssessmentID,
		&sop.CreditEnquiryID,
		&sop.CreditEnquiryVersion,
		&sop.ServiceabilityAssessmentID,
		&sop.TotalMonthlyNetIncomeAmount,
		&sop.TotalAnnualGrossIncome,
		&sop.TotalSavingsAmount,
		&continuingLoans,
	)
	if err != nil {
		return nil, err
	}

	if continuingLoans != nil {
		val := int32(*continuingLoans)
		sop.TotalNumberOfContinuingHomeLoans = &val
	}

	return &sop, nil
}
