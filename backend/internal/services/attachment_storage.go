package services

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"firemail/internal/models"
)

// AttachmentStorage 附件存储接口
type AttachmentStorage interface {
	// Store 存储附件数据
	Store(ctx context.Context, attachment *models.Attachment, data io.Reader) error
	
	// Retrieve 获取附件数据
	Retrieve(ctx context.Context, attachment *models.Attachment) (io.ReadCloser, error)
	
	// Delete 删除附件
	Delete(ctx context.Context, attachment *models.Attachment) error
	
	// Exists 检查附件是否存在
	Exists(ctx context.Context, attachment *models.Attachment) bool
	
	// GetStoragePath 获取存储路径
	GetStoragePath(attachment *models.Attachment) string
	
	// GetStorageInfo 获取存储信息
	GetStorageInfo(ctx context.Context, attachment *models.Attachment) (*StorageInfo, error)
}

// StorageInfo 存储信息
type StorageInfo struct {
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"mod_time"`
	Checksum     string    `json:"checksum"`
	IsCompressed bool      `json:"is_compressed"`
}

// AttachmentStorageConfig 附件存储配置
type AttachmentStorageConfig struct {
	BaseDir       string `json:"base_dir"`
	MaxFileSize   int64  `json:"max_file_size"`   // 最大文件大小（字节）
	CompressText  bool   `json:"compress_text"`   // 是否压缩文本文件
	CreateDirs    bool   `json:"create_dirs"`     // 是否自动创建目录
	ChecksumType  string `json:"checksum_type"`   // 校验和类型
}

// LocalFileStorage 本地文件存储实现
type LocalFileStorage struct {
	config *AttachmentStorageConfig
}

// NewLocalFileStorage 创建本地文件存储
func NewLocalFileStorage(config *AttachmentStorageConfig) AttachmentStorage {
	if config == nil {
		config = &AttachmentStorageConfig{
			BaseDir:      "attachments",
			MaxFileSize:  100 * 1024 * 1024, // 100MB
			CompressText: true,
			CreateDirs:   true,
			ChecksumType: "md5",
		}
	}
	
	return &LocalFileStorage{
		config: config,
	}
}

// Store 存储附件数据
func (s *LocalFileStorage) Store(ctx context.Context, attachment *models.Attachment, data io.Reader) error {
	// 检查文件大小限制
	if attachment.Size > s.config.MaxFileSize {
		return fmt.Errorf("file size %d exceeds maximum allowed size %d", attachment.Size, s.config.MaxFileSize)
	}

	// 获取存储路径
	storagePath := s.GetStoragePath(attachment)
	
	// 创建目录
	if s.config.CreateDirs {
		dir := filepath.Dir(storagePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// 创建临时文件
	tempPath := storagePath + ".tmp"
	tempFile, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		tempFile.Close()
		os.Remove(tempPath) // 清理临时文件
	}()

	// 复制数据并计算校验和
	hasher := md5.New()
	multiWriter := io.MultiWriter(tempFile, hasher)
	
	written, err := io.Copy(multiWriter, data)
	if err != nil {
		return fmt.Errorf("failed to write attachment data: %w", err)
	}

	// 验证文件大小 - 放宽验证以处理IMAP编码格式差异
	// IMAP返回的数据可能包含格式字符（如base64换行符），导致大小不完全匹配
	// 只在差异过大时报错，允许合理的编码格式差异
	sizeDiff := written - attachment.Size
	if sizeDiff < 0 {
		sizeDiff = -sizeDiff
	}
	// 允许10%的大小差异或最多5KB的差异（以较大者为准）
	maxAllowedDiff := attachment.Size / 10
	if maxAllowedDiff < 5120 { // 5KB
		maxAllowedDiff = 5120
	}
	if sizeDiff > maxAllowedDiff {
		return fmt.Errorf("size mismatch too large: expected %d, got %d (diff: %d, max allowed: %d)",
			attachment.Size, written, sizeDiff, maxAllowedDiff)
	}

	// 关闭临时文件
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// 原子性移动到最终位置
	if err := os.Rename(tempPath, storagePath); err != nil {
		return fmt.Errorf("failed to move temp file to final location: %w", err)
	}

	// 更新附件存储信息
	attachment.StoragePath = storagePath
	attachment.IsDownloaded = true
	// 注意：Checksum字段暂时不在模型中，可以在需要时添加到数据库

	return nil
}

// Retrieve 获取附件数据
func (s *LocalFileStorage) Retrieve(ctx context.Context, attachment *models.Attachment) (io.ReadCloser, error) {
	storagePath := s.GetStoragePath(attachment)
	
	// 检查文件是否存在
	if !s.Exists(ctx, attachment) {
		return nil, fmt.Errorf("attachment file not found: %s", storagePath)
	}

	// 打开文件
	file, err := os.Open(storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open attachment file: %w", err)
	}

	return file, nil
}

// Delete 删除附件
func (s *LocalFileStorage) Delete(ctx context.Context, attachment *models.Attachment) error {
	storagePath := s.GetStoragePath(attachment)
	
	// 删除文件
	if err := os.Remove(storagePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete attachment file: %w", err)
	}

	// 尝试删除空目录
	dir := filepath.Dir(storagePath)
	os.Remove(dir) // 忽略错误，因为目录可能不为空

	// 更新附件状态
	attachment.StoragePath = ""
	attachment.IsDownloaded = false

	return nil
}

// Exists 检查附件是否存在
func (s *LocalFileStorage) Exists(ctx context.Context, attachment *models.Attachment) bool {
	storagePath := s.GetStoragePath(attachment)
	
	// 检查文件是否存在
	if _, err := os.Stat(storagePath); os.IsNotExist(err) {
		return false
	}

	return true
}

// GetStoragePath 获取存储路径
func (s *LocalFileStorage) GetStoragePath(attachment *models.Attachment) string {
	// 如果已有存储路径，直接返回
	if attachment.StoragePath != "" {
		return attachment.StoragePath
	}

	// 生成安全的文件名
	safeFilename := s.sanitizeFilename(attachment.Filename)

	// 检查是否为临时附件（EmailID为nil）
	if attachment.EmailID == nil {
		// 临时附件使用特殊路径，避免空指针异常
		return filepath.Join(
			s.config.BaseDir,
			"temp",
			fmt.Sprintf("attachment_%d_%s", attachment.ID, safeFilename),
		)
	}

	// 正常附件需要检查Email关联是否存在
	if attachment.Email.ID == 0 {
		// 如果Email关联为空，使用EmailID构建路径
		return filepath.Join(
			s.config.BaseDir,
			"orphaned",
			fmt.Sprintf("email_%d", *attachment.EmailID),
			fmt.Sprintf("attachment_%d_%s", attachment.ID, safeFilename),
		)
	}

	// 原有的路径生成逻辑（正常附件）
	return filepath.Join(
		s.config.BaseDir,
		fmt.Sprintf("account_%d", attachment.Email.AccountID),
		fmt.Sprintf("email_%d", *attachment.EmailID),
		fmt.Sprintf("attachment_%d_%s", attachment.ID, safeFilename),
	)
}

// GetStorageInfo 获取存储信息
func (s *LocalFileStorage) GetStorageInfo(ctx context.Context, attachment *models.Attachment) (*StorageInfo, error) {
	storagePath := s.GetStoragePath(attachment)
	
	// 获取文件信息
	fileInfo, err := os.Stat(storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// 计算校验和
	checksum, err := s.calculateChecksum(storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	return &StorageInfo{
		Path:         storagePath,
		Size:         fileInfo.Size(),
		ModTime:      fileInfo.ModTime(),
		Checksum:     checksum,
		IsCompressed: false, // 暂时不支持压缩
	}, nil
}

// sanitizeFilename 清理文件名，确保安全
func (s *LocalFileStorage) sanitizeFilename(filename string) string {
	// 移除路径分隔符和特殊字符
	reg := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	safe := reg.ReplaceAllString(filename, "_")
	
	// 限制文件名长度
	if len(safe) > 200 {
		ext := filepath.Ext(safe)
		name := strings.TrimSuffix(safe, ext)
		if len(name) > 200-len(ext) {
			name = name[:200-len(ext)]
		}
		safe = name + ext
	}
	
	// 确保不为空
	if safe == "" {
		safe = "unnamed_file"
	}
	
	return safe
}

// calculateChecksum 计算文件校验和
func (s *LocalFileStorage) calculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := md5.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// AttachmentStorageError 附件存储错误
type AttachmentStorageError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
}

func (e *AttachmentStorageError) Error() string {
	return fmt.Sprintf("attachment storage error [%s]: %s", e.Type, e.Message)
}

// 预定义错误类型
var (
	ErrAttachmentNotFound     = &AttachmentStorageError{Type: "not_found", Message: "attachment not found"}
	ErrAttachmentTooLarge     = &AttachmentStorageError{Type: "too_large", Message: "attachment too large"}
	ErrStoragePermission      = &AttachmentStorageError{Type: "permission", Message: "storage permission denied"}
	ErrStorageSpace           = &AttachmentStorageError{Type: "space", Message: "insufficient storage space"}
	ErrChecksumMismatch       = &AttachmentStorageError{Type: "checksum", Message: "checksum mismatch"}
)
