package services

import (
	"strings"
)

// isUniqueConstraintError 检查是否为唯一约束错误
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	errStrLower := strings.ToLower(errStr)

	// 检查常见的唯一约束错误关键词
	uniqueKeywords := []string{
		"unique constraint",
		"duplicate key",
		"duplicate entry",
		"unique violation",
		"constraint violation",
		"unique index",
		"duplicate",
		"unique",
	}

	for _, keyword := range uniqueKeywords {
		if strings.Contains(errStrLower, keyword) {
			return true
		}
	}

	return false
}
