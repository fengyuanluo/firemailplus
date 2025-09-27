package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"firemail/internal/models"

	"gorm.io/gorm"
)

// findAccountGroup 获取指定用户的分组
func (s *EmailServiceImpl) findAccountGroup(db *gorm.DB, userID, groupID uint) (*models.EmailAccountGroup, error) {
	var group models.EmailAccountGroup
	if err := db.Where("id = ? AND user_id = ?", groupID, userID).First(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("account group not found")
		}
		return nil, err
	}
	return &group, nil
}

// nextAccountSortOrderDB 计算下一个账号排序值
func (s *EmailServiceImpl) nextAccountSortOrderDB(db *gorm.DB, userID uint, groupID *uint) (int, error) {
	var maxOrder sql.NullInt64
	query := db.Model(&models.EmailAccount{}).Where("user_id = ?", userID)
	if groupID == nil {
		query = query.Where("group_id IS NULL")
	} else {
		query = query.Where("group_id = ?", *groupID)
	}
	if err := query.Select("MAX(sort_order)").Scan(&maxOrder).Error; err != nil {
		return 0, err
	}
	if maxOrder.Valid {
		return int(maxOrder.Int64) + 1, nil
	}
	return 0, nil
}

// nextGroupSortOrder 计算下一个分组排序值
func (s *EmailServiceImpl) nextGroupSortOrder(db *gorm.DB, userID uint) (int, error) {
	var maxOrder sql.NullInt64
	if err := db.Model(&models.EmailAccountGroup{}).
		Where("user_id = ?", userID).
		Select("MAX(sort_order)").
		Scan(&maxOrder).Error; err != nil {
		return 0, err
	}
	if maxOrder.Valid {
		return int(maxOrder.Int64) + 1, nil
	}
	return 0, nil
}

// GetAccountGroups 获取用户的邮箱分组列表
func (s *EmailServiceImpl) GetAccountGroups(ctx context.Context, userID uint) ([]*models.EmailAccountGroup, error) {
	var groups []*models.EmailAccountGroup
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("sort_order ASC, created_at ASC").
		Find(&groups).Error; err != nil {
		return nil, fmt.Errorf("failed to get account groups: %w", err)
	}
	return groups, nil
}

// CreateAccountGroup 创建新的邮箱分组
func (s *EmailServiceImpl) CreateAccountGroup(ctx context.Context, userID uint, req *CreateAccountGroupRequest) (*models.EmailAccountGroup, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("group name is required")
	}

	group := &models.EmailAccountGroup{
		UserID: userID,
		Name:   name,
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		order, err := s.nextGroupSortOrder(tx, userID)
		if err != nil {
			return err
		}
		group.SortOrder = order
		return tx.Create(group).Error
	}); err != nil {
		return nil, fmt.Errorf("failed to create account group: %w", err)
	}

	return group, nil
}

// UpdateAccountGroup 更新邮箱分组信息
func (s *EmailServiceImpl) UpdateAccountGroup(ctx context.Context, userID, groupID uint, req *UpdateAccountGroupRequest) (*models.EmailAccountGroup, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}

	var group *models.EmailAccountGroup
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		group, err = s.findAccountGroup(tx, userID, groupID)
		if err != nil {
			return err
		}
		if req.Name != nil {
			name := strings.TrimSpace(*req.Name)
			if name == "" {
				return fmt.Errorf("group name cannot be empty")
			}
			group.Name = name
		}
		if req.SortOrder != nil {
			group.SortOrder = *req.SortOrder
		}
		return tx.Save(group).Error
	}); err != nil {
		return nil, fmt.Errorf("failed to update account group: %w", err)
	}

	return group, nil
}

// DeleteAccountGroup 删除邮箱分组
func (s *EmailServiceImpl) DeleteAccountGroup(ctx context.Context, userID, groupID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := s.findAccountGroup(tx, userID, groupID); err != nil {
			return err
		}

		var accounts []models.EmailAccount
		if err := tx.Where("user_id = ? AND group_id = ?", userID, groupID).
			Order("sort_order ASC, id ASC").
			Find(&accounts).Error; err != nil {
			return err
		}

		if len(accounts) > 0 {
			nextOrder, err := s.nextAccountSortOrderDB(tx, userID, nil)
			if err != nil {
				return err
			}
			for idx, acc := range accounts {
				update := map[string]interface{}{
					"group_id":   nil,
					"sort_order": nextOrder + idx,
				}
				if err := tx.Model(&models.EmailAccount{}).
					Where("id = ? AND user_id = ?", acc.ID, userID).
					Updates(update).Error; err != nil {
					return err
				}
			}
		}

		if err := tx.Delete(&models.EmailAccountGroup{}, groupID).Error; err != nil {
			return err
		}
		return nil
	})
}

// ReorderAccountGroups 批量更新分组排序
func (s *EmailServiceImpl) ReorderAccountGroups(ctx context.Context, userID uint, orders []AccountGroupOrder) error {
	if len(orders) == 0 {
		return nil
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, order := range orders {
			res := tx.Model(&models.EmailAccountGroup{}).
				Where("id = ? AND user_id = ?", order.ID, userID).
				Update("sort_order", order.SortOrder)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return fmt.Errorf("account group not found")
			}
		}
		return nil
	})
}

// MoveAccountsToGroup 批量移动邮箱账户到指定分组
func (s *EmailServiceImpl) MoveAccountsToGroup(ctx context.Context, userID uint, req *MoveAccountsToGroupRequest) error {
	if req == nil || len(req.AccountIDs) == 0 {
		return fmt.Errorf("account_ids is required")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if req.GroupID != nil {
			if _, err := s.findAccountGroup(tx, userID, *req.GroupID); err != nil {
				return err
			}
		}

		nextOrder, err := s.nextAccountSortOrderDB(tx, userID, req.GroupID)
		if err != nil {
			return err
		}

		for idx, accountID := range req.AccountIDs {
			update := map[string]interface{}{
				"sort_order": nextOrder + idx,
			}
			if req.GroupID == nil {
				update["group_id"] = nil
			} else {
				update["group_id"] = *req.GroupID
			}

			res := tx.Model(&models.EmailAccount{}).
				Where("id = ? AND user_id = ?", accountID, userID).
				Updates(update)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return fmt.Errorf("email account not found")
			}
		}

		return nil
	})
}

// ReorderAccounts 批量更新邮箱账户排序
func (s *EmailServiceImpl) ReorderAccounts(ctx context.Context, userID uint, orders []AccountOrder) error {
	if len(orders) == 0 {
		return nil
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, order := range orders {
			res := tx.Model(&models.EmailAccount{}).
				Where("id = ? AND user_id = ?", order.AccountID, userID).
				Update("sort_order", order.SortOrder)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return fmt.Errorf("email account not found")
			}
		}
		return nil
	})
}
