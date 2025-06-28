package models

import (
	"fmt"
)

// Attachment 附件模型
type Attachment struct {
	BaseModel
	EmailID     *uint  `gorm:"index" json:"email_id,omitempty"`                 // 允许为空，用于临时上传的附件
	UserID      *uint  `gorm:"index" json:"user_id,omitempty"`                  // 用于临时附件的用户权限检查
	Filename    string `gorm:"not null;size:255" json:"filename"`
	ContentType string `gorm:"size:100" json:"content_type"`
	Size        int64  `gorm:"not null" json:"size"`
	ContentID   string `gorm:"size:255" json:"content_id,omitempty"`            // 用于内联附件
	Disposition string `gorm:"size:20;default:'attachment'" json:"disposition"` // attachment, inline

	// 存储信息
	StoragePath  string `gorm:"column:file_path;size:500" json:"storage_path,omitempty"` // 本地存储路径
	IsDownloaded bool   `gorm:"column:is_downloaded;not null;default:false" json:"is_downloaded"` // 是否已下载到本地
	IsInline     bool   `gorm:"column:is_inline;not null;default:false" json:"is_inline"` // 是否为内联附件
	Encoding     string `gorm:"size:50;not null;default:'7bit'" json:"encoding"` // 传输编码类型：base64, quoted-printable, 7bit, 8bit等

	// IMAP信息
	PartID string `gorm:"column:part_id;size:50" json:"part_id"` // IMAP part ID，用于从IMAP服务器下载附件

	// 关联关系
	Email Email `gorm:"foreignKey:EmailID" json:"email,omitempty"`
}

// TableName 指定表名
func (Attachment) TableName() string {
	return "attachments"
}

// IsInlineAttachment 检查是否为内联附件（基于disposition和content_id）
func (a *Attachment) IsInlineAttachment() bool {
	return a.Disposition == "inline" || a.ContentID != ""
}

// GetFileExtension 获取文件扩展名
func (a *Attachment) GetFileExtension() string {
	if a.Filename == "" {
		return ""
	}

	for i := len(a.Filename) - 1; i >= 0; i-- {
		if a.Filename[i] == '.' {
			return a.Filename[i+1:]
		}
	}
	return ""
}

// IsImage 检查是否为图片文件
func (a *Attachment) IsImage() bool {
	imageTypes := []string{
		"image/jpeg", "image/jpg", "image/png", "image/gif",
		"image/bmp", "image/webp", "image/svg+xml",
	}

	for _, imgType := range imageTypes {
		if a.ContentType == imgType {
			return true
		}
	}
	return false
}

// IsDocument 检查是否为文档文件
func (a *Attachment) IsDocument() bool {
	docTypes := []string{
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.ms-powerpoint",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"text/plain",
		"text/csv",
	}

	for _, docType := range docTypes {
		if a.ContentType == docType {
			return true
		}
	}
	return false
}

// GetHumanReadableSize 获取人类可读的文件大小
func (a *Attachment) GetHumanReadableSize() string {
	const unit = 1024
	if a.Size < unit {
		return fmt.Sprintf("%d B", a.Size)
	}

	div, exp := int64(unit), 0
	for n := a.Size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(a.Size)/float64(div), "KMGTPE"[exp])
}
