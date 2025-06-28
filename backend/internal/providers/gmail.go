package providers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"firemail/internal/config"
	"firemail/internal/encoding"
	"firemail/internal/models"
	"firemail/internal/oauth2"

	goauth2 "golang.org/x/oauth2"
)

// GmailProvider Gmail邮件提供商
type GmailProvider struct {
	*BaseProvider
	oauth2Service   *oauth2.OAuth2Service
	encodingHelper  *encoding.EmailEncodingHelper
}

// newGmailProviderImpl 创建Gmail提供商实例的内部实现
func newGmailProviderImpl(config *config.EmailProviderConfig) *GmailProvider {
	base := NewBaseProvider(config)

	provider := &GmailProvider{
		BaseProvider:   base,
		oauth2Service:  oauth2.NewOAuth2Service(),
		encodingHelper: encoding.NewEmailEncodingHelper(),
	}

	// 设置IMAP和SMTP客户端
	provider.SetIMAPClient(NewStandardIMAPClient())
	provider.SetSMTPClient(NewStandardSMTPClient())

	return provider
}

// NewGmailProvider 创建Gmail提供商实例（工厂方法）
func NewGmailProvider(config *config.EmailProviderConfig) EmailProvider {
	return newGmailProviderImpl(config)
}

// Connect 连接到Gmail服务器
func (p *GmailProvider) Connect(ctx context.Context, account *models.EmailAccount) error {
	// 确保使用正确的服务器配置
	p.ensureGmailConfig(account)

	// 验证认证方式和凭据
	if err := p.validateGmailAuth(ctx, account); err != nil {
		return fmt.Errorf("Gmail authentication validation failed: %w", err)
	}

	// 使用重试机制连接
	return p.connectWithRetry(ctx, account)
}

// validateGmailAuth 验证Gmail认证方式和凭据
func (p *GmailProvider) validateGmailAuth(ctx context.Context, account *models.EmailAccount) error {
	switch account.AuthMethod {
	case "oauth2":
		return p.validateOAuth2Credentials(ctx, account)
	case "password":
		return p.validateAppPassword(account)
	default:
		return fmt.Errorf("unsupported authentication method: %s. Gmail supports 'oauth2' and 'password' (app password)", account.AuthMethod)
	}
}

// validateAppPassword 验证Gmail应用专用密码
func (p *GmailProvider) validateAppPassword(account *models.EmailAccount) error {
	// Gmail应用专用密码格式：16个字符，通常显示为4组，每组4个字符
	password := strings.ReplaceAll(account.Password, " ", "") // 移除空格

	if len(password) != 16 {
		return fmt.Errorf("Gmail app password must be 16 characters long. Please generate app password in Google Account settings")
	}

	// 检查字符是否为字母和数字
	for _, char := range password {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
			return fmt.Errorf("invalid app password format. Gmail app password should only contain letters and numbers")
		}
	}

	// 更新密码（移除空格）
	account.Password = password

	return nil
}

// validateOAuth2Credentials 验证OAuth2凭据
func (p *GmailProvider) validateOAuth2Credentials(ctx context.Context, account *models.EmailAccount) error {
	tokenData, err := account.GetOAuth2Token()
	if err != nil {
		return fmt.Errorf("failed to get OAuth2 token: %w", err)
	}

	if tokenData == nil {
		return fmt.Errorf("OAuth2 token not found. Please complete OAuth2 authentication first")
	}

	if tokenData.AccessToken == "" {
		return fmt.Errorf("OAuth2 access token is empty")
	}

	// 检查token是否过期
	if time.Now().After(tokenData.Expiry) && tokenData.RefreshToken == "" {
		return fmt.Errorf("OAuth2 token has expired and no refresh token available")
	}

	// 使用OAuth2服务验证token
	token := &goauth2.Token{
		AccessToken:  tokenData.AccessToken,
		RefreshToken: tokenData.RefreshToken,
		TokenType:    tokenData.TokenType,
		Expiry:       tokenData.Expiry,
	}

	if err := p.oauth2Service.ValidateToken(ctx, "gmail", token); err != nil {
		return fmt.Errorf("OAuth2 token validation failed: %w", err)
	}

	return nil
}

// connectWithRetry 带重试机制的连接
func (p *GmailProvider) connectWithRetry(ctx context.Context, account *models.EmailAccount) error {
	maxRetries := 3
	baseDelay := time.Second * 2

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := p.BaseProvider.Connect(ctx, account)
		if err == nil {
			return nil
		}

		// 处理Gmail特定错误
		gmailErr := p.HandleGmailError(err)

		// 某些错误不需要重试
		if p.isNonRetryableError(err) {
			return gmailErr
		}

		// 如果是最后一次尝试，返回错误
		if attempt == maxRetries-1 {
			return gmailErr
		}

		// 指数退避延迟
		delay := baseDelay * time.Duration(1<<uint(attempt))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// 继续下一次重试
		}
	}

	return fmt.Errorf("failed to connect after %d attempts", maxRetries)
}

// HandleGmailError 处理Gmail特定错误
func (p *GmailProvider) HandleGmailError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// 常见Gmail错误处理
	switch {
	case strings.Contains(errStr, "535"):
		if strings.Contains(errStr, "Username and Password not accepted") {
			return fmt.Errorf("authentication failed: please check your Gmail address and app password. Make sure you have enabled 2-Step Verification and generated app password")
		}
		return fmt.Errorf("authentication failed: invalid credentials")
	case strings.Contains(errStr, "534"):
		return fmt.Errorf("authentication failed: please enable 2-Step Verification and use app password instead of regular password")
	case strings.Contains(errStr, "550"):
		return fmt.Errorf("sending limit exceeded. Gmail has rate limits for external clients")
	case strings.Contains(errStr, "552"):
		return fmt.Errorf("message size exceeds Gmail limits (25MB)")
	case strings.Contains(errStr, "554"):
		return fmt.Errorf("message rejected: content may be considered spam or violate Gmail policies")
	case strings.Contains(errStr, "421"):
		return fmt.Errorf("service temporarily unavailable: Gmail server is busy, please try again later")
	case strings.Contains(errStr, "452"):
		return fmt.Errorf("insufficient storage: Gmail account storage is full")
	case strings.Contains(errStr, "553"):
		return fmt.Errorf("invalid recipient address or sender not authorized")
	case strings.Contains(errStr, "invalid_grant"):
		return fmt.Errorf("OAuth2 token is invalid or expired. Please re-authenticate")
	case strings.Contains(errStr, "insufficient_scope"):
		return fmt.Errorf("OAuth2 token does not have required Gmail permissions")
	default:
		return fmt.Errorf("Gmail error: %v", err)
	}
}

// isNonRetryableError 判断是否为不可重试的错误
func (p *GmailProvider) isNonRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// 认证错误、配置错误等不需要重试
	nonRetryableErrors := []string{
		"535", // 认证失败
		"534", // 需要应用专用密码
		"553", // 无效地址
		"552", // 邮件过大
		"authentication failed",
		"invalid app password",
		"invalid_grant",
		"insufficient_scope",
		"unsupported auth method",
	}

	for _, nonRetryable := range nonRetryableErrors {
		if strings.Contains(errStr, nonRetryable) {
			return true
		}
	}

	return false
}

// GetAppPasswordInstructions 获取Gmail应用专用密码设置说明
func (p *GmailProvider) GetAppPasswordInstructions() string {
	return `Gmail应用专用密码设置步骤：
1. 访问 Google 账户管理页面 (https://myaccount.google.com)
2. 点击左侧的"安全性"
3. 在"登录 Google"部分，确保已启用"两步验证"
4. 启用两步验证后，点击"应用专用密码"
5. 选择"邮件"和您的设备类型
6. 点击"生成"
7. 复制生成的16位应用专用密码
8. 在邮件客户端中使用此密码而不是您的Google密码

注意事项：
- 必须先启用两步验证才能生成应用专用密码
- 应用专用密码是16位字符，只包含字母和数字
- 每个应用专用密码只能用于一个应用
- 可以随时撤销不再使用的应用专用密码
- 如果更改了Google密码，所有应用专用密码将保持有效

OAuth2 vs 应用专用密码：
- OAuth2：更安全，支持细粒度权限控制，推荐用于新应用
- 应用专用密码：适用于不支持OAuth2的传统邮件客户端`
}

// ensureGmailConfig 确保Gmail配置正确
func (p *GmailProvider) ensureGmailConfig(account *models.EmailAccount) {
	// 如果没有设置服务器配置，使用Gmail默认配置
	if account.IMAPHost == "" {
		account.IMAPHost = p.config.IMAPHost
		account.IMAPPort = p.config.IMAPPort
		account.IMAPSecurity = p.config.IMAPSecurity
	}

	if account.SMTPHost == "" {
		account.SMTPHost = p.config.SMTPHost
		account.SMTPPort = p.config.SMTPPort
		account.SMTPSecurity = p.config.SMTPSecurity
	}
}

// TestConnection 测试Gmail连接
func (p *GmailProvider) TestConnection(ctx context.Context, account *models.EmailAccount) error {
	// 确保配置正确
	p.ensureGmailConfig(account)

	// 对于OAuth2认证，额外验证token
	if account.AuthMethod == "oauth2" {
		if err := p.validateOAuth2Token(ctx, account); err != nil {
			return fmt.Errorf("OAuth2 token validation failed: %w", err)
		}
	}

	// 调用基类测试方法
	return p.BaseProvider.TestConnection(ctx, account)
}

// validateOAuth2Token 验证OAuth2 token
func (p *GmailProvider) validateOAuth2Token(ctx context.Context, account *models.EmailAccount) error {
	tokenData, err := account.GetOAuth2Token()
	if err != nil {
		return fmt.Errorf("failed to get OAuth2 token: %w", err)
	}

	if tokenData == nil {
		return fmt.Errorf("OAuth2 token not found")
	}

	// 基本token验证
	if tokenData.AccessToken == "" {
		return fmt.Errorf("OAuth2 access token is empty")
	}

	// 检查token是否过期
	if time.Now().After(tokenData.Expiry) && tokenData.RefreshToken == "" {
		return fmt.Errorf("OAuth2 token has expired and no refresh token available")
	}

	// 验证scope是否包含邮件权限
	if tokenData.Scope != "" && !p.hasRequiredScopes(tokenData.Scope) {
		return fmt.Errorf("token does not have required mail scopes")
	}

	return nil
}

// hasRequiredScopes 检查是否有必需的权限范围
func (p *GmailProvider) hasRequiredScopes(scope string) bool {
	// Gmail需要的基本权限
	requiredScopes := []string{
		"https://mail.google.com/",
		"https://www.googleapis.com/auth/gmail.modify",
	}

	for _, required := range requiredScopes {
		if !strings.Contains(scope, required) {
			// 检查是否有更广泛的权限
			if required == "https://www.googleapis.com/auth/gmail.modify" &&
				strings.Contains(scope, "https://mail.google.com/") {
				continue // mail.google.com 包含了 gmail.modify 的权限
			}
			return false
		}
	}

	return true
}

// GetSpecialFolders 获取Gmail特殊文件夹映射
func (p *GmailProvider) GetSpecialFolders() map[string]string {
	return map[string]string{
		"inbox":  "INBOX",
		"sent":   "[Gmail]/Sent Mail",
		"drafts": "[Gmail]/Drafts",
		"trash":  "[Gmail]/Trash",
		"spam":   "[Gmail]/Spam",
		"all":    "[Gmail]/All Mail",
	}
}

// GetFolderDisplayName 获取文件夹显示名称
func (p *GmailProvider) GetFolderDisplayName(folderName string) string {
	displayNames := map[string]string{
		"INBOX":             "收件箱",
		"[Gmail]/Sent Mail": "已发送",
		"[Gmail]/Drafts":    "草稿",
		"[Gmail]/Trash":     "垃圾箱",
		"[Gmail]/Spam":      "垃圾邮件",
		"[Gmail]/All Mail":  "所有邮件",
		"[Gmail]/Important": "重要",
		"[Gmail]/Starred":   "已加星标",
	}

	if displayName, exists := displayNames[folderName]; exists {
		return displayName
	}

	return folderName
}

// SyncEmails 同步Gmail邮件
func (p *GmailProvider) SyncEmails(ctx context.Context, account *models.EmailAccount, folderName string, lastUID uint32) ([]*EmailMessage, error) {
	if !p.IsConnected() {
		if err := p.Connect(ctx, account); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
	}

	imapClient := p.IMAPClient()
	if imapClient == nil {
		return nil, fmt.Errorf("IMAP client not available")
	}

	// 获取新邮件
	emails, err := imapClient.GetNewEmails(ctx, folderName, lastUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get new emails: %w", err)
	}

	// Gmail特殊处理：处理标签和编码
	for _, email := range emails {
		p.processGmailLabels(email)
		// 重新启用编码处理，确保邮件内容正确解码
		p.processGmailEncoding(email)
	}

	return emails, nil
}

// processGmailLabels 处理Gmail标签
func (p *GmailProvider) processGmailLabels(email *EmailMessage) {
	// Gmail使用标签而不是传统的文件夹
	// 这里可以将IMAP标志转换为Gmail标签
	var labels []string

	for _, flag := range email.Flags {
		switch flag {
		case "\\Flagged":
			labels = append(labels, "Starred")
		case "\\Important":
			labels = append(labels, "Important")
		case "\\Seen":
			// 已读状态不作为标签处理
		case "\\Draft":
			labels = append(labels, "Draft")
		default:
			// 其他自定义标签
			if flag[0] != '\\' {
				labels = append(labels, flag)
			}
		}
	}

	// 设置标签（这里需要根据实际的EmailMessage结构调整）
	if len(labels) > 0 {
		email.SetLabels(labels)
	}
}

// processGmailEncoding 处理Gmail邮件编码
func (p *GmailProvider) processGmailEncoding(email *EmailMessage) {
	// 处理邮件主题编码
	if email.Subject != "" {
		email.Subject = p.encodingHelper.DecodeEmailSubject(email.Subject)
	}

	// 处理发件人编码
	if email.From != nil && email.From.Name != "" {
		email.From.Name = p.encodingHelper.DecodeEmailFrom(email.From.Name)
	}

	// 处理邮件内容编码
	if len(email.TextBody) > 0 {
		if decoded, err := p.encodingHelper.DecodeEmailContent([]byte(email.TextBody), ""); err == nil {
			email.TextBody = string(decoded)
		}
	}

	if len(email.HTMLBody) > 0 {
		if decoded, err := p.encodingHelper.DecodeEmailContent([]byte(email.HTMLBody), ""); err == nil {
			email.HTMLBody = string(decoded)
		}
	}
}

// SendEmail 发送Gmail邮件
func (p *GmailProvider) SendEmail(ctx context.Context, account *models.EmailAccount, message *OutgoingMessage) error {
	if !p.IsConnected() {
		if err := p.Connect(ctx, account); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
	}

	smtpClient := p.SMTPClient()
	if smtpClient == nil {
		return fmt.Errorf("SMTP client not available")
	}

	// Gmail特殊处理：确保发件人地址正确
	if message.From == nil {
		message.From = &models.EmailAddress{
			Address: account.Email,
			Name:    account.Name,
		}
	}

	// 发送邮件
	return smtpClient.SendEmail(ctx, message)
}

// GetQuota 获取Gmail存储配额（如果支持）
func (p *GmailProvider) GetQuota(ctx context.Context, account *models.EmailAccount) (*QuotaInfo, error) {
	// Gmail通过IMAP不直接支持配额查询
	// 可以通过Gmail API获取，但这里暂时返回nil
	return nil, fmt.Errorf("quota information not available via IMAP")
}

// GetProviderInfo 获取Gmail提供商信息
func (p *GmailProvider) GetProviderInfo() map[string]interface{} {
	return map[string]interface{}{
		"name":         "Gmail",
		"display_name": "Gmail（Google）",
		"auth_methods": []string{"oauth2", "password"},
		"domains":      []string{"gmail.com", "googlemail.com"},
		"servers": map[string]interface{}{
			"imap": map[string]interface{}{
				"host":     "imap.gmail.com",
				"port":     993,
				"security": "SSL",
			},
			"smtp": map[string]interface{}{
				"host":     "smtp.gmail.com",
				"port":     465,
				"security": "SSL",
				"alt_port": 587, // STARTTLS
			},
		},
		"features": map[string]bool{
			"imap":       true,
			"smtp":       true,
			"oauth2":     true,
			"push":       true,
			"threading":  true,
			"labels":     true,
			"folders":    false, // Gmail使用标签而不是传统文件夹
			"search":     true,
			"idle":       true,
			"extensions": true,
		},
		"limits": map[string]interface{}{
			"attachment_size":   25 * 1024 * 1024,        // 25MB
			"daily_send":        500,                     // 每日发送限制（免费账户）
			"rate_limit_window": 86400,                   // 24小时窗口
			"max_recipients":    500,                     // 单封邮件最大收件人数
			"storage_free":      15 * 1024 * 1024 * 1024, // 15GB 免费存储
			"connection_limit":  15,                      // 同时IMAP连接数限制
		},
		"oauth2": map[string]interface{}{
			"scopes": []string{
				"https://mail.google.com/",
				"https://www.googleapis.com/auth/gmail.modify",
				"https://www.googleapis.com/auth/gmail.readonly",
				"https://www.googleapis.com/auth/gmail.send",
			},
			"auth_url":  "https://accounts.google.com/o/oauth2/auth",
			"token_url": "https://oauth2.googleapis.com/token",
		},
		"error_codes": map[string]string{
			"535": "认证失败，请检查邮箱地址和应用专用密码",
			"534": "需要启用两步验证并使用应用专用密码",
			"550": "发送频率超限，请稍后重试",
			"552": "邮件大小超过25MB限制",
			"554": "邮件被拒绝，内容可能被识别为垃圾邮件",
			"421": "服务暂时不可用，服务器繁忙",
			"452": "存储空间不足，账户存储已满",
			"553": "收件人地址无效或发件人未授权",
		},
		"help_urls": map[string]string{
			"google_account": "https://myaccount.google.com/",
			"app_passwords":  "https://support.google.com/accounts/answer/185833",
			"two_factor":     "https://support.google.com/accounts/answer/185839",
			"gmail_help":     "https://support.google.com/gmail/",
			"oauth2_setup":   "https://developers.google.com/gmail/imap/oauth2",
			"imap_setup":     "https://support.google.com/mail/answer/7126229",
		},
		"auth_instructions": map[string]string{
			"app_password": p.GetAppPasswordInstructions(),
			"oauth2":       "Use OAuth2 for enhanced security and fine-grained permissions",
		},
		"requirements": map[string]interface{}{
			"two_factor_auth": true,  // 应用专用密码需要
			"oauth2_app":      false, // OAuth2需要注册应用
		},
	}
}

// QuotaInfo 配额信息
type QuotaInfo struct {
	Used  int64  `json:"used"`
	Total int64  `json:"total"`
	Unit  string `json:"unit"`
}
