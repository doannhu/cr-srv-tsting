package product_assessment

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-loan-service-v3/internal/credit_enquiry/utils"
	"go-loan-service-v3/proto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockProductService is a mock implementation of the product service
type MockProductService struct {
	mock.Mock
}

func (m *MockProductService) GetProductRate(ctx context.Context, request *proto.ProductRateRequest) (*proto.ProductRateResponse, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.ProductRateResponse), args.Error(1)
}

func TestProductAssessmentService_GetProductRate(t *testing.T) {
	retryConfig := &utils.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    time.Millisecond * 10,
	}

	ctx := context.Background()
	request := &proto.ProductRateRequest{
		ProductCode: "PROD001",
		ProductName: "Test Product",
	}

	t.Run("successful get", func(t *testing.T) {
		mockProductService := new(MockProductService)
		expectedResponse := &proto.ProductRateResponse{
			InitialStructureIndexRate: 5,
		}
		mockProductService.On("GetProductRate", ctx, request).Return(expectedResponse, nil)

		service := NewProductAssessmentService(retryConfig)
		response, err := service.GetProductRate(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, expectedResponse.InitialStructureIndexRate, response.InitialStructureIndexRate)
		mockProductService.AssertExpectations(t)
	})

	t.Run("error from product service", func(t *testing.T) {
		mockProductService := new(MockProductService)
		expectedErr := errors.New("product service error")
		mockProductService.On("GetProductRate", ctx, request).Return(nil, expectedErr)

		service := NewProductAssessmentService(retryConfig)
		response, err := service.GetProductRate(ctx, request)
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "failed to get product rate")
		mockProductService.AssertExpectations(t)
	})

	t.Run("retry on error", func(t *testing.T) {
		mockProductService := new(MockProductService)
		expectedResponse := &proto.ProductRateResponse{
			InitialStructureIndexRate: 5,
		}
		// First call fails, second call succeeds
		mockProductService.On("GetProductRate", ctx, request).Return(nil, errors.New("temporary error")).Once()
		mockProductService.On("GetProductRate", ctx, request).Return(expectedResponse, nil).Once()

		service := NewProductAssessmentService(retryConfig)
		response, err := service.GetProductRate(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, expectedResponse.InitialStructureIndexRate, response.InitialStructureIndexRate)
		mockProductService.AssertExpectations(t)
	})
}
