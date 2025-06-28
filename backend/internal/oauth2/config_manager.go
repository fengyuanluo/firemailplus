package oauth2

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/microsoft"
)

// OAuth2Config OAuth2配置接口
type OAuth2Config interface {
	GetConfig() *oauth2.Config
	GetProvider() string
	Validate() error
	GetScopes() []string
	GetEndpoint() oauth2.Endpoint
}

// OAuth2ConfigManager OAuth2配置管理器接口
type OAuth2ConfigManager interface {
	GetConfig(provider string) (OAuth2Config, error)
	RegisterConfig(provider string, config OAuth2Config) error
	ValidateConfig(provider string) error
	RefreshConfig(provider string) error
	ListProviders() []string
}

// StandardOAuth2Config 标准OAuth2配置实现
type StandardOAuth2Config struct {
	Provider     string   `json:"provider"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURL  string   `json:"redirect_url"`
	Scopes       []string `json:"scopes"`
	Endpoint     oauth2.Endpoint `json:"endpoint"`
	
	// 配置元数据
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// GetConfig 获取OAuth2配置
func (c *StandardOAuth2Config) GetConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		RedirectURL:  c.RedirectURL,
		Scopes:       c.Scopes,
		Endpoint:     c.Endpoint,
	}
}

// GetProvider 获取提供商名称
func (c *StandardOAuth2Config) GetProvider() string {
	return c.Provider
}

// Validate 验证配置
func (c *StandardOAuth2Config) Validate() error {
	if c.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	
	if c.ClientID == "" {
		return fmt.Errorf("client_id is required for provider %s", c.Provider)
	}
	
	if c.ClientSecret == "" {
		return fmt.Errorf("client_secret is required for provider %s", c.Provider)
	}
	
	if c.RedirectURL == "" {
		return fmt.Errorf("redirect_url is required for provider %s", c.Provider)
	}
	
	if len(c.Scopes) == 0 {
		return fmt.Errorf("scopes are required for provider %s", c.Provider)
	}
	
	// 验证Endpoint
	if c.Endpoint.AuthURL == "" || c.Endpoint.TokenURL == "" {
		return fmt.Errorf("invalid endpoint configuration for provider %s", c.Provider)
	}
	
	return nil
}

// GetScopes 获取权限范围
func (c *StandardOAuth2Config) GetScopes() []string {
	return c.Scopes
}

// GetEndpoint 获取端点配置
func (c *StandardOAuth2Config) GetEndpoint() oauth2.Endpoint {
	return c.Endpoint
}

// StandardOAuth2ConfigManager 标准OAuth2配置管理器
type StandardOAuth2ConfigManager struct {
	configs map[string]OAuth2Config
	mutex   sync.RWMutex
	
	// 配置加载器
	configLoader ConfigLoader
	
	// 配置验证器
	validator ConfigValidator
	
	// 配置缓存
	cache ConfigCache
}

// ConfigLoader 配置加载器接口
type ConfigLoader interface {
	LoadConfig(provider string) (OAuth2Config, error)
	LoadAllConfigs() (map[string]OAuth2Config, error)
}

// ConfigValidator 配置验证器接口
type ConfigValidator interface {
	ValidateConfig(config OAuth2Config) error
	ValidateProvider(provider string) error
}

// ConfigCache 配置缓存接口
type ConfigCache interface {
	Get(provider string) (OAuth2Config, bool)
	Set(provider string, config OAuth2Config, ttl time.Duration)
	Delete(provider string)
	Clear()
}

// NewStandardOAuth2ConfigManager 创建标准OAuth2配置管理器
func NewStandardOAuth2ConfigManager() OAuth2ConfigManager {
	manager := &StandardOAuth2ConfigManager{
		configs:      make(map[string]OAuth2Config),
		configLoader: NewEnvironmentConfigLoader(),
		validator:    NewStandardConfigValidator(),
		cache:        NewMemoryConfigCache(),
	}
	
	// 初始化内置配置
	manager.initializeBuiltinConfigs()
	
	return manager
}

// GetConfig 获取OAuth2配置
func (m *StandardOAuth2ConfigManager) GetConfig(provider string) (OAuth2Config, error) {
	// 首先检查缓存
	if config, exists := m.cache.Get(provider); exists {
		return config, nil
	}
	
	m.mutex.RLock()
	config, exists := m.configs[provider]
	m.mutex.RUnlock()
	
	if !exists {
		// 尝试从配置加载器加载
		loadedConfig, err := m.configLoader.LoadConfig(provider)
		if err != nil {
			return nil, fmt.Errorf("config not found for provider %s: %w", provider, err)
		}
		
		// 验证配置
		if err := m.validator.ValidateConfig(loadedConfig); err != nil {
			return nil, fmt.Errorf("invalid config for provider %s: %w", provider, err)
		}
		
		// 注册配置
		m.RegisterConfig(provider, loadedConfig)
		config = loadedConfig
	}
	
	// 缓存配置
	m.cache.Set(provider, config, 30*time.Minute)
	
	return config, nil
}

// RegisterConfig 注册OAuth2配置
func (m *StandardOAuth2ConfigManager) RegisterConfig(provider string, config OAuth2Config) error {
	if err := m.validator.ValidateConfig(config); err != nil {
		return fmt.Errorf("invalid config for provider %s: %w", provider, err)
	}
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.configs[provider] = config
	
	// 清除缓存以强制重新加载
	m.cache.Delete(provider)
	
	return nil
}

// ValidateConfig 验证配置
func (m *StandardOAuth2ConfigManager) ValidateConfig(provider string) error {
	config, err := m.GetConfig(provider)
	if err != nil {
		return err
	}
	
	return config.Validate()
}

// RefreshConfig 刷新配置
func (m *StandardOAuth2ConfigManager) RefreshConfig(provider string) error {
	// 清除缓存
	m.cache.Delete(provider)
	
	// 重新加载配置
	_, err := m.GetConfig(provider)
	return err
}

// ListProviders 列出所有提供商
func (m *StandardOAuth2ConfigManager) ListProviders() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	providers := make([]string, 0, len(m.configs))
	for provider := range m.configs {
		providers = append(providers, provider)
	}
	
	return providers
}

// initializeBuiltinConfigs 初始化内置配置
func (m *StandardOAuth2ConfigManager) initializeBuiltinConfigs() {
	// Gmail配置
	gmailConfig := &StandardOAuth2Config{
		Provider:    "gmail",
		Name:        "Gmail",
		Description: "Google Gmail OAuth2 Configuration",
		Version:     "1.0",
		Scopes:      []string{"https://www.googleapis.com/auth/gmail.readonly", "https://www.googleapis.com/auth/gmail.send"},
		Endpoint:    google.Endpoint,
		UpdatedAt:   time.Now(),
	}
	
	// Outlook配置
	outlookConfig := &StandardOAuth2Config{
		Provider:    "outlook",
		Name:        "Outlook",
		Description: "Microsoft Outlook OAuth2 Configuration",
		Version:     "1.0",
		Scopes:      []string{"https://graph.microsoft.com/IMAP.AccessAsUser.All", "https://graph.microsoft.com/SMTP.Send"},
		Endpoint:    microsoft.AzureADEndpoint("common"),
		UpdatedAt:   time.Now(),
	}
	
	// 注册内置配置（不验证，因为缺少客户端凭据）
	m.mutex.Lock()
	m.configs["gmail"] = gmailConfig
	m.configs["outlook"] = outlookConfig
	m.mutex.Unlock()
}

// EnvironmentConfigLoader 环境变量配置加载器
type EnvironmentConfigLoader struct{}

// NewEnvironmentConfigLoader 创建环境变量配置加载器
func NewEnvironmentConfigLoader() ConfigLoader {
	return &EnvironmentConfigLoader{}
}

// LoadConfig 从环境变量加载配置
func (l *EnvironmentConfigLoader) LoadConfig(provider string) (OAuth2Config, error) {
	upperProvider := strings.ToUpper(provider)
	
	clientID := os.Getenv(fmt.Sprintf("%s_CLIENT_ID", upperProvider))
	clientSecret := os.Getenv(fmt.Sprintf("%s_CLIENT_SECRET", upperProvider))
	redirectURL := os.Getenv(fmt.Sprintf("%s_REDIRECT_URL", upperProvider))
	
	if clientID == "" || clientSecret == "" || redirectURL == "" {
		return nil, fmt.Errorf("missing environment variables for provider %s", provider)
	}
	
	config := &StandardOAuth2Config{
		Provider:     provider,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		UpdatedAt:    time.Now(),
	}
	
	// 根据提供商设置特定配置
	switch provider {
	case "gmail":
		config.Scopes = []string{"https://www.googleapis.com/auth/gmail.readonly", "https://www.googleapis.com/auth/gmail.send"}
		config.Endpoint = google.Endpoint
	case "outlook":
		config.Scopes = []string{"https://graph.microsoft.com/IMAP.AccessAsUser.All", "https://graph.microsoft.com/SMTP.Send"}
		config.Endpoint = microsoft.AzureADEndpoint("common")
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
	
	return config, nil
}

// LoadAllConfigs 加载所有配置
func (l *EnvironmentConfigLoader) LoadAllConfigs() (map[string]OAuth2Config, error) {
	configs := make(map[string]OAuth2Config)
	
	providers := []string{"gmail", "outlook"}
	for _, provider := range providers {
		if config, err := l.LoadConfig(provider); err == nil {
			configs[provider] = config
		}
	}
	
	return configs, nil
}

// StandardConfigValidator 标准配置验证器
type StandardConfigValidator struct{}

// NewStandardConfigValidator 创建标准配置验证器
func NewStandardConfigValidator() ConfigValidator {
	return &StandardConfigValidator{}
}

// ValidateConfig 验证配置
func (v *StandardConfigValidator) ValidateConfig(config OAuth2Config) error {
	return config.Validate()
}

// ValidateProvider 验证提供商
func (v *StandardConfigValidator) ValidateProvider(provider string) error {
	supportedProviders := []string{"gmail", "outlook"}
	
	for _, supported := range supportedProviders {
		if provider == supported {
			return nil
		}
	}
	
	return fmt.Errorf("unsupported provider: %s", provider)
}

// MemoryConfigCache 内存配置缓存
type MemoryConfigCache struct {
	cache map[string]cacheItem
	mutex sync.RWMutex
}

type cacheItem struct {
	config    OAuth2Config
	expiresAt time.Time
}

// NewMemoryConfigCache 创建内存配置缓存
func NewMemoryConfigCache() ConfigCache {
	cache := &MemoryConfigCache{
		cache: make(map[string]cacheItem),
	}
	
	// 启动清理goroutine
	go cache.cleanup()
	
	return cache
}

// Get 获取缓存配置
func (c *MemoryConfigCache) Get(provider string) (OAuth2Config, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	item, exists := c.cache[provider]
	if !exists || time.Now().After(item.expiresAt) {
		return nil, false
	}
	
	return item.config, true
}

// Set 设置缓存配置
func (c *MemoryConfigCache) Set(provider string, config OAuth2Config, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.cache[provider] = cacheItem{
		config:    config,
		expiresAt: time.Now().Add(ttl),
	}
}

// Delete 删除缓存配置
func (c *MemoryConfigCache) Delete(provider string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	delete(c.cache, provider)
}

// Clear 清空缓存
func (c *MemoryConfigCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.cache = make(map[string]cacheItem)
}

// cleanup 清理过期缓存
func (c *MemoryConfigCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		c.mutex.Lock()
		now := time.Now()
		for provider, item := range c.cache {
			if now.After(item.expiresAt) {
				delete(c.cache, provider)
			}
		}
		c.mutex.Unlock()
	}
}
