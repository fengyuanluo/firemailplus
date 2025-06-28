package mime

import (
	"mime/multipart"
	"net/textproto"
)

// MIMEType MIME类型信息
type MIMEType struct {
	// 主类型，如 "text", "multipart", "application"
	MainType string
	// 子类型，如 "plain", "html", "alternative", "related"
	SubType string
	// 完整类型，如 "text/plain", "multipart/alternative"
	FullType string
	// 参数，如 boundary, charset, type等
	Parameters map[string]string
}

// IsMultipart 检查是否为multipart类型
func (mt *MIMEType) IsMultipart() bool {
	return mt.MainType == "multipart"
}

// IsText 检查是否为text类型
func (mt *MIMEType) IsText() bool {
	return mt.MainType == "text"
}

// IsPlainText 检查是否为text/plain
func (mt *MIMEType) IsPlainText() bool {
	return mt.FullType == "text/plain"
}

// IsHTML 检查是否为text/html
func (mt *MIMEType) IsHTML() bool {
	return mt.FullType == "text/html"
}

// IsAlternative 检查是否为multipart/alternative
func (mt *MIMEType) IsAlternative() bool {
	return mt.FullType == "multipart/alternative"
}

// IsRelated 检查是否为multipart/related
func (mt *MIMEType) IsRelated() bool {
	return mt.FullType == "multipart/related"
}

// IsMixed 检查是否为multipart/mixed
func (mt *MIMEType) IsMixed() bool {
	return mt.FullType == "multipart/mixed"
}

// GetBoundary 获取boundary参数
func (mt *MIMEType) GetBoundary() string {
	if mt.Parameters == nil {
		return ""
	}
	return mt.Parameters["boundary"]
}

// GetCharset 获取charset参数
func (mt *MIMEType) GetCharset() string {
	if mt.Parameters == nil {
		return ""
	}
	return mt.Parameters["charset"]
}

// GetType 获取type参数（用于multipart/related）
func (mt *MIMEType) GetType() string {
	if mt.Parameters == nil {
		return ""
	}
	return mt.Parameters["type"]
}

// MIMEPart MIME部分信息
type MIMEPart struct {
	// 头部信息
	Headers textproto.MIMEHeader
	// MIME类型信息
	Type *MIMEType
	// 内容数据
	Content []byte
	// 传输编码
	TransferEncoding string
	// Content-Disposition信息
	Disposition *DispositionInfo
	// 子部分（用于multipart）
	Parts []*MIMEPart
	// 部分索引
	Index int
	// IMAP PartID（层级化）
	PartID string
}

// DispositionInfo Content-Disposition信息
type DispositionInfo struct {
	// 类型：attachment, inline
	Type string
	// 参数，如filename
	Parameters map[string]string
}

// GetFilename 获取文件名
func (di *DispositionInfo) GetFilename() string {
	if di.Parameters == nil {
		return ""
	}
	return di.Parameters["filename"]
}

// IsAttachment 检查是否为附件
func (di *DispositionInfo) IsAttachment() bool {
	return di.Type == "attachment"
}

// IsInline 检查是否为内联
func (di *DispositionInfo) IsInline() bool {
	return di.Type == "inline"
}

// ParsedEmail 解析后的邮件结构
type ParsedEmail struct {
	// 文本正文
	TextBody string
	// HTML正文
	HTMLBody string
	// 附件信息
	Attachments []*AttachmentInfo
	// 内联附件
	InlineAttachments []*AttachmentInfo
	// 原始MIME结构
	RootPart *MIMEPart
}

// AttachmentInfo 附件信息
type AttachmentInfo struct {
	// 部分ID
	PartID string
	// 文件名
	Filename string
	// 内容类型
	ContentType string
	// 大小
	Size int64
	// Content-ID（用于内联附件）
	ContentID string
	// 处置类型
	Disposition string
	// 传输编码
	Encoding string
	// 内容数据（可选）
	Content []byte
}

// ParseOptions 解析选项
type ParseOptions struct {
	// 是否包含附件内容
	IncludeAttachmentContent bool
	// 最大附件大小
	MaxAttachmentSize int64
	// 是否严格模式（遇到错误时停止）
	StrictMode bool
	// 是否保留原始MIME结构
	PreserveMIMEStructure bool
}

// DefaultParseOptions 默认解析选项
func DefaultParseOptions() *ParseOptions {
	return &ParseOptions{
		IncludeAttachmentContent: false,
		MaxAttachmentSize:        25 * 1024 * 1024, // 25MB
		StrictMode:               false,
		PreserveMIMEStructure:    true,
	}
}

// PartReader 部分读取器接口
type PartReader interface {
	// 读取下一个部分
	NextPart() (*multipart.Part, error)
}

// MIMEError MIME解析错误
type MIMEError struct {
	Message string
	Cause   error
	PartID  string
}

// Error 实现error接口
func (e *MIMEError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap 实现errors.Unwrap接口
func (e *MIMEError) Unwrap() error {
	return e.Cause
}

// NewMIMEError 创建MIME错误
func NewMIMEError(message string, cause error, partID string) *MIMEError {
	return &MIMEError{
		Message: message,
		Cause:   cause,
		PartID:  partID,
	}
}
