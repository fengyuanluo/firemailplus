package services

import (
	"context"
	"fmt"
	"text/template"
	"bytes"
	htmlTemplate "html/template"

	"firemail/internal/models"

	"gorm.io/gorm"
)

// EmailTemplateService 邮件模板服务接口
type EmailTemplateService interface {
	// CreateTemplate 创建模板
	CreateTemplate(ctx context.Context, userID uint, req *CreateEmailTemplateRequest) (*models.EmailTemplate, error)

	// UpdateTemplate 更新模板
	UpdateTemplate(ctx context.Context, userID, templateID uint, req *UpdateEmailTemplateRequest) (*models.EmailTemplate, error)

	// GetTemplate 获取模板
	GetTemplate(ctx context.Context, userID, templateID uint) (*models.EmailTemplate, error)

	// ListTemplates 列出模板
	ListTemplates(ctx context.Context, userID uint, req *ListEmailTemplatesRequest) (*ListEmailTemplatesResponse, error)

	// DeleteTemplate 删除模板
	DeleteTemplate(ctx context.Context, userID, templateID uint) error

	// ProcessTemplate 处理模板，替换变量
	ProcessTemplate(ctx context.Context, templateID uint, data map[string]interface{}) (*ProcessedTemplate, error)

	// GetBuiltInTemplates 获取内置模板
	GetBuiltInTemplates(ctx context.Context) ([]*models.EmailTemplate, error)
}

// CreateEmailTemplateRequest 创建邮件模板请求
type CreateEmailTemplateRequest struct {
	Name        string                       `json:"name" binding:"required"`
	Description string                       `json:"description"`
	Subject     string                       `json:"subject" binding:"required"`
	TextBody    string                       `json:"text_body"`
	HTMLBody    string                       `json:"html_body"`
	Variables   []models.TemplateVariable    `json:"variables"`
	Category    string                       `json:"category"`
	Tags        []string                     `json:"tags"`
	IsShared    bool                         `json:"is_shared"`
}

// UpdateEmailTemplateRequest 更新邮件模板请求
type UpdateEmailTemplateRequest struct {
	Name        *string                      `json:"name"`
	Description *string                      `json:"description"`
	Subject     *string                      `json:"subject"`
	TextBody    *string                      `json:"text_body"`
	HTMLBody    *string                      `json:"html_body"`
	Variables   []models.TemplateVariable    `json:"variables"`
	Category    *string                      `json:"category"`
	Tags        []string                     `json:"tags"`
	IsActive    *bool                        `json:"is_active"`
	IsShared    *bool                        `json:"is_shared"`
}

// ListEmailTemplatesRequest 列出邮件模板请求
type ListEmailTemplatesRequest struct {
	Category      string `form:"category"`
	Tag           string `form:"tag"`
	IsActive      *bool  `form:"is_active"`
	IsShared      *bool  `form:"is_shared"`
	IncludeBuiltIn bool  `form:"include_built_in"`
	Search        string `form:"search"`
	Page          int    `form:"page"`
	PageSize      int    `form:"page_size"`
	SortBy        string `form:"sort_by"`
	SortOrder     string `form:"sort_order"`
}

// ListEmailTemplatesResponse 列出邮件模板响应
type ListEmailTemplatesResponse struct {
	Templates  []*models.EmailTemplate `json:"templates"`
	Total      int64                   `json:"total"`
	Page       int                     `json:"page"`
	PageSize   int                     `json:"page_size"`
	TotalPages int                     `json:"total_pages"`
}

// ProcessedTemplate 处理后的模板
type ProcessedTemplate struct {
	Subject  string `json:"subject"`
	TextBody string `json:"text_body"`
	HTMLBody string `json:"html_body"`
}

// EmailTemplateServiceImpl 邮件模板服务实现
type EmailTemplateServiceImpl struct {
	db *gorm.DB
}

// NewEmailTemplateService 创建邮件模板服务
func NewEmailTemplateService(db *gorm.DB) EmailTemplateService {
	return &EmailTemplateServiceImpl{
		db: db,
	}
}

// CreateTemplate 创建模板
func (s *EmailTemplateServiceImpl) CreateTemplate(ctx context.Context, userID uint, req *CreateEmailTemplateRequest) (*models.EmailTemplate, error) {
	// 验证请求
	if req.Name == "" {
		return nil, fmt.Errorf("template name is required")
	}
	
	if req.Subject == "" {
		return nil, fmt.Errorf("template subject is required")
	}
	
	if req.TextBody == "" && req.HTMLBody == "" {
		return nil, fmt.Errorf("template body is required")
	}
	
	// 检查模板名称是否已存在
	var existingTemplate models.EmailTemplate
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND name = ? AND deleted_at IS NULL", userID, req.Name).
		First(&existingTemplate).Error
	
	if err == nil {
		return nil, fmt.Errorf("template with name '%s' already exists", req.Name)
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to check template name: %w", err)
	}
	
	// 创建模板
	template := &models.EmailTemplate{
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Subject:     req.Subject,
		TextBody:    req.TextBody,
		HTMLBody:    req.HTMLBody,
		Category:    req.Category,
		IsActive:    true,
		IsShared:    req.IsShared,
		IsBuiltIn:   false,
		UsageCount:  0,
	}
	
	// 设置变量
	if len(req.Variables) > 0 {
		if err := template.SetVariables(req.Variables); err != nil {
			return nil, fmt.Errorf("failed to set template variables: %w", err)
		}
	}
	
	// 设置标签
	if len(req.Tags) > 0 {
		if err := template.SetTags(req.Tags); err != nil {
			return nil, fmt.Errorf("failed to set template tags: %w", err)
		}
	}
	
	// 保存到数据库
	if err := s.db.WithContext(ctx).Create(template).Error; err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}
	
	return template, nil
}

// UpdateTemplate 更新模板
func (s *EmailTemplateServiceImpl) UpdateTemplate(ctx context.Context, userID, templateID uint, req *UpdateEmailTemplateRequest) (*models.EmailTemplate, error) {
	// 查找模板
	var template models.EmailTemplate
	err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", templateID).
		First(&template).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("template not found")
		}
		return nil, fmt.Errorf("failed to find template: %w", err)
	}

	// 检查权限
	if !template.CanEdit(userID) {
		return nil, fmt.Errorf("permission denied: cannot edit this template")
	}

	// 更新字段
	if req.Name != nil {
		// 检查名称是否已存在
		if *req.Name != template.Name {
			var existingTemplate models.EmailTemplate
			err := s.db.WithContext(ctx).
				Where("user_id = ? AND name = ? AND id != ? AND deleted_at IS NULL", userID, *req.Name, templateID).
				First(&existingTemplate).Error

			if err == nil {
				return nil, fmt.Errorf("template with name '%s' already exists", *req.Name)
			} else if err != gorm.ErrRecordNotFound {
				return nil, fmt.Errorf("failed to check template name: %w", err)
			}
		}
		template.Name = *req.Name
	}

	if req.Description != nil {
		template.Description = *req.Description
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

	if req.Category != nil {
		template.Category = *req.Category
	}

	if req.IsActive != nil {
		template.IsActive = *req.IsActive
	}

	if req.IsShared != nil {
		template.IsShared = *req.IsShared
	}

	// 更新变量
	if req.Variables != nil {
		if err := template.SetVariables(req.Variables); err != nil {
			return nil, fmt.Errorf("failed to set template variables: %w", err)
		}
	}

	// 更新标签
	if req.Tags != nil {
		if err := template.SetTags(req.Tags); err != nil {
			return nil, fmt.Errorf("failed to set template tags: %w", err)
		}
	}

	// 保存更新
	if err := s.db.WithContext(ctx).Save(&template).Error; err != nil {
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	return &template, nil
}

// GetTemplate 获取模板
func (s *EmailTemplateServiceImpl) GetTemplate(ctx context.Context, userID, templateID uint) (*models.EmailTemplate, error) {
	var template models.EmailTemplate
	err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", templateID).
		First(&template).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("template not found")
		}
		return nil, fmt.Errorf("failed to find template: %w", err)
	}

	// 检查权限
	if !template.IsOwnedBy(userID) {
		return nil, fmt.Errorf("permission denied: cannot access this template")
	}

	return &template, nil
}

// DeleteTemplate 删除模板
func (s *EmailTemplateServiceImpl) DeleteTemplate(ctx context.Context, userID, templateID uint) error {
	// 查找模板
	var template models.EmailTemplate
	err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", templateID).
		First(&template).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("template not found")
		}
		return fmt.Errorf("failed to find template: %w", err)
	}

	// 检查权限
	if !template.CanDelete(userID) {
		return fmt.Errorf("permission denied: cannot delete this template")
	}

	// 软删除
	if err := s.db.WithContext(ctx).Delete(&template).Error; err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	return nil
}

// ListTemplates 列出模板
func (s *EmailTemplateServiceImpl) ListTemplates(ctx context.Context, userID uint, req *ListEmailTemplatesRequest) (*ListEmailTemplatesResponse, error) {
	// 构建查询
	query := s.db.WithContext(ctx).Model(&models.EmailTemplate{}).
		Where("deleted_at IS NULL")

	// 权限过滤：用户自己的模板 + 共享模板 + 内置模板
	query = query.Where("user_id = ? OR is_shared = ? OR is_built_in = ?", userID, true, true)

	// 应用过滤条件
	if req.Category != "" {
		query = query.Where("category = ?", req.Category)
	}

	if req.Tag != "" {
		query = query.Where("tags LIKE ?", "%\""+req.Tag+"\"%")
	}

	if req.IsActive != nil {
		query = query.Where("is_active = ?", *req.IsActive)
	}

	if req.IsShared != nil {
		query = query.Where("is_shared = ?", *req.IsShared)
	}

	if !req.IncludeBuiltIn {
		query = query.Where("is_built_in = ?", false)
	}

	if req.Search != "" {
		searchTerm := "%" + req.Search + "%"
		query = query.Where("(name LIKE ? OR description LIKE ? OR subject LIKE ?)",
			searchTerm, searchTerm, searchTerm)
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count templates: %w", err)
	}

	// 设置默认值
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 排序
	sortBy := req.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortOrder := req.SortOrder
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	// 分页查询
	var templates []*models.EmailTemplate
	offset := (page - 1) * pageSize
	err := query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder)).
		Limit(pageSize).
		Offset(offset).
		Find(&templates).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	// 计算总页数
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &ListEmailTemplatesResponse{
		Templates:  templates,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// ProcessTemplate 处理模板，替换变量
func (s *EmailTemplateServiceImpl) ProcessTemplate(ctx context.Context, templateID uint, data map[string]interface{}) (*ProcessedTemplate, error) {
	// 获取模板
	var tmpl models.EmailTemplate
	err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", templateID).
		First(&tmpl).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("template not found")
		}
		return nil, fmt.Errorf("failed to find template: %w", err)
	}

	// 处理主题
	subject, err := s.processTemplateText(tmpl.Subject, data)
	if err != nil {
		return nil, fmt.Errorf("failed to process subject: %w", err)
	}

	// 处理文本正文
	textBody, err := s.processTemplateText(tmpl.TextBody, data)
	if err != nil {
		return nil, fmt.Errorf("failed to process text body: %w", err)
	}

	// 处理HTML正文
	htmlBody, err := s.processTemplateHTML(tmpl.HTMLBody, data)
	if err != nil {
		return nil, fmt.Errorf("failed to process HTML body: %w", err)
	}

	// 增加使用次数
	tmpl.IncrementUsage()
	s.db.WithContext(ctx).Save(&tmpl)

	return &ProcessedTemplate{
		Subject:  subject,
		TextBody: textBody,
		HTMLBody: htmlBody,
	}, nil
}

// GetBuiltInTemplates 获取内置模板
func (s *EmailTemplateServiceImpl) GetBuiltInTemplates(ctx context.Context) ([]*models.EmailTemplate, error) {
	var templates []*models.EmailTemplate
	err := s.db.WithContext(ctx).
		Where("is_built_in = ? AND deleted_at IS NULL", true).
		Order("category ASC, name ASC").
		Find(&templates).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get built-in templates: %w", err)
	}

	return templates, nil
}

// processTemplateText 处理文本模板
func (s *EmailTemplateServiceImpl) processTemplateText(templateText string, data map[string]interface{}) (string, error) {
	if templateText == "" {
		return "", nil
	}

	tmpl, err := template.New("text").Parse(templateText)
	if err != nil {
		return "", fmt.Errorf("failed to parse text template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute text template: %w", err)
	}

	return buf.String(), nil
}

// processTemplateHTML 处理HTML模板
func (s *EmailTemplateServiceImpl) processTemplateHTML(templateHTML string, data map[string]interface{}) (string, error) {
	if templateHTML == "" {
		return "", nil
	}

	tmpl, err := htmlTemplate.New("html").Parse(templateHTML)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute HTML template: %w", err)
	}

	return buf.String(), nil
}
