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

type DashboardPayload struct {
	DatabasePath     string               `json:"databasePath"`
	MemberCount      int64                `json:"memberCount"`
	ProductCount     int64                `json:"productCount"`
	DispatchCount    int64                `json:"dispatchCount"`
	TemplateCount    int64                `json:"templateCount"`
	AddressCount     int64                `json:"addressCount"`
	MissingAddresses int64                `json:"missingAddresses"`
	PendingAddresses int64                `json:"pendingAddresses"`
	BatchCount       int64                `json:"batchCount"`
	Batches          []BatchSummary       `json:"batches"`
	RecentDispatches []DispatchRecordItem `json:"recentDispatches"`
	Warnings         []DashboardWarning   `json:"warnings"`
}

type DashboardWarning struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Type   string `json:"type"`
}

type BatchSummary struct {
	BatchName             string    `json:"batchName"`
	TotalRecords          int64     `json:"totalRecords"`
	TotalQuantity         int64     `json:"totalQuantity"`
	PendingAddressRecords int64     `json:"pendingAddressRecords"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

type MemberItem struct {
	ID                 uint      `json:"id"`
	Platform           string    `json:"platform"`
	PlatformUID        string    `json:"platformUid"`
	LatestNickname     string    `json:"latestNickname"`
	AddressCount       int       `json:"addressCount"`
	ActiveAddressCount int       `json:"activeAddressCount"`
	LatestRecipient    string    `json:"latestRecipient"`
	LatestPhone        string    `json:"latestPhone"`
	LatestAddress      string    `json:"latestAddress"`
	DispatchCount      int64     `json:"dispatchCount"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type ProductItem struct {
	ID            uint      `json:"id"`
	Factory       string    `json:"factory"`
	FactorySKU    string    `json:"factorySku"`
	Name          string    `json:"name"`
	ImagePath     string    `json:"imagePath"`
	ExtraData     string    `json:"extraData"`
	DispatchCount int64     `json:"dispatchCount"`
	TotalQuantity int64     `json:"totalQuantity"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type DispatchRecordItem struct {
	ID             uint      `json:"id"`
	BatchName      string    `json:"batchName"`
	Quantity       int       `json:"quantity"`
	Status         string    `json:"status"`
	MemberID       uint      `json:"memberId"`
	Platform       string    `json:"platform"`
	PlatformUID    string    `json:"platformUid"`
	MemberNickname string    `json:"memberNickname"`
	ProductID      uint      `json:"productId"`
	ProductName    string    `json:"productName"`
	FactorySKU     string    `json:"factorySku"`
	RecipientName  string    `json:"recipientName"`
	Phone          string    `json:"phone"`
	Address        string    `json:"address"`
	HasAddress     bool      `json:"hasAddress"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type TemplateItem struct {
	ID           uint      `json:"id"`
	Type         string    `json:"type"`
	Name         string    `json:"name"`
	MappingRules string    `json:"mappingRules"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
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

func (a *App) GetDashboard() (DashboardPayload, error) {
	gormDB, closeDB, err := a.openDatabase()
	if err != nil {
		return DashboardPayload{}, err
	}
	defer closeDB()

	dbPath, _ := a.resolveDatabasePath()
	payload := DashboardPayload{DatabasePath: dbPath}

	if err := gormDB.Model(&model.Member{}).Count(&payload.MemberCount).Error; err != nil {
		return DashboardPayload{}, err
	}
	if err := gormDB.Model(&model.Product{}).Count(&payload.ProductCount).Error; err != nil {
		return DashboardPayload{}, err
	}
	if err := gormDB.Model(&model.DispatchRecord{}).Count(&payload.DispatchCount).Error; err != nil {
		return DashboardPayload{}, err
	}
	if err := gormDB.Model(&model.TemplateConfig{}).Count(&payload.TemplateCount).Error; err != nil {
		return DashboardPayload{}, err
	}
	if err := gormDB.Model(&model.MemberAddress{}).Where("is_deleted = ?", false).Count(&payload.AddressCount).Error; err != nil {
		return DashboardPayload{}, err
	}
	if err := gormDB.Model(&model.DispatchRecord{}).Where("status = ?", model.DispatchStatusPendingAddress).Count(&payload.PendingAddresses).Error; err != nil {
		return DashboardPayload{}, err
	}

	activeAddressMembers := gormDB.Model(&model.MemberAddress{}).Select("member_id").Where("is_deleted = ?", false)
	if err := gormDB.Model(&model.Member{}).Where("id NOT IN (?)", activeAddressMembers).Count(&payload.MissingAddresses).Error; err != nil {
		return DashboardPayload{}, err
	}
	if err := gormDB.Model(&model.DispatchRecord{}).Distinct("batch_name").Count(&payload.BatchCount).Error; err != nil {
		return DashboardPayload{}, err
	}

	batches, err := queryBatchSummaries(gormDB, 8)
	if err != nil {
		return DashboardPayload{}, err
	}
	payload.Batches = batches

	recentDispatches, err := queryDispatchRecords(gormDB, 8)
	if err != nil {
		return DashboardPayload{}, err
	}
	payload.RecentDispatches = recentDispatches
	payload.Warnings = buildDashboardWarnings(payload)

	return payload, nil
}

func (a *App) ListMembers() ([]MemberItem, error) {
	gormDB, closeDB, err := a.openDatabase()
	if err != nil {
		return nil, err
	}
	defer closeDB()

	var members []model.Member
	if err := gormDB.Preload("Nicknames", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at DESC")
	}).Preload("Addresses", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at DESC")
	}).Order("updated_at DESC").Find(&members).Error; err != nil {
		return nil, err
	}

	items := make([]MemberItem, 0, len(members))
	for _, member := range members {
		item := MemberItem{
			ID:             member.ID,
			Platform:       member.Platform,
			PlatformUID:    member.PlatformUID,
			LatestNickname: latestNickname(member),
			AddressCount:   len(member.Addresses),
			UpdatedAt:      member.UpdatedAt,
		}

		for _, address := range member.Addresses {
			if address.IsDeleted {
				continue
			}
			item.ActiveAddressCount++
			if item.LatestAddress == "" {
				item.LatestRecipient = address.RecipientName
				item.LatestPhone = address.Phone
				item.LatestAddress = address.Address
			}
		}

		if err := gormDB.Model(&model.DispatchRecord{}).Where("member_id = ?", member.ID).Count(&item.DispatchCount).Error; err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, nil
}

func (a *App) ListProducts() ([]ProductItem, error) {
	gormDB, closeDB, err := a.openDatabase()
	if err != nil {
		return nil, err
	}
	defer closeDB()

	var products []model.Product
	if err := gormDB.Order("updated_at DESC").Find(&products).Error; err != nil {
		return nil, err
	}

	items := make([]ProductItem, 0, len(products))
	for _, product := range products {
		item := ProductItem{
			ID:         product.ID,
			Factory:    product.Factory,
			FactorySKU: product.FactorySKU,
			Name:       product.Name,
			ImagePath:  product.ImagePath,
			ExtraData:  product.ExtraData,
			UpdatedAt:  product.UpdatedAt,
		}

		if err := gormDB.Model(&model.DispatchRecord{}).
			Where("product_id = ?", product.ID).
			Select("COUNT(*) AS dispatch_count, COALESCE(SUM(quantity), 0) AS total_quantity").
			Row().Scan(&item.DispatchCount, &item.TotalQuantity); err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, nil
}

func (a *App) ListDispatchRecords() ([]DispatchRecordItem, error) {
	gormDB, closeDB, err := a.openDatabase()
	if err != nil {
		return nil, err
	}
	defer closeDB()

	return queryDispatchRecords(gormDB, 500)
}

func (a *App) ListTemplates() ([]TemplateItem, error) {
	gormDB, closeDB, err := a.openDatabase()
	if err != nil {
		return nil, err
	}
	defer closeDB()

	var templates []model.TemplateConfig
	if err := gormDB.Order("updated_at DESC").Find(&templates).Error; err != nil {
		return nil, err
	}

	items := make([]TemplateItem, 0, len(templates))
	for _, template := range templates {
		items = append(items, TemplateItem{
			ID:           template.ID,
			Type:         template.Type,
			Name:         template.Name,
			MappingRules: template.MappingRules,
			CreatedAt:    template.CreatedAt,
			UpdatedAt:    template.UpdatedAt,
		})
	}

	return items, nil
}

func queryBatchSummaries(gormDB *gorm.DB, limit int) ([]BatchSummary, error) {
	var batches []BatchSummary
	err := gormDB.Model(&model.DispatchRecord{}).
		Select("batch_name, COUNT(*) AS total_records, COALESCE(SUM(quantity), 0) AS total_quantity, SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) AS pending_address_records, MAX(updated_at) AS updated_at", model.DispatchStatusPendingAddress).
		Group("batch_name").
		Order("updated_at DESC").
		Limit(limit).
		Scan(&batches).Error
	return batches, err
}

func queryDispatchRecords(gormDB *gorm.DB, limit int) ([]DispatchRecordItem, error) {
	var records []model.DispatchRecord
	if err := gormDB.Preload("Member.Nicknames", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at DESC")
	}).Preload("Product").Preload("MemberAddress").Order("updated_at DESC").Limit(limit).Find(&records).Error; err != nil {
		return nil, err
	}

	items := make([]DispatchRecordItem, 0, len(records))
	for _, record := range records {
		item := DispatchRecordItem{
			ID:             record.ID,
			BatchName:      record.BatchName,
			Quantity:       record.Quantity,
			Status:         record.Status,
			MemberID:       record.MemberID,
			Platform:       record.Member.Platform,
			PlatformUID:    record.Member.PlatformUID,
			MemberNickname: latestNickname(record.Member),
			ProductID:      record.ProductID,
			ProductName:    record.Product.Name,
			FactorySKU:     record.Product.FactorySKU,
			UpdatedAt:      record.UpdatedAt,
		}

		if record.MemberAddress != nil && !record.MemberAddress.IsDeleted {
			item.HasAddress = true
			item.RecipientName = record.MemberAddress.RecipientName
			item.Phone = record.MemberAddress.Phone
			item.Address = record.MemberAddress.Address
		}

		items = append(items, item)
	}

	return items, nil
}

func latestNickname(member model.Member) string {
	if len(member.Nicknames) > 0 {
		return member.Nicknames[0].Nickname
	}
	return member.PlatformUID
}

func buildDashboardWarnings(payload DashboardPayload) []DashboardWarning {
	warnings := make([]DashboardWarning, 0, 3)
	if payload.PendingAddresses > 0 {
		warnings = append(warnings, DashboardWarning{Title: "待补全地址", Detail: fmt.Sprintf("%d 条派发记录缺少可用收件地址", payload.PendingAddresses), Type: "warning"})
	}
	if payload.MissingAddresses > 0 {
		warnings = append(warnings, DashboardWarning{Title: "会员地址缺失", Detail: fmt.Sprintf("%d 位会员还没有有效地址", payload.MissingAddresses), Type: "error"})
	}
	if payload.TemplateCount == 0 {
		warnings = append(warnings, DashboardWarning{Title: "模板未配置", Detail: "数据库中还没有导入/导出模板配置", Type: "info"})
	}
	return warnings
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
