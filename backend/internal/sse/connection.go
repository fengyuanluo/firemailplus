package sse

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// SSEConnection SSE连接实现
type SSEConnection struct {
	clientID     string
	userID       uint
	writer       http.ResponseWriter
	flusher      http.Flusher
	connectedAt  time.Time
	lastActivity time.Time
	closed       bool
	mutex        sync.RWMutex
}

// NewSSEConnection 创建新的SSE连接
func NewSSEConnection(clientID string, userID uint, w http.ResponseWriter) (*SSEConnection, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming unsupported")
	}

	// 设置SSE响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	now := time.Now()
	conn := &SSEConnection{
		clientID:     clientID,
		userID:       userID,
		writer:       w,
		flusher:      flusher,
		connectedAt:  now,
		lastActivity: now,
		closed:       false,
	}

	return conn, nil
}

// Send 发送数据到客户端
func (c *SSEConnection) Send(data []byte) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.closed {
		return fmt.Errorf("connection is closed")
	}

	// 写入数据
	_, err := c.writer.Write(data)
	if err != nil {
		c.closed = true
		return fmt.Errorf("failed to write data: %w", err)
	}

	// 立即刷新
	c.flusher.Flush()
	c.lastActivity = time.Now()

	return nil
}

// Close 关闭连接
func (c *SSEConnection) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.closed {
		c.closed = true
	}
	return nil
}

// IsActive 检查连接是否活跃
func (c *SSEConnection) IsActive() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return !c.closed
}

// GetClientID 获取客户端ID
func (c *SSEConnection) GetClientID() string {
	return c.clientID
}

// GetUserID 获取用户ID
func (c *SSEConnection) GetUserID() uint {
	return c.userID
}

// GetConnectedAt 获取连接时间
func (c *SSEConnection) GetConnectedAt() time.Time {
	return c.connectedAt
}

// GetLastActivity 获取最后活动时间
func (c *SSEConnection) GetLastActivity() time.Time {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.lastActivity
}

// UpdateActivity 更新活动时间
func (c *SSEConnection) UpdateActivity() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.lastActivity = time.Now()
}

// ConnectionManagerImpl 连接管理器实现
type ConnectionManagerImpl struct {
	connections map[uint]map[string]ClientConnection // userID -> clientID -> connection
	mutex       sync.RWMutex
	maxConnPerUser int
	cleanupInterval time.Duration
	connectionTimeout time.Duration
}

// NewConnectionManager 创建连接管理器
func NewConnectionManager(maxConnPerUser int, cleanupInterval, connectionTimeout time.Duration) *ConnectionManagerImpl {
	return &ConnectionManagerImpl{
		connections:       make(map[uint]map[string]ClientConnection),
		maxConnPerUser:    maxConnPerUser,
		cleanupInterval:   cleanupInterval,
		connectionTimeout: connectionTimeout,
	}
}

// AddConnection 添加客户端连接
func (cm *ConnectionManagerImpl) AddConnection(userID uint, clientID string, conn ClientConnection) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 初始化用户连接映射
	if cm.connections[userID] == nil {
		cm.connections[userID] = make(map[string]ClientConnection)
	}

	// 检查连接数限制
	if len(cm.connections[userID]) >= cm.maxConnPerUser {
		// 移除最旧的连接
		cm.removeOldestConnection(userID)
	}

	// 如果已存在相同clientID的连接，先关闭旧连接
	if existingConn, exists := cm.connections[userID][clientID]; exists {
		existingConn.Close()
	}

	// 添加新连接
	cm.connections[userID][clientID] = conn

	return nil
}

// RemoveConnection 移除客户端连接
func (cm *ConnectionManagerImpl) RemoveConnection(userID uint, clientID string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if userConns, exists := cm.connections[userID]; exists {
		if conn, exists := userConns[clientID]; exists {
			conn.Close()
			delete(userConns, clientID)

			// 如果用户没有连接了，删除用户映射
			if len(userConns) == 0 {
				delete(cm.connections, userID)
			}
		}
	}

	return nil
}

// GetConnections 获取用户的所有连接
func (cm *ConnectionManagerImpl) GetConnections(userID uint) []ClientConnection {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	var connections []ClientConnection
	if userConns, exists := cm.connections[userID]; exists {
		for _, conn := range userConns {
			if conn.IsActive() {
				connections = append(connections, conn)
			}
		}
	}

	return connections
}

// GetConnectionCount 获取连接数统计
func (cm *ConnectionManagerImpl) GetConnectionCount() (total int, byUser map[uint]int) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	byUser = make(map[uint]int)
	total = 0

	for userID, userConns := range cm.connections {
		activeCount := 0
		for _, conn := range userConns {
			if conn.IsActive() {
				activeCount++
			}
		}
		if activeCount > 0 {
			byUser[userID] = activeCount
			total += activeCount
		}
	}

	return total, byUser
}

// SendToUser 发送消息给指定用户的所有连接
func (cm *ConnectionManagerImpl) SendToUser(userID uint, data []byte) error {
	connections := cm.GetConnections(userID)
	
	var errors []error
	for _, conn := range connections {
		if err := conn.Send(data); err != nil {
			errors = append(errors, err)
			// 发送失败时移除连接
			cm.RemoveConnection(userID, conn.GetClientID())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to send to %d connections", len(errors))
	}

	return nil
}

// SendToConnection 发送消息给指定连接
func (cm *ConnectionManagerImpl) SendToConnection(userID uint, clientID string, data []byte) error {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if userConns, exists := cm.connections[userID]; exists {
		if conn, exists := userConns[clientID]; exists && conn.IsActive() {
			return conn.Send(data)
		}
	}

	return fmt.Errorf("connection not found")
}

// CleanupInactiveConnections 清理非活跃连接
func (cm *ConnectionManagerImpl) CleanupInactiveConnections() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	now := time.Now()
	var removedCount int

	for userID, userConns := range cm.connections {
		for clientID, conn := range userConns {
			// 检查连接是否超时或已关闭
			if !conn.IsActive() || now.Sub(conn.GetLastActivity()) > cm.connectionTimeout {
				conn.Close()
				delete(userConns, clientID)
				removedCount++
			}
		}

		// 如果用户没有连接了，删除用户映射
		if len(userConns) == 0 {
			delete(cm.connections, userID)
		}
	}

	return nil
}

// removeOldestConnection 移除最旧的连接（内部方法，调用时需要持有锁）
func (cm *ConnectionManagerImpl) removeOldestConnection(userID uint) {
	userConns := cm.connections[userID]
	if len(userConns) == 0 {
		return
	}

	var oldestClientID string
	var oldestTime time.Time

	for clientID, conn := range userConns {
		connTime := conn.GetConnectedAt()
		if oldestClientID == "" || connTime.Before(oldestTime) {
			oldestClientID = clientID
			oldestTime = connTime
		}
	}

	if oldestClientID != "" {
		if conn, exists := userConns[oldestClientID]; exists {
			conn.Close()
			delete(userConns, oldestClientID)
		}
	}
}

// StartCleanupRoutine 启动清理例程
func (cm *ConnectionManagerImpl) StartCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(cm.cleanupInterval)
		defer ticker.Stop()

		for range ticker.C {
			cm.CleanupInactiveConnections()
		}
	}()
}
