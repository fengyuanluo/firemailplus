package services

import (
	"bytes"
	"context"
	"errors"
	"firemail/internal/cache"
	"firemail/internal/encoding/transfer"
	"firemail/internal/models"
	"firemail/internal/providers"
	"firemail/internal/sse"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// SyncService 邮件同步服务
type SyncService struct {
	db                  *gorm.DB
	providerFactory     providers.ProviderFactoryInterface
	eventPublisher      sse.EventPublisher
	deduplicatorFactory DeduplicatorFactory
	retryManager        *providers.RetryManager
	attachmentStorage   AttachmentStorage   // 添加附件存储
	cacheManager        *cache.CacheManager // 添加缓存管理器
	accountLocks        sync.Map
}

// NewSyncService 创建同步服务实例
func NewSyncService(db *gorm.DB, providerFactory providers.ProviderFactoryInterface, eventPublisher sse.EventPublisher, deduplicatorFactory DeduplicatorFactory, attachmentStorage AttachmentStorage, cacheManager *cache.CacheManager) *SyncService {
	return &SyncService{
		db:                  db,
		providerFactory:     providerFactory,
		eventPublisher:      eventPublisher,
		deduplicatorFactory: deduplicatorFactory,
		retryManager:        providers.GetGlobalRetryManager(),
		attachmentStorage:   attachmentStorage,
		cacheManager:        cacheManager,
	}
}

// SyncEmails 同步指定账户的邮件
func (s *SyncService) SyncEmails(ctx context.Context, accountID uint) error {
	// 为邮件同步创建一个更长的超时上下文（10分钟）；避免直接使用可能已被 HTTP 关闭的请求上下文导致立即取消
	baseCtx := context.Background()
	if ctx != nil && ctx.Err() == nil {
		baseCtx = ctx
	}
	syncCtx, cancel := context.WithTimeout(baseCtx, 10*time.Minute)
	defer cancel()

	lock := s.getAccountLock(accountID)
	lock.Lock()
	defer lock.Unlock()

	var account models.EmailAccount
	if err := s.db.WithContext(syncCtx).First(&account, accountID).Error; err != nil {
		return fmt.Errorf("account not found: %w", err)
	}

	// 检查账户是否激活
	if !account.IsActive {
		return fmt.Errorf("account is not active")
	}

	// 更新同步状态
	account.SyncStatus = "syncing"
	s.db.WithContext(syncCtx).Save(&account)

	// 发布同步开始事件
	if s.eventPublisher != nil {
		syncStartEvent := sse.NewSyncEvent(sse.EventSyncStarted, account.ID, account.Name, account.UserID)
		if err := s.eventPublisher.PublishToUser(syncCtx, account.UserID, syncStartEvent); err != nil {
			log.Printf("Failed to publish sync start event: %v", err)
		}
	}

	// 创建提供商实例
	provider, err := s.providerFactory.CreateProviderForAccount(&account)
	if err != nil {
		s.updateSyncError(&account, fmt.Errorf("failed to create provider: %w", err))
		return err
	}

	// 连接到服务器
	if err := provider.Connect(syncCtx, &account); err != nil {
		s.updateSyncError(&account, fmt.Errorf("failed to connect: %w", err))
		return err
	}
	defer provider.Disconnect()

	// 获取账户的文件夹
	var folders []models.Folder
	if err := s.db.WithContext(syncCtx).Where("account_id = ? AND is_selectable = ?", accountID, true).
		Find(&folders).Error; err != nil {
		s.updateSyncError(&account, fmt.Errorf("failed to get folders: %w", err))
		return err
	}

	// 如果没有文件夹，先进行文件夹同步
	if len(folders) == 0 {
		fmt.Printf("📁 [SYNC] No folders found for account %s, syncing folders first...\n", account.Email)
		if err := s.syncFoldersForAccount(syncCtx, provider, &account); err != nil {
			s.updateSyncError(&account, fmt.Errorf("failed to sync folders: %w", err))
			return err
		}

		// 重新查询文件夹
		if err := s.db.WithContext(syncCtx).Where("account_id = ? AND is_selectable = ?", accountID, true).
			Find(&folders).Error; err != nil {
			s.updateSyncError(&account, fmt.Errorf("failed to get folders after sync: %w", err))
			return err
		}
		fmt.Printf("📁 [SYNC] Folder sync completed, found %d selectable folders\n", len(folders))
	}

	// 同步每个文件夹
	var syncErrors []error
	for _, folder := range folders {
		if err := s.syncFolder(syncCtx, provider, &account, &folder); err != nil {
			log.Printf("Failed to sync folder %s: %v", folder.Name, err)
			syncErrors = append(syncErrors, err)
		}
	}

	// 统计账户的总邮件数量（避免重复计算）
	var totalSyncedEmails int64
	s.db.WithContext(syncCtx).Model(&models.Email{}).Where("account_id = ?", accountID).Count(&totalSyncedEmails)

	// 更新邮件统计（无论是否有错误都要更新）
	account.TotalEmails = int(totalSyncedEmails)
	var unreadCount int64
	s.db.WithContext(syncCtx).Model(&models.Email{}).Where("account_id = ? AND is_read = ?", account.ID, false).Count(&unreadCount)
	account.UnreadEmails = int(unreadCount)

	now := time.Now()
	account.LastSyncAt = &now

	// 更新同步状态
	if len(syncErrors) > 0 {
		account.SyncStatus = "error"
		account.ErrorMessage = fmt.Sprintf("sync completed with %d errors", len(syncErrors))
		s.db.WithContext(syncCtx).Save(&account)

		// 发布同步错误事件
		if s.eventPublisher != nil {
			syncErrorEvent := sse.NewSyncEvent(sse.EventSyncError, account.ID, account.Name, account.UserID)
			if syncErrorEvent.Data != nil {
				if syncData, ok := syncErrorEvent.Data.(*sse.SyncEventData); ok {
					syncData.ErrorMessage = fmt.Sprintf("Sync completed with %d errors", len(syncErrors))
					syncData.ProcessedEmails = int(totalSyncedEmails)
					syncData.TotalEmails = int(totalSyncedEmails)
				}
			}
			if err := s.eventPublisher.PublishToUser(ctx, account.UserID, syncErrorEvent); err != nil {
				log.Printf("Failed to publish sync error event: %v", err)
			}
		}
	} else {
		account.SyncStatus = "success"
		account.ErrorMessage = ""
		s.db.WithContext(syncCtx).Save(&account)

		// 发布同步完成事件
		if s.eventPublisher != nil {
			syncCompleteEvent := sse.NewSyncEvent(sse.EventSyncCompleted, account.ID, account.Name, account.UserID)
			if syncCompleteEvent.Data != nil {
				if syncData, ok := syncCompleteEvent.Data.(*sse.SyncEventData); ok {
					syncData.ProcessedEmails = int(totalSyncedEmails)
					syncData.TotalEmails = int(totalSyncedEmails)
				}
			}
			if err := s.eventPublisher.PublishToUser(syncCtx, account.UserID, syncCompleteEvent); err != nil {
				log.Printf("Failed to publish sync complete event: %v", err)
			}
		}
	}

	return nil
}

// syncFoldersForAccount 同步账户的文件夹
func (s *SyncService) syncFoldersForAccount(ctx context.Context, provider providers.EmailProvider, account *models.EmailAccount) error {
	fmt.Printf("📁 [FOLDER_SYNC] Starting folder sync for account: %s\n", account.Email)

	// 获取IMAP客户端
	imapClient := provider.IMAPClient()
	if imapClient == nil {
		fmt.Printf("❌ [FOLDER_SYNC] IMAP client not available\n")
		return fmt.Errorf("IMAP client not available")
	}

	// 获取文件夹列表
	fmt.Printf("📋 [FOLDER_SYNC] Listing folders from IMAP server...\n")
	folders, err := imapClient.ListFolders(ctx)
	if err != nil {
		fmt.Printf("❌ [FOLDER_SYNC] Failed to list folders: %v\n", err)
		return fmt.Errorf("failed to list folders: %w", err)
	}

	fmt.Printf("📊 [FOLDER_SYNC] Found %d folders on server\n", len(folders))

	// 保存文件夹到数据库
	for i, folderInfo := range folders {
		fmt.Printf("📁 [FOLDER_SYNC] Processing folder %d/%d: %s (selectable: %t)\n",
			i+1, len(folders), folderInfo.Name, folderInfo.IsSelectable)

		folder := &models.Folder{
			AccountID:    account.ID,
			Name:         folderInfo.Name,
			DisplayName:  folderInfo.DisplayName,
			Type:         folderInfo.Type,
			Path:         folderInfo.Path,
			Delimiter:    folderInfo.Delimiter,
			IsSelectable: folderInfo.IsSelectable,
			IsSubscribed: folderInfo.IsSubscribed,
		}

		// 检查文件夹是否已存在
		var existingFolder models.Folder
		err := s.db.Where("account_id = ? AND path = ?", account.ID, folderInfo.Path).
			First(&existingFolder).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 文件夹不存在，创建新的
				if err := s.db.Create(folder).Error; err != nil {
					fmt.Printf("❌ [FOLDER_SYNC] Failed to create folder %s: %v\n", folderInfo.Name, err)
					return fmt.Errorf("failed to create folder %s: %w", folderInfo.Name, err)
				}
				fmt.Printf("✅ [FOLDER_SYNC] Created new folder: %s\n", folderInfo.Name)
			} else {
				fmt.Printf("❌ [FOLDER_SYNC] Database error for folder %s: %v\n", folderInfo.Name, err)
				return fmt.Errorf("database error for folder %s: %w", folderInfo.Name, err)
			}
		} else {
			// 文件夹已存在，更新属性
			existingFolder.DisplayName = folderInfo.DisplayName
			existingFolder.Type = folderInfo.Type
			existingFolder.IsSelectable = folderInfo.IsSelectable
			existingFolder.IsSubscribed = folderInfo.IsSubscribed

			if err := s.db.Save(&existingFolder).Error; err != nil {
				fmt.Printf("❌ [FOLDER_SYNC] Failed to update folder %s: %v\n", folderInfo.Name, err)
				return fmt.Errorf("failed to update folder %s: %w", folderInfo.Name, err)
			}
			fmt.Printf("✅ [FOLDER_SYNC] Updated existing folder: %s\n", folderInfo.Name)
		}
	}

	fmt.Printf("✅ [FOLDER_SYNC] Folder sync completed for account: %s\n", account.Email)
	return nil
}

// SyncAccount 同步指定账户（别名方法，用于向后兼容）
func (s *SyncService) SyncAccount(ctx context.Context, accountID, userID uint) error {
	return s.SyncEmails(ctx, accountID)
}

// SyncEmailsForUser 同步用户的所有邮件账户
func (s *SyncService) SyncEmailsForUser(ctx context.Context, userID uint) error {
	var accounts []models.EmailAccount
	if err := s.db.Where("user_id = ? AND is_active = ?", userID, true).
		Find(&accounts).Error; err != nil {
		return fmt.Errorf("failed to get user accounts: %w", err)
	}

	var syncErrors []error
	for _, account := range accounts {
		if err := s.SyncEmails(ctx, account.ID); err != nil {
			log.Printf("Failed to sync account %d: %v", account.ID, err)
			syncErrors = append(syncErrors, err)
		}
	}

	if len(syncErrors) > 0 {
		return fmt.Errorf("sync completed with %d errors", len(syncErrors))
	}

	return nil
}

// SyncFolder 同步指定文件夹
func (s *SyncService) SyncFolder(ctx context.Context, accountID uint, folderName string) error {
	var account models.EmailAccount
	if err := s.db.First(&account, accountID).Error; err != nil {
		return fmt.Errorf("account not found: %w", err)
	}

	var folder models.Folder
	if err := s.db.Where("account_id = ? AND (name = ? OR path = ?)",
		accountID, folderName, folderName).First(&folder).Error; err != nil {
		return fmt.Errorf("folder not found: %w", err)
	}

	// 创建提供商实例
	provider, err := s.providerFactory.CreateProviderForAccount(&account)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	// 连接到服务器
	if err := provider.Connect(ctx, &account); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer provider.Disconnect()

	return s.syncFolder(ctx, provider, &account, &folder)
}

// syncFolder 同步单个文件夹的内部实现
func (s *SyncService) syncFolder(ctx context.Context, provider providers.EmailProvider,
	account *models.EmailAccount, folder *models.Folder) error {

	fmt.Printf("📁 [FOLDER] Starting sync for folder: %s (ID: %d, Account: %s)\n",
		folder.Name, folder.ID, account.Email)

	imapClient := provider.IMAPClient()
	if imapClient == nil {
		fmt.Printf("❌ [FOLDER] IMAP client not available for folder: %s\n", folder.Name)
		return fmt.Errorf("IMAP client not available")
	}

	// 检查文件夹是否可选择
	if !folder.IsSelectable {
		fmt.Printf("⏭️ [FOLDER] Skipping non-selectable folder: %s\n", folder.Name)
		log.Printf("Skipping non-selectable folder: %s", folder.Name)
		return nil
	}

	fmt.Printf("🔄 [FOLDER] Performing incremental sync for folder: %s\n", folder.Name)

	// 实现真正的增量同步
	newEmails, err := s.performIncrementalSync(ctx, provider, imapClient, folder, account)
	if err != nil {
		fmt.Printf("❌ [FOLDER] Failed to perform incremental sync for folder %s: %v\n", folder.Name, err)
		log.Printf("Failed to perform incremental sync for folder %s: %v", folder.Name, err)
		return fmt.Errorf("failed to perform incremental sync: %w", err)
	}

	fmt.Printf("📊 [FOLDER] Incremental sync completed for folder %s: %d new emails\n",
		folder.Name, len(newEmails))

	// 保存新邮件到数据库
	var newEmailCount int
	totalEmails := len(newEmails)
	log.Printf("Retrieved %d new emails for folder %s", totalEmails, folder.Name)

	for i, emailMsg := range newEmails {
		if err := s.saveEmailToDatabase(ctx, emailMsg, account.ID, folder.ID, account.UserID); err != nil {
			log.Printf("Failed to save email %s: %v", emailMsg.MessageID, err)
		} else {
			newEmailCount++
		}

		// 发布同步进度事件
		if s.eventPublisher != nil && totalEmails > 0 {
			progress := float64(i+1) / float64(totalEmails)
			syncProgressEvent := sse.NewSyncEvent(sse.EventSyncProgress, account.ID, account.Name, account.UserID)
			if syncProgressEvent.Data != nil {
				if syncData, ok := syncProgressEvent.Data.(*sse.SyncEventData); ok {
					syncData.Progress = progress
					syncData.ProcessedEmails = i + 1
					syncData.TotalEmails = totalEmails
					syncData.FolderName = folder.Name
				}
			}
			if err := s.eventPublisher.PublishToUser(ctx, account.UserID, syncProgressEvent); err != nil {
				log.Printf("Failed to publish sync progress event: %v", err)
			}
		}
	}

	log.Printf("Synced %d new emails for folder %s", newEmailCount, folder.Name)

	// 发布文件夹同步进度事件
	if s.eventPublisher != nil && newEmailCount > 0 {
		folderSyncEvent := sse.NewSyncEvent(sse.EventSyncProgress, account.ID, account.Name, account.UserID)
		if folderSyncEvent.Data != nil {
			if syncData, ok := folderSyncEvent.Data.(*sse.SyncEventData); ok {
				syncData.FolderName = folder.Name
				syncData.ProcessedEmails = newEmailCount
				syncData.TotalEmails = totalEmails
				syncData.Progress = 1.0 // 文件夹同步完成
			}
		}
		if err := s.eventPublisher.PublishToUser(ctx, account.UserID, folderSyncEvent); err != nil {
			log.Printf("Failed to publish folder sync progress event: %v", err)
		}
	}

	return nil
}

// saveEmailToDatabase 保存邮件到数据库（使用去重功能）
func (s *SyncService) saveEmailToDatabase(ctx context.Context, emailMsg *providers.EmailMessage, accountID, folderID, userID uint) error {
	// 获取账户信息以确定提供商类型
	var account models.EmailAccount
	if err := s.db.First(&account, accountID).Error; err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}

	// 创建对应的去重器
	deduplicator := s.deduplicatorFactory.CreateDeduplicator(account.Provider)

	// 检查邮件是否重复
	duplicateResult, err := deduplicator.CheckDuplicate(ctx, emailMsg, accountID, folderID)
	if err != nil {
		return fmt.Errorf("failed to check duplicate: %w", err)
	}

	// 处理重复邮件
	if duplicateResult.IsDuplicate {
		switch duplicateResult.Action {
		case "skip":
			log.Printf("Skipping duplicate email: %s (reason: %s)", emailMsg.MessageID, duplicateResult.Reason)
			return nil
		case "update", "create_label_reference":
			if err := deduplicator.HandleDuplicate(ctx, duplicateResult.ExistingEmail, emailMsg, folderID); err != nil {
				return fmt.Errorf("failed to handle duplicate: %w", err)
			}
			log.Printf("Updated duplicate email: %s (action: %s)", emailMsg.MessageID, duplicateResult.Action)
			return nil
		default:
			log.Printf("Unknown duplicate action: %s, creating new email", duplicateResult.Action)
		}
	}

	// 使用事务创建新邮件，确保数据一致性
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 创建新邮件
		email := &models.Email{
			AccountID:     accountID,
			FolderID:      &folderID,
			MessageID:     emailMsg.MessageID,
			UID:           emailMsg.UID,
			Subject:       emailMsg.Subject,
			Date:          emailMsg.Date,
			TextBody:      emailMsg.TextBody,
			HTMLBody:      emailMsg.HTMLBody,
			Size:          emailMsg.Size,
			IsRead:        s.isEmailRead(emailMsg.Flags),
			IsStarred:     s.isEmailStarred(emailMsg.Flags),
			IsDraft:       s.isEmailDraft(emailMsg.Flags),
			HasAttachment: len(emailMsg.Attachments) > 0,
		}

		// 设置发件人
		if emailMsg.From != nil {
			email.From = emailMsg.From.Address
			if emailMsg.From.Name != "" {
				email.From = fmt.Sprintf("%s <%s>", emailMsg.From.Name, emailMsg.From.Address)
			}
		}

		// 设置收件人
		if err := email.SetToAddresses(convertEmailAddresses(emailMsg.To)); err != nil {
			log.Printf("Failed to set To addresses: %v", err)
		}

		// 设置抄送
		if err := email.SetCCAddresses(convertEmailAddresses(emailMsg.CC)); err != nil {
			log.Printf("Failed to set CC addresses: %v", err)
		}

		// 设置密送
		if err := email.SetBCCAddresses(convertEmailAddresses(emailMsg.BCC)); err != nil {
			log.Printf("Failed to set BCC addresses: %v", err)
		}

		// 设置回复地址
		if emailMsg.ReplyTo != nil {
			email.ReplyTo = emailMsg.ReplyTo.Address
			if emailMsg.ReplyTo.Name != "" {
				email.ReplyTo = fmt.Sprintf("%s <%s>", emailMsg.ReplyTo.Name, emailMsg.ReplyTo.Address)
			}
		}

		// 保存邮件（在事务中）
		if err := tx.Create(email).Error; err != nil {
			// 检查是否是唯一约束冲突
			if isUniqueConstraintError(err) {
				log.Printf("Unique constraint violation for email %s, attempting to handle gracefully", emailMsg.MessageID)
				// 重新检查重复并处理
				deduplicator := s.deduplicatorFactory.CreateDeduplicator(account.Provider)
				duplicateResult, checkErr := deduplicator.CheckDuplicate(ctx, emailMsg, accountID, folderID)
				if checkErr != nil {
					return fmt.Errorf("failed to recheck duplicate after constraint violation: %w", checkErr)
				}
				if duplicateResult.IsDuplicate && duplicateResult.ExistingEmail != nil {
					return deduplicator.HandleDuplicate(ctx, duplicateResult.ExistingEmail, emailMsg, folderID)
				}
			}
			return fmt.Errorf("failed to create email: %w", err)
		}

		// 保存附件（在事务中）
		for _, attachmentInfo := range emailMsg.Attachments {
			attachment := &models.Attachment{
				EmailID:     &email.ID, // 使用指针类型
				Filename:    attachmentInfo.Filename,
				ContentType: attachmentInfo.ContentType,
				Size:        attachmentInfo.Size,
				ContentID:   attachmentInfo.ContentID,
				Disposition: attachmentInfo.Disposition,
				PartID:      attachmentInfo.PartID,
				Encoding:    attachmentInfo.Encoding,
			}

			if err := tx.Create(attachment).Error; err != nil {
				log.Printf("Failed to save attachment %s: %v", attachmentInfo.Filename, err)
				// 附件保存失败不应该回滚整个事务，只记录错误
				continue
			}

			// 如果有附件内容，立即保存到本地存储
			if len(attachmentInfo.Content) > 0 && s.attachmentStorage != nil {
				if err := s.saveAttachmentContent(ctx, attachment, attachmentInfo.Content); err != nil {
					log.Printf("Failed to save attachment content for %s: %v", attachmentInfo.Filename, err)
					// 内容保存失败，更新数据库记录
					tx.Model(attachment).Update("is_downloaded", false)
				} else {
					// 内容保存成功，标记为已下载
					tx.Model(attachment).Updates(map[string]interface{}{
						"is_downloaded": true,
						"file_path":     s.attachmentStorage.GetStoragePath(attachment),
					})
					log.Printf("Successfully saved attachment content: %s (%d bytes)", attachmentInfo.Filename, len(attachmentInfo.Content))
				}
			}
		}

		// 事务成功后发布新邮件事件
		if s.eventPublisher != nil {
			newEmailEvent := sse.NewNewEmailEvent(email, userID)
			if err := s.eventPublisher.PublishToUser(ctx, userID, newEmailEvent); err != nil {
				log.Printf("Failed to publish new email event: %v", err)
				// 事件发布失败不应该回滚事务
			}
		}

		// 清除邮件列表缓存，确保前端能看到新邮件
		if s.cacheManager != nil {
			s.invalidateEmailListCache(userID)
		}

		return nil
	})
}

// updateExistingEmail 更新现有邮件
func (s *SyncService) updateExistingEmail(email *models.Email, emailMsg *providers.EmailMessage, folderID uint) error {
	// 更新可能变化的字段
	email.FolderID = &folderID
	email.IsRead = s.isEmailRead(emailMsg.Flags)
	email.IsStarred = s.isEmailStarred(emailMsg.Flags)
	email.IsDraft = s.isEmailDraft(emailMsg.Flags)

	return s.db.Save(email).Error
}

// invalidateEmailListCache 使邮件列表缓存失效
func (s *SyncService) invalidateEmailListCache(userID uint) {
	if s.cacheManager == nil {
		return
	}

	// 获取所有缓存键
	keys := s.cacheManager.EmailListCache().Keys()

	// 删除与该用户相关的缓存
	// 由于我们使用MD5哈希，这里简单地清除所有缓存
	// 在实际应用中可以通过在缓存键中包含用户ID前缀来优化
	for _, key := range keys {
		s.cacheManager.EmailListCache().Delete(key)
	}

	log.Printf("Invalidated email list cache for user %d", userID)
}

// 辅助函数

// isEmailRead 检查邮件是否已读
func (s *SyncService) isEmailRead(flags []string) bool {
	for _, flag := range flags {
		if flag == "\\Seen" {
			return true
		}
	}
	return false
}

// isEmailStarred 检查邮件是否加星
func (s *SyncService) isEmailStarred(flags []string) bool {
	for _, flag := range flags {
		if flag == "\\Flagged" {
			return true
		}
	}
	return false
}

// isEmailDraft 检查邮件是否为草稿
func (s *SyncService) isEmailDraft(flags []string) bool {
	for _, flag := range flags {
		if flag == "\\Draft" {
			return true
		}
	}
	return false
}

// convertEmailAddresses 转换邮件地址格式
func convertEmailAddresses(addrs []*models.EmailAddress) []models.EmailAddress {
	var result []models.EmailAddress
	for _, addr := range addrs {
		result = append(result, models.EmailAddress{
			Name:    addr.Name,
			Address: addr.Address,
		})
	}
	return result
}

// updateSyncError 更新同步错误状态
func (s *SyncService) updateSyncError(account *models.EmailAccount, err error) {
	account.SyncStatus = "error"
	account.ErrorMessage = err.Error()
	s.db.Save(account)
}

// performIncrementalSync 执行真正的增量同步
func (s *SyncService) performIncrementalSync(ctx context.Context, provider providers.EmailProvider, imapClient providers.IMAPClient, folder *models.Folder, account *models.EmailAccount) ([]*providers.EmailMessage, error) {
	fmt.Printf("🔍 [INCREMENTAL] Starting incremental sync for folder: %s\n", folder.Name)

	// 获取当前文件夹状态，包含文件夹存在性检查
	fmt.Printf("📊 [INCREMENTAL] Getting folder status for: %s\n", folder.Path)

	var status *providers.FolderStatus
	err := s.executeWithConnectionRetry(ctx, provider, account, func() error {
		var err error
		status, err = imapClient.GetFolderStatus(ctx, folder.Path)
		return err
	})

	if err != nil {
		fmt.Printf("❌ [INCREMENTAL] Failed to get folder status for %s: %v\n", folder.Name, err)

		// 检查是否是文件夹不存在的错误
		if s.isFolderNotExistError(err) {
			fmt.Printf("⚠️ [INCREMENTAL] Folder %s does not exist on server, attempting recovery...\n", folder.Name)
			return s.handleMissingFolder(ctx, imapClient, folder, account)
		}

		return nil, fmt.Errorf("failed to get folder status: %w", err)
	}

	fmt.Printf("📊 [INCREMENTAL] Folder %s status: UIDValidity=%d, UIDNext=%d, Total=%d, Unread=%d\n",
		folder.Name, status.UIDValidity, status.UIDNext, status.TotalEmails, status.UnreadEmails)

	// 检查文件夹是否有有效的UID信息
	if status.UIDValidity == 0 {
		fmt.Printf("⚠️ [INCREMENTAL] Skipping folder with invalid UID validity: %s\n", folder.Name)
		log.Printf("Skipping folder with invalid UID validity: %s", folder.Name)
		return []*providers.EmailMessage{}, nil
	}

	log.Printf("Folder %s status: UIDValidity=%d, UIDNext=%d, Total=%d, Unread=%d",
		folder.Name, status.UIDValidity, status.UIDNext, status.TotalEmails, status.UnreadEmails)

	// 检查UIDVALIDITY是否发生变化
	needFullSync := false
	if folder.UIDValidity != 0 && folder.UIDValidity != status.UIDValidity {
		log.Printf("UIDVALIDITY changed for folder %s (old: %d, new: %d), performing full sync",
			folder.Name, folder.UIDValidity, status.UIDValidity)
		needFullSync = true
	}

	// 特殊处理：如果UIDNext=0但文件夹有邮件，强制全量同步（163邮箱等特殊情况）
	if status.UIDNext == 0 && status.TotalEmails > 0 {
		log.Printf("UIDNext=0 but folder %s has %d emails, forcing full sync", folder.Name, status.TotalEmails)
		needFullSync = true
	}

	// 更新文件夹状态
	folder.TotalEmails = status.TotalEmails
	folder.UnreadEmails = status.UnreadEmails
	folder.UIDValidity = status.UIDValidity
	folder.UIDNext = status.UIDNext
	s.db.Save(folder)

	var newEmails []*providers.EmailMessage

	if needFullSync {
		// 执行全量同步
		newEmails, err = s.performFullSync(ctx, provider, imapClient, folder, account)
	} else {
		// 执行增量同步
		newEmails, err = s.performDeltaSync(ctx, provider, imapClient, folder, account, status)
	}

	if err != nil {
		return nil, err
	}

	log.Printf("Incremental sync completed for folder %s: %d new emails", folder.Name, len(newEmails))
	return newEmails, nil
}

// performFullSync 执行全量同步（当UIDVALIDITY变化时）
func (s *SyncService) performFullSync(ctx context.Context, provider providers.EmailProvider, imapClient providers.IMAPClient, folder *models.Folder, account *models.EmailAccount) ([]*providers.EmailMessage, error) {
	log.Printf("Performing full sync for folder %s", folder.Name)

	// 删除该文件夹的所有现有邮件（因为UIDVALIDITY变化，所有UID都无效了）
	// 使用硬删除来避免UNIQUE约束冲突，同时先清理附件防止孤儿数据
	if err := s.db.WithContext(ctx).
		Where("email_id IN (?)", s.db.Model(&models.Email{}).Select("id").Where("account_id = ? AND folder_id = ?", account.ID, folder.ID)).
		Delete(&models.Attachment{}).Error; err != nil {
		log.Printf("Warning: failed to delete attachments for folder %s: %v", folder.Name, err)
	}
	if err := s.db.WithContext(ctx).Unscoped().Where("account_id = ? AND folder_id = ?", account.ID, folder.ID).Delete(&models.Email{}).Error; err != nil {
		log.Printf("Warning: failed to delete existing emails for folder %s: %v", folder.Name, err)
	}

	// 特殊处理：如果UIDNext=0，使用序列号范围而不是UID范围
	if folder.UIDNext == 0 && folder.TotalEmails > 0 {
		log.Printf("UIDNext=0, using sequence number range for folder %s (1:%d)", folder.Name, folder.TotalEmails)
		return s.getEmailsBySequenceRange(ctx, imapClient, folder, 1, uint32(folder.TotalEmails))
	}

	// 获取所有邮件（从UID 1开始），使用UIDNext限定上界，避免无限抓取
	var endUID uint32
	if folder.UIDNext > 0 {
		endUID = folder.UIDNext - 1
	}
	if endUID == 0 {
		return []*providers.EmailMessage{}, nil
	}

	return s.getEmailsInBatches(ctx, provider, imapClient, folder, account, 1, endUID)
}

// performDeltaSync 执行增量同步
func (s *SyncService) performDeltaSync(ctx context.Context, provider providers.EmailProvider, imapClient providers.IMAPClient, folder *models.Folder, account *models.EmailAccount, status *providers.FolderStatus) ([]*providers.EmailMessage, error) {
	// 获取最后同步的UID
	var lastUID uint32
	var lastEmail models.Email
	err := s.db.Where("account_id = ? AND folder_id = ?", account.ID, folder.ID).
		Order("uid DESC").First(&lastEmail).Error

	if err == nil {
		lastUID = lastEmail.UID
		log.Printf("Found last synced email with UID %d for folder %s", lastUID, folder.Name)
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to get last UID: %w", err)
	} else {
		log.Printf("No previous emails found for folder %s, starting from UID 1", folder.Name)
		lastUID = 0
	}

	// 特殊处理：如果UIDNext和Total不匹配，可能存在UID不连续的情况
	var gapEmails []*providers.EmailMessage
	if status.UIDNext-1 != uint32(status.TotalEmails) && status.TotalEmails > 0 {
		fmt.Printf("⚠️ [INCREMENTAL] UID/Total mismatch - UIDNext: %d, Total: %d, checking for UID gaps\n",
			status.UIDNext, status.TotalEmails)
		log.Printf("UID/Total mismatch for folder %s - UIDNext: %d, Total: %d",
			folder.Name, status.UIDNext, status.TotalEmails)

		// 检查是否有UID缺口需要填补
		if lastUID > 0 && status.UIDNext > lastUID+1 {
			// 尝试获取从lastUID+1到UIDNext-1之间可能遗漏的邮件
			log.Printf("Checking for missing UIDs in range %d to %d for folder %s",
				lastUID+1, status.UIDNext-1, folder.Name)

			// 使用更智能的UID范围检测
			missingEmails, err := s.getEmailsWithGapDetection(ctx, imapClient, folder, lastUID+1, status.UIDNext-1)
			if err != nil {
				log.Printf("Failed to get emails with gap detection: %v", err)
				// 降级到原有逻辑
			} else if len(missingEmails) > 0 {
				log.Printf("Found %d missing emails in UID gaps for folder %s", len(missingEmails), folder.Name)
				gapEmails = append(gapEmails, missingEmails...)
			}
		}
	}

	// 如果没有新邮件，直接返回
	if status.UIDNext <= lastUID+1 {
		log.Printf("No new emails in folder %s (UIDNext: %d, lastUID: %d)", folder.Name, status.UIDNext, lastUID)
		return gapEmails, nil
	}

	log.Printf("Fetching new emails for folder %s from UID %d to %d", folder.Name, lastUID+1, status.UIDNext-1)

	// 获取新邮件（从lastUID+1到UIDNext-1）
	latestEmails, err := s.getEmailsInBatches(ctx, provider, imapClient, folder, account, lastUID+1, status.UIDNext-1)
	if err != nil {
		return nil, err
	}

	return append(gapEmails, latestEmails...), nil
}

// getEmailsInBatches 分批获取邮件
func (s *SyncService) getEmailsInBatches(ctx context.Context, provider providers.EmailProvider, imapClient providers.IMAPClient, folder *models.Folder, account *models.EmailAccount, startUID, endUID uint32) ([]*providers.EmailMessage, error) {
	const maxBatchSize = 50
	var allEmails []*providers.EmailMessage

	// 如果endUID为0，表示获取到最新
	if endUID == 0 {
		var emails []*providers.EmailMessage
		err := s.executeWithConnectionRetry(ctx, provider, account, func() error {
			var err error
			emails, err = imapClient.GetEmailsInUIDRange(ctx, folder.Path, startUID, 0)
			return err
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get emails from UID %d: %w", startUID, err)
		}

		return emails, nil
	}

	// 分批处理指定范围
	currentUID := startUID
	for currentUID <= endUID {
		batchEndUID := currentUID + maxBatchSize - 1
		if batchEndUID > endUID {
			batchEndUID = endUID
		}

		log.Printf("Fetching email batch: UID %d to %d", currentUID, batchEndUID)

		var batchEmails []*providers.EmailMessage
		err := s.executeWithConnectionRetry(ctx, provider, account, func() error {
			var err error
			batchEmails, err = imapClient.GetEmailsInUIDRange(ctx, folder.Path, currentUID, batchEndUID)
			return err
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get email batch %d-%d: %w", currentUID, batchEndUID, err)
		}

		allEmails = append(allEmails, batchEmails...)

		currentUID = batchEndUID + 1
	}

	return allEmails, nil
}

// 获取账户级锁，确保单账户同步串行化
func (s *SyncService) getAccountLock(accountID uint) *sync.Mutex {
	lock, _ := s.accountLocks.LoadOrStore(accountID, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

// isConnectionError 检查是否是连接错误
func (s *SyncService) isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	connectionErrors := []string{
		"connection closed",
		"connection reset",
		"connection refused",
		"connection timeout",
		"broken pipe",
		"network is unreachable",
		"no route to host",
		"timeout",
		"eof",
		"i/o timeout",
	}

	for _, connErr := range connectionErrors {
		if strings.Contains(errStr, connErr) {
			return true
		}
	}

	return false
}

// ensureConnection 确保IMAP连接有效，如果断开则重连
func (s *SyncService) ensureConnection(ctx context.Context, provider providers.EmailProvider, account *models.EmailAccount) error {
	// 检查provider是否连接
	if !provider.IsIMAPConnected() {
		log.Printf("IMAP connection lost for account %s, attempting to reconnect", account.Email)
		return provider.Connect(ctx, account)
	}

	// 检查IMAP客户端连接状态
	imapClient := provider.IMAPClient()
	if imapClient == nil {
		log.Printf("IMAP client not available for account %s, attempting to reconnect", account.Email)
		return provider.Connect(ctx, account)
	}

	// 如果IMAP客户端支持连接状态检查，使用它
	if connChecker, ok := imapClient.(interface{ IsConnectionAlive() bool }); ok {
		if !connChecker.IsConnectionAlive() {
			log.Printf("IMAP connection not alive for account %s, attempting to reconnect", account.Email)
			provider.Disconnect()
			return provider.Connect(ctx, account)
		}
	}

	// 刷新连接超时
	if timeoutRefresher, ok := imapClient.(interface{ RefreshConnectionTimeout() error }); ok {
		if err := timeoutRefresher.RefreshConnectionTimeout(); err != nil {
			log.Printf("Failed to refresh connection timeout for account %s: %v", account.Email, err)
		}
	}

	return nil
}

// executeWithConnectionRetry 执行IMAP操作，如果连接断开则重连并重试
func (s *SyncService) executeWithConnectionRetry(ctx context.Context, provider providers.EmailProvider, account *models.EmailAccount, operation func() error) error {
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		// 确保连接有效
		if err := s.ensureConnection(ctx, provider, account); err != nil {
			log.Printf("Failed to ensure connection for account %s (attempt %d): %v", account.Email, attempt+1, err)
			if attempt == maxRetries-1 {
				return fmt.Errorf("failed to establish connection after %d attempts: %w", maxRetries, err)
			}
			time.Sleep(time.Duration(attempt+1) * 2 * time.Second)
			continue
		}

		// 执行操作
		err := operation()
		if err == nil {
			return nil
		}

		// 检查是否是连接错误
		if s.isConnectionError(err) {
			log.Printf("Connection error detected for account %s (attempt %d): %v", account.Email, attempt+1, err)

			// 断开连接，下次循环会重连
			provider.Disconnect()

			if attempt == maxRetries-1 {
				return fmt.Errorf("operation failed after %d attempts due to connection issues: %w", maxRetries, err)
			}

			// 等待后重试
			time.Sleep(time.Duration(attempt+1) * 2 * time.Second)
			continue
		}

		// 非连接错误，直接返回
		return err
	}

	return fmt.Errorf("operation failed after %d attempts", maxRetries)
}

// getEmailsWithGapDetection 使用UID缺口检测获取邮件
func (s *SyncService) getEmailsWithGapDetection(ctx context.Context, imapClient providers.IMAPClient, folder *models.Folder, startUID, endUID uint32) ([]*providers.EmailMessage, error) {
	log.Printf("Performing UID gap detection for folder %s, range %d-%d", folder.Name, startUID, endUID)

	// 首先尝试直接搜索这个范围内的所有邮件
	emails, err := imapClient.GetEmailsInUIDRange(ctx, folder.Path, startUID, endUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get emails in UID range: %w", err)
	}

	// 如果找到邮件，检查UID连续性
	if len(emails) > 0 {
		log.Printf("Found %d emails in UID range %d-%d", len(emails), startUID, endUID)

		// 创建UID映射以检测缺口
		foundUIDs := make(map[uint32]bool)
		for _, email := range emails {
			foundUIDs[email.UID] = true
		}

		// 检查是否有缺失的UID
		var missingUIDs []uint32
		for uid := startUID; uid <= endUID; uid++ {
			if !foundUIDs[uid] {
				missingUIDs = append(missingUIDs, uid)
			}
		}

		if len(missingUIDs) > 0 {
			log.Printf("Detected %d missing UIDs in range %d-%d: %v",
				len(missingUIDs), startUID, endUID, missingUIDs)
		}
	}

	return emails, nil
}

// isFolderNotExistError 检查是否是文件夹不存在的错误
func (s *SyncService) isFolderNotExistError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "folder not exist") ||
		strings.Contains(errStr, "mailbox does not exist") ||
		strings.Contains(errStr, "no such mailbox") ||
		strings.Contains(errStr, "mailbox not found")
}

// handleMissingFolder 处理缺失的文件夹
func (s *SyncService) handleMissingFolder(ctx context.Context, imapClient providers.IMAPClient, folder *models.Folder, account *models.EmailAccount) ([]*providers.EmailMessage, error) {
	log.Printf("Handling missing folder %s (type: %s) for account %s", folder.Name, folder.Type, account.Email)

	// 根据文件夹类型采取不同的处理策略
	switch folder.Type {
	case "archive":
		// 对于归档文件夹，尝试重新创建
		log.Printf("Attempting to recreate missing archive folder: %s", folder.Name)
		if err := imapClient.CreateFolder(ctx, folder.Path); err != nil {
			log.Printf("Failed to recreate archive folder %s: %v", folder.Name, err)
			// 标记文件夹为无效，但不中断同步
			s.markFolderAsInvalid(folder)
			return []*providers.EmailMessage{}, nil
		}
		log.Printf("Successfully recreated archive folder: %s", folder.Name)

		// 重新获取文件夹状态
		status, err := imapClient.GetFolderStatus(ctx, folder.Path)
		if err != nil {
			log.Printf("Failed to get status of recreated folder %s: %v", folder.Name, err)
			return []*providers.EmailMessage{}, nil
		}

		// 更新文件夹状态并继续同步
		folder.TotalEmails = status.TotalEmails
		folder.UnreadEmails = status.UnreadEmails
		folder.UIDValidity = status.UIDValidity
		folder.UIDNext = status.UIDNext
		s.db.Save(folder)

		return []*providers.EmailMessage{}, nil

	case "custom":
		// 对于自定义文件夹，标记为无效但保留记录
		log.Printf("Marking custom folder %s as invalid", folder.Name)
		s.markFolderAsInvalid(folder)
		return []*providers.EmailMessage{}, nil

	default:
		// 对于系统文件夹，记录警告但不删除
		log.Printf("System folder %s missing on server, skipping sync", folder.Name)
		return []*providers.EmailMessage{}, nil
	}
}

// markFolderAsInvalid 标记文件夹为无效
func (s *SyncService) markFolderAsInvalid(folder *models.Folder) {
	// 可以添加一个字段来标记文件夹状态，这里暂时只记录日志
	log.Printf("Folder %s marked as invalid due to server absence", folder.Name)
	// TODO: 可以考虑添加 is_valid 字段到 Folder 模型
}

// getEmailsBySequenceRange 通过序列号范围获取邮件（用于UIDNext=0的情况）
func (s *SyncService) getEmailsBySequenceRange(ctx context.Context, imapClient providers.IMAPClient, folder *models.Folder, startSeq, endSeq uint32) ([]*providers.EmailMessage, error) {
	log.Printf("Fetching emails for folder %s using sequence range %d-%d (UIDNext=0 fallback)", folder.Name, startSeq, endSeq)

	// 对于UIDNext=0的情况，我们使用GetEmailsInUIDRange但传入序列号
	// 这是一个权宜之计，因为163邮箱的UIDNext=0是异常情况
	// 我们尝试获取前50封邮件
	const maxEmails = 50
	actualEndSeq := endSeq
	if actualEndSeq > maxEmails {
		actualEndSeq = maxEmails
		log.Printf("Limiting to first %d emails due to UIDNext=0", maxEmails)
	}

	// 使用FetchCriteria获取所有邮件，然后取前N封
	criteria := &providers.FetchCriteria{
		FolderName:  folder.Path,
		IncludeBody: true,
		Limit:       int(actualEndSeq),
	}

	emails, err := imapClient.FetchEmails(ctx, criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch emails by sequence: %w", err)
	}

	log.Printf("Fetched %d emails for folder %s using sequence fallback", len(emails), folder.Name)
	return emails, nil
}

// saveAttachmentContent 保存附件内容到本地存储
func (s *SyncService) saveAttachmentContent(ctx context.Context, attachment *models.Attachment, rawContent []byte) error {
	if s.attachmentStorage == nil {
		return fmt.Errorf("attachment storage not configured")
	}

	// 解码附件内容
	decodedContent, err := transfer.DecodeWithFallback(rawContent, attachment.Encoding)
	if err != nil {
		log.Printf("Warning: Failed to decode attachment %s with encoding %s: %v, using raw content",
			attachment.Filename, attachment.Encoding, err)
		decodedContent = rawContent
	}

	// 更新附件大小为解码后的实际大小
	actualSize := int64(len(decodedContent))
	if actualSize != attachment.Size {
		log.Printf("Attachment %s size changed after decoding: %d -> %d (encoding: %s)",
			attachment.Filename, attachment.Size, actualSize, attachment.Encoding)
		attachment.Size = actualSize
	}

	// 创建内容读取器
	contentReader := bytes.NewReader(decodedContent)

	// 保存到存储
	if err := s.attachmentStorage.Store(ctx, attachment, contentReader); err != nil {
		return fmt.Errorf("failed to store attachment content: %w", err)
	}

	return nil
}
