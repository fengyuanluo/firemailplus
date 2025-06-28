package sse

import (
	"context"
	"net/http"
	"time"
)

// EventPublisher 事件发布器接口
type EventPublisher interface {
	// Publish 发布事件给指定用户
	Publish(ctx context.Context, event *Event) error
	
	// PublishToUser 发布事件给指定用户
	PublishToUser(ctx context.Context, userID uint, event *Event) error
	
	// PublishToAccount 发布事件给指定账户的用户
	PublishToAccount(ctx context.Context, accountID uint, event *Event) error
	
	// Broadcast 广播事件给所有连接的用户
	Broadcast(ctx context.Context, event *Event) error
}

// ConnectionManager 连接管理器接口
type ConnectionManager interface {
	// AddConnection 添加客户端连接
	AddConnection(userID uint, clientID string, conn ClientConnection) error
	
	// RemoveConnection 移除客户端连接
	RemoveConnection(userID uint, clientID string) error
	
	// GetConnections 获取用户的所有连接
	GetConnections(userID uint) []ClientConnection
	
	// GetConnectionCount 获取连接数统计
	GetConnectionCount() (total int, byUser map[uint]int)
	
	// SendToUser 发送消息给指定用户的所有连接
	SendToUser(userID uint, data []byte) error
	
	// SendToConnection 发送消息给指定连接
	SendToConnection(userID uint, clientID string, data []byte) error
	
	// CleanupInactiveConnections 清理非活跃连接
	CleanupInactiveConnections() error
}

// ClientConnection 客户端连接接口
type ClientConnection interface {
	// Send 发送数据到客户端
	Send(data []byte) error
	
	// Close 关闭连接
	Close() error
	
	// IsActive 检查连接是否活跃
	IsActive() bool
	
	// GetClientID 获取客户端ID
	GetClientID() string
	
	// GetUserID 获取用户ID
	GetUserID() uint
	
	// GetConnectedAt 获取连接时间
	GetConnectedAt() time.Time
	
	// GetLastActivity 获取最后活动时间
	GetLastActivity() time.Time
	
	// UpdateActivity 更新活动时间
	UpdateActivity()
}

// EventDispatcher 事件分发器接口
type EventDispatcher interface {
	// Dispatch 分发事件
	Dispatch(ctx context.Context, event *Event) error
	
	// Subscribe 订阅事件类型
	Subscribe(eventType EventType, handler EventHandler) error
	
	// Unsubscribe 取消订阅事件类型
	Unsubscribe(eventType EventType, handler EventHandler) error
	
	// Start 启动事件分发器
	Start(ctx context.Context) error
	
	// Stop 停止事件分发器
	Stop() error
}

// EventHandler 事件处理器接口
type EventHandler interface {
	// Handle 处理事件
	Handle(ctx context.Context, event *Event) error
	
	// GetHandlerID 获取处理器ID
	GetHandlerID() string
}

// EventFilter 事件过滤器接口
type EventFilter interface {
	// ShouldProcess 判断是否应该处理该事件
	ShouldProcess(event *Event, userID uint) bool
}

// EventStore 事件存储接口（可选，用于事件持久化）
type EventStore interface {
	// Store 存储事件
	Store(ctx context.Context, event *Event) error
	
	// GetEvents 获取用户的历史事件
	GetEvents(ctx context.Context, userID uint, limit int, offset int) ([]*Event, error)
	
	// GetEventsByType 根据类型获取事件
	GetEventsByType(ctx context.Context, userID uint, eventType EventType, limit int) ([]*Event, error)
	
	// CleanupOldEvents 清理旧事件
	CleanupOldEvents(ctx context.Context, olderThan time.Time) error
}

// SSEService SSE服务接口
type SSEService interface {
	// Start 启动SSE服务
	Start(ctx context.Context) error
	
	// Stop 停止SSE服务
	Stop() error
	
	// HandleConnection 处理新的SSE连接
	HandleConnection(ctx context.Context, userID uint, clientID string, w http.ResponseWriter, r *http.Request) error
	
	// PublishEvent 发布事件
	PublishEvent(ctx context.Context, event *Event) error
	
	// GetStats 获取服务统计信息
	GetStats() ServiceStats
}

// ServiceStats 服务统计信息
type ServiceStats struct {
	TotalConnections    int            `json:"total_connections"`
	ConnectionsByUser   map[uint]int   `json:"connections_by_user"`
	EventsPublished     int64          `json:"events_published"`
	EventsByType        map[EventType]int64 `json:"events_by_type"`
	StartTime           time.Time      `json:"start_time"`
	LastEventTime       *time.Time     `json:"last_event_time,omitempty"`
}
