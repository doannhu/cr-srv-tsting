package product_assessment

import (
	"context"
	"testing"
	"time"

	"go-loan-service-v3/internal/credit_enquiry/utils"
	"go-loan-service-v3/proto"

	"github.com/stretchr/testify/assert"
)

func TestProductAssessmentService_GetProductRate_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

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

	t.Run("successful get from product service", func(t *testing.T) {
		service := NewProductAssessmentService(retryConfig)
		response, err := service.GetProductRate(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Greater(t, response.InitialStructureIndexRate, int64(0))
	})

	t.Run("invalid product code", func(t *testing.T) {
		service := NewProductAssessmentService(retryConfig)
		invalidRequest := &proto.ProductRateRequest{
			ProductCode: "INVALID",
			ProductName: "Invalid Product",
		}
		response, err := service.GetProductRate(ctx, invalidRequest)
		assert.Error(t, err)
		assert.Nil(t, response)
	})
}
