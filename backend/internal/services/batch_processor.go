package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"firemail/internal/models"
)

// BatchProcessor 批量处理器
type BatchProcessor struct {
	db                  *gorm.DB
	batchSize          int
	maxRetries         int
	retryDelay         time.Duration
	enableOptimization bool
}

// NewBatchProcessor 创建批量处理器
func NewBatchProcessor(db *gorm.DB) *BatchProcessor {
	return &BatchProcessor{
		db:                  db,
		batchSize:          100,
		maxRetries:         3,
		retryDelay:         time.Second,
		enableOptimization: true,
	}
}

// BatchInsertEmails 批量插入邮件
func (p *BatchProcessor) BatchInsertEmails(ctx context.Context, emails []*models.Email) error {
	if len(emails) == 0 {
		return nil
	}

	// 分批处理
	for i := 0; i < len(emails); i += p.batchSize {
		end := i + p.batchSize
		if end > len(emails) {
			end = len(emails)
		}

		batch := emails[i:end]
		if err := p.insertEmailBatch(ctx, batch); err != nil {
			return fmt.Errorf("failed to insert batch %d-%d: %w", i, end-1, err)
		}
	}

	return nil
}

// insertEmailBatch 插入一批邮件
func (p *BatchProcessor) insertEmailBatch(ctx context.Context, emails []*models.Email) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 使用批量插入
		if err := tx.CreateInBatches(emails, len(emails)).Error; err != nil {
			// 如果批量插入失败，尝试逐个插入以处理重复
			return p.insertEmailsOneByOne(tx, emails)
		}
		return nil
	})
}

// insertEmailsOneByOne 逐个插入邮件（处理重复）
func (p *BatchProcessor) insertEmailsOneByOne(tx *gorm.DB, emails []*models.Email) error {
	for _, email := range emails {
		if err := tx.Create(email).Error; err != nil {
			if isUniqueConstraintError(err) {
				log.Printf("Duplicate email detected: %s, skipping", email.MessageID)
				continue
			}
			return fmt.Errorf("failed to create email %s: %w", email.MessageID, err)
		}
	}
	return nil
}

// BatchUpdateEmails 批量更新邮件
func (p *BatchProcessor) BatchUpdateEmails(ctx context.Context, updates []EmailUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	// 按更新类型分组
	updateGroups := p.groupUpdatesByType(updates)

	// 分别处理每种更新类型
	for updateType, groupUpdates := range updateGroups {
		if err := p.processBatchUpdate(ctx, updateType, groupUpdates); err != nil {
			return fmt.Errorf("failed to process %s updates: %w", updateType, err)
		}
	}

	return nil
}

// EmailUpdate 邮件更新
type EmailUpdate struct {
	EmailID uint
	Type    string // "read", "star", "flag", "folder"
	Value   interface{}
}

// groupUpdatesByType 按更新类型分组
func (p *BatchProcessor) groupUpdatesByType(updates []EmailUpdate) map[string][]EmailUpdate {
	groups := make(map[string][]EmailUpdate)
	for _, update := range updates {
		groups[update.Type] = append(groups[update.Type], update)
	}
	return groups
}

// processBatchUpdate 处理批量更新
func (p *BatchProcessor) processBatchUpdate(ctx context.Context, updateType string, updates []EmailUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	switch updateType {
	case "read":
		return p.batchUpdateReadStatus(ctx, updates)
	case "star":
		return p.batchUpdateStarStatus(ctx, updates)
	case "folder":
		return p.batchUpdateFolder(ctx, updates)
	default:
		return fmt.Errorf("unsupported update type: %s", updateType)
	}
}

// batchUpdateReadStatus 批量更新已读状态
func (p *BatchProcessor) batchUpdateReadStatus(ctx context.Context, updates []EmailUpdate) error {
	// 按值分组（true/false）
	trueIDs := make([]uint, 0)
	falseIDs := make([]uint, 0)

	for _, update := range updates {
		if isRead, ok := update.Value.(bool); ok {
			if isRead {
				trueIDs = append(trueIDs, update.EmailID)
			} else {
				falseIDs = append(falseIDs, update.EmailID)
			}
		}
	}

	// 批量更新
	if len(trueIDs) > 0 {
		if err := p.db.WithContext(ctx).Model(&models.Email{}).
			Where("id IN ?", trueIDs).
			Update("is_read", true).Error; err != nil {
			return fmt.Errorf("failed to update read status to true: %w", err)
		}
	}

	if len(falseIDs) > 0 {
		if err := p.db.WithContext(ctx).Model(&models.Email{}).
			Where("id IN ?", falseIDs).
			Update("is_read", false).Error; err != nil {
			return fmt.Errorf("failed to update read status to false: %w", err)
		}
	}

	return nil
}

// batchUpdateStarStatus 批量更新星标状态
func (p *BatchProcessor) batchUpdateStarStatus(ctx context.Context, updates []EmailUpdate) error {
	// 按值分组（true/false）
	trueIDs := make([]uint, 0)
	falseIDs := make([]uint, 0)

	for _, update := range updates {
		if isStarred, ok := update.Value.(bool); ok {
			if isStarred {
				trueIDs = append(trueIDs, update.EmailID)
			} else {
				falseIDs = append(falseIDs, update.EmailID)
			}
		}
	}

	// 批量更新
	if len(trueIDs) > 0 {
		if err := p.db.WithContext(ctx).Model(&models.Email{}).
			Where("id IN ?", trueIDs).
			Update("is_starred", true).Error; err != nil {
			return fmt.Errorf("failed to update star status to true: %w", err)
		}
	}

	if len(falseIDs) > 0 {
		if err := p.db.WithContext(ctx).Model(&models.Email{}).
			Where("id IN ?", falseIDs).
			Update("is_starred", false).Error; err != nil {
			return fmt.Errorf("failed to update star status to false: %w", err)
		}
	}

	return nil
}

// batchUpdateFolder 批量更新文件夹
func (p *BatchProcessor) batchUpdateFolder(ctx context.Context, updates []EmailUpdate) error {
	// 按文件夹ID分组
	folderGroups := make(map[uint][]uint)
	for _, update := range updates {
		if folderID, ok := update.Value.(uint); ok {
			folderGroups[folderID] = append(folderGroups[folderID], update.EmailID)
		}
	}

	// 批量更新每个文件夹
	for folderID, emailIDs := range folderGroups {
		if err := p.db.WithContext(ctx).Model(&models.Email{}).
			Where("id IN ?", emailIDs).
			Update("folder_id", folderID).Error; err != nil {
			return fmt.Errorf("failed to update folder to %d: %w", folderID, err)
		}
	}

	return nil
}

// BatchDeleteEmails 批量删除邮件
func (p *BatchProcessor) BatchDeleteEmails(ctx context.Context, emailIDs []uint) error {
	if len(emailIDs) == 0 {
		return nil
	}

	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先删除附件
		if err := tx.Where("email_id IN ?", emailIDs).Delete(&models.Attachment{}).Error; err != nil {
			return fmt.Errorf("failed to delete attachments: %w", err)
		}

		// 再删除邮件
		if err := tx.Where("id IN ?", emailIDs).Delete(&models.Email{}).Error; err != nil {
			return fmt.Errorf("failed to delete emails: %w", err)
		}

		return nil
	})
}

// BatchInsertAttachments 批量插入附件
func (p *BatchProcessor) BatchInsertAttachments(ctx context.Context, attachments []*models.Attachment) error {
	if len(attachments) == 0 {
		return nil
	}

	// 分批处理
	for i := 0; i < len(attachments); i += p.batchSize {
		end := i + p.batchSize
		if end > len(attachments) {
			end = len(attachments)
		}

		batch := attachments[i:end]
		if err := p.db.WithContext(ctx).CreateInBatches(batch, len(batch)).Error; err != nil {
			return fmt.Errorf("failed to insert attachment batch %d-%d: %w", i, end-1, err)
		}
	}

	return nil
}

// OptimizedEmailQuery 优化的邮件查询
type OptimizedEmailQuery struct {
	db *gorm.DB
}

// NewOptimizedEmailQuery 创建优化的邮件查询
func NewOptimizedEmailQuery(db *gorm.DB) *OptimizedEmailQuery {
	return &OptimizedEmailQuery{db: db}
}

// GetEmailsWithPagination 分页获取邮件（优化版）
func (q *OptimizedEmailQuery) GetEmailsWithPagination(
	ctx context.Context,
	folderID uint,
	page, pageSize int,
	includeAttachments bool,
) ([]*models.Email, int64, error) {
	var emails []*models.Email
	var total int64

	// 构建基础查询
	baseQuery := q.db.WithContext(ctx).Model(&models.Email{}).Where("folder_id = ?", folderID)

	// 获取总数
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count emails: %w", err)
	}

	// 构建分页查询
	query := baseQuery.Order("date DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize)

	// 预加载关联数据
	if includeAttachments {
		query = query.Preload("Attachments")
	}

	if err := query.Find(&emails).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get emails: %w", err)
	}

	return emails, total, nil
}

// GetEmailsByUIDs 根据UID列表获取邮件
func (q *OptimizedEmailQuery) GetEmailsByUIDs(ctx context.Context, folderID uint, uids []uint32) ([]*models.Email, error) {
	if len(uids) == 0 {
		return []*models.Email{}, nil
	}

	var emails []*models.Email
	err := q.db.WithContext(ctx).
		Where("folder_id = ? AND uid IN ?", folderID, uids).
		Find(&emails).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get emails by UIDs: %w", err)
	}

	return emails, nil
}


