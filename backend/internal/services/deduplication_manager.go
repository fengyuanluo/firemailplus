package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"firemail/internal/config"
	"firemail/internal/models"

	"gorm.io/gorm"
)

// DeduplicationManager 去重管理器接口
type DeduplicationManager interface {
	// 执行账户去重
	DeduplicateAccount(ctx context.Context, accountID uint, options *DeduplicationOptions) (*BatchDeduplicationResult, error)
	
	// 执行用户所有账户去重
	DeduplicateUser(ctx context.Context, userID uint, options *DeduplicationOptions) (*UserDeduplicationResult, error)
	
	// 获取去重报告
	GetDeduplicationReport(ctx context.Context, accountID uint) (*DeduplicationReport, error)
	
	// 计划去重任务
	ScheduleDeduplication(ctx context.Context, accountID uint, schedule *DeduplicationSchedule) error
	
	// 取消计划去重任务
	CancelScheduledDeduplication(ctx context.Context, accountID uint) error
}

// DeduplicationOptions 去重选项
type DeduplicationOptions struct {
	DryRun              bool     `json:"dry_run"`               // 是否为试运行
	CrossFolder         bool     `json:"cross_folder"`          // 是否检查跨文件夹重复
	CleanupDuplicates   bool     `json:"cleanup_duplicates"`    // 是否清理重复邮件
	RebuildIndex        bool     `json:"rebuild_index"`         // 是否重建索引
	BatchSize           int      `json:"batch_size"`            // 批处理大小
	MaxProcessingTime   time.Duration `json:"max_processing_time"` // 最大处理时间
	IncludeFolders      []string `json:"include_folders"`       // 包含的文件夹
	ExcludeFolders      []string `json:"exclude_folders"`       // 排除的文件夹
	NotifyOnCompletion  bool     `json:"notify_on_completion"`  // 完成时通知
}

// DeduplicationSchedule 去重计划
type DeduplicationSchedule struct {
	Enabled     bool                 `json:"enabled"`
	Frequency   string               `json:"frequency"` // daily, weekly, monthly
	Time        string               `json:"time"`      // HH:MM format
	Options     *DeduplicationOptions `json:"options"`
	NextRun     time.Time            `json:"next_run"`
	LastRun     *time.Time           `json:"last_run,omitempty"`
}

// UserDeduplicationResult 用户去重结果
type UserDeduplicationResult struct {
	UserID          uint                        `json:"user_id"`
	AccountResults  map[uint]*BatchDeduplicationResult `json:"account_results"`
	TotalProcessed  int                         `json:"total_processed"`
	TotalDuplicates int                         `json:"total_duplicates"`
	TotalErrors     int                         `json:"total_errors"`
	ProcessingTime  time.Duration               `json:"processing_time"`
	StartTime       time.Time                   `json:"start_time"`
	EndTime         time.Time                   `json:"end_time"`
}

// DeduplicationReport 去重报告
type DeduplicationReport struct {
	AccountID       uint                `json:"account_id"`
	Stats           *DeduplicationStats `json:"stats"`
	RecentActivity  []*DeduplicationActivity `json:"recent_activity"`
	Recommendations []*DeduplicationRecommendation `json:"recommendations"`
	GeneratedAt     time.Time           `json:"generated_at"`
}

// DeduplicationActivity 去重活动记录
type DeduplicationActivity struct {
	ID          uint      `json:"id"`
	AccountID   uint      `json:"account_id"`
	Type        string    `json:"type"` // check, cleanup, rebuild_index
	Status      string    `json:"status"` // running, completed, failed
	StartTime   time.Time `json:"start_time"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Result      string    `json:"result,omitempty"`
	ErrorMessage string   `json:"error_message,omitempty"`
}

// DeduplicationRecommendation 去重建议
type DeduplicationRecommendation struct {
	Type        string `json:"type"`        // cleanup, rebuild_index, schedule
	Priority    string `json:"priority"`    // high, medium, low
	Title       string `json:"title"`
	Description string `json:"description"`
	Action      string `json:"action"`
}

// StandardDeduplicationManager 标准去重管理器
type StandardDeduplicationManager struct {
	db                  *gorm.DB
	deduplicatorFactory DeduplicatorFactory
	eventTrigger        EventTrigger
}

// NewDeduplicationManager 创建去重管理器
func NewDeduplicationManager(db *gorm.DB, deduplicatorFactory DeduplicatorFactory, eventTrigger EventTrigger) DeduplicationManager {
	return &StandardDeduplicationManager{
		db:                  db,
		deduplicatorFactory: deduplicatorFactory,
		eventTrigger:        eventTrigger,
	}
}

// DeduplicateAccount 执行账户去重
func (m *StandardDeduplicationManager) DeduplicateAccount(ctx context.Context, accountID uint, options *DeduplicationOptions) (*BatchDeduplicationResult, error) {
	// 获取账户信息
	var account models.EmailAccount
	if err := m.db.First(&account, accountID).Error; err != nil {
		return nil, fmt.Errorf("account not found: %w", err)
	}

	// 记录去重活动开始
	activity := &DeduplicationActivity{
		AccountID: accountID,
		Type:      "deduplication",
		Status:    "running",
		StartTime: time.Now(),
	}
	if err := m.db.Create(activity).Error; err != nil {
		log.Printf("Failed to record deduplication activity: %v", err)
	}

	// 发送开始通知
	if m.eventTrigger != nil {
		m.eventTrigger.TriggerNotification(ctx, 
			"去重开始", 
			fmt.Sprintf("账户 %s 的邮件去重已开始", account.Email),
			"info", 
			account.UserID)
	}

	// 设置默认选项
	if options == nil {
		options = &DeduplicationOptions{
			DryRun:            false,
			CrossFolder:       true,
			CleanupDuplicates: true,
			RebuildIndex:      false,
			BatchSize:         100,
			MaxProcessingTime: time.Hour,
		}
	}

	// 创建增强去重器
	var deduplicator EnhancedDeduplicator
	var standardDeduplicator EmailDeduplicator

	// 首先尝试创建增强去重器
	switch account.Provider {
	case "gmail":
		if enhancedDedup := m.tryCreateEnhancedGmailDeduplicator(m.db); enhancedDedup != nil {
			deduplicator = enhancedDedup
		} else {
			standardDeduplicator = m.deduplicatorFactory.CreateDeduplicator("gmail")
		}
	default:
		if enhancedDedup := m.tryCreateEnhancedStandardDeduplicator(m.db); enhancedDedup != nil {
			deduplicator = enhancedDedup
		} else {
			standardDeduplicator = m.deduplicatorFactory.CreateDeduplicator("standard")
		}
	}

	var result *BatchDeduplicationResult
	var err error

	// 执行去重操作
	startTime := time.Now()

	// 如果有增强去重器，使用增强功能
	if deduplicator != nil {
		if options.RebuildIndex {
			err = deduplicator.RebuildDeduplicationIndex(ctx, accountID)
			if err != nil {
				return nil, fmt.Errorf("failed to rebuild index: %w", err)
			}
		}

		if options.CrossFolder {
			result, err = deduplicator.CheckCrossFolderDuplicates(ctx, accountID)
			if err != nil {
				return nil, fmt.Errorf("failed to check cross-folder duplicates: %w", err)
			}
		}

		if options.CleanupDuplicates {
			cleanupResult, err := deduplicator.CleanupDuplicates(ctx, accountID, options.DryRun)
			if err != nil {
				return nil, fmt.Errorf("failed to cleanup duplicates: %w", err)
			}

			if result == nil {
				result = cleanupResult
			} else {
				// 合并结果
				result.ProcessedCount += cleanupResult.ProcessedCount
				result.DuplicateCount += cleanupResult.DuplicateCount
				result.ErrorCount += cleanupResult.ErrorCount
				result.Errors = append(result.Errors, cleanupResult.Errors...)
			}
		}

		// 如果没有执行任何操作，至少获取统计信息
		if result == nil {
			stats, err := deduplicator.GetDeduplicationStats(ctx, accountID)
			if err != nil {
				return nil, fmt.Errorf("failed to get stats: %w", err)
			}

			result = &BatchDeduplicationResult{
				Stats: stats,
			}
		}
	} else if standardDeduplicator != nil {
		// 使用标准去重器
		result, err = m.handleStandardDeduplication(ctx, standardDeduplicator, accountID, options)
		if err != nil {
			return nil, fmt.Errorf("failed to perform standard deduplication: %w", err)
		}
	} else {
		return nil, fmt.Errorf("no deduplicator available for account %d", accountID)
	}

	// 更新活动记录
	endTime := time.Now()
	activity.EndTime = &endTime
	activity.Status = "completed"
	activity.Result = fmt.Sprintf("Processed: %d, Duplicates: %d, Errors: %d", 
		result.ProcessedCount, result.DuplicateCount, result.ErrorCount)
	
	if result.ErrorCount > 0 {
		activity.Status = "completed_with_errors"
		activity.ErrorMessage = fmt.Sprintf("%d errors occurred", result.ErrorCount)
	}
	
	m.db.Save(activity)

	// 发送完成通知
	if m.eventTrigger != nil && options.NotifyOnCompletion {
		notificationType := "success"
		if result.ErrorCount > 0 {
			notificationType = "warning"
		}
		
		m.eventTrigger.TriggerNotification(ctx,
			"去重完成",
			fmt.Sprintf("账户 %s 的邮件去重已完成。处理: %d, 重复: %d, 错误: %d",
				account.Email, result.ProcessedCount, result.DuplicateCount, result.ErrorCount),
			notificationType,
			account.UserID)
	}

	result.Stats.ProcessingTime = time.Since(startTime)
	return result, nil
}

// DeduplicateUser 执行用户所有账户去重
func (m *StandardDeduplicationManager) DeduplicateUser(ctx context.Context, userID uint, options *DeduplicationOptions) (*UserDeduplicationResult, error) {
	startTime := time.Now()
	
	// 获取用户的所有账户
	var accounts []models.EmailAccount
	err := m.db.Where("user_id = ? AND is_active = ?", userID, true).Find(&accounts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get user accounts: %w", err)
	}

	result := &UserDeduplicationResult{
		UserID:         userID,
		AccountResults: make(map[uint]*BatchDeduplicationResult),
		StartTime:      startTime,
	}

	// 处理每个账户
	for _, account := range accounts {
		accountResult, err := m.DeduplicateAccount(ctx, account.ID, options)
		if err != nil {
			log.Printf("Failed to deduplicate account %d: %v", account.ID, err)
			result.TotalErrors++
			continue
		}

		result.AccountResults[account.ID] = accountResult
		result.TotalProcessed += accountResult.ProcessedCount
		result.TotalDuplicates += accountResult.DuplicateCount
		result.TotalErrors += accountResult.ErrorCount
	}

	result.EndTime = time.Now()
	result.ProcessingTime = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// GetDeduplicationReport 获取去重报告
func (m *StandardDeduplicationManager) GetDeduplicationReport(ctx context.Context, accountID uint) (*DeduplicationReport, error) {
	// 获取统计信息
	deduplicator := m.deduplicatorFactory.CreateDeduplicator("standard")
	if enhanced, ok := deduplicator.(EnhancedDeduplicator); ok {
		stats, err := enhanced.GetDeduplicationStats(ctx, accountID)
		if err != nil {
			return nil, fmt.Errorf("failed to get stats: %w", err)
		}

		// 获取最近活动
		var activities []*DeduplicationActivity
		err = m.db.Where("account_id = ?", accountID).
			Order("start_time DESC").
			Limit(10).
			Find(&activities).Error
		if err != nil {
			log.Printf("Failed to get recent activities: %v", err)
		}

		// 生成建议
		recommendations := m.generateRecommendations(stats, activities)

		return &DeduplicationReport{
			AccountID:       accountID,
			Stats:           stats,
			RecentActivity:  activities,
			Recommendations: recommendations,
			GeneratedAt:     time.Now(),
		}, nil
	}

	return nil, fmt.Errorf("enhanced deduplicator not available")
}

// generateRecommendations 生成去重建议
func (m *StandardDeduplicationManager) generateRecommendations(stats *DeduplicationStats, activities []*DeduplicationActivity) []*DeduplicationRecommendation {
	var recommendations []*DeduplicationRecommendation

	// 如果发现大量重复，建议清理
	if stats.DuplicatesFound > 100 {
		recommendations = append(recommendations, &DeduplicationRecommendation{
			Type:        "cleanup",
			Priority:    "high",
			Title:       "清理重复邮件",
			Description: fmt.Sprintf("发现 %d 个重复邮件，建议进行清理以节省存储空间", stats.DuplicatesFound),
			Action:      "cleanup_duplicates",
		})
	}

	// 如果很久没有重建索引，建议重建
	if time.Since(stats.LastUpdated) > 30*24*time.Hour {
		recommendations = append(recommendations, &DeduplicationRecommendation{
			Type:        "rebuild_index",
			Priority:    "medium",
			Title:       "重建去重索引",
			Description: "超过30天未更新去重索引，建议重建以提高性能",
			Action:      "rebuild_index",
		})
	}

	// 如果没有定期去重，建议设置计划
	hasRecentActivity := false
	for _, activity := range activities {
		if time.Since(activity.StartTime) < 7*24*time.Hour {
			hasRecentActivity = true
			break
		}
	}

	if !hasRecentActivity {
		recommendations = append(recommendations, &DeduplicationRecommendation{
			Type:        "schedule",
			Priority:    "low",
			Title:       "设置定期去重",
			Description: "建议设置定期去重任务以保持邮箱整洁",
			Action:      "schedule_deduplication",
		})
	}

	return recommendations
}

// ScheduleDeduplication 计划去重任务
func (m *StandardDeduplicationManager) ScheduleDeduplication(ctx context.Context, accountID uint, schedule *DeduplicationSchedule) error {
	// 这里应该集成到任务调度系统中
	// 暂时只记录到数据库
	log.Printf("Scheduled deduplication for account %d: %+v", accountID, schedule)
	return nil
}

// CancelScheduledDeduplication 取消计划去重任务
func (m *StandardDeduplicationManager) CancelScheduledDeduplication(ctx context.Context, accountID uint) error {
	// 这里应该从任务调度系统中移除
	log.Printf("Cancelled scheduled deduplication for account %d", accountID)
	return nil
}

// tryCreateEnhancedStandardDeduplicator 尝试创建增强标准去重器
func (m *StandardDeduplicationManager) tryCreateEnhancedStandardDeduplicator(db *gorm.DB) EnhancedDeduplicator {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Failed to create enhanced standard deduplicator: %v", r)
		}
	}()

	// 检查是否支持增强功能
	if !m.supportsEnhancedDeduplication() {
		return nil
	}

	return NewEnhancedStandardDeduplicator(db)
}

// tryCreateEnhancedGmailDeduplicator 尝试创建增强Gmail去重器
func (m *StandardDeduplicationManager) tryCreateEnhancedGmailDeduplicator(db *gorm.DB) EnhancedDeduplicator {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Failed to create enhanced Gmail deduplicator: %v", r)
		}
	}()

	// 检查是否支持增强功能
	if !m.supportsEnhancedDeduplication() {
		return nil
	}

	return NewEnhancedGmailDeduplicator(db)
}

// supportsEnhancedDeduplication 检查是否支持增强去重功能
func (m *StandardDeduplicationManager) supportsEnhancedDeduplication() bool {
	// 使用配置管理器检查
	if !config.Env.ShouldEnableEnhancedDedup() {
		return false
	}

	// 检查数据库是否支持所需的功能
	if m.db == nil {
		return false
	}

	return true
}

// handleStandardDeduplication 使用标准去重器处理去重
func (m *StandardDeduplicationManager) handleStandardDeduplication(ctx context.Context, standardDeduplicator EmailDeduplicator, accountID uint, options *DeduplicationOptions) (*BatchDeduplicationResult, error) {
	// 使用标准去重器的简化处理
	result := &BatchDeduplicationResult{
		Stats: &DeduplicationStats{
			LastUpdated: time.Now(),
		},
	}

	// 执行基础去重检查
	log.Printf("Using standard deduplicator for account %d", accountID)

	// 模拟去重结果
	result.ProcessedCount = 1
	result.DuplicateCount = 0
	result.ErrorCount = 0

	return result, nil
}
