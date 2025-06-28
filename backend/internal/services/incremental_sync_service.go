package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"firemail/internal/models"
	"firemail/internal/providers"
	"firemail/internal/sse"
)

// IncrementalSyncService 增量同步服务
type IncrementalSyncService struct {
	db                   *gorm.DB
	providerFactory      providers.ProviderFactory
	deduplicatorFactory  DeduplicatorFactory
	eventPublisher       sse.EventPublisher
	batchSize           int
	maxConcurrentFolders int
}

// NewIncrementalSyncService 创建增量同步服务
func NewIncrementalSyncService(
	db *gorm.DB,
	providerFactory providers.ProviderFactory,
	deduplicatorFactory DeduplicatorFactory,
	eventPublisher sse.EventPublisher,
) *IncrementalSyncService {
	return &IncrementalSyncService{
		db:                   db,
		providerFactory:      providerFactory,
		deduplicatorFactory:  deduplicatorFactory,
		eventPublisher:       eventPublisher,
		batchSize:           100, // 批量处理邮件数量
		maxConcurrentFolders: 3,  // 最大并发文件夹数
	}
}

// SyncStrategy 同步策略
type SyncStrategy struct {
	AccountID        uint
	FolderIDs        []uint // 指定要同步的文件夹，空则同步所有
	OnlyNewEmails    bool   // 只同步新邮件
	MaxEmailsPerSync int    // 每次同步的最大邮件数
	SinceTime        *time.Time // 只同步此时间之后的邮件
}

// SyncResult 同步结果
type SyncResult struct {
	AccountID       uint
	TotalFolders    int
	ProcessedEmails int
	NewEmails       int
	UpdatedEmails   int
	Errors          []error
	Duration        time.Duration
}

// SyncEmailsIncremental 增量同步邮件
func (s *IncrementalSyncService) SyncEmailsIncremental(ctx context.Context, strategy *SyncStrategy) (*SyncResult, error) {
	startTime := time.Now()
	result := &SyncResult{
		AccountID: strategy.AccountID,
		Errors:    make([]error, 0),
	}

	// 获取账户信息
	var account models.EmailAccount
	if err := s.db.First(&account, strategy.AccountID).Error; err != nil {
		return nil, fmt.Errorf("account not found: %w", err)
	}

	if !account.IsActive {
		return nil, fmt.Errorf("account is not active")
	}

	// 更新同步状态
	account.SyncStatus = "syncing"
	s.db.Save(&account)

	// 发布同步开始事件
	if s.eventPublisher != nil {
		syncStartEvent := sse.NewSyncEvent(sse.EventSyncStarted, account.ID, account.Name, account.UserID)
		if err := s.eventPublisher.PublishToUser(ctx, account.UserID, syncStartEvent); err != nil {
			log.Printf("Failed to publish sync start event: %v", err)
		}
	}

	defer func() {
		// 更新同步状态
		if len(result.Errors) > 0 {
			account.SyncStatus = "error"
		} else {
			account.SyncStatus = "completed"
		}
		now := time.Now()
		account.LastSyncAt = &now
		s.db.Save(&account)

		// 发布同步完成事件
		if s.eventPublisher != nil {
			syncCompleteEvent := sse.NewSyncEvent(sse.EventSyncCompleted, account.ID, account.Name, account.UserID)
			if err := s.eventPublisher.PublishToUser(ctx, account.UserID, syncCompleteEvent); err != nil {
				log.Printf("Failed to publish sync complete event: %v", err)
			}
		}

		result.Duration = time.Since(startTime)
		log.Printf("Incremental sync completed for account %d: %+v", strategy.AccountID, result)
	}()

	// 获取要同步的文件夹
	folders, err := s.getFoldersToSync(ctx, strategy)
	if err != nil {
		result.Errors = append(result.Errors, err)
		return result, err
	}

	result.TotalFolders = len(folders)

	// 创建提供商实例
	provider, err := s.providerFactory.CreateProvider(account.Provider)
	if err != nil {
		result.Errors = append(result.Errors, err)
		return result, err
	}

	// 连接到邮件服务器
	if err := provider.Connect(ctx, &account); err != nil {
		result.Errors = append(result.Errors, err)
		return result, err
	}
	defer provider.Disconnect()

	// 并发同步文件夹
	folderChan := make(chan *models.Folder, len(folders))
	resultChan := make(chan *folderSyncResult, len(folders))

	// 启动工作协程
	for i := 0; i < s.maxConcurrentFolders && i < len(folders); i++ {
		go s.syncFolderWorker(ctx, provider, &account, strategy, folderChan, resultChan)
	}

	// 发送文件夹到工作队列
	for _, folder := range folders {
		folderChan <- folder
	}
	close(folderChan)

	// 收集结果
	for i := 0; i < len(folders); i++ {
		folderResult := <-resultChan
		result.ProcessedEmails += folderResult.ProcessedEmails
		result.NewEmails += folderResult.NewEmails
		result.UpdatedEmails += folderResult.UpdatedEmails
		if folderResult.Error != nil {
			result.Errors = append(result.Errors, folderResult.Error)
		}
	}

	return result, nil
}

// folderSyncResult 文件夹同步结果
type folderSyncResult struct {
	FolderID        uint
	ProcessedEmails int
	NewEmails       int
	UpdatedEmails   int
	Error           error
}

// syncFolderWorker 文件夹同步工作协程
func (s *IncrementalSyncService) syncFolderWorker(
	ctx context.Context,
	provider providers.EmailProvider,
	account *models.EmailAccount,
	strategy *SyncStrategy,
	folderChan <-chan *models.Folder,
	resultChan chan<- *folderSyncResult,
) {
	for folder := range folderChan {
		result := &folderSyncResult{FolderID: folder.ID}
		
		// 同步单个文件夹
		err := s.syncSingleFolder(ctx, provider, account, folder, strategy, result)
		if err != nil {
			result.Error = err
			log.Printf("Failed to sync folder %s: %v", folder.Name, err)
		}
		
		resultChan <- result
	}
}

// getFoldersToSync 获取要同步的文件夹
func (s *IncrementalSyncService) getFoldersToSync(ctx context.Context, strategy *SyncStrategy) ([]*models.Folder, error) {
	query := s.db.Where("account_id = ?", strategy.AccountID)
	
	if len(strategy.FolderIDs) > 0 {
		query = query.Where("id IN ?", strategy.FolderIDs)
	}
	
	var folders []*models.Folder
	if err := query.Find(&folders).Error; err != nil {
		return nil, fmt.Errorf("failed to get folders: %w", err)
	}
	
	return folders, nil
}

// syncSingleFolder 同步单个文件夹
func (s *IncrementalSyncService) syncSingleFolder(
	ctx context.Context,
	provider providers.EmailProvider,
	account *models.EmailAccount,
	folder *models.Folder,
	strategy *SyncStrategy,
	result *folderSyncResult,
) error {
	// 获取最后同步的UID
	lastUID, err := s.getLastSyncUID(folder.ID)
	if err != nil {
		return fmt.Errorf("failed to get last UID: %w", err)
	}

	// 获取新邮件
	newEmails, err := provider.SyncEmails(ctx, account, folder.Name, lastUID)
	if err != nil {
		return fmt.Errorf("failed to sync emails: %w", err)
	}

	if len(newEmails) == 0 {
		return nil
	}

	// 应用策略过滤
	filteredEmails := s.applyStrategy(newEmails, strategy)
	result.ProcessedEmails = len(filteredEmails)

	// 批量保存邮件
	newCount, updateCount, err := s.batchSaveEmails(ctx, filteredEmails, account.ID, folder.ID, account.UserID)
	if err != nil {
		return fmt.Errorf("failed to batch save emails: %w", err)
	}

	result.NewEmails = newCount
	result.UpdatedEmails = updateCount

	return nil
}

// getLastSyncUID 获取最后同步的UID
func (s *IncrementalSyncService) getLastSyncUID(folderID uint) (uint32, error) {
	var lastEmail models.Email
	err := s.db.Where("folder_id = ?", folderID).
		Order("uid DESC").
		First(&lastEmail).Error
	
	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	
	return lastEmail.UID, nil
}

// applyStrategy 应用同步策略过滤邮件
func (s *IncrementalSyncService) applyStrategy(emails []*providers.EmailMessage, strategy *SyncStrategy) []*providers.EmailMessage {
	if strategy.MaxEmailsPerSync > 0 && len(emails) > strategy.MaxEmailsPerSync {
		emails = emails[:strategy.MaxEmailsPerSync]
	}
	
	if strategy.SinceTime != nil {
		filtered := make([]*providers.EmailMessage, 0, len(emails))
		for _, email := range emails {
			if email.Date.After(*strategy.SinceTime) {
				filtered = append(filtered, email)
			}
		}
		emails = filtered
	}
	
	return emails
}

// batchSaveEmails 批量保存邮件
func (s *IncrementalSyncService) batchSaveEmails(
	ctx context.Context,
	emails []*providers.EmailMessage,
	accountID, folderID, userID uint,
) (newCount, updateCount int, err error) {
	if len(emails) == 0 {
		return 0, 0, nil
	}

	// 获取账户信息以确定提供商类型
	var account models.EmailAccount
	if err := s.db.First(&account, accountID).Error; err != nil {
		return 0, 0, fmt.Errorf("failed to get account: %w", err)
	}

	// 创建去重器
	deduplicator := s.deduplicatorFactory.CreateDeduplicator(account.Provider)

	// 分批处理邮件
	for i := 0; i < len(emails); i += s.batchSize {
		end := i + s.batchSize
		if end > len(emails) {
			end = len(emails)
		}

		batch := emails[i:end]
		batchNewCount, batchUpdateCount, err := s.processBatch(ctx, batch, deduplicator, accountID, folderID, userID)
		if err != nil {
			log.Printf("Failed to process batch %d-%d: %v", i, end-1, err)
			// 继续处理下一批，不中断整个同步
			continue
		}

		newCount += batchNewCount
		updateCount += batchUpdateCount
	}

	return newCount, updateCount, nil
}

// processBatch 处理一批邮件
func (s *IncrementalSyncService) processBatch(
	ctx context.Context,
	batch []*providers.EmailMessage,
	deduplicator EmailDeduplicator,
	accountID, folderID, userID uint,
) (newCount, updateCount int, err error) {
	// 使用事务处理整个批次
	return newCount, updateCount, s.db.Transaction(func(tx *gorm.DB) error {
		for _, emailMsg := range batch {
			// 检查重复
			duplicateResult, err := deduplicator.CheckDuplicate(ctx, emailMsg, accountID, folderID)
			if err != nil {
				log.Printf("Failed to check duplicate for email %s: %v", emailMsg.MessageID, err)
				continue
			}

			if duplicateResult.IsDuplicate && duplicateResult.ExistingEmail != nil {
				// 更新现有邮件
				if err := s.updateExistingEmailInTx(tx, duplicateResult.ExistingEmail, emailMsg, folderID); err != nil {
					log.Printf("Failed to update existing email %s: %v", emailMsg.MessageID, err)
					continue
				}
				updateCount++
			} else {
				// 创建新邮件
				if err := s.createNewEmailInTx(tx, emailMsg, accountID, folderID, userID); err != nil {
					log.Printf("Failed to create new email %s: %v", emailMsg.MessageID, err)
					continue
				}
				newCount++
			}
		}
		return nil
	})
}

// createNewEmailInTx 在事务中创建新邮件
func (s *IncrementalSyncService) createNewEmailInTx(
	tx *gorm.DB,
	emailMsg *providers.EmailMessage,
	accountID, folderID, userID uint,
) error {
	// 创建邮件记录
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
	if len(emailMsg.To) > 0 {
		toAddresses := make([]string, len(emailMsg.To))
		for i, addr := range emailMsg.To {
			if addr.Name != "" {
				toAddresses[i] = fmt.Sprintf("%s <%s>", addr.Name, addr.Address)
			} else {
				toAddresses[i] = addr.Address
			}
		}
		email.To = fmt.Sprintf("%v", toAddresses)
	}

	// 设置抄送
	if len(emailMsg.CC) > 0 {
		ccAddresses := make([]string, len(emailMsg.CC))
		for i, addr := range emailMsg.CC {
			if addr.Name != "" {
				ccAddresses[i] = fmt.Sprintf("%s <%s>", addr.Name, addr.Address)
			} else {
				ccAddresses[i] = addr.Address
			}
		}
		email.CC = fmt.Sprintf("%v", ccAddresses)
	}

	// 保存邮件
	if err := tx.Create(email).Error; err != nil {
		return fmt.Errorf("failed to create email: %w", err)
	}

	// 处理附件
	if len(emailMsg.Attachments) > 0 {
		if err := s.saveAttachmentsInTx(tx, email.ID, emailMsg.Attachments); err != nil {
			log.Printf("Failed to save attachments for email %d: %v", email.ID, err)
			// 附件保存失败不影响邮件保存
		}
	}

	// 发布新邮件事件（在事务外部）
	go func() {
		if s.eventPublisher != nil {
			newEmailEvent := sse.NewNewEmailEvent(email, userID)
			if err := s.eventPublisher.PublishToUser(context.Background(), userID, newEmailEvent); err != nil {
				log.Printf("Failed to publish new email event: %v", err)
			}
		}
	}()

	return nil
}

// updateExistingEmailInTx 在事务中更新现有邮件
func (s *IncrementalSyncService) updateExistingEmailInTx(
	tx *gorm.DB,
	email *models.Email,
	emailMsg *providers.EmailMessage,
	folderID uint,
) error {
	// 更新可能变化的字段
	email.FolderID = &folderID
	email.IsRead = s.isEmailRead(emailMsg.Flags)
	email.IsStarred = s.isEmailStarred(emailMsg.Flags)
	email.IsDraft = s.isEmailDraft(emailMsg.Flags)

	return tx.Save(email).Error
}

// saveAttachmentsInTx 在事务中保存附件
func (s *IncrementalSyncService) saveAttachmentsInTx(
	tx *gorm.DB,
	emailID uint,
	attachments []*providers.AttachmentInfo,
) error {
	for _, att := range attachments {
		attachment := &models.Attachment{
			EmailID:     &emailID, // 使用指针类型
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Size:        att.Size,
			PartID:      att.PartID,
			ContentID:   att.ContentID,
			Disposition: att.Disposition,
			Encoding:    att.Encoding,
		}

		if err := tx.Create(attachment).Error; err != nil {
			return fmt.Errorf("failed to create attachment %s: %w", att.Filename, err)
		}
	}
	return nil
}

// 辅助方法
func (s *IncrementalSyncService) isEmailRead(flags []string) bool {
	for _, flag := range flags {
		if flag == "\\Seen" {
			return true
		}
	}
	return false
}

func (s *IncrementalSyncService) isEmailStarred(flags []string) bool {
	for _, flag := range flags {
		if flag == "\\Flagged" {
			return true
		}
	}
	return false
}

func (s *IncrementalSyncService) isEmailDraft(flags []string) bool {
	for _, flag := range flags {
		if flag == "\\Draft" {
			return true
		}
	}
	return false
}
