package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/config"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
	"gorm.io/gorm"
)

// App 表示 Wails 生命周期与桌面应用配置的组合入口。
type App struct {
	ctx context.Context
	cfg config.App
}

// BootstrapPayload 表示前端启动阶段需要的基础元数据。
type BootstrapPayload struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Module      string   `json:"module"`
	Description string   `json:"description"`
	Runtime     string   `json:"runtime"`
	Frontend    string   `json:"frontend"`
	Highlights  []string `json:"highlights"`
}

// NewApp 创建应用实例，并注入静态配置。
func NewApp(cfg config.App) *App {
	return &App{cfg: cfg}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Bootstrap 返回前端初始界面所需的基础元数据。
func (a *App) Bootstrap() BootstrapPayload {
	return BootstrapPayload{
		Name:        a.cfg.Name,
		Version:     a.cfg.Version,
		Module:      a.cfg.Module,
		Description: a.cfg.Description,
		Runtime:     runtime.Version(),
		Frontend:    a.cfg.FrontendRuntime,
		Highlights: []string{
			"Go backend uses internal packages for app configuration.",
			"Vue 3 single-file components are compiled through Vite.",
			"Deno installs npm dependencies and runs frontend tasks without a local Node.js installation.",
			"Wails remains the desktop shell, binding layer, and packaging toolchain.",
		},
	}
}

// PingDB 执行一次最小化的 SQLite 事务读写测试，并将结果返回给前端。
func (a *App) PingDB() string {
	gormDB, closeDB, err := a.openDatabase()
	if err != nil {
		return fmt.Sprintf("SQLite 读写失败：%v", err)
	}
	defer closeDB()

	tx := gormDB.Begin()
	if tx.Error != nil {
		return fmt.Sprintf("SQLite 读写失败：开启事务失败：%v", tx.Error)
	}

	probeRecord := model.TemplateConfig{
		Type:         model.TemplateTypeImportMember,
		Name:         fmt.Sprintf("ping-template-%s", time.Now().Format("20060102150405")),
		MappingRules: `{"nickname":"昵称","platform_uid":"用户ID"}`,
	}

	if err := tx.Create(&probeRecord).Error; err != nil {
		_ = tx.Rollback()
		return fmt.Sprintf("SQLite 读写失败：写入测试记录失败：%v", err)
	}

	var storedRecord model.TemplateConfig
	if err := tx.First(&storedRecord, probeRecord.ID).Error; err != nil {
		_ = tx.Rollback()
		return fmt.Sprintf("SQLite 读写失败：读取测试记录失败：%v", err)
	}

	if err := tx.Rollback().Error; err != nil {
		return fmt.Sprintf("SQLite 读写失败：回滚测试事务失败：%v", err)
	}

	return fmt.Sprintf(
		"SQLite 读写成功：已完成事务内写入与读取，测试记录 ID=%d，模板名=%s",
		storedRecord.ID,
		storedRecord.Name,
	)
}

// ValidateBatch 执行批次导出前的地址预校验，并返回缺失地址会员名单。
func (a *App) ValidateBatch(batchName string) (model.BatchValidationResult, error) {
	gormDB, closeDB, err := a.openDatabase()
	if err != nil {
		return model.BatchValidationResult{}, fmt.Errorf("批次预校验失败: %w", err)
	}
	defer closeDB()

	return service.ValidateBatch(gormDB, batchName)
}

func (a *App) openDatabase() (*gorm.DB, func(), error) {
	dbPath, err := a.resolveDatabasePath()
	if err != nil {
		return nil, nil, err
	}

	gormDB, err := database.InitDB(dbPath)
	if err != nil {
		return nil, nil, err
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("获取底层数据库连接失败：%w", err)
	}

	return gormDB, func() {
		_ = sqlDB.Close()
	}, nil
}

func (a *App) resolveDatabasePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err == nil {
		return filepath.Join(configDir, a.cfg.Name, "data", "eligiftmanager.db"), nil
	}

	workDir, workDirErr := os.Getwd()
	if workDirErr != nil {
		return "", fmt.Errorf("获取用户配置目录失败：%w；获取当前工作目录也失败：%v", err, workDirErr)
	}

	return filepath.Join(workDir, ".local", "data", "eligiftmanager.db"), nil
}
