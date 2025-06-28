package middleware

import (
	"log"
	"net/http"

	"firemail/internal/auth"
	"firemail/internal/models"

	"github.com/gin-gonic/gin"
)

// 全局认证服务实例（用于向后兼容）
var globalAuthService AuthService

// AuthService 认证服务接口
type AuthService interface {
	ValidateToken(tokenString string) (*models.User, error)
}

// SetGlobalAuthService 设置全局认证服务（用于向后兼容）
func SetGlobalAuthService(authService AuthService) {
	globalAuthService = authService
}

// AuthRequired 认证中间件（向后兼容版本）
func AuthRequired() gin.HandlerFunc {
	log.Printf("AuthRequired: Creating middleware, globalAuthService is nil: %t", globalAuthService == nil)
	if globalAuthService == nil {
		panic("Global auth service not set. Call SetGlobalAuthService() first.")
	}
	return AuthRequiredWithService(globalAuthService)
}

// OptionalAuth 可选认证中间件（向后兼容版本）
func OptionalAuth() gin.HandlerFunc {
	if globalAuthService == nil {
		panic("Global auth service not set. Call SetGlobalAuthService() first.")
	}
	return OptionalAuthWithService(globalAuthService)
}

// AuthRequiredWithService 认证中间件（带服务参数）
func AuthRequiredWithService(authService AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("AuthRequiredWithService: Processing request %s %s", c.Request.Method, c.Request.URL.Path)

		// 从header中获取token
		authHeader := c.GetHeader("Authorization")
		log.Printf("AuthRequiredWithService: Authorization header: %s", authHeader[:min(50, len(authHeader))])

		if authHeader == "" {
			log.Printf("AuthRequiredWithService: No authorization header")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header is required",
			})
			c.Abort()
			return
		}

		// 提取token
		token := auth.ExtractTokenFromHeader(authHeader)
		log.Printf("AuthRequiredWithService: Extracted token: %s", token[:min(50, len(token))])

		if token == "" {
			log.Printf("AuthRequiredWithService: Failed to extract token")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		// 验证token
		user, err := authService.ValidateToken(token)
		if err != nil {
			// 添加调试日志
			tokenPreview := token
			if len(token) > 50 {
				tokenPreview = token[:50] + "..."
			}
			log.Printf("Token validation failed: %v, token: %s", err, tokenPreview)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// 将用户信息存储到context中
		c.Set("user", user)
		c.Set("userID", user.ID)
		c.Set("username", user.Username)
		c.Set("role", user.Role)

		c.Next()
	}
}

// OptionalAuthWithService 可选认证中间件（带服务参数）
func OptionalAuthWithService(authService AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从header中获取token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		// 提取token
		token := auth.ExtractTokenFromHeader(authHeader)
		if token == "" {
			c.Next()
			return
		}

		// 验证token
		user, err := authService.ValidateToken(token)
		if err != nil {
			c.Next()
			return
		}

		// 将用户信息存储到context中
		c.Set("user", user)
		c.Set("userID", user.ID)
		c.Set("username", user.Username)
		c.Set("role", user.Role)

		c.Next()
	}
}

// RoleRequired 角色权限中间件
func RoleRequired(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户角色
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User role not found",
			})
			c.Abort()
			return
		}

		userRole := role.(string)

		// 检查角色权限
		hasPermission := false
		for _, requiredRole := range requiredRoles {
			if userRole == requiredRole {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetCurrentUser 从context中获取当前用户
func GetCurrentUser(c *gin.Context) (*models.User, bool) {
	user, exists := c.Get("user")
	if !exists {
		return nil, false
	}
	return user.(*models.User), true
}

// GetCurrentUserID 从context中获取当前用户ID
func GetCurrentUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		return 0, false
	}
	return userID.(uint), true
}

// GetUserID 从context中获取当前用户ID（简化版本，用于handlers）
func GetUserID(c *gin.Context) uint {
	userID, exists := c.Get("userID")
	if !exists {
		// 如果没有用户ID，返回0（这种情况下应该在认证中间件中被拦截）
		return 0
	}
	return userID.(uint)
}

// AdminRequired 管理员权限中间件
func AdminRequired() gin.HandlerFunc {
	return RoleRequired("admin")
}
