package migration

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// GolangMigrator golang-migrate的实现
// 遵循单一职责原则，专门处理golang-migrate相关逻辑
type GolangMigrator struct {
	migrate *migrate.Migrate
	db      *sql.DB
	config  MigrationConfig
}

// NewGolangMigrator 创建新的golang-migrate迁移器
func NewGolangMigrator(db *sql.DB, config MigrationConfig) (*GolangMigrator, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection cannot be nil")
	}
	
	if config.MigrationsPath == "" {
		return nil, fmt.Errorf("migrations path cannot be empty")
	}
	
	// 创建SQLite驱动实例
	// 注意：不要让migrate关闭数据库连接
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{
		MigrationsTable: config.TableName,
		NoTxWrap:        false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create sqlite3 driver: %w", err)
	}
	
	// 构建文件路径URL
	sourceURL := fmt.Sprintf("file://%s", filepath.ToSlash(config.MigrationsPath))
	
	// 创建migrate实例
	m, err := migrate.NewWithDatabaseInstance(sourceURL, config.DatabaseName, driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}
	
	return &GolangMigrator{
		migrate: m,
		db:      db,
		config:  config,
	}, nil
}

// Up 执行向上迁移
func (g *GolangMigrator) Up(ctx context.Context) error {
	if err := g.migrate.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run up migrations: %w", err)
	}
	return nil
}

// Down 执行向下迁移
func (g *GolangMigrator) Down(ctx context.Context) error {
	if err := g.migrate.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run down migrations: %w", err)
	}
	return nil
}

// Steps 执行指定步数的迁移
func (g *GolangMigrator) Steps(ctx context.Context, n int) error {
	if err := g.migrate.Steps(n); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run %d migration steps: %w", n, err)
	}
	return nil
}

// Force 强制设置迁移版本
func (g *GolangMigrator) Force(ctx context.Context, version int) error {
	if err := g.migrate.Force(version); err != nil {
		return fmt.Errorf("failed to force version %d: %w", version, err)
	}
	return nil
}

// Version 获取当前迁移版本
func (g *GolangMigrator) Version(ctx context.Context) (version int, dirty bool, err error) {
	v, dirty, err := g.migrate.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}
	return int(v), dirty, nil
}

// Close 关闭迁移器
// 注意：只关闭migrate实例，不关闭底层数据库连接
func (g *GolangMigrator) Close() error {
	sourceErr, _ := g.migrate.Close()
	if sourceErr != nil {
		return fmt.Errorf("failed to close migration source: %w", sourceErr)
	}
	// 不关闭数据库连接，因为它可能还在被其他地方使用
	return nil
}
