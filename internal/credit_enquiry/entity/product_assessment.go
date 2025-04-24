package entity

import "time"

// ProductAssessment represents the product assessment entity
type ProductAssessment struct {
	ProductAssessmentID        string
	ServiceabilityAssessmentID string
	CreditEnquiryID            string
	CreditEnquiryVersion       string
	ProductCode                string
	ProductName                string
	LoanAmount                 float64
	LoanPurpose                string
	InitialStructureTermMonth  int64
	InitialStructureIndexRate  int64
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}
