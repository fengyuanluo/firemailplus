package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// OAuth2StateService OAuth2状态管理服务接口
type OAuth2StateService interface {
	// GenerateState 生成state参数
	GenerateState(ctx context.Context, userID uint, provider string, metadata map[string]string) (string, error)
	
	// ValidateState 验证state参数
	ValidateState(ctx context.Context, state string) (*OAuth2StateInfo, error)
	
	// ConsumeState 消费state参数（验证后删除）
	ConsumeState(ctx context.Context, state string) (*OAuth2StateInfo, error)
	
	// CleanupExpiredStates 清理过期的state记录
	CleanupExpiredStates(ctx context.Context) error
}

// OAuth2StateInfo OAuth2状态信息
type OAuth2StateInfo struct {
	State     string            `json:"state"`
	UserID    uint              `json:"user_id"`
	Provider  string            `json:"provider"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt time.Time         `json:"created_at"`
	ExpiresAt time.Time         `json:"expires_at"`
}

// OAuth2State OAuth2状态数据模型
type OAuth2State struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	State     string    `gorm:"uniqueIndex;size:128;not null" json:"state"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Provider  string    `gorm:"size:50;not null" json:"provider"`
	Metadata  string    `gorm:"type:text" json:"metadata"` // JSON格式的元数据
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	ExpiresAt time.Time `gorm:"index;not null" json:"expires_at"`
}

// TableName 指定表名
func (OAuth2State) TableName() string {
	return "oauth2_states"
}

// OAuth2StateServiceImpl OAuth2状态管理服务实现
type OAuth2StateServiceImpl struct {
	db             *gorm.DB
	stateExpiry    time.Duration
	cleanupEnabled bool
}

// NewOAuth2StateService 创建OAuth2状态管理服务
func NewOAuth2StateService(db *gorm.DB) OAuth2StateService {
	service := &OAuth2StateServiceImpl{
		db:             db,
		stateExpiry:    10 * time.Minute, // 默认10分钟过期
		cleanupEnabled: true,
	}
	
	// 启动定期清理
	if service.cleanupEnabled {
		go service.startPeriodicCleanup()
	}
	
	return service
}

// GenerateState 生成state参数
func (s *OAuth2StateServiceImpl) GenerateState(ctx context.Context, userID uint, provider string, metadata map[string]string) (string, error) {
	// 生成随机state
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", fmt.Errorf("failed to generate random state: %w", err)
	}
	
	state := hex.EncodeToString(stateBytes)
	
	// 序列化元数据
	metadataJSON := ""
	if len(metadata) > 0 {
		// 简单的键值对序列化，实际项目中应该使用JSON
		for k, v := range metadata {
			if metadataJSON != "" {
				metadataJSON += ","
			}
			metadataJSON += fmt.Sprintf("%s:%s", k, v)
		}
	}
	
	// 创建state记录
	oauth2State := &OAuth2State{
		State:     state,
		UserID:    userID,
		Provider:  provider,
		Metadata:  metadataJSON,
		ExpiresAt: time.Now().Add(s.stateExpiry),
	}
	
	// 保存到数据库
	if err := s.db.WithContext(ctx).Create(oauth2State).Error; err != nil {
		return "", fmt.Errorf("failed to save state: %w", err)
	}
	
	return state, nil
}

// ValidateState 验证state参数
func (s *OAuth2StateServiceImpl) ValidateState(ctx context.Context, state string) (*OAuth2StateInfo, error) {
	if state == "" {
		return nil, fmt.Errorf("state parameter is empty")
	}
	
	var oauth2State OAuth2State
	err := s.db.WithContext(ctx).
		Where("state = ? AND expires_at > ?", state, time.Now()).
		First(&oauth2State).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("invalid or expired state")
		}
		return nil, fmt.Errorf("failed to validate state: %w", err)
	}
	
	// 解析元数据
	metadata := make(map[string]string)
	if oauth2State.Metadata != "" {
		// 简单的键值对解析
		pairs := splitString(oauth2State.Metadata, ",")
		for _, pair := range pairs {
			kv := splitString(pair, ":")
			if len(kv) == 2 {
				metadata[kv[0]] = kv[1]
			}
		}
	}
	
	return &OAuth2StateInfo{
		State:     oauth2State.State,
		UserID:    oauth2State.UserID,
		Provider:  oauth2State.Provider,
		Metadata:  metadata,
		CreatedAt: oauth2State.CreatedAt,
		ExpiresAt: oauth2State.ExpiresAt,
	}, nil
}

// ConsumeState 消费state参数（验证后删除）
func (s *OAuth2StateServiceImpl) ConsumeState(ctx context.Context, state string) (*OAuth2StateInfo, error) {
	// 先验证state
	stateInfo, err := s.ValidateState(ctx, state)
	if err != nil {
		return nil, err
	}
	
	// 删除state记录（防止重复使用）
	if err := s.db.WithContext(ctx).Where("state = ?", state).Delete(&OAuth2State{}).Error; err != nil {
		return nil, fmt.Errorf("failed to consume state: %w", err)
	}
	
	return stateInfo, nil
}

// CleanupExpiredStates 清理过期的state记录
func (s *OAuth2StateServiceImpl) CleanupExpiredStates(ctx context.Context) error {
	result := s.db.WithContext(ctx).Where("expires_at < ?", time.Now()).Delete(&OAuth2State{})
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup expired states: %w", result.Error)
	}
	
	if result.RowsAffected > 0 {
		fmt.Printf("Cleaned up %d expired OAuth2 states\n", result.RowsAffected)
	}
	
	return nil
}

// startPeriodicCleanup 启动定期清理
func (s *OAuth2StateServiceImpl) startPeriodicCleanup() {
	ticker := time.NewTicker(30 * time.Minute) // 每30分钟清理一次
	defer ticker.Stop()
	
	for range ticker.C {
		ctx := context.Background()
		if err := s.CleanupExpiredStates(ctx); err != nil {
			fmt.Printf("Failed to cleanup expired OAuth2 states: %v\n", err)
		}
	}
}

// splitString 简单的字符串分割函数
func splitString(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}

// SetStateExpiry 设置state过期时间
func (s *OAuth2StateServiceImpl) SetStateExpiry(expiry time.Duration) {
	s.stateExpiry = expiry
}

// DisableCleanup 禁用自动清理
func (s *OAuth2StateServiceImpl) DisableCleanup() {
	s.cleanupEnabled = false
}
