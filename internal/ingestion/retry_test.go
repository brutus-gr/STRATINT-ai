package ingestion

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetry_Success(t *testing.T) {
	policy := RetryPolicy{
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		BackoffFactor:  2.0,
	}

	attempts := 0
	fn := func() error {
		attempts++
		return nil
	}

	err := Retry(context.Background(), policy, fn)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestRetry_SuccessAfterRetries(t *testing.T) {
	policy := RetryPolicy{
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		BackoffFactor:  2.0,
	}

	attempts := 0
	fn := func() error {
		attempts++
		if attempts < 3 {
			return NewRetryableError(errors.New("temporary error"))
		}
		return nil
	}

	err := Retry(context.Background(), policy, fn)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetry_MaxRetriesExceeded(t *testing.T) {
	policy := RetryPolicy{
		MaxRetries:     2,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		BackoffFactor:  2.0,
	}

	attempts := 0
	fn := func() error {
		attempts++
		return NewRetryableError(errors.New("persistent error"))
	}

	err := Retry(context.Background(), policy, fn)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts (initial + 2 retries), got %d", attempts)
	}
}

func TestRetry_NonRetryableError(t *testing.T) {
	policy := RetryPolicy{
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		BackoffFactor:  2.0,
	}

	attempts := 0
	fn := func() error {
		attempts++
		return errors.New("non-retryable error")
	}

	err := Retry(context.Background(), policy, fn)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if attempts != 1 {
		t.Errorf("expected 1 attempt (non-retryable), got %d", attempts)
	}
}

func TestRetry_ContextCancellation(t *testing.T) {
	policy := RetryPolicy{
		MaxRetries:     5,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
		BackoffFactor:  2.0,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	attempts := 0
	fn := func() error {
		attempts++
		return NewRetryableError(errors.New("retryable error"))
	}

	err := Retry(ctx, policy, fn)
	if err == nil {
		t.Fatal("expected context error, got nil")
	}

	// Should have attempted once, then cancelled during backoff
	if attempts < 1 {
		t.Errorf("expected at least 1 attempt, got %d", attempts)
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"regular error", errors.New("regular"), false},
		{"retryable error", NewRetryableError(errors.New("retry")), true},
		{"retryable with delay", NewRetryableErrorWithDelay(errors.New("retry"), 1*time.Second), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.expected {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCalculateBackoff(t *testing.T) {
	policy := RetryPolicy{
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     10 * time.Second,
		BackoffFactor:  2.0,
		Jitter:         false,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 1 * time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{4, 10 * time.Second}, // Capped at max
		{5, 10 * time.Second}, // Stays at max
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			backoff := calculateBackoff(policy, tt.attempt)
			// Allow 10% tolerance for jitter
			if backoff < tt.expected-100*time.Millisecond || backoff > tt.expected+100*time.Millisecond {
				t.Errorf("attempt %d: expected ~%v, got %v", tt.attempt, tt.expected, backoff)
			}
		})
	}
}

func TestDefaultRetryPolicy(t *testing.T) {
	policy := DefaultRetryPolicy()

	if policy.MaxRetries != 3 {
		t.Errorf("expected MaxRetries=3, got %d", policy.MaxRetries)
	}
	if policy.InitialBackoff != 1*time.Second {
		t.Errorf("expected InitialBackoff=1s, got %v", policy.InitialBackoff)
	}
	if !policy.Jitter {
		t.Error("expected Jitter=true")
	}
}

func TestRetryableError(t *testing.T) {
	err := NewRetryableError(errors.New("test error"))
	if err.Error() != "test error" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	errWithDelay := NewRetryableErrorWithDelay(errors.New("test error"), 5*time.Second)
	var retryErr *RetryableError
	if !errors.As(errWithDelay, &retryErr) {
		t.Error("should be a RetryableError")
	}
}

func TestWithRetry(t *testing.T) {
	policy := RetryPolicy{
		MaxRetries:     2,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		BackoffFactor:  2.0,
	}

	attempts := 0
	retryableFn := WithRetry(policy, func(ctx context.Context) error {
		attempts++
		if attempts < 2 {
			return NewRetryableError(errors.New("retry"))
		}
		return nil
	})

	err := retryableFn(context.Background())
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}
