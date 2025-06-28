package providers

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ErrorType 错误类型
type ErrorType string

const (
	// 连接错误
	ErrorTypeConnection   ErrorType = "connection"
	ErrorTypeTimeout      ErrorType = "timeout"
	ErrorTypeNetworkError ErrorType = "network"

	// 认证错误
	ErrorTypeAuth        ErrorType = "authentication"
	ErrorTypeCredentials ErrorType = "credentials"
	ErrorTypePermission  ErrorType = "permission"
	ErrorTypeOAuth2      ErrorType = "oauth2"

	// 协议错误
	ErrorTypeIMAP     ErrorType = "imap"
	ErrorTypeSMTP     ErrorType = "smtp"
	ErrorTypeProtocol ErrorType = "protocol"

	// 服务器错误
	ErrorTypeServerError        ErrorType = "server_error"
	ErrorTypeRateLimit          ErrorType = "rate_limit"
	ErrorTypeQuotaExceeded      ErrorType = "quota_exceeded"
	ErrorTypeServiceUnavailable ErrorType = "service_unavailable"

	// 配置错误
	ErrorTypeConfig     ErrorType = "configuration"
	ErrorTypeValidation ErrorType = "validation"

	// 数据错误
	ErrorTypeDataFormat ErrorType = "data_format"
	ErrorTypeEncoding   ErrorType = "encoding"

	// 未知错误
	ErrorTypeUnknown ErrorType = "unknown"
)

// ErrorSeverity 错误严重程度
type ErrorSeverity string

const (
	SeverityCritical ErrorSeverity = "critical"
	SeverityHigh     ErrorSeverity = "high"
	SeverityMedium   ErrorSeverity = "medium"
	SeverityLow      ErrorSeverity = "low"
	SeverityInfo     ErrorSeverity = "info"
)

// ProviderError 提供商错误
type ProviderError struct {
	Type        ErrorType              `json:"type"`
	Code        string                 `json:"code"`
	Message     string                 `json:"message"`
	Provider    string                 `json:"provider"`
	Severity    ErrorSeverity          `json:"severity"`
	Retryable   bool                   `json:"retryable"`
	Temporary   bool                   `json:"temporary"`
	Cause       error                  `json:"-"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Suggestions []string               `json:"suggestions,omitempty"`
}

// Error 实现error接口
func (e *ProviderError) Error() string {
	if e.Provider != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Provider, e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap 实现errors.Unwrap接口
func (e *ProviderError) Unwrap() error {
	return e.Cause
}

// Is 实现errors.Is接口
func (e *ProviderError) Is(target error) bool {
	if pe, ok := target.(*ProviderError); ok {
		return e.Type == pe.Type && e.Code == pe.Code
	}
	return false
}

// ErrorClassifier 错误分类器
type ErrorClassifier struct {
	patterns map[ErrorType][]ErrorPattern
}

// ErrorPattern 错误模式
type ErrorPattern struct {
	Keywords    []string      `json:"keywords"`
	Codes       []string      `json:"codes"`
	Type        ErrorType     `json:"type"`
	Severity    ErrorSeverity `json:"severity"`
	Retryable   bool          `json:"retryable"`
	Temporary   bool          `json:"temporary"`
	Suggestions []string      `json:"suggestions"`
}

// NewErrorClassifier 创建错误分类器
func NewErrorClassifier() *ErrorClassifier {
	classifier := &ErrorClassifier{
		patterns: make(map[ErrorType][]ErrorPattern),
	}

	classifier.initializePatterns()
	return classifier
}

// initializePatterns 初始化错误模式
func (ec *ErrorClassifier) initializePatterns() {
	// 连接错误模式
	ec.patterns[ErrorTypeConnection] = []ErrorPattern{
		{
			Keywords:    []string{"connection refused", "connection failed", "connection timeout"},
			Type:        ErrorTypeConnection,
			Severity:    SeverityHigh,
			Retryable:   true,
			Temporary:   true,
			Suggestions: []string{"Check network connectivity", "Verify server address and port"},
		},
		{
			Keywords:    []string{"connection closed", "connection reset", "broken pipe", "eof"},
			Type:        ErrorTypeConnection,
			Severity:    SeverityMedium,
			Retryable:   true,
			Temporary:   true,
			Suggestions: []string{"Connection was closed by server", "Retry the operation", "Check connection stability"},
		},
		{
			Keywords:    []string{"network is unreachable", "no route to host", "host is down"},
			Type:        ErrorTypeNetworkError,
			Severity:    SeverityHigh,
			Retryable:   true,
			Temporary:   true,
			Suggestions: []string{"Check network connectivity", "Verify server is accessible", "Check firewall settings"},
		},
	}

	// 超时错误模式
	ec.patterns[ErrorTypeTimeout] = []ErrorPattern{
		{
			Keywords:    []string{"timeout", "timed out", "deadline exceeded", "i/o timeout", "read timeout", "write timeout"},
			Type:        ErrorTypeTimeout,
			Severity:    SeverityMedium,
			Retryable:   true,
			Temporary:   true,
			Suggestions: []string{"Increase timeout duration", "Check network stability"},
		},
	}

	// 认证错误模式
	ec.patterns[ErrorTypeAuth] = []ErrorPattern{
		{
			Codes:       []string{"535", "534"},
			Keywords:    []string{"authentication failed", "invalid credentials", "login failed"},
			Type:        ErrorTypeAuth,
			Severity:    SeverityHigh,
			Retryable:   false,
			Temporary:   false,
			Suggestions: []string{"Check username and password", "Verify authentication method", "Check if 2FA is required"},
		},
		{
			Keywords:    []string{"invalid_grant", "token expired", "access denied"},
			Type:        ErrorTypeOAuth2,
			Severity:    SeverityHigh,
			Retryable:   false,
			Temporary:   false,
			Suggestions: []string{"Refresh OAuth2 token", "Re-authenticate", "Check token permissions"},
		},
	}

	// 速率限制错误模式
	ec.patterns[ErrorTypeRateLimit] = []ErrorPattern{
		{
			Codes:       []string{"550", "552", "554"},
			Keywords:    []string{"rate limit", "too many requests", "frequency limit"},
			Type:        ErrorTypeRateLimit,
			Severity:    SeverityMedium,
			Retryable:   true,
			Temporary:   true,
			Suggestions: []string{"Reduce request frequency", "Wait before retrying", "Check rate limits"},
		},
	}

	// 服务不可用错误模式
	ec.patterns[ErrorTypeServiceUnavailable] = []ErrorPattern{
		{
			Codes:       []string{"421", "450", "451"},
			Keywords:    []string{"service unavailable", "server busy", "temporarily unavailable"},
			Type:        ErrorTypeServiceUnavailable,
			Severity:    SeverityMedium,
			Retryable:   true,
			Temporary:   true,
			Suggestions: []string{"Wait and retry", "Check service status", "Try again later"},
		},
	}

	// 服务器错误模式
	ec.patterns[ErrorTypeServerError] = []ErrorPattern{
		{
			Codes:       []string{"452", "553"},
			Keywords:    []string{"quota exceeded", "mailbox full", "storage limit"},
			Type:        ErrorTypeQuotaExceeded,
			Severity:    SeverityHigh,
			Retryable:   false,
			Temporary:   false,
			Suggestions: []string{"Free up storage space", "Upgrade storage plan", "Delete old emails"},
		},
	}

	// 协议错误模式
	ec.patterns[ErrorTypeProtocol] = []ErrorPattern{
		{
			Keywords:    []string{"protocol error", "invalid command", "syntax error"},
			Type:        ErrorTypeProtocol,
			Severity:    SeverityHigh,
			Retryable:   false,
			Temporary:   false,
			Suggestions: []string{"Check protocol implementation", "Verify command syntax", "Update client"},
		},
	}
}

// ClassifyError 分类错误
func (ec *ErrorClassifier) ClassifyError(err error, provider string) *ProviderError {
	if err == nil {
		return nil
	}

	// 如果已经是ProviderError，更新provider字段和时间戳后返回
	if pe, ok := err.(*ProviderError); ok {
		if pe.Provider == "" {
			pe.Provider = provider
		}
		if pe.Timestamp.IsZero() {
			pe.Timestamp = time.Now()
		}
		return pe
	}

	errStr := strings.ToLower(err.Error())

	// 遍历所有模式进行匹配
	for errorType, patterns := range ec.patterns {
		for _, pattern := range patterns {
			if ec.matchesPattern(errStr, pattern) {
				return &ProviderError{
					Type:        errorType,
					Code:        ec.extractCode(errStr, pattern),
					Message:     err.Error(),
					Provider:    provider,
					Severity:    pattern.Severity,
					Retryable:   pattern.Retryable,
					Temporary:   pattern.Temporary,
					Cause:       err,
					Context:     make(map[string]interface{}),
					Timestamp:   time.Now(),
					Suggestions: pattern.Suggestions,
				}
			}
		}
	}

	// 如果没有匹配的模式，返回未知错误
	return &ProviderError{
		Type:      ErrorTypeUnknown,
		Code:      "UNKNOWN_ERROR",
		Message:   err.Error(),
		Provider:  provider,
		Severity:  SeverityMedium,
		Retryable: false,
		Temporary: false,
		Cause:     err,
		Context:   make(map[string]interface{}),
		Timestamp: time.Now(),
	}
}

// matchesPattern 检查是否匹配模式
func (ec *ErrorClassifier) matchesPattern(errStr string, pattern ErrorPattern) bool {
	// 转换为小写进行比较
	errStrLower := strings.ToLower(errStr)

	// 检查错误代码
	for _, code := range pattern.Codes {
		if strings.Contains(errStrLower, strings.ToLower(code)) {
			return true
		}
	}

	// 检查关键词
	for _, keyword := range pattern.Keywords {
		if strings.Contains(errStrLower, strings.ToLower(keyword)) {
			return true
		}
	}

	return false
}

// extractCode 提取错误代码
func (ec *ErrorClassifier) extractCode(errStr string, pattern ErrorPattern) string {
	// 优先返回匹配的错误代码
	for _, code := range pattern.Codes {
		if strings.Contains(errStr, code) {
			return code
		}
	}

	// 特殊处理：如果是连接错误但包含timeout关键词，返回TIMEOUT
	if pattern.Type == ErrorTypeConnection && strings.Contains(errStr, "timeout") {
		return "TIMEOUT"
	}

	// 如果没有错误代码，根据错误类型返回特定代码
	switch pattern.Type {
	case ErrorTypeTimeout:
		return "TIMEOUT"
	case ErrorTypeConnection:
		return "CONNECTION"
	default:
		return strings.ToUpper(string(pattern.Type))
	}
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxAttempts     int           `json:"max_attempts"`
	BaseDelay       time.Duration `json:"base_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor"`
	Jitter          bool          `json:"jitter"`
	RetryableErrors []ErrorType   `json:"retryable_errors"`
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:   3,
		BaseDelay:     time.Second * 2,
		MaxDelay:      time.Minute * 2,
		BackoffFactor: 2.0,
		Jitter:        true,
		RetryableErrors: []ErrorType{
			ErrorTypeConnection,
			ErrorTypeTimeout,
			ErrorTypeNetworkError,
			ErrorTypeRateLimit,
			ErrorTypeServiceUnavailable,
		},
	}
}

// RetryHandler 重试处理器
type RetryHandler struct {
	config     *RetryConfig
	classifier *ErrorClassifier
}

// NewRetryHandler 创建重试处理器
func NewRetryHandler(config *RetryConfig) *RetryHandler {
	if config == nil {
		config = DefaultRetryConfig()
	}

	return &RetryHandler{
		config:     config,
		classifier: NewErrorClassifier(),
	}
}

// ShouldRetry 判断是否应该重试
func (rh *RetryHandler) ShouldRetry(err error, attempt int) bool {
	if err == nil {
		return false
	}

	if attempt >= rh.config.MaxAttempts {
		return false
	}

	// 分类错误
	providerErr := rh.classifier.ClassifyError(err, "")

	// 检查是否为可重试错误
	if !providerErr.Retryable {
		return false
	}

	// 检查错误类型是否在可重试列表中
	for _, retryableType := range rh.config.RetryableErrors {
		if providerErr.Type == retryableType {
			return true
		}
	}

	return false
}

// CalculateDelay 计算重试延迟
func (rh *RetryHandler) CalculateDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return rh.config.BaseDelay
	}

	// 指数退避
	delay := rh.config.BaseDelay
	for i := 0; i < attempt; i++ {
		delay = time.Duration(float64(delay) * rh.config.BackoffFactor)
	}

	// 限制最大延迟
	if delay > rh.config.MaxDelay {
		delay = rh.config.MaxDelay
	}

	// 添加抖动
	if rh.config.Jitter {
		jitter := time.Duration(float64(delay) * 0.1) // 10%的抖动
		randomFactor := float64(time.Now().UnixNano()%1000) / 1000.0
		delay += time.Duration(float64(jitter) * (2*randomFactor - 1))
	}

	return delay
}

// ExecuteWithRetry 执行带重试的操作
func (rh *RetryHandler) ExecuteWithRetry(ctx context.Context, operation func() error, provider string) error {
	var lastErr error

	for attempt := 0; attempt < rh.config.MaxAttempts; attempt++ {
		// 执行操作
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// 检查是否应该重试
		if !rh.ShouldRetry(err, attempt) {
			break
		}

		// 如果是最后一次尝试，不需要等待
		if attempt == rh.config.MaxAttempts-1 {
			break
		}

		// 计算延迟并等待
		delay := rh.CalculateDelay(attempt)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// 继续下一次重试
		}
	}

	// 返回分类后的错误
	return rh.classifier.ClassifyError(lastErr, provider)
}
