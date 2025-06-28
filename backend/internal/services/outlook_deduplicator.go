package services

import (
	"context"
	"fmt"
	"log"
	"strings"

	"firemail/internal/models"
	"firemail/internal/providers"

	"gorm.io/gorm"
)

// OutlookDeduplicator Outlook专用去重器
// Outlook/Exchange有一些特殊的邮件处理逻辑
type OutlookDeduplicator struct {
	*StandardDeduplicator
}

// NewOutlookDeduplicator 创建Outlook去重器
func NewOutlookDeduplicator(db *gorm.DB) EmailDeduplicator {
	return &OutlookDeduplicator{
		StandardDeduplicator: &StandardDeduplicator{db: db},
	}
}

// GetProviderType 获取提供商类型
func (d *OutlookDeduplicator) GetProviderType() string {
	return "outlook"
}

// CheckDuplicate Outlook特殊的重复检查逻辑
func (d *OutlookDeduplicator) CheckDuplicate(ctx context.Context, email *providers.EmailMessage, accountID, folderID uint) (*DuplicateCheckResult, error) {
	// Outlook特殊处理：检查Exchange特有的邮件ID
	if email.MessageID != "" {
		result, err := d.checkOutlookMessageIDDuplicate(ctx, email.MessageID, accountID, folderID)
		if err != nil {
			return nil, fmt.Errorf("failed to check Outlook message ID duplicate: %w", err)
		}
		if result.IsDuplicate {
			return result, nil
		}
	}

	// 检查Exchange特有的ConversationID（如果有的话）
	result, err := d.checkConversationIDDuplicate(ctx, email, accountID, folderID)
	if err != nil {
		log.Printf("Warning: Outlook conversation ID check failed: %v", err)
	} else if result.IsDuplicate {
		return result, nil
	}

	// 检查UID重复
	result, err = d.checkUIDDuplicate(ctx, email.UID, accountID, folderID)
	if err != nil {
		return nil, fmt.Errorf("failed to check UID duplicate: %w", err)
	}
	if result.IsDuplicate {
		return result, nil
	}

	// 内容相似性检查
	if email.MessageID == "" {
		result, err := d.checkContentSimilarity(ctx, email, accountID, folderID)
		if err != nil {
			log.Printf("Warning: Outlook content similarity check failed: %v", err)
		} else if result.IsDuplicate {
			return result, nil
		}
	}

	return &DuplicateCheckResult{
		IsDuplicate: false,
		Action:      "create",
		Reason:      "No duplicate found in Outlook",
	}, nil
}

// checkOutlookMessageIDDuplicate Outlook特殊的MessageID检查
func (d *OutlookDeduplicator) checkOutlookMessageIDDuplicate(ctx context.Context, messageID string, accountID, folderID uint) (*DuplicateCheckResult, error) {
	// Outlook/Exchange的MessageID可能有特殊格式
	normalizedMessageID := d.normalizeOutlookMessageID(messageID)

	var existing models.Email
	err := d.db.WithContext(ctx).
		Where("account_id = ? AND (message_id = ? OR message_id = ?)", 
			accountID, messageID, normalizedMessageID).
		First(&existing).Error

	if err == nil {
		action := "skip"
		reason := "Email with same MessageID already exists in Outlook"
		
		// 检查是否在不同文件夹
		if existing.FolderID == nil || *existing.FolderID != folderID {
			action = "update"
			reason = "Email exists in different Outlook folder, updating folder reference"
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

// checkConversationIDDuplicate 检查Exchange会话ID重复
func (d *OutlookDeduplicator) checkConversationIDDuplicate(ctx context.Context, email *providers.EmailMessage, accountID, folderID uint) (*DuplicateCheckResult, error) {
	// 从邮件头中提取ConversationID
	conversationID := d.extractConversationID(email)
	if conversationID == "" {
		return &DuplicateCheckResult{IsDuplicate: false}, nil
	}

	// 检查是否有相同会话ID的邮件
	var existing models.Email
	err := d.db.WithContext(ctx).
		Where("account_id = ? AND subject = ? AND labels LIKE ?", 
			accountID, email.Subject, "%\"conversation_id\":\""+conversationID+"\"%").
		First(&existing).Error

	if err == nil {
		return &DuplicateCheckResult{
			IsDuplicate:   true,
			ExistingEmail: &existing,
			ConflictType:  "conversation_id",
			Action:        "update",
			Reason:        "Email with same ConversationID found in Outlook",
		}, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	return &DuplicateCheckResult{IsDuplicate: false}, nil
}

// HandleDuplicate Outlook特殊的重复处理逻辑
func (d *OutlookDeduplicator) HandleDuplicate(ctx context.Context, existing *models.Email, new *providers.EmailMessage, folderID uint) error {
	// 更新文件夹信息
	if existing.FolderID == nil || *existing.FolderID != folderID {
		existing.FolderID = &folderID
	}

	// 更新UID（Outlook中UID可能会变化）
	existing.UID = new.UID

	// 更新邮件状态
	existing.IsRead = d.isEmailRead(new.Flags)
	existing.IsStarred = d.isEmailStarred(new.Flags)
	existing.IsDraft = d.isEmailDraft(new.Flags)

	// 更新Outlook特有的属性
	if err := d.updateOutlookProperties(ctx, existing, new); err != nil {
		log.Printf("Warning: failed to update Outlook properties: %v", err)
	}

	// 补充MessageID（如果原来为空）
	if existing.MessageID == "" && new.MessageID != "" {
		existing.MessageID = new.MessageID
	}

	return d.db.WithContext(ctx).Save(existing).Error
}

// normalizeOutlookMessageID 标准化Outlook MessageID
func (d *OutlookDeduplicator) normalizeOutlookMessageID(messageID string) string {
	// Outlook/Exchange的MessageID可能包含特殊字符或格式
	// 这里进行标准化处理
	
	// 移除可能的尖括号
	messageID = strings.Trim(messageID, "<>")
	
	// 转换为小写（某些Exchange服务器可能大小写不一致）
	messageID = strings.ToLower(messageID)
	
	return messageID
}

// extractConversationID 从邮件中提取会话ID
func (d *OutlookDeduplicator) extractConversationID(email *providers.EmailMessage) string {
	// 这里需要根据Outlook/Exchange的具体实现来提取ConversationID
	// 通常在邮件头的Thread-Index或其他Exchange特有的头部中
	
	// 暂时返回空字符串，后续可以根据需要实现
	// 可以从email.Headers中查找相关信息
	return ""
}

// updateOutlookProperties 更新Outlook特有的属性
func (d *OutlookDeduplicator) updateOutlookProperties(ctx context.Context, email *models.Email, new *providers.EmailMessage) error {
	// 获取当前标签
	currentLabels, err := email.GetLabels()
	if err != nil {
		currentLabels = []string{}
	}

	// 创建标签映射
	labelMap := make(map[string]interface{})
	for _, label := range currentLabels {
		labelMap[label] = true
	}

	// 添加Outlook特有的属性
	conversationID := d.extractConversationID(new)
	if conversationID != "" {
		labelMap["conversation_id"] = conversationID
	}

	// 添加Exchange特有的标志
	d.addExchangeFlags(labelMap, new.Flags)

	// 转换回标签数组
	var updatedLabels []string
	for key, value := range labelMap {
		if strValue, ok := value.(string); ok {
			updatedLabels = append(updatedLabels, key+":"+strValue)
		} else if boolValue, ok := value.(bool); ok && boolValue {
			updatedLabels = append(updatedLabels, key)
		}
	}

	return email.SetLabels(updatedLabels)
}

// addExchangeFlags 添加Exchange特有的标志
func (d *OutlookDeduplicator) addExchangeFlags(labelMap map[string]interface{}, flags []string) {
	for _, flag := range flags {
		switch flag {
		case "\\Answered":
			labelMap["replied"] = true
		case "\\Forwarded":
			labelMap["forwarded"] = true
		case "$MDNSent":
			labelMap["read_receipt_sent"] = true
		case "\\Recent":
			labelMap["recent"] = true
		default:
			// 保留其他Exchange特有的标志
			if strings.HasPrefix(flag, "$") || strings.HasPrefix(flag, "\\") {
				labelMap[flag] = true
			}
		}
	}
}

// isOutlookImportant 检查是否为Outlook重要邮件
func (d *OutlookDeduplicator) isOutlookImportant(flags []string) bool {
	for _, flag := range flags {
		if flag == "$Important" || flag == "\\Flagged" {
			return true
		}
	}
	return false
}

// extractOutlookPriority 提取Outlook邮件优先级
func (d *OutlookDeduplicator) extractOutlookPriority(email *providers.EmailMessage) string {
	// 检查标志中的优先级信息
	for _, flag := range email.Flags {
		switch flag {
		case "$HighPriority", "\\Important":
			return "high"
		case "$LowPriority":
			return "low"
		}
	}
	
	// 默认优先级
	return "normal"
}
