package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 应用配置结构
type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Auth     AuthConfig     `json:"auth"`
	OAuth    OAuthConfig    `json:"oauth"`
	CORS     CORSConfig     `json:"cors"`
	Logging  LoggingConfig  `json:"logging"`
	SSE      SSEConfig      `json:"sse"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `json:"host"`
	Port string `json:"port"`
	Env  string `json:"env"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Path               string `json:"path"`
	BackupDir          string `json:"backup_dir"`
	BackupMaxCount     int    `json:"backup_max_count"`
	BackupIntervalHours int   `json:"backup_interval_hours"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	AdminUsername string        `json:"admin_username"`
	AdminPassword string        `json:"admin_password"`
	JWTSecret     string        `json:"jwt_secret"`
	JWTExpiry     time.Duration `json:"jwt_expiry"`
}

// OAuthConfig OAuth2配置
type OAuthConfig struct {
	Gmail           OAuthProviderConfig `json:"gmail"`
	Outlook         OAuthProviderConfig `json:"outlook"`
	ExternalServer  ExternalOAuthConfig `json:"external_server"`
}

// ExternalOAuthConfig 外部OAuth服务器配置
type ExternalOAuthConfig struct {
	BaseURL string `json:"base_url"`
	Enabled bool   `json:"enabled"`
}

// OAuthProviderConfig OAuth2提供商配置
type OAuthProviderConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURL  string `json:"redirect_url"`
}

// CORSConfig CORS配置
type CORSConfig struct {
	Origins []string `json:"origins"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"`
}

// SSEConfig SSE配置
type SSEConfig struct {
	MaxConnectionsPerUser int           `json:"max_connections_per_user"`
	ConnectionTimeout     time.Duration `json:"connection_timeout"`
	HeartbeatInterval     time.Duration `json:"heartbeat_interval"`
	CleanupInterval       time.Duration `json:"cleanup_interval"`
	BufferSize            int           `json:"buffer_size"`
	EnableHeartbeat       bool          `json:"enable_heartbeat"`
}



// Load 加载配置
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host: getEnv("HOST", "localhost"),
			Port: getEnv("PORT", "8080"),
			Env:  getEnv("ENV", "development"),
		},
		Database: DatabaseConfig{
			Path:                getEnv("DB_PATH", "./firemail.db"),
			BackupDir:           getEnv("DB_BACKUP_DIR", "./backups"),
			BackupMaxCount:      parseInt(getEnv("DB_BACKUP_MAX_COUNT", "7"), 7),
			BackupIntervalHours: parseInt(getEnv("DB_BACKUP_INTERVAL_HOURS", "24"), 24),
		},
		Auth: AuthConfig{
			AdminUsername: getEnv("ADMIN_USERNAME", "admin"),
			AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
			JWTSecret:     getEnv("JWT_SECRET", "your-secret-key"),
			JWTExpiry:     parseDuration(getEnv("JWT_EXPIRY", "24h")),
		},
		OAuth: OAuthConfig{
			Gmail: OAuthProviderConfig{
				ClientID:     getEnv("GMAIL_CLIENT_ID", ""),
				ClientSecret: getEnv("GMAIL_CLIENT_SECRET", ""),
				RedirectURL:  getEnv("GMAIL_REDIRECT_URL", ""), // 已废弃：仅使用外部OAuth服务器
			},
			Outlook: OAuthProviderConfig{
				ClientID:     getEnv("OUTLOOK_CLIENT_ID", ""),
				ClientSecret: getEnv("OUTLOOK_CLIENT_SECRET", ""),
				RedirectURL:  getEnv("OUTLOOK_REDIRECT_URL", ""), // 已废弃：仅使用外部OAuth服务器
			},
			ExternalServer: ExternalOAuthConfig{
				BaseURL: getEnv("EXTERNAL_OAUTH_SERVER_URL", "http://localhost:8080"),
				Enabled: parseBool(getEnv("EXTERNAL_OAUTH_SERVER_ENABLED", "true")),
			},
		},
		CORS: CORSConfig{
			Origins: parseStringSlice(getEnv("CORS_ORIGINS", "http://localhost:3000,http://localhost:8080")),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		SSE: SSEConfig{
			MaxConnectionsPerUser: parseInt(getEnv("SSE_MAX_CONNECTIONS_PER_USER", "5"), 5),
			ConnectionTimeout:     parseDuration(getEnv("SSE_CONNECTION_TIMEOUT", "30m")),
			HeartbeatInterval:     parseDuration(getEnv("SSE_HEARTBEAT_INTERVAL", "30s")),
			CleanupInterval:       parseDuration(getEnv("SSE_CLEANUP_INTERVAL", "5m")),
			BufferSize:            parseInt(getEnv("SSE_BUFFER_SIZE", "1024"), 1024),
			EnableHeartbeat:       parseBool(getEnv("SSE_ENABLE_HEARTBEAT", "true")),
		},
	}
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// parseDuration 解析时间间隔
func parseDuration(s string) time.Duration {
	duration, err := time.ParseDuration(s)
	if err != nil {
		return 24 * time.Hour // 默认24小时
	}
	return duration
}

// parseStringSlice 解析字符串切片
func parseStringSlice(s string) []string {
	if s == "" {
		return []string{}
	}
	return strings.Split(s, ",")
}

// parseBool 解析布尔值
func parseBool(s string) bool {
	b, err := strconv.ParseBool(s)
	if err != nil {
		return false
	}
	return b
}

// parseInt 解析整数
func parseInt(s string, defaultValue int) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}
	return i
}
