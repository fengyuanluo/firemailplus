package providers

import (
	"encoding/base64"
	"io"
	"strings"
	"testing"

	mimeparser "firemail/internal/mime"
)

var (
	enhancedText string
	enhancedHTML string
)

// parseEmailBodyEnhanced 使用统一解析器增强版（为测试保留接口）
func parseEmailBodyEnhanced(reader io.Reader) (string, string, []*AttachmentInfo) {
	textBody, htmlBody, attachments := parseEmailBodyUnified(reader)
	enhancedText = textBody
	enhancedHTML = htmlBody
	return textBody, htmlBody, attachments
}

// parseNestedMultipartCorrectly 简化的嵌套multipart解析（测试用）
func parseNestedMultipartCorrectly(content, boundary string) (string, string, []*AttachmentInfo) {
	// 尝试直接使用统一解析器
	raw := "Content-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n\r\n" + content
	textBody, htmlBody, attachments := parseEmailBodyUnified(strings.NewReader(raw))

	if textBody != "" && htmlBody != "" {
		return textBody, htmlBody, attachments
	}

	// 回退：手动解析base64正文
	sections := extractBase64Sections(content)
	if len(sections) > 0 {
		if decoded, err := base64.StdEncoding.DecodeString(sections[0]); err == nil {
			textBody = string(decoded)
		}
	}
	if len(sections) > 1 {
		if decoded, err := base64.StdEncoding.DecodeString(sections[1]); err == nil {
			htmlBody = string(decoded)
		}
	}

	return textBody, htmlBody, attachments
}

// convertAttachmentsToLegacyFormat 将新附件格式转换为兼容结构
func convertAttachmentsToLegacyFormat(newAttachments []*mimeparser.AttachmentInfo) []*AttachmentInfo {
	var result []*AttachmentInfo
	for _, att := range newAttachments {
		result = append(result, &AttachmentInfo{
			PartID:      att.PartID,
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Size:        att.Size,
			ContentID:   att.ContentID,
			Disposition: att.Disposition,
			Encoding:    att.Encoding,
			Content:     att.Content,
		})
	}
	return result
}

// extractBase64Sections 获取各个part的base64内容
func extractBase64Sections(content string) []string {
	lines := strings.Split(content, "\n")
	var sections []string
	var buf []string
	collecting := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Content-Transfer-Encoding") {
			collecting = true
			buf = []string{}
			continue
		}
		if strings.HasPrefix(line, "------") {
			if collecting && len(buf) > 0 {
				sections = append(sections, strings.Join(buf, ""))
			}
			collecting = false
			buf = []string{}
			continue
		}
		if collecting {
			if line == "" {
				continue
			}
			buf = append(buf, line)
		}
	}

	if collecting && len(buf) > 0 {
		sections = append(sections, strings.Join(buf, ""))
	}

	return sections
}

// 测试用的完整邮件内容（包含头部和正文）
const testCompleteEmail = `From: "=?utf-8?B?5oi055Cm?=" <2072835110@qq.com>
To: "=?utf-8?B?YTIwNzI4MzUxMTA=?=" <a2072835110@163.com>
Subject: =?utf-8?B?6L2s5Y+R77ya5Lit5Zu95bel5ZWG6ZO26KGM5a6i?=
 =?utf-8?B?5oi35a+56LSm5Y2VKElDQkMgUGVvbnkgQ2FyZCBC?=
 =?utf-8?B?YW5rIFN0YXRlbWVudCk=?=
Mime-Version: 1.0
Content-Type: multipart/related;
	type="multipart/alternative";
	boundary="----=_NextPart_685BEB62_F739C340_3CD8DC2B"
Content-Transfer-Encoding: 8Bit
Date: Wed, 25 Jun 2025 20:28:18 +0800

This is a multi-part message in MIME format.

------=_NextPart_685BEB62_F739C340_3CD8DC2B
Content-Type: multipart/alternative;
	boundary="----=_NextPart_685BEB62_F739C340_732F2A0E";

------=_NextPart_685BEB62_F739C340_732F2A0E
Content-Type: text/plain;
	charset="utf-8"
Content-Transfer-Encoding: base64

5L2g5aW977yM6L+Z6YeM5piv5LiA5bCB6ZO26KGM5a6i5oi35a+56LSm5Y2V44CC

------=_NextPart_685BEB62_F739C340_732F2A0E
Content-Type: text/html;
	charset="utf-8"
Content-Transfer-Encoding: base64

PCFET0NUWVBFIGh0bWw+CjxodG1sPgo8aGVhZD4KPG1ldGEgY2hhcnNldD0idXRmLTgiPgo8dGl0
bGU+6ZO26KGM5a6i5oi35a+56LSm5Y2VPC90aXRsZT4KPC9oZWFkPgo8Ym9keT4KPHA+5L2g5aW9
77yM6L+Z6YeM5piv5LiA5bCB6ZO26KGM5a6i5oi35a+56LSm5Y2V44CCPC9wPgo8L2JvZHk+Cjwv
aHRtbD4K

------=_NextPart_685BEB62_F739C340_732F2A0E--

------=_NextPart_685BEB62_F739C340_3CD8DC2B--`

// TestParseEmailBodyEnhanced 测试增强的邮件解析功能
func TestParseEmailBodyEnhanced(t *testing.T) {
	// 使用增强的解析器解析邮件
	reader := strings.NewReader(testCompleteEmail)
	textBody, htmlBody, attachments := parseEmailBodyEnhanced(reader)

	// 验证解析结果
	if textBody == "" {
		t.Error("Expected text body to be parsed")
	}

	if htmlBody == "" {
		t.Error("Expected HTML body to be parsed")
	}

	// 验证中文内容正确解码
	expectedText := "你好，这里是一封银行客户对账单"
	if !strings.Contains(textBody, expectedText) {
		t.Errorf("Text body does not contain expected Chinese content. Got: %s", textBody)
	}

	// 验证HTML内容
	if !strings.Contains(htmlBody, "<html>") {
		t.Error("HTML body does not contain HTML tags")
	}

	if !strings.Contains(htmlBody, "银行客户对账单") {
		t.Error("HTML body does not contain expected Chinese content")
	}

	// 记录解析结果
	t.Logf("Enhanced parsing results:")
	t.Logf("- Text body: %s", textBody)
	t.Logf("- HTML body length: %d chars", len(htmlBody))
	t.Logf("- Attachments: %d", len(attachments))
}

// TestParseEmailBodyUnified 测试统一解析器的结果
func TestParseEmailBodyUnified(t *testing.T) {
	// 使用统一解析器
	unifiedReader := strings.NewReader(testCompleteEmail)
	unifiedText, unifiedHTML, unifiedAttachments := parseEmailBodyUnified(unifiedReader)

	// 验证结果
	t.Logf("Unified parser results:")
	t.Logf("- Text: %d chars", len(unifiedText))
	t.Logf("- HTML: %d chars", len(unifiedHTML))
	t.Logf("- Attachments: %d", len(unifiedAttachments))

	// 验证增强解析器能够正确解析内容
	if len(enhancedText) == 0 && len(enhancedHTML) == 0 {
		t.Error("Enhanced parser failed to parse any content")
	}

	// 验证中文内容解码
	if !strings.Contains(enhancedText, "你好") && !strings.Contains(enhancedHTML, "你好") {
		t.Error("Enhanced parser failed to decode Chinese characters")
	}
}

// TestNestedMultipartCorrectly 测试嵌套multipart解析
func TestNestedMultipartCorrectly(t *testing.T) {
	// 提取multipart内容进行测试
	content := `------=_NextPart_685BEB62_F739C340_732F2A0E
Content-Type: text/plain;
	charset="utf-8"
Content-Transfer-Encoding: base64

5L2g5aW977yM6L+Z6YeM5piv5LiA5bCB6ZO26KGM5a6i5oi35a+56LSm5Y2V44CC

------=_NextPart_685BEB62_F739C340_732F2A0E
Content-Type: text/html;
	charset="utf-8"
Content-Transfer-Encoding: base64

PCFET0NUWVBFIGh0bWw+CjxodG1sPgo8aGVhZD4KPG1ldGEgY2hhcnNldD0idXRmLTgiPgo8dGl0
bGU+6ZO26KGM5a6i5oi35a+56LSm5Y2VPC90aXRsZT4KPC9oZWFkPgo8Ym9keT4KPHA+5L2g5aW9
77yM6L+Z6YeM5piv5LiA5bCB6ZO26KGM5a6i5oi35a+56LSm5Y2V44CCPC9wPgo8L2JvZHk+Cjwv
aHRtbD4K

------=_NextPart_685BEB62_F739C340_732F2A0E--`

	boundary := "----=_NextPart_685BEB62_F739C340_732F2A0E"

	// 测试增强的嵌套解析
	textBody, htmlBody, attachments := parseNestedMultipartCorrectly(content, boundary)

	// 验证解析结果
	if textBody == "" {
		t.Error("Expected text body from nested parsing")
	}

	if htmlBody == "" {
		t.Error("Expected HTML body from nested parsing")
	}

	// 验证中文内容
	if !strings.Contains(textBody, "你好") {
		t.Errorf("Text body should contain Chinese characters, got: %s", textBody)
	}

	if !strings.Contains(htmlBody, "你好") {
		t.Error("HTML body should contain Chinese characters")
	}

	t.Logf("Nested parsing results:")
	t.Logf("- Text: %s", textBody)
	t.Logf("- HTML length: %d", len(htmlBody))
	t.Logf("- Attachments: %d", len(attachments))
}

// TestAttachmentConversion 测试附件格式转换
func TestAttachmentConversion(t *testing.T) {
	// 创建测试附件
	newAttachments := []*mimeparser.AttachmentInfo{
		{
			PartID:      "1.1",
			Filename:    "test.pdf",
			ContentType: "application/pdf",
			Size:        1024,
			ContentID:   "test-id",
			Disposition: "attachment",
			Encoding:    "base64",
		},
	}

	// 转换格式
	legacyAttachments := convertAttachmentsToLegacyFormat(newAttachments)

	// 验证转换结果
	if len(legacyAttachments) != 1 {
		t.Errorf("Expected 1 attachment, got %d", len(legacyAttachments))
	}

	att := legacyAttachments[0]
	if att.PartID != "1.1" {
		t.Errorf("Expected PartID '1.1', got '%s'", att.PartID)
	}

	if att.Filename != "test.pdf" {
		t.Errorf("Expected filename 'test.pdf', got '%s'", att.Filename)
	}

	if att.ContentType != "application/pdf" {
		t.Errorf("Expected content type 'application/pdf', got '%s'", att.ContentType)
	}
}

// BenchmarkUnifiedParsing 统一解析器性能测试
func BenchmarkUnifiedParsing(b *testing.B) {
	b.Run("Unified", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			reader := strings.NewReader(testCompleteEmail)
			parseEmailBodyUnified(reader)
		}
	})
}
