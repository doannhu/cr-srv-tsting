package utils

import (
	"context"
	"fmt"
	"math"
	"time"
)

// RetryConfig holds the configuration for retry operations
type RetryConfig struct {
	MaxAttempts int           // Maximum number of retry attempts
	BaseDelay   time.Duration // Initial delay between retries
	MaxDelay    time.Duration // Maximum delay between retries
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts: 5,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    5 * time.Second,
	}
}

// RetryFunc is the function type that will be retried
type RetryFunc func() error

// Retry executes the given function with retry logic
func Retry(ctx context.Context, config *RetryConfig, operation string, fn RetryFunc) error {
	var lastErr error

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Check if context is done
		if ctx.Err() != nil {
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		}

		// Execute the operation
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Calculate delay with exponential backoff
		delay := time.Duration(math.Pow(2, float64(attempt))) * config.BaseDelay
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}

		// Log retry attempt
		fmt.Printf("Retry attempt %d for %s after %v: %v\n",
			attempt+1, operation, delay, err)

		// Wait before retrying
		select {
		case <-time.After(delay):
			continue
		case <-ctx.Done():
			return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
		}
	}

	return fmt.Errorf("max retry attempts (%d) reached for %s: %w",
		config.MaxAttempts, operation, lastErr)
}
