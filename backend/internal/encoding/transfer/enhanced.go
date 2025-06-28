package transfer

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/emersion/go-message"
)

// EnhancedDecoder 增强解码器，集成emersion/go-message库
type EnhancedDecoder struct {
	standardManager *DecoderManager
}

// NewEnhancedDecoder 创建增强解码器
func NewEnhancedDecoder() *EnhancedDecoder {
	return &EnhancedDecoder{
		standardManager: NewStandardDecoderManager(),
	}
}

// DecodeWithHeaders 使用邮件头信息解码内容
func (e *EnhancedDecoder) DecodeWithHeaders(data []byte, headers message.Header) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	// 获取Content-Transfer-Encoding
	encoding := headers.Get("Content-Transfer-Encoding")

	// 如果没有指定编码，默认使用7bit
	if encoding == "" {
		encoding = "7bit"
	}

	// 使用标准解码器解码
	return e.standardManager.DecodeWithFallback(data, encoding)
}

// DecodeEntity 解码邮件实体
func (e *EnhancedDecoder) DecodeEntity(entity *message.Entity) ([]byte, error) {
	if entity == nil {
		return nil, fmt.Errorf("entity is nil")
	}

	// 读取实体内容
	content, err := io.ReadAll(entity.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read entity body: %w", err)
	}

	// 使用头信息解码
	return e.DecodeWithHeaders(content, entity.Header)
}

// DecodeEntityToString 解码邮件实体为字符串
func (e *EnhancedDecoder) DecodeEntityToString(entity *message.Entity) (string, error) {
	decoded, err := e.DecodeEntity(entity)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// ParseAndDecodeMessage 解析并解码完整的邮件消息
func (e *EnhancedDecoder) ParseAndDecodeMessage(rawMessage []byte) (*DecodedMessage, error) {
	// 解析邮件消息
	msg, err := message.Read(bytes.NewReader(rawMessage))
	if err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}

	result := &DecodedMessage{
		Headers: make(map[string]string),
	}

	// 复制头信息
	for field := msg.Header.Fields(); field.Next(); {
		result.Headers[field.Key()] = field.Value()
	}

	// 解析邮件结构
	if err := e.parseMessageParts(msg, result); err != nil {
		return nil, fmt.Errorf("failed to parse message parts: %w", err)
	}

	return result, nil
}

// parseMessageParts 解析邮件部分
func (e *EnhancedDecoder) parseMessageParts(entity *message.Entity, result *DecodedMessage) error {
	mediaType, params, err := entity.Header.ContentType()
	if err != nil {
		// 如果无法解析Content-Type，尝试作为文本处理
		mediaType = "text/plain"
		params = make(map[string]string)
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		// 处理多部分消息
		return e.parseMultipartMessage(entity, result, params)
	} else {
		// 处理单部分消息
		return e.parseSinglePartMessage(entity, result, mediaType)
	}
}

// parseMultipartMessage 解析多部分消息
func (e *EnhancedDecoder) parseMultipartMessage(entity *message.Entity, result *DecodedMessage, params map[string]string) error {
	mr := entity.MultipartReader()
	if mr == nil {
		return fmt.Errorf("failed to create multipart reader")
	}

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read multipart: %w", err)
		}

		if err := e.parseMessageParts(part, result); err != nil {
			return err
		}
	}

	return nil
}

// parseSinglePartMessage 解析单部分消息
func (e *EnhancedDecoder) parseSinglePartMessage(entity *message.Entity, result *DecodedMessage, mediaType string) error {
	// 解码内容
	decoded, err := e.DecodeEntity(entity)
	if err != nil {
		return fmt.Errorf("failed to decode entity: %w", err)
	}

	// 根据媒体类型分配内容
	switch {
	case strings.HasPrefix(mediaType, "text/plain"):
		if result.TextBody == "" {
			result.TextBody = string(decoded)
		}
	case strings.HasPrefix(mediaType, "text/html"):
		if result.HTMLBody == "" {
			result.HTMLBody = string(decoded)
		}
	default:
		// 处理附件
		attachment := &DecodedAttachment{
			ContentType: mediaType,
			Content:     decoded,
		}

		// 获取文件名
		if disposition, params, err := entity.Header.ContentDisposition(); err == nil {
			if filename, ok := params["filename"]; ok {
				attachment.Filename = filename
			}
			attachment.Disposition = disposition
		}

		// 获取Content-ID
		if contentID := entity.Header.Get("Content-ID"); contentID != "" {
			attachment.ContentID = strings.Trim(contentID, "<>")
		}

		result.Attachments = append(result.Attachments, attachment)
	}

	return nil
}

// DecodedMessage 解码后的邮件消息
type DecodedMessage struct {
	Headers     map[string]string      `json:"headers"`
	TextBody    string                 `json:"text_body"`
	HTMLBody    string                 `json:"html_body"`
	Attachments []*DecodedAttachment   `json:"attachments"`
}

// DecodedAttachment 解码后的附件
type DecodedAttachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	ContentID   string `json:"content_id,omitempty"`
	Disposition string `json:"disposition"`
	Content     []byte `json:"-"` // 不序列化到JSON
	Size        int64  `json:"size"`
}

// GetSize 获取附件大小
func (a *DecodedAttachment) GetSize() int64 {
	if a.Size == 0 && len(a.Content) > 0 {
		a.Size = int64(len(a.Content))
	}
	return a.Size
}

// 全局增强解码器实例
var defaultEnhancedDecoder *EnhancedDecoder
var enhancedDecoderOnce sync.Once

// GetDefaultEnhancedDecoder 获取默认增强解码器
func GetDefaultEnhancedDecoder() *EnhancedDecoder {
	enhancedDecoderOnce.Do(func() {
		defaultEnhancedDecoder = NewEnhancedDecoder()
	})
	return defaultEnhancedDecoder
}

// 便利函数

// ParseMessage 解析邮件消息
func ParseMessage(rawMessage []byte) (*DecodedMessage, error) {
	return GetDefaultEnhancedDecoder().ParseAndDecodeMessage(rawMessage)
}

// DecodeWithHeaders 使用头信息解码内容
func DecodeWithHeaders(data []byte, headers message.Header) ([]byte, error) {
	return GetDefaultEnhancedDecoder().DecodeWithHeaders(data, headers)
}
