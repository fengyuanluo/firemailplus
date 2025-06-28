package providers

import (
	"context"
	"log"
	"sync"
	"time"

	"firemail/internal/models"
)

// RetryableOperation 可重试操作接口
type RetryableOperation interface {
	Execute(ctx context.Context) error
	GetOperationType() string
	GetProvider() string
}

// ConnectionOperation 连接操作
type ConnectionOperation struct {
	Provider EmailProvider
	Account  *models.EmailAccount
}

func (op *ConnectionOperation) Execute(ctx context.Context) error {
	return op.Provider.Connect(ctx, op.Account)
}

func (op *ConnectionOperation) GetOperationType() string {
	return "connection"
}

func (op *ConnectionOperation) GetProvider() string {
	return op.Account.Provider
}

// SendEmailOperation 发送邮件操作
type SendEmailOperation struct {
	Provider EmailProvider
	Account  *models.EmailAccount
	Message  *OutgoingMessage
}

func (op *SendEmailOperation) Execute(ctx context.Context) error {
	return op.Provider.SendEmail(ctx, op.Account, op.Message)
}

func (op *SendEmailOperation) GetOperationType() string {
	return "send_email"
}

func (op *SendEmailOperation) GetProvider() string {
	return op.Account.Provider
}

// SyncEmailsOperation 同步邮件操作
type SyncEmailsOperation struct {
	Provider   EmailProvider
	Account    *models.EmailAccount
	FolderName string
	LastUID    uint32
}

func (op *SyncEmailsOperation) Execute(ctx context.Context) error {
	_, err := op.Provider.SyncEmails(ctx, op.Account, op.FolderName, op.LastUID)
	return err
}

func (op *SyncEmailsOperation) GetOperationType() string {
	return "sync_emails"
}

func (op *SyncEmailsOperation) GetProvider() string {
	return op.Account.Provider
}

// RetryManager 重试管理器
type RetryManager struct {
	handler    *RetryHandler
	configs    map[string]*RetryConfig // 按操作类型配置
	statistics *RetryStatistics
	mutex      sync.RWMutex
}

// RetryStatistics 重试统计
type RetryStatistics struct {
	TotalOperations   int64                      `json:"total_operations"`
	SuccessfulRetries int64                      `json:"successful_retries"`
	FailedRetries     int64                      `json:"failed_retries"`
	OperationStats    map[string]*OperationStats `json:"operation_stats"`
	ProviderStats     map[string]*ProviderStats  `json:"provider_stats"`
	LastUpdated       time.Time                  `json:"last_updated"`
	mutex             sync.RWMutex
}

// OperationStats 操作统计
type OperationStats struct {
	Count        int64         `json:"count"`
	SuccessCount int64         `json:"success_count"`
	RetryCount   int64         `json:"retry_count"`
	FailureCount int64         `json:"failure_count"`
	AverageDelay time.Duration `json:"average_delay"`
	TotalDelay   time.Duration `json:"total_delay"`
}

// ProviderStats 提供商统计
type ProviderStats struct {
	Count        int64               `json:"count"`
	SuccessCount int64               `json:"success_count"`
	RetryCount   int64               `json:"retry_count"`
	FailureCount int64               `json:"failure_count"`
	ErrorTypes   map[ErrorType]int64 `json:"error_types"`
	LastError    *ProviderError      `json:"last_error,omitempty"`
	LastSuccess  time.Time           `json:"last_success"`
}

// NewRetryManager 创建重试管理器
func NewRetryManager() *RetryManager {
	return &RetryManager{
		handler: NewRetryHandler(DefaultRetryConfig()),
		configs: make(map[string]*RetryConfig),
		statistics: &RetryStatistics{
			OperationStats: make(map[string]*OperationStats),
			ProviderStats:  make(map[string]*ProviderStats),
			LastUpdated:    time.Now(),
		},
	}
}

// SetRetryConfig 设置特定操作类型的重试配置
func (rm *RetryManager) SetRetryConfig(operationType string, config *RetryConfig) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	rm.configs[operationType] = config
}

// GetRetryConfig 获取特定操作类型的重试配置
func (rm *RetryManager) GetRetryConfig(operationType string) *RetryConfig {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	if config, exists := rm.configs[operationType]; exists {
		return config
	}

	// 返回默认配置
	return DefaultRetryConfig()
}

// ExecuteWithRetry 执行带重试的操作
func (rm *RetryManager) ExecuteWithRetry(ctx context.Context, operation RetryableOperation) error {
	operationType := operation.GetOperationType()
	provider := operation.GetProvider()

	// 获取配置
	config := rm.GetRetryConfig(operationType)
	handler := NewRetryHandler(config)

	// 记录开始时间
	startTime := time.Now()

	// 执行操作
	err := handler.ExecuteWithRetry(ctx, func() error {
		return operation.Execute(ctx)
	}, provider)

	// 更新统计
	rm.updateStatistics(operationType, provider, err, time.Since(startTime))

	return err
}

// updateStatistics 更新统计信息
func (rm *RetryManager) updateStatistics(operationType, provider string, err error, duration time.Duration) {
	rm.statistics.mutex.Lock()
	defer rm.statistics.mutex.Unlock()

	rm.statistics.TotalOperations++
	rm.statistics.LastUpdated = time.Now()

	// 更新操作统计
	if rm.statistics.OperationStats[operationType] == nil {
		rm.statistics.OperationStats[operationType] = &OperationStats{}
	}
	opStats := rm.statistics.OperationStats[operationType]
	opStats.Count++
	opStats.TotalDelay += duration
	opStats.AverageDelay = opStats.TotalDelay / time.Duration(opStats.Count)

	// 更新提供商统计
	if rm.statistics.ProviderStats[provider] == nil {
		rm.statistics.ProviderStats[provider] = &ProviderStats{
			ErrorTypes: make(map[ErrorType]int64),
		}
	}
	providerStats := rm.statistics.ProviderStats[provider]
	providerStats.Count++

	if err != nil {
		opStats.FailureCount++
		providerStats.FailureCount++
		rm.statistics.FailedRetries++

		// 记录错误类型
		if providerErr, ok := err.(*ProviderError); ok {
			providerStats.ErrorTypes[providerErr.Type]++
			providerStats.LastError = providerErr
		}
	} else {
		opStats.SuccessCount++
		providerStats.SuccessCount++
		providerStats.LastSuccess = time.Now()
		rm.statistics.SuccessfulRetries++
	}
}

// GetStatistics 获取统计信息
func (rm *RetryManager) GetStatistics() *RetryStatistics {
	rm.statistics.mutex.RLock()
	defer rm.statistics.mutex.RUnlock()

	// 创建副本以避免并发问题
	stats := &RetryStatistics{
		TotalOperations:   rm.statistics.TotalOperations,
		SuccessfulRetries: rm.statistics.SuccessfulRetries,
		FailedRetries:     rm.statistics.FailedRetries,
		OperationStats:    make(map[string]*OperationStats),
		ProviderStats:     make(map[string]*ProviderStats),
		LastUpdated:       rm.statistics.LastUpdated,
	}

	// 复制操作统计
	for opType, opStats := range rm.statistics.OperationStats {
		stats.OperationStats[opType] = &OperationStats{
			Count:        opStats.Count,
			SuccessCount: opStats.SuccessCount,
			RetryCount:   opStats.RetryCount,
			FailureCount: opStats.FailureCount,
			AverageDelay: opStats.AverageDelay,
			TotalDelay:   opStats.TotalDelay,
		}
	}

	// 复制提供商统计
	for provider, providerStats := range rm.statistics.ProviderStats {
		errorTypes := make(map[ErrorType]int64)
		for errorType, count := range providerStats.ErrorTypes {
			errorTypes[errorType] = count
		}

		stats.ProviderStats[provider] = &ProviderStats{
			Count:        providerStats.Count,
			SuccessCount: providerStats.SuccessCount,
			RetryCount:   providerStats.RetryCount,
			FailureCount: providerStats.FailureCount,
			ErrorTypes:   errorTypes,
			LastError:    providerStats.LastError,
			LastSuccess:  providerStats.LastSuccess,
		}
	}

	return stats
}

// ResetStatistics 重置统计信息
func (rm *RetryManager) ResetStatistics() {
	rm.statistics.mutex.Lock()
	defer rm.statistics.mutex.Unlock()

	rm.statistics.TotalOperations = 0
	rm.statistics.SuccessfulRetries = 0
	rm.statistics.FailedRetries = 0
	rm.statistics.OperationStats = make(map[string]*OperationStats)
	rm.statistics.ProviderStats = make(map[string]*ProviderStats)
	rm.statistics.LastUpdated = time.Now()
}

// GetHealthStatus 获取健康状态
func (rm *RetryManager) GetHealthStatus() *HealthStatus {
	stats := rm.GetStatistics()

	status := &HealthStatus{
		Overall:     "healthy",
		Providers:   make(map[string]string),
		Operations:  make(map[string]string),
		LastChecked: time.Now(),
	}

	// 检查整体健康状态
	if stats.TotalOperations > 0 {
		failureRate := float64(stats.FailedRetries) / float64(stats.TotalOperations)
		if failureRate > 0.5 {
			status.Overall = "unhealthy"
		} else if failureRate > 0.2 {
			status.Overall = "degraded"
		}
	}

	// 检查提供商健康状态
	for provider, providerStats := range stats.ProviderStats {
		if providerStats.Count > 0 {
			failureRate := float64(providerStats.FailureCount) / float64(providerStats.Count)
			if failureRate > 0.5 {
				status.Providers[provider] = "unhealthy"
			} else if failureRate > 0.2 {
				status.Providers[provider] = "degraded"
			} else {
				status.Providers[provider] = "healthy"
			}
		}
	}

	// 检查操作健康状态
	for opType, opStats := range stats.OperationStats {
		if opStats.Count > 0 {
			failureRate := float64(opStats.FailureCount) / float64(opStats.Count)
			if failureRate > 0.5 {
				status.Operations[opType] = "unhealthy"
			} else if failureRate > 0.2 {
				status.Operations[opType] = "degraded"
			} else {
				status.Operations[opType] = "healthy"
			}
		}
	}

	return status
}

// HealthStatus 健康状态
type HealthStatus struct {
	Overall     string            `json:"overall"`
	Providers   map[string]string `json:"providers"`
	Operations  map[string]string `json:"operations"`
	LastChecked time.Time         `json:"last_checked"`
}

// LogStatistics 记录统计信息到日志
func (rm *RetryManager) LogStatistics() {
	stats := rm.GetStatistics()

	log.Printf("Retry Statistics:")
	log.Printf("  Total Operations: %d", stats.TotalOperations)
	log.Printf("  Successful Retries: %d", stats.SuccessfulRetries)
	log.Printf("  Failed Retries: %d", stats.FailedRetries)

	if stats.TotalOperations > 0 {
		successRate := float64(stats.SuccessfulRetries) / float64(stats.TotalOperations) * 100
		log.Printf("  Success Rate: %.2f%%", successRate)
	}

	log.Printf("  Provider Statistics:")
	for provider, providerStats := range stats.ProviderStats {
		if providerStats.Count > 0 {
			successRate := float64(providerStats.SuccessCount) / float64(providerStats.Count) * 100
			log.Printf("    %s: %d operations, %.2f%% success rate", provider, providerStats.Count, successRate)
		}
	}

	log.Printf("  Operation Statistics:")
	for opType, opStats := range stats.OperationStats {
		if opStats.Count > 0 {
			successRate := float64(opStats.SuccessCount) / float64(opStats.Count) * 100
			log.Printf("    %s: %d operations, %.2f%% success rate, avg delay: %v",
				opType, opStats.Count, successRate, opStats.AverageDelay)
		}
	}
}

// 全局重试管理器实例
var globalRetryManager *RetryManager
var retryManagerOnce sync.Once

// GetGlobalRetryManager 获取全局重试管理器
func GetGlobalRetryManager() *RetryManager {
	retryManagerOnce.Do(func() {
		globalRetryManager = NewRetryManager()

		// 设置不同操作类型的重试配置
		globalRetryManager.SetRetryConfig("connection", &RetryConfig{
			MaxAttempts:   3,
			BaseDelay:     time.Second * 2,
			MaxDelay:      time.Minute * 1,
			BackoffFactor: 2.0,
			Jitter:        true,
			RetryableErrors: []ErrorType{
				ErrorTypeConnection,
				ErrorTypeTimeout,
				ErrorTypeNetworkError,
				ErrorTypeServiceUnavailable,
			},
		})

		globalRetryManager.SetRetryConfig("send_email", &RetryConfig{
			MaxAttempts:   2,
			BaseDelay:     time.Second * 5,
			MaxDelay:      time.Minute * 2,
			BackoffFactor: 1.5,
			Jitter:        true,
			RetryableErrors: []ErrorType{
				ErrorTypeRateLimit,
				ErrorTypeServiceUnavailable,
				ErrorTypeTimeout,
			},
		})

		globalRetryManager.SetRetryConfig("sync_emails", &RetryConfig{
			MaxAttempts:   2,
			BaseDelay:     time.Second * 3,
			MaxDelay:      time.Minute * 1,
			BackoffFactor: 2.0,
			Jitter:        false,
			RetryableErrors: []ErrorType{
				ErrorTypeConnection,
				ErrorTypeTimeout,
				ErrorTypeServiceUnavailable,
			},
		})
	})

	return globalRetryManager
}
