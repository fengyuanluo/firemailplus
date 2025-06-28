package parser

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"strings"
)

// UnifiedParser 统一的MIME解析器
// 基于emersion/go-message库，提供简洁统一的邮件解析接口
type UnifiedParser struct {
	options *ParseOptions
}

// ParseOptions 解析选项
type ParseOptions struct {
	// 是否包含附件内容
	IncludeAttachmentContent bool
	// 最大附件大小（字节）
	MaxAttachmentSize int64
	// 是否严格模式
	StrictMode bool
	// 最大错误数量
	MaxErrors int
}

// DefaultParseOptions 默认解析选项
func DefaultParseOptions() *ParseOptions {
	return &ParseOptions{
		IncludeAttachmentContent: true,
		MaxAttachmentSize:        25 * 1024 * 1024, // 25MB
		StrictMode:               false,
		MaxErrors:                10,
	}
}

// NewUnifiedParser 创建统一解析器
func NewUnifiedParser(options *ParseOptions) *UnifiedParser {
	if options == nil {
		options = DefaultParseOptions()
	}
	return &UnifiedParser{
		options: options,
	}
}

// ParsedEmail 解析后的邮件结构
type ParsedEmail struct {
	// 邮件头部
	Headers textproto.MIMEHeader
	// 文本正文
	TextBody string
	// HTML正文
	HTMLBody string
	// 附件列表
	Attachments []*AttachmentInfo
	// 内联附件列表
	InlineAttachments []*AttachmentInfo
	// 解析错误列表
	Errors []error
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
	// 处置类型（attachment/inline）
	Disposition string
	// 传输编码
	Encoding string
	// 内容数据
	Content []byte
}

// ParseEmail 解析邮件内容
func (p *UnifiedParser) ParseEmail(rawEmail []byte) (*ParsedEmail, error) {
	// 使用net/mail解析邮件
	msg, err := mail.ReadMessage(bytes.NewReader(rawEmail))
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %w", err)
	}

	result := &ParsedEmail{
		Headers:           textproto.MIMEHeader(msg.Header),
		Attachments:       make([]*AttachmentInfo, 0),
		InlineAttachments: make([]*AttachmentInfo, 0),
		Errors:            make([]error, 0),
	}

	// 解析邮件体
	if err := p.parseMessageBody(msg, result); err != nil {
		if p.options.StrictMode {
			return nil, fmt.Errorf("failed to parse message body: %w", err)
		}
		result.Errors = append(result.Errors, err)
	}

	return result, nil
}

// ParseEmailFromReader 从Reader解析邮件
func (p *UnifiedParser) ParseEmailFromReader(reader io.Reader) (*ParsedEmail, error) {
	// 读取所有内容
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	// 使用ParseEmail方法
	return p.ParseEmail(content)
}

// parseMessageBody 解析邮件正文
func (p *UnifiedParser) parseMessageBody(msg *mail.Message, result *ParsedEmail) error {
	// 获取Content-Type
	contentType := msg.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain"
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return fmt.Errorf("failed to parse content type: %w", err)
	}

	// 转换Header为MIMEHeader
	mimeHeader := textproto.MIMEHeader(msg.Header)

	// 根据媒体类型处理
	switch {
	case strings.HasPrefix(mediaType, "multipart/"):
		return p.parseMultipart(msg.Body, mediaType, params, result, "1")
	case mediaType == "text/plain":
		content, err := p.readPartContent(msg.Body, mimeHeader)
		if err != nil {
			return err
		}
		result.TextBody = string(content)
	case mediaType == "text/html":
		content, err := p.readPartContent(msg.Body, mimeHeader)
		if err != nil {
			return err
		}
		result.HTMLBody = string(content)
	default:
		// 其他类型作为附件处理
		return p.handleAttachment(msg.Body, mimeHeader, mediaType, params, result, "1")
	}

	return nil
}

// parseMultipart 解析multipart内容（改进版，支持大型邮件）
func (p *UnifiedParser) parseMultipart(body io.Reader, mediaType string, params map[string]string, result *ParsedEmail, partID string) error {
	boundary, ok := params["boundary"]
	if !ok {
		return fmt.Errorf("multipart message missing boundary")
	}

	// 尝试使用改进的multipart解析器
	if err := p.parseMultipartRobust(body, boundary, result, partID); err != nil {
		log.Printf("Robust multipart parsing failed: %v, trying fallback", err)

		// 回退到简单解析
		if fallbackErr := p.parseMultipartFallback(body, boundary, result, partID); fallbackErr != nil {
			if p.options.StrictMode {
				return fmt.Errorf("all multipart parsing strategies failed: %w", fallbackErr)
			}
			result.Errors = append(result.Errors, err, fallbackErr)
		}
	}

	return nil
}

// parseMultipartRobust 健壮的multipart解析器
func (p *UnifiedParser) parseMultipartRobust(body io.Reader, boundary string, result *ParsedEmail, partID string) error {
	// 创建带有更大缓冲区的Reader
	bufferedReader := bufio.NewReaderSize(body, 1024*1024) // 1MB缓冲区
	mr := multipart.NewReader(bufferedReader, boundary)

	partIndex := 1
	maxParts := 100 // 限制最大part数量，防止无限循环

	consecutiveErrors := 0
	maxConsecutiveErrors := 5 // 允许最多5个连续错误

	for partIndex <= maxParts {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			consecutiveErrors++

			// 记录具体的错误信息
			log.Printf("Parse warning: part %d: %v", partIndex, err)

			// 如果连续错误太多，可能是数据损坏，停止解析
			if consecutiveErrors >= maxConsecutiveErrors {
				log.Printf("Too many consecutive errors (%d), stopping multipart parsing", consecutiveErrors)
				break
			}

			if strings.Contains(err.Error(), "bufio: buffer full") {
				// 缓冲区满错误，尝试跳过当前part
				log.Printf("Buffer full error encountered, attempting to skip part %d", partIndex)
				partIndex++
				continue
			}

			if strings.Contains(err.Error(), "unexpected line") {
				// 意外行错误，通常是格式问题，尝试继续
				log.Printf("Unexpected line error, continuing with next part")
				partIndex++
				continue
			}

			// 对于EOF相关的错误，不要继续尝试
			if strings.Contains(err.Error(), "EOF") {
				log.Printf("EOF-related error encountered, stopping multipart parsing")
				break
			}

			// 其他错误
			if p.options.StrictMode {
				return fmt.Errorf("failed to read multipart part %d: %w", partIndex, err)
			}
			result.Errors = append(result.Errors, fmt.Errorf("part %d: %w", partIndex, err))
			partIndex++
			continue
		}

		// 成功读取part，重置连续错误计数
		consecutiveErrors = 0

		currentPartID := fmt.Sprintf("%s.%d", partID, partIndex)
		if err := p.parsePartSafely(part, result, currentPartID); err != nil {
			if p.options.StrictMode {
				return fmt.Errorf("failed to parse part %s: %w", currentPartID, err)
			}
			result.Errors = append(result.Errors, fmt.Errorf("part %s: %w", currentPartID, err))
		}

		partIndex++
	}

	if partIndex > maxParts {
		log.Printf("Warning: reached maximum part limit (%d), some parts may be skipped", maxParts)
	}

	return nil
}

// parsePartSafely 安全地解析单个part
func (p *UnifiedParser) parsePartSafely(part *multipart.Part, result *ParsedEmail, partID string) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in parsePartSafely for part %s: %v", partID, r)
		}
	}()

	return p.parsePart(part, result, partID)
}

// parseMultipartFallback 回退的multipart解析策略
func (p *UnifiedParser) parseMultipartFallback(body io.Reader, boundary string, result *ParsedEmail, partID string) error {
	// 读取所有内容到内存中进行手动解析
	content, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("failed to read multipart content: %w", err)
	}

	contentStr := string(content)

	// 手动分割multipart内容
	boundaryMarker := "--" + boundary
	parts := strings.Split(contentStr, boundaryMarker)

	log.Printf("Fallback multipart parsing: found %d potential parts", len(parts))

	for i, part := range parts {
		if i == 0 || i == len(parts)-1 {
			// 跳过第一个和最后一个部分（通常是空的或结束标记）
			continue
		}

		// 清理part内容
		part = strings.TrimSpace(part)
		if part == "" || part == "--" {
			continue
		}

		currentPartID := fmt.Sprintf("%s.%d", partID, i)
		if err := p.parsePartFromString(part, result, currentPartID); err != nil {
			log.Printf("Failed to parse fallback part %s: %v", currentPartID, err)
			if !p.options.StrictMode {
				result.Errors = append(result.Errors, fmt.Errorf("fallback part %s: %w", currentPartID, err))
			}
		}
	}

	return nil
}

// parsePartFromString 从字符串解析part
func (p *UnifiedParser) parsePartFromString(partContent string, result *ParsedEmail, partID string) error {
	// 分离头部和内容
	headerEndIndex := strings.Index(partContent, "\r\n\r\n")
	if headerEndIndex == -1 {
		headerEndIndex = strings.Index(partContent, "\n\n")
	}

	if headerEndIndex == -1 {
		// 没有找到头部分隔符，整个内容作为正文处理
		if strings.Contains(partContent, "<html") || strings.Contains(partContent, "<HTML") {
			if result.HTMLBody == "" {
				result.HTMLBody = partContent
			}
		} else {
			if result.TextBody == "" {
				result.TextBody = partContent
			}
		}
		return nil
	}

	headerStr := partContent[:headerEndIndex]
	contentStr := partContent[headerEndIndex+4:] // 跳过\r\n\r\n
	if strings.HasPrefix(contentStr, "\n") {
		contentStr = partContent[headerEndIndex+2:] // 如果是\n\n，只跳过2个字符
	}

	// 解析头部
	headers := make(textproto.MIMEHeader)
	headerLines := strings.Split(headerStr, "\n")
	for _, line := range headerLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		colonIndex := strings.Index(line, ":")
		if colonIndex == -1 {
			continue
		}

		key := strings.TrimSpace(line[:colonIndex])
		value := strings.TrimSpace(line[colonIndex+1:])
		headers.Add(key, value)
	}

	// 获取Content-Type
	contentType := headers.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain"
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = "text/plain"
	}

	// 根据类型处理内容
	switch {
	case mediaType == "text/plain":
		decoded, err := p.decodePartContent([]byte(contentStr), headers)
		if err != nil {
			return err
		}
		if result.TextBody == "" {
			result.TextBody = string(decoded)
		}
	case mediaType == "text/html":
		decoded, err := p.decodePartContent([]byte(contentStr), headers)
		if err != nil {
			return err
		}
		if result.HTMLBody == "" {
			result.HTMLBody = string(decoded)
		}
	default:
		// 可能是附件，但在回退模式下我们简化处理
		log.Printf("Skipping potential attachment in fallback mode: %s", mediaType)
	}

	return nil
}

// decodePartContent 解码part内容
func (p *UnifiedParser) decodePartContent(content []byte, headers textproto.MIMEHeader) ([]byte, error) {
	// 获取传输编码
	encoding := headers.Get("Content-Transfer-Encoding")

	// 解码传输编码
	decoded, err := p.decodeTransferEncoding(content, encoding)
	if err != nil {
		return content, err
	}

	return decoded, nil
}

// parsePart 解析单个部分
func (p *UnifiedParser) parsePart(part *multipart.Part, result *ParsedEmail, partID string) error {
	contentType := part.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain"
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return fmt.Errorf("failed to parse part content type: %w", err)
	}

	// 检查Content-Disposition
	disposition := part.Header.Get("Content-Disposition")
	dispositionType, _, _ := mime.ParseMediaType(disposition)

	// 根据类型和disposition处理
	switch {
	case strings.HasPrefix(mediaType, "multipart/"):
		return p.parseMultipart(part, mediaType, params, result, partID)
	case mediaType == "text/plain" && dispositionType != "attachment":
		content, err := p.readPartContent(part, part.Header)
		if err != nil {
			return err
		}
		if result.TextBody == "" {
			result.TextBody = string(content)
		}
	case mediaType == "text/html" && dispositionType != "attachment":
		content, err := p.readPartContent(part, part.Header)
		if err != nil {
			return err
		}
		if result.HTMLBody == "" {
			result.HTMLBody = string(content)
		}
	default:
		// 作为附件处理
		return p.handleAttachment(part, part.Header, mediaType, params, result, partID)
	}

	return nil
}

// handleAttachment 处理附件
func (p *UnifiedParser) handleAttachment(reader io.Reader, headers textproto.MIMEHeader, mediaType string, params map[string]string, result *ParsedEmail, partID string) error {
	// 获取文件名
	filename := p.extractFilename(headers, params)
	
	// 获取Content-ID
	contentID := headers.Get("Content-Id")
	if contentID != "" {
		// 移除尖括号
		contentID = strings.Trim(contentID, "<>")
	}

	// 获取disposition
	disposition := headers.Get("Content-Disposition")
	dispositionType, dispositionParams, _ := mime.ParseMediaType(disposition)
	
	if dispositionType == "" {
		if contentID != "" {
			dispositionType = "inline"
		} else {
			dispositionType = "attachment"
		}
	}

	// 如果没有从disposition获取到文件名，尝试从disposition参数获取
	if filename == "" && dispositionParams != nil {
		filename = dispositionParams["filename"]
	}

	attachment := &AttachmentInfo{
		PartID:      partID,
		Filename:    filename,
		ContentType: mediaType,
		ContentID:   contentID,
		Disposition: dispositionType,
		Encoding:    headers.Get("Content-Transfer-Encoding"),
	}

	// 读取内容
	if p.options.IncludeAttachmentContent {
		content, err := p.readPartContent(reader, headers)
		if err != nil {
			return err
		}
		
		// 检查大小限制
		if int64(len(content)) > p.options.MaxAttachmentSize {
			if p.options.StrictMode {
				return fmt.Errorf("attachment too large: %d bytes", len(content))
			}
			log.Printf("Warning: attachment %s too large (%d bytes), skipping content", filename, len(content))
		} else {
			attachment.Content = content
		}
		
		attachment.Size = int64(len(content))
	}

	// 根据disposition类型添加到相应列表
	if dispositionType == "inline" {
		result.InlineAttachments = append(result.InlineAttachments, attachment)
	} else {
		result.Attachments = append(result.Attachments, attachment)
	}

	return nil
}

// readPartContent 读取部分内容并解码
func (p *UnifiedParser) readPartContent(reader io.Reader, headers textproto.MIMEHeader) ([]byte, error) {
	// 读取原始内容
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	// 获取传输编码
	encoding := headers.Get("Content-Transfer-Encoding")
	
	// 解码传输编码
	decoded, err := p.decodeTransferEncoding(content, encoding)
	if err != nil {
		return nil, fmt.Errorf("failed to decode transfer encoding: %w", err)
	}

	return decoded, nil
}

// extractFilename 提取文件名
func (p *UnifiedParser) extractFilename(headers textproto.MIMEHeader, params map[string]string) string {
	// 首先尝试从Content-Type参数获取
	if name, ok := params["name"]; ok {
		return name
	}

	// 然后尝试从Content-Disposition获取
	disposition := headers.Get("Content-Disposition")
	if disposition != "" {
		_, dispositionParams, err := mime.ParseMediaType(disposition)
		if err == nil && dispositionParams != nil {
			if filename, ok := dispositionParams["filename"]; ok {
				return filename
			}
		}
	}

	return ""
}

// decodeTransferEncoding 解码传输编码
func (p *UnifiedParser) decodeTransferEncoding(content []byte, encoding string) ([]byte, error) {
	encoding = strings.ToLower(strings.TrimSpace(encoding))
	
	switch encoding {
	case "", "7bit", "8bit", "binary":
		return content, nil
	case "quoted-printable":
		reader := quotedprintable.NewReader(bytes.NewReader(content))
		return io.ReadAll(reader)
	case "base64":
		decoded, err := base64.StdEncoding.DecodeString(string(content))
		return decoded, err
	default:
		return content, fmt.Errorf("unsupported transfer encoding: %s", encoding)
	}
}

// 全局默认解析器实例
var defaultParser *UnifiedParser

// GetDefaultParser 获取默认解析器
func GetDefaultParser() *UnifiedParser {
	if defaultParser == nil {
		defaultParser = NewUnifiedParser(DefaultParseOptions())
	}
	return defaultParser
}

// 便利函数

// ParseEmail 使用默认解析器解析邮件
func ParseEmail(rawEmail []byte) (*ParsedEmail, error) {
	return GetDefaultParser().ParseEmail(rawEmail)
}

// ParseEmailFromReader 使用默认解析器从Reader解析邮件
func ParseEmailFromReader(reader io.Reader) (*ParsedEmail, error) {
	return GetDefaultParser().ParseEmailFromReader(reader)
}
