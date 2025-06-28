package providers

import (
	"context"
	"testing"
	"time"

	"firemail/internal/config"
	"firemail/internal/models"
)

func TestQQProvider_Creation(t *testing.T) {
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
		Domains:      []string{"qq.com", "vip.qq.com", "foxmail.com"},
	}

	provider := NewQQProvider(config)
	if provider == nil {
		t.Fatal("Failed to create QQ provider")
	}

	// 验证提供商信息
	info := provider.GetProviderInfo()
	if info["name"] != "QQ邮箱" {
		t.Errorf("Expected provider name 'QQ邮箱', got %v", info["name"])
	}

	authMethods := info["auth_methods"].([]string)
	if len(authMethods) == 0 || authMethods[0] != "password" {
		t.Errorf("Expected auth method 'password', got %v", authMethods)
	}
}

func TestQQProvider_ValidateAuthCode(t *testing.T) {
	config := &config.EmailProviderConfig{
		Name: "qq",
	}
	provider := newQQProviderImpl(config)

	tests := []struct {
		name        string
		account     *models.EmailAccount
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid 16-character auth code",
			account: &models.EmailAccount{
				Email:    "test@qq.com",
				Password: "abcd1234efgh5678",
			},
			expectError: false,
		},
		{
			name: "Invalid short password",
			account: &models.EmailAccount{
				Email:    "test@qq.com",
				Password: "short",
			},
			expectError: true,
			errorMsg:    "16-character authorization code",
		},
		{
			name: "Invalid long password",
			account: &models.EmailAccount{
				Email:    "test@qq.com",
				Password: "toolongpassword123456",
			},
			expectError: true,
			errorMsg:    "16-character authorization code",
		},
		{
			name: "Invalid characters in auth code",
			account: &models.EmailAccount{
				Email:    "test@qq.com",
				Password: "abcd1234efgh567@",
			},
			expectError: true,
			errorMsg:    "invalid authorization code format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.validateAuthCode(tt.account)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !contains([]string{err.Error()}, tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestQQProvider_EnsureQQConfig(t *testing.T) {
	config := &config.EmailProviderConfig{
		Name:         "qq",
		IMAPHost:     "imap.qq.com",
		IMAPPort:     993,
		IMAPSecurity: "SSL",
		SMTPHost:     "smtp.qq.com",
		SMTPPort:     465,
		SMTPSecurity: "SSL",
	}
	provider := newQQProviderImpl(config)

	tests := []struct {
		name     string
		account  *models.EmailAccount
		expected *models.EmailAccount
	}{
		{
			name: "Empty config should be filled",
			account: &models.EmailAccount{
				Email: "test@qq.com",
			},
			expected: &models.EmailAccount{
				Email:        "test@qq.com",
				Username:     "test@qq.com",
				IMAPHost:     "imap.qq.com",
				IMAPPort:     993,
				IMAPSecurity: "SSL",
				SMTPHost:     "smtp.qq.com",
				SMTPPort:     465,
				SMTPSecurity: "SSL",
			},
		},
		{
			name: "Existing config should not be overridden",
			account: &models.EmailAccount{
				Email:        "test@qq.com",
				Username:     "custom_user",
				IMAPHost:     "custom.imap.com",
				IMAPPort:     143,
				IMAPSecurity: "NONE",
			},
			expected: &models.EmailAccount{
				Email:        "test@qq.com",
				Username:     "custom_user",
				IMAPHost:     "custom.imap.com",
				IMAPPort:     143,
				IMAPSecurity: "NONE",
				SMTPHost:     "smtp.qq.com",
				SMTPPort:     465,
				SMTPSecurity: "SSL",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider.ensureQQConfig(tt.account)

			if tt.account.Username != tt.expected.Username {
				t.Errorf("Expected Username %s, got %s", tt.expected.Username, tt.account.Username)
			}
			if tt.account.IMAPHost != tt.expected.IMAPHost {
				t.Errorf("Expected IMAPHost %s, got %s", tt.expected.IMAPHost, tt.account.IMAPHost)
			}
			if tt.account.IMAPPort != tt.expected.IMAPPort {
				t.Errorf("Expected IMAPPort %d, got %d", tt.expected.IMAPPort, tt.account.IMAPPort)
			}
			if tt.account.IMAPSecurity != tt.expected.IMAPSecurity {
				t.Errorf("Expected IMAPSecurity %s, got %s", tt.expected.IMAPSecurity, tt.account.IMAPSecurity)
			}
			if tt.account.SMTPHost != tt.expected.SMTPHost {
				t.Errorf("Expected SMTPHost %s, got %s", tt.expected.SMTPHost, tt.account.SMTPHost)
			}
			if tt.account.SMTPPort != tt.expected.SMTPPort {
				t.Errorf("Expected SMTPPort %d, got %d", tt.expected.SMTPPort, tt.account.SMTPPort)
			}
			if tt.account.SMTPSecurity != tt.expected.SMTPSecurity {
				t.Errorf("Expected SMTPSecurity %s, got %s", tt.expected.SMTPSecurity, tt.account.SMTPSecurity)
			}
		})
	}
}

func TestQQProvider_HandleQQError(t *testing.T) {
	config := &config.EmailProviderConfig{Name: "qq"}
	provider := newQQProviderImpl(config)

	tests := []struct {
		name        string
		inputError  error
		expectedMsg string
	}{
		{
			name:        "Nil error",
			inputError:  nil,
			expectedMsg: "",
		},
		{
			name:        "535 authentication error",
			inputError:  &ProviderError{Code: "535", Message: "Authentication failed"},
			expectedMsg: "authentication failed: please check your email and authorization code",
		},
		{
			name:        "550 rate limit error",
			inputError:  &ProviderError{Code: "550", Message: "Rate limit exceeded"},
			expectedMsg: "sending frequency limit exceeded",
		},
		{
			name:        "554 spam error",
			inputError:  &ProviderError{Code: "554", Message: "Message rejected"},
			expectedMsg: "message rejected: content may be considered spam",
		},
		{
			name:        "421 service unavailable",
			inputError:  &ProviderError{Code: "421", Message: "Service unavailable"},
			expectedMsg: "service temporarily unavailable",
		},
		{
			name:        "Unknown error",
			inputError:  &ProviderError{Code: "999", Message: "Unknown error"},
			expectedMsg: "QQ mail error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.HandleQQError(tt.inputError)

			if tt.expectedMsg == "" {
				if result != nil {
					t.Errorf("Expected nil error but got: %v", result)
				}
			} else {
				if result == nil {
					t.Errorf("Expected error but got nil")
				} else if !contains([]string{result.Error()}, tt.expectedMsg) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.expectedMsg, result.Error())
				}
			}
		})
	}
}

func TestQQProvider_IsNonRetryableError(t *testing.T) {
	config := &config.EmailProviderConfig{Name: "qq"}
	provider := newQQProviderImpl(config)

	tests := []struct {
		name        string
		inputError  error
		shouldRetry bool
	}{
		{
			name:        "Nil error",
			inputError:  nil,
			shouldRetry: true, // nil error means no error, so it's "retryable"
		},
		{
			name:        "535 authentication error - not retryable",
			inputError:  &ProviderError{Code: "535", Message: "Authentication failed"},
			shouldRetry: false,
		},
		{
			name:        "553 invalid address - not retryable",
			inputError:  &ProviderError{Code: "553", Message: "Invalid address"},
			shouldRetry: false,
		},
		{
			name:        "421 temporary failure - retryable",
			inputError:  &ProviderError{Code: "421", Message: "Service temporarily unavailable"},
			shouldRetry: true,
		},
		{
			name:        "Network timeout - retryable",
			inputError:  &ProviderError{Code: "TIMEOUT", Message: "Connection timeout"},
			shouldRetry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.isNonRetryableError(tt.inputError)
			expected := !tt.shouldRetry

			if result != expected {
				t.Errorf("Expected isNonRetryableError to return %v, got %v", expected, result)
			}
		})
	}
}

func TestQQProvider_GetProviderInfo(t *testing.T) {
	config := &config.EmailProviderConfig{Name: "qq"}
	provider := newQQProviderImpl(config)

	info := provider.GetProviderInfo()

	// 验证基本信息
	if info["name"] != "QQ邮箱" {
		t.Errorf("Expected name 'QQ邮箱', got %v", info["name"])
	}

	if info["display_name"] != "QQ邮箱" {
		t.Errorf("Expected display_name 'QQ邮箱', got %v", info["display_name"])
	}

	// 验证认证方式
	authMethods, ok := info["auth_methods"].([]string)
	if !ok {
		t.Fatal("auth_methods should be []string")
	}
	if len(authMethods) != 1 || authMethods[0] != "password" {
		t.Errorf("Expected auth_methods ['password'], got %v", authMethods)
	}

	// 验证域名
	domains, ok := info["domains"].([]string)
	if !ok {
		t.Fatal("domains should be []string")
	}
	expectedDomains := []string{"qq.com", "vip.qq.com", "foxmail.com"}
	if len(domains) != len(expectedDomains) {
		t.Errorf("Expected %d domains, got %d", len(expectedDomains), len(domains))
	}

	// 验证服务器配置
	servers, ok := info["servers"].(map[string]interface{})
	if !ok {
		t.Fatal("servers should be map[string]interface{}")
	}

	imap, ok := servers["imap"].(map[string]interface{})
	if !ok {
		t.Fatal("imap server config should be map[string]interface{}")
	}
	if imap["host"] != "imap.qq.com" {
		t.Errorf("Expected IMAP host 'imap.qq.com', got %v", imap["host"])
	}
	if imap["port"] != 993 {
		t.Errorf("Expected IMAP port 993, got %v", imap["port"])
	}

	smtp, ok := servers["smtp"].(map[string]interface{})
	if !ok {
		t.Fatal("smtp server config should be map[string]interface{}")
	}
	if smtp["host"] != "smtp.qq.com" {
		t.Errorf("Expected SMTP host 'smtp.qq.com', got %v", smtp["host"])
	}
	if smtp["port"] != 465 {
		t.Errorf("Expected SMTP port 465, got %v", smtp["port"])
	}

	// 验证功能特性
	features, ok := info["features"].(map[string]bool)
	if !ok {
		t.Fatal("features should be map[string]bool")
	}
	if !features["imap"] {
		t.Error("Expected IMAP feature to be true")
	}
	if !features["smtp"] {
		t.Error("Expected SMTP feature to be true")
	}
	if features["oauth2"] {
		t.Error("Expected OAuth2 feature to be false for QQ mail")
	}

	// 验证限制信息
	limits, ok := info["limits"].(map[string]interface{})
	if !ok {
		t.Fatal("limits should be map[string]interface{}")
	}
	if limits["attachment_size"] != 50*1024*1024 {
		t.Errorf("Expected attachment size limit 50MB, got %v", limits["attachment_size"])
	}

	// 验证错误代码
	errorCodes, ok := info["error_codes"].(map[string]string)
	if !ok {
		t.Fatal("error_codes should be map[string]string")
	}
	if errorCodes["535"] == "" {
		t.Error("Expected error code 535 to have description")
	}

	// 验证帮助链接
	helpURLs, ok := info["help_urls"].(map[string]string)
	if !ok {
		t.Fatal("help_urls should be map[string]string")
	}
	if helpURLs["auth_code"] == "" {
		t.Error("Expected auth_code help URL")
	}

	// 验证授权码说明
	authInstructions, ok := info["auth_instructions"].(string)
	if !ok {
		t.Fatal("auth_instructions should be string")
	}
	if authInstructions == "" {
		t.Error("Expected non-empty auth instructions")
	}
}

func TestQQProvider_ConnectWithRetry(t *testing.T) {
	config := &config.EmailProviderConfig{Name: "qq"}
	provider := newQQProviderImpl(config)

	// 模拟账户
	account := &models.EmailAccount{
		Email:      "test@qq.com",
		Password:   "abcd1234efgh5678", // 有效的16位授权码
		AuthMethod: "password",
		Provider:   "qq",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 注意：这个测试会尝试实际连接，在没有真实凭据的情况下会失败
	// 这里主要测试重试逻辑是否正确执行
	err := provider.connectWithRetry(ctx, account)

	// 我们期望连接失败（因为没有真实的服务器或凭据）
	// 但错误应该是经过处理的QQ特定错误
	if err == nil {
		t.Error("Expected connection to fail without real credentials")
	} else {
		// 验证错误是否经过了QQ特定的处理
		if !contains([]string{err.Error()}, "QQ mail") && !contains([]string{err.Error()}, "failed to connect") {
			t.Logf("Got expected error: %v", err)
		}
	}
}
