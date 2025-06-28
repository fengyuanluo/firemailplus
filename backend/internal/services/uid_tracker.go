package services

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"firemail/internal/models"
)

// UIDTracker UID跟踪器接口
type UIDTracker interface {
	// GetLastUID 获取文件夹的最后同步UID
	GetLastUID(ctx context.Context, folderID uint) (uint32, error)
	
	// UpdateLastUID 更新文件夹的最后同步UID
	UpdateLastUID(ctx context.Context, folderID uint, uid uint32) error
	
	// GetUIDRange 获取文件夹的UID范围
	GetUIDRange(ctx context.Context, folderID uint) (minUID, maxUID uint32, err error)
	
	// IsUIDProcessed 检查UID是否已处理
	IsUIDProcessed(ctx context.Context, folderID uint, uid uint32) (bool, error)
	
	// MarkUIDProcessed 标记UID为已处理
	MarkUIDProcessed(ctx context.Context, folderID uint, uid uint32) error
	
	// GetMissingUIDs 获取缺失的UID列表
	GetMissingUIDs(ctx context.Context, folderID uint, startUID, endUID uint32) ([]uint32, error)
}

// FolderSyncState 文件夹同步状态
type FolderSyncState struct {
	ID           uint      `gorm:"primaryKey"`
	FolderID     uint      `gorm:"uniqueIndex;not null"`
	LastUID      uint32    `gorm:"not null;default:0"`
	LastSyncAt   time.Time `gorm:"not null"`
	TotalEmails  int       `gorm:"not null;default:0"`
	SyncVersion  int       `gorm:"not null;default:1"` // 用于检测重置
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// UIDRange UID范围记录
type UIDRange struct {
	ID        uint      `gorm:"primaryKey"`
	FolderID  uint      `gorm:"index;not null"`
	StartUID  uint32    `gorm:"not null"`
	EndUID    uint32    `gorm:"not null"`
	IsGap     bool      `gorm:"not null;default:false"` // 是否为缺失范围
	CreatedAt time.Time
}

// DatabaseUIDTracker 基于数据库的UID跟踪器
type DatabaseUIDTracker struct {
	db *gorm.DB
}

// NewDatabaseUIDTracker 创建数据库UID跟踪器
func NewDatabaseUIDTracker(db *gorm.DB) *DatabaseUIDTracker {
	return &DatabaseUIDTracker{db: db}
}

// GetLastUID 获取文件夹的最后同步UID
func (t *DatabaseUIDTracker) GetLastUID(ctx context.Context, folderID uint) (uint32, error) {
	var state FolderSyncState
	err := t.db.WithContext(ctx).
		Where("folder_id = ?", folderID).
		First(&state).Error
	
	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get last UID: %w", err)
	}
	
	return state.LastUID, nil
}

// UpdateLastUID 更新文件夹的最后同步UID
func (t *DatabaseUIDTracker) UpdateLastUID(ctx context.Context, folderID uint, uid uint32) error {
	now := time.Now()
	
	// 使用 UPSERT 操作
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var state FolderSyncState
		err := tx.Where("folder_id = ?", folderID).First(&state).Error
		
		if err == gorm.ErrRecordNotFound {
			// 创建新记录
			state = FolderSyncState{
				FolderID:   folderID,
				LastUID:    uid,
				LastSyncAt: now,
				SyncVersion: 1,
			}
			return tx.Create(&state).Error
		} else if err != nil {
			return fmt.Errorf("failed to get sync state: %w", err)
		}
		
		// 更新现有记录
		if uid > state.LastUID {
			state.LastUID = uid
			state.LastSyncAt = now
			return tx.Save(&state).Error
		}
		
		return nil
	})
}

// GetUIDRange 获取文件夹的UID范围
func (t *DatabaseUIDTracker) GetUIDRange(ctx context.Context, folderID uint) (minUID, maxUID uint32, err error) {
	var result struct {
		MinUID uint32
		MaxUID uint32
	}
	
	err = t.db.WithContext(ctx).
		Model(&models.Email{}).
		Select("MIN(uid) as min_uid, MAX(uid) as max_uid").
		Where("folder_id = ?", folderID).
		Scan(&result).Error
	
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get UID range: %w", err)
	}
	
	return result.MinUID, result.MaxUID, nil
}

// IsUIDProcessed 检查UID是否已处理
func (t *DatabaseUIDTracker) IsUIDProcessed(ctx context.Context, folderID uint, uid uint32) (bool, error) {
	var count int64
	err := t.db.WithContext(ctx).
		Model(&models.Email{}).
		Where("folder_id = ? AND uid = ?", folderID, uid).
		Count(&count).Error
	
	if err != nil {
		return false, fmt.Errorf("failed to check UID: %w", err)
	}
	
	return count > 0, nil
}

// MarkUIDProcessed 标记UID为已处理
func (t *DatabaseUIDTracker) MarkUIDProcessed(ctx context.Context, folderID uint, uid uint32) error {
	// 这个方法在当前实现中不需要，因为邮件创建时就标记了UID
	// 但保留接口以备将来扩展
	return nil
}

// GetMissingUIDs 获取缺失的UID列表
func (t *DatabaseUIDTracker) GetMissingUIDs(ctx context.Context, folderID uint, startUID, endUID uint32) ([]uint32, error) {
	if startUID >= endUID {
		return []uint32{}, nil
	}
	
	// 获取已存在的UID列表
	var existingUIDs []uint32
	err := t.db.WithContext(ctx).
		Model(&models.Email{}).
		Where("folder_id = ? AND uid BETWEEN ? AND ?", folderID, startUID, endUID).
		Pluck("uid", &existingUIDs).Error
	
	if err != nil {
		return nil, fmt.Errorf("failed to get existing UIDs: %w", err)
	}
	
	// 创建UID映射
	uidMap := make(map[uint32]bool)
	for _, uid := range existingUIDs {
		uidMap[uid] = true
	}
	
	// 找出缺失的UID
	var missingUIDs []uint32
	for uid := startUID; uid <= endUID; uid++ {
		if !uidMap[uid] {
			missingUIDs = append(missingUIDs, uid)
		}
	}
	
	return missingUIDs, nil
}

// InMemoryUIDTracker 内存UID跟踪器（用于测试和临时使用）
type InMemoryUIDTracker struct {
	lastUIDs    map[uint]uint32
	processedUIDs map[uint]map[uint32]bool
}

// NewInMemoryUIDTracker 创建内存UID跟踪器
func NewInMemoryUIDTracker() *InMemoryUIDTracker {
	return &InMemoryUIDTracker{
		lastUIDs:      make(map[uint]uint32),
		processedUIDs: make(map[uint]map[uint32]bool),
	}
}

// GetLastUID 获取文件夹的最后同步UID
func (t *InMemoryUIDTracker) GetLastUID(ctx context.Context, folderID uint) (uint32, error) {
	return t.lastUIDs[folderID], nil
}

// UpdateLastUID 更新文件夹的最后同步UID
func (t *InMemoryUIDTracker) UpdateLastUID(ctx context.Context, folderID uint, uid uint32) error {
	if uid > t.lastUIDs[folderID] {
		t.lastUIDs[folderID] = uid
	}
	return nil
}

// GetUIDRange 获取文件夹的UID范围
func (t *InMemoryUIDTracker) GetUIDRange(ctx context.Context, folderID uint) (minUID, maxUID uint32, err error) {
	folderUIDs := t.processedUIDs[folderID]
	if len(folderUIDs) == 0 {
		return 0, 0, nil
	}
	
	var min, max uint32 = ^uint32(0), 0
	for uid := range folderUIDs {
		if uid < min {
			min = uid
		}
		if uid > max {
			max = uid
		}
	}
	
	return min, max, nil
}

// IsUIDProcessed 检查UID是否已处理
func (t *InMemoryUIDTracker) IsUIDProcessed(ctx context.Context, folderID uint, uid uint32) (bool, error) {
	folderUIDs := t.processedUIDs[folderID]
	if folderUIDs == nil {
		return false, nil
	}
	return folderUIDs[uid], nil
}

// MarkUIDProcessed 标记UID为已处理
func (t *InMemoryUIDTracker) MarkUIDProcessed(ctx context.Context, folderID uint, uid uint32) error {
	if t.processedUIDs[folderID] == nil {
		t.processedUIDs[folderID] = make(map[uint32]bool)
	}
	t.processedUIDs[folderID][uid] = true
	return nil
}

// GetMissingUIDs 获取缺失的UID列表
func (t *InMemoryUIDTracker) GetMissingUIDs(ctx context.Context, folderID uint, startUID, endUID uint32) ([]uint32, error) {
	folderUIDs := t.processedUIDs[folderID]
	if folderUIDs == nil {
		// 所有UID都缺失
		var missing []uint32
		for uid := startUID; uid <= endUID; uid++ {
			missing = append(missing, uid)
		}
		return missing, nil
	}
	
	var missing []uint32
	for uid := startUID; uid <= endUID; uid++ {
		if !folderUIDs[uid] {
			missing = append(missing, uid)
		}
	}
	
	return missing, nil
}

// UIDGapDetector UID缺口检测器
type UIDGapDetector struct {
	tracker UIDTracker
	db      *gorm.DB
}

// NewUIDGapDetector 创建UID缺口检测器
func NewUIDGapDetector(tracker UIDTracker, db *gorm.DB) *UIDGapDetector {
	return &UIDGapDetector{
		tracker: tracker,
		db:      db,
	}
}

// DetectGaps 检测UID缺口
func (d *UIDGapDetector) DetectGaps(ctx context.Context, folderID uint) ([]UIDRange, error) {
	minUID, maxUID, err := d.tracker.GetUIDRange(ctx, folderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get UID range: %w", err)
	}
	
	if minUID == 0 && maxUID == 0 {
		return []UIDRange{}, nil
	}
	
	// 检测缺失的UID
	missingUIDs, err := d.tracker.GetMissingUIDs(ctx, folderID, minUID, maxUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get missing UIDs: %w", err)
	}
	
	// 将连续的缺失UID合并为范围
	return d.mergeConsecutiveUIDs(folderID, missingUIDs), nil
}

// mergeConsecutiveUIDs 将连续的UID合并为范围
func (d *UIDGapDetector) mergeConsecutiveUIDs(folderID uint, uids []uint32) []UIDRange {
	if len(uids) == 0 {
		return []UIDRange{}
	}
	
	var ranges []UIDRange
	start := uids[0]
	end := uids[0]
	
	for i := 1; i < len(uids); i++ {
		if uids[i] == end+1 {
			end = uids[i]
		} else {
			ranges = append(ranges, UIDRange{
				FolderID: folderID,
				StartUID: start,
				EndUID:   end,
				IsGap:    true,
			})
			start = uids[i]
			end = uids[i]
		}
	}
	
	// 添加最后一个范围
	ranges = append(ranges, UIDRange{
		FolderID: folderID,
		StartUID: start,
		EndUID:   end,
		IsGap:    true,
	})
	
	return ranges
}
