package product_assessment

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-loan-service-v3/internal/credit_enquiry/utils"
	pb "go-loan-service-v3/proto/product_assessment"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// MockProductAssessmentServiceClient is a mock implementation of pb.ProductAssessmentServiceClient
type MockProductAssessmentServiceClient struct {
	mock.Mock
}

func (m *MockProductAssessmentServiceClient) GetProductRate(ctx context.Context, request *pb.ProductRateRequest, opts ...grpc.CallOption) (*pb.ProductRateResponse, error) {
	args := m.Called(ctx, request, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.ProductRateResponse), args.Error(1)
}

func TestProductAssessmentService_GetProductRate(t *testing.T) {
	retryConfig := &utils.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    time.Millisecond * 10,
	}

	ctx := context.Background()
	request := &pb.ProductRateRequest{
		ProductCode: "PROD001",
		ProductName: "Test Product",
	}

	t.Run("successful get", func(t *testing.T) {
		mockClient := new(MockProductAssessmentServiceClient)
		expectedResponse := &pb.ProductRateResponse{
			InitialStructureIndexRate: 500,
		}
		mockClient.On("GetProductRate", ctx, request, mock.Anything).Return(expectedResponse, nil)

		service := NewProductAssessmentService(retryConfig, nil).(*productAssessmentService)
		service.client = mockClient

		response, err := service.GetProductRate(ctx, request)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, expectedResponse.InitialStructureIndexRate, response.InitialStructureIndexRate)
		mockClient.AssertExpectations(t)
	})

	t.Run("nil request", func(t *testing.T) {
		service := NewProductAssessmentService(retryConfig, nil)
		response, err := service.GetProductRate(ctx, nil)

		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "request cannot be nil")
	})

	t.Run("service error", func(t *testing.T) {
		mockClient := new(MockProductAssessmentServiceClient)
		expectedErr := errors.New("service error")
		mockClient.On("GetProductRate", ctx, request, mock.Anything).Return(nil, expectedErr)

		service := NewProductAssessmentService(retryConfig, nil).(*productAssessmentService)
		service.client = mockClient

		response, err := service.GetProductRate(ctx, request)

		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "failed to get product rate")
		mockClient.AssertExpectations(t)
	})

	t.Run("retry on error", func(t *testing.T) {
		mockClient := new(MockProductAssessmentServiceClient)
		expectedResponse := &pb.ProductRateResponse{
			InitialStructureIndexRate: 500,
		}
		// First call fails, second call succeeds
		mockClient.On("GetProductRate", ctx, request, mock.Anything).Return(nil, errors.New("temporary error")).Once()
		mockClient.On("GetProductRate", ctx, request, mock.Anything).Return(expectedResponse, nil).Once()

		service := NewProductAssessmentService(retryConfig, nil).(*productAssessmentService)
		service.client = mockClient

		response, err := service.GetProductRate(ctx, request)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, expectedResponse.InitialStructureIndexRate, response.InitialStructureIndexRate)
		mockClient.AssertExpectations(t)
	})

	t.Run("empty product code uses default", func(t *testing.T) {
		mockClient := new(MockProductAssessmentServiceClient)
		emptyRequest := &pb.ProductRateRequest{
			ProductCode: "",
			ProductName: "Test Product",
		}
		expectedRequest := &pb.ProductRateRequest{
			ProductCode: defaultProductCode,
			ProductName: "Test Product",
		}
		expectedResponse := &pb.ProductRateResponse{
			InitialStructureIndexRate: 500,
		}
		mockClient.On("GetProductRate", ctx, expectedRequest, mock.Anything).Return(expectedResponse, nil)

		service := NewProductAssessmentService(retryConfig, nil).(*productAssessmentService)
		service.client = mockClient

		response, err := service.GetProductRate(ctx, emptyRequest)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, expectedResponse.InitialStructureIndexRate, response.InitialStructureIndexRate)
		mockClient.AssertExpectations(t)
	})
}
