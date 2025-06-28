package providers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"firemail/internal/config"
	"firemail/internal/models"
)

// iCloudProvider iCloud邮箱提供商
type iCloudProvider struct {
	*BaseProvider
}

// newiCloudProviderImpl 创建iCloud邮箱提供商实例的内部实现
func newiCloudProviderImpl(config *config.EmailProviderConfig) *iCloudProvider {
	base := NewBaseProvider(config)

	provider := &iCloudProvider{
		BaseProvider: base,
	}

	// 设置IMAP和SMTP客户端
	provider.SetIMAPClient(NewStandardIMAPClient())
	provider.SetSMTPClient(NewStandardSMTPClient())

	return provider
}

// NewiCloudProvider 创建iCloud邮箱提供商实例（工厂方法）
func NewiCloudProvider(config *config.EmailProviderConfig) EmailProvider {
	return newiCloudProviderImpl(config)
}

// Connect 连接到iCloud邮箱服务器
func (p *iCloudProvider) Connect(ctx context.Context, account *models.EmailAccount) error {
	// 确保使用正确的服务器配置
	p.ensureiCloudConfig(account)

	// iCloud邮箱只支持密码认证（应用专用密码）
	if account.AuthMethod != "password" {
		return fmt.Errorf("iCloud mail only supports password authentication with app-specific passwords")
	}

	// 验证是否使用了应用专用密码
	if err := p.validateAppSpecificPassword(account); err != nil {
		return fmt.Errorf("iCloud mail authentication failed: %w", err)
	}

	// 使用重试机制连接
	return p.connectWithRetry(ctx, account)
}

// ensureiCloudConfig 确保iCloud邮箱配置正确
func (p *iCloudProvider) ensureiCloudConfig(account *models.EmailAccount) {
	// iCloud邮箱使用固定的服务器配置
	if account.IMAPHost == "" {
		account.IMAPHost = "imap.mail.me.com"
		account.IMAPPort = 993
		account.IMAPSecurity = "SSL"
	}

	if account.SMTPHost == "" {
		account.SMTPHost = "smtp.mail.me.com"
		account.SMTPPort = 587
		account.SMTPSecurity = "STARTTLS"
	}

	// iCloud邮箱用户名通常是完整的邮箱地址
	if account.Username == "" {
		account.Username = account.Email
	}
}

// validateAppSpecificPassword 验证iCloud应用专用密码
func (p *iCloudProvider) validateAppSpecificPassword(account *models.EmailAccount) error {
	// iCloud应用专用密码格式：xxxx-xxxx-xxxx-xxxx（16个字符，包含3个连字符）
	password := account.Password

	// 先检查是否包含连字符，如果包含连字符但格式不对，返回格式错误
	if strings.Contains(password, "-") {
		// 检查连字符数量和位置
		parts := strings.Split(password, "-")
		if len(parts) != 4 {
			return fmt.Errorf("invalid app-specific password format")
		}

		// 检查每部分长度
		for _, part := range parts {
			if len(part) != 4 {
				return fmt.Errorf("invalid app-specific password format")
			}
		}

		// 检查连字符位置
		if len(password) != 19 || password[4] != '-' || password[9] != '-' || password[14] != '-' {
			return fmt.Errorf("invalid app-specific password format")
		}

		// 检查字符是否为字母和数字
		for _, part := range parts {
			for _, char := range part {
				if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
					return fmt.Errorf("invalid app-specific password format")
				}
			}
		}
	} else {
		// 如果没有连字符，检查基本长度
		if len(password) != 19 {
			return fmt.Errorf("app-specific password in format xxxx-xxxx-xxxx-xxxx")
		}
		return fmt.Errorf("invalid app-specific password format")
	}

	return nil
}

// connectWithRetry 带重试机制的连接
func (p *iCloudProvider) connectWithRetry(ctx context.Context, account *models.EmailAccount) error {
	maxRetries := 3
	baseDelay := time.Second * 2

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := p.BaseProvider.Connect(ctx, account)
		if err == nil {
			return nil
		}

		// 处理iCloud邮箱特定错误
		iCloudErr := p.HandleiCloudError(err)

		// 某些错误不需要重试
		if p.isNonRetryableError(err) {
			return iCloudErr
		}

		// 如果是最后一次尝试，返回错误
		if attempt == maxRetries-1 {
			return iCloudErr
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

// HandleiCloudError 处理iCloud邮箱特定错误
func (p *iCloudProvider) HandleiCloudError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// 常见iCloud邮箱错误处理
	switch {
	case strings.Contains(errStr, "535"):
		return fmt.Errorf("authentication failed: please check your iCloud email and app-specific password")
	case strings.Contains(errStr, "550"):
		return fmt.Errorf("sending limit exceeded")
	case strings.Contains(errStr, "554"):
		return fmt.Errorf("message rejected: content may be considered spam or violate iCloud mail policies")
	case strings.Contains(errStr, "421"):
		return fmt.Errorf("service temporarily unavailable: iCloud mail server is busy, please try again later")
	case strings.Contains(errStr, "452"):
		return fmt.Errorf("insufficient storage: iCloud mailbox is full or quota exceeded")
	case strings.Contains(errStr, "553"):
		return fmt.Errorf("invalid recipient address or sender not authorized")
	case strings.Contains(errStr, "connection refused"):
		return fmt.Errorf("connection refused: please check your network connection")
	case strings.Contains(errStr, "timeout"):
		return fmt.Errorf("connection timeout: iCloud mail server may be experiencing issues")
	default:
		return fmt.Errorf("iCloud mail error")
	}
}

// isNonRetryableError 判断是否为不可重试的错误
func (p *iCloudProvider) isNonRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// 认证错误、配置错误等不需要重试
	nonRetryableErrors := []string{
		"535", // 认证失败
		"553", // 无效地址
		"authentication failed",
		"invalid app-specific password",
		"unsupported auth method",
	}

	for _, nonRetryable := range nonRetryableErrors {
		if strings.Contains(errStr, nonRetryable) {
			return true
		}
	}

	return false
}

// TestConnection 测试iCloud邮箱连接
func (p *iCloudProvider) TestConnection(ctx context.Context, account *models.EmailAccount) error {
	// 确保配置正确
	p.ensureiCloudConfig(account)

	// 验证应用专用密码
	if err := p.validateAppSpecificPassword(account); err != nil {
		return err
	}

	// 调用基类测试方法
	return p.BaseProvider.TestConnection(ctx, account)
}

// GetSpecialFolders 获取iCloud邮箱特殊文件夹映射
func (p *iCloudProvider) GetSpecialFolders() map[string]string {
	return map[string]string{
		"inbox":   "INBOX",
		"sent":    "Sent Messages",
		"drafts":  "Drafts",
		"trash":   "Deleted Messages",
		"spam":    "Junk",
		"archive": "Archive",
	}
}

// GetFolderDisplayName 获取文件夹显示名称
func (p *iCloudProvider) GetFolderDisplayName(folderName string) string {
	displayNames := map[string]string{
		"INBOX":            "收件箱",
		"Sent Messages":    "已发送",
		"Drafts":           "草稿",
		"Deleted Messages": "已删除",
		"Junk":             "垃圾邮件",
		"Archive":          "归档",
		"Notes":            "备忘录",
	}

	if displayName, exists := displayNames[folderName]; exists {
		return displayName
	}

	return folderName
}

// SendEmail 发送iCloud邮箱邮件
func (p *iCloudProvider) SendEmail(ctx context.Context, account *models.EmailAccount, message *OutgoingMessage) error {
	if !p.IsConnected() {
		if err := p.Connect(ctx, account); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
	}

	smtpClient := p.SMTPClient()
	if smtpClient == nil {
		return fmt.Errorf("SMTP client not available")
	}

	// iCloud邮箱特殊处理：确保发件人地址正确
	if message.From == nil {
		message.From = &models.EmailAddress{
			Address: account.Email,
			Name:    account.Name,
		}
	}

	// 验证发件人地址是否匹配账户
	if message.From.Address != account.Email {
		return fmt.Errorf("sender address must match account email for iCloud mail")
	}

	// 检查发送限制
	if err := p.checkSendingLimits(message); err != nil {
		return err
	}

	// 发送邮件
	return smtpClient.SendEmail(ctx, message)
}

// checkSendingLimits 检查发送限制
func (p *iCloudProvider) checkSendingLimits(message *OutgoingMessage) error {
	// 检查收件人数量
	totalRecipients := len(message.To) + len(message.CC) + len(message.BCC)
	if totalRecipients > 500 {
		return fmt.Errorf("too many recipients")
	}

	// 检查邮件大小（这里简化处理，实际应该计算完整邮件大小）
	if len(message.TextBody)+len(message.HTMLBody) > 20*1024*1024 {
		return fmt.Errorf("message too large")
	}

	return nil
}

// ValidateEmailAddress 验证iCloud邮箱地址格式
func (p *iCloudProvider) ValidateEmailAddress(email string) error {
	email = strings.ToLower(email)

	// 检查是否是支持的iCloud邮箱域名
	supportedDomains := []string{"icloud.com", "me.com", "mac.com"}

	for _, domain := range supportedDomains {
		if strings.HasSuffix(email, "@"+domain) {
			return nil
		}
	}

	return fmt.Errorf("unsupported iCloud mail domain. Supported domains: %s", strings.Join(supportedDomains, ", "))
}

// GetAppSpecificPasswordInstructions 获取iCloud应用专用密码设置说明
func (p *iCloudProvider) GetAppSpecificPasswordInstructions() string {
	return `iCloud应用专用密码设置步骤：
1. 访问 Apple ID 账户页面 (https://appleid.apple.com)
2. 使用您的 Apple ID 和密码登录
3. 在"安全"部分中，点击"生成密码"
4. 输入应用标签（如"FireMail"）
5. 点击"创建"
6. 复制生成的应用专用密码（格式：xxxx-xxxx-xxxx-xxxx）
7. 在邮件客户端中使用此密码而不是您的 Apple ID 密码

前提条件：
- 必须为您的 Apple ID 启用双重认证
- 应用专用密码只能查看一次，请妥善保存
- 每个应用专用密码只能用于一个应用或服务
- 可以随时撤销不再使用的应用专用密码

注意：
- 应用专用密码格式为 xxxx-xxxx-xxxx-xxxx
- 不要与他人分享您的应用专用密码
- 如果更改了 Apple ID 密码，所有应用专用密码将自动撤销`
}

// GetProviderInfo 获取提供商信息
func (p *iCloudProvider) GetProviderInfo() map[string]interface{} {
	return map[string]interface{}{
		"name":         "iCloud邮箱",
		"display_name": "iCloud邮箱（Apple）",
		"auth_methods": []string{"password"},
		"domains":      []string{"icloud.com", "me.com", "mac.com"},
		"servers": map[string]interface{}{
			"imap": map[string]interface{}{
				"host":     "imap.mail.me.com",
				"port":     993,
				"security": "SSL",
			},
			"smtp": map[string]interface{}{
				"host":     "smtp.mail.me.com",
				"port":     587,
				"security": "STARTTLS",
			},
		},
		"features": map[string]bool{
			"imap":      true,
			"smtp":      true,
			"oauth2":    false,
			"push":      true,
			"threading": true,
			"labels":    false,
			"folders":   true,
			"search":    true,
			"idle":      true,
			"notes":     true, // iCloud支持备忘录
		},
		"limits": map[string]interface{}{
			"attachment_size":   20 * 1024 * 1024,       // 20MB
			"daily_send":        1000,                   // 每日发送限制
			"hourly_send":       100,                    // 每小时发送限制
			"rate_limit_window": 3600,                   // 频率限制窗口（秒）
			"max_recipients":    500,                    // 单封邮件最大收件人数
			"mailbox_size":      5 * 1024 * 1024 * 1024, // 5GB 免费存储空间
			"connection_limit":  5,                      // 同时连接数限制
		},
		"error_codes": map[string]string{
			"535": "认证失败，请检查邮箱地址和应用专用密码",
			"550": "发送频率超限，请稍后重试",
			"554": "邮件被拒绝，内容可能被识别为垃圾邮件",
			"421": "服务暂时不可用，服务器繁忙",
			"452": "存储空间不足，邮箱已满",
			"553": "收件人地址无效或发件人未授权",
		},
		"help_urls": map[string]string{
			"apple_id":      "https://appleid.apple.com/",
			"app_passwords": "https://support.apple.com/zh-cn/102654",
			"mail_setup":    "https://support.apple.com/zh-cn/102525",
			"two_factor":    "https://support.apple.com/zh-cn/HT204915",
			"icloud_help":   "https://support.apple.com/zh-cn/icloud",
		},
		"auth_instructions": p.GetAppSpecificPasswordInstructions(),
		"requirements": map[string]interface{}{
			"two_factor_auth": true,
			"app_password":    true,
			"apple_id":        true,
		},
	}
}

// SyncEmails 同步iCloud邮箱邮件
func (p *iCloudProvider) SyncEmails(ctx context.Context, account *models.EmailAccount, folderName string, lastUID uint32) ([]*EmailMessage, error) {
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

	// iCloud特殊处理：处理备忘录和特殊标志
	for _, email := range emails {
		p.processiCloudFeatures(email)
	}

	return emails, nil
}

// processiCloudFeatures 处理iCloud特性
func (p *iCloudProvider) processiCloudFeatures(email *EmailMessage) {
	// 处理iCloud特有的标志和属性
	var labels []string

	for _, flag := range email.Flags {
		switch flag {
		case "\\Flagged":
			labels = append(labels, "Flagged")
		case "\\Answered":
			labels = append(labels, "Replied")
		case "\\Forwarded":
			labels = append(labels, "Forwarded")
		case "$NotJunk":
			labels = append(labels, "NotJunk")
		case "\\Recent":
			labels = append(labels, "Recent")
		}
	}

	// 设置标签
	if len(labels) > 0 {
		email.SetLabels(labels)
	}

	// 检查是否为备忘录
	if strings.Contains(email.Subject, "Note:") ||
		(email.From != nil && strings.Contains(email.From.Address, "noreply@icloud.com")) {
		email.SetLabels(append(email.GetLabels(), "Note"))
	}
}
