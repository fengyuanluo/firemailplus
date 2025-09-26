package providers

import (
	"context"
	"fmt"
	"log"
	"sync"

	"firemail/internal/config"
	"firemail/internal/models"
)

// TokenUpdateCallback OAuth2 token更新回调函数类型
type TokenUpdateCallback func(ctx context.Context, account *models.EmailAccount) error

// TokenCallbackSetter 支持设置token更新回调的接口
type TokenCallbackSetter interface {
	SetTokenUpdateCallback(callback TokenUpdateCallback)
}

// BaseProvider 基础邮件提供商实现
type BaseProvider struct {
	config              *config.EmailProviderConfig
	imapClient          IMAPClient
	smtpClient          SMTPClient
	oauth2Client        OAuth2Client
	connected           bool // 保持向后兼容，表示任一连接成功
	imapConnected       bool // IMAP连接状态
	smtpConnected       bool // SMTP连接状态
	mutex               sync.RWMutex
	tokenUpdateCallback TokenUpdateCallback // OAuth2 token更新回调
}

// NewBaseProvider 创建基础提供商
func NewBaseProvider(config *config.EmailProviderConfig) *BaseProvider {
	return &BaseProvider{
		config:    config,
		connected: false,
	}
}

// SetTokenUpdateCallback 设置OAuth2 token更新回调
func (p *BaseProvider) SetTokenUpdateCallback(callback TokenUpdateCallback) {
	p.tokenUpdateCallback = callback
}

// EnsureValidToken 确保OAuth2 token有效，如果需要则自动刷新
func (p *BaseProvider) EnsureValidToken(ctx context.Context, account *models.EmailAccount) error {
	// 只对OAuth2认证的账户进行检查
	if account.AuthMethod != "oauth2" {
		return nil
	}

	// 检查是否需要刷新token
	if !account.NeedsOAuth2Refresh() {
		return nil // token仍然有效
	}

	// 如果没有OAuth2客户端，无法刷新
	if p.oauth2Client == nil {
		return fmt.Errorf("OAuth2 client not available for token refresh")
	}

	log.Printf("Proactively refreshing OAuth2 token for account %s (%s)", account.Email, account.Provider)

	// 获取当前token
	tokenData, err := account.GetOAuth2Token()
	if err != nil {
		return fmt.Errorf("failed to get current OAuth2 token: %w", err)
	}

	// 刷新token
	newToken, err := p.oauth2Client.RefreshToken(ctx, tokenData.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to refresh OAuth2 token: %w", err)
	}

	// 更新account中的token数据
	newTokenData := &models.OAuth2TokenData{
		AccessToken:  newToken.AccessToken,
		RefreshToken: newToken.RefreshToken,
		TokenType:    newToken.TokenType,
		Expiry:       newToken.Expiry,
		Scope:        newToken.Scope,
	}
	if err := account.SetOAuth2Token(newTokenData); err != nil {
		return fmt.Errorf("failed to update OAuth2 token in memory: %w", err)
	}

	// 通过回调保存到数据库
	if p.tokenUpdateCallback != nil {
		if err := p.tokenUpdateCallback(ctx, account); err != nil {
			log.Printf("Warning: Failed to save refreshed OAuth2 token to database: %v", err)
			// 不要因为数据库保存失败而返回错误，token已经在内存中更新
		} else {
			log.Printf("Successfully refreshed and saved OAuth2 token for account %s", account.Email)
		}
	} else {
		log.Printf("Warning: No token update callback set, refreshed token not saved to database")
	}

	// 如果已连接，需要重新连接以使用新token
	if p.connected {
		log.Printf("Reconnecting with refreshed token for account %s", account.Email)
		if err := p.Disconnect(); err != nil {
			log.Printf("Warning: Failed to disconnect before reconnecting: %v", err)
		}
		if err := p.Connect(ctx, account); err != nil {
			return fmt.Errorf("failed to reconnect with refreshed token: %w", err)
		}
	}

	return nil
}

// GetName 获取提供商名称
func (p *BaseProvider) GetName() string {
	return p.config.Name
}

// GetDisplayName 获取提供商显示名称
func (p *BaseProvider) GetDisplayName() string {
	return p.config.DisplayName
}

// GetSupportedAuthMethods 获取支持的认证方式
func (p *BaseProvider) GetSupportedAuthMethods() []string {
	return p.config.AuthMethods
}

// GetProviderInfo 获取提供商信息
func (p *BaseProvider) GetProviderInfo() map[string]interface{} {
	oauth2Enabled := p.config.OAuth2Config != nil &&
		len(p.config.OAuth2Config.AuthURL) > 0 &&
		len(p.config.OAuth2Config.TokenURL) > 0

	return map[string]interface{}{
		"name":           p.config.Name,
		"display_name":   p.config.DisplayName,
		"auth_methods":   p.config.AuthMethods,
		"imap_host":      p.config.IMAPHost,
		"imap_port":      p.config.IMAPPort,
		"smtp_host":      p.config.SMTPHost,
		"smtp_port":      p.config.SMTPPort,
		"domains":        p.config.Domains,
		"oauth2_enabled": oauth2Enabled,
	}
}

// Connect 连接到邮件服务器
func (p *BaseProvider) Connect(ctx context.Context, account *models.EmailAccount) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.connected {
		return nil
	}

	// 根据认证方式连接
	switch account.AuthMethod {
	case "password":
		return p.connectWithPassword(ctx, account)
	case "oauth2":
		return p.connectWithOAuth2(ctx, account)
	default:
		return fmt.Errorf("unsupported auth method: %s", account.AuthMethod)
	}
}

// connectWithPassword 使用密码连接
func (p *BaseProvider) connectWithPassword(ctx context.Context, account *models.EmailAccount) error {
	var imapErr, smtpErr error

	// 重置连接状态
	p.imapConnected = false
	p.smtpConnected = false

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
		if err := p.imapClient.Connect(ctx, imapConfig); err != nil {
			imapErr = fmt.Errorf("failed to connect IMAP: %w", err)
			log.Printf("IMAP connection failed: %v", imapErr)
		} else {
			p.imapConnected = true
			log.Printf("IMAP connection successful")
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
			smtpErr = fmt.Errorf("failed to connect SMTP: %w", err)
			log.Printf("SMTP connection failed: %v", smtpErr)
		} else {
			p.smtpConnected = true
			log.Printf("SMTP connection successful")
		}
	}

	// 更新总体连接状态
	p.connected = p.imapConnected || p.smtpConnected

	// 如果两者都失败，返回错误
	if !p.imapConnected && !p.smtpConnected {
		if imapErr != nil && smtpErr != nil {
			return fmt.Errorf("both IMAP and SMTP connections failed - IMAP: %v, SMTP: %v", imapErr, smtpErr)
		} else if imapErr != nil {
			return imapErr
		} else if smtpErr != nil {
			return smtpErr
		}
	}

	// 如果至少一个成功，记录部分连接状态
	if p.imapConnected && !p.smtpConnected && smtpErr != nil {
		log.Printf("Partial connection: IMAP connected, SMTP failed: %v", smtpErr)
	} else if !p.imapConnected && p.smtpConnected && imapErr != nil {
		log.Printf("Partial connection: SMTP connected, IMAP failed: %v", imapErr)
	}

	return nil
}

// connectWithOAuth2 使用OAuth2连接
func (p *BaseProvider) connectWithOAuth2(ctx context.Context, account *models.EmailAccount) error {
	// 获取OAuth2 token
	tokenData, err := account.GetOAuth2Token()
	if err != nil {
		return fmt.Errorf("failed to get OAuth2 token: %w", err)
	}

	if tokenData == nil {
		return fmt.Errorf("OAuth2 token not found")
	}

	oauth2Token := &OAuth2Token{
		AccessToken:  tokenData.AccessToken,
		RefreshToken: tokenData.RefreshToken,
		TokenType:    tokenData.TokenType,
		Expiry:       tokenData.Expiry,
		Scope:        tokenData.Scope,
	}

	// 检查token是否需要刷新
	if account.NeedsOAuth2Refresh() && p.oauth2Client != nil {
		log.Printf("OAuth2 token needs refresh for account %s (%s)", account.Email, account.Provider)

		newToken, err := p.oauth2Client.RefreshToken(ctx, oauth2Token.RefreshToken)
		if err != nil {
			return fmt.Errorf("failed to refresh OAuth2 token: %w", err)
		}
		oauth2Token = newToken

		// 更新account中的token数据
		newTokenData := &models.OAuth2TokenData{
			AccessToken:  newToken.AccessToken,
			RefreshToken: newToken.RefreshToken,
			TokenType:    newToken.TokenType,
			Expiry:       newToken.Expiry,
			Scope:        newToken.Scope,
		}
		if err := account.SetOAuth2Token(newTokenData); err != nil {
			return fmt.Errorf("failed to update OAuth2 token in memory: %w", err)
		}

		// 通过回调保存到数据库
		if p.tokenUpdateCallback != nil {
			if err := p.tokenUpdateCallback(ctx, account); err != nil {
				log.Printf("Warning: Failed to save refreshed OAuth2 token to database: %v", err)
				// 不要因为数据库保存失败而中断连接，只记录警告
			} else {
				log.Printf("Successfully refreshed and saved OAuth2 token for account %s", account.Email)
			}
		} else {
			log.Printf("Warning: No token update callback set, refreshed token not saved to database")
		}
	}

	var imapErr, smtpErr error

	// 重置连接状态
	p.imapConnected = false
	p.smtpConnected = false

	// 连接IMAP
	if p.imapClient != nil {
		imapConfig := IMAPClientConfig{
			Host:        account.IMAPHost,
			Port:        account.IMAPPort,
			Security:    account.IMAPSecurity,
			Username:    account.Username,
			OAuth2Token: oauth2Token,
			ProxyConfig: p.createProxyConfig(account),
		}
		if err := p.imapClient.Connect(ctx, imapConfig); err != nil {
			imapErr = fmt.Errorf("failed to connect IMAP with OAuth2: %w", err)
			log.Printf("IMAP OAuth2 connection failed: %v", imapErr)
		} else {
			p.imapConnected = true
			log.Printf("IMAP OAuth2 connection successful")
		}
	}

	// 连接SMTP
	if p.smtpClient != nil {
		smtpConfig := SMTPClientConfig{
			Host:        account.SMTPHost,
			Port:        account.SMTPPort,
			Security:    account.SMTPSecurity,
			Username:    account.Username,
			OAuth2Token: oauth2Token,
			ProxyConfig: p.createProxyConfig(account),
		}
		if err := p.smtpClient.Connect(ctx, smtpConfig); err != nil {
			smtpErr = fmt.Errorf("failed to connect SMTP with OAuth2: %w", err)
			log.Printf("SMTP OAuth2 connection failed: %v", smtpErr)
		} else {
			p.smtpConnected = true
			log.Printf("SMTP OAuth2 connection successful")
		}
	}

	// 更新总体连接状态
	p.connected = p.imapConnected || p.smtpConnected

	// 如果两者都失败，返回错误
	if !p.imapConnected && !p.smtpConnected {
		if imapErr != nil && smtpErr != nil {
			return fmt.Errorf("both IMAP and SMTP OAuth2 connections failed - IMAP: %v, SMTP: %v", imapErr, smtpErr)
		} else if imapErr != nil {
			return imapErr
		} else if smtpErr != nil {
			return smtpErr
		}
	}

	// 如果至少一个成功，记录部分连接状态
	if p.imapConnected && !p.smtpConnected && smtpErr != nil {
		log.Printf("Partial OAuth2 connection: IMAP connected, SMTP failed: %v", smtpErr)
	} else if !p.imapConnected && p.smtpConnected && imapErr != nil {
		log.Printf("Partial OAuth2 connection: SMTP connected, IMAP failed: %v", imapErr)
	}

	return nil
}

// Disconnect 断开连接
func (p *BaseProvider) Disconnect() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.connected {
		return nil
	}

	var errors []error

	// 断开IMAP连接
	if p.imapClient != nil && p.imapConnected {
		if err := p.imapClient.Disconnect(); err != nil {
			errors = append(errors, fmt.Errorf("failed to disconnect IMAP: %w", err))
		} else {
			p.imapConnected = false
			log.Printf("IMAP disconnected successfully")
		}
	}

	// 断开SMTP连接
	if p.smtpClient != nil && p.smtpConnected {
		if err := p.smtpClient.Disconnect(); err != nil {
			errors = append(errors, fmt.Errorf("failed to disconnect SMTP: %w", err))
		} else {
			p.smtpConnected = false
			log.Printf("SMTP disconnected successfully")
		}
	}

	// 更新总体连接状态
	p.connected = p.imapConnected || p.smtpConnected

	if len(errors) > 0 {
		return fmt.Errorf("disconnect errors: %v", errors)
	}

	return nil
}

// IsConnected 检查是否已连接（任一连接成功即为true）
func (p *BaseProvider) IsConnected() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.connected
}

// IsIMAPConnected 检查IMAP是否已连接
func (p *BaseProvider) IsIMAPConnected() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.imapConnected
}

// IsSMTPConnected 检查SMTP是否已连接
func (p *BaseProvider) IsSMTPConnected() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.smtpConnected
}

// TestConnection 测试连接
func (p *BaseProvider) TestConnection(ctx context.Context, account *models.EmailAccount) error {
	// 临时连接测试
	if err := p.Connect(ctx, account); err != nil {
		return err
	}

	// 测试IMAP连接
	if p.imapClient != nil && p.imapClient.IsConnected() {
		if _, err := p.imapClient.ListFolders(ctx); err != nil {
			return fmt.Errorf("IMAP test failed: %w", err)
		}
	}

	// 测试SMTP连接（这里只测试连接，不发送邮件）
	if p.smtpClient != nil && p.smtpClient.IsConnected() {
		// SMTP连接测试通常在连接时就会验证
	}

	return nil
}

// IMAPClient 获取IMAP客户端
func (p *BaseProvider) IMAPClient() IMAPClient {
	return p.imapClient
}

// SMTPClient 获取SMTP客户端
func (p *BaseProvider) SMTPClient() SMTPClient {
	return p.smtpClient
}

// OAuth2Client 获取OAuth2客户端
func (p *BaseProvider) OAuth2Client() OAuth2Client {
	return p.oauth2Client
}

// SetIMAPClient 设置IMAP客户端
func (p *BaseProvider) SetIMAPClient(client IMAPClient) {
	p.imapClient = client
}

// SetSMTPClient 设置SMTP客户端
func (p *BaseProvider) SetSMTPClient(client SMTPClient) {
	p.smtpClient = client
}

// SetOAuth2Client 设置OAuth2客户端
func (p *BaseProvider) SetOAuth2Client(client OAuth2Client) {
	p.oauth2Client = client
}

// SyncEmails 同步邮件（默认实现）
func (p *BaseProvider) SyncEmails(ctx context.Context, account *models.EmailAccount, folderName string, lastUID uint32) ([]*EmailMessage, error) {
	// 确保token有效（如果需要则自动刷新）
	if err := p.EnsureValidToken(ctx, account); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	// 检查IMAP连接状态
	if !p.IsIMAPConnected() {
		if err := p.Connect(ctx, account); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
		// 连接后再次检查IMAP状态
		if !p.IsIMAPConnected() {
			return nil, fmt.Errorf("IMAP connection not available")
		}
	}

	imapClient := p.IMAPClient()
	if imapClient == nil {
		return nil, fmt.Errorf("IMAP client not available")
	}

	// 获取新邮件
	return imapClient.GetNewEmails(ctx, folderName, lastUID)
}

// SendEmail 发送邮件（默认实现）
func (p *BaseProvider) SendEmail(ctx context.Context, account *models.EmailAccount, message *OutgoingMessage) error {
	// 确保token有效（如果需要则自动刷新）
	if err := p.EnsureValidToken(ctx, account); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	// 检查SMTP连接状态
	if !p.IsSMTPConnected() {
		if err := p.Connect(ctx, account); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		// 连接后再次检查SMTP状态
		if !p.IsSMTPConnected() {
			return fmt.Errorf("SMTP connection not available")
		}
	}

	smtpClient := p.SMTPClient()
	if smtpClient == nil {
		return fmt.Errorf("SMTP client not available")
	}

	// 确保发件人地址正确
	if message.From == nil {
		message.From = &models.EmailAddress{
			Address: account.Email,
			Name:    account.Name,
		}
	}

	// 发送邮件
	return smtpClient.SendEmail(ctx, message)
}

// createProxyConfig 从EmailAccount创建代理配置
func (p *BaseProvider) createProxyConfig(account *models.EmailAccount) *ProxyConfig {
	// 如果没有配置代理URL，返回nil
	if account.ProxyURL == "" {
		return nil
	}

	// 获取解析后的代理配置
	proxyData := account.GetProxyConfig()
	if proxyData.Type == "none" {
		return nil
	}

	return &ProxyConfig{
		Type:     proxyData.Type,
		Host:     proxyData.Host,
		Port:     proxyData.Port,
		Username: proxyData.Username,
		Password: proxyData.Password,
	}
}
