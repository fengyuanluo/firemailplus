package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Environment 环境配置管理器
type Environment struct {
	// 基础配置
	Mode        string // development, production, test
	Debug       bool
	LogLevel    string
	
	// 数据库配置
	DatabaseURL string
	IsMemoryDB  bool
	
	// 邮件服务配置
	EnableRealEmailSync bool
	MockEmailProviders  bool
	
	// 性能配置
	MaxConcurrency      int
	RequestTimeout      time.Duration
	
	// 功能开关
	EnableEnhancedDedup bool
	EnableSSE           bool
	EnableMetrics       bool
}

// Env 全局环境配置实例
var Env *Environment

// init 初始化配置
func init() {
	Env = LoadEnvironment()
}

// LoadEnvironment 加载环境配置
func LoadEnvironment() *Environment {
	env := &Environment{
		// 默认值
		Mode:                "development",
		Debug:               false,
		LogLevel:            "info",
		DatabaseURL:         ":memory:",
		IsMemoryDB:          true,
		EnableRealEmailSync: false,
		MockEmailProviders:  true,
		MaxConcurrency:      10,
		RequestTimeout:      30 * time.Second,
		EnableEnhancedDedup: true,
		EnableSSE:           true,
		EnableMetrics:       false,
	}
	
	// 从环境变量加载配置
	if mode := os.Getenv("GIN_MODE"); mode != "" {
		env.Mode = mode
	}
	
	if mode := os.Getenv("GO_ENV"); mode != "" {
		env.Mode = mode
	}
	
	if debug := os.Getenv("DEBUG"); debug != "" {
		env.Debug = strings.ToLower(debug) == "true"
	}
	
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		env.LogLevel = logLevel
	}
	
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		env.DatabaseURL = dbURL
		env.IsMemoryDB = strings.Contains(dbURL, ":memory:")
	}
	
	if enableSync := os.Getenv("ENABLE_REAL_EMAIL_SYNC"); enableSync != "" {
		env.EnableRealEmailSync = strings.ToLower(enableSync) == "true"
	}
	
	if mockProviders := os.Getenv("MOCK_EMAIL_PROVIDERS"); mockProviders != "" {
		env.MockEmailProviders = strings.ToLower(mockProviders) == "true"
	}
	
	if maxConcurrency := os.Getenv("MAX_CONCURRENCY"); maxConcurrency != "" {
		if val, err := strconv.Atoi(maxConcurrency); err == nil {
			env.MaxConcurrency = val
		}
	}
	
	if timeout := os.Getenv("REQUEST_TIMEOUT"); timeout != "" {
		if val, err := time.ParseDuration(timeout); err == nil {
			env.RequestTimeout = val
		}
	}
	
	if enableDedup := os.Getenv("ENABLE_ENHANCED_DEDUP"); enableDedup != "" {
		env.EnableEnhancedDedup = strings.ToLower(enableDedup) == "true"
	}
	
	if enableSSE := os.Getenv("ENABLE_SSE"); enableSSE != "" {
		env.EnableSSE = strings.ToLower(enableSSE) == "true"
	}
	
	if enableMetrics := os.Getenv("ENABLE_METRICS"); enableMetrics != "" {
		env.EnableMetrics = strings.ToLower(enableMetrics) == "true"
	}
	
	// 根据模式调整配置
	switch env.Mode {
	case "test":
		env.adjustForTestMode()
	case "production":
		env.adjustForProductionMode()
	case "development":
		env.adjustForDevelopmentMode()
	}

	// 调试信息：打印环境变量和最终配置
	fmt.Printf("Environment variables: ENABLE_REAL_EMAIL_SYNC=%s, MOCK_EMAIL_PROVIDERS=%s\n",
		os.Getenv("ENABLE_REAL_EMAIL_SYNC"), os.Getenv("MOCK_EMAIL_PROVIDERS"))
	fmt.Printf("Environment loaded: Mode=%s, EnableRealEmailSync=%t, MockEmailProviders=%t\n",
		env.Mode, env.EnableRealEmailSync, env.MockEmailProviders)

	return env
}

// IsTestMode 检查是否为测试模式
func (e *Environment) IsTestMode() bool {
	return e.Mode == "test"
}

// IsProductionMode 检查是否为生产模式
func (e *Environment) IsProductionMode() bool {
	return e.Mode == "production"
}

// IsDevelopmentMode 检查是否为开发模式
func (e *Environment) IsDevelopmentMode() bool {
	return e.Mode == "development"
}

// ShouldMockEmailProviders 是否应该模拟邮件提供商
func (e *Environment) ShouldMockEmailProviders() bool {
	// 强制禁用模拟，始终使用真实环境
	return false
}

// ShouldEnableRealEmailSync 是否应该启用真实邮件同步
func (e *Environment) ShouldEnableRealEmailSync() bool {
	// 强制启用真实邮件同步
	return true
}

// ShouldEnableEnhancedDedup 是否应该启用增强去重
func (e *Environment) ShouldEnableEnhancedDedup() bool {
	return e.EnableEnhancedDedup && !e.IsTestMode()
}

// adjustForTestMode 调整测试模式配置
func (e *Environment) adjustForTestMode() {
	e.Debug = true
	e.LogLevel = "debug"
	e.IsMemoryDB = true
	e.EnableRealEmailSync = false
	e.MockEmailProviders = true
	e.EnableEnhancedDedup = false // 测试模式下禁用增强功能
	e.EnableMetrics = false
	e.RequestTimeout = 5 * time.Second
}

// adjustForProductionMode 调整生产模式配置
func (e *Environment) adjustForProductionMode() {
	e.Debug = false
	e.LogLevel = "warn"
	e.EnableRealEmailSync = true
	e.MockEmailProviders = false
	e.EnableEnhancedDedup = true
	e.EnableMetrics = true
	e.RequestTimeout = 30 * time.Second
}

// adjustForDevelopmentMode 调整开发模式配置
func (e *Environment) adjustForDevelopmentMode() {
	e.Debug = true
	e.LogLevel = "debug"
	// 强制启用真实邮件同步
	e.EnableRealEmailSync = true
	// 强制禁用模拟提供商
	e.MockEmailProviders = false
	e.EnableEnhancedDedup = true
	e.EnableMetrics = false
	e.RequestTimeout = 10 * time.Second
}

// GetDatabaseConfig 获取数据库配置
func (e *Environment) GetDatabaseConfig() map[string]interface{} {
	return map[string]interface{}{
		"url":         e.DatabaseURL,
		"is_memory":   e.IsMemoryDB,
		"debug":       e.Debug,
		"log_level":   e.LogLevel,
	}
}

// GetEmailConfig 获取邮件配置
func (e *Environment) GetEmailConfig() map[string]interface{} {
	return map[string]interface{}{
		"enable_real_sync":    e.EnableRealEmailSync,
		"mock_providers":      e.MockEmailProviders,
		"enable_enhanced_dedup": e.EnableEnhancedDedup,
		"max_concurrency":     e.MaxConcurrency,
		"request_timeout":     e.RequestTimeout,
	}
}

// GetFeatureFlags 获取功能开关
func (e *Environment) GetFeatureFlags() map[string]bool {
	return map[string]bool{
		"enhanced_dedup": e.EnableEnhancedDedup,
		"sse":            e.EnableSSE,
		"metrics":        e.EnableMetrics,
		"real_email_sync": e.EnableRealEmailSync,
		"mock_providers":  e.MockEmailProviders,
	}
}

// Validate 验证配置
func (e *Environment) Validate() error {
	// 验证必要的配置项
	if e.Mode == "" {
		e.Mode = "development"
	}
	
	if e.MaxConcurrency <= 0 {
		e.MaxConcurrency = 10
	}
	
	if e.RequestTimeout <= 0 {
		e.RequestTimeout = 30 * time.Second
	}
	
	return nil
}

// String 返回配置的字符串表示
func (e *Environment) String() string {
	return fmt.Sprintf("Environment{Mode: %s, Debug: %t, IsMemoryDB: %t, EnableRealEmailSync: %t, MockEmailProviders: %t}",
		e.Mode, e.Debug, e.IsMemoryDB, e.EnableRealEmailSync, e.MockEmailProviders)
}
