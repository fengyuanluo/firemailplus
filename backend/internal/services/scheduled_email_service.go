package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"firemail/internal/models"
	"gorm.io/gorm"
)

// ScheduledEmailService 定时邮件服务接口
type ScheduledEmailService interface {
	// StartScheduler 启动定时任务调度器
	StartScheduler(ctx context.Context) error
	
	// StopScheduler 停止定时任务调度器
	StopScheduler()
	
	// ProcessScheduledEmails 处理到期的定时邮件
	ProcessScheduledEmails(ctx context.Context) error
}

// ScheduledEmailServiceImpl 定时邮件服务实现
type ScheduledEmailServiceImpl struct {
	db            *gorm.DB
	emailService  EmailService
	emailComposer EmailComposer
	emailSender   EmailSender
	stopChan      chan struct{}
	ticker        *time.Ticker
}

// NewScheduledEmailService 创建定时邮件服务
func NewScheduledEmailService(
	db *gorm.DB,
	emailService EmailService,
	emailComposer EmailComposer,
	emailSender EmailSender,
) ScheduledEmailService {
	return &ScheduledEmailServiceImpl{
		db:            db,
		emailService:  emailService,
		emailComposer: emailComposer,
		emailSender:   emailSender,
		stopChan:      make(chan struct{}),
	}
}

// StartScheduler 启动定时任务调度器
func (s *ScheduledEmailServiceImpl) StartScheduler(ctx context.Context) error {
	log.Println("Starting scheduled email service...")
	
	// 每分钟检查一次
	s.ticker = time.NewTicker(1 * time.Minute)
	
	go func() {
		for {
			select {
			case <-s.ticker.C:
				if err := s.ProcessScheduledEmails(ctx); err != nil {
					log.Printf("Failed to process scheduled emails: %v", err)
				}
			case <-s.stopChan:
				log.Println("Stopping scheduled email service...")
				return
			case <-ctx.Done():
				log.Println("Context cancelled, stopping scheduled email service...")
				return
			}
		}
	}()
	
	return nil
}

// StopScheduler 停止定时任务调度器
func (s *ScheduledEmailServiceImpl) StopScheduler() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.stopChan)
}

// ProcessScheduledEmails 处理到期的定时邮件
func (s *ScheduledEmailServiceImpl) ProcessScheduledEmails(ctx context.Context) error {
	// 查找到期的定时邮件
	var scheduledEmails []models.SendQueue
	now := time.Now()
	
	err := s.db.WithContext(ctx).
		Where("status = ? AND scheduled_at <= ?", "scheduled", now).
		Find(&scheduledEmails).Error
	if err != nil {
		return fmt.Errorf("failed to query scheduled emails: %w", err)
	}
	
	if len(scheduledEmails) == 0 {
		return nil
	}
	
	log.Printf("Processing %d scheduled emails", len(scheduledEmails))
	
	for _, scheduledEmail := range scheduledEmails {
		if err := s.processScheduledEmail(ctx, &scheduledEmail); err != nil {
			log.Printf("Failed to process scheduled email %s: %v", scheduledEmail.SendID, err)
			
			// 更新错误信息和重试次数
			s.updateScheduledEmailError(ctx, &scheduledEmail, err)
		}
	}
	
	return nil
}

// processScheduledEmail 处理单个定时邮件
func (s *ScheduledEmailServiceImpl) processScheduledEmail(ctx context.Context, scheduledEmail *models.SendQueue) error {
	// 更新状态为处理中
	err := s.db.WithContext(ctx).
		Model(scheduledEmail).
		Updates(map[string]interface{}{
			"status":       "processing",
			"last_attempt": time.Now(),
		}).Error
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	
	// 反序列化邮件数据
	var composeRequest ComposeEmailRequest
	if err := json.Unmarshal([]byte(scheduledEmail.EmailData), &composeRequest); err != nil {
		return fmt.Errorf("failed to unmarshal email data: %w", err)
	}
	
	// 组装邮件
	composedEmail, err := s.emailComposer.ComposeEmail(ctx, &composeRequest)
	if err != nil {
		return fmt.Errorf("failed to compose email: %w", err)
	}
	
	// 发送邮件
	_, err = s.emailSender.SendEmail(ctx, composedEmail, scheduledEmail.AccountID)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	
	// 更新状态为已发送
	err = s.db.WithContext(ctx).
		Model(scheduledEmail).
		Updates(map[string]interface{}{
			"status":    "sent",
			"attempts":  gorm.Expr("attempts + 1"),
		}).Error
	if err != nil {
		log.Printf("Failed to update sent status: %v", err)
	}
	
	log.Printf("Successfully sent scheduled email %s", scheduledEmail.SendID)
	return nil
}

// updateScheduledEmailError 更新定时邮件错误信息
func (s *ScheduledEmailServiceImpl) updateScheduledEmailError(ctx context.Context, scheduledEmail *models.SendQueue, sendErr error) {
	scheduledEmail.Attempts++
	scheduledEmail.LastError = sendErr.Error()
	now := time.Now()
	scheduledEmail.LastAttempt = &now
	
	// 如果超过最大重试次数，标记为失败
	if scheduledEmail.Attempts >= scheduledEmail.MaxAttempts {
		scheduledEmail.Status = "failed"
	} else {
		// 计算下次重试时间（指数退避）
		retryDelay := time.Duration(scheduledEmail.Attempts*scheduledEmail.Attempts) * time.Minute
		nextAttempt := now.Add(retryDelay)
		scheduledEmail.NextAttempt = &nextAttempt
		scheduledEmail.Status = "retry"
	}
	
	// 保存更新
	if err := s.db.WithContext(ctx).Save(scheduledEmail).Error; err != nil {
		log.Printf("Failed to update scheduled email error: %v", err)
	}
}
