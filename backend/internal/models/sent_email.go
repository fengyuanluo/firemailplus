package models

import (
	"encoding/json"
	"time"
)

// SentEmail 已发送邮件模型
type SentEmail struct {
	BaseModel
	
	// 基本信息
	SendID      string `gorm:"uniqueIndex;size:100;not null" json:"send_id"`
	AccountID   uint   `gorm:"index;not null" json:"account_id"`
	MessageID   string `gorm:"size:255;not null" json:"message_id"`
	
	// 邮件内容
	Subject     string `gorm:"size:500;not null" json:"subject"`
	Recipients  string `gorm:"type:text" json:"recipients"` // 逗号分隔的收件人列表
	
	// 发送信息
	SentAt      time.Time `gorm:"index;not null" json:"sent_at"`
	Status      string    `gorm:"size:50;not null;default:'sent'" json:"status"`
	Size        int64     `gorm:"default:0" json:"size"`
	
	// 错误信息
	Error       string `gorm:"type:text" json:"error,omitempty"`
	RetryCount  int    `gorm:"default:0" json:"retry_count"`
	
	// 关联
	Account     EmailAccount `gorm:"foreignKey:AccountID" json:"account,omitempty"`
}

// TableName 返回表名
func (SentEmail) TableName() string {
	return "sent_emails"
}

// EmailTemplate 邮件模板模型
type EmailTemplate struct {
	BaseModel

	// 基本信息
	Name        string `gorm:"size:100;not null" json:"name"`
	Description string `gorm:"size:500" json:"description"`
	UserID      uint   `gorm:"index;not null" json:"user_id"`

	// 模板内容
	Subject     string `gorm:"size:500;not null" json:"subject"`
	TextBody    string `gorm:"type:text" json:"text_body"`
	HTMLBody    string `gorm:"type:text" json:"html_body"`

	// 模板变量和分类
	Variables   string `gorm:"type:text" json:"variables"` // JSON格式的变量定义
	Category    string `gorm:"size:50" json:"category"`
	Tags        string `gorm:"type:text" json:"tags"` // JSON格式的标签列表

	// 状态
	IsActive    bool   `gorm:"default:true" json:"is_active"`
	IsDefault   bool   `gorm:"default:false" json:"is_default"`
	IsShared    bool   `gorm:"default:false" json:"is_shared"`
	IsBuiltIn   bool   `gorm:"default:false" json:"is_built_in"`

	// 使用统计
	UsageCount  int        `gorm:"default:0" json:"usage_count"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`

	// 关联
	User        User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 返回表名
func (EmailTemplate) TableName() string {
	return "email_templates"
}

// TemplateVariable 模板变量结构
type TemplateVariable struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`        // string, number, date, boolean
	Description  string      `json:"description"`
	Required     bool        `json:"required"`
	DefaultValue interface{} `json:"default_value,omitempty"`
}

// GetVariables 获取模板变量列表
func (t *EmailTemplate) GetVariables() ([]TemplateVariable, error) {
	if t.Variables == "" {
		return []TemplateVariable{}, nil
	}

	var variables []TemplateVariable
	err := json.Unmarshal([]byte(t.Variables), &variables)
	return variables, err
}

// SetVariables 设置模板变量列表
func (t *EmailTemplate) SetVariables(variables []TemplateVariable) error {
	data, err := json.Marshal(variables)
	if err != nil {
		return err
	}
	t.Variables = string(data)
	return nil
}

// GetTags 获取标签列表
func (t *EmailTemplate) GetTags() ([]string, error) {
	if t.Tags == "" {
		return []string{}, nil
	}

	var tags []string
	err := json.Unmarshal([]byte(t.Tags), &tags)
	return tags, err
}

// SetTags 设置标签列表
func (t *EmailTemplate) SetTags(tags []string) error {
	data, err := json.Marshal(tags)
	if err != nil {
		return err
	}
	t.Tags = string(data)
	return nil
}

// IncrementUsage 增加使用次数
func (t *EmailTemplate) IncrementUsage() {
	t.UsageCount++
	now := time.Now()
	t.LastUsedAt = &now
}

// IsOwnedBy 检查模板是否属于指定用户
func (t *EmailTemplate) IsOwnedBy(userID uint) bool {
	return t.UserID == userID || t.IsBuiltIn || t.IsShared
}

// CanEdit 检查用户是否可以编辑模板
func (t *EmailTemplate) CanEdit(userID uint) bool {
	return t.UserID == userID && !t.IsBuiltIn
}

// CanDelete 检查用户是否可以删除模板
func (t *EmailTemplate) CanDelete(userID uint) bool {
	return t.UserID == userID && !t.IsBuiltIn
}

// EmailDraft 邮件草稿模型
type EmailDraft struct {
	BaseModel
	
	// 基本信息
	UserID      uint   `gorm:"index;not null" json:"user_id"`
	AccountID   uint   `gorm:"index;not null" json:"account_id"`
	Subject     string `gorm:"size:500" json:"subject"`
	
	// 收件人信息
	ToAddresses  string `gorm:"type:text" json:"to_addresses"`   // JSON格式
	CCAddresses  string `gorm:"type:text" json:"cc_addresses"`   // JSON格式
	BCCAddresses string `gorm:"type:text" json:"bcc_addresses"`  // JSON格式
	ReplyTo      string `gorm:"size:255" json:"reply_to"`
	
	// 邮件内容
	TextBody    string `gorm:"type:text" json:"text_body"`
	HTMLBody    string `gorm:"type:text" json:"html_body"`
	
	// 附件信息
	Attachments string `gorm:"type:text" json:"attachments"` // JSON格式的附件信息
	
	// 其他设置
	Priority    string `gorm:"size:20;default:'normal'" json:"priority"`
	Headers     string `gorm:"type:text" json:"headers"` // JSON格式
	
	// 状态
	IsAutoSaved bool      `gorm:"default:false" json:"is_auto_saved"`
	LastSavedAt time.Time `gorm:"autoUpdateTime" json:"last_saved_at"`
	
	// 关联
	User        User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Account     EmailAccount `gorm:"foreignKey:AccountID" json:"account,omitempty"`
}

// TableName 返回表名
func (EmailDraft) TableName() string {
	return "email_drafts"
}

// SendQueue 发送队列模型
type SendQueue struct {
	BaseModel
	
	// 基本信息
	SendID      string `gorm:"uniqueIndex;size:100;not null" json:"send_id"`
	UserID      uint   `gorm:"index;not null" json:"user_id"`
	AccountID   uint   `gorm:"index;not null" json:"account_id"`
	
	// 邮件内容
	EmailData   string `gorm:"type:text;not null" json:"email_data"` // JSON格式的邮件数据
	
	// 发送设置
	ScheduledAt *time.Time `gorm:"index" json:"scheduled_at,omitempty"` // 计划发送时间
	Priority    int        `gorm:"default:5" json:"priority"`           // 优先级 1-10
	
	// 状态
	Status      string `gorm:"size:50;not null;default:'pending'" json:"status"`
	Attempts    int    `gorm:"default:0" json:"attempts"`
	MaxAttempts int    `gorm:"default:3" json:"max_attempts"`
	
	// 错误信息
	LastError   string     `gorm:"type:text" json:"last_error,omitempty"`
	LastAttempt *time.Time `json:"last_attempt,omitempty"`
	NextAttempt *time.Time `gorm:"index" json:"next_attempt,omitempty"`
	
	// 关联
	User        User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Account     EmailAccount `gorm:"foreignKey:AccountID" json:"account,omitempty"`
}

// TableName 返回表名
func (SendQueue) TableName() string {
	return "send_queue"
}

// EmailQuota 邮件配额模型
type EmailQuota struct {
	BaseModel
	
	// 基本信息
	UserID      uint `gorm:"uniqueIndex;not null" json:"user_id"`
	
	// 配额设置
	DailyLimit    int `gorm:"default:1000" json:"daily_limit"`     // 每日发送限制
	MonthlyLimit  int `gorm:"default:30000" json:"monthly_limit"`  // 每月发送限制
	AttachmentSizeLimit int64 `gorm:"default:26214400" json:"attachment_size_limit"` // 附件大小限制(25MB)
	
	// 使用统计
	DailyUsed     int       `gorm:"default:0" json:"daily_used"`
	MonthlyUsed   int       `gorm:"default:0" json:"monthly_used"`
	LastResetDate time.Time `gorm:"autoCreateTime" json:"last_reset_date"`
	
	// 状态
	IsBlocked     bool   `gorm:"default:false" json:"is_blocked"`
	BlockReason   string `gorm:"size:255" json:"block_reason,omitempty"`
	
	// 关联
	User          User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 返回表名
func (EmailQuota) TableName() string {
	return "email_quotas"
}

// CanSendEmail 检查是否可以发送邮件
func (q *EmailQuota) CanSendEmail() bool {
	if q.IsBlocked {
		return false
	}
	
	if q.DailyUsed >= q.DailyLimit {
		return false
	}
	
	if q.MonthlyUsed >= q.MonthlyLimit {
		return false
	}
	
	return true
}

// IncrementUsage 增加使用量
func (q *EmailQuota) IncrementUsage() {
	q.DailyUsed++
	q.MonthlyUsed++
}

// ResetDailyUsage 重置每日使用量
func (q *EmailQuota) ResetDailyUsage() {
	q.DailyUsed = 0
	q.LastResetDate = time.Now()
}

// ResetMonthlyUsage 重置每月使用量
func (q *EmailQuota) ResetMonthlyUsage() {
	q.MonthlyUsed = 0
	q.LastResetDate = time.Now()
}
