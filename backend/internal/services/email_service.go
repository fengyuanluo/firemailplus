package services

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"firemail/internal/cache"
	"firemail/internal/config"
	"firemail/internal/models"
	"firemail/internal/providers"
	"firemail/internal/sse"

	"gorm.io/gorm"
)

// EmailService 邮件服务接口
type EmailService interface {
	// 邮件账户管理
	CreateEmailAccount(ctx context.Context, userID uint, req *CreateEmailAccountRequest) (*models.EmailAccount, error)
	GetEmailAccounts(ctx context.Context, userID uint) ([]*models.EmailAccount, error)
	GetEmailAccount(ctx context.Context, userID, accountID uint) (*models.EmailAccount, error)
	UpdateEmailAccount(ctx context.Context, userID, accountID uint, req *UpdateEmailAccountRequest) (*models.EmailAccount, error)
	DeleteEmailAccount(ctx context.Context, userID, accountID uint) error
	TestEmailAccount(ctx context.Context, userID, accountID uint) error

	// 邮件同步
	SyncEmails(ctx context.Context, accountID uint) error
	SyncEmailsForUser(ctx context.Context, userID uint) error
	SyncFolder(ctx context.Context, accountID uint, folderName string) error

	// 邮件操作
	GetEmails(ctx context.Context, userID uint, req *GetEmailsRequest) (*GetEmailsResponse, error)
	GetEmail(ctx context.Context, userID, emailID uint) (*models.Email, error)
	SendEmail(ctx context.Context, userID uint, req *SendEmailRequest) error
	DeleteEmail(ctx context.Context, userID, emailID uint) error
	MarkEmailAsRead(ctx context.Context, userID, emailID uint) error
	MarkEmailAsUnread(ctx context.Context, userID, emailID uint) error
	MarkAccountAsRead(ctx context.Context, userID, accountID uint) error
	MarkAccountsAsRead(ctx context.Context, userID uint, accountIDs []uint) error
	ToggleEmailStar(ctx context.Context, userID, emailID uint) error
	ToggleEmailImportant(ctx context.Context, userID, emailID uint) error
	MoveEmail(ctx context.Context, userID, emailID uint, targetFolderID uint) error

	// 邮件回复、转发、归档操作
	ReplyEmail(ctx context.Context, userID, emailID uint, req *ReplyEmailRequest) error
	ReplyAllEmail(ctx context.Context, userID, emailID uint, req *ReplyEmailRequest) error
	ForwardEmail(ctx context.Context, userID, emailID uint, req *ForwardEmailRequest) error
	ArchiveEmail(ctx context.Context, userID, emailID uint) error

	// 文件夹管理
	GetFolders(ctx context.Context, userID, accountID uint) ([]*models.Folder, error)
	GetFolder(ctx context.Context, userID, folderID uint) (*models.Folder, error)
	CreateFolder(ctx context.Context, userID, accountID uint, req *CreateFolderRequest) (*models.Folder, error)
	UpdateFolder(ctx context.Context, userID, folderID uint, req *UpdateFolderRequest) (*models.Folder, error)
	DeleteFolder(ctx context.Context, userID, folderID uint) error
	MarkFolderAsRead(ctx context.Context, userID, folderID uint) error
	SyncSpecificFolder(ctx context.Context, userID, folderID uint) error

	// 邮箱分组管理
	GetEmailGroups(ctx context.Context, userID uint) ([]*models.EmailGroup, error)
	CreateEmailGroup(ctx context.Context, userID uint, req *CreateEmailGroupRequest) (*models.EmailGroup, error)
	UpdateEmailGroup(ctx context.Context, userID, groupID uint, req *UpdateEmailGroupRequest) (*models.EmailGroup, error)
	DeleteEmailGroup(ctx context.Context, userID, groupID uint) error
	ReorderEmailGroups(ctx context.Context, userID uint, order []uint) ([]*models.EmailGroup, error)
	MoveAccountToGroup(ctx context.Context, userID, accountID uint, groupID *uint) error
	SetDefaultEmailGroup(ctx context.Context, userID, groupID uint) (*models.EmailGroup, error)

	// 搜索
	SearchEmails(ctx context.Context, userID uint, req *SearchEmailsRequest) (*GetEmailsResponse, error)
}

// EmailServiceImpl 邮件服务实现
type EmailServiceImpl struct {
	db                *gorm.DB
	providerFactory   *providers.ProviderFactory
	eventPublisher    sse.EventPublisher
	syncService       *SyncService // 添加同步服务依赖
	cacheManager      *cache.CacheManager
	attachmentService AttachmentDownloader // 添加附件服务依赖
}

// NewEmailService 创建邮件服务实例
func NewEmailService(db *gorm.DB, providerFactory *providers.ProviderFactory, eventPublisher sse.EventPublisher) EmailService {
	return &EmailServiceImpl{
		db:              db,
		providerFactory: providerFactory,
		eventPublisher:  eventPublisher,
		cacheManager:    cache.GlobalCacheManager,
	}
}

// SetSyncService 设置同步服务依赖
func (s *EmailServiceImpl) SetSyncService(syncService *SyncService) {
	s.syncService = syncService
}

// SetAttachmentService 设置附件服务依赖
func (s *EmailServiceImpl) SetAttachmentService(attachmentService AttachmentDownloader) {
	s.attachmentService = attachmentService
}

// 请求和响应结构体

// CreateEmailAccountRequest 创建邮件账户请求
type CreateEmailAccountRequest struct {
	Name         string `json:"name" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	Provider     string `json:"provider"`
	AuthMethod   string `json:"auth_method" binding:"required,oneof=password oauth2"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	IMAPHost     string `json:"imap_host"`
	IMAPPort     int    `json:"imap_port"`
	IMAPSecurity string `json:"imap_security"`
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPSecurity string `json:"smtp_security"`
	GroupID      *uint  `json:"group_id"`
}

// UpdateEmailAccountRequest 更新邮件账户请求
type UpdateEmailAccountRequest struct {
	Name         *string `json:"name"`
	Password     *string `json:"password"`
	IMAPHost     *string `json:"imap_host"`
	IMAPPort     *int    `json:"imap_port"`
	IMAPSecurity *string `json:"imap_security"`
	SMTPHost     *string `json:"smtp_host"`
	SMTPPort     *int    `json:"smtp_port"`
	SMTPSecurity *string `json:"smtp_security"`
	IsActive     *bool   `json:"is_active"`
	GroupID      *uint   `json:"group_id"`
}

// GetEmailsRequest 获取邮件列表请求
type GetEmailsRequest struct {
	AccountID   *uint  `json:"account_id"`
	FolderID    *uint  `json:"folder_id"`
	IsRead      *bool  `json:"is_read"`
	IsStarred   *bool  `json:"is_starred"`
	IsImportant *bool  `json:"is_important"`
	Page        int    `json:"page"`
	PageSize    int    `json:"page_size"`
	SortBy      string `json:"sort_by"`
	SortOrder   string `json:"sort_order"`
	SearchQuery string `json:"search_query"`
}

// GetEmailsResponse 获取邮件列表响应
type GetEmailsResponse struct {
	Emails     []*models.Email `json:"emails"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}

// SendEmailRequest 发送邮件请求
type SendEmailRequest struct {
	AccountID     uint                   `json:"account_id" binding:"required"`
	To            []*models.EmailAddress `json:"to" binding:"required"`
	CC            []*models.EmailAddress `json:"cc"`
	BCC           []*models.EmailAddress `json:"bcc"`
	Subject       string                 `json:"subject" binding:"required"`
	TextBody      string                 `json:"text_body"`
	HTMLBody      string                 `json:"html_body"`
	Attachments   []*SendEmailAttachment `json:"attachments"`
	AttachmentIDs []uint                 `json:"attachment_ids"`
	Priority      string                 `json:"priority"`
	ReplyToID     *uint                  `json:"reply_to_id"`
}

// SendEmailAttachment 发送邮件附件
type SendEmailAttachment struct {
	Filename    string `json:"filename" binding:"required"`
	ContentType string `json:"content_type"`
	Content     []byte `json:"content" binding:"required"`
	Size        int64  `json:"size"`
	Disposition string `json:"disposition"`
	ContentID   string `json:"content_id"`
}

// CreateFolderRequest 创建文件夹请求
type CreateFolderRequest struct {
	Name        string `json:"name" binding:"required"`
	DisplayName string `json:"display_name"`
	ParentID    *uint  `json:"parent_id"`
}

// UpdateFolderRequest 更新文件夹请求
type UpdateFolderRequest struct {
	Name        *string `json:"name"`
	DisplayName *string `json:"display_name"`
	ParentID    *uint   `json:"parent_id"`
}

// CreateEmailGroupRequest 创建邮箱分组请求
type CreateEmailGroupRequest struct {
	Name string `json:"name" binding:"required"`
}

// UpdateEmailGroupRequest 更新邮箱分组请求
type UpdateEmailGroupRequest struct {
	Name *string `json:"name"`
}

// SearchEmailsRequest 搜索邮件请求
type SearchEmailsRequest struct {
	AccountID     *uint      `json:"account_id"`
	FolderID      *uint      `json:"folder_id"`
	Query         string     `json:"query" binding:"required"`
	Subject       string     `json:"subject"`
	From          string     `json:"from"`
	To            string     `json:"to"`
	Body          string     `json:"body"`
	Since         *time.Time `json:"since"`
	Before        *time.Time `json:"before"`
	HasAttachment *bool      `json:"has_attachment"`
	IsRead        *bool      `json:"is_read"`
	IsStarred     *bool      `json:"is_starred"`
	Page          int        `json:"page"`
	PageSize      int        `json:"page_size"`
}

// ReplyEmailRequest 回复邮件请求
type ReplyEmailRequest struct {
	AccountID uint                  `json:"account_id" binding:"required"`
	To        []models.EmailAddress `json:"to,omitempty"`
	CC        []models.EmailAddress `json:"cc,omitempty"`
	BCC       []models.EmailAddress `json:"bcc,omitempty"`
	Subject   string                `json:"subject"`
	TextBody  string                `json:"text_body"`
	HTMLBody  string                `json:"html_body"`
}

// ForwardEmailRequest 转发邮件请求
type ForwardEmailRequest struct {
	AccountID uint                  `json:"account_id" binding:"required"`
	To        []models.EmailAddress `json:"to" binding:"required"`
	CC        []models.EmailAddress `json:"cc,omitempty"`
	BCC       []models.EmailAddress `json:"bcc,omitempty"`
	Subject   string                `json:"subject"`
	TextBody  string                `json:"text_body"`
	HTMLBody  string                `json:"html_body"`
}

// CreateEmailAccount 创建邮件账户
func (s *EmailServiceImpl) CreateEmailAccount(ctx context.Context, userID uint, req *CreateEmailAccountRequest) (*models.EmailAccount, error) {
	// 验证用户是否存在
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// 自动检测提供商（如果未指定）
	if req.Provider == "" {
		providerConfig := s.providerFactory.DetectProvider(req.Email)
		if providerConfig != nil {
			req.Provider = providerConfig.Name
		} else {
			req.Provider = "custom"
		}
	}

	// 获取提供商配置
	providerConfig := s.providerFactory.GetProviderConfig(req.Provider)
	if providerConfig == nil {
		return nil, fmt.Errorf("unknown provider: %s", req.Provider)
	}

	// 解析目标分组（为空则回退到默认分组）
	targetGroup, err := s.resolveAccountGroup(ctx, userID, req.GroupID)
	if err != nil {
		return nil, fmt.Errorf("invalid group: %w", err)
	}

	// 创建邮件账户
	account := &models.EmailAccount{
		UserID:     userID,
		Name:       req.Name,
		Email:      req.Email,
		Provider:   req.Provider,
		AuthMethod: req.AuthMethod,
		IsActive:   true,
		SyncStatus: "pending",
		GroupID:    &targetGroup.ID,
	}

	// 根据邮箱类型设置配置
	if err := s.configureAccountByProvider(account, req, providerConfig); err != nil {
		return nil, fmt.Errorf("failed to configure account: %w", err)
	}

	// 调试日志
	log.Printf("Account before validation: Provider=%s, IMAPHost=%s, IMAPPort=%d, SMTPHost=%s, SMTPPort=%d",
		account.Provider, account.IMAPHost, account.IMAPPort, account.SMTPHost, account.SMTPPort)

	// 验证配置
	if err := s.providerFactory.ValidateProviderConfig(account); err != nil {
		return nil, fmt.Errorf("invalid provider configuration: %w", err)
	}

	// 保存到数据库
	if err := s.db.Create(account).Error; err != nil {
		return nil, fmt.Errorf("failed to create email account: %w", err)
	}

	// 测试连接
	if err := s.TestEmailAccount(ctx, userID, account.ID); err != nil {
		// 如果测试失败，标记为错误状态但不删除账户
		account.SyncStatus = "error"
		account.ErrorMessage = err.Error()
		s.db.Save(account)
	} else {
		// 测试成功，开始同步文件夹
		go func() {
			if err := s.syncFoldersForAccount(context.Background(), account.ID); err != nil {
				// 记录错误但不影响账户创建
				account.SyncStatus = "error"
				account.ErrorMessage = fmt.Sprintf("Failed to sync folders: %v", err)
				s.db.Save(account)
			}
		}()
	}

	return account, nil
}

// GetEmailAccounts 获取用户的邮件账户列表
func (s *EmailServiceImpl) GetEmailAccounts(ctx context.Context, userID uint) ([]*models.EmailAccount, error) {
	var accounts []*models.EmailAccount

	err := s.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&accounts).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get email accounts: %w", err)
	}

	// 确保存在默认分组并回填缺失的分组信息
	defaultGroup, err := s.ensureDefaultGroup(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure default group: %w", err)
	}

	for _, account := range accounts {
		if account.GroupID == nil && defaultGroup != nil {
			account.GroupID = &defaultGroup.ID
			_ = s.db.Model(&models.EmailAccount{}).
				Where("id = ?", account.ID).
				Update("group_id", defaultGroup.ID).Error
		}
	}

	return accounts, nil
}

// GetEmailAccount 获取指定的邮件账户
func (s *EmailServiceImpl) GetEmailAccount(ctx context.Context, userID, accountID uint) (*models.EmailAccount, error) {
	var account models.EmailAccount

	err := s.db.Where("id = ? AND user_id = ?", accountID, userID).
		First(&account).Error

	if err != nil {
		return nil, fmt.Errorf("email account not found: %w", err)
	}

	return &account, nil
}

// UpdateEmailAccount 更新邮件账户
func (s *EmailServiceImpl) UpdateEmailAccount(ctx context.Context, userID, accountID uint, req *UpdateEmailAccountRequest) (*models.EmailAccount, error) {
	account, err := s.GetEmailAccount(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}

	// 更新字段
	if req.Name != nil {
		account.Name = *req.Name
	}
	if req.Password != nil {
		account.Password = *req.Password
	}
	if req.IMAPHost != nil {
		account.IMAPHost = *req.IMAPHost
	}
	if req.IMAPPort != nil {
		account.IMAPPort = *req.IMAPPort
	}
	if req.IMAPSecurity != nil {
		account.IMAPSecurity = *req.IMAPSecurity
	}
	if req.SMTPHost != nil {
		account.SMTPHost = *req.SMTPHost
	}
	if req.SMTPPort != nil {
		account.SMTPPort = *req.SMTPPort
	}
	if req.SMTPSecurity != nil {
		account.SMTPSecurity = *req.SMTPSecurity
	}
	if req.IsActive != nil {
		account.IsActive = *req.IsActive
	}
	if req.GroupID != nil {
		targetGroup, err := s.resolveAccountGroup(ctx, userID, req.GroupID)
		if err != nil {
			return nil, fmt.Errorf("invalid group: %w", err)
		}
		account.GroupID = &targetGroup.ID
	}

	// 验证更新后的配置
	if err := s.providerFactory.ValidateProviderConfig(account); err != nil {
		return nil, fmt.Errorf("invalid provider configuration: %w", err)
	}

	// 保存更新
	if err := s.db.Save(account).Error; err != nil {
		return nil, fmt.Errorf("failed to update email account: %w", err)
	}

	// 如果更新了连接相关的配置，测试连接
	if req.Password != nil || req.IMAPHost != nil || req.IMAPPort != nil ||
		req.IMAPSecurity != nil || req.SMTPHost != nil || req.SMTPPort != nil ||
		req.SMTPSecurity != nil {
		if err := s.TestEmailAccount(ctx, userID, accountID); err != nil {
			account.SyncStatus = "error"
			account.ErrorMessage = err.Error()
			s.db.Save(account)
		}
	}

	return account, nil
}

// DeleteEmailAccount 删除邮件账户
func (s *EmailServiceImpl) DeleteEmailAccount(ctx context.Context, userID, accountID uint) error {
	// 验证账户存在且属于用户
	account, err := s.GetEmailAccount(ctx, userID, accountID)
	if err != nil {
		return err
	}

	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除相关的附件（使用子查询避免外键约束问题）
	if err := tx.Unscoped().Where("email_id IN (SELECT id FROM emails WHERE account_id = ?)", accountID).Delete(&models.Attachment{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete attachments: %w", err)
	}

	// 删除相关的邮件（硬删除）
	if err := tx.Unscoped().Where("account_id = ?", accountID).Delete(&models.Email{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete emails: %w", err)
	}

	// 删除相关的文件夹（硬删除）
	if err := tx.Unscoped().Where("account_id = ?", accountID).Delete(&models.Folder{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete folders: %w", err)
	}

	// 删除账户（硬删除）
	if err := tx.Unscoped().Delete(account).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete email account: %w", err)
	}

	return tx.Commit().Error
}

// TestEmailAccount 测试邮件账户连接
func (s *EmailServiceImpl) TestEmailAccount(ctx context.Context, userID, accountID uint) error {
	account, err := s.GetEmailAccount(ctx, userID, accountID)
	if err != nil {
		return err
	}

	// 创建提供商实例
	provider, err := s.providerFactory.CreateProviderForAccount(account)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	// 设置OAuth2 token更新回调
	s.setupProviderTokenCallback(provider)

	// 测试连接
	if err := provider.TestConnection(ctx, account); err != nil {
		// 发布账户连接错误事件
		if s.eventPublisher != nil {
			accountEvent := sse.NewAccountEvent(sse.EventAccountError, account.ID, account.Email, account.Provider, userID)
			if accountEvent.Data != nil {
				if accountData, ok := accountEvent.Data.(*sse.AccountEventData); ok {
					accountData.ErrorMessage = err.Error()
				}
			}
			if publishErr := s.eventPublisher.PublishToUser(ctx, userID, accountEvent); publishErr != nil {
				log.Printf("Failed to publish account error event: %v", publishErr)
			}
		}
		return fmt.Errorf("connection test failed: %w", err)
	}

	// 更新状态
	account.SyncStatus = "success"
	account.ErrorMessage = ""
	account.LastSyncAt = &time.Time{}
	*account.LastSyncAt = time.Now()

	// 发布账户连接成功事件
	if s.eventPublisher != nil {
		accountEvent := sse.NewAccountEvent(sse.EventAccountConnected, account.ID, account.Email, account.Provider, userID)
		if err := s.eventPublisher.PublishToUser(ctx, userID, accountEvent); err != nil {
			log.Printf("Failed to publish account connected event: %v", err)
		}
	}

	return s.db.Save(account).Error
}

// setupProviderTokenCallback 为provider设置OAuth2 token更新回调
func (s *EmailServiceImpl) setupProviderTokenCallback(provider providers.EmailProvider) {
	// 设置OAuth2 token更新回调（如果支持）
	if tokenSetter, ok := provider.(providers.TokenCallbackSetter); ok {
		tokenSetter.SetTokenUpdateCallback(func(ctx context.Context, account *models.EmailAccount) error {
			// 使用Select只更新OAuth2Token字段，避免触发其他钩子和触发器
			return s.db.Model(account).Select("oauth2_token").Updates(map[string]interface{}{
				"oauth2_token": account.OAuth2Token,
			}).Error
		})
	}
}

// syncFoldersForAccount 同步账户的文件夹
func (s *EmailServiceImpl) syncFoldersForAccount(ctx context.Context, accountID uint) error {
	var account models.EmailAccount
	if err := s.db.First(&account, accountID).Error; err != nil {
		return fmt.Errorf("account not found: %w", err)
	}

	// 创建提供商实例
	provider, err := s.providerFactory.CreateProviderForAccount(&account)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	// 设置OAuth2 token更新回调
	s.setupProviderTokenCallback(provider)

	// 连接到服务器
	if err := provider.Connect(ctx, &account); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer provider.Disconnect()

	// 获取IMAP客户端
	imapClient := provider.IMAPClient()
	if imapClient == nil {
		return fmt.Errorf("IMAP client not available")
	}

	// 获取文件夹列表
	folders, err := imapClient.ListFolders(ctx)
	if err != nil {
		return fmt.Errorf("failed to list folders: %w", err)
	}

	// 保存文件夹到数据库
	for _, folderInfo := range folders {
		folder := &models.Folder{
			AccountID:    accountID,
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
		err := s.db.Where("account_id = ? AND path = ?", accountID, folderInfo.Path).
			First(&existingFolder).Error

		if err == gorm.ErrRecordNotFound {
			// 创建新文件夹
			if err := s.db.Create(folder).Error; err != nil {
				return fmt.Errorf("failed to create folder %s: %w", folderInfo.Name, err)
			}
		} else if err == nil {
			// 更新现有文件夹
			existingFolder.Name = folderInfo.Name
			existingFolder.DisplayName = folderInfo.DisplayName
			existingFolder.Type = folderInfo.Type
			existingFolder.IsSelectable = folderInfo.IsSelectable
			existingFolder.IsSubscribed = folderInfo.IsSubscribed

			if err := s.db.Save(&existingFolder).Error; err != nil {
				return fmt.Errorf("failed to update folder %s: %w", folderInfo.Name, err)
			}
		} else {
			return fmt.Errorf("failed to check folder existence: %w", err)
		}
	}

	return nil
}

// ensureDefaultGroup 确保存在默认分组
func (s *EmailServiceImpl) ensureDefaultGroup(ctx context.Context, userID uint) (*models.EmailGroup, error) {
	var group models.EmailGroup
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND is_default = ?", userID, true).
		First(&group).Error

	if err == gorm.ErrRecordNotFound {
		group = models.EmailGroup{
			UserID:    userID,
			Name:      "未分组",
			SortOrder: 0,
			IsDefault: true,
		}
		if createErr := s.db.WithContext(ctx).Create(&group).Error; createErr != nil {
			return nil, fmt.Errorf("failed to create default group: %w", createErr)
		}
		return &group, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load default group: %w", err)
	}

	return &group, nil
}

// resolveAccountGroup 解析账户分组（为空时返回默认分组）
func (s *EmailServiceImpl) resolveAccountGroup(ctx context.Context, userID uint, groupID *uint) (*models.EmailGroup, error) {
	defaultGroup, err := s.ensureDefaultGroup(ctx, userID)
	if err != nil {
		return nil, err
	}

	if groupID == nil {
		return defaultGroup, nil
	}

	var group models.EmailGroup
	err = s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", *groupID, userID).
		First(&group).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("分组不存在")
		}
		return nil, err
	}

	return &group, nil
}

// GetEmailGroups 获取分组列表（包含账户数量）
func (s *EmailServiceImpl) GetEmailGroups(ctx context.Context, userID uint) ([]*models.EmailGroup, error) {
	if _, err := s.ensureDefaultGroup(ctx, userID); err != nil {
		return nil, err
	}

	var groups []*models.EmailGroup
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("is_default DESC, sort_order ASC, id ASC").
		Find(&groups).Error; err != nil {
		return nil, fmt.Errorf("failed to load groups: %w", err)
	}

	var counts []struct {
		GroupID *uint
		Count   int64
	}
	if err := s.db.WithContext(ctx).
		Model(&models.EmailAccount{}).
		Select("group_id, COUNT(*) as count").
		Where("user_id = ?", userID).
		Group("group_id").
		Scan(&counts).Error; err != nil {
		return nil, fmt.Errorf("failed to count accounts: %w", err)
	}

	countMap := make(map[uint]int64)
	var ungroupedCount int64
	for _, c := range counts {
		if c.GroupID == nil {
			ungroupedCount += c.Count
			continue
		}
		countMap[*c.GroupID] = c.Count
	}

	for _, group := range groups {
		group.AccountCnt = countMap[group.ID]
		if group.IsDefault {
			group.AccountCnt += ungroupedCount
		}
	}

	return groups, nil
}

// CreateEmailGroup 创建分组
func (s *EmailServiceImpl) CreateEmailGroup(ctx context.Context, userID uint, req *CreateEmailGroupRequest) (*models.EmailGroup, error) {
	if req == nil || strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("分组名称不能为空")
	}

	if _, err := s.ensureDefaultGroup(ctx, userID); err != nil {
		return nil, err
	}

	var maxOrder int
	if err := s.db.WithContext(ctx).
		Model(&models.EmailGroup{}).
		Where("user_id = ? AND is_default = 0", userID).
		Select("COALESCE(MAX(sort_order), 0)").
		Scan(&maxOrder).Error; err != nil {
		return nil, fmt.Errorf("failed to calculate sort order: %w", err)
	}

	group := &models.EmailGroup{
		UserID:    userID,
		Name:      strings.TrimSpace(req.Name),
		SortOrder: maxOrder + 1,
		IsDefault: false,
	}

	if err := s.db.WithContext(ctx).Create(group).Error; err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	// 如果这是用户创建的第一个自定义分组，则将其设为默认分组
	var nonDefaultCount int64
	if err := s.db.WithContext(ctx).Model(&models.EmailGroup{}).
		Where("user_id = ? AND is_default = 0", userID).
		Count(&nonDefaultCount).Error; err == nil && nonDefaultCount == 1 {
		if _, err := s.SetDefaultEmailGroup(ctx, userID, group.ID); err != nil {
			log.Printf("Warning: failed to set first group as default: %v", err)
		}
	}

	return group, nil
}

// UpdateEmailGroup 更新分组
func (s *EmailServiceImpl) UpdateEmailGroup(ctx context.Context, userID, groupID uint, req *UpdateEmailGroupRequest) (*models.EmailGroup, error) {
	var group models.EmailGroup
	if err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", groupID, userID).
		First(&group).Error; err != nil {
		return nil, fmt.Errorf("group not found: %w", err)
	}

	if group.IsDefault {
		return nil, fmt.Errorf("默认分组不可编辑")
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, fmt.Errorf("分组名称不能为空")
		}
		group.Name = name
	}

	if err := s.db.WithContext(ctx).Save(&group).Error; err != nil {
		return nil, fmt.Errorf("failed to update group: %w", err)
	}

	return &group, nil
}

// DeleteEmailGroup 删除分组并将账户回退到默认分组
func (s *EmailServiceImpl) DeleteEmailGroup(ctx context.Context, userID, groupID uint) error {
	var group models.EmailGroup
	if err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", groupID, userID).
		First(&group).Error; err != nil {
		return fmt.Errorf("group not found: %w", err)
	}

	if group.IsDefault {
		return fmt.Errorf("默认分组不可删除")
	}

	defaultGroup, err := s.ensureDefaultGroup(ctx, userID)
	if err != nil {
		return err
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.EmailAccount{}).
			Where("user_id = ? AND group_id = ?", userID, groupID).
			Update("group_id", defaultGroup.ID).Error; err != nil {
			return fmt.Errorf("failed to move accounts to default group: %w", err)
		}

		if err := tx.Delete(&group).Error; err != nil {
			return fmt.Errorf("failed to delete group: %w", err)
		}

		return nil
	})
}

// ReorderEmailGroups 调整分组排序
func (s *EmailServiceImpl) ReorderEmailGroups(ctx context.Context, userID uint, order []uint) ([]*models.EmailGroup, error) {
	if _, err := s.ensureDefaultGroup(ctx, userID); err != nil {
		return nil, err
	}

	var groups []models.EmailGroup
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&groups).Error; err != nil {
		return nil, fmt.Errorf("failed to load groups: %w", err)
	}

	groupMap := make(map[uint]models.EmailGroup)
	for _, g := range groups {
		groupMap[g.ID] = g
	}

	for _, id := range order {
		if g, ok := groupMap[id]; !ok || g.UserID != userID {
			return nil, fmt.Errorf("invalid group id: %d", id)
		}
	}

	listed := make(map[uint]bool)
	sortOrder := 1

	tx := s.db.WithContext(ctx).Begin()
	for _, id := range order {
		g := groupMap[id]
		if g.IsDefault {
			continue
		}
		listed[id] = true
		if err := tx.Model(&models.EmailGroup{}).
			Where("id = ? AND user_id = ?", id, userID).
			Update("sort_order", sortOrder).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to update sort order: %w", err)
		}
		sortOrder++
	}

	for _, g := range groups {
		if g.IsDefault || listed[g.ID] {
			continue
		}
		if err := tx.Model(&models.EmailGroup{}).
			Where("id = ? AND user_id = ?", g.ID, userID).
			Update("sort_order", sortOrder).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to finalize sort order: %w", err)
		}
		sortOrder++
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit sort order: %w", err)
	}

	return s.GetEmailGroups(ctx, userID)
}

// MoveAccountToGroup 将账户移动到指定分组
func (s *EmailServiceImpl) MoveAccountToGroup(ctx context.Context, userID, accountID uint, groupID *uint) error {
	account, err := s.GetEmailAccount(ctx, userID, accountID)
	if err != nil {
		return err
	}

	targetGroup, err := s.resolveAccountGroup(ctx, userID, groupID)
	if err != nil {
		return err
	}

	account.GroupID = &targetGroup.ID
	return s.db.WithContext(ctx).Save(account).Error
}

// SetDefaultEmailGroup 设置默认分组
func (s *EmailServiceImpl) SetDefaultEmailGroup(ctx context.Context, userID, groupID uint) (*models.EmailGroup, error) {
	var target models.EmailGroup
	if err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", groupID, userID).
		First(&target).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("分组不存在")
		}
		return nil, err
	}

	if target.IsDefault {
		return &target, nil
	}

	var prevDefault models.EmailGroup
	_ = s.db.WithContext(ctx).
		Where("user_id = ? AND is_default = 1", userID).
		First(&prevDefault).Error

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 取消其他默认
		if err := tx.Model(&models.EmailGroup{}).
			Where("user_id = ?", userID).
			Update("is_default", false).Error; err != nil {
			return fmt.Errorf("failed to reset default flag: %w", err)
		}

		// 将目标设为默认并置顶排序
		if err := tx.Model(&models.EmailGroup{}).
			Where("id = ? AND user_id = ?", groupID, userID).
			Updates(map[string]interface{}{
				"is_default": true,
				"sort_order": 0,
			}).Error; err != nil {
			return fmt.Errorf("failed to set default group: %w", err)
		}

		// 将其他分组排序顺延，保持默认在前
		if err := tx.Model(&models.EmailGroup{}).
			Where("user_id = ? AND id != ?", userID, groupID).
			Update("sort_order", gorm.Expr("sort_order + 1")).Error; err != nil {
			return fmt.Errorf("failed to normalize sort order: %w", err)
		}

		// 若存在旧默认分组，将其账户及未分组账户迁移到新默认
		var defaultID uint = groupID
		if prevDefault.ID != 0 {
			if err := tx.Model(&models.EmailAccount{}).
				Where("user_id = ? AND (group_id IS NULL OR group_id = ?)", userID, prevDefault.ID).
				Update("group_id", defaultID).Error; err != nil {
				return fmt.Errorf("failed to move accounts to new default: %w", err)
			}
		} else {
			if err := tx.Model(&models.EmailAccount{}).
				Where("user_id = ? AND group_id IS NULL", userID).
				Update("group_id", defaultID).Error; err != nil {
				return fmt.Errorf("failed to move ungrouped accounts: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).
		Where("id = ?", groupID).
		First(&target).Error; err != nil {
		return nil, err
	}

	return &target, nil
}

// configureAccountByProvider 根据提供商类型配置账户
func (s *EmailServiceImpl) configureAccountByProvider(account *models.EmailAccount, req *CreateEmailAccountRequest, providerConfig *config.EmailProviderConfig) error {
	// 设置用户名，默认使用邮箱地址
	if req.Username != "" {
		account.Username = req.Username
	} else {
		account.Username = account.Email
	}

	// 设置密码
	account.Password = req.Password

	// 根据提供商类型配置服务器设置
	switch account.Provider {
	case "qq", "163", "icloud", "sina":
		// 这些提供商使用固定配置，不允许自定义
		account.IMAPHost = providerConfig.IMAPHost
		account.IMAPPort = providerConfig.IMAPPort
		account.IMAPSecurity = providerConfig.IMAPSecurity
		account.SMTPHost = providerConfig.SMTPHost
		account.SMTPPort = providerConfig.SMTPPort
		account.SMTPSecurity = providerConfig.SMTPSecurity

	case "gmail", "outlook":
		// Gmail和Outlook使用固定配置，但支持OAuth2
		account.IMAPHost = providerConfig.IMAPHost
		account.IMAPPort = providerConfig.IMAPPort
		account.IMAPSecurity = providerConfig.IMAPSecurity
		account.SMTPHost = providerConfig.SMTPHost
		account.SMTPPort = providerConfig.SMTPPort
		account.SMTPSecurity = providerConfig.SMTPSecurity

	case "custom":
		// 自定义邮箱允许用户配置服务器设置
		if req.IMAPHost != "" {
			account.IMAPHost = req.IMAPHost
			account.IMAPPort = req.IMAPPort
			account.IMAPSecurity = req.IMAPSecurity
		}
		if req.SMTPHost != "" {
			account.SMTPHost = req.SMTPHost
			account.SMTPPort = req.SMTPPort
			account.SMTPSecurity = req.SMTPSecurity
		}

	default:
		// 其他提供商，如果有配置则使用，否则使用请求中的配置
		if providerConfig.IMAPHost != "" {
			account.IMAPHost = providerConfig.IMAPHost
			account.IMAPPort = providerConfig.IMAPPort
			account.IMAPSecurity = providerConfig.IMAPSecurity
		} else if req.IMAPHost != "" {
			account.IMAPHost = req.IMAPHost
			account.IMAPPort = req.IMAPPort
			account.IMAPSecurity = req.IMAPSecurity
		}

		if providerConfig.SMTPHost != "" {
			account.SMTPHost = providerConfig.SMTPHost
			account.SMTPPort = providerConfig.SMTPPort
			account.SMTPSecurity = providerConfig.SMTPSecurity
		} else if req.SMTPHost != "" {
			account.SMTPHost = req.SMTPHost
			account.SMTPPort = req.SMTPPort
			account.SMTPSecurity = req.SMTPSecurity
		}
	}

	return nil
}

// 实现缺少的接口方法

// SyncEmails 同步邮件（委托给SyncService）
func (s *EmailServiceImpl) SyncEmails(ctx context.Context, accountID uint) error {
	if s.syncService != nil {
		return s.syncService.SyncEmails(ctx, accountID)
	}
	return fmt.Errorf("sync service not available")
}

// SyncEmailsForUser 为用户同步所有邮件（委托给SyncService）
func (s *EmailServiceImpl) SyncEmailsForUser(ctx context.Context, userID uint) error {
	if s.syncService != nil {
		return s.syncService.SyncEmailsForUser(ctx, userID)
	}
	return fmt.Errorf("sync service not available")
}

// SyncFolder 同步文件夹（委托给SyncService）
func (s *EmailServiceImpl) SyncFolder(ctx context.Context, accountID uint, folderName string) error {
	if s.syncService != nil {
		return s.syncService.SyncFolder(ctx, accountID, folderName)
	}
	return fmt.Errorf("sync service not available")
}

// GetEmails 获取邮件列表
func (s *EmailServiceImpl) GetEmails(ctx context.Context, userID uint, req *GetEmailsRequest) (*GetEmailsResponse, error) {
	// 生成缓存键
	cacheKey := s.generateEmailListCacheKey(userID, req)

	// 尝试从缓存获取
	if cached, found := s.cacheManager.EmailListCache().Get(cacheKey); found {
		if response, ok := cached.(*GetEmailsResponse); ok {
			log.Printf("Cache hit for email list: %s", cacheKey)
			return response, nil
		}
	}

	// 构建查询
	query := s.db.Model(&models.Email{}).
		Joins("JOIN email_accounts ON emails.account_id = email_accounts.id").
		Where("email_accounts.user_id = ?", userID)

	// 添加过滤条件
	if req.AccountID != nil {
		query = query.Where("emails.account_id = ?", *req.AccountID)
	}

	if req.FolderID != nil {
		query = query.Where("emails.folder_id = ?", *req.FolderID)
	}

	if req.IsRead != nil {
		query = query.Where("emails.is_read = ?", *req.IsRead)
	}

	if req.IsStarred != nil {
		query = query.Where("emails.is_starred = ?", *req.IsStarred)
	}

	if req.IsImportant != nil {
		query = query.Where("emails.is_important = ?", *req.IsImportant)
	}

	// 搜索查询
	if req.SearchQuery != "" {
		searchPattern := "%" + req.SearchQuery + "%"
		query = query.Where("emails.subject LIKE ? OR emails.from_address LIKE ? OR emails.text_body LIKE ? OR emails.html_body LIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern)
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count emails: %w", err)
	}

	// 设置默认值
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	// 排序字段映射
	sortBy := req.SortBy
	if sortBy == "" {
		sortBy = "date"
	}

	// 映射前端字段名到数据库字段名
	switch sortBy {
	case "date", "received_at":
		sortBy = "date"
	case "subject":
		sortBy = "subject"
	case "from":
		sortBy = "from_address"
	case "size":
		sortBy = "size"
	default:
		sortBy = "date" // 默认按日期排序
	}

	sortOrder := req.SortOrder
	if sortOrder == "" {
		sortOrder = "DESC"
	}

	// 分页查询
	var emails []*models.Email
	offset := (page - 1) * pageSize
	err := query.Order(fmt.Sprintf("emails.%s %s", sortBy, sortOrder)).
		Limit(pageSize).
		Offset(offset).
		Find(&emails).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get emails: %w", err)
	}

	// 计算总页数
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	response := &GetEmailsResponse{
		Emails:     emails,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	// 缓存结果（缓存5分钟）
	s.cacheManager.EmailListCache().Set(cacheKey, response, 5*time.Minute)
	log.Printf("Cached email list: %s", cacheKey)

	return response, nil
}

// GetEmail 获取单个邮件
func (s *EmailServiceImpl) GetEmail(ctx context.Context, userID, emailID uint) (*models.Email, error) {
	var email models.Email

	// 查询邮件，确保用户只能访问自己的邮件
	err := s.db.WithContext(ctx).
		Joins("JOIN email_accounts ON emails.account_id = email_accounts.id").
		Where("emails.id = ? AND email_accounts.user_id = ?", emailID, userID).
		Preload("Account").
		Preload("Folder").
		Preload("Attachments").
		First(&email).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("email not found")
		}
		return nil, fmt.Errorf("failed to get email: %w", err)
	}

	return &email, nil
}

// SendEmail 发送邮件
func (s *EmailServiceImpl) SendEmail(ctx context.Context, userID uint, req *SendEmailRequest) error {
	// 验证账户存在且属于用户
	account, err := s.GetEmailAccount(ctx, userID, req.AccountID)
	if err != nil {
		return fmt.Errorf("invalid account: %w", err)
	}

	// 创建提供商实例
	provider, err := s.providerFactory.CreateProviderForAccount(account)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	// 设置OAuth2 token更新回调
	s.setupProviderTokenCallback(provider)

	// 连接到服务器
	if err := provider.Connect(ctx, account); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer provider.Disconnect()

	// 获取SMTP客户端
	smtpClient := provider.SMTPClient()
	if smtpClient == nil {
		return fmt.Errorf("SMTP client not available")
	}

	// 构建发送邮件消息
	message := &providers.OutgoingMessage{
		Subject:  req.Subject,
		TextBody: req.TextBody,
		HTMLBody: req.HTMLBody,
		To:       req.To,
		CC:       req.CC,
		BCC:      req.BCC,
		Priority: req.Priority,
	}

	// 设置发件人
	message.From = &models.EmailAddress{
		Name:    account.Name,
		Address: account.Email,
	}

	// 处理附件
	for _, attachment := range req.Attachments {
		message.Attachments = append(message.Attachments, &providers.OutgoingAttachment{
			Filename:    attachment.Filename,
			ContentType: attachment.ContentType,
			Content:     bytes.NewReader(attachment.Content),
			Size:        attachment.Size,
			Disposition: attachment.Disposition,
			ContentID:   attachment.ContentID,
		})
	}

	// 处理附件ID（从数据库加载已上传的附件）
	if len(req.AttachmentIDs) > 0 {
		if err := s.loadAttachmentsFromIDs(ctx, message, req.AttachmentIDs); err != nil {
			return fmt.Errorf("failed to load attachments from IDs: %w", err)
		}
	}

	// 发送邮件
	if err := smtpClient.SendEmail(ctx, message); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	// 发布邮件发送事件
	if s.eventPublisher != nil {
		sendEvent := sse.NewEmailSendEvent(sse.EventEmailSendCompleted, "", "", userID)
		if sendEvent.Data != nil {
			if sendData, ok := sendEvent.Data.(*sse.EmailSendEventData); ok {
				sendData.Status = "completed"
				sendData.Message = "Email sent successfully"
			}
		}
		if err := s.eventPublisher.PublishToUser(ctx, userID, sendEvent); err != nil {
			log.Printf("Failed to publish email send event: %v", err)
		}
	}

	return nil
}

// DeleteEmail 删除邮件
func (s *EmailServiceImpl) DeleteEmail(ctx context.Context, userID, emailID uint) error {
	// 查找邮件并验证权限，同时预加载账户和文件夹信息
	var email models.Email
	err := s.db.Joins("JOIN email_accounts ON emails.account_id = email_accounts.id").
		Where("emails.id = ? AND email_accounts.user_id = ?", emailID, userID).
		Preload("Account").
		Preload("Folder").
		First(&email).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("email not found")
		}
		return fmt.Errorf("failed to find email: %w", err)
	}

	// 如果已经是删除状态，直接返回
	if email.IsDeleted {
		return nil
	}

	// 先在IMAP服务器上删除邮件
	if email.Folder != nil && email.UID > 0 {
		// 获取邮件提供商
		provider, err := s.providerFactory.CreateProviderForAccount(&email.Account)
		if err != nil {
			log.Printf("Warning: failed to create provider for email deletion: %v", err)
		} else {
			// 连接到IMAP服务器
			if err := provider.Connect(ctx, &email.Account); err != nil {
				log.Printf("Warning: failed to connect to IMAP for email deletion: %v", err)
			} else {
				defer provider.Disconnect()

				// 获取IMAP客户端
				imapClient := provider.IMAPClient()
				if imapClient != nil {
					// 选择文件夹
					if _, err := imapClient.SelectFolder(ctx, email.Folder.Path); err != nil {
						log.Printf("Warning: failed to select folder for email deletion: %v", err)
					} else {
						// 删除邮件
						if err := imapClient.DeleteEmails(ctx, []uint32{email.UID}); err != nil {
							log.Printf("Warning: failed to delete email from IMAP server: %v", err)
						} else {
							log.Printf("Successfully deleted email %d (UID: %d) from IMAP server", emailID, email.UID)
						}
					}
				}
			}
		}
	}

	// 标记为删除（软删除）
	email.IsDeleted = true
	if err := s.db.Save(&email).Error; err != nil {
		return fmt.Errorf("failed to delete email: %w", err)
	}

	// 发布邮件删除事件
	if s.eventPublisher != nil {
		isDeleted := true
		event := sse.NewEmailStatusEvent(emailID, email.AccountID, userID, nil, nil, nil, &isDeleted)
		if err := s.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
			// 记录错误但不影响主要操作
			fmt.Printf("Failed to publish email delete event: %v\n", err)
		}
	}

	return nil
}

// MarkEmailAsRead 标记邮件为已读
func (s *EmailServiceImpl) MarkEmailAsRead(ctx context.Context, userID, emailID uint) error {
	// 查找邮件并验证权限
	var email models.Email
	err := s.db.Joins("JOIN email_accounts ON emails.account_id = email_accounts.id").
		Where("emails.id = ? AND email_accounts.user_id = ?", emailID, userID).
		First(&email).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("email not found")
		}
		return fmt.Errorf("failed to find email: %w", err)
	}

	// 如果已经是已读状态，直接返回
	if email.IsRead {
		return nil
	}

	// 更新邮件状态
	email.MarkAsRead()
	if err := s.db.Save(&email).Error; err != nil {
		return fmt.Errorf("failed to update email status: %w", err)
	}

	// 更新未读计数并清理缓存
	if err := s.updateUnreadCounters(ctx, userID, email.AccountID, email.FolderID); err != nil {
		return err
	}

	// 发布邮件状态变更事件
	if s.eventPublisher != nil {
		isRead := true
		event := sse.NewEmailStatusEvent(emailID, email.AccountID, userID, &isRead, nil, nil, nil)
		if err := s.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
			// 记录错误但不影响主要操作
			fmt.Printf("Failed to publish email read event: %v\n", err)
		}
	}

	return nil
}

// MarkEmailAsUnread 标记邮件为未读
func (s *EmailServiceImpl) MarkEmailAsUnread(ctx context.Context, userID, emailID uint) error {
	// 查找邮件并验证权限
	var email models.Email
	err := s.db.Joins("JOIN email_accounts ON emails.account_id = email_accounts.id").
		Where("emails.id = ? AND email_accounts.user_id = ?", emailID, userID).
		First(&email).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("email not found")
		}
		return fmt.Errorf("failed to find email: %w", err)
	}

	// 如果已经是未读状态，直接返回
	if !email.IsRead {
		return nil
	}

	// 更新邮件状态
	email.MarkAsUnread()
	if err := s.db.Save(&email).Error; err != nil {
		return fmt.Errorf("failed to update email status: %w", err)
	}

	// 更新未读计数并清理缓存
	if err := s.updateUnreadCounters(ctx, userID, email.AccountID, email.FolderID); err != nil {
		return err
	}

	// 发布邮件状态变更事件
	if s.eventPublisher != nil {
		isRead := false
		event := sse.NewEmailStatusEvent(emailID, email.AccountID, userID, &isRead, nil, nil, nil)
		if err := s.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
			// 记录错误但不影响主要操作
			fmt.Printf("Failed to publish email unread event: %v\n", err)
		}
	}

	return nil
}

// ToggleEmailStar 切换邮件星标状态
func (s *EmailServiceImpl) ToggleEmailStar(ctx context.Context, userID, emailID uint) error {
	// 查找邮件并验证权限
	var email models.Email
	err := s.db.Joins("JOIN email_accounts ON emails.account_id = email_accounts.id").
		Where("emails.id = ? AND email_accounts.user_id = ?", emailID, userID).
		First(&email).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("email not found")
		}
		return fmt.Errorf("failed to find email: %w", err)
	}

	// 切换星标状态
	email.ToggleStar()

	if err := s.db.Save(&email).Error; err != nil {
		return fmt.Errorf("failed to update email star status: %w", err)
	}

	// 发布邮件星标变更事件
	if s.eventPublisher != nil {
		isStarred := email.IsStarred
		var event *sse.Event

		if isStarred {
			event = sse.NewEmailStatusEvent(emailID, email.AccountID, userID, nil, &isStarred, nil, nil)
			event.Type = sse.EventEmailStarred
		} else {
			event = sse.NewEmailStatusEvent(emailID, email.AccountID, userID, nil, &isStarred, nil, nil)
			event.Type = sse.EventEmailUnstarred
		}

		if err := s.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
			// 记录错误但不影响主要操作
			fmt.Printf("Failed to publish email star event: %v\n", err)
		}
	}

	return nil
}

// ToggleEmailImportant 切换邮件重要状态
func (s *EmailServiceImpl) ToggleEmailImportant(ctx context.Context, userID, emailID uint) error {
	// 查找邮件并验证权限
	var email models.Email
	err := s.db.Joins("JOIN email_accounts ON emails.account_id = email_accounts.id").
		Where("emails.id = ? AND email_accounts.user_id = ?", emailID, userID).
		First(&email).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("email not found")
		}
		return fmt.Errorf("failed to find email: %w", err)
	}

	// 切换重要状态
	email.ToggleImportant()

	if err := s.db.Save(&email).Error; err != nil {
		return fmt.Errorf("failed to update email important status: %w", err)
	}

	// 发布邮件重要状态变更事件
	if s.eventPublisher != nil {
		isImportant := email.IsImportant
		var event *sse.Event

		if isImportant {
			event = sse.NewEmailStatusEvent(emailID, email.AccountID, userID, nil, nil, &isImportant, nil)
			event.Type = sse.EventEmailImportant
		} else {
			event = sse.NewEmailStatusEvent(emailID, email.AccountID, userID, nil, nil, &isImportant, nil)
			event.Type = sse.EventEmailUnimportant
		}

		if err := s.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
			// 记录错误但不影响主要操作
			fmt.Printf("Failed to publish email important event: %v\n", err)
		}
	}

	return nil
}

// MoveEmail 移动邮件
func (s *EmailServiceImpl) MoveEmail(ctx context.Context, userID, emailID uint, targetFolderID uint) error {
	// 查找邮件并验证权限
	var email models.Email
	err := s.db.Joins("JOIN email_accounts ON emails.account_id = email_accounts.id").
		Where("emails.id = ? AND email_accounts.user_id = ?", emailID, userID).
		First(&email).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("email not found")
		}
		return fmt.Errorf("failed to find email: %w", err)
	}

	// 验证目标文件夹存在且属于同一账户
	var targetFolder models.Folder
	err = s.db.Where("id = ? AND account_id = ?", targetFolderID, email.AccountID).
		First(&targetFolder).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("target folder not found")
		}
		return fmt.Errorf("failed to find target folder: %w", err)
	}

	// 如果已经在目标文件夹，直接返回
	if email.FolderID != nil && *email.FolderID == targetFolderID {
		return nil
	}

	// 获取源文件夹信息（如果存在）
	var sourceFolder *models.Folder
	if email.FolderID != nil {
		var srcFolder models.Folder
		if err := s.db.First(&srcFolder, *email.FolderID).Error; err == nil {
			sourceFolder = &srcFolder
		}
	}

	// 获取邮件账户信息以建立IMAP连接
	var account models.EmailAccount
	if err := s.db.First(&account, email.AccountID).Error; err != nil {
		return fmt.Errorf("failed to get email account: %w", err)
	}

	// 建立IMAP连接
	provider, err := s.providerFactory.CreateProvider(account.Provider)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	if err := provider.Connect(ctx, &account); err != nil {
		return fmt.Errorf("failed to connect to IMAP server: %w", err)
	}
	defer provider.Disconnect()

	imapClient := provider.IMAPClient()
	if imapClient == nil {
		return fmt.Errorf("failed to get IMAP client")
	}

	// 在IMAP服务器上移动邮件
	if email.UID > 0 && sourceFolder != nil {
		// 先选择源文件夹
		if _, err := imapClient.SelectFolder(ctx, sourceFolder.Path); err != nil {
			return fmt.Errorf("failed to select source folder: %w", err)
		}

		// 移动邮件到目标文件夹
		uids := []uint32{uint32(email.UID)}
		if err := imapClient.MoveEmails(ctx, uids, targetFolder.Path); err != nil {
			return fmt.Errorf("failed to move email on server: %w", err)
		}
	}

	// 更新数据库中的邮件文件夹
	email.FolderID = &targetFolderID
	if err := s.db.Save(&email).Error; err != nil {
		return fmt.Errorf("failed to update email folder in database: %w", err)
	}

	// 发布邮件移动通知事件
	if s.eventPublisher != nil {
		event := sse.NewNotificationEvent(
			"邮件已移动",
			fmt.Sprintf("邮件已移动到文件夹: %s", targetFolder.Name),
			"info",
			userID,
		)
		if err := s.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
			// 记录错误但不影响主要操作
			fmt.Printf("Failed to publish email move event: %v\n", err)
		}
	}

	return nil
}

// GetFolders 获取文件夹列表
func (s *EmailServiceImpl) GetFolders(ctx context.Context, userID, accountID uint) ([]*models.Folder, error) {
	// 验证账户存在且属于用户
	_, err := s.GetEmailAccount(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}

	// 从数据库获取文件夹列表
	var folders []*models.Folder
	err = s.db.Where("account_id = ?", accountID).
		Order("type ASC, name ASC").
		Find(&folders).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get folders: %w", err)
	}

	// 如果没有文件夹，尝试同步
	if len(folders) == 0 {
		log.Printf("No folders found for account %d, attempting to sync", accountID)
		if syncErr := s.syncFoldersForAccount(ctx, accountID); syncErr != nil {
			log.Printf("Failed to sync folders for account %d: %v", accountID, syncErr)
			// 即使同步失败，也返回空列表而不是错误
			return []*models.Folder{}, nil
		}

		// 重新查询文件夹
		err = s.db.Where("account_id = ?", accountID).
			Order("type ASC, name ASC").
			Find(&folders).Error

		if err != nil {
			return nil, fmt.Errorf("failed to get folders after sync: %w", err)
		}
	}

	return folders, nil
}

// GetFolder 获取单个文件夹
func (s *EmailServiceImpl) GetFolder(ctx context.Context, userID, folderID uint) (*models.Folder, error) {
	var folder models.Folder

	// 查询文件夹，确保用户只能访问自己的文件夹
	err := s.db.WithContext(ctx).
		Joins("JOIN email_accounts ON folders.account_id = email_accounts.id").
		Where("folders.id = ? AND email_accounts.user_id = ?", folderID, userID).
		Preload("Account").
		First(&folder).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("folder not found")
		}
		return nil, fmt.Errorf("failed to get folder: %w", err)
	}

	return &folder, nil
}

// CreateFolder 创建文件夹
func (s *EmailServiceImpl) CreateFolder(ctx context.Context, userID, accountID uint, req *CreateFolderRequest) (*models.Folder, error) {
	// 验证账户存在且属于用户
	account, err := s.GetEmailAccount(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}

	// 验证请求参数
	if req.Name == "" {
		return nil, fmt.Errorf("folder name is required")
	}

	// 检查文件夹名称是否已存在
	var existingFolder models.Folder
	err = s.db.Where("account_id = ? AND name = ?", accountID, req.Name).
		First(&existingFolder).Error

	if err == nil {
		return nil, fmt.Errorf("folder with name '%s' already exists", req.Name)
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to check folder existence: %w", err)
	}

	// 创建提供商实例并连接
	provider, err := s.providerFactory.CreateProviderForAccount(account)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// 设置OAuth2 token更新回调
	s.setupProviderTokenCallback(provider)

	if err := provider.Connect(ctx, account); err != nil {
		return nil, fmt.Errorf("failed to connect to email server: %w", err)
	}
	defer provider.Disconnect()

	// 获取IMAP客户端
	imapClient := provider.IMAPClient()
	if imapClient == nil {
		return nil, fmt.Errorf("IMAP client not available")
	}

	// 构建文件夹路径
	folderPath := req.Name
	if req.ParentID != nil {
		// 获取父文件夹信息
		var parentFolder models.Folder
		err = s.db.Where("id = ? AND account_id = ?", *req.ParentID, accountID).
			First(&parentFolder).Error

		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, fmt.Errorf("parent folder not found")
			}
			return nil, fmt.Errorf("failed to find parent folder: %w", err)
		}

		// 构建层级路径
		folderPath = parentFolder.Path + parentFolder.Delimiter + req.Name
	}

	// 在IMAP服务器上创建文件夹
	if err := imapClient.CreateFolder(ctx, folderPath); err != nil {
		return nil, fmt.Errorf("failed to create folder on server: %w", err)
	}

	// 创建文件夹模型
	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.Name
	}

	folder := &models.Folder{
		AccountID:    accountID,
		Name:         req.Name,
		DisplayName:  displayName,
		Type:         "custom",
		ParentID:     req.ParentID,
		Path:         folderPath,
		Delimiter:    "/", // 默认分隔符，实际应该从IMAP获取
		IsSelectable: true,
		IsSubscribed: true,
	}

	// 保存到数据库
	if err := s.db.Create(folder).Error; err != nil {
		// 如果数据库保存失败，尝试删除服务器上的文件夹
		imapClient.DeleteFolder(ctx, folderPath)
		return nil, fmt.Errorf("failed to save folder to database: %w", err)
	}

	// 发布文件夹创建事件
	if s.eventPublisher != nil {
		event := sse.NewNotificationEvent(
			"文件夹已创建",
			fmt.Sprintf("文件夹 '%s' 创建成功", folder.DisplayName),
			"success",
			userID,
		)
		if err := s.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
			log.Printf("Failed to publish folder creation event: %v", err)
		}
	}

	return folder, nil
}

// UpdateFolder 更新文件夹
func (s *EmailServiceImpl) UpdateFolder(ctx context.Context, userID, folderID uint, req *UpdateFolderRequest) (*models.Folder, error) {
	// 获取文件夹并验证权限
	folder, err := s.GetFolder(ctx, userID, folderID)
	if err != nil {
		return nil, err
	}

	// 检查是否为系统文件夹（不允许修改）
	if folder.Type != "custom" {
		return nil, fmt.Errorf("cannot modify system folder")
	}

	// 如果没有任何更新，直接返回
	if req.Name == nil && req.DisplayName == nil && req.ParentID == nil {
		return folder, nil
	}

	// 获取账户信息
	var account models.EmailAccount
	if err := s.db.First(&account, folder.AccountID).Error; err != nil {
		return nil, fmt.Errorf("failed to find account: %w", err)
	}

	// 创建提供商实例并连接
	provider, err := s.providerFactory.CreateProviderForAccount(&account)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// 设置OAuth2 token更新回调
	s.setupProviderTokenCallback(provider)

	if err := provider.Connect(ctx, &account); err != nil {
		return nil, fmt.Errorf("failed to connect to email server: %w", err)
	}
	defer provider.Disconnect()

	// 获取IMAP客户端
	imapClient := provider.IMAPClient()
	if imapClient == nil {
		return nil, fmt.Errorf("IMAP client not available")
	}

	oldPath := folder.Path
	newPath := oldPath

	// 处理名称更新
	if req.Name != nil && *req.Name != folder.Name {
		// 检查新名称是否已存在
		var existingFolder models.Folder
		err = s.db.Where("account_id = ? AND name = ? AND id != ?", folder.AccountID, *req.Name, folderID).
			First(&existingFolder).Error

		if err == nil {
			return nil, fmt.Errorf("folder with name '%s' already exists", *req.Name)
		} else if err != gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("failed to check folder name: %w", err)
		}

		// 构建新路径
		if folder.ParentID != nil {
			var parentFolder models.Folder
			if err := s.db.First(&parentFolder, *folder.ParentID).Error; err != nil {
				return nil, fmt.Errorf("failed to find parent folder: %w", err)
			}
			newPath = parentFolder.Path + parentFolder.Delimiter + *req.Name
		} else {
			newPath = *req.Name
		}

		// 在IMAP服务器上重命名文件夹
		if err := imapClient.RenameFolder(ctx, oldPath, newPath); err != nil {
			return nil, fmt.Errorf("failed to rename folder on server: %w", err)
		}

		// 更新模型
		folder.Name = *req.Name
		folder.Path = newPath
	}

	// 更新显示名称
	if req.DisplayName != nil {
		folder.DisplayName = *req.DisplayName
	}

	// TODO: 处理父文件夹更新（移动文件夹）
	// 这需要更复杂的逻辑，包括重新构建路径和移动所有子文件夹
	if req.ParentID != nil {
		return nil, fmt.Errorf("moving folders between parents is not yet supported")
	}

	// 保存更新到数据库
	if err := s.db.Save(folder).Error; err != nil {
		// 如果数据库更新失败且路径已更改，尝试回滚IMAP操作
		if newPath != oldPath {
			imapClient.RenameFolder(ctx, newPath, oldPath)
		}
		return nil, fmt.Errorf("failed to update folder in database: %w", err)
	}

	// 发布文件夹更新事件
	if s.eventPublisher != nil {
		event := sse.NewNotificationEvent(
			"文件夹已更新",
			fmt.Sprintf("文件夹 '%s' 更新成功", folder.DisplayName),
			"success",
			userID,
		)
		if err := s.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
			log.Printf("Failed to publish folder update event: %v", err)
		}
	}

	return folder, nil
}

// DeleteFolder 删除文件夹
func (s *EmailServiceImpl) DeleteFolder(ctx context.Context, userID, folderID uint) error {
	// 获取文件夹并验证权限
	folder, err := s.GetFolder(ctx, userID, folderID)
	if err != nil {
		return err
	}

	// 检查是否为系统文件夹（不允许删除）
	if folder.Type != "custom" {
		return fmt.Errorf("cannot delete system folder")
	}

	// 检查文件夹是否包含邮件
	var emailCount int64
	err = s.db.Model(&models.Email{}).
		Where("folder_id = ?", folderID).
		Count(&emailCount).Error

	if err != nil {
		return fmt.Errorf("failed to check emails in folder: %w", err)
	}

	if emailCount > 0 {
		return fmt.Errorf("cannot delete folder containing %d emails", emailCount)
	}

	// 检查是否有子文件夹
	var childCount int64
	err = s.db.Model(&models.Folder{}).
		Where("parent_id = ?", folderID).
		Count(&childCount).Error

	if err != nil {
		return fmt.Errorf("failed to check child folders: %w", err)
	}

	if childCount > 0 {
		return fmt.Errorf("cannot delete folder containing %d subfolders", childCount)
	}

	// 获取账户信息
	var account models.EmailAccount
	if err := s.db.First(&account, folder.AccountID).Error; err != nil {
		return fmt.Errorf("failed to find account: %w", err)
	}

	// 创建提供商实例并连接
	provider, err := s.providerFactory.CreateProviderForAccount(&account)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	// 设置OAuth2 token更新回调
	s.setupProviderTokenCallback(provider)

	if err := provider.Connect(ctx, &account); err != nil {
		return fmt.Errorf("failed to connect to email server: %w", err)
	}
	defer provider.Disconnect()

	// 获取IMAP客户端
	imapClient := provider.IMAPClient()
	if imapClient == nil {
		return fmt.Errorf("IMAP client not available")
	}

	// 在IMAP服务器上删除文件夹
	if err := imapClient.DeleteFolder(ctx, folder.Path); err != nil {
		return fmt.Errorf("failed to delete folder on server: %w", err)
	}

	// 从数据库中删除文件夹
	if err := s.db.Delete(folder).Error; err != nil {
		// 如果数据库删除失败，文件夹已经从服务器删除，记录错误但不回滚
		log.Printf("Failed to delete folder from database after server deletion: %v", err)
		return fmt.Errorf("folder deleted from server but failed to update database: %w", err)
	}

	// 发布文件夹删除事件
	if s.eventPublisher != nil {
		event := sse.NewNotificationEvent(
			"文件夹已删除",
			fmt.Sprintf("文件夹 '%s' 删除成功", folder.DisplayName),
			"success",
			userID,
		)
		if err := s.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
			log.Printf("Failed to publish folder deletion event: %v", err)
		}
	}

	return nil
}

// MarkFolderAsRead 标记文件夹内所有邮件为已读
func (s *EmailServiceImpl) MarkFolderAsRead(ctx context.Context, userID, folderID uint) error {
	// 获取文件夹并验证权限
	folder, err := s.GetFolder(ctx, userID, folderID)
	if err != nil {
		return err
	}

	// 查找文件夹内所有未读邮件
	var emails []models.Email
	err = s.db.WithContext(ctx).
		Where("folder_id = ? AND is_read = ?", folderID, false).
		Find(&emails).Error

	if err != nil {
		return fmt.Errorf("failed to find unread emails in folder: %w", err)
	}

	// 如果没有未读邮件，直接返回
	if len(emails) == 0 {
		return nil
	}

	// 批量更新邮件为已读状态
	err = s.db.WithContext(ctx).
		Model(&models.Email{}).
		Where("folder_id = ? AND is_read = ?", folderID, false).
		Update("is_read", true).Error

	if err != nil {
		return fmt.Errorf("failed to mark emails as read: %w", err)
	}

	// 更新未读计数并清理缓存
	if err := s.updateUnreadCounters(ctx, userID, folder.AccountID, &folderID); err != nil {
		return err
	}

	// 发布文件夹标记已读事件
	if s.eventPublisher != nil {
		event := sse.NewNotificationEvent(
			"文件夹已标记为已读",
			fmt.Sprintf("文件夹 '%s' 内的 %d 封邮件已标记为已读", folder.DisplayName, len(emails)),
			"success",
			userID,
		)
		if err := s.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
			log.Printf("Failed to publish folder mark as read event: %v", err)
		}
	}

	return nil
}

// MarkAccountAsRead 标记账户下所有邮件为已读
func (s *EmailServiceImpl) MarkAccountAsRead(ctx context.Context, userID, accountID uint) error {
	// 验证账户归属
	account, err := s.GetEmailAccount(ctx, userID, accountID)
	if err != nil {
		return err
	}

	// 批量更新邮件为已读状态
	if err := s.db.WithContext(ctx).
		Model(&models.Email{}).
		Where("account_id = ? AND is_read = ?", accountID, false).
		Update("is_read", true).Error; err != nil {
		return fmt.Errorf("failed to mark account emails as read: %w", err)
	}

	// 将账户下所有文件夹的未读计数重置为 0，避免前端显示残留未读
	if err := s.db.WithContext(ctx).
		Model(&models.Folder{}).
		Where("account_id = ?", accountID).
		Update("unread_emails", 0).Error; err != nil {
		return fmt.Errorf("failed to reset folder unread count: %w", err)
	}

	// 更新未读计数并清理缓存
	if err := s.updateUnreadCounters(ctx, userID, accountID, nil); err != nil {
		return err
	}

	// 发布通知事件（非关键路径，失败不阻断）
	if s.eventPublisher != nil {
		event := sse.NewNotificationEvent(
			"邮箱已标记为已读",
			fmt.Sprintf("账户 %s 的所有邮件已标记为已读", account.Email),
			"success",
			userID,
		)
		if err := s.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
			log.Printf("Failed to publish account mark as read event: %v", err)
		}
	}

	return nil
}

// MarkAccountsAsRead 批量标记多个账户的邮件为已读
func (s *EmailServiceImpl) MarkAccountsAsRead(ctx context.Context, userID uint, accountIDs []uint) error {
	if len(accountIDs) == 0 {
		return fmt.Errorf("accountIDs cannot be empty")
	}

	for _, accountID := range accountIDs {
		if err := s.MarkAccountAsRead(ctx, userID, accountID); err != nil {
			return err
		}
	}

	return nil
}

type parsedSearchQuery struct {
	FreeText  string
	From      string
	To        string
	Subject   string
	Body      string
	HasTokens bool
}

var searchQueryTokenRegexp = regexp.MustCompile(`(?i)\b(from|to|subject|body):`)

// 解析搜索语法：from:xxx subject:xxx body:xxx
func parseSearchQueryTokens(input string) parsedSearchQuery {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return parsedSearchQuery{}
	}

	matches := searchQueryTokenRegexp.FindAllStringSubmatchIndex(trimmed, -1)
	if len(matches) == 0 {
		return parsedSearchQuery{FreeText: trimmed}
	}

	result := parsedSearchQuery{HasTokens: true}
	if matches[0][0] > 0 {
		result.FreeText = strings.TrimSpace(trimmed[:matches[0][0]])
	}

	for i, match := range matches {
		key := strings.ToLower(trimmed[match[2]:match[3]])
		valueStart := match[1]
		valueEnd := len(trimmed)
		if i+1 < len(matches) {
			valueEnd = matches[i+1][0]
		}
		value := strings.TrimSpace(trimmed[valueStart:valueEnd])
		value = strings.Trim(value, "\"'")
		if value == "" {
			continue
		}
		switch key {
		case "from":
			result.From = value
		case "to":
			result.To = value
		case "subject":
			result.Subject = value
		case "body":
			result.Body = value
		}
	}

	return result
}

// SearchEmails 搜索邮件
func (s *EmailServiceImpl) SearchEmails(ctx context.Context, userID uint, req *SearchEmailsRequest) (*GetEmailsResponse, error) {
	// 构建基础查询
	query := s.db.WithContext(ctx).Model(&models.Email{}).
		Joins("JOIN email_accounts ON emails.account_id = email_accounts.id").
		Where("email_accounts.user_id = ?", userID)

	parsedQuery := parseSearchQueryTokens(req.Query)
	if parsedQuery.HasTokens {
		hasParsedValue := parsedQuery.FreeText != "" || parsedQuery.From != "" || parsedQuery.To != "" || parsedQuery.Subject != "" || parsedQuery.Body != ""
		if hasParsedValue {
			if req.From == "" {
				req.From = parsedQuery.From
			}
			if req.To == "" {
				req.To = parsedQuery.To
			}
			if req.Subject == "" {
				req.Subject = parsedQuery.Subject
			}
			if req.Body == "" {
				req.Body = parsedQuery.Body
			}
			req.Query = parsedQuery.FreeText
		}
	}

	// 应用过滤条件
	if req.AccountID != nil {
		query = query.Where("emails.account_id = ?", *req.AccountID)
	}

	if req.FolderID != nil {
		query = query.Where("emails.folder_id = ?", *req.FolderID)
	}

	if req.IsRead != nil {
		query = query.Where("emails.is_read = ?", *req.IsRead)
	}

	if req.IsStarred != nil {
		query = query.Where("emails.is_starred = ?", *req.IsStarred)
	}

	if req.HasAttachment != nil {
		query = query.Where("emails.has_attachment = ?", *req.HasAttachment)
	}

	// 应用搜索条件
	if req.Query != "" {
		searchTerm := "%" + req.Query + "%"
		query = query.Where("(emails.subject LIKE ? OR emails.text_body LIKE ? OR emails.html_body LIKE ? OR emails.from_address LIKE ? OR emails.to_addresses LIKE ?)",
			searchTerm, searchTerm, searchTerm, searchTerm, searchTerm)
	}

	if req.Subject != "" {
		query = query.Where("emails.subject LIKE ?", "%"+req.Subject+"%")
	}

	if req.From != "" {
		query = query.Where("emails.from_address LIKE ?", "%"+req.From+"%")
	}

	if req.To != "" {
		query = query.Where("emails.to_addresses LIKE ?", "%"+req.To+"%")
	}

	if req.Body != "" {
		bodyTerm := "%" + req.Body + "%"
		query = query.Where("(emails.text_body LIKE ? OR emails.html_body LIKE ?)", bodyTerm, bodyTerm)
	}

	// 时间范围过滤
	if req.Since != nil {
		query = query.Where("emails.date >= ?", *req.Since)
	}

	if req.Before != nil {
		query = query.Where("emails.date <= ?", *req.Before)
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count search results: %w", err)
	}

	// 应用分页
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// 获取邮件列表
	var emails []*models.Email
	err := query.Order("emails.date DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&emails).Error

	if err != nil {
		return nil, fmt.Errorf("failed to search emails: %w", err)
	}

	// 计算总页数
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &GetEmailsResponse{
		Emails:     emails,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// generateEmailListCacheKey 生成邮件列表缓存键
func (s *EmailServiceImpl) generateEmailListCacheKey(userID uint, req *GetEmailsRequest) string {
	// 将请求参数序列化为JSON
	reqBytes, _ := json.Marshal(req)

	// 生成MD5哈希
	hash := md5.Sum([]byte(fmt.Sprintf("emails:%d:%s", userID, string(reqBytes))))
	return hex.EncodeToString(hash[:])
}

// updateUnreadCounters 更新账户/文件夹的未读计数并清理相关缓存
func (s *EmailServiceImpl) updateUnreadCounters(ctx context.Context, userID, accountID uint, folderID *uint) error {
	// 账户未读计数
	var accountUnread int64
	if err := s.db.WithContext(ctx).
		Model(&models.Email{}).
		Where("account_id = ? AND is_read = ? AND is_deleted = ?", accountID, false, false).
		Count(&accountUnread).Error; err != nil {
		return fmt.Errorf("failed to count account unread emails: %w", err)
	}

	if err := s.db.WithContext(ctx).
		Model(&models.EmailAccount{}).
		Where("id = ?", accountID).
		Update("unread_emails", accountUnread).Error; err != nil {
		return fmt.Errorf("failed to update account unread count: %w", err)
	}

	// 文件夹未读计数
	if folderID != nil {
		var folderUnread int64
		if err := s.db.WithContext(ctx).
			Model(&models.Email{}).
			Where("folder_id = ? AND is_read = ? AND is_deleted = ?", *folderID, false, false).
			Count(&folderUnread).Error; err != nil {
			return fmt.Errorf("failed to count folder unread emails: %w", err)
		}

		if err := s.db.WithContext(ctx).
			Model(&models.Folder{}).
			Where("id = ?", *folderID).
			Update("unread_emails", folderUnread).Error; err != nil {
			return fmt.Errorf("failed to update folder unread count: %w", err)
		}
	}

	// 清理邮件列表缓存，避免返回陈旧数据
	s.invalidateEmailListCache(userID)

	return nil
}

// invalidateEmailListCache 使邮件列表缓存失效
func (s *EmailServiceImpl) invalidateEmailListCache(userID uint) {
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

// ReplyEmail 回复邮件
func (s *EmailServiceImpl) ReplyEmail(ctx context.Context, userID, emailID uint, req *ReplyEmailRequest) error {
	// 获取原邮件
	originalEmail, err := s.GetEmail(ctx, userID, emailID)
	if err != nil {
		return fmt.Errorf("failed to get original email: %w", err)
	}

	// 验证账户权限
	_, err = s.GetEmailAccount(ctx, userID, req.AccountID)
	if err != nil {
		return fmt.Errorf("invalid account: %w", err)
	}

	// 构建回复邮件
	replySubject := req.Subject
	if replySubject == "" {
		replySubject = "Re: " + originalEmail.Subject
		if !strings.HasPrefix(strings.ToLower(originalEmail.Subject), "re:") {
			replySubject = "Re: " + originalEmail.Subject
		} else {
			replySubject = originalEmail.Subject
		}
	}

	// 设置收件人（如果未指定，则回复给原发件人）
	var toAddresses []*models.EmailAddress
	if len(req.To) > 0 {
		for _, addr := range req.To {
			toAddresses = append(toAddresses, &models.EmailAddress{
				Name:    addr.Name,
				Address: addr.Address,
			})
		}
	} else {
		// 解析原邮件的发件人
		fromAddr := parseEmailAddress(originalEmail.From)
		if fromAddr != nil {
			toAddresses = append(toAddresses, fromAddr)
		}
	}

	// 构建引用内容
	quotedBody := s.buildQuotedContent(originalEmail, req.TextBody, req.HTMLBody)

	// 创建发送请求
	sendReq := &SendEmailRequest{
		AccountID: req.AccountID,
		To:        toAddresses,
		CC:        convertToEmailAddressPointers(req.CC),
		BCC:       convertToEmailAddressPointers(req.BCC),
		Subject:   replySubject,
		TextBody:  quotedBody.TextBody,
		HTMLBody:  quotedBody.HTMLBody,
		ReplyToID: &emailID,
	}

	// 发送邮件
	if err := s.SendEmail(ctx, userID, sendReq); err != nil {
		return fmt.Errorf("failed to send reply: %w", err)
	}

	// 发布回复事件
	if s.eventPublisher != nil {
		event := sse.NewNotificationEvent(
			"邮件已回复",
			fmt.Sprintf("已回复邮件: %s", originalEmail.Subject),
			"success",
			userID,
		)
		if err := s.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
			log.Printf("Failed to publish reply event: %v", err)
		}
	}

	return nil
}

// ReplyAllEmail 回复全部邮件
func (s *EmailServiceImpl) ReplyAllEmail(ctx context.Context, userID, emailID uint, req *ReplyEmailRequest) error {
	// 获取原邮件
	originalEmail, err := s.GetEmail(ctx, userID, emailID)
	if err != nil {
		return fmt.Errorf("failed to get original email: %w", err)
	}

	// 验证账户权限
	account, err := s.GetEmailAccount(ctx, userID, req.AccountID)
	if err != nil {
		return fmt.Errorf("invalid account: %w", err)
	}

	// 构建回复邮件主题
	replySubject := req.Subject
	if replySubject == "" {
		if !strings.HasPrefix(strings.ToLower(originalEmail.Subject), "re:") {
			replySubject = "Re: " + originalEmail.Subject
		} else {
			replySubject = originalEmail.Subject
		}
	}

	// 获取所有收件人（排除自己的邮箱地址）
	var toAddresses []*models.EmailAddress
	var ccAddresses []*models.EmailAddress

	// 添加原发件人到收件人
	fromAddr := parseEmailAddress(originalEmail.From)
	if fromAddr != nil && !isOwnEmailAddress(fromAddr.Address, account.Email) {
		toAddresses = append(toAddresses, fromAddr)
	}

	// 添加原收件人到收件人（排除自己）
	originalToAddresses, _ := parseEmailAddressList(originalEmail.To)
	for _, addr := range originalToAddresses {
		if !isOwnEmailAddress(addr.Address, account.Email) {
			toAddresses = append(toAddresses, addr)
		}
	}

	// 添加原抄送人到抄送（排除自己）
	originalCCAddresses, _ := parseEmailAddressList(originalEmail.CC)
	for _, addr := range originalCCAddresses {
		if !isOwnEmailAddress(addr.Address, account.Email) {
			ccAddresses = append(ccAddresses, addr)
		}
	}

	// 如果用户指定了额外的收件人，添加到列表中
	if len(req.To) > 0 {
		for _, addr := range req.To {
			toAddresses = append(toAddresses, &models.EmailAddress{
				Name:    addr.Name,
				Address: addr.Address,
			})
		}
	}

	// 构建引用内容
	quotedBody := s.buildQuotedContent(originalEmail, req.TextBody, req.HTMLBody)

	// 创建发送请求
	sendReq := &SendEmailRequest{
		AccountID: req.AccountID,
		To:        toAddresses,
		CC:        ccAddresses,
		BCC:       convertToEmailAddressPointers(req.BCC),
		Subject:   replySubject,
		TextBody:  quotedBody.TextBody,
		HTMLBody:  quotedBody.HTMLBody,
		ReplyToID: &emailID,
	}

	// 发送邮件
	if err := s.SendEmail(ctx, userID, sendReq); err != nil {
		return fmt.Errorf("failed to send reply all: %w", err)
	}

	// 发布回复全部事件
	if s.eventPublisher != nil {
		event := sse.NewNotificationEvent(
			"邮件已回复全部",
			fmt.Sprintf("已回复全部: %s", originalEmail.Subject),
			"success",
			userID,
		)
		if err := s.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
			log.Printf("Failed to publish reply all event: %v", err)
		}
	}

	return nil
}

// ForwardEmail 转发邮件
func (s *EmailServiceImpl) ForwardEmail(ctx context.Context, userID, emailID uint, req *ForwardEmailRequest) error {
	// 获取原邮件
	originalEmail, err := s.GetEmail(ctx, userID, emailID)
	if err != nil {
		return fmt.Errorf("failed to get original email: %w", err)
	}

	// 验证账户权限
	_, err = s.GetEmailAccount(ctx, userID, req.AccountID)
	if err != nil {
		return fmt.Errorf("invalid account: %w", err)
	}

	// 构建转发邮件主题
	forwardSubject := req.Subject
	if forwardSubject == "" {
		if !strings.HasPrefix(strings.ToLower(originalEmail.Subject), "fwd:") &&
			!strings.HasPrefix(strings.ToLower(originalEmail.Subject), "fw:") {
			forwardSubject = "Fwd: " + originalEmail.Subject
		} else {
			forwardSubject = originalEmail.Subject
		}
	}

	// 构建转发内容
	forwardedBody := s.buildForwardedContent(originalEmail, req.TextBody, req.HTMLBody)

	// 获取原邮件的附件
	var attachments []*SendEmailAttachment
	if originalEmail.HasAttachment && len(originalEmail.Attachments) > 0 {
		// 转换原邮件的附件为发送格式
		for _, attachment := range originalEmail.Attachments {
			// 读取附件内容
			var content []byte
			if attachment.IsDownloaded && attachment.StoragePath != "" {
				// 如果附件已下载，直接读取文件
				fileData, err := os.ReadFile(attachment.StoragePath)
				if err != nil {
					log.Printf("Failed to read attachment file %s: %v", attachment.StoragePath, err)
					continue // 跳过无法读取的附件
				}
				content = fileData
			} else {
				// 如果附件未下载，尝试通过AttachmentService获取内容
				if s.attachmentService != nil {
					contentReader, err := s.attachmentService.GetAttachmentContent(ctx, attachment.ID, userID)
					if err != nil {
						log.Printf("Failed to get attachment content for ID %d: %v", attachment.ID, err)
						continue // 跳过无法获取的附件
					}
					defer contentReader.Close()

					contentData, err := io.ReadAll(contentReader)
					if err != nil {
						log.Printf("Failed to read attachment content for ID %d: %v", attachment.ID, err)
						continue // 跳过无法读取的附件
					}
					content = contentData
				} else {
					log.Printf("AttachmentService not available, skipping attachment %d", attachment.ID)
					continue
				}
			}

			// 创建SendEmailAttachment
			sendAttachment := &SendEmailAttachment{
				Filename:    attachment.Filename,
				ContentType: attachment.ContentType,
				Content:     content,
				Size:        attachment.Size,
				Disposition: attachment.Disposition,
				ContentID:   attachment.ContentID,
			}

			attachments = append(attachments, sendAttachment)
		}
	}

	// 创建发送请求
	sendReq := &SendEmailRequest{
		AccountID:   req.AccountID,
		To:          convertToEmailAddressPointers(req.To),
		CC:          convertToEmailAddressPointers(req.CC),
		BCC:         convertToEmailAddressPointers(req.BCC),
		Subject:     forwardSubject,
		TextBody:    forwardedBody.TextBody,
		HTMLBody:    forwardedBody.HTMLBody,
		Attachments: attachments,
	}

	// 发送邮件
	if err := s.SendEmail(ctx, userID, sendReq); err != nil {
		return fmt.Errorf("failed to forward email: %w", err)
	}

	// 发布转发事件
	if s.eventPublisher != nil {
		event := sse.NewNotificationEvent(
			"邮件已转发",
			fmt.Sprintf("已转发邮件: %s", originalEmail.Subject),
			"success",
			userID,
		)
		if err := s.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
			log.Printf("Failed to publish forward event: %v", err)
		}
	}

	return nil
}

// ArchiveEmail 归档邮件
func (s *EmailServiceImpl) ArchiveEmail(ctx context.Context, userID, emailID uint) error {
	// 获取邮件
	email, err := s.GetEmail(ctx, userID, emailID)
	if err != nil {
		return fmt.Errorf("failed to get email: %w", err)
	}

	// 查找或创建归档文件夹
	archiveFolder, err := s.findOrCreateArchiveFolder(ctx, email.AccountID)
	if err != nil {
		return fmt.Errorf("failed to get archive folder: %w", err)
	}

	// 移动邮件到归档文件夹
	if err := s.MoveEmail(ctx, userID, emailID, archiveFolder.ID); err != nil {
		return fmt.Errorf("failed to move email to archive: %w", err)
	}

	// 发布归档事件
	if s.eventPublisher != nil {
		event := sse.NewNotificationEvent(
			"邮件已归档",
			fmt.Sprintf("邮件已归档: %s", email.Subject),
			"success",
			userID,
		)
		if err := s.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
			log.Printf("Failed to publish archive event: %v", err)
		}
	}

	return nil
}

// SyncSpecificFolder 同步指定文件夹
func (s *EmailServiceImpl) SyncSpecificFolder(ctx context.Context, userID, folderID uint) error {
	// 获取文件夹信息
	folder, err := s.GetFolder(ctx, userID, folderID)
	if err != nil {
		return fmt.Errorf("failed to get folder: %w", err)
	}

	// 验证账户权限
	account, err := s.GetEmailAccount(ctx, userID, folder.AccountID)
	if err != nil {
		return fmt.Errorf("invalid account: %w", err)
	}

	// 委托给同步服务
	if s.syncService != nil {
		if err := s.syncService.SyncFolder(ctx, account.ID, folder.Name); err != nil {
			return fmt.Errorf("failed to sync folder: %w", err)
		}
	} else {
		return fmt.Errorf("sync service not available")
	}

	// 发布文件夹同步事件
	if s.eventPublisher != nil {
		event := sse.NewNotificationEvent(
			"文件夹已同步",
			fmt.Sprintf("文件夹 '%s' 同步完成", folder.DisplayName),
			"success",
			userID,
		)
		if err := s.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
			log.Printf("Failed to publish folder sync event: %v", err)
		}
	}

	return nil
}

// 辅助函数

// parseEmailAddress 解析邮件地址字符串
func parseEmailAddress(addressStr string) *models.EmailAddress {
	if addressStr == "" {
		return nil
	}

	// 简单的邮件地址解析
	if strings.Contains(addressStr, "<") && strings.Contains(addressStr, ">") {
		// 格式: "Name <email@example.com>"
		parts := strings.Split(addressStr, "<")
		if len(parts) == 2 {
			name := strings.TrimSpace(parts[0])
			email := strings.TrimSpace(strings.Replace(parts[1], ">", "", 1))
			return &models.EmailAddress{
				Name:    name,
				Address: email,
			}
		}
	}

	// 格式: "email@example.com"
	return &models.EmailAddress{
		Name:    "",
		Address: strings.TrimSpace(addressStr),
	}
}

// parseEmailAddressList 解析邮件地址列表
func parseEmailAddressList(addressListStr string) ([]*models.EmailAddress, error) {
	if addressListStr == "" {
		return []*models.EmailAddress{}, nil
	}

	var addresses []*models.EmailAddress
	err := json.Unmarshal([]byte(addressListStr), &addresses)
	return addresses, err
}

// isOwnEmailAddress 检查是否是自己的邮箱地址
func isOwnEmailAddress(address, ownAddress string) bool {
	return strings.EqualFold(strings.TrimSpace(address), strings.TrimSpace(ownAddress))
}

// convertToEmailAddressPointers 转换邮件地址切片为指针切片
func convertToEmailAddressPointers(addresses []models.EmailAddress) []*models.EmailAddress {
	var result []*models.EmailAddress
	for _, addr := range addresses {
		result = append(result, &models.EmailAddress{
			Name:    addr.Name,
			Address: addr.Address,
		})
	}
	return result
}

// QuotedContent 引用内容结构
type QuotedContent struct {
	TextBody string
	HTMLBody string
}

// buildQuotedContent 构建引用内容（用于回复）
func (s *EmailServiceImpl) buildQuotedContent(originalEmail *models.Email, userText, userHTML string) *QuotedContent {
	// 构建文本引用
	textQuote := fmt.Sprintf("\n\n--- Original Message ---\nFrom: %s\nDate: %s\nSubject: %s\n\n%s",
		originalEmail.From,
		originalEmail.Date.Format("2006-01-02 15:04:05"),
		originalEmail.Subject,
		originalEmail.TextBody)

	// 构建HTML引用
	htmlQuote := fmt.Sprintf(`
<br><br>
<div style="border-left: 2px solid #ccc; padding-left: 10px; margin-left: 10px;">
<p><strong>--- Original Message ---</strong></p>
<p><strong>From:</strong> %s</p>
<p><strong>Date:</strong> %s</p>
<p><strong>Subject:</strong> %s</p>
<br>
%s
</div>`,
		html.EscapeString(originalEmail.From),
		originalEmail.Date.Format("2006-01-02 15:04:05"),
		html.EscapeString(originalEmail.Subject),
		originalEmail.HTMLBody)

	return &QuotedContent{
		TextBody: userText + textQuote,
		HTMLBody: userHTML + htmlQuote,
	}
}

// buildForwardedContent 构建转发内容
func (s *EmailServiceImpl) buildForwardedContent(originalEmail *models.Email, userText, userHTML string) *QuotedContent {
	// 构建文本转发内容
	textForward := fmt.Sprintf("%s\n\n--- Forwarded Message ---\nFrom: %s\nTo: %s\nDate: %s\nSubject: %s\n\n%s",
		userText,
		originalEmail.From,
		originalEmail.To,
		originalEmail.Date.Format("2006-01-02 15:04:05"),
		originalEmail.Subject,
		originalEmail.TextBody)

	// 构建HTML转发内容
	htmlForward := fmt.Sprintf(`%s
<br><br>
<div style="border: 1px solid #ccc; padding: 10px; margin: 10px 0;">
<p><strong>--- Forwarded Message ---</strong></p>
<p><strong>From:</strong> %s</p>
<p><strong>To:</strong> %s</p>
<p><strong>Date:</strong> %s</p>
<p><strong>Subject:</strong> %s</p>
<br>
%s
</div>`,
		userHTML,
		html.EscapeString(originalEmail.From),
		html.EscapeString(originalEmail.To),
		originalEmail.Date.Format("2006-01-02 15:04:05"),
		html.EscapeString(originalEmail.Subject),
		originalEmail.HTMLBody)

	return &QuotedContent{
		TextBody: textForward,
		HTMLBody: htmlForward,
	}
}

// findOrCreateArchiveFolder 查找或创建归档文件夹
func (s *EmailServiceImpl) findOrCreateArchiveFolder(ctx context.Context, accountID uint) (*models.Folder, error) {
	// 首先尝试查找现有的归档文件夹
	var archiveFolder models.Folder
	err := s.db.Where("account_id = ? AND (type = ? OR name = ? OR name = ?)",
		accountID, "archive", "Archive", "已归档").First(&archiveFolder).Error

	if err == nil {
		return &archiveFolder, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to query archive folder: %w", err)
	}

	// 获取邮件账户信息以建立IMAP连接
	var account models.EmailAccount
	if err := s.db.First(&account, accountID).Error; err != nil {
		return nil, fmt.Errorf("failed to get email account: %w", err)
	}

	// 建立IMAP连接
	provider, err := s.providerFactory.CreateProvider(account.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	if err := provider.Connect(ctx, &account); err != nil {
		return nil, fmt.Errorf("failed to connect to IMAP server: %w", err)
	}
	defer provider.Disconnect()

	imapClient := provider.IMAPClient()
	if imapClient == nil {
		return nil, fmt.Errorf("failed to get IMAP client")
	}

	// 在IMAP服务器上创建归档文件夹
	folderPath := "Archive"
	if err := imapClient.CreateFolder(ctx, folderPath); err != nil {
		return nil, fmt.Errorf("failed to create archive folder on server: %w", err)
	}

	// 创建数据库记录
	archiveFolder = models.Folder{
		AccountID:    accountID,
		Name:         "Archive",
		DisplayName:  "归档",
		Type:         "archive",
		Path:         folderPath,
		Delimiter:    "/",
		IsSelectable: true,
		IsSubscribed: true,
	}

	if err := s.db.Create(&archiveFolder).Error; err != nil {
		// 如果数据库保存失败，尝试删除服务器上的文件夹
		imapClient.DeleteFolder(ctx, folderPath)
		return nil, fmt.Errorf("failed to create archive folder in database: %w", err)
	}

	return &archiveFolder, nil
}

// loadAttachmentsFromIDs 从数据库加载附件并添加到消息中
func (s *EmailServiceImpl) loadAttachmentsFromIDs(ctx context.Context, message *providers.OutgoingMessage, attachmentIDs []uint) error {
	if len(attachmentIDs) == 0 {
		return nil
	}

	// 从数据库查询附件
	var attachments []models.Attachment
	if err := s.db.WithContext(ctx).Where("id IN ?", attachmentIDs).Find(&attachments).Error; err != nil {
		return fmt.Errorf("failed to query attachments: %w", err)
	}

	// 转换为OutgoingAttachment并添加到消息
	for _, attachment := range attachments {
		// 读取附件文件内容
		var content io.Reader
		if attachment.StoragePath != "" {
			// 处理相对路径：如果不是绝对路径，则基于当前工作目录
			storagePath := attachment.StoragePath
			if !filepath.IsAbs(storagePath) {
				// 获取当前工作目录
				wd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get working directory: %w", err)
				}
				storagePath = filepath.Join(wd, storagePath)
			}

			// 检查文件是否存在
			if _, err := os.Stat(storagePath); os.IsNotExist(err) {
				return fmt.Errorf("attachment file does not exist: %s", storagePath)
			}

			fileData, err := os.ReadFile(storagePath)
			if err != nil {
				return fmt.Errorf("failed to read attachment file %s (resolved from %s): %w", storagePath, attachment.StoragePath, err)
			}

			content = bytes.NewReader(fileData)
		}

		outgoingAttachment := &providers.OutgoingAttachment{
			Filename:    attachment.Filename,
			ContentType: attachment.ContentType,
			Content:     content,
			Size:        attachment.Size,
			Disposition: "attachment",
		}

		message.Attachments = append(message.Attachments, outgoingAttachment)
	}

	return nil
}
