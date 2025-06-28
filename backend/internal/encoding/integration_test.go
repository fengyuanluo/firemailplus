package encoding

import (
	"encoding/base64"
	"mime/quotedprintable"
	"strings"
	"testing"
)

func TestEmailEncodingHelper_DecodeTransferEncoding(t *testing.T) {
	helper := NewEmailEncodingHelper()
	
	// 测试Base64解码
	testData := "Hello, World! 你好世界！"
	encoded := base64.StdEncoding.EncodeToString([]byte(testData))
	
	decoded, err := helper.DecodeTransferEncoding([]byte(encoded), "base64")
	if err != nil {
		t.Fatalf("Failed to decode base64: %v", err)
	}
	
	if string(decoded) != testData {
		t.Errorf("Expected %s, got %s", testData, string(decoded))
	}
	
	// 测试Quoted-Printable解码
	var qpEncoded strings.Builder
	writer := quotedprintable.NewWriter(&qpEncoded)
	writer.Write([]byte(testData))
	writer.Close()
	
	decoded, err = helper.DecodeTransferEncoding([]byte(qpEncoded.String()), "quoted-printable")
	if err != nil {
		t.Fatalf("Failed to decode quoted-printable: %v", err)
	}
	
	if string(decoded) != testData {
		t.Errorf("Expected %s, got %s", testData, string(decoded))
	}
	
	// 测试7bit（无需解码）
	decoded, err = helper.DecodeTransferEncoding([]byte(testData), "7bit")
	if err != nil {
		t.Fatalf("Failed to decode 7bit: %v", err)
	}
	
	if string(decoded) != testData {
		t.Errorf("Expected %s, got %s", testData, string(decoded))
	}
	
	// 测试8bit（无需解码）
	decoded, err = helper.DecodeTransferEncoding([]byte(testData), "8bit")
	if err != nil {
		t.Fatalf("Failed to decode 8bit: %v", err)
	}
	
	if string(decoded) != testData {
		t.Errorf("Expected %s, got %s", testData, string(decoded))
	}
	
	// 测试binary（无需解码）
	binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
	decoded, err = helper.DecodeTransferEncoding(binaryData, "binary")
	if err != nil {
		t.Fatalf("Failed to decode binary: %v", err)
	}
	
	if len(decoded) != len(binaryData) {
		t.Errorf("Expected length %d, got %d", len(binaryData), len(decoded))
	}
	
	for i, b := range decoded {
		if b != binaryData[i] {
			t.Errorf("Expected byte %d at position %d, got %d", binaryData[i], i, b)
		}
	}
}

func TestEmailEncodingHelper_DecodeEmailContentWithTransferEncoding(t *testing.T) {
	helper := NewEmailEncodingHelper()
	
	// 测试Base64 + UTF-8
	testData := "Hello, World! 你好世界！"
	encoded := base64.StdEncoding.EncodeToString([]byte(testData))
	
	decoded, err := helper.DecodeEmailContentWithTransferEncoding([]byte(encoded), "base64", "utf-8")
	if err != nil {
		t.Fatalf("Failed to decode base64 + utf-8: %v", err)
	}
	
	if string(decoded) != testData {
		t.Errorf("Expected %s, got %s", testData, string(decoded))
	}
	
	// 测试Quoted-Printable + 自动检测
	var qpEncoded strings.Builder
	writer := quotedprintable.NewWriter(&qpEncoded)
	writer.Write([]byte(testData))
	writer.Close()
	
	decoded, err = helper.DecodeEmailContentWithTransferEncoding([]byte(qpEncoded.String()), "quoted-printable", "")
	if err != nil {
		t.Fatalf("Failed to decode quoted-printable + auto: %v", err)
	}
	
	if string(decoded) != testData {
		t.Errorf("Expected %s, got %s", testData, string(decoded))
	}
	
	// 测试7bit + 无字符编码
	decoded, err = helper.DecodeEmailContentWithTransferEncoding([]byte(testData), "7bit", "")
	if err != nil {
		t.Fatalf("Failed to decode 7bit + auto: %v", err)
	}
	
	if string(decoded) != testData {
		t.Errorf("Expected %s, got %s", testData, string(decoded))
	}
}

func TestEmailEncodingHelper_DecodeEmailContentComplete(t *testing.T) {
	helper := NewEmailEncodingHelper()
	
	// 测试完整的解码流程
	testData := "Hello, World! 你好世界！"
	encoded := base64.StdEncoding.EncodeToString([]byte(testData))
	
	decoded, err := helper.DecodeEmailContentComplete([]byte(encoded), "base64", "utf-8")
	if err != nil {
		t.Fatalf("Failed to decode complete: %v", err)
	}
	
	if string(decoded) != testData {
		t.Errorf("Expected %s, got %s", testData, string(decoded))
	}
}

func TestEmailEncodingHelper_ErrorHandling(t *testing.T) {
	helper := NewEmailEncodingHelper()

	// 测试无效的Base64数据 - 应该回退到原内容
	invalidData := []byte("invalid base64 data!@#")
	decoded, err := helper.DecodeTransferEncoding(invalidData, "base64")
	if err != nil {
		t.Fatalf("Unexpected error for invalid base64 data: %v", err)
	}
	// 由于使用了回退机制，应该返回原内容
	if string(decoded) != string(invalidData) {
		t.Errorf("Expected fallback to original data, got %s", string(decoded))
	}

	// 测试不支持的传输编码 - 应该回退到原内容
	testData := []byte("test data")
	decoded, err = helper.DecodeTransferEncoding(testData, "unsupported-encoding")
	if err != nil {
		t.Fatalf("Unexpected error for unsupported encoding: %v", err)
	}
	// 由于使用了回退机制，应该返回原内容
	if string(decoded) != string(testData) {
		t.Errorf("Expected fallback to original data, got %s", string(decoded))
	}
}

func TestEmailEncodingHelper_EmptyContent(t *testing.T) {
	helper := NewEmailEncodingHelper()
	
	// 测试空内容
	decoded, err := helper.DecodeTransferEncoding([]byte{}, "base64")
	if err != nil {
		t.Fatalf("Failed to decode empty content: %v", err)
	}
	
	if len(decoded) != 0 {
		t.Errorf("Expected empty result, got %d bytes", len(decoded))
	}
	
	// 测试nil内容
	decoded, err = helper.DecodeTransferEncoding(nil, "base64")
	if err != nil {
		t.Fatalf("Failed to decode nil content: %v", err)
	}
	
	if len(decoded) != 0 {
		t.Errorf("Expected empty result, got %d bytes", len(decoded))
	}
}

func TestEmailEncodingHelper_CaseInsensitiveEncoding(t *testing.T) {
	helper := NewEmailEncodingHelper()
	
	testData := "Hello, World!"
	encoded := base64.StdEncoding.EncodeToString([]byte(testData))
	
	// 测试大写编码名称
	decoded, err := helper.DecodeTransferEncoding([]byte(encoded), "BASE64")
	if err != nil {
		t.Fatalf("Failed to decode BASE64: %v", err)
	}
	
	if string(decoded) != testData {
		t.Errorf("Expected %s, got %s", testData, string(decoded))
	}
	
	// 测试混合大小写编码名称
	decoded, err = helper.DecodeTransferEncoding([]byte(encoded), "Base64")
	if err != nil {
		t.Fatalf("Failed to decode Base64: %v", err)
	}
	
	if string(decoded) != testData {
		t.Errorf("Expected %s, got %s", testData, string(decoded))
	}
}

func TestEmailEncodingHelper_BackwardCompatibility(t *testing.T) {
	helper := NewEmailEncodingHelper()
	
	// 测试现有的DecodeEmailContent方法仍然工作
	testData := "Hello, World! 你好世界！"
	
	decoded, err := helper.DecodeEmailContent([]byte(testData), "utf-8")
	if err != nil {
		t.Fatalf("Failed to decode with existing method: %v", err)
	}
	
	if string(decoded) != testData {
		t.Errorf("Expected %s, got %s", testData, string(decoded))
	}
	
	// 测试自动检测
	decoded, err = helper.DecodeEmailContent([]byte(testData), "")
	if err != nil {
		t.Fatalf("Failed to decode with auto detection: %v", err)
	}
	
	if string(decoded) != testData {
		t.Errorf("Expected %s, got %s", testData, string(decoded))
	}
}
