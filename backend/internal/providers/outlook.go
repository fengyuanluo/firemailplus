package providers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"firemail/internal/config"
	"firemail/internal/models"
)

// OutlookProvider Outlook邮件提供商
type OutlookProvider struct {
	*BaseProvider
	oauth2Client *OutlookOAuth2Client
}

// newOutlookProviderImpl 创建Outlook提供商实例的内部实现
func newOutlookProviderImpl(config *config.EmailProviderConfig) *OutlookProvider {
	base := NewBaseProvider(config)

	provider := &OutlookProvider{
		BaseProvider: base,
	}

	// 设置IMAP和SMTP客户端
	provider.SetIMAPClient(NewStandardIMAPClient())
	provider.SetSMTPClient(NewStandardSMTPClient())

	return provider
}

// NewOutlookProvider 创建Outlook提供商实例（工厂方法）
func NewOutlookProvider(config *config.EmailProviderConfig) EmailProvider {
	return newOutlookProviderImpl(config)
}

// Connect 连接到Outlook服务器
func (p *OutlookProvider) Connect(ctx context.Context, account *models.EmailAccount) error {
	// 设置OAuth2客户端
	if account.AuthMethod == "oauth2" && p.oauth2Client == nil {
		// 从账户的OAuth2Token中获取client_id
		tokenData, err := account.GetOAuth2Token()
		if err != nil {
			return fmt.Errorf("failed to get OAuth2 token data: %w", err)
		}

		if tokenData == nil {
			return fmt.Errorf("OAuth2 token data not found")
		}

		// 从token数据中提取client_id（在手动配置时存储）
		clientID := tokenData.ClientID
		if clientID == "" {
			return fmt.Errorf("OAuth2 client ID not found in token data")
		}

		// 对于手动配置，我们不需要client_secret和redirect_url
		// 因为我们只使用refresh token来获取access token
		p.oauth2Client = NewOutlookOAuth2Client(clientID, "", "")
		p.SetOAuth2Client(p.oauth2Client)
	}

	// 确保使用正确的服务器配置
	p.ensureOutlookConfig(account)

	// 验证认证方式和凭据，并刷新token
	if err := p.validateOutlookAuth(ctx, account); err != nil {
		return fmt.Errorf("outlook authentication validation failed: %w", err)
	}

	// 使用重试机制连接
	return p.connectWithRetry(ctx, account)
}

// validateOutlookAuth 验证Outlook认证方式和凭据 - 只支持OAuth2，并刷新token
func (p *OutlookProvider) validateOutlookAuth(ctx context.Context, account *models.EmailAccount) error {
	if account.AuthMethod != "oauth2" {
		return fmt.Errorf("only OAuth2 authentication is supported for Outlook")
	}
	// 调用token验证和刷新
	return p.validateOAuth2Token(ctx, account)
}

// connectWithRetry 带重试机制的连接
func (p *OutlookProvider) connectWithRetry(ctx context.Context, account *models.EmailAccount) error {
	maxRetries := 3
	baseDelay := time.Second * 2

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := p.BaseProvider.Connect(ctx, account)
		if err == nil {
			return nil
		}

		// 处理Outlook特定错误
		outlookErr := p.HandleOutlookError(err)

		// 某些错误不需要重试
		if p.isNonRetryableError(err) {
			return outlookErr
		}

		// 如果是最后一次尝试，返回错误
		if attempt == maxRetries-1 {
			return outlookErr
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

// HandleOutlookError 处理Outlook特定错误
func (p *OutlookProvider) HandleOutlookError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// 添加调试信息
	fmt.Printf("🔍 [OUTLOOK ERROR] Original error: %v\n", err)
	fmt.Printf("🔍 [OUTLOOK ERROR] Error type: %T\n", err)
	fmt.Printf("🔍 [OUTLOOK ERROR] Error string: %s\n", errStr)

	// 常见Outlook错误处理
	switch {
	case strings.Contains(errStr, "535"):
		return fmt.Errorf("authentication failed: please check your credentials. For personal accounts, use OAuth2. For enterprise accounts, ensure basic auth is enabled")
	case strings.Contains(errStr, "534"):
		return fmt.Errorf("authentication mechanism not supported. Please use OAuth2 for Outlook.com accounts")
	case strings.Contains(errStr, "550"):
		return fmt.Errorf("sending limit exceeded. Outlook has rate limits for external clients")
	case strings.Contains(errStr, "552"):
		return fmt.Errorf("message size exceeds Outlook limits (25MB for personal, varies for enterprise)")
	case strings.Contains(errStr, "554"):
		return fmt.Errorf("message rejected: content may be considered spam or violate Outlook policies")
	case strings.Contains(errStr, "421"):
		return fmt.Errorf("service temporarily unavailable: Outlook server is busy, please try again later")
	case strings.Contains(errStr, "452"):
		return fmt.Errorf("insufficient storage: Outlook mailbox is full")
	case strings.Contains(errStr, "553"):
		return fmt.Errorf("invalid recipient address or sender not authorized")
	case strings.Contains(errStr, "invalid_grant"):
		return fmt.Errorf("oAuth2 token is invalid or expired. Please re-authenticate")
	case strings.Contains(errStr, "insufficient_scope"):
		return fmt.Errorf("OAuth2 token does not have required Outlook permissions")
	case strings.Contains(errStr, "AADSTS"):
		return fmt.Errorf("azure AD authentication error: %v. Please check your OAuth2 configuration", err)
	default:
		return fmt.Errorf("outlook error: %v", err)
	}
}

// isNonRetryableError 判断是否为不可重试的错误
func (p *OutlookProvider) isNonRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// 认证错误、配置错误等不需要重试
	nonRetryableErrors := []string{
		"535", // 认证失败
		"534", // 认证机制不支持
		"553", // 无效地址
		"552", // 邮件过大
		"authentication failed",
		"invalid_grant",
		"insufficient_scope",
		"AADSTS", // Azure AD错误
		"unsupported auth method",
	}

	for _, nonRetryable := range nonRetryableErrors {
		if strings.Contains(errStr, nonRetryable) {
			return true
		}
	}

	return false
}

// GetOAuth2Instructions 获取OAuth2设置说明
func (p *OutlookProvider) GetOAuth2Instructions() string {
	return `Outlook OAuth2设置步骤：
1. 访问 Azure Portal (https://portal.azure.com)
2. 注册新的应用程序或使用现有应用
3. 配置重定向URI
4. 获取客户端ID和客户端密钥（如果是机密客户端）
5. 配置API权限：
   - IMAP.AccessAsUser.All
   - SMTP.Send
   - Mail.Read（可选）
   - Mail.ReadWrite（可选）
6. 管理员同意权限（企业账户）

个人账户 vs 企业账户：
- 个人账户（outlook.com, hotmail.com等）：必须使用OAuth2
- 企业账户：可以使用OAuth2或基本认证（如果管理员启用）

重要提醒：
- Microsoft已弃用基本认证，强烈建议使用OAuth2
- 个人Microsoft账户不再支持基本认证
- 企业账户的基本认证需要管理员明确启用

OAuth2权限说明：
- IMAP.AccessAsUser.All: 允许应用代表用户访问IMAP
- SMTP.Send: 允许应用代表用户发送邮件
- offline_access: 获取刷新令牌以长期访问`
}

// ensureOutlookConfig 确保Outlook配置正确
func (p *OutlookProvider) ensureOutlookConfig(account *models.EmailAccount) {
	// 如果没有设置服务器配置，使用Outlook默认配置
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

	// Outlook用户名通常是完整的邮箱地址
	if account.Username == "" {
		account.Username = account.Email
	}
}

// TestConnection 测试Outlook连接
func (p *OutlookProvider) TestConnection(ctx context.Context, account *models.EmailAccount) error {
	// 确保配置正确
	p.ensureOutlookConfig(account)

	// 对于OAuth2认证，验证token
	if account.AuthMethod == "oauth2" {
		if err := p.validateOAuth2Token(ctx, account); err != nil {
			return fmt.Errorf("OAuth2 token validation failed: %w", err)
		}
	}

	// 调用基类测试方法
	return p.BaseProvider.TestConnection(ctx, account)
}

// validateOAuth2Token 验证OAuth2 token - 简化版本，按照Python代码逻辑
func (p *OutlookProvider) validateOAuth2Token(ctx context.Context, account *models.EmailAccount) error {
	tokenData, err := account.GetOAuth2Token()
	if err != nil {
		return fmt.Errorf("failed to get OAuth2 token: %w", err)
	}

	if tokenData == nil {
		return fmt.Errorf("OAuth2 token not found")
	}

	if tokenData.RefreshToken == "" {
		return fmt.Errorf("refresh token is required")
	}

	// 总是刷新token以获取最新的access_token，按照Python代码逻辑
	if p.oauth2Client != nil {
		newToken, err := p.oauth2Client.RefreshToken(ctx, tokenData.RefreshToken)
		if err != nil {
			return fmt.Errorf("failed to refresh token: %w", err)
		}

		// 更新账户中的token
		newTokenData := &models.OAuth2TokenData{
			AccessToken:  newToken.AccessToken,
			RefreshToken: newToken.RefreshToken, // 使用新的refresh token
			TokenType:    newToken.TokenType,
			Expiry:       newToken.Expiry,
			Scope:        tokenData.Scope,
			ClientID:     tokenData.ClientID,
		}

		if err := account.SetOAuth2Token(newTokenData); err != nil {
			return fmt.Errorf("failed to save refreshed token: %w", err)
		}
	}

	return nil
}

// GetSpecialFolders 获取Outlook特殊文件夹映射
func (p *OutlookProvider) GetSpecialFolders() map[string]string {
	return map[string]string{
		"inbox":   "INBOX",
		"sent":    "Sent Items",
		"drafts":  "Drafts",
		"trash":   "Deleted Items",
		"spam":    "Junk Email",
		"archive": "Archive",
	}
}

// GetFolderDisplayName 获取文件夹显示名称
func (p *OutlookProvider) GetFolderDisplayName(folderName string) string {
	displayNames := map[string]string{
		"INBOX":         "收件箱",
		"Sent Items":    "已发送邮件",
		"Drafts":        "草稿",
		"Deleted Items": "已删除邮件",
		"Junk Email":    "垃圾邮件",
		"Archive":       "存档",
		"Outbox":        "发件箱",
	}

	if displayName, exists := displayNames[folderName]; exists {
		return displayName
	}

	return folderName
}

// SyncEmails 同步Outlook邮件
func (p *OutlookProvider) SyncEmails(ctx context.Context, account *models.EmailAccount, folderName string, lastUID uint32) ([]*EmailMessage, error) {
	fmt.Printf("📧 [SYNC] Starting Outlook email sync for account: %s, folder: %s, lastUID: %d\n",
		account.Email, folderName, lastUID)

	if !p.IsConnected() {
		fmt.Printf("🔄 [SYNC] Not connected, attempting to connect...\n")
		if err := p.Connect(ctx, account); err != nil {
			fmt.Printf("❌ [SYNC] Failed to connect: %v\n", err)
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
		fmt.Printf("✅ [SYNC] Successfully connected\n")
	}

	imapClient := p.IMAPClient()
	if imapClient == nil {
		fmt.Printf("❌ [SYNC] IMAP client not available\n")
		return nil, fmt.Errorf("IMAP client not available")
	}

	fmt.Printf("📬 [SYNC] Getting new emails from folder: %s, after UID: %d\n", folderName, lastUID)

	// 获取新邮件
	emails, err := imapClient.GetNewEmails(ctx, folderName, lastUID)
	if err != nil {
		fmt.Printf("❌ [SYNC] Failed to get new emails: %v\n", err)
		return nil, fmt.Errorf("failed to get new emails: %w", err)
	}

	fmt.Printf("📊 [SYNC] Retrieved %d emails from folder: %s\n", len(emails), folderName)

	// Outlook特殊处理：处理Exchange特性
	for i, email := range emails {
		fmt.Printf("📝 [SYNC] Processing email %d/%d - UID: %d, Subject: %s\n",
			i+1, len(emails), email.UID, email.Subject)
		p.processOutlookFeatures(email)
	}

	fmt.Printf("✅ [SYNC] Completed Outlook email sync, returning %d emails\n", len(emails))
	return emails, nil
}

// processOutlookFeatures 处理Outlook特性
func (p *OutlookProvider) processOutlookFeatures(email *EmailMessage) {
	// 处理Outlook/Exchange特有的标志和属性
	var labels []string

	for _, flag := range email.Flags {
		switch flag {
		case "\\Flagged":
			labels = append(labels, "Flagged")
		case "$MDNSent":
			labels = append(labels, "ReadReceiptSent")
		case "\\Answered":
			labels = append(labels, "Replied")
		case "\\Forwarded":
			labels = append(labels, "Forwarded")
		case "$Junk":
			labels = append(labels, "Junk")
		case "$NotJunk":
			labels = append(labels, "NotJunk")
		}
	}

	// 设置标签
	if len(labels) > 0 {
		email.SetLabels(labels)
	}

	// 处理重要性标记
	if contains(email.Flags, "$Important") {
		email.Priority = "high"
	} else if contains(email.Flags, "$LowImportance") {
		email.Priority = "low"
	}
}

// SendEmail 发送Outlook邮件
func (p *OutlookProvider) SendEmail(ctx context.Context, account *models.EmailAccount, message *OutgoingMessage) error {
	if !p.IsConnected() {
		if err := p.Connect(ctx, account); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
	}

	smtpClient := p.SMTPClient()
	if smtpClient == nil {
		return fmt.Errorf("SMTP client not available")
	}

	// Outlook特殊处理：确保发件人地址正确
	if message.From == nil {
		message.From = &models.EmailAddress{
			Address: account.Email,
			Name:    account.Name,
		}
	}

	// 验证发件人地址是否匹配账户
	if message.From.Address != account.Email {
		return fmt.Errorf("sender address must match account email for Outlook")
	}

	// 添加Outlook特有的头信息
	if message.Headers == nil {
		message.Headers = make(map[string]string)
	}

	// 添加Exchange相关头信息
	message.Headers["X-Mailer"] = "FireMail"

	// 如果设置了优先级，添加相应的头信息
	if message.Priority == "high" {
		message.Headers["Importance"] = "high"
		message.Headers["X-Priority"] = "1"
	} else if message.Priority == "low" {
		message.Headers["Importance"] = "low"
		message.Headers["X-Priority"] = "5"
	}

	// 发送邮件
	return smtpClient.SendEmail(ctx, message)
}

// ValidateEmailAddress 验证Outlook邮箱地址格式
func (p *OutlookProvider) ValidateEmailAddress(email string) error {
	email = strings.ToLower(email)

	// 检查是否是支持的Outlook邮箱域名
	supportedDomains := []string{
		"outlook.com", "hotmail.com", "live.com", "msn.com",
		"outlook.co.uk", "hotmail.co.uk", "live.co.uk",
		"outlook.fr", "hotmail.fr", "live.fr",
	}

	for _, domain := range supportedDomains {
		if strings.HasSuffix(email, "@"+domain) {
			return nil
		}
	}

	return fmt.Errorf("unsupported Outlook domain. Supported domains include: outlook.com, hotmail.com, live.com, msn.com")
}

// 重复的GetOAuth2Instructions方法已删除

// GetProviderInfo 获取提供商信息
func (p *OutlookProvider) GetProviderInfo() map[string]interface{} {
	return map[string]interface{}{
		"name":         "Outlook",
		"display_name": "Outlook/Hotmail（Microsoft）",
		"auth_methods": []string{"oauth2", "password"},
		"domains":      []string{"outlook.com", "hotmail.com", "live.com", "msn.com"},
		"servers": map[string]interface{}{
			"imap": map[string]interface{}{
				"host":     "outlook.office365.com",
				"port":     993,
				"security": "SSL",
			},
			"smtp": map[string]interface{}{
				"host":     "smtp-mail.outlook.com",
				"port":     587,
				"security": "STARTTLS",
			},
		},
		"features": map[string]bool{
			"imap":       true,
			"smtp":       true,
			"oauth2":     true,
			"push":       true,
			"threading":  true,
			"labels":     false,
			"folders":    true,
			"categories": true,
			"search":     true,
			"idle":       true,
			"rules":      true,
		},
		"limits": map[string]interface{}{
			"attachment_size":   25 * 1024 * 1024,        // 25MB（个人账户）
			"daily_send":        300,                     // 每日发送限制（个人账户）
			"rate_limit_window": 86400,                   // 24小时窗口
			"max_recipients":    500,                     // 单封邮件最大收件人数
			"storage_free":      15 * 1024 * 1024 * 1024, // 15GB 免费存储
			"connection_limit":  16,                      // 同时IMAP连接数限制
		},
		"oauth2": map[string]interface{}{
			"scopes": []string{
				"https://outlook.office.com/IMAP.AccessAsUser.All",
				"https://outlook.office.com/SMTP.Send",
				"https://graph.microsoft.com/Mail.Read",
				"https://graph.microsoft.com/Mail.ReadWrite",
				"https://graph.microsoft.com/Mail.Send",
				"offline_access",
			},
			"auth_url":  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			"token_url": "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		},
		"error_codes": map[string]string{
			"535":    "认证失败，个人账户请使用OAuth2",
			"534":    "认证机制不支持，请使用OAuth2",
			"550":    "发送频率超限，请稍后重试",
			"552":    "邮件大小超过限制",
			"554":    "邮件被拒绝，内容可能被识别为垃圾邮件",
			"421":    "服务暂时不可用，服务器繁忙",
			"452":    "存储空间不足，邮箱已满",
			"553":    "收件人地址无效或发件人未授权",
			"AADSTS": "Azure AD认证错误",
		},
		"help_urls": map[string]string{
			"azure_portal":     "https://portal.azure.com/",
			"app_registration": "https://docs.microsoft.com/en-us/azure/active-directory/develop/quickstart-register-app",
			"oauth2_setup":     "https://docs.microsoft.com/en-us/exchange/client-developer/legacy-protocols/how-to-authenticate-an-imap-pop-smtp-application-by-using-oauth",
			"outlook_help":     "https://support.microsoft.com/en-us/office/outlook-help",
			"basic_auth":       "https://docs.microsoft.com/en-us/exchange/clients-and-mobile-in-exchange-online/deprecation-of-basic-authentication-exchange-online",
		},
		"auth_instructions": map[string]string{
			"oauth2":   p.GetOAuth2Instructions(),
			"password": "Basic authentication is deprecated. Use OAuth2 for better security.",
		},
		"requirements": map[string]interface{}{
			"oauth2_app":    true,  // OAuth2需要注册应用
			"admin_consent": false, // 个人账户不需要管理员同意
		},
		"deprecation_notice": "Basic authentication is deprecated for personal Microsoft accounts and will be disabled for enterprise accounts. Please migrate to OAuth2.",
	}
}
