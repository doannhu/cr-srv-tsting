package credit_enquiry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-loan-service-v3/internal/credit_enquiry/interfaces"
	creditEnquiryProto "go-loan-service-v3/proto/credit_enquiry"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type spannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository creates a new Spanner repository instance
func NewSpannerRepository(client *spanner.Client) interfaces.CreditEnquiryRepository {
	return &spannerRepository{
		client: client,
	}
}

// SaveCreditEnquiry implements CreditEnquiryRepository
func (r *spannerRepository) SaveCreditEnquiry(ctx context.Context, request *creditEnquiryProto.CreditEnquiryRequest) error {
	// Convert request ID bytes to string
	requestID := string(request.RequestId)
	version := "1.0" // Fixed version as per requirement

	// Check if record already exists
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

	// Execute the transaction
	_, err = r.client.Apply(ctx, m)
	if err != nil {
		return fmt.Errorf("failed to save credit enquiry: %w", err)
	}
	return nil
}

// GetCreditEnquiry implements CreditEnquiryRepository
func (r *spannerRepository) GetCreditEnquiry(ctx context.Context, requestID string, version string) (*creditEnquiryProto.CreditEnquiryRequest, error) {
	var result *creditEnquiryProto.CreditEnquiryRequest

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
		return nil, errors.New("credit enquiry not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get credit enquiry: %w", err)
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
		return nil, fmt.Errorf("failed to parse credit enquiry: %w", err)
	}

	result = &creditEnquiryProto.CreditEnquiryRequest{
		RequestId:         []byte(enquiryID),
		EnquiryState:      enquiryState,
		ApplicationNumber: appNumber,
	}

	return result, nil
}

// SaveRequest implements CreditEnquiryRepository
func (r *spannerRepository) SaveRequest(req *creditEnquiryProto.CreditEnquiryRequest) error {
	// For Spanner, we'll just call SaveCreditEnquiry with a background context
	return r.SaveCreditEnquiry(context.Background(), req)
}
