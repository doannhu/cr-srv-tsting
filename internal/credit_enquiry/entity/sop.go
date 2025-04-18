package entity

// Sop represents a Statement of Position
type Sop struct {
	SopAssessmentID                  string
	CreditEnquiryID                  string
	CreditEnquiryVersion             string
	ServiceabilityAssessmentID       string
	TotalMonthlyNetIncomeAmount      *float64
	TotalAnnualGrossIncome           *float64
	TotalSavingsAmount               *float64
	TotalNumberOfContinuingHomeLoans *int32
}
