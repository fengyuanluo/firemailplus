package providers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"firemail/internal/config"
	"firemail/internal/models"
)

// NetEaseProvider 网易邮箱提供商（163/126/yeah）
type NetEaseProvider struct {
	*BaseProvider
}

// newNetEaseProviderImpl 创建网易邮箱提供商实例的内部实现
func newNetEaseProviderImpl(config *config.EmailProviderConfig) *NetEaseProvider {
	base := NewBaseProvider(config)

	provider := &NetEaseProvider{
		BaseProvider: base,
	}

	// 设置IMAP和SMTP客户端
	provider.SetIMAPClient(NewStandardIMAPClient())
	provider.SetSMTPClient(NewStandardSMTPClient())

	return provider
}

// NewNetEaseProvider 创建网易邮箱提供商实例（工厂方法）
func NewNetEaseProvider(config *config.EmailProviderConfig) EmailProvider {
	return newNetEaseProviderImpl(config)
}

// Connect 连接到网易邮箱服务器
func (p *NetEaseProvider) Connect(ctx context.Context, account *models.EmailAccount) error {
	// 确保使用正确的服务器配置
	p.ensureNetEaseConfig(account)

	// 网易邮箱只支持密码认证
	if account.AuthMethod != "password" {
		return fmt.Errorf("NetEase mail only supports password authentication")
	}

	// 验证是否使用了客户端授权码
	if err := p.validateAuthCode(account); err != nil {
		return fmt.Errorf("NetEase mail authentication failed: %w", err)
	}

	// 使用重试机制连接，为163邮箱添加IMAP ID支持
	return p.connectWithRetryAndIMAPID(ctx, account)
}

// is163Email 检查是否是163邮箱
func (p *NetEaseProvider) is163Email(email string) bool {
	return strings.HasSuffix(strings.ToLower(email), "@163.com") ||
		strings.HasSuffix(strings.ToLower(email), "@vip.163.com")
}

// get163IMAPIDInfo 获取163邮箱的IMAP ID信息
// 根据163邮箱文档要求，需要提供客户端身份信息
func (p *NetEaseProvider) get163IMAPIDInfo() map[string]string {
	return map[string]string{
		"name":          "FireMail",
		"version":       "1.0.0",
		"vendor":        "FireMail Team",
		"support-email": "support@firemail.com",
		"os":            "Linux",
		"os-version":    "Ubuntu 20.04",
	}
}

// ensureNetEaseConfig 确保网易邮箱配置正确
func (p *NetEaseProvider) ensureNetEaseConfig(account *models.EmailAccount) {
	domain := extractDomainFromEmail(account.Email)

	// 根据不同的网易邮箱域名设置服务器地址
	switch domain {
	case "163.com":
		// 强制使用正确的163邮箱配置
		account.IMAPHost = "imap.163.com"
		account.IMAPPort = 993
		account.IMAPSecurity = "SSL"
		account.SMTPHost = "smtp.163.com"
		account.SMTPPort = 465
		account.SMTPSecurity = "SSL"
	case "126.com":
		if account.IMAPHost == "" {
			account.IMAPHost = "imap.126.com"
			account.IMAPPort = 993
			account.IMAPSecurity = "SSL"
		}
		if account.SMTPHost == "" {
			account.SMTPHost = "smtp.126.com"
			account.SMTPPort = 465
			account.SMTPSecurity = "SSL"
		}
	case "yeah.net":
		if account.IMAPHost == "" {
			account.IMAPHost = "imap.yeah.net"
			account.IMAPPort = 993
			account.IMAPSecurity = "SSL"
		}
		if account.SMTPHost == "" {
			account.SMTPHost = "smtp.yeah.net"
			account.SMTPPort = 465
			account.SMTPSecurity = "SSL"
		}
	default:
		// 默认使用163配置
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

	// 网易邮箱用户名通常是完整的邮箱地址
	if account.Username == "" {
		account.Username = account.Email
	}
}

// validateAuthCode 验证网易邮箱客户端授权码
func (p *NetEaseProvider) validateAuthCode(account *models.EmailAccount) error {
	// 网易邮箱需要使用客户端授权码而不是登录密码
	// 授权码通常是16位字符
	if len(account.Password) != 16 {
		return fmt.Errorf("16-character client authorization code")
	}

	// 检查是否包含非法字符（授权码通常只包含字母和数字）
	for _, char := range account.Password {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
			return fmt.Errorf("invalid authorization code format")
		}
	}

	return nil
}

// connectWithRetryAndIMAPID 带重试机制和IMAP ID支持的连接
func (p *NetEaseProvider) connectWithRetryAndIMAPID(ctx context.Context, account *models.EmailAccount) error {
	maxRetries := 3
	baseDelay := time.Second * 2

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := p.connectWithIMAPID(ctx, account)
		if err == nil {
			return nil
		}

		// 处理网易邮箱特定错误
		netEaseErr := p.HandleNetEaseError(err)

		// 某些错误不需要重试
		if p.isNonRetryableError(err) {
			return netEaseErr
		}

		// 如果是最后一次尝试，返回错误
		if attempt == maxRetries-1 {
			return netEaseErr
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

// connectWithIMAPID 使用IMAP ID支持的连接
func (p *NetEaseProvider) connectWithIMAPID(ctx context.Context, account *models.EmailAccount) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.connected {
		return nil
	}

	// 连接IMAP
	if p.imapClient != nil {
		imapConfig := IMAPClientConfig{
			Host:        account.IMAPHost,
			Port:        account.IMAPPort,
			Security:    account.IMAPSecurity,
			Username:    account.Username,
			Password:    account.Password,
			ProxyConfig: p.createProxyConfig(account),
		}

		// 为163邮箱添加IMAP ID信息（可信部分）
		if p.is163Email(account.Email) {
			imapConfig.IMAPIDInfo = p.get163IMAPIDInfo()
		}

		if err := p.imapClient.Connect(ctx, imapConfig); err != nil {
			return fmt.Errorf("failed to connect IMAP: %w", err)
		}
	}

	// 连接SMTP
	if p.smtpClient != nil {
		smtpConfig := SMTPClientConfig{
			Host:        account.SMTPHost,
			Port:        account.SMTPPort,
			Security:    account.SMTPSecurity,
			Username:    account.Username,
			Password:    account.Password,
			ProxyConfig: p.createProxyConfig(account),
		}
		if err := p.smtpClient.Connect(ctx, smtpConfig); err != nil {
			return fmt.Errorf("failed to connect SMTP: %w", err)
		}
	}

	p.connected = true
	return nil
}

// HandleNetEaseError 处理网易邮箱特定错误
func (p *NetEaseProvider) HandleNetEaseError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// 常见网易邮箱错误处理
	switch {
	case strings.Contains(errStr, "535"):
		return fmt.Errorf("authentication failed: please check your email and client authorization code")
	case strings.Contains(errStr, "550"):
		return fmt.Errorf("sending frequency limit exceeded")
	case strings.Contains(errStr, "554"):
		return fmt.Errorf("message rejected: content may be considered spam or violate NetEase mail policies")
	case strings.Contains(errStr, "451"):
		if strings.Contains(errStr, "RP:CEL") {
			return fmt.Errorf("too many error commands from sender")
		}
		if strings.Contains(errStr, "MI:DMC") {
			return fmt.Errorf("too many messages sent in current connection")
		}
		return fmt.Errorf("temporary failure: %v", err)
	case strings.Contains(errStr, "421"):
		return fmt.Errorf("service temporarily unavailable: NetEase mail server is busy, please try again later")
	case strings.Contains(errStr, "452"):
		return fmt.Errorf("insufficient storage: mailbox is full or quota exceeded")
	case strings.Contains(errStr, "553"):
		return fmt.Errorf("invalid recipient address or sender not authorized")
	default:
		return fmt.Errorf("NetEase mail error: %v", err)
	}
}

// isNonRetryableError 判断是否为不可重试的错误
func (p *NetEaseProvider) isNonRetryableError(err error) bool {
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

// extractDomainFromEmail 从邮箱地址中提取域名
func extractDomainFromEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(parts[1])
}

// TestConnection 测试网易邮箱连接
func (p *NetEaseProvider) TestConnection(ctx context.Context, account *models.EmailAccount) error {
	// 确保配置正确
	p.ensureNetEaseConfig(account)

	// 验证授权码
	if err := p.validateAuthCode(account); err != nil {
		return err
	}

	// 调用基类测试方法
	return p.BaseProvider.TestConnection(ctx, account)
}

// GetSpecialFolders 获取网易邮箱特殊文件夹映射
func (p *NetEaseProvider) GetSpecialFolders() map[string]string {
	return map[string]string{
		"inbox":  "INBOX",
		"sent":   "已发送",
		"drafts": "草稿箱",
		"trash":  "已删除",
		"spam":   "垃圾邮件",
	}
}

// GetFolderDisplayName 获取文件夹显示名称
func (p *NetEaseProvider) GetFolderDisplayName(folderName string) string {
	displayNames := map[string]string{
		"INBOX":  "收件箱",
		"已发送":    "已发送",
		"草稿箱":    "草稿箱",
		"已删除":    "已删除",
		"垃圾邮件":   "垃圾邮件",
		"Sent":   "已发送",
		"Drafts": "草稿箱",
		"Trash":  "已删除",
		"Spam":   "垃圾邮件",
		"Junk":   "垃圾邮件",
	}

	if displayName, exists := displayNames[folderName]; exists {
		return displayName
	}

	return folderName
}

// SendEmail 发送网易邮箱邮件
func (p *NetEaseProvider) SendEmail(ctx context.Context, account *models.EmailAccount, message *OutgoingMessage) error {
	if !p.IsConnected() {
		if err := p.Connect(ctx, account); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
	}

	smtpClient := p.SMTPClient()
	if smtpClient == nil {
		return fmt.Errorf("SMTP client not available")
	}

	// 网易邮箱特殊处理：确保发件人地址正确
	if message.From == nil {
		message.From = &models.EmailAddress{
			Address: account.Email,
			Name:    account.Name,
		}
	}

	// 验证发件人地址是否匹配账户
	if message.From.Address != account.Email {
		return fmt.Errorf("sender address must match account email for NetEase mail")
	}

	// 检查发送频率限制
	if err := p.checkSendingLimits(message); err != nil {
		return err
	}

	// 发送邮件
	return smtpClient.SendEmail(ctx, message)
}

// checkSendingLimits 检查发送限制
func (p *NetEaseProvider) checkSendingLimits(message *OutgoingMessage) error {
	// 检查收件人数量
	totalRecipients := len(message.To) + len(message.CC) + len(message.BCC)
	if totalRecipients > 100 {
		return fmt.Errorf("too many recipients")
	}

	// 检查邮件大小（这里简化处理，实际应该计算完整邮件大小）
	if len(message.TextBody)+len(message.HTMLBody) > 10*1024*1024 {
		return fmt.Errorf("message too large")
	}

	return nil
}

// ValidateEmailAddress 验证网易邮箱地址格式
func (p *NetEaseProvider) ValidateEmailAddress(email string) error {
	email = strings.ToLower(email)

	// 检查是否是支持的网易邮箱域名
	supportedDomains := []string{"163.com", "126.com", "yeah.net", "188.com", "vip.163.com", "vip.126.com"}

	for _, domain := range supportedDomains {
		if strings.HasSuffix(email, "@"+domain) {
			return nil
		}
	}

	return fmt.Errorf("unsupported NetEase mail domain. Supported domains: %s", strings.Join(supportedDomains, ", "))
}

// GetAuthCodeInstructions 获取网易邮箱授权码设置说明
func (p *NetEaseProvider) GetAuthCodeInstructions() string {
	return `网易邮箱客户端授权码设置步骤：
1. 登录网易邮箱网页版（163.com/126.com/yeah.net）
2. 点击"设置" -> "POP3/SMTP/IMAP"
3. 开启"IMAP/SMTP服务"或"POP3/SMTP服务"
4. 点击"客户端授权密码"
5. 按照提示发送短信验证
6. 获得16位客户端授权码
7. 将授权码作为密码使用

注意：
- 授权码不是邮箱登录密码，是专门用于第三方客户端的密码
- 授权码为16位字符，只包含字母和数字
- 如果忘记授权码，可以重新生成
- 不同的网易邮箱（163/126/yeah）设置方法相同`
}

// GetProviderInfo 获取提供商信息
func (p *NetEaseProvider) GetProviderInfo() map[string]interface{} {
	return map[string]interface{}{
		"name":         "网易邮箱",
		"display_name": "网易邮箱（163/126/yeah）",
		"auth_methods": []string{"password"},
		"domains":      []string{"163.com", "126.com", "yeah.net", "188.com", "vip.163.com", "vip.126.com"},
		"servers": map[string]interface{}{
			"163": map[string]interface{}{
				"imap": map[string]interface{}{
					"host":     "imap.163.com",
					"port":     993,
					"security": "SSL",
				},
				"smtp": map[string]interface{}{
					"host":     "smtp.163.com",
					"port":     465,
					"security": "SSL",
					"alt_port": 994,
				},
			},
			"126": map[string]interface{}{
				"imap": map[string]interface{}{
					"host":     "imap.126.com",
					"port":     993,
					"security": "SSL",
				},
				"smtp": map[string]interface{}{
					"host":     "smtp.126.com",
					"port":     465,
					"security": "SSL",
				},
			},
			"yeah": map[string]interface{}{
				"imap": map[string]interface{}{
					"host":     "imap.yeah.net",
					"port":     993,
					"security": "SSL",
				},
				"smtp": map[string]interface{}{
					"host":     "smtp.yeah.net",
					"port":     465,
					"security": "SSL",
				},
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
			"daily_send":        200,                    // 每日发送限制（免费邮箱）
			"hourly_send":       50,                     // 每小时发送限制
			"rate_limit_window": 3600,                   // 频率限制窗口（秒）
			"max_recipients":    100,                    // 单封邮件最大收件人数
			"mailbox_size":      3 * 1024 * 1024 * 1024, // 3GB 免费邮箱容量
			"connection_limit":  10,                     // 同时连接数限制
		},
		"error_codes": map[string]string{
			"535":    "认证失败，请检查邮箱地址和客户端授权码",
			"550":    "发送频率超限，请稍后重试",
			"554":    "邮件被拒绝，内容可能被识别为垃圾邮件",
			"451":    "临时失败，服务器繁忙或连接异常",
			"RP:CEL": "发送方错误指令过多，请检查客户端配置",
			"MI:DMC": "当前连接发送邮件数量超限，请控制发送频率",
			"421":    "服务暂时不可用，服务器繁忙",
			"452":    "存储空间不足，邮箱已满",
			"553":    "收件人地址无效或发件人未授权",
		},
		"help_urls": map[string]string{
			"163_help":   "http://help.mail.163.com/",
			"126_help":   "http://help.mail.126.com/",
			"yeah_help":  "http://help.mail.yeah.net/",
			"imap_setup": "http://help.mail.163.com/faqDetail.do?code=d7a5dc8471cd0c0e8b4b8f4f8e49998b374173cfe9171305fa1ce630d7f67ac2cce4f9a7a3d7e8e8b",
			"smtp_setup": "http://help.mail.163.com/faqDetail.do?code=d7a5dc8471cd0c0e8b4b8f4f8e49998b374173cfe9171305fa1ce630d7f67ac2cce4f9a7a3d7e8e8b",
			"auth_code":  "http://help.mail.163.com/faqDetail.do?code=d7a5dc8471cd0c0e8b4b8f4f8e49998b374173cfe9171305fa1ce630d7f67ac2cce4f9a7a3d7e8e8b",
		},
		"auth_instructions": p.GetAuthCodeInstructions(),
	}
}
