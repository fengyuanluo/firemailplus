package services

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// BackupService 备份服务接口
type BackupService interface {
	// 创建备份
	CreateBackup(ctx context.Context) (*BackupInfo, error)
	
	// 列出所有备份
	ListBackups(ctx context.Context) ([]*BackupInfo, error)
	
	// 恢复备份
	RestoreBackup(ctx context.Context, backupPath string) error
	
	// 删除备份
	DeleteBackup(ctx context.Context, backupPath string) error
	
	// 清理过期备份
	CleanupOldBackups(ctx context.Context) error
	
	// 验证备份文件
	ValidateBackup(ctx context.Context, backupPath string) error
	
	// 启动自动备份
	StartAutoBackup(ctx context.Context) error
	
	// 停止自动备份
	StopAutoBackup()
}

// BackupInfo 备份信息
type BackupInfo struct {
	Path      string    `json:"path"`
	Filename  string    `json:"filename"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
	IsValid   bool      `json:"is_valid"`
}

// BackupServiceImpl 备份服务实现
type BackupServiceImpl struct {
	db            *gorm.DB
	dbPath        string
	backupDir     string
	maxBackups    int
	intervalHours int
	stopChan      chan struct{}
}

// NewBackupService 创建备份服务
func NewBackupService(db *gorm.DB, dbPath, backupDir string, maxBackups, intervalHours int) BackupService {
	// 设置默认值
	if maxBackups <= 0 {
		maxBackups = 7
	}
	if intervalHours <= 0 {
		intervalHours = 24
	}

	return &BackupServiceImpl{
		db:            db,
		dbPath:        dbPath,
		backupDir:     backupDir,
		maxBackups:    maxBackups,
		intervalHours: intervalHours,
		stopChan:      make(chan struct{}),
	}
}

// CreateBackup 创建备份
func (s *BackupServiceImpl) CreateBackup(ctx context.Context) (*BackupInfo, error) {
	// 确保备份目录存在
	if err := os.MkdirAll(s.backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// 生成备份文件名
	timestamp := time.Now().Format("20060102_150405")
	backupFilename := fmt.Sprintf("firemail_backup_%s.db", timestamp)
	backupPath := filepath.Join(s.backupDir, backupFilename)

	log.Printf("Creating backup: %s", backupPath)

	// 使用SQLite的VACUUM INTO命令创建备份
	if err := s.db.Exec(fmt.Sprintf("VACUUM INTO '%s'", backupPath)).Error; err != nil {
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}

	// 获取备份文件信息
	fileInfo, err := os.Stat(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup file info: %w", err)
	}

	backupInfo := &BackupInfo{
		Path:      backupPath,
		Filename:  backupFilename,
		Size:      fileInfo.Size(),
		CreatedAt: fileInfo.ModTime(),
		IsValid:   true,
	}

	// 验证备份文件
	if err := s.ValidateBackup(ctx, backupPath); err != nil {
		log.Printf("Warning: backup validation failed: %v", err)
		backupInfo.IsValid = false
	}

	log.Printf("Backup created successfully: %s (size: %d bytes)", backupPath, fileInfo.Size())
	return backupInfo, nil
}

// ListBackups 列出所有备份
func (s *BackupServiceImpl) ListBackups(ctx context.Context) ([]*BackupInfo, error) {
	// 确保备份目录存在
	if _, err := os.Stat(s.backupDir); os.IsNotExist(err) {
		return []*BackupInfo{}, nil
	}

	// 读取备份目录
	files, err := os.ReadDir(s.backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []*BackupInfo
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".db") {
			continue
		}

		filePath := filepath.Join(s.backupDir, file.Name())
		fileInfo, err := file.Info()
		if err != nil {
			log.Printf("Warning: failed to get file info for %s: %v", file.Name(), err)
			continue
		}

		backup := &BackupInfo{
			Path:      filePath,
			Filename:  file.Name(),
			Size:      fileInfo.Size(),
			CreatedAt: fileInfo.ModTime(),
			IsValid:   true,
		}

		// 验证备份文件（异步，不阻塞列表操作）
		go func(path string, info *BackupInfo) {
			if err := s.ValidateBackup(context.Background(), path); err != nil {
				info.IsValid = false
			}
		}(filePath, backup)

		backups = append(backups, backup)
	}

	// 按创建时间排序（最新的在前）
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// RestoreBackup 恢复备份
func (s *BackupServiceImpl) RestoreBackup(ctx context.Context, backupPath string) error {
	// 验证备份文件
	if err := s.ValidateBackup(ctx, backupPath); err != nil {
		return fmt.Errorf("backup validation failed: %w", err)
	}

	log.Printf("Restoring backup from: %s", backupPath)

	// 关闭当前数据库连接
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	// 备份当前数据库文件
	currentBackupPath := s.dbPath + ".restore_backup"
	if err := s.copyFile(s.dbPath, currentBackupPath); err != nil {
		log.Printf("Warning: failed to backup current database: %v", err)
	}

	// 复制备份文件到数据库位置
	if err := s.copyFile(backupPath, s.dbPath); err != nil {
		// 如果恢复失败，尝试恢复原文件
		if restoreErr := s.copyFile(currentBackupPath, s.dbPath); restoreErr != nil {
			log.Printf("Critical: failed to restore original database: %v", restoreErr)
		}
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	// 清理临时备份文件
	os.Remove(currentBackupPath)

	log.Printf("Backup restored successfully from: %s", backupPath)
	return nil
}

// DeleteBackup 删除备份
func (s *BackupServiceImpl) DeleteBackup(ctx context.Context, backupPath string) error {
	// 验证路径在备份目录内
	if !strings.HasPrefix(backupPath, s.backupDir) {
		return fmt.Errorf("backup path is not in backup directory")
	}

	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	log.Printf("Backup deleted: %s", backupPath)
	return nil
}

// CleanupOldBackups 清理过期备份
func (s *BackupServiceImpl) CleanupOldBackups(ctx context.Context) error {
	backups, err := s.ListBackups(ctx)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	// 如果备份数量超过限制，删除最旧的备份
	if len(backups) > s.maxBackups {
		toDelete := backups[s.maxBackups:]
		for _, backup := range toDelete {
			if err := s.DeleteBackup(ctx, backup.Path); err != nil {
				log.Printf("Warning: failed to delete old backup %s: %v", backup.Path, err)
			}
		}
		log.Printf("Cleaned up %d old backups", len(toDelete))
	}

	return nil
}

// ValidateBackup 验证备份文件
func (s *BackupServiceImpl) ValidateBackup(ctx context.Context, backupPath string) error {
	// 检查文件是否存在
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	// 尝试打开SQLite数据库文件
	testDB, err := gorm.Open(sqlite.Open(backupPath), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to open backup database: %w", err)
	}

	// 关闭测试连接
	sqlDB, err := testDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}
	defer sqlDB.Close()

	// 执行简单查询验证数据库完整性
	if err := testDB.Exec("PRAGMA integrity_check").Error; err != nil {
		return fmt.Errorf("backup integrity check failed: %w", err)
	}

	return nil
}

// StartAutoBackup 启动自动备份
func (s *BackupServiceImpl) StartAutoBackup(ctx context.Context) error {
	log.Printf("Starting automatic backup service (interval: %d hours, max backups: %d)...", s.intervalHours, s.maxBackups)

	go func() {
		// 立即执行一次备份
		if _, err := s.CreateBackup(ctx); err != nil {
			log.Printf("Initial backup failed: %v", err)
		}

		// 使用配置的时间间隔执行备份
		interval := time.Duration(s.intervalHours) * time.Hour
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				log.Println("Running scheduled backup...")
				if _, err := s.CreateBackup(ctx); err != nil {
					log.Printf("Scheduled backup failed: %v", err)
				} else {
					// 清理过期备份
					if err := s.CleanupOldBackups(ctx); err != nil {
						log.Printf("Failed to cleanup old backups: %v", err)
					}
				}
			case <-s.stopChan:
				log.Println("Stopping automatic backup service...")
				return
			case <-ctx.Done():
				log.Println("Context cancelled, stopping automatic backup service...")
				return
			}
		}
	}()

	return nil
}

// StopAutoBackup 停止自动备份
func (s *BackupServiceImpl) StopAutoBackup() {
	close(s.stopChan)
}

// copyFile 复制文件
func (s *BackupServiceImpl) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
