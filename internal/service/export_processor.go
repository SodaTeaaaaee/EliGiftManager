package service

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/gorm"
)

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

	var rules model.DynamicExportRules
	if err := json.Unmarshal([]byte(template.MappingRules), &rules); err != nil {
		return fmt.Errorf("export order CSV failed: parse template mapping rules: %w", err)
	}
	if len(rules.Columns) == 0 {
		return fmt.Errorf("export order CSV failed: template has no columns defined")
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
		Preload("Member.Nicknames", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC")
		}).
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

	if rules.HasHeader {
		headers := make([]string, len(rules.Columns))
		for i, col := range rules.Columns {
			headers[i] = col.HeaderName
		}
		if err := writer.Write(headers); err != nil {
			return fmt.Errorf("export order CSV failed: write header: %w", err)
		}
	}

	for _, record := range records {
		if record.MemberAddress == nil {
			return fmt.Errorf("export order CSV failed: record %d has nil MemberAddress (precheck should have caught this)", record.ID)
		}

		row := make([]string, len(rules.Columns))
		for i, col := range rules.Columns {
			var val string
			switch col.ValueType {
			case "order_no":
				val = col.Prefix + strconv.Itoa(int(record.ID))
			case "recipient":
				if record.MemberAddress != nil {
					val = record.MemberAddress.RecipientName
				}
			case "phone":
				if record.MemberAddress != nil {
					val = record.MemberAddress.Phone
				}
			case "address":
				if record.MemberAddress != nil {
					val = record.MemberAddress.Address
				}
			case "sku":
				val = record.Product.FactorySKU
			case "quantity":
				val = strconv.Itoa(record.Quantity)
			case "member_uid":
				val = record.Member.PlatformUID
			case "member_nickname":
				val = record.Member.PlatformUID
				if len(record.Member.Nicknames) > 0 {
					val = record.Member.Nicknames[0].Nickname
				} else {
					log.Printf("[WARNING] ExportOrderCSV: member %d has no nicknames, fallback to PlatformUID", record.MemberID)
				}
			case "static":
				val = col.DefaultValue
			case "empty":
				val = ""
			default:
				val = ""
			}
			row[i] = val
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
	if err := db.Model(&model.Wave{}).Where("id = ?", waveID).Update("status", model.WaveStatusExported).Error; err != nil {
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
