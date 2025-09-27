package models

// EmailAccountGroup 邮箱账户分组
type EmailAccountGroup struct {
	BaseModel
	UserID    uint   `gorm:"not null;index" json:"user_id"`
	Name      string `gorm:"not null;size:100" json:"name"`
	SortOrder int    `gorm:"not null;default:0" json:"sort_order"`

	// 关联关系
	User     User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Accounts []EmailAccount `gorm:"foreignKey:GroupID" json:"accounts,omitempty"`
}

// TableName 指定表名
func (EmailAccountGroup) TableName() string {
	return "email_account_groups"
}
