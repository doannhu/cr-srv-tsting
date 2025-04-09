package credit_enquiry

import (
	"errors"
	"fmt"
	"log"

	"go-loan-service-v3/proto"

	"github.com/google/uuid"
)

const (
	maxEnquiryStateLength      = 40
	maxApplicationNumberLength = 37
	maxLoanPurposeLength       = 50
	maxLoanAmount              = 999999999
)

// Validator interface defines the contract for request validation
type Validator interface {
	ValidateRequest(request *proto.CreditEnquiryRequest) error
}

type validator struct {
	logger *log.Logger
}

func NewValidator(logger *log.Logger) Validator {
	return &validator{
		logger: logger,
	}
}

func (v *validator) ValidateRequest(req *proto.CreditEnquiryRequest) error {
	if err := v.validateUUID(req.RequestId); err != nil {
		v.logger.Printf("UUID validation failed: %v", err)
		return err
	}

	if err := v.validateStringLengths(req); err != nil {
		v.logger.Printf("String length validation failed: %v", err)
		return err
	}

	if err := v.validateNumericFields(req); err != nil {
		v.logger.Printf("Numeric field validation failed: %v", err)
		return err
	}

	return nil
}

func (v *validator) validateUUID(id []byte) error {
	if len(id) != 16 {
		return errors.New("invalid UUID length")
	}

	parsedUUID, err := uuid.FromBytes(id)
	if err != nil {
		return fmt.Errorf("invalid UUID format: %w", err)
	}

	if parsedUUID.Version() != 4 {
		return errors.New("UUID must be version 4")
	}

	return nil
}

func (v *validator) validateStringLengths(req *proto.CreditEnquiryRequest) error {
	if len(req.EnquiryState) > maxEnquiryStateLength {
		return fmt.Errorf("enquiry state exceeds maximum length of %d", maxEnquiryStateLength)
	}

	if len(req.ApplicationNumber) > maxApplicationNumberLength {
		return fmt.Errorf("application number exceeds maximum length of %d", maxApplicationNumberLength)
	}

	if len(req.LoanPurpose) > maxLoanPurposeLength {
		return fmt.Errorf("loan purpose exceeds maximum length of %d", maxLoanPurposeLength)
	}

	return nil
}

func (v *validator) validateNumericFields(req *proto.CreditEnquiryRequest) error {
	if req.LoanAmount <= 0 || req.LoanAmount > maxLoanAmount {
		return fmt.Errorf("loan amount must be positive and less than %d", maxLoanAmount)
	}

	if req.InitialStructureTermMonth <= 0 {
		return errors.New("initial structure term month must be positive")
	}

	if req.TotalMonthlyNetIncomeAmount <= 0 {
		return errors.New("total monthly net income amount must be positive")
	}

	if req.TotalAnnualGrossIncome <= 0 {
		return errors.New("total annual gross income must be positive")
	}

	if req.TotalSavingsAmount < 0 {
		return errors.New("total savings amount cannot be negative")
	}

	if req.TotalNumberOfContinuingHomeLoans < 0 {
		return errors.New("total number of continuing home loans cannot be negative")
	}

	return nil
}
