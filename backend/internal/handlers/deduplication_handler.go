package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"firemail/internal/middleware"
	"firemail/internal/models"
	"firemail/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DeduplicationHandler 去重处理器
type DeduplicationHandler struct {
	deduplicationManager services.DeduplicationManager
	db                   *gorm.DB
}

// NewDeduplicationHandler 创建去重处理器
func NewDeduplicationHandler(deduplicationManager services.DeduplicationManager, db *gorm.DB) *DeduplicationHandler {
	return &DeduplicationHandler{
		deduplicationManager: deduplicationManager,
		db:                   db,
	}
}

// RegisterRoutes 注册路由
func (h *DeduplicationHandler) RegisterRoutes(router *gin.RouterGroup) {
	dedup := router.Group("/deduplication")
	dedup.Use(middleware.AuthRequired())
	{
		// 账户去重
		dedup.POST("/accounts/:id/deduplicate", h.DeduplicateAccount)

		// 用户所有账户去重
		dedup.POST("/user/deduplicate", h.DeduplicateUser)

		// 获取去重报告
		dedup.GET("/accounts/:id/report", h.GetDeduplicationReport)

		// 计划去重任务
		dedup.POST("/accounts/:id/schedule", h.ScheduleDeduplication)

		// 取消计划去重任务
		dedup.DELETE("/accounts/:id/schedule", h.CancelScheduledDeduplication)

		// 获取去重统计
		dedup.GET("/accounts/:id/stats", h.GetDeduplicationStats)
	}
}

// DeduplicateAccountRequest 账户去重请求
type DeduplicateAccountRequest struct {
	DryRun             bool     `json:"dry_run"`
	CrossFolder        bool     `json:"cross_folder"`
	CleanupDuplicates  bool     `json:"cleanup_duplicates"`
	RebuildIndex       bool     `json:"rebuild_index"`
	BatchSize          int      `json:"batch_size"`
	IncludeFolders     []string `json:"include_folders"`
	ExcludeFolders     []string `json:"exclude_folders"`
	NotifyOnCompletion bool     `json:"notify_on_completion"`
}

// DeduplicateAccount 执行账户去重
func (h *DeduplicationHandler) DeduplicateAccount(c *gin.Context) {
	userID := middleware.GetUserID(c)

	accountID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid account ID",
			Message: err.Error(),
		})
		return
	}

	var req DeduplicateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	// 验证账户权限
	if err := h.validateAccountAccess(c, uint(accountID), userID); err != nil {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Access denied",
			Message: "You don't have access to this email account",
		})
		return
	}

	// 转换请求为选项
	options := &services.DeduplicationOptions{
		DryRun:             req.DryRun,
		CrossFolder:        req.CrossFolder,
		CleanupDuplicates:  req.CleanupDuplicates,
		RebuildIndex:       req.RebuildIndex,
		BatchSize:          req.BatchSize,
		IncludeFolders:     req.IncludeFolders,
		ExcludeFolders:     req.ExcludeFolders,
		NotifyOnCompletion: req.NotifyOnCompletion,
	}

	// 设置默认值
	if options.BatchSize <= 0 {
		options.BatchSize = 100
	}

	// 执行去重
	result, err := h.deduplicationManager.DeduplicateAccount(c.Request.Context(), uint(accountID), options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to deduplicate account",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Account deduplication completed",
		Data:    result,
	})
}

// DeduplicateUserRequest 用户去重请求
type DeduplicateUserRequest struct {
	DryRun             bool     `json:"dry_run"`
	CrossFolder        bool     `json:"cross_folder"`
	CleanupDuplicates  bool     `json:"cleanup_duplicates"`
	RebuildIndex       bool     `json:"rebuild_index"`
	BatchSize          int      `json:"batch_size"`
	IncludeFolders     []string `json:"include_folders"`
	ExcludeFolders     []string `json:"exclude_folders"`
	NotifyOnCompletion bool     `json:"notify_on_completion"`
}

// DeduplicateUser 执行用户所有账户去重
func (h *DeduplicationHandler) DeduplicateUser(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req DeduplicateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	// 转换请求为选项
	options := &services.DeduplicationOptions{
		DryRun:             req.DryRun,
		CrossFolder:        req.CrossFolder,
		CleanupDuplicates:  req.CleanupDuplicates,
		RebuildIndex:       req.RebuildIndex,
		BatchSize:          req.BatchSize,
		IncludeFolders:     req.IncludeFolders,
		ExcludeFolders:     req.ExcludeFolders,
		NotifyOnCompletion: req.NotifyOnCompletion,
	}

	// 设置默认值
	if options.BatchSize <= 0 {
		options.BatchSize = 100
	}

	// 执行用户去重
	result, err := h.deduplicationManager.DeduplicateUser(c.Request.Context(), userID, options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to deduplicate user accounts",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "User accounts deduplication completed",
		Data:    result,
	})
}

// GetDeduplicationReport 获取去重报告
func (h *DeduplicationHandler) GetDeduplicationReport(c *gin.Context) {
	userID := middleware.GetUserID(c)

	accountID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid account ID",
			Message: err.Error(),
		})
		return
	}

	// 验证账户权限
	if err := h.validateAccountAccess(c, uint(accountID), userID); err != nil {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Access denied",
			Message: "You don't have access to this email account",
		})
		return
	}

	// 获取去重报告
	report, err := h.deduplicationManager.GetDeduplicationReport(c.Request.Context(), uint(accountID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get deduplication report",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    report,
	})
}

// ScheduleDeduplicationRequest 计划去重请求
type ScheduleDeduplicationRequest struct {
	Enabled   bool                           `json:"enabled"`
	Frequency string                         `json:"frequency"` // daily, weekly, monthly
	Time      string                         `json:"time"`      // HH:MM format
	Options   *services.DeduplicationOptions `json:"options"`
}

// ScheduleDeduplication 计划去重任务
func (h *DeduplicationHandler) ScheduleDeduplication(c *gin.Context) {
	userID := middleware.GetUserID(c)

	accountID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid account ID",
			Message: err.Error(),
		})
		return
	}

	var req ScheduleDeduplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	// 验证账户权限
	if err := h.validateAccountAccess(c, uint(accountID), userID); err != nil {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Access denied",
			Message: "You don't have access to this email account",
		})
		return
	}

	// 创建计划
	schedule := &services.DeduplicationSchedule{
		Enabled:   req.Enabled,
		Frequency: req.Frequency,
		Time:      req.Time,
		Options:   req.Options,
	}

	// 计划去重任务
	err = h.deduplicationManager.ScheduleDeduplication(c.Request.Context(), uint(accountID), schedule)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to schedule deduplication",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Deduplication scheduled successfully",
		Data:    schedule,
	})
}

// CancelScheduledDeduplication 取消计划去重任务
func (h *DeduplicationHandler) CancelScheduledDeduplication(c *gin.Context) {
	userID := middleware.GetUserID(c)

	accountID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid account ID",
			Message: err.Error(),
		})
		return
	}

	// 验证账户权限
	if err := h.validateAccountAccess(c, uint(accountID), userID); err != nil {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Access denied",
			Message: "You don't have access to this email account",
		})
		return
	}

	// 取消计划去重任务
	err = h.deduplicationManager.CancelScheduledDeduplication(c.Request.Context(), uint(accountID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to cancel scheduled deduplication",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Scheduled deduplication cancelled successfully",
	})
}

// GetDeduplicationStats 获取去重统计
func (h *DeduplicationHandler) GetDeduplicationStats(c *gin.Context) {
	userID := middleware.GetUserID(c)

	accountID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid account ID",
			Message: err.Error(),
		})
		return
	}

	// 验证账户权限
	if err := h.validateAccountAccess(c, uint(accountID), userID); err != nil {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Access denied",
			Message: "You don't have access to this email account",
		})
		return
	}

	// 获取去重报告（包含统计信息）
	report, err := h.deduplicationManager.GetDeduplicationReport(c.Request.Context(), uint(accountID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get deduplication stats",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    report.Stats,
	})
}

// validateAccountAccess 验证账户访问权限
func (h *DeduplicationHandler) validateAccountAccess(c *gin.Context, accountID, userID uint) error {
	var count int64
	err := h.db.WithContext(c.Request.Context()).
		Model(&models.EmailAccount{}).
		Where("id = ? AND user_id = ?", accountID, userID).
		Count(&count).Error

	if err != nil {
		return fmt.Errorf("failed to validate account access: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("account not found or access denied")
	}

	return nil
}
