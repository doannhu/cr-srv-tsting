package credit_enquiry

import (
	"fmt"
	"log"

	"go-loan-service-v3/internal/credit_enquiry/errors"
	"go-loan-service-v3/internal/credit_enquiry/interfaces"
	proto "go-loan-service-v3/proto/credit_enquiry"

	"github.com/google/uuid"
)

const (
	maxEnquiryStateLength      = 40
	maxApplicationNumberLength = 37
	maxLoanPurposeLength       = 50
	maxLoanAmount              = 999999999
)

type validator struct {
	logger *log.Logger
}

func NewValidator(logger *log.Logger) interfaces.Validator {
	return &validator{
		logger: logger,
	}
}

func (v *validator) ValidateRequest(req *proto.CreditEnquiryRequest) (*errors.ValidationResponse, error) {
	v.logger.Printf("Starting validation for request: %+v", req)

	// Validate UUID
	if err := v.validateUUID(req.RequestId); err != nil {
		v.logger.Printf("UUID validation failed: %v", err)
		return errors.NewValidationResponse(false, err, nil), nil
	}

	// Validate string lengths
	if err := v.validateStringLengths(req); err != nil {
		v.logger.Printf("String length validation failed: %v", err)
		return errors.NewValidationResponse(false, err, nil), nil
	}

	// Validate numeric fields
	if err := v.validateNumericFields(req); err != nil {
		v.logger.Printf("Numeric field validation failed: %v", err)
		return errors.NewValidationResponse(false, err, nil), nil
	}

	v.logger.Printf("Validation successful for request: %+v", req)
	return errors.NewValidationResponse(true, nil, nil), nil
}

func (v *validator) validateUUID(id []byte) *errors.ValidationError {
	if len(id) == 0 {
		return errors.NewValidationError(
			errors.ErrRequiredField,
			"request_id is required",
			map[string]interface{}{
				"field": "request_id",
			},
		)
	}

	if len(id) != 16 {
		return errors.NewValidationError(
			errors.ErrInvalidUUID,
			"invalid UUID length",
			map[string]interface{}{
				"field":    "request_id",
				"expected": 16,
				"actual":   len(id),
			},
		)
	}

	parsedUUID, err := uuid.FromBytes(id)
	if err != nil {
		return errors.NewValidationError(
			errors.ErrInvalidUUID,
			"invalid UUID format",
			map[string]interface{}{
				"field": "request_id",
				"error": err.Error(),
			},
		)
	}

	if parsedUUID.Version() != 4 {
		return errors.NewValidationError(
			errors.ErrInvalidUUID,
			"UUID must be version 4",
			map[string]interface{}{
				"field":   "request_id",
				"version": parsedUUID.Version(),
			},
		)
	}

	return nil
}

func (v *validator) validateStringLengths(req *proto.CreditEnquiryRequest) *errors.ValidationError {
	// Check for empty required fields
	if req.EnquiryState == "" {
		return errors.NewValidationError(
			errors.ErrBadRequest,
			"enquiry state is required",
			map[string]interface{}{
				"field": "enquiry_state",
			},
		)
	}

	if req.ApplicationNumber == "" {
		return errors.NewValidationError(
			errors.ErrBadRequest,
			"application number is required",
			map[string]interface{}{
				"field": "application_number",
			},
		)
	}

	if req.LoanPurpose == "" {
		return errors.NewValidationError(
			errors.ErrBadRequest,
			"loan purpose is required",
			map[string]interface{}{
				"field": "loan_purpose",
			},
		)
	}

	// Check for maximum lengths
	if len(req.EnquiryState) > maxEnquiryStateLength {
		return errors.NewValidationError(
			errors.ErrInvalidLength,
			fmt.Sprintf("enquiry state exceeds maximum length of %d", maxEnquiryStateLength),
			map[string]interface{}{
				"field":  "enquiry_state",
				"max":    maxEnquiryStateLength,
				"actual": len(req.EnquiryState),
			},
		)
	}

	if len(req.ApplicationNumber) > maxApplicationNumberLength {
		return errors.NewValidationError(
			errors.ErrInvalidLength,
			fmt.Sprintf("application number exceeds maximum length of %d", maxApplicationNumberLength),
			map[string]interface{}{
				"field":  "application_number",
				"max":    maxApplicationNumberLength,
				"actual": len(req.ApplicationNumber),
			},
		)
	}

	if len(req.LoanPurpose) > maxLoanPurposeLength {
		return errors.NewValidationError(
			errors.ErrInvalidLength,
			fmt.Sprintf("loan purpose exceeds maximum length of %d", maxLoanPurposeLength),
			map[string]interface{}{
				"field":  "loan_purpose",
				"max":    maxLoanPurposeLength,
				"actual": len(req.LoanPurpose),
			},
		)
	}

	return nil
}

func (v *validator) validateNumericFields(req *proto.CreditEnquiryRequest) *errors.ValidationError {
	if req.LoanAmount <= 0 || req.LoanAmount > maxLoanAmount {
		return errors.NewValidationError(
			errors.ErrInvalidNumeric,
			fmt.Sprintf("loan amount must be positive and less than %d", maxLoanAmount),
			map[string]interface{}{
				"field":  "loan_amount",
				"min":    0,
				"max":    maxLoanAmount,
				"actual": req.LoanAmount,
			},
		)
	}

	if req.InitialStructureTermMonth <= 0 {
		return errors.NewValidationError(
			errors.ErrInvalidNumeric,
			"initial structure term month must be positive",
			map[string]interface{}{
				"field":  "initial_structure_term_month",
				"actual": req.InitialStructureTermMonth,
			},
		)
	}

	if req.TotalMonthlyNetIncomeAmount <= 0 {
		return errors.NewValidationError(
			errors.ErrInvalidNumeric,
			"total monthly net income amount must be positive",
			map[string]interface{}{
				"field":  "total_monthly_net_income_amount",
				"actual": req.TotalMonthlyNetIncomeAmount,
			},
		)
	}

	if req.TotalAnnualGrossIncome <= 0 {
		return errors.NewValidationError(
			errors.ErrInvalidNumeric,
			"total annual gross income must be positive",
			map[string]interface{}{
				"field":  "total_annual_gross_income",
				"actual": req.TotalAnnualGrossIncome,
			},
		)
	}

	if req.TotalSavingsAmount < 0 {
		return errors.NewValidationError(
			errors.ErrInvalidNumeric,
			"total savings amount cannot be negative",
			map[string]interface{}{
				"field":  "total_savings_amount",
				"actual": req.TotalSavingsAmount,
			},
		)
	}

	if req.TotalNumberOfContinuingHomeLoans < 0 {
		return errors.NewValidationError(
			errors.ErrInvalidNumeric,
			"total number of continuing home loans cannot be negative",
			map[string]interface{}{
				"field":  "total_number_of_continuing_home_loans",
				"actual": req.TotalNumberOfContinuingHomeLoans,
			},
		)
	}

	return nil
}
