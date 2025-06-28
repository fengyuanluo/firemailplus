package parser

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime/quotedprintable"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// EncodingProcessor 统一的编码处理器
// 支持所有标准的Content-Transfer-Encoding和字符编码转换
type EncodingProcessor struct {
	// 字符编码映射
	charsetMap map[string]encoding.Encoding
}

// NewEncodingProcessor 创建编码处理器
func NewEncodingProcessor() *EncodingProcessor {
	processor := &EncodingProcessor{
		charsetMap: make(map[string]encoding.Encoding),
	}
	processor.initCharsetMap()
	return processor
}

// initCharsetMap 初始化字符编码映射
func (p *EncodingProcessor) initCharsetMap() {
	// UTF编码
	p.charsetMap["utf-8"] = unicode.UTF8
	p.charsetMap["utf8"] = unicode.UTF8
	p.charsetMap["utf-16"] = unicode.UTF16(unicode.LittleEndian, unicode.UseBOM)
	p.charsetMap["utf-16le"] = unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	p.charsetMap["utf-16be"] = unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)

	// ASCII和Latin编码
	p.charsetMap["ascii"] = charmap.Windows1252 // ASCII兼容
	p.charsetMap["us-ascii"] = charmap.Windows1252
	p.charsetMap["iso-8859-1"] = charmap.ISO8859_1
	p.charsetMap["iso-8859-2"] = charmap.ISO8859_2
	p.charsetMap["iso-8859-3"] = charmap.ISO8859_3
	p.charsetMap["iso-8859-4"] = charmap.ISO8859_4
	p.charsetMap["iso-8859-5"] = charmap.ISO8859_5
	p.charsetMap["iso-8859-6"] = charmap.ISO8859_6
	p.charsetMap["iso-8859-7"] = charmap.ISO8859_7
	p.charsetMap["iso-8859-8"] = charmap.ISO8859_8
	p.charsetMap["iso-8859-9"] = charmap.ISO8859_9
	p.charsetMap["iso-8859-10"] = charmap.ISO8859_10
	p.charsetMap["iso-8859-13"] = charmap.ISO8859_13
	p.charsetMap["iso-8859-14"] = charmap.ISO8859_14
	p.charsetMap["iso-8859-15"] = charmap.ISO8859_15
	p.charsetMap["iso-8859-16"] = charmap.ISO8859_16

	// Windows编码
	p.charsetMap["windows-1250"] = charmap.Windows1250
	p.charsetMap["windows-1251"] = charmap.Windows1251
	p.charsetMap["windows-1252"] = charmap.Windows1252
	p.charsetMap["windows-1253"] = charmap.Windows1253
	p.charsetMap["windows-1254"] = charmap.Windows1254
	p.charsetMap["windows-1255"] = charmap.Windows1255
	p.charsetMap["windows-1256"] = charmap.Windows1256
	p.charsetMap["windows-1257"] = charmap.Windows1257
	p.charsetMap["windows-1258"] = charmap.Windows1258

	// 中文编码
	p.charsetMap["gb2312"] = simplifiedchinese.HZGB2312
	p.charsetMap["gbk"] = simplifiedchinese.GBK
	p.charsetMap["gb18030"] = simplifiedchinese.GB18030
	p.charsetMap["big5"] = traditionalchinese.Big5

	// 日文编码
	p.charsetMap["shift_jis"] = japanese.ShiftJIS
	p.charsetMap["shift-jis"] = japanese.ShiftJIS
	p.charsetMap["sjis"] = japanese.ShiftJIS
	p.charsetMap["iso-2022-jp"] = japanese.ISO2022JP
	p.charsetMap["euc-jp"] = japanese.EUCJP

	// 韩文编码
	p.charsetMap["euc-kr"] = korean.EUCKR

	// KOI8编码
	p.charsetMap["koi8-r"] = charmap.KOI8R
	p.charsetMap["koi8-u"] = charmap.KOI8U
}

// DecodeTransferEncoding 解码Content-Transfer-Encoding
func (p *EncodingProcessor) DecodeTransferEncoding(content []byte, encoding string) ([]byte, error) {
	if len(content) == 0 {
		return content, nil
	}

	encoding = strings.ToLower(strings.TrimSpace(encoding))

	switch encoding {
	case "", "7bit", "8bit", "binary":
		// 无需解码
		return content, nil
	case "quoted-printable":
		return p.decodeQuotedPrintable(content)
	case "base64":
		return p.decodeBase64(content)
	default:
		return content, fmt.Errorf("unsupported transfer encoding: %s", encoding)
	}
}

// decodeQuotedPrintable 解码quoted-printable编码
func (p *EncodingProcessor) decodeQuotedPrintable(content []byte) ([]byte, error) {
	reader := quotedprintable.NewReader(bytes.NewReader(content))
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return content, fmt.Errorf("failed to decode quoted-printable: %w", err)
	}
	return decoded, nil
}

// decodeBase64 解码base64编码
func (p *EncodingProcessor) decodeBase64(content []byte) ([]byte, error) {
	// 清理base64内容（移除空白字符）
	cleaned := p.cleanBase64Content(content)
	
	decoded, err := base64.StdEncoding.DecodeString(string(cleaned))
	if err != nil {
		return content, fmt.Errorf("failed to decode base64: %w", err)
	}
	return decoded, nil
}

// cleanBase64Content 清理base64内容
func (p *EncodingProcessor) cleanBase64Content(content []byte) []byte {
	var result []byte
	for _, b := range content {
		// 保留base64有效字符
		if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || 
		   (b >= '0' && b <= '9') || b == '+' || b == '/' || b == '=' {
			result = append(result, b)
		}
	}
	return result
}

// ConvertCharset 转换字符编码为UTF-8
func (p *EncodingProcessor) ConvertCharset(content []byte, charset string) ([]byte, error) {
	if len(content) == 0 {
		return content, nil
	}

	charset = strings.ToLower(strings.TrimSpace(charset))
	
	// 如果已经是UTF-8，直接返回
	if charset == "" || charset == "utf-8" || charset == "utf8" {
		return content, nil
	}

	// 查找编码器
	enc, ok := p.charsetMap[charset]
	if !ok {
		return content, fmt.Errorf("unsupported charset: %s", charset)
	}

	// 转换为UTF-8
	decoder := enc.NewDecoder()
	decoded, _, err := transform.Bytes(decoder, content)
	if err != nil {
		return content, fmt.Errorf("failed to convert charset %s to UTF-8: %w", charset, err)
	}

	return decoded, nil
}

// DecodeEmailContent 完整的邮件内容解码
// 先解码传输编码，再转换字符编码
func (p *EncodingProcessor) DecodeEmailContent(content []byte, transferEncoding, charset string) ([]byte, error) {
	if len(content) == 0 {
		return content, nil
	}

	// 第一步：解码传输编码
	decoded, err := p.DecodeTransferEncoding(content, transferEncoding)
	if err != nil {
		return content, fmt.Errorf("transfer encoding decode failed: %w", err)
	}

	// 第二步：转换字符编码
	converted, err := p.ConvertCharset(decoded, charset)
	if err != nil {
		return decoded, fmt.Errorf("charset conversion failed: %w", err)
	}

	return converted, nil
}

// DecodeWithFallback 带回退策略的解码
func (p *EncodingProcessor) DecodeWithFallback(content []byte, transferEncoding, charset string) []byte {
	// 尝试完整解码
	decoded, err := p.DecodeEmailContent(content, transferEncoding, charset)
	if err == nil {
		return decoded
	}

	// 回退策略1：只解码传输编码
	if transferDecoded, err := p.DecodeTransferEncoding(content, transferEncoding); err == nil {
		return transferDecoded
	}

	// 回退策略2：只转换字符编码
	if charsetConverted, err := p.ConvertCharset(content, charset); err == nil {
		return charsetConverted
	}

	// 最后回退：返回原内容
	return content
}

// SupportedCharsets 返回支持的字符编码列表
func (p *EncodingProcessor) SupportedCharsets() []string {
	charsets := make([]string, 0, len(p.charsetMap))
	for charset := range p.charsetMap {
		charsets = append(charsets, charset)
	}
	return charsets
}

// SupportedTransferEncodings 返回支持的传输编码列表
func (p *EncodingProcessor) SupportedTransferEncodings() []string {
	return []string{
		"7bit",
		"8bit", 
		"binary",
		"quoted-printable",
		"base64",
	}
}

// 全局默认编码处理器实例
var defaultEncodingProcessor *EncodingProcessor

// GetDefaultEncodingProcessor 获取默认编码处理器
func GetDefaultEncodingProcessor() *EncodingProcessor {
	if defaultEncodingProcessor == nil {
		defaultEncodingProcessor = NewEncodingProcessor()
	}
	return defaultEncodingProcessor
}

// 便利函数

// DecodeTransferEncoding 使用默认处理器解码传输编码
func DecodeTransferEncoding(content []byte, encoding string) ([]byte, error) {
	return GetDefaultEncodingProcessor().DecodeTransferEncoding(content, encoding)
}

// ConvertCharset 使用默认处理器转换字符编码
func ConvertCharset(content []byte, charset string) ([]byte, error) {
	return GetDefaultEncodingProcessor().ConvertCharset(content, charset)
}

// DecodeEmailContent 使用默认处理器完整解码邮件内容
func DecodeEmailContent(content []byte, transferEncoding, charset string) ([]byte, error) {
	return GetDefaultEncodingProcessor().DecodeEmailContent(content, transferEncoding, charset)
}
