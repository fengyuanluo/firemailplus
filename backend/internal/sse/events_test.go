package sse

import (
	"testing"
	"time"

	"firemail/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventTypes(t *testing.T) {
	tests := []struct {
		name      string
		eventType EventType
		expected  string
	}{
		{"新邮件事件", EventNewEmail, "new_email"},
		{"邮件已读事件", EventEmailRead, "email_read"},
		{"邮件未读事件", EventEmailUnread, "email_unread"},
		{"邮件删除事件", EventEmailDeleted, "email_deleted"},
		{"邮件星标事件", EventEmailStarred, "email_starred"},
		{"邮件取消星标事件", EventEmailUnstarred, "email_unstarred"},
		{"同步开始事件", EventSyncStarted, "sync_started"},
		{"同步完成事件", EventSyncCompleted, "sync_completed"},
		{"同步错误事件", EventSyncError, "sync_error"},
		{"通知事件", EventNotification, "notification"},
		{"心跳事件", EventHeartbeat, "heartbeat"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.eventType))
		})
	}
}

func TestEventPriority(t *testing.T) {
	assert.Equal(t, 1, int(PriorityLow))
	assert.Equal(t, 2, int(PriorityNormal))
	assert.Equal(t, 3, int(PriorityHigh))
	assert.Equal(t, 4, int(PriorityUrgent))
}

func TestNewEvent(t *testing.T) {
	userID := uint(123)
	data := map[string]interface{}{"test": "data"}
	
	event := NewEvent(EventNewEmail, data, userID)
	
	assert.NotEmpty(t, event.ID)
	assert.Equal(t, EventNewEmail, event.Type)
	assert.Equal(t, data, event.Data)
	assert.Equal(t, userID, event.UserID)
	assert.Equal(t, PriorityNormal, event.Priority)
	assert.WithinDuration(t, time.Now(), event.Timestamp, time.Second)
}

func TestNewNewEmailEvent(t *testing.T) {
	email := &models.Email{
		BaseModel: models.BaseModel{ID: 1},
		AccountID: 2,
		FolderID:  func() *uint { id := uint(3); return &id }(),
		Subject:   "Test Subject",
		From:      "test@example.com",
		Date:      time.Now(),
		IsRead:    false,
		HasAttachment: true,
		TextBody:  "This is a test email body with more than 100 characters to test the preview truncation functionality.",
	}
	userID := uint(123)
	
	event := NewNewEmailEvent(email, userID)
	
	assert.Equal(t, EventNewEmail, event.Type)
	assert.Equal(t, userID, event.UserID)
	assert.Equal(t, email.AccountID, *event.AccountID)
	assert.Equal(t, PriorityHigh, event.Priority)
	
	// 验证事件数据
	data, ok := event.Data.(*NewEmailEventData)
	require.True(t, ok)
	assert.Equal(t, email.ID, data.EmailID)
	assert.Equal(t, email.AccountID, data.AccountID)
	assert.Equal(t, email.FolderID, data.FolderID)
	assert.Equal(t, email.Subject, data.Subject)
	assert.Equal(t, email.From, data.From)
	assert.Equal(t, email.Date, data.Date)
	assert.Equal(t, email.IsRead, data.IsRead)
	assert.Equal(t, email.HasAttachment, data.HasAttachment)
	assert.Equal(t, "This is a test email body with more than 100 characters to test the preview truncation functional...", data.Preview)
}

func TestNewEmailStatusEvent(t *testing.T) {
	emailID := uint(1)
	accountID := uint(2)
	userID := uint(123)
	
	tests := []struct {
		name         string
		isRead       *bool
		isStarred    *bool
		isDeleted    *bool
		expectedType EventType
	}{
		{
			name:         "标记为已读",
			isRead:       func() *bool { b := true; return &b }(),
			expectedType: EventEmailRead,
		},
		{
			name:         "标记为未读",
			isRead:       func() *bool { b := false; return &b }(),
			expectedType: EventEmailUnread,
		},
		{
			name:         "添加星标",
			isStarred:    func() *bool { b := true; return &b }(),
			expectedType: EventEmailStarred,
		},
		{
			name:         "移除星标",
			isStarred:    func() *bool { b := false; return &b }(),
			expectedType: EventEmailUnstarred,
		},
		{
			name:         "删除邮件",
			isDeleted:    func() *bool { b := true; return &b }(),
			expectedType: EventEmailDeleted,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := NewEmailStatusEvent(emailID, accountID, userID, tt.isRead, tt.isStarred, tt.isDeleted)
			
			assert.Equal(t, tt.expectedType, event.Type)
			assert.Equal(t, userID, event.UserID)
			assert.Equal(t, accountID, *event.AccountID)
			
			// 验证事件数据
			data, ok := event.Data.(*EmailStatusEventData)
			require.True(t, ok)
			assert.Equal(t, emailID, data.EmailID)
			assert.Equal(t, accountID, data.AccountID)
			assert.Equal(t, tt.isRead, data.IsRead)
			assert.Equal(t, tt.isStarred, data.IsStarred)
			assert.Equal(t, tt.isDeleted, data.IsDeleted)
		})
	}
}

func TestNewSyncEvent(t *testing.T) {
	accountID := uint(1)
	accountName := "Test Account"
	userID := uint(123)
	
	tests := []struct {
		name         string
		eventType    EventType
		expectedStatus string
	}{
		{"同步开始", EventSyncStarted, "started"},
		{"同步完成", EventSyncCompleted, "completed"},
		{"同步错误", EventSyncError, "error"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := NewSyncEvent(tt.eventType, accountID, accountName, userID)
			
			assert.Equal(t, tt.eventType, event.Type)
			assert.Equal(t, userID, event.UserID)
			assert.Equal(t, accountID, *event.AccountID)
			
			// 验证事件数据
			data, ok := event.Data.(*SyncEventData)
			require.True(t, ok)
			assert.Equal(t, accountID, data.AccountID)
			assert.Equal(t, accountName, data.AccountName)
			assert.Equal(t, tt.expectedStatus, data.Status)
		})
	}
}

func TestNewNotificationEvent(t *testing.T) {
	title := "Test Title"
	message := "Test Message"
	notificationType := "info"
	userID := uint(123)
	
	event := NewNotificationEvent(title, message, notificationType, userID)
	
	assert.Equal(t, EventNotification, event.Type)
	assert.Equal(t, userID, event.UserID)
	assert.Equal(t, PriorityHigh, event.Priority)
	
	// 验证事件数据
	data, ok := event.Data.(*NotificationEventData)
	require.True(t, ok)
	assert.Equal(t, title, data.Title)
	assert.Equal(t, message, data.Message)
	assert.Equal(t, notificationType, data.Type)
}

func TestNewHeartbeatEvent(t *testing.T) {
	clientID := "test-client-123"
	
	event := NewHeartbeatEvent(clientID)
	
	assert.Equal(t, EventHeartbeat, event.Type)
	assert.Equal(t, uint(0), event.UserID) // 心跳事件不绑定特定用户
	assert.Equal(t, PriorityLow, event.Priority)
	assert.NotEmpty(t, event.ID)
	
	// 验证事件数据
	data, ok := event.Data.(*HeartbeatEventData)
	require.True(t, ok)
	assert.Equal(t, clientID, data.ClientID)
	assert.WithinDuration(t, time.Now(), data.ServerTime, time.Second)
}

func TestEventToSSEFormat(t *testing.T) {
	event := &Event{
		ID:        "test-123",
		Type:      EventNewEmail,
		Data:      map[string]interface{}{"test": "data"},
		UserID:    123,
		Priority:  PriorityNormal,
		Timestamp: time.Now(),
		Retry:     func() *int { r := 3000; return &r }(),
	}
	
	sseData, err := event.ToSSEFormat()
	require.NoError(t, err)
	
	sseString := string(sseData)
	
	// 验证SSE格式
	assert.Contains(t, sseString, "id: test-123")
	assert.Contains(t, sseString, "event: new_email")
	assert.Contains(t, sseString, "retry: 3000")
	assert.Contains(t, sseString, "data: ")
	assert.Contains(t, sseString, `"test":"data"`)
	assert.Contains(t, sseString, `"user_id":123`)
	
	// 验证结束标记
	assert.Contains(t, sseString, "\n\n")
}

func TestEventToSSEFormatWithoutRetry(t *testing.T) {
	event := &Event{
		ID:        "test-456",
		Type:      EventHeartbeat,
		Data:      map[string]interface{}{"ping": "pong"},
		UserID:    0,
		Priority:  PriorityLow,
		Timestamp: time.Now(),
	}
	
	sseData, err := event.ToSSEFormat()
	require.NoError(t, err)
	
	sseString := string(sseData)
	
	// 验证SSE格式
	assert.Contains(t, sseString, "id: test-456")
	assert.Contains(t, sseString, "event: heartbeat")
	assert.NotContains(t, sseString, "retry:")
	assert.Contains(t, sseString, "data: ")
}

func TestEventToSSEFormatInvalidData(t *testing.T) {
	event := &Event{
		ID:   "test-invalid",
		Type: EventNewEmail,
		Data: make(chan int), // 无法序列化的数据类型
	}
	
	_, err := event.ToSSEFormat()
	assert.Error(t, err)
}

func TestGenerateEventID(t *testing.T) {
	id1 := generateEventID()
	id2 := generateEventID()
	
	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2) // 应该生成不同的ID
}

func TestTruncateText(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLength int
		expected  string
	}{
		{
			name:      "短文本不截断",
			input:     "Short text",
			maxLength: 20,
			expected:  "Short text",
		},
		{
			name:      "长文本截断",
			input:     "This is a very long text that should be truncated",
			maxLength: 20,
			expected:  "This is a very lo...",
		},
		{
			name:      "空文本",
			input:     "",
			maxLength: 10,
			expected:  "",
		},
		{
			name:      "正好等于最大长度",
			input:     "Exactly twenty chars",
			maxLength: 20,
			expected:  "Exactly twenty chars",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateText(tt.input, tt.maxLength)
			assert.Equal(t, tt.expected, result)
		})
	}
}
