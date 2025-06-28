package handlers

import (
	"context"
	"fmt"
	"net/http"

	"firemail/internal/auth"
	"firemail/internal/cache"
	"firemail/internal/config"
	"firemail/internal/middleware"
	"firemail/internal/providers"
	"firemail/internal/services"
	"firemail/internal/sse"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler HTTP处理器
type Handler struct {
	db                *gorm.DB
	config            *config.Config
	authService       *auth.Service
	emailService      services.EmailService
	syncService       *services.SyncService
	providerFactory   *providers.ProviderFactory
	sseService        sse.SSEService
	oauthStateService services.OAuth2StateService
	backupService         services.BackupService
	softDeleteService     services.SoftDeleteService
	attachmentService     services.AttachmentDownloader
	scheduledEmailService services.ScheduledEmailService
}

// New 创建处理器实例
func New(db *gorm.DB, cfg *config.Config) *Handler {
	// 创建JWT管理器
	jwtManager := auth.NewJWTManager(cfg.Auth.JWTSecret, cfg.Auth.JWTExpiry)

	// 创建认证服务
	authService := auth.NewService(db, jwtManager)

	// 创建提供商工厂
	providerFactory := providers.NewProviderFactory()

	// 创建SSE配置
	sseConfig := &sse.SSEConfig{
		MaxConnectionsPerUser: cfg.SSE.MaxConnectionsPerUser,
		ConnectionTimeout:     cfg.SSE.ConnectionTimeout,
		HeartbeatInterval:     cfg.SSE.HeartbeatInterval,
		CleanupInterval:       cfg.SSE.CleanupInterval,
		BufferSize:            cfg.SSE.BufferSize,
		EnableHeartbeat:       cfg.SSE.EnableHeartbeat,
	}

	// 创建SSE服务
	sseService := sse.NewSSEService(db, sseConfig)

	// 创建邮件服务
	emailService := services.NewEmailService(db, providerFactory, sseService.GetEventPublisher())

	// 创建去重工厂
	deduplicatorFactory := services.NewDeduplicatorFactory(db)

	// 创建附件存储（需要在同步服务之前创建）
	attachmentStorage := services.NewLocalFileStorage(nil) // 使用默认配置

	// 创建同步服务（现在包含附件存储和缓存管理器）
	syncService := services.NewSyncService(db, providerFactory, sseService.GetEventPublisher(), deduplicatorFactory, attachmentStorage, cache.GlobalCacheManager)

	// 设置EmailService的SyncService依赖
	if emailServiceImpl, ok := emailService.(*services.EmailServiceImpl); ok {
		emailServiceImpl.SetSyncService(syncService)
	}

	// 创建OAuth2状态管理服务
	oauthStateService := services.NewOAuth2StateService(db)

	// 创建备份服务
	backupService := services.NewBackupService(db, cfg.Database.Path, cfg.Database.BackupDir, cfg.Database.BackupMaxCount, cfg.Database.BackupIntervalHours)

	// 创建软删除管理服务
	softDeleteService := services.NewSoftDeleteService(db)

	// 创建附件服务
	attachmentService := services.NewAttachmentService(db, attachmentStorage, providerFactory)

	// 设置EmailService的AttachmentService依赖
	if emailServiceImpl, ok := emailService.(*services.EmailServiceImpl); ok {
		emailServiceImpl.SetAttachmentService(attachmentService)
	}

	// 创建邮件组装器和发送器
	emailComposer := services.NewStandardEmailComposer(&services.EmailComposerConfig{}, db)
	emailSender := services.NewStandardEmailSender(db, providerFactory, sseService.GetEventPublisher())

	// 创建定时邮件服务
	scheduledEmailService := services.NewScheduledEmailService(db, emailService, emailComposer, emailSender)

	return &Handler{
		db:                    db,
		config:                cfg,
		authService:           authService,
		emailService:          emailService,
		syncService:           syncService,
		providerFactory:       providerFactory,
		sseService:            sseService,
		oauthStateService:     oauthStateService,
		backupService:         backupService,
		softDeleteService:     softDeleteService,
		attachmentService:     attachmentService,
		scheduledEmailService: scheduledEmailService,
	}
}

// AuthRequired 返回认证中间件
func (h *Handler) AuthRequired() gin.HandlerFunc {
	return middleware.AuthRequiredWithService(h.authService)
}

// OptionalAuth 返回可选认证中间件
func (h *Handler) OptionalAuth() gin.HandlerFunc {
	return middleware.OptionalAuthWithService(h.authService)
}

// GetAuthService 获取认证服务（用于向后兼容）
func (h *Handler) GetAuthService() middleware.AuthService {
	return h.authService
}

// GetDB 获取数据库连接
func (h *Handler) GetDB() *gorm.DB {
	return h.db
}

// GetProviderFactory 获取提供商工厂
func (h *Handler) GetProviderFactory() *providers.ProviderFactory {
	return h.providerFactory
}

// HealthCheck 健康检查
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "FireMail",
		"version": "1.0.0",
	})
}

// ErrorResponse 错误响应结构
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// SuccessResponse 成功响应结构
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// respondWithError 返回错误响应
func (h *Handler) respondWithError(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	})
}

// respondWithSuccess 返回成功响应
func (h *Handler) respondWithSuccess(c *gin.Context, data interface{}, message ...string) {
	response := SuccessResponse{
		Success: true,
		Data:    data,
	}

	if len(message) > 0 {
		response.Message = message[0]
	}

	c.JSON(http.StatusOK, response)
}

// respondWithCreated 返回创建成功响应
func (h *Handler) respondWithCreated(c *gin.Context, data interface{}, message ...string) {
	response := SuccessResponse{
		Success: true,
		Data:    data,
	}

	if len(message) > 0 {
		response.Message = message[0]
	}

	c.JSON(http.StatusCreated, response)
}

// bindJSON 绑定JSON请求体
func (h *Handler) bindJSON(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return false
	}
	return true
}

// getCurrentUserID 获取当前用户ID
func (h *Handler) getCurrentUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		h.respondWithError(c, http.StatusUnauthorized, "User not authenticated")
		return 0, false
	}
	return userID.(uint), true
}

// getCurrentUser 获取当前用户
func (h *Handler) getCurrentUser(c *gin.Context) (*auth.JWTClaims, bool) {
	user, exists := c.Get("user")
	if !exists {
		h.respondWithError(c, http.StatusUnauthorized, "User not authenticated")
		return nil, false
	}
	return user.(*auth.JWTClaims), true
}

// validatePagination 验证分页参数
func (h *Handler) validatePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}

// parseUintParam 解析uint参数
func (h *Handler) parseUintParam(c *gin.Context, paramName string) (uint, bool) {
	paramStr := c.Param(paramName)
	if paramStr == "" {
		h.respondWithError(c, http.StatusBadRequest, "Missing parameter: "+paramName)
		return 0, false
	}

	var paramValue uint
	if _, err := fmt.Sscanf(paramStr, "%d", &paramValue); err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Invalid parameter: "+paramName)
		return 0, false
	}

	return paramValue, true
}

// parseOptionalUintQuery 解析可选的uint查询参数
func (h *Handler) parseOptionalUintQuery(c *gin.Context, queryName string) *uint {
	queryStr := c.Query(queryName)
	if queryStr == "" {
		return nil
	}

	var queryValue uint
	if _, err := fmt.Sscanf(queryStr, "%d", &queryValue); err != nil {
		return nil
	}

	return &queryValue
}

// parseOptionalBoolQuery 解析可选的bool查询参数
func (h *Handler) parseOptionalBoolQuery(c *gin.Context, queryName string) *bool {
	queryStr := c.Query(queryName)
	if queryStr == "" {
		return nil
	}

	var queryValue bool
	switch queryStr {
	case "true", "1", "yes":
		queryValue = true
	case "false", "0", "no":
		queryValue = false
	default:
		return nil
	}

	return &queryValue
}

// parseIntQuery 解析int查询参数
func (h *Handler) parseIntQuery(c *gin.Context, queryName string, defaultValue int) int {
	queryStr := c.Query(queryName)
	if queryStr == "" {
		return defaultValue
	}

	var queryValue int
	if _, err := fmt.Sscanf(queryStr, "%d", &queryValue); err != nil {
		return defaultValue
	}

	return queryValue
}

// parseUintQuery 解析uint查询参数
func (h *Handler) parseUintQuery(c *gin.Context, queryName string, defaultValue uint) uint {
	queryStr := c.Query(queryName)
	if queryStr == "" {
		return defaultValue
	}

	var queryValue uint
	if _, err := fmt.Sscanf(queryStr, "%d", &queryValue); err != nil {
		return defaultValue
	}

	return queryValue
}

// validateSortParams 验证排序参数
func (h *Handler) validateSortParams(sortBy, sortOrder string) (string, string) {
	validSortFields := map[string]bool{
		"date":    true,
		"subject": true,
		"from":    true,
		"size":    true,
	}

	if !validSortFields[sortBy] {
		sortBy = "date"
	}

	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	return sortBy, sortOrder
}

// StartBackupService 启动备份服务
func (h *Handler) StartBackupService(ctx context.Context) error {
	return h.backupService.StartAutoBackup(ctx)
}

// StartSoftDeleteCleanup 启动软删除清理服务
func (h *Handler) StartSoftDeleteCleanup(ctx context.Context, retentionDays int) error {
	return h.softDeleteService.StartAutoCleanup(ctx, retentionDays)
}

// StartTemporaryAttachmentCleanup 启动临时附件清理服务
func (h *Handler) StartTemporaryAttachmentCleanup(ctx context.Context, maxAgeHours int) error {
	// 获取AttachmentService实例
	if attachmentService, ok := h.attachmentService.(*services.AttachmentService); ok {
		return attachmentService.StartAutoCleanup(ctx, maxAgeHours)
	}
	return fmt.Errorf("attachment service does not support auto cleanup")
}

// StartScheduledEmailService 启动定时邮件服务
func (h *Handler) StartScheduledEmailService(ctx context.Context) error {
	return h.scheduledEmailService.StartScheduler(ctx)
}
