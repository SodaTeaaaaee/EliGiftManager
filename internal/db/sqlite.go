package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// InitDB 初始化 SQLite 数据库连接，并确保数据库目录已存在。
func InitDB(dbPath string) (*gorm.DB, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("初始化 SQLite 数据库失败: 数据库路径不能为空")
	}

	cleanedPath := filepath.Clean(dbPath)

	if err := ensureDatabaseDir(cleanedPath); err != nil {
		return nil, fmt.Errorf("初始化 SQLite 数据库失败: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(cleanedPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("初始化 SQLite 数据库失败: 打开数据库文件 %q 失败: %w", cleanedPath, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("初始化 SQLite 数据库失败: 获取底层数据库连接失败: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("初始化 SQLite 数据库失败: 连接数据库失败: %w", err)
	}

	if err := autoMigrateTables(db); err != nil {
		return nil, fmt.Errorf("初始化 SQLite 数据库失败: %w", err)
	}

	return db, nil
}

func ensureDatabaseDir(dbPath string) error {
	dir := filepath.Dir(dbPath)
	if dir == "." || dir == "" {
		return nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建数据库目录 %q 失败: %w", dir, err)
	}

	return nil
}

func autoMigrateTables(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.Member{},
		&model.MemberNickname{},
		&model.MemberAddress{},
		&model.Product{},
		&model.DispatchRecord{},
		&model.TemplateConfig{},
	); err != nil {
		return fmt.Errorf("自动迁移数据库表失败: %w", err)
	}

	return nil
}
