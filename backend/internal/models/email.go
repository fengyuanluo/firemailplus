package models

import (
	"encoding/json"
	"time"
)

// Email 邮件模型
type Email struct {
	BaseModel
	AccountID uint   `gorm:"not null;index" json:"account_id"`
	FolderID  *uint  `gorm:"index" json:"folder_id,omitempty"`
	MessageID string `gorm:"not null;size:255;index" json:"message_id"` // 邮件唯一标识
	UID       uint32 `gorm:"not null;index" json:"uid"`                 // IMAP UID

	// 邮件头信息
	Subject string    `gorm:"size:500" json:"subject"`
	From    string    `gorm:"column:from_address;size:255" json:"from"`
	To      string    `gorm:"column:to_addresses;type:text" json:"to"`  // JSON数组格式
	CC      string    `gorm:"column:cc_addresses;type:text" json:"cc"`  // JSON数组格式
	BCC     string    `gorm:"column:bcc_addresses;type:text" json:"bcc"` // JSON数组格式
	ReplyTo string    `gorm:"size:255" json:"reply_to"`
	Date    time.Time `gorm:"index" json:"date"`

	// 邮件内容
	TextBody string `gorm:"type:text" json:"text_body"`
	HTMLBody string `gorm:"type:text" json:"html_body"`

	// 邮件状态
	IsRead      bool `gorm:"not null;default:false;index" json:"is_read"`
	IsStarred   bool `gorm:"not null;default:false" json:"is_starred"`
	IsImportant bool `gorm:"not null;default:false" json:"is_important"`
	IsDeleted   bool `gorm:"not null;default:false;index" json:"is_deleted"`
	IsDraft     bool `gorm:"not null;default:false" json:"is_draft"`
	IsSent      bool `gorm:"not null;default:false" json:"is_sent"`

	// 邮件大小和附件信息
	Size          int64 `gorm:"default:0" json:"size"`
	HasAttachment bool  `gorm:"not null;default:false" json:"has_attachment"`

	// 邮件标签和分类
	Labels   string `gorm:"type:text" json:"labels"`                  // JSON数组格式
	Priority string `gorm:"size:20;default:normal" json:"priority"` // low, normal, high

	// 同步信息
	SyncedAt *time.Time `json:"synced_at"`

	// 关联关系
	Account     EmailAccount `gorm:"foreignKey:AccountID" json:"account,omitempty"`
	Folder      *Folder      `gorm:"foreignKey:FolderID" json:"folder,omitempty"`
	Attachments []Attachment `gorm:"foreignKey:EmailID" json:"attachments,omitempty"`
}

// TableName 指定表名
func (Email) TableName() string {
	return "emails"
}

// EmailAddress 邮件地址结构
type EmailAddress struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// GetToAddresses 获取收件人地址列表
func (e *Email) GetToAddresses() ([]EmailAddress, error) {
	if e.To == "" {
		return []EmailAddress{}, nil
	}

	var addresses []EmailAddress
	err := json.Unmarshal([]byte(e.To), &addresses)
	return addresses, err
}

// SetToAddresses 设置收件人地址列表
func (e *Email) SetToAddresses(addresses []EmailAddress) error {
	data, err := json.Marshal(addresses)
	if err != nil {
		return err
	}
	e.To = string(data)
	return nil
}

// GetCCAddresses 获取抄送地址列表
func (e *Email) GetCCAddresses() ([]EmailAddress, error) {
	if e.CC == "" {
		return []EmailAddress{}, nil
	}

	var addresses []EmailAddress
	err := json.Unmarshal([]byte(e.CC), &addresses)
	return addresses, err
}

// SetCCAddresses 设置抄送地址列表
func (e *Email) SetCCAddresses(addresses []EmailAddress) error {
	data, err := json.Marshal(addresses)
	if err != nil {
		return err
	}
	e.CC = string(data)
	return nil
}

// GetBCCAddresses 获取密送地址列表
func (e *Email) GetBCCAddresses() ([]EmailAddress, error) {
	if e.BCC == "" {
		return []EmailAddress{}, nil
	}

	var addresses []EmailAddress
	err := json.Unmarshal([]byte(e.BCC), &addresses)
	return addresses, err
}

// SetBCCAddresses 设置密送地址列表
func (e *Email) SetBCCAddresses(addresses []EmailAddress) error {
	data, err := json.Marshal(addresses)
	if err != nil {
		return err
	}
	e.BCC = string(data)
	return nil
}

// GetLabels 获取标签列表
func (e *Email) GetLabels() ([]string, error) {
	if e.Labels == "" {
		return []string{}, nil
	}

	var labels []string
	err := json.Unmarshal([]byte(e.Labels), &labels)
	return labels, err
}

// SetLabels 设置标签列表
func (e *Email) SetLabels(labels []string) error {
	data, err := json.Marshal(labels)
	if err != nil {
		return err
	}
	e.Labels = string(data)
	return nil
}

// MarkAsRead 标记为已读
func (e *Email) MarkAsRead() {
	e.IsRead = true
}

// MarkAsUnread 标记为未读
func (e *Email) MarkAsUnread() {
	e.IsRead = false
}

// ToggleStar 切换星标状态
func (e *Email) ToggleStar() {
	e.IsStarred = !e.IsStarred
}

// ToggleImportant 切换重要状态
func (e *Email) ToggleImportant() {
	e.IsImportant = !e.IsImportant
}
