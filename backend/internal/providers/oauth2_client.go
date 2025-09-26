package providers

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"firemail/internal/proxy"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// StandardOAuth2Client 标准OAuth2客户端实现
type StandardOAuth2Client struct {
	config      *oauth2.Config
	proxyConfig *ProxyConfig
}

// NewStandardOAuth2Client 创建标准OAuth2客户端
func NewStandardOAuth2Client(clientID, clientSecret, authURL, tokenURL, redirectURL string, scopes []string) *StandardOAuth2Client {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
		RedirectURL: redirectURL,
		Scopes:      scopes,
	}

	return &StandardOAuth2Client{
		config:      config,
		proxyConfig: nil,
	}
}

// SetProxyConfig 设置代理配置
func (c *StandardOAuth2Client) SetProxyConfig(config *ProxyConfig) {
	c.proxyConfig = config
}

// GetAuthURL 获取授权URL
func (c *StandardOAuth2Client) GetAuthURL(state string, scopes []string) string {
	// 如果提供了特定的scopes，使用它们；否则使用默认的
	if len(scopes) > 0 {
		// 创建临时配置
		tempConfig := *c.config
		tempConfig.Scopes = scopes
		return tempConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	}

	return c.config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// ExchangeCode 交换授权码获取token
func (c *StandardOAuth2Client) ExchangeCode(ctx context.Context, code string) (*OAuth2Token, error) {
	token, err := c.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	return convertOAuth2Token(token), nil
}

// RefreshToken 刷新访问令牌
func (c *StandardOAuth2Client) RefreshToken(ctx context.Context, refreshToken string) (*OAuth2Token, error) {
	tokenSource := c.config.TokenSource(ctx, &oauth2.Token{
		RefreshToken: refreshToken,
	})

	token, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return convertOAuth2Token(token), nil
}

// ValidateToken 验证令牌
func (c *StandardOAuth2Client) ValidateToken(ctx context.Context, token *OAuth2Token) error {
	// 检查token是否过期
	if time.Now().After(token.Expiry) {
		return fmt.Errorf("token has expired")
	}

	// 可以通过调用API来验证token的有效性
	// 这里实现一个通用的验证方法
	return c.validateTokenWithAPI(ctx, token)
}

// RevokeToken 撤销令牌
func (c *StandardOAuth2Client) RevokeToken(ctx context.Context, token string) error {
	// 构建撤销请求
	revokeURL := c.getRevokeURL()
	if revokeURL == "" {
		return fmt.Errorf("revoke URL not configured")
	}

	data := url.Values{}
	data.Set("token", token)

	req, err := http.NewRequestWithContext(ctx, "POST", revokeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create revoke request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to revoke token, status: %d", resp.StatusCode)
	}

	return nil
}

// validateTokenWithAPI 通过API验证token
func (c *StandardOAuth2Client) validateTokenWithAPI(ctx context.Context, token *OAuth2Token) error {
	// 这是一个通用的验证方法，具体的提供商可能需要重写
	client := c.config.Client(ctx, convertToOAuth2Token(token))

	// 尝试调用一个简单的API来验证token
	validationURL := c.getValidationURL()
	if validationURL == "" {
		// 如果没有验证URL，只检查过期时间
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", validationURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create validation request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("token is invalid or expired")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token validation failed, status: %d", resp.StatusCode)
	}

	return nil
}

// getRevokeURL 获取撤销URL（需要子类实现）
func (c *StandardOAuth2Client) getRevokeURL() string {
	// 这个方法应该由具体的提供商实现
	return ""
}

// getValidationURL 获取验证URL（需要子类实现）
func (c *StandardOAuth2Client) getValidationURL() string {
	// 这个方法应该由具体的提供商实现
	return ""
}

// convertOAuth2Token 转换golang.org/x/oauth2.Token为自定义Token
func convertOAuth2Token(token *oauth2.Token) *OAuth2Token {
	return &OAuth2Token{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry,
	}
}

// convertToOAuth2Token 转换自定义Token为golang.org/x/oauth2.Token
func convertToOAuth2Token(token *OAuth2Token) *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry,
	}
}

// GmailOAuth2Client Gmail OAuth2客户端
type GmailOAuth2Client struct {
	*StandardOAuth2Client
}

// NewGmailOAuth2Client 创建Gmail OAuth2客户端
func NewGmailOAuth2Client(clientID, clientSecret, redirectURL string) *GmailOAuth2Client {
	scopes := []string{"https://mail.google.com/"}

	client := NewStandardOAuth2Client(
		clientID,
		clientSecret,
		"https://accounts.google.com/o/oauth2/auth",
		"https://oauth2.googleapis.com/token",
		redirectURL,
		scopes,
	)

	return &GmailOAuth2Client{
		StandardOAuth2Client: client,
	}
}

// OutlookOAuth2Client Outlook OAuth2客户端 - 严格按照Python代码重写
type OutlookOAuth2Client struct {
	ClientID    string
	httpClient  *http.Client
	proxyConfig *ProxyConfig
}

// NewOutlookOAuth2Client 创建Outlook OAuth2客户端 - 简化版本，只支持手动配置
func NewOutlookOAuth2Client(clientID, clientSecret, redirectURL string) *OutlookOAuth2Client {
	return &OutlookOAuth2Client{
		ClientID:    clientID,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		proxyConfig: nil,
	}
}

// SetProxyConfig 设置代理配置
func (c *OutlookOAuth2Client) SetProxyConfig(config *ProxyConfig) {
	c.proxyConfig = config
	// 重新创建httpClient以应用代理配置
	c.httpClient = c.createHTTPClient()
}

// createHTTPClient 创建自定义HTTP客户端（支持代理）
func (c *OutlookOAuth2Client) createHTTPClient() *http.Client {
	// 如果没有代理配置，返回默认客户端
	if c.proxyConfig == nil {
		return &http.Client{Timeout: 30 * time.Second}
	}

	// 创建自定义Transport
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		DialContext: (&net.Dialer{
			Timeout: 30 * time.Second,
		}).DialContext,
	}

	// 根据代理类型配置
	switch c.proxyConfig.Type {
	case "http", "https":
		// HTTP/HTTPS代理
		proxyURL := &url.URL{
			Scheme: "http", // HTTPS代理实际上也是HTTP协议
			Host:   fmt.Sprintf("%s:%d", c.proxyConfig.Host, c.proxyConfig.Port),
		}
		if c.proxyConfig.Username != "" {
			proxyURL.User = url.UserPassword(c.proxyConfig.Username, c.proxyConfig.Password)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	case "socks5":
		// SOCKS5代理需要使用自定义DialContext
		proxyDialer, err := proxy.CreateDialer(c.proxyConfig.ToProxyConfig())
		if err == nil {
			transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return proxyDialer.Dial(network, addr)
			}
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
}

// RefreshToken 刷新访问令牌 - 严格按照Python代码实现
func (c *OutlookOAuth2Client) RefreshToken(ctx context.Context, refreshToken string) (*OAuth2Token, error) {
	// 严格按照Python代码：def get_new_access_token(refresh_token)
	tenantID := "common"
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)

	fmt.Printf("🔄 [DEBUG] Starting token refresh for client_id: %s\n", c.ClientID)
	fmt.Printf("🔄 [DEBUG] Token URL: %s\n", tokenURL)
	fmt.Printf("🔄 [DEBUG] Refresh token (first 20 chars): %s...\n", refreshToken[:20])

	// 构建请求数据，严格按照Python代码格式
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", c.ClientID)
	// 手动配置模式下不需要client_secret

	fmt.Printf("🔄 [DEBUG] Request data: grant_type=refresh_token, client_id=%s\n", c.ClientID)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		fmt.Printf("❌ [DEBUG] Failed to create request: %v\n", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	fmt.Printf("🔄 [DEBUG] Sending token refresh request...\n")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		fmt.Printf("❌ [DEBUG] Failed to send request: %v\n", err)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("🔄 [DEBUG] Response status: %d\n", resp.StatusCode)

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("❌ [DEBUG] Token refresh failed with status %d: %s\n", resp.StatusCode, string(body))
		return nil, fmt.Errorf("Error: %d - %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ [DEBUG] Failed to read response body: %v\n", err)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	fmt.Printf("🔄 [DEBUG] Response body (first 200 chars): %s...\n", string(body)[:min(200, len(body))])

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		fmt.Printf("❌ [DEBUG] Failed to parse token response: %v\n", err)
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		fmt.Printf("❌ [DEBUG] No access token in response\n")
		return nil, fmt.Errorf("no access token in response")
	}

	fmt.Printf("✅ [DEBUG] Successfully obtained access token (first 20 chars): %s...\n", tokenResp.AccessToken[:20])
	if tokenResp.RefreshToken != "" {
		fmt.Printf("✅ [DEBUG] Successfully obtained new refresh token (first 20 chars): %s...\n", tokenResp.RefreshToken[:20])
	}

	// 计算过期时间
	expiry := time.Now().Add(3600 * time.Second) // 默认1小时过期
	if tokenResp.ExpiresIn > 0 {
		expiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	// 返回完整的token结构，包含新的refresh_token
	result := &OAuth2Token{
		AccessToken: tokenResp.AccessToken,
		TokenType:   "Bearer",
		Expiry:      expiry,
	}

	// 如果响应中包含新的refresh_token，使用新的；否则保持原有的
	if tokenResp.RefreshToken != "" {
		result.RefreshToken = tokenResp.RefreshToken
	} else {
		result.RefreshToken = refreshToken // 保持原有的refresh_token
	}

	return result, nil
}

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetAuthURL 获取授权URL - 删除，只支持手动配置
func (c *OutlookOAuth2Client) GetAuthURL(state string, scopes []string) string {
	return ""
}

// ExchangeCode 交换授权码获取token - 删除，只支持手动配置
func (c *OutlookOAuth2Client) ExchangeCode(ctx context.Context, code string) (*OAuth2Token, error) {
	return nil, fmt.Errorf("web callback authentication not supported, use manual configuration")
}

// ValidateToken 验证令牌 - 简化版本
func (c *OutlookOAuth2Client) ValidateToken(ctx context.Context, token *OAuth2Token) error {
	// 只检查token是否过期
	if time.Now().After(token.Expiry) {
		return fmt.Errorf("token has expired")
	}
	return nil
}

// RevokeToken 撤销令牌
func (c *OutlookOAuth2Client) RevokeToken(ctx context.Context, token string) error {
	// Microsoft Graph没有标准的撤销端点
	// 通常通过删除应用授权来撤销，这里返回不支持的错误
	return fmt.Errorf("microsoft Graph does not support token revocation via API. Please revoke access through Azure Portal or Microsoft account settings")
}

// TokenInfo token信息结构
type TokenInfo struct {
	Audience  string `json:"aud"`
	ClientID  string `json:"client_id"`
	ExpiresIn int    `json:"expires_in"`
	IssuedTo  string `json:"issued_to"`
	Scope     string `json:"scope"`
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
}

// GetTokenInfo 获取token信息（Gmail专用）
func (c *GmailOAuth2Client) GetTokenInfo(ctx context.Context, accessToken string) (*TokenInfo, error) {
	url := fmt.Sprintf("https://www.googleapis.com/oauth2/v1/tokeninfo?access_token=%s", accessToken)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get token info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get token info, status: %d", resp.StatusCode)
	}

	var tokenInfo TokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
		return nil, fmt.Errorf("failed to decode token info: %w", err)
	}

	return &tokenInfo, nil
}
