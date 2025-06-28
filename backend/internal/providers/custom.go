package providers

import (
	"context"
	"fmt"
	"strings"

	"firemail/internal/config"
	"firemail/internal/models"
)

// CustomProvider 自定义邮件提供商
type CustomProvider struct {
	*BaseProvider
	config *config.EmailProviderConfig
}

// NewCustomProvider 创建自定义邮件提供商
func NewCustomProvider(config *config.EmailProviderConfig) EmailProvider {
	provider := &CustomProvider{
		BaseProvider: NewBaseProvider(config),
		config:       config,
	}

	// 设置客户端
	provider.SetIMAPClient(NewStandardIMAPClient())
	provider.SetSMTPClient(NewStandardSMTPClient())

	return provider
}

// Connect 连接到自定义邮件服务器
func (p *CustomProvider) Connect(ctx context.Context, account *models.EmailAccount) error {
	// 验证自定义配置
	if err := p.validateCustomConfig(account); err != nil {
		return fmt.Errorf("custom provider validation failed: %w", err)
	}

	// 确保使用账户中的自定义配置
	p.ensureCustomConfig(account)

	// 使用基础连接逻辑
	return p.BaseProvider.Connect(ctx, account)
}

// validateCustomConfig 验证自定义配置
func (p *CustomProvider) validateCustomConfig(account *models.EmailAccount) error {
	// 检查是否至少配置了IMAP或SMTP
	hasIMAP := account.IMAPHost != "" && account.IMAPPort > 0
	hasSMTP := account.SMTPHost != "" && account.SMTPPort > 0

	if !hasIMAP && !hasSMTP {
		return fmt.Errorf("at least one of IMAP or SMTP configuration is required")
	}

	// 验证认证方式
	if account.AuthMethod != "password" && account.AuthMethod != "oauth2" {
		return fmt.Errorf("unsupported authentication method: %s. Custom provider supports 'password' and 'oauth2'", account.AuthMethod)
	}

	// 验证认证信息
	switch account.AuthMethod {
	case "password":
		if account.Username == "" || account.Password == "" {
			return fmt.Errorf("username and password are required for password authentication")
		}
	case "oauth2":
		tokenData, err := account.GetOAuth2Token()
		if err != nil {
			return fmt.Errorf("failed to get OAuth2 token: %w", err)
		}
		if tokenData == nil {
			return fmt.Errorf("OAuth2 token is required for OAuth2 authentication")
		}
	}

	// 验证安全设置
	if hasIMAP {
		if err := p.validateSecuritySetting(account.IMAPSecurity, "IMAP"); err != nil {
			return err
		}
	}

	if hasSMTP {
		if err := p.validateSecuritySetting(account.SMTPSecurity, "SMTP"); err != nil {
			return err
		}
	}

	return nil
}

// validateSecuritySetting 验证安全设置
func (p *CustomProvider) validateSecuritySetting(security, protocol string) error {
	validSettings := []string{"SSL", "TLS", "STARTTLS", "NONE"}
	security = strings.ToUpper(security)

	for _, valid := range validSettings {
		if security == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid %s security setting: %s. Valid options: %s", 
		protocol, security, strings.Join(validSettings, ", "))
}

// ensureCustomConfig 确保使用自定义配置
func (p *CustomProvider) ensureCustomConfig(account *models.EmailAccount) {
	// 自定义提供商使用账户中的配置，不需要覆盖
	// 但需要确保用户名设置正确
	if account.Username == "" && account.AuthMethod == "password" {
		// 如果没有设置用户名，使用邮箱地址
		account.Username = account.Email
	}
}

// TestConnection 测试自定义邮件服务器连接
func (p *CustomProvider) TestConnection(ctx context.Context, account *models.EmailAccount) error {
	// 验证配置
	if err := p.validateCustomConfig(account); err != nil {
		return err
	}

	// 分别测试IMAP和SMTP连接
	var errors []error

	// 测试IMAP连接（如果配置了）
	if account.IMAPHost != "" && account.IMAPPort > 0 {
		if err := p.testIMAPConnection(ctx, account); err != nil {
			errors = append(errors, fmt.Errorf("IMAP connection failed: %w", err))
		}
	}

	// 测试SMTP连接（如果配置了）
	if account.SMTPHost != "" && account.SMTPPort > 0 {
		if err := p.testSMTPConnection(ctx, account); err != nil {
			errors = append(errors, fmt.Errorf("SMTP connection failed: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("connection test failed: %v", errors)
	}

	return nil
}

// testIMAPConnection 测试IMAP连接
func (p *CustomProvider) testIMAPConnection(ctx context.Context, account *models.EmailAccount) error {
	// 创建临时IMAP客户端进行测试
	imapClient := NewStandardIMAPClient()

	var imapConfig IMAPClientConfig
	switch account.AuthMethod {
	case "password":
		imapConfig = IMAPClientConfig{
			Host:     account.IMAPHost,
			Port:     account.IMAPPort,
			Security: account.IMAPSecurity,
			Username: account.Username,
			Password: account.Password,
		}
	case "oauth2":
		tokenData, err := account.GetOAuth2Token()
		if err != nil {
			return fmt.Errorf("failed to get OAuth2 token: %w", err)
		}

		oauth2Token := &OAuth2Token{
			AccessToken:  tokenData.AccessToken,
			RefreshToken: tokenData.RefreshToken,
			TokenType:    tokenData.TokenType,
			Expiry:       tokenData.Expiry,
			Scope:        tokenData.Scope,
		}

		imapConfig = IMAPClientConfig{
			Host:        account.IMAPHost,
			Port:        account.IMAPPort,
			Security:    account.IMAPSecurity,
			Username:    account.Username,
			OAuth2Token: oauth2Token,
		}
	}

	// 连接并测试
	if err := imapClient.Connect(ctx, imapConfig); err != nil {
		return err
	}
	defer imapClient.Disconnect()

	// 尝试列出文件夹以验证连接
	_, err := imapClient.ListFolders(ctx)
	return err
}

// testSMTPConnection 测试SMTP连接
func (p *CustomProvider) testSMTPConnection(ctx context.Context, account *models.EmailAccount) error {
	// 创建临时SMTP客户端进行测试
	smtpClient := NewStandardSMTPClient()

	var smtpConfig SMTPClientConfig
	switch account.AuthMethod {
	case "password":
		smtpConfig = SMTPClientConfig{
			Host:     account.SMTPHost,
			Port:     account.SMTPPort,
			Security: account.SMTPSecurity,
			Username: account.Username,
			Password: account.Password,
		}
	case "oauth2":
		tokenData, err := account.GetOAuth2Token()
		if err != nil {
			return fmt.Errorf("failed to get OAuth2 token: %w", err)
		}

		oauth2Token := &OAuth2Token{
			AccessToken:  tokenData.AccessToken,
			RefreshToken: tokenData.RefreshToken,
			TokenType:    tokenData.TokenType,
			Expiry:       tokenData.Expiry,
			Scope:        tokenData.Scope,
		}

		smtpConfig = SMTPClientConfig{
			Host:        account.SMTPHost,
			Port:        account.SMTPPort,
			Security:    account.SMTPSecurity,
			Username:    account.Username,
			OAuth2Token: oauth2Token,
		}
	}

	// 连接并测试
	if err := smtpClient.Connect(ctx, smtpConfig); err != nil {
		return err
	}
	defer smtpClient.Disconnect()

	// SMTP连接成功即表示测试通过
	return nil
}

// GetProviderInfo 获取自定义提供商信息
func (p *CustomProvider) GetProviderInfo() map[string]interface{} {
	return map[string]interface{}{
		"name":         "custom",
		"display_name": "自定义IMAP/SMTP",
		"auth_methods": []string{"password", "oauth2"},
		"domains":      []string{}, // 支持任意域名
		"features": map[string]bool{
			"imap":       true,  // 可选
			"smtp":       true,  // 可选
			"oauth2":     true,  // 可选
			"push":       false, // 通常不支持
			"threading":  false, // 取决于服务器
			"labels":     false, // 取决于服务器
			"folders":    true,  // 通常支持
			"search":     true,  // 通常支持
			"idle":       true,  // 通常支持
		},
		"configuration": map[string]interface{}{
			"flexible":     true,
			"imap_only":    true, // 支持仅IMAP配置
			"smtp_only":    true, // 支持仅SMTP配置
			"custom_ports": true, // 支持自定义端口
			"all_security": true, // 支持所有安全选项
		},
		"help_text": "自定义IMAP/SMTP配置允许您连接到任何支持标准协议的邮件服务器。您可以只配置IMAP（仅收件）、只配置SMTP（仅发件）或同时配置两者。",
	}
}

// ValidateEmailAddress 验证邮箱地址（自定义提供商支持任意域名）
func (p *CustomProvider) ValidateEmailAddress(email string) error {
	// 基本的邮箱格式验证
	if !strings.Contains(email, "@") {
		return fmt.Errorf("invalid email format: missing @ symbol")
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid email format")
	}

	// 自定义提供商支持任意域名，所以只做基本格式检查
	return nil
}

// GetSupportedFeatures 获取支持的功能列表
func (p *CustomProvider) GetSupportedFeatures(account *models.EmailAccount) map[string]bool {
	features := map[string]bool{
		"folders": true,
		"search":  true,
		"idle":    true,
	}

	// 根据配置确定支持的功能
	if account.IMAPHost != "" && account.IMAPPort > 0 {
		features["imap"] = true
		features["receive"] = true
	}

	if account.SMTPHost != "" && account.SMTPPort > 0 {
		features["smtp"] = true
		features["send"] = true
	}

	if account.AuthMethod == "oauth2" {
		features["oauth2"] = true
	}

	return features
}
