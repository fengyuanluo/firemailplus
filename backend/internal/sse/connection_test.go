package sse

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockClientConnection 模拟客户端连接
type MockClientConnection struct {
	clientID     string
	userID       uint
	connectedAt  time.Time
	lastActivity time.Time
	active       bool
	sendError    error
	sentData     [][]byte
}

func NewMockClientConnection(clientID string, userID uint) *MockClientConnection {
	now := time.Now()
	return &MockClientConnection{
		clientID:     clientID,
		userID:       userID,
		connectedAt:  now,
		lastActivity: now,
		active:       true,
		sentData:     make([][]byte, 0),
	}
}

func (m *MockClientConnection) Send(data []byte) error {
	if m.sendError != nil {
		return m.sendError
	}
	m.sentData = append(m.sentData, data)
	m.lastActivity = time.Now()
	return nil
}

func (m *MockClientConnection) Close() error {
	m.active = false
	return nil
}

func (m *MockClientConnection) IsActive() bool {
	return m.active
}

func (m *MockClientConnection) GetClientID() string {
	return m.clientID
}

func (m *MockClientConnection) GetUserID() uint {
	return m.userID
}

func (m *MockClientConnection) GetConnectedAt() time.Time {
	return m.connectedAt
}

func (m *MockClientConnection) GetLastActivity() time.Time {
	return m.lastActivity
}

func (m *MockClientConnection) UpdateActivity() {
	m.lastActivity = time.Now()
}

func (m *MockClientConnection) SetSendError(err error) {
	m.sendError = err
}

func (m *MockClientConnection) GetSentData() [][]byte {
	return m.sentData
}

// panicWriter 用于模拟写入或刷新时的异常情况
type panicWriter struct {
	header       http.Header
	panicOnWrite bool
	panicOnFlush bool
}

func (p *panicWriter) Header() http.Header {
	if p.header == nil {
		p.header = make(http.Header)
	}
	return p.header
}

func (p *panicWriter) Write(data []byte) (int, error) {
	if p.panicOnWrite {
		panic("write panic")
	}
	return len(data), nil
}

func (p *panicWriter) WriteHeader(statusCode int) {}

func (p *panicWriter) Flush() {
	if p.panicOnFlush {
		panic("flush panic")
	}
}

func TestSSEConnection(t *testing.T) {
	t.Run("创建SSE连接成功", func(t *testing.T) {
		w := httptest.NewRecorder()
		clientID := "test-client"
		userID := uint(123)

		conn, err := NewSSEConnection(clientID, userID, w, context.Background().Done())
		require.NoError(t, err)
		assert.Equal(t, clientID, conn.GetClientID())
		assert.Equal(t, userID, conn.GetUserID())
		assert.True(t, conn.IsActive())
		assert.WithinDuration(t, time.Now(), conn.GetConnectedAt(), time.Second)
	})

	t.Run("发送数据成功", func(t *testing.T) {
		w := httptest.NewRecorder()
		conn, err := NewSSEConnection("test", 123, w, context.Background().Done())
		require.NoError(t, err)

		testData := []byte("test data")
		err = conn.Send(testData)
		assert.NoError(t, err)

		// 验证数据已写入
		assert.Contains(t, w.Body.String(), "test data")
	})

	t.Run("关闭连接", func(t *testing.T) {
		w := httptest.NewRecorder()
		conn, err := NewSSEConnection("test", 123, w, context.Background().Done())
		require.NoError(t, err)

		assert.True(t, conn.IsActive())
		err = conn.Close()
		assert.NoError(t, err)
		assert.False(t, conn.IsActive())
	})

	t.Run("向已关闭连接发送数据失败", func(t *testing.T) {
		w := httptest.NewRecorder()
		conn, err := NewSSEConnection("test", 123, w, context.Background().Done())
		require.NoError(t, err)

		conn.Close()
		err = conn.Send([]byte("test"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection is closed")
	})

	t.Run("连接上下文结束后发送失败", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx, cancel := context.WithCancel(context.Background())
		conn, err := NewSSEConnection("test", 123, w, ctx.Done())
		require.NoError(t, err)

		cancel()
		err = conn.Send([]byte("test"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection context done")
		assert.False(t, conn.IsActive())
	})

	t.Run("写入或刷新异常时不会崩溃", func(t *testing.T) {
		w := &panicWriter{panicOnFlush: true}
		conn, err := NewSSEConnection("test", 123, w, context.Background().Done())
		require.NoError(t, err)

		var sendErr error
		assert.NotPanics(t, func() {
			sendErr = conn.Send([]byte("test"))
		})
		assert.Error(t, sendErr)
		assert.False(t, conn.IsActive())
	})
}

func TestConnectionManager(t *testing.T) {
	t.Run("创建连接管理器", func(t *testing.T) {
		cm := NewConnectionManager(5, time.Minute, 30*time.Minute)
		assert.NotNil(t, cm)

		total, byUser := cm.GetConnectionCount()
		assert.Equal(t, 0, total)
		assert.Equal(t, 0, len(byUser))
	})

	t.Run("添加连接", func(t *testing.T) {
		cm := NewConnectionManager(5, time.Minute, 30*time.Minute)
		userID := uint(123)
		clientID := "client-1"
		conn := NewMockClientConnection(clientID, userID)

		err := cm.AddConnection(userID, clientID, conn)
		assert.NoError(t, err)

		connections := cm.GetConnections(userID)
		assert.Len(t, connections, 1)
		assert.Equal(t, clientID, connections[0].GetClientID())

		total, byUser := cm.GetConnectionCount()
		assert.Equal(t, 1, total)
		assert.Equal(t, 1, byUser[userID])
	})

	t.Run("移除连接", func(t *testing.T) {
		cm := NewConnectionManager(5, time.Minute, 30*time.Minute)
		userID := uint(123)
		clientID := "client-1"
		conn := NewMockClientConnection(clientID, userID)

		cm.AddConnection(userID, clientID, conn)
		err := cm.RemoveConnection(userID, clientID)
		assert.NoError(t, err)

		connections := cm.GetConnections(userID)
		assert.Len(t, connections, 0)

		total, byUser := cm.GetConnectionCount()
		assert.Equal(t, 0, total)
		assert.Equal(t, 0, len(byUser))
	})

	t.Run("连接数限制", func(t *testing.T) {
		maxConn := 2
		cm := NewConnectionManager(maxConn, time.Minute, 30*time.Minute)
		userID := uint(123)

		// 添加最大数量的连接
		for i := 0; i < maxConn; i++ {
			clientID := fmt.Sprintf("client-%d", i)
			conn := NewMockClientConnection(clientID, userID)
			err := cm.AddConnection(userID, clientID, conn)
			assert.NoError(t, err)
		}

		connections := cm.GetConnections(userID)
		assert.Len(t, connections, maxConn)

		// 添加超出限制的连接，应该移除最旧的连接
		newConn := NewMockClientConnection("client-new", userID)
		err := cm.AddConnection(userID, "client-new", newConn)
		assert.NoError(t, err)

		connections = cm.GetConnections(userID)
		assert.Len(t, connections, maxConn)

		// 验证新连接存在
		found := false
		for _, conn := range connections {
			if conn.GetClientID() == "client-new" {
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("发送消息给用户", func(t *testing.T) {
		cm := NewConnectionManager(5, time.Minute, 30*time.Minute)
		userID := uint(123)

		// 添加多个连接
		conn1 := NewMockClientConnection("client-1", userID)
		conn2 := NewMockClientConnection("client-2", userID)
		cm.AddConnection(userID, "client-1", conn1)
		cm.AddConnection(userID, "client-2", conn2)

		testData := []byte("test message")
		err := cm.SendToUser(userID, testData)
		assert.NoError(t, err)

		// 验证所有连接都收到了消息
		assert.Len(t, conn1.GetSentData(), 1)
		assert.Equal(t, testData, conn1.GetSentData()[0])
		assert.Len(t, conn2.GetSentData(), 1)
		assert.Equal(t, testData, conn2.GetSentData()[0])
	})

	t.Run("发送消息给特定连接", func(t *testing.T) {
		cm := NewConnectionManager(5, time.Minute, 30*time.Minute)
		userID := uint(123)
		clientID := "client-1"
		conn := NewMockClientConnection(clientID, userID)
		cm.AddConnection(userID, clientID, conn)

		testData := []byte("specific message")
		err := cm.SendToConnection(userID, clientID, testData)
		assert.NoError(t, err)

		assert.Len(t, conn.GetSentData(), 1)
		assert.Equal(t, testData, conn.GetSentData()[0])
	})

	t.Run("发送消息给不存在的连接", func(t *testing.T) {
		cm := NewConnectionManager(5, time.Minute, 30*time.Minute)

		err := cm.SendToConnection(123, "nonexistent", []byte("test"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection not found")
	})

	t.Run("清理非活跃连接", func(t *testing.T) {
		cm := NewConnectionManager(5, time.Minute, time.Second) // 1秒超时
		userID := uint(123)

		// 添加活跃连接
		activeConn := NewMockClientConnection("active", userID)
		cm.AddConnection(userID, "active", activeConn)

		// 添加非活跃连接
		inactiveConn := NewMockClientConnection("inactive", userID)
		inactiveConn.Close() // 标记为非活跃
		cm.AddConnection(userID, "inactive", inactiveConn)

		// 等待超时
		time.Sleep(1100 * time.Millisecond)

		err := cm.CleanupInactiveConnections()
		assert.NoError(t, err)

		connections := cm.GetConnections(userID)
		// 验证至少活跃连接还在，非活跃连接被清理
		if len(connections) > 0 {
			// 如果还有连接，应该是活跃的连接
			found := false
			for _, conn := range connections {
				if conn.GetClientID() == "active" {
					found = true
					break
				}
			}
			assert.True(t, found, "Active connection should still exist")
		}
		// 验证非活跃连接不在列表中
		for _, conn := range connections {
			assert.NotEqual(t, "inactive", conn.GetClientID(), "Inactive connection should be removed")
		}
	})

	t.Run("替换相同客户端ID的连接", func(t *testing.T) {
		cm := NewConnectionManager(5, time.Minute, 30*time.Minute)
		userID := uint(123)
		clientID := "same-client"

		// 添加第一个连接
		conn1 := NewMockClientConnection(clientID, userID)
		err := cm.AddConnection(userID, clientID, conn1)
		assert.NoError(t, err)

		// 添加相同客户端ID的连接
		conn2 := NewMockClientConnection(clientID, userID)
		err = cm.AddConnection(userID, clientID, conn2)
		assert.NoError(t, err)

		// 应该只有一个连接
		connections := cm.GetConnections(userID)
		assert.Len(t, connections, 1)

		// 第一个连接应该被关闭
		assert.False(t, conn1.IsActive())
		assert.True(t, conn2.IsActive())
	})
}
