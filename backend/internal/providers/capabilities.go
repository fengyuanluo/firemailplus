package providers

import (
	"context"
	"fmt"
	"time"

	"firemail/internal/models"
)

// ProviderCapabilities 提供商能力定义
type ProviderCapabilities struct {
	// 基础功能
	IMAP      bool `json:"imap"`
	SMTP      bool `json:"smtp"`
	OAuth2    bool `json:"oauth2"`
	BasicAuth bool `json:"basic_auth"`

	// 高级功能
	Push       bool `json:"push"`
	Threading  bool `json:"threading"`
	Labels     bool `json:"labels"`
	Folders    bool `json:"folders"`
	Search     bool `json:"search"`
	IDLE       bool `json:"idle"`
	Extensions bool `json:"extensions"`

	// 特殊功能
	Categories bool `json:"categories"`
	Rules      bool `json:"rules"`
	Notes      bool `json:"notes"`
	Calendar   bool `json:"calendar"`

	// 限制信息
	Limits ProviderLimits `json:"limits"`

	// 支持的认证方式
	AuthMethods []string `json:"auth_methods"`

	// 支持的域名
	Domains []string `json:"domains"`

	// 服务器配置
	Servers ServerConfig `json:"servers"`
}

// ProviderLimits 提供商限制
type ProviderLimits struct {
	AttachmentSize  int64 `json:"attachment_size"`   // 附件大小限制（字节）
	DailySend       int   `json:"daily_send"`        // 每日发送限制
	HourlySend      int   `json:"hourly_send"`       // 每小时发送限制
	RateLimitWindow int   `json:"rate_limit_window"` // 频率限制窗口（秒）
	MaxRecipients   int   `json:"max_recipients"`    // 单封邮件最大收件人数
	MailboxSize     int64 `json:"mailbox_size"`      // 邮箱容量限制（字节）
	ConnectionLimit int   `json:"connection_limit"`  // 同时连接数限制
}

// ServerConfig 服务器配置
type ServerConfig struct {
	IMAP IMAPConfig `json:"imap"`
	SMTP SMTPConfig `json:"smtp"`
}

// IMAPConfig IMAP服务器配置
type IMAPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Security string `json:"security"`
}

// SMTPConfig SMTP服务器配置
type SMTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Security string `json:"security"`
	AltPort  int    `json:"alt_port,omitempty"`
}

// CapabilityDetector 能力检测器接口
type CapabilityDetector interface {
	// DetectCapabilities 检测提供商能力
	DetectCapabilities(ctx context.Context, account *models.EmailAccount) (*ProviderCapabilities, error)

	// TestFeature 测试特定功能
	TestFeature(ctx context.Context, account *models.EmailAccount, feature string) (bool, error)

	// GetOptimalConfig 获取最优配置
	GetOptimalConfig(ctx context.Context, account *models.EmailAccount) (*OptimalConfig, error)
}

// OptimalConfig 最优配置
type OptimalConfig struct {
	AuthMethod          string                 `json:"auth_method"`
	IMAPConfig          IMAPConfig             `json:"imap_config"`
	SMTPConfig          SMTPConfig             `json:"smtp_config"`
	RecommendedSettings map[string]interface{} `json:"recommended_settings"`
	Warnings            []string               `json:"warnings"`
}

// StandardCapabilityDetector 标准能力检测器实现
type StandardCapabilityDetector struct {
	provider        EmailProvider
	providerFactory *ProviderFactory
}

// NewCapabilityDetector 创建能力检测器
func NewCapabilityDetector(provider EmailProvider) CapabilityDetector {
	return &StandardCapabilityDetector{
		provider: provider,
	}
}

// NewCapabilityDetectorWithFactory 创建带工厂的能力检测器
func NewCapabilityDetectorWithFactory(provider EmailProvider, factory *ProviderFactory) CapabilityDetector {
	return &StandardCapabilityDetector{
		provider:        provider,
		providerFactory: factory,
	}
}

// DetectCapabilities 检测提供商能力
func (d *StandardCapabilityDetector) DetectCapabilities(ctx context.Context, account *models.EmailAccount) (*ProviderCapabilities, error) {
	// 获取提供商信息
	providerInfo := d.provider.GetProviderInfo()

	capabilities := &ProviderCapabilities{
		AuthMethods: getStringSlice(providerInfo, "auth_methods"),
		Domains:     getStringSlice(providerInfo, "domains"),
	}

	// 解析功能特性
	if features, ok := providerInfo["features"].(map[string]bool); ok {
		capabilities.IMAP = features["imap"]
		capabilities.SMTP = features["smtp"]
		capabilities.OAuth2 = features["oauth2"]
		capabilities.BasicAuth = features["basic_auth"]
		capabilities.Push = features["push"]
		capabilities.Threading = features["threading"]
		capabilities.Labels = features["labels"]
		capabilities.Folders = features["folders"]
		capabilities.Search = features["search"]
		capabilities.IDLE = features["idle"]
		capabilities.Extensions = features["extensions"]
		capabilities.Categories = features["categories"]
		capabilities.Rules = features["rules"]
		capabilities.Notes = features["notes"]
		capabilities.Calendar = features["calendar"]
	}

	// 解析限制信息
	if limits, ok := providerInfo["limits"].(map[string]interface{}); ok {
		capabilities.Limits = ProviderLimits{
			AttachmentSize:  getInt64(limits, "attachment_size"),
			DailySend:       getInt(limits, "daily_send"),
			HourlySend:      getInt(limits, "hourly_send"),
			RateLimitWindow: getInt(limits, "rate_limit_window"),
			MaxRecipients:   getInt(limits, "max_recipients"),
			MailboxSize:     getInt64(limits, "mailbox_size"),
			ConnectionLimit: getInt(limits, "connection_limit"),
		}
	}

	// 解析服务器配置
	if servers, ok := providerInfo["servers"].(map[string]interface{}); ok {
		capabilities.Servers = d.parseServerConfig(servers)
	}

	return capabilities, nil
}

// TestFeature 测试特定功能
func (d *StandardCapabilityDetector) TestFeature(ctx context.Context, account *models.EmailAccount, feature string) (bool, error) {
	// 创建测试上下文，设置超时
	testCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	switch feature {
	case "imap":
		return d.testIMAPConnection(testCtx, account)
	case "smtp":
		return d.testSMTPConnection(testCtx, account)
	case "oauth2":
		return d.testOAuth2Auth(testCtx, account)
	case "basic_auth":
		return d.testBasicAuth(testCtx, account)
	case "idle":
		return d.testIDLESupport(testCtx, account)
	case "search":
		return d.testSearchSupport(testCtx, account)
	default:
		return false, fmt.Errorf("unsupported feature test: %s", feature)
	}
}

// GetOptimalConfig 获取最优配置
func (d *StandardCapabilityDetector) GetOptimalConfig(ctx context.Context, account *models.EmailAccount) (*OptimalConfig, error) {
	capabilities, err := d.DetectCapabilities(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to detect capabilities: %w", err)
	}

	config := &OptimalConfig{
		RecommendedSettings: make(map[string]interface{}),
		Warnings:            make([]string, 0),
	}

	// 选择最佳认证方式
	config.AuthMethod = d.selectBestAuthMethod(capabilities, account)

	// 设置服务器配置
	config.IMAPConfig = capabilities.Servers.IMAP
	config.SMTPConfig = capabilities.Servers.SMTP

	// 添加推荐设置
	d.addRecommendedSettings(config, capabilities)

	// 添加警告信息
	d.addWarnings(config, capabilities, account)

	return config, nil
}

// selectBestAuthMethod 选择最佳认证方式
func (d *StandardCapabilityDetector) selectBestAuthMethod(capabilities *ProviderCapabilities, account *models.EmailAccount) string {
	// 优先选择OAuth2（更安全）
	if capabilities.OAuth2 && contains(capabilities.AuthMethods, "oauth2") {
		return "oauth2"
	}

	// 其次选择基本认证
	if capabilities.BasicAuth && contains(capabilities.AuthMethods, "password") {
		return "password"
	}

	// 默认返回第一个支持的认证方式
	if len(capabilities.AuthMethods) > 0 {
		return capabilities.AuthMethods[0]
	}

	return "password" // 默认
}

// addRecommendedSettings 添加推荐设置
func (d *StandardCapabilityDetector) addRecommendedSettings(config *OptimalConfig, capabilities *ProviderCapabilities) {
	// 连接设置
	config.RecommendedSettings["use_ssl"] = true
	config.RecommendedSettings["verify_certificate"] = true

	// 功能设置
	if capabilities.IDLE {
		config.RecommendedSettings["enable_idle"] = true
		config.RecommendedSettings["idle_timeout"] = 1740 // 29分钟
	}

	if capabilities.Push {
		config.RecommendedSettings["enable_push"] = true
	}

	// 性能设置
	if capabilities.Limits.ConnectionLimit > 0 {
		config.RecommendedSettings["max_connections"] = capabilities.Limits.ConnectionLimit
	}

	// 发送限制
	if capabilities.Limits.DailySend > 0 {
		config.RecommendedSettings["daily_send_limit"] = capabilities.Limits.DailySend
	}

	if capabilities.Limits.AttachmentSize > 0 {
		config.RecommendedSettings["max_attachment_size"] = capabilities.Limits.AttachmentSize
	}
}

// addWarnings 添加警告信息
func (d *StandardCapabilityDetector) addWarnings(config *OptimalConfig, capabilities *ProviderCapabilities, account *models.EmailAccount) {
	// 认证方式警告
	if config.AuthMethod == "password" && capabilities.OAuth2 {
		config.Warnings = append(config.Warnings, "OAuth2 authentication is recommended for better security")
	}

	// 基本认证弃用警告
	if config.AuthMethod == "password" {
		domain := extractDomainFromEmail(account.Email)
		if domain == "outlook.com" || domain == "hotmail.com" || domain == "live.com" {
			config.Warnings = append(config.Warnings, "Basic authentication is deprecated for Microsoft personal accounts")
		}
	}

	// 功能限制警告
	if !capabilities.IDLE {
		config.Warnings = append(config.Warnings, "IDLE not supported, will use polling for new messages")
	}

	if !capabilities.Push {
		config.Warnings = append(config.Warnings, "Push notifications not supported")
	}

	// 容量警告
	if capabilities.Limits.MailboxSize > 0 && capabilities.Limits.MailboxSize < 1024*1024*1024 {
		config.Warnings = append(config.Warnings, fmt.Sprintf("Limited mailbox size: %d MB", capabilities.Limits.MailboxSize/(1024*1024)))
	}
}

// 测试方法实现

// testIMAPConnection 测试IMAP连接
func (d *StandardCapabilityDetector) testIMAPConnection(ctx context.Context, account *models.EmailAccount) (bool, error) {
	// 尝试连接IMAP服务器
	err := d.provider.TestConnection(ctx, account)
	return err == nil, err
}

// testSMTPConnection 测试SMTP连接
func (d *StandardCapabilityDetector) testSMTPConnection(ctx context.Context, account *models.EmailAccount) (bool, error) {
	// 这里需要实现SMTP连接测试
	// 暂时返回true，实际实现中应该测试SMTP连接
	return true, nil
}

// testOAuth2Auth 测试OAuth2认证
func (d *StandardCapabilityDetector) testOAuth2Auth(ctx context.Context, account *models.EmailAccount) (bool, error) {
	if account.AuthMethod != "oauth2" {
		return false, fmt.Errorf("account not configured for OAuth2")
	}

	tokenData, err := account.GetOAuth2Token()
	if err != nil {
		return false, err
	}

	return tokenData != nil && tokenData.AccessToken != "", nil
}

// testBasicAuth 测试基本认证
func (d *StandardCapabilityDetector) testBasicAuth(ctx context.Context, account *models.EmailAccount) (bool, error) {
	if account.AuthMethod != "password" {
		return false, fmt.Errorf("account not configured for basic auth")
	}

	return account.Password != "", nil
}

// testIDLESupport 测试IDLE支持
func (d *StandardCapabilityDetector) testIDLESupport(ctx context.Context, account *models.EmailAccount) (bool, error) {
	// 这里需要实现IDLE功能测试
	// 暂时基于提供商信息返回
	capabilities, err := d.DetectCapabilities(ctx, account)
	if err != nil {
		return false, err
	}

	return capabilities.IDLE, nil
}

// testSearchSupport 测试搜索支持
func (d *StandardCapabilityDetector) testSearchSupport(ctx context.Context, account *models.EmailAccount) (bool, error) {
	// 如果没有工厂，使用当前提供商
	var provider EmailProvider
	if d.providerFactory != nil {
		var err error
		provider, err = d.providerFactory.CreateProviderForAccount(account)
		if err != nil {
			return false, fmt.Errorf("failed to create provider: %w", err)
		}
	} else {
		provider = d.provider
	}

	// 连接到邮件服务器
	if err := provider.Connect(ctx, account); err != nil {
		return false, fmt.Errorf("failed to connect to email server: %w", err)
	}
	defer provider.Disconnect()

	// 获取IMAP客户端
	imapClient := provider.IMAPClient()
	if imapClient == nil {
		return false, fmt.Errorf("IMAP client not available")
	}

	// 测试基本搜索功能
	return d.performSearchTest(ctx, imapClient)
}

// performSearchTest 执行搜索功能测试
func (d *StandardCapabilityDetector) performSearchTest(ctx context.Context, imapClient IMAPClient) (bool, error) {
	// 获取文件夹列表
	folders, err := imapClient.ListFolders(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to list folders: %w", err)
	}

	// 找到一个可选择的文件夹进行测试
	var testFolder string
	for _, folder := range folders {
		if folder.IsSelectable {
			testFolder = folder.Name
			break
		}
	}

	if testFolder == "" {
		return false, fmt.Errorf("no selectable folder found for search test")
	}

	// 测试多种搜索条件
	searchTests := []struct {
		name     string
		criteria *SearchCriteria
	}{
		{
			name: "主题搜索",
			criteria: &SearchCriteria{
				FolderName: testFolder,
				Subject:    "subject", // 搜索主题包含"subject"的邮件
			},
		},
		{
			name: "发件人搜索",
			criteria: &SearchCriteria{
				FolderName: testFolder,
				From:       "@", // 搜索包含@符号的发件人（基本的邮箱格式）
			},
		},
		{
			name: "正文搜索",
			criteria: &SearchCriteria{
				FolderName: testFolder,
				Body:       "test", // 搜索正文包含"test"的邮件
			},
		},
		{
			name: "未读邮件搜索",
			criteria: &SearchCriteria{
				FolderName: testFolder,
				Seen:       boolPtr(false), // 搜索未读邮件
			},
		},
		{
			name: "日期范围搜索",
			criteria: &SearchCriteria{
				FolderName: testFolder,
				Since:      func() *time.Time { t := time.Now().AddDate(0, 0, -30); return &t }(), // 最近30天的邮件
			},
		},
	}

	// 执行搜索测试
	successCount := 0
	for _, test := range searchTests {
		// 创建带超时的上下文
		testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)

		// 执行搜索
		_, err := imapClient.SearchEmails(testCtx, test.criteria)
		cancel()

		if err == nil {
			successCount++
		} else {
			// 记录搜索失败的详细信息，但不立即返回错误
			fmt.Printf("Search test '%s' failed: %v\n", test.name, err)
		}
	}

	// 如果至少有一半的搜索测试成功，认为搜索功能可用
	threshold := len(searchTests) / 2
	if threshold == 0 {
		threshold = 1
	}

	searchSupported := successCount >= threshold

	if !searchSupported {
		return false, fmt.Errorf("search functionality test failed: only %d/%d tests passed", successCount, len(searchTests))
	}

	return true, nil
}

// 辅助函数

// boolPtr 返回bool指针
func boolPtr(b bool) *bool {
	return &b
}

// timePtr 返回time指针
func timePtr(t time.Time) *time.Time {
	return &t
}

// parseServerConfig 解析服务器配置
func (d *StandardCapabilityDetector) parseServerConfig(servers map[string]interface{}) ServerConfig {
	config := ServerConfig{}

	if imap, ok := servers["imap"].(map[string]interface{}); ok {
		config.IMAP = IMAPConfig{
			Host:     getString(imap, "host"),
			Port:     getInt(imap, "port"),
			Security: getString(imap, "security"),
		}
	}

	if smtp, ok := servers["smtp"].(map[string]interface{}); ok {
		config.SMTP = SMTPConfig{
			Host:     getString(smtp, "host"),
			Port:     getInt(smtp, "port"),
			Security: getString(smtp, "security"),
			AltPort:  getInt(smtp, "alt_port"),
		}
	}

	return config
}

// 辅助函数

// getStringSlice 从map中获取字符串切片
func getStringSlice(m map[string]interface{}, key string) []string {
	if val, ok := m[key].([]string); ok {
		return val
	}
	if val, ok := m[key].([]interface{}); ok {
		result := make([]string, len(val))
		for i, v := range val {
			if s, ok := v.(string); ok {
				result[i] = s
			}
		}
		return result
	}
	return []string{}
}

// getString 从map中获取字符串
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// getInt 从map中获取整数
func getInt(m map[string]interface{}, key string) int {
	if val, ok := m[key].(int); ok {
		return val
	}
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	return 0
}

// getInt64 从map中获取int64
func getInt64(m map[string]interface{}, key string) int64 {
	if val, ok := m[key].(int64); ok {
		return val
	}
	if val, ok := m[key].(int); ok {
		return int64(val)
	}
	if val, ok := m[key].(float64); ok {
		return int64(val)
	}
	return 0
}

// contains 检查字符串切片是否包含指定字符串
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
