package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"firemail/internal/models"
	"firemail/internal/providers"

	"gorm.io/gorm"
)

// DuplicateCheckResult 重复检查结果
type DuplicateCheckResult struct {
	IsDuplicate   bool           `json:"is_duplicate"`
	ExistingEmail *models.Email  `json:"existing_email,omitempty"`
	ConflictType  string         `json:"conflict_type"` // "message_id", "uid", "content"
	Action        string         `json:"action"`        // "skip", "update", "merge"
	Reason        string         `json:"reason"`
}

// EmailDeduplicator 邮件去重接口
type EmailDeduplicator interface {
	// CheckDuplicate 检查邮件是否重复
	CheckDuplicate(ctx context.Context, email *providers.EmailMessage, accountID, folderID uint) (*DuplicateCheckResult, error)
	
	// HandleDuplicate 处理重复邮件
	HandleDuplicate(ctx context.Context, existing *models.Email, new *providers.EmailMessage, folderID uint) error
	
	// GetProviderType 获取支持的提供商类型
	GetProviderType() string
}

// DeduplicatorFactory 去重器工厂接口
type DeduplicatorFactory interface {
	CreateDeduplicator(provider string) EmailDeduplicator
}

// StandardDeduplicatorFactory 标准去重器工厂
type StandardDeduplicatorFactory struct {
	db *gorm.DB
}

// NewDeduplicatorFactory 创建去重器工厂
func NewDeduplicatorFactory(db *gorm.DB) DeduplicatorFactory {
	return &StandardDeduplicatorFactory{
		db: db,
	}
}

// CreateDeduplicator 创建去重器
func (f *StandardDeduplicatorFactory) CreateDeduplicator(provider string) EmailDeduplicator {
	switch strings.ToLower(provider) {
	case "gmail":
		return NewGmailDeduplicator(f.db)
	case "outlook", "hotmail":
		return NewOutlookDeduplicator(f.db)
	default:
		return NewStandardDeduplicator(f.db)
	}
}

// StandardDeduplicator 标准邮件去重器
type StandardDeduplicator struct {
	db *gorm.DB
}

// NewStandardDeduplicator 创建标准去重器
func NewStandardDeduplicator(db *gorm.DB) EmailDeduplicator {
	return &StandardDeduplicator{
		db: db,
	}
}

// GetProviderType 获取提供商类型
func (d *StandardDeduplicator) GetProviderType() string {
	return "standard"
}

// CheckDuplicate 检查邮件是否重复
func (d *StandardDeduplicator) CheckDuplicate(ctx context.Context, email *providers.EmailMessage, accountID, folderID uint) (*DuplicateCheckResult, error) {
	// 1. 首先检查MessageID重复（如果存在）
	if email.MessageID != "" {
		result, err := d.checkMessageIDDuplicate(ctx, email.MessageID, accountID, folderID)
		if err != nil {
			return nil, fmt.Errorf("failed to check message ID duplicate: %w", err)
		}
		if result.IsDuplicate {
			return result, nil
		}
	}

	// 2. 检查UID重复（在同一文件夹内）
	result, err := d.checkUIDDuplicate(ctx, email.UID, accountID, folderID)
	if err != nil {
		return nil, fmt.Errorf("failed to check UID duplicate: %w", err)
	}
	if result.IsDuplicate {
		return result, nil
	}

	// 3. 如果MessageID为空，进行内容相似性检查
	if email.MessageID == "" {
		result, err := d.checkContentSimilarity(ctx, email, accountID, folderID)
		if err != nil {
			log.Printf("Warning: content similarity check failed: %v", err)
			// 内容检查失败不应该阻止邮件保存
		} else if result.IsDuplicate {
			return result, nil
		}
	}

	// 没有发现重复
	return &DuplicateCheckResult{
		IsDuplicate: false,
		Action:      "create",
		Reason:      "No duplicate found",
	}, nil
}

// checkMessageIDDuplicate 检查MessageID重复
func (d *StandardDeduplicator) checkMessageIDDuplicate(ctx context.Context, messageID string, accountID, folderID uint) (*DuplicateCheckResult, error) {
	// 检查上下文是否已取消
	if err := ctx.Err(); err != nil {
		// 如果上下文已取消，创建一个新的带超时的上下文进行必要的数据库操作
		newCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ctx = newCtx
		log.Printf("Original context canceled, using new context for duplicate check")
	}

	var existing models.Email
	err := d.db.WithContext(ctx).
		Where("account_id = ? AND message_id = ?", accountID, messageID).
		First(&existing).Error

	if err == nil {
		// 找到重复邮件
		action := "skip"
		reason := "Email with same MessageID already exists"

		// 如果在不同文件夹，可能需要更新文件夹信息
		if existing.FolderID == nil || *existing.FolderID != folderID {
			action = "update"
			reason = "Email exists in different folder, updating folder reference"
		}

		return &DuplicateCheckResult{
			IsDuplicate:   true,
			ExistingEmail: &existing,
			ConflictType:  "message_id",
			Action:        action,
			Reason:        reason,
		}, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	return &DuplicateCheckResult{IsDuplicate: false}, nil
}

// checkUIDDuplicate 检查UID重复
func (d *StandardDeduplicator) checkUIDDuplicate(ctx context.Context, uid uint32, accountID, folderID uint) (*DuplicateCheckResult, error) {
	// 检查上下文是否已取消
	if err := ctx.Err(); err != nil {
		newCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ctx = newCtx
		log.Printf("Original context canceled, using new context for UID duplicate check")
	}

	var existing models.Email
	err := d.db.WithContext(ctx).
		Where("account_id = ? AND folder_id = ? AND uid = ?", accountID, folderID, uid).
		First(&existing).Error

	if err == nil {
		return &DuplicateCheckResult{
			IsDuplicate:   true,
			ExistingEmail: &existing,
			ConflictType:  "uid",
			Action:        "update",
			Reason:        "Email with same UID exists in folder, updating content",
		}, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	return &DuplicateCheckResult{IsDuplicate: false}, nil
}

// checkContentSimilarity 检查内容相似性（用于MessageID为空的情况）
func (d *StandardDeduplicator) checkContentSimilarity(ctx context.Context, email *providers.EmailMessage, accountID, folderID uint) (*DuplicateCheckResult, error) {
	// 基于主题、发件人、日期进行相似性检查
	var existing models.Email
	err := d.db.WithContext(ctx).
		Where("account_id = ? AND subject = ? AND from_address = ? AND ABS(julianday(date) - julianday(?)) < 1",
			accountID, email.Subject, email.From.Address, email.Date).
		First(&existing).Error

	if err == nil {
		return &DuplicateCheckResult{
			IsDuplicate:   true,
			ExistingEmail: &existing,
			ConflictType:  "content",
			Action:        "skip",
			Reason:        "Similar email found based on subject, sender and date",
		}, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	return &DuplicateCheckResult{IsDuplicate: false}, nil
}

// HandleDuplicate 处理重复邮件
func (d *StandardDeduplicator) HandleDuplicate(ctx context.Context, existing *models.Email, new *providers.EmailMessage, folderID uint) error {
	// 检查上下文是否已取消
	if err := ctx.Err(); err != nil {
		newCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ctx = newCtx
		log.Printf("Original context canceled, using new context for duplicate handling")
	}

	switch {
	case existing.FolderID == nil || *existing.FolderID != folderID:
		// 更新文件夹信息
		existing.FolderID = &folderID
		return d.db.WithContext(ctx).Save(existing).Error

	case existing.MessageID == "" && new.MessageID != "":
		// 补充MessageID信息
		existing.MessageID = new.MessageID
		return d.db.WithContext(ctx).Save(existing).Error

	default:
		// 更新邮件状态（如已读状态等）
		existing.IsRead = d.isEmailRead(new.Flags)
		existing.IsStarred = d.isEmailStarred(new.Flags)
		existing.IsDraft = d.isEmailDraft(new.Flags)
		return d.db.WithContext(ctx).Save(existing).Error
	}
}

// 辅助方法：检查邮件标志
func (d *StandardDeduplicator) isEmailRead(flags []string) bool {
	for _, flag := range flags {
		if flag == "\\Seen" {
			return true
		}
	}
	return false
}

func (d *StandardDeduplicator) isEmailStarred(flags []string) bool {
	for _, flag := range flags {
		if flag == "\\Flagged" {
			return true
		}
	}
	return false
}

func (d *StandardDeduplicator) isEmailDraft(flags []string) bool {
	for _, flag := range flags {
		if flag == "\\Draft" {
			return true
		}
	}
	return false
}
