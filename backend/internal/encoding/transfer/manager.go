package transfer

import (
	"fmt"
	"log"
	"strings"
	"sync"
)

// DecoderManager 解码器管理器
type DecoderManager struct {
	factory DecoderFactory
	cache   map[string]TransferDecoder
	mutex   sync.RWMutex
}

// NewDecoderManager 创建解码器管理器
func NewDecoderManager(factory DecoderFactory) *DecoderManager {
	if factory == nil {
		factory = NewStandardDecoderFactory()
	}
	
	return &DecoderManager{
		factory: factory,
		cache:   make(map[string]TransferDecoder),
	}
}

// NewStandardDecoderManager 创建标准解码器管理器
func NewStandardDecoderManager() *DecoderManager {
	return NewDecoderManager(NewStandardDecoderFactory())
}

// Decode 解码数据
func (m *DecoderManager) Decode(data []byte, encoding string) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	
	decoder, err := m.getDecoder(encoding)
	if err != nil {
		return nil, fmt.Errorf("failed to get decoder for encoding %s: %w", encoding, err)
	}
	
	decoded, err := decoder.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode data with encoding %s: %w", encoding, err)
	}
	
	return decoded, nil
}

// DecodeString 解码字符串
func (m *DecoderManager) DecodeString(data string, encoding string) (string, error) {
	decoded, err := m.Decode([]byte(data), encoding)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// getDecoder 获取解码器（带缓存）
func (m *DecoderManager) getDecoder(encoding string) (TransferDecoder, error) {
	// 标准化编码名称
	normalizedEncoding := normalizeEncoding(encoding)
	
	// 先尝试从缓存获取
	m.mutex.RLock()
	if decoder, exists := m.cache[normalizedEncoding]; exists {
		m.mutex.RUnlock()
		return decoder, nil
	}
	m.mutex.RUnlock()
	
	// 缓存中没有，创建新的解码器
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// 双重检查，防止并发创建
	if decoder, exists := m.cache[normalizedEncoding]; exists {
		return decoder, nil
	}
	
	// 创建解码器
	decoder, err := m.factory.CreateDecoder(normalizedEncoding)
	if err != nil {
		return nil, err
	}
	
	// 缓存解码器
	m.cache[normalizedEncoding] = decoder
	
	return decoder, nil
}

// GetSupportedEncodings 获取支持的编码类型
func (m *DecoderManager) GetSupportedEncodings() []string {
	return m.factory.GetSupportedEncodings()
}

// IsEncodingSupported 检查是否支持指定的编码
func (m *DecoderManager) IsEncodingSupported(encoding string) bool {
	normalizedEncoding := normalizeEncoding(encoding)
	supportedEncodings := m.GetSupportedEncodings()
	
	for _, supported := range supportedEncodings {
		if supported == normalizedEncoding {
			return true
		}
	}
	
	return false
}

// ClearCache 清空解码器缓存
func (m *DecoderManager) ClearCache() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.cache = make(map[string]TransferDecoder)
}

// GetCacheSize 获取缓存大小
func (m *DecoderManager) GetCacheSize() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return len(m.cache)
}

// DecodeWithFallback 带回退机制的解码
func (m *DecoderManager) DecodeWithFallback(data []byte, encoding string) ([]byte, error) {
	// 首先尝试指定的编码
	if encoding != "" {
		decoded, err := m.Decode(data, encoding)
		if err == nil {
			return decoded, nil
		}
		
		log.Printf("Warning: Failed to decode with encoding %s: %v, falling back to 7bit", encoding, err)
	}
	
	// 回退到7bit（无需解码）
	return m.Decode(data, "7bit")
}

// DecodeStringWithFallback 带回退机制的字符串解码
func (m *DecoderManager) DecodeStringWithFallback(data string, encoding string) (string, error) {
	decoded, err := m.DecodeWithFallback([]byte(data), encoding)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// normalizeEncoding 标准化编码名称
func normalizeEncoding(encoding string) string {
	if encoding == "" {
		return "7bit"
	}
	
	// 转换为小写并去除空格
	normalized := strings.ToLower(strings.TrimSpace(encoding))
	
	// 处理一些常见的别名
	switch normalized {
	case "base-64", "base_64":
		return "base64"
	case "quoted_printable", "quoted-printable":
		return "quoted-printable"
	case "7-bit", "7_bit":
		return "7bit"
	case "8-bit", "8_bit":
		return "8bit"
	default:
		return normalized
	}
}

// 全局默认解码器管理器实例
var defaultManager *DecoderManager
var defaultManagerOnce sync.Once

// GetDefaultManager 获取默认解码器管理器
func GetDefaultManager() *DecoderManager {
	defaultManagerOnce.Do(func() {
		defaultManager = NewStandardDecoderManager()
	})
	return defaultManager
}

// 便利函数，使用默认管理器

// Decode 使用默认管理器解码数据
func Decode(data []byte, encoding string) ([]byte, error) {
	return GetDefaultManager().Decode(data, encoding)
}

// DecodeString 使用默认管理器解码字符串
func DecodeString(data string, encoding string) (string, error) {
	return GetDefaultManager().DecodeString(data, encoding)
}

// DecodeWithFallback 使用默认管理器带回退机制解码
func DecodeWithFallback(data []byte, encoding string) ([]byte, error) {
	return GetDefaultManager().DecodeWithFallback(data, encoding)
}

// DecodeStringWithFallback 使用默认管理器带回退机制解码字符串
func DecodeStringWithFallback(data string, encoding string) (string, error) {
	return GetDefaultManager().DecodeStringWithFallback(data, encoding)
}
