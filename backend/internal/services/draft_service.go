package services

import (
	"context"
	"fmt"

	"firemail/internal/models"

	"gorm.io/gorm"
)

// DraftService 草稿服务接口
type DraftService interface {
	// 草稿管理
	CreateDraft(ctx context.Context, userID uint, req *CreateDraftRequest) (*models.Draft, error)
	UpdateDraft(ctx context.Context, userID, draftID uint, req *UpdateDraftRequest) (*models.Draft, error)
	GetDraft(ctx context.Context, userID, draftID uint) (*models.Draft, error)
	ListDrafts(ctx context.Context, userID uint, req *ListDraftsRequest) (*ListDraftsResponse, error)
	DeleteDraft(ctx context.Context, userID, draftID uint) error
	
	// 模板管理
	CreateTemplate(ctx context.Context, userID uint, req *CreateTemplateRequest) (*models.Draft, error)
	UpdateTemplate(ctx context.Context, userID, templateID uint, req *UpdateTemplateRequest) (*models.Draft, error)
	GetTemplate(ctx context.Context, userID, templateID uint) (*models.Draft, error)
	ListTemplates(ctx context.Context, userID uint, req *ListTemplatesRequest) (*ListTemplatesResponse, error)
	DeleteTemplate(ctx context.Context, userID, templateID uint) error
	
	// 转换操作
	ConvertDraftToTemplate(ctx context.Context, userID, draftID uint, templateName string) (*models.Draft, error)
	ConvertTemplateToDraft(ctx context.Context, userID, templateID uint) (*models.Draft, error)
}

// DraftServiceImpl 草稿服务实现
type DraftServiceImpl struct {
	db *gorm.DB
}

// NewDraftService 创建草稿服务
func NewDraftService(db *gorm.DB) DraftService {
	return &DraftServiceImpl{
		db: db,
	}
}

// CreateDraftRequest 创建草稿请求
type CreateDraftRequest struct {
	AccountID     uint                    `json:"account_id" binding:"required"`
	Subject       string                  `json:"subject"`
	To            []models.EmailAddress   `json:"to"`
	CC            []models.EmailAddress   `json:"cc"`
	BCC           []models.EmailAddress   `json:"bcc"`
	TextBody      string                  `json:"text_body"`
	HTMLBody      string                  `json:"html_body"`
	AttachmentIDs []uint                  `json:"attachment_ids"`
	Priority      string                  `json:"priority"`
}

// UpdateDraftRequest 更新草稿请求
type UpdateDraftRequest struct {
	Subject       *string                 `json:"subject"`
	To            *[]models.EmailAddress  `json:"to"`
	CC            *[]models.EmailAddress  `json:"cc"`
	BCC           *[]models.EmailAddress  `json:"bcc"`
	TextBody      *string                 `json:"text_body"`
	HTMLBody      *string                 `json:"html_body"`
	AttachmentIDs *[]uint                 `json:"attachment_ids"`
	Priority      *string                 `json:"priority"`
}

// ListDraftsRequest 列出草稿请求
type ListDraftsRequest struct {
	AccountID *uint  `json:"account_id"`
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
	SortBy    string `json:"sort_by"`    // created_at, updated_at, last_edited_at, subject
	SortOrder string `json:"sort_order"` // asc, desc
}

// ListDraftsResponse 列出草稿响应
type ListDraftsResponse struct {
	Drafts     []*models.Draft `json:"drafts"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}

// CreateTemplateRequest 创建模板请求
type CreateTemplateRequest struct {
	AccountID     uint                    `json:"account_id" binding:"required"`
	TemplateName  string                  `json:"template_name" binding:"required"`
	Subject       string                  `json:"subject"`
	To            []models.EmailAddress   `json:"to"`
	CC            []models.EmailAddress   `json:"cc"`
	BCC           []models.EmailAddress   `json:"bcc"`
	TextBody      string                  `json:"text_body"`
	HTMLBody      string                  `json:"html_body"`
	AttachmentIDs []uint                  `json:"attachment_ids"`
	Priority      string                  `json:"priority"`
}

// UpdateTemplateRequest 更新模板请求
type UpdateTemplateRequest struct {
	TemplateName  *string                 `json:"template_name"`
	Subject       *string                 `json:"subject"`
	To            *[]models.EmailAddress  `json:"to"`
	CC            *[]models.EmailAddress  `json:"cc"`
	BCC           *[]models.EmailAddress  `json:"bcc"`
	TextBody      *string                 `json:"text_body"`
	HTMLBody      *string                 `json:"html_body"`
	AttachmentIDs *[]uint                 `json:"attachment_ids"`
	Priority      *string                 `json:"priority"`
}

// ListTemplatesRequest 列出模板请求
type ListTemplatesRequest struct {
	AccountID *uint  `json:"account_id"`
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
	SortBy    string `json:"sort_by"`    // created_at, updated_at, template_name
	SortOrder string `json:"sort_order"` // asc, desc
}

// ListTemplatesResponse 列出模板响应
type ListTemplatesResponse struct {
	Templates  []*models.Draft `json:"templates"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}

// CreateDraft 创建草稿
func (s *DraftServiceImpl) CreateDraft(ctx context.Context, userID uint, req *CreateDraftRequest) (*models.Draft, error) {
	// 验证账户权限
	if err := s.validateAccountAccess(ctx, req.AccountID, userID); err != nil {
		return nil, err
	}
	
	// 创建草稿
	draft := &models.Draft{
		UserID:    userID,
		AccountID: req.AccountID,
		Subject:   req.Subject,
		TextBody:  req.TextBody,
		HTMLBody:  req.HTMLBody,
		Priority:  req.Priority,
	}
	
	// 设置默认优先级
	if draft.Priority == "" {
		draft.Priority = "normal"
	}
	
	// 设置收件人
	if err := draft.SetToAddresses(req.To); err != nil {
		return nil, fmt.Errorf("failed to set to addresses: %w", err)
	}
	
	if err := draft.SetCCAddresses(req.CC); err != nil {
		return nil, fmt.Errorf("failed to set cc addresses: %w", err)
	}
	
	if err := draft.SetBCCAddresses(req.BCC); err != nil {
		return nil, fmt.Errorf("failed to set bcc addresses: %w", err)
	}
	
	// 设置附件
	if err := draft.SetAttachmentIDs(req.AttachmentIDs); err != nil {
		return nil, fmt.Errorf("failed to set attachment ids: %w", err)
	}
	
	// 设置最后编辑时间
	draft.UpdateLastEditedAt()
	
	// 保存到数据库
	if err := s.db.WithContext(ctx).Create(draft).Error; err != nil {
		return nil, fmt.Errorf("failed to create draft: %w", err)
	}
	
	return draft, nil
}

// UpdateDraft 更新草稿
func (s *DraftServiceImpl) UpdateDraft(ctx context.Context, userID, draftID uint, req *UpdateDraftRequest) (*models.Draft, error) {
	// 获取草稿
	draft, err := s.getDraftWithPermissionCheck(ctx, draftID, userID)
	if err != nil {
		return nil, err
	}
	
	// 确保不是模板
	if draft.IsTemplate {
		return nil, fmt.Errorf("cannot update template as draft")
	}
	
	// 更新字段
	if req.Subject != nil {
		draft.Subject = *req.Subject
	}
	
	if req.TextBody != nil {
		draft.TextBody = *req.TextBody
	}
	
	if req.HTMLBody != nil {
		draft.HTMLBody = *req.HTMLBody
	}
	
	if req.Priority != nil {
		draft.Priority = *req.Priority
	}
	
	if req.To != nil {
		if err := draft.SetToAddresses(*req.To); err != nil {
			return nil, fmt.Errorf("failed to set to addresses: %w", err)
		}
	}
	
	if req.CC != nil {
		if err := draft.SetCCAddresses(*req.CC); err != nil {
			return nil, fmt.Errorf("failed to set cc addresses: %w", err)
		}
	}
	
	if req.BCC != nil {
		if err := draft.SetBCCAddresses(*req.BCC); err != nil {
			return nil, fmt.Errorf("failed to set bcc addresses: %w", err)
		}
	}
	
	if req.AttachmentIDs != nil {
		if err := draft.SetAttachmentIDs(*req.AttachmentIDs); err != nil {
			return nil, fmt.Errorf("failed to set attachment ids: %w", err)
		}
	}
	
	// 更新最后编辑时间
	draft.UpdateLastEditedAt()
	
	// 保存到数据库
	if err := s.db.WithContext(ctx).Save(draft).Error; err != nil {
		return nil, fmt.Errorf("failed to update draft: %w", err)
	}
	
	return draft, nil
}

// GetDraft 获取草稿
func (s *DraftServiceImpl) GetDraft(ctx context.Context, userID, draftID uint) (*models.Draft, error) {
	draft, err := s.getDraftWithPermissionCheck(ctx, draftID, userID)
	if err != nil {
		return nil, err
	}

	// 确保不是模板
	if draft.IsTemplate {
		return nil, fmt.Errorf("not a draft")
	}

	return draft, nil
}

// ListDrafts 列出草稿
func (s *DraftServiceImpl) ListDrafts(ctx context.Context, userID uint, req *ListDraftsRequest) (*ListDraftsResponse, error) {
	// 构建查询
	query := s.db.WithContext(ctx).Model(&models.Draft{}).
		Where("user_id = ? AND is_template = ?", userID, false)

	// 添加过滤条件
	if req.AccountID != nil {
		query = query.Where("account_id = ?", *req.AccountID)
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count drafts: %w", err)
	}

	// 应用分页
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// 应用排序
	sortBy := req.SortBy
	if sortBy == "" {
		sortBy = "last_edited_at"
	}
	sortOrder := req.SortOrder
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	// 获取草稿列表
	var drafts []*models.Draft
	err := query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder)).
		Limit(pageSize).
		Offset(offset).
		Find(&drafts).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list drafts: %w", err)
	}

	// 计算总页数
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &ListDraftsResponse{
		Drafts:     drafts,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// DeleteDraft 删除草稿
func (s *DraftServiceImpl) DeleteDraft(ctx context.Context, userID, draftID uint) error {
	// 获取草稿
	draft, err := s.getDraftWithPermissionCheck(ctx, draftID, userID)
	if err != nil {
		return err
	}

	// 确保不是模板
	if draft.IsTemplate {
		return fmt.Errorf("cannot delete template as draft")
	}

	// 删除草稿
	if err := s.db.WithContext(ctx).Delete(draft).Error; err != nil {
		return fmt.Errorf("failed to delete draft: %w", err)
	}

	return nil
}

// CreateTemplate 创建模板
func (s *DraftServiceImpl) CreateTemplate(ctx context.Context, userID uint, req *CreateTemplateRequest) (*models.Draft, error) {
	// 验证账户权限
	if err := s.validateAccountAccess(ctx, req.AccountID, userID); err != nil {
		return nil, err
	}

	// 创建模板
	template := &models.Draft{
		UserID:       userID,
		AccountID:    req.AccountID,
		Subject:      req.Subject,
		TextBody:     req.TextBody,
		HTMLBody:     req.HTMLBody,
		Priority:     req.Priority,
		IsTemplate:   true,
		TemplateName: req.TemplateName,
	}

	// 设置默认优先级
	if template.Priority == "" {
		template.Priority = "normal"
	}

	// 设置收件人
	if err := template.SetToAddresses(req.To); err != nil {
		return nil, fmt.Errorf("failed to set to addresses: %w", err)
	}

	if err := template.SetCCAddresses(req.CC); err != nil {
		return nil, fmt.Errorf("failed to set cc addresses: %w", err)
	}

	if err := template.SetBCCAddresses(req.BCC); err != nil {
		return nil, fmt.Errorf("failed to set bcc addresses: %w", err)
	}

	// 设置附件
	if err := template.SetAttachmentIDs(req.AttachmentIDs); err != nil {
		return nil, fmt.Errorf("failed to set attachment ids: %w", err)
	}

	// 保存到数据库
	if err := s.db.WithContext(ctx).Create(template).Error; err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	return template, nil
}

// UpdateTemplate 更新模板
func (s *DraftServiceImpl) UpdateTemplate(ctx context.Context, userID, templateID uint, req *UpdateTemplateRequest) (*models.Draft, error) {
	// 获取模板
	template, err := s.getTemplateWithPermissionCheck(ctx, templateID, userID)
	if err != nil {
		return nil, err
	}

	// 更新字段
	if req.TemplateName != nil {
		template.TemplateName = *req.TemplateName
	}

	if req.Subject != nil {
		template.Subject = *req.Subject
	}

	if req.TextBody != nil {
		template.TextBody = *req.TextBody
	}

	if req.HTMLBody != nil {
		template.HTMLBody = *req.HTMLBody
	}

	if req.Priority != nil {
		template.Priority = *req.Priority
	}

	if req.To != nil {
		if err := template.SetToAddresses(*req.To); err != nil {
			return nil, fmt.Errorf("failed to set to addresses: %w", err)
		}
	}

	if req.CC != nil {
		if err := template.SetCCAddresses(*req.CC); err != nil {
			return nil, fmt.Errorf("failed to set cc addresses: %w", err)
		}
	}

	if req.BCC != nil {
		if err := template.SetBCCAddresses(*req.BCC); err != nil {
			return nil, fmt.Errorf("failed to set bcc addresses: %w", err)
		}
	}

	if req.AttachmentIDs != nil {
		if err := template.SetAttachmentIDs(*req.AttachmentIDs); err != nil {
			return nil, fmt.Errorf("failed to set attachment ids: %w", err)
		}
	}

	// 保存到数据库
	if err := s.db.WithContext(ctx).Save(template).Error; err != nil {
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	return template, nil
}

// GetTemplate 获取模板
func (s *DraftServiceImpl) GetTemplate(ctx context.Context, userID, templateID uint) (*models.Draft, error) {
	return s.getTemplateWithPermissionCheck(ctx, templateID, userID)
}

// ListTemplates 列出模板
func (s *DraftServiceImpl) ListTemplates(ctx context.Context, userID uint, req *ListTemplatesRequest) (*ListTemplatesResponse, error) {
	// 构建查询
	query := s.db.WithContext(ctx).Model(&models.Draft{}).
		Where("user_id = ? AND is_template = ?", userID, true)

	// 添加过滤条件
	if req.AccountID != nil {
		query = query.Where("account_id = ?", *req.AccountID)
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count templates: %w", err)
	}

	// 应用分页
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// 应用排序
	sortBy := req.SortBy
	if sortBy == "" {
		sortBy = "template_name"
	}
	sortOrder := req.SortOrder
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "asc"
	}

	// 获取模板列表
	var templates []*models.Draft
	err := query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder)).
		Limit(pageSize).
		Offset(offset).
		Find(&templates).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	// 计算总页数
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &ListTemplatesResponse{
		Templates:  templates,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// DeleteTemplate 删除模板
func (s *DraftServiceImpl) DeleteTemplate(ctx context.Context, userID, templateID uint) error {
	// 获取模板
	template, err := s.getTemplateWithPermissionCheck(ctx, templateID, userID)
	if err != nil {
		return err
	}

	// 删除模板
	if err := s.db.WithContext(ctx).Delete(template).Error; err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	return nil
}

// ConvertDraftToTemplate 将草稿转换为模板
func (s *DraftServiceImpl) ConvertDraftToTemplate(ctx context.Context, userID, draftID uint, templateName string) (*models.Draft, error) {
	// 获取草稿
	draft, err := s.getDraftWithPermissionCheck(ctx, draftID, userID)
	if err != nil {
		return nil, err
	}

	// 确保是草稿
	if draft.IsTemplate {
		return nil, fmt.Errorf("already a template")
	}

	// 转换为模板
	draft.IsTemplate = true
	draft.TemplateName = templateName

	// 保存到数据库
	if err := s.db.WithContext(ctx).Save(draft).Error; err != nil {
		return nil, fmt.Errorf("failed to convert draft to template: %w", err)
	}

	return draft, nil
}

// ConvertTemplateToDraft 将模板转换为草稿
func (s *DraftServiceImpl) ConvertTemplateToDraft(ctx context.Context, userID, templateID uint) (*models.Draft, error) {
	// 获取模板
	template, err := s.getTemplateWithPermissionCheck(ctx, templateID, userID)
	if err != nil {
		return nil, err
	}

	// 创建新的草稿（基于模板）
	draft := &models.Draft{
		UserID:        userID,
		AccountID:     template.AccountID,
		Subject:       template.Subject,
		To:            template.To,
		CC:            template.CC,
		BCC:           template.BCC,
		TextBody:      template.TextBody,
		HTMLBody:      template.HTMLBody,
		AttachmentIDs: template.AttachmentIDs,
		Priority:      template.Priority,
		IsTemplate:    false,
	}

	// 设置最后编辑时间
	draft.UpdateLastEditedAt()

	// 保存到数据库
	if err := s.db.WithContext(ctx).Create(draft).Error; err != nil {
		return nil, fmt.Errorf("failed to convert template to draft: %w", err)
	}

	return draft, nil
}

// validateAccountAccess 验证账户访问权限
func (s *DraftServiceImpl) validateAccountAccess(ctx context.Context, accountID, userID uint) error {
	var count int64
	err := s.db.WithContext(ctx).
		Model(&models.EmailAccount{}).
		Where("id = ? AND user_id = ?", accountID, userID).
		Count(&count).Error

	if err != nil {
		return fmt.Errorf("failed to validate account access: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("account not found or access denied")
	}

	return nil
}

// getDraftWithPermissionCheck 获取草稿并检查权限
func (s *DraftServiceImpl) getDraftWithPermissionCheck(ctx context.Context, draftID, userID uint) (*models.Draft, error) {
	var draft models.Draft
	err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", draftID, userID).
		First(&draft).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("draft not found or access denied")
		}
		return nil, fmt.Errorf("failed to get draft: %w", err)
	}

	return &draft, nil
}

// getTemplateWithPermissionCheck 获取模板并检查权限
func (s *DraftServiceImpl) getTemplateWithPermissionCheck(ctx context.Context, templateID, userID uint) (*models.Draft, error) {
	var template models.Draft
	err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND is_template = ?", templateID, userID, true).
		First(&template).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("template not found or access denied")
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	return &template, nil
}
