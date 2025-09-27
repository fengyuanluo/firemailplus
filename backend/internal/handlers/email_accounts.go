package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"firemail/internal/services"

	"github.com/gin-gonic/gin"
)

// GetEmailAccounts 获取邮件账户列表
func (h *Handler) GetEmailAccounts(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	accounts, err := h.emailService.GetEmailAccounts(c.Request.Context(), userID)
	if err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "Failed to get email accounts")
		return
	}

	h.respondWithSuccess(c, accounts)
}

// CreateEmailAccount 创建邮件账户
func (h *Handler) CreateEmailAccount(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	var req services.CreateEmailAccountRequest
	if !h.bindJSON(c, &req) {
		return
	}

	account, err := h.emailService.CreateEmailAccount(c.Request.Context(), userID, &req)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to create email account: "+err.Error())
		return
	}

	h.respondWithCreated(c, account, "Email account created successfully")
}

// GetEmailAccount 获取指定邮件账户
func (h *Handler) GetEmailAccount(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	accountID, exists := h.parseUintParam(c, "id")
	if !exists {
		return
	}

	account, err := h.emailService.GetEmailAccount(c.Request.Context(), userID, accountID)
	if err != nil {
		h.respondWithError(c, http.StatusNotFound, "Email account not found")
		return
	}

	h.respondWithSuccess(c, account)
}

// UpdateEmailAccount 更新邮件账户
func (h *Handler) UpdateEmailAccount(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	accountID, exists := h.parseUintParam(c, "id")
	if !exists {
		return
	}

	var req services.UpdateEmailAccountRequest
	if !h.bindJSON(c, &req) {
		return
	}

	account, err := h.emailService.UpdateEmailAccount(c.Request.Context(), userID, accountID, &req)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to update email account: "+err.Error())
		return
	}

	h.respondWithSuccess(c, account, "Email account updated successfully")
}

// DeleteEmailAccount 删除邮件账户
func (h *Handler) DeleteEmailAccount(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	accountID, exists := h.parseUintParam(c, "id")
	if !exists {
		return
	}

	err := h.emailService.DeleteEmailAccount(c.Request.Context(), userID, accountID)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to delete email account: "+err.Error())
		return
	}

	h.respondWithSuccess(c, nil, "Email account deleted successfully")
}

// TestEmailAccount 测试邮件账户连接
func (h *Handler) TestEmailAccount(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	accountID, exists := h.parseUintParam(c, "id")
	if !exists {
		return
	}

	err := h.emailService.TestEmailAccount(c.Request.Context(), userID, accountID)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Connection test failed: "+err.Error())
		return
	}

	h.respondWithSuccess(c, nil, "Connection test successful")
}

// SyncEmailAccount 同步邮件账户
func (h *Handler) SyncEmailAccount(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	accountID, exists := h.parseUintParam(c, "id")
	if !exists {
		return
	}

	// 验证账户属于当前用户
	_, err := h.emailService.GetEmailAccount(c.Request.Context(), userID, accountID)
	if err != nil {
		h.respondWithError(c, http.StatusNotFound, "Email account not found")
		return
	}

	// 启动异步同步
	go func() {
		if err := h.syncService.SyncEmails(c.Request.Context(), accountID); err != nil {
			// 记录错误，但不影响响应
			// 可以通过WebSocket或其他方式通知前端
		}
	}()

	h.respondWithSuccess(c, nil, "Email sync started")
}

// GetProviders 获取支持的邮件提供商列表
func (h *Handler) GetProviders(c *gin.Context) {
	providers := h.providerFactory.GetAvailableProviders()

	var providerList []map[string]interface{}
	for _, providerName := range providers {
		config := h.providerFactory.GetProviderConfig(providerName)
		if config != nil {
			providerInfo := map[string]interface{}{
				"name":         config.Name,
				"display_name": config.DisplayName,
				"auth_methods": config.AuthMethods,
				"domains":      config.Domains,
			}

			// 添加服务器配置（如果不为空）
			if config.IMAPHost != "" {
				providerInfo["imap"] = map[string]interface{}{
					"host":     config.IMAPHost,
					"port":     config.IMAPPort,
					"security": config.IMAPSecurity,
				}
			}

			if config.SMTPHost != "" {
				providerInfo["smtp"] = map[string]interface{}{
					"host":     config.SMTPHost,
					"port":     config.SMTPPort,
					"security": config.SMTPSecurity,
				}
			}

			providerList = append(providerList, providerInfo)
		}
	}

	h.respondWithSuccess(c, providerList)
}

// DetectProvider 根据邮箱地址检测提供商
func (h *Handler) DetectProvider(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		h.respondWithError(c, http.StatusBadRequest, "Email parameter is required")
		return
	}

	config := h.providerFactory.DetectProvider(email)
	if config == nil {
		h.respondWithError(c, http.StatusNotFound, "No provider found for this email domain")
		return
	}

	providerInfo := map[string]interface{}{
		"name":         config.Name,
		"display_name": config.DisplayName,
		"auth_methods": config.AuthMethods,
		"imap": map[string]interface{}{
			"host":     config.IMAPHost,
			"port":     config.IMAPPort,
			"security": config.IMAPSecurity,
		},
		"smtp": map[string]interface{}{
			"host":     config.SMTPHost,
			"port":     config.SMTPPort,
			"security": config.SMTPSecurity,
		},
	}

	h.respondWithSuccess(c, providerInfo)
}

// CreateCustomEmailAccount 创建自定义邮件账户
func (h *Handler) CreateCustomEmailAccount(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	var req services.CreateEmailAccountRequest
	if !h.bindJSON(c, &req) {
		return
	}

	// 强制设置为自定义提供商
	req.Provider = "custom"

	// 验证自定义邮箱的必要配置
	if err := h.validateCustomAccountRequest(&req); err != nil {
		h.respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	account, err := h.emailService.CreateEmailAccount(c.Request.Context(), userID, &req)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to create custom email account: "+err.Error())
		return
	}

	h.respondWithCreated(c, account, "Custom email account created successfully")
}

// validateCustomAccountRequest 验证自定义邮箱账户请求
func (h *Handler) validateCustomAccountRequest(req *services.CreateEmailAccountRequest) error {
	// 检查是否至少配置了IMAP或SMTP
	hasIMAP := req.IMAPHost != "" && req.IMAPPort > 0
	hasSMTP := req.SMTPHost != "" && req.SMTPPort > 0

	if !hasIMAP && !hasSMTP {
		return fmt.Errorf("at least one of IMAP or SMTP configuration is required for custom email accounts")
	}

	// 验证认证方式
	if req.AuthMethod != "password" && req.AuthMethod != "oauth2" {
		return fmt.Errorf("custom email accounts support 'password' and 'oauth2' authentication methods")
	}

	// 验证认证信息
	switch req.AuthMethod {
	case "password":
		if req.Username == "" || req.Password == "" {
			return fmt.Errorf("username and password are required for password authentication")
		}
	case "oauth2":
		return fmt.Errorf("OAuth2 authentication for custom accounts should use the manual OAuth2 configuration endpoint")
	}

	// 验证端口范围
	if hasIMAP && (req.IMAPPort < 1 || req.IMAPPort > 65535) {
		return fmt.Errorf("IMAP port must be between 1 and 65535")
	}

	if hasSMTP && (req.SMTPPort < 1 || req.SMTPPort > 65535) {
		return fmt.Errorf("SMTP port must be between 1 and 65535")
	}

	// 验证安全设置
	validSecurityOptions := []string{"SSL", "TLS", "STARTTLS", "NONE"}

	if hasIMAP {
		if !h.isValidSecurityOption(req.IMAPSecurity, validSecurityOptions) {
			return fmt.Errorf("invalid IMAP security option. Valid options: SSL, TLS, STARTTLS, NONE")
		}
	}

	if hasSMTP {
		if !h.isValidSecurityOption(req.SMTPSecurity, validSecurityOptions) {
			return fmt.Errorf("invalid SMTP security option. Valid options: SSL, TLS, STARTTLS, NONE")
		}
	}

	return nil
}

// GetAccountGroups 获取邮箱分组列表
func (h *Handler) GetAccountGroups(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	groups, err := h.emailService.GetAccountGroups(c.Request.Context(), userID)
	if err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "Failed to get account groups")
		return
	}

	h.respondWithSuccess(c, groups)
}

// CreateAccountGroup 创建邮箱分组
func (h *Handler) CreateAccountGroup(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	var req services.CreateAccountGroupRequest
	if !h.bindJSON(c, &req) {
		return
	}

	group, err := h.emailService.CreateAccountGroup(c.Request.Context(), userID, &req)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	h.respondWithCreated(c, group, "Account group created successfully")
}

// UpdateAccountGroup 更新邮箱分组
func (h *Handler) UpdateAccountGroup(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	groupID, exists := h.parseUintParam(c, "id")
	if !exists {
		return
	}

	var req services.UpdateAccountGroupRequest
	if !h.bindJSON(c, &req) {
		return
	}

	group, err := h.emailService.UpdateAccountGroup(c.Request.Context(), userID, groupID, &req)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	h.respondWithSuccess(c, group, "Account group updated successfully")
}

// DeleteAccountGroup 删除邮箱分组
func (h *Handler) DeleteAccountGroup(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	groupID, exists := h.parseUintParam(c, "id")
	if !exists {
		return
	}

	if err := h.emailService.DeleteAccountGroup(c.Request.Context(), userID, groupID); err != nil {
		h.respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	h.respondWithSuccess(c, nil, "Account group deleted successfully")
}

// ReorderAccountGroups 调整邮箱分组排序
func (h *Handler) ReorderAccountGroups(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	var req struct {
		Orders []services.AccountGroupOrder `json:"orders" binding:"required"`
	}

	if !h.bindJSON(c, &req) {
		return
	}

	if len(req.Orders) == 0 {
		h.respondWithError(c, http.StatusBadRequest, "orders cannot be empty")
		return
	}

	if err := h.emailService.ReorderAccountGroups(c.Request.Context(), userID, req.Orders); err != nil {
		h.respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	h.respondWithSuccess(c, nil, "Account groups reordered successfully")
}

// MoveAccountsToGroup 批量移动邮箱账户
func (h *Handler) MoveAccountsToGroup(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	var req services.MoveAccountsToGroupRequest
	if !h.bindJSON(c, &req) {
		return
	}

	if err := h.emailService.MoveAccountsToGroup(c.Request.Context(), userID, &req); err != nil {
		h.respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	h.respondWithSuccess(c, nil, "Accounts moved successfully")
}

// ReorderAccounts 批量调整邮箱账户排序
func (h *Handler) ReorderAccounts(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	var req struct {
		Orders []services.AccountOrder `json:"orders" binding:"required"`
	}

	if !h.bindJSON(c, &req) {
		return
	}

	if len(req.Orders) == 0 {
		h.respondWithError(c, http.StatusBadRequest, "orders cannot be empty")
		return
	}

	if err := h.emailService.ReorderAccounts(c.Request.Context(), userID, req.Orders); err != nil {
		h.respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	h.respondWithSuccess(c, nil, "Accounts reordered successfully")
}

// isValidSecurityOption 检查安全选项是否有效
func (h *Handler) isValidSecurityOption(option string, validOptions []string) bool {
	if option == "" {
		return true // 空值将使用默认值
	}

	for _, valid := range validOptions {
		if strings.ToUpper(option) == valid {
			return true
		}
	}
	return false
}
