package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"

	"firemail/internal/models"
	"gorm.io/gorm"
)

// EmailComposer 邮件组装器接口
type EmailComposer interface {
	// ComposeEmail 组装邮件
	ComposeEmail(ctx context.Context, request *ComposeEmailRequest) (*ComposedEmail, error)
	
	// ValidateEmail 验证邮件
	ValidateEmail(email *ComposedEmail) error
	
	// AddAttachment 添加附件
	AddAttachment(email *ComposedEmail, attachment *EmailAttachment) error
	
	// AddInlineAttachment 添加内联附件
	AddInlineAttachment(email *ComposedEmail, attachment *InlineAttachment) error
}

// ComposeEmailRequest 邮件组装请求
type ComposeEmailRequest struct {
	From                    *models.EmailAddress   `json:"from" binding:"required"`
	To                      []*models.EmailAddress `json:"to" binding:"required,min=1"`
	CC                      []*models.EmailAddress `json:"cc,omitempty"`
	BCC                     []*models.EmailAddress `json:"bcc,omitempty"`
	ReplyTo                 *models.EmailAddress   `json:"reply_to,omitempty"`
	Subject                 string                 `json:"subject" binding:"required"`
	TextBody                string                 `json:"text_body,omitempty"`
	HTMLBody                string                 `json:"html_body,omitempty"`
	Attachments             []*EmailAttachment     `json:"attachments,omitempty"`
	AttachmentIDs           []uint                 `json:"attachment_ids,omitempty"`
	InlineAttachments       []*InlineAttachment    `json:"inline_attachments,omitempty"`
	Priority                string                 `json:"priority,omitempty"` // high, normal, low
	Importance              string                 `json:"importance,omitempty"` // high, normal, low
	ScheduledTime           *string                `json:"scheduled_time,omitempty"` // ISO 8601 format
	RequestReadReceipt      bool                   `json:"request_read_receipt,omitempty"`
	RequestDeliveryReceipt  bool                   `json:"request_delivery_receipt,omitempty"`
	Headers                 map[string]string      `json:"headers,omitempty"`
	TemplateID              *uint                  `json:"template_id,omitempty"`
	TemplateData            map[string]interface{} `json:"template_data,omitempty"`
}

// EmailAttachment 邮件附件
type EmailAttachment struct {
	Filename    string    `json:"filename" binding:"required"`
	ContentType string    `json:"content_type"`
	Content     io.Reader `json:"-"`
	Data        []byte    `json:"data,omitempty"`
	Size        int64     `json:"size"`
	Encoding    string    `json:"encoding,omitempty"` // base64, quoted-printable
}

// InlineAttachment 内联附件
type InlineAttachment struct {
	ContentID   string    `json:"content_id" binding:"required"`
	Filename    string    `json:"filename" binding:"required"`
	ContentType string    `json:"content_type"`
	Content     io.Reader `json:"-"`
	Data        []byte    `json:"data,omitempty"`
	Size        int64     `json:"size"`
}

// ComposedEmail 组装完成的邮件
type ComposedEmail struct {
	ID                string                 `json:"id"`
	From              *models.EmailAddress   `json:"from"`
	To                []*models.EmailAddress `json:"to"`
	CC                []*models.EmailAddress `json:"cc"`
	BCC               []*models.EmailAddress `json:"bcc"`
	ReplyTo           *models.EmailAddress   `json:"reply_to"`
	Subject           string                 `json:"subject"`
	TextBody          string                 `json:"text_body"`
	HTMLBody          string                 `json:"html_body"`
	Attachments       []*EmailAttachment     `json:"attachments"`
	InlineAttachments []*InlineAttachment    `json:"inline_attachments"`
	Priority          string                 `json:"priority"`
	Headers           map[string]string      `json:"headers"`
	MIMEContent       []byte                 `json:"-"`
	CreatedAt         time.Time              `json:"created_at"`
	Size              int64                  `json:"size"`
}

// StandardEmailComposer 标准邮件组装器
type StandardEmailComposer struct {
	config *EmailComposerConfig
	db     *gorm.DB
}

// EmailComposerConfig 邮件组装器配置
type EmailComposerConfig struct {
	MaxAttachmentSize   int64    `json:"max_attachment_size"`   // 最大附件大小
	MaxAttachments      int      `json:"max_attachments"`       // 最大附件数量
	AllowedFileTypes    []string `json:"allowed_file_types"`    // 允许的文件类型
	EnableHTMLFilter    bool     `json:"enable_html_filter"`    // 启用HTML过滤
	MaxRecipientsPerEmail int    `json:"max_recipients_per_email"` // 每封邮件最大收件人数
	DefaultEncoding     string   `json:"default_encoding"`      // 默认编码
}

// NewStandardEmailComposer 创建标准邮件组装器
func NewStandardEmailComposer(config *EmailComposerConfig, db *gorm.DB) EmailComposer {
	if config == nil {
		config = &EmailComposerConfig{
			MaxAttachmentSize:     25 * 1024 * 1024, // 25MB
			MaxAttachments:        10,
			AllowedFileTypes:      []string{"pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx", "txt", "jpg", "jpeg", "png", "gif"},
			EnableHTMLFilter:      true,
			MaxRecipientsPerEmail: 100,
			DefaultEncoding:       "base64",
		}
	}
	
	return &StandardEmailComposer{
		config: config,
		db:     db,
	}
}

// ComposeEmail 组装邮件
func (c *StandardEmailComposer) ComposeEmail(ctx context.Context, request *ComposeEmailRequest) (*ComposedEmail, error) {
	// 验证请求
	if err := c.validateRequest(request); err != nil {
		return nil, fmt.Errorf("invalid email request: %w", err)
	}

	// 创建邮件对象
	email := &ComposedEmail{
		ID:                generateEmailID(),
		From:              request.From,
		To:                request.To,
		CC:                request.CC,
		BCC:               request.BCC,
		ReplyTo:           request.ReplyTo,
		Subject:           request.Subject,
		TextBody:          request.TextBody,
		HTMLBody:          request.HTMLBody,
		Priority:          request.Priority,
		Headers:           request.Headers,
		CreatedAt:         time.Now(),
	}

	// 处理模板
	if request.TemplateID != nil {
		if err := c.processTemplate(ctx, email, *request.TemplateID, request.TemplateData); err != nil {
			return nil, fmt.Errorf("failed to process template: %w", err)
		}
	}

	// 处理HTML内容
	if email.HTMLBody != "" && c.config.EnableHTMLFilter {
		email.HTMLBody = c.sanitizeHTML(email.HTMLBody)
	}

	// 处理附件
	for _, attachment := range request.Attachments {
		if err := c.AddAttachment(email, attachment); err != nil {
			return nil, fmt.Errorf("failed to add attachment: %w", err)
		}
	}

	// 处理附件ID（从数据库加载已上传的附件）
	if len(request.AttachmentIDs) > 0 {
		if err := c.loadAttachmentsFromIDs(ctx, email, request.AttachmentIDs); err != nil {
			return nil, fmt.Errorf("failed to load attachments from IDs: %w", err)
		}
	}

	// 处理内联附件
	for _, inlineAttachment := range request.InlineAttachments {
		if err := c.AddInlineAttachment(email, inlineAttachment); err != nil {
			return nil, fmt.Errorf("failed to add inline attachment: %w", err)
		}
	}

	// 构建MIME内容
	if err := c.buildMIMEContent(email); err != nil {
		return nil, fmt.Errorf("failed to build MIME content: %w", err)
	}

	// 验证最终邮件
	if err := c.ValidateEmail(email); err != nil {
		return nil, fmt.Errorf("email validation failed: %w", err)
	}

	return email, nil
}

// ValidateEmail 验证邮件
func (c *StandardEmailComposer) ValidateEmail(email *ComposedEmail) error {
	if email.From == nil {
		return fmt.Errorf("sender is required")
	}

	if len(email.To) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}

	totalRecipients := len(email.To) + len(email.CC) + len(email.BCC)
	if totalRecipients > c.config.MaxRecipientsPerEmail {
		return fmt.Errorf("too many recipients: %d (max: %d)", totalRecipients, c.config.MaxRecipientsPerEmail)
	}

	if email.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	if email.TextBody == "" && email.HTMLBody == "" {
		return fmt.Errorf("email body is required")
	}

	if len(email.Attachments) > c.config.MaxAttachments {
		return fmt.Errorf("too many attachments: %d (max: %d)", len(email.Attachments), c.config.MaxAttachments)
	}

	return nil
}

// AddAttachment 添加附件
func (c *StandardEmailComposer) AddAttachment(email *ComposedEmail, attachment *EmailAttachment) error {
	// 验证附件
	if err := c.validateAttachment(attachment); err != nil {
		return err
	}

	// 读取附件内容
	if attachment.Content != nil && len(attachment.Data) == 0 {
		data, err := io.ReadAll(attachment.Content)
		if err != nil {
			return fmt.Errorf("failed to read attachment content: %w", err)
		}
		attachment.Data = data
		attachment.Size = int64(len(data))
	}

	// 检测MIME类型
	if attachment.ContentType == "" {
		attachment.ContentType = c.detectContentType(attachment.Filename, attachment.Data)
	}

	// 设置默认编码
	if attachment.Encoding == "" {
		attachment.Encoding = c.config.DefaultEncoding
	}

	email.Attachments = append(email.Attachments, attachment)
	return nil
}

// AddInlineAttachment 添加内联附件
func (c *StandardEmailComposer) AddInlineAttachment(email *ComposedEmail, attachment *InlineAttachment) error {
	// 验证内联附件
	if attachment.ContentID == "" {
		return fmt.Errorf("content ID is required for inline attachment")
	}

	// 读取内容
	if attachment.Content != nil && len(attachment.Data) == 0 {
		data, err := io.ReadAll(attachment.Content)
		if err != nil {
			return fmt.Errorf("failed to read inline attachment content: %w", err)
		}
		attachment.Data = data
		attachment.Size = int64(len(data))
	}

	// 检测MIME类型
	if attachment.ContentType == "" {
		attachment.ContentType = c.detectContentType(attachment.Filename, attachment.Data)
	}

	email.InlineAttachments = append(email.InlineAttachments, attachment)
	return nil
}

// validateRequest 验证请求
func (c *StandardEmailComposer) validateRequest(request *ComposeEmailRequest) error {
	if request.From == nil {
		return fmt.Errorf("sender is required")
	}

	if len(request.To) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}

	if request.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	if request.TextBody == "" && request.HTMLBody == "" && request.TemplateID == nil {
		return fmt.Errorf("email body or template is required")
	}

	return nil
}

// validateAttachment 验证附件
func (c *StandardEmailComposer) validateAttachment(attachment *EmailAttachment) error {
	if attachment.Filename == "" {
		return fmt.Errorf("attachment filename is required")
	}

	if attachment.Size > c.config.MaxAttachmentSize {
		return fmt.Errorf("attachment too large: %d bytes (max: %d)", attachment.Size, c.config.MaxAttachmentSize)
	}

	// 检查文件类型
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(attachment.Filename), "."))
	if len(c.config.AllowedFileTypes) > 0 {
		allowed := false
		for _, allowedType := range c.config.AllowedFileTypes {
			if ext == strings.ToLower(allowedType) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("file type not allowed: %s", ext)
		}
	}

	return nil
}

// detectContentType 检测内容类型
func (c *StandardEmailComposer) detectContentType(filename string, data []byte) string {
	// 首先尝试根据文件扩展名检测
	ext := filepath.Ext(filename)
	if mimeType := mime.TypeByExtension(ext); mimeType != "" {
		return mimeType
	}

	// 如果有数据，尝试检测内容
	if len(data) > 0 {
		// 这里可以使用更复杂的内容检测逻辑
		// 暂时返回默认类型
	}

	return "application/octet-stream"
}

// sanitizeHTML 清理HTML内容
func (c *StandardEmailComposer) sanitizeHTML(htmlContent string) string {
	// 基础的HTML清理
	// 在实际应用中，应该使用专门的HTML清理库
	return html.EscapeString(htmlContent)
}

// processTemplate 处理邮件模板
func (c *StandardEmailComposer) processTemplate(ctx context.Context, email *ComposedEmail, templateID uint, data map[string]interface{}) error {
	// 注意：这里需要模板服务的实例，但为了避免循环依赖，
	// 我们将在后续重构中通过依赖注入来解决
	// 暂时返回提示信息
	return fmt.Errorf("template processing requires TemplateService dependency injection - will be implemented in service integration")
}

// buildMIMEContent 构建MIME内容
func (c *StandardEmailComposer) buildMIMEContent(email *ComposedEmail) error {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// 设置邮件头
	if err := c.writeEmailHeaders(&buf, email, writer.Boundary()); err != nil {
		return fmt.Errorf("failed to write email headers: %w", err)
	}

	// 写入邮件体
	if err := c.writeEmailBody(writer, email); err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}

	// 写入附件
	if err := c.writeAttachments(writer, email); err != nil {
		return fmt.Errorf("failed to write attachments: %w", err)
	}

	// 关闭multipart writer
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	email.MIMEContent = buf.Bytes()
	email.Size = int64(len(email.MIMEContent))

	return nil
}

// writeEmailHeaders 写入邮件头
func (c *StandardEmailComposer) writeEmailHeaders(buf *bytes.Buffer, email *ComposedEmail, boundary string) error {
	// From
	buf.WriteString(fmt.Sprintf("From: %s\r\n", c.formatEmailAddress(email.From)))

	// To
	if len(email.To) > 0 {
		buf.WriteString(fmt.Sprintf("To: %s\r\n", c.formatEmailAddresses(email.To)))
	}

	// CC
	if len(email.CC) > 0 {
		buf.WriteString(fmt.Sprintf("Cc: %s\r\n", c.formatEmailAddresses(email.CC)))
	}

	// Reply-To
	if email.ReplyTo != nil {
		buf.WriteString(fmt.Sprintf("Reply-To: %s\r\n", c.formatEmailAddress(email.ReplyTo)))
	}

	// Subject
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", c.encodeHeader(email.Subject)))

	// Date
	buf.WriteString(fmt.Sprintf("Date: %s\r\n", email.CreatedAt.Format(time.RFC1123Z)))

	// Message-ID
	buf.WriteString(fmt.Sprintf("Message-ID: <%s@firemail>\r\n", email.ID))

	// Priority
	if email.Priority != "" {
		switch strings.ToLower(email.Priority) {
		case "high":
			buf.WriteString("X-Priority: 1\r\n")
			buf.WriteString("Importance: high\r\n")
		case "low":
			buf.WriteString("X-Priority: 5\r\n")
			buf.WriteString("Importance: low\r\n")
		}
	}

	// Custom headers
	for key, value := range email.Headers {
		buf.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}

	// MIME headers
	buf.WriteString("MIME-Version: 1.0\r\n")

	// Content-Type
	hasAttachments := len(email.Attachments) > 0 || len(email.InlineAttachments) > 0
	hasHTML := email.HTMLBody != ""
	hasText := email.TextBody != ""

	if hasAttachments {
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n", boundary))
	} else if hasHTML && hasText {
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%s\r\n", boundary))
	} else if hasHTML {
		buf.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	} else {
		buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	}

	buf.WriteString("\r\n")
	return nil
}

// writeEmailBody 写入邮件体
func (c *StandardEmailComposer) writeEmailBody(writer *multipart.Writer, email *ComposedEmail) error {
	hasHTML := email.HTMLBody != ""
	hasText := email.TextBody != ""
	hasInlineAttachments := len(email.InlineAttachments) > 0

	if hasHTML && hasText {
		// 创建alternative部分
		altWriter, err := c.createAlternativePart(writer)
		if err != nil {
			return err
		}

		// 写入文本部分
		if err := c.writeTextPart(altWriter, email.TextBody); err != nil {
			return err
		}

		// 写入HTML部分（可能包含内联附件）
		if hasInlineAttachments {
			if err := c.writeHTMLWithInlineAttachments(altWriter, email); err != nil {
				return err
			}
		} else {
			if err := c.writeHTMLPart(altWriter, email.HTMLBody); err != nil {
				return err
			}
		}

		return altWriter.Close()
	} else if hasHTML {
		if hasInlineAttachments {
			return c.writeHTMLWithInlineAttachments(writer, email)
		} else {
			return c.writeHTMLPart(writer, email.HTMLBody)
		}
	} else if hasText {
		return c.writeTextPart(writer, email.TextBody)
	}

	return nil
}

// writeAttachments 写入附件
func (c *StandardEmailComposer) writeAttachments(writer *multipart.Writer, email *ComposedEmail) error {
	for _, attachment := range email.Attachments {
		if err := c.writeAttachment(writer, attachment); err != nil {
			return fmt.Errorf("failed to write attachment %s: %w", attachment.Filename, err)
		}
	}
	return nil
}

// writeAttachment 写入单个附件
func (c *StandardEmailComposer) writeAttachment(writer *multipart.Writer, attachment *EmailAttachment) error {
	// 创建附件头
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", attachment.ContentType)
	header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", c.encodeFilename(attachment.Filename)))

	if attachment.Encoding == "base64" {
		header.Set("Content-Transfer-Encoding", "base64")
	}

	// 创建附件部分
	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}

	// 写入附件内容
	if attachment.Encoding == "base64" {
		encoder := base64.NewEncoder(base64.StdEncoding, part)
		_, err = encoder.Write(attachment.Data)
		if err != nil {
			return err
		}
		return encoder.Close()
	} else {
		_, err = part.Write(attachment.Data)
		return err
	}
}

// 辅助方法
func (c *StandardEmailComposer) formatEmailAddress(addr *models.EmailAddress) string {
	if addr.Name != "" {
		return fmt.Sprintf("%s <%s>", c.encodeHeader(addr.Name), addr.Address)
	}
	return addr.Address
}

func (c *StandardEmailComposer) formatEmailAddresses(addrs []*models.EmailAddress) string {
	var formatted []string
	for _, addr := range addrs {
		formatted = append(formatted, c.formatEmailAddress(addr))
	}
	return strings.Join(formatted, ", ")
}

func (c *StandardEmailComposer) encodeHeader(header string) string {
	// 简单的头部编码，实际应该使用RFC 2047编码
	return header
}

func (c *StandardEmailComposer) encodeFilename(filename string) string {
	// 简单的文件名编码
	return filename
}

func (c *StandardEmailComposer) createAlternativePart(writer *multipart.Writer) (*multipart.Writer, error) {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", "multipart/alternative")

	part, err := writer.CreatePart(header)
	if err != nil {
		return nil, err
	}

	return multipart.NewWriter(part), nil
}

func (c *StandardEmailComposer) writeTextPart(writer *multipart.Writer, text string) error {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", "text/plain; charset=utf-8")
	header.Set("Content-Transfer-Encoding", "quoted-printable")

	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}

	// 对文本内容进行quoted-printable编码
	encodedText := c.encodeQuotedPrintable(text)
	_, err = part.Write([]byte(encodedText))
	return err
}

func (c *StandardEmailComposer) writeHTMLPart(writer *multipart.Writer, html string) error {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", "text/html; charset=utf-8")
	header.Set("Content-Transfer-Encoding", "quoted-printable")

	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}

	// 对HTML内容进行quoted-printable编码
	encodedHTML := c.encodeQuotedPrintable(html)
	_, err = part.Write([]byte(encodedHTML))
	return err
}

// encodeQuotedPrintable 对文本进行quoted-printable编码
func (c *StandardEmailComposer) encodeQuotedPrintable(text string) string {
	if text == "" {
		return text
	}

	var buf bytes.Buffer
	writer := quotedprintable.NewWriter(&buf)

	_, err := writer.Write([]byte(text))
	if err != nil {
		// 如果编码失败，返回原文本
		return text
	}

	err = writer.Close()
	if err != nil {
		// 如果关闭失败，返回原文本
		return text
	}

	return buf.String()
}

func (c *StandardEmailComposer) writeHTMLWithInlineAttachments(writer *multipart.Writer, email *ComposedEmail) error {
	// 创建related部分用于HTML和内联附件
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", "multipart/related")

	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}

	relatedWriter := multipart.NewWriter(part)

	// 写入HTML部分
	if err := c.writeHTMLPart(relatedWriter, email.HTMLBody); err != nil {
		return err
	}

	// 写入内联附件
	for _, inlineAttachment := range email.InlineAttachments {
		if err := c.writeInlineAttachment(relatedWriter, inlineAttachment); err != nil {
			return err
		}
	}

	return relatedWriter.Close()
}

func (c *StandardEmailComposer) writeInlineAttachment(writer *multipart.Writer, attachment *InlineAttachment) error {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", attachment.ContentType)
	header.Set("Content-Disposition", "inline")
	header.Set("Content-ID", fmt.Sprintf("<%s>", attachment.ContentID))
	header.Set("Content-Transfer-Encoding", "base64")

	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}

	encoder := base64.NewEncoder(base64.StdEncoding, part)
	_, err = encoder.Write(attachment.Data)
	if err != nil {
		return err
	}

	return encoder.Close()
}

// loadAttachmentsFromIDs 从数据库加载附件
func (c *StandardEmailComposer) loadAttachmentsFromIDs(ctx context.Context, email *ComposedEmail, attachmentIDs []uint) error {
	if len(attachmentIDs) == 0 {
		return nil
	}

	// 从数据库查询附件
	var attachments []models.Attachment
	if err := c.db.WithContext(ctx).Where("id IN ?", attachmentIDs).Find(&attachments).Error; err != nil {
		return fmt.Errorf("failed to query attachments: %w", err)
	}

	// 转换为EmailAttachment并添加到邮件
	for _, attachment := range attachments {
		// 读取附件文件内容
		var data []byte
		if attachment.StoragePath != "" {
			fileData, err := os.ReadFile(attachment.StoragePath)
			if err != nil {
				return fmt.Errorf("failed to read attachment file %s: %w", attachment.StoragePath, err)
			}
			data = fileData
		}

		emailAttachment := &EmailAttachment{
			Filename:    attachment.Filename,
			ContentType: attachment.ContentType,
			Data:        data,
			Size:        attachment.Size,
			Encoding:    c.config.DefaultEncoding,
		}

		if err := c.AddAttachment(email, emailAttachment); err != nil {
			return fmt.Errorf("failed to add attachment %s: %w", attachment.Filename, err)
		}
	}

	return nil
}

// generateEmailID 生成邮件ID
func generateEmailID() string {
	return fmt.Sprintf("email_%d_%d", time.Now().Unix(), time.Now().Nanosecond())
}
