package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"firemail/internal/sse"
)

// SyncProgressTracker 同步进度跟踪器
type SyncProgressTracker struct {
	eventPublisher sse.EventPublisher
	progressMap    sync.Map // map[string]*SyncProgress
	mu             sync.RWMutex
}

// SyncProgress 同步进度
type SyncProgress struct {
	SyncID          string                 `json:"sync_id"`
	AccountID       uint                   `json:"account_id"`
	UserID          uint                   `json:"user_id"`
	Status          SyncStatus             `json:"status"`
	StartTime       time.Time              `json:"start_time"`
	EndTime         *time.Time             `json:"end_time,omitempty"`
	TotalFolders    int                    `json:"total_folders"`
	ProcessedFolders int                   `json:"processed_folders"`
	TotalEmails     int                    `json:"total_emails"`
	ProcessedEmails int                    `json:"processed_emails"`
	NewEmails       int                    `json:"new_emails"`
	UpdatedEmails   int                    `json:"updated_emails"`
	Errors          []string               `json:"errors"`
	FolderProgress  map[string]*FolderProgress `json:"folder_progress"`
	LastUpdateTime  time.Time              `json:"last_update_time"`
}

// FolderProgress 文件夹同步进度
type FolderProgress struct {
	FolderID        uint       `json:"folder_id"`
	FolderName      string     `json:"folder_name"`
	Status          SyncStatus `json:"status"`
	TotalEmails     int        `json:"total_emails"`
	ProcessedEmails int        `json:"processed_emails"`
	NewEmails       int        `json:"new_emails"`
	UpdatedEmails   int        `json:"updated_emails"`
	StartTime       time.Time  `json:"start_time"`
	EndTime         *time.Time `json:"end_time,omitempty"`
	Error           string     `json:"error,omitempty"`
}

// SyncStatus 同步状态
type SyncStatus string

const (
	SyncStatusPending    SyncStatus = "pending"
	SyncStatusRunning    SyncStatus = "running"
	SyncStatusCompleted  SyncStatus = "completed"
	SyncStatusFailed     SyncStatus = "failed"
	SyncStatusCancelled  SyncStatus = "cancelled"
)

// NewSyncProgressTracker 创建同步进度跟踪器
func NewSyncProgressTracker(eventPublisher sse.EventPublisher) *SyncProgressTracker {
	return &SyncProgressTracker{
		eventPublisher: eventPublisher,
	}
}

// StartSync 开始同步
func (t *SyncProgressTracker) StartSync(ctx context.Context, syncID string, accountID, userID uint, totalFolders int) *SyncProgress {
	progress := &SyncProgress{
		SyncID:         syncID,
		AccountID:      accountID,
		UserID:         userID,
		Status:         SyncStatusRunning,
		StartTime:      time.Now(),
		TotalFolders:   totalFolders,
		FolderProgress: make(map[string]*FolderProgress),
		LastUpdateTime: time.Now(),
	}

	t.progressMap.Store(syncID, progress)
	t.publishProgress(ctx, progress)
	return progress
}

// UpdateSyncProgress 更新同步进度
func (t *SyncProgressTracker) UpdateSyncProgress(ctx context.Context, syncID string, update func(*SyncProgress)) {
	if value, ok := t.progressMap.Load(syncID); ok {
		progress := value.(*SyncProgress)
		t.mu.Lock()
		update(progress)
		progress.LastUpdateTime = time.Now()
		t.mu.Unlock()
		t.publishProgress(ctx, progress)
	}
}

// StartFolderSync 开始文件夹同步
func (t *SyncProgressTracker) StartFolderSync(ctx context.Context, syncID string, folderID uint, folderName string, totalEmails int) {
	t.UpdateSyncProgress(ctx, syncID, func(progress *SyncProgress) {
		folderKey := fmt.Sprintf("%d", folderID)
		progress.FolderProgress[folderKey] = &FolderProgress{
			FolderID:    folderID,
			FolderName:  folderName,
			Status:      SyncStatusRunning,
			TotalEmails: totalEmails,
			StartTime:   time.Now(),
		}
	})
}

// UpdateFolderProgress 更新文件夹进度
func (t *SyncProgressTracker) UpdateFolderProgress(ctx context.Context, syncID string, folderID uint, processedEmails, newEmails, updatedEmails int) {
	t.UpdateSyncProgress(ctx, syncID, func(progress *SyncProgress) {
		folderKey := fmt.Sprintf("%d", folderID)
		if folderProgress, exists := progress.FolderProgress[folderKey]; exists {
			folderProgress.ProcessedEmails = processedEmails
			folderProgress.NewEmails = newEmails
			folderProgress.UpdatedEmails = updatedEmails
		}
		
		// 更新总体进度
		progress.ProcessedEmails += processedEmails
		progress.NewEmails += newEmails
		progress.UpdatedEmails += updatedEmails
	})
}

// CompleteFolderSync 完成文件夹同步
func (t *SyncProgressTracker) CompleteFolderSync(ctx context.Context, syncID string, folderID uint, err error) {
	t.UpdateSyncProgress(ctx, syncID, func(progress *SyncProgress) {
		folderKey := fmt.Sprintf("%d", folderID)
		if folderProgress, exists := progress.FolderProgress[folderKey]; exists {
			now := time.Now()
			folderProgress.EndTime = &now
			
			if err != nil {
				folderProgress.Status = SyncStatusFailed
				folderProgress.Error = err.Error()
				progress.Errors = append(progress.Errors, fmt.Sprintf("Folder %s: %v", folderProgress.FolderName, err))
			} else {
				folderProgress.Status = SyncStatusCompleted
			}
		}
		
		progress.ProcessedFolders++
	})
}

// CompleteSync 完成同步
func (t *SyncProgressTracker) CompleteSync(ctx context.Context, syncID string, err error) {
	t.UpdateSyncProgress(ctx, syncID, func(progress *SyncProgress) {
		now := time.Now()
		progress.EndTime = &now
		
		if err != nil {
			progress.Status = SyncStatusFailed
			progress.Errors = append(progress.Errors, err.Error())
		} else {
			progress.Status = SyncStatusCompleted
		}
	})
	
	// 延迟清理进度数据
	go func() {
		time.Sleep(5 * time.Minute)
		t.progressMap.Delete(syncID)
	}()
}

// GetProgress 获取同步进度
func (t *SyncProgressTracker) GetProgress(syncID string) (*SyncProgress, bool) {
	if value, ok := t.progressMap.Load(syncID); ok {
		progress := value.(*SyncProgress)
		t.mu.RLock()
		defer t.mu.RUnlock()
		
		// 返回副本以避免并发修改
		progressCopy := *progress
		progressCopy.FolderProgress = make(map[string]*FolderProgress)
		for k, v := range progress.FolderProgress {
			folderCopy := *v
			progressCopy.FolderProgress[k] = &folderCopy
		}
		
		return &progressCopy, true
	}
	return nil, false
}

// GetAllProgress 获取所有同步进度
func (t *SyncProgressTracker) GetAllProgress() map[string]*SyncProgress {
	result := make(map[string]*SyncProgress)
	
	t.progressMap.Range(func(key, value interface{}) bool {
		syncID := key.(string)
		progress := value.(*SyncProgress)
		
		t.mu.RLock()
		progressCopy := *progress
		progressCopy.FolderProgress = make(map[string]*FolderProgress)
		for k, v := range progress.FolderProgress {
			folderCopy := *v
			progressCopy.FolderProgress[k] = &folderCopy
		}
		t.mu.RUnlock()
		
		result[syncID] = &progressCopy
		return true
	})
	
	return result
}

// publishProgress 发布进度事件
func (t *SyncProgressTracker) publishProgress(ctx context.Context, progress *SyncProgress) {
	if t.eventPublisher == nil {
		return
	}
	
	// 创建进度事件
	event := &sse.Event{
		Type: "sync_progress",
		Data: progress,
	}
	
	// 发布到用户
	if err := t.eventPublisher.PublishToUser(ctx, progress.UserID, event); err != nil {
		// 记录错误但不影响同步
		fmt.Printf("Failed to publish sync progress: %v\n", err)
	}
}

// SyncMetrics 同步指标
type SyncMetrics struct {
	TotalSyncs      int64         `json:"total_syncs"`
	SuccessfulSyncs int64         `json:"successful_syncs"`
	FailedSyncs     int64         `json:"failed_syncs"`
	AverageDuration time.Duration `json:"average_duration"`
	TotalEmails     int64         `json:"total_emails"`
	TotalErrors     int64         `json:"total_errors"`
}

// MetricsCollector 指标收集器
type MetricsCollector struct {
	metrics sync.Map // map[uint]*AccountMetrics
}

// AccountMetrics 账户指标
type AccountMetrics struct {
	AccountID       uint          `json:"account_id"`
	TotalSyncs      int64         `json:"total_syncs"`
	SuccessfulSyncs int64         `json:"successful_syncs"`
	FailedSyncs     int64         `json:"failed_syncs"`
	LastSyncTime    time.Time     `json:"last_sync_time"`
	AverageDuration time.Duration `json:"average_duration"`
	TotalEmails     int64         `json:"total_emails"`
	TotalErrors     int64         `json:"total_errors"`
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{}
}

// RecordSyncStart 记录同步开始
func (c *MetricsCollector) RecordSyncStart(accountID uint) {
	c.getOrCreateMetrics(accountID).TotalSyncs++
}

// RecordSyncComplete 记录同步完成
func (c *MetricsCollector) RecordSyncComplete(accountID uint, duration time.Duration, emailCount int, errorCount int, success bool) {
	metrics := c.getOrCreateMetrics(accountID)
	
	if success {
		metrics.SuccessfulSyncs++
	} else {
		metrics.FailedSyncs++
	}
	
	metrics.LastSyncTime = time.Now()
	metrics.TotalEmails += int64(emailCount)
	metrics.TotalErrors += int64(errorCount)
	
	// 计算平均持续时间
	if metrics.SuccessfulSyncs > 0 {
		totalDuration := time.Duration(metrics.SuccessfulSyncs) * metrics.AverageDuration + duration
		metrics.AverageDuration = totalDuration / time.Duration(metrics.SuccessfulSyncs)
	}
}

// GetMetrics 获取账户指标
func (c *MetricsCollector) GetMetrics(accountID uint) (*AccountMetrics, bool) {
	if value, ok := c.metrics.Load(accountID); ok {
		metrics := value.(*AccountMetrics)
		metricsCopy := *metrics
		return &metricsCopy, true
	}
	return nil, false
}

// GetAllMetrics 获取所有指标
func (c *MetricsCollector) GetAllMetrics() map[uint]*AccountMetrics {
	result := make(map[uint]*AccountMetrics)
	
	c.metrics.Range(func(key, value interface{}) bool {
		accountID := key.(uint)
		metrics := value.(*AccountMetrics)
		metricsCopy := *metrics
		result[accountID] = &metricsCopy
		return true
	})
	
	return result
}

// getOrCreateMetrics 获取或创建账户指标
func (c *MetricsCollector) getOrCreateMetrics(accountID uint) *AccountMetrics {
	if value, ok := c.metrics.Load(accountID); ok {
		return value.(*AccountMetrics)
	}
	
	metrics := &AccountMetrics{
		AccountID: accountID,
	}
	c.metrics.Store(accountID, metrics)
	return metrics
}

// generateSyncID 生成同步ID
func generateSyncID(accountID uint) string {
	return fmt.Sprintf("sync_%d_%d", accountID, time.Now().UnixNano())
}
