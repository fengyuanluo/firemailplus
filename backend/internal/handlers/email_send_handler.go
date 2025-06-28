package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"firemail/internal/middleware"
	"firemail/internal/models"
	"firemail/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// EmailSendHandler 邮件发送处理器
type EmailSendHandler struct {
	emailComposer   services.EmailComposer
	emailSender     services.EmailSender
	draftService    services.DraftService
	templateService services.EmailTemplateService
	db              *gorm.DB
}

// NewEmailSendHandler 创建邮件发送处理器
func NewEmailSendHandler(emailComposer services.EmailComposer, emailSender services.EmailSender, draftService services.DraftService, templateService services.EmailTemplateService, db *gorm.DB) *EmailSendHandler {
	return &EmailSendHandler{
		emailComposer:   emailComposer,
		emailSender:     emailSender,
		draftService:    draftService,
		templateService: templateService,
		db:              db,
	}
}

// RegisterRoutes 注册路由
func (h *EmailSendHandler) RegisterRoutes(router *gin.RouterGroup) {
	emails := router.Group("/emails")
	emails.Use(middleware.AuthRequired())
	{
		// 发送邮件
		emails.POST("/send", h.SendEmail)
		
		// 批量发送邮件
		emails.POST("/send/bulk", h.SendBulkEmails)
		
		// 获取发送状态
		emails.GET("/send/:send_id/status", h.GetSendStatus)
		
		// 重新发送邮件
		emails.POST("/send/:send_id/resend", h.ResendEmail)
		
		// 草稿相关
		emails.POST("/draft", h.SaveDraft)
		emails.PUT("/draft/:id", h.UpdateDraft)
		emails.GET("/draft/:id", h.GetDraft)
		emails.GET("/drafts", h.ListDrafts)
		emails.DELETE("/draft/:id", h.DeleteDraft)
		
		// 模板相关
		emails.POST("/template", h.CreateTemplate)
		emails.PUT("/template/:id", h.UpdateTemplate)
		emails.GET("/template/:id", h.GetTemplate)
		emails.GET("/templates", h.ListTemplates)
		emails.DELETE("/template/:id", h.DeleteTemplate)
	}
}

// SendEmailRequest 发送邮件请求
type SendEmailRequest struct {
	services.ComposeEmailRequest
	AccountID uint `json:"account_id" binding:"required"`
}

// SendEmail 发送邮件
func (h *EmailSendHandler) SendEmail(c *gin.Context) {
	userID := middleware.GetUserID(c)
	
	var req SendEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	// 验证账户权限
	if err := h.validateAccountAccess(c, req.AccountID, userID); err != nil {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Access denied",
			Message: "You don't have access to this email account",
		})
		return
	}

	// 检查是否为定时发送
	if req.ScheduledTime != nil && *req.ScheduledTime != "" {
		// 定时发送：保存到发送队列
		err := h.scheduleEmail(c.Request.Context(), userID, &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Failed to schedule email",
				Message: err.Error(),
			})
			return
		}

		c.JSON(http.StatusAccepted, SuccessResponse{
			Success: true,
			Message: "Email scheduled successfully",
			Data:    map[string]interface{}{
				"scheduled_time": *req.ScheduledTime,
			},
		})
		return
	}

	// 立即发送
	composedEmail, err := h.emailComposer.ComposeEmail(c.Request.Context(), &req.ComposeEmailRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Failed to compose email",
			Message: err.Error(),
		})
		return
	}

	// 发送邮件
	result, err := h.emailSender.SendEmail(c.Request.Context(), composedEmail, req.AccountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to send email",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, SuccessResponse{
		Success: true,
		Message: "Email queued for sending",
		Data:    result,
	})
}

// SendBulkEmailsRequest 批量发送邮件请求
type SendBulkEmailsRequest struct {
	Emails    []services.ComposeEmailRequest `json:"emails" binding:"required,min=1"`
	AccountID uint                           `json:"account_id" binding:"required"`
}

// SendBulkEmails 批量发送邮件
func (h *EmailSendHandler) SendBulkEmails(c *gin.Context) {
	userID := middleware.GetUserID(c)
	
	var req SendBulkEmailsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	// 验证账户权限
	if err := h.validateAccountAccess(c, req.AccountID, userID); err != nil {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Access denied",
			Message: "You don't have access to this email account",
		})
		return
	}

	// 组装所有邮件
	var composedEmails []*services.ComposedEmail
	for i, emailReq := range req.Emails {
		composedEmail, err := h.emailComposer.ComposeEmail(c.Request.Context(), &emailReq)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Failed to compose email",
				Message: fmt.Sprintf("Error in email %d: %v", i+1, err),
			})
			return
		}
		composedEmails = append(composedEmails, composedEmail)
	}

	// 批量发送邮件
	results, err := h.emailSender.SendBulkEmails(c.Request.Context(), composedEmails, req.AccountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to send bulk emails",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, SuccessResponse{
		Success: true,
		Message: "Emails queued for sending",
		Data:    results,
	})
}

// GetSendStatus 获取发送状态
func (h *EmailSendHandler) GetSendStatus(c *gin.Context) {
	sendID := c.Param("send_id")
	
	status, err := h.emailSender.GetSendStatus(c.Request.Context(), sendID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Send status not found",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    status,
	})
}

// ResendEmail 重新发送邮件
func (h *EmailSendHandler) ResendEmail(c *gin.Context) {
	sendID := c.Param("send_id")
	
	result, err := h.emailSender.ResendEmail(c.Request.Context(), sendID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to resend email",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, SuccessResponse{
		Success: true,
		Message: "Email queued for resending",
		Data:    result,
	})
}

// SaveDraftRequest 保存草稿请求
type SaveDraftRequest struct {
	services.ComposeEmailRequest
	AccountID uint `json:"account_id" binding:"required"`
}

// SaveDraft 保存草稿
func (h *EmailSendHandler) SaveDraft(c *gin.Context) {
	userID := middleware.GetUserID(c)
	
	var req SaveDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	// 验证账户权限
	if err := h.validateAccountAccess(c, req.AccountID, userID); err != nil {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Access denied",
			Message: "You don't have access to this email account",
		})
		return
	}

	// 转换地址类型
	var toAddresses []models.EmailAddress
	for _, addr := range req.To {
		if addr != nil {
			toAddresses = append(toAddresses, *addr)
		}
	}

	var ccAddresses []models.EmailAddress
	for _, addr := range req.CC {
		if addr != nil {
			ccAddresses = append(ccAddresses, *addr)
		}
	}

	var bccAddresses []models.EmailAddress
	for _, addr := range req.BCC {
		if addr != nil {
			bccAddresses = append(bccAddresses, *addr)
		}
	}

	// 转换附件ID（从ComposeEmailRequest中获取）
	var attachmentIDs []uint
	if len(req.AttachmentIDs) > 0 {
		attachmentIDs = req.AttachmentIDs
	}

	// 转换为DraftService请求格式
	draftReq := &services.CreateDraftRequest{
		AccountID:     req.AccountID,
		Subject:       req.Subject,
		To:            toAddresses,
		CC:            ccAddresses,
		BCC:           bccAddresses,
		TextBody:      req.TextBody,
		HTMLBody:      req.HTMLBody,
		AttachmentIDs: attachmentIDs,
		Priority:      req.Priority,
	}

	// 创建草稿
	draft, err := h.draftService.CreateDraft(c.Request.Context(), userID, draftReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to save draft",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Message: "Draft saved successfully",
		Data:    draft,
	})
}

// UpdateDraft 更新草稿
func (h *EmailSendHandler) UpdateDraft(c *gin.Context) {
	userID := middleware.GetUserID(c)

	draftID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid draft ID",
			Message: err.Error(),
		})
		return
	}

	var req services.UpdateDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	// 更新草稿
	draft, err := h.draftService.UpdateDraft(c.Request.Context(), userID, uint(draftID), &req)
	if err != nil {
		if err.Error() == "draft not found or access denied" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Draft not found",
				Message: "Draft not found or access denied",
			})
		} else {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Failed to update draft",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Draft updated successfully",
		Data:    draft,
	})
}

// GetDraft 获取草稿
func (h *EmailSendHandler) GetDraft(c *gin.Context) {
	userID := middleware.GetUserID(c)

	draftID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid draft ID",
			Message: err.Error(),
		})
		return
	}

	// 获取草稿
	draft, err := h.draftService.GetDraft(c.Request.Context(), userID, uint(draftID))
	if err != nil {
		if err.Error() == "draft not found or access denied" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Draft not found",
				Message: "Draft not found or access denied",
			})
		} else {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Failed to get draft",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    draft,
	})
}

// ListDrafts 列出草稿
func (h *EmailSendHandler) ListDrafts(c *gin.Context) {
	userID := middleware.GetUserID(c)

	// 解析查询参数
	var req services.ListDraftsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid query parameters",
			Message: err.Error(),
		})
		return
	}

	// 设置默认值
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	// 获取草稿列表
	response, err := h.draftService.ListDrafts(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to list drafts",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
	})
}

// scheduleEmail 安排定时发送邮件
func (h *EmailSendHandler) scheduleEmail(ctx context.Context, userID uint, req *SendEmailRequest) error {
	// 解析定时发送时间
	scheduledTime, err := time.Parse(time.RFC3339, *req.ScheduledTime)
	if err != nil {
		return fmt.Errorf("invalid scheduled time format: %w", err)
	}

	// 检查时间是否在未来
	if scheduledTime.Before(time.Now()) {
		return fmt.Errorf("scheduled time must be in the future")
	}

	// 序列化邮件数据
	emailData, err := json.Marshal(req.ComposeEmailRequest)
	if err != nil {
		return fmt.Errorf("failed to serialize email data: %w", err)
	}

	// 创建发送队列记录
	sendQueue := &models.SendQueue{
		SendID:      fmt.Sprintf("scheduled_%d_%d", time.Now().Unix(), userID),
		UserID:      userID,
		AccountID:   req.AccountID,
		EmailData:   string(emailData),
		ScheduledAt: &scheduledTime,
		Priority:    5, // 默认优先级
		Status:      "scheduled",
		MaxAttempts: 3,
	}

	// 保存到数据库
	return h.db.WithContext(ctx).Create(sendQueue).Error
}

// DeleteDraft 删除草稿
func (h *EmailSendHandler) DeleteDraft(c *gin.Context) {
	userID := middleware.GetUserID(c)

	draftID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid draft ID",
			Message: err.Error(),
		})
		return
	}

	// 删除草稿
	err = h.draftService.DeleteDraft(c.Request.Context(), userID, uint(draftID))
	if err != nil {
		if err.Error() == "draft not found or access denied" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Draft not found",
				Message: "Draft not found or access denied",
			})
		} else {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Failed to delete draft",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Draft deleted successfully",
	})
}

// CreateTemplate 创建模板
func (h *EmailSendHandler) CreateTemplate(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req services.CreateEmailTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	template, err := h.templateService.CreateTemplate(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create template",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Message: "Template created successfully",
		Data:    template,
	})
}

// UpdateTemplate 更新模板
func (h *EmailSendHandler) UpdateTemplate(c *gin.Context) {
	userID := middleware.GetUserID(c)

	templateID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid template ID",
			Message: err.Error(),
		})
		return
	}

	var req services.UpdateEmailTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	template, err := h.templateService.UpdateTemplate(c.Request.Context(), userID, uint(templateID), &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Template not found",
				Message: err.Error(),
			})
		} else if strings.Contains(err.Error(), "permission denied") {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "Permission denied",
				Message: err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Failed to update template",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Template updated successfully",
		Data:    template,
	})
}

// GetTemplate 获取模板
func (h *EmailSendHandler) GetTemplate(c *gin.Context) {
	userID := middleware.GetUserID(c)

	templateID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid template ID",
			Message: err.Error(),
		})
		return
	}

	template, err := h.templateService.GetTemplate(c.Request.Context(), userID, uint(templateID))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Template not found",
				Message: err.Error(),
			})
		} else if strings.Contains(err.Error(), "permission denied") {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "Permission denied",
				Message: err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Failed to get template",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    template,
	})
}

// ListTemplates 列出模板
func (h *EmailSendHandler) ListTemplates(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req services.ListEmailTemplatesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid query parameters",
			Message: err.Error(),
		})
		return
	}

	// 设置默认值
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	response, err := h.templateService.ListTemplates(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to list templates",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
	})
}

// DeleteTemplate 删除模板
func (h *EmailSendHandler) DeleteTemplate(c *gin.Context) {
	userID := middleware.GetUserID(c)

	templateID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid template ID",
			Message: err.Error(),
		})
		return
	}

	err = h.templateService.DeleteTemplate(c.Request.Context(), userID, uint(templateID))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Template not found",
				Message: err.Error(),
			})
		} else if strings.Contains(err.Error(), "permission denied") {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "Permission denied",
				Message: err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Failed to delete template",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Template deleted successfully",
	})
}

// validateAccountAccess 验证账户访问权限
func (h *EmailSendHandler) validateAccountAccess(c *gin.Context, accountID, userID uint) error {
	var count int64
	err := h.db.WithContext(c.Request.Context()).
		Model(&models.EmailAccount{}).
		Where("id = ? AND user_id = ?", accountID, userID).
		Count(&count).Error
	
	if err != nil {
		return err
	}
	
	if count == 0 {
		return fmt.Errorf("account not found or access denied")
	}
	
	return nil
}
