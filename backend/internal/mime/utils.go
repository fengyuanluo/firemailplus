package mime

import (
	"fmt"
	"log"
	"strings"
)

// MIMEUtils MIME工具集合
type MIMEUtils struct{}

// NewMIMEUtils 创建MIME工具
func NewMIMEUtils() *MIMEUtils {
	return &MIMEUtils{}
}

// ExtractTextAndHTML 从解析结果中提取文本和HTML内容
func (u *MIMEUtils) ExtractTextAndHTML(parsed *ParsedEmail) (textBody, htmlBody string) {
	if parsed == nil {
		return "", ""
	}
	return parsed.TextBody, parsed.HTMLBody
}

// ExtractAttachments 从解析结果中提取附件信息
func (u *MIMEUtils) ExtractAttachments(parsed *ParsedEmail) []*AttachmentInfo {
	if parsed == nil {
		return nil
	}
	return parsed.Attachments
}

// ExtractInlineAttachments 从解析结果中提取内联附件信息
func (u *MIMEUtils) ExtractInlineAttachments(parsed *ParsedEmail) []*AttachmentInfo {
	if parsed == nil {
		return nil
	}
	return parsed.InlineAttachments
}

// CountParts 统计MIME部分数量
func (u *MIMEUtils) CountParts(part *MIMEPart) int {
	if part == nil {
		return 0
	}

	count := 1 // 当前部分
	for _, subPart := range part.Parts {
		count += u.CountParts(subPart)
	}

	return count
}

// FindPartByContentID 根据Content-ID查找部分
func (u *MIMEUtils) FindPartByContentID(part *MIMEPart, contentID string) *MIMEPart {
	if part == nil || contentID == "" {
		return nil
	}

	// 检查当前部分
	if partContentID := part.Headers.Get("Content-ID"); partContentID != "" {
		// 移除尖括号进行比较
		cleanContentID := strings.Trim(partContentID, "<>")
		if cleanContentID == contentID {
			return part
		}
	}

	// 递归查找子部分
	for _, subPart := range part.Parts {
		if found := u.FindPartByContentID(subPart, contentID); found != nil {
			return found
		}
	}

	return nil
}

// FindPartsByType 根据MIME类型查找部分
func (u *MIMEUtils) FindPartsByType(part *MIMEPart, mimeType string) []*MIMEPart {
	var result []*MIMEPart

	if part == nil || mimeType == "" {
		return result
	}

	// 检查当前部分
	if part.Type != nil && part.Type.FullType == strings.ToLower(mimeType) {
		result = append(result, part)
	}

	// 递归查找子部分
	for _, subPart := range part.Parts {
		subResult := u.FindPartsByType(subPart, mimeType)
		result = append(result, subResult...)
	}

	return result
}

// GetMIMEStructure 获取MIME结构描述
func (u *MIMEUtils) GetMIMEStructure(part *MIMEPart) string {
	if part == nil {
		return ""
	}

	var builder strings.Builder
	u.buildMIMEStructure(part, &builder, 0)
	return builder.String()
}

// buildMIMEStructure 构建MIME结构描述
func (u *MIMEUtils) buildMIMEStructure(part *MIMEPart, builder *strings.Builder, depth int) {
	if part == nil {
		return
	}

	// 添加缩进
	indent := strings.Repeat("  ", depth)
	
	// 添加当前部分信息
	mimeType := "unknown"
	if part.Type != nil {
		mimeType = part.Type.FullType
	}
	
	builder.WriteString(fmt.Sprintf("%s- %s", indent, mimeType))
	
	// 添加额外信息
	if part.Disposition != nil {
		builder.WriteString(fmt.Sprintf(" [%s]", part.Disposition.Type))
		if filename := part.Disposition.GetFilename(); filename != "" {
			builder.WriteString(fmt.Sprintf(" (%s)", filename))
		}
	}
	
	if part.Type != nil && part.Type.IsMultipart() {
		if boundary := part.Type.GetBoundary(); boundary != "" {
			builder.WriteString(fmt.Sprintf(" boundary=%s", boundary))
		}
	}
	
	builder.WriteString("\n")

	// 递归处理子部分
	for _, subPart := range part.Parts {
		u.buildMIMEStructure(subPart, builder, depth+1)
	}
}

// ValidateEmailStructure 验证邮件结构
func (u *MIMEUtils) ValidateEmailStructure(parsed *ParsedEmail) []string {
	var warnings []string

	if parsed == nil {
		warnings = append(warnings, "parsed email is nil")
		return warnings
	}

	// 检查是否有内容
	if parsed.TextBody == "" && parsed.HTMLBody == "" && len(parsed.Attachments) == 0 {
		warnings = append(warnings, "email has no content")
	}

	// 检查根部分
	if parsed.RootPart == nil {
		warnings = append(warnings, "root part is missing")
	} else {
		partWarnings := u.validateMIMEPart(parsed.RootPart, "root")
		warnings = append(warnings, partWarnings...)
	}

	return warnings
}

// validateMIMEPart 验证MIME部分
func (u *MIMEUtils) validateMIMEPart(part *MIMEPart, partPath string) []string {
	var warnings []string

	if part == nil {
		warnings = append(warnings, fmt.Sprintf("%s: part is nil", partPath))
		return warnings
	}

	// 检查MIME类型
	if part.Type == nil {
		warnings = append(warnings, fmt.Sprintf("%s: MIME type is missing", partPath))
	} else {
		if err := GetDefaultDetector().ValidateMIMEType(part.Type); err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: invalid MIME type: %v", partPath, err))
		}
	}

	// 检查multipart的boundary
	if part.Type != nil && part.Type.IsMultipart() {
		if part.Type.GetBoundary() == "" {
			warnings = append(warnings, fmt.Sprintf("%s: multipart missing boundary", partPath))
		}
	}

	// 递归检查子部分
	for i, subPart := range part.Parts {
		subPath := fmt.Sprintf("%s.%d", partPath, i+1)
		subWarnings := u.validateMIMEPart(subPart, subPath)
		warnings = append(warnings, subWarnings...)
	}

	return warnings
}

// ConvertToLegacyFormat 转换为旧格式（兼容性）
func (u *MIMEUtils) ConvertToLegacyFormat(parsed *ParsedEmail) (textBody, htmlBody string, attachments []*LegacyAttachmentInfo) {
	if parsed == nil {
		return "", "", nil
	}

	textBody = parsed.TextBody
	htmlBody = parsed.HTMLBody

	// 转换附件格式
	for _, att := range parsed.Attachments {
		legacyAtt := &LegacyAttachmentInfo{
			PartID:      att.PartID,
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Size:        att.Size,
			ContentID:   att.ContentID,
			Disposition: att.Disposition,
			Encoding:    att.Encoding,
		}
		attachments = append(attachments, legacyAtt)
	}

	return textBody, htmlBody, attachments
}

// LegacyAttachmentInfo 旧版附件信息格式
type LegacyAttachmentInfo struct {
	PartID      string
	Filename    string
	ContentType string
	Size        int64
	ContentID   string
	Disposition string
	Encoding    string
}

// LogMIMEStructure 记录MIME结构到日志
func (u *MIMEUtils) LogMIMEStructure(parsed *ParsedEmail, prefix string) {
	if parsed == nil {
		log.Printf("%s: parsed email is nil", prefix)
		return
	}

	if parsed.RootPart == nil {
		log.Printf("%s: root part is missing", prefix)
		return
	}

	structure := u.GetMIMEStructure(parsed.RootPart)
	log.Printf("%s: MIME structure:\n%s", prefix, structure)

	// 记录内容统计
	log.Printf("%s: content summary - text: %d chars, html: %d chars, attachments: %d, inline: %d",
		prefix,
		len(parsed.TextBody),
		len(parsed.HTMLBody),
		len(parsed.Attachments),
		len(parsed.InlineAttachments),
	)
}

// GetContentSummary 获取内容摘要
func (u *MIMEUtils) GetContentSummary(parsed *ParsedEmail) map[string]interface{} {
	summary := make(map[string]interface{})

	if parsed == nil {
		summary["error"] = "parsed email is nil"
		return summary
	}

	summary["text_length"] = len(parsed.TextBody)
	summary["html_length"] = len(parsed.HTMLBody)
	summary["attachment_count"] = len(parsed.Attachments)
	summary["inline_attachment_count"] = len(parsed.InlineAttachments)

	if parsed.RootPart != nil {
		summary["part_count"] = u.CountParts(parsed.RootPart)
		if parsed.RootPart.Type != nil {
			summary["root_type"] = parsed.RootPart.Type.FullType
		}
	}

	return summary
}

// 全局默认工具实例
var defaultUtils *MIMEUtils

// GetDefaultUtils 获取默认工具
func GetDefaultUtils() *MIMEUtils {
	if defaultUtils == nil {
		defaultUtils = NewMIMEUtils()
	}
	return defaultUtils
}

// 便利函数

// GetMIMEStructureString 获取MIME结构字符串
func GetMIMEStructureString(parsed *ParsedEmail) string {
	return GetDefaultUtils().GetMIMEStructure(parsed.RootPart)
}

// ValidateEmail 验证邮件结构
func ValidateEmail(parsed *ParsedEmail) []string {
	return GetDefaultUtils().ValidateEmailStructure(parsed)
}

// LogStructure 记录结构到日志
func LogStructure(parsed *ParsedEmail, prefix string) {
	GetDefaultUtils().LogMIMEStructure(parsed, prefix)
}
