package entity

import "time"

// ProductType represents the type of product
type ProductType string

const (
	ProductTypeOwner    ProductType = "OWNER"
	ProductTypeInvester ProductType = "INVESTER"
)

// InitialStructureRepaymentType represents the type of repayment structure
type InitialStructureRepaymentType string

const (
	RepaymentTypePrincipalAndInterest InitialStructureRepaymentType = "PRINCIPLE AND INTEREST"
	RepaymentTypeInterestOnly         InitialStructureRepaymentType = "INTEREST ONLY"
)

// ProductAssessment represents the product assessment entity
type ProductAssessment struct {
	ProductAssessmentID           string
	ServiceabilityAssessmentID    string
	CreditEnquiryID               string
	CreditEnquiryVersion          string
	ProductCode                   string
	ProductName                   string
	ProductType                   ProductType
	LoanAmount                    float64
	LoanPurpose                   string
	InitialStructureTermMonth     int64
	InitialStructureIndexRate     int64
	InitialStructureRepaymentType InitialStructureRepaymentType
	CreatedAt                     time.Time
	UpdatedAt                     time.Time
}
