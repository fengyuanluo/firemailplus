package providers

import (
	"context"
	"io"
	"time"

	"firemail/internal/models"
	"firemail/internal/proxy"
)

// EmailProvider 邮件提供商接口
type EmailProvider interface {
	// 基础信息
	GetName() string
	GetDisplayName() string
	GetSupportedAuthMethods() []string
	GetProviderInfo() map[string]interface{}

	// 连接管理
	Connect(ctx context.Context, account *models.EmailAccount) error
	Disconnect() error
	IsConnected() bool
	IsIMAPConnected() bool
	IsSMTPConnected() bool
	TestConnection(ctx context.Context, account *models.EmailAccount) error

	// IMAP操作
	IMAPClient() IMAPClient

	// SMTP操作
	SMTPClient() SMTPClient

	// OAuth2操作（如果支持）
	OAuth2Client() OAuth2Client

	// 邮件操作
	SendEmail(ctx context.Context, account *models.EmailAccount, message *OutgoingMessage) error
	SyncEmails(ctx context.Context, account *models.EmailAccount, folderName string, lastUID uint32) ([]*EmailMessage, error)
}

// IMAPClient IMAP客户端接口
type IMAPClient interface {
	// 连接管理
	Connect(ctx context.Context, config IMAPClientConfig) error
	Disconnect() error
	IsConnected() bool

	// 文件夹操作
	ListFolders(ctx context.Context) ([]*FolderInfo, error)
	SelectFolder(ctx context.Context, folderName string) (*FolderStatus, error)
	CreateFolder(ctx context.Context, folderName string) error
	DeleteFolder(ctx context.Context, folderName string) error
	RenameFolder(ctx context.Context, oldName, newName string) error

	// 邮件操作
	FetchEmails(ctx context.Context, criteria *FetchCriteria) ([]*EmailMessage, error)
	FetchEmailByUID(ctx context.Context, uid uint32) (*EmailMessage, error)
	FetchEmailHeaders(ctx context.Context, uids []uint32) ([]*EmailHeader, error)

	// 邮件状态操作
	MarkAsRead(ctx context.Context, uids []uint32) error
	MarkAsUnread(ctx context.Context, uids []uint32) error
	DeleteEmails(ctx context.Context, uids []uint32) error
	MoveEmails(ctx context.Context, uids []uint32, targetFolder string) error
	CopyEmails(ctx context.Context, uids []uint32, targetFolder string) error

	// 搜索操作
	SearchEmails(ctx context.Context, criteria *SearchCriteria) ([]uint32, error)

	// 同步操作
	GetFolderStatus(ctx context.Context, folderName string) (*FolderStatus, error)
	GetNewEmails(ctx context.Context, folderName string, lastUID uint32) ([]*EmailMessage, error)
	GetEmailsInUIDRange(ctx context.Context, folderName string, startUID, endUID uint32) ([]*EmailMessage, error)

	// 附件操作
	GetAttachment(ctx context.Context, folderName string, uid uint32, partID string) (io.ReadCloser, error)
}

// SMTPClient SMTP客户端接口
type SMTPClient interface {
	// 连接管理
	Connect(ctx context.Context, config SMTPClientConfig) error
	Disconnect() error
	IsConnected() bool

	// 邮件发送
	SendEmail(ctx context.Context, message *OutgoingMessage) error
	SendRawEmail(ctx context.Context, from string, to []string, data []byte) error
}

// OAuth2Client OAuth2客户端接口
type OAuth2Client interface {
	// OAuth2流程
	GetAuthURL(state string, scopes []string) string
	ExchangeCode(ctx context.Context, code string) (*OAuth2Token, error)
	RefreshToken(ctx context.Context, refreshToken string) (*OAuth2Token, error)

	// Token验证
	ValidateToken(ctx context.Context, token *OAuth2Token) error
	RevokeToken(ctx context.Context, token string) error

	// 代理配置
	SetProxyConfig(config *ProxyConfig)
}

// 配置结构体

// IMAPClientConfig IMAP客户端配置
type IMAPClientConfig struct {
	Host        string
	Port        int
	Security    string // SSL, TLS, STARTTLS, NONE
	Username    string
	Password    string
	OAuth2Token *OAuth2Token
	IMAPIDInfo  map[string]string // IMAP ID信息，用于163等邮箱的可信部分

	// 代理配置
	ProxyConfig *ProxyConfig
}

// ProxyConfig 代理配置
type ProxyConfig struct {
	Type     string // none, http, socks5
	Host     string // 代理服务器地址
	Port     int    // 代理服务器端口
	Username string // 用户名（可选）
	Password string // 密码（可选）
}

// ToProxyConfig 转换为proxy包的ProxyConfig
func (pc *ProxyConfig) ToProxyConfig() *proxy.ProxyConfig {
	if pc == nil {
		return nil
	}
	return &proxy.ProxyConfig{
		Type:     pc.Type,
		Host:     pc.Host,
		Port:     pc.Port,
		Username: pc.Username,
		Password: pc.Password,
	}
}

// SMTPClientConfig SMTP客户端配置
type SMTPClientConfig struct {
	Host        string
	Port        int
	Security    string // SSL, TLS, STARTTLS, NONE
	Username    string
	Password    string
	OAuth2Token *OAuth2Token

	// 代理配置
	ProxyConfig *ProxyConfig
}

// OAuth2Token OAuth2令牌
type OAuth2Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
	Scope        string    `json:"scope,omitempty"`
}

// 数据结构体

// FolderInfo 文件夹信息
type FolderInfo struct {
	Name         string
	DisplayName  string
	Type         string // inbox, sent, drafts, trash, spam, custom
	Path         string
	Delimiter    string
	IsSelectable bool
	IsSubscribed bool
	Parent       string
	Children     []string
}

// FolderStatus 文件夹状态
type FolderStatus struct {
	Name         string
	TotalEmails  int
	UnreadEmails int
	UIDValidity  uint32
	UIDNext      uint32
}

// EmailMessage 邮件消息
type EmailMessage struct {
	UID         uint32
	MessageID   string
	Subject     string
	From        *models.EmailAddress
	To          []*models.EmailAddress
	CC          []*models.EmailAddress
	BCC         []*models.EmailAddress
	ReplyTo     *models.EmailAddress
	Date        time.Time
	TextBody    string
	HTMLBody    string
	Attachments []*AttachmentInfo
	Headers     map[string][]string
	Size        int64
	Flags       []string
	Labels      []string
	Priority    string
}

// SetLabels 设置邮件标签
func (e *EmailMessage) SetLabels(labels []string) {
	e.Labels = labels
}

// GetLabels 获取邮件标签
func (e *EmailMessage) GetLabels() []string {
	return e.Labels
}

// EmailHeader 邮件头信息
type EmailHeader struct {
	UID       uint32
	MessageID string
	Subject   string
	From      *models.EmailAddress
	Date      time.Time
	Size      int64
	Flags     []string
}

// EmailAddress 邮件地址在models包中定义

// AttachmentInfo 附件信息
type AttachmentInfo struct {
	PartID      string
	Filename    string
	ContentType string
	Size        int64
	ContentID   string
	Disposition string
	Encoding    string
	Content     []byte // 附件内容（可选，用于同步时保存）
}

// OutgoingMessage 发送邮件消息
type OutgoingMessage struct {
	From        *models.EmailAddress
	To          []*models.EmailAddress
	CC          []*models.EmailAddress
	BCC         []*models.EmailAddress
	ReplyTo     *models.EmailAddress
	Subject     string
	TextBody    string
	HTMLBody    string
	Attachments []*OutgoingAttachment
	Headers     map[string]string
	Priority    string
}

// OutgoingAttachment 发送附件
type OutgoingAttachment struct {
	Filename    string
	ContentType string
	Content     io.Reader
	Size        int64
	Disposition string // attachment, inline
	ContentID   string
}

// 查询条件

// FetchCriteria 获取邮件条件
type FetchCriteria struct {
	FolderName         string
	UIDs               []uint32
	Limit              int
	Offset             int
	SortBy             string // date, subject, from, size
	SortOrder          string // asc, desc
	IncludeBody        bool
	IncludeAttachments bool
}

// SearchCriteria 搜索条件
type SearchCriteria struct {
	FolderName string
	Subject    string
	From       string
	To         string
	Body       string
	Since      *time.Time
	Before     *time.Time
	Seen       *bool
	Flagged    *bool
	Deleted    *bool
	Draft      *bool
	Answered   *bool
	Size       *SizeCondition
}

// SizeCondition 大小条件
type SizeCondition struct {
	Operator string // gt, lt, eq
	Size     int64
}
