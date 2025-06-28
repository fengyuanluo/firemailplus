package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"firemail/internal/models"
	"firemail/internal/providers"

	"gorm.io/gorm"
)

// GmailDeduplicator Gmail专用去重器
// Gmail使用标签系统，同一邮件可能出现在多个"文件夹"（标签）中
type GmailDeduplicator struct {
	*StandardDeduplicator
}

// NewGmailDeduplicator 创建Gmail去重器
func NewGmailDeduplicator(db *gorm.DB) EmailDeduplicator {
	return &GmailDeduplicator{
		StandardDeduplicator: &StandardDeduplicator{db: db},
	}
}

// GetProviderType 获取提供商类型
func (d *GmailDeduplicator) GetProviderType() string {
	return "gmail"
}

// CheckDuplicate Gmail特殊的重复检查逻辑
func (d *GmailDeduplicator) CheckDuplicate(ctx context.Context, email *providers.EmailMessage, accountID, folderID uint) (*DuplicateCheckResult, error) {
	// Gmail特殊处理：优先检查MessageID，允许同一邮件在多个标签中存在
	if email.MessageID != "" {
		result, err := d.checkGmailMessageIDDuplicate(ctx, email.MessageID, accountID, folderID)
		if err != nil {
			return nil, fmt.Errorf("failed to check Gmail message ID duplicate: %w", err)
		}
		if result.IsDuplicate {
			return result, nil
		}
	}

	// 检查UID重复（在同一文件夹内）
	result, err := d.checkUIDDuplicate(ctx, email.UID, accountID, folderID)
	if err != nil {
		return nil, fmt.Errorf("failed to check UID duplicate: %w", err)
	}
	if result.IsDuplicate {
		return result, nil
	}

	// Gmail很少有MessageID为空的情况，但仍然提供内容检查
	if email.MessageID == "" {
		result, err := d.checkContentSimilarity(ctx, email, accountID, folderID)
		if err != nil {
			log.Printf("Warning: Gmail content similarity check failed: %v", err)
		} else if result.IsDuplicate {
			return result, nil
		}
	}

	return &DuplicateCheckResult{
		IsDuplicate: false,
		Action:      "create",
		Reason:      "No duplicate found in Gmail",
	}, nil
}

// checkGmailMessageIDDuplicate Gmail特殊的MessageID重复检查
func (d *GmailDeduplicator) checkGmailMessageIDDuplicate(ctx context.Context, messageID string, accountID, folderID uint) (*DuplicateCheckResult, error) {
	// 检查上下文是否已取消，如果是则创建新的上下文
	if err := ctx.Err(); err != nil {
		newCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ctx = newCtx
		log.Printf("Original context canceled, using new context for Gmail duplicate check")
	}

	var existing models.Email
	err := d.db.WithContext(ctx).
		Where("account_id = ? AND message_id = ?", accountID, messageID).
		First(&existing).Error

	if err == nil {
		// Gmail特殊处理：检查是否在同一标签中
		if existing.FolderID != nil && *existing.FolderID == folderID {
			// 完全重复，跳过
			return &DuplicateCheckResult{
				IsDuplicate:   true,
				ExistingEmail: &existing,
				ConflictType:  "message_id",
				Action:        "skip",
				Reason:        "Email already exists in the same Gmail label",
			}, nil
		}

		// 同一邮件在不同标签中，这在Gmail中是正常的
		// 但我们需要创建一个新的记录来表示这个标签关系
		return &DuplicateCheckResult{
			IsDuplicate:   true,
			ExistingEmail: &existing,
			ConflictType:  "message_id",
			Action:        "create_label_reference",
			Reason:        "Same email in different Gmail label, creating label reference",
		}, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	return &DuplicateCheckResult{IsDuplicate: false}, nil
}

// HandleDuplicate Gmail特殊的重复处理逻辑
func (d *GmailDeduplicator) HandleDuplicate(ctx context.Context, existing *models.Email, new *providers.EmailMessage, folderID uint) error {
	// 检查处理动作
	result, err := d.checkGmailMessageIDDuplicate(ctx, new.MessageID, existing.AccountID, folderID)
	if err != nil {
		return fmt.Errorf("failed to determine duplicate action: %w", err)
	}

	switch result.Action {
	case "skip":
		// 完全重复，不需要任何操作
		return nil

	case "create_label_reference":
		// Gmail标签系统：同一邮件在不同标签中
		return d.createGmailLabelReference(ctx, existing, new, folderID)

	case "update":
		// 更新现有邮件信息
		return d.updateExistingGmailEmail(ctx, existing, new, folderID)

	default:
		// 默认使用标准处理逻辑
		return d.StandardDeduplicator.HandleDuplicate(ctx, existing, new, folderID)
	}
}

// createGmailLabelReference 为Gmail创建标签引用
func (d *GmailDeduplicator) createGmailLabelReference(ctx context.Context, existing *models.Email, new *providers.EmailMessage, folderID uint) error {
	// 在Gmail中，同一邮件可能出现在多个标签中
	// 我们创建一个新的记录来表示这个标签关系，但共享相同的MessageID
	
	// 首先检查是否已经有这个标签的引用
	var labelRef models.Email
	err := d.db.WithContext(ctx).
		Where("account_id = ? AND message_id = ? AND folder_id = ?", 
			existing.AccountID, existing.MessageID, folderID).
		First(&labelRef).Error

	if err == nil {
		// 标签引用已存在，更新状态
		return d.updateExistingGmailEmail(ctx, &labelRef, new, folderID)
	}

	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check label reference: %w", err)
	}

	// 创建新的标签引用
	labelEmail := &models.Email{
		AccountID:     existing.AccountID,
		FolderID:      &folderID,
		MessageID:     existing.MessageID,
		UID:           new.UID,
		Subject:       existing.Subject,
		From:          existing.From,
		Date:          existing.Date,
		TextBody:      existing.TextBody,
		HTMLBody:      existing.HTMLBody,
		Size:          existing.Size,
		IsRead:        d.isEmailRead(new.Flags),
		IsStarred:     d.isEmailStarred(new.Flags),
		IsDraft:       d.isEmailDraft(new.Flags),
		HasAttachment: existing.HasAttachment,
		Priority:      existing.Priority,
	}

	// 复制邮件地址信息
	if err := d.copyEmailAddresses(existing, labelEmail); err != nil {
		log.Printf("Warning: failed to copy email addresses: %v", err)
	}

	// 更新Gmail标签信息
	if err := d.updateGmailLabels(ctx, labelEmail, new); err != nil {
		log.Printf("Warning: failed to update Gmail labels: %v", err)
	}

	return d.db.WithContext(ctx).Create(labelEmail).Error
}

// updateExistingGmailEmail 更新现有Gmail邮件
func (d *GmailDeduplicator) updateExistingGmailEmail(ctx context.Context, existing *models.Email, new *providers.EmailMessage, folderID uint) error {
	// 更新文件夹信息
	existing.FolderID = &folderID
	existing.UID = new.UID

	// 更新邮件状态
	existing.IsRead = d.isEmailRead(new.Flags)
	existing.IsStarred = d.isEmailStarred(new.Flags)
	existing.IsDraft = d.isEmailDraft(new.Flags)

	// 更新Gmail标签信息
	if err := d.updateGmailLabels(ctx, existing, new); err != nil {
		log.Printf("Warning: failed to update Gmail labels: %v", err)
	}

	return d.db.WithContext(ctx).Save(existing).Error
}

// copyEmailAddresses 复制邮件地址信息
func (d *GmailDeduplicator) copyEmailAddresses(source, target *models.Email) error {
	target.To = source.To
	target.CC = source.CC
	target.BCC = source.BCC
	target.ReplyTo = source.ReplyTo
	return nil
}

// updateGmailLabels 更新Gmail标签信息
func (d *GmailDeduplicator) updateGmailLabels(ctx context.Context, email *models.Email, new *providers.EmailMessage) error {
	// Gmail标签处理逻辑
	// 这里可以根据需要实现Gmail特有的标签处理
	
	// 获取当前标签
	currentLabels, err := email.GetLabels()
	if err != nil {
		currentLabels = []string{}
	}

	// 从新邮件中提取标签信息（如果有的话）
	// 这需要根据Gmail IMAP的具体实现来处理
	newLabels := d.extractGmailLabels(new)

	// 合并标签
	mergedLabels := d.mergeLabels(currentLabels, newLabels)

	// 更新标签
	return email.SetLabels(mergedLabels)
}

// extractGmailLabels 从Gmail邮件中提取标签
func (d *GmailDeduplicator) extractGmailLabels(email *providers.EmailMessage) []string {
	// 这里需要根据Gmail IMAP的具体实现来提取标签
	// 暂时返回空数组，后续可以根据需要实现
	return []string{}
}

// mergeLabels 合并标签列表
func (d *GmailDeduplicator) mergeLabels(current, new []string) []string {
	labelSet := make(map[string]bool)
	
	// 添加现有标签
	for _, label := range current {
		labelSet[label] = true
	}
	
	// 添加新标签
	for _, label := range new {
		labelSet[label] = true
	}
	
	// 转换为数组
	var merged []string
	for label := range labelSet {
		merged = append(merged, label)
	}
	
	return merged
}
