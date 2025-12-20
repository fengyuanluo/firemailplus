package providers

import "strings"

// isMicrosoftPersonalDomain 判断是否为微软个人邮箱域名
func isMicrosoftPersonalDomain(domain string) bool {
	if domain == "" {
		return false
	}
	// 统一为小写后判断，避免大小写混用导致的匹配遗漏
	d := strings.ToLower(domain)
	return strings.HasPrefix(d, "outlook.") ||
		strings.HasPrefix(d, "hotmail.") ||
		strings.HasPrefix(d, "live.") ||
		strings.HasPrefix(d, "msn.")
}
