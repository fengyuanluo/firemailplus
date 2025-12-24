package sse

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SSEServiceImpl SSE服务实现
type SSEServiceImpl struct {
	connectionManager ConnectionManager
	eventPublisher    EventPublisher
	db                *gorm.DB
	config            *SSEConfig
	stats             *ServiceStats
	heartbeatTicker   *time.Ticker
	stopChan          chan struct{}
	mutex             sync.RWMutex
}

// SSEConfig SSE配置
type SSEConfig struct {
	MaxConnectionsPerUser int           `json:"max_connections_per_user"`
	ConnectionTimeout     time.Duration `json:"connection_timeout"`
	HeartbeatInterval     time.Duration `json:"heartbeat_interval"`
	CleanupInterval       time.Duration `json:"cleanup_interval"`
	BufferSize            int           `json:"buffer_size"`
	EnableHeartbeat       bool          `json:"enable_heartbeat"`
}

// DefaultSSEConfig 默认SSE配置
func DefaultSSEConfig() *SSEConfig {
	return &SSEConfig{
		MaxConnectionsPerUser: 5,
		ConnectionTimeout:     30 * time.Minute,
		HeartbeatInterval:     30 * time.Second,
		CleanupInterval:       5 * time.Minute,
		BufferSize:            1024,
		EnableHeartbeat:       true,
	}
}

// NewSSEService 创建SSE服务
func NewSSEService(db *gorm.DB, config *SSEConfig) *SSEServiceImpl {
	if config == nil {
		config = DefaultSSEConfig()
	}

	connectionManager := NewConnectionManager(
		config.MaxConnectionsPerUser,
		config.CleanupInterval,
		config.ConnectionTimeout,
	)

	eventPublisher := NewEventPublisher(connectionManager, db)

	return &SSEServiceImpl{
		connectionManager: connectionManager,
		eventPublisher:    eventPublisher,
		db:                db,
		config:            config,
		stats: &ServiceStats{
			EventsByType:      make(map[EventType]int64),
			ConnectionsByUser: make(map[uint]int),
			StartTime:         time.Now(),
		},
		stopChan: make(chan struct{}),
	}
}

// Start 启动SSE服务
func (s *SSEServiceImpl) Start(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 启动连接清理例程
	if cm, ok := s.connectionManager.(*ConnectionManagerImpl); ok {
		cm.StartCleanupRoutine()
	}

	// 启动心跳例程
	if s.config.EnableHeartbeat {
		s.startHeartbeatRoutine()
	}

	log.Println("SSE service started")
	return nil
}

// Stop 停止SSE服务
func (s *SSEServiceImpl) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 停止心跳
	if s.heartbeatTicker != nil {
		s.heartbeatTicker.Stop()
	}

	// 发送停止信号
	close(s.stopChan)

	log.Println("SSE service stopped")
	return nil
}

// HandleConnection 处理新的SSE连接
func (s *SSEServiceImpl) HandleConnection(ctx context.Context, userID uint, clientID string, w http.ResponseWriter, r *http.Request) error {
	// 创建SSE连接
	conn, err := NewSSEConnection(clientID, userID, w, r.Context().Done())
	if err != nil {
		return fmt.Errorf("failed to create SSE connection: %w", err)
	}

	// 添加到连接管理器
	if err := s.connectionManager.AddConnection(userID, clientID, conn); err != nil {
		return fmt.Errorf("failed to add connection: %w", err)
	}

	// 更新统计信息
	s.updateConnectionStats(userID, 1)

	// 发送连接确认事件
	welcomeEvent := NewNotificationEvent(
		"连接成功",
		"SSE连接已建立，您将收到实时邮件通知",
		"success",
		userID,
	)

	if err := s.PublishEvent(ctx, welcomeEvent); err != nil {
		log.Printf("Failed to send welcome event: %v", err)
	}

	// 监听连接断开
	go s.monitorConnection(ctx, userID, clientID, r)

	log.Printf("SSE connection established for user %d, client %s", userID, clientID)
	return nil
}

// PublishEvent 发布事件
func (s *SSEServiceImpl) PublishEvent(ctx context.Context, event *Event) error {
	if err := s.eventPublisher.Publish(ctx, event); err != nil {
		return err
	}

	// 更新统计信息
	s.updateEventStats(event)
	return nil
}

// GetStats 获取服务统计信息
func (s *SSEServiceImpl) GetStats() ServiceStats {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// 获取当前连接统计
	totalConnections, connectionsByUser := s.connectionManager.GetConnectionCount()

	stats := ServiceStats{
		TotalConnections:  totalConnections,
		ConnectionsByUser: connectionsByUser,
		EventsPublished:   s.stats.EventsPublished,
		EventsByType:      make(map[EventType]int64),
		StartTime:         s.stats.StartTime,
	}

	// 复制事件类型统计
	for k, v := range s.stats.EventsByType {
		stats.EventsByType[k] = v
	}

	if s.stats.LastEventTime != nil {
		lastTime := *s.stats.LastEventTime
		stats.LastEventTime = &lastTime
	}

	return stats
}

// GetEventPublisher 获取事件发布器
func (s *SSEServiceImpl) GetEventPublisher() EventPublisher {
	return s.eventPublisher
}

// startHeartbeatRoutine 启动心跳例程
func (s *SSEServiceImpl) startHeartbeatRoutine() {
	s.heartbeatTicker = time.NewTicker(s.config.HeartbeatInterval)

	go func() {
		for {
			select {
			case <-s.heartbeatTicker.C:
				s.sendHeartbeat()
			case <-s.stopChan:
				return
			}
		}
	}()
}

// sendHeartbeat 发送心跳
func (s *SSEServiceImpl) sendHeartbeat() {
	_, userConnections := s.connectionManager.GetConnectionCount()

	for userID := range userConnections {
		heartbeatEvent := NewHeartbeatEvent("")
		if err := s.eventPublisher.PublishToUser(context.Background(), userID, heartbeatEvent); err != nil {
			log.Printf("Failed to send heartbeat to user %d: %v", userID, err)
		}
	}
}

// monitorConnection 监控连接状态
func (s *SSEServiceImpl) monitorConnection(ctx context.Context, userID uint, clientID string, r *http.Request) {
	// 等待连接断开
	<-r.Context().Done()

	// 移除连接
	if err := s.connectionManager.RemoveConnection(userID, clientID); err != nil {
		log.Printf("Failed to remove connection: %v", err)
	}

	// 更新统计信息
	s.updateConnectionStats(userID, -1)

	log.Printf("SSE connection closed for user %d, client %s", userID, clientID)
}

// updateConnectionStats 更新连接统计
func (s *SSEServiceImpl) updateConnectionStats(userID uint, delta int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.stats.ConnectionsByUser == nil {
		s.stats.ConnectionsByUser = make(map[uint]int)
	}

	s.stats.ConnectionsByUser[userID] += delta
	if s.stats.ConnectionsByUser[userID] <= 0 {
		delete(s.stats.ConnectionsByUser, userID)
	}

	// 更新总连接数
	total := 0
	for _, count := range s.stats.ConnectionsByUser {
		total += count
	}
	s.stats.TotalConnections = total
}

// updateEventStats 更新事件统计
func (s *SSEServiceImpl) updateEventStats(event *Event) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.stats.EventsPublished++
	s.stats.EventsByType[event.Type]++

	now := time.Now()
	s.stats.LastEventTime = &now
}

// SSEHandler Gin SSE处理器
func SSEHandler(sseService SSEService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户ID（从认证中间件）
		userIDValue, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		userID, ok := userIDValue.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
			return
		}

		// 获取或生成客户端ID
		clientID := c.Query("client_id")
		if clientID == "" {
			clientID = uuid.New().String()
		}

		// 检查连接是否支持SSE
		if c.GetHeader("Accept") != "text/event-stream" {
			c.Header("Content-Type", "text/plain")
			c.String(http.StatusBadRequest, "This endpoint requires SSE support (Accept: text/event-stream)")
			return
		}

		// 处理SSE连接
		if err := sseService.HandleConnection(c.Request.Context(), userID, clientID, c.Writer, c.Request); err != nil {
			log.Printf("SSE connection error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to establish SSE connection"})
			return
		}

		// 保持连接打开
		select {
		case <-c.Request.Context().Done():
			return
		}
	}
}

// SSEStatsHandler SSE统计信息处理器
func SSEStatsHandler(sseService SSEService) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats := sseService.GetStats()
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    stats,
		})
	}
}
