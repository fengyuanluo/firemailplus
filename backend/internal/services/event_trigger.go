package services

import (
	"context"
	"log"

	"firemail/internal/models"
	"firemail/internal/sse"
)

// EventTrigger 事件触发器接口
type EventTrigger interface {
	// 账户相关事件
	TriggerAccountConnected(ctx context.Context, account *models.EmailAccount, userID uint)
	TriggerAccountDisconnected(ctx context.Context, account *models.EmailAccount, userID uint)
	TriggerAccountError(ctx context.Context, account *models.EmailAccount, userID uint, err error)

	// 同步相关事件
	TriggerSyncStarted(ctx context.Context, account *models.EmailAccount, userID uint)
	TriggerSyncProgress(ctx context.Context, account *models.EmailAccount, userID uint, progress float64, processed, total int, folderName string)
	TriggerSyncCompleted(ctx context.Context, account *models.EmailAccount, userID uint)
	TriggerSyncError(ctx context.Context, account *models.EmailAccount, userID uint, err error)

	// 邮件相关事件
	TriggerNewEmail(ctx context.Context, email *models.Email, userID uint)
	TriggerEmailStatusChanged(ctx context.Context, emailID, accountID, userID uint, isRead, isStarred, isDeleted *bool)

	// 邮件发送事件
	TriggerEmailSendStarted(ctx context.Context, sendID, emailID string, userID uint)
	TriggerEmailSendProgress(ctx context.Context, sendID, emailID string, userID uint, progress float64)
	TriggerEmailSendCompleted(ctx context.Context, sendID, emailID string, userID uint)
	TriggerEmailSendFailed(ctx context.Context, sendID, emailID string, userID uint, err error)

	// 通知事件
	TriggerNotification(ctx context.Context, title, message, notificationType string, userID uint)
}

// StandardEventTrigger 标准事件触发器
type StandardEventTrigger struct {
	eventPublisher sse.EventPublisher
}

// NewEventTrigger 创建事件触发器
func NewEventTrigger(eventPublisher sse.EventPublisher) EventTrigger {
	return &StandardEventTrigger{
		eventPublisher: eventPublisher,
	}
}

// TriggerAccountConnected 触发账户连接事件
func (t *StandardEventTrigger) TriggerAccountConnected(ctx context.Context, account *models.EmailAccount, userID uint) {
	if t.eventPublisher == nil {
		return
	}

	event := sse.NewAccountEvent(sse.EventAccountConnected, account.ID, account.Email, account.Provider, userID)
	if err := t.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
		log.Printf("Failed to publish account connected event: %v", err)
	}
}

// TriggerAccountDisconnected 触发账户断开连接事件
func (t *StandardEventTrigger) TriggerAccountDisconnected(ctx context.Context, account *models.EmailAccount, userID uint) {
	if t.eventPublisher == nil {
		return
	}

	event := sse.NewAccountEvent(sse.EventAccountDisconnected, account.ID, account.Email, account.Provider, userID)
	if err := t.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
		log.Printf("Failed to publish account disconnected event: %v", err)
	}
}

// TriggerAccountError 触发账户错误事件
func (t *StandardEventTrigger) TriggerAccountError(ctx context.Context, account *models.EmailAccount, userID uint, err error) {
	if t.eventPublisher == nil {
		return
	}

	event := sse.NewAccountEvent(sse.EventAccountError, account.ID, account.Email, account.Provider, userID)
	if event.Data != nil {
		if accountData, ok := event.Data.(*sse.AccountEventData); ok {
			accountData.ErrorMessage = err.Error()
		}
	}

	if publishErr := t.eventPublisher.PublishToUser(ctx, userID, event); publishErr != nil {
		log.Printf("Failed to publish account error event: %v", publishErr)
	}
}

// TriggerSyncStarted 触发同步开始事件
func (t *StandardEventTrigger) TriggerSyncStarted(ctx context.Context, account *models.EmailAccount, userID uint) {
	if t.eventPublisher == nil {
		return
	}

	event := sse.NewSyncEvent(sse.EventSyncStarted, account.ID, account.Email, userID)
	if err := t.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
		log.Printf("Failed to publish sync started event: %v", err)
	}
}

// TriggerSyncProgress 触发同步进度事件
func (t *StandardEventTrigger) TriggerSyncProgress(ctx context.Context, account *models.EmailAccount, userID uint, progress float64, processed, total int, folderName string) {
	if t.eventPublisher == nil {
		return
	}

	event := sse.NewSyncEvent(sse.EventSyncProgress, account.ID, account.Email, userID)
	if event.Data != nil {
		if syncData, ok := event.Data.(*sse.SyncEventData); ok {
			syncData.Progress = progress
			syncData.ProcessedEmails = processed
			syncData.TotalEmails = total
			syncData.FolderName = folderName
		}
	}

	if err := t.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
		log.Printf("Failed to publish sync progress event: %v", err)
	}
}

// TriggerSyncCompleted 触发同步完成事件
func (t *StandardEventTrigger) TriggerSyncCompleted(ctx context.Context, account *models.EmailAccount, userID uint) {
	if t.eventPublisher == nil {
		return
	}

	event := sse.NewSyncEvent(sse.EventSyncCompleted, account.ID, account.Email, userID)
	if err := t.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
		log.Printf("Failed to publish sync completed event: %v", err)
	}
}

// TriggerSyncError 触发同步错误事件
func (t *StandardEventTrigger) TriggerSyncError(ctx context.Context, account *models.EmailAccount, userID uint, err error) {
	if t.eventPublisher == nil {
		return
	}

	event := sse.NewSyncEvent(sse.EventSyncError, account.ID, account.Email, userID)
	if event.Data != nil {
		if syncData, ok := event.Data.(*sse.SyncEventData); ok {
			syncData.ErrorMessage = err.Error()
		}
	}

	if publishErr := t.eventPublisher.PublishToUser(ctx, userID, event); publishErr != nil {
		log.Printf("Failed to publish sync error event: %v", publishErr)
	}
}

// TriggerNewEmail 触发新邮件事件
func (t *StandardEventTrigger) TriggerNewEmail(ctx context.Context, email *models.Email, userID uint) {
	if t.eventPublisher == nil {
		return
	}

	event := sse.NewNewEmailEvent(email, userID)
	if err := t.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
		log.Printf("Failed to publish new email event: %v", err)
	}
}

// TriggerEmailStatusChanged 触发邮件状态变更事件
func (t *StandardEventTrigger) TriggerEmailStatusChanged(ctx context.Context, emailID, accountID, userID uint, isRead, isStarred, isDeleted *bool) {
	if t.eventPublisher == nil {
		return
	}

	event := sse.NewEmailStatusEvent(emailID, accountID, userID, isRead, isStarred, isDeleted, nil)
	if err := t.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
		log.Printf("Failed to publish email status event: %v", err)
	}
}

// TriggerEmailSendStarted 触发邮件发送开始事件
func (t *StandardEventTrigger) TriggerEmailSendStarted(ctx context.Context, sendID, emailID string, userID uint) {
	if t.eventPublisher == nil {
		return
	}

	event := sse.NewEmailSendEvent(sse.EventEmailSendStarted, sendID, emailID, userID)
	if err := t.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
		log.Printf("Failed to publish email send started event: %v", err)
	}
}

// TriggerEmailSendProgress 触发邮件发送进度事件
func (t *StandardEventTrigger) TriggerEmailSendProgress(ctx context.Context, sendID, emailID string, userID uint, progress float64) {
	if t.eventPublisher == nil {
		return
	}

	event := sse.NewEmailSendEvent(sse.EventEmailSendProgress, sendID, emailID, userID)
	if event.Data != nil {
		if sendData, ok := event.Data.(*sse.EmailSendEventData); ok {
			sendData.Progress = progress
		}
	}

	if err := t.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
		log.Printf("Failed to publish email send progress event: %v", err)
	}
}

// TriggerEmailSendCompleted 触发邮件发送完成事件
func (t *StandardEventTrigger) TriggerEmailSendCompleted(ctx context.Context, sendID, emailID string, userID uint) {
	if t.eventPublisher == nil {
		return
	}

	event := sse.NewEmailSendEvent(sse.EventEmailSendCompleted, sendID, emailID, userID)
	if err := t.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
		log.Printf("Failed to publish email send completed event: %v", err)
	}
}

// TriggerEmailSendFailed 触发邮件发送失败事件
func (t *StandardEventTrigger) TriggerEmailSendFailed(ctx context.Context, sendID, emailID string, userID uint, err error) {
	if t.eventPublisher == nil {
		return
	}

	event := sse.NewEmailSendEvent(sse.EventEmailSendFailed, sendID, emailID, userID)
	if event.Data != nil {
		if sendData, ok := event.Data.(*sse.EmailSendEventData); ok {
			sendData.Error = err.Error()
		}
	}

	if publishErr := t.eventPublisher.PublishToUser(ctx, userID, event); publishErr != nil {
		log.Printf("Failed to publish email send failed event: %v", publishErr)
	}
}

// TriggerNotification 触发通知事件
func (t *StandardEventTrigger) TriggerNotification(ctx context.Context, title, message, notificationType string, userID uint) {
	if t.eventPublisher == nil {
		return
	}

	event := sse.NewNotificationEvent(title, message, notificationType, userID)
	if err := t.eventPublisher.PublishToUser(ctx, userID, event); err != nil {
		log.Printf("Failed to publish notification event: %v", err)
	}
}
