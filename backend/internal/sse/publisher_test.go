package sse

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MockConnectionManager 模拟连接管理器
type MockConnectionManager struct {
	connections map[uint][]ClientConnection
	sentData    map[uint][][]byte
}

func NewMockConnectionManager() *MockConnectionManager {
	return &MockConnectionManager{
		connections: make(map[uint][]ClientConnection),
		sentData:    make(map[uint][][]byte),
	}
}

func (m *MockConnectionManager) AddConnection(userID uint, clientID string, conn ClientConnection) error {
	if m.connections[userID] == nil {
		m.connections[userID] = make([]ClientConnection, 0)
	}
	m.connections[userID] = append(m.connections[userID], conn)
	return nil
}

func (m *MockConnectionManager) RemoveConnection(userID uint, clientID string) error {
	connections := m.connections[userID]
	for i, conn := range connections {
		if conn.GetClientID() == clientID {
			m.connections[userID] = append(connections[:i], connections[i+1:]...)
			break
		}
	}
	return nil
}

func (m *MockConnectionManager) GetConnections(userID uint) []ClientConnection {
	return m.connections[userID]
}

func (m *MockConnectionManager) GetConnectionCount() (total int, byUser map[uint]int) {
	byUser = make(map[uint]int)
	for userID, connections := range m.connections {
		count := 0
		for _, conn := range connections {
			if conn.IsActive() {
				count++
			}
		}
		if count > 0 {
			byUser[userID] = count
			total += count
		}
	}
	return total, byUser
}

func (m *MockConnectionManager) SendToUser(userID uint, data []byte) error {
	if m.sentData[userID] == nil {
		m.sentData[userID] = make([][]byte, 0)
	}
	m.sentData[userID] = append(m.sentData[userID], data)
	
	// 模拟发送给所有连接
	for _, conn := range m.connections[userID] {
		if conn.IsActive() {
			conn.Send(data)
		}
	}
	return nil
}

func (m *MockConnectionManager) SendToConnection(userID uint, clientID string, data []byte) error {
	for _, conn := range m.connections[userID] {
		if conn.GetClientID() == clientID && conn.IsActive() {
			return conn.Send(data)
		}
	}
	return fmt.Errorf("connection not found")
}

func (m *MockConnectionManager) CleanupInactiveConnections() error {
	return nil
}

func (m *MockConnectionManager) GetSentData(userID uint) [][]byte {
	return m.sentData[userID]
}

func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	
	// 创建测试表
	db.Exec(`CREATE TABLE email_accounts (
		id INTEGER PRIMARY KEY,
		user_id INTEGER NOT NULL
	)`)
	
	// 插入测试数据
	db.Exec("INSERT INTO email_accounts (id, user_id) VALUES (1, 123)")
	db.Exec("INSERT INTO email_accounts (id, user_id) VALUES (2, 456)")
	
	return db
}

func TestEventPublisher(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database tests in short mode")
	}
	t.Run("创建事件发布器", func(t *testing.T) {
		cm := NewMockConnectionManager()
		db := setupTestDB()
		
		publisher := NewEventPublisher(cm, db)
		assert.NotNil(t, publisher)
		
		stats := publisher.GetStats()
		assert.Equal(t, int64(0), stats.EventsPublished)
		assert.Equal(t, int64(0), stats.FailedEvents)
		assert.WithinDuration(t, time.Now(), stats.StartTime, time.Second)
	})

	t.Run("发布事件给指定用户", func(t *testing.T) {
		cm := NewMockConnectionManager()
		db := setupTestDB()
		publisher := NewEventPublisher(cm, db)
		
		userID := uint(123)
		event := NewNotificationEvent("Test", "Test message", "info", userID)
		
		err := publisher.Publish(context.Background(), event)
		assert.NoError(t, err)
		
		// 验证事件已发送
		sentData := cm.GetSentData(userID)
		assert.Len(t, sentData, 1)
		
		// 验证统计信息
		stats := publisher.GetStats()
		assert.Equal(t, int64(1), stats.EventsPublished)
		assert.Equal(t, int64(1), stats.EventsByType[EventNotification])
		assert.Equal(t, int64(1), stats.EventsByUser[userID])
	})

	t.Run("发布事件给账户用户", func(t *testing.T) {
		cm := NewMockConnectionManager()
		db := setupTestDB()
		publisher := NewEventPublisher(cm, db)
		
		accountID := uint(1)
		event := NewNotificationEvent("Account Test", "Account message", "info", 0)
		
		err := publisher.PublishToAccount(context.Background(), accountID, event)
		assert.NoError(t, err)
		
		// 验证事件发送给了正确的用户
		sentData := cm.GetSentData(123) // accountID 1 对应 userID 123
		assert.Len(t, sentData, 1)
		
		// 验证事件包含账户ID
		assert.Equal(t, accountID, *event.AccountID)
		assert.Equal(t, uint(123), event.UserID)
	})

	t.Run("发布事件给不存在的账户", func(t *testing.T) {
		cm := NewMockConnectionManager()
		db := setupTestDB()
		publisher := NewEventPublisher(cm, db)
		
		event := NewNotificationEvent("Test", "Test message", "info", 0)
		
		err := publisher.PublishToAccount(context.Background(), 999, event)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "account not found")
	})

	t.Run("广播事件", func(t *testing.T) {
		cm := NewMockConnectionManager()
		db := setupTestDB()
		publisher := NewEventPublisher(cm, db)
		
		// 添加多个用户的连接
		conn1 := NewMockClientConnection("client1", 123)
		conn2 := NewMockClientConnection("client2", 456)
		cm.AddConnection(123, "client1", conn1)
		cm.AddConnection(456, "client2", conn2)
		
		event := NewNotificationEvent("Broadcast", "Broadcast message", "info", 0)
		
		err := publisher.Broadcast(context.Background(), event)
		assert.NoError(t, err)
		
		// 验证所有用户都收到了消息
		assert.Len(t, cm.GetSentData(123), 1)
		assert.Len(t, cm.GetSentData(456), 1)
	})

	t.Run("发布空事件", func(t *testing.T) {
		cm := NewMockConnectionManager()
		db := setupTestDB()
		publisher := NewEventPublisher(cm, db)
		
		err := publisher.Publish(context.Background(), nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "event cannot be nil")
	})

	t.Run("事件过滤器", func(t *testing.T) {
		cm := NewMockConnectionManager()
		db := setupTestDB()
		publisher := NewEventPublisher(cm, db)
		
		// 添加用户过滤器，只允许用户123
		userFilter := NewUserEventFilter([]uint{123})
		publisher.AddEventFilter(userFilter)
		
		// 发布给允许的用户
		event1 := NewNotificationEvent("Allowed", "Allowed message", "info", 123)
		err := publisher.Publish(context.Background(), event1)
		assert.NoError(t, err)
		
		// 发布给不允许的用户
		event2 := NewNotificationEvent("Blocked", "Blocked message", "info", 456)
		err = publisher.Publish(context.Background(), event2)
		assert.NoError(t, err) // 不会报错，但事件被过滤
		
		// 验证统计信息
		stats := publisher.GetStats()
		assert.Equal(t, int64(1), stats.EventsPublished) // 只有一个事件被发布
		assert.Equal(t, int64(1), stats.EventsByUser[123])
		assert.Equal(t, int64(0), stats.EventsByUser[456])
	})

	t.Run("事件类型过滤器", func(t *testing.T) {
		cm := NewMockConnectionManager()
		db := setupTestDB()
		publisher := NewEventPublisher(cm, db)
		
		// 添加事件类型过滤器，只允许通知事件
		typeFilter := NewEventTypeFilter([]EventType{EventNotification})
		publisher.AddEventFilter(typeFilter)
		
		userID := uint(123)
		
		// 发布允许的事件类型
		event1 := NewNotificationEvent("Allowed", "Allowed message", "info", userID)
		err := publisher.Publish(context.Background(), event1)
		assert.NoError(t, err)
		
		// 发布不允许的事件类型
		event2 := NewHeartbeatEvent("client")
		event2.UserID = userID
		err = publisher.Publish(context.Background(), event2)
		assert.NoError(t, err) // 不会报错，但事件被过滤
		
		// 验证统计信息
		stats := publisher.GetStats()
		assert.Equal(t, int64(1), stats.EventsPublished) // 只有一个事件被发布
		assert.Equal(t, int64(1), stats.EventsByType[EventNotification])
		assert.Equal(t, int64(0), stats.EventsByType[EventHeartbeat])
	})
}

func TestUserEventFilter(t *testing.T) {
	t.Run("创建用户过滤器", func(t *testing.T) {
		allowedUsers := []uint{123, 456}
		filter := NewUserEventFilter(allowedUsers)
		assert.NotNil(t, filter)
		
		// 测试允许的用户
		event := NewNotificationEvent("Test", "Test", "info", 123)
		assert.True(t, filter.ShouldProcess(event, 123))
		assert.True(t, filter.ShouldProcess(event, 456))
		
		// 测试不允许的用户
		assert.False(t, filter.ShouldProcess(event, 789))
	})

	t.Run("空过滤器允许所有用户", func(t *testing.T) {
		filter := NewUserEventFilter([]uint{})
		event := NewNotificationEvent("Test", "Test", "info", 123)
		
		assert.True(t, filter.ShouldProcess(event, 123))
		assert.True(t, filter.ShouldProcess(event, 456))
		assert.True(t, filter.ShouldProcess(event, 789))
	})

	t.Run("添加和移除用户", func(t *testing.T) {
		filter := NewUserEventFilter([]uint{123})
		event := NewNotificationEvent("Test", "Test", "info", 456)
		
		// 初始不允许
		assert.False(t, filter.ShouldProcess(event, 456))
		
		// 添加用户
		filter.AddUser(456)
		assert.True(t, filter.ShouldProcess(event, 456))
		
		// 移除用户
		filter.RemoveUser(456)
		assert.False(t, filter.ShouldProcess(event, 456))
	})
}

func TestEventTypeFilter(t *testing.T) {
	t.Run("创建事件类型过滤器", func(t *testing.T) {
		allowedTypes := []EventType{EventNotification, EventNewEmail}
		filter := NewEventTypeFilter(allowedTypes)
		assert.NotNil(t, filter)
		
		// 测试允许的事件类型
		event1 := NewNotificationEvent("Test", "Test", "info", 123)
		assert.True(t, filter.ShouldProcess(event1, 123))
		
		event2 := NewEvent(EventNewEmail, nil, 123)
		assert.True(t, filter.ShouldProcess(event2, 123))
		
		// 测试不允许的事件类型
		event3 := NewHeartbeatEvent("client")
		assert.False(t, filter.ShouldProcess(event3, 123))
	})

	t.Run("空过滤器允许所有事件类型", func(t *testing.T) {
		filter := NewEventTypeFilter([]EventType{})
		
		event1 := NewNotificationEvent("Test", "Test", "info", 123)
		assert.True(t, filter.ShouldProcess(event1, 123))
		
		event2 := NewHeartbeatEvent("client")
		assert.True(t, filter.ShouldProcess(event2, 123))
	})

	t.Run("添加和移除事件类型", func(t *testing.T) {
		filter := NewEventTypeFilter([]EventType{EventNotification})
		
		// 初始不允许心跳事件
		event := NewHeartbeatEvent("client")
		assert.False(t, filter.ShouldProcess(event, 123))
		
		// 添加心跳事件类型
		filter.AddEventType(EventHeartbeat)
		assert.True(t, filter.ShouldProcess(event, 123))
		
		// 移除心跳事件类型
		filter.RemoveEventType(EventHeartbeat)
		assert.False(t, filter.ShouldProcess(event, 123))
	})
}
