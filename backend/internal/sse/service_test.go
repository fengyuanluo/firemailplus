package sse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestSSEService() *SSEServiceImpl {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	
	config := &SSEConfig{
		MaxConnectionsPerUser: 3,
		ConnectionTimeout:     time.Minute,
		HeartbeatInterval:     10 * time.Second,
		CleanupInterval:       30 * time.Second,
		BufferSize:            1024,
		EnableHeartbeat:       true,
	}
	
	return NewSSEService(db, config)
}

func TestSSEService(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database tests in short mode")
	}
	t.Run("创建SSE服务", func(t *testing.T) {
		service := setupTestSSEService()
		assert.NotNil(t, service)
		assert.NotNil(t, service.connectionManager)
		assert.NotNil(t, service.eventPublisher)
		assert.NotNil(t, service.config)
	})

	t.Run("使用默认配置创建服务", func(t *testing.T) {
		db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		service := NewSSEService(db, nil)
		assert.NotNil(t, service)
		
		// 验证默认配置
		impl := service
		assert.Equal(t, 5, impl.config.MaxConnectionsPerUser)
		assert.Equal(t, 30*time.Minute, impl.config.ConnectionTimeout)
		assert.Equal(t, 30*time.Second, impl.config.HeartbeatInterval)
		assert.True(t, impl.config.EnableHeartbeat)
	})

	t.Run("启动和停止服务", func(t *testing.T) {
		service := setupTestSSEService()
		
		err := service.Start(context.Background())
		assert.NoError(t, err)
		
		err = service.Stop()
		assert.NoError(t, err)
	})

	t.Run("发布事件", func(t *testing.T) {
		service := setupTestSSEService()
		
		event := NewNotificationEvent("Test", "Test message", "info", 123)
		err := service.PublishEvent(context.Background(), event)
		assert.NoError(t, err)
		
		stats := service.GetStats()
		assert.Equal(t, int64(1), stats.EventsPublished)
		assert.Equal(t, int64(1), stats.EventsByType[EventNotification])
	})

	t.Run("获取统计信息", func(t *testing.T) {
		service := setupTestSSEService()
		
		stats := service.GetStats()
		assert.Equal(t, 0, stats.TotalConnections)
		assert.Equal(t, int64(0), stats.EventsPublished)
		assert.NotNil(t, stats.EventsByType)
		assert.NotNil(t, stats.ConnectionsByUser)
		assert.WithinDuration(t, time.Now(), stats.StartTime, time.Second)
	})

	t.Run("获取事件发布器", func(t *testing.T) {
		service := setupTestSSEService()
		
		publisher := service.GetEventPublisher()
		assert.NotNil(t, publisher)
		assert.Implements(t, (*EventPublisher)(nil), publisher)
	})
}

func TestSSEHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	t.Run("SSE处理器成功连接", func(t *testing.T) {
		service := setupTestSSEService()
		handler := SSEHandler(SSEService(service))
		
		router := gin.New()
		router.GET("/sse", func(c *gin.Context) {
			// 模拟认证中间件设置用户ID
			c.Set("user_id", uint(123))
			handler(c)
		})
		
		req := httptest.NewRequest("GET", "/sse?client_id=test-client", nil)
		req.Header.Set("Accept", "text/event-stream")
		w := httptest.NewRecorder()
		
		// 由于SSE连接会保持打开，我们需要在goroutine中处理
		go func() {
			router.ServeHTTP(w, req)
		}()
		
		// 等待一小段时间让连接建立
		time.Sleep(100 * time.Millisecond)
		
		// 验证响应头
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
		assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
		assert.Equal(t, "keep-alive", w.Header().Get("Connection"))
	})

	t.Run("未认证用户访问SSE", func(t *testing.T) {
		service := setupTestSSEService()
		handler := SSEHandler(SSEService(service))
		
		router := gin.New()
		router.GET("/sse", handler)
		
		req := httptest.NewRequest("GET", "/sse", nil)
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("不支持SSE的客户端", func(t *testing.T) {
		service := setupTestSSEService()
		handler := SSEHandler(SSEService(service))
		
		router := gin.New()
		router.GET("/sse", func(c *gin.Context) {
			c.Set("user_id", uint(123))
			handler(c)
		})
		
		req := httptest.NewRequest("GET", "/sse", nil)
		// 不设置Accept头或设置错误的Accept头
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "SSE support")
	})

	t.Run("自动生成客户端ID", func(t *testing.T) {
		service := setupTestSSEService()
		handler := SSEHandler(SSEService(service))
		
		router := gin.New()
		router.GET("/sse", func(c *gin.Context) {
			c.Set("user_id", uint(123))
			handler(c)
		})
		
		req := httptest.NewRequest("GET", "/sse", nil)
		req.Header.Set("Accept", "text/event-stream")
		w := httptest.NewRecorder()
		
		go func() {
			router.ServeHTTP(w, req)
		}()
		
		time.Sleep(100 * time.Millisecond)
		
		// 验证连接已建立（通过统计信息）
		stats := service.GetStats()
		assert.Equal(t, 1, stats.TotalConnections)
	})
}

func TestSSEStatsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	t.Run("获取SSE统计信息", func(t *testing.T) {
		service := setupTestSSEService()
		handler := SSEStatsHandler(SSEService(service))
		
		router := gin.New()
		router.GET("/stats", handler)
		
		req := httptest.NewRequest("GET", "/stats", nil)
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"success":true`)
		assert.Contains(t, w.Body.String(), `"total_connections"`)
		assert.Contains(t, w.Body.String(), `"events_published"`)
	})
}

func TestSSEConfig(t *testing.T) {
	t.Run("默认配置", func(t *testing.T) {
		config := DefaultSSEConfig()
		
		assert.Equal(t, 5, config.MaxConnectionsPerUser)
		assert.Equal(t, 30*time.Minute, config.ConnectionTimeout)
		assert.Equal(t, 30*time.Second, config.HeartbeatInterval)
		assert.Equal(t, 5*time.Minute, config.CleanupInterval)
		assert.Equal(t, 1024, config.BufferSize)
		assert.True(t, config.EnableHeartbeat)
	})
}

func TestServiceStats(t *testing.T) {
	t.Run("服务统计信息结构", func(t *testing.T) {
		stats := ServiceStats{
			TotalConnections:  5,
			ConnectionsByUser: map[uint]int{123: 2, 456: 3},
			EventsPublished:   100,
			EventsByType:      map[EventType]int64{EventNewEmail: 50, EventNotification: 50},
			StartTime:         time.Now(),
		}
		
		assert.Equal(t, 5, stats.TotalConnections)
		assert.Equal(t, 2, stats.ConnectionsByUser[123])
		assert.Equal(t, 3, stats.ConnectionsByUser[456])
		assert.Equal(t, int64(100), stats.EventsPublished)
		assert.Equal(t, int64(50), stats.EventsByType[EventNewEmail])
		assert.Equal(t, int64(50), stats.EventsByType[EventNotification])
	})
}

// 集成测试
func TestSSEIntegration(t *testing.T) {
	t.Run("完整的SSE流程", func(t *testing.T) {
		service := setupTestSSEService()
		err := service.Start(context.Background())
		require.NoError(t, err)
		defer service.Stop()
		
		// 模拟客户端连接
		userID := uint(123)
		clientID := "integration-test-client"
		
		// 创建模拟的HTTP响应写入器
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/sse", nil)
		
		// 模拟连接建立
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		
		go func() {
			err := service.HandleConnection(ctx, userID, clientID, w, req.WithContext(ctx))
			assert.NoError(t, err)
		}()
		
		// 等待连接建立
		time.Sleep(100 * time.Millisecond)
		
		// 验证连接统计
		stats := service.GetStats()
		assert.Equal(t, 1, stats.TotalConnections)
		assert.Equal(t, 1, stats.ConnectionsByUser[userID])
		
		// 发布事件
		event := NewNotificationEvent("Integration Test", "Test message", "info", userID)
		err = service.PublishEvent(context.Background(), event)
		assert.NoError(t, err)
		
		// 验证事件统计
		stats = service.GetStats()
		assert.Equal(t, int64(2), stats.EventsPublished) // 包括欢迎事件
		assert.Equal(t, int64(2), stats.EventsByType[EventNotification]) // 欢迎事件 + 测试事件
		
		// 关闭连接
		cancel()
		time.Sleep(100 * time.Millisecond)
		
		// 验证连接已关闭
		stats = service.GetStats()
		assert.Equal(t, 0, stats.TotalConnections)
	})
}
