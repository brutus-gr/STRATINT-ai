package ingestion

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"
)

// RetryPolicy defines how retries should be handled.
type RetryPolicy struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	BackoffFactor  float64
	Jitter         bool
}

// DefaultRetryPolicy returns a sensible default retry policy.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:     3,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
		BackoffFactor:  2.0,
		Jitter:         true,
	}
}

// RetryableError wraps an error to indicate it should be retried.
type RetryableError struct {
	Err        error
	RetryAfter time.Duration
}

func (e *RetryableError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("%v (retry after %v)", e.Err, e.RetryAfter)
	}
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// IsRetryable checks if an error should trigger a retry.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	var retryable *RetryableError
	return errors.As(err, &retryable)
}

// Retry executes a function with exponential backoff retry logic.
func Retry(ctx context.Context, policy RetryPolicy, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		// Execute the function
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !IsRetryable(err) {
			return err
		}

		// Don't sleep after the last attempt
		if attempt == policy.MaxRetries {
			break
		}

		// Calculate backoff duration
		backoff := calculateBackoff(policy, attempt)

		// Check for RetryAfter hint
		var retryErr *RetryableError
		if errors.As(err, &retryErr) && retryErr.RetryAfter > 0 {
			backoff = retryErr.RetryAfter
		}

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		case <-time.After(backoff):
			// Continue to next attempt
		}
	}

	return fmt.Errorf("max retries exceeded (%d): %w", policy.MaxRetries, lastErr)
}

// calculateBackoff computes the backoff duration for a given attempt.
func calculateBackoff(policy RetryPolicy, attempt int) time.Duration {
	// Exponential backoff: initialBackoff * (factor ^ attempt)
	backoff := float64(policy.InitialBackoff) * math.Pow(policy.BackoffFactor, float64(attempt))

	// Cap at max backoff
	if backoff > float64(policy.MaxBackoff) {
		backoff = float64(policy.MaxBackoff)
	}

	duration := time.Duration(backoff)

	// Add jitter to prevent thundering herd
	if policy.Jitter {
		jitter := time.Duration(float64(duration) * 0.1 * (2*fakeRand() - 1))
		duration += jitter
	}

	return duration
}

// fakeRand returns a pseudo-random value between 0 and 1.
// Uses time-based seed for simplicity (not cryptographically secure).
func fakeRand() float64 {
	// Simple time-based pseudo-random for jitter
	nanos := time.Now().UnixNano()
	return float64(nanos%1000) / 1000.0
}

// RetryableFunc wraps a function to make it retryable with a policy.
type RetryableFunc func(ctx context.Context) error

// WithRetry returns a function that executes with retry logic.
func WithRetry(policy RetryPolicy, fn RetryableFunc) RetryableFunc {
	return func(ctx context.Context) error {
		return Retry(ctx, policy, func() error {
			return fn(ctx)
		})
	}
}

// NewRetryableError creates a new retryable error.
func NewRetryableError(err error) error {
	return &RetryableError{Err: err}
}

// NewRetryableErrorWithDelay creates a retryable error with a specific retry delay.
func NewRetryableErrorWithDelay(err error, delay time.Duration) error {
	return &RetryableError{Err: err, RetryAfter: delay}
}
