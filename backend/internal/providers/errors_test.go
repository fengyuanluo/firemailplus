package providers

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestProviderError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ProviderError
		expected string
	}{
		{
			name: "Error with provider",
			err: &ProviderError{
				Provider: "gmail",
				Code:     "535",
				Message:  "Authentication failed",
			},
			expected: "[gmail] 535: Authentication failed",
		},
		{
			name: "Error without provider",
			err: &ProviderError{
				Code:    "TIMEOUT",
				Message: "Connection timeout",
			},
			expected: "TIMEOUT: Connection timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("Expected error string '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestProviderError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	providerErr := &ProviderError{
		Code:    "TEST",
		Message: "Test error",
		Cause:   originalErr,
	}

	unwrapped := providerErr.Unwrap()
	if unwrapped != originalErr {
		t.Errorf("Expected unwrapped error to be original error, got %v", unwrapped)
	}
}

func TestProviderError_Is(t *testing.T) {
	err1 := &ProviderError{Type: ErrorTypeAuth, Code: "535"}
	err2 := &ProviderError{Type: ErrorTypeAuth, Code: "535"}
	err3 := &ProviderError{Type: ErrorTypeAuth, Code: "534"}
	err4 := &ProviderError{Type: ErrorTypeConnection, Code: "535"}

	if !err1.Is(err2) {
		t.Error("Expected err1.Is(err2) to be true")
	}

	if err1.Is(err3) {
		t.Error("Expected err1.Is(err3) to be false (different codes)")
	}

	if err1.Is(err4) {
		t.Error("Expected err1.Is(err4) to be false (different types)")
	}

	regularErr := errors.New("regular error")
	if err1.Is(regularErr) {
		t.Error("Expected err1.Is(regularErr) to be false")
	}
}

func TestErrorClassifier_ClassifyError(t *testing.T) {
	classifier := NewErrorClassifier()

	tests := []struct {
		name         string
		inputError   error
		provider     string
		expectedType ErrorType
		expectedCode string
		retryable    bool
	}{
		{
			name:         "535 authentication error",
			inputError:   errors.New("535 Authentication failed"),
			provider:     "gmail",
			expectedType: ErrorTypeAuth,
			expectedCode: "535",
			retryable:    false,
		},
		{
			name:         "Connection timeout",
			inputError:   errors.New("connection timeout occurred"),
			provider:     "outlook",
			expectedType: ErrorTypeConnection,
			expectedCode: "TIMEOUT",
			retryable:    true,
		},
		{
			name:         "Rate limit error",
			inputError:   errors.New("550 rate limit exceeded"),
			provider:     "qq",
			expectedType: ErrorTypeRateLimit,
			expectedCode: "550",
			retryable:    true,
		},
		{
			name:         "Service unavailable",
			inputError:   errors.New("421 service temporarily unavailable"),
			provider:     "163",
			expectedType: ErrorTypeServiceUnavailable,
			expectedCode: "421",
			retryable:    true,
		},
		{
			name:         "Unknown error",
			inputError:   errors.New("some unknown error"),
			provider:     "custom",
			expectedType: ErrorTypeUnknown,
			expectedCode: "UNKNOWN_ERROR",
			retryable:    false,
		},
		{
			name:         "Already classified error",
			inputError:   &ProviderError{Type: ErrorTypeAuth, Code: "535"},
			provider:     "gmail",
			expectedType: ErrorTypeAuth,
			expectedCode: "535",
			retryable:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.ClassifyError(tt.inputError, tt.provider)

			if result == nil {
				t.Fatal("Expected classified error but got nil")
			}

			if result.Type != tt.expectedType {
				t.Errorf("Expected error type %s, got %s", tt.expectedType, result.Type)
			}

			if result.Code != tt.expectedCode {
				t.Errorf("Expected error code %s, got %s", tt.expectedCode, result.Code)
			}

			if result.Provider != tt.provider {
				t.Errorf("Expected provider %s, got %s", tt.provider, result.Provider)
			}

			if result.Retryable != tt.retryable {
				t.Errorf("Expected retryable %v, got %v", tt.retryable, result.Retryable)
			}

			if result.Timestamp.IsZero() {
				t.Error("Expected timestamp to be set")
			}
		})
	}
}

func TestErrorClassifier_MatchesPattern(t *testing.T) {
	classifier := NewErrorClassifier()

	pattern := ErrorPattern{
		Keywords: []string{"timeout", "connection failed"},
		Codes:    []string{"535", "421"},
	}

	tests := []struct {
		name     string
		errStr   string
		expected bool
	}{
		{
			name:     "Matches keyword",
			errStr:   "connection timeout occurred",
			expected: true,
		},
		{
			name:     "Matches code",
			errStr:   "535 authentication failed",
			expected: true,
		},
		{
			name:     "Matches multiple keywords",
			errStr:   "connection failed due to timeout",
			expected: true,
		},
		{
			name:     "No match",
			errStr:   "some other error",
			expected: false,
		},
		{
			name:     "Case insensitive match",
			errStr:   "CONNECTION TIMEOUT",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.matchesPattern(tt.errStr, pattern)
			if result != tt.expected {
				t.Errorf("Expected matchesPattern to return %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestRetryHandler_ShouldRetry(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts: 3,
		RetryableErrors: []ErrorType{
			ErrorTypeConnection,
			ErrorTypeTimeout,
			ErrorTypeRateLimit,
		},
	}
	handler := NewRetryHandler(config)

	tests := []struct {
		name     string
		err      error
		attempt  int
		expected bool
	}{
		{
			name:     "Nil error should not retry",
			err:      nil,
			attempt:  1,
			expected: false,
		},
		{
			name:     "Max attempts reached",
			err:      errors.New("connection timeout"),
			attempt:  3,
			expected: false,
		},
		{
			name:     "Retryable error within attempts",
			err:      errors.New("connection timeout"),
			attempt:  1,
			expected: true,
		},
		{
			name:     "Non-retryable error",
			err:      errors.New("535 authentication failed"),
			attempt:  1,
			expected: false,
		},
		{
			name:     "Rate limit error should retry",
			err:      errors.New("550 rate limit exceeded"),
			attempt:  1,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.ShouldRetry(tt.err, tt.attempt)
			if result != tt.expected {
				t.Errorf("Expected ShouldRetry to return %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestRetryHandler_CalculateDelay(t *testing.T) {
	config := &RetryConfig{
		BaseDelay:     time.Second * 2,
		MaxDelay:      time.Minute * 2,
		BackoffFactor: 2.0,
		Jitter:        false, // 禁用抖动以便测试
	}
	handler := NewRetryHandler(config)

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{
			name:     "First attempt",
			attempt:  0,
			expected: time.Second * 2,
		},
		{
			name:     "Second attempt",
			attempt:  1,
			expected: time.Second * 4,
		},
		{
			name:     "Third attempt",
			attempt:  2,
			expected: time.Second * 8,
		},
		{
			name:     "Large attempt should be capped",
			attempt:  10,
			expected: time.Minute * 2, // 应该被限制在MaxDelay
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.CalculateDelay(tt.attempt)
			if result != tt.expected {
				t.Errorf("Expected delay %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestRetryHandler_ExecuteWithRetry(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:   3,
		BaseDelay:     time.Millisecond * 10, // 短延迟以加快测试
		BackoffFactor: 1.5,
		Jitter:        false,
		RetryableErrors: []ErrorType{
			ErrorTypeConnection,
			ErrorTypeTimeout,
		},
	}
	handler := NewRetryHandler(config)

	t.Run("Success on first attempt", func(t *testing.T) {
		attempts := 0
		operation := func() error {
			attempts++
			return nil
		}

		ctx := context.Background()
		err := handler.ExecuteWithRetry(ctx, operation, "test")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if attempts != 1 {
			t.Errorf("Expected 1 attempt, got %d", attempts)
		}
	})

	t.Run("Success after retries", func(t *testing.T) {
		attempts := 0
		operation := func() error {
			attempts++
			if attempts < 3 {
				return errors.New("connection timeout")
			}
			return nil
		}

		ctx := context.Background()
		err := handler.ExecuteWithRetry(ctx, operation, "test")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if attempts != 3 {
			t.Errorf("Expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("Failure after max retries", func(t *testing.T) {
		attempts := 0
		operation := func() error {
			attempts++
			return errors.New("connection timeout")
		}

		ctx := context.Background()
		err := handler.ExecuteWithRetry(ctx, operation, "test")

		if err == nil {
			t.Error("Expected error after max retries")
		}
		if attempts != 3 {
			t.Errorf("Expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("Non-retryable error", func(t *testing.T) {
		attempts := 0
		operation := func() error {
			attempts++
			return errors.New("535 authentication failed")
		}

		ctx := context.Background()
		err := handler.ExecuteWithRetry(ctx, operation, "test")

		if err == nil {
			t.Error("Expected error for non-retryable error")
		}
		if attempts != 1 {
			t.Errorf("Expected 1 attempt for non-retryable error, got %d", attempts)
		}
	})

	t.Run("Context cancellation", func(t *testing.T) {
		attempts := 0
		operation := func() error {
			attempts++
			return errors.New("connection timeout")
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // 立即取消

		err := handler.ExecuteWithRetry(ctx, operation, "test")

		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got %v", err)
		}
	})
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxAttempts != 3 {
		t.Errorf("Expected MaxAttempts 3, got %d", config.MaxAttempts)
	}

	if config.BaseDelay != time.Second*2 {
		t.Errorf("Expected BaseDelay 2s, got %v", config.BaseDelay)
	}

	if config.BackoffFactor != 2.0 {
		t.Errorf("Expected BackoffFactor 2.0, got %f", config.BackoffFactor)
	}

	if !config.Jitter {
		t.Error("Expected Jitter to be true")
	}

	expectedRetryableErrors := []ErrorType{
		ErrorTypeConnection,
		ErrorTypeTimeout,
		ErrorTypeNetworkError,
		ErrorTypeRateLimit,
		ErrorTypeServiceUnavailable,
	}

	if len(config.RetryableErrors) != len(expectedRetryableErrors) {
		t.Errorf("Expected %d retryable errors, got %d", len(expectedRetryableErrors), len(config.RetryableErrors))
	}

	for i, expected := range expectedRetryableErrors {
		if config.RetryableErrors[i] != expected {
			t.Errorf("Expected retryable error %s at index %d, got %s", expected, i, config.RetryableErrors[i])
		}
	}
}
