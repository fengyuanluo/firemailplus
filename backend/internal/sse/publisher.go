package sse

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
)

// EventPublisherImpl 事件发布器实现
type EventPublisherImpl struct {
	connectionManager ConnectionManager
	db                *gorm.DB
	eventFilters      []EventFilter
	stats             *PublisherStats
	mutex             sync.RWMutex
}

// PublisherStats 发布器统计信息
type PublisherStats struct {
	EventsPublished   int64                   `json:"events_published"`
	EventsByType      map[EventType]int64     `json:"events_by_type"`
	EventsByUser      map[uint]int64          `json:"events_by_user"`
	FailedEvents      int64                   `json:"failed_events"`
	LastEventTime     *time.Time              `json:"last_event_time,omitempty"`
	StartTime         time.Time               `json:"start_time"`
	mutex             sync.RWMutex
}

// NewEventPublisher 创建事件发布器
func NewEventPublisher(connectionManager ConnectionManager, db *gorm.DB) *EventPublisherImpl {
	return &EventPublisherImpl{
		connectionManager: connectionManager,
		db:                db,
		eventFilters:      make([]EventFilter, 0),
		stats: &PublisherStats{
			EventsByType: make(map[EventType]int64),
			EventsByUser: make(map[uint]int64),
			StartTime:    time.Now(),
		},
	}
}

// Publish 发布事件
func (p *EventPublisherImpl) Publish(ctx context.Context, event *Event) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	// 应用事件过滤器
	if !p.shouldPublishEvent(event) {
		return nil
	}

	// 转换为SSE格式
	sseData, err := event.ToSSEFormat()
	if err != nil {
		p.incrementFailedEvents()
		return fmt.Errorf("failed to format event: %w", err)
	}

	// 发送给目标用户
	if event.UserID > 0 {
		err = p.connectionManager.SendToUser(event.UserID, sseData)
	} else {
		// 如果没有指定用户，广播给所有用户
		err = p.broadcastToAll(sseData)
	}

	if err != nil {
		p.incrementFailedEvents()
		return fmt.Errorf("failed to send event: %w", err)
	}

	// 更新统计信息
	p.updateStats(event)

	return nil
}

// PublishToUser 发布事件给指定用户
func (p *EventPublisherImpl) PublishToUser(ctx context.Context, userID uint, event *Event) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	// 设置目标用户
	event.UserID = userID

	return p.Publish(ctx, event)
}

// PublishToAccount 发布事件给指定账户的用户
func (p *EventPublisherImpl) PublishToAccount(ctx context.Context, accountID uint, event *Event) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	// 查询账户对应的用户ID
	var account struct {
		UserID uint `gorm:"column:user_id"`
	}

	err := p.db.Table("email_accounts").
		Select("user_id").
		Where("id = ?", accountID).
		First(&account).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("account not found: %d", accountID)
		}
		return fmt.Errorf("failed to query account: %w", err)
	}

	// 设置账户ID和用户ID
	event.AccountID = &accountID
	event.UserID = account.UserID

	return p.Publish(ctx, event)
}

// Broadcast 广播事件给所有连接的用户
func (p *EventPublisherImpl) Broadcast(ctx context.Context, event *Event) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	// 清除用户ID，表示广播
	event.UserID = 0

	return p.Publish(ctx, event)
}

// AddEventFilter 添加事件过滤器
func (p *EventPublisherImpl) AddEventFilter(filter EventFilter) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.eventFilters = append(p.eventFilters, filter)
}

// RemoveEventFilter 移除事件过滤器
func (p *EventPublisherImpl) RemoveEventFilter(filter EventFilter) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for i, f := range p.eventFilters {
		if f == filter {
			p.eventFilters = append(p.eventFilters[:i], p.eventFilters[i+1:]...)
			break
		}
	}
}

// GetStats 获取统计信息
func (p *EventPublisherImpl) GetStats() *PublisherStats {
	p.stats.mutex.RLock()
	defer p.stats.mutex.RUnlock()

	// 创建副本以避免并发访问问题
	stats := &PublisherStats{
		EventsPublished: p.stats.EventsPublished,
		FailedEvents:    p.stats.FailedEvents,
		StartTime:       p.stats.StartTime,
		EventsByType:    make(map[EventType]int64),
		EventsByUser:    make(map[uint]int64),
	}

	// 复制映射
	for k, v := range p.stats.EventsByType {
		stats.EventsByType[k] = v
	}
	for k, v := range p.stats.EventsByUser {
		stats.EventsByUser[k] = v
	}

	if p.stats.LastEventTime != nil {
		lastTime := *p.stats.LastEventTime
		stats.LastEventTime = &lastTime
	}

	return stats
}

// shouldPublishEvent 检查是否应该发布事件
func (p *EventPublisherImpl) shouldPublishEvent(event *Event) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// 应用所有过滤器
	for _, filter := range p.eventFilters {
		if !filter.ShouldProcess(event, event.UserID) {
			return false
		}
	}

	return true
}

// broadcastToAll 广播给所有用户
func (p *EventPublisherImpl) broadcastToAll(data []byte) error {
	_, userConnections := p.connectionManager.GetConnectionCount()
	
	var errors []error
	for userID := range userConnections {
		if err := p.connectionManager.SendToUser(userID, data); err != nil {
			errors = append(errors, fmt.Errorf("failed to send to user %d: %w", userID, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("broadcast failed for %d users", len(errors))
	}

	return nil
}

// updateStats 更新统计信息
func (p *EventPublisherImpl) updateStats(event *Event) {
	p.stats.mutex.Lock()
	defer p.stats.mutex.Unlock()

	atomic.AddInt64(&p.stats.EventsPublished, 1)
	p.stats.EventsByType[event.Type]++
	
	if event.UserID > 0 {
		p.stats.EventsByUser[event.UserID]++
	}

	now := time.Now()
	p.stats.LastEventTime = &now
}

// incrementFailedEvents 增加失败事件计数
func (p *EventPublisherImpl) incrementFailedEvents() {
	atomic.AddInt64(&p.stats.FailedEvents, 1)
}

// UserEventFilter 用户事件过滤器
type UserEventFilter struct {
	allowedUsers map[uint]bool
	mutex        sync.RWMutex
}

// NewUserEventFilter 创建用户事件过滤器
func NewUserEventFilter(allowedUsers []uint) *UserEventFilter {
	filter := &UserEventFilter{
		allowedUsers: make(map[uint]bool),
	}

	for _, userID := range allowedUsers {
		filter.allowedUsers[userID] = true
	}

	return filter
}

// ShouldProcess 判断是否应该处理该事件
func (f *UserEventFilter) ShouldProcess(event *Event, userID uint) bool {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	// 如果没有限制，允许所有用户
	if len(f.allowedUsers) == 0 {
		return true
	}

	// 检查用户是否在允许列表中
	return f.allowedUsers[userID]
}

// AddUser 添加允许的用户
func (f *UserEventFilter) AddUser(userID uint) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.allowedUsers[userID] = true
}

// RemoveUser 移除允许的用户
func (f *UserEventFilter) RemoveUser(userID uint) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	delete(f.allowedUsers, userID)
}

// EventTypeFilter 事件类型过滤器
type EventTypeFilter struct {
	allowedTypes map[EventType]bool
	mutex        sync.RWMutex
}

// NewEventTypeFilter 创建事件类型过滤器
func NewEventTypeFilter(allowedTypes []EventType) *EventTypeFilter {
	filter := &EventTypeFilter{
		allowedTypes: make(map[EventType]bool),
	}

	for _, eventType := range allowedTypes {
		filter.allowedTypes[eventType] = true
	}

	return filter
}

// ShouldProcess 判断是否应该处理该事件
func (f *EventTypeFilter) ShouldProcess(event *Event, userID uint) bool {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	// 如果没有限制，允许所有事件类型
	if len(f.allowedTypes) == 0 {
		return true
	}

	// 检查事件类型是否在允许列表中
	return f.allowedTypes[event.Type]
}

// AddEventType 添加允许的事件类型
func (f *EventTypeFilter) AddEventType(eventType EventType) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.allowedTypes[eventType] = true
}

// RemoveEventType 移除允许的事件类型
func (f *EventTypeFilter) RemoveEventType(eventType EventType) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	delete(f.allowedTypes, eventType)
}
