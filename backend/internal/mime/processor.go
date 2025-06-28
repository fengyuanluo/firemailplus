package mime

import (
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/textproto"
	"regexp"
	"strings"

	"firemail/internal/encoding"
)

// MIMEPartProcessor MIME部分处理器
type MIMEPartProcessor struct {
	detector      *MIMETypeDetector
	encodingHelper *encoding.EmailEncodingHelper
}

// NewMIMEPartProcessor 创建MIME部分处理器
func NewMIMEPartProcessor() *MIMEPartProcessor {
	return &MIMEPartProcessor{
		detector:       NewMIMETypeDetector(),
		encodingHelper: encoding.NewEmailEncodingHelper(),
	}
}

// ProcessPart 处理单个MIME部分
func (p *MIMEPartProcessor) ProcessPart(part *multipart.Part, partIndex int) (*MIMEPart, error) {
	if part == nil {
		return nil, fmt.Errorf("part is nil")
	}

	// 读取部分内容
	content, err := io.ReadAll(part)
	if err != nil {
		return nil, fmt.Errorf("failed to read part content: %w", err)
	}

	// 检测MIME类型
	mimeType, err := p.detector.DetectFromHeaders(part.Header)
	if err != nil {
		log.Printf("Warning: failed to detect MIME type: %v", err)
		mimeType = p.detector.createDefaultMIMEType()
	}

	// 解析Content-Disposition
	disposition := p.parseContentDisposition(part.Header)

	// 获取传输编码
	transferEncoding := part.Header.Get("Content-Transfer-Encoding")

	// 创建MIME部分对象
	mimePart := &MIMEPart{
		Headers:          part.Header,
		Type:             mimeType,
		Content:          content,
		TransferEncoding: transferEncoding,
		Disposition:      disposition,
		Index:            partIndex,
		PartID:           "", // PartID将在解析器中设置
	}

	return mimePart, nil
}

// ProcessPartFromRaw 从原始数据处理MIME部分
func (p *MIMEPartProcessor) ProcessPartFromRaw(rawHeaders, rawContent string, partIndex int) (*MIMEPart, error) {
	// 解析头部
	headers := p.parseRawHeaders(rawHeaders)

	// 检测MIME类型
	mimeType, err := p.detector.DetectFromHeaders(headers)
	if err != nil {
		log.Printf("Warning: failed to detect MIME type: %v", err)
		mimeType = p.detector.createDefaultMIMEType()
	}

	// 解析Content-Disposition
	disposition := p.parseContentDisposition(headers)

	// 获取传输编码
	transferEncoding := headers.Get("Content-Transfer-Encoding")

	// 创建MIME部分对象
	mimePart := &MIMEPart{
		Headers:          headers,
		Type:             mimeType,
		Content:          []byte(rawContent),
		TransferEncoding: transferEncoding,
		Disposition:      disposition,
		Index:            partIndex,
		PartID:           "", // PartID将在解析器中设置
	}

	return mimePart, nil
}

// DecodePartContent 解码部分内容
func (p *MIMEPartProcessor) DecodePartContent(part *MIMEPart) ([]byte, error) {
	if part == nil {
		return nil, fmt.Errorf("part is nil")
	}

	if len(part.Content) == 0 {
		return part.Content, nil
	}

	// 获取字符编码
	charset := part.Type.GetCharset()

	// 使用完整的解码流程
	decoded, err := p.encodingHelper.DecodeEmailContentWithTransferEncoding(
		part.Content,
		part.TransferEncoding,
		charset,
	)
	if err != nil {
		log.Printf("Warning: failed to decode part content: %v", err)
		// 如果解码失败，返回原内容
		return part.Content, nil
	}

	return decoded, nil
}

// DecodePartContentAsString 解码部分内容为字符串
func (p *MIMEPartProcessor) DecodePartContentAsString(part *MIMEPart) (string, error) {
	decoded, err := p.DecodePartContent(part)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// IsTextPart 检查是否为文本部分
func (p *MIMEPartProcessor) IsTextPart(part *MIMEPart) bool {
	if part == nil || part.Type == nil {
		return false
	}
	return part.Type.IsText()
}

// IsAttachmentPart 检查是否为附件部分
func (p *MIMEPartProcessor) IsAttachmentPart(part *MIMEPart) bool {
	if part == nil || part.Disposition == nil {
		return false
	}
	return part.Disposition.IsAttachment()
}

// IsInlinePart 检查是否为内联部分
func (p *MIMEPartProcessor) IsInlinePart(part *MIMEPart) bool {
	if part == nil || part.Disposition == nil {
		return false
	}
	return part.Disposition.IsInline()
}

// ExtractAttachmentInfo 提取附件信息
func (p *MIMEPartProcessor) ExtractAttachmentInfo(part *MIMEPart) *AttachmentInfo {
	if part == nil {
		return nil
	}

	attachment := &AttachmentInfo{
		PartID:      part.PartID, // 使用正确的层级化PartID
		ContentType: part.Type.FullType,
		Size:        int64(len(part.Content)),
		Encoding:    part.TransferEncoding,
	}

	// 获取文件名
	if part.Disposition != nil {
		attachment.Filename = part.Disposition.GetFilename()
		attachment.Disposition = part.Disposition.Type
	}

	// 获取Content-ID（用于内联附件）
	contentID := part.Headers.Get("Content-ID")
	if contentID != "" {
		// 移除尖括号
		contentID = strings.Trim(contentID, "<>")
		attachment.ContentID = contentID
	}

	// 如果没有从Disposition获取到文件名，尝试从Content-Type获取
	if attachment.Filename == "" {
		if filename := p.getFilenameFromContentType(part.Headers.Get("Content-Type")); filename != "" {
			attachment.Filename = filename
		}
	}

	return attachment
}

// parseContentDisposition 解析Content-Disposition头部
func (p *MIMEPartProcessor) parseContentDisposition(headers textproto.MIMEHeader) *DispositionInfo {
	disposition := headers.Get("Content-Disposition")
	if disposition == "" {
		return nil
	}

	// 使用标准库解析
	dispositionType, params, err := mime.ParseMediaType(disposition)
	if err != nil {
		log.Printf("Warning: failed to parse Content-Disposition: %v", err)
		return nil
	}

	// 解码RFC 2047编码的参数
	decodedParams := make(map[string]string)
	for key, value := range params {
		if strings.Contains(value, "=?") {
			dec := new(mime.WordDecoder)
			if decoded, err := dec.DecodeHeader(value); err == nil {
				value = decoded
			}
		}
		decodedParams[strings.ToLower(key)] = value
	}

	return &DispositionInfo{
		Type:       strings.ToLower(dispositionType),
		Parameters: decodedParams,
	}
}

// parseRawHeaders 解析原始头部字符串
func (p *MIMEPartProcessor) parseRawHeaders(rawHeaders string) textproto.MIMEHeader {
	headers := make(textproto.MIMEHeader)

	// 分割头部行
	lines := strings.Split(rawHeaders, "\n")
	var currentKey string
	var currentValue strings.Builder

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")

		// 检查是否为续行（以空格或制表符开头）
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			if currentKey != "" {
				currentValue.WriteString(" ")
				currentValue.WriteString(strings.TrimSpace(line))
			}
			continue
		}

		// 保存前一个头部
		if currentKey != "" {
			headers.Add(currentKey, currentValue.String())
		}

		// 解析新的头部
		colonIndex := strings.Index(line, ":")
		if colonIndex == -1 {
			currentKey = ""
			currentValue.Reset()
			continue
		}

		currentKey = strings.TrimSpace(line[:colonIndex])
		currentValue.Reset()
		currentValue.WriteString(strings.TrimSpace(line[colonIndex+1:]))
	}

	// 保存最后一个头部
	if currentKey != "" {
		headers.Add(currentKey, currentValue.String())
	}

	return headers
}

// getFilenameFromContentType 从Content-Type中获取文件名
func (p *MIMEPartProcessor) getFilenameFromContentType(contentType string) string {
	if contentType == "" {
		return ""
	}

	// 使用正则表达式查找name参数
	nameRegex := regexp.MustCompile(`(?i)name\s*=\s*"?([^";]+)"?`)
	matches := nameRegex.FindStringSubmatch(contentType)
	if len(matches) > 1 {
		filename := strings.TrimSpace(matches[1])
		// 解码RFC 2047编码的文件名
		if strings.Contains(filename, "=?") {
			dec := new(mime.WordDecoder)
			if decoded, err := dec.DecodeHeader(filename); err == nil {
				filename = decoded
			}
		}
		return filename
	}

	return ""
}

// 全局默认处理器实例
var defaultProcessor *MIMEPartProcessor

// GetDefaultProcessor 获取默认处理器
func GetDefaultProcessor() *MIMEPartProcessor {
	if defaultProcessor == nil {
		defaultProcessor = NewMIMEPartProcessor()
	}
	return defaultProcessor
}

// 便利函数

// ProcessMultipartPart 处理multipart部分
func ProcessMultipartPart(part *multipart.Part, partIndex int) (*MIMEPart, error) {
	return GetDefaultProcessor().ProcessPart(part, partIndex)
}

// DecodeContent 解码内容
func DecodeContent(part *MIMEPart) ([]byte, error) {
	return GetDefaultProcessor().DecodePartContent(part)
}

// DecodeContentAsString 解码内容为字符串
func DecodeContentAsString(part *MIMEPart) (string, error) {
	return GetDefaultProcessor().DecodePartContentAsString(part)
}
