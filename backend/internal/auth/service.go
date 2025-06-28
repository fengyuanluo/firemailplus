package auth

import (
	"errors"
	"fmt"
	"time"

	"firemail/internal/cache"
	"firemail/internal/models"

	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserInactive       = errors.New("user account is inactive")
	ErrInvalidToken       = errors.New("invalid token")
)

// Service 认证服务
type Service struct {
	db           *gorm.DB
	jwtManager   *JWTManager
	cacheManager *cache.CacheManager
}

// NewService 创建认证服务
func NewService(db *gorm.DB, jwtManager *JWTManager) *Service {
	return &Service{
		db:           db,
		jwtManager:   jwtManager,
		cacheManager: cache.GlobalCacheManager,
	}
}

// LoginRequest 登录请求结构
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应结构
type LoginResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      *models.User `json:"user"`
}

// Login 用户登录
func (s *Service) Login(req *LoginRequest) (*LoginResponse, error) {
	// 查找用户
	var user models.User
	if err := s.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// 检查用户是否激活
	if !user.IsActive {
		return nil, ErrUserInactive
	}

	// 验证密码
	if !user.CheckPassword(req.Password) {
		return nil, ErrInvalidCredentials
	}

	// 生成JWT token
	token, err := s.jwtManager.GenerateToken(&user)
	if err != nil {
		return nil, err
	}

	// 更新登录信息
	now := time.Now()
	user.LastLoginAt = &now
	user.LoginCount++
	s.db.Save(&user)

	// 清除密码字段
	user.Password = ""

	return &LoginResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(s.jwtManager.expiry),
		User:      &user,
	}, nil
}

// ValidateToken 验证token
func (s *Service) ValidateToken(tokenString string) (*models.User, error) {
	// 生成缓存键
	cacheKey := fmt.Sprintf("token:%s", tokenString)

	// 尝试从缓存获取用户信息
	if cached, found := s.cacheManager.AuthCache().Get(cacheKey); found {
		if user, ok := cached.(*models.User); ok {
			return user, nil
		}
	}

	claims, err := s.jwtManager.ValidateToken(tokenString)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// 从数据库获取最新的用户信息
	var user models.User
	if err := s.db.Where("id = ?", claims.UserID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// 检查用户是否仍然激活
	if !user.IsActive {
		return nil, ErrUserInactive
	}

	// 清除密码字段
	user.Password = ""

	// 缓存用户信息（缓存15分钟）
	s.cacheManager.AuthCache().Set(cacheKey, &user, 15*time.Minute)

	return &user, nil
}

// RefreshToken 刷新token
func (s *Service) RefreshToken(tokenString string) (*LoginResponse, error) {
	// 验证当前token
	user, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	// 生成新token
	newToken, err := s.jwtManager.RefreshToken(tokenString)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token:     newToken,
		ExpiresAt: time.Now().Add(s.jwtManager.expiry),
		User:      user,
	}, nil
}

// GetUserByID 根据ID获取用户
func (s *Service) GetUserByID(userID uint) (*models.User, error) {
	var user models.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// 清除密码字段
	user.Password = ""

	return &user, nil
}

// ChangePassword 修改密码
func (s *Service) ChangePassword(userID uint, oldPassword, newPassword string) error {
	var user models.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	// 验证旧密码
	if !user.CheckPassword(oldPassword) {
		return ErrInvalidCredentials
	}

	// 设置新密码
	if err := user.SetPassword(newPassword); err != nil {
		return err
	}

	// 保存到数据库
	return s.db.Save(&user).Error
}

// UpdateProfile 更新用户资料
func (s *Service) UpdateProfile(userID uint, displayName, email string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// 更新字段
	if displayName != "" {
		user.DisplayName = displayName
	}
	if email != "" {
		user.Email = email
	}

	// 保存到数据库
	if err := s.db.Save(&user).Error; err != nil {
		return nil, err
	}

	// 清除密码字段
	user.Password = ""

	return &user, nil
}
