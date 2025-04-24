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

	"go-loan-service-v3/internal/credit_enquiry/interfaces"
	pb "go-loan-service-v3/proto"
)

type SpannerTestSuite struct {
	suite.Suite
	emulatorContainer testcontainers.Container
	spannerClient     *spanner.Client
	repository        interfaces.CreditEnquiryRepository
	projectID         string
	instanceID        string
	databaseID        string
}

func (s *SpannerTestSuite) SetupSuite() {
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
			`CREATE TABLE credit_enquiry (
				credit_enquiry_id STRING(36) NOT NULL,
				credit_enquiry_version STRING(36) NOT NULL,
				enquiry_state STRING(50),
				application_number STRING(50),
				created_time TIMESTAMP NOT NULL,
				updated_time TIMESTAMP,
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
	s.repository = NewSpannerRepository(client)
}

func (s *SpannerTestSuite) TearDownSuite() {
	ctx := context.Background()
	if s.spannerClient != nil {
		s.spannerClient.Close()
	}
	if s.emulatorContainer != nil {
		s.Require().NoError(s.emulatorContainer.Terminate(ctx))
	}
}

func (s *SpannerTestSuite) TestSaveCreditEnquiry() {
	ctx := context.Background()
	request := &pb.CreditEnquiryRequest{
		RequestId:                        []byte("test-request-id"),
		EnquiryState:                     "NEW",
		ApplicationNumber:                "APP123",
		LoanAmount:                       100000,
		LoanPurpose:                      "Home Purchase",
		InitialStructureTermMonth:        360,
		TotalMonthlyNetIncomeAmount:      5000,
		TotalAnnualGrossIncome:           60000,
		TotalSavingsAmount:               10000,
		TotalNumberOfContinuingHomeLoans: 0,
	}

	// Test saving
	err := s.repository.SaveCreditEnquiry(ctx, request)
	s.Require().NoError(err, "Failed to save credit enquiry")

	// Verify data exists in Spanner
	stmt := spanner.Statement{
		SQL: `SELECT COUNT(*) FROM credit_enquiry WHERE credit_enquiry_id = @requestID`,
		Params: map[string]interface{}{
			"requestID": string(request.RequestId),
		},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	s.Require().NoError(err, "Failed to query credit enquiry")
	var count int64
	err = row.Column(0, &count)
	s.Require().NoError(err, "Failed to get count")
	s.Equal(int64(1), count, "Expected one record in Spanner")

	// Test retrieving
	retrieved, err := s.repository.GetCreditEnquiry(ctx, string(request.RequestId), "1.0")
	s.Require().NoError(err, "Failed to get credit enquiry")
	s.Require().NotNil(retrieved)
	s.Equal("NEW", retrieved.EnquiryState)
	s.Equal(request.ApplicationNumber, retrieved.ApplicationNumber)
}

func (s *SpannerTestSuite) TestGetCreditEnquiry() {
	ctx := context.Background()
	requestID := "existing-request-id"
	request := &pb.CreditEnquiryRequest{
		RequestId:                        []byte(requestID),
		EnquiryState:                     "NEW",
		ApplicationNumber:                "APP456",
		LoanAmount:                       200000,
		LoanPurpose:                      "Home Purchase",
		InitialStructureTermMonth:        360,
		TotalMonthlyNetIncomeAmount:      6000,
		TotalAnnualGrossIncome:           72000,
		TotalSavingsAmount:               20000,
		TotalNumberOfContinuingHomeLoans: 0,
	}

	// Save request first
	err := s.repository.SaveCreditEnquiry(ctx, request)
	s.Require().NoError(err, "Failed to save credit enquiry")

	// Verify data exists in Spanner
	stmt := spanner.Statement{
		SQL: `SELECT COUNT(*) FROM credit_enquiry WHERE credit_enquiry_id = @requestID`,
		Params: map[string]interface{}{
			"requestID": requestID,
		},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	s.Require().NoError(err, "Failed to query credit enquiry")
	var count int64
	err = row.Column(0, &count)
	s.Require().NoError(err, "Failed to get count")
	s.Equal(int64(1), count, "Expected one record in Spanner")

	// Test retrieving
	retrieved, err := s.repository.GetCreditEnquiry(ctx, requestID, "1.0")
	s.Require().NoError(err, "Failed to get credit enquiry")
	s.Require().NotNil(retrieved)
	s.Equal("NEW", retrieved.EnquiryState)
	s.Equal(request.ApplicationNumber, retrieved.ApplicationNumber)
}

func (s *SpannerTestSuite) TestErrorHandling() {
	ctx := context.Background()

	// Test non-existent request
	retrieved, err := s.repository.GetCreditEnquiry(ctx, "non-existent-id", "1.0")
	s.Require().Error(err, "Expected error for non-existent request")
	s.Equal("credit enquiry not found", err.Error())
	s.Nil(retrieved)

	// Test invalid request
	invalidRequest := &pb.CreditEnquiryRequest{
		RequestId: []byte(""), // Empty request ID
	}
	err = s.repository.SaveCreditEnquiry(ctx, invalidRequest)
	s.Require().NoError(err, "Empty request ID should be allowed")
}

func TestSpannerSuite(t *testing.T) {
	suite.Run(t, new(SpannerTestSuite))
}
