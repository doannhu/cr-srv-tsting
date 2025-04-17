package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-loan-service-v3/internal/credit_enquiry/interfaces"
	"go-loan-service-v3/internal/credit_enquiry/utils"
	creditEnquiryProto "go-loan-service-v3/proto"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type spannerRepository struct {
	client *spanner.Client
	config *utils.RetryConfig
}

// NewSpannerRepository creates a new Spanner repository instance
func NewSpannerRepository(client *spanner.Client) interfaces.CreditEnquiryRepository {
	return &spannerRepository{
		client: client,
		config: utils.DefaultRetryConfig(),
	}
}

// SaveCreditEnquiry implements CreditEnquiryRepository
func (r *spannerRepository) SaveCreditEnquiry(ctx context.Context, request *creditEnquiryProto.CreditEnquiryRequest) error {
	// Convert request ID bytes to string
	requestID := string(request.RequestId)
	version := "1.0" // Fixed version as per requirement

	// Check if record already exists
	checkExists := func() error {
		stmt := spanner.Statement{
			SQL: `SELECT credit_enquiry_id, credit_enquiry_version 
                  FROM credit_enquiry 
                  WHERE credit_enquiry_id = @requestID 
                  AND credit_enquiry_version = @version`,
			Params: map[string]interface{}{
				"requestID": requestID,
				"version":   version,
			},
		}

		iter := r.client.Single().Query(ctx, stmt)
		defer iter.Stop()

		_, err := iter.Next()
		if err != nil && err != iterator.Done {
			return fmt.Errorf("failed to check existing record: %w", err)
		}

		if err != iterator.Done {
			return errors.New("credit enquiry already exists")
		}

		return nil
	}

	// Execute check with retry
	if err := utils.Retry(ctx, r.config, "check existing record", checkExists); err != nil {
		return err
	}

	// Get current time for created_time
	now := time.Now()

	// Prepare mutation for insert
	m := []*spanner.Mutation{
		spanner.Insert(
			"credit_enquiry",
			[]string{
				"credit_enquiry_id",
				"credit_enquiry_version",
				"enquiry_state",
				"application_number",
				"created_time",
			},
			[]interface{}{
				requestID,
				version,
				"NEW", // Fixed state as per requirement
				request.ApplicationNumber,
				now,
			},
		),
	}

	// Execute the transaction with retry
	saveOperation := func() error {
		_, err := r.client.Apply(ctx, m)
		if err != nil {
			return fmt.Errorf("failed to save credit enquiry: %w", err)
		}
		return nil
	}

	return utils.Retry(ctx, r.config, "save credit enquiry", saveOperation)
}

// GetCreditEnquiry implements CreditEnquiryRepository
func (r *spannerRepository) GetCreditEnquiry(ctx context.Context, requestID string, version string) (*creditEnquiryProto.CreditEnquiryRequest, error) {
	var result *creditEnquiryProto.CreditEnquiryRequest

	getOperation := func() error {
		stmt := spanner.Statement{
			SQL: `SELECT credit_enquiry_id, credit_enquiry_version, enquiry_state, 
                     application_number, created_time
                  FROM credit_enquiry 
                  WHERE credit_enquiry_id = @requestID 
                  AND credit_enquiry_version = @version`,
			Params: map[string]interface{}{
				"requestID": requestID,
				"version":   version,
			},
		}

		iter := r.client.Single().Query(ctx, stmt)
		defer iter.Stop()

		row, err := iter.Next()
		if err == iterator.Done {
			return errors.New("credit enquiry not found")
		}
		if err != nil {
			return fmt.Errorf("failed to get credit enquiry: %w", err)
		}

		var (
			enquiryID      string
			enquiryVersion string
			enquiryState   string
			appNumber      string
			createdTime    time.Time
		)

		if err := row.Columns(
			&enquiryID,
			&enquiryVersion,
			&enquiryState,
			&appNumber,
			&createdTime,
		); err != nil {
			return fmt.Errorf("failed to parse credit enquiry: %w", err)
		}

		result = &creditEnquiryProto.CreditEnquiryRequest{
			RequestId:         []byte(enquiryID),
			EnquiryState:      enquiryState,
			ApplicationNumber: appNumber,
		}

		return nil
	}

	if err := utils.Retry(ctx, r.config, "get credit enquiry", getOperation); err != nil {
		return nil, err
	}

	return result, nil
}

// SaveRequest implements CreditEnquiryRepository
func (r *spannerRepository) SaveRequest(req *creditEnquiryProto.CreditEnquiryRequest) error {
	// For Spanner, we'll just call SaveCreditEnquiry with a background context
	return r.SaveCreditEnquiry(context.Background(), req)
}
