package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"firemail/internal/services"

	"github.com/gin-gonic/gin"
)

// GetFolders 获取文件夹列表
func (h *Handler) GetFolders(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	// 从查询参数获取account_id
	accountID := h.parseUintQuery(c, "account_id", 0)
	if accountID == 0 {
		h.respondWithError(c, http.StatusBadRequest, "account_id parameter is required")
		return
	}

	folders, err := h.emailService.GetFolders(c.Request.Context(), userID, accountID)
	if err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "Failed to get folders")
		return
	}

	h.respondWithSuccess(c, folders)
}

// GetFolder 获取单个文件夹
func (h *Handler) GetFolder(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	// 从路径参数获取folder_id
	folderID, exists := h.parseUintParam(c, "id")
	if !exists {
		return
	}

	folder, err := h.emailService.GetFolder(c.Request.Context(), userID, folderID)
	if err != nil {
		if err.Error() == "folder not found" {
			h.respondWithError(c, http.StatusNotFound, "Folder not found")
		} else {
			h.respondWithError(c, http.StatusInternalServerError, "Failed to get folder")
		}
		return
	}

	h.respondWithSuccess(c, folder)
}

// CreateFolder 创建文件夹
func (h *Handler) CreateFolder(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	// 从查询参数获取account_id
	accountID := h.parseUintQuery(c, "account_id", 0)
	if accountID == 0 {
		h.respondWithError(c, http.StatusBadRequest, "account_id parameter is required")
		return
	}

	var req services.CreateFolderRequest
	if !h.bindJSON(c, &req) {
		return
	}

	folder, err := h.emailService.CreateFolder(c.Request.Context(), userID, accountID, &req)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to create folder: "+err.Error())
		return
	}

	h.respondWithCreated(c, folder, "Folder created successfully")
}



// UpdateFolder 更新文件夹
func (h *Handler) UpdateFolder(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	folderID, exists := h.parseUintParam(c, "id")
	if !exists {
		return
	}

	var req services.UpdateFolderRequest
	if !h.bindJSON(c, &req) {
		return
	}

	folder, err := h.emailService.UpdateFolder(c.Request.Context(), userID, folderID, &req)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to update folder: "+err.Error())
		return
	}

	h.respondWithSuccess(c, folder, "Folder updated successfully")
}

// DeleteFolder 删除文件夹
func (h *Handler) DeleteFolder(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	folderID, exists := h.parseUintParam(c, "id")
	if !exists {
		return
	}

	err := h.emailService.DeleteFolder(c.Request.Context(), userID, folderID)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to delete folder: "+err.Error())
		return
	}

	h.respondWithSuccess(c, nil, "Folder deleted successfully")
}

// MarkFolderAsRead 标记文件夹内所有邮件为已读
func (h *Handler) MarkFolderAsRead(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	folderID, exists := h.parseUintParam(c, "id")
	if !exists {
		return
	}

	err := h.emailService.MarkFolderAsRead(c.Request.Context(), userID, folderID)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to mark folder as read: "+err.Error())
		return
	}

	h.respondWithSuccess(c, nil, "Folder marked as read successfully")
}

// SyncFolder 同步指定文件夹
func (h *Handler) SyncFolder(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	folderID, exists := h.parseUintParam(c, "id")
	if !exists {
		return
	}

	// 启动异步同步
	go func() {
		// 为异步操作创建独立的context，避免使用HTTP请求的context
		// HTTP请求的context在响应返回后会被取消，导致异步操作失败
		syncCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if err := h.emailService.SyncSpecificFolder(syncCtx, userID, folderID); err != nil {
			// 记录错误，但不影响响应
			// 可以通过SSE通知前端同步失败
			log.Printf("Failed to sync folder %d for user %d: %v", folderID, userID, err)
		}
	}()

	h.respondWithSuccess(c, nil, "Folder sync started")
}
