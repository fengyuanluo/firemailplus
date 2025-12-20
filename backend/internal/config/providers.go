package config

import "strings"

// EmailProviderConfig 邮件提供商配置
type EmailProviderConfig struct {
	Name         string                 `json:"name"`
	DisplayName  string                 `json:"display_name"`
	IMAPHost     string                 `json:"imap_host"`
	IMAPPort     int                    `json:"imap_port"`
	IMAPSecurity string                 `json:"imap_security"` // "SSL", "TLS", "STARTTLS", "NONE"
	SMTPHost     string                 `json:"smtp_host"`
	SMTPPort     int                    `json:"smtp_port"`
	SMTPSecurity string                 `json:"smtp_security"` // "SSL", "TLS", "STARTTLS", "NONE"
	AuthMethods  []string               `json:"auth_methods"`  // "password", "oauth2"
	OAuth2Config *OAuth2Config          `json:"oauth2_config,omitempty"`
	Domains      []string               `json:"domains"`               // 支持的域名
	Features     map[string]bool        `json:"features,omitempty"`    // 功能特性
	Limits       map[string]interface{} `json:"limits,omitempty"`      // 限制信息
	ErrorCodes   map[string]string      `json:"error_codes,omitempty"` // 错误代码说明
	HelpURLs     map[string]string      `json:"help_urls,omitempty"`   // 帮助链接
	Metadata     map[string]string      `json:"metadata,omitempty"`
}

// OAuth2Config OAuth2配置
type OAuth2Config struct {
	AuthURL      string   `json:"auth_url"`
	TokenURL     string   `json:"token_url"`
	Scopes       []string `json:"scopes"`
	ResponseType string   `json:"response_type"`
}

// GetBuiltinProviders 获取内置邮件提供商配置
func GetBuiltinProviders() map[string]EmailProviderConfig {
	return map[string]EmailProviderConfig{
		"gmail": {
			Name:         "gmail",
			DisplayName:  "Gmail",
			IMAPHost:     "imap.gmail.com",
			IMAPPort:     993,
			IMAPSecurity: "SSL",
			SMTPHost:     "smtp.gmail.com",
			SMTPPort:     465,
			SMTPSecurity: "SSL",
			AuthMethods:  []string{"oauth2", "password"},
			OAuth2Config: &OAuth2Config{
				AuthURL:      "https://accounts.google.com/o/oauth2/auth",
				TokenURL:     "https://oauth2.googleapis.com/token",
				Scopes:       []string{"https://mail.google.com/"},
				ResponseType: "code",
			},
			Domains: []string{"gmail.com", "googlemail.com"},
			Features: map[string]bool{
				"imap":       true,
				"smtp":       true,
				"oauth2":     true,
				"basic_auth": true,
				"push":       true,
				"threading":  true,
				"labels":     true,
				"folders":    false,
				"search":     true,
				"idle":       true,
				"extensions": true,
			},
			Limits: map[string]interface{}{
				"attachment_size":  25 * 1024 * 1024,
				"daily_send":       500,
				"max_recipients":   500,
				"storage_free":     15 * 1024 * 1024 * 1024,
				"connection_limit": 15,
			},
			ErrorCodes: map[string]string{
				"535": "认证失败，请检查邮箱地址和应用专用密码",
				"534": "需要启用两步验证并使用应用专用密码",
				"550": "发送频率超限，请稍后重试",
				"552": "邮件大小超过25MB限制",
			},
			HelpURLs: map[string]string{
				"google_account": "https://myaccount.google.com/",
				"app_passwords":  "https://support.google.com/accounts/answer/185833",
				"two_factor":     "https://support.google.com/accounts/answer/185839",
				"oauth2_setup":   "https://developers.google.com/gmail/imap/oauth2",
			},
			Metadata: map[string]string{
				"app_password_url": "https://myaccount.google.com/apppasswords",
				"help_url":         "https://support.google.com/mail/answer/7126229",
				"requires_2fa":     "true",
			},
		},
		"outlook": {
			Name:         "outlook",
			DisplayName:  "Outlook/Hotmail",
			IMAPHost:     "outlook.office365.com", // 严格按照Python代码使用此服务器
			IMAPPort:     993,
			IMAPSecurity: "SSL",
			SMTPHost:     "smtp.office365.com", // 使用正确的SMTP服务器
			SMTPPort:     587,
			SMTPSecurity: "STARTTLS",
			AuthMethods:  []string{"oauth2"}, // 只支持OAuth2手动配置
			OAuth2Config: &OAuth2Config{
				AuthURL:      "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
				TokenURL:     "https://login.microsoftonline.com/common/oauth2/v2.0/token",
				Scopes:       []string{"https://outlook.office.com/IMAP.AccessAsUser.All", "https://outlook.office.com/SMTP.Send", "offline_access"},
				ResponseType: "code",
			},
			// 支持全球地域性后缀：outlook.*、hotmail.*、live.*、msn.*
			Domains: []string{"outlook.*", "hotmail.*", "live.*", "msn.*"},
			Features: map[string]bool{
				"imap":       true,
				"smtp":       true,
				"oauth2":     true,
				"basic_auth": false,
				"push":       true,
				"threading":  true,
				"folders":    true,
				"categories": true,
				"search":     true,
				"idle":       true,
				"rules":      true,
			},
			Limits: map[string]interface{}{
				"attachment_size":  25 * 1024 * 1024,
				"daily_send":       300,
				"max_recipients":   500,
				"storage_free":     15 * 1024 * 1024 * 1024,
				"connection_limit": 16,
			},
			ErrorCodes: map[string]string{
				"535":    "认证失败，个人账户请使用OAuth2",
				"534":    "认证机制不支持，请使用OAuth2",
				"550":    "发送频率超限，请稍后重试",
				"AADSTS": "Azure AD认证错误",
			},
			HelpURLs: map[string]string{
				"azure_portal":     "https://portal.azure.com/",
				"app_registration": "https://docs.microsoft.com/en-us/azure/active-directory/develop/quickstart-register-app",
				"oauth2_setup":     "https://docs.microsoft.com/en-us/exchange/client-developer/legacy-protocols/how-to-authenticate-an-imap-pop-smtp-application-by-using-oauth",
			},
			Metadata: map[string]string{
				"help_url":              "https://support.microsoft.com/en-us/office/pop-imap-and-smtp-settings-for-outlook-com-d088b986-291d-42b8-9564-9c414e2aa040",
				"basic_auth_deprecated": "true",
				"personal_oauth2_only":  "true",
			},
		},
		"qq": {
			Name:         "qq",
			DisplayName:  "QQ邮箱",
			IMAPHost:     "imap.qq.com",
			IMAPPort:     993,
			IMAPSecurity: "SSL",
			SMTPHost:     "smtp.qq.com",
			SMTPPort:     465,
			SMTPSecurity: "SSL",
			AuthMethods:  []string{"password"},
			Domains:      []string{"qq.com", "vip.qq.com", "foxmail.com"},
			Features: map[string]bool{
				"imap":       true,
				"smtp":       true,
				"oauth2":     false,
				"basic_auth": true,
				"search":     true,
				"idle":       true,
				"folders":    true,
			},
			Limits: map[string]interface{}{
				"attachment_size":  50 * 1024 * 1024,
				"daily_send":       500,
				"hourly_send":      50,
				"max_recipients":   100,
				"mailbox_size":     2 * 1024 * 1024 * 1024,
				"connection_limit": 10,
			},
			ErrorCodes: map[string]string{
				"535": "认证失败，请检查邮箱地址和授权码",
				"550": "发送频率超限，请稍后重试",
				"554": "邮件被拒绝，内容可能被识别为垃圾邮件",
				"421": "服务暂时不可用，服务器繁忙",
			},
			HelpURLs: map[string]string{
				"auth_code":  "https://service.mail.qq.com/cgi-bin/help?subtype=1&&id=28&&no=1001256",
				"imap_setup": "https://service.mail.qq.com/detail/0/339",
				"smtp_setup": "https://service.mail.qq.com/detail/0/340",
			},
			Metadata: map[string]string{
				"app_password_url":   "https://service.mail.qq.com/cgi-bin/help?subtype=1&&id=28&&no=1001256",
				"help_url":           "https://service.mail.qq.com/cgi-bin/help?subtype=1&&no=1000585",
				"requires_auth_code": "true",
			},
		},
		"163": {
			Name:         "163",
			DisplayName:  "网易163邮箱",
			IMAPHost:     "imap.163.com",
			IMAPPort:     993,
			IMAPSecurity: "SSL",
			SMTPHost:     "smtp.163.com",
			SMTPPort:     465,
			SMTPSecurity: "SSL",
			AuthMethods:  []string{"password"},
			Domains:      []string{"163.com", "126.com", "yeah.net"},
			Features: map[string]bool{
				"imap":       true,
				"smtp":       true,
				"oauth2":     false,
				"basic_auth": true,
				"search":     true,
				"idle":       true,
				"folders":    true,
			},
			Limits: map[string]interface{}{
				"attachment_size":  50 * 1024 * 1024,
				"daily_send":       200,
				"max_recipients":   100,
				"mailbox_size":     3 * 1024 * 1024 * 1024,
				"connection_limit": 10,
			},
			Metadata: map[string]string{
				"app_password_url": "https://mail.163.com/",
				"help_url":         "https://help.mail.163.com/faqDetail.do?code=d7a5dc8471cd0c0e8b4b8f4f8e49998b374173cfe9171312",
			},
		},
		"icloud": {
			Name:         "icloud",
			DisplayName:  "iCloud邮箱",
			IMAPHost:     "imap.mail.me.com",
			IMAPPort:     993,
			IMAPSecurity: "SSL",
			SMTPHost:     "smtp.mail.me.com",
			SMTPPort:     587,
			SMTPSecurity: "STARTTLS",
			AuthMethods:  []string{"password"},
			Domains:      []string{"icloud.com", "me.com", "mac.com"},
			Features: map[string]bool{
				"imap":       true,
				"smtp":       true,
				"oauth2":     false,
				"basic_auth": true,
				"push":       true,
				"threading":  true,
				"search":     true,
				"idle":       true,
				"folders":    true,
				"notes":      true,
			},
			Limits: map[string]interface{}{
				"attachment_size":  20 * 1024 * 1024,
				"daily_send":       1000,
				"max_recipients":   500,
				"mailbox_size":     5 * 1024 * 1024 * 1024,
				"connection_limit": 5,
			},
			HelpURLs: map[string]string{
				"apple_id":      "https://appleid.apple.com/",
				"app_passwords": "https://support.apple.com/zh-cn/102654",
				"mail_setup":    "https://support.apple.com/zh-cn/102525",
			},
			Metadata: map[string]string{
				"requires_2fa":          "true",
				"app_password_required": "true",
				"help_url":              "https://support.apple.com/zh-cn/icloud",
			},
		},
		"sina": {
			Name:         "sina",
			DisplayName:  "新浪邮箱",
			IMAPHost:     "imap.sina.com",
			IMAPPort:     993,
			IMAPSecurity: "SSL",
			SMTPHost:     "smtp.sina.com",
			SMTPPort:     587,
			SMTPSecurity: "STARTTLS",
			AuthMethods:  []string{"password"},
			Domains:      []string{"sina.com", "sina.cn"},
			Metadata: map[string]string{
				"help_url": "https://help.sina.com.cn/",
			},
		},
		"custom": {
			Name:        "custom",
			DisplayName: "自定义IMAP/SMTP",
			AuthMethods: []string{"password"},
			Domains:     []string{}, // 支持任意域名
			Metadata: map[string]string{
				"description": "自定义IMAP/SMTP服务器配置",
			},
		},
	}
}

// GetProviderByDomain 根据邮箱域名获取提供商配置
func GetProviderByDomain(domain string) *EmailProviderConfig {
	providers := GetBuiltinProviders()

	for _, provider := range providers {
		for _, supportedDomain := range provider.Domains {
			if DomainMatches(supportedDomain, domain) {
				return &provider
			}
		}
	}

	// 如果没有找到匹配的提供商，返回自定义配置
	custom := providers["custom"]
	return &custom
}

// GetProviderByName 根据名称获取提供商配置
func GetProviderByName(name string) *EmailProviderConfig {
	providers := GetBuiltinProviders()
	if provider, exists := providers[name]; exists {
		return &provider
	}
	return nil
}

// DomainMatches 判断域匹配，支持后缀通配 *.（例如 outlook.* 匹配 outlook.com/outlook.fr）
func DomainMatches(pattern string, domain string) bool {
	if pattern == "" || domain == "" {
		return false
	}

	// 通配符后缀：xxx.*
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		return domain == prefix || strings.HasPrefix(domain, prefix+".")
	}

	// 精确匹配
	return pattern == domain
}
