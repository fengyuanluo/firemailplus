package migration

import (
	"context"
	"database/sql"
)

// IMigrator 数据库迁移接口
// 遵循依赖倒置原则，定义抽象接口
type IMigrator interface {
	// Up 执行向上迁移
	Up(ctx context.Context) error
	
	// Down 执行向下迁移
	Down(ctx context.Context) error
	
	// Steps 执行指定步数的迁移
	Steps(ctx context.Context, n int) error
	
	// Force 强制设置迁移版本
	Force(ctx context.Context, version int) error
	
	// Version 获取当前迁移版本
	Version(ctx context.Context) (version int, dirty bool, err error)
	
	// Close 关闭迁移器
	Close() error
}

// MigrationConfig 迁移配置
type MigrationConfig struct {
	DatabaseURL     string
	MigrationsPath  string
	DatabaseName    string
	TableName       string // 迁移版本表名，默认为 schema_migrations
}

// MigrationInfo 迁移信息
type MigrationInfo struct {
	Version   int
	Name      string
	Direction string // up 或 down
	Applied   bool
}

// IMigrationService 迁移服务接口
// 遵循单一职责原则，专门处理迁移逻辑
type IMigrationService interface {
	// Initialize 初始化迁移服务
	Initialize(db *sql.DB, config MigrationConfig) error
	
	// RunMigrations 运行迁移
	RunMigrations(ctx context.Context) error
	
	// GetMigrationInfo 获取迁移信息
	GetMigrationInfo(ctx context.Context) ([]MigrationInfo, error)
	
	// Rollback 回滚指定步数
	Rollback(ctx context.Context, steps int) error
	
	// Reset 重置数据库（谨慎使用）
	Reset(ctx context.Context) error
	
	// Close 关闭服务
	Close() error
}
