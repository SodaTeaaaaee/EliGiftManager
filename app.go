package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/config"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

// ---- globals (init-time wiring) ----

// controllerCtx is kept for controller_wave.go (ExportOrderCSV SaveFileDialog).
// It will be migrated to an instance field in a follow-up.
var controllerCtx context.Context

// sysCtrl is set by main() so App.startup can feed it the Wails runtime context.
var sysCtrl *SystemController

// SetControllerContext stores the Wails context for controller use.
func SetControllerContext(ctx context.Context) { controllerCtx = ctx }

// ---- App struct ----

type App struct {
	ctx context.Context
	cfg config.App
}

// ---- shared payload types (used by controllers) ----

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
	LevelTags             string    `json:"levelTags"`
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
type ProductItemWithTags struct {
	ID         uint      `json:"id"`
	Platform   string    `json:"platform"`
	Factory    string    `json:"factory"`
	FactorySKU string    `json:"factorySku"`
	Name       string    `json:"name"`
	CoverImage string    `json:"coverImage"`
	ExtraData  string    `json:"extraData"`
	Tags       []string  `json:"tags"`
	UpdatedAt  time.Time `json:"updatedAt"`
}
type ProductListWithTagsPayload struct {
	Items     []ProductItemWithTags `json:"items"`
	Total     int64                 `json:"total"`
	Platforms []string              `json:"platforms"`
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

// ---- App: lifecycle + file-picker ----

func NewApp(cfg config.App) *App { return &App{cfg: cfg} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	controllerCtx = ctx
	if sysCtrl != nil {
		sysCtrl.appCtx = ctx
	}
}

func (a *App) PickCSVFile() (string, error) {
	if a.ctx != nil {
		selected, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
			Title:   "选择 CSV 文件",
			Filters: []wailsruntime.FileFilter{{DisplayName: "CSV 文件 (*.csv)", Pattern: "*.csv"}},
		})
		if err != nil {
			return "", err
		}
		return selected, nil
	}
	return "", fmt.Errorf("pick CSV file: context not available")
}

func (a *App) PickZIPFile() (string, error) {
	if a.ctx != nil {
		selected, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
			Title:   "选择 ZIP 文件",
			Filters: []wailsruntime.FileFilter{{DisplayName: "ZIP 压缩文件 (*.zip)", Pattern: "*.zip"}},
		})
		if err != nil {
			return "", err
		}
		return selected, nil
	}
	return "", fmt.Errorf("pick ZIP file: context not available")
}

func (a *App) resolveDatabasePath() (string, error) {
	dataDir, err := service.ResolveDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "eligiftmanager.db"), nil
}

// ---- shared helpers (used by controllers) ----

func queryWaves(db *gorm.DB, limit int, status string) ([]WaveItem, error) {
	var waves []model.Wave
	q := db
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Order("updated_at DESC").Limit(limit).Find(&waves).Error; err != nil {
		return nil, err
	}
	items := make([]WaveItem, 0, len(waves))
	for _, wave := range waves {
		item := WaveItem{ID: wave.ID, WaveNo: wave.WaveNo, Name: wave.Name, Status: wave.Status, LevelTags: wave.LevelTags, UpdatedAt: wave.UpdatedAt}
		if err := db.Model(&model.DispatchRecord{}).Where("wave_id = ?", wave.ID).Select("COUNT(*) AS total_records, COALESCE(SUM(quantity), 0) AS total_quantity, COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS pending_address_records", model.DispatchStatusPendingAddress).Row().Scan(&item.TotalRecords, &item.TotalQuantity, &item.PendingAddressRecords); err != nil {
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
		ProductID     uint
		DispatchCount int64
		TotalQuantity int64
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
