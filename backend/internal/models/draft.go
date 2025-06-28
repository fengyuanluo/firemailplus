package models

import (
	"encoding/json"
	"time"
)

// Draft 草稿模型
type Draft struct {
	BaseModel
	UserID    uint   `gorm:"not null;index" json:"user_id"`
	AccountID uint   `gorm:"not null;index" json:"account_id"`
	Subject   string `gorm:"size:500" json:"subject"`
	
	// 收件人信息
	To  string `gorm:"column:to_addresses;type:text" json:"to"`   // JSON格式的收件人列表
	CC  string `gorm:"column:cc_addresses;type:text" json:"cc"`   // JSON格式的抄送列表
	BCC string `gorm:"column:bcc_addresses;type:text" json:"bcc"`  // JSON格式的密送列表
	
	// 邮件内容
	TextBody string `gorm:"type:text" json:"text_body"`
	HTMLBody string `gorm:"type:text" json:"html_body"`
	
	// 附件信息
	AttachmentIDs string `gorm:"type:text" json:"attachment_ids"` // JSON格式的附件ID列表
	
	// 元数据
	Priority     string     `gorm:"size:20;default:'normal'" json:"priority"` // low, normal, high
	IsTemplate   bool       `gorm:"default:false" json:"is_template"`
	TemplateName string     `gorm:"size:100" json:"template_name,omitempty"`
	LastEditedAt *time.Time `json:"last_edited_at"`
	
	// 关联关系
	User    User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Account EmailAccount `gorm:"foreignKey:AccountID" json:"account,omitempty"`
}

// TableName 指定表名
func (Draft) TableName() string {
	return "drafts"
}

// GetToAddresses 获取收件人地址列表
func (d *Draft) GetToAddresses() ([]EmailAddress, error) {
	if d.To == "" {
		return []EmailAddress{}, nil
	}
	
	var addresses []EmailAddress
	if err := json.Unmarshal([]byte(d.To), &addresses); err != nil {
		return nil, err
	}
	
	return addresses, nil
}

// SetToAddresses 设置收件人地址列表
func (d *Draft) SetToAddresses(addresses []EmailAddress) error {
	data, err := json.Marshal(addresses)
	if err != nil {
		return err
	}
	
	d.To = string(data)
	return nil
}

// GetCCAddresses 获取抄送地址列表
func (d *Draft) GetCCAddresses() ([]EmailAddress, error) {
	if d.CC == "" {
		return []EmailAddress{}, nil
	}
	
	var addresses []EmailAddress
	if err := json.Unmarshal([]byte(d.CC), &addresses); err != nil {
		return nil, err
	}
	
	return addresses, nil
}

// SetCCAddresses 设置抄送地址列表
func (d *Draft) SetCCAddresses(addresses []EmailAddress) error {
	data, err := json.Marshal(addresses)
	if err != nil {
		return err
	}
	
	d.CC = string(data)
	return nil
}

// GetBCCAddresses 获取密送地址列表
func (d *Draft) GetBCCAddresses() ([]EmailAddress, error) {
	if d.BCC == "" {
		return []EmailAddress{}, nil
	}
	
	var addresses []EmailAddress
	if err := json.Unmarshal([]byte(d.BCC), &addresses); err != nil {
		return nil, err
	}
	
	return addresses, nil
}

// SetBCCAddresses 设置密送地址列表
func (d *Draft) SetBCCAddresses(addresses []EmailAddress) error {
	data, err := json.Marshal(addresses)
	if err != nil {
		return err
	}
	
	d.BCC = string(data)
	return nil
}

// GetAttachmentIDs 获取附件ID列表
func (d *Draft) GetAttachmentIDs() ([]uint, error) {
	if d.AttachmentIDs == "" {
		return []uint{}, nil
	}
	
	var ids []uint
	if err := json.Unmarshal([]byte(d.AttachmentIDs), &ids); err != nil {
		return nil, err
	}
	
	return ids, nil
}

// SetAttachmentIDs 设置附件ID列表
func (d *Draft) SetAttachmentIDs(ids []uint) error {
	data, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	
	d.AttachmentIDs = string(data)
	return nil
}

// UpdateLastEditedAt 更新最后编辑时间
func (d *Draft) UpdateLastEditedAt() {
	now := time.Now()
	d.LastEditedAt = &now
}
