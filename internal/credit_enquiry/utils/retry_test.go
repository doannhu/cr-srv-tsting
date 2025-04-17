package utils

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetry_SuccessOnFirstAttempt(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	counter := 0

	err := Retry(ctx, config, "test operation", func() error {
		counter++
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if counter != 1 {
		t.Errorf("Expected 1 attempt, got %d", counter)
	}
}

func TestRetry_SuccessAfterRetries(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	counter := 0

	err := Retry(ctx, config, "test operation", func() error {
		counter++
		if counter < 3 {
			return errors.New("temporary error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if counter != 3 {
		t.Errorf("Expected 3 attempts, got %d", counter)
	}
}

func TestRetry_MaxAttemptsReached(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	counter := 0

	err := Retry(ctx, config, "test operation", func() error {
		counter++
		return errors.New("permanent error")
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}

	if counter != config.MaxAttempts {
		t.Errorf("Expected %d attempts, got %d", config.MaxAttempts, counter)
	}
}

func TestRetry_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	config := DefaultRetryConfig()
	counter := 0

	// Cancel context after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := Retry(ctx, config, "test operation", func() error {
		counter++
		return errors.New("temporary error")
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}

	if counter >= config.MaxAttempts {
		t.Errorf("Expected fewer than %d attempts due to context cancellation, got %d",
			config.MaxAttempts, counter)
	}
}

func TestRetry_ExponentialBackoff(t *testing.T) {
	ctx := context.Background()
	config := &RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
	}
	startTime := time.Now()
	counter := 0

	err := Retry(ctx, config, "test operation", func() error {
		counter++
		if counter < 3 {
			return errors.New("temporary error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	elapsed := time.Since(startTime)
	// Expected delays: 100ms, 200ms
	expectedMin := 300 * time.Millisecond
	expectedMax := 500 * time.Millisecond

	if elapsed < expectedMin || elapsed > expectedMax {
		t.Errorf("Expected elapsed time between %v and %v, got %v",
			expectedMin, expectedMax, elapsed)
	}
}
