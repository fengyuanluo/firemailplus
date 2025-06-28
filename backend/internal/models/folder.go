package models

// Folder 邮件文件夹模型
type Folder struct {
	BaseModel
	AccountID   uint   `gorm:"not null;index" json:"account_id"`
	Name        string `gorm:"not null;size:100" json:"name"`
	DisplayName string `gorm:"size:100" json:"display_name"`
	Type        string `gorm:"not null;size:20" json:"type"` // inbox, sent, drafts, trash, spam, custom
	ParentID    *uint  `gorm:"index" json:"parent_id,omitempty"`
	Path        string `gorm:"size:500" json:"path"`     // IMAP文件夹路径
	Delimiter   string `gorm:"size:10" json:"delimiter"` // IMAP路径分隔符

	// 文件夹属性
	IsSelectable bool `gorm:"not null;default:true" json:"is_selectable"`
	IsSubscribed bool `gorm:"not null;default:true" json:"is_subscribed"`

	// 统计信息
	TotalEmails  int `gorm:"default:0" json:"total_emails"`
	UnreadEmails int `gorm:"default:0" json:"unread_emails"`

	// 同步信息
	UIDValidity uint32 `gorm:"column:uid_validity;default:0" json:"uid_validity"`
	UIDNext     uint32 `gorm:"column:uid_next;default:0" json:"uid_next"`

	// 关联关系
	Account  EmailAccount `gorm:"foreignKey:AccountID" json:"account,omitempty"`
	Parent   *Folder      `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []Folder     `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Emails   []Email      `gorm:"foreignKey:FolderID" json:"emails,omitempty"`
}

// TableName 指定表名
func (Folder) TableName() string {
	return "folders"
}

// FolderType 文件夹类型常量
const (
	FolderTypeInbox  = "inbox"
	FolderTypeSent   = "sent"
	FolderTypeDrafts = "drafts"
	FolderTypeTrash  = "trash"
	FolderTypeSpam   = "spam"
	FolderTypeCustom = "custom"
)

// IsSystemFolder 检查是否为系统文件夹
func (f *Folder) IsSystemFolder() bool {
	systemTypes := []string{
		FolderTypeInbox,
		FolderTypeSent,
		FolderTypeDrafts,
		FolderTypeTrash,
		FolderTypeSpam,
	}

	for _, sysType := range systemTypes {
		if f.Type == sysType {
			return true
		}
	}
	return false
}

// GetFullPath 获取完整路径
func (f *Folder) GetFullPath() string {
	if f.Path != "" {
		return f.Path
	}
	return f.Name
}

// UpdateCounts 更新邮件计数
func (f *Folder) UpdateCounts(total, unread int) {
	f.TotalEmails = total
	f.UnreadEmails = unread
}
