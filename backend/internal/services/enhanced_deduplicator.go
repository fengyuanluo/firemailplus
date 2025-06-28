package services

import (
	"context"
	"crypto/md5"
	"fmt"
	"log"
	"sync"
	"time"

	"firemail/internal/models"
	"firemail/internal/providers"

	"gorm.io/gorm"
)

// DeduplicationStats 去重统计信息
type DeduplicationStats struct {
	TotalChecked     int64     `json:"total_checked"`
	DuplicatesFound  int64     `json:"duplicates_found"`
	DuplicatesSkipped int64    `json:"duplicates_skipped"`
	DuplicatesUpdated int64    `json:"duplicates_updated"`
	DuplicatesMerged  int64    `json:"duplicates_merged"`
	ProcessingTime   time.Duration `json:"processing_time"`
	LastUpdated      time.Time `json:"last_updated"`
}

// BatchDeduplicationResult 批量去重结果
type BatchDeduplicationResult struct {
	ProcessedCount   int                    `json:"processed_count"`
	DuplicateCount   int                    `json:"duplicate_count"`
	ErrorCount       int                    `json:"error_count"`
	Stats            *DeduplicationStats    `json:"stats"`
	Errors           []string               `json:"errors,omitempty"`
	Details          []*DuplicateCheckResult `json:"details,omitempty"`
}

// EnhancedDeduplicator 增强的去重器接口
type EnhancedDeduplicator interface {
	EmailDeduplicator
	
	// 批量去重
	BatchCheckDuplicates(ctx context.Context, emails []*providers.EmailMessage, accountID, folderID uint) (*BatchDeduplicationResult, error)
	
	// 跨文件夹去重
	CheckCrossFolderDuplicates(ctx context.Context, accountID uint) (*BatchDeduplicationResult, error)
	
	// 获取去重统计
	GetDeduplicationStats(ctx context.Context, accountID uint) (*DeduplicationStats, error)
	
	// 清理重复邮件
	CleanupDuplicates(ctx context.Context, accountID uint, dryRun bool) (*BatchDeduplicationResult, error)
	
	// 重建去重索引
	RebuildDeduplicationIndex(ctx context.Context, accountID uint) error
}

// EnhancedStandardDeduplicator 增强的标准去重器
type EnhancedStandardDeduplicator struct {
	*StandardDeduplicator
	cache      sync.Map // 缓存最近检查的结果
	stats      *DeduplicationStats
	statsMutex sync.RWMutex
}

// NewEnhancedStandardDeduplicator 创建增强的标准去重器
func NewEnhancedStandardDeduplicator(db *gorm.DB) EnhancedDeduplicator {
	return &EnhancedStandardDeduplicator{
		StandardDeduplicator: &StandardDeduplicator{db: db},
		stats: &DeduplicationStats{
			LastUpdated: time.Now(),
		},
	}
}

// BatchCheckDuplicates 批量检查重复邮件
func (d *EnhancedStandardDeduplicator) BatchCheckDuplicates(ctx context.Context, emails []*providers.EmailMessage, accountID, folderID uint) (*BatchDeduplicationResult, error) {
	startTime := time.Now()
	result := &BatchDeduplicationResult{
		Stats:   &DeduplicationStats{},
		Details: make([]*DuplicateCheckResult, 0, len(emails)),
	}

	// 预处理：构建MessageID映射以提高性能
	messageIDMap := make(map[string]*providers.EmailMessage)
	for _, email := range emails {
		if email.MessageID != "" {
			messageIDMap[email.MessageID] = email
		}
	}

	// 批量查询现有邮件
	existingEmails, err := d.batchQueryExistingEmails(ctx, messageIDMap, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to query existing emails: %w", err)
	}

	// 处理每封邮件
	for _, email := range emails {
		checkResult, err := d.checkDuplicateWithCache(ctx, email, accountID, folderID, existingEmails)
		if err != nil {
			result.ErrorCount++
			result.Errors = append(result.Errors, fmt.Sprintf("Email %s: %v", email.MessageID, err))
			continue
		}

		result.ProcessedCount++
		result.Details = append(result.Details, checkResult)

		if checkResult.IsDuplicate {
			result.DuplicateCount++
			d.updateStats(checkResult.Action)
		}
	}

	result.Stats.ProcessingTime = time.Since(startTime)
	result.Stats.TotalChecked = int64(result.ProcessedCount)
	result.Stats.DuplicatesFound = int64(result.DuplicateCount)
	result.Stats.LastUpdated = time.Now()

	return result, nil
}

// batchQueryExistingEmails 批量查询现有邮件
func (d *EnhancedStandardDeduplicator) batchQueryExistingEmails(ctx context.Context, messageIDMap map[string]*providers.EmailMessage, accountID uint) (map[string]*models.Email, error) {
	if len(messageIDMap) == 0 {
		return make(map[string]*models.Email), nil
	}

	messageIDs := make([]string, 0, len(messageIDMap))
	for messageID := range messageIDMap {
		messageIDs = append(messageIDs, messageID)
	}

	var existingEmails []models.Email
	err := d.db.WithContext(ctx).
		Where("account_id = ? AND message_id IN ?", accountID, messageIDs).
		Find(&existingEmails).Error

	if err != nil {
		return nil, err
	}

	result := make(map[string]*models.Email)
	for i := range existingEmails {
		result[existingEmails[i].MessageID] = &existingEmails[i]
	}

	return result, nil
}

// checkDuplicateWithCache 使用缓存检查重复
func (d *EnhancedStandardDeduplicator) checkDuplicateWithCache(ctx context.Context, email *providers.EmailMessage, accountID, folderID uint, existingEmails map[string]*models.Email) (*DuplicateCheckResult, error) {
	// 生成缓存键
	cacheKey := d.generateCacheKey(email, accountID, folderID)
	
	// 检查缓存
	if cached, ok := d.cache.Load(cacheKey); ok {
		if result, ok := cached.(*DuplicateCheckResult); ok {
			return result, nil
		}
	}

	// 使用预查询的结果进行检查
	var result *DuplicateCheckResult
	var err error

	if email.MessageID != "" {
		if existing, found := existingEmails[email.MessageID]; found {
			result = d.buildDuplicateResult(existing, email, folderID, "message_id")
		}
	}

	if result == nil {
		// 如果MessageID检查没有找到重复，进行标准检查
		result, err = d.StandardDeduplicator.CheckDuplicate(ctx, email, accountID, folderID)
		if err != nil {
			return nil, err
		}
	}

	// 缓存结果（设置过期时间）
	d.cache.Store(cacheKey, result)
	
	// 定期清理缓存
	go d.cleanupCacheIfNeeded()

	return result, nil
}

// buildDuplicateResult 构建重复检查结果
func (d *EnhancedStandardDeduplicator) buildDuplicateResult(existing *models.Email, new *providers.EmailMessage, folderID uint, conflictType string) *DuplicateCheckResult {
	action := "skip"
	reason := "Email with same MessageID already exists"
	
	// 如果在不同文件夹，可能需要更新文件夹信息
	if existing.FolderID == nil || *existing.FolderID != folderID {
		action = "update"
		reason = "Email exists in different folder, updating folder reference"
	}

	return &DuplicateCheckResult{
		IsDuplicate:   true,
		ExistingEmail: existing,
		ConflictType:  conflictType,
		Action:        action,
		Reason:        reason,
	}
}

// CheckCrossFolderDuplicates 检查跨文件夹重复
func (d *EnhancedStandardDeduplicator) CheckCrossFolderDuplicates(ctx context.Context, accountID uint) (*BatchDeduplicationResult, error) {
	startTime := time.Now()
	result := &BatchDeduplicationResult{
		Stats: &DeduplicationStats{},
	}

	// 查找具有相同MessageID但在不同文件夹的邮件
	query := `
		SELECT message_id, COUNT(*) as count, GROUP_CONCAT(folder_id) as folder_ids
		FROM emails 
		WHERE account_id = ? AND message_id != '' 
		GROUP BY message_id 
		HAVING count > 1
	`

	type duplicateGroup struct {
		MessageID string `gorm:"column:message_id"`
		Count     int    `gorm:"column:count"`
		FolderIDs string `gorm:"column:folder_ids"`
	}

	var duplicateGroups []duplicateGroup
	err := d.db.WithContext(ctx).Raw(query, accountID).Scan(&duplicateGroups).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find cross-folder duplicates: %w", err)
	}

	result.ProcessedCount = len(duplicateGroups)
	result.DuplicateCount = len(duplicateGroups)

	// 处理每个重复组
	for _, group := range duplicateGroups {
		err := d.processCrossFolderDuplicateGroup(ctx, group.MessageID, accountID)
		if err != nil {
			result.ErrorCount++
			result.Errors = append(result.Errors, fmt.Sprintf("MessageID %s: %v", group.MessageID, err))
		}
	}

	result.Stats.ProcessingTime = time.Since(startTime)
	result.Stats.TotalChecked = int64(result.ProcessedCount)
	result.Stats.DuplicatesFound = int64(result.DuplicateCount)
	result.Stats.LastUpdated = time.Now()

	return result, nil
}

// processCrossFolderDuplicateGroup 处理跨文件夹重复组
func (d *EnhancedStandardDeduplicator) processCrossFolderDuplicateGroup(ctx context.Context, messageID string, accountID uint) error {
	var emails []models.Email
	err := d.db.WithContext(ctx).
		Where("account_id = ? AND message_id = ?", accountID, messageID).
		Order("created_at ASC").
		Find(&emails).Error

	if err != nil {
		return err
	}

	if len(emails) <= 1 {
		return nil // 没有重复
	}

	// 保留最早的邮件，标记其他为重复
	primary := &emails[0]
	for i := 1; i < len(emails); i++ {
		duplicate := &emails[i]
		
		// 如果重复邮件在不同文件夹，可能需要保留（如Gmail标签系统）
		if d.shouldKeepCrossFolderDuplicate(primary, duplicate) {
			continue
		}

		// 标记为重复或合并信息
		err := d.mergeDuplicateEmails(ctx, primary, duplicate)
		if err != nil {
			log.Printf("Failed to merge duplicate emails: %v", err)
		}
	}

	return nil
}

// shouldKeepCrossFolderDuplicate 判断是否应该保留跨文件夹重复
func (d *EnhancedStandardDeduplicator) shouldKeepCrossFolderDuplicate(primary, duplicate *models.Email) bool {
	// 如果是不同文件夹且文件夹类型不同，可能需要保留
	// 这里可以根据具体需求实现更复杂的逻辑
	return primary.FolderID != duplicate.FolderID
}

// mergeDuplicateEmails 合并重复邮件
func (d *EnhancedStandardDeduplicator) mergeDuplicateEmails(ctx context.Context, primary, duplicate *models.Email) error {
	// 合并有用的信息到主邮件
	updated := false

	// 如果主邮件缺少某些信息，从重复邮件中补充
	if primary.HTMLBody == "" && duplicate.HTMLBody != "" {
		primary.HTMLBody = duplicate.HTMLBody
		updated = true
	}

	if primary.TextBody == "" && duplicate.TextBody != "" {
		primary.TextBody = duplicate.TextBody
		updated = true
	}

	// 保存更新
	if updated {
		err := d.db.WithContext(ctx).Save(primary).Error
		if err != nil {
			return err
		}
	}

	// 删除重复邮件
	return d.db.WithContext(ctx).Delete(duplicate).Error
}

// generateCacheKey 生成缓存键
func (d *EnhancedStandardDeduplicator) generateCacheKey(email *providers.EmailMessage, accountID, folderID uint) string {
	content := fmt.Sprintf("%d:%d:%s:%d", accountID, folderID, email.MessageID, email.UID)
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// cleanupCacheIfNeeded 定期清理缓存
func (d *EnhancedStandardDeduplicator) cleanupCacheIfNeeded() {
	// 简单的缓存清理策略：定期清理所有缓存
	// 在生产环境中，可以实现更复杂的LRU或TTL策略
	d.cache.Range(func(key, value interface{}) bool {
		d.cache.Delete(key)
		return true
	})
}

// updateStats 更新统计信息
func (d *EnhancedStandardDeduplicator) updateStats(action string) {
	d.statsMutex.Lock()
	defer d.statsMutex.Unlock()

	switch action {
	case "skip":
		d.stats.DuplicatesSkipped++
	case "update":
		d.stats.DuplicatesUpdated++
	case "merge":
		d.stats.DuplicatesMerged++
	}
	
	d.stats.LastUpdated = time.Now()
}

// GetDeduplicationStats 获取去重统计
func (d *EnhancedStandardDeduplicator) GetDeduplicationStats(ctx context.Context, accountID uint) (*DeduplicationStats, error) {
	d.statsMutex.RLock()
	defer d.statsMutex.RUnlock()

	// 创建统计信息副本
	stats := &DeduplicationStats{
		TotalChecked:      d.stats.TotalChecked,
		DuplicatesFound:   d.stats.DuplicatesFound,
		DuplicatesSkipped: d.stats.DuplicatesSkipped,
		DuplicatesUpdated: d.stats.DuplicatesUpdated,
		DuplicatesMerged:  d.stats.DuplicatesMerged,
		ProcessingTime:    d.stats.ProcessingTime,
		LastUpdated:       d.stats.LastUpdated,
	}

	return stats, nil
}

// CleanupDuplicates 清理重复邮件
func (d *EnhancedStandardDeduplicator) CleanupDuplicates(ctx context.Context, accountID uint, dryRun bool) (*BatchDeduplicationResult, error) {
	startTime := time.Now()
	result := &BatchDeduplicationResult{
		Stats: &DeduplicationStats{},
	}

	// 查找所有重复邮件
	duplicateQuery := `
		SELECT message_id, COUNT(*) as count
		FROM emails
		WHERE account_id = ? AND message_id != ''
		GROUP BY message_id
		HAVING count > 1
	`

	type duplicateInfo struct {
		MessageID string `gorm:"column:message_id"`
		Count     int    `gorm:"column:count"`
	}

	var duplicates []duplicateInfo
	err := d.db.WithContext(ctx).Raw(duplicateQuery, accountID).Scan(&duplicates).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find duplicates: %w", err)
	}

	result.ProcessedCount = len(duplicates)

	// 处理每个重复组
	for _, dup := range duplicates {
		err := d.cleanupDuplicateGroup(ctx, dup.MessageID, accountID, dryRun)
		if err != nil {
			result.ErrorCount++
			result.Errors = append(result.Errors, fmt.Sprintf("MessageID %s: %v", dup.MessageID, err))
		} else {
			result.DuplicateCount += dup.Count - 1 // 保留一个，删除其余的
		}
	}

	result.Stats.ProcessingTime = time.Since(startTime)
	result.Stats.TotalChecked = int64(result.ProcessedCount)
	result.Stats.DuplicatesFound = int64(result.DuplicateCount)
	result.Stats.LastUpdated = time.Now()

	return result, nil
}

// cleanupDuplicateGroup 清理重复邮件组
func (d *EnhancedStandardDeduplicator) cleanupDuplicateGroup(ctx context.Context, messageID string, accountID uint, dryRun bool) error {
	var emails []models.Email
	err := d.db.WithContext(ctx).
		Where("account_id = ? AND message_id = ?", accountID, messageID).
		Order("created_at ASC").
		Find(&emails).Error

	if err != nil {
		return err
	}

	if len(emails) <= 1 {
		return nil // 没有重复
	}

	// 保留最早的邮件，删除其他重复邮件
	primary := &emails[0]
	for i := 1; i < len(emails); i++ {
		duplicate := &emails[i]

		if dryRun {
			log.Printf("DRY RUN: Would delete duplicate email ID %d (MessageID: %s)", duplicate.ID, duplicate.MessageID)
			continue
		}

		// 合并有用信息到主邮件
		err := d.mergeDuplicateEmails(ctx, primary, duplicate)
		if err != nil {
			return fmt.Errorf("failed to merge and delete duplicate: %w", err)
		}

		log.Printf("Deleted duplicate email ID %d (MessageID: %s)", duplicate.ID, duplicate.MessageID)
	}

	return nil
}

// RebuildDeduplicationIndex 重建去重索引
func (d *EnhancedStandardDeduplicator) RebuildDeduplicationIndex(ctx context.Context, accountID uint) error {
	// 清理缓存
	d.cleanupCacheIfNeeded()

	// 重置统计信息
	d.statsMutex.Lock()
	d.stats = &DeduplicationStats{
		LastUpdated: time.Now(),
	}
	d.statsMutex.Unlock()

	// 重建数据库索引（如果需要）
	indexQueries := []string{
		"CREATE INDEX IF NOT EXISTS idx_emails_account_message_id ON emails(account_id, message_id)",
		"CREATE INDEX IF NOT EXISTS idx_emails_account_folder_uid ON emails(account_id, folder_id, uid)",
		"CREATE INDEX IF NOT EXISTS idx_emails_account_subject_from_date ON emails(account_id, subject, from, date)",
	}

	for _, query := range indexQueries {
		err := d.db.WithContext(ctx).Exec(query).Error
		if err != nil {
			log.Printf("Warning: failed to create index: %v", err)
		}
	}

	log.Printf("Rebuilt deduplication index for account %d", accountID)
	return nil
}

// EnhancedGmailDeduplicator Gmail增强去重器
type EnhancedGmailDeduplicator struct {
	*EnhancedStandardDeduplicator
	*GmailDeduplicator
}

// NewEnhancedGmailDeduplicator 创建Gmail增强去重器
func NewEnhancedGmailDeduplicator(db *gorm.DB) EnhancedDeduplicator {
	return &EnhancedGmailDeduplicator{
		EnhancedStandardDeduplicator: &EnhancedStandardDeduplicator{
			StandardDeduplicator: &StandardDeduplicator{db: db},
			stats: &DeduplicationStats{
				LastUpdated: time.Now(),
			},
		},
		GmailDeduplicator: &GmailDeduplicator{
			StandardDeduplicator: &StandardDeduplicator{db: db},
		},
	}
}

// CheckDuplicate Gmail增强重复检查
func (d *EnhancedGmailDeduplicator) CheckDuplicate(ctx context.Context, email *providers.EmailMessage, accountID, folderID uint) (*DuplicateCheckResult, error) {
	// 使用Gmail特殊逻辑
	return d.GmailDeduplicator.CheckDuplicate(ctx, email, accountID, folderID)
}

// HandleDuplicate Gmail增强重复处理
func (d *EnhancedGmailDeduplicator) HandleDuplicate(ctx context.Context, existing *models.Email, new *providers.EmailMessage, folderID uint) error {
	// 使用Gmail特殊逻辑
	return d.GmailDeduplicator.HandleDuplicate(ctx, existing, new, folderID)
}

// shouldKeepCrossFolderDuplicate Gmail特殊的跨文件夹重复保留逻辑
func (d *EnhancedGmailDeduplicator) shouldKeepCrossFolderDuplicate(primary, duplicate *models.Email) bool {
	// Gmail标签系统：同一邮件在不同标签中是正常的，应该保留
	return true
}
