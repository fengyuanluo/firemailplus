package providers

import (
	"fmt"
	"strings"

	"firemail/internal/config"
	"firemail/internal/models"
)

// ProviderFactoryInterface 提供商工厂接口
type ProviderFactoryInterface interface {
	CreateProviderForAccount(account *models.EmailAccount) (EmailProvider, error)
}

// ProviderFactory 提供商工厂
type ProviderFactory struct {
	providers map[string]func(*config.EmailProviderConfig) EmailProvider
}

// NewProviderFactory 创建提供商工厂
func NewProviderFactory() *ProviderFactory {
	factory := &ProviderFactory{
		providers: make(map[string]func(*config.EmailProviderConfig) EmailProvider),
	}

	// 注册内置提供商
	factory.RegisterProvider("gmail", NewGmailProvider)
	factory.RegisterProvider("outlook", NewOutlookProvider)
	factory.RegisterProvider("qq", NewQQProvider)
	factory.RegisterProvider("163", NewNetEaseProvider)
	factory.RegisterProvider("icloud", NewiCloudProvider)
	factory.RegisterProvider("custom", NewCustomProvider)
	// TODO: 实现新浪邮箱提供商
	// factory.RegisterProvider("sina", NewSinaProvider)

	return factory
}

// RegisterProvider 注册提供商
func (f *ProviderFactory) RegisterProvider(name string, constructor func(*config.EmailProviderConfig) EmailProvider) {
	f.providers[name] = constructor
}

// CreateProvider 创建提供商实例
func (f *ProviderFactory) CreateProvider(name string) (EmailProvider, error) {
	constructor, exists := f.providers[name]
	if !exists {
		return nil, fmt.Errorf("unknown provider: %s", name)
	}

	// 获取提供商配置
	providerConfig := config.GetProviderByName(name)
	if providerConfig == nil {
		return nil, fmt.Errorf("provider config not found: %s", name)
	}

	return constructor(providerConfig), nil
}

// CreateProviderForAccount 为邮件账户创建提供商
func (f *ProviderFactory) CreateProviderForAccount(account *models.EmailAccount) (EmailProvider, error) {
	// 如果指定了提供商，直接使用
	if account.Provider != "" {
		return f.CreateProvider(account.Provider)
	}

	// 根据邮箱域名自动检测提供商
	domain := extractDomain(account.Email)
	if domain == "" {
		return nil, fmt.Errorf("invalid email address: %s", account.Email)
	}

	// 查找匹配的提供商
	providerConfig := config.GetProviderByDomain(domain)
	if providerConfig == nil {
		return nil, fmt.Errorf("no provider found for domain: %s", domain)
	}

	return f.CreateProvider(providerConfig.Name)
}

// GetAvailableProviders 获取可用的提供商列表
func (f *ProviderFactory) GetAvailableProviders() []string {
	providers := make([]string, 0, len(f.providers))
	for name := range f.providers {
		providers = append(providers, name)
	}
	return providers
}

// GetProviderConfig 获取提供商配置
func (f *ProviderFactory) GetProviderConfig(name string) *config.EmailProviderConfig {
	return config.GetProviderByName(name)
}

// GetProviderConfigByDomain 根据域名获取提供商配置
func (f *ProviderFactory) GetProviderConfigByDomain(domain string) *config.EmailProviderConfig {
	return config.GetProviderByDomain(domain)
}

// DetectProvider 检测邮箱的提供商
func (f *ProviderFactory) DetectProvider(email string) *config.EmailProviderConfig {
	domain := extractDomain(email)
	if domain == "" {
		return nil
	}
	return config.GetProviderByDomain(domain)
}

// ValidateProviderConfig 验证提供商配置
func (f *ProviderFactory) ValidateProviderConfig(account *models.EmailAccount) error {
	provider, err := f.CreateProviderForAccount(account)
	if err != nil {
		return err
	}

	// 检查认证方式是否支持
	supportedMethods := provider.GetSupportedAuthMethods()
	supported := false
	for _, method := range supportedMethods {
		if method == account.AuthMethod {
			supported = true
			break
		}
	}

	if !supported {
		return fmt.Errorf("auth method %s not supported by provider %s", account.AuthMethod, provider.GetName())
	}

	// 检查必要的配置字段
	switch account.AuthMethod {
	case "password":
		if account.Username == "" || account.Password == "" {
			return fmt.Errorf("username and password are required for password auth")
		}
	case "oauth2":
		tokenData, err := account.GetOAuth2Token()
		if err != nil {
			return fmt.Errorf("failed to get OAuth2 token: %w", err)
		}
		if tokenData == nil {
			return fmt.Errorf("OAuth2 token is required for oauth2 auth")
		}
	}

	// 检查服务器配置（自定义提供商允许只配置IMAP或SMTP）
	if account.Provider == "custom" {
		// 自定义提供商至少需要IMAP或SMTP配置
		hasIMAP := account.IMAPHost != "" && account.IMAPPort != 0
		hasSMTP := account.SMTPHost != "" && account.SMTPPort != 0

		if !hasIMAP && !hasSMTP {
			return fmt.Errorf("custom provider requires at least IMAP or SMTP server configuration")
		}
	} else {
		// 其他提供商需要完整的IMAP和SMTP配置
		if account.IMAPHost == "" || account.IMAPPort == 0 {
			return fmt.Errorf("IMAP server configuration is required")
		}

		if account.SMTPHost == "" || account.SMTPPort == 0 {
			return fmt.Errorf("SMTP server configuration is required")
		}
	}

	return nil
}

// extractDomain 从邮箱地址中提取域名
func extractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(parts[1])
}

// 提供商构造函数在各自的实现文件中定义
