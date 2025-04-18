package sop

import (
	"context"
	"fmt"
	"os"
	"testing"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/api/option"
	databasepb "google.golang.org/genproto/googleapis/spanner/admin/database/v1"
	instancepb "google.golang.org/genproto/googleapis/spanner/admin/instance/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"go-loan-service-v3/internal/credit_enquiry/entity"
	"go-loan-service-v3/internal/credit_enquiry/interfaces"
)

// Helper functions for creating test data
func float64Ptr(v float64) *float64 { return &v }
func int32Ptr(v int32) *int32       { return &v }

type SpannerSopTestSuite struct {
	suite.Suite
	emulatorContainer testcontainers.Container
	spannerClient     *spanner.Client
	repository        interfaces.SopRepository
	projectID         string
	instanceID        string
	databaseID        string
}

func (s *SpannerSopTestSuite) SetupSuite() {
	ctx := context.Background()

	// Set up environment variables
	s.projectID = "test-project"
	s.instanceID = "test-instance"
	s.databaseID = "testdb"
	os.Setenv("SPANNER_EMULATOR_HOST", "localhost:9010")
	os.Setenv("GOOGLE_CLOUD_PROJECT", s.projectID)

	// Start Spanner emulator container
	req := testcontainers.ContainerRequest{
		Image:        "gcr.io/cloud-spanner-emulator/emulator",
		ExposedPorts: []string{"9010/tcp"},
		WaitingFor:   wait.ForLog("Cloud Spanner emulator running"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	s.Require().NoError(err)
	s.emulatorContainer = container

	// Get container host and port
	host, err := container.Host(ctx)
	s.Require().NoError(err)

	port, err := container.MappedPort(ctx, "9010")
	s.Require().NoError(err)

	// Create instance admin client
	instanceAdmin, err := instance.NewInstanceAdminClient(ctx,
		option.WithEndpoint(host+":"+port.Port()),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithoutAuthentication(),
	)
	s.Require().NoError(err)
	defer instanceAdmin.Close()

	// Create instance
	op, err := instanceAdmin.CreateInstance(ctx, &instancepb.CreateInstanceRequest{
		Parent:     fmt.Sprintf("projects/%s", s.projectID),
		InstanceId: s.instanceID,
		Instance: &instancepb.Instance{
			Config:      fmt.Sprintf("projects/%s/instanceConfigs/emulator-config", s.projectID),
			DisplayName: s.instanceID,
			NodeCount:   1,
		},
	})
	s.Require().NoError(err)
	_, err = op.Wait(ctx)
	s.Require().NoError(err)

	// Create database admin client
	databaseAdmin, err := database.NewDatabaseAdminClient(ctx,
		option.WithEndpoint(host+":"+port.Port()),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithoutAuthentication(),
	)
	s.Require().NoError(err)
	defer databaseAdmin.Close()

	// Create database with tables
	opDB, err := databaseAdmin.CreateDatabase(ctx, &databasepb.CreateDatabaseRequest{
		Parent:          fmt.Sprintf("projects/%s/instances/%s", s.projectID, s.instanceID),
		CreateStatement: fmt.Sprintf("CREATE DATABASE `%s`", s.databaseID),
		ExtraStatements: []string{
			`CREATE TABLE sop (
				sop_assessment_id STRING(64) NOT NULL,
				credit_enquiry_id STRING(64) NOT NULL,
				credit_enquiry_version STRING(64) NOT NULL,
				serviceability_assessment_id STRING(64) NOT NULL,
				total_monthly_net_income_amount FLOAT64,
				total_annual_gross_income FLOAT64,
				total_savings_amount FLOAT64,
				total_number_of_continuing_home_loans INT64,
				created_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (credit_enquiry_id, credit_enquiry_version)`,
		},
	})
	s.Require().NoError(err)
	_, err = opDB.Wait(ctx)
	s.Require().NoError(err)

	// Initialize Spanner client
	client, err := spanner.NewClient(ctx,
		fmt.Sprintf("projects/%s/instances/%s/databases/%s", s.projectID, s.instanceID, s.databaseID),
		option.WithEndpoint(host+":"+port.Port()),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithoutAuthentication(),
	)
	s.Require().NoError(err)
	s.spannerClient = client

	// Initialize repository
	s.repository = NewSpannerSopRepository(client)
}

func (s *SpannerSopTestSuite) TearDownSuite() {
	ctx := context.Background()
	if s.spannerClient != nil {
		s.spannerClient.Close()
	}
	if s.emulatorContainer != nil {
		s.Require().NoError(s.emulatorContainer.Terminate(ctx))
	}
}

func (s *SpannerSopTestSuite) TestSaveSop() {
	ctx := context.Background()

	// Test saving new SOP
	sop := &entity.Sop{
		SopAssessmentID:                  "sop-123",
		CreditEnquiryID:                  "credit-123",
		CreditEnquiryVersion:             "v1",
		ServiceabilityAssessmentID:       "sa-123",
		TotalMonthlyNetIncomeAmount:      float64Ptr(5000.0),
		TotalAnnualGrossIncome:           float64Ptr(60000.0),
		TotalSavingsAmount:               float64Ptr(10000.0),
		TotalNumberOfContinuingHomeLoans: int32Ptr(2),
	}

	err := s.repository.SaveSop(ctx, sop)
	s.Require().NoError(err, "Failed to save SOP")

	// Verify data exists in Spanner
	stmt := spanner.Statement{
		SQL: `SELECT COUNT(*) FROM sop WHERE credit_enquiry_id = @creditEnquiryID AND credit_enquiry_version = @version`,
		Params: map[string]interface{}{
			"creditEnquiryID": sop.CreditEnquiryID,
			"version":         sop.CreditEnquiryVersion,
		},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	s.Require().NoError(err, "Failed to query SOP")
	var count int64
	err = row.Column(0, &count)
	s.Require().NoError(err, "Failed to get count")
	s.Equal(int64(1), count, "Expected one record in Spanner")

	// Test saving duplicate SOP
	err = s.repository.SaveSop(ctx, sop)
	s.Require().Error(err, "Expected error for duplicate SOP")
	s.Contains(err.Error(), "sop already exists")

	// Test saving SOP with null values
	nullSop := &entity.Sop{
		SopAssessmentID:                  "sop-456",
		CreditEnquiryID:                  "credit-456",
		CreditEnquiryVersion:             "v1",
		ServiceabilityAssessmentID:       "sa-456",
		TotalMonthlyNetIncomeAmount:      nil,
		TotalAnnualGrossIncome:           nil,
		TotalSavingsAmount:               nil,
		TotalNumberOfContinuingHomeLoans: nil,
	}

	err = s.repository.SaveSop(ctx, nullSop)
	s.Require().NoError(err, "Failed to save SOP with null values")
}

func (s *SpannerSopTestSuite) TestGetSop() {
	ctx := context.Background()

	// Test getting non-existent SOP
	sop, err := s.repository.GetSop(ctx, "non-existent", "v1")
	s.Require().Error(err, "Expected error for non-existent SOP")
	s.Nil(sop)

	// Test getting existing SOP
	expectedSop := &entity.Sop{
		SopAssessmentID:                  "sop-789",
		CreditEnquiryID:                  "credit-789",
		CreditEnquiryVersion:             "v1",
		ServiceabilityAssessmentID:       "sa-789",
		TotalMonthlyNetIncomeAmount:      float64Ptr(7000.0),
		TotalAnnualGrossIncome:           float64Ptr(84000.0),
		TotalSavingsAmount:               float64Ptr(15000.0),
		TotalNumberOfContinuingHomeLoans: int32Ptr(1),
	}

	err = s.repository.SaveSop(ctx, expectedSop)
	s.Require().NoError(err, "Failed to save SOP")

	// Retrieve the SOP
	gotSop, err := s.repository.GetSop(ctx, expectedSop.CreditEnquiryID, expectedSop.CreditEnquiryVersion)
	s.Require().NoError(err, "Failed to get SOP")
	s.Require().NotNil(gotSop)
	s.Equal(expectedSop.SopAssessmentID, gotSop.SopAssessmentID)
	s.Equal(expectedSop.CreditEnquiryID, gotSop.CreditEnquiryID)
	s.Equal(expectedSop.CreditEnquiryVersion, gotSop.CreditEnquiryVersion)
	s.Equal(expectedSop.ServiceabilityAssessmentID, gotSop.ServiceabilityAssessmentID)
	s.Equal(*expectedSop.TotalMonthlyNetIncomeAmount, *gotSop.TotalMonthlyNetIncomeAmount)
	s.Equal(*expectedSop.TotalAnnualGrossIncome, *gotSop.TotalAnnualGrossIncome)
	s.Equal(*expectedSop.TotalSavingsAmount, *gotSop.TotalSavingsAmount)
	s.Equal(*expectedSop.TotalNumberOfContinuingHomeLoans, *gotSop.TotalNumberOfContinuingHomeLoans)

	// Test getting SOP with null values
	nullSop := &entity.Sop{
		SopAssessmentID:                  "sop-101",
		CreditEnquiryID:                  "credit-101",
		CreditEnquiryVersion:             "v1",
		ServiceabilityAssessmentID:       "sa-101",
		TotalMonthlyNetIncomeAmount:      nil,
		TotalAnnualGrossIncome:           nil,
		TotalSavingsAmount:               nil,
		TotalNumberOfContinuingHomeLoans: nil,
	}

	err = s.repository.SaveSop(ctx, nullSop)
	s.Require().NoError(err, "Failed to save SOP with null values")

	gotNullSop, err := s.repository.GetSop(ctx, nullSop.CreditEnquiryID, nullSop.CreditEnquiryVersion)
	s.Require().NoError(err, "Failed to get SOP with null values")
	s.Require().NotNil(gotNullSop)
	s.Equal(nullSop.SopAssessmentID, gotNullSop.SopAssessmentID)
	s.Equal(nullSop.CreditEnquiryID, gotNullSop.CreditEnquiryID)
	s.Equal(nullSop.CreditEnquiryVersion, gotNullSop.CreditEnquiryVersion)
	s.Equal(nullSop.ServiceabilityAssessmentID, gotNullSop.ServiceabilityAssessmentID)
	s.Nil(gotNullSop.TotalMonthlyNetIncomeAmount)
	s.Nil(gotNullSop.TotalAnnualGrossIncome)
	s.Nil(gotNullSop.TotalSavingsAmount)
	s.Nil(gotNullSop.TotalNumberOfContinuingHomeLoans)
}

func TestSpannerSopSuite(t *testing.T) {
	suite.Run(t, new(SpannerSopTestSuite))
}
