package controller

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type scenarioAState struct {
	gdb                 *gorm.DB
	profileID           uint
	factoryProfileID    uint
	waveID              uint
	secondWaveID        uint
	documentIDs         []uint
	goldProductID       uint
	silverProductID     uint
	fulfillmentLines    []persistence.FulfillmentLine
	supplierOrder       persistence.SupplierOrder
	supplierOrderLines  []persistence.SupplierOrderLine
	shipments           []dto.ShipmentDTO
	supplierOrderOutput string
	syncOutput          string
}

func setupScenarioATestDB(t *testing.T) *gorm.DB {
	t.Helper()
	previous := db.GetDB()
	dsn := fmt.Sprintf("file:scenario_a_%d?mode=memory&cache=shared", time.Now().UnixNano())
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("scenario A setup: open in-memory SQLite: %v", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("scenario A setup: get sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	if err := gdb.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatalf("scenario A setup: enable foreign keys: %v", err)
	}
	if err := gdb.AutoMigrate(
		&persistence.CustomerProfile{},
		&persistence.CustomerMergeRecord{},
		&persistence.CustomerIdentity{},
		&persistence.CustomerAddress{},
		&persistence.CustomerNameObservation{},
		&persistence.CustomerNameEvent{},
		&persistence.CustomerProfileOrigin{},
		&persistence.DemandDocument{},
		&persistence.DemandLine{},
		&persistence.Wave{},
		&persistence.WaveParticipantSnapshot{},
		&persistence.FulfillmentLine{},
		&persistence.AllocationPolicyRule{},
		&persistence.SupplierOrder{},
		&persistence.SupplierOrderLine{},
		&persistence.WaveDemandAssignment{},
		&persistence.Shipment{},
		&persistence.ShipmentLine{},
		&persistence.ChannelSyncJob{},
		&persistence.ChannelSyncItem{},
		&persistence.IntegrationProfile{},
		&persistence.ChannelClosureDecisionRecord{},
		&persistence.FulfillmentAdjustment{},
		&persistence.DocumentTemplate{},
		&persistence.IntegrationProfileTemplateBinding{},
		&persistence.HistoryScope{},
		&persistence.HistoryNode{},
		&persistence.HistoryCheckpoint{},
		&persistence.HistoryPin{},
		&persistence.ProductMaster{},
		&persistence.Product{},
		&persistence.CarrierMapping{},
		&persistence.MergeSuggestion{},
	); err != nil {
		t.Fatalf("scenario A setup: complete AutoMigrate: %v", err)
	}
	seedEnabledFeaturePolicy(t, gdb)
	if err := gdb.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_binding_one_default
		ON integration_profile_template_bindings (integration_profile_id, document_type)
		WHERE is_default = true`).Error; err != nil {
		t.Fatalf("scenario A setup: create default-binding index: %v", err)
	}
	db.SetDefaultDB(gdb)
	t.Cleanup(func() {
		db.SetDefaultDB(previous)
		_ = sqlDB.Close()
	})
	return gdb
}

func scenarioARequireCount(t *testing.T, gdb *gorm.DB, model any, query string, args []any, want int64) {
	t.Helper()
	var got int64
	if err := gdb.Model(model).Where(query, args...).Count(&got).Error; err != nil {
		t.Fatalf("count %T: %v", model, err)
	}
	if got != want {
		t.Fatalf("count %T = %d, want %d", model, got, want)
	}
}

func scenarioAOverview(t *testing.T, waveID uint) dto.WaveOverviewDTO {
	t.Helper()
	overview, err := NewWaveController().GetWaveOverview(waveID)
	if err != nil {
		t.Fatalf("GetWaveOverview(%d): %v", waveID, err)
	}
	return overview
}

func scenarioALines(t *testing.T, gdb *gorm.DB, waveID uint) []persistence.FulfillmentLine {
	t.Helper()
	var lines []persistence.FulfillmentLine
	if err := gdb.Where("wave_id = ?", waveID).Order("id").Find(&lines).Error; err != nil {
		t.Fatalf("list fulfillment lines for wave %d: %v", waveID, err)
	}
	return lines
}

func scenarioAAssertProductQuantities(t *testing.T, lines []persistence.FulfillmentLine, want map[uint]int) {
	t.Helper()
	got := make(map[uint]int)
	for _, line := range lines {
		if line.ProductID == nil {
			t.Fatalf("fulfillment line %d has no product", line.ID)
		}
		got[*line.ProductID] += line.Quantity
	}
	if len(got) != len(want) {
		t.Fatalf("product quantity groups = %v, want %v", got, want)
	}
	for productID, quantity := range want {
		if got[productID] != quantity {
			t.Fatalf("product %d quantity = %d, want %d (all quantities: %v)", productID, got[productID], quantity, got)
		}
	}
}

func scenarioARenderFactoryReturn(t *testing.T, lines []persistence.SupplierOrderLine) string {
	t.Helper()
	raw, err := os.ReadFile(testdataPath(t, "scenario_a", "factory_return.csv.tmpl"))
	if err != nil {
		t.Fatalf("read factory return template: %v", err)
	}
	rendered := string(raw)
	for i, line := range lines {
		rendered = strings.ReplaceAll(rendered, fmt.Sprintf("{{SOL_%d}}", i+1), strconv.FormatUint(uint64(line.ID), 10))
		rendered = strings.ReplaceAll(rendered, fmt.Sprintf("{{FL_%d}}", i+1), strconv.FormatUint(uint64(line.FulfillmentLineID), 10))
	}
	if strings.Contains(rendered, "{{") {
		t.Fatalf("factory return template has unresolved placeholders: %s", rendered)
	}
	path := filepath.Join(t.TempDir(), "factory_return.csv")
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		t.Fatalf("write rendered factory return: %v", err)
	}
	return path
}

func scenarioAParseFactoryReturn(t *testing.T, path string) []dto.ImportShipmentEntry {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open rendered factory return: %v", err)
	}
	defer f.Close()
	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("parse rendered factory return: %v", err)
	}
	if len(records) != 7 {
		t.Fatalf("factory return record count = %d, want header + 6 rows", len(records))
	}
	entries := make([]dto.ImportShipmentEntry, 0, 6)
	for i, row := range records[1:] {
		if len(row) != 8 {
			t.Fatalf("factory return row %d has %d columns, want 8", i+1, len(row))
		}
		solID, err := strconv.ParseUint(row[0], 10, 64)
		if err != nil {
			t.Fatalf("factory return row %d supplierOrderLineId: %v", i+1, err)
		}
		flID, err := strconv.ParseUint(row[1], 10, 64)
		if err != nil {
			t.Fatalf("factory return row %d fulfillmentLineId: %v", i+1, err)
		}
		quantity, err := strconv.Atoi(row[6])
		if err != nil {
			t.Fatalf("factory return row %d quantity: %v", i+1, err)
		}
		shippedAt, err := time.Parse(time.RFC3339, row[7])
		if err != nil {
			t.Fatalf("factory return row %d shippedAt: %v", i+1, err)
		}
		entries = append(entries, dto.ImportShipmentEntry{
			SupplierOrderLineID: uint(solID), FulfillmentLineID: uint(flID),
			ExternalShipmentNo: row[2], CarrierCode: row[3], CarrierName: row[4],
			TrackingNo: row[5], Quantity: quantity, ShippedAt: &shippedAt,
		})
	}
	return entries
}

func TestScenarioA_MonthlyMemberWave(t *testing.T) {
	s := &scenarioAState{gdb: setupScenarioATestDB(t)}

	if !t.Run("01 import patreon members", func(t *testing.T) {
		profileRepo := infra.NewIntegrationProfileRepository(s.gdb)
		profile := &domain.IntegrationProfile{
			ProfileKey: "scenario-a-patreon", SourceChannel: "patreon", SourceSurface: "membership",
			DemandKind: "membership_entitlement", InitialAllocationStrategy: "policy_driven",
			IdentityStrategy: "platform_uid", EntitlementAuthorityMode: "upstream_authoritative",
			RecipientInputMode: "per_recipient", ReferenceStrategy: "member_level",
			TrackingSyncMode: "document_export", ClosurePolicy: "close_after_manual_confirmation",
			ConnectorKey: "eli.local_export",
		}
		if err := profileRepo.Create(appContext, profile); err != nil {
			t.Fatalf("create Patreon integration profile: %v", err)
		}
		s.profileID = profile.ID
		templateRepo := infra.NewDocumentTemplateRepository(s.gdb)
		tmpl := &domain.DocumentTemplate{
			TemplateKey: "scenario-a-patreon-import", DocumentType: "import_entitlement", Format: "csv",
			MappingRules: `{"columns":{"gift_level_snapshot":"Current Tier","entitlement_code":"Entitlement Code","requested_quantity":"Quantity","recipient_input_state":"Recipient Input State"},"defaults":{"line_type":"entitlement_rule","obligation_trigger_kind":"periodic_membership","entitlement_authority":"upstream_platform","routing_disposition":"accepted"}}`,
		}
		if err := templateRepo.Create(appContext, tmpl); err != nil {
			t.Fatalf("create import_entitlement template: %v", err)
		}
		bindingRepo := infra.NewProfileTemplateBindingRepository(s.gdb)
		if err := bindingRepo.Create(appContext, &domain.IntegrationProfileTemplateBinding{
			IntegrationProfileID: profile.ID, DocumentType: "import_entitlement", TemplateID: tmpl.ID, IsDefault: true,
		}); err != nil {
			t.Fatalf("bind default import_entitlement template: %v", err)
		}

		demandController := NewDemandController()
		preview, err := demandController.ParseCSVFile(testdataPath(t, "scenario_a", "patreon_members.csv"))
		if err != nil {
			t.Fatalf("ParseCSVFile: %v", err)
		}
		if len(preview.Rows) != 6 {
			t.Fatalf("parsed Patreon rows = %d, want 6", len(preview.Rows))
		}
		// Production gap: bulk multi-row CSV import creates one document-level customer,
		// so it cannot create a distinct customer per Patreon row. This per-row adapter
		// deliberately calls ImportDemandCSV once per row until production supports that.
		for i, row := range preview.Rows {
			result, err := demandController.ImportDemandCSV(dto.ImportDemandCSVInput{
				IntegrationProfileID: profile.ID, DocumentType: "import_entitlement",
				SourceDocumentNo: fmt.Sprintf("PATREON-JULY-%02d", i+1), SourceCustomerRef: row["User ID"],
				ImportMode: "reject_all", Rows: []map[string]string{row},
			})
			if err != nil {
				t.Fatalf("ImportDemandCSV row %d: %v", i+1, err)
			}
			if result.Document == nil || result.SuccessCount != 1 || result.ErrorCount != 0 {
				t.Fatalf("ImportDemandCSV row %d result = %+v", i+1, result)
			}
			s.documentIDs = append(s.documentIDs, result.Document.ID)
		}

		wave, err := NewWaveController().CreateWave(dto.CreateWaveInput{Name: "July Patreon member wave", WaveType: "membership"})
		if err != nil {
			t.Fatalf("CreateWave: %v", err)
		}
		s.waveID = wave.ID
		assigned, err := NewWaveController().BatchAssignDemandToWave(dto.BatchAssignDemandInput{WaveID: wave.ID, DocIDs: s.documentIDs})
		if err != nil {
			t.Fatalf("BatchAssignDemandToWave: %v", err)
		}
		if assigned.SuccessCount != 6 || assigned.FailureCount != 0 {
			t.Fatalf("batch assignment result = %+v", assigned)
		}
		scenarioARequireCount(t, s.gdb, &persistence.DemandDocument{}, "1 = 1", nil, 6)
		scenarioARequireCount(t, s.gdb, &persistence.CustomerProfile{}, "1 = 1", nil, 6)
		scenarioARequireCount(t, s.gdb, &persistence.DemandLine{}, "routing_disposition = ? AND recipient_input_state = ?", []any{"accepted", "ready"}, 6)
		if got := scenarioAOverview(t, wave.ID); got.ProjectedLifecycleStage != "allocation" || got.DemandCount != 6 || got.AcceptedReadyOrNotRequired != 6 {
			t.Fatalf("post-intake overview = %+v", got)
		}

		pagination := NewListPaginationController()
		seen := make(map[uint]bool)
		for offset := 0; offset < 6; offset += 2 {
			page, err := pagination.ListDemandInboxRowsPage(dto.DemandInboxFilterInput{
				WaveID: &s.waveID, Assignment: "assigned", Limit: 2, Offset: offset, SortBy: "created_at", SortDir: "asc",
			})
			if err != nil {
				t.Fatalf("ListDemandInboxRowsPage offset %d: %v", offset, err)
			}
			if page.TotalCount != 6 || len(page.Items) != 2 {
				t.Fatalf("inbox page offset %d = total %d/items %d, want 6/2", offset, page.TotalCount, len(page.Items))
			}
			for _, item := range page.Items {
				if seen[item.DemandDocumentID] {
					t.Fatalf("duplicate demand document %d across pages", item.DemandDocumentID)
				}
				seen[item.DemandDocumentID] = true
				if item.AssignedWaveID == nil || *item.AssignedWaveID != s.waveID {
					t.Fatalf("demand document %d leaked from another wave: %+v", item.DemandDocumentID, item)
				}
			}
		}
		if len(seen) != 6 {
			t.Fatalf("unique paginated demand rows = %d, want 6", len(seen))
		}
	}) {
		t.FailNow()
	}

	if !t.Run("02 rule allocation", func(t *testing.T) {
		productController := NewProductController()
		goldMaster, err := productController.CreateProductMaster(dto.CreateProductMasterInput{SupplierPlatform: "factory-a", FactorySKU: "GOLD-JULY", Name: "Gold July gift", ProductKind: "badge"})
		if err != nil {
			t.Fatalf("create Gold product master: %v", err)
		}
		silverMaster, err := productController.CreateProductMaster(dto.CreateProductMasterInput{SupplierPlatform: "factory-a", FactorySKU: "SILVER-JULY", Name: "Silver July gift", ProductKind: "charm"})
		if err != nil {
			t.Fatalf("create Silver product master: %v", err)
		}
		products, err := productController.SnapshotProductsForWave(dto.SnapshotProductsInput{WaveID: s.waveID, MasterIDs: []uint{goldMaster.ID, silverMaster.ID}})
		if err != nil || len(products) != 2 {
			t.Fatalf("SnapshotProductsForWave = %+v, %v", products, err)
		}
		s.goldProductID, s.silverProductID = products[0].ID, products[1].ID
		generated, err := NewWaveController().GenerateParticipants(s.waveID)
		if err != nil || generated != 6 {
			t.Fatalf("GenerateParticipants = %d, %v; want 6", generated, err)
		}
		policy := NewAllocationPolicyController()
		for _, rule := range []dto.CreateAllocationPolicyRuleInput{
			{WaveID: s.waveID, ProductID: s.goldProductID, SelectorPayload: json.RawMessage(`{"type":"identity_level","platform":"patreon","level":"Gold"}`), ProductTargetRef: "GOLD-JULY", ContributionQuantity: 1, RuleKind: "entitlement", Priority: 10, Active: true},
			{WaveID: s.waveID, ProductID: s.silverProductID, SelectorPayload: json.RawMessage(`{"type":"identity_level","platform":"patreon","level":"Silver"}`), ProductTargetRef: "SILVER-JULY", ContributionQuantity: 1, RuleKind: "entitlement", Priority: 20, Active: true},
		} {
			if _, err := policy.CreateAllocationPolicyRule(rule); err != nil {
				t.Fatalf("CreateAllocationPolicyRule(%s): %v", rule.ProductTargetRef, err)
			}
		}
		reconciled, err := policy.ReconcileWave(s.waveID)
		if err != nil {
			t.Fatalf("ReconcileWave: %v", err)
		}
		if reconciled.Created != 6 || reconciled.ReplayedCount != 0 || len(reconciled.Failures) != 0 {
			t.Fatalf("initial reconcile result = %+v", reconciled)
		}
		scenarioARequireCount(t, s.gdb, &persistence.WaveParticipantSnapshot{}, "wave_id = ?", []any{s.waveID}, 6)
		s.fulfillmentLines = scenarioALines(t, s.gdb, s.waveID)
		if len(s.fulfillmentLines) != 6 {
			t.Fatalf("fulfillment line count = %d, want 6", len(s.fulfillmentLines))
		}
		scenarioAAssertProductQuantities(t, s.fulfillmentLines, map[uint]int{s.goldProductID: 3, s.silverProductID: 3})
		for _, line := range s.fulfillmentLines {
			if line.AllocationState != "ready" || line.AddressState != "missing" || line.SupplierState != "not_submitted" {
				t.Fatalf("unexpected initial fulfillment states on line %d: allocation=%q address=%q supplier=%q", line.ID, line.AllocationState, line.AddressState, line.SupplierState)
			}
		}
		if got := scenarioAOverview(t, s.waveID); got.ProjectedLifecycleStage != "review" {
			t.Fatalf("post-allocation stage = %q, want review", got.ProjectedLifecycleStage)
		}
	}) {
		t.FailNow()
	}

	if !t.Run("03 adjustments", func(t *testing.T) {
		var goldLine, silverLine persistence.FulfillmentLine
		for _, line := range s.fulfillmentLines {
			if *line.ProductID == s.goldProductID && goldLine.ID == 0 {
				goldLine = line
			}
			if *line.ProductID == s.silverProductID && silverLine.ID == 0 {
				silverLine = line
			}
		}
		entries := []dto.RecordAdjustmentInput{
			{WaveID: s.waveID, TargetKind: "fulfillment_line", FulfillmentLineID: &goldLine.ID, AdjustmentKind: "add", QuantityDelta: 1, ReasonCode: "survey_add", OperatorID: "scenario-a"},
			{WaveID: s.waveID, TargetKind: "fulfillment_line", FulfillmentLineID: &goldLine.ID, AdjustmentKind: "reduce", QuantityDelta: -1, ReasonCode: "survey_reduce", OperatorID: "scenario-a"},
			{WaveID: s.waveID, TargetKind: "fulfillment_line", FulfillmentLineID: &silverLine.ID, AdjustmentKind: "replace", FromProductID: &s.silverProductID, ToProductID: &s.goldProductID, ReasonCode: "survey_replace", OperatorID: "scenario-a"},
		}
		batch, err := NewAdjustmentController().BatchRecordAdjustments(dto.BatchRecordAdjustmentsInput{Entries: entries})
		if err != nil || batch.SuccessCount != 3 || batch.FailureCount != 0 {
			t.Fatalf("BatchRecordAdjustments = %+v, %v", batch, err)
		}
		reconciled, err := NewAllocationPolicyController().ReconcileWave(s.waveID)
		if err != nil {
			t.Fatalf("reconcile after adjustments: %v", err)
		}
		if reconciled.ReplayedCount != 3 || len(reconciled.Failures) != 0 {
			t.Fatalf("adjustment replay result = %+v", reconciled)
		}
		scenarioARequireCount(t, s.gdb, &persistence.FulfillmentAdjustment{}, "wave_id = ?", []any{s.waveID}, 3)
		s.fulfillmentLines = scenarioALines(t, s.gdb, s.waveID)
		if len(s.fulfillmentLines) != 6 {
			t.Fatalf("post-adjustment fulfillment lines = %d, want 6", len(s.fulfillmentLines))
		}
		scenarioAAssertProductQuantities(t, s.fulfillmentLines, map[uint]int{s.goldProductID: 4, s.silverProductID: 2})
	}) {
		t.FailNow()
	}

	if !t.Run("04 address tracking", func(t *testing.T) {
		addressController := NewAddressController()
		first := s.fulfillmentLines[0]
		invalid, err := addressController.CreateAddress(dto.CreateAddressInput{CustomerProfileID: *first.CustomerProfileID, Label: "shipping", RecipientName: "Scenario A member", Country: "US", AddressLine1: "1 Test Way", PostalCode: "00000", ValidationStatus: "invalid"})
		if err != nil {
			t.Fatalf("create invalid address: %v", err)
		}
		if _, err := addressController.BindAddressToLine(dto.BindAddressInput{FulfillmentLineID: first.ID, CustomerAddressID: invalid.ID}); err != nil {
			t.Fatalf("bind invalid address: %v", err)
		}
		var rebound persistence.FulfillmentLine
		if err := s.gdb.First(&rebound, first.ID).Error; err != nil || rebound.AddressState != "invalid" {
			t.Fatalf("bound invalid address state = %q, %v; want invalid", rebound.AddressState, err)
		}
		valid, err := addressController.UpdateAddress(dto.UpdateAddressInput{ID: invalid.ID, CustomerProfileID: *first.CustomerProfileID, Label: invalid.Label, RecipientName: invalid.RecipientName, Country: "US", City: "Seattle", Province: "WA", AddressLine1: "1 Test Way", PostalCode: "98101", ValidationStatus: "valid"})
		if err != nil {
			t.Fatalf("update address to valid: %v", err)
		}
		if _, err := addressController.BindAddressToLine(dto.BindAddressInput{FulfillmentLineID: first.ID, CustomerAddressID: valid.ID}); err != nil {
			t.Fatalf("rebind updated address: %v", err)
		}
		if err := s.gdb.First(&rebound, first.ID).Error; err != nil || rebound.AddressState != "ready" {
			t.Fatalf("rebound valid address state = %q, %v; want ready", rebound.AddressState, err)
		}
		for _, line := range s.fulfillmentLines[1:] {
			addr, err := addressController.CreateAddress(dto.CreateAddressInput{CustomerProfileID: *line.CustomerProfileID, Label: "shipping", RecipientName: "Scenario A member", Country: "US", City: "Seattle", Province: "WA", AddressLine1: fmt.Sprintf("%d Test Way", line.ID), PostalCode: "98101", ValidationStatus: "valid"})
			if err != nil {
				t.Fatalf("create valid address for line %d: %v", line.ID, err)
			}
			if _, err := addressController.BindAddressToLine(dto.BindAddressInput{FulfillmentLineID: line.ID, CustomerAddressID: addr.ID}); err != nil {
				t.Fatalf("bind valid address to line %d: %v", line.ID, err)
			}
		}
		s.fulfillmentLines = scenarioALines(t, s.gdb, s.waveID)
		for _, line := range s.fulfillmentLines {
			if line.AddressState != "ready" || line.CustomerAddressID == nil {
				t.Fatalf("line %d address projection = %q/%v, want ready/bound", line.ID, line.AddressState, line.CustomerAddressID)
			}
		}
	}) {
		t.FailNow()
	}

	if !t.Run("05 parallel second wave", func(t *testing.T) {
		second, err := NewWaveController().CreateWave(dto.CreateWaveInput{Name: "August Patreon member wave", WaveType: "membership"})
		if err != nil {
			t.Fatalf("CreateWave second wave: %v", err)
		}
		s.secondWaveID = second.ID
		firstOverview := scenarioAOverview(t, s.waveID)
		secondOverview := scenarioAOverview(t, s.secondWaveID)
		if firstOverview.ProjectedLifecycleStage != "review" || secondOverview.ProjectedLifecycleStage != "intake" {
			t.Fatalf("parallel stages first=%q second=%q, want review/intake", firstOverview.ProjectedLifecycleStage, secondOverview.ProjectedLifecycleStage)
		}
		scenarioARequireCount(t, s.gdb, &persistence.WaveDemandAssignment{}, "wave_id = ?", []any{s.secondWaveID}, 0)
		scenarioARequireCount(t, s.gdb, &persistence.WaveParticipantSnapshot{}, "wave_id = ?", []any{s.secondWaveID}, 0)
	}) {
		t.FailNow()
	}

	if !t.Run("06 factory order and file", func(t *testing.T) {
		factoryProfile := persistence.IntegrationProfile{
			ProfileKey: "scenario-a-factory", SourceSurface: "factory",
			SupportsExportSupplierOrder: true, FactorySupplierPlatform: "factory-a",
		}
		if err := s.gdb.Create(&factoryProfile).Error; err != nil {
			t.Fatalf("create factory profile: %v", err)
		}
		s.factoryProfileID = factoryProfile.ID
		orderTemplate := persistence.DocumentTemplate{
			TemplateKey: "scenario-a-supplier-order", DocumentType: "export_supplier_order", Format: "xlsx",
			MappingRules: `{"version":2,"mode":"header","columns":{"export.factory_sku":"SKU","export.quantity":"Qty"},"columnOrder":["export.factory_sku","export.quantity"]}`,
		}
		if err := s.gdb.Create(&orderTemplate).Error; err != nil {
			t.Fatalf("create supplier-order template: %v", err)
		}
		if err := s.gdb.Create(&persistence.IntegrationProfileTemplateBinding{
			IntegrationProfileID: factoryProfile.ID, DocumentType: "export_supplier_order", TemplateID: orderTemplate.ID, IsDefault: true,
		}).Error; err != nil {
			t.Fatalf("bind supplier-order template: %v", err)
		}
		exportController := NewExportController()
		orders, err := exportController.ExportSupplierOrderForProfile(s.waveID, factoryProfile.ID)
		if err != nil {
			t.Fatalf("ExportSupplierOrder: %v", err)
		}
		if len(orders) != 1 || orders[0].Status != "draft" {
			t.Fatalf("draft supplier orders = %+v, want one draft", orders)
		}
		if err := s.gdb.First(&s.supplierOrder, orders[0].ID).Error; err != nil {
			t.Fatalf("load supplier order: %v", err)
		}
		if err := s.gdb.Where("supplier_order_id = ?", orders[0].ID).Order("supplier_line_no").Find(&s.supplierOrderLines).Error; err != nil {
			t.Fatalf("load supplier order lines: %v", err)
		}
		if len(s.supplierOrderLines) != 6 {
			t.Fatalf("supplier order line coverage = %d, want 6", len(s.supplierOrderLines))
		}
		covered := make(map[uint]bool)
		for _, line := range s.supplierOrderLines {
			covered[line.FulfillmentLineID] = true
		}
		if len(covered) != 6 {
			t.Fatalf("distinct fulfillment-line coverage = %d, want 6", len(covered))
		}
		writer := app.NewSupplierOrderFileWriter(infra.NewSupplierOrderRepository(s.gdb), t.TempDir(), nil)
		fileResult, err := writer.GenerateSupplierOrderFile(context.Background(), orders[0].ID)
		if err != nil {
			t.Fatalf("GenerateSupplierOrderFile: %v", err)
		}
		s.supplierOrderOutput = fileResult.FilePath
		data, err := os.ReadFile(fileResult.FilePath)
		if err != nil || !json.Valid(data) || fileResult.LineCount != 6 {
			t.Fatalf("supplier order file path=%q lines=%d read=%v validJSON=%v", fileResult.FilePath, fileResult.LineCount, err, json.Valid(data))
		}
		submittedAt := time.Now()
		submitted, err := exportController.MarkSupplierOrderSubmitted(dto.MarkSupplierOrderSubmittedInput{OrderID: orders[0].ID, ExternalOrderNo: "FACTORY-ORDER-JULY", SubmittedAt: &submittedAt})
		if err != nil || submitted.Status != "submitted" {
			t.Fatalf("MarkSupplierOrderSubmitted = %+v, %v", submitted, err)
		}
		if got := scenarioAOverview(t, s.waveID); got.ProjectedLifecycleStage != "execution" || got.SupplierOrderCount != 1 {
			t.Fatalf("post-submit overview = %+v", got)
		}
	}) {
		t.FailNow()
	}

	if !t.Run("07 factory return import", func(t *testing.T) {
		// Production gap: there is no backend CSV parser or batch-number/supplier-line-number
		// resolver for factory return sheets. ImportShipments requires internal IDs, so this
		// test-side adapter renders those IDs into the surveyed return template and parses it.
		path := scenarioARenderFactoryReturn(t, s.supplierOrderLines)
		entries := scenarioAParseFactoryReturn(t, path)
		result, err := NewShipmentController().ImportShipments(dto.ImportShipmentInput{WaveID: s.waveID, IntegrationProfileID: s.factoryProfileID, ImportMode: "reject_all", Entries: entries})
		if err != nil {
			t.Fatalf("ImportShipments: %v", err)
		}
		if result.SuccessCount != 6 || result.ErrorCount != 0 || result.TotalProcessed != 6 || len(result.CreatedShipments) != 2 {
			t.Fatalf("shipment import result = %+v", result)
		}
		s.shipments = result.CreatedShipments
		seenFL := make(map[uint]int)
		tracking := make(map[string]bool)
		for _, shipment := range result.CreatedShipments {
			if shipment.ExternalShipmentNo != "FACTORY-JULY-A" && shipment.ExternalShipmentNo != "FACTORY-JULY-B" {
				t.Fatalf("unexpected shipment grouping key %q", shipment.ExternalShipmentNo)
			}
			tracking[shipment.TrackingNo] = true
			for _, line := range shipment.Lines {
				seenFL[line.FulfillmentLineID] += line.Quantity
			}
		}
		if len(seenFL) != 6 || len(tracking) != 2 {
			t.Fatalf("shipment reconciliation coverage=%v tracking=%v", seenFL, tracking)
		}
		for _, line := range s.fulfillmentLines {
			if seenFL[line.ID] != line.Quantity {
				t.Fatalf("shipment quantity for fulfillment line %d = %d, want %d", line.ID, seenFL[line.ID], line.Quantity)
			}
		}
		for _, line := range scenarioALines(t, s.gdb, s.waveID) {
			if line.SupplierState != "shipped" {
				t.Fatalf("line %d supplier state = %q, want shipped", line.ID, line.SupplierState)
			}
		}
	}) {
		t.FailNow()
	}

	if !t.Run("08 sync back file", func(t *testing.T) {
		items := make([]dto.CreateChannelSyncItemInput, 0, 6)
		for _, shipment := range s.shipments {
			for _, line := range shipment.Lines {
				items = append(items, dto.CreateChannelSyncItemInput{FulfillmentLineID: line.FulfillmentLineID, ShipmentID: shipment.ID, ExternalDocumentNo: "PATREON-JULY", ExternalLineNo: strconv.FormatUint(uint64(line.FulfillmentLineID), 10), TrackingNo: shipment.TrackingNo, CarrierCode: shipment.CarrierCode})
			}
		}
		job, err := NewChannelSyncController().CreateChannelSyncJob(dto.CreateChannelSyncJobInput{WaveID: s.waveID, IntegrationProfileID: s.profileID, Direction: "push_tracking", Items: items})
		if err != nil {
			t.Fatalf("CreateChannelSyncJob: %v", err)
		}
		if job.Status != "pending" || len(job.Items) != 6 {
			t.Fatalf("pending channel sync job = %+v", job)
		}
		if got := scenarioAOverview(t, s.waveID); got.ProjectedLifecycleStage != "syncing_back" {
			t.Fatalf("pending sync projected stage = %q, want syncing_back", got.ProjectedLifecycleStage)
		}
		syncDir := filepath.Join(t.TempDir(), "sync")
		executor := app.NewDocumentExportExecutor(syncDir, nil)
		provider := app.NewRuntimeExecutorProviderWith(map[string]map[string]app.ChannelSyncExecutor{"document_export": {"eli.local_export": executor}})
		execute := app.NewExecuteSyncUseCase(infra.NewChannelSyncRepository(s.gdb), infra.NewIntegrationProfileRepository(s.gdb), provider, infra.NewFulfillmentRepository(s.gdb))
		result, err := execute.ExecuteChannelSyncJob(context.Background(), job.ID)
		if err != nil {
			t.Fatalf("ExecuteChannelSyncJob: %v", err)
		}
		if result.JobStatus != "success" || len(result.Items) != 6 {
			t.Fatalf("execute sync result = %+v", result)
		}
		for _, line := range scenarioALines(t, s.gdb, s.waveID) {
			if line.ChannelSyncState != "synced" {
				t.Fatalf("line %d channel state = %q, want synced", line.ID, line.ChannelSyncState)
			}
		}
		var response struct {
			OutputFile string `json:"output_file"`
		}
		if err := json.Unmarshal([]byte(result.ResponsePayload), &response); err != nil || response.OutputFile == "" {
			t.Fatalf("sync response payload = %q, err=%v", result.ResponsePayload, err)
		}
		s.syncOutput = response.OutputFile
		data, err := os.ReadFile(response.OutputFile)
		if err != nil {
			t.Fatalf("read sync output file: %v", err)
		}
		var payload struct {
			Items []struct {
				FulfillmentLineID uint   `json:"fulfillment_line_id"`
				TrackingNo        string `json:"tracking_no"`
			} `json:"items"`
		}
		if err := json.Unmarshal(data, &payload); err != nil || len(payload.Items) != 6 {
			t.Fatalf("sync output reconciliation payload items=%d err=%v", len(payload.Items), err)
		}
		for _, item := range payload.Items {
			if item.FulfillmentLineID == 0 || item.TrackingNo == "" {
				t.Fatalf("sync output missing reconciliation/tracking fields: %+v", item)
			}
		}
		if got := scenarioAOverview(t, s.waveID); got.ProjectedLifecycleStage != "awaiting_manual_closure" || got.AutoClosureCandidateCount != 0 || got.ManualClosureCandidateCount != 6 {
			t.Fatalf("successful sync overview stage=%q auto candidates=%d manual candidates=%d; want awaiting_manual_closure, 0, 6", got.ProjectedLifecycleStage, got.AutoClosureCandidateCount, got.ManualClosureCandidateCount)
		}
	}) {
		t.FailNow()
	}

	if !t.Run("09 close", func(t *testing.T) {
		closed, err := NewWaveController().CloseWave(dto.CloseWaveInput{WaveID: s.waveID})
		if err != nil {
			t.Fatalf("CloseWave without force: %v", err)
		}
		if closed.Forced || closed.ResidualItemCount != 0 || closed.Wave.LifecycleStage != "closed" {
			t.Fatalf("CloseWave result = %+v", closed)
		}
		second, err := NewWaveController().GetWave(s.secondWaveID)
		if err != nil || second.LifecycleStage != "intake" {
			t.Fatalf("second wave after first close = %+v, %v; want intake", second, err)
		}
	}) {
		t.FailNow()
	}
}
