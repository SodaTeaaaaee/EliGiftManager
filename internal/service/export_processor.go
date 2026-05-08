package service

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/gorm"
)

type exportTemplateConfig struct {
	Headers            []string `json:"headers"`
	Prefix             string   `json:"prefix"`
	BlankLeadingColumn bool     `json:"blankLeadingColumn"`
}

// ExportOrderCSV writes a factory-ready batch mini-order CSV for the given wave.
// Headers and order-number prefix are driven by the export_order template.
// It aborts if any record still has no address, and updates the wave
// status to "exported" on success.
func ExportOrderCSV(db *gorm.DB, waveID uint, outputPath string, template model.TemplateConfig) error {
	if db == nil {
		return fmt.Errorf("export order CSV failed: database connection is required")
	}
	if waveID == 0 {
		return fmt.Errorf("export order CSV failed: wave ID is required")
	}

	var cfg exportTemplateConfig
	if err := json.Unmarshal([]byte(template.MappingRules), &cfg); err != nil {
		return fmt.Errorf("export order CSV failed: parse template mapping rules: %w", err)
	}
	if len(cfg.Headers) == 0 {
		return fmt.Errorf("export order CSV failed: template headers are empty")
	}
	if cfg.Prefix == "" {
		cfg.Prefix = template.Platform + "-"
	}

	// 1. Precheck: abort if any record still has NULL member_address_id.
	// Only check records whose product platform matches the export template platform.
	var count int64
	if err := db.Model(&model.DispatchRecord{}).
		Where("wave_id = ? AND member_address_id IS NULL AND product_id IN (SELECT id FROM products WHERE platform = ?)", waveID, template.Platform).
		Count(&count).Error; err != nil {
		return fmt.Errorf("export order CSV failed: precheck error: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("export aborted: %d records in wave %d have no address", count, waveID)
	}

	// 2. Load records filtered by product platform (not member platform).
	// A wave may contain products from multiple factory platforms;
	// each export run only includes records whose product platform
	// matches the export template platform.
	var records []model.DispatchRecord
	if err := db.
		Preload("MemberAddress").
		Preload("Product").
		Preload("Member").
		Where("wave_id = ? AND product_id IN (SELECT id FROM products WHERE platform = ?)", waveID, template.Platform).
		Find(&records).Error; err != nil {
		return fmt.Errorf("export order CSV failed: query error: %w", err)
	}
	if len(records) == 0 {
		return fmt.Errorf("export order CSV failed: wave %d has no dispatch records", waveID)
	}

	// 3. Write CSV with UTF-8 BOM for Excel compatibility.
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("export order CSV failed: cannot create file %q: %w", outputPath, err)
	}
	defer file.Close()

	// Write BOM.
	if _, err := file.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return fmt.Errorf("export order CSV failed: write BOM: %w", err)
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write(cfg.Headers); err != nil {
		return fmt.Errorf("export order CSV failed: write header: %w", err)
	}

	// Validate header column count matches expected row layout
	expectedCols := 6
	if cfg.BlankLeadingColumn {
		expectedCols = 7
	}
	if len(cfg.Headers) != expectedCols {
		return fmt.Errorf("export template header count mismatch: got %d headers, expected %d (blankLeadingColumn=%v)", len(cfg.Headers), expectedCols, cfg.BlankLeadingColumn)
	}

	for _, record := range records {
		if record.MemberAddress == nil {
			return fmt.Errorf("export order CSV failed: record %d has nil MemberAddress (precheck should have caught this)", record.ID)
		}

		recipient := record.MemberAddress.RecipientName
		phone := record.MemberAddress.Phone
		address := record.MemberAddress.Address
		sku := record.Product.FactorySKU

		// Validate required fields per row.
		if recipient == "" {
			return fmt.Errorf("export order CSV failed: record %d has empty recipient name", record.ID)
		}
		if phone == "" {
			return fmt.Errorf("export order CSV failed: record %d has empty phone", record.ID)
		}
		if address == "" {
			return fmt.Errorf("export order CSV failed: record %d has empty address", record.ID)
		}
		if sku == "" {
			return fmt.Errorf("export order CSV failed: record %d has empty factory SKU", record.ID)
		}

		orderNo := cfg.Prefix + strconv.Itoa(int(record.ID))
		var row []string
		if cfg.BlankLeadingColumn {
			row = []string{"", orderNo, recipient, phone, address, sku, strconv.Itoa(record.Quantity)}
		} else {
			row = []string{orderNo, recipient, phone, address, sku, strconv.Itoa(record.Quantity)}
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("export order CSV failed: write row %d: %w", record.ID, err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("export order CSV failed: flush error: %w", err)
	}

	// 4. Update wave status to "exported".
	if err := db.Model(&model.Wave{}).Where("id = ?", waveID).Update("status", "exported").Error; err != nil {
		return fmt.Errorf("export order CSV failed: update wave status: %w", err)
	}

	return nil
}

// ExportWavePreview returns the total record count and the number of records
// that are still missing an address for the given wave. It does not write any file.
func ExportWavePreview(db *gorm.DB, waveID uint) (totalRecords int, missingAddressCount int, err error) {
	if db == nil {
		return 0, 0, fmt.Errorf("export preview failed: database connection is required")
	}
	if waveID == 0 {
		return 0, 0, fmt.Errorf("export preview failed: wave ID is required")
	}

	var total int64
	if err := db.Model(&model.DispatchRecord{}).
		Where("wave_id = ?", waveID).
		Count(&total).Error; err != nil {
		return 0, 0, fmt.Errorf("export preview failed: count total: %w", err)
	}

	var missing int64
	if err := db.Model(&model.DispatchRecord{}).
		Where("wave_id = ? AND member_address_id IS NULL", waveID).
		Count(&missing).Error; err != nil {
		return 0, 0, fmt.Errorf("export preview failed: count missing addresses: %w", err)
	}

	return int(total), int(missing), nil
}
