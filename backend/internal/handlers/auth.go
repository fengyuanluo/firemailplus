package handlers

import (
	"net/http"

	"firemail/internal/auth"

	"github.com/gin-gonic/gin"
)

// Login 用户登录
func (h *Handler) Login(c *gin.Context) {
	var req auth.LoginRequest
	if !h.bindJSON(c, &req) {
		return
	}

	// 执行登录
	response, err := h.authService.Login(&req)
	if err != nil {
		switch err {
		case auth.ErrInvalidCredentials:
			h.respondWithError(c, http.StatusUnauthorized, "Invalid username or password")
		case auth.ErrUserInactive:
			h.respondWithError(c, http.StatusForbidden, "User account is inactive")
		default:
			h.respondWithError(c, http.StatusInternalServerError, "Login failed")
		}
		return
	}

	h.respondWithSuccess(c, response, "Login successful")
}

// Logout 用户登出
func (h *Handler) Logout(c *gin.Context) {
	// 对于JWT，登出通常在客户端处理（删除token）
	// 这里可以实现token黑名单机制
	h.respondWithSuccess(c, nil, "Logout successful")
}

// GetCurrentUser 获取当前用户信息
func (h *Handler) GetCurrentUser(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		h.respondWithError(c, http.StatusNotFound, "User not found")
		return
	}

	h.respondWithSuccess(c, user)
}

// RefreshToken 刷新访问令牌
func (h *Handler) RefreshToken(c *gin.Context) {
	// 从Authorization header获取当前token
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		h.respondWithError(c, http.StatusBadRequest, "Authorization header is required")
		return
	}

	token := auth.ExtractTokenFromHeader(authHeader)
	if token == "" {
		h.respondWithError(c, http.StatusBadRequest, "Invalid authorization header format")
		return
	}

	// 刷新token
	response, err := h.authService.RefreshToken(token)
	if err != nil {
		h.respondWithError(c, http.StatusUnauthorized, "Token refresh failed")
		return
	}

	h.respondWithSuccess(c, response, "Token refreshed successfully")
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// ChangePassword 修改密码
func (h *Handler) ChangePassword(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	var req ChangePasswordRequest
	if !h.bindJSON(c, &req) {
		return
	}

	err := h.authService.ChangePassword(userID, req.OldPassword, req.NewPassword)
	if err != nil {
		switch err {
		case auth.ErrInvalidCredentials:
			h.respondWithError(c, http.StatusBadRequest, "Current password is incorrect")
		case auth.ErrUserNotFound:
			h.respondWithError(c, http.StatusNotFound, "User not found")
		default:
			h.respondWithError(c, http.StatusInternalServerError, "Failed to change password")
		}
		return
	}

	h.respondWithSuccess(c, nil, "Password changed successfully")
}

// UpdateProfileRequest 更新用户资料请求
type UpdateProfileRequest struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

// UpdateProfile 更新用户资料
func (h *Handler) UpdateProfile(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	var req UpdateProfileRequest
	if !h.bindJSON(c, &req) {
		return
	}

	user, err := h.authService.UpdateProfile(userID, req.DisplayName, req.Email)
	if err != nil {
		switch err {
		case auth.ErrUserNotFound:
			h.respondWithError(c, http.StatusNotFound, "User not found")
		default:
			h.respondWithError(c, http.StatusInternalServerError, "Failed to update profile")
		}
		return
	}

	h.respondWithSuccess(c, user, "Profile updated successfully")
}
