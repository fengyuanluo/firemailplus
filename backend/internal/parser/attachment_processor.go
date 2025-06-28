package parser

import (
	"fmt"
	"io"
	"mime"
	"net/textproto"
	"path/filepath"
	"strings"
)

// AttachmentProcessor 统一的附件处理器
// 负责提取和处理邮件附件，区分附件和内联内容
type AttachmentProcessor struct {
	encodingProcessor *EncodingProcessor
	options           *AttachmentOptions
}

// AttachmentOptions 附件处理选项
type AttachmentOptions struct {
	// 是否包含附件内容
	IncludeContent bool
	// 最大附件大小（字节）
	MaxSize int64
	// 允许的文件类型（空表示允许所有）
	AllowedTypes []string
	// 禁止的文件类型
	ForbiddenTypes []string
	// 是否处理内联附件
	ProcessInline bool
}

// DefaultAttachmentOptions 默认附件选项
func DefaultAttachmentOptions() *AttachmentOptions {
	return &AttachmentOptions{
		IncludeContent:   true,
		MaxSize:          25 * 1024 * 1024, // 25MB
		AllowedTypes:     []string{},        // 允许所有类型
		ForbiddenTypes:   []string{},        // 无禁止类型
		ProcessInline:    true,
	}
}

// NewAttachmentProcessor 创建附件处理器
func NewAttachmentProcessor(options *AttachmentOptions) *AttachmentProcessor {
	if options == nil {
		options = DefaultAttachmentOptions()
	}
	return &AttachmentProcessor{
		encodingProcessor: GetDefaultEncodingProcessor(),
		options:           options,
	}
}

// AttachmentResult 附件处理结果
type AttachmentResult struct {
	// 常规附件
	Attachments []*AttachmentInfo
	// 内联附件
	InlineAttachments []*AttachmentInfo
	// 处理错误
	Errors []error
}

// ProcessAttachment 处理单个附件
func (p *AttachmentProcessor) ProcessAttachment(reader io.Reader, headers textproto.MIMEHeader, partID string) (*AttachmentInfo, error) {
	// 解析Content-Type
	contentType := headers.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse content type: %w", err)
	}

	// 解析Content-Disposition
	disposition := headers.Get("Content-Disposition")
	dispositionType, dispositionParams, _ := mime.ParseMediaType(disposition)

	// 确定附件类型
	if dispositionType == "" {
		// 如果没有明确的disposition，根据Content-ID判断
		contentID := headers.Get("Content-Id")
		if contentID != "" {
			dispositionType = "inline"
		} else {
			dispositionType = "attachment"
		}
	}

	// 如果不处理内联附件且这是内联附件，跳过
	if !p.options.ProcessInline && dispositionType == "inline" {
		return nil, nil
	}

	// 提取文件名
	filename := p.extractFilename(headers, params, dispositionParams)

	// 检查文件类型是否允许
	if !p.isTypeAllowed(mediaType, filename) {
		return nil, fmt.Errorf("file type not allowed: %s", mediaType)
	}

	// 创建附件信息
	attachment := &AttachmentInfo{
		PartID:      partID,
		Filename:    filename,
		ContentType: mediaType,
		ContentID:   p.extractContentID(headers),
		Disposition: dispositionType,
		Encoding:    headers.Get("Content-Transfer-Encoding"),
	}

	// 读取和处理内容
	if p.options.IncludeContent {
		content, err := p.readAndDecodeContent(reader, headers)
		if err != nil {
			return nil, fmt.Errorf("failed to read attachment content: %w", err)
		}

		// 检查大小限制
		if int64(len(content)) > p.options.MaxSize {
			return nil, fmt.Errorf("attachment too large: %d bytes (max: %d)", len(content), p.options.MaxSize)
		}

		attachment.Content = content
		attachment.Size = int64(len(content))
	}

	return attachment, nil
}

// extractFilename 提取文件名
func (p *AttachmentProcessor) extractFilename(headers textproto.MIMEHeader, contentParams, dispositionParams map[string]string) string {
	// 优先从Content-Disposition的filename参数获取
	if dispositionParams != nil {
		if filename := dispositionParams["filename"]; filename != "" {
			return p.decodeFilename(filename)
		}
	}

	// 然后从Content-Type的name参数获取
	if contentParams != nil {
		if name := contentParams["name"]; name != "" {
			return p.decodeFilename(name)
		}
	}

	// 最后尝试从Content-Disposition头部直接解析
	disposition := headers.Get("Content-Disposition")
	if disposition != "" {
		// 尝试手动解析filename参数（处理一些非标准格式）
		if filename := p.extractFilenameFromDisposition(disposition); filename != "" {
			return p.decodeFilename(filename)
		}
	}

	return ""
}

// extractFilenameFromDisposition 从disposition头部提取文件名
func (p *AttachmentProcessor) extractFilenameFromDisposition(disposition string) string {
	// 查找filename=
	filenameIndex := strings.Index(strings.ToLower(disposition), "filename=")
	if filenameIndex == -1 {
		return ""
	}

	// 提取filename值
	start := filenameIndex + 9 // len("filename=")
	if start >= len(disposition) {
		return ""
	}

	value := disposition[start:]
	
	// 处理引号
	if strings.HasPrefix(value, "\"") {
		// 查找结束引号
		endIndex := strings.Index(value[1:], "\"")
		if endIndex != -1 {
			return value[1 : endIndex+1]
		}
	} else {
		// 查找分号或结束
		endIndex := strings.Index(value, ";")
		if endIndex != -1 {
			return strings.TrimSpace(value[:endIndex])
		}
		return strings.TrimSpace(value)
	}

	return ""
}

// decodeFilename 解码文件名（处理RFC2047编码）
func (p *AttachmentProcessor) decodeFilename(filename string) string {
	// 移除引号
	filename = strings.Trim(filename, "\"'")
	
	// 如果包含RFC2047编码，尝试解码
	if strings.Contains(filename, "=?") && strings.Contains(filename, "?=") {
		// 这里可以添加RFC2047解码逻辑
		// 暂时返回原文件名
		return filename
	}

	return filename
}

// extractContentID 提取Content-ID
func (p *AttachmentProcessor) extractContentID(headers textproto.MIMEHeader) string {
	contentID := headers.Get("Content-Id")
	if contentID == "" {
		contentID = headers.Get("Content-ID") // 尝试大写版本
	}
	
	// 移除尖括号
	contentID = strings.Trim(contentID, "<>")
	return contentID
}

// readAndDecodeContent 读取并解码内容
func (p *AttachmentProcessor) readAndDecodeContent(reader io.Reader, headers textproto.MIMEHeader) ([]byte, error) {
	// 读取原始内容
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	// 获取传输编码
	transferEncoding := headers.Get("Content-Transfer-Encoding")
	
	// 解码传输编码
	decoded, err := p.encodingProcessor.DecodeTransferEncoding(content, transferEncoding)
	if err != nil {
		return nil, fmt.Errorf("failed to decode transfer encoding: %w", err)
	}

	return decoded, nil
}

// isTypeAllowed 检查文件类型是否允许
func (p *AttachmentProcessor) isTypeAllowed(mediaType, filename string) bool {
	// 检查禁止类型
	for _, forbidden := range p.options.ForbiddenTypes {
		if strings.EqualFold(mediaType, forbidden) {
			return false
		}
		// 也检查文件扩展名
		if filename != "" {
			ext := strings.ToLower(filepath.Ext(filename))
			if strings.EqualFold(ext, forbidden) {
				return false
			}
		}
	}

	// 如果没有指定允许类型，则允许所有（除了禁止的）
	if len(p.options.AllowedTypes) == 0 {
		return true
	}

	// 检查允许类型
	for _, allowed := range p.options.AllowedTypes {
		if strings.EqualFold(mediaType, allowed) {
			return true
		}
		// 也检查文件扩展名
		if filename != "" {
			ext := strings.ToLower(filepath.Ext(filename))
			if strings.EqualFold(ext, allowed) {
				return true
			}
		}
	}

	return false
}

// IsAttachment 判断是否为附件
func (p *AttachmentProcessor) IsAttachment(headers textproto.MIMEHeader, mediaType string) bool {
	// 检查Content-Disposition
	disposition := headers.Get("Content-Disposition")
	if disposition != "" {
		dispositionType, _, _ := mime.ParseMediaType(disposition)
		return dispositionType == "attachment"
	}

	// 如果没有disposition，根据媒体类型判断
	switch {
	case strings.HasPrefix(mediaType, "text/"):
		return false // 文本通常不是附件
	case strings.HasPrefix(mediaType, "multipart/"):
		return false // multipart不是附件
	case mediaType == "message/rfc822":
		return true // 嵌入的邮件消息通常是附件
	default:
		// 其他类型默认认为是附件
		return true
	}
}

// IsInlineAttachment 判断是否为内联附件
func (p *AttachmentProcessor) IsInlineAttachment(headers textproto.MIMEHeader) bool {
	// 检查Content-Disposition
	disposition := headers.Get("Content-Disposition")
	if disposition != "" {
		dispositionType, _, _ := mime.ParseMediaType(disposition)
		return dispositionType == "inline"
	}

	// 检查Content-ID（有Content-ID通常表示内联）
	contentID := headers.Get("Content-Id")
	if contentID == "" {
		contentID = headers.Get("Content-ID")
	}
	
	return contentID != ""
}

// GetSafeFilename 获取安全的文件名
func (p *AttachmentProcessor) GetSafeFilename(filename string) string {
	if filename == "" {
		return "unnamed_attachment"
	}

	// 移除路径分隔符和其他危险字符
	unsafe := []string{"/", "\\", "..", ":", "*", "?", "\"", "<", ">", "|"}
	safe := filename
	for _, char := range unsafe {
		safe = strings.ReplaceAll(safe, char, "_")
	}

	// 确保不以点开头（隐藏文件）
	if strings.HasPrefix(safe, ".") {
		safe = "file" + safe
	}

	return safe
}

// 全局默认附件处理器实例
var defaultAttachmentProcessor *AttachmentProcessor

// GetDefaultAttachmentProcessor 获取默认附件处理器
func GetDefaultAttachmentProcessor() *AttachmentProcessor {
	if defaultAttachmentProcessor == nil {
		defaultAttachmentProcessor = NewAttachmentProcessor(DefaultAttachmentOptions())
	}
	return defaultAttachmentProcessor
}

// 便利函数

// ProcessAttachment 使用默认处理器处理附件
func ProcessAttachment(reader io.Reader, headers textproto.MIMEHeader, partID string) (*AttachmentInfo, error) {
	return GetDefaultAttachmentProcessor().ProcessAttachment(reader, headers, partID)
}

// IsAttachment 使用默认处理器判断是否为附件
func IsAttachment(headers textproto.MIMEHeader, mediaType string) bool {
	return GetDefaultAttachmentProcessor().IsAttachment(headers, mediaType)
}

// IsInlineAttachment 使用默认处理器判断是否为内联附件
func IsInlineAttachment(headers textproto.MIMEHeader) bool {
	return GetDefaultAttachmentProcessor().IsInlineAttachment(headers)
}
