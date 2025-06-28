package handlers

import (
	"net/http"

	"firemail/internal/sse"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HandleSSE 处理SSE连接
func (h *Handler) HandleSSE(c *gin.Context) {
	// 尝试从查询参数获取token进行认证
	token := c.Query("token")
	var userID uint
	var exists bool

	if token != "" {
		// 使用token认证
		user, err := h.authService.ValidateToken(token)
		if err != nil {
			h.respondWithError(c, http.StatusUnauthorized, "Invalid token")
			return
		}
		userID = user.ID
		exists = true
	} else {
		// 尝试从中间件获取用户ID
		userID, exists = h.getCurrentUserID(c)
		if !exists {
			h.respondWithError(c, http.StatusUnauthorized, "Authentication required")
			return
		}
	}

	// 获取或生成客户端ID
	clientID := c.Query("client_id")
	if clientID == "" {
		clientID = uuid.New().String()
	}

	// 检查Accept头（放宽要求，支持EventSource默认行为）
	accept := c.GetHeader("Accept")
	if accept != "" && accept != "text/event-stream" && accept != "*/*" {
		h.respondWithError(c, http.StatusBadRequest, "This endpoint requires SSE support")
		return
	}

	// 设置SSE响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")

	// 处理SSE连接
	if err := h.sseService.HandleConnection(c.Request.Context(), userID, clientID, c.Writer, c.Request); err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "Failed to establish SSE connection: "+err.Error())
		return
	}

	// 保持连接打开
	select {
	case <-c.Request.Context().Done():
		return
	}
}

// GetSSEStats 获取SSE统计信息
func (h *Handler) GetSSEStats(c *gin.Context) {
	stats := h.sseService.GetStats()
	h.respondWithSuccess(c, stats)
}

// SendTestEvent 发送测试事件（仅用于开发和调试）
func (h *Handler) SendTestEvent(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	var req struct {
		Type    string      `json:"type" binding:"required"`
		Message string      `json:"message" binding:"required"`
		Data    interface{} `json:"data,omitempty"`
	}

	if !h.bindJSON(c, &req) {
		return
	}

	// 创建测试事件
	var event *sse.Event
	switch req.Type {
	case "notification":
		event = sse.NewNotificationEvent("测试通知", req.Message, "info", userID)
	case "heartbeat":
		event = sse.NewHeartbeatEvent("")
	default:
		event = sse.NewEvent(sse.EventType(req.Type), req.Data, userID)
	}

	// 发布事件
	if err := h.sseService.PublishEvent(c.Request.Context(), event); err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "Failed to publish event: "+err.Error())
		return
	}

	h.respondWithSuccess(c, gin.H{
		"event_id": event.ID,
		"type":     event.Type,
		"message":  "Event published successfully",
	})
}

// StartSSEService 启动SSE服务
func (h *Handler) StartSSEService() error {
	return h.sseService.Start(nil)
}

// StopSSEService 停止SSE服务
func (h *Handler) StopSSEService() error {
	return h.sseService.Stop()
}
