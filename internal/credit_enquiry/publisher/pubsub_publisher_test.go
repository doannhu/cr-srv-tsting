package publisher

import (
	"context"
	"os"
	"testing"
	"time"

	creditEnquiryProto "go-loan-service-v3/proto/credit_enquiry"
	creditEnquiryEventProto "go-loan-service-v3/proto/credit_enquiry_event"

	"cloud.google.com/go/pubsub"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
)

type pubsubPublisherTestSuite struct {
	suite.Suite
	publisher    *pubsubPublisher
	ctx          context.Context
	emulatorHost string
	client       *pubsub.Client
	topic        *pubsub.Topic
	projectID    string
	topicID      string
}

func TestPubSubPublisherSuite(t *testing.T) {
	suite.Run(t, new(pubsubPublisherTestSuite))
}

func (s *pubsubPublisherTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.emulatorHost = os.Getenv("PUBSUB_EMULATOR_HOST")
	if s.emulatorHost == "" {
		s.T().Skip("PUBSUB_EMULATOR_HOST not set, skipping integration tests")
	}

	s.projectID = "test-project"
	s.topicID = "test-topic-" + time.Now().Format("20060102150405")

	var err error
	s.client, err = pubsub.NewClient(s.ctx, s.projectID)
	s.Require().NoError(err)

	// Create test topic
	s.topic, err = s.client.CreateTopic(s.ctx, s.topicID)
	s.Require().NoError(err)

	// Create publisher
	publisher, err := NewPubSubPublisher(s.ctx, s.projectID, s.topicID)
	s.Require().NoError(err)
	s.publisher = publisher.(*pubsubPublisher)
}

func (s *pubsubPublisherTestSuite) TearDownSuite() {
	if s.topic != nil {
		s.topic.Delete(s.ctx)
	}
	if s.client != nil {
		s.client.Close()
	}
}

func (s *pubsubPublisherTestSuite) TestPublishCreditEnquiryEvent() {
	// Create a subscription to verify the message
	subID := "test-sub-" + time.Now().Format("20060102150405")
	sub, err := s.client.CreateSubscription(s.ctx, subID, pubsub.SubscriptionConfig{
		Topic: s.topic,
	})
	s.Require().NoError(err)
	defer sub.Delete(s.ctx)

	// Create a test request
	requestID := uuid.New()
	request := &creditEnquiryProto.CreditEnquiryRequest{
		RequestId:                        requestID[:],
		ApplicationNumber:                "APP123",
		EnquiryState:                     "NEW",
		LoanAmount:                       100000,
		LoanPurpose:                      "HOME",
		InitialStructureTermMonth:        360,
		TotalMonthlyNetIncomeAmount:      5000,
		TotalAnnualGrossIncome:           60000,
		TotalSavingsAmount:               10000,
		TotalNumberOfContinuingHomeLoans: 1,
	}

	// Publish the event
	err = s.publisher.PublishCreditEnquiryEvent(s.ctx, request)
	s.Require().NoError(err)

	// Verify the message was received
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	received := false
	err = sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		// Unmarshal the event
		var event creditEnquiryEventProto.CreditEnquiryEvent
		err := proto.Unmarshal(msg.Data, &event)
		s.Require().NoError(err)

		// Verify event data
		s.Equal(requestID.String(), event.Data.CreditEnquiryId)
		s.Equal(request.ApplicationNumber, event.Data.ApplicationNumber)
		s.Equal(request.EnquiryState, event.Data.EnquiryState)
		s.Equal(request.LoanAmount, event.Data.LoanAmount)
		s.Equal(request.LoanPurpose, event.Data.LoanPurpose)
		s.Equal(request.InitialStructureTermMonth, event.Data.InitialStructureTermMonth)
		s.Equal(request.TotalMonthlyNetIncomeAmount, event.Data.TotalMonthlyNetIncomeAmount)
		s.Equal(request.TotalAnnualGrossIncome, event.Data.TotalAnnualGrossIncome)
		s.Equal(request.TotalSavingsAmount, event.Data.TotalSavingsAmount)
		s.Equal(request.TotalNumberOfContinuingHomeLoans, event.Data.TotalNumberOfContinuingHomeLoans)

		received = true
		msg.Ack()
	})
	s.Require().NoError(err)
	s.True(received, "Message was not received")
}

func (s *pubsubPublisherTestSuite) TestPublishCreditEnquiryEventWithInvalidRequest() {
	// Create a request with invalid UUID
	request := &creditEnquiryProto.CreditEnquiryRequest{
		RequestId: []byte("invalid-uuid"),
	}

	// Publish the event
	err := s.publisher.PublishCreditEnquiryEvent(s.ctx, request)
	s.Require().Error(err)
}

func (s *pubsubPublisherTestSuite) TestPublishCreditEnquiryEventWithDeletedTopic() {
	// Delete the topic
	err := s.topic.Delete(s.ctx)
	s.Require().NoError(err)

	// Create a test request
	requestID := uuid.New()
	request := &creditEnquiryProto.CreditEnquiryRequest{
		RequestId: requestID[:],
	}

	// Publish the event
	err = s.publisher.PublishCreditEnquiryEvent(s.ctx, request)
	s.Require().Error(err)
}
