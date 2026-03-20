package models

const (
	EmailGroupSystemKeyDefaultPlaceholder = "default_placeholder"
)

// EmailGroup 邮箱分组模型
type EmailGroup struct {
	BaseModel
	UserID     uint           `gorm:"not null;index" json:"user_id"`
	Name       string         `gorm:"not null;size:100" json:"name"`
	SortOrder  int            `gorm:"not null;default:0;index" json:"sort_order"`
	IsDefault  bool           `gorm:"not null;default:false;index" json:"is_default"`
	SystemKey  *string        `gorm:"size:50;index" json:"system_key,omitempty"`
	Accounts   []EmailAccount `gorm:"foreignKey:GroupID" json:"accounts,omitempty"`
	AccountCnt int64          `gorm:"-" json:"account_count"`
}

// TableName 指定表名
func (EmailGroup) TableName() string {
	return "email_groups"
}

// IsSystemGroup 检查是否为系统管理的邮箱分组
func (g *EmailGroup) IsSystemGroup() bool {
	return g != nil && g.SystemKey != nil && *g.SystemKey != ""
}
