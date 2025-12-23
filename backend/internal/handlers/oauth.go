package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"firemail/internal/external_oauth"
	"firemail/internal/models"
	"firemail/internal/providers"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// InitOAuth 通用OAuth2认证初始化
func (h *Handler) InitOAuth(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	provider := c.Param("provider")
	if provider == "" {
		h.respondWithError(c, http.StatusBadRequest, "Provider parameter is required")
		return
	}

	// 验证支持的提供商
	if provider != "gmail" && provider != "outlook" {
		h.respondWithError(c, http.StatusBadRequest, "Unsupported provider: "+provider)
		return
	}

	var oauth2Client providers.OAuth2Client
	var authURL string
	var state string

	switch provider {
	case "gmail":
		// 检查Gmail OAuth2配置
		if h.config.OAuth.Gmail.ClientID == "" || h.config.OAuth.Gmail.ClientSecret == "" {
			h.respondWithError(c, http.StatusServiceUnavailable, "Gmail OAuth2 not configured")
			return
		}

		// 创建Gmail OAuth2客户端
		oauth2Client = providers.NewGmailOAuth2Client(
			h.config.OAuth.Gmail.ClientID,
			h.config.OAuth.Gmail.ClientSecret,
			h.config.OAuth.Gmail.RedirectURL,
		)

		// 生成安全的state参数
		var err error
		state, err = h.oauthStateService.GenerateState(c.Request.Context(), userID, "gmail", map[string]string{
			"provider": "gmail",
			"flow":     "oauth2",
		})
		if err != nil {
			h.respondWithError(c, http.StatusInternalServerError, "Failed to generate state: "+err.Error())
			return
		}

	case "outlook":
		// 检查Outlook OAuth2配置
		if h.config.OAuth.Outlook.ClientID == "" || h.config.OAuth.Outlook.ClientSecret == "" {
			h.respondWithError(c, http.StatusServiceUnavailable, "Outlook OAuth2 not configured")
			return
		}

		// 创建Outlook OAuth2客户端
		oauth2Client = providers.NewOutlookOAuth2Client(
			h.config.OAuth.Outlook.ClientID,
			h.config.OAuth.Outlook.ClientSecret,
			h.config.OAuth.Outlook.RedirectURL,
		)

		// 生成安全的state参数
		var err error
		state, err = h.oauthStateService.GenerateState(c.Request.Context(), userID, "outlook", map[string]string{
			"provider": "outlook",
			"flow":     "oauth2",
		})
		if err != nil {
			h.respondWithError(c, http.StatusInternalServerError, "Failed to generate state: "+err.Error())
			return
		}
	}

	// 获取授权URL
	authURL = oauth2Client.GetAuthURL(state, nil)

	response := map[string]string{
		"auth_url": authURL,
		"state":    state,
	}

	h.respondWithSuccess(c, response, fmt.Sprintf("%s OAuth2 authorization URL generated", provider))
}

// InitGmailOAuth 初始化Gmail OAuth2认证
func (h *Handler) InitGmailOAuth(c *gin.Context) {
	// OAuth初始化不需要用户认证，因为这是OAuth流程的开始
	// 用户可能还没有登录系统，但需要通过OAuth来添加邮箱账户

	// 检查外部OAuth服务器是否启用
	if !h.config.OAuth.ExternalServer.Enabled {
		h.respondWithError(c, http.StatusServiceUnavailable, "External OAuth server is disabled")
		return
	}

	// 生成简单的state参数（不存储到数据库，避免外键约束问题）
	state := fmt.Sprintf("gmail_%d_%s", time.Now().Unix(), generateRandomString(16))

	// 获取前端传递的回调URL，如果没有则使用默认值
	frontendCallbackURL := c.Query("callback_url")
	if frontendCallbackURL == "" {
		frontendCallbackURL = "http://localhost:3000/oauth/callback"
	}

	// 获取外部OAuth服务器的授权URL，传递前端回调地址
	externalClient := external_oauth.NewClient(h.config.OAuth.ExternalServer.BaseURL)
	authURL, err := externalClient.GetAuthURLWithRedirect("gmail", frontendCallbackURL)
	if err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "Failed to get auth URL: "+err.Error())
		return
	}

	response := map[string]string{
		"auth_url": authURL,
		"state":    state,
	}

	h.respondWithSuccess(c, response, "Gmail OAuth2 authorization URL generated")
}

// InitOutlookOAuth 初始化Outlook OAuth2认证
func (h *Handler) InitOutlookOAuth(c *gin.Context) {
	// OAuth初始化不需要用户认证，因为这是OAuth流程的开始
	// 用户可能还没有登录系统，但需要通过OAuth来添加邮箱账户

	// 检查外部OAuth服务器是否启用
	if !h.config.OAuth.ExternalServer.Enabled {
		h.respondWithError(c, http.StatusServiceUnavailable, "External OAuth server is disabled")
		return
	}

	// 生成简单的state参数（不存储到数据库，避免外键约束问题）
	state := fmt.Sprintf("outlook_%d_%s", time.Now().Unix(), generateRandomString(16))

	// 获取前端传递的回调URL，如果没有则使用默认值
	frontendCallbackURL := c.Query("callback_url")
	if frontendCallbackURL == "" {
		frontendCallbackURL = "http://localhost:3000/oauth/callback"
	}

	// 获取外部OAuth服务器的授权URL，传递前端回调地址
	externalClient := external_oauth.NewClient(h.config.OAuth.ExternalServer.BaseURL)
	authURL, err := externalClient.GetAuthURLWithRedirect("outlook", frontendCallbackURL)
	if err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "Failed to get auth URL: "+err.Error())
		return
	}

	response := map[string]string{
		"auth_url": authURL,
		"state":    state,
	}

	h.respondWithSuccess(c, response, "Outlook OAuth2 authorization URL generated")
}

// HandleOAuth2Callback 处理OAuth2回调
func (h *Handler) HandleOAuth2Callback(c *gin.Context) {
	// 获取提供商参数
	provider := c.Param("provider")
	if provider == "" {
		h.respondWithError(c, http.StatusBadRequest, "Provider parameter is required")
		return
	}

	// 验证支持的提供商
	if provider != "gmail" && provider != "outlook" {
		h.respondWithError(c, http.StatusBadRequest, "Unsupported provider: "+provider)
		return
	}

	// 获取URL查询参数
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")
	errorDescription := c.Query("error_description")

	// 检查是否有错误
	if errorParam != "" {
		errorMsg := errorParam
		if errorDescription != "" {
			errorMsg = errorDescription
		}
		h.respondWithError(c, http.StatusBadRequest, "OAuth2 error: "+errorMsg)
		return
	}

	// 检查必需参数
	if code == "" || state == "" {
		h.respondWithError(c, http.StatusBadRequest, "Missing code or state parameter")
		return
	}

	// 检查外部OAuth服务器是否启用
	if !h.config.OAuth.ExternalServer.Enabled {
		h.respondWithError(c, http.StatusServiceUnavailable, "External OAuth server is disabled")
		return
	}

	// 使用外部OAuth服务器交换token
	externalClient := external_oauth.NewClient(h.config.OAuth.ExternalServer.BaseURL)
	tokenResponse, err := externalClient.ExchangeToken(c.Request.Context(), provider, code, state)
	if err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "Failed to exchange token: "+err.Error())
		return
	}

	// 返回token信息给前端
	h.respondWithSuccess(c, gin.H{
		"access_token":  tokenResponse.AccessToken,
		"refresh_token": tokenResponse.RefreshToken,
		"token_type":    tokenResponse.TokenType,
		"expires_in":    tokenResponse.ExpiresIn,
		"scope":         tokenResponse.Scope,
		"client_id":     tokenResponse.ClientID,
	}, "OAuth2 callback processed successfully")
}

// CreateOAuth2AccountRequest 创建OAuth2邮件账户请求
type CreateOAuth2AccountRequest struct {
	Name         string `json:"name" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	Provider     string `json:"provider" binding:"required,oneof=gmail outlook"`
	AccessToken  string `json:"access_token" binding:"required"`
	RefreshToken string `json:"refresh_token" binding:"required"` // 必需，用于token刷新验证
	ExpiresAt    int64  `json:"expires_at" binding:"required"`
	Scope        string `json:"scope"`
	ClientID     string `json:"client_id" binding:"required"` // OAuth2客户端ID，必需用于后续token刷新
	GroupID      *uint  `json:"group_id"`
}

// CreateManualOAuth2AccountRequest 手动创建OAuth2邮件账户请求
type CreateManualOAuth2AccountRequest struct {
	Name         string `json:"name" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	Provider     string `json:"provider" binding:"required,oneof=gmail outlook"`
	ClientID     string `json:"client_id" binding:"required"`
	ClientSecret string `json:"client_secret"` // 可选，某些情况下不需要
	RefreshToken string `json:"refresh_token" binding:"required"`
	Scope        string `json:"scope"`
	// 可选的自定义端点配置
	AuthURL  string `json:"auth_url,omitempty"`
	TokenURL string `json:"token_url,omitempty"`
	GroupID  *uint  `json:"group_id"`
}

// CreateOAuth2Account 使用OAuth2 token创建邮件账户
func (h *Handler) CreateOAuth2Account(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	var req CreateOAuth2AccountRequest
	if !h.bindJSON(c, &req) {
		return
	}

	// 验证必需的ClientID
	if req.ClientID == "" {
		h.respondWithError(c, http.StatusBadRequest, "Client ID is required for OAuth2 account creation")
		return
	}

	// 创建临时OAuth2客户端来验证和刷新token - 与手动添加流程保持一致
	ctx := c.Request.Context()
	var newToken *providers.OAuth2Token
	var err error

	if req.Provider == "outlook" {
		// 使用重写的OutlookOAuth2Client，严格按照Python代码逻辑
		outlookClient := providers.NewOutlookOAuth2Client(req.ClientID, "", "")
		newToken, err = outlookClient.RefreshToken(ctx, req.RefreshToken)
	} else if req.Provider == "gmail" {
		// Gmail 刷新令牌必须携带 client_secret（Google Web 应用要求），否则会返回 invalid_client。
		// 如果后端未配置 client_secret，则退化为直接信任外部 OAuth 返回的令牌，避免误杀流程。
		gmailClientSecret := h.config.OAuth.Gmail.ClientSecret
		if gmailClientSecret == "" {
			if req.AccessToken == "" {
				h.respondWithError(c, http.StatusServiceUnavailable, "Gmail OAuth2 client_secret 未配置，且缺少访问令牌，无法验证")
				return
			}
			log.Printf("Gmail client_secret 未配置，跳过刷新，直接使用外部 OAuth 返回的令牌")
			expiry := time.UnixMilli(req.ExpiresAt)
			if expiry.IsZero() {
				expiry = time.Now().Add(3600 * time.Second)
			}
			newToken = &providers.OAuth2Token{
				AccessToken:  req.AccessToken,
				RefreshToken: req.RefreshToken,
				TokenType:    "Bearer",
				Expiry:       expiry,
			}
		} else {
			// Gmail使用标准客户端
			oauth2Client := providers.NewStandardOAuth2Client(
				req.ClientID,
				gmailClientSecret,
				"https://accounts.google.com/o/oauth2/auth",
				"https://oauth2.googleapis.com/token",
				"", // redirect URL不需要，因为我们直接使用refresh token
				[]string{"https://www.googleapis.com/auth/gmail.readonly", "https://www.googleapis.com/auth/gmail.send"},
			)
			newToken, err = oauth2Client.RefreshToken(ctx, req.RefreshToken)
		}
	} else {
		h.respondWithError(c, http.StatusBadRequest, "Unsupported provider: "+req.Provider)
		return
	}

	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, fmt.Sprintf("Failed to validate and refresh token: %v", err))
		return
	}

	// 使用刷新后的token数据，确保token有效性
	tokenData := &models.OAuth2TokenData{
		AccessToken:  newToken.AccessToken,
		RefreshToken: newToken.RefreshToken, // 使用新的refresh token
		TokenType:    newToken.TokenType,
		Expiry:       newToken.Expiry,
		Scope:        req.Scope,
		ClientID:     req.ClientID, // 保存ClientID用于后续token刷新
	}

	// 检查是否已存在相同的邮箱账户
	var existingAccount models.EmailAccount
	result := h.db.Where("user_id = ? AND email = ? AND provider = ?", userID, req.Email, req.Provider).First(&existingAccount)
	if result.Error == nil {
		h.respondWithError(c, http.StatusConflict, "该邮箱账户已存在")
		return
	}
	// 注意：gorm.ErrRecordNotFound 是正常情况，表示账户不存在，可以继续创建

	// 获取提供商配置
	providerConfig := h.providerFactory.GetProviderConfig(req.Provider)
	if providerConfig == nil {
		h.respondWithError(c, http.StatusBadRequest, "Unknown provider: "+req.Provider)
		return
	}

	// 创建邮件账户
	account := &models.EmailAccount{
		UserID:       userID,
		Name:         req.Name,
		Email:        req.Email,
		Provider:     req.Provider,
		AuthMethod:   "oauth2",
		Username:     req.Email,
		IMAPHost:     providerConfig.IMAPHost,
		IMAPPort:     providerConfig.IMAPPort,
		IMAPSecurity: providerConfig.IMAPSecurity,
		SMTPHost:     providerConfig.SMTPHost,
		SMTPPort:     providerConfig.SMTPPort,
		SMTPSecurity: providerConfig.SMTPSecurity,
		IsActive:     true,
		SyncStatus:   "pending",
	}

	// 设置OAuth2 token
	if err := account.SetOAuth2Token(tokenData); err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "Failed to set OAuth2 token: "+err.Error())
		return
	}

	// 保存到数据库（使用事务确保原子性）
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		return tx.Create(account).Error
	}); err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "Failed to create email account: "+err.Error())
		return
	}

	if err := h.emailService.MoveAccountToGroup(ctx, userID, account.ID, req.GroupID); err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to apply group: "+err.Error())
		return
	}

	// 异步测试连接和同步，避免阻塞数据库事务
	go func(accountID uint) {
		// 测试连接
		if err := h.emailService.TestEmailAccount(context.Background(), userID, accountID); err != nil {
			// 如果测试失败，标记为错误状态但不删除账户
			h.db.Model(&models.EmailAccount{}).Where("id = ?", accountID).Updates(map[string]interface{}{
				"sync_status":   "error",
				"error_message": err.Error(),
			})
		} else {
			// 测试成功，开始同步文件夹
			if err := h.syncService.SyncEmails(context.Background(), accountID); err != nil {
				// 记录错误但不影响账户创建
				h.db.Model(&models.EmailAccount{}).Where("id = ?", accountID).Updates(map[string]interface{}{
					"sync_status":   "error",
					"error_message": fmt.Sprintf("Failed to sync: %v", err),
				})
			}
		}
	}(account.ID)

	h.respondWithCreated(c, account, "OAuth2 email account created successfully")
}

// CreateManualOAuth2Account 使用手动配置创建OAuth2邮件账户
func (h *Handler) CreateManualOAuth2Account(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	var req CreateManualOAuth2AccountRequest
	if !h.bindJSON(c, &req) {
		return
	}

	// 验证提供商
	if req.Provider != "outlook" && req.Provider != "gmail" {
		h.respondWithError(c, http.StatusBadRequest, "Only outlook and gmail providers are supported for manual configuration")
		return
	}

	// 设置默认的OAuth2端点
	authURL := req.AuthURL
	tokenURL := req.TokenURL
	scopes := []string{}

	switch req.Provider {
	case "outlook":
		if authURL == "" {
			authURL = "https://login.microsoftonline.com/common/oauth2/v2.0/authorize"
		}
		if tokenURL == "" {
			tokenURL = "https://login.microsoftonline.com/common/oauth2/v2.0/token"
		}
		if req.Scope == "" {
			scopes = []string{
				"https://outlook.office.com/IMAP.AccessAsUser.All",
				"https://outlook.office.com/SMTP.Send",
				"offline_access",
			}
		}
	case "gmail":
		if authURL == "" {
			authURL = "https://accounts.google.com/o/oauth2/auth"
		}
		if tokenURL == "" {
			tokenURL = "https://oauth2.googleapis.com/token"
		}
		if req.Scope == "" {
			scopes = []string{
				"https://www.googleapis.com/auth/gmail.readonly",
				"https://www.googleapis.com/auth/gmail.send",
			}
		}
	}

	// 如果用户提供了自定义scope，解析它
	if req.Scope != "" {
		scopes = strings.Split(req.Scope, " ")
	}

	// 创建临时OAuth2客户端来验证refresh token - 使用重写的Outlook客户端
	ctx := c.Request.Context()
	var newToken *providers.OAuth2Token
	var err error

	if req.Provider == "outlook" {
		// 使用重写的OutlookOAuth2Client，严格按照Python代码逻辑
		outlookClient := providers.NewOutlookOAuth2Client(req.ClientID, "", "")
		newToken, err = outlookClient.RefreshToken(ctx, req.RefreshToken)
	} else {
		// Gmail使用标准客户端
		oauth2Client := providers.NewStandardOAuth2Client(
			req.ClientID,
			req.ClientSecret,
			authURL,
			tokenURL,
			"", // redirect URL不需要，因为我们直接使用refresh token
			scopes,
		)
		newToken, err = oauth2Client.RefreshToken(ctx, req.RefreshToken)
	}

	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, fmt.Sprintf("Failed to validate refresh token: %v", err))
		return
	}

	// 创建OAuth2 token数据
	tokenData := &models.OAuth2TokenData{
		AccessToken:  newToken.AccessToken,
		RefreshToken: newToken.RefreshToken, // 使用新的refresh token
		TokenType:    newToken.TokenType,
		Expiry:       newToken.Expiry,
		Scope:        strings.Join(scopes, " "),
		ClientID:     req.ClientID, // 存储client_id用于后续token刷新
	}

	// 根据提供商设置服务器配置
	var imapHost string
	var imapPort int
	var smtpHost string
	var smtpPort int

	switch req.Provider {
	case "outlook":
		imapHost = "outlook.office365.com" // 严格按照Python代码使用此服务器
		imapPort = 993
		smtpHost = "outlook.office365.com" // SMTP也使用同一服务器
		smtpPort = 587
	case "gmail":
		imapHost = "imap.gmail.com"
		imapPort = 993
		smtpHost = "smtp.gmail.com"
		smtpPort = 587
	}

	// 创建邮件账户
	account := &models.EmailAccount{
		UserID:       userID,
		Name:         req.Name,
		Email:        req.Email,
		Provider:     req.Provider,
		AuthMethod:   "oauth2",
		IMAPHost:     imapHost,
		IMAPPort:     imapPort,
		IMAPSecurity: "SSL", // Outlook使用SSL
		SMTPHost:     smtpHost,
		SMTPPort:     smtpPort,
		SMTPSecurity: "STARTTLS", // SMTP使用STARTTLS
		IsActive:     true,
		SyncStatus:   "pending",
	}

	// 设置OAuth2 token
	if err := account.SetOAuth2Token(tokenData); err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "Failed to set OAuth2 token: "+err.Error())
		return
	}

	// 保存到数据库（使用事务确保原子性）
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		return tx.Create(account).Error
	}); err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "Failed to create email account: "+err.Error())
		return
	}

	if err := h.emailService.MoveAccountToGroup(c.Request.Context(), userID, account.ID, req.GroupID); err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to apply group: "+err.Error())
		return
	}

	// 清除敏感信息
	account.OAuth2Token = ""

	// 异步测试连接和同步
	go func(accountID uint) {
		// 重新从数据库加载账户信息（包含OAuth2Token）
		var fullAccount models.EmailAccount
		if err := h.db.First(&fullAccount, accountID).Error; err != nil {
			log.Printf("Failed to reload account for testing: %v", err)
			return
		}

		// 测试连接
		provider, err := h.providerFactory.CreateProvider(fullAccount.Provider)
		if err != nil {
			h.db.Model(&models.EmailAccount{}).Where("id = ?", accountID).Updates(map[string]interface{}{
				"sync_status":   "error",
				"error_message": fmt.Sprintf("Failed to create provider: %v", err),
			})
			return
		}

		if err := provider.Connect(context.Background(), &fullAccount); err != nil {
			h.db.Model(&models.EmailAccount{}).Where("id = ?", fullAccount.ID).Updates(map[string]interface{}{
				"sync_status":   "error",
				"error_message": err.Error(),
			})
		} else {
			// 测试成功，开始同步文件夹
			if err := h.syncService.SyncEmails(context.Background(), fullAccount.ID); err != nil {
				// 记录错误但不影响账户创建
				h.db.Model(&models.EmailAccount{}).Where("id = ?", fullAccount.ID).Updates(map[string]interface{}{
					"sync_status":   "error",
					"error_message": fmt.Sprintf("Failed to sync: %v", err),
				})
			}
		}
	}(account.ID)

	h.respondWithCreated(c, account, "Manual OAuth2 email account created successfully")
}

// generateRandomString 生成指定长度的随机字符串
func generateRandomString(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		// 如果随机数生成失败，使用时间戳作为后备
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}
