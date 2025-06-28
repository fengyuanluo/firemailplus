package handlers

import (
	"fmt"
	"net/http"
	"time"

	"firemail/internal/services"

	"github.com/gin-gonic/gin"
)

// GetEmails 获取邮件列表
func (h *Handler) GetEmails(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	// 解析查询参数
	req := &services.GetEmailsRequest{
		AccountID:   h.parseOptionalUintQuery(c, "account_id"),
		FolderID:    h.parseOptionalUintQuery(c, "folder_id"),
		IsRead:      h.parseOptionalBoolQuery(c, "is_read"),
		IsStarred:   h.parseOptionalBoolQuery(c, "is_starred"),
		IsImportant: h.parseOptionalBoolQuery(c, "is_important"),
		Page:        h.parseIntQuery(c, "page", 1),
		PageSize:    h.parseIntQuery(c, "page_size", 20),
		SortBy:      c.DefaultQuery("sort_by", "date"),
		SortOrder:   c.DefaultQuery("sort_order", "desc"),
		SearchQuery: c.Query("search"),
	}

	// 验证分页参数
	req.Page, req.PageSize = h.validatePagination(req.Page, req.PageSize)

	// 验证排序参数
	req.SortBy, req.SortOrder = h.validateSortParams(req.SortBy, req.SortOrder)

	response, err := h.emailService.GetEmails(c.Request.Context(), userID, req)
	if err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "Failed to get emails")
		return
	}

	h.respondWithSuccess(c, response)
}

// GetEmail 获取指定邮件
func (h *Handler) GetEmail(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	emailID, exists := h.parseUintParam(c, "id")
	if !exists {
		return
	}

	email, err := h.emailService.GetEmail(c.Request.Context(), userID, emailID)
	if err != nil {
		h.respondWithError(c, http.StatusNotFound, "Email not found")
		return
	}

	h.respondWithSuccess(c, email)
}

// UpdateEmailRequest 通用邮件更新请求
type UpdateEmailRequest struct {
	IsRead      *bool `json:"is_read,omitempty"`
	IsStarred   *bool `json:"is_starred,omitempty"`
	IsImportant *bool `json:"is_important,omitempty"`
	FolderID    *uint `json:"folder_id,omitempty"`
}

// UpdateEmail 通用邮件更新处理器
func (h *Handler) UpdateEmail(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	emailID, exists := h.parseUintParam(c, "id")
	if !exists {
		return
	}

	var req UpdateEmailRequest
	if !h.bindJSON(c, &req) {
		return
	}

	// 验证至少有一个字段需要更新
	if req.IsRead == nil && req.IsStarred == nil && req.IsImportant == nil && req.FolderID == nil {
		h.respondWithError(c, http.StatusBadRequest, "At least one field must be provided for update")
		return
	}

	// 执行更新操作
	var err error

	// 处理已读状态更新
	if req.IsRead != nil {
		if *req.IsRead {
			err = h.emailService.MarkEmailAsRead(c.Request.Context(), userID, emailID)
		} else {
			err = h.emailService.MarkEmailAsUnread(c.Request.Context(), userID, emailID)
		}
		if err != nil {
			h.respondWithError(c, http.StatusBadRequest, "Failed to update read status: "+err.Error())
			return
		}
	}

	// 处理星标状态更新
	if req.IsStarred != nil {
		// 先获取当前状态
		email, getErr := h.emailService.GetEmail(c.Request.Context(), userID, emailID)
		if getErr != nil {
			h.respondWithError(c, http.StatusNotFound, "Email not found")
			return
		}

		// 只有当目标状态与当前状态不同时才切换
		if email.IsStarred != *req.IsStarred {
			err = h.emailService.ToggleEmailStar(c.Request.Context(), userID, emailID)
			if err != nil {
				h.respondWithError(c, http.StatusBadRequest, "Failed to update star status: "+err.Error())
				return
			}
		}
	}

	// 处理重要状态更新
	if req.IsImportant != nil {
		// 先获取当前状态
		email, getErr := h.emailService.GetEmail(c.Request.Context(), userID, emailID)
		if getErr != nil {
			h.respondWithError(c, http.StatusNotFound, "Email not found")
			return
		}

		// 只有当目标状态与当前状态不同时才切换
		if email.IsImportant != *req.IsImportant {
			err = h.emailService.ToggleEmailImportant(c.Request.Context(), userID, emailID)
			if err != nil {
				h.respondWithError(c, http.StatusBadRequest, "Failed to update important status: "+err.Error())
				return
			}
		}
	}

	// 处理文件夹移动
	if req.FolderID != nil {
		err = h.emailService.MoveEmail(c.Request.Context(), userID, emailID, *req.FolderID)
		if err != nil {
			h.respondWithError(c, http.StatusBadRequest, "Failed to move email: "+err.Error())
			return
		}
	}

	// 获取更新后的邮件信息
	updatedEmail, err := h.emailService.GetEmail(c.Request.Context(), userID, emailID)
	if err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "Failed to get updated email")
		return
	}

	h.respondWithSuccess(c, updatedEmail, "Email updated successfully")
}

// SendEmail 发送邮件
func (h *Handler) SendEmail(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	var req services.SendEmailRequest
	if !h.bindJSON(c, &req) {
		return
	}

	err := h.emailService.SendEmail(c.Request.Context(), userID, &req)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to send email: "+err.Error())
		return
	}

	h.respondWithSuccess(c, nil, "Email sent successfully")
}

// DeleteEmail 删除邮件
func (h *Handler) DeleteEmail(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	emailID, exists := h.parseUintParam(c, "id")
	if !exists {
		return
	}

	err := h.emailService.DeleteEmail(c.Request.Context(), userID, emailID)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to delete email: "+err.Error())
		return
	}

	h.respondWithSuccess(c, nil, "Email deleted successfully")
}

// MarkEmailAsRead 标记邮件为已读
func (h *Handler) MarkEmailAsRead(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	emailID, exists := h.parseUintParam(c, "id")
	if !exists {
		return
	}

	err := h.emailService.MarkEmailAsRead(c.Request.Context(), userID, emailID)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to mark email as read: "+err.Error())
		return
	}

	h.respondWithSuccess(c, nil, "Email marked as read")
}

// MarkEmailAsUnread 标记邮件为未读
func (h *Handler) MarkEmailAsUnread(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	emailID, exists := h.parseUintParam(c, "id")
	if !exists {
		return
	}

	err := h.emailService.MarkEmailAsUnread(c.Request.Context(), userID, emailID)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to mark email as unread: "+err.Error())
		return
	}

	h.respondWithSuccess(c, nil, "Email marked as unread")
}

// ToggleEmailStar 切换邮件星标
func (h *Handler) ToggleEmailStar(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	emailID, exists := h.parseUintParam(c, "id")
	if !exists {
		return
	}

	err := h.emailService.ToggleEmailStar(c.Request.Context(), userID, emailID)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to toggle email star: "+err.Error())
		return
	}

	h.respondWithSuccess(c, nil, "Email star toggled")
}

// MoveEmailRequest 移动邮件请求
type MoveEmailRequest struct {
	TargetFolderID uint `json:"target_folder_id" binding:"required"`
}

// MoveEmail 移动邮件
func (h *Handler) MoveEmail(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	emailID, exists := h.parseUintParam(c, "id")
	if !exists {
		return
	}

	var req MoveEmailRequest
	if !h.bindJSON(c, &req) {
		return
	}

	err := h.emailService.MoveEmail(c.Request.Context(), userID, emailID, req.TargetFolderID)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to move email: "+err.Error())
		return
	}

	h.respondWithSuccess(c, nil, "Email moved successfully")
}

// SearchEmails 搜索邮件
func (h *Handler) SearchEmails(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	// 解析查询参数
	req := &services.SearchEmailsRequest{
		AccountID:     h.parseOptionalUintQuery(c, "account_id"),
		FolderID:      h.parseOptionalUintQuery(c, "folder_id"),
		Query:         c.Query("q"),
		Subject:       c.Query("subject"),
		From:          c.Query("from"),
		To:            c.Query("to"),
		Body:          c.Query("body"),
		HasAttachment: h.parseOptionalBoolQuery(c, "has_attachment"),
		IsRead:        h.parseOptionalBoolQuery(c, "is_read"),
		IsStarred:     h.parseOptionalBoolQuery(c, "is_starred"),
		Page:          h.parseIntQuery(c, "page", 1),
		PageSize:      h.parseIntQuery(c, "page_size", 20),
	}

	// 解析时间参数
	if sinceStr := c.Query("since"); sinceStr != "" {
		if since, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			req.Since = &since
		}
	}

	if beforeStr := c.Query("before"); beforeStr != "" {
		if before, err := time.Parse(time.RFC3339, beforeStr); err == nil {
			req.Before = &before
		}
	}

	// 验证查询参数
	if req.Query == "" && req.Subject == "" && req.From == "" && req.To == "" && req.Body == "" {
		h.respondWithError(c, http.StatusBadRequest, "At least one search parameter is required")
		return
	}

	// 验证分页参数
	req.Page, req.PageSize = h.validatePagination(req.Page, req.PageSize)

	response, err := h.emailService.SearchEmails(c.Request.Context(), userID, req)
	if err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "Failed to search emails")
		return
	}

	h.respondWithSuccess(c, response)
}

// BatchEmailOperation 批量邮件操作请求
type BatchEmailOperation struct {
	EmailIDs  []uint `json:"email_ids" binding:"required"`
	Operation string `json:"operation" binding:"required,oneof=read unread delete star unstar"`
	FolderID  *uint  `json:"folder_id"` // 用于移动操作
}



// ReplyEmail 回复邮件
func (h *Handler) ReplyEmail(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	emailID, exists := h.parseUintParam(c, "id")
	if !exists {
		return
	}

	var req services.ReplyEmailRequest
	if !h.bindJSON(c, &req) {
		return
	}

	err := h.emailService.ReplyEmail(c.Request.Context(), userID, emailID, &req)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to reply email: "+err.Error())
		return
	}

	h.respondWithSuccess(c, nil, "Email reply sent successfully")
}

// ReplyAllEmail 回复全部邮件
func (h *Handler) ReplyAllEmail(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	emailID, exists := h.parseUintParam(c, "id")
	if !exists {
		return
	}

	var req services.ReplyEmailRequest
	if !h.bindJSON(c, &req) {
		return
	}

	err := h.emailService.ReplyAllEmail(c.Request.Context(), userID, emailID, &req)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to reply all email: "+err.Error())
		return
	}

	h.respondWithSuccess(c, nil, "Email reply all sent successfully")
}



// ForwardEmail 转发邮件
func (h *Handler) ForwardEmail(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	emailID, exists := h.parseUintParam(c, "id")
	if !exists {
		return
	}

	var req services.ForwardEmailRequest
	if !h.bindJSON(c, &req) {
		return
	}

	err := h.emailService.ForwardEmail(c.Request.Context(), userID, emailID, &req)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to forward email: "+err.Error())
		return
	}

	h.respondWithSuccess(c, nil, "Email forwarded successfully")
}

// ArchiveEmail 归档邮件
func (h *Handler) ArchiveEmail(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	emailID, exists := h.parseUintParam(c, "id")
	if !exists {
		return
	}

	err := h.emailService.ArchiveEmail(c.Request.Context(), userID, emailID)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to archive email: "+err.Error())
		return
	}

	h.respondWithSuccess(c, nil, "Email archived successfully")
}

// BatchEmailOperations 批量邮件操作
func (h *Handler) BatchEmailOperations(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	var req BatchEmailOperation
	if !h.bindJSON(c, &req) {
		return
	}

	if len(req.EmailIDs) == 0 {
		h.respondWithError(c, http.StatusBadRequest, "No email IDs provided")
		return
	}

	if len(req.EmailIDs) > 100 {
		h.respondWithError(c, http.StatusBadRequest, "Too many emails (max 100)")
		return
	}

	var errors []string
	successCount := 0

	for _, emailID := range req.EmailIDs {
		var err error

		switch req.Operation {
		case "read":
			err = h.emailService.MarkEmailAsRead(c.Request.Context(), userID, emailID)
		case "unread":
			err = h.emailService.MarkEmailAsUnread(c.Request.Context(), userID, emailID)
		case "delete":
			err = h.emailService.DeleteEmail(c.Request.Context(), userID, emailID)
		case "star", "unstar":
			err = h.emailService.ToggleEmailStar(c.Request.Context(), userID, emailID)
		case "move":
			if req.FolderID == nil {
				err = fmt.Errorf("folder_id is required for move operation")
			} else {
				err = h.emailService.MoveEmail(c.Request.Context(), userID, emailID, *req.FolderID)
			}
		default:
			err = fmt.Errorf("unsupported operation: %s", req.Operation)
		}

		if err != nil {
			errors = append(errors, fmt.Sprintf("Email %d: %v", emailID, err))
		} else {
			successCount++
		}
	}

	result := map[string]interface{}{
		"success_count": successCount,
		"total_count":   len(req.EmailIDs),
		"errors":        errors,
	}

	if len(errors) > 0 {
		h.respondWithSuccess(c, result, fmt.Sprintf("Batch operation completed with %d errors", len(errors)))
	} else {
		h.respondWithSuccess(c, result, "Batch operation completed successfully")
	}
}
