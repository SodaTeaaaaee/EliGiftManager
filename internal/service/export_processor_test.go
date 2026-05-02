package service

import (
	"bufio"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
)

// TestExportFilterByProductPlatform verifies that ExportOrderCSV only includes
// dispatch records whose product platform matches the export template platform,
// even when the same wave contains products from multiple factories.
func TestExportFilterByProductPlatform(t *testing.T) {
	t.Parallel()
	db := newServiceTestDB(t)

	// -- Seed data --
	wave := model.Wave{WaveNo: "PLATFORM-FILTER-001", Name: "multi-platform wave", Status: "draft"}
	productBili := model.Product{Platform: "BILIBILI", Factory: "factory-A", FactorySKU: "BILI-001", Name: "B站挂件", ExtraData: "{}"}
	productRouzo := model.Product{Platform: "柔造", Factory: "factory-B", FactorySKU: "ROUZO-001", Name: "柔造立牌", ExtraData: "{}"}
	member1 := model.Member{Platform: "BILIBILI", PlatformUID: "uid-bili-01", ExtraData: "{}"}
	member2 := model.Member{Platform: "柔造", PlatformUID: "uid-rouzo-01", ExtraData: "{}"}

	for _, r := range []any{&wave, &productBili, &productRouzo, &member1, &member2} {
		if err := db.Create(r).Error; err != nil {
			t.Fatalf("seed failed: %v", err)
		}
	}

	// Give each record an address so the precheck passes.
	addr1 := model.MemberAddress{MemberID: member1.ID, RecipientName: "Alice", Phone: "13800001111", Address: "Shanghai"}
	addr2 := model.MemberAddress{MemberID: member2.ID, RecipientName: "Bob", Phone: "13800002222", Address: "Beijing"}
	for _, a := range []*model.MemberAddress{&addr1, &addr2} {
		if err := db.Create(a).Error; err != nil {
			t.Fatalf("seed address failed: %v", err)
		}
	}

	dispatchBili := model.DispatchRecord{WaveID: wave.ID, MemberID: member1.ID, ProductID: productBili.ID, MemberAddressID: &addr1.ID, Quantity: 2, Status: model.DispatchStatusPending}
	dispatchRouzo := model.DispatchRecord{WaveID: wave.ID, MemberID: member2.ID, ProductID: productRouzo.ID, MemberAddressID: &addr2.ID, Quantity: 3, Status: model.DispatchStatusPending}
	for _, d := range []*model.DispatchRecord{&dispatchBili, &dispatchRouzo} {
		if err := db.Create(d).Error; err != nil {
			t.Fatalf("seed dispatch record failed: %v", err)
		}
	}

	// -- Export with BILIBILI template --
	biliTemplate := model.TemplateConfig{
		Platform:     "BILIBILI",
		Type:         model.TemplateTypeExportOrder,
		Name:         "B站发货单模板",
		MappingRules: `{"headers":["订单号","收件人","电话","地址","SKU","数量"],"prefix":"BILI-"}`,
	}
	if err := db.Create(&biliTemplate).Error; err != nil {
		t.Fatalf("seed BILIBILI template failed: %v", err)
	}

	tempDir := t.TempDir()
	biliOutputPath := filepath.Join(tempDir, "bili-export.csv")
	if err := ExportOrderCSV(db, wave.ID, biliOutputPath, biliTemplate); err != nil {
		t.Fatalf("ExportOrderCSV (BILIBILI) failed: %v", err)
	}

	// Read back CSV and verify only BILIBILI product record is present.
	biliLines := readCSVRecords(t, biliOutputPath)
	// 1 header line + 1 data line = 2 lines
	if len(biliLines) != 2 {
		t.Fatalf("BILIBILI export: expected 2 CSV lines (header + 1 record), got %d. Lines: %v", len(biliLines), biliLines)
	}
	// Verify the exported SKU belongs to the BILIBILI product.
	if len(biliLines) > 1 {
		row := biliLines[1]
		if len(row) < 5 {
			t.Fatalf("BILIBILI export: row has %d fields, expected >=5", len(row))
		}
		if row[4] != "BILI-001" {
			t.Fatalf("BILIBILI export: expected SKU BILI-001, got %q", row[4])
		}
	}

	// -- Export with 柔造 template (same wave, different platform) --
	rouzoTemplate := model.TemplateConfig{
		Platform:     "柔造",
		Type:         model.TemplateTypeExportOrder,
		Name:         "柔造发货单模板",
		MappingRules: `{"headers":["订单号","收件人","电话","地址","SKU","数量"],"prefix":"ROUZO-"}`,
	}
	if err := db.Create(&rouzoTemplate).Error; err != nil {
		t.Fatalf("seed 柔造 template failed: %v", err)
	}

	// Reset wave status back to draft so the second export can proceed.
	if err := db.Model(&model.Wave{}).Where("id = ?", wave.ID).Update("status", "draft").Error; err != nil {
		t.Fatalf("reset wave status failed: %v", err)
	}

	rouzoOutputPath := filepath.Join(tempDir, "rouzo-export.csv")
	if err := ExportOrderCSV(db, wave.ID, rouzoOutputPath, rouzoTemplate); err != nil {
		t.Fatalf("ExportOrderCSV (柔造) failed: %v", err)
	}

	rouzoLines := readCSVRecords(t, rouzoOutputPath)
	if len(rouzoLines) != 2 {
		t.Fatalf("柔造 export: expected 2 CSV lines, got %d", len(rouzoLines))
	}
	if len(rouzoLines) > 1 {
		row := rouzoLines[1]
		if row[4] != "ROUZO-001" {
			t.Fatalf("柔造 export: expected SKU ROUZO-001, got %q", row[4])
		}
	}
}

// TestExportOrderCSVErrorsOnEmptyPlatformFilter verifies that when no dispatch
// records match the template platform, ExportOrderCSV returns a clear error
// even though the wave has records from other platforms.
func TestExportOrderCSVErrorsOnEmptyPlatformFilter(t *testing.T) {
	t.Parallel()
	db := newServiceTestDB(t)

	wave := model.Wave{WaveNo: "EMPTY-FILTER-001", Name: "single-platform wave", Status: "draft"}
	productBili := model.Product{Platform: "BILIBILI", Factory: "factory-A", FactorySKU: "BILI-001", Name: "B站挂件", ExtraData: "{}"}
	member1 := model.Member{Platform: "BILIBILI", PlatformUID: "uid-bili-01", ExtraData: "{}"}
	for _, r := range []any{&wave, &productBili, &member1} {
		if err := db.Create(r).Error; err != nil {
			t.Fatalf("seed failed: %v", err)
		}
	}
	addr := model.MemberAddress{MemberID: member1.ID, RecipientName: "Alice", Phone: "13800001111", Address: "Shanghai"}
	if err := db.Create(&addr).Error; err != nil {
		t.Fatalf("seed address failed: %v", err)
	}
	dispatch := model.DispatchRecord{WaveID: wave.ID, MemberID: member1.ID, ProductID: productBili.ID, MemberAddressID: &addr.ID, Quantity: 1, Status: model.DispatchStatusPending}
	if err := db.Create(&dispatch).Error; err != nil {
		t.Fatalf("seed dispatch record failed: %v", err)
	}

	// Export with a template whose platform does NOT match any product in the wave.
	rouzoTemplate := model.TemplateConfig{
		Platform:     "柔造",
		Type:         model.TemplateTypeExportOrder,
		Name:         "柔造发货单模板",
		MappingRules: `{"headers":["订单号","收件人","电话","地址","SKU","数量"],"prefix":"ROUZO-"}`,
	}
	if err := db.Create(&rouzoTemplate).Error; err != nil {
		t.Fatalf("seed template failed: %v", err)
	}

	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "empty-export.csv")
	err := ExportOrderCSV(db, wave.ID, outputPath, rouzoTemplate)
	if err == nil {
		t.Fatal("expected error when no records match platform filter, got nil")
	}
	if !strings.Contains(err.Error(), "has no dispatch records") {
		t.Fatalf("expected 'has no dispatch records' error, got: %v", err)
	}
}

// TestWaveIsolation verifies that the same global member can participate in
// multiple waves with different gift levels (quantities) without interference.
func TestWaveIsolation(t *testing.T) {
	t.Parallel()
	db := newServiceTestDB(t)

	// Shared member and product.
	member := model.Member{Platform: "BILIBILI", PlatformUID: "uid-shared-01", ExtraData: "{}"}
	product := model.Product{Platform: "BILIBILI", Factory: "factory-A", FactorySKU: "SHARED-001", Name: "通用挂件", ExtraData: "{}"}
	for _, r := range []any{&member, &product} {
		if err := db.Create(r).Error; err != nil {
			t.Fatalf("seed failed: %v", err)
		}
	}

	// Wave A: higher gift level.
	waveA := model.Wave{WaveNo: "ISO-A-001", Name: "Wave A - 提督级", Status: "draft"}
	if err := db.Create(&waveA).Error; err != nil {
		t.Fatalf("seed wave A failed: %v", err)
	}
	wm := model.WaveMember{WaveID: waveA.ID, MemberID: member.ID}
	if err := db.Create(&wm).Error; err != nil {
		t.Fatalf("seed wave_member A failed: %v", err)
	}
	addrA := model.MemberAddress{MemberID: member.ID, RecipientName: "Alice", Phone: "13800001111", Address: "Addr-A"}
	if err := db.Create(&addrA).Error; err != nil {
		t.Fatalf("seed address A failed: %v", err)
	}
	drA := model.DispatchRecord{WaveID: waveA.ID, MemberID: member.ID, ProductID: product.ID, MemberAddressID: &addrA.ID, Quantity: 5, Status: model.DispatchStatusPending}
	if err := db.Create(&drA).Error; err != nil {
		t.Fatalf("seed dispatch record A failed: %v", err)
	}

	// Wave B: lower gift level, same member.
	waveB := model.Wave{WaveNo: "ISO-B-001", Name: "Wave B - 舰长级", Status: "draft"}
	if err := db.Create(&waveB).Error; err != nil {
		t.Fatalf("seed wave B failed: %v", err)
	}
	wmB := model.WaveMember{WaveID: waveB.ID, MemberID: member.ID}
	if err := db.Create(&wmB).Error; err != nil {
		t.Fatalf("seed wave_member B failed: %v", err)
	}
	addrB := model.MemberAddress{MemberID: member.ID, RecipientName: "Alice", Phone: "13800001111", Address: "Addr-B"}
	if err := db.Create(&addrB).Error; err != nil {
		t.Fatalf("seed address B failed: %v", err)
	}
	drB := model.DispatchRecord{WaveID: waveB.ID, MemberID: member.ID, ProductID: product.ID, MemberAddressID: &addrB.ID, Quantity: 2, Status: model.DispatchStatusPending}
	if err := db.Create(&drB).Error; err != nil {
		t.Fatalf("seed dispatch record B failed: %v", err)
	}

	// Verify Wave A records.
	var recordsA []model.DispatchRecord
	if err := db.Where("wave_id = ?", waveA.ID).Find(&recordsA).Error; err != nil {
		t.Fatalf("query wave A records failed: %v", err)
	}
	if len(recordsA) != 1 {
		t.Fatalf("wave A: expected 1 record, got %d", len(recordsA))
	}
	if recordsA[0].Quantity != 5 {
		t.Fatalf("wave A: expected quantity 5, got %d", recordsA[0].Quantity)
	}

	// Verify Wave B records.
	var recordsB []model.DispatchRecord
	if err := db.Where("wave_id = ?", waveB.ID).Find(&recordsB).Error; err != nil {
		t.Fatalf("query wave B records failed: %v", err)
	}
	if len(recordsB) != 1 {
		t.Fatalf("wave B: expected 1 record, got %d", len(recordsB))
	}
	if recordsB[0].Quantity != 2 {
		t.Fatalf("wave B: expected quantity 2, got %d", recordsB[0].Quantity)
	}

	// Verify wave_members are independent.
	var wmCount int64
	db.Model(&model.WaveMember{}).Where("wave_id = ?", waveA.ID).Count(&wmCount)
	if wmCount != 1 {
		t.Fatalf("wave A: expected 1 wave_member, got %d", wmCount)
	}
	db.Model(&model.WaveMember{}).Where("wave_id = ?", waveB.ID).Count(&wmCount)
	if wmCount != 1 {
		t.Fatalf("wave B: expected 1 wave_member, got %d", wmCount)
	}

	// Wave A export should succeed with only its own record.
	templateA := model.TemplateConfig{
		Platform:     "BILIBILI",
		Type:         model.TemplateTypeExportOrder,
		Name:         "B站发货单模板",
		MappingRules: `{"headers":["订单号","收件人","电话","地址","SKU","数量"],"prefix":"BILI-"}`,
	}
	if err := db.Create(&templateA).Error; err != nil {
		t.Fatalf("seed template failed: %v", err)
	}
	tempDir := t.TempDir()
	outA := filepath.Join(tempDir, "iso-a.csv")
	if err := ExportOrderCSV(db, waveA.ID, outA, templateA); err != nil {
		t.Fatalf("ExportOrderCSV wave A failed: %v", err)
	}
	aLines := readCSVRecords(t, outA)
	if len(aLines) != 2 {
		t.Fatalf("wave A export: expected 2 lines, got %d", len(aLines))
	}
	// Verify the quantity is 5 (from the A wave, not mixed with B's 2).
	if len(aLines) > 1 {
		row := aLines[1]
		if row[5] != "5" {
			t.Fatalf("wave A export: expected quantity 5, got %q", row[5])
		}
	}
}

// TestWaveCascadeDelete verifies that deleting a wave cascades to its
// WaveMembers and DispatchRecords. Also verifies that user-level ProductTags
// (TagType="user") are NOT removed by wave deletion (they belong to products,
// not waves — application-level cleanup is handled by the controller).
func TestWaveCascadeDelete(t *testing.T) {
	t.Parallel()
	db := newServiceTestDB(t)

	// -- Seed: Wave, Member, WaveMember, Product, DispatchRecord, user ProductTag --
	wave := model.Wave{WaveNo: "CASCADE-001", Name: "cascade test wave", Status: "draft"}
	member := model.Member{Platform: "BILIBILI", PlatformUID: "uid-cascade-01", ExtraData: "{}"}
	product := model.Product{Platform: "BILIBILI", Factory: "factory-A", FactorySKU: "CAS-001", Name: "cascade产品", ExtraData: "{}"}
	for _, r := range []any{&wave, &member, &product} {
		if err := db.Create(r).Error; err != nil {
			t.Fatalf("seed failed: %v", err)
		}
	}

	wm := model.WaveMember{WaveID: wave.ID, MemberID: member.ID}
	if err := db.Create(&wm).Error; err != nil {
		t.Fatalf("seed wave_member failed: %v", err)
	}
	addr := model.MemberAddress{MemberID: member.ID, RecipientName: "Alice", Phone: "13800001111", Address: "CascadeAddr"}
	if err := db.Create(&addr).Error; err != nil {
		t.Fatalf("seed address failed: %v", err)
	}
	dr := model.DispatchRecord{WaveID: wave.ID, MemberID: member.ID, ProductID: product.ID, MemberAddressID: &addr.ID, Quantity: 1, Status: model.DispatchStatusPending}
	if err := db.Create(&dr).Error; err != nil {
		t.Fatalf("seed dispatch record failed: %v", err)
	}

	// Create a user tag (ProductTag with TagType="user") — this is
	// application-level metadata tied to a product, not a wave.
	userTag := model.ProductTag{ProductID: product.ID, Platform: member.Platform, TagName: member.PlatformUID, TagType: "user", Quantity: 1}
	if err := db.Create(&userTag).Error; err != nil {
		t.Fatalf("seed user tag failed: %v", err)
	}

	// -- Act: Delete the wave --
	if err := db.Delete(&wave).Error; err != nil {
		t.Fatalf("delete wave failed: %v", err)
	}

	// -- Assert: WaveMembers CASCADE deleted --
	var wmCount int64
	db.Model(&model.WaveMember{}).Where("wave_id = ?", wave.ID).Count(&wmCount)
	if wmCount != 0 {
		t.Fatalf("expected 0 wave_members after wave delete, got %d", wmCount)
	}

	// -- Assert: DispatchRecords CASCADE deleted --
	var drCount int64
	db.Model(&model.DispatchRecord{}).Where("wave_id = ?", wave.ID).Count(&drCount)
	if drCount != 0 {
		t.Fatalf("expected 0 dispatch_records after wave delete, got %d", drCount)
	}

	// -- Assert: User tag still exists (tied to Product, not Wave) --
	var tagCount int64
	db.Model(&model.ProductTag{}).Where("id = ?", userTag.ID).Count(&tagCount)
	if tagCount != 1 {
		t.Fatalf("expected user tag to survive wave deletion (it belongs to the product), got count %d", tagCount)
	}

	// -- Assert: Product still exists --
	var prodCount int64
	db.Model(&model.Product{}).Where("id = ?", product.ID).Count(&prodCount)
	if prodCount != 1 {
		t.Fatalf("expected product to survive wave deletion, got count %d", prodCount)
	}

	// -- Assert: Member still exists --
	var memCount int64
	db.Model(&model.Member{}).Where("id = ?", member.ID).Count(&memCount)
	if memCount != 1 {
		t.Fatalf("expected member to survive wave deletion, got count %d", memCount)
	}
}

// TestExportWavePreviewUnchanged verifies ExportWavePreview still returns
// total wave counts (not platform-filtered) per the intentional design
// decision to leave the preview signature unchanged.
func TestExportWavePreviewUnchanged(t *testing.T) {
	t.Parallel()
	db := newServiceTestDB(t)

	wave := model.Wave{WaveNo: "PREVIEW-001", Name: "preview test", Status: "draft"}
	product := model.Product{Platform: "BILIBILI", Factory: "f", FactorySKU: "SKU", Name: "X", ExtraData: "{}"}
	member := model.Member{Platform: "BILIBILI", PlatformUID: "uid-pv", ExtraData: "{}"}
	for _, r := range []any{&wave, &product, &member} {
		if err := db.Create(r).Error; err != nil {
			t.Fatalf("seed failed: %v", err)
		}
	}
	addr := model.MemberAddress{MemberID: member.ID, RecipientName: "X", Phone: "1", Address: "X"}
	if err := db.Create(&addr).Error; err != nil {
		t.Fatalf("seed address: %v", err)
	}
	member2 := model.Member{Platform: "BILIBILI", PlatformUID: "uid-pv2", ExtraData: "{}"}
	if err := db.Create(&member2).Error; err != nil {
		t.Fatalf("seed member2: %v", err)
	}
	dr1 := model.DispatchRecord{WaveID: wave.ID, MemberID: member.ID, ProductID: product.ID, MemberAddressID: &addr.ID, Quantity: 1, Status: model.DispatchStatusPending}
	dr2 := model.DispatchRecord{WaveID: wave.ID, MemberID: member2.ID, ProductID: product.ID, MemberAddressID: nil, Quantity: 1, Status: model.DispatchStatusPendingAddress}
	for _, d := range []*model.DispatchRecord{&dr1, &dr2} {
		if err := db.Create(d).Error; err != nil {
			t.Fatalf("seed dispatch: %v", err)
		}
	}

	total, missing, err := ExportWavePreview(db, wave.ID)
	if err != nil {
		t.Fatalf("ExportWavePreview: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2, got %d", total)
	}
	if missing != 1 {
		t.Fatalf("expected 1 missing address, got %d", missing)
	}
}

// --- helpers ---

// readCSVRecords opens a CSV file and returns all records (including header).
func readCSVRecords(t *testing.T, path string) [][]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open CSV %q: %v", path, err)
	}
	defer f.Close()
	// Skip UTF-8 BOM if present.
	reader := bufio.NewReader(f)
	bom := []byte{0xEF, 0xBB, 0xBF}
	peek, _ := reader.Peek(3)
	if len(peek) >= 3 && peek[0] == bom[0] && peek[1] == bom[1] && peek[2] == bom[2] {
		_, _ = reader.Discard(3)
	}
	csvReader := csv.NewReader(reader)
	records, err := csvReader.ReadAll()
	if err != nil {
		t.Fatalf("read CSV %q: %v", path, err)
	}
	return records
}
