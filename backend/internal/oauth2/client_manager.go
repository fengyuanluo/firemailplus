package oauth2

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// OAuth2Client OAuth2客户端接口
type OAuth2Client interface {
	GetAuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error)
	RefreshToken(ctx context.Context, token *oauth2.Token) (*oauth2.Token, error)
	GetHTTPClient(ctx context.Context, token *oauth2.Token) *http.Client
	ValidateToken(ctx context.Context, token *oauth2.Token) error
	RevokeToken(ctx context.Context, token *oauth2.Token) error
}

// OAuth2ClientManager OAuth2客户端管理器接口
type OAuth2ClientManager interface {
	GetClient(provider string) (OAuth2Client, error)
	CreateClient(provider string) (OAuth2Client, error)
	RefreshClient(provider string) error
	ListClients() []string
}

// StandardOAuth2Client 标准OAuth2客户端实现
type StandardOAuth2Client struct {
	config   OAuth2Config
	oauth2Config *oauth2.Config
	provider string
	
	// 客户端元数据
	createdAt time.Time
	lastUsed  time.Time
	mutex     sync.RWMutex
}

// NewStandardOAuth2Client 创建标准OAuth2客户端
func NewStandardOAuth2Client(config OAuth2Config) OAuth2Client {
	return &StandardOAuth2Client{
		config:       config,
		oauth2Config: config.GetConfig(),
		provider:     config.GetProvider(),
		createdAt:    time.Now(),
		lastUsed:     time.Now(),
	}
}

// GetAuthURL 获取授权URL
func (c *StandardOAuth2Client) GetAuthURL(state string) string {
	c.updateLastUsed()
	
	// 添加额外的OAuth2选项
	options := []oauth2.AuthCodeOption{
		oauth2.AccessTypeOffline, // 获取refresh token
	}
	
	// 根据提供商添加特定选项
	switch c.provider {
	case "gmail":
		options = append(options, oauth2.SetAuthURLParam("prompt", "consent"))
	case "outlook":
		options = append(options, oauth2.SetAuthURLParam("prompt", "consent"))
	}
	
	return c.oauth2Config.AuthCodeURL(state, options...)
}

// ExchangeCode 交换授权码获取token
func (c *StandardOAuth2Client) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	c.updateLastUsed()
	
	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	token, err := c.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}
	
	// 验证token
	if err := c.ValidateToken(ctx, token); err != nil {
		return nil, fmt.Errorf("invalid token received: %w", err)
	}
	
	return token, nil
}

// RefreshToken 刷新token
func (c *StandardOAuth2Client) RefreshToken(ctx context.Context, token *oauth2.Token) (*oauth2.Token, error) {
	c.updateLastUsed()
	
	if token.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}
	
	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	tokenSource := c.oauth2Config.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	
	// 验证新token
	if err := c.ValidateToken(ctx, newToken); err != nil {
		return nil, fmt.Errorf("invalid refreshed token: %w", err)
	}
	
	return newToken, nil
}

// GetHTTPClient 获取HTTP客户端
func (c *StandardOAuth2Client) GetHTTPClient(ctx context.Context, token *oauth2.Token) *http.Client {
	c.updateLastUsed()
	
	return c.oauth2Config.Client(ctx, token)
}

// ValidateToken 验证token
func (c *StandardOAuth2Client) ValidateToken(ctx context.Context, token *oauth2.Token) error {
	if token == nil {
		return fmt.Errorf("token is nil")
	}
	
	if token.AccessToken == "" {
		return fmt.Errorf("access token is empty")
	}
	
	if token.Expiry.Before(time.Now()) {
		return fmt.Errorf("token is expired")
	}
	
	// 根据提供商进行特定验证
	switch c.provider {
	case "gmail":
		return c.validateGmailToken(ctx, token)
	case "outlook":
		return c.validateOutlookToken(ctx, token)
	default:
		return nil // 基本验证已通过
	}
}

// RevokeToken 撤销token
func (c *StandardOAuth2Client) RevokeToken(ctx context.Context, token *oauth2.Token) error {
	c.updateLastUsed()
	
	// 根据提供商实现token撤销
	switch c.provider {
	case "gmail":
		return c.revokeGmailToken(ctx, token)
	case "outlook":
		return c.revokeOutlookToken(ctx, token)
	default:
		return fmt.Errorf("token revocation not supported for provider: %s", c.provider)
	}
}

// updateLastUsed 更新最后使用时间
func (c *StandardOAuth2Client) updateLastUsed() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.lastUsed = time.Now()
}

// validateGmailToken 验证Gmail token
func (c *StandardOAuth2Client) validateGmailToken(ctx context.Context, token *oauth2.Token) error {
	client := c.GetHTTPClient(ctx, token)
	
	// 调用Gmail API验证token
	resp, err := client.Get("https://www.googleapis.com/oauth2/v1/tokeninfo?access_token=" + token.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to validate Gmail token: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Gmail token validation failed with status: %d", resp.StatusCode)
	}
	
	return nil
}

// validateOutlookToken 验证Outlook token
func (c *StandardOAuth2Client) validateOutlookToken(ctx context.Context, token *oauth2.Token) error {
	client := c.GetHTTPClient(ctx, token)
	
	// 调用Microsoft Graph API验证token
	resp, err := client.Get("https://graph.microsoft.com/v1.0/me")
	if err != nil {
		return fmt.Errorf("failed to validate Outlook token: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Outlook token validation failed with status: %d", resp.StatusCode)
	}
	
	return nil
}

// revokeGmailToken 撤销Gmail token
func (c *StandardOAuth2Client) revokeGmailToken(ctx context.Context, token *oauth2.Token) error {
	client := &http.Client{Timeout: 10 * time.Second}
	
	revokeURL := "https://oauth2.googleapis.com/revoke?token=" + token.AccessToken
	resp, err := client.PostForm(revokeURL, nil)
	if err != nil {
		return fmt.Errorf("failed to revoke Gmail token: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Gmail token revocation failed with status: %d", resp.StatusCode)
	}
	
	return nil
}

// revokeOutlookToken 撤销Outlook token
func (c *StandardOAuth2Client) revokeOutlookToken(ctx context.Context, token *oauth2.Token) error {
	// Microsoft Graph API不直接支持token撤销
	// 通常需要通过应用程序管理界面撤销
	return fmt.Errorf("Outlook token revocation requires manual action through Azure portal")
}

// StandardOAuth2ClientManager 标准OAuth2客户端管理器
type StandardOAuth2ClientManager struct {
	clients       map[string]OAuth2Client
	configManager OAuth2ConfigManager
	mutex         sync.RWMutex
}

// NewStandardOAuth2ClientManager 创建标准OAuth2客户端管理器
func NewStandardOAuth2ClientManager(configManager OAuth2ConfigManager) OAuth2ClientManager {
	return &StandardOAuth2ClientManager{
		clients:       make(map[string]OAuth2Client),
		configManager: configManager,
	}
}

// GetClient 获取OAuth2客户端
func (m *StandardOAuth2ClientManager) GetClient(provider string) (OAuth2Client, error) {
	m.mutex.RLock()
	client, exists := m.clients[provider]
	m.mutex.RUnlock()
	
	if exists {
		return client, nil
	}
	
	// 创建新客户端
	return m.CreateClient(provider)
}

// CreateClient 创建OAuth2客户端
func (m *StandardOAuth2ClientManager) CreateClient(provider string) (OAuth2Client, error) {
	// 获取配置
	config, err := m.configManager.GetConfig(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get config for provider %s: %w", provider, err)
	}
	
	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config for provider %s: %w", provider, err)
	}
	
	// 创建客户端
	client := NewStandardOAuth2Client(config)
	
	// 存储客户端
	m.mutex.Lock()
	m.clients[provider] = client
	m.mutex.Unlock()
	
	return client, nil
}

// RefreshClient 刷新客户端
func (m *StandardOAuth2ClientManager) RefreshClient(provider string) error {
	// 删除现有客户端
	m.mutex.Lock()
	delete(m.clients, provider)
	m.mutex.Unlock()
	
	// 刷新配置
	if err := m.configManager.RefreshConfig(provider); err != nil {
		return fmt.Errorf("failed to refresh config for provider %s: %w", provider, err)
	}
	
	// 创建新客户端
	_, err := m.CreateClient(provider)
	return err
}

// ListClients 列出所有客户端
func (m *StandardOAuth2ClientManager) ListClients() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	clients := make([]string, 0, len(m.clients))
	for provider := range m.clients {
		clients = append(clients, provider)
	}
	
	return clients
}

// OAuth2Service OAuth2服务
type OAuth2Service struct {
	configManager OAuth2ConfigManager
	clientManager OAuth2ClientManager
}

// NewOAuth2Service 创建OAuth2服务
func NewOAuth2Service() *OAuth2Service {
	configManager := NewStandardOAuth2ConfigManager()
	clientManager := NewStandardOAuth2ClientManager(configManager)
	
	return &OAuth2Service{
		configManager: configManager,
		clientManager: clientManager,
	}
}

// GetAuthURL 获取授权URL
func (s *OAuth2Service) GetAuthURL(provider, state string) (string, error) {
	client, err := s.clientManager.GetClient(provider)
	if err != nil {
		return "", fmt.Errorf("failed to get OAuth2 client for provider %s: %w", provider, err)
	}
	
	return client.GetAuthURL(state), nil
}

// ExchangeCode 交换授权码
func (s *OAuth2Service) ExchangeCode(ctx context.Context, provider, code string) (*oauth2.Token, error) {
	client, err := s.clientManager.GetClient(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth2 client for provider %s: %w", provider, err)
	}
	
	return client.ExchangeCode(ctx, code)
}

// RefreshToken 刷新token
func (s *OAuth2Service) RefreshToken(ctx context.Context, provider string, token *oauth2.Token) (*oauth2.Token, error) {
	client, err := s.clientManager.GetClient(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth2 client for provider %s: %w", provider, err)
	}
	
	return client.RefreshToken(ctx, token)
}

// ValidateToken 验证token
func (s *OAuth2Service) ValidateToken(ctx context.Context, provider string, token *oauth2.Token) error {
	client, err := s.clientManager.GetClient(provider)
	if err != nil {
		return fmt.Errorf("failed to get OAuth2 client for provider %s: %w", provider, err)
	}
	
	return client.ValidateToken(ctx, token)
}

// GetHTTPClient 获取HTTP客户端
func (s *OAuth2Service) GetHTTPClient(ctx context.Context, provider string, token *oauth2.Token) (*http.Client, error) {
	client, err := s.clientManager.GetClient(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth2 client for provider %s: %w", provider, err)
	}
	
	return client.GetHTTPClient(ctx, token), nil
}
