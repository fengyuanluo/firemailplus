package providers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"firemail/internal/config"
	"firemail/internal/encoding"
	"firemail/internal/models"
)

// QQProvider QQ邮箱提供商
type QQProvider struct {
	*BaseProvider
	encodingHelper *encoding.EmailEncodingHelper
}

// newQQProviderImpl 创建QQ邮箱提供商实例的内部实现
func newQQProviderImpl(config *config.EmailProviderConfig) *QQProvider {
	base := NewBaseProvider(config)

	provider := &QQProvider{
		BaseProvider:   base,
		encodingHelper: encoding.NewEmailEncodingHelper(),
	}

	// 设置IMAP和SMTP客户端
	provider.SetIMAPClient(NewStandardIMAPClient())
	provider.SetSMTPClient(NewStandardSMTPClient())

	return provider
}

// NewQQProvider 创建QQ邮箱提供商实例（工厂方法）
func NewQQProvider(config *config.EmailProviderConfig) EmailProvider {
	return newQQProviderImpl(config)
}

// Connect 连接到QQ邮箱服务器
func (p *QQProvider) Connect(ctx context.Context, account *models.EmailAccount) error {
	// 确保使用正确的服务器配置
	p.ensureQQConfig(account)

	// QQ邮箱只支持密码认证
	if account.AuthMethod != "password" {
		return fmt.Errorf("QQ mail only supports password authentication")
	}

	// 验证是否使用了授权码
	if err := p.validateAuthCode(account); err != nil {
		return fmt.Errorf("QQ mail authentication failed: %w", err)
	}

	// 使用重试机制连接
	return p.connectWithRetry(ctx, account)
}

// connectWithRetry 带重试机制的连接
func (p *QQProvider) connectWithRetry(ctx context.Context, account *models.EmailAccount) error {
	maxRetries := 3
	baseDelay := time.Second * 2

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := p.BaseProvider.Connect(ctx, account)
		if err == nil {
			return nil
		}

		// 处理QQ邮箱特定错误
		qqErr := p.HandleQQError(err)

		// 某些错误不需要重试
		if p.isNonRetryableError(err) {
			return qqErr
		}

		// 如果是最后一次尝试，返回错误
		if attempt == maxRetries-1 {
			return qqErr
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

// isNonRetryableError 判断是否为不可重试的错误
func (p *QQProvider) isNonRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// 认证错误、配置错误等不需要重试
	nonRetryableErrors := []string{
		"535", // 认证失败
		"553", // 无效地址
		"authentication failed",
		"invalid authorization code",
		"unsupported auth method",
	}

	for _, nonRetryable := range nonRetryableErrors {
		if strings.Contains(errStr, nonRetryable) {
			return true
		}
	}

	return false
}

// ensureQQConfig 确保QQ邮箱配置正确
func (p *QQProvider) ensureQQConfig(account *models.EmailAccount) {
	// 如果没有设置服务器配置，使用QQ邮箱默认配置
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

	// QQ邮箱用户名通常是完整的邮箱地址
	if account.Username == "" {
		account.Username = account.Email
	}
}

// validateAuthCode 验证QQ邮箱授权码
func (p *QQProvider) validateAuthCode(account *models.EmailAccount) error {
	// QQ邮箱需要使用授权码而不是登录密码
	// 授权码通常是16位字符
	if len(account.Password) != 16 {
		return fmt.Errorf("16-character authorization code")
	}

	// 检查是否包含非法字符（授权码通常只包含字母和数字）
	for _, char := range account.Password {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
			return fmt.Errorf("invalid authorization code format")
		}
	}

	return nil
}

// HandleQQError 处理QQ邮箱特定错误
func (p *QQProvider) HandleQQError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// 常见QQ邮箱错误处理
	switch {
	case strings.Contains(errStr, "535"):
		return fmt.Errorf("authentication failed: please check your email and authorization code")
	case strings.Contains(errStr, "550"):
		return fmt.Errorf("sending frequency limit exceeded")
	case strings.Contains(errStr, "554"):
		return fmt.Errorf("message rejected: content may be considered spam")
	case strings.Contains(errStr, "421"):
		return fmt.Errorf("service temporarily unavailable")
	case strings.Contains(errStr, "452"):
		return fmt.Errorf("insufficient storage: mailbox is full or quota exceeded")
	case strings.Contains(errStr, "553"):
		return fmt.Errorf("invalid recipient address or sender not authorized")
	default:
		return fmt.Errorf("QQ mail error")
	}
}

// TestConnection 测试QQ邮箱连接
func (p *QQProvider) TestConnection(ctx context.Context, account *models.EmailAccount) error {
	// 确保配置正确
	p.ensureQQConfig(account)

	// 验证授权码
	if err := p.validateAuthCode(account); err != nil {
		return err
	}

	// 调用基类测试方法
	return p.BaseProvider.TestConnection(ctx, account)
}

// GetSpecialFolders 获取QQ邮箱特殊文件夹映射
func (p *QQProvider) GetSpecialFolders() map[string]string {
	return map[string]string{
		"inbox":  "INBOX",
		"sent":   "Sent Messages",
		"drafts": "Drafts",
		"trash":  "Deleted Messages",
		"spam":   "Junk",
	}
}

// GetFolderDisplayName 获取文件夹显示名称
func (p *QQProvider) GetFolderDisplayName(folderName string) string {
	displayNames := map[string]string{
		"INBOX":            "收件箱",
		"Sent Messages":    "已发送",
		"Drafts":           "草稿箱",
		"Deleted Messages": "已删除",
		"Junk":             "垃圾邮件",
	}

	if displayName, exists := displayNames[folderName]; exists {
		return displayName
	}

	return folderName
}

// SyncEmails 同步QQ邮箱邮件
func (p *QQProvider) SyncEmails(ctx context.Context, account *models.EmailAccount, folderName string, lastUID uint32) ([]*EmailMessage, error) {
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

	// QQ邮箱特殊处理：处理编码
	for _, email := range emails {
		p.processQQEncoding(email)
	}

	return emails, nil
}

// processQQEncoding 处理QQ邮箱编码
func (p *QQProvider) processQQEncoding(email *EmailMessage) {
	// 处理邮件主题编码
	if email.Subject != "" {
		email.Subject = p.encodingHelper.DecodeEmailSubject(email.Subject)
	}

	// 处理发件人编码
	if email.From != nil && email.From.Name != "" {
		email.From.Name = p.encodingHelper.DecodeEmailFrom(email.From.Name)
	}

	// 处理邮件内容编码 - QQ邮箱特殊处理
	if len(email.TextBody) > 0 {
		// 尝试多种编码方式
		if decoded := p.tryDecodeQQContent(email.TextBody); decoded != "" {
			email.TextBody = decoded
		}
	}

	if len(email.HTMLBody) > 0 {
		// 尝试多种编码方式
		if decoded := p.tryDecodeQQContent(email.HTMLBody); decoded != "" {
			email.HTMLBody = decoded
		}
	}
}

// tryDecodeQQContent 尝试解码QQ邮箱内容
func (p *QQProvider) tryDecodeQQContent(content string) string {
	if content == "" {
		return content
	}

	// 首先尝试标准解码
	if decoded, err := p.encodingHelper.DecodeEmailContent([]byte(content), ""); err == nil {
		if string(decoded) != content && len(decoded) > 0 {
			return string(decoded)
		}
	}

	// 尝试GBK编码
	if decoded, err := p.encodingHelper.DecodeEmailContent([]byte(content), "gbk"); err == nil {
		if string(decoded) != content && len(decoded) > 0 {
			return string(decoded)
		}
	}

	// 尝试GB2312编码
	if decoded, err := p.encodingHelper.DecodeEmailContent([]byte(content), "gb2312"); err == nil {
		if string(decoded) != content && len(decoded) > 0 {
			return string(decoded)
		}
	}

	return content
}

// SendEmail 发送QQ邮箱邮件
func (p *QQProvider) SendEmail(ctx context.Context, account *models.EmailAccount, message *OutgoingMessage) error {
	if !p.IsConnected() {
		if err := p.Connect(ctx, account); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
	}

	smtpClient := p.SMTPClient()
	if smtpClient == nil {
		return fmt.Errorf("SMTP client not available")
	}

	// QQ邮箱特殊处理：确保发件人地址正确
	if message.From == nil {
		message.From = &models.EmailAddress{
			Address: account.Email,
			Name:    account.Name,
		}
	}

	// 验证发件人地址是否匹配账户
	if message.From.Address != account.Email {
		return fmt.Errorf("sender address must match account email for QQ mail")
	}

	// 发送邮件
	return smtpClient.SendEmail(ctx, message)
}

// GetAuthCodeInstructions 获取QQ邮箱授权码设置说明
func (p *QQProvider) GetAuthCodeInstructions() string {
	return `QQ邮箱授权码设置步骤：
1. 登录QQ邮箱网页版
2. 点击"设置" -> "账户"
3. 找到"POP3/IMAP/SMTP/Exchange/CardDAV/CalDAV服务"
4. 开启"IMAP/SMTP服务"
5. 点击"生成授权码"
6. 按照提示发送短信获取授权码
7. 将获得的16位授权码作为密码使用

注意：
- 授权码不是QQ密码，是专门用于第三方客户端的密码
- 授权码为16位字符，只包含字母和数字
- 如果忘记授权码，可以重新生成`
}

// ValidateEmailAddress 验证QQ邮箱地址格式
func (p *QQProvider) ValidateEmailAddress(email string) error {
	email = strings.ToLower(email)

	// 检查是否是支持的QQ邮箱域名
	supportedDomains := []string{"qq.com", "vip.qq.com", "foxmail.com"}

	for _, domain := range supportedDomains {
		if strings.HasSuffix(email, "@"+domain) {
			return nil
		}
	}

	return fmt.Errorf("unsupported QQ mail domain. Supported domains: %s", strings.Join(supportedDomains, ", "))
}

// GetProviderInfo 获取提供商信息
func (p *QQProvider) GetProviderInfo() map[string]interface{} {
	return map[string]interface{}{
		"name":         "QQ邮箱",
		"display_name": "QQ邮箱",
		"auth_methods": []string{"password"},
		"domains":      []string{"qq.com", "vip.qq.com", "foxmail.com"},
		"servers": map[string]interface{}{
			"imap": map[string]interface{}{
				"host":     "imap.qq.com",
				"port":     993,
				"security": "SSL",
			},
			"smtp": map[string]interface{}{
				"host":     "smtp.qq.com",
				"port":     465,
				"security": "SSL",
				"alt_port": 587, // 备用端口
			},
		},
		"features": map[string]bool{
			"imap":      true,
			"smtp":      true,
			"oauth2":    false,
			"push":      false,
			"threading": false,
			"labels":    false,
			"folders":   true,
			"search":    true,
			"idle":      true,
		},
		"limits": map[string]interface{}{
			"attachment_size":   50 * 1024 * 1024,       // 50MB
			"daily_send":        500,                    // 每日发送限制（个人邮箱）
			"hourly_send":       50,                     // 每小时发送限制
			"rate_limit_window": 3600,                   // 频率限制窗口（秒）
			"max_recipients":    100,                    // 单封邮件最大收件人数
			"mailbox_size":      2 * 1024 * 1024 * 1024, // 2GB 免费邮箱容量
		},
		"error_codes": map[string]string{
			"535": "认证失败，请检查邮箱地址和授权码",
			"550": "发送频率超限，请稍后重试",
			"554": "邮件被拒绝，内容可能被识别为垃圾邮件",
			"421": "服务暂时不可用，服务器繁忙",
			"452": "存储空间不足，邮箱已满",
			"553": "收件人地址无效或发件人未授权",
		},
		"help_urls": map[string]string{
			"auth_code":   "https://service.mail.qq.com/cgi-bin/help?subtype=1&&id=28&&no=1001256",
			"imap_setup":  "https://service.mail.qq.com/detail/0/339",
			"smtp_setup":  "https://service.mail.qq.com/detail/0/340",
			"settings":    "https://mail.qq.com/",
			"help_center": "https://service.mail.qq.com/",
		},
		"auth_instructions": p.GetAuthCodeInstructions(),
	}
}
