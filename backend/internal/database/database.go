package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"firemail/internal/database/migration"
	"firemail/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	// 导入SQLite驱动
	_ "github.com/mattn/go-sqlite3"
	_ "modernc.org/sqlite"
)

// Initialize 初始化数据库连接
func Initialize(dbPath string) (*gorm.DB, error) {
	return InitializeWithDriver(dbPath, false)
}

// InitializeWithDriver 使用指定驱动初始化数据库连接
func InitializeWithDriver(dbPath string, usePureGo bool) (*gorm.DB, error) {
	// 确保数据库目录存在
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// 配置GORM日志
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			LogLevel: logger.Info,
			Colorful: true,
		},
	)

	// 打开数据库连接
	var db *gorm.DB
	var err error

	if usePureGo {
		// 使用纯Go SQLite驱动（用于测试）
		db, err = gorm.Open(sqlite.Dialector{
			DriverName: "sqlite",
			DSN:        dbPath,
		}, &gorm.Config{
			Logger: gormLogger,
		})
	} else {
		// 使用标准CGO SQLite驱动
		db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			Logger: gormLogger,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 配置SQLite
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// 优化连接池参数
	if err := optimizeConnectionPool(sqlDB); err != nil {
		return nil, fmt.Errorf("failed to optimize connection pool: %w", err)
	}

	// 应用SQLite性能优化
	if err := applySQLiteOptimizations(db); err != nil {
		return nil, fmt.Errorf("failed to apply SQLite optimizations: %w", err)
	}

	// 执行数据库迁移
	if err := runMigrations(dbPath); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// 迁移完成后，重新创建GORM连接，因为迁移可能会影响连接状态
	if oldDB, err := db.DB(); err == nil {
		oldDB.Close() // 关闭可能被影响的连接
	}

	// 重新打开数据库连接
	if usePureGo {
		db, err = gorm.Open(sqlite.Dialector{
			DriverName: "sqlite",
			DSN:        dbPath,
		}, &gorm.Config{
			Logger: gormLogger,
		})
	} else {
		db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			Logger: gormLogger,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to reconnect to database after migration: %w", err)
	}

	// 重新配置数据库连接
	sqlDB, err = db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB after migration: %w", err)
	}

	// 重新优化连接池参数
	if err := optimizeConnectionPool(sqlDB); err != nil {
		return nil, fmt.Errorf("failed to re-optimize connection pool: %w", err)
	}

	// 重新应用SQLite性能优化
	if err := applySQLiteOptimizations(db); err != nil {
		return nil, fmt.Errorf("failed to re-apply SQLite optimizations: %w", err)
	}

	// 创建默认管理员用户
	if err := createDefaultAdmin(db); err != nil {
		return nil, fmt.Errorf("failed to create default admin: %w", err)
	}

	// 同步管理员密码与环境变量
	if err := syncAdminPassword(db); err != nil {
		return nil, fmt.Errorf("failed to sync admin password: %w", err)
	}

	log.Println("Database initialized successfully")
	return db, nil
}

// runMigrations 执行数据库迁移
// 使用golang-migrate进行版本化迁移，遵循最佳实践
// 为迁移创建单独的数据库连接，避免连接被关闭的问题
func runMigrations(dbPath string) error {

	// 为迁移创建单独的数据库连接
	migrationDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open migration database connection: %w", err)
	}
	defer migrationDB.Close()

	// 创建迁移服务
	migrationService := migration.NewMigrationService(nil)

	// 配置迁移参数
	config := migration.MigrationConfig{
		DatabaseURL:    "", // SQLite不需要URL
		MigrationsPath: "database/migrations",
		DatabaseName:   "sqlite3",
		TableName:      "schema_migrations",
	}

	// 初始化迁移服务
	if err := migrationService.Initialize(migrationDB, config); err != nil {
		return fmt.Errorf("failed to initialize migration service: %w", err)
	}
	defer migrationService.Close()

	// 运行迁移
	ctx := context.Background()
	if err := migrationService.RunMigrations(ctx); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Database migration completed successfully")
	return nil
}

// 注意：索引和约束创建逻辑已移至迁移文件中
// 这样可以确保数据库schema的版本化管理和可回滚性

// createDefaultAdmin 创建默认管理员用户
func createDefaultAdmin(db *gorm.DB) error {
	// 检查是否已存在管理员用户
	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		return err
	}

	// 如果已有用户，跳过创建
	if count > 0 {
		return nil
	}

	// 从环境变量获取管理员账号信息
	adminUsername := os.Getenv("ADMIN_USERNAME")
	adminPassword := os.Getenv("ADMIN_PASSWORD")

	if adminUsername == "" {
		adminUsername = "admin"
	}
	if adminPassword == "" {
		adminPassword = "admin123"
	}

	// 创建管理员用户
	admin := &models.User{
		Username:    adminUsername,
		Password:    adminPassword, // 会在BeforeCreate钩子中自动加密
		DisplayName: "Administrator",
		Role:        "admin",
		IsActive:    true,
	}

	if err := db.Create(admin).Error; err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	log.Printf("Default admin user created: %s", adminUsername)
	return nil
}

// syncAdminPassword 同步管理员密码与环境变量
func syncAdminPassword(db *gorm.DB) error {
	// 从环境变量获取管理员账号信息
	adminUsername := os.Getenv("ADMIN_USERNAME")
	adminPassword := os.Getenv("ADMIN_PASSWORD")

	if adminUsername == "" {
		adminUsername = "admin"
	}
	if adminPassword == "" {
		adminPassword = "admin123"
	}

	// 查找管理员用户
	var admin models.User
	if err := db.Where("username = ?", adminUsername).First(&admin).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 如果找不到指定用户名的管理员，可能是用户名变更了
			// 尝试查找第一个admin角色的用户
			if err := db.Where("role = ?", "admin").First(&admin).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					log.Printf("No admin user found to sync password")
					return nil // 没有管理员用户，跳过同步
				}
				return fmt.Errorf("failed to find admin user: %w", err)
			}

			// 更新用户名
			admin.Username = adminUsername
			log.Printf("Admin username updated from '%s' to '%s'", admin.Username, adminUsername)
		} else {
			return fmt.Errorf("failed to query admin user: %w", err)
		}
	}

	// 检查密码是否需要更新
	if !admin.CheckPassword(adminPassword) {
		// 密码不匹配，需要更新
		if err := admin.SetPassword(adminPassword); err != nil {
			return fmt.Errorf("failed to set new password: %w", err)
		}

		// 确保用户处于激活状态
		admin.IsActive = true

		// 保存更新
		if err := db.Save(&admin).Error; err != nil {
			return fmt.Errorf("failed to update admin user: %w", err)
		}

		log.Printf("Admin password synchronized with environment variables for user: %s", admin.Username)
	} else {
		// 密码匹配，但仍需确保用户名和激活状态正确
		needUpdate := false

		if admin.Username != adminUsername {
			admin.Username = adminUsername
			needUpdate = true
			log.Printf("Admin username updated to: %s", adminUsername)
		}

		if !admin.IsActive {
			admin.IsActive = true
			needUpdate = true
			log.Printf("Admin user activated: %s", admin.Username)
		}

		if needUpdate {
			// 只更新需要的字段，避免触发器递归
			updates := make(map[string]interface{})
			if admin.DisplayName != "Administrator" {
				updates["display_name"] = "Administrator"
			}
			if admin.Role != "admin" {
				updates["role"] = "admin"
			}
			if !admin.IsActive {
				updates["is_active"] = true
			}

			if len(updates) > 0 {
				if err := db.Model(&admin).Updates(updates).Error; err != nil {
					return fmt.Errorf("failed to update admin user info: %w", err)
				}
			}
		}

		log.Printf("Admin user is already synchronized: %s", admin.Username)
	}

	return nil
}

// optimizeConnectionPool 优化连接池配置
func optimizeConnectionPool(sqlDB *sql.DB) error {
	// SQLite在WAL模式下支持并发读取，但写入仍然是串行的
	// 对于个人项目，适度增加连接数以支持并发读取
	sqlDB.SetMaxOpenConns(5)    // 允许最多5个并发连接
	sqlDB.SetMaxIdleConns(2)    // 保持2个空闲连接
	sqlDB.SetConnMaxLifetime(time.Hour) // 连接最大生命周期1小时
	sqlDB.SetConnMaxIdleTime(15 * time.Minute) // 空闲连接最大时间15分钟

	return nil
}

// applySQLiteOptimizations 应用SQLite性能优化
func applySQLiteOptimizations(db *gorm.DB) error {
	optimizations := []string{
		// 启用外键约束
		"PRAGMA foreign_keys = ON",

		// 启用WAL模式以提高并发性能
		"PRAGMA journal_mode = WAL",

		// 设置同步模式为NORMAL，平衡性能和安全性
		"PRAGMA synchronous = NORMAL",

		// 增加缓存大小到64MB
		"PRAGMA cache_size = -65536",

		// 设置临时存储为内存
		"PRAGMA temp_store = MEMORY",

		// 启用内存映射I/O，提高读取性能
		"PRAGMA mmap_size = 268435456", // 256MB

		// 优化页面大小
		"PRAGMA page_size = 4096",

		// 启用查询优化器
		"PRAGMA optimize",

		// 设置WAL自动检查点
		"PRAGMA wal_autocheckpoint = 1000",

		// 禁用递归触发器，避免触发器递归问题
		"PRAGMA recursive_triggers = OFF",
	}

	for _, pragma := range optimizations {
		if err := db.Exec(pragma).Error; err != nil {
			log.Printf("Warning: failed to execute %s: %v", pragma, err)
			// 对于个人项目，优化失败不应该阻止启动
		}
	}

	log.Println("SQLite performance optimizations applied")
	return nil
}

// BatchTransaction 批量事务处理
func BatchTransaction(db *gorm.DB, batchSize int, fn func(*gorm.DB, int) error) error {
	return db.Transaction(func(tx *gorm.DB) error {
		return fn(tx, batchSize)
	})
}

// Close 关闭数据库连接
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
