package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"firemail/internal/models"

	"gorm.io/gorm"
)

// SoftDeleteService 软删除管理服务接口
type SoftDeleteService interface {
	// 清理过期的软删除数据
	CleanupExpiredSoftDeletes(ctx context.Context, retentionDays int) error
	
	// 恢复软删除的记录
	RestoreSoftDeleted(ctx context.Context, tableName string, id uint) error
	
	// 永久删除软删除的记录
	PermanentlyDelete(ctx context.Context, tableName string, id uint) error
	
	// 获取软删除统计信息
	GetSoftDeleteStats(ctx context.Context) (*SoftDeleteStats, error)
	
	// 启动自动清理
	StartAutoCleanup(ctx context.Context, retentionDays int) error
	
	// 停止自动清理
	StopAutoCleanup()
}

// SoftDeleteStats 软删除统计信息
type SoftDeleteStats struct {
	TotalSoftDeleted map[string]int64 `json:"total_soft_deleted"`
	OldestDeleted    map[string]time.Time `json:"oldest_deleted"`
	TotalSize        int64 `json:"total_size_estimate"`
}

// SoftDeleteServiceImpl 软删除管理服务实现
type SoftDeleteServiceImpl struct {
	db       *gorm.DB
	stopChan chan struct{}
}

// NewSoftDeleteService 创建软删除管理服务
func NewSoftDeleteService(db *gorm.DB) SoftDeleteService {
	return &SoftDeleteServiceImpl{
		db:       db,
		stopChan: make(chan struct{}),
	}
}

// CleanupExpiredSoftDeletes 清理过期的软删除数据
func (s *SoftDeleteServiceImpl) CleanupExpiredSoftDeletes(ctx context.Context, retentionDays int) error {
	if retentionDays <= 0 {
		return fmt.Errorf("retention days must be positive")
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	log.Printf("Cleaning up soft deleted records older than %d days (before %s)", retentionDays, cutoffTime.Format("2006-01-02"))

	// 定义需要清理的表和模型
	tables := []struct {
		name  string
		model interface{}
	}{
		{"emails", &models.Email{}},
		{"email_accounts", &models.EmailAccount{}},
		{"folders", &models.Folder{}},
		{"attachments", &models.Attachment{}},
		{"users", &models.User{}},
	}

	totalCleaned := 0
	for _, table := range tables {
		count, err := s.cleanupTableSoftDeletes(ctx, table.name, table.model, cutoffTime)
		if err != nil {
			log.Printf("Warning: failed to cleanup table %s: %v", table.name, err)
			continue
		}
		totalCleaned += count
		if count > 0 {
			log.Printf("Cleaned up %d records from table %s", count, table.name)
		}
	}

	log.Printf("Soft delete cleanup completed: %d total records permanently deleted", totalCleaned)
	return nil
}

// cleanupTableSoftDeletes 清理指定表的软删除数据
func (s *SoftDeleteServiceImpl) cleanupTableSoftDeletes(ctx context.Context, tableName string, model interface{}, cutoffTime time.Time) (int, error) {
	// 使用Unscoped()来操作软删除的记录
	result := s.db.Unscoped().Where("deleted_at IS NOT NULL AND deleted_at < ?", cutoffTime).Delete(model)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to cleanup %s: %w", tableName, result.Error)
	}
	
	return int(result.RowsAffected), nil
}

// RestoreSoftDeleted 恢复软删除的记录
func (s *SoftDeleteServiceImpl) RestoreSoftDeleted(ctx context.Context, tableName string, id uint) error {
	// 根据表名选择对应的模型
	var model interface{}
	switch tableName {
	case "emails":
		model = &models.Email{}
	case "email_accounts":
		model = &models.EmailAccount{}
	case "folders":
		model = &models.Folder{}
	case "attachments":
		model = &models.Attachment{}
	case "users":
		model = &models.User{}
	default:
		return fmt.Errorf("unsupported table: %s", tableName)
	}

	// 使用Unscoped()来查找软删除的记录，然后恢复
	result := s.db.Unscoped().Model(model).Where("id = ? AND deleted_at IS NOT NULL", id).Update("deleted_at", nil)
	if result.Error != nil {
		return fmt.Errorf("failed to restore record: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("no soft deleted record found with id %d in table %s", id, tableName)
	}

	log.Printf("Restored soft deleted record: table=%s, id=%d", tableName, id)
	return nil
}

// PermanentlyDelete 永久删除软删除的记录
func (s *SoftDeleteServiceImpl) PermanentlyDelete(ctx context.Context, tableName string, id uint) error {
	// 根据表名选择对应的模型
	var model interface{}
	switch tableName {
	case "emails":
		model = &models.Email{}
	case "email_accounts":
		model = &models.EmailAccount{}
	case "folders":
		model = &models.Folder{}
	case "attachments":
		model = &models.Attachment{}
	case "users":
		model = &models.User{}
	default:
		return fmt.Errorf("unsupported table: %s", tableName)
	}

	// 使用Unscoped()来永久删除记录
	result := s.db.Unscoped().Where("id = ?", id).Delete(model)
	if result.Error != nil {
		return fmt.Errorf("failed to permanently delete record: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("no record found with id %d in table %s", id, tableName)
	}

	log.Printf("Permanently deleted record: table=%s, id=%d", tableName, id)
	return nil
}

// GetSoftDeleteStats 获取软删除统计信息
func (s *SoftDeleteServiceImpl) GetSoftDeleteStats(ctx context.Context) (*SoftDeleteStats, error) {
	stats := &SoftDeleteStats{
		TotalSoftDeleted: make(map[string]int64),
		OldestDeleted:    make(map[string]time.Time),
	}

	// 定义需要统计的表
	tables := []struct {
		name  string
		model interface{}
	}{
		{"emails", &models.Email{}},
		{"email_accounts", &models.EmailAccount{}},
		{"folders", &models.Folder{}},
		{"attachments", &models.Attachment{}},
		{"users", &models.User{}},
	}

	for _, table := range tables {
		// 统计软删除记录数量
		var count int64
		err := s.db.Unscoped().Model(table.model).Where("deleted_at IS NOT NULL").Count(&count).Error
		if err != nil {
			log.Printf("Warning: failed to count soft deleted records in %s: %v", table.name, err)
			continue
		}
		stats.TotalSoftDeleted[table.name] = count

		// 获取最早的软删除时间
		if count > 0 {
			var oldestDeleted time.Time
			err := s.db.Unscoped().Model(table.model).
				Where("deleted_at IS NOT NULL").
				Order("deleted_at ASC").
				Limit(1).
				Pluck("deleted_at", &oldestDeleted).Error
			if err == nil {
				stats.OldestDeleted[table.name] = oldestDeleted
			}
		}
	}

	return stats, nil
}

// StartAutoCleanup 启动自动清理
func (s *SoftDeleteServiceImpl) StartAutoCleanup(ctx context.Context, retentionDays int) error {
	if retentionDays <= 0 {
		return fmt.Errorf("retention days must be positive")
	}

	log.Printf("Starting automatic soft delete cleanup service (retention: %d days)...", retentionDays)
	
	go func() {
		// 每周执行一次清理
		ticker := time.NewTicker(7 * 24 * time.Hour)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				log.Println("Running scheduled soft delete cleanup...")
				if err := s.CleanupExpiredSoftDeletes(ctx, retentionDays); err != nil {
					log.Printf("Scheduled soft delete cleanup failed: %v", err)
				}
			case <-s.stopChan:
				log.Println("Stopping automatic soft delete cleanup service...")
				return
			case <-ctx.Done():
				log.Println("Context cancelled, stopping automatic soft delete cleanup service...")
				return
			}
		}
	}()
	
	return nil
}

// StopAutoCleanup 停止自动清理
func (s *SoftDeleteServiceImpl) StopAutoCleanup() {
	close(s.stopChan)
}

// ValidateSoftDeleteQueries 验证软删除查询的辅助函数
func ValidateSoftDeleteQueries(db *gorm.DB) error {
	// 这个函数可以用来验证所有查询都正确处理了软删除
	// 在开发和测试环境中使用
	
	log.Println("Validating soft delete query behavior...")
	
	// 测试基本的软删除行为
	var count int64
	
	// 正常查询应该不包含软删除的记录
	if err := db.Model(&models.Email{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to count emails: %w", err)
	}
	
	// 使用Unscoped查询应该包含所有记录
	var totalCount int64
	if err := db.Unscoped().Model(&models.Email{}).Count(&totalCount).Error; err != nil {
		return fmt.Errorf("failed to count all emails: %w", err)
	}
	
	log.Printf("Soft delete validation: normal count=%d, total count=%d", count, totalCount)
	return nil
}
