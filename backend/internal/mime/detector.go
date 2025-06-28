package mime

import (
	"fmt"
	"mime"
	"net/textproto"
	"regexp"
	"strings"
)

// MIMETypeDetector MIME类型检测器
type MIMETypeDetector struct {
	// 缓存解析结果
	cache map[string]*MIMEType
}

// NewMIMETypeDetector 创建MIME类型检测器
func NewMIMETypeDetector() *MIMETypeDetector {
	return &MIMETypeDetector{
		cache: make(map[string]*MIMEType),
	}
}

// DetectFromContentType 从Content-Type头部检测MIME类型
func (d *MIMETypeDetector) DetectFromContentType(contentType string) (*MIMEType, error) {
	if contentType == "" {
		return d.createDefaultMIMEType(), nil
	}

	// 检查缓存
	if cached, exists := d.cache[contentType]; exists {
		return cached, nil
	}

	// 清理Content-Type值（处理多行情况）
	cleanContentType := d.cleanContentType(contentType)

	// 使用标准库解析
	mediaType, params, err := mime.ParseMediaType(cleanContentType)
	if err != nil {
		// 如果标准库解析失败，尝试手动解析
		return d.parseContentTypeManually(cleanContentType)
	}

	// 创建MIMEType对象
	mimeType := d.createMIMEType(mediaType, params)

	// 缓存结果
	d.cache[contentType] = mimeType

	return mimeType, nil
}

// DetectFromHeaders 从邮件头部检测MIME类型
func (d *MIMETypeDetector) DetectFromHeaders(headers textproto.MIMEHeader) (*MIMEType, error) {
	contentType := headers.Get("Content-Type")
	return d.DetectFromContentType(contentType)
}

// DetectFromRawHeaders 从原始头部字符串检测MIME类型
func (d *MIMETypeDetector) DetectFromRawHeaders(rawHeaders string) (*MIMEType, error) {
	// 使用正则表达式提取Content-Type（支持多行）
	contentTypeRegex := regexp.MustCompile(`(?i)Content-Type:\s*([^\r\n]+(?:\r?\n\s+[^\r\n]+)*)`)
	matches := contentTypeRegex.FindStringSubmatch(rawHeaders)

	if len(matches) < 2 {
		return d.createDefaultMIMEType(), nil
	}

	return d.DetectFromContentType(matches[1])
}

// cleanContentType 清理Content-Type值
func (d *MIMETypeDetector) cleanContentType(contentType string) string {
	// 移除换行符和多余空格
	cleaned := strings.ReplaceAll(contentType, "\r\n", "")
	cleaned = strings.ReplaceAll(cleaned, "\n", "")
	cleaned = regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " ")
	return strings.TrimSpace(cleaned)
}

// parseContentTypeManually 手动解析Content-Type
func (d *MIMETypeDetector) parseContentTypeManually(contentType string) (*MIMEType, error) {
	// 分离主类型和参数
	parts := strings.Split(contentType, ";")
	if len(parts) == 0 {
		return d.createDefaultMIMEType(), nil
	}

	// 解析主类型
	mainType := strings.TrimSpace(parts[0])
	if mainType == "" {
		return d.createDefaultMIMEType(), nil
	}

	// 解析参数
	params := make(map[string]string)
	for i := 1; i < len(parts); i++ {
		param := strings.TrimSpace(parts[i])
		if param == "" {
			continue
		}

		// 分离参数名和值
		paramParts := strings.SplitN(param, "=", 2)
		if len(paramParts) != 2 {
			continue
		}

		key := strings.TrimSpace(paramParts[0])
		value := strings.TrimSpace(paramParts[1])

		// 移除引号
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = value[1 : len(value)-1]
		}

		params[strings.ToLower(key)] = value
	}

	return d.createMIMEType(mainType, params), nil
}

// createMIMEType 创建MIMEType对象
func (d *MIMETypeDetector) createMIMEType(mediaType string, params map[string]string) *MIMEType {
	// 标准化媒体类型
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))

	// 分离主类型和子类型
	typeParts := strings.Split(mediaType, "/")
	var mainType, subType string

	if len(typeParts) >= 1 {
		mainType = strings.TrimSpace(typeParts[0])
	}
	if len(typeParts) >= 2 {
		subType = strings.TrimSpace(typeParts[1])
	}

	// 标准化参数键名
	normalizedParams := make(map[string]string)
	for key, value := range params {
		normalizedParams[strings.ToLower(key)] = value
	}

	return &MIMEType{
		MainType:   mainType,
		SubType:    subType,
		FullType:   mediaType,
		Parameters: normalizedParams,
	}
}

// createDefaultMIMEType 创建默认MIME类型
func (d *MIMETypeDetector) createDefaultMIMEType() *MIMEType {
	return &MIMEType{
		MainType:   "text",
		SubType:    "plain",
		FullType:   "text/plain",
		Parameters: make(map[string]string),
	}
}

// IsMultipartType 检查是否为multipart类型
func (d *MIMETypeDetector) IsMultipartType(contentType string) bool {
	mimeType, err := d.DetectFromContentType(contentType)
	if err != nil {
		return false
	}
	return mimeType.IsMultipart()
}

// IsTextType 检查是否为text类型
func (d *MIMETypeDetector) IsTextType(contentType string) bool {
	mimeType, err := d.DetectFromContentType(contentType)
	if err != nil {
		return false
	}
	return mimeType.IsText()
}

// GetBoundaryFromContentType 从Content-Type中提取boundary
func (d *MIMETypeDetector) GetBoundaryFromContentType(contentType string) string {
	mimeType, err := d.DetectFromContentType(contentType)
	if err != nil {
		return ""
	}
	return mimeType.GetBoundary()
}

// GetCharsetFromContentType 从Content-Type中提取charset
func (d *MIMETypeDetector) GetCharsetFromContentType(contentType string) string {
	mimeType, err := d.DetectFromContentType(contentType)
	if err != nil {
		return ""
	}
	return mimeType.GetCharset()
}

// ValidateMIMEType 验证MIME类型是否有效
func (d *MIMETypeDetector) ValidateMIMEType(mimeType *MIMEType) error {
	if mimeType == nil {
		return fmt.Errorf("MIME type is nil")
	}

	if mimeType.MainType == "" {
		return fmt.Errorf("main type is empty")
	}

	if mimeType.SubType == "" {
		return fmt.Errorf("sub type is empty")
	}

	if mimeType.FullType == "" {
		return fmt.Errorf("full type is empty")
	}

	// 检查multipart类型是否有boundary
	if mimeType.IsMultipart() && mimeType.GetBoundary() == "" {
		return fmt.Errorf("multipart type missing boundary parameter")
	}

	return nil
}

// ClearCache 清空缓存
func (d *MIMETypeDetector) ClearCache() {
	d.cache = make(map[string]*MIMEType)
}

// GetCacheSize 获取缓存大小
func (d *MIMETypeDetector) GetCacheSize() int {
	return len(d.cache)
}

// 全局默认检测器实例
var defaultDetector *MIMETypeDetector

// GetDefaultDetector 获取默认检测器
func GetDefaultDetector() *MIMETypeDetector {
	if defaultDetector == nil {
		defaultDetector = NewMIMETypeDetector()
	}
	return defaultDetector
}

// 便利函数

// DetectMIMEType 检测MIME类型
func DetectMIMEType(contentType string) (*MIMEType, error) {
	return GetDefaultDetector().DetectFromContentType(contentType)
}

// IsMultipart 检查是否为multipart类型
func IsMultipart(contentType string) bool {
	return GetDefaultDetector().IsMultipartType(contentType)
}

// IsText 检查是否为text类型
func IsText(contentType string) bool {
	return GetDefaultDetector().IsTextType(contentType)
}

// GetBoundary 获取boundary参数
func GetBoundary(contentType string) string {
	return GetDefaultDetector().GetBoundaryFromContentType(contentType)
}

// GetCharset 获取charset参数
func GetCharset(contentType string) string {
	return GetDefaultDetector().GetCharsetFromContentType(contentType)
}
