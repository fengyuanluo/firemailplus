package encoding

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"

	"firemail/internal/encoding/transfer"
)

// StandardEncodingConverter 标准编码转换器
type StandardEncodingConverter struct {
	detector  EncodingDetector
	encodings map[string]encoding.Encoding
}

// NewStandardEncodingConverter 创建标准编码转换器
func NewStandardEncodingConverter(detector EncodingDetector) EncodingConverter {
	converter := &StandardEncodingConverter{
		detector:  detector,
		encodings: make(map[string]encoding.Encoding),
	}
	
	converter.initializeEncodings()
	return converter
}

// initializeEncodings 初始化编码映射
func (c *StandardEncodingConverter) initializeEncodings() {
	// 复用检测器中的编码映射
	if standardDetector, ok := c.detector.(*StandardEncodingDetector); ok {
		c.encodings = standardDetector.encodings
	}
}

// ConvertToUTF8 转换为UTF-8
func (c *StandardEncodingConverter) ConvertToUTF8(data []byte, sourceEncoding string) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	
	// 如果已经是UTF-8，直接返回
	if strings.ToLower(sourceEncoding) == "utf-8" {
		if utf8.Valid(data) {
			return data, nil
		}
		return nil, fmt.Errorf("invalid UTF-8 data")
	}
	
	// 获取源编码
	enc, err := c.getEncoding(sourceEncoding)
	if err != nil {
		return nil, fmt.Errorf("unsupported source encoding %s: %w", sourceEncoding, err)
	}
	
	// 创建解码器
	decoder := enc.NewDecoder()
	
	// 执行转换
	result, err := c.transformData(data, decoder)
	if err != nil {
		return nil, fmt.Errorf("failed to convert from %s to UTF-8: %w", sourceEncoding, err)
	}
	
	// 验证结果是否为有效UTF-8
	if !utf8.Valid(result) {
		return nil, fmt.Errorf("conversion result is not valid UTF-8")
	}
	
	return result, nil
}

// ConvertFromUTF8 从UTF-8转换
func (c *StandardEncodingConverter) ConvertFromUTF8(data []byte, targetEncoding string) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	
	// 验证输入是否为有效UTF-8
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("input data is not valid UTF-8")
	}
	
	// 如果目标编码是UTF-8，直接返回
	if strings.ToLower(targetEncoding) == "utf-8" {
		return data, nil
	}
	
	// 获取目标编码
	enc, err := c.getEncoding(targetEncoding)
	if err != nil {
		return nil, fmt.Errorf("unsupported target encoding %s: %w", targetEncoding, err)
	}
	
	// 创建编码器
	encoder := enc.NewEncoder()
	
	// 执行转换
	result, err := c.transformData(data, encoder)
	if err != nil {
		return nil, fmt.Errorf("failed to convert from UTF-8 to %s: %w", targetEncoding, err)
	}
	
	return result, nil
}

// ConvertString 转换字符串
func (c *StandardEncodingConverter) ConvertString(text, sourceEncoding, targetEncoding string) (string, error) {
	if text == "" {
		return text, nil
	}
	
	// 如果源编码和目标编码相同，直接返回
	if strings.ToLower(sourceEncoding) == strings.ToLower(targetEncoding) {
		return text, nil
	}
	
	// 先转换为UTF-8
	utf8Data, err := c.ConvertToUTF8([]byte(text), sourceEncoding)
	if err != nil {
		return "", fmt.Errorf("failed to convert to UTF-8: %w", err)
	}
	
	// 如果目标编码是UTF-8，直接返回
	if strings.ToLower(targetEncoding) == "utf-8" {
		return string(utf8Data), nil
	}
	
	// 转换为目标编码
	targetData, err := c.ConvertFromUTF8(utf8Data, targetEncoding)
	if err != nil {
		return "", fmt.Errorf("failed to convert from UTF-8: %w", err)
	}
	
	return string(targetData), nil
}

// IsValidUTF8 检查是否为有效UTF-8
func (c *StandardEncodingConverter) IsValidUTF8(data []byte) bool {
	return utf8.Valid(data)
}

// getEncoding 获取编码实例
func (c *StandardEncodingConverter) getEncoding(encodingName string) (encoding.Encoding, error) {
	normalizedName := strings.ToLower(strings.ReplaceAll(encodingName, "_", "-"))
	
	// 尝试直接匹配
	if enc, exists := c.encodings[normalizedName]; exists {
		return enc, nil
	}
	
	// 尝试别名匹配
	aliases := c.getEncodingAliases()
	if canonical, exists := aliases[normalizedName]; exists {
		if enc, exists := c.encodings[canonical]; exists {
			return enc, nil
		}
	}
	
	return nil, fmt.Errorf("encoding not found: %s", encodingName)
}

// getEncodingAliases 获取编码别名映射
func (c *StandardEncodingConverter) getEncodingAliases() map[string]string {
	return map[string]string{
		"gb2312":      "gb2312",
		"gb-2312":     "gb2312",
		"gbk":         "gbk",
		"gb18030":     "gb18030",
		"gb-18030":    "gb18030",
		"big5":        "big5",
		"big-5":       "big5",
		"shift-jis":   "shift_jis",
		"shiftjis":    "shift_jis",
		"sjis":        "shift_jis",
		"euc-jp":      "euc-jp",
		"eucjp":       "euc-jp",
		"euc-kr":      "euc-kr",
		"euckr":       "euc-kr",
		"iso88591":    "iso-8859-1",
		"iso-8859-1":  "iso-8859-1",
		"latin1":      "iso-8859-1",
		"windows1252": "windows-1252",
		"cp1252":      "windows-1252",
		"utf8":        "utf-8",
		"utf-8":       "utf-8",
		"utf16":       "utf-16",
		"utf-16":      "utf-16",
		"utf16le":     "utf-16le",
		"utf-16le":    "utf-16le",
		"utf16be":     "utf-16be",
		"utf-16be":    "utf-16be",
	}
}

// transformData 执行数据转换
func (c *StandardEncodingConverter) transformData(data []byte, transformer transform.Transformer) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(data), transformer)
	
	var result bytes.Buffer
	_, err := io.Copy(&result, reader)
	if err != nil {
		return nil, err
	}
	
	return result.Bytes(), nil
}

// StandardEncodingService 标准编码服务
type StandardEncodingService struct {
	detector  EncodingDetector
	converter EncodingConverter
}

// NewStandardEncodingService 创建标准编码服务
func NewStandardEncodingService() EncodingService {
	detector := NewStandardEncodingDetector()
	converter := NewStandardEncodingConverter(detector)
	
	return &StandardEncodingService{
		detector:  detector,
		converter: converter,
	}
}

// DetectEncoding 检测编码
func (s *StandardEncodingService) DetectEncoding(data []byte) (string, float64, error) {
	return s.detector.DetectEncoding(data)
}

// DetectEncodingFromString 从字符串检测编码
func (s *StandardEncodingService) DetectEncodingFromString(text string) (string, float64, error) {
	return s.detector.DetectEncodingFromString(text)
}

// GetSupportedEncodings 获取支持的编码
func (s *StandardEncodingService) GetSupportedEncodings() []string {
	return s.detector.GetSupportedEncodings()
}

// ConvertToUTF8 转换为UTF-8
func (s *StandardEncodingService) ConvertToUTF8(data []byte, sourceEncoding string) ([]byte, error) {
	return s.converter.ConvertToUTF8(data, sourceEncoding)
}

// ConvertFromUTF8 从UTF-8转换
func (s *StandardEncodingService) ConvertFromUTF8(data []byte, targetEncoding string) ([]byte, error) {
	return s.converter.ConvertFromUTF8(data, targetEncoding)
}

// ConvertString 转换字符串
func (s *StandardEncodingService) ConvertString(text, sourceEncoding, targetEncoding string) (string, error) {
	return s.converter.ConvertString(text, sourceEncoding, targetEncoding)
}

// IsValidUTF8 检查是否为有效UTF-8
func (s *StandardEncodingService) IsValidUTF8(data []byte) bool {
	return s.converter.IsValidUTF8(data)
}

// AutoConvertToUTF8 自动检测并转换为UTF-8
func (s *StandardEncodingService) AutoConvertToUTF8(data []byte) ([]byte, string, error) {
	if len(data) == 0 {
		return data, "utf-8", nil
	}
	
	// 如果已经是有效的UTF-8，直接返回
	if s.converter.IsValidUTF8(data) {
		return data, "utf-8", nil
	}
	
	// 检测编码
	detectedEncoding, confidence, err := s.detector.DetectEncoding(data)
	if err != nil {
		return nil, "", fmt.Errorf("failed to detect encoding: %w", err)
	}
	
	// 如果置信度太低，尝试常见的中文编码
	if confidence < 0.6 {
		commonEncodings := []string{"gbk", "gb2312", "big5", "gb18030"}
		for _, encoding := range commonEncodings {
			if converted, err := s.converter.ConvertToUTF8(data, encoding); err == nil {
				if s.converter.IsValidUTF8(converted) {
					return converted, encoding, nil
				}
			}
		}
	}
	
	// 使用检测到的编码进行转换
	converted, err := s.converter.ConvertToUTF8(data, detectedEncoding)
	if err != nil {
		return nil, "", fmt.Errorf("failed to convert from %s to UTF-8: %w", detectedEncoding, err)
	}
	
	return converted, detectedEncoding, nil
}

// AutoConvertStringToUTF8 自动检测并转换字符串为UTF-8
func (s *StandardEncodingService) AutoConvertStringToUTF8(text string) (string, string, error) {
	if text == "" {
		return text, "utf-8", nil
	}
	
	data := []byte(text)
	converted, detectedEncoding, err := s.AutoConvertToUTF8(data)
	if err != nil {
		return "", "", err
	}
	
	return string(converted), detectedEncoding, nil
}

// EmailEncodingHelper 邮件编码助手
type EmailEncodingHelper struct {
	service EncodingService
}

// NewEmailEncodingHelper 创建邮件编码助手
func NewEmailEncodingHelper() *EmailEncodingHelper {
	return &EmailEncodingHelper{
		service: NewStandardEncodingService(),
	}
}

// DecodeEmailSubject 解码邮件主题
func (h *EmailEncodingHelper) DecodeEmailSubject(subject string) string {
	if subject == "" {
		return subject
	}

	// 处理MIME编码的主题（如 =?UTF-8?B?...?=）
	if strings.Contains(subject, "=?") && strings.Contains(subject, "?=") {
		// 使用标准库的RFC 2047解码器
		dec := new(mime.WordDecoder)
		if decoded, err := dec.DecodeHeader(subject); err == nil && decoded != subject {
			return decoded
		}

		// 如果标准解码失败，回退到自动转换
		if converted, _, err := h.service.AutoConvertStringToUTF8(subject); err == nil {
			return converted
		}
	}

	// 尝试自动转换编码
	if converted, _, err := h.service.AutoConvertStringToUTF8(subject); err == nil {
		return converted
	}

	return subject
}

// DecodeEmailContent 解码邮件内容
func (h *EmailEncodingHelper) DecodeEmailContent(content []byte, charset string) ([]byte, error) {
	if len(content) == 0 {
		return content, nil
	}
	
	// 如果指定了字符集，使用指定的字符集转换
	if charset != "" && strings.ToLower(charset) != "utf-8" {
		return h.service.ConvertToUTF8(content, charset)
	}
	
	// 自动检测并转换
	converted, _, err := h.service.AutoConvertToUTF8(content)
	return converted, err
}

// DecodeEmailFrom 解码发件人信息
func (h *EmailEncodingHelper) DecodeEmailFrom(from string) string {
	return h.DecodeEmailSubject(from) // 使用相同的解码逻辑
}

// DecodeTransferEncoding 解码Content-Transfer-Encoding
func (h *EmailEncodingHelper) DecodeTransferEncoding(content []byte, transferEncoding string) ([]byte, error) {
	if len(content) == 0 {
		return content, nil
	}

	// 使用新的模块化解码器系统
	decoded, err := transfer.DecodeWithFallback(content, transferEncoding)
	if err != nil {
		return nil, fmt.Errorf("failed to decode transfer encoding %s: %w", transferEncoding, err)
	}

	return decoded, nil
}

// DecodeEmailContentWithTransferEncoding 解码邮件内容（包含传输编码和字符编码）
func (h *EmailEncodingHelper) DecodeEmailContentWithTransferEncoding(content []byte, transferEncoding, charset string) ([]byte, error) {
	if len(content) == 0 {
		return content, nil
	}

	// 第一步：解码Content-Transfer-Encoding
	decoded, err := h.DecodeTransferEncoding(content, transferEncoding)
	if err != nil {
		return nil, fmt.Errorf("transfer encoding decode failed: %w", err)
	}

	// 第二步：处理字符编码转换
	if charset != "" && strings.ToLower(charset) != "utf-8" {
		// 使用指定的字符集转换
		converted, err := h.service.ConvertToUTF8(decoded, charset)
		if err != nil {
			return nil, fmt.Errorf("charset conversion failed: %w", err)
		}
		return converted, nil
	}

	// 自动检测并转换字符编码
	converted, _, err := h.service.AutoConvertToUTF8(decoded)
	if err != nil {
		return nil, fmt.Errorf("auto charset conversion failed: %w", err)
	}

	return converted, nil
}

// DecodeEmailContentComplete 完整的邮件内容解码（推荐使用）
// 这个方法提供了最完整的解码流程，包括传输编码和字符编码
func (h *EmailEncodingHelper) DecodeEmailContentComplete(content []byte, transferEncoding, charset string) ([]byte, error) {
	return h.DecodeEmailContentWithTransferEncoding(content, transferEncoding, charset)
}

// DecodeWithFallbackStrategies 使用多种策略尝试解码内容
// 当标准解码失败时，尝试所有可能的编码组合
func (h *EmailEncodingHelper) DecodeWithFallbackStrategies(content []byte, transferEncoding, charset string) ([]byte, error) {
	if len(content) == 0 {
		return content, nil
	}

	// 首先尝试标准解码流程
	if decoded, err := h.DecodeEmailContentWithTransferEncoding(content, transferEncoding, charset); err == nil {
		return decoded, nil
	}

	// 如果标准解码失败，尝试各种编码组合
	transferEncodings := []string{"quoted-printable", "base64", "7bit", "8bit", "binary"}
	charsets := []string{"utf-8", "gbk", "gb2312", "big5", "gb18030", "iso-8859-1"}

	// 尝试不同的传输编码
	for _, te := range transferEncodings {
		if decoded, err := h.DecodeTransferEncoding(content, te); err == nil {
			// 尝试不同的字符编码
			for _, cs := range charsets {
				if cs == "utf-8" && h.service.IsValidUTF8(decoded) {
					return decoded, nil
				}
				if converted, err := h.service.ConvertToUTF8(decoded, cs); err == nil && h.service.IsValidUTF8(converted) {
					return converted, nil
				}
			}
			// 如果字符编码转换都失败，但传输编码成功，返回解码后的内容
			if h.service.IsValidUTF8(decoded) {
				return decoded, nil
			}
		}
	}

	// 如果所有解码都失败，返回原内容
	return content, nil
}
