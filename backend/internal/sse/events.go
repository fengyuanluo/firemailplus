package sse

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"firemail/internal/models"
)

// EventType 事件类型
type EventType string

const (
	// 邮件相关事件
	EventNewEmail                EventType = "new_email"
	EventEmailRead               EventType = "email_read"
	EventEmailUnread             EventType = "email_unread"
	EventEmailDeleted            EventType = "email_deleted"
	EventEmailStarred            EventType = "email_starred"
	EventEmailUnstarred          EventType = "email_unstarred"
	EventEmailImportant          EventType = "email_important"
	EventEmailUnimportant        EventType = "email_unimportant"
	EventEmailMoved              EventType = "email_moved"
	EventFolderReadStateChanged  EventType = "folder_read_state_changed"
	EventAccountReadStateChanged EventType = "account_read_state_changed"

	// 邮件发送事件
	EventEmailSendStarted   EventType = "email_send_started"
	EventEmailSendProgress  EventType = "email_send_progress"
	EventEmailSendCompleted EventType = "email_send_completed"
	EventEmailSendFailed    EventType = "email_send_failed"

	// 同步相关事件
	EventSyncStarted   EventType = "sync_started"
	EventSyncProgress  EventType = "sync_progress"
	EventSyncCompleted EventType = "sync_completed"
	EventSyncError     EventType = "sync_error"

	// 账户相关事件
	EventAccountConnected    EventType = "account_connected"
	EventAccountDisconnected EventType = "account_disconnected"
	EventAccountError        EventType = "account_error"

	// 邮箱分组相关事件
	EventGroupCreated        EventType = "group_created"
	EventGroupUpdated        EventType = "group_updated"
	EventGroupDeleted        EventType = "group_deleted"
	EventGroupReordered      EventType = "group_reordered"
	EventGroupDefaultChanged EventType = "group_default_changed"
	EventAccountGroupChanged EventType = "account_group_changed"

	// 系统事件
	EventHeartbeat    EventType = "heartbeat"
	EventNotification EventType = "notification"
)

// EventPriority 事件优先级
type EventPriority int

const (
	PriorityLow    EventPriority = 1
	PriorityNormal EventPriority = 2
	PriorityHigh   EventPriority = 3
	PriorityUrgent EventPriority = 4
)

// Event SSE事件结构
type Event struct {
	ID        string        `json:"id"`
	Type      EventType     `json:"type"`
	Data      interface{}   `json:"data"`
	UserID    uint          `json:"user_id,omitempty"`
	AccountID *uint         `json:"account_id,omitempty"`
	Priority  EventPriority `json:"priority"`
	Timestamp time.Time     `json:"timestamp"`
	Retry     *int          `json:"retry,omitempty"` // 重试间隔（毫秒）
}

// NewEmailEventData 新邮件事件数据
type NewEmailEventData struct {
	EmailID       uint      `json:"email_id"`
	AccountID     uint      `json:"account_id"`
	FolderID      *uint     `json:"folder_id,omitempty"`
	Subject       string    `json:"subject"`
	From          string    `json:"from"`
	Date          time.Time `json:"date"`
	IsRead        bool      `json:"is_read"`
	HasAttachment bool      `json:"has_attachment"`
	Preview       string    `json:"preview,omitempty"` // 邮件预览文本
}

// EmailStatusEventData 邮件状态变更事件数据
type EmailStatusEventData struct {
	EmailID     uint  `json:"email_id"`
	AccountID   uint  `json:"account_id"`
	FolderID    *uint `json:"folder_id,omitempty"`
	IsRead      *bool `json:"is_read,omitempty"`
	IsStarred   *bool `json:"is_starred,omitempty"`
	IsImportant *bool `json:"is_important,omitempty"`
	IsDeleted   *bool `json:"is_deleted,omitempty"`
	UnreadDelta *int  `json:"unread_delta,omitempty"`
}

// EmailMovedEventData 邮件移动事件数据
type EmailMovedEventData struct {
	EmailID        uint  `json:"email_id"`
	AccountID      uint  `json:"account_id"`
	SourceFolderID *uint `json:"source_folder_id,omitempty"`
	TargetFolderID uint  `json:"target_folder_id"`
	IsRead         bool  `json:"is_read"`
}

// FolderReadStateEventData 文件夹读状态批量变更事件数据
type FolderReadStateEventData struct {
	AccountID     uint `json:"account_id"`
	FolderID      uint `json:"folder_id"`
	AffectedCount int  `json:"affected_count"`
}

// AccountReadStateEventData 账户读状态批量变更事件数据
type AccountReadStateEventData struct {
	AccountID     uint `json:"account_id"`
	AffectedCount int  `json:"affected_count"`
}

// SyncEventData 同步事件数据
type SyncEventData struct {
	AccountID       uint    `json:"account_id"`
	AccountName     string  `json:"account_name"`
	Status          string  `json:"status"`             // started, progress, completed, error
	Progress        float64 `json:"progress,omitempty"` // 0.0-1.0
	TotalEmails     int     `json:"total_emails,omitempty"`
	ProcessedEmails int     `json:"processed_emails,omitempty"`
	FolderName      string  `json:"folder_name,omitempty"`
	ErrorMessage    string  `json:"error_message,omitempty"`
}

// AccountEventData 账户事件数据
type AccountEventData struct {
	AccountID    uint   `json:"account_id"`
	AccountName  string `json:"account_name"`
	Provider     string `json:"provider"`
	Status       string `json:"status"` // connected, disconnected, error
	ErrorMessage string `json:"error_message,omitempty"`
}

// GroupEventData 邮箱分组事件数据
type GroupEventData struct {
	GroupID                uint    `json:"group_id,omitempty"`
	Name                   string  `json:"name,omitempty"`
	SortOrder              int     `json:"sort_order,omitempty"`
	IsDefault              bool    `json:"is_default"`
	SystemKey              *string `json:"system_key,omitempty"`
	GroupIDs               []uint  `json:"group_ids,omitempty"`
	PreviousDefaultGroupID *uint   `json:"previous_default_group_id,omitempty"`
}

// AccountGroupEventData 账户分组变更事件数据
type AccountGroupEventData struct {
	AccountID       uint   `json:"account_id"`
	AccountName     string `json:"account_name"`
	Email           string `json:"email"`
	GroupID         *uint  `json:"group_id,omitempty"`
	PreviousGroupID *uint  `json:"previous_group_id,omitempty"`
}

// NotificationEventData 通知事件数据
type NotificationEventData struct {
	Title    string `json:"title"`
	Message  string `json:"message"`
	Type     string `json:"type"`               // info, success, warning, error
	Duration *int   `json:"duration,omitempty"` // 显示时长（毫秒）
}

// HeartbeatEventData 心跳事件数据
type HeartbeatEventData struct {
	ServerTime time.Time `json:"server_time"`
	ClientID   string    `json:"client_id,omitempty"`
}

// ToSSEFormat 将事件转换为SSE格式
func (e *Event) ToSSEFormat() ([]byte, error) {
	// 构建SSE消息
	var sseMessage []byte

	// 添加事件ID
	if e.ID != "" {
		sseMessage = append(sseMessage, []byte("id: "+e.ID+"\n")...)
	}

	// 添加事件类型
	sseMessage = append(sseMessage, []byte("event: "+string(e.Type)+"\n")...)

	// 添加重试间隔
	if e.Retry != nil {
		sseMessage = append(sseMessage, []byte(fmt.Sprintf("retry: %d\n", *e.Retry))...)
	}

	// 序列化数据
	dataBytes, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	// 添加数据（可能需要多行）
	dataLines := strings.Split(string(dataBytes), "\n")
	for _, line := range dataLines {
		sseMessage = append(sseMessage, []byte("data: "+line+"\n")...)
	}

	// 添加结束标记
	sseMessage = append(sseMessage, []byte("\n")...)

	return sseMessage, nil
}

// NewEvent 创建新事件
func NewEvent(eventType EventType, data interface{}, userID uint) *Event {
	return &Event{
		ID:        generateEventID(),
		Type:      eventType,
		Data:      data,
		UserID:    userID,
		Priority:  PriorityNormal,
		Timestamp: time.Now(),
	}
}

// NewNewEmailEvent 创建新邮件事件
func NewNewEmailEvent(email *models.Email, userID uint) *Event {
	data := &NewEmailEventData{
		EmailID:       email.ID,
		AccountID:     email.AccountID,
		FolderID:      email.FolderID,
		Subject:       email.Subject,
		From:          email.From,
		Date:          email.Date,
		IsRead:        email.IsRead,
		HasAttachment: email.HasAttachment,
		Preview:       truncateText(email.TextBody, 100),
	}

	event := NewEvent(EventNewEmail, data, userID)
	event.AccountID = &email.AccountID
	event.Priority = PriorityHigh

	return event
}

// NewEmailStatusEvent 创建邮件状态变更事件
func NewEmailStatusEvent(emailID, accountID, userID uint, folderID *uint, isRead, isStarred, isImportant, isDeleted *bool, unreadDelta *int) *Event {
	data := &EmailStatusEventData{
		EmailID:     emailID,
		AccountID:   accountID,
		FolderID:    folderID,
		IsRead:      isRead,
		IsStarred:   isStarred,
		IsImportant: isImportant,
		IsDeleted:   isDeleted,
		UnreadDelta: unreadDelta,
	}

	event := NewEvent(EventEmailRead, data, userID)
	if isDeleted != nil && *isDeleted {
		event.Type = EventEmailDeleted
	} else if isRead != nil && *isRead {
		event.Type = EventEmailRead
	} else if isRead != nil && !*isRead {
		event.Type = EventEmailUnread
	} else if isStarred != nil && *isStarred {
		event.Type = EventEmailStarred
	} else if isStarred != nil && !*isStarred {
		event.Type = EventEmailUnstarred
	} else if isImportant != nil && *isImportant {
		event.Type = EventEmailImportant
	} else if isImportant != nil && !*isImportant {
		event.Type = EventEmailUnimportant
	}

	event.AccountID = &accountID

	return event
}

// NewEmailMovedEvent 创建邮件移动事件
func NewEmailMovedEvent(emailID, accountID, userID uint, sourceFolderID *uint, targetFolderID uint, isRead bool) *Event {
	data := &EmailMovedEventData{
		EmailID:        emailID,
		AccountID:      accountID,
		SourceFolderID: sourceFolderID,
		TargetFolderID: targetFolderID,
		IsRead:         isRead,
	}

	event := NewEvent(EventEmailMoved, data, userID)
	event.AccountID = &accountID
	event.Priority = PriorityHigh

	return event
}

// NewFolderReadStateChangedEvent 创建文件夹批量已读事件
func NewFolderReadStateChangedEvent(accountID, folderID, userID uint, affectedCount int) *Event {
	data := &FolderReadStateEventData{
		AccountID:     accountID,
		FolderID:      folderID,
		AffectedCount: affectedCount,
	}

	event := NewEvent(EventFolderReadStateChanged, data, userID)
	event.AccountID = &accountID
	event.Priority = PriorityHigh

	return event
}

// NewAccountReadStateChangedEvent 创建账户批量已读事件
func NewAccountReadStateChangedEvent(accountID, userID uint, affectedCount int) *Event {
	data := &AccountReadStateEventData{
		AccountID:     accountID,
		AffectedCount: affectedCount,
	}

	event := NewEvent(EventAccountReadStateChanged, data, userID)
	event.AccountID = &accountID
	event.Priority = PriorityHigh

	return event
}

// NewSyncEvent 创建同步事件
func NewSyncEvent(eventType EventType, accountID uint, accountName string, userID uint) *Event {
	data := &SyncEventData{
		AccountID:   accountID,
		AccountName: accountName,
		Status:      string(eventType)[5:], // 移除"sync_"前缀
	}

	event := NewEvent(eventType, data, userID)
	event.AccountID = &accountID

	return event
}

// NewNotificationEvent 创建通知事件
func NewNotificationEvent(title, message, notificationType string, userID uint) *Event {
	data := &NotificationEventData{
		Title:   title,
		Message: message,
		Type:    notificationType,
	}

	event := NewEvent(EventNotification, data, userID)
	event.Priority = PriorityHigh

	return event
}

// NewHeartbeatEvent 创建心跳事件
func NewHeartbeatEvent(clientID string) *Event {
	data := &HeartbeatEventData{
		ServerTime: time.Now(),
		ClientID:   clientID,
	}

	event := &Event{
		ID:        generateEventID(),
		Type:      EventHeartbeat,
		Data:      data,
		Priority:  PriorityLow,
		Timestamp: time.Now(),
	}

	return event
}

// 辅助函数
func generateEventID() string {
	return fmt.Sprintf("%d_%d", time.Now().UnixNano(), rand.Intn(1000))
}

func truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	if maxLength <= 3 {
		return "..."[:maxLength]
	}
	return text[:maxLength-3] + "..."
}

// EmailSendEventData 邮件发送事件数据
type EmailSendEventData struct {
	SendID     string   `json:"send_id"`
	EmailID    string   `json:"email_id"`
	Status     string   `json:"status"`
	Message    string   `json:"message,omitempty"`
	Error      string   `json:"error,omitempty"`
	Progress   float64  `json:"progress,omitempty"`
	Recipients []string `json:"recipients,omitempty"`
}

// NewEmailSendEvent 创建邮件发送事件
func NewEmailSendEvent(eventType EventType, sendID, emailID string, userID uint) *Event {
	data := &EmailSendEventData{
		SendID:  sendID,
		EmailID: emailID,
	}

	event := &Event{
		ID:        generateEventID(),
		Type:      eventType,
		UserID:    userID,
		Data:      data,
		Priority:  PriorityNormal,
		Timestamp: time.Now(),
	}

	return event
}

// NewAccountEvent 创建账户事件
func NewAccountEvent(eventType EventType, accountID uint, accountName, provider string, userID uint) *Event {
	data := &AccountEventData{
		AccountID:   accountID,
		AccountName: accountName,
		Provider:    provider,
	}

	// 根据事件类型设置状态
	switch eventType {
	case EventAccountConnected:
		data.Status = "connected"
	case EventAccountDisconnected:
		data.Status = "disconnected"
	case EventAccountError:
		data.Status = "error"
	}

	event := &Event{
		ID:        generateEventID(),
		Type:      eventType,
		UserID:    userID,
		Data:      data,
		Priority:  PriorityNormal,
		Timestamp: time.Now(),
	}

	return event
}

// NewGroupEvent 创建邮箱分组事件
func NewGroupEvent(eventType EventType, group *models.EmailGroup, userID uint) *Event {
	data := &GroupEventData{}
	if group != nil {
		data.GroupID = group.ID
		data.Name = group.Name
		data.SortOrder = group.SortOrder
		data.IsDefault = group.IsDefault
		if group.SystemKey != nil {
			systemKey := *group.SystemKey
			data.SystemKey = &systemKey
		}
	}

	return &Event{
		ID:        generateEventID(),
		Type:      eventType,
		UserID:    userID,
		Data:      data,
		Priority:  PriorityNormal,
		Timestamp: time.Now(),
	}
}

// NewGroupReorderedEvent 创建邮箱分组排序变更事件
func NewGroupReorderedEvent(groupIDs []uint, userID uint) *Event {
	data := &GroupEventData{
		GroupIDs: append([]uint(nil), groupIDs...),
	}

	return &Event{
		ID:        generateEventID(),
		Type:      EventGroupReordered,
		UserID:    userID,
		Data:      data,
		Priority:  PriorityNormal,
		Timestamp: time.Now(),
	}
}

// NewDefaultGroupChangedEvent 创建默认分组切换事件
func NewDefaultGroupChangedEvent(group *models.EmailGroup, previousDefaultGroupID *uint, userID uint) *Event {
	event := NewGroupEvent(EventGroupDefaultChanged, group, userID)
	if data, ok := event.Data.(*GroupEventData); ok && previousDefaultGroupID != nil {
		prevID := *previousDefaultGroupID
		data.PreviousDefaultGroupID = &prevID
	}
	return event
}

// NewAccountGroupEvent 创建账户分组变更事件
func NewAccountGroupEvent(account *models.EmailAccount, previousGroupID *uint, userID uint) *Event {
	data := &AccountGroupEventData{
		PreviousGroupID: previousGroupID,
	}
	if account != nil {
		data.AccountID = account.ID
		data.AccountName = account.Name
		data.Email = account.Email
		if account.GroupID != nil {
			groupID := *account.GroupID
			data.GroupID = &groupID
		}
	}

	event := &Event{
		ID:        generateEventID(),
		Type:      EventAccountGroupChanged,
		UserID:    userID,
		Data:      data,
		Priority:  PriorityNormal,
		Timestamp: time.Now(),
	}
	if account != nil {
		event.AccountID = &account.ID
	}

	return event
}
