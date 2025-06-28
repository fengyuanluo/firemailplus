package models

import (
	"encoding/json"
	"time"
)

// EmailAccount 邮件账户模型
type EmailAccount struct {
	BaseModel
	UserID     uint   `gorm:"not null;index" json:"user_id"`
	Name       string `gorm:"not null;size:100" json:"name"`       // 账户显示名称
	Email      string `gorm:"not null;size:100" json:"email"`      // 邮箱地址
	Provider   string `gorm:"not null;size:50" json:"provider"`    // 提供商名称 (gmail, outlook, qq, etc.)
	AuthMethod string `gorm:"not null;size:20" json:"auth_method"` // 认证方式 (password, oauth2)

	// IMAP配置
	IMAPHost     string `gorm:"size:100" json:"imap_host"`
	IMAPPort     int    `gorm:"default:993" json:"imap_port"`
	IMAPSecurity string `gorm:"size:20;default:'SSL'" json:"imap_security"` // SSL, TLS, STARTTLS, NONE

	// SMTP配置
	SMTPHost     string `gorm:"size:100" json:"smtp_host"`
	SMTPPort     int    `gorm:"default:587" json:"smtp_port"`
	SMTPSecurity string `gorm:"size:20;default:'STARTTLS'" json:"smtp_security"` // SSL, TLS, STARTTLS, NONE

	// 认证信息（加密存储）
	Username string `gorm:"size:100" json:"username,omitempty"`
	Password string `gorm:"size:255" json:"-"` // 密码不在JSON中返回

	// OAuth2信息
	OAuth2Token string `gorm:"column:oauth2_token;type:text" json:"-"` // OAuth2 token（JSON格式，加密存储）

	// 状态信息
	IsActive     bool       `gorm:"not null;default:true" json:"is_active"`
	LastSyncAt   *time.Time `json:"last_sync_at"`
	SyncStatus   string     `gorm:"size:20;default:'pending'" json:"sync_status"` // pending, syncing, success, error
	ErrorMessage string     `gorm:"type:text" json:"error_message,omitempty"`

	// 统计信息
	TotalEmails  int `gorm:"default:0" json:"total_emails"`
	UnreadEmails int `gorm:"default:0" json:"unread_emails"`

	// 关联关系
	User    User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Emails  []Email  `gorm:"foreignKey:AccountID" json:"emails,omitempty"`
	Folders []Folder `gorm:"foreignKey:AccountID" json:"folders,omitempty"`
}

// TableName 指定表名
func (EmailAccount) TableName() string {
	return "email_accounts"
}

// OAuth2TokenData OAuth2 token数据结构
type OAuth2TokenData struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
	Scope        string    `json:"scope,omitempty"`
	ClientID     string    `json:"client_id,omitempty"` // 用于手动OAuth2配置
}

// SetOAuth2Token 设置OAuth2 token
func (ea *EmailAccount) SetOAuth2Token(token *OAuth2TokenData) error {
	tokenBytes, err := json.Marshal(token)
	if err != nil {
		return err
	}
	// TODO: 这里应该加密存储
	ea.OAuth2Token = string(tokenBytes)
	return nil
}

// GetOAuth2Token 获取OAuth2 token
func (ea *EmailAccount) GetOAuth2Token() (*OAuth2TokenData, error) {
	if ea.OAuth2Token == "" {
		return nil, nil
	}

	// TODO: 这里应该解密
	var token OAuth2TokenData
	err := json.Unmarshal([]byte(ea.OAuth2Token), &token)
	if err != nil {
		return nil, err
	}

	return &token, nil
}

// IsOAuth2TokenValid 检查OAuth2 token是否有效
func (ea *EmailAccount) IsOAuth2TokenValid() bool {
	token, err := ea.GetOAuth2Token()
	if err != nil || token == nil {
		return false
	}

	// 检查token是否过期（提前5分钟判断）
	return time.Now().Add(5 * time.Minute).Before(token.Expiry)
}

// NeedsOAuth2Refresh 检查是否需要刷新OAuth2 token
func (ea *EmailAccount) NeedsOAuth2Refresh() bool {
	if ea.AuthMethod != "oauth2" {
		return false
	}

	token, err := ea.GetOAuth2Token()
	if err != nil || token == nil {
		return true
	}

	// 如果token在30分钟内过期，则需要刷新
	return time.Now().Add(30 * time.Minute).After(token.Expiry)
}
