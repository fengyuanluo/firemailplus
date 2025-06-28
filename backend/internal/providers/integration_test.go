//go:build integration
// +build integration

package providers

import (
	"context"
	"os"
	"testing"
	"time"

	"firemail/internal/config"
	"firemail/internal/models"
)

// 集成测试需要真实的邮箱凭据
// 通过环境变量提供测试凭据

func TestGmailIntegration(t *testing.T) {
	email := os.Getenv("GMAIL_TEST_EMAIL")
	password := os.Getenv("GMAIL_TEST_APP_PASSWORD")

	if email == "" || password == "" {
		t.Skip("Skipping Gmail integration test: GMAIL_TEST_EMAIL and GMAIL_TEST_APP_PASSWORD not set")
	}

	config := &config.EmailProviderConfig{
		Name:         "gmail",
		DisplayName:  "Gmail",
		IMAPHost:     "imap.gmail.com",
		IMAPPort:     993,
		IMAPSecurity: "SSL",
		SMTPHost:     "smtp.gmail.com",
		SMTPPort:     465,
		SMTPSecurity: "SSL",
		AuthMethods:  []string{"password"},
		Domains:      []string{"gmail.com"},
	}

	provider := NewGmailProvider(config)
	account := &models.EmailAccount{
		Email:      email,
		Password:   password,
		AuthMethod: "password",
		Provider:   "gmail",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Connection Test", func(t *testing.T) {
		err := provider.Connect(ctx, account)
		if err != nil {
			t.Fatalf("Failed to connect to Gmail: %v", err)
		}

		if !provider.IsConnected() {
			t.Error("Provider should be connected")
		}

		provider.Disconnect()
		if provider.IsConnected() {
			t.Error("Provider should be disconnected")
		}
	})

	t.Run("Test Connection", func(t *testing.T) {
		err := provider.TestConnection(ctx, account)
		if err != nil {
			t.Errorf("Test connection failed: %v", err)
		}
	})

	t.Run("Get Folders", func(t *testing.T) {
		err := provider.Connect(ctx, account)
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer provider.Disconnect()

		folders, err := provider.GetFolders(ctx, account)
		if err != nil {
			t.Errorf("Failed to get folders: %v", err)
		}

		if len(folders) == 0 {
			t.Error("Expected at least one folder")
		}

		// Gmail应该有INBOX
		found := false
		for _, folder := range folders {
			if folder.Name == "INBOX" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find INBOX folder")
		}
	})

	t.Run("Sync Emails", func(t *testing.T) {
		err := provider.Connect(ctx, account)
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer provider.Disconnect()

		emails, err := provider.SyncEmails(ctx, account, "INBOX", 0)
		if err != nil {
			t.Errorf("Failed to sync emails: %v", err)
		}

		t.Logf("Synced %d emails from INBOX", len(emails))
	})
}

func TestQQIntegration(t *testing.T) {
	email := os.Getenv("QQ_TEST_EMAIL")
	authCode := os.Getenv("QQ_TEST_AUTH_CODE")

	if email == "" || authCode == "" {
		t.Skip("Skipping QQ integration test: QQ_TEST_EMAIL and QQ_TEST_AUTH_CODE not set")
	}

	config := &config.EmailProviderConfig{
		Name:         "qq",
		DisplayName:  "QQ邮箱",
		IMAPHost:     "imap.qq.com",
		IMAPPort:     993,
		IMAPSecurity: "SSL",
		SMTPHost:     "smtp.qq.com",
		SMTPPort:     465,
		SMTPSecurity: "SSL",
		AuthMethods:  []string{"password"},
		Domains:      []string{"qq.com"},
	}

	provider := NewQQProvider(config)
	account := &models.EmailAccount{
		Email:      email,
		Password:   authCode,
		AuthMethod: "password",
		Provider:   "qq",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Connection Test", func(t *testing.T) {
		err := provider.Connect(ctx, account)
		if err != nil {
			t.Fatalf("Failed to connect to QQ Mail: %v", err)
		}

		if !provider.IsConnected() {
			t.Error("Provider should be connected")
		}

		provider.Disconnect()
	})

	t.Run("Auth Code Validation", func(t *testing.T) {
		// 测试有效的授权码
		validAccount := &models.EmailAccount{
			Email:    email,
			Password: authCode,
		}

		qqProvider := newQQProviderImpl(config)
		err := qqProvider.validateAuthCode(validAccount)
		if err != nil {
			t.Errorf("Valid auth code should not produce error: %v", err)
		}

		// 测试无效的授权码
		invalidAccount := &models.EmailAccount{
			Email:    email,
			Password: "invalid",
		}

		err = qqProvider.validateAuthCode(invalidAccount)
		if err == nil {
			t.Error("Invalid auth code should produce error")
		}
	})
}

func TestNetEaseIntegration(t *testing.T) {
	email := os.Getenv("NETEASE_TEST_EMAIL")
	authCode := os.Getenv("NETEASE_TEST_AUTH_CODE")

	if email == "" || authCode == "" {
		t.Skip("Skipping NetEase integration test: NETEASE_TEST_EMAIL and NETEASE_TEST_AUTH_CODE not set")
	}

	config := &config.EmailProviderConfig{
		Name:         "163",
		DisplayName:  "网易邮箱",
		IMAPHost:     "imap.163.com",
		IMAPPort:     993,
		IMAPSecurity: "SSL",
		SMTPHost:     "smtp.163.com",
		SMTPPort:     465,
		SMTPSecurity: "SSL",
		AuthMethods:  []string{"password"},
		Domains:      []string{"163.com"},
	}

	provider := NewNetEaseProvider(config)
	account := &models.EmailAccount{
		Email:      email,
		Password:   authCode,
		AuthMethod: "password",
		Provider:   "163",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Connection Test", func(t *testing.T) {
		err := provider.Connect(ctx, account)
		if err != nil {
			t.Fatalf("Failed to connect to NetEase Mail: %v", err)
		}

		provider.Disconnect()
	})

	t.Run("Domain Configuration", func(t *testing.T) {
		netEaseProvider := newNetEaseProviderImpl(config)

		// 测试163.com域名
		account163 := &models.EmailAccount{Email: "test@163.com"}
		netEaseProvider.ensureNetEaseConfig(account163)
		if account163.IMAPHost != "imap.163.com" {
			t.Errorf("Expected IMAP host imap.163.com for 163.com, got %s", account163.IMAPHost)
		}

		// 测试126.com域名
		account126 := &models.EmailAccount{Email: "test@126.com"}
		netEaseProvider.ensureNetEaseConfig(account126)
		if account126.IMAPHost != "imap.126.com" {
			t.Errorf("Expected IMAP host imap.126.com for 126.com, got %s", account126.IMAPHost)
		}
	})
}

func TestiCloudIntegration(t *testing.T) {
	email := os.Getenv("ICLOUD_TEST_EMAIL")
	appPassword := os.Getenv("ICLOUD_TEST_APP_PASSWORD")

	if email == "" || appPassword == "" {
		t.Skip("Skipping iCloud integration test: ICLOUD_TEST_EMAIL and ICLOUD_TEST_APP_PASSWORD not set")
	}

	config := &config.EmailProviderConfig{
		Name:         "icloud",
		DisplayName:  "iCloud邮箱",
		IMAPHost:     "imap.mail.me.com",
		IMAPPort:     993,
		IMAPSecurity: "SSL",
		SMTPHost:     "smtp.mail.me.com",
		SMTPPort:     587,
		SMTPSecurity: "STARTTLS",
		AuthMethods:  []string{"password"},
		Domains:      []string{"icloud.com"},
	}

	provider := NewiCloudProvider(config)
	account := &models.EmailAccount{
		Email:      email,
		Password:   appPassword,
		AuthMethod: "password",
		Provider:   "icloud",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Connection Test", func(t *testing.T) {
		err := provider.Connect(ctx, account)
		if err != nil {
			t.Fatalf("Failed to connect to iCloud Mail: %v", err)
		}

		provider.Disconnect()
	})

	t.Run("App Password Validation", func(t *testing.T) {
		iCloudProvider := newiCloudProviderImpl(config)

		// 测试有效的应用专用密码格式
		validAccount := &models.EmailAccount{
			Email:    email,
			Password: appPassword,
		}

		err := iCloudProvider.validateAppSpecificPassword(validAccount)
		if err != nil {
			t.Errorf("Valid app password should not produce error: %v", err)
		}

		// 测试无效格式
		invalidAccount := &models.EmailAccount{
			Email:    email,
			Password: "invalid-format",
		}

		err = iCloudProvider.validateAppSpecificPassword(invalidAccount)
		if err == nil {
			t.Error("Invalid app password format should produce error")
		}
	})
}

func TestProviderFactory(t *testing.T) {
	factory := NewProviderFactory()

	t.Run("Available Providers", func(t *testing.T) {
		providers := factory.GetAvailableProviders()

		expectedProviders := []string{"gmail", "outlook", "qq", "163", "icloud"}

		if len(providers) < len(expectedProviders) {
			t.Errorf("Expected at least %d providers, got %d", len(expectedProviders), len(providers))
		}

		for _, expected := range expectedProviders {
			found := false
			for _, provider := range providers {
				if provider == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected provider %s not found", expected)
			}
		}
	})

	t.Run("Create Providers", func(t *testing.T) {
		testCases := []struct {
			providerName string
			email        string
		}{
			{"gmail", "test@gmail.com"},
			{"outlook", "test@outlook.com"},
			{"qq", "test@qq.com"},
			{"163", "test@163.com"},
			{"icloud", "test@icloud.com"},
		}

		for _, tc := range testCases {
			t.Run(tc.providerName, func(t *testing.T) {
				account := &models.EmailAccount{
					Email:    tc.email,
					Provider: tc.providerName,
				}

				provider, err := factory.CreateProviderForAccount(account)
				if err != nil {
					t.Errorf("Failed to create provider %s: %v", tc.providerName, err)
				}

				if provider == nil {
					t.Errorf("Provider %s should not be nil", tc.providerName)
				}

				info := provider.GetProviderInfo()
				if info == nil {
					t.Errorf("Provider info for %s should not be nil", tc.providerName)
				}
			})
		}
	})

	t.Run("Auto Detect Provider", func(t *testing.T) {
		testCases := []struct {
			email            string
			expectedProvider string
		}{
			{"test@gmail.com", "gmail"},
			{"test@googlemail.com", "gmail"},
			{"test@outlook.com", "outlook"},
			{"test@hotmail.com", "outlook"},
			{"test@qq.com", "qq"},
			{"test@163.com", "163"},
			{"test@126.com", "163"},
			{"test@icloud.com", "icloud"},
			{"test@me.com", "icloud"},
		}

		for _, tc := range testCases {
			t.Run(tc.email, func(t *testing.T) {
				provider := factory.AutoDetectProvider(tc.email)
				if provider != tc.expectedProvider {
					t.Errorf("Expected provider %s for email %s, got %s",
						tc.expectedProvider, tc.email, provider)
				}
			})
		}
	})
}

func TestRetryManagerIntegration(t *testing.T) {
	retryManager := NewRetryManager()

	t.Run("Statistics Tracking", func(t *testing.T) {
		// 重置统计
		retryManager.ResetStatistics()

		// 模拟一些操作
		account := &models.EmailAccount{
			Email:    "test@example.com",
			Provider: "test",
		}

		// 模拟成功的连接操作
		successOp := &ConnectionOperation{
			Account: account,
		}

		ctx := context.Background()
		err := retryManager.ExecuteWithRetry(ctx, successOp)
		// 这会失败，但我们主要测试统计功能

		stats := retryManager.GetStatistics()
		if stats.TotalOperations == 0 {
			t.Error("Expected total operations to be greater than 0")
		}

		if len(stats.OperationStats) == 0 {
			t.Error("Expected operation stats to be recorded")
		}

		if len(stats.ProviderStats) == 0 {
			t.Error("Expected provider stats to be recorded")
		}
	})

	t.Run("Health Status", func(t *testing.T) {
		health := retryManager.GetHealthStatus()

		if health.Overall == "" {
			t.Error("Expected overall health status to be set")
		}

		if health.LastChecked.IsZero() {
			t.Error("Expected last checked time to be set")
		}
	})
}
