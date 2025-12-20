package handlers

import (
	"net/http"

	"firemail/internal/services"

	"github.com/gin-gonic/gin"
)

// SetDefaultGroupRequest 空体占位
type SetDefaultGroupRequest struct{}

// ReorderEmailGroupsRequest 分组排序请求
type ReorderEmailGroupsRequest struct {
	GroupIDs []uint `json:"group_ids" binding:"required"`
}

// GetEmailGroups 获取分组列表
func (h *Handler) GetEmailGroups(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	groups, err := h.emailService.GetEmailGroups(c.Request.Context(), userID)
	if err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "Failed to load groups: "+err.Error())
		return
	}

	h.respondWithSuccess(c, groups)
}

// CreateEmailGroup 创建分组
func (h *Handler) CreateEmailGroup(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	var req services.CreateEmailGroupRequest
	if !h.bindJSON(c, &req) {
		return
	}

	group, err := h.emailService.CreateEmailGroup(c.Request.Context(), userID, &req)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to create group: "+err.Error())
		return
	}

	h.respondWithCreated(c, group, "Group created successfully")
}

// UpdateEmailGroup 更新分组
func (h *Handler) UpdateEmailGroup(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	groupID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}

	var req services.UpdateEmailGroupRequest
	if !h.bindJSON(c, &req) {
		return
	}

	group, err := h.emailService.UpdateEmailGroup(c.Request.Context(), userID, groupID, &req)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to update group: "+err.Error())
		return
	}

	h.respondWithSuccess(c, group, "Group updated successfully")
}

// DeleteEmailGroup 删除分组并回退账户
func (h *Handler) DeleteEmailGroup(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	groupID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}

	if err := h.emailService.DeleteEmailGroup(c.Request.Context(), userID, groupID); err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to delete group: "+err.Error())
		return
	}

	h.respondWithSuccess(c, nil, "Group deleted successfully")
}

// SetDefaultEmailGroup 设置默认分组
func (h *Handler) SetDefaultEmailGroup(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	groupID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}

	group, err := h.emailService.SetDefaultEmailGroup(c.Request.Context(), userID, groupID)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to set default group: "+err.Error())
		return
	}

	h.respondWithSuccess(c, group, "Default group updated")
}

// ReorderEmailGroups 分组排序
func (h *Handler) ReorderEmailGroups(c *gin.Context) {
	userID, exists := h.getCurrentUserID(c)
	if !exists {
		return
	}

	var req ReorderEmailGroupsRequest
	if !h.bindJSON(c, &req) {
		return
	}

	if len(req.GroupIDs) == 0 {
		h.respondWithError(c, http.StatusBadRequest, "group_ids cannot be empty")
		return
	}

	groups, err := h.emailService.ReorderEmailGroups(c.Request.Context(), userID, req.GroupIDs)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Failed to reorder groups: "+err.Error())
		return
	}

	h.respondWithSuccess(c, groups, "Groups reordered successfully")
}
