package handlers

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// CreateBackup 创建备份
func (h *Handler) CreateBackup(c *gin.Context) {
	backup, err := h.backupService.CreateBackup(c.Request.Context())
	if err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "Failed to create backup: "+err.Error())
		return
	}

	h.respondWithSuccess(c, backup, "Backup created successfully")
}

// ListBackups 列出所有备份
func (h *Handler) ListBackups(c *gin.Context) {
	backups, err := h.backupService.ListBackups(c.Request.Context())
	if err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "Failed to list backups: "+err.Error())
		return
	}

	h.respondWithSuccess(c, gin.H{
		"backups": backups,
		"count":   len(backups),
	}, "Backups retrieved successfully")
}

// RestoreBackup 恢复备份
func (h *Handler) RestoreBackup(c *gin.Context) {
	var req struct {
		BackupPath string `json:"backup_path" binding:"required"`
	}

	if !h.bindJSON(c, &req) {
		return
	}

	// 验证备份路径安全性
	if !filepath.IsAbs(req.BackupPath) {
		h.respondWithError(c, http.StatusBadRequest, "Backup path must be absolute")
		return
	}

	err := h.backupService.RestoreBackup(c.Request.Context(), req.BackupPath)
	if err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "Failed to restore backup: "+err.Error())
		return
	}

	h.respondWithSuccess(c, nil, "Backup restored successfully. Please restart the application.")
}

// DeleteBackup 删除备份
func (h *Handler) DeleteBackup(c *gin.Context) {
	var req struct {
		BackupPath string `json:"backup_path" binding:"required"`
	}

	if !h.bindJSON(c, &req) {
		return
	}

	// 验证备份路径安全性
	if !filepath.IsAbs(req.BackupPath) {
		h.respondWithError(c, http.StatusBadRequest, "Backup path must be absolute")
		return
	}

	err := h.backupService.DeleteBackup(c.Request.Context(), req.BackupPath)
	if err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "Failed to delete backup: "+err.Error())
		return
	}

	h.respondWithSuccess(c, nil, "Backup deleted successfully")
}

// ValidateBackup 验证备份文件
func (h *Handler) ValidateBackup(c *gin.Context) {
	var req struct {
		BackupPath string `json:"backup_path" binding:"required"`
	}

	if !h.bindJSON(c, &req) {
		return
	}

	// 验证备份路径安全性
	if !filepath.IsAbs(req.BackupPath) {
		h.respondWithError(c, http.StatusBadRequest, "Backup path must be absolute")
		return
	}

	err := h.backupService.ValidateBackup(c.Request.Context(), req.BackupPath)
	if err != nil {
		h.respondWithSuccess(c, gin.H{
			"is_valid": false,
			"error":    err.Error(),
		}, "Backup validation completed")
		return
	}

	h.respondWithSuccess(c, gin.H{
		"is_valid": true,
	}, "Backup is valid")
}

// CleanupOldBackups 清理过期备份
func (h *Handler) CleanupOldBackups(c *gin.Context) {
	err := h.backupService.CleanupOldBackups(c.Request.Context())
	if err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "Failed to cleanup old backups: "+err.Error())
		return
	}

	h.respondWithSuccess(c, nil, "Old backups cleaned up successfully")
}
