package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"firemail/internal/middleware"
	"firemail/internal/models"
	"firemail/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AttachmentHandler 附件处理器
type AttachmentHandler struct {
	attachmentService services.AttachmentDownloader
	db                *gorm.DB
}

// NewAttachmentHandler 创建附件处理器
func NewAttachmentHandler(attachmentService services.AttachmentDownloader, db *gorm.DB) *AttachmentHandler {
	return &AttachmentHandler{
		attachmentService: attachmentService,
		db:                db,
	}
}

// RegisterRoutes 注册路由
func (h *AttachmentHandler) RegisterRoutes(router *gin.RouterGroup) {
	attachments := router.Group("/attachments")
	attachments.Use(middleware.AuthRequired())
	{
		// 上传附件（用于邮件发送）
		attachments.POST("/upload", h.UploadAttachment)

		// 下载附件
		attachments.GET("/:id/download", h.DownloadAttachment)

		// 预览附件
		attachments.GET("/:id/preview", h.PreviewAttachment)

		// 获取下载进度
		attachments.GET("/:id/progress", h.GetDownloadProgress)

		// 强制重新下载
		attachments.POST("/:id/download", h.ForceDownloadAttachment)
	}

	// 邮件相关的附件操作
	emails := router.Group("/emails")
	emails.Use(middleware.AuthRequired())
	{
		// 获取邮件的所有附件
		emails.GET("/:id/attachments", h.GetEmailAttachments)
		
		// 下载邮件的所有附件
		emails.POST("/:id/attachments/download", h.DownloadEmailAttachments)
	}
}

// DownloadAttachment 下载附件
func (h *AttachmentHandler) DownloadAttachment(c *gin.Context) {
	userID := middleware.GetUserID(c)
	
	// 获取附件ID
	attachmentID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid attachment ID"})
		return
	}

	// 获取附件内容
	content, err := h.attachmentService.GetAttachmentContent(c.Request.Context(), uint(attachmentID), userID)
	if err != nil {
		log.Printf("Failed to get attachment content: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get attachment"})
		return
	}
	defer content.Close()

	// 获取附件信息（用于设置响应头）
	attachmentInfo, err := h.getAttachmentInfo(c, uint(attachmentID), userID)
	if err != nil {
		log.Printf("Failed to get attachment info: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get attachment info"})
		return
	}

	// 获取实际文件大小以修复Content-Length不匹配问题
	attachment, err := h.attachmentService.(*services.AttachmentService).GetAttachmentWithPermissionCheck(c.Request.Context(), uint(attachmentID), userID)
	if err == nil {
		storage := h.attachmentService.(*services.AttachmentService).GetStorage()
		if storageInfo, err := storage.GetStorageInfo(c.Request.Context(), attachment); err == nil {
			// 使用实际文件大小而不是数据库中的大小
			attachmentInfo.Size = storageInfo.Size
		}
	}

	// 设置响应头
	h.setDownloadHeaders(c, attachmentInfo)

	// 流式传输文件内容
	_, err = io.Copy(c.Writer, content)
	if err != nil {
		log.Printf("Failed to stream attachment content: %v", err)
	}
}

// PreviewAttachment 预览附件
func (h *AttachmentHandler) PreviewAttachment(c *gin.Context) {
	userID := middleware.GetUserID(c)
	
	// 获取附件ID
	attachmentID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid attachment ID"})
		return
	}

	// 获取预览
	preview, err := h.attachmentService.PreviewAttachment(c.Request.Context(), uint(attachmentID), userID)
	if err != nil {
		log.Printf("Failed to get attachment preview: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get preview"})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    preview,
	})
}

// GetDownloadProgress 获取下载进度
func (h *AttachmentHandler) GetDownloadProgress(c *gin.Context) {
	// 获取附件ID
	attachmentID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid attachment ID"})
		return
	}

	// 获取下载进度
	progress, err := h.attachmentService.GetDownloadProgress(c.Request.Context(), uint(attachmentID))
	if err != nil {
		log.Printf("Failed to get download progress: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get progress"})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    progress,
	})
}

// ForceDownloadAttachment 强制重新下载附件
func (h *AttachmentHandler) ForceDownloadAttachment(c *gin.Context) {
	userID := middleware.GetUserID(c)
	
	// 获取附件ID
	attachmentID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid attachment ID"})
		return
	}

	// 异步下载附件
	go func() {
		if err := h.attachmentService.DownloadAttachment(c.Request.Context(), uint(attachmentID), userID); err != nil {
			log.Printf("Failed to download attachment %d: %v", attachmentID, err)
		}
	}()

	c.JSON(http.StatusAccepted, SuccessResponse{
		Success: true,
		Message: "Download started",
	})
}

// GetEmailAttachments 获取邮件的所有附件
func (h *AttachmentHandler) GetEmailAttachments(c *gin.Context) {
	userID := middleware.GetUserID(c)
	
	// 获取邮件ID
	emailID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email ID"})
		return
	}

	// 获取附件列表
	attachments, err := h.getEmailAttachmentsList(c, uint(emailID), userID)
	if err != nil {
		log.Printf("Failed to get email attachments: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get attachments"})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    attachments,
	})
}

// DownloadEmailAttachments 下载邮件的所有附件
func (h *AttachmentHandler) DownloadEmailAttachments(c *gin.Context) {
	userID := middleware.GetUserID(c)
	
	// 获取邮件ID
	emailID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email ID"})
		return
	}

	// 异步下载所有附件
	go func() {
		if err := h.attachmentService.DownloadEmailAttachments(c.Request.Context(), uint(emailID), userID); err != nil {
			log.Printf("Failed to download email attachments %d: %v", emailID, err)
		}
	}()

	c.JSON(http.StatusAccepted, SuccessResponse{
		Success: true,
		Message: "Download started for all attachments",
	})
}

// AttachmentInfo 附件信息
type AttachmentInfo struct {
	ID          uint   `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	Disposition string `json:"disposition"`
	IsDownloaded bool  `json:"is_downloaded"`
	IsInline    bool   `json:"is_inline"`
}

// getAttachmentInfo 获取附件信息
func (h *AttachmentHandler) getAttachmentInfo(c *gin.Context, attachmentID uint, userID uint) (*AttachmentInfo, error) {
	var attachment models.Attachment

	// 首先尝试查找临时附件（email_id为NULL）
	err := h.db.WithContext(c.Request.Context()).
		Where("id = ? AND email_id IS NULL AND user_id = ?", attachmentID, userID).
		First(&attachment).Error

	if err == nil {
		// 找到临时附件，构造AttachmentInfo并返回
		return &AttachmentInfo{
			ID:          attachment.ID,
			Filename:    attachment.Filename,
			ContentType: attachment.ContentType,
			Size:        attachment.Size,
			Disposition: attachment.Disposition,
		}, nil
	}

	if err != gorm.ErrRecordNotFound {
		// 数据库查询错误
		return nil, fmt.Errorf("failed to query temporary attachment: %w", err)
	}

	// 没有找到临时附件，尝试查找正常附件（通过邮件账户权限）
	err = h.db.WithContext(c.Request.Context()).
		Joins("JOIN emails ON attachments.email_id = emails.id").
		Joins("JOIN email_accounts ON emails.account_id = email_accounts.id").
		Where("attachments.id = ? AND email_accounts.user_id = ?", attachmentID, userID).
		First(&attachment).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("attachment not found or access denied")
		}
		return nil, fmt.Errorf("failed to get attachment: %w", err)
	}

	return &AttachmentInfo{
		ID:          attachment.ID,
		Filename:    attachment.Filename,
		ContentType: attachment.ContentType,
		Size:        attachment.Size,
		Disposition: attachment.Disposition,
		IsDownloaded: attachment.IsDownloaded,
		IsInline:    attachment.IsInline,
	}, nil
}

// getEmailAttachmentsList 获取邮件附件列表
func (h *AttachmentHandler) getEmailAttachmentsList(c *gin.Context, emailID uint, userID uint) ([]AttachmentInfo, error) {
	var attachments []models.Attachment
	err := h.db.WithContext(c.Request.Context()).
		Joins("JOIN emails ON attachments.email_id = emails.id").
		Joins("JOIN email_accounts ON emails.account_id = email_accounts.id").
		Where("emails.id = ? AND email_accounts.user_id = ?", emailID, userID).
		Find(&attachments).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get email attachments: %w", err)
	}

	var result []AttachmentInfo
	for _, attachment := range attachments {
		result = append(result, AttachmentInfo{
			ID:          attachment.ID,
			Filename:    attachment.Filename,
			ContentType: attachment.ContentType,
			Size:        attachment.Size,
			Disposition: attachment.Disposition,
			IsDownloaded: attachment.IsDownloaded,
			IsInline:    attachment.IsInline,
		})
	}

	return result, nil
}

// setDownloadHeaders 设置下载响应头
func (h *AttachmentHandler) setDownloadHeaders(c *gin.Context, info *AttachmentInfo) {
	// 设置内容类型
	if info.ContentType != "" {
		c.Header("Content-Type", info.ContentType)
	} else {
		c.Header("Content-Type", "application/octet-stream")
	}

	// 设置文件大小
	if info.Size > 0 {
		c.Header("Content-Length", fmt.Sprintf("%d", info.Size))
	}

	// 设置文件名
	filename := info.Filename
	if filename == "" {
		filename = fmt.Sprintf("attachment_%d", info.ID)
	}

	// 处理文件名中的特殊字符
	safeFilename := h.sanitizeFilename(filename)
	
	// 设置下载头
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, safeFilename))
	
	// 设置缓存控制
	c.Header("Cache-Control", "private, max-age=3600")
	
	// 设置其他安全头
	c.Header("X-Content-Type-Options", "nosniff")
}

// sanitizeFilename 清理文件名
func (h *AttachmentHandler) sanitizeFilename(filename string) string {
	// 移除路径分隔符和危险字符
	safe := strings.ReplaceAll(filename, "/", "_")
	safe = strings.ReplaceAll(safe, "\\", "_")
	safe = strings.ReplaceAll(safe, "..", "_")
	
	// 限制长度
	if len(safe) > 200 {
		ext := filepath.Ext(safe)
		name := strings.TrimSuffix(safe, ext)
		if len(name) > 200-len(ext) {
			name = name[:200-len(ext)]
		}
		safe = name + ext
	}
	
	return safe
}

// AttachmentUploadRequest 附件上传请求
type AttachmentUploadRequest struct {
	EmailID     uint   `json:"email_id" binding:"required"`
	Filename    string `json:"filename" binding:"required"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

// UploadAttachment 上传附件（用于邮件发送）
func (h *AttachmentHandler) UploadAttachment(c *gin.Context) {
	userID := middleware.GetUserID(c) // 获取用户ID用于权限验证

	// 获取上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	// 验证文件大小
	if header.Size > 25*1024*1024 { // 25MB限制
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large (max 25MB)"})
		return
	}

	// 验证文件名
	if header.Filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid filename"})
		return
	}

	// 创建临时附件记录（用于邮件发送）
	attachment := &models.Attachment{
		EmailID:      nil,    // 临时上传的附件，暂不关联邮件
		UserID:       &userID, // 设置用户ID用于权限检查
		Filename:     header.Filename,
		ContentType:  header.Header.Get("Content-Type"),
		Size:         header.Size,
		Disposition:  "attachment",
		IsDownloaded: true, // 上传的文件直接标记为已下载
	}

	// 如果没有Content-Type，根据文件扩展名推断
	if attachment.ContentType == "" {
		attachment.ContentType = h.getContentTypeByExtension(header.Filename)
	}

	// 保存到数据库
	if err := h.db.WithContext(c.Request.Context()).Create(attachment).Error; err != nil {
		log.Printf("Failed to create attachment record: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save attachment"})
		return
	}

	// 保存文件到存储
	storage := h.attachmentService.(*services.AttachmentService).GetStorage()
	if err := storage.Store(c.Request.Context(), attachment, file); err != nil {
		// 如果存储失败，删除数据库记录
		h.db.Delete(attachment)
		log.Printf("Failed to store attachment file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store attachment"})
		return
	}

	// 更新存储路径（只更新file_path字段，避免触发器递归）
	storagePath := storage.GetStoragePath(attachment)
	if err := h.db.Model(attachment).Update("file_path", storagePath).Error; err != nil {
		log.Printf("Failed to update attachment storage path: %v", err)
		// 不返回错误，因为文件已经保存成功，只是路径更新失败
	}
	// 更新内存中的对象，以便返回正确的数据
	attachment.StoragePath = storagePath

	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Message: "Attachment uploaded successfully",
		Data: gin.H{
			"attachment_id": attachment.ID,
			"filename":      attachment.Filename,
			"size":          attachment.Size,
			"content_type":  attachment.ContentType,
		},
	})
}

// getContentTypeByExtension 根据文件扩展名获取Content-Type
func (h *AttachmentHandler) getContentTypeByExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	contentTypes := map[string]string{
		".txt":  "text/plain",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".bmp":  "image/bmp",
		".svg":  "image/svg+xml",
		".zip":  "application/zip",
		".rar":  "application/x-rar-compressed",
		".7z":   "application/x-7z-compressed",
		".mp3":  "audio/mpeg",
		".mp4":  "video/mp4",
		".avi":  "video/x-msvideo",
		".mov":  "video/quicktime",
	}

	if contentType, exists := contentTypes[ext]; exists {
		return contentType
	}

	return "application/octet-stream"
}
