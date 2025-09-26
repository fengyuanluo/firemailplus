package providers

import (
	"context"
	"fmt"
	"log"
	"time"

	"firemail/internal/models"
)

// AdaptiveManager 自适应管理器
type AdaptiveManager struct {
	factory *ProviderFactory
}

// NewAdaptiveManager 创建自适应管理器
func NewAdaptiveManager(factory *ProviderFactory) *AdaptiveManager {
	return &AdaptiveManager{
		factory: factory,
	}
}

// OptimizeAccount 优化邮件账户配置
func (am *AdaptiveManager) OptimizeAccount(ctx context.Context, account *models.EmailAccount) (*OptimalConfig, error) {
	// 创建提供商实例
	provider, err := am.factory.CreateProviderForAccount(account)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// 创建能力检测器
	detector := NewCapabilityDetector(provider)

	// 获取最优配置
	config, err := detector.GetOptimalConfig(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to get optimal config: %w", err)
	}

	return config, nil
}

// AutoConfigureAccount 自动配置邮件账户
func (am *AdaptiveManager) AutoConfigureAccount(ctx context.Context, account *models.EmailAccount) error {
	// 获取最优配置
	config, err := am.OptimizeAccount(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to optimize account: %w", err)
	}

	// 应用配置
	return am.applyOptimalConfig(account, config)
}

// applyOptimalConfig 应用最优配置
func (am *AdaptiveManager) applyOptimalConfig(account *models.EmailAccount, config *OptimalConfig) error {
	// 更新认证方式
	if config.AuthMethod != "" && config.AuthMethod != account.AuthMethod {
		log.Printf("Updating auth method from %s to %s for account %s",
			account.AuthMethod, config.AuthMethod, account.Email)
		account.AuthMethod = config.AuthMethod
	}

	// 更新IMAP配置
	if config.IMAPConfig.Host != "" {
		account.IMAPHost = config.IMAPConfig.Host
		account.IMAPPort = config.IMAPConfig.Port
		account.IMAPSecurity = config.IMAPConfig.Security
	}

	// 更新SMTP配置
	if config.SMTPConfig.Host != "" {
		account.SMTPHost = config.SMTPConfig.Host
		account.SMTPPort = config.SMTPConfig.Port
		account.SMTPSecurity = config.SMTPConfig.Security
	}

	// 记录警告信息
	for _, warning := range config.Warnings {
		log.Printf("Warning for account %s: %s", account.Email, warning)
	}

	return nil
}

// TestAccountCapabilities 测试账户能力
func (am *AdaptiveManager) TestAccountCapabilities(ctx context.Context, account *models.EmailAccount) (*CapabilityTestResult, error) {
	// 创建提供商实例
	provider, err := am.factory.CreateProviderForAccount(account)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// 创建能力检测器
	detector := NewCapabilityDetector(provider)

	// 测试结果
	result := &CapabilityTestResult{
		AccountEmail: account.Email,
		Provider:     account.Provider,
		TestTime:     time.Now(),
		Tests:        make(map[string]FeatureTestResult),
	}

	// 测试各项功能
	features := []string{"imap", "smtp", "oauth2", "basic_auth", "idle", "search"}

	for _, feature := range features {
		testResult := FeatureTestResult{
			Feature: feature,
			Tested:  true,
		}

		supported, err := detector.TestFeature(ctx, account, feature)
		if err != nil {
			testResult.Error = err.Error()
			testResult.Supported = false
		} else {
			testResult.Supported = supported
		}

		result.Tests[feature] = testResult
	}

	// 获取能力信息
	capabilities, err := detector.DetectCapabilities(ctx, account)
	if err != nil {
		result.Error = err.Error()
	} else {
		result.Capabilities = capabilities
	}

	return result, nil
}

// CapabilityTestResult 能力测试结果
type CapabilityTestResult struct {
	AccountEmail string                       `json:"account_email"`
	Provider     string                       `json:"provider"`
	TestTime     time.Time                    `json:"test_time"`
	Tests        map[string]FeatureTestResult `json:"tests"`
	Capabilities *ProviderCapabilities        `json:"capabilities,omitempty"`
	Error        string                       `json:"error,omitempty"`
}

// FeatureTestResult 功能测试结果
type FeatureTestResult struct {
	Feature   string `json:"feature"`
	Supported bool   `json:"supported"`
	Tested    bool   `json:"tested"`
	Error     string `json:"error,omitempty"`
}

// RecommendProvider 推荐邮件提供商
func (am *AdaptiveManager) RecommendProvider(email string) (*ProviderRecommendation, error) {
	// 检测邮箱域名
	domain := extractDomainFromEmail(email)
	if domain == "" {
		return nil, fmt.Errorf("invalid email address")
	}

	// 获取所有可用提供商
	providers := am.factory.GetAvailableProviders()

	recommendation := &ProviderRecommendation{
		Email:      email,
		Domain:     domain,
		Candidates: make([]ProviderCandidate, 0),
	}

	// 评估每个提供商
	for _, providerName := range providers {
		config := am.factory.GetProviderConfig(providerName)
		if config == nil {
			continue
		}

		candidate := ProviderCandidate{
			Name:        providerName,
			DisplayName: config.DisplayName,
			Score:       0,
			Reasons:     make([]string, 0),
		}

		// 检查域名匹配
		if am.isDomainSupported(domain, config.Domains) {
			candidate.Score += 100 // 完全匹配得高分
			candidate.Reasons = append(candidate.Reasons, "Domain exactly matches")
		} else if am.isDomainSimilar(domain, config.Domains) {
			candidate.Score += 50 // 相似域名得中等分
			candidate.Reasons = append(candidate.Reasons, "Domain is similar")
		}

		// 评估认证方式
		if contains(config.AuthMethods, "oauth2") {
			candidate.Score += 30
			candidate.Reasons = append(candidate.Reasons, "Supports OAuth2")
		}

		if contains(config.AuthMethods, "password") {
			candidate.Score += 10
			candidate.Reasons = append(candidate.Reasons, "Supports password auth")
		}

		// 评估功能特性
		if config.Features["push"] {
			candidate.Score += 10
			candidate.Reasons = append(candidate.Reasons, "Supports push notifications")
		}

		if config.Features["search"] {
			candidate.Score += 5
			candidate.Reasons = append(candidate.Reasons, "Supports server-side search")
		}

		// 只添加有分数的候选者
		if candidate.Score > 0 {
			recommendation.Candidates = append(recommendation.Candidates, candidate)
		}
	}

	// 按分数排序
	am.sortCandidatesByScore(recommendation.Candidates)

	// 设置推荐结果
	if len(recommendation.Candidates) > 0 {
		recommendation.Recommended = &recommendation.Candidates[0]
	}

	return recommendation, nil
}

// ProviderRecommendation 提供商推荐结果
type ProviderRecommendation struct {
	Email       string              `json:"email"`
	Domain      string              `json:"domain"`
	Recommended *ProviderCandidate  `json:"recommended,omitempty"`
	Candidates  []ProviderCandidate `json:"candidates"`
}

// ProviderCandidate 提供商候选者
type ProviderCandidate struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Score       int      `json:"score"`
	Reasons     []string `json:"reasons"`
}

// isDomainSupported 检查域名是否被支持
func (am *AdaptiveManager) isDomainSupported(domain string, supportedDomains []string) bool {
	for _, supported := range supportedDomains {
		if domain == supported {
			return true
		}
	}
	return false
}

// isDomainSimilar 检查域名是否相似
func (am *AdaptiveManager) isDomainSimilar(domain string, supportedDomains []string) bool {
	// 简单的相似性检查，可以扩展为更复杂的算法
	for _, supported := range supportedDomains {
		if len(domain) > 3 && len(supported) > 3 {
			// 检查是否包含相同的子字符串
			if contains([]string{supported}, domain[:3]) || contains([]string{domain}, supported[:3]) {
				return true
			}
		}
	}
	return false
}

// sortCandidatesByScore 按分数排序候选者
func (am *AdaptiveManager) sortCandidatesByScore(candidates []ProviderCandidate) {
	// 简单的冒泡排序，按分数降序
	n := len(candidates)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if candidates[j].Score < candidates[j+1].Score {
				candidates[j], candidates[j+1] = candidates[j+1], candidates[j]
			}
		}
	}
}

// GetProviderMigrationPlan 获取提供商迁移计划
func (am *AdaptiveManager) GetProviderMigrationPlan(ctx context.Context, fromAccount, toAccount *models.EmailAccount) (*MigrationPlan, error) {
	plan := &MigrationPlan{
		FromProvider: fromAccount.Provider,
		ToProvider:   toAccount.Provider,
		Steps:        make([]MigrationStep, 0),
		Warnings:     make([]string, 0),
	}

	// 检测源和目标提供商能力
	fromCapabilities, err := am.getAccountCapabilities(ctx, fromAccount)
	if err != nil {
		return nil, fmt.Errorf("failed to detect source capabilities: %w", err)
	}

	toCapabilities, err := am.getAccountCapabilities(ctx, toAccount)
	if err != nil {
		return nil, fmt.Errorf("failed to detect target capabilities: %w", err)
	}

	// 生成迁移步骤
	plan.Steps = append(plan.Steps, MigrationStep{
		Step:        1,
		Description: "Backup current email data",
		Action:      "backup",
		Required:    true,
	})

	plan.Steps = append(plan.Steps, MigrationStep{
		Step:        2,
		Description: "Configure new email account",
		Action:      "configure",
		Required:    true,
	})

	plan.Steps = append(plan.Steps, MigrationStep{
		Step:        3,
		Description: "Migrate email folders and messages",
		Action:      "migrate",
		Required:    true,
	})

	// 检查功能兼容性并添加警告
	am.addMigrationWarnings(plan, fromCapabilities, toCapabilities)

	return plan, nil
}

// MigrationPlan 迁移计划
type MigrationPlan struct {
	FromProvider  string          `json:"from_provider"`
	ToProvider    string          `json:"to_provider"`
	Steps         []MigrationStep `json:"steps"`
	Warnings      []string        `json:"warnings"`
	EstimatedTime string          `json:"estimated_time"`
}

// MigrationStep 迁移步骤
type MigrationStep struct {
	Step        int    `json:"step"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Required    bool   `json:"required"`
}

// getAccountCapabilities 获取账户能力
func (am *AdaptiveManager) getAccountCapabilities(ctx context.Context, account *models.EmailAccount) (*ProviderCapabilities, error) {
	provider, err := am.factory.CreateProviderForAccount(account)
	if err != nil {
		return nil, err
	}

	detector := NewCapabilityDetector(provider)
	return detector.DetectCapabilities(ctx, account)
}

// addMigrationWarnings 添加迁移警告
func (am *AdaptiveManager) addMigrationWarnings(plan *MigrationPlan, from, to *ProviderCapabilities) {
	// 检查标签vs文件夹
	if from.Labels && !to.Labels {
		plan.Warnings = append(plan.Warnings, "Source provider uses labels, target uses folders. Some organization may be lost.")
	}

	// 检查搜索功能
	if from.Search && !to.Search {
		plan.Warnings = append(plan.Warnings, "Target provider has limited search capabilities.")
	}

	// 检查存储限制
	if to.Limits.MailboxSize > 0 && from.Limits.MailboxSize > to.Limits.MailboxSize {
		plan.Warnings = append(plan.Warnings, "Target provider has smaller storage capacity.")
	}

	// 检查发送限制
	if to.Limits.DailySend > 0 && from.Limits.DailySend > to.Limits.DailySend {
		plan.Warnings = append(plan.Warnings, "Target provider has lower daily sending limits.")
	}
}
