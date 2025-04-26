package publisher

import (
	"context"
	"time"

	"go-loan-service-v3/internal/credit_enquiry/interfaces"
	creditEnquiryProto "go-loan-service-v3/proto/credit_enquiry"
	creditEnquiryEventProto "go-loan-service-v3/proto/credit_enquiry_event"

	"cloud.google.com/go/pubsub"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type pubsubPublisher struct {
	client    *pubsub.Client
	topic     *pubsub.Topic
	projectID string
	topicID   string
}

// NewPubSubPublisher creates a new Pub/Sub publisher
func NewPubSubPublisher(ctx context.Context, projectID, topicID string) (interfaces.CreditEnquiryPublisher, error) {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	topic := client.Topic(topicID)
	exists, err := topic.Exists(ctx)
	if err != nil {
		return nil, err
	}

	if !exists {
		topic, err = client.CreateTopic(ctx, topicID)
		if err != nil {
			return nil, err
		}
	}

	return &pubsubPublisher{
		client:    client,
		topic:     topic,
		projectID: projectID,
		topicID:   topicID,
	}, nil
}

// PublishCreditEnquiryEvent implements Publisher interface
func (p *pubsubPublisher) PublishCreditEnquiryEvent(ctx context.Context, request *creditEnquiryProto.CreditEnquiryRequest) error {
	// Convert request ID bytes to UUID
	requestUUID, err := uuid.FromBytes(request.RequestId)
	if err != nil {
		return err
	}

	// Create event data
	eventData := &creditEnquiryEventProto.CreditEnquiryEventData{
		CreditEnquiryId:                  requestUUID.String(),
		ApplicationNumber:                request.ApplicationNumber,
		EnquiryState:                     request.EnquiryState,
		LoanAmount:                       request.LoanAmount,
		LoanPurpose:                      request.LoanPurpose,
		InitialStructureTermMonth:        request.InitialStructureTermMonth,
		TotalMonthlyNetIncomeAmount:      request.TotalMonthlyNetIncomeAmount,
		TotalAnnualGrossIncome:           request.TotalAnnualGrossIncome,
		TotalSavingsAmount:               request.TotalSavingsAmount,
		TotalNumberOfContinuingHomeLoans: request.TotalNumberOfContinuingHomeLoans,
	}

	// Create CloudEvent
	event := &creditEnquiryEventProto.CreditEnquiryEvent{
		Id:              uuid.New().String(),
		Source:          "enquiries.gearbox.anzx",
		SpecVersion:     "1.0",
		Type:            "com.anzx.gearbox.enquiries.v1beta2.creditenquiry.created",
		DataContentType: "application/protobuf",
		DataSchema:      "proto://buf.anz/...#CreditEnquiryCreatedEvent",
		Subject:         "credit_enquiry/" + requestUUID.String(),
		Time:            timestamppb.New(time.Now()),
		Data:            eventData,
	}

	// Validate event
	if err := event.Validate(); err != nil {
		return err
	}

	// Marshal event to protobuf
	data, err := proto.Marshal(event)
	if err != nil {
		return err
	}

	// Publish message
	result := p.topic.Publish(ctx, &pubsub.Message{
		Data: data,
	})

	// Wait for the message to be published
	_, err = result.Get(ctx)
	return err
}
