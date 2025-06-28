package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"firemail/internal/encoding/transfer"
	"firemail/internal/models"
	"firemail/internal/providers"

	"gorm.io/gorm"
)

// AttachmentDownloader 附件下载器接口
type AttachmentDownloader interface {
	// DownloadAttachment 下载指定附件
	DownloadAttachment(ctx context.Context, attachmentID uint, userID uint) error

	// DownloadEmailAttachments 下载邮件的所有附件
	DownloadEmailAttachments(ctx context.Context, emailID uint, userID uint) error

	// GetAttachmentContent 获取附件内容
	GetAttachmentContent(ctx context.Context, attachmentID uint, userID uint) (io.ReadCloser, error)

	// PreviewAttachment 预览附件
	PreviewAttachment(ctx context.Context, attachmentID uint, userID uint) (*AttachmentPreview, error)

	// GetDownloadProgress 获取下载进度
	GetDownloadProgress(ctx context.Context, attachmentID uint) (*DownloadProgress, error)

	// CleanupTemporaryAttachments 清理临时附件
	CleanupTemporaryAttachments(ctx context.Context, maxAgeHours int) error
}

// ProviderFactory 提供商工厂接口（本地别名）
type ProviderFactory = providers.ProviderFactoryInterface

// AttachmentService 附件服务实现
type AttachmentService struct {
	db                     *gorm.DB
	storage                AttachmentStorage
	providerFactory        ProviderFactory
	downloadProgress       map[uint]*DownloadProgress
	progressMutex          sync.RWMutex
	maxConcurrentDownloads int
	downloadSemaphore      chan struct{}
	cleanupStopChan        chan struct{}
}

// AttachmentPreview 附件预览信息
type AttachmentPreview struct {
	AttachmentID uint   `json:"attachment_id"`
	Type         string `json:"type"` // "image", "text", "pdf", "unknown"
	Content      []byte `json:"content,omitempty"`
	Thumbnail    []byte `json:"thumbnail,omitempty"`
	Text         string `json:"text,omitempty"`
	Error        string `json:"error,omitempty"`
}

// DownloadProgress 下载进度
type DownloadProgress struct {
	AttachmentID uint      `json:"attachment_id"`
	Status       string    `json:"status"` // "pending", "downloading", "completed", "failed"
	Progress     float64   `json:"progress"` // 0.0 - 1.0
	BytesTotal   int64     `json:"bytes_total"`
	BytesLoaded  int64     `json:"bytes_loaded"`
	StartTime    time.Time `json:"start_time"`
	EndTime      *time.Time `json:"end_time,omitempty"`
	Error        string    `json:"error,omitempty"`
}

// NewAttachmentService 创建附件服务
func NewAttachmentService(db *gorm.DB, storage AttachmentStorage, providerFactory ProviderFactory) AttachmentDownloader {
	maxConcurrent := 5 // 默认最大并发下载数
	
	return &AttachmentService{
		db:                     db,
		storage:                storage,
		providerFactory:        providerFactory,
		downloadProgress:       make(map[uint]*DownloadProgress),
		maxConcurrentDownloads: maxConcurrent,
		downloadSemaphore:      make(chan struct{}, maxConcurrent),
		cleanupStopChan:        make(chan struct{}),
	}
}

// DownloadAttachment 下载指定附件
func (s *AttachmentService) DownloadAttachment(ctx context.Context, attachmentID uint, userID uint) error {
	// 获取附件信息
	attachment, err := s.getAttachmentWithPermissionCheck(ctx, attachmentID, userID)
	if err != nil {
		return err
	}

	// 检查是否已下载
	if attachment.IsDownloaded && s.storage.Exists(ctx, attachment) {
		log.Printf("Attachment %d already downloaded", attachmentID)
		return nil
	}

	// 获取下载信号量
	select {
	case s.downloadSemaphore <- struct{}{}:
		defer func() { <-s.downloadSemaphore }()
	case <-ctx.Done():
		return ctx.Err()
	}

	// 初始化下载进度
	progress := &DownloadProgress{
		AttachmentID: attachmentID,
		Status:       "downloading",
		Progress:     0.0,
		BytesTotal:   attachment.Size,
		BytesLoaded:  0,
		StartTime:    time.Now(),
	}
	s.setDownloadProgress(attachmentID, progress)

	// 执行下载
	err = s.downloadAttachmentContent(ctx, attachment, progress)
	
	// 更新最终状态
	endTime := time.Now()
	progress.EndTime = &endTime
	
	if err != nil {
		progress.Status = "failed"
		progress.Error = err.Error()
		s.setDownloadProgress(attachmentID, progress)
		return fmt.Errorf("failed to download attachment %d: %w", attachmentID, err)
	}

	progress.Status = "completed"
	progress.Progress = 1.0
	progress.BytesLoaded = attachment.Size
	s.setDownloadProgress(attachmentID, progress)

	log.Printf("Successfully downloaded attachment %d (%s)", attachmentID, attachment.Filename)
	return nil
}

// DownloadEmailAttachments 下载邮件的所有附件
func (s *AttachmentService) DownloadEmailAttachments(ctx context.Context, emailID uint, userID uint) error {
	// 获取邮件的所有附件
	var attachments []models.Attachment
	err := s.db.WithContext(ctx).
		Joins("JOIN emails ON attachments.email_id = emails.id").
		Joins("JOIN email_accounts ON emails.account_id = email_accounts.id").
		Where("emails.id = ? AND email_accounts.user_id = ?", emailID, userID).
		Find(&attachments).Error

	if err != nil {
		return fmt.Errorf("failed to get email attachments: %w", err)
	}

	// 并发下载所有附件
	var wg sync.WaitGroup
	errChan := make(chan error, len(attachments))

	for _, attachment := range attachments {
		wg.Add(1)
		go func(att models.Attachment) {
			defer wg.Done()
			if err := s.DownloadAttachment(ctx, att.ID, userID); err != nil {
				errChan <- fmt.Errorf("failed to download attachment %d: %w", att.ID, err)
			}
		}(attachment)
	}

	wg.Wait()
	close(errChan)

	// 收集错误
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to download %d attachments: %v", len(errors), errors[0])
	}

	return nil
}

// GetAttachmentContent 获取附件内容
func (s *AttachmentService) GetAttachmentContent(ctx context.Context, attachmentID uint, userID uint) (io.ReadCloser, error) {
	// 获取附件信息
	attachment, err := s.getAttachmentWithPermissionCheck(ctx, attachmentID, userID)
	if err != nil {
		return nil, err
	}

	// 检查是否已下载
	if !attachment.IsDownloaded || !s.storage.Exists(ctx, attachment) {
		// 尝试下载
		if err := s.DownloadAttachment(ctx, attachmentID, userID); err != nil {
			return nil, fmt.Errorf("failed to download attachment before retrieval: %w", err)
		}
	}

	// 获取内容
	return s.storage.Retrieve(ctx, attachment)
}

// PreviewAttachment 预览附件
func (s *AttachmentService) PreviewAttachment(ctx context.Context, attachmentID uint, userID uint) (*AttachmentPreview, error) {
	// 获取附件信息
	attachment, err := s.getAttachmentWithPermissionCheck(ctx, attachmentID, userID)
	if err != nil {
		return nil, err
	}

	preview := &AttachmentPreview{
		AttachmentID: attachmentID,
		Type:         s.getPreviewType(attachment.ContentType),
	}

	// 如果附件未下载，先下载
	if !attachment.IsDownloaded || !s.storage.Exists(ctx, attachment) {
		if err := s.DownloadAttachment(ctx, attachmentID, userID); err != nil {
			preview.Error = fmt.Sprintf("Failed to download attachment: %v", err)
			return preview, nil
		}
	}

	// 生成预览内容
	if err := s.generatePreviewContent(ctx, attachment, preview); err != nil {
		preview.Error = fmt.Sprintf("Failed to generate preview: %v", err)
	}

	return preview, nil
}

// GetDownloadProgress 获取下载进度
func (s *AttachmentService) GetDownloadProgress(ctx context.Context, attachmentID uint) (*DownloadProgress, error) {
	s.progressMutex.RLock()
	defer s.progressMutex.RUnlock()

	if progress, exists := s.downloadProgress[attachmentID]; exists {
		return progress, nil
	}

	return &DownloadProgress{
		AttachmentID: attachmentID,
		Status:       "not_started",
		Progress:     0.0,
	}, nil
}

// downloadAttachmentContent 下载附件内容的核心逻辑
func (s *AttachmentService) downloadAttachmentContent(ctx context.Context, attachment *models.Attachment, progress *DownloadProgress) error {
	// 获取邮件信息
	var email models.Email
	if err := s.db.WithContext(ctx).Preload("Account").First(&email, attachment.EmailID).Error; err != nil {
		return fmt.Errorf("failed to get email: %w", err)
	}

	// 创建提供商实例
	provider, err := s.providerFactory.CreateProviderForAccount(&email.Account)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	// 连接到服务器
	if err := provider.Connect(ctx, &email.Account); err != nil {
		return fmt.Errorf("failed to connect to provider: %w", err)
	}
	defer provider.Disconnect()

	// 获取IMAP客户端
	imapClient := provider.IMAPClient()
	if imapClient == nil {
		return fmt.Errorf("IMAP client not available")
	}

	// 获取文件夹信息
	var folder models.Folder
	if email.FolderID != nil {
		if err := s.db.WithContext(ctx).First(&folder, *email.FolderID).Error; err != nil {
			return fmt.Errorf("failed to get folder: %w", err)
		}
	}

	// 下载附件内容
	attachmentData, err := imapClient.GetAttachment(ctx, folder.Path, email.UID, attachment.PartID)
	if err != nil {
		return fmt.Errorf("failed to get attachment from IMAP: %w", err)
	}
	defer attachmentData.Close()

	// 读取所有原始数据到内存
	rawData, err := io.ReadAll(attachmentData)
	if err != nil {
		return fmt.Errorf("failed to read attachment data: %w", err)
	}

	// 解码附件数据
	// 使用附件的编码信息进行解码，如果解码失败则使用原始数据
	decodedData, err := transfer.DecodeWithFallback(rawData, attachment.Encoding)
	if err != nil {
		log.Printf("Warning: Failed to decode attachment %d with encoding %s: %v, using raw data", attachment.ID, attachment.Encoding, err)
		decodedData = rawData
	}

	// 更新附件大小为解码后的实际大小
	actualSize := int64(len(decodedData))
	if actualSize != attachment.Size {
		log.Printf("Attachment %d size changed after decoding: %d -> %d (encoding: %s)",
			attachment.ID, attachment.Size, actualSize, attachment.Encoding)
		// 更新进度跟踪的总大小
		progress.BytesTotal = actualSize
	}

	// 创建解码后数据的Reader
	decodedReader := bytes.NewReader(decodedData)

	// 创建进度跟踪的Reader
	progressReader := &progressReader{
		reader:   decodedReader,
		progress: progress,
		service:  s,
	}

	// 存储解码后的附件数据
	if err := s.storage.Store(ctx, attachment, progressReader); err != nil {
		return fmt.Errorf("failed to store attachment: %w", err)
	}

	// 更新数据库（只更新必要字段，避免触发器递归）
	return s.db.WithContext(ctx).Model(attachment).Updates(map[string]interface{}{
		"file_path":      attachment.StoragePath,
		"is_downloaded":  attachment.IsDownloaded,
	}).Error
}

// getAttachmentWithPermissionCheck 获取附件并检查权限
func (s *AttachmentService) getAttachmentWithPermissionCheck(ctx context.Context, attachmentID uint, userID uint) (*models.Attachment, error) {
	var attachment models.Attachment

	// 首先尝试查找临时附件（email_id为NULL）
	err := s.db.WithContext(ctx).
		Where("id = ? AND email_id IS NULL AND user_id = ?", attachmentID, userID).
		First(&attachment).Error

	if err == nil {
		// 找到临时附件，直接返回
		return &attachment, nil
	}

	if err != gorm.ErrRecordNotFound {
		// 数据库查询错误
		return nil, fmt.Errorf("failed to query temporary attachment: %w", err)
	}

	// 没有找到临时附件，尝试查找正常附件（通过邮件账户权限）
	err = s.db.WithContext(ctx).
		Preload("Email").
		Preload("Email.Account").
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

	return &attachment, nil
}

// GetStorage 获取存储接口（用于外部访问）
func (s *AttachmentService) GetStorage() AttachmentStorage {
	return s.storage
}

// GetAttachmentWithPermissionCheck 获取附件并检查权限（公开方法）
func (s *AttachmentService) GetAttachmentWithPermissionCheck(ctx context.Context, attachmentID uint, userID uint) (*models.Attachment, error) {
	return s.getAttachmentWithPermissionCheck(ctx, attachmentID, userID)
}

// setDownloadProgress 设置下载进度
func (s *AttachmentService) setDownloadProgress(attachmentID uint, progress *DownloadProgress) {
	s.progressMutex.Lock()
	defer s.progressMutex.Unlock()
	s.downloadProgress[attachmentID] = progress
}

// getPreviewType 获取预览类型
func (s *AttachmentService) getPreviewType(contentType string) string {
	switch {
	case strings.HasPrefix(contentType, "image/"):
		return "image"
	case strings.HasPrefix(contentType, "text/"):
		return "text"
	case contentType == "application/pdf":
		return "pdf"
	default:
		return "unknown"
	}
}

// generatePreviewContent 生成预览内容
func (s *AttachmentService) generatePreviewContent(ctx context.Context, attachment *models.Attachment, preview *AttachmentPreview) error {
	// 获取文件内容
	content, err := s.storage.Retrieve(ctx, attachment)
	if err != nil {
		return err
	}
	defer content.Close()

	switch preview.Type {
	case "image":
		return s.generateImagePreview(content, preview)
	case "text":
		return s.generateTextPreview(content, preview)
	case "pdf":
		return s.generatePDFPreview(content, preview)
	default:
		return nil // 不支持的类型
	}
}

// generateImagePreview 生成图片预览
func (s *AttachmentService) generateImagePreview(content io.Reader, preview *AttachmentPreview) error {
	// 暂时简单实现：读取前1KB作为预览
	buffer := make([]byte, 1024)
	n, _ := content.Read(buffer)
	preview.Content = buffer[:n]
	return nil
}

// generateTextPreview 生成文本预览
func (s *AttachmentService) generateTextPreview(content io.Reader, preview *AttachmentPreview) error {
	// 读取前1KB文本内容
	buffer := make([]byte, 1024)
	n, _ := content.Read(buffer)
	preview.Text = string(buffer[:n])
	return nil
}

// generatePDFPreview 生成PDF预览
func (s *AttachmentService) generatePDFPreview(content io.Reader, preview *AttachmentPreview) error {
	// PDF预览需要专门的库，暂时返回文件信息
	preview.Text = "PDF document preview not implemented"
	return nil
}

// progressReader 带进度跟踪的Reader
type progressReader struct {
	reader   io.Reader
	progress *DownloadProgress
	service  *AttachmentService
}

func (pr *progressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	if n > 0 {
		pr.progress.BytesLoaded += int64(n)
		if pr.progress.BytesTotal > 0 {
			pr.progress.Progress = float64(pr.progress.BytesLoaded) / float64(pr.progress.BytesTotal)
		}
		pr.service.setDownloadProgress(pr.progress.AttachmentID, pr.progress)
	}
	return n, err
}

// CleanupTemporaryAttachments 清理临时附件
func (s *AttachmentService) CleanupTemporaryAttachments(ctx context.Context, maxAgeHours int) error {
	if maxAgeHours <= 0 {
		return fmt.Errorf("max age hours must be positive")
	}

	cutoffTime := time.Now().Add(-time.Duration(maxAgeHours) * time.Hour)
	log.Printf("Cleaning up temporary attachments older than %d hours (before %s)", maxAgeHours, cutoffTime.Format("2006-01-02 15:04:05"))

	// 查询需要清理的临时附件
	var tempAttachments []models.Attachment
	err := s.db.WithContext(ctx).
		Where("email_id IS NULL AND created_at < ?", cutoffTime).
		Find(&tempAttachments).Error

	if err != nil {
		return fmt.Errorf("failed to query temporary attachments: %w", err)
	}

	if len(tempAttachments) == 0 {
		log.Printf("No temporary attachments to clean up")
		return nil
	}

	log.Printf("Found %d temporary attachments to clean up", len(tempAttachments))

	// 清理每个临时附件
	cleanedCount := 0
	for _, attachment := range tempAttachments {
		if err := s.cleanupSingleTemporaryAttachment(ctx, &attachment); err != nil {
			log.Printf("Warning: failed to cleanup temporary attachment %d: %v", attachment.ID, err)
			continue
		}
		cleanedCount++
	}

	log.Printf("Temporary attachment cleanup completed: %d/%d attachments cleaned up", cleanedCount, len(tempAttachments))
	return nil
}

// cleanupSingleTemporaryAttachment 清理单个临时附件
func (s *AttachmentService) cleanupSingleTemporaryAttachment(ctx context.Context, attachment *models.Attachment) error {
	// 删除存储文件
	if attachment.StoragePath != "" {
		if err := s.storage.Delete(ctx, attachment); err != nil {
			log.Printf("Warning: failed to delete attachment file %s: %v", attachment.StoragePath, err)
			// 继续删除数据库记录，即使文件删除失败
		}
	}

	// 删除数据库记录
	if err := s.db.WithContext(ctx).Delete(attachment).Error; err != nil {
		return fmt.Errorf("failed to delete attachment record: %w", err)
	}

	log.Printf("Cleaned up temporary attachment %d (%s)", attachment.ID, attachment.Filename)
	return nil
}

// StartAutoCleanup 启动自动清理临时附件
func (s *AttachmentService) StartAutoCleanup(ctx context.Context, maxAgeHours int) error {
	if maxAgeHours <= 0 {
		return fmt.Errorf("max age hours must be positive")
	}

	log.Printf("Starting automatic temporary attachment cleanup service (max age: %d hours)...", maxAgeHours)

	go func() {
		// 每天执行一次清理
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				log.Println("Running scheduled temporary attachment cleanup...")
				if err := s.CleanupTemporaryAttachments(ctx, maxAgeHours); err != nil {
					log.Printf("Scheduled temporary attachment cleanup failed: %v", err)
				}
			case <-s.cleanupStopChan:
				log.Println("Stopping automatic temporary attachment cleanup service...")
				return
			case <-ctx.Done():
				log.Println("Context cancelled, stopping automatic temporary attachment cleanup service...")
				return
			}
		}
	}()

	return nil
}

// StopAutoCleanup 停止自动清理
func (s *AttachmentService) StopAutoCleanup() {
	if s.cleanupStopChan != nil {
		close(s.cleanupStopChan)
	}
}
