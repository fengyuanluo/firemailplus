package services

import (
	"context"
	"errors"
	"fmt"

	"firemail/internal/models"

	"gorm.io/gorm"
)

var ErrEmailGroupInvariantViolation = errors.New("email group invariant violation")

func RepairAllEmailGroupInvariants(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	var userIDs []uint
	if err := db.WithContext(ctx).
		Model(&models.User{}).
		Order("id ASC").
		Pluck("id", &userIDs).Error; err != nil {
		return fmt.Errorf("failed to list users for email group repair: %w", err)
	}

	for _, userID := range userIDs {
		if err := RepairEmailGroupInvariantsForUser(ctx, db, userID); err != nil {
			return err
		}
	}

	return nil
}

func RepairEmailGroupInvariantsForUser(ctx context.Context, db *gorm.DB, userID uint) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var groups []models.EmailGroup
		if err := tx.Where("user_id = ?", userID).
			Order("is_default DESC, sort_order ASC, id ASC").
			Find(&groups).Error; err != nil {
			return fmt.Errorf("failed to load email groups for repair: %w", err)
		}

		placeholderGroup := findDefaultPlaceholderGroup(groups)

		if placeholderGroup == nil {
			placeholderGroup = findHistoricalDefaultPlaceholderGroup(groups)
		}

		if placeholderGroup != nil && !isDefaultPlaceholderSystemGroup(placeholderGroup) {
			defaultKey := models.EmailGroupSystemKeyDefaultPlaceholder
			if err := tx.Model(&models.EmailGroup{}).
				Where("id = ?", placeholderGroup.ID).
				Update("system_key", defaultKey).Error; err != nil {
				return fmt.Errorf("failed to backfill system placeholder key: %w", err)
			}
			placeholderGroup.SystemKey = &defaultKey
		}

		defaultGroup := chooseCanonicalDefaultGroup(groups)
		if defaultGroup == nil {
			if placeholderGroup != nil {
				if err := tx.Model(&models.EmailGroup{}).
					Where("id = ?", placeholderGroup.ID).
					Updates(map[string]interface{}{
						"is_default": true,
						"sort_order": 0,
					}).Error; err != nil {
					return fmt.Errorf("failed to promote placeholder group as default: %w", err)
				}
				placeholderGroup.IsDefault = true
				placeholderGroup.SortOrder = 0
				defaultGroup = placeholderGroup
			} else {
				defaultKey := models.EmailGroupSystemKeyDefaultPlaceholder
				created := &models.EmailGroup{
					UserID:    userID,
					Name:      "未分组",
					SortOrder: 0,
					IsDefault: true,
					SystemKey: &defaultKey,
				}
				if err := tx.Create(created).Error; err != nil {
					return fmt.Errorf("failed to create default placeholder group: %w", err)
				}
				defaultGroup = created
			}
		}

		sourceGroupIDs := collectSystemManagedSourceGroupIDs(groups, defaultGroup.ID)

		// 若存在多个默认分组，保留 canonical default，其他取消默认标记
		if err := tx.Model(&models.EmailGroup{}).
			Where("user_id = ? AND id != ? AND is_default = 1", userID, defaultGroup.ID).
			Update("is_default", false).Error; err != nil {
			return fmt.Errorf("failed to normalize default group flags: %w", err)
		}

		if !defaultGroup.IsDefault || defaultGroup.SortOrder != 0 {
			if err := tx.Model(&models.EmailGroup{}).
				Where("id = ?", defaultGroup.ID).
				Updates(map[string]interface{}{
					"is_default": true,
					"sort_order": 0,
				}).Error; err != nil {
				return fmt.Errorf("failed to normalize canonical default group: %w", err)
			}
		}

		if err := tx.Model(&models.EmailAccount{}).
			Where("user_id = ? AND group_id IS NULL", userID).
			Update("group_id", defaultGroup.ID).Error; err != nil {
			return fmt.Errorf("failed to repair ungrouped email accounts: %w", err)
		}

		if len(sourceGroupIDs) > 0 {
			if err := tx.Model(&models.EmailAccount{}).
				Where("user_id = ? AND group_id IN ?", userID, sourceGroupIDs).
				Update("group_id", defaultGroup.ID).Error; err != nil {
				return fmt.Errorf("failed to move accounts from demoted system/default groups: %w", err)
			}
		}

		return nil
	})
}

func ValidateEmailGroupInvariantsForUser(ctx context.Context, db *gorm.DB, userID uint) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	var defaultCount int64
	if err := db.WithContext(ctx).
		Model(&models.EmailGroup{}).
		Where("user_id = ? AND is_default = 1", userID).
		Count(&defaultCount).Error; err != nil {
		return fmt.Errorf("failed to validate default email group count: %w", err)
	}
	if defaultCount != 1 {
		return fmt.Errorf("%w: 默认分组数量异常（期望 1，实际 %d）", ErrEmailGroupInvariantViolation, defaultCount)
	}

	var nilGroupAccountCount int64
	if err := db.WithContext(ctx).
		Model(&models.EmailAccount{}).
		Where("user_id = ? AND group_id IS NULL", userID).
		Count(&nilGroupAccountCount).Error; err != nil {
		return fmt.Errorf("failed to validate email account group linkage: %w", err)
	}
	if nilGroupAccountCount != 0 {
		return fmt.Errorf("%w: 存在 %d 个邮箱账户未绑定分组", ErrEmailGroupInvariantViolation, nilGroupAccountCount)
	}

	var placeholderCount int64
	if err := db.WithContext(ctx).
		Model(&models.EmailGroup{}).
		Where("user_id = ? AND system_key = ?", userID, models.EmailGroupSystemKeyDefaultPlaceholder).
		Count(&placeholderCount).Error; err != nil {
		return fmt.Errorf("failed to validate placeholder email group count: %w", err)
	}
	if placeholderCount > 1 {
		return fmt.Errorf("%w: 系统占位分组数量异常（期望至多 1，实际 %d）", ErrEmailGroupInvariantViolation, placeholderCount)
	}

	var hiddenSystemGroupAccountCount int64
	if err := db.WithContext(ctx).
		Model(&models.EmailAccount{}).
		Joins("JOIN email_groups ON email_groups.id = email_accounts.group_id").
		Where("email_accounts.user_id = ? AND email_groups.system_key IS NOT NULL AND email_groups.is_default = 0", userID).
		Count(&hiddenSystemGroupAccountCount).Error; err != nil {
		return fmt.Errorf("failed to validate hidden system group assignments: %w", err)
	}
	if hiddenSystemGroupAccountCount != 0 {
		return fmt.Errorf("%w: 存在 %d 个邮箱账户挂在隐藏系统分组上", ErrEmailGroupInvariantViolation, hiddenSystemGroupAccountCount)
	}

	return nil
}

func chooseCanonicalDefaultGroup(groups []models.EmailGroup) *models.EmailGroup {
	var firstDefault *models.EmailGroup
	for i := range groups {
		group := &groups[i]
		if !group.IsDefault {
			continue
		}
		if firstDefault == nil {
			firstDefault = group
		}
		if !group.IsSystemGroup() {
			return group
		}
	}
	return firstDefault
}

func findDefaultPlaceholderGroup(groups []models.EmailGroup) *models.EmailGroup {
	for i := range groups {
		group := &groups[i]
		if isDefaultPlaceholderSystemGroup(group) {
			return group
		}
	}
	return nil
}

func findHistoricalDefaultPlaceholderGroup(groups []models.EmailGroup) *models.EmailGroup {
	var candidate *models.EmailGroup
	for i := range groups {
		group := &groups[i]
		if !isHistoricalDefaultPlaceholderName(group.Name) {
			continue
		}

		if group.IsDefault {
			return group
		}

		if candidate == nil ||
			group.SortOrder < candidate.SortOrder ||
			(group.SortOrder == candidate.SortOrder && group.ID < candidate.ID) {
			candidate = group
		}
	}
	return candidate
}

func collectSystemManagedSourceGroupIDs(groups []models.EmailGroup, defaultGroupID uint) []uint {
	sourceGroupIDs := make([]uint, 0)
	for _, group := range groups {
		if group.ID == defaultGroupID {
			continue
		}
		if group.IsDefault || group.IsSystemGroup() {
			sourceGroupIDs = append(sourceGroupIDs, group.ID)
		}
	}
	return sourceGroupIDs
}

func isDefaultPlaceholderSystemGroup(group *models.EmailGroup) bool {
	return group != nil && group.SystemKey != nil && *group.SystemKey == models.EmailGroupSystemKeyDefaultPlaceholder
}

func isHistoricalDefaultPlaceholderName(name string) bool {
	switch name {
	case "未分组", "未分":
		return true
	default:
		return false
	}
}
