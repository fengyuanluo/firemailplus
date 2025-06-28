package external_oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// TokenResponse 外部OAuth服务器的token响应结构
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
}

// ErrorResponse 外部OAuth服务器的错误响应结构
type ErrorResponse struct {
	Error string `json:"error"`
}

// Client 外部OAuth服务器客户端
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient 创建新的外部OAuth服务器客户端
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetAuthURL 获取授权URL
func (c *Client) GetAuthURL(provider string) (string, error) {
	if provider != "gmail" && provider != "outlook" {
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}

	authURL := fmt.Sprintf("%s/auth/%s", c.baseURL, provider)
	return authURL, nil
}

// GetAuthURLWithRedirect 获取授权URL并指定重定向地址
func (c *Client) GetAuthURLWithRedirect(provider, redirectURI string) (string, error) {
	if provider != "gmail" && provider != "outlook" {
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}

	authURL := fmt.Sprintf("%s/auth/%s?redirect_uri=%s", c.baseURL, provider, url.QueryEscape(redirectURI))
	return authURL, nil
}

// ExchangeToken 使用授权码交换token
func (c *Client) ExchangeToken(ctx context.Context, provider, code, state string) (*TokenResponse, error) {
	if provider != "gmail" && provider != "outlook" {
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
	
	// 构建回调URL
	callbackURL := fmt.Sprintf("%s/callback/%s", c.baseURL, provider)
	
	// 添加查询参数
	params := url.Values{}
	params.Add("code", code)
	params.Add("state", state)
	
	fullURL := fmt.Sprintf("%s?%s", callbackURL, params.Encode())
	
	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		var errorResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return nil, fmt.Errorf("external OAuth server returned status %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("external OAuth server error: %s", errorResp.Error)
	}
	
	// 解析响应
	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}
	
	// 验证响应
	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("invalid token response: missing access_token")
	}
	
	return &tokenResp, nil
}

// HealthCheck 检查外部OAuth服务器健康状态
func (c *Client) HealthCheck(ctx context.Context) error {
	healthURL := fmt.Sprintf("%s/health", c.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send health check request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("external OAuth server health check failed with status %d", resp.StatusCode)
	}
	
	return nil
}
