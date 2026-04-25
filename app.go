package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/config"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

type App struct {
	ctx context.Context
	cfg config.App
}
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
	WaveCount        int64                `json:"waveCount"`
	RecentWaves      []WaveItem           `json:"recentWaves"`
	RecentDispatches []DispatchRecordItem `json:"recentDispatches"`
	Warnings         []DashboardWarning   `json:"warnings"`
}
type DashboardWarning struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Type   string `json:"type"`
}
type WaveItem struct {
	ID                    uint      `json:"id"`
	WaveNo                string    `json:"waveNo"`
	Name                  string    `json:"name"`
	Status                string    `json:"status"`
	TotalRecords          int64     `json:"totalRecords"`
	TotalQuantity         int64     `json:"totalQuantity"`
	PendingAddressRecords int64     `json:"pendingAddressRecords"`
	UpdatedAt             time.Time `json:"updatedAt"`
}
type MemberItem struct {
	ID                 uint                   `json:"id"`
	Platform           string                 `json:"platform"`
	PlatformUID        string                 `json:"platformUid"`
	LatestNickname     string                 `json:"latestNickname"`
	ExtraData          string                 `json:"extraData"`
	AddressCount       int                    `json:"addressCount"`
	ActiveAddressCount int                    `json:"activeAddressCount"`
	LatestRecipient    string                 `json:"latestRecipient"`
	LatestPhone        string                 `json:"latestPhone"`
	LatestAddress      string                 `json:"latestAddress"`
	DispatchCount      int64                  `json:"dispatchCount"`
	UpdatedAt          time.Time              `json:"updatedAt"`
	Addresses          []model.MemberAddress  `json:"addresses"`
	Nicknames          []model.MemberNickname `json:"nicknames"`
}
type MemberListPayload struct {
	Items     []MemberItem `json:"items"`
	Total     int64        `json:"total"`
	Platforms []string     `json:"platforms"`
}
type ProductItem struct {
	ID            uint      `json:"id"`
	Platform      string    `json:"platform"`
	Factory       string    `json:"factory"`
	FactorySKU    string    `json:"factorySku"`
	Name          string    `json:"name"`
	CoverImage    string    `json:"coverImage"`
	ExtraData     string    `json:"extraData"`
	DispatchCount int64     `json:"dispatchCount"`
	TotalQuantity int64     `json:"totalQuantity"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
type ProductListPayload struct {
	Items     []ProductItem `json:"items"`
	Total     int64         `json:"total"`
	Platforms []string      `json:"platforms"`
}
type DispatchRecordItem struct {
	ID              uint      `json:"id"`
	WaveID          uint      `json:"waveId"`
	WaveNo          string    `json:"waveNo"`
	Quantity        int       `json:"quantity"`
	Status          string    `json:"status"`
	MemberID        uint      `json:"memberId"`
	Platform        string    `json:"platform"`
	PlatformUID     string    `json:"platformUid"`
	MemberNickname  string    `json:"memberNickname"`
	ProductID       uint      `json:"productId"`
	ProductName     string    `json:"productName"`
	FactorySKU      string    `json:"factorySku"`
	MemberAddressID *uint     `json:"memberAddressId"`
	RecipientName   string    `json:"recipientName"`
	Phone           string    `json:"phone"`
	Address         string    `json:"address"`
	HasAddress      bool      `json:"hasAddress"`
	UpdatedAt       time.Time `json:"updatedAt"`
}
type TemplateItem struct {
	ID           uint      `json:"id"`
	Platform     string    `json:"platform"`
	Type         string    `json:"type"`
	Name         string    `json:"name"`
	MappingRules string    `json:"mappingRules"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func NewApp(cfg config.App) *App           { return &App{cfg: cfg} }
func (a *App) startup(ctx context.Context) { a.ctx = ctx }
func (a *App) Bootstrap() BootstrapPayload {
	return BootstrapPayload{Name: a.cfg.Name, Version: a.cfg.Version, Module: a.cfg.Module, Description: a.cfg.Description, Runtime: goruntime.Version(), Frontend: a.cfg.FrontendRuntime, Highlights: []string{"Wave workflow", "Platform isolation", "SQLite backup and restore"}}
}
func (a *App) PingDB() string {
	db, closeDB, err := a.openDatabase()
	if err != nil {
		return fmt.Sprintf("database connection failed: %v", err)
	}
	defer closeDB()
	if err := db.Exec("SELECT 1").Error; err != nil {
		return fmt.Sprintf("database probe failed: %v", err)
	}
	return "SQLite database connection is healthy"
}

func (a *App) CreateWave(name string) (model.Wave, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.Wave{}, fmt.Errorf("create wave failed: name is required")
	}
	db, closeDB, err := a.openDatabase()
	if err != nil {
		return model.Wave{}, fmt.Errorf("create wave failed: %w", err)
	}
	defer closeDB()
	wave := model.Wave{Name: name, Status: "draft"}
	const maxRetries = 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		err = db.Transaction(func(tx *gorm.DB) error {
			prefix := time.Now().Format("TASK-20060102")
			var count int64
			if err := tx.Model(&model.Wave{}).Where("wave_no LIKE ?", prefix+"-%").Count(&count).Error; err != nil {
				return err
			}
			wave.WaveNo = fmt.Sprintf("%s-%03d", prefix, count+1)
			return tx.Create(&wave).Error
		})
		if err == nil {
			return wave, nil
		}
		if !strings.Contains(err.Error(), "UNIQUE constraint failed") || attempt == maxRetries-1 {
			return model.Wave{}, fmt.Errorf("create wave failed: %w", err)
		}
	}
	return model.Wave{}, fmt.Errorf("create wave failed: max retries exceeded")
}
func (a *App) DeleteWave(waveID uint) error {
	db, closeDB, err := a.openDatabase()
	if err != nil {
		return fmt.Errorf("delete wave failed: %w", err)
	}
	defer closeDB()
	result := db.Delete(&model.Wave{}, waveID)
	if result.Error != nil {
		return fmt.Errorf("delete wave failed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("delete wave failed: wave not found")
	}
	return nil
}
func (a *App) ListWaves() ([]WaveItem, error) {
	db, closeDB, err := a.openDatabase()
	if err != nil {
		return nil, fmt.Errorf("list waves failed: %w", err)
	}
	defer closeDB()
	return queryWaves(db, 100)
}

func (a *App) ImportToWave(waveID uint, csvPath string, templateID uint) error {
	db, closeDB, err := a.openDatabase()
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}
	defer closeDB()
	var wave model.Wave
	if err := db.First(&wave, waveID).Error; err != nil {
		return fmt.Errorf("import failed: wave not found: %w", err)
	}
	var template model.TemplateConfig
	if err := db.First(&template, templateID).Error; err != nil {
		return fmt.Errorf("import failed: template not found: %w", err)
	}
	switch template.Type {
	case model.TemplateTypeImportMember:
		_, err = service.ImportMembersFromCSV(db, csvPath, template)
	case model.TemplateTypeImportProduct:
		var products []model.Product
		products, err = service.ParseProductCSV(csvPath, template)
		if err == nil {
			err = db.Transaction(func(tx *gorm.DB) error {
				for i := range products {
					products[i].Platform = template.Platform
					if products[i].ExtraData == "" {
						products[i].ExtraData = "{}"
					}
					if err := tx.Create(&products[i]).Error; err != nil {
						return err
					}
				}
				return nil
			})
		}
	case model.TemplateTypeImportDispatchRecord:
		var records []model.DispatchRecord
		records, err = service.ParseDispatchRecordCSV(csvPath, template)
		if err == nil {
			err = db.Transaction(func(tx *gorm.DB) error {
				for i := range records {
					records[i].WaveID = wave.ID
					if err := tx.Create(&records[i]).Error; err != nil {
						return err
					}
				}
				return nil
			})
		}
	default:
		err = fmt.Errorf("template type %q cannot import", template.Type)
	}
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}
	return nil
}
func (a *App) AutoAllocateWave(waveID uint, ruleTemplateID uint) error {
	db, closeDB, err := a.openDatabase()
	if err != nil {
		return fmt.Errorf("auto allocate failed: %w", err)
	}
	defer closeDB()
	var wave model.Wave
	if err := db.First(&wave, waveID).Error; err != nil {
		return fmt.Errorf("auto allocate failed: wave not found: %w", err)
	}
	var template model.TemplateConfig
	if err := db.First(&template, ruleTemplateID).Error; err != nil {
		return fmt.Errorf("auto allocate failed: template not found: %w", err)
	}
	if template.Type != model.TemplateTypeAllocation {
		return fmt.Errorf("auto allocate failed: template is not allocation type")
	}
	productIDs, err := parseAllocationProductIDs(template.MappingRules)
	if err != nil {
		return fmt.Errorf("auto allocate failed: %w", err)
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var members []model.Member
		if err := tx.Where("platform = ?", template.Platform).Order("updated_at DESC").Find(&members).Error; err != nil {
			return err
		}
		for i, member := range members {
			productID := productIDs[i%len(productIDs)]
			if err := ensureProductPlatform(tx, productID, template.Platform); err != nil {
				return err
			}
			addressID, status := defaultAddressForMember(tx, member.ID)
			record := model.DispatchRecord{WaveID: wave.ID, MemberID: member.ID, ProductID: productID, MemberAddressID: addressID, Quantity: 1, Status: status}
			if err := tx.Create(&record).Error; err != nil {
				return err
			}
		}
		return tx.Model(&wave).Update("status", "allocating").Error
	})
}
func (a *App) ExportWaveByPlatform(waveID uint, platform string, exportTemplateID uint) (string, error) {
	db, closeDB, err := a.openDatabase()
	if err != nil {
		return "", fmt.Errorf("export failed: %w", err)
	}
	defer closeDB()
	var template model.TemplateConfig
	if err := db.First(&template, exportTemplateID).Error; err != nil {
		return "", fmt.Errorf("export failed: template not found: %w", err)
	}
	if template.Type != model.TemplateTypeExportOrder {
		return "", fmt.Errorf("export failed: template is not export type")
	}
	if template.Platform != "" && template.Platform != platform {
		return "", fmt.Errorf("export failed: platform mismatch")
	}
	path := filepath.Join(os.TempDir(), fmt.Sprintf("eligift-task-%d-%s.csv", waveID, time.Now().Format("20060102150405")))
	if a.ctx != nil {
		selected, dialogErr := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{DefaultFilename: filepath.Base(path)})
		if dialogErr != nil {
			return "", fmt.Errorf("export failed: %w", dialogErr)
		}
		if selected == "" {
			return "", fmt.Errorf("export canceled")
		}
		path = selected
	}
	if err := writeWavePlatformCSV(db, waveID, platform, path); err != nil {
		return "", fmt.Errorf("export failed: %w", err)
	}
	if err := db.Model(&model.Wave{}).Where("id = ?", waveID).Update("status", "exported").Error; err != nil {
		return path, fmt.Errorf("export failed: %w", err)
	}
	return path, nil
}

func (a *App) ListMembers(page, pageSize int, keyword, platform string) (MemberListPayload, error) {
	db, closeDB, err := a.openDatabase()
	if err != nil {
		return MemberListPayload{}, fmt.Errorf("list members failed: %w", err)
	}
	defer closeDB()
	page, pageSize = normalizePagination(page, pageSize)

	countQuery := db.Model(&model.Member{})
	if platform = strings.TrimSpace(platform); platform != "" {
		countQuery = countQuery.Where("platform = ?", platform)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		sub := db.Model(&model.MemberNickname{}).Select("member_id").Where("nickname LIKE ?", like)
		countQuery = countQuery.Where("platform_uid LIKE ? OR id IN (?)", like, sub)
	}
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return MemberListPayload{}, err
	}

	query := db.Model(&model.Member{}).
		Preload("Nicknames", func(d *gorm.DB) *gorm.DB { return d.Order("created_at DESC") }).
		Preload("Addresses", func(d *gorm.DB) *gorm.DB { return d.Order("is_default DESC, created_at DESC") })
	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		sub := db.Model(&model.MemberNickname{}).Select("member_id").Where("nickname LIKE ?", like)
		query = query.Where("platform_uid LIKE ? OR id IN (?)", like, sub)
	}
	var members []model.Member
	if err := query.Order("updated_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&members).Error; err != nil {
		return MemberListPayload{}, err
	}
	items, err := buildMemberItems(db, members)
	if err != nil {
		return MemberListPayload{}, err
	}
	platforms, err := queryMemberPlatforms(db)
	if err != nil {
		return MemberListPayload{}, err
	}
	return MemberListPayload{Items: items, Total: total, Platforms: platforms}, nil
}
func (a *App) ListProducts(page, pageSize int, keyword, platform string) (ProductListPayload, error) {
	db, closeDB, err := a.openDatabase()
	if err != nil {
		return ProductListPayload{}, fmt.Errorf("list products failed: %w", err)
	}
	defer closeDB()
	page, pageSize = normalizePagination(page, pageSize)

	q := db.Model(&model.Product{})
	if platform = strings.TrimSpace(platform); platform != "" {
		q = q.Where("platform = ?", platform)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("name LIKE ? OR factory_sku LIKE ? OR factory LIKE ?", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return ProductListPayload{}, err
	}
	var products []model.Product
	if err := q.Order("updated_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&products).Error; err != nil {
		return ProductListPayload{}, err
	}
	items, err := buildProductItems(db, products)
	if err != nil {
		return ProductListPayload{}, err
	}
	platforms, err := queryProductPlatforms(db)
	if err != nil {
		return ProductListPayload{}, err
	}
	return ProductListPayload{Items: items, Total: total, Platforms: platforms}, nil
}
func (a *App) SetDefaultAddress(memberID, addressID uint) error {
	db, closeDB, err := a.openDatabase()
	if err != nil {
		return fmt.Errorf("set default address failed: %w", err)
	}
	defer closeDB()
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.MemberAddress{}).Where("member_id = ?", memberID).Update("is_default", false).Error; err != nil {
			return err
		}
		r := tx.Model(&model.MemberAddress{}).Where("id = ? AND member_id = ? AND is_deleted = ?", addressID, memberID, false).Update("is_default", true)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return fmt.Errorf("address not found")
		}
		return nil
	})
}
func (a *App) UpdateProduct(product model.Product) error {
	db, closeDB, err := a.openDatabase()
	if err != nil {
		return fmt.Errorf("update product failed: %w", err)
	}
	defer closeDB()
	if product.ID == 0 {
		return fmt.Errorf("product id is required")
	}
	updates := map[string]any{"platform": strings.TrimSpace(product.Platform), "name": strings.TrimSpace(product.Name), "cover_image": product.CoverImage, "extra_data": product.ExtraData}
	if updates["platform"] == "" || updates["name"] == "" {
		return fmt.Errorf("platform and name are required")
	}
	if updates["extra_data"] == "" {
		updates["extra_data"] = "{}"
	}
	return db.Model(&model.Product{}).Where("id = ?", product.ID).Updates(updates).Error
}
func (a *App) ListDispatchRecords(waveID uint) ([]DispatchRecordItem, error) {
	db, closeDB, err := a.openDatabase()
	if err != nil {
		return nil, err
	}
	defer closeDB()
	return queryDispatchRecords(db, waveID, 500)
}

func (a *App) CreateTemplate(platform, templateType, name, mappingRules string) (TemplateItem, error) {
	platform = strings.TrimSpace(platform)
	templateType = strings.TrimSpace(templateType)
	name = strings.TrimSpace(name)
	mappingRules = strings.TrimSpace(mappingRules)
	if platform == "" || templateType == "" || name == "" {
		return TemplateItem{}, fmt.Errorf("platform, type and name are required")
	}
	if mappingRules == "" {
		mappingRules = "{}"
	}
	var probe map[string]any
	if err := json.Unmarshal([]byte(mappingRules), &probe); err != nil {
		return TemplateItem{}, fmt.Errorf("mapping rules must be valid JSON: %w", err)
	}
	db, closeDB, err := a.openDatabase()
	if err != nil {
		return TemplateItem{}, err
	}
	defer closeDB()
	template := model.TemplateConfig{Platform: platform, Type: templateType, Name: name, MappingRules: mappingRules}
	if err := db.Create(&template).Error; err != nil {
		return TemplateItem{}, err
	}
	return templateItemFromModel(template), nil
}
func (a *App) ListTemplates() ([]TemplateItem, error) {
	db, closeDB, err := a.openDatabase()
	if err != nil {
		return nil, err
	}
	defer closeDB()
	var templates []model.TemplateConfig
	if err := db.Order("platform ASC, type ASC, updated_at DESC").Find(&templates).Error; err != nil {
		return nil, err
	}
	items := make([]TemplateItem, 0, len(templates))
	for _, template := range templates {
		items = append(items, templateItemFromModel(template))
	}
	return items, nil
}

func (a *App) BackupDatabase() (string, error) {
	dbPath, err := a.resolveDatabasePath()
	if err != nil {
		return "", err
	}
	target := filepath.Join(os.TempDir(), "eligiftmanager-backup.db")
	if a.ctx != nil {
		selected, dialogErr := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{DefaultFilename: fmt.Sprintf("eligiftmanager-%s.db", time.Now().Format("20060102150405"))})
		if dialogErr != nil {
			return "", dialogErr
		}
		if selected == "" {
			return "", fmt.Errorf("backup canceled")
		}
		target = selected
	}
	if err := copyFile(dbPath, target); err != nil {
		return "", err
	}
	return target, nil
}
func (a *App) RestoreDatabase() error {
	dbPath, err := a.resolveDatabasePath()
	if err != nil {
		return err
	}
	if a.ctx == nil {
		return fmt.Errorf("Wails runtime is required")
	}
	source, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{Title: "Select EliGiftManager backup database"})
	if err != nil {
		return err
	}
	if source == "" {
		return fmt.Errorf("restore canceled")
	}
	sameFile, err := sameFilePath(source, dbPath)
	if err != nil {
		return err
	}
	if sameFile {
		return fmt.Errorf("restore source must be different from the active database")
	}
	if err := validateDatabaseFile(source); err != nil {
		return err
	}
	backupPath := dbPath + ".before-restore-" + time.Now().Format("20060102150405")
	if _, statErr := os.Stat(dbPath); statErr == nil {
		if err := copyFile(dbPath, backupPath); err != nil {
			return err
		}
	} else if !os.IsNotExist(statErr) {
		return statErr
	}
	if err := copyFile(source, dbPath); err != nil {
		return err
	}
	return validateDatabaseFile(dbPath)
}

func (a *App) GetDashboard() (DashboardPayload, error) {
	db, closeDB, err := a.openDatabase()
	if err != nil {
		return DashboardPayload{}, err
	}
	defer closeDB()
	dbPath, _ := a.resolveDatabasePath()
	payload := DashboardPayload{DatabasePath: dbPath}
	if err := db.Model(&model.Member{}).Count(&payload.MemberCount).Error; err != nil {
		return payload, err
	}
	if err := db.Model(&model.Product{}).Count(&payload.ProductCount).Error; err != nil {
		return payload, err
	}
	if err := db.Model(&model.DispatchRecord{}).Count(&payload.DispatchCount).Error; err != nil {
		return payload, err
	}
	if err := db.Model(&model.TemplateConfig{}).Count(&payload.TemplateCount).Error; err != nil {
		return payload, err
	}
	if err := db.Model(&model.Wave{}).Count(&payload.WaveCount).Error; err != nil {
		return payload, err
	}
	if err := db.Model(&model.MemberAddress{}).Where("is_deleted = ?", false).Count(&payload.AddressCount).Error; err != nil {
		return payload, err
	}
	if err := db.Model(&model.DispatchRecord{}).Where("status = ?", model.DispatchStatusPendingAddress).Count(&payload.PendingAddresses).Error; err != nil {
		return payload, err
	}
	active := db.Model(&model.MemberAddress{}).Select("member_id").Where("is_deleted = ?", false)
	if err := db.Model(&model.Member{}).Where("id NOT IN (?)", active).Count(&payload.MissingAddresses).Error; err != nil {
		return payload, err
	}
	payload.RecentWaves, err = queryWaves(db, 8)
	if err != nil {
		return payload, err
	}
	payload.RecentDispatches, err = queryDispatchRecords(db, 0, 8)
	if err != nil {
		return payload, err
	}
	payload.Warnings = buildDashboardWarnings(payload)
	return payload, nil
}

func queryWaves(db *gorm.DB, limit int) ([]WaveItem, error) {
	var waves []model.Wave
	if err := db.Order("updated_at DESC").Limit(limit).Find(&waves).Error; err != nil {
		return nil, err
	}
	items := make([]WaveItem, 0, len(waves))
	for _, wave := range waves {
		item := WaveItem{ID: wave.ID, WaveNo: wave.WaveNo, Name: wave.Name, Status: wave.Status, UpdatedAt: wave.UpdatedAt}
		if err := db.Model(&model.DispatchRecord{}).Where("wave_id = ?", wave.ID).Select("COUNT(*) AS total_records, COALESCE(SUM(quantity), 0) AS total_quantity, SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) AS pending_address_records", model.DispatchStatusPendingAddress).Row().Scan(&item.TotalRecords, &item.TotalQuantity, &item.PendingAddressRecords); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}
func queryDispatchRecords(db *gorm.DB, waveID uint, limit int) ([]DispatchRecordItem, error) {
	var records []model.DispatchRecord
	q := db.Preload("Wave").Preload("Member.Nicknames", func(d *gorm.DB) *gorm.DB { return d.Order("created_at DESC") }).Preload("Product").Preload("MemberAddress").Order("updated_at DESC").Limit(limit)
	if waveID != 0 {
		q = q.Where("wave_id = ?", waveID)
	}
	if err := q.Find(&records).Error; err != nil {
		return nil, err
	}
	items := make([]DispatchRecordItem, 0, len(records))
	for _, record := range records {
		item := DispatchRecordItem{ID: record.ID, WaveID: record.WaveID, WaveNo: record.Wave.WaveNo, Quantity: record.Quantity, Status: record.Status, MemberID: record.MemberID, Platform: record.Member.Platform, PlatformUID: record.Member.PlatformUID, MemberNickname: latestNickname(record.Member), ProductID: record.ProductID, ProductName: record.Product.Name, FactorySKU: record.Product.FactorySKU, MemberAddressID: record.MemberAddressID, UpdatedAt: record.UpdatedAt}
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
func buildMemberItems(db *gorm.DB, members []model.Member) ([]MemberItem, error) {
	items := make([]MemberItem, 0, len(members))
	dispatchCounts, err := queryDispatchCountsByMemberID(db, members)
	if err != nil {
		return nil, err
	}
	for _, member := range members {
		item := MemberItem{ID: member.ID, Platform: member.Platform, PlatformUID: member.PlatformUID, LatestNickname: latestNickname(member), ExtraData: member.ExtraData, AddressCount: len(member.Addresses), UpdatedAt: member.UpdatedAt, Addresses: member.Addresses, Nicknames: member.Nicknames, DispatchCount: dispatchCounts[member.ID]}
		for _, address := range member.Addresses {
			if address.IsDeleted {
				continue
			}
			item.ActiveAddressCount++
			if item.LatestAddress == "" || address.IsDefault {
				item.LatestRecipient = address.RecipientName
				item.LatestPhone = address.Phone
				item.LatestAddress = address.Address
			}
		}
		items = append(items, item)
	}
	return items, nil
}
func buildProductItems(db *gorm.DB, products []model.Product) ([]ProductItem, error) {
	items := make([]ProductItem, 0, len(products))
	aggregates, err := queryProductDispatchAggregates(db, products)
	if err != nil {
		return nil, err
	}
	for _, product := range products {
		aggregate := aggregates[product.ID]
		item := ProductItem{ID: product.ID, Platform: product.Platform, Factory: product.Factory, FactorySKU: product.FactorySKU, Name: product.Name, CoverImage: product.CoverImage, ExtraData: product.ExtraData, DispatchCount: aggregate.DispatchCount, TotalQuantity: aggregate.TotalQuantity, UpdatedAt: product.UpdatedAt}
		items = append(items, item)
	}
	return items, nil
}
func queryDispatchCountsByMemberID(db *gorm.DB, members []model.Member) (map[uint]int64, error) {
	memberIDs := make([]uint, 0, len(members))
	for _, member := range members {
		memberIDs = append(memberIDs, member.ID)
	}
	if len(memberIDs) == 0 {
		return map[uint]int64{}, nil
	}
	type dispatchCountRow struct {
		MemberID      uint
		DispatchCount int64
	}
	var rows []dispatchCountRow
	if err := db.Model(&model.DispatchRecord{}).
		Select("member_id, COUNT(*) AS dispatch_count").
		Where("member_id IN ?", memberIDs).
		Group("member_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	counts := make(map[uint]int64, len(rows))
	for _, row := range rows {
		counts[row.MemberID] = row.DispatchCount
	}
	return counts, nil
}

type productDispatchAggregate struct {
	DispatchCount int64
	TotalQuantity int64
}

func queryProductDispatchAggregates(db *gorm.DB, products []model.Product) (map[uint]productDispatchAggregate, error) {
	productIDs := make([]uint, 0, len(products))
	for _, product := range products {
		productIDs = append(productIDs, product.ID)
	}
	if len(productIDs) == 0 {
		return map[uint]productDispatchAggregate{}, nil
	}
	type productDispatchAggregateRow struct {
		ProductID      uint
		DispatchCount  int64
		TotalQuantity  int64
	}
	var rows []productDispatchAggregateRow
	if err := db.Model(&model.DispatchRecord{}).
		Select("product_id, COUNT(*) AS dispatch_count, COALESCE(SUM(quantity), 0) AS total_quantity").
		Where("product_id IN ?", productIDs).
		Group("product_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	aggregates := make(map[uint]productDispatchAggregate, len(rows))
	for _, row := range rows {
		aggregates[row.ProductID] = productDispatchAggregate{DispatchCount: row.DispatchCount, TotalQuantity: row.TotalQuantity}
	}
	return aggregates, nil
}

func queryProductPlatforms(db *gorm.DB) ([]string, error) {
	var platforms []string
	if err := db.Model(&model.Product{}).Distinct().Order("platform ASC").Pluck("platform", &platforms).Error; err != nil {
		return nil, err
	}
	return platforms, nil
}

func queryMemberPlatforms(db *gorm.DB) ([]string, error) {
	var platforms []string
	if err := db.Model(&model.Member{}).Distinct().Order("platform ASC").Pluck("platform", &platforms).Error; err != nil {
		return nil, err
	}
	return platforms, nil
}

func latestNickname(member model.Member) string {
	if len(member.Nicknames) > 0 {
		return member.Nicknames[0].Nickname
	}
	return member.PlatformUID
}
func normalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}
func parseAllocationProductIDs(raw string) ([]uint, error) {
	var generic map[string]any
	if err := json.Unmarshal([]byte(raw), &generic); err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(generic))
	for _, v := range generic {
		switch value := v.(type) {
		case float64:
			ids = append(ids, uint(value))
		case string:
			parsed, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
			if err != nil {
				return nil, err
			}
			ids = append(ids, uint(parsed))
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("at least one product id is required")
	}
	return ids, nil
}
func ensureProductPlatform(tx *gorm.DB, productID uint, platform string) error {
	var count int64
	if err := tx.Model(&model.Product{}).Where("id = ? AND platform = ?", productID, platform).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("product %d not found in platform %s", productID, platform)
	}
	return nil
}
func defaultAddressForMember(tx *gorm.DB, memberID uint) (*uint, string) {
	var address model.MemberAddress
	if err := tx.Where("member_id = ? AND is_deleted = ?", memberID, false).Order("is_default DESC, created_at DESC").First(&address).Error; err != nil {
		return nil, model.DispatchStatusPendingAddress
	}
	return &address.ID, model.DispatchStatusPending
}
func writeWavePlatformCSV(db *gorm.DB, waveID uint, platform, path string) error {
	var records []model.DispatchRecord
	if err := db.Preload("Wave").Preload("Member.Nicknames", func(d *gorm.DB) *gorm.DB { return d.Order("created_at DESC") }).Preload("Product").Preload("MemberAddress").Joins("JOIN products ON products.id = dispatch_records.product_id").Where("dispatch_records.wave_id = ? AND products.platform = ?", waveID, platform).Order("dispatch_records.id ASC").Find(&records).Error; err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write([]string{"WaveNo", "Platform", "PlatformUID", "Nickname", "Product", "FactorySKU", "Quantity", "Recipient", "Phone", "Address"}); err != nil {
		return err
	}
	for _, record := range records {
		recipient, phone, address := "", "", ""
		if record.MemberAddress != nil {
			recipient = record.MemberAddress.RecipientName
			phone = record.MemberAddress.Phone
			address = record.MemberAddress.Address
		}
		if err := writer.Write([]string{record.Wave.WaveNo, platform, record.Member.PlatformUID, latestNickname(record.Member), record.Product.Name, record.Product.FactorySKU, strconv.Itoa(record.Quantity), recipient, phone, address}); err != nil {
			return err
		}
	}
	return writer.Error()
}
func buildDashboardWarnings(payload DashboardPayload) []DashboardWarning {
	warnings := make([]DashboardWarning, 0, 3)
	if payload.PendingAddresses > 0 {
		warnings = append(warnings, DashboardWarning{Title: "Pending addresses", Detail: fmt.Sprintf("%d dispatch records need addresses", payload.PendingAddresses), Type: "warning"})
	}
	if payload.MissingAddresses > 0 {
		warnings = append(warnings, DashboardWarning{Title: "Missing member addresses", Detail: fmt.Sprintf("%d members have no active address", payload.MissingAddresses), Type: "error"})
	}
	if payload.TemplateCount == 0 {
		warnings = append(warnings, DashboardWarning{Title: "No templates", Detail: "No import/export/allocation templates configured", Type: "info"})
	}
	return warnings
}
func templateItemFromModel(template model.TemplateConfig) TemplateItem {
	return TemplateItem{ID: template.ID, Platform: template.Platform, Type: template.Type, Name: template.Name, MappingRules: template.MappingRules, CreatedAt: template.CreatedAt, UpdatedAt: template.UpdatedAt}
}

func (a *App) openDatabase() (*gorm.DB, func(), error) {
	dbPath, err := a.resolveDatabasePath()
	if err != nil {
		return nil, nil, err
	}
	db, err := database.InitDB(dbPath)
	if err != nil {
		return nil, nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}
	return db, func() { _ = sqlDB.Close() }, nil
}
func (a *App) resolveDatabasePath() (string, error) {
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		// When running via wails dev the binary is compiled to a temp
		// directory. Fall back to the working directory (project root) so
		// the database persists across dev restarts.
		if !strings.HasPrefix(execDir, os.TempDir()) {
			return filepath.Join(execDir, "data", "eligiftmanager.db"), nil
		}
	}
	workDir, workDirErr := os.Getwd()
	if workDirErr != nil {
		return "", fmt.Errorf("resolve database path failed: %w", workDirErr)
	}
	return filepath.Join(workDir, "data", "eligiftmanager.db"), nil
}
func copyFile(source, target string) error {
	sameFile, err := sameFilePath(source, target)
	if err != nil {
		return err
	}
	if sameFile {
		return fmt.Errorf("source and target must be different files")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func sameFilePath(left, right string) (bool, error) {
	leftPath, err := filepath.Abs(left)
	if err != nil {
		return false, err
	}
	rightPath, err := filepath.Abs(right)
	if err != nil {
		return false, err
	}
	cleanLeft := filepath.Clean(leftPath)
	cleanRight := filepath.Clean(rightPath)
	if goruntime.GOOS == "windows" {
		return strings.EqualFold(cleanLeft, cleanRight), nil
	}
	return cleanLeft == cleanRight, nil
}

func validateDatabaseFile(dbPath string) error {
	probe, err := sql.Open("sqlite3", filepath.Clean(dbPath))
	if err != nil {
		return fmt.Errorf("validate database file failed: %w", err)
	}
	defer probe.Close()
	if err := probe.Ping(); err != nil {
		return fmt.Errorf("validate database file failed: %w", err)
	}
	var quickCheck string
	if err := probe.QueryRow("PRAGMA quick_check(1)").Scan(&quickCheck); err != nil {
		return fmt.Errorf("validate database file failed: %w", err)
	}
	if quickCheck != "ok" {
		return fmt.Errorf("validate database file failed: integrity check returned %q", quickCheck)
	}
	var tableName string
	if err := probe.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = ? LIMIT 1", model.Member{}.TableName()).Scan(&tableName); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("validate database file failed: missing required table %q", model.Member{}.TableName())
		}
		return fmt.Errorf("validate database file failed: %w", err)
	}
	return nil
}
