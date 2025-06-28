package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"firemail/internal/models"

	"gorm.io/gorm"
)

// DataRepairService 数据修复服务
type DataRepairService struct {
	db *gorm.DB
}

// NewDataRepairService 创建数据修复服务
func NewDataRepairService(db *gorm.DB) *DataRepairService {
	return &DataRepairService{
		db: db,
	}
}

// RepairDuplicateEmails 修复重复邮件
func (s *DataRepairService) RepairDuplicateEmails(ctx context.Context, dryRun bool) (*RepairResult, error) {
	result := &RepairResult{
		StartTime: time.Now(),
		DryRun:    dryRun,
	}

	log.Printf("Starting duplicate email repair (dry run: %v)", dryRun)

	// 1. 查找MessageID重复的邮件
	messageIDDuplicates, err := s.findMessageIDDuplicates(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find MessageID duplicates: %w", err)
	}
	result.MessageIDDuplicates = len(messageIDDuplicates)

	// 2. 查找UID重复的邮件
	uidDuplicates, err := s.findUIDDuplicates(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find UID duplicates: %w", err)
	}
	result.UIDDuplicates = len(uidDuplicates)

	// 3. 查找内容相似的邮件
	contentDuplicates, err := s.findContentDuplicates(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find content duplicates: %w", err)
	}
	result.ContentDuplicates = len(contentDuplicates)

	if !dryRun {
		// 4. 修复MessageID重复
		fixed, err := s.repairMessageIDDuplicates(ctx, messageIDDuplicates)
		if err != nil {
			return nil, fmt.Errorf("failed to repair MessageID duplicates: %w", err)
		}
		result.MessageIDFixed = fixed

		// 5. 修复UID重复
		fixed, err = s.repairUIDDuplicates(ctx, uidDuplicates)
		if err != nil {
			return nil, fmt.Errorf("failed to repair UID duplicates: %w", err)
		}
		result.UIDFixed = fixed

		// 6. 修复内容重复
		fixed, err = s.repairContentDuplicates(ctx, contentDuplicates)
		if err != nil {
			return nil, fmt.Errorf("failed to repair content duplicates: %w", err)
		}
		result.ContentFixed = fixed
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	log.Printf("Duplicate email repair completed in %v", result.Duration)
	return result, nil
}

// RepairResult 修复结果
type RepairResult struct {
	StartTime           time.Time     `json:"start_time"`
	EndTime             time.Time     `json:"end_time"`
	Duration            time.Duration `json:"duration"`
	DryRun              bool          `json:"dry_run"`
	MessageIDDuplicates int           `json:"message_id_duplicates"`
	UIDDuplicates       int           `json:"uid_duplicates"`
	ContentDuplicates   int           `json:"content_duplicates"`
	MessageIDFixed      int           `json:"message_id_fixed"`
	UIDFixed            int           `json:"uid_fixed"`
	ContentFixed        int           `json:"content_fixed"`
	Errors              []string      `json:"errors,omitempty"`
}

// DuplicateGroup 重复邮件组
type DuplicateGroup struct {
	Key    string         `json:"key"`
	Emails []models.Email `json:"emails"`
	Count  int            `json:"count"`
}

// findMessageIDDuplicates 查找MessageID重复的邮件
func (s *DataRepairService) findMessageIDDuplicates(ctx context.Context) ([]DuplicateGroup, error) {
	var duplicates []struct {
		AccountID uint   `json:"account_id"`
		MessageID string `json:"message_id"`
		Count     int    `json:"count"`
	}

	err := s.db.WithContext(ctx).
		Model(&models.Email{}).
		Select("account_id, message_id, COUNT(*) as count").
		Where("message_id IS NOT NULL AND message_id != ''").
		Group("account_id, message_id").
		Having("COUNT(*) > 1").
		Find(&duplicates).Error

	if err != nil {
		return nil, err
	}

	var groups []DuplicateGroup
	for _, dup := range duplicates {
		var emails []models.Email
		err := s.db.WithContext(ctx).
			Where("account_id = ? AND message_id = ?", dup.AccountID, dup.MessageID).
			Order("created_at ASC").
			Find(&emails).Error

		if err != nil {
			log.Printf("Failed to get emails for MessageID %s: %v", dup.MessageID, err)
			continue
		}

		groups = append(groups, DuplicateGroup{
			Key:    fmt.Sprintf("account_%d_message_%s", dup.AccountID, dup.MessageID),
			Emails: emails,
			Count:  dup.Count,
		})
	}

	return groups, nil
}

// findUIDDuplicates 查找UID重复的邮件
func (s *DataRepairService) findUIDDuplicates(ctx context.Context) ([]DuplicateGroup, error) {
	var duplicates []struct {
		AccountID uint `json:"account_id"`
		FolderID  uint `json:"folder_id"`
		UID       uint `json:"uid"`
		Count     int  `json:"count"`
	}

	err := s.db.WithContext(ctx).
		Model(&models.Email{}).
		Select("account_id, folder_id, uid, COUNT(*) as count").
		Where("folder_id IS NOT NULL").
		Group("account_id, folder_id, uid").
		Having("COUNT(*) > 1").
		Find(&duplicates).Error

	if err != nil {
		return nil, err
	}

	var groups []DuplicateGroup
	for _, dup := range duplicates {
		var emails []models.Email
		err := s.db.WithContext(ctx).
			Where("account_id = ? AND folder_id = ? AND uid = ?", 
				dup.AccountID, dup.FolderID, dup.UID).
			Order("created_at ASC").
			Find(&emails).Error

		if err != nil {
			log.Printf("Failed to get emails for UID %d: %v", dup.UID, err)
			continue
		}

		groups = append(groups, DuplicateGroup{
			Key:    fmt.Sprintf("account_%d_folder_%d_uid_%d", dup.AccountID, dup.FolderID, dup.UID),
			Emails: emails,
			Count:  dup.Count,
		})
	}

	return groups, nil
}

// findContentDuplicates 查找内容相似的邮件
func (s *DataRepairService) findContentDuplicates(ctx context.Context) ([]DuplicateGroup, error) {
	var duplicates []struct {
		AccountID uint   `json:"account_id"`
		Subject   string `json:"subject"`
		From      string `json:"from"`
		Date      string `json:"date"`
		Count     int    `json:"count"`
	}

	err := s.db.WithContext(ctx).
		Model(&models.Email{}).
		Select("account_id, subject, \"from\", DATE(date) as date, COUNT(*) as count").
		Where("(message_id IS NULL OR message_id = '') AND subject != '' AND \"from\" != ''").
		Group("account_id, subject, \"from\", DATE(date)").
		Having("COUNT(*) > 1").
		Find(&duplicates).Error

	if err != nil {
		return nil, err
	}

	var groups []DuplicateGroup
	for _, dup := range duplicates {
		var emails []models.Email
		err := s.db.WithContext(ctx).
			Where("account_id = ? AND subject = ? AND \"from\" = ? AND DATE(date) = ?", 
				dup.AccountID, dup.Subject, dup.From, dup.Date).
			Order("created_at ASC").
			Find(&emails).Error

		if err != nil {
			log.Printf("Failed to get emails for content duplicate: %v", err)
			continue
		}

		groups = append(groups, DuplicateGroup{
			Key:    fmt.Sprintf("account_%d_content_%s_%s_%s", dup.AccountID, dup.Subject, dup.From, dup.Date),
			Emails: emails,
			Count:  dup.Count,
		})
	}

	return groups, nil
}

// repairMessageIDDuplicates 修复MessageID重复
func (s *DataRepairService) repairMessageIDDuplicates(ctx context.Context, groups []DuplicateGroup) (int, error) {
	fixed := 0
	for _, group := range groups {
		if len(group.Emails) <= 1 {
			continue
		}

		// 保留第一个（最早创建的），删除其余的
		keepEmail := group.Emails[0]
		duplicateEmails := group.Emails[1:]

		for _, email := range duplicateEmails {
			// 合并附件到保留的邮件
			if err := s.mergeAttachments(ctx, keepEmail.ID, email.ID); err != nil {
				log.Printf("Failed to merge attachments from email %d to %d: %v", email.ID, keepEmail.ID, err)
			}

			// 删除重复邮件
			if err := s.db.WithContext(ctx).Delete(&email).Error; err != nil {
				log.Printf("Failed to delete duplicate email %d: %v", email.ID, err)
			} else {
				fixed++
			}
		}
	}
	return fixed, nil
}

// repairUIDDuplicates 修复UID重复
func (s *DataRepairService) repairUIDDuplicates(ctx context.Context, groups []DuplicateGroup) (int, error) {
	fixed := 0
	for _, group := range groups {
		if len(group.Emails) <= 1 {
			continue
		}

		// 保留最新的邮件（可能有更准确的状态）
		keepEmail := group.Emails[len(group.Emails)-1]
		duplicateEmails := group.Emails[:len(group.Emails)-1]

		for _, email := range duplicateEmails {
			// 合并附件
			if err := s.mergeAttachments(ctx, keepEmail.ID, email.ID); err != nil {
				log.Printf("Failed to merge attachments from email %d to %d: %v", email.ID, keepEmail.ID, err)
			}

			// 删除重复邮件
			if err := s.db.WithContext(ctx).Delete(&email).Error; err != nil {
				log.Printf("Failed to delete duplicate email %d: %v", email.ID, err)
			} else {
				fixed++
			}
		}
	}
	return fixed, nil
}

// repairContentDuplicates 修复内容重复
func (s *DataRepairService) repairContentDuplicates(ctx context.Context, groups []DuplicateGroup) (int, error) {
	fixed := 0
	for _, group := range groups {
		if len(group.Emails) <= 1 {
			continue
		}

		// 保留第一个，删除其余的
		keepEmail := group.Emails[0]
		duplicateEmails := group.Emails[1:]

		for _, email := range duplicateEmails {
			// 合并附件
			if err := s.mergeAttachments(ctx, keepEmail.ID, email.ID); err != nil {
				log.Printf("Failed to merge attachments from email %d to %d: %v", email.ID, keepEmail.ID, err)
			}

			// 删除重复邮件
			if err := s.db.WithContext(ctx).Delete(&email).Error; err != nil {
				log.Printf("Failed to delete duplicate email %d: %v", email.ID, err)
			} else {
				fixed++
			}
		}
	}
	return fixed, nil
}

// mergeAttachments 合并附件
func (s *DataRepairService) mergeAttachments(ctx context.Context, keepEmailID, deleteEmailID uint) error {
	return s.db.WithContext(ctx).
		Model(&models.Attachment{}).
		Where("email_id = ?", deleteEmailID).
		Update("email_id", keepEmailID).Error
}
