package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	"firemail/internal/config"

	"github.com/gin-gonic/gin"
)

// ErrorResponse 统一错误响应格式
type ErrorResponse struct {
	Success   bool        `json:"success"`
	Error     string      `json:"error"`
	Message   string      `json:"message,omitempty"`
	Details   interface{} `json:"details,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	Code      string      `json:"code,omitempty"`
}

// ErrorHandler 错误处理中间件
func ErrorHandler() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			handlePanicError(c, err)
		} else if err, ok := recovered.(error); ok {
			handlePanicError(c, err.Error())
		} else {
			handlePanicError(c, fmt.Sprintf("Unknown error: %v", recovered))
		}
	})
}

// handlePanicError 处理panic错误
func handlePanicError(c *gin.Context, err string) {
	// 记录错误日志
	log.Printf("Panic recovered: %s", err)
	
	// 在开发模式下记录堆栈信息
	if config.Env.IsDevelopmentMode() {
		log.Printf("Stack trace: %s", debug.Stack())
	}
	
	// 返回统一错误响应
	response := ErrorResponse{
		Success: false,
		Error:   "Internal Server Error",
		Message: "An unexpected error occurred",
		Code:    "INTERNAL_ERROR",
	}
	
	// 在开发模式下返回详细错误信息
	if config.Env.IsDevelopmentMode() {
		response.Details = err
	}
	
	c.JSON(http.StatusInternalServerError, response)
	c.Abort()
}

// HandleError 处理业务错误
func HandleError(c *gin.Context, err error, statusCode int) {
	if err == nil {
		return
	}
	
	// 记录错误日志
	log.Printf("Business error: %v", err)
	
	response := ErrorResponse{
		Success: false,
		Error:   getErrorMessage(statusCode),
		Message: err.Error(),
		Code:    getErrorCode(statusCode),
	}
	
	// 在开发模式下添加更多调试信息
	if config.Env.IsDevelopmentMode() {
		response.Details = map[string]interface{}{
			"error_type": fmt.Sprintf("%T", err),
			"stack":      string(debug.Stack()),
		}
	}
	
	c.JSON(statusCode, response)
	c.Abort()
}

// HandleValidationError 处理验证错误
func HandleValidationError(c *gin.Context, field string, message string) {
	response := ErrorResponse{
		Success: false,
		Error:   "Validation Error",
		Message: fmt.Sprintf("Validation failed for field '%s': %s", field, message),
		Code:    "VALIDATION_ERROR",
		Details: map[string]string{
			"field":   field,
			"message": message,
		},
	}
	
	c.JSON(http.StatusBadRequest, response)
	c.Abort()
}

// HandleNotFoundError 处理资源不存在错误
func HandleNotFoundError(c *gin.Context, resource string, id interface{}) {
	response := ErrorResponse{
		Success: false,
		Error:   "Resource Not Found",
		Message: fmt.Sprintf("%s with ID '%v' not found", resource, id),
		Code:    "NOT_FOUND",
		Details: map[string]interface{}{
			"resource": resource,
			"id":       id,
		},
	}
	
	c.JSON(http.StatusNotFound, response)
	c.Abort()
}

// HandleUnauthorizedError 处理未授权错误
func HandleUnauthorizedError(c *gin.Context, message string) {
	response := ErrorResponse{
		Success: false,
		Error:   "Unauthorized",
		Message: message,
		Code:    "UNAUTHORIZED",
	}
	
	c.JSON(http.StatusUnauthorized, response)
	c.Abort()
}

// HandleForbiddenError 处理禁止访问错误
func HandleForbiddenError(c *gin.Context, message string) {
	response := ErrorResponse{
		Success: false,
		Error:   "Forbidden",
		Message: message,
		Code:    "FORBIDDEN",
	}
	
	c.JSON(http.StatusForbidden, response)
	c.Abort()
}

// HandleServiceUnavailableError 处理服务不可用错误
func HandleServiceUnavailableError(c *gin.Context, service string, reason string) {
	response := ErrorResponse{
		Success: false,
		Error:   "Service Unavailable",
		Message: fmt.Sprintf("Service '%s' is currently unavailable: %s", service, reason),
		Code:    "SERVICE_UNAVAILABLE",
		Details: map[string]string{
			"service": service,
			"reason":  reason,
		},
	}
	
	c.JSON(http.StatusServiceUnavailable, response)
	c.Abort()
}

// HandleTestModeError 处理测试模式错误
func HandleTestModeError(c *gin.Context, operation string) {
	response := ErrorResponse{
		Success: false,
		Error:   "Test Mode",
		Message: fmt.Sprintf("Operation '%s' is not available in test mode", operation),
		Code:    "TEST_MODE",
		Details: map[string]interface{}{
			"operation":    operation,
			"test_mode":    true,
			"mock_enabled": config.Env.ShouldMockEmailProviders(),
		},
	}
	
	c.JSON(http.StatusServiceUnavailable, response)
	c.Abort()
}

// getErrorMessage 根据状态码获取错误消息
func getErrorMessage(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return "Bad Request"
	case http.StatusUnauthorized:
		return "Unauthorized"
	case http.StatusForbidden:
		return "Forbidden"
	case http.StatusNotFound:
		return "Not Found"
	case http.StatusMethodNotAllowed:
		return "Method Not Allowed"
	case http.StatusConflict:
		return "Conflict"
	case http.StatusUnprocessableEntity:
		return "Unprocessable Entity"
	case http.StatusInternalServerError:
		return "Internal Server Error"
	case http.StatusServiceUnavailable:
		return "Service Unavailable"
	default:
		return "Unknown Error"
	}
}

// getErrorCode 根据状态码获取错误代码
func getErrorCode(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return "BAD_REQUEST"
	case http.StatusUnauthorized:
		return "UNAUTHORIZED"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusMethodNotAllowed:
		return "METHOD_NOT_ALLOWED"
	case http.StatusConflict:
		return "CONFLICT"
	case http.StatusUnprocessableEntity:
		return "UNPROCESSABLE_ENTITY"
	case http.StatusInternalServerError:
		return "INTERNAL_SERVER_ERROR"
	case http.StatusServiceUnavailable:
		return "SERVICE_UNAVAILABLE"
	default:
		return "UNKNOWN_ERROR"
	}
}

// IsTestModeEnabled 检查是否启用测试模式
func IsTestModeEnabled() bool {
	return config.Env.IsTestMode()
}

// ShouldMockServices 检查是否应该模拟服务
func ShouldMockServices() bool {
	return config.Env.ShouldMockEmailProviders()
}
