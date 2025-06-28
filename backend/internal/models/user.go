package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	BaseModel
	Username    string     `gorm:"uniqueIndex;not null;size:50" json:"username"`
	Password    string     `gorm:"not null;size:255" json:"-"` // 不在JSON中返回密码
	Email       string     `gorm:"size:100" json:"email"`
	DisplayName string     `gorm:"size:100" json:"display_name"`
	Role        string     `gorm:"not null;default:'admin';size:20" json:"role"`
	IsActive    bool       `gorm:"not null;default:true" json:"is_active"`
	LastLoginAt *time.Time `json:"last_login_at"`
	LoginCount  int        `gorm:"default:0" json:"login_count"`

	// 关联关系
	EmailAccounts []EmailAccount `gorm:"foreignKey:UserID" json:"email_accounts,omitempty"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// BeforeCreate 创建前钩子，加密密码
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u.Password = string(hashedPassword)
	}
	return nil
}

// BeforeUpdate 更新前钩子，如果密码有变化则加密
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	// 检查密码是否被修改
	if tx.Statement.Changed("Password") && u.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u.Password = string(hashedPassword)
	}
	return nil
}

// CheckPassword 验证密码
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// SetPassword 设置密码（手动加密）
func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}
