package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"firemail/internal/models"

	"gorm.io/gorm"
)

var ErrEmailAccountAlreadyExists = errors.New("email account already exists")

func duplicateEmailAccountError() error {
	return fmt.Errorf("%w: 该邮箱账户已存在", ErrEmailAccountAlreadyExists)
}

// EnsureEmailAccountUnique 检查同一用户下 email + provider 是否已存在。
func EnsureEmailAccountUnique(ctx context.Context, db *gorm.DB, userID uint, email, provider string) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	var existing models.EmailAccount
	err := db.WithContext(ctx).
		Where("user_id = ? AND email = ? AND provider = ?", userID, email, provider).
		First(&existing).Error
	if err == nil {
		return duplicateEmailAccountError()
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	return fmt.Errorf("failed to check duplicate email account: %w", err)
}

// NormalizeEmailAccountCreateError 将数据库唯一约束错误映射为统一的重复账户错误。
func NormalizeEmailAccountCreateError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrEmailAccountAlreadyExists) {
		return err
	}
	if isEmailAccountUniqueConstraintError(err) {
		return duplicateEmailAccountError()
	}
	return err
}

func isEmailAccountUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "idx_email_accounts_user_email_provider_unique") ||
		strings.Contains(message, "unique constraint failed: email_accounts.user_id, email_accounts.email, email_accounts.provider")
}
