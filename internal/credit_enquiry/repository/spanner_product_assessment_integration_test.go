package repository

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

type SpannerProductAssessmentTestSuite struct {
	suite.Suite
	emulatorContainer testcontainers.Container
	spannerClient     *spanner.Client
	repository        interfaces.ProductAssessmentRepository
	projectID         string
	instanceID        string
	databaseID        string
}

func (s *SpannerProductAssessmentTestSuite) SetupSuite() {
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
			`CREATE TABLE product_assessment (
				product_assessment_id STRING(37) NOT NULL,
				serviceability_assessment_id STRING(37),
				credit_enquiry_id STRING(37) NOT NULL,
				credit_enquiry_version STRING(50) NOT NULL,
				product_code STRING(20),
				product_name STRING(250),
				loan_amount NUMERIC,
				loan_purpose STRING(50),
				initial_structure_term_month INT64,
				initial_structure_index_rate INT64,
				created_at TIMESTAMP,
				updated_at TIMESTAMP,
			) PRIMARY KEY (product_assessment_id)`,
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
	s.repository = NewSpannerProductAssessmentRepository(client)
}

func (s *SpannerProductAssessmentTestSuite) TearDownSuite() {
	ctx := context.Background()
	if s.spannerClient != nil {
		s.spannerClient.Close()
	}
	if s.emulatorContainer != nil {
		s.Require().NoError(s.emulatorContainer.Terminate(ctx))
	}
}

func (s *SpannerProductAssessmentTestSuite) TestSaveProductAssessment() {
	ctx := context.Background()
	assessment := &entity.ProductAssessment{
		ProductAssessmentID:        "test-assessment-id",
		ServiceabilityAssessmentID: "test-serviceability-id",
		CreditEnquiryID:            "test-credit-enquiry-id",
		CreditEnquiryVersion:       "1.0",
		ProductCode:                "PROD001",
		ProductName:                "Test Product",
		LoanAmount:                 100000,
		LoanPurpose:                "Home Purchase",
		InitialStructureTermMonth:  360,
		InitialStructureIndexRate:  5,
	}

	// Test saving
	err := s.repository.SaveProductAssessment(ctx, assessment)
	s.Require().NoError(err, "Failed to save product assessment")

	// Verify data exists in Spanner
	stmt := spanner.Statement{
		SQL: `SELECT COUNT(*) FROM product_assessment WHERE product_assessment_id = @assessmentID`,
		Params: map[string]interface{}{
			"assessmentID": assessment.ProductAssessmentID,
		},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	s.Require().NoError(err, "Failed to query product assessment")
	var count int64
	err = row.Column(0, &count)
	s.Require().NoError(err, "Failed to get count")
	s.Equal(int64(1), count, "Expected one record in Spanner")

	// Test retrieving
	retrieved, err := s.repository.GetProductAssessment(ctx, assessment.CreditEnquiryID, assessment.CreditEnquiryVersion)
	s.Require().NoError(err, "Failed to get product assessment")
	s.Require().NotNil(retrieved)
	s.Equal(assessment.ProductCode, retrieved.ProductCode)
	s.Equal(assessment.ProductName, retrieved.ProductName)
	s.Equal(assessment.LoanAmount, retrieved.LoanAmount)
	s.Equal(assessment.LoanPurpose, retrieved.LoanPurpose)
	s.Equal(assessment.InitialStructureTermMonth, retrieved.InitialStructureTermMonth)
	s.Equal(assessment.InitialStructureIndexRate, retrieved.InitialStructureIndexRate)
}

func (s *SpannerProductAssessmentTestSuite) TestGetProductAssessment() {
	ctx := context.Background()
	assessment := &entity.ProductAssessment{
		ProductAssessmentID:        "existing-assessment-id",
		ServiceabilityAssessmentID: "existing-serviceability-id",
		CreditEnquiryID:            "existing-credit-enquiry-id",
		CreditEnquiryVersion:       "1.0",
		ProductCode:                "PROD002",
		ProductName:                "Existing Product",
		LoanAmount:                 200000,
		LoanPurpose:                "Home Purchase",
		InitialStructureTermMonth:  360,
		InitialStructureIndexRate:  6,
	}

	// Save test data
	err := s.repository.SaveProductAssessment(ctx, assessment)
	s.Require().NoError(err, "Failed to save test data")

	// Test retrieving
	retrieved, err := s.repository.GetProductAssessment(ctx, assessment.CreditEnquiryID, assessment.CreditEnquiryVersion)
	s.Require().NoError(err, "Failed to get product assessment")
	s.Require().NotNil(retrieved)
	s.Equal(assessment.ProductCode, retrieved.ProductCode)
	s.Equal(assessment.ProductName, retrieved.ProductName)
	s.Equal(assessment.LoanAmount, retrieved.LoanAmount)
	s.Equal(assessment.LoanPurpose, retrieved.LoanPurpose)
	s.Equal(assessment.InitialStructureTermMonth, retrieved.InitialStructureTermMonth)
	s.Equal(assessment.InitialStructureIndexRate, retrieved.InitialStructureIndexRate)
}

func (s *SpannerProductAssessmentTestSuite) TestGetProductAssessmentNotFound() {
	ctx := context.Background()
	_, err := s.repository.GetProductAssessment(ctx, "non-existent-id", "1.0")
	s.Require().Error(err, "Expected error for non-existent record")
}

func TestSpannerProductAssessmentSuite(t *testing.T) {
	suite.Run(t, new(SpannerProductAssessmentTestSuite))
}
