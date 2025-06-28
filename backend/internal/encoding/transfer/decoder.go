package transfer

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime/quotedprintable"
	"strings"
)

// TransferDecoder 定义Content-Transfer-Encoding解码器接口
type TransferDecoder interface {
	// Decode 解码数据
	Decode(data []byte) ([]byte, error)
	
	// GetEncodingType 获取编码类型
	GetEncodingType() string
}

// DecoderFactory 解码器工厂接口
type DecoderFactory interface {
	// CreateDecoder 根据编码类型创建解码器
	CreateDecoder(encoding string) (TransferDecoder, error)
	
	// GetSupportedEncodings 获取支持的编码类型
	GetSupportedEncodings() []string
}

// Base64Decoder Base64解码器
type Base64Decoder struct{}

// NewBase64Decoder 创建Base64解码器
func NewBase64Decoder() *Base64Decoder {
	return &Base64Decoder{}
}

// Decode 解码Base64数据
func (d *Base64Decoder) Decode(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	
	// 清理base64内容（移除换行符、回车符和空格）
	cleanContent := string(data)
	cleanContent = strings.ReplaceAll(cleanContent, "\n", "")
	cleanContent = strings.ReplaceAll(cleanContent, "\r", "")
	cleanContent = strings.ReplaceAll(cleanContent, " ", "")
	cleanContent = strings.ReplaceAll(cleanContent, "\t", "")
	
	// 解码Base64
	decoded, err := base64.StdEncoding.DecodeString(cleanContent)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}
	
	return decoded, nil
}

// GetEncodingType 获取编码类型
func (d *Base64Decoder) GetEncodingType() string {
	return "base64"
}

// QuotedPrintableDecoder Quoted-Printable解码器
type QuotedPrintableDecoder struct{}

// NewQuotedPrintableDecoder 创建Quoted-Printable解码器
func NewQuotedPrintableDecoder() *QuotedPrintableDecoder {
	return &QuotedPrintableDecoder{}
}

// Decode 解码Quoted-Printable数据
func (d *QuotedPrintableDecoder) Decode(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	
	reader := quotedprintable.NewReader(bytes.NewReader(data))
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode quoted-printable: %w", err)
	}
	
	return decoded, nil
}

// GetEncodingType 获取编码类型
func (d *QuotedPrintableDecoder) GetEncodingType() string {
	return "quoted-printable"
}

// SevenBitDecoder 7bit解码器（无需解码）
type SevenBitDecoder struct{}

// NewSevenBitDecoder 创建7bit解码器
func NewSevenBitDecoder() *SevenBitDecoder {
	return &SevenBitDecoder{}
}

// Decode 7bit编码无需解码，直接返回原数据
func (d *SevenBitDecoder) Decode(data []byte) ([]byte, error) {
	return data, nil
}

// GetEncodingType 获取编码类型
func (d *SevenBitDecoder) GetEncodingType() string {
	return "7bit"
}

// EightBitDecoder 8bit解码器（无需解码）
type EightBitDecoder struct{}

// NewEightBitDecoder 创建8bit解码器
func NewEightBitDecoder() *EightBitDecoder {
	return &EightBitDecoder{}
}

// Decode 8bit编码无需解码，直接返回原数据
func (d *EightBitDecoder) Decode(data []byte) ([]byte, error) {
	return data, nil
}

// GetEncodingType 获取编码类型
func (d *EightBitDecoder) GetEncodingType() string {
	return "8bit"
}

// BinaryDecoder Binary解码器（无需解码）
type BinaryDecoder struct{}

// NewBinaryDecoder 创建Binary解码器
func NewBinaryDecoder() *BinaryDecoder {
	return &BinaryDecoder{}
}

// Decode Binary编码无需解码，直接返回原数据
func (d *BinaryDecoder) Decode(data []byte) ([]byte, error) {
	return data, nil
}

// GetEncodingType 获取编码类型
func (d *BinaryDecoder) GetEncodingType() string {
	return "binary"
}

// StandardDecoderFactory 标准解码器工厂
type StandardDecoderFactory struct {
	decoders map[string]func() TransferDecoder
}

// NewStandardDecoderFactory 创建标准解码器工厂
func NewStandardDecoderFactory() *StandardDecoderFactory {
	factory := &StandardDecoderFactory{
		decoders: make(map[string]func() TransferDecoder),
	}
	
	// 注册标准解码器
	factory.registerStandardDecoders()
	
	return factory
}

// registerStandardDecoders 注册标准解码器
func (f *StandardDecoderFactory) registerStandardDecoders() {
	f.decoders["base64"] = func() TransferDecoder { return NewBase64Decoder() }
	f.decoders["quoted-printable"] = func() TransferDecoder { return NewQuotedPrintableDecoder() }
	f.decoders["7bit"] = func() TransferDecoder { return NewSevenBitDecoder() }
	f.decoders["8bit"] = func() TransferDecoder { return NewEightBitDecoder() }
	f.decoders["binary"] = func() TransferDecoder { return NewBinaryDecoder() }
}

// CreateDecoder 根据编码类型创建解码器
func (f *StandardDecoderFactory) CreateDecoder(encoding string) (TransferDecoder, error) {
	// 标准化编码名称
	normalizedEncoding := strings.ToLower(strings.TrimSpace(encoding))
	
	// 如果编码为空，默认使用7bit
	if normalizedEncoding == "" {
		normalizedEncoding = "7bit"
	}
	
	// 查找解码器创建函数
	createFunc, exists := f.decoders[normalizedEncoding]
	if !exists {
		return nil, fmt.Errorf("unsupported transfer encoding: %s", encoding)
	}
	
	return createFunc(), nil
}

// GetSupportedEncodings 获取支持的编码类型
func (f *StandardDecoderFactory) GetSupportedEncodings() []string {
	encodings := make([]string, 0, len(f.decoders))
	for encoding := range f.decoders {
		encodings = append(encodings, encoding)
	}
	return encodings
}

// RegisterDecoder 注册自定义解码器
func (f *StandardDecoderFactory) RegisterDecoder(encoding string, createFunc func() TransferDecoder) {
	normalizedEncoding := strings.ToLower(strings.TrimSpace(encoding))
	f.decoders[normalizedEncoding] = createFunc
}
