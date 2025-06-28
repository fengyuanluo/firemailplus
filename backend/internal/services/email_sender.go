package services

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"firemail/internal/models"
	"firemail/internal/providers"
	"firemail/internal/sse"

	"gorm.io/gorm"
)

// EmailSender 邮件发送器接口
type EmailSender interface {
	// SendEmail 发送邮件
	SendEmail(ctx context.Context, email *ComposedEmail, accountID uint) (*SendResult, error)
	
	// SendBulkEmails 批量发送邮件
	SendBulkEmails(ctx context.Context, emails []*ComposedEmail, accountID uint) ([]*SendResult, error)
	
	// GetSendStatus 获取发送状态
	GetSendStatus(ctx context.Context, sendID string) (*SendStatus, error)
	
	// ResendEmail 重新发送邮件
	ResendEmail(ctx context.Context, sendID string) (*SendResult, error)
}

// SendResult 发送结果
type SendResult struct {
	SendID      string    `json:"send_id"`
	EmailID     string    `json:"email_id"`
	Status      string    `json:"status"` // pending, sending, sent, failed
	Message     string    `json:"message,omitempty"`
	SentAt      *time.Time `json:"sent_at,omitempty"`
	Error       string    `json:"error,omitempty"`
	RetryCount  int       `json:"retry_count"`
	Recipients  []string  `json:"recipients"`
}

// SendStatus 发送状态
type SendStatus struct {
	SendID       string                 `json:"send_id"`
	EmailID      string                 `json:"email_id"`
	Status       string                 `json:"status"`
	Progress     float64                `json:"progress"` // 0.0 - 1.0
	TotalRecipients int                 `json:"total_recipients"`
	SentRecipients  int                 `json:"sent_recipients"`
	FailedRecipients int                `json:"failed_recipients"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

// StandardEmailSender 标准邮件发送器
type StandardEmailSender struct {
	db              *gorm.DB
	providerFactory *providers.ProviderFactory
	eventPublisher  sse.EventPublisher
	sendStatus      map[string]*SendStatus
	statusMutex     sync.RWMutex
	config          *EmailSenderConfig
}

// EmailSenderConfig 邮件发送器配置
type EmailSenderConfig struct {
	MaxRetries          int           `json:"max_retries"`           // 最大重试次数
	RetryInterval       time.Duration `json:"retry_interval"`       // 重试间隔
	MaxConcurrentSends  int           `json:"max_concurrent_sends"`  // 最大并发发送数
	SendTimeout         time.Duration `json:"send_timeout"`         // 发送超时
	EnableStatusTracking bool         `json:"enable_status_tracking"` // 启用状态跟踪
	SaveSentEmails      bool          `json:"save_sent_emails"`     // 保存已发送邮件
}

// NewStandardEmailSender 创建标准邮件发送器
func NewStandardEmailSender(db *gorm.DB, providerFactory *providers.ProviderFactory, eventPublisher sse.EventPublisher) EmailSender {
	config := &EmailSenderConfig{
		MaxRetries:          3,
		RetryInterval:       time.Minute * 5,
		MaxConcurrentSends:  10,
		SendTimeout:         time.Minute * 5,
		EnableStatusTracking: true,
		SaveSentEmails:      true,
	}

	return &StandardEmailSender{
		db:              db,
		providerFactory: providerFactory,
		eventPublisher:  eventPublisher,
		sendStatus:      make(map[string]*SendStatus),
		config:          config,
	}
}

// SendEmail 发送邮件
func (s *StandardEmailSender) SendEmail(ctx context.Context, email *ComposedEmail, accountID uint) (*SendResult, error) {
	// 获取邮件账户
	account, err := s.getEmailAccount(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get email account: %w", err)
	}

	// 创建发送结果
	sendID := generateSendID()
	result := &SendResult{
		SendID:     sendID,
		EmailID:    email.ID,
		Status:     "pending",
		Recipients: s.getAllRecipients(email),
	}

	// 创建发送状态
	if s.config.EnableStatusTracking {
		status := &SendStatus{
			SendID:          sendID,
			EmailID:         email.ID,
			Status:          "pending",
			Progress:        0.0,
			TotalRecipients: len(result.Recipients),
			StartTime:       time.Now(),
		}
		s.setSendStatus(sendID, status)
	}

	// 异步发送邮件
	go func() {
		if err := s.sendEmailAsync(ctx, email, account, result); err != nil {
			log.Printf("Failed to send email %s: %v", email.ID, err)
		}
	}()

	return result, nil
}

// SendBulkEmails 批量发送邮件
func (s *StandardEmailSender) SendBulkEmails(ctx context.Context, emails []*ComposedEmail, accountID uint) ([]*SendResult, error) {
	var results []*SendResult
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, s.config.MaxConcurrentSends)

	for _, email := range emails {
		wg.Add(1)
		go func(e *ComposedEmail) {
			defer wg.Done()
			
			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result, err := s.SendEmail(ctx, e, accountID)
			if err != nil {
				log.Printf("Failed to send bulk email %s: %v", e.ID, err)
				result = &SendResult{
					SendID:  generateSendID(),
					EmailID: e.ID,
					Status:  "failed",
					Error:   err.Error(),
				}
			}
			results = append(results, result)
		}(email)
	}

	wg.Wait()
	return results, nil
}

// GetSendStatus 获取发送状态
func (s *StandardEmailSender) GetSendStatus(ctx context.Context, sendID string) (*SendStatus, error) {
	s.statusMutex.RLock()
	defer s.statusMutex.RUnlock()

	if status, exists := s.sendStatus[sendID]; exists {
		return status, nil
	}

	// 如果内存中没有，尝试从数据库加载
	return s.loadSendStatusFromDB(ctx, sendID)
}

// ResendEmail 重新发送邮件
func (s *StandardEmailSender) ResendEmail(ctx context.Context, sendID string) (*SendResult, error) {
	// 获取原始发送记录
	originalStatus, err := s.GetSendStatus(ctx, sendID)
	if err != nil {
		return nil, fmt.Errorf("failed to get original send status: %w", err)
	}

	// 从数据库加载邮件内容和账户信息
	sentEmail, accountID, err := s.loadSentEmailWithAccountFromDB(ctx, originalStatus.EmailID)
	if err != nil {
		return nil, fmt.Errorf("failed to load sent email: %w", err)
	}

	// 重新发送
	return s.SendEmail(ctx, sentEmail, accountID)
}

// sendEmailAsync 异步发送邮件
func (s *StandardEmailSender) sendEmailAsync(ctx context.Context, email *ComposedEmail, account *models.EmailAccount, result *SendResult) error {
	// 更新状态为发送中
	result.Status = "sending"
	if s.config.EnableStatusTracking {
		s.updateSendStatus(result.SendID, func(status *SendStatus) {
			status.Status = "sending"
			status.Progress = 0.1
		})
	}

	// 发布发送开始事件
	if s.eventPublisher != nil {
		event := sse.NewEmailSendEvent("email_send_started", result.SendID, email.ID, account.UserID)
		s.eventPublisher.PublishToUser(ctx, account.UserID, event)
	}

	// 创建提供商实例
	provider, err := s.providerFactory.CreateProviderForAccount(account)
	if err != nil {
		return s.handleSendError(ctx, result, account.UserID, fmt.Errorf("failed to create provider: %w", err))
	}

	// 连接到SMTP服务器
	if err := provider.Connect(ctx, account); err != nil {
		return s.handleSendError(ctx, result, account.UserID, fmt.Errorf("failed to connect to SMTP: %w", err))
	}
	defer provider.Disconnect()

	// 获取SMTP客户端
	smtpClient := provider.SMTPClient()
	if smtpClient == nil {
		return s.handleSendError(ctx, result, account.UserID, fmt.Errorf("SMTP client not available"))
	}

	// 构建发送消息
	outgoingMessage, err := s.buildOutgoingMessage(email)
	if err != nil {
		return s.handleSendError(ctx, result, account.UserID, fmt.Errorf("failed to build outgoing message: %w", err))
	}

	// 发送邮件
	if err := smtpClient.SendEmail(ctx, outgoingMessage); err != nil {
		return s.handleSendError(ctx, result, account.UserID, fmt.Errorf("failed to send email: %w", err))
	}

	// 发送成功
	return s.handleSendSuccess(ctx, result, account, email)
}

// buildOutgoingMessage 构建发送消息
func (s *StandardEmailSender) buildOutgoingMessage(email *ComposedEmail) (*providers.OutgoingMessage, error) {
	message := &providers.OutgoingMessage{
		From:     email.From,
		To:       email.To,
		CC:       email.CC,
		BCC:      email.BCC,
		ReplyTo:  email.ReplyTo,
		Subject:  email.Subject,
		TextBody: email.TextBody,
		HTMLBody: email.HTMLBody,
		Priority: email.Priority,
		Headers:  email.Headers,
	}

	// 转换附件
	for _, attachment := range email.Attachments {
		outgoingAttachment := &providers.OutgoingAttachment{
			Filename:    attachment.Filename,
			ContentType: attachment.ContentType,
			Content:     bytes.NewReader(attachment.Data),
			Size:        attachment.Size,
			Disposition: "attachment",
		}
		message.Attachments = append(message.Attachments, outgoingAttachment)
	}

	// 转换内联附件
	for _, inlineAttachment := range email.InlineAttachments {
		outgoingAttachment := &providers.OutgoingAttachment{
			Filename:    inlineAttachment.Filename,
			ContentType: inlineAttachment.ContentType,
			Content:     bytes.NewReader(inlineAttachment.Data),
			Size:        inlineAttachment.Size,
			Disposition: "inline",
			ContentID:   inlineAttachment.ContentID,
		}
		message.Attachments = append(message.Attachments, outgoingAttachment)
	}

	return message, nil
}

// handleSendSuccess 处理发送成功
func (s *StandardEmailSender) handleSendSuccess(ctx context.Context, result *SendResult, account *models.EmailAccount, email *ComposedEmail) error {
	now := time.Now()
	result.Status = "sent"
	result.SentAt = &now

	// 更新发送状态
	if s.config.EnableStatusTracking {
		s.updateSendStatus(result.SendID, func(status *SendStatus) {
			status.Status = "sent"
			status.Progress = 1.0
			status.SentRecipients = status.TotalRecipients
			status.EndTime = &now
		})
	}

	// 保存已发送邮件
	if s.config.SaveSentEmails {
		if err := s.saveSentEmail(ctx, email, account.ID, result); err != nil {
			log.Printf("Failed to save sent email: %v", err)
		}
	}

	// 发布发送成功事件
	if s.eventPublisher != nil {
		event := sse.NewEmailSendEvent("email_send_completed", result.SendID, email.ID, account.UserID)
		s.eventPublisher.PublishToUser(ctx, account.UserID, event)
	}

	return nil
}

// handleSendError 处理发送错误
func (s *StandardEmailSender) handleSendError(ctx context.Context, result *SendResult, userID uint, err error) error {
	result.Status = "failed"
	result.Error = err.Error()
	result.RetryCount++

	// 更新发送状态
	if s.config.EnableStatusTracking {
		s.updateSendStatus(result.SendID, func(status *SendStatus) {
			status.Status = "failed"
			status.Error = err.Error()
			status.FailedRecipients = status.TotalRecipients
			now := time.Now()
			status.EndTime = &now
		})
	}

	// 发布发送失败事件
	if s.eventPublisher != nil {
		event := sse.NewEmailSendEvent("email_send_failed", result.SendID, result.EmailID, userID)
		if event.Data != nil {
			if sendData, ok := event.Data.(*sse.EmailSendEventData); ok {
				sendData.Error = err.Error()
			}
		}
		s.eventPublisher.PublishToUser(ctx, userID, event)
	}

	return err
}

// 辅助方法

// getEmailAccount 获取邮件账户
func (s *StandardEmailSender) getEmailAccount(ctx context.Context, accountID uint) (*models.EmailAccount, error) {
	var account models.EmailAccount
	if err := s.db.WithContext(ctx).First(&account, accountID).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

// getAllRecipients 获取所有收件人
func (s *StandardEmailSender) getAllRecipients(email *ComposedEmail) []string {
	var recipients []string

	for _, addr := range email.To {
		recipients = append(recipients, addr.Address)
	}
	for _, addr := range email.CC {
		recipients = append(recipients, addr.Address)
	}
	for _, addr := range email.BCC {
		recipients = append(recipients, addr.Address)
	}

	return recipients
}

// setSendStatus 设置发送状态
func (s *StandardEmailSender) setSendStatus(sendID string, status *SendStatus) {
	s.statusMutex.Lock()
	defer s.statusMutex.Unlock()
	s.sendStatus[sendID] = status
}

// updateSendStatus 更新发送状态
func (s *StandardEmailSender) updateSendStatus(sendID string, updateFunc func(*SendStatus)) {
	s.statusMutex.Lock()
	defer s.statusMutex.Unlock()

	if status, exists := s.sendStatus[sendID]; exists {
		updateFunc(status)
	}
}

// loadSendStatusFromDB 从数据库加载发送状态
func (s *StandardEmailSender) loadSendStatusFromDB(ctx context.Context, sendID string) (*SendStatus, error) {
	// 这里应该从数据库加载发送状态
	// 暂时返回未找到错误
	return nil, fmt.Errorf("send status not found: %s", sendID)
}

// loadSentEmailWithAccountFromDB 从数据库加载已发送邮件和账户信息
func (s *StandardEmailSender) loadSentEmailWithAccountFromDB(ctx context.Context, emailID string) (*ComposedEmail, uint, error) {
	// 这里应该从数据库加载已发送邮件和账户信息
	// 暂时返回未实现错误
	return nil, 0, fmt.Errorf("load sent email not implemented")
}

// saveSentEmail 保存已发送邮件
func (s *StandardEmailSender) saveSentEmail(ctx context.Context, email *ComposedEmail, accountID uint, result *SendResult) error {
	// 创建已发送邮件记录
	sentEmail := &models.SentEmail{
		SendID:      result.SendID,
		AccountID:   accountID,
		MessageID:   email.ID,
		Subject:     email.Subject,
		Recipients:  strings.Join(result.Recipients, ","),
		SentAt:      *result.SentAt,
		Status:      result.Status,
		Size:        email.Size,
	}

	return s.db.WithContext(ctx).Create(sentEmail).Error
}

// generateSendID 生成发送ID
func generateSendID() string {
	return fmt.Sprintf("send_%d_%d", time.Now().Unix(), time.Now().Nanosecond())
}
