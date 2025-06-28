package mime

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"strings"
)

// NestedMultipartParser 嵌套multipart解析器
type NestedMultipartParser struct {
	detector  *MIMETypeDetector
	processor *MIMEPartProcessor
	options   *ParseOptions
}

// NewNestedMultipartParser 创建嵌套multipart解析器
func NewNestedMultipartParser(options *ParseOptions) *NestedMultipartParser {
	if options == nil {
		options = DefaultParseOptions()
	}

	return &NestedMultipartParser{
		detector:  NewMIMETypeDetector(),
		processor: NewMIMEPartProcessor(),
		options:   options,
	}
}

// ParseEmail 解析完整邮件（从字符串）
func (p *NestedMultipartParser) ParseEmail(rawEmail string) (*ParsedEmail, error) {
	return p.ParseEmailFromBytes([]byte(rawEmail))
}

// ParseEmailFromBytes 解析完整邮件（从字节数组，避免二进制数据损坏）
func (p *NestedMultipartParser) ParseEmailFromBytes(rawEmailBytes []byte) (*ParsedEmail, error) {
	// 查找头部和正文的分隔符
	headerEndIndex := p.findHeaderEndIndex(rawEmailBytes)
	var headerBytes, contentBytes []byte

	if headerEndIndex == -1 {
		// 没有找到头部分隔符，整个内容作为正文处理
		return p.parseSimpleEmailFromBytes(rawEmailBytes)
	}

	headerBytes = rawEmailBytes[:headerEndIndex]
	// 跳过分隔符
	if headerEndIndex+4 <= len(rawEmailBytes) &&
		rawEmailBytes[headerEndIndex] == '\r' && rawEmailBytes[headerEndIndex+1] == '\n' &&
		rawEmailBytes[headerEndIndex+2] == '\r' && rawEmailBytes[headerEndIndex+3] == '\n' {
		contentBytes = rawEmailBytes[headerEndIndex+4:]
	} else if headerEndIndex+2 <= len(rawEmailBytes) &&
		rawEmailBytes[headerEndIndex] == '\n' && rawEmailBytes[headerEndIndex+1] == '\n' {
		contentBytes = rawEmailBytes[headerEndIndex+2:]
	} else {
		contentBytes = rawEmailBytes[headerEndIndex:]
	}

	// 将头部转换为字符串进行解析（头部通常是ASCII安全的）
	headerStr := string(headerBytes)

	// 检测根部分的MIME类型
	rootMIMEType, err := p.detector.DetectFromRawHeaders(headerStr)
	if err != nil {
		log.Printf("Warning: failed to detect root MIME type: %v", err)
		return p.parseSimpleEmailFromBytes(rawEmailBytes)
	}

	// 创建根部分 - 保持内容为字节数组
	rootPart := &MIMEPart{
		Headers: p.processor.parseRawHeaders(headerStr),
		Type:    rootMIMEType,
		Content: contentBytes,
		Index:   0,
		PartID:  "", // 根部分的PartID为空
	}

	// 解析邮件结构
	result := &ParsedEmail{
		RootPart: rootPart,
	}

	if rootMIMEType.IsMultipart() {
		// 解析multipart结构，传入空的父PartID
		err = p.parseMultipartStructureWithPartID(rootPart, result, "")
		if err != nil {
			log.Printf("Warning: failed to parse multipart structure: %v", err)
			// 回退到简单解析
			return p.parseSimpleEmailFromBytes(rawEmailBytes)
		}
	} else {
		// 单部分邮件
		rootPart.PartID = "1"
		err = p.parseSinglePart(rootPart, result)
		if err != nil {
			log.Printf("Warning: failed to parse single part: %v", err)
			return p.parseSimpleEmailFromBytes(rawEmailBytes)
		}
	}

	return result, nil
}

// findHeaderEndIndex 查找头部结束位置
func (p *NestedMultipartParser) findHeaderEndIndex(data []byte) int {
	// 查找 \r\n\r\n
	for i := 0; i < len(data)-3; i++ {
		if data[i] == '\r' && data[i+1] == '\n' && data[i+2] == '\r' && data[i+3] == '\n' {
			return i
		}
	}
	// 查找 \n\n
	for i := 0; i < len(data)-1; i++ {
		if data[i] == '\n' && data[i+1] == '\n' {
			return i
		}
	}
	return -1
}

// parseSimpleEmailFromBytes 简单邮件解析（从字节数组）
func (p *NestedMultipartParser) parseSimpleEmailFromBytes(rawEmailBytes []byte) (*ParsedEmail, error) {
	// 简单地将内容作为文本正文
	return &ParsedEmail{
		TextBody: string(rawEmailBytes),
	}, nil
}

// parseMultipartStructure 解析multipart结构（兼容性方法）
func (p *NestedMultipartParser) parseMultipartStructure(part *MIMEPart, result *ParsedEmail) error {
	return p.parseMultipartStructureWithPartID(part, result, "")
}

// parseMultipartStructureWithPartID 解析multipart结构，正确处理PartID层级
func (p *NestedMultipartParser) parseMultipartStructureWithPartID(part *MIMEPart, result *ParsedEmail, parentPartID string) error {
	boundary := part.Type.GetBoundary()
	if boundary == "" {
		return fmt.Errorf("multipart missing boundary")
	}

	// 使用multipart.Reader解析 - 从字节数组创建Reader以避免编码问题
	reader := multipart.NewReader(bytes.NewReader(part.Content), boundary)
	if reader == nil {
		return fmt.Errorf("failed to create multipart reader")
	}

	partIndex := 0
	for {
		multipartPart, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			// 检查是否为multipart数据不完整导致的EOF错误
			if strings.Contains(err.Error(), "multipart: NextPart: EOF") {
				log.Printf("Warning: multipart data appears to be truncated, stopping parsing: %v", err)
				break // 停止解析，而不是继续尝试
			}
			if !p.options.StrictMode {
				log.Printf("Warning: error reading multipart: %v", err)
				continue
			}
			return fmt.Errorf("failed to read multipart: %w", err)
		}

		partIndex++

		// 生成正确的层级化PartID
		var currentPartID string
		if parentPartID == "" {
			// 根级别的part
			currentPartID = fmt.Sprintf("%d", partIndex)
		} else {
			// 嵌套级别的part
			currentPartID = fmt.Sprintf("%s.%d", parentPartID, partIndex)
		}

		// 处理单个部分
		mimePart, err := p.processor.ProcessPart(multipartPart, partIndex)
		multipartPart.Close()

		if err != nil {
			if !p.options.StrictMode {
				log.Printf("Warning: failed to process part %d: %v", partIndex, err)
				continue
			}
			return fmt.Errorf("failed to process part %d: %w", partIndex, err)
		}

		// 设置正确的PartID
		mimePart.PartID = currentPartID

		// 添加到父部分
		if p.options.PreserveMIMEStructure {
			part.Parts = append(part.Parts, mimePart)
		}

		// 递归处理嵌套结构
		err = p.processNestedPartWithPartID(mimePart, result, currentPartID)
		if err != nil && p.options.StrictMode {
			return err
		}
	}

	return nil
}

// processNestedPart 处理嵌套部分（兼容性方法）
func (p *NestedMultipartParser) processNestedPart(part *MIMEPart, result *ParsedEmail) error {
	return p.processNestedPartWithPartID(part, result, "")
}

// processNestedPartWithPartID 处理嵌套部分，正确处理PartID层级
func (p *NestedMultipartParser) processNestedPartWithPartID(part *MIMEPart, result *ParsedEmail, currentPartID string) error {
	if part.Type.IsMultipart() {
		// 递归处理嵌套的multipart
		return p.parseMultipartStructureWithPartID(part, result, currentPartID)
	}

	// 检查是否为附件
	if p.processor.IsAttachmentPart(part) {
		attachment := p.processor.ExtractAttachmentInfo(part)
		if attachment != nil {
			if p.options.IncludeAttachmentContent {
				content, err := p.processor.DecodePartContent(part)
				if err == nil {
					attachment.Content = content
				}
			}
			result.Attachments = append(result.Attachments, attachment)
		}
		return nil
	}

	// 检查是否为内联附件
	if p.processor.IsInlinePart(part) {
		attachment := p.processor.ExtractAttachmentInfo(part)
		if attachment != nil {
			if p.options.IncludeAttachmentContent {
				content, err := p.processor.DecodePartContent(part)
				if err == nil {
					attachment.Content = content
				}
			}
			result.InlineAttachments = append(result.InlineAttachments, attachment)
		}
		return nil
	}

	// 处理文本内容
	if part.Type.IsPlainText() && result.TextBody == "" {
		content, err := p.processor.DecodePartContentAsString(part)
		if err != nil {
			log.Printf("Warning: failed to decode text content: %v, trying fallback strategies", err)
			// 使用回退策略尝试解码
			if fallbackContent, fallbackErr := p.processor.encodingHelper.DecodeWithFallbackStrategies(
				part.Content, part.TransferEncoding, part.Type.GetCharset()); fallbackErr == nil {
				result.TextBody = string(fallbackContent)
			} else {
				log.Printf("Warning: all text decoding strategies failed: %v", fallbackErr)
			}
		} else {
			result.TextBody = content
		}
	} else if part.Type.IsHTML() && result.HTMLBody == "" {
		content, err := p.processor.DecodePartContentAsString(part)
		if err != nil {
			log.Printf("Warning: failed to decode HTML content: %v, trying fallback strategies", err)
			// 使用回退策略尝试解码
			if fallbackContent, fallbackErr := p.processor.encodingHelper.DecodeWithFallbackStrategies(
				part.Content, part.TransferEncoding, part.Type.GetCharset()); fallbackErr == nil {
				result.HTMLBody = string(fallbackContent)
			} else {
				log.Printf("Warning: all HTML decoding strategies failed: %v", fallbackErr)
			}
		} else {
			result.HTMLBody = content
		}
	}

	return nil
}

// parseSinglePart 解析单部分邮件
func (p *NestedMultipartParser) parseSinglePart(part *MIMEPart, result *ParsedEmail) error {
	if part.Type.IsPlainText() {
		content, err := p.processor.DecodePartContentAsString(part)
		if err != nil {
			log.Printf("Warning: failed to decode single part text content: %v, trying fallback strategies", err)
			// 使用回退策略尝试解码
			if fallbackContent, fallbackErr := p.processor.encodingHelper.DecodeWithFallbackStrategies(
				part.Content, part.TransferEncoding, part.Type.GetCharset()); fallbackErr == nil {
				result.TextBody = string(fallbackContent)
			} else {
				return fmt.Errorf("all single part text decoding strategies failed: %w", fallbackErr)
			}
		} else {
			result.TextBody = content
		}
	} else if part.Type.IsHTML() {
		content, err := p.processor.DecodePartContentAsString(part)
		if err != nil {
			log.Printf("Warning: failed to decode single part HTML content: %v, trying fallback strategies", err)
			// 使用回退策略尝试解码
			if fallbackContent, fallbackErr := p.processor.encodingHelper.DecodeWithFallbackStrategies(
				part.Content, part.TransferEncoding, part.Type.GetCharset()); fallbackErr == nil {
				result.HTMLBody = string(fallbackContent)
			} else {
				return fmt.Errorf("all single part HTML decoding strategies failed: %w", fallbackErr)
			}
		} else {
			result.HTMLBody = content
		}
	}

	return nil
}

// parseSimpleEmail 简单邮件解析（回退方案）
func (p *NestedMultipartParser) parseSimpleEmail(rawEmail string) (*ParsedEmail, error) {
	// 分离头部和正文
	headerEndIndex := strings.Index(rawEmail, "\r\n\r\n")
	var content string

	if headerEndIndex != -1 {
		content = rawEmail[headerEndIndex+4:]
	} else {
		headerEndIndex = strings.Index(rawEmail, "\n\n")
		if headerEndIndex != -1 {
			content = rawEmail[headerEndIndex+2:]
		} else {
			content = rawEmail
		}
	}

	// 简单地将内容作为文本正文
	return &ParsedEmail{
		TextBody: content,
	}, nil
}

// ParseNestedMultipart 解析嵌套multipart内容
func (p *NestedMultipartParser) ParseNestedMultipart(content, boundary string) (*ParsedEmail, error) {
	// 创建临时的multipart部分
	tempPart := &MIMEPart{
		Type: &MIMEType{
			MainType:   "multipart",
			SubType:    "mixed",
			FullType:   "multipart/mixed",
			Parameters: map[string]string{"boundary": boundary},
		},
		Content: []byte(content),
		Index:   0,
	}

	result := &ParsedEmail{}
	err := p.parseMultipartStructure(tempPart, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetParseStrategy 根据MIME类型获取解析策略
func (p *NestedMultipartParser) GetParseStrategy(mimeType *MIMEType) string {
	if mimeType == nil {
		return "simple"
	}

	if mimeType.IsAlternative() {
		return "alternative"
	}

	if mimeType.IsRelated() {
		return "related"
	}

	if mimeType.IsMixed() {
		return "mixed"
	}

	if mimeType.IsMultipart() {
		return "multipart"
	}

	return "simple"
}

// 全局默认解析器实例
var defaultParser *NestedMultipartParser

// GetDefaultParser 获取默认解析器
func GetDefaultParser() *NestedMultipartParser {
	if defaultParser == nil {
		defaultParser = NewNestedMultipartParser(DefaultParseOptions())
	}
	return defaultParser
}

// 便利函数

// ParseEmailContent 解析邮件内容
func ParseEmailContent(rawEmail string) (*ParsedEmail, error) {
	return GetDefaultParser().ParseEmail(rawEmail)
}

// ParseEmailContentFromBytes 解析邮件内容（从字节数组）
func ParseEmailContentFromBytes(rawEmailBytes []byte) (*ParsedEmail, error) {
	return GetDefaultParser().ParseEmailFromBytes(rawEmailBytes)
}

// ParseMultipartContent 解析multipart内容
func ParseMultipartContent(content, boundary string) (*ParsedEmail, error) {
	return GetDefaultParser().ParseNestedMultipart(content, boundary)
}
