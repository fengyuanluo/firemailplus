package providers

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"firemail/internal/config"
	"firemail/internal/models"
)

// ProviderValidator 提供商验证器
type ProviderValidator struct {
	factory *ProviderFactory
}

// NewProviderValidator 创建提供商验证器
func NewProviderValidator(factory *ProviderFactory) *ProviderValidator {
	return &ProviderValidator{
		factory: factory,
	}
}

// ValidationResult 验证结果
type ValidationResult struct {
	Valid       bool                   `json:"valid"`
	Errors      []ValidationError      `json:"errors,omitempty"`
	Warnings    []ValidationWarning    `json:"warnings,omitempty"`
	Suggestions []ValidationSuggestion `json:"suggestions,omitempty"`
	Score       int                    `json:"score"` // 配置质量评分 (0-100)
}

// ValidationError 验证错误
type ValidationError struct {
	Field    string `json:"field"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // "error", "warning", "info"
}

// ValidationWarning 验证警告
type ValidationWarning struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ValidationSuggestion 验证建议
type ValidationSuggestion struct {
	Field      string      `json:"field"`
	Code       string      `json:"code"`
	Message    string      `json:"message"`
	Suggestion string      `json:"suggestion"`
	NewValue   interface{} `json:"new_value,omitempty"`
}

// ValidateAccount 验证邮件账户配置
func (v *ProviderValidator) ValidateAccount(ctx context.Context, account *models.EmailAccount) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:       true,
		Errors:      make([]ValidationError, 0),
		Warnings:    make([]ValidationWarning, 0),
		Suggestions: make([]ValidationSuggestion, 0),
		Score:       100,
	}

	// 基础字段验证
	v.validateBasicFields(account, result)

	// 邮箱地址验证
	v.validateEmailAddress(account, result)

	// 提供商特定验证
	v.validateProviderSpecific(account, result)

	// 服务器配置验证
	v.validateServerConfig(ctx, account, result)

	// 认证配置验证
	v.validateAuthConfig(account, result)

	// 安全配置验证
	v.validateSecurityConfig(account, result)

	// 计算最终分数和有效性
	v.calculateFinalScore(result)

	return result, nil
}

// validateBasicFields 验证基础字段
func (v *ProviderValidator) validateBasicFields(account *models.EmailAccount, result *ValidationResult) {
	// 验证邮箱地址
	if account.Email == "" {
		v.addError(result, "email", "REQUIRED", "Email address is required", "error")
		return
	}

	// 验证提供商
	if account.Provider == "" {
		v.addError(result, "provider", "REQUIRED", "Provider is required", "error")
		return
	}

	// 验证认证方式
	if account.AuthMethod == "" {
		v.addError(result, "auth_method", "REQUIRED", "Authentication method is required", "error")
		return
	}

	// 验证用户名
	if account.Username == "" {
		v.addWarning(result, "username", "EMPTY", "Username is empty, will use email address")
		v.addSuggestion(result, "username", "AUTO_FILL", "Consider setting username explicitly", "Use email address as username", account.Email)
	}
}

// validateEmailAddress 验证邮箱地址
func (v *ProviderValidator) validateEmailAddress(account *models.EmailAccount, result *ValidationResult) {
	email := account.Email

	// 基本格式验证
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		v.addError(result, "email", "INVALID_FORMAT", "Invalid email address format", "error")
		return
	}

	// 提取域名
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		v.addError(result, "email", "INVALID_FORMAT", "Invalid email address format", "error")
		return
	}

	domain := strings.ToLower(parts[1])

	// 验证域名是否与提供商匹配
	v.validateDomainProviderMatch(domain, account.Provider, result)

	// DNS验证（可选）
	v.validateDomainDNS(domain, result)
}

// validateDomainProviderMatch 验证域名与提供商匹配
func (v *ProviderValidator) validateDomainProviderMatch(domain, provider string, result *ValidationResult) {
	config := v.factory.GetProviderConfig(provider)
	if config == nil {
		v.addError(result, "provider", "NOT_FOUND", fmt.Sprintf("Provider '%s' not found", provider), "error")
		return
	}

	// 检查域名是否在支持列表中
	supported := false
	for _, supportedDomain := range config.Domains {
		if domain == supportedDomain {
			supported = true
			break
		}
	}

	if !supported && len(config.Domains) > 0 {
		v.addWarning(result, "email", "DOMAIN_MISMATCH",
			fmt.Sprintf("Domain '%s' is not in the supported domains list for provider '%s'", domain, provider))

		// 建议更合适的提供商
		v.suggestBetterProvider(domain, result)
	}
}

// validateDomainDNS 验证域名DNS
func (v *ProviderValidator) validateDomainDNS(domain string, result *ValidationResult) {
	// 检查MX记录
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mxRecords, err := net.DefaultResolver.LookupMX(ctx, domain)
	if err != nil {
		v.addWarning(result, "email", "DNS_CHECK_FAILED",
			fmt.Sprintf("Could not verify MX records for domain '%s': %v", domain, err))
		return
	}

	if len(mxRecords) == 0 {
		v.addWarning(result, "email", "NO_MX_RECORDS",
			fmt.Sprintf("No MX records found for domain '%s'", domain))
	}
}

// validateProviderSpecific 验证提供商特定配置
func (v *ProviderValidator) validateProviderSpecific(account *models.EmailAccount, result *ValidationResult) {
	provider, err := v.factory.CreateProviderForAccount(account)
	if err != nil {
		v.addError(result, "provider", "CREATION_FAILED",
			fmt.Sprintf("Failed to create provider: %v", err), "error")
		return
	}

	// 获取提供商信息
	providerInfo := provider.GetProviderInfo()

	// 验证认证方式支持
	authMethods := getStringSlice(providerInfo, "auth_methods")
	if !contains(authMethods, account.AuthMethod) {
		v.addError(result, "auth_method", "UNSUPPORTED",
			fmt.Sprintf("Authentication method '%s' is not supported by provider '%s'", account.AuthMethod, account.Provider), "error")

		if len(authMethods) > 0 {
			v.addSuggestion(result, "auth_method", "SUPPORTED_METHODS",
				"Use a supported authentication method",
				fmt.Sprintf("Supported methods: %s", strings.Join(authMethods, ", ")),
				authMethods[0])
		}
	}

	// 检查弃用警告
	v.checkDeprecationWarnings(account, providerInfo, result)
}

// validateServerConfig 验证服务器配置
func (v *ProviderValidator) validateServerConfig(ctx context.Context, account *models.EmailAccount, result *ValidationResult) {
	// 验证IMAP配置
	if account.IMAPHost != "" {
		v.validateIMAPConfig(ctx, account, result)
	} else {
		v.addWarning(result, "imap_host", "EMPTY", "IMAP host is not configured")
	}

	// 验证SMTP配置
	if account.SMTPHost != "" {
		v.validateSMTPConfig(ctx, account, result)
	} else {
		v.addWarning(result, "smtp_host", "EMPTY", "SMTP host is not configured")
	}
}

// validateIMAPConfig 验证IMAP配置
func (v *ProviderValidator) validateIMAPConfig(ctx context.Context, account *models.EmailAccount, result *ValidationResult) {
	// 验证端口
	if account.IMAPPort <= 0 || account.IMAPPort > 65535 {
		v.addError(result, "imap_port", "INVALID_PORT",
			fmt.Sprintf("Invalid IMAP port: %d", account.IMAPPort), "error")
	}

	// 验证安全设置
	validSecurities := []string{"SSL", "TLS", "STARTTLS", "NONE"}
	if !contains(validSecurities, account.IMAPSecurity) {
		v.addError(result, "imap_security", "INVALID_SECURITY",
			fmt.Sprintf("Invalid IMAP security setting: %s", account.IMAPSecurity), "error")
	}

	// 建议安全端口
	v.suggestSecureIMAPPort(account, result)

	// 测试连接（可选，可能耗时）
	// v.testIMAPConnection(ctx, account, result)
}

// validateSMTPConfig 验证SMTP配置
func (v *ProviderValidator) validateSMTPConfig(ctx context.Context, account *models.EmailAccount, result *ValidationResult) {
	// 验证端口
	if account.SMTPPort <= 0 || account.SMTPPort > 65535 {
		v.addError(result, "smtp_port", "INVALID_PORT",
			fmt.Sprintf("Invalid SMTP port: %d", account.SMTPPort), "error")
	}

	// 验证安全设置
	validSecurities := []string{"SSL", "TLS", "STARTTLS", "NONE"}
	if !contains(validSecurities, account.SMTPSecurity) {
		v.addError(result, "smtp_security", "INVALID_SECURITY",
			fmt.Sprintf("Invalid SMTP security setting: %s", account.SMTPSecurity), "error")
	}

	// 建议安全端口
	v.suggestSecureSMTPPort(account, result)
}

// validateAuthConfig 验证认证配置
func (v *ProviderValidator) validateAuthConfig(account *models.EmailAccount, result *ValidationResult) {
	switch account.AuthMethod {
	case "password":
		v.validatePasswordAuth(account, result)
	case "oauth2":
		v.validateOAuth2Auth(account, result)
	default:
		v.addError(result, "auth_method", "UNKNOWN",
			fmt.Sprintf("Unknown authentication method: %s", account.AuthMethod), "error")
	}
}

// validatePasswordAuth 验证密码认证
func (v *ProviderValidator) validatePasswordAuth(account *models.EmailAccount, result *ValidationResult) {
	if account.Password == "" {
		v.addError(result, "password", "REQUIRED", "Password is required for password authentication", "error")
		return
	}

	// 检查是否需要应用专用密码
	v.checkAppPasswordRequirement(account, result)
}

// validateOAuth2Auth 验证OAuth2认证
func (v *ProviderValidator) validateOAuth2Auth(account *models.EmailAccount, result *ValidationResult) {
	tokenData, err := account.GetOAuth2Token()
	if err != nil {
		v.addError(result, "oauth2_token", "INVALID",
			fmt.Sprintf("Invalid OAuth2 token: %v", err), "error")
		return
	}

	if tokenData == nil {
		v.addError(result, "oauth2_token", "MISSING", "OAuth2 token is missing", "error")
		return
	}

	if tokenData.AccessToken == "" {
		v.addError(result, "oauth2_token", "EMPTY_ACCESS_TOKEN", "OAuth2 access token is empty", "error")
	}

	// 检查token过期
	if time.Now().After(tokenData.Expiry) {
		if tokenData.RefreshToken == "" {
			v.addError(result, "oauth2_token", "EXPIRED_NO_REFRESH",
				"OAuth2 token has expired and no refresh token available", "error")
		} else {
			v.addWarning(result, "oauth2_token", "EXPIRED_HAS_REFRESH",
				"OAuth2 token has expired but can be refreshed")
		}
	}
}

// validateSecurityConfig 验证安全配置
func (v *ProviderValidator) validateSecurityConfig(account *models.EmailAccount, result *ValidationResult) {
	// 检查是否使用安全连接
	if account.IMAPSecurity == "NONE" {
		v.addWarning(result, "imap_security", "INSECURE",
			"IMAP connection is not encrypted. Consider using SSL or STARTTLS")
		result.Score -= 20
	}

	if account.SMTPSecurity == "NONE" {
		v.addWarning(result, "smtp_security", "INSECURE",
			"SMTP connection is not encrypted. Consider using SSL or STARTTLS")
		result.Score -= 20
	}

	// 检查弱密码（如果是密码认证）
	if account.AuthMethod == "password" && len(account.Password) < 8 {
		v.addWarning(result, "password", "WEAK",
			"Password is too short. Consider using a longer password or app-specific password")
		result.Score -= 10
	}
}

// 辅助方法

// addError 添加验证错误
func (v *ProviderValidator) addError(result *ValidationResult, field, code, message, severity string) {
	result.Errors = append(result.Errors, ValidationError{
		Field:    field,
		Code:     code,
		Message:  message,
		Severity: severity,
	})
	result.Valid = false

	// 根据严重程度扣分
	switch severity {
	case "error":
		result.Score -= 25
	case "warning":
		result.Score -= 10
	case "info":
		result.Score -= 5
	}
}

// addWarning 添加验证警告
func (v *ProviderValidator) addWarning(result *ValidationResult, field, code, message string) {
	result.Warnings = append(result.Warnings, ValidationWarning{
		Field:   field,
		Code:    code,
		Message: message,
	})
	result.Score -= 5
}

// addSuggestion 添加验证建议
func (v *ProviderValidator) addSuggestion(result *ValidationResult, field, code, message, suggestion string, newValue interface{}) {
	result.Suggestions = append(result.Suggestions, ValidationSuggestion{
		Field:      field,
		Code:       code,
		Message:    message,
		Suggestion: suggestion,
		NewValue:   newValue,
	})
}

// suggestBetterProvider 建议更好的提供商
func (v *ProviderValidator) suggestBetterProvider(domain string, result *ValidationResult) {
	providers := v.factory.GetAvailableProviders()

	for _, providerName := range providers {
		providerConfig := v.factory.GetProviderConfig(providerName)
		if providerConfig == nil {
			continue
		}

		for _, supportedDomain := range providerConfig.Domains {
			if config.DomainMatches(supportedDomain, domain) {
				v.addSuggestion(result, "provider", "BETTER_MATCH",
					fmt.Sprintf("Provider '%s' supports domain '%s'", providerName, domain),
					fmt.Sprintf("Consider using provider '%s'", providerName),
					providerName)
				return
			}
		}
	}
}

// checkDeprecationWarnings 检查弃用警告
func (v *ProviderValidator) checkDeprecationWarnings(account *models.EmailAccount, providerInfo map[string]interface{}, result *ValidationResult) {
	// 检查基本认证弃用
	if account.AuthMethod == "password" {
		domain := extractDomainFromEmail(account.Email)

		// Microsoft个人账户
		if isMicrosoftPersonalDomain(domain) {
			v.addWarning(result, "auth_method", "DEPRECATED",
				"Basic authentication is deprecated for Microsoft personal accounts. Use OAuth2 instead")
			v.addSuggestion(result, "auth_method", "USE_OAUTH2",
				"Switch to OAuth2 for better security", "Use OAuth2 authentication", "oauth2")
		}
	}

	// 检查提供商特定的弃用信息
	if deprecationNotice, ok := providerInfo["deprecation_notice"].(string); ok && deprecationNotice != "" {
		v.addWarning(result, "provider", "DEPRECATION_NOTICE", deprecationNotice)
	}
}

// checkAppPasswordRequirement 检查应用专用密码要求
func (v *ProviderValidator) checkAppPasswordRequirement(account *models.EmailAccount, result *ValidationResult) {
	domain := extractDomainFromEmail(account.Email)

	// Gmail需要应用专用密码
	if domain == "gmail.com" || domain == "googlemail.com" {
		if len(account.Password) != 16 {
			v.addWarning(result, "password", "APP_PASSWORD_REQUIRED",
				"Gmail requires 16-character app-specific password, not regular password")
			v.addSuggestion(result, "password", "GENERATE_APP_PASSWORD",
				"Generate app-specific password in Google Account settings",
				"Visit https://myaccount.google.com/apppasswords", nil)
		}
	}

	// QQ邮箱需要授权码
	if domain == "qq.com" || domain == "vip.qq.com" || domain == "foxmail.com" {
		if len(account.Password) != 16 {
			v.addWarning(result, "password", "AUTH_CODE_REQUIRED",
				"QQ Mail requires 16-character authorization code, not regular password")
			v.addSuggestion(result, "password", "GENERATE_AUTH_CODE",
				"Generate authorization code in QQ Mail settings",
				"Visit QQ Mail settings to generate authorization code", nil)
		}
	}

	// 网易邮箱需要客户端授权码
	neteaseDomains := []string{"163.com", "126.com", "yeah.net"}
	if contains(neteaseDomains, domain) {
		if len(account.Password) != 16 {
			v.addWarning(result, "password", "CLIENT_AUTH_CODE_REQUIRED",
				"NetEase Mail requires 16-character client authorization code, not regular password")
			v.addSuggestion(result, "password", "GENERATE_CLIENT_AUTH_CODE",
				"Generate client authorization code in NetEase Mail settings",
				"Visit NetEase Mail settings to generate client authorization code", nil)
		}
	}

	// iCloud需要应用专用密码
	icloudDomains := []string{"icloud.com", "me.com", "mac.com"}
	if contains(icloudDomains, domain) {
		if len(account.Password) != 19 || !strings.Contains(account.Password, "-") {
			v.addWarning(result, "password", "APP_SPECIFIC_PASSWORD_REQUIRED",
				"iCloud requires app-specific password in format xxxx-xxxx-xxxx-xxxx")
			v.addSuggestion(result, "password", "GENERATE_APP_SPECIFIC_PASSWORD",
				"Generate app-specific password in Apple ID settings",
				"Visit https://appleid.apple.com to generate app-specific password", nil)
		}
	}
}

// suggestSecureIMAPPort 建议安全的IMAP端口
func (v *ProviderValidator) suggestSecureIMAPPort(account *models.EmailAccount, result *ValidationResult) {
	if account.IMAPSecurity == "NONE" && account.IMAPPort == 143 {
		v.addSuggestion(result, "imap_port", "USE_SECURE_PORT",
			"Use secure IMAP port for better security",
			"Use port 993 with SSL encryption", 993)
		v.addSuggestion(result, "imap_security", "USE_SSL",
			"Enable SSL encryption for IMAP",
			"Use SSL security", "SSL")
	}
}

// suggestSecureSMTPPort 建议安全的SMTP端口
func (v *ProviderValidator) suggestSecureSMTPPort(account *models.EmailAccount, result *ValidationResult) {
	if account.SMTPSecurity == "NONE" && account.SMTPPort == 25 {
		v.addSuggestion(result, "smtp_port", "USE_SECURE_PORT",
			"Use secure SMTP port for better security",
			"Use port 465 (SSL) or 587 (STARTTLS)", 587)
		v.addSuggestion(result, "smtp_security", "USE_ENCRYPTION",
			"Enable encryption for SMTP",
			"Use STARTTLS or SSL security", "STARTTLS")
	}
}

// calculateFinalScore 计算最终分数
func (v *ProviderValidator) calculateFinalScore(result *ValidationResult) {
	// 确保分数在0-100范围内
	if result.Score < 0 {
		result.Score = 0
	}
	if result.Score > 100 {
		result.Score = 100
	}

	// 如果有错误，配置无效
	if len(result.Errors) > 0 {
		result.Valid = false
	}
}

// QuickValidate 快速验证（只检查基本字段）
func (v *ProviderValidator) QuickValidate(account *models.EmailAccount) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:       true,
		Errors:      make([]ValidationError, 0),
		Warnings:    make([]ValidationWarning, 0),
		Suggestions: make([]ValidationSuggestion, 0),
		Score:       100,
	}

	// 只进行基础验证
	v.validateBasicFields(account, result)
	v.validateEmailAddress(account, result)

	v.calculateFinalScore(result)

	return result, nil
}

// ValidateAndSuggestFix 验证并建议修复
func (v *ProviderValidator) ValidateAndSuggestFix(ctx context.Context, account *models.EmailAccount) (*models.EmailAccount, *ValidationResult, error) {
	// 先进行完整验证
	result, err := v.ValidateAccount(ctx, account)
	if err != nil {
		return nil, nil, err
	}

	// 创建修复后的账户副本
	fixedAccount := *account

	// 应用建议的修复
	for _, suggestion := range result.Suggestions {
		if suggestion.NewValue != nil {
			v.applySuggestion(&fixedAccount, suggestion)
		}
	}

	return &fixedAccount, result, nil
}

// applySuggestion 应用建议
func (v *ProviderValidator) applySuggestion(account *models.EmailAccount, suggestion ValidationSuggestion) {
	switch suggestion.Field {
	case "username":
		if suggestion.Code == "AUTO_FILL" {
			if str, ok := suggestion.NewValue.(string); ok {
				account.Username = str
			}
		}
	case "auth_method":
		if str, ok := suggestion.NewValue.(string); ok {
			account.AuthMethod = str
		}
	case "provider":
		if str, ok := suggestion.NewValue.(string); ok {
			account.Provider = str
		}
	case "imap_port":
		if port, ok := suggestion.NewValue.(int); ok {
			account.IMAPPort = port
		}
	case "smtp_port":
		if port, ok := suggestion.NewValue.(int); ok {
			account.SMTPPort = port
		}
	case "imap_security":
		if str, ok := suggestion.NewValue.(string); ok {
			account.IMAPSecurity = str
		}
	case "smtp_security":
		if str, ok := suggestion.NewValue.(string); ok {
			account.SMTPSecurity = str
		}
	}
}

func isMicrosoftPersonalDomain(domain string) bool {
	if domain == "" {
		return false
	}
	return strings.HasPrefix(domain, "outlook.") ||
		strings.HasPrefix(domain, "hotmail.") ||
		strings.HasPrefix(domain, "live.") ||
		strings.HasPrefix(domain, "msn.")
}
