package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type scenarioDTestState struct {
	gdb                  *gorm.DB
	waveController       *WaveController
	demandController     *DemandController
	profileController    *CustomerProfileController
	productController    *ProductController
	policyController     *AllocationPolicyController
	adjustmentController *AdjustmentController
	addressController    *AddressController
	exportController     *ExportController
	shipmentController   *ShipmentController

	wave       dto.WaveDTO
	profiles   []dto.CustomerProfileDTO
	addresses  map[uint]uint
	product    dto.ProductDTO
	lineIDs    []uint
	xProfileID uint
	xLineID    uint
}

func setupScenarioDTestDB(t *testing.T) *scenarioDTestState {
	t.Helper()

	gdb, err := gorm.Open(sqlite.Open("file:scenario_d?mode=memory&cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: false,
	})
	if err != nil {
		t.Fatalf("open scenario D database: %v", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("get scenario D SQL database: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	if err := gdb.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatalf("enable scenario D foreign keys: %v", err)
	}
	if err := gdb.AutoMigrate(
		&persistence.CustomerProfile{},
		&persistence.CustomerMergeRecord{},
		&persistence.CustomerIdentity{},
		&persistence.CustomerAddress{},
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
		t.Fatalf("migrate scenario D database: %v", err)
	}

	previousDB := db.GetDB()
	db.SetDefaultDB(gdb)
	t.Cleanup(func() {
		db.SetDefaultDB(previousDB)
		_ = sqlDB.Close()
	})

	return &scenarioDTestState{
		gdb:                  gdb,
		waveController:       NewWaveController(),
		demandController:     NewDemandController(),
		profileController:    NewCustomerProfileController(),
		productController:    NewProductController(),
		policyController:     NewAllocationPolicyController(),
		adjustmentController: NewAdjustmentController(),
		addressController:    NewAddressController(),
		exportController:     NewExportController(),
		shipmentController:   NewShipmentController(),
		addresses:            make(map[uint]uint),
	}
}

func scenarioDUintIDs(lines []persistence.FulfillmentLine) []uint {
	ids := make([]uint, len(lines))
	for i := range lines {
		ids[i] = lines[i].ID
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func scenarioDEqualIDs(got, want []uint) bool {
	if len(got) != len(want) {
		return false
	}
	gotCopy := append([]uint(nil), got...)
	wantCopy := append([]uint(nil), want...)
	sort.Slice(gotCopy, func(i, j int) bool { return gotCopy[i] < gotCopy[j] })
	sort.Slice(wantCopy, func(i, j int) bool { return wantCopy[i] < wantCopy[j] })
	for i := range gotCopy {
		if gotCopy[i] != wantCopy[i] {
			return false
		}
	}
	return true
}

func TestScenarioD_MixedWaveQueries(t *testing.T) {
	s := setupScenarioDTestDB(t)

	if !t.Run("1_create_mixed_wave_and_assign_five_demands", func(t *testing.T) {
		wave, err := s.waveController.CreateWave(dto.CreateWaveInput{
			Name:     "Scenario D Mixed Wave",
			WaveType: string(domain.WaveTypeMixed),
		})
		if err != nil {
			t.Fatalf("create mixed wave: %v", err)
		}
		s.wave = wave

		master, err := s.productController.CreateProductMaster(dto.CreateProductMasterInput{
			SupplierPlatform:   "scenario-d-factory",
			FactorySKU:         "SCENARIO-D-SKU",
			SupplierProductRef: "scenario-d-product",
			Name:               "Scenario D Gift",
			ProductKind:        string(domain.ProductKindOther),
		})
		if err != nil {
			t.Fatalf("create product master: %v", err)
		}
		products, err := s.productController.SnapshotProductsForWave(dto.SnapshotProductsInput{
			WaveID:    wave.ID,
			MasterIDs: []uint{master.ID},
		})
		if err != nil {
			t.Fatalf("snapshot wave product: %v", err)
		}
		if len(products) != 1 {
			t.Fatalf("snapshotted products = %d, want 1", len(products))
		}
		s.product = products[0]

		for i := 0; i < 5; i++ {
			profileType := string(domain.ProfileTypeMember)
			if i >= 3 {
				profileType = string(domain.ProfileTypeBuyer)
			}
			profile, err := s.profileController.CreateCustomerProfile(dto.CreateCustomerProfileInput{
				DisplayName: fmt.Sprintf("Scenario D Person %d", i+1),
				ProfileType: profileType,
			})
			if err != nil {
				t.Fatalf("create profile %d: %v", i+1, err)
			}
			s.profiles = append(s.profiles, *profile)

			address, err := s.addressController.CreateAddress(dto.CreateAddressInput{
				CustomerProfileID: profile.ID,
				Label:             "Scenario D valid address",
				RecipientName:     profile.DisplayName,
				Phone:             "555-0100",
				Country:           "US",
				Province:          "CA",
				City:              "Los Angeles",
				AddressLine1:      fmt.Sprintf("%d Scenario Avenue", i+1),
				PostalCode:        "90001",
				IsDefault:         true,
				ValidationStatus:  string(domain.AddressValidationStatusValid),
			})
			if err != nil {
				t.Fatalf("create address for profile %d: %v", profile.ID, err)
			}
			s.addresses[profile.ID] = address.ID

			isMember := i < 3
			kind := string(domain.DemandKindRetailOrder)
			captureMode := string(domain.CaptureModeDocumentImport)
			lineType := string(domain.DemandLineTypeSKUOrder)
			inputState := string(domain.RecipientInputStateReady)
			channel := "scenario-d-shop"
			giftLevel := "retail"
			if isMember {
				kind = string(domain.DemandKindMembershipEntitlement)
				captureMode = string(domain.CaptureModeAPIIngest)
				lineType = string(domain.DemandLineTypeEntitlementRule)
				inputState = string(domain.RecipientInputStateNotRequired)
				channel = "bilibili"
				giftLevel = "gold"
			}
			doc, err := s.demandController.ImportDemandDocument(dto.CreateDemandInput{
				Kind:              kind,
				CaptureMode:       captureMode,
				SourceChannel:     channel,
				SourceSurface:     "scenario-d",
				SourceDocumentNo:  fmt.Sprintf("SCENARIO-D-%d", i+1),
				SourceCustomerRef: fmt.Sprintf("scenario-d-person-%d", i+1),
				CustomerProfileID: &profile.ID,
				Lines: []dto.CreateDemandLineInput{{
					LineType:              lineType,
					ObligationTriggerKind: string(domain.ObligationTriggerKindPeriodicMembership),
					EntitlementAuthority:  string(domain.EntitlementAuthorityLocalPolicy),
					RecipientInputState:   inputState,
					RoutingDisposition:    string(domain.RoutingDispositionAccepted),
					GiftLevelSnapshot:     giftLevel,
					ProductMasterID:       &master.ID,
					ExternalTitle:         "Scenario D Gift",
					RequestedQuantity:     1,
				}},
			})
			if err != nil {
				t.Fatalf("import demand %d: %v", i+1, err)
			}
			if err := s.waveController.AssignDemandToWave(wave.ID, doc.ID); err != nil {
				t.Fatalf("assign demand %d to wave: %v", doc.ID, err)
			}
		}
		s.xProfileID = s.profiles[0].ID

		var assignments int64
		if err := s.gdb.Model(&persistence.WaveDemandAssignment{}).Where("wave_id = ?", wave.ID).Count(&assignments).Error; err != nil {
			t.Fatalf("count wave assignments: %v", err)
		}
		if assignments != 5 {
			t.Fatalf("wave assignments = %d, want 5", assignments)
		}
	}) {
		t.FailNow()
	}

	if !t.Run("2_generate_policy_and_demand_lines_together", func(t *testing.T) {
		generated, err := s.waveController.GenerateParticipants(s.wave.ID)
		if err != nil {
			t.Fatalf("generate participants: %v", err)
		}
		if generated != 5 {
			t.Fatalf("generated participants = %d, want 5", generated)
		}

		selectorPayload, err := json.Marshal(domain.SelectorPayload{
			Type:     "identity_level",
			Platform: "bilibili",
			Level:    "gold",
		})
		if err != nil {
			t.Fatalf("marshal identity-level selector: %v", err)
		}
		if _, err := s.policyController.CreateAllocationPolicyRule(dto.CreateAllocationPolicyRuleInput{
			WaveID:               s.wave.ID,
			ProductID:            s.product.ID,
			SelectorPayload:      selectorPayload,
			ProductTargetRef:     s.product.FactorySKU,
			ContributionQuantity: 1,
			RuleKind:             string(domain.LineReasonEntitlement),
			Priority:             1,
			Active:               true,
		}); err != nil {
			t.Fatalf("create identity-level allocation rule: %v", err)
		}
		if _, err := s.policyController.ReconcileWave(s.wave.ID); err != nil {
			t.Fatalf("reconcile membership allocation: %v", err)
		}
		mapping, err := s.waveController.MapDemandLines(s.wave.ID)
		if err != nil {
			t.Fatalf("map retail demand lines: %v", err)
		}
		if len(mapping.CreatedLines) != 2 || len(mapping.BlockedLines) != 0 {
			t.Fatalf("retail mapping created=%d blocked=%d, want 2/0", len(mapping.CreatedLines), len(mapping.BlockedLines))
		}

		var lines []persistence.FulfillmentLine
		if err := s.gdb.Where("wave_id = ?", s.wave.ID).Find(&lines).Error; err != nil {
			t.Fatalf("list fulfillment lines: %v", err)
		}
		counts := map[string]int{}
		for _, line := range lines {
			counts[line.GeneratedBy+"/"+string(line.LineReason)]++
		}
		if len(lines) != 5 || counts["allocation_policy_driven/entitlement"] != 3 || counts["allocation_demand_driven/retail_order"] != 2 {
			t.Fatalf("mixed line coexistence mismatch: total=%d counts=%v", len(lines), counts)
		}
	}) {
		t.FailNow()
	}

	if !t.Run("3_record_three_reissues_and_reconcile", func(t *testing.T) {
		var participants []persistence.WaveParticipantSnapshot
		if err := s.gdb.Where("wave_id = ? AND customer_profile_id IN ?", s.wave.ID, []uint{
			s.profiles[0].ID, s.profiles[1].ID, s.profiles[2].ID,
		}).Order("customer_profile_id ASC").Find(&participants).Error; err != nil {
			t.Fatalf("list member participants: %v", err)
		}
		if len(participants) != 3 {
			t.Fatalf("member participants = %d, want 3", len(participants))
		}

		entries := make([]dto.RecordAdjustmentInput, 3)
		for i := range participants {
			participantID := participants[i].ID
			entries[i] = dto.RecordAdjustmentInput{
				WaveID:                    s.wave.ID,
				TargetKind:                "participant",
				WaveParticipantSnapshotID: &participantID,
				AdjustmentKind:            string(domain.AdjustmentKindReissue),
				QuantityDelta:             i + 1,
				ReasonCode:                "scenario_d_reissue",
				OperatorID:                "scenario-d-operator",
				Note:                      "Reissue after delivery exception",
			}
		}
		// A reissue is an adjustment, not manual_entry demand: manual demand would rewrite source truth.
		result, err := s.adjustmentController.BatchRecordAdjustments(dto.BatchRecordAdjustmentsInput{Entries: entries})
		if err != nil {
			t.Fatalf("batch record reissues: %v", err)
		}
		if result.SuccessCount != 3 || result.FailureCount != 0 || len(result.Results) != 3 {
			t.Fatalf("batch reissues success=%d failure=%d results=%d, want 3/0/3", result.SuccessCount, result.FailureCount, len(result.Results))
		}

		reconcile, err := s.policyController.ReconcileWave(s.wave.ID)
		if err != nil {
			t.Fatalf("reconcile reissues: %v", err)
		}
		if reconcile.ReplayedCount != 3 || len(reconcile.Failures) != 0 {
			t.Fatalf("replayed adjustments=%d failures=%v, want 3/none", reconcile.ReplayedCount, reconcile.Failures)
		}

		var adjustments []persistence.FulfillmentAdjustment
		if err := s.gdb.Where("wave_id = ?", s.wave.ID).Order("id ASC").Find(&adjustments).Error; err != nil {
			t.Fatalf("list durable adjustments: %v", err)
		}
		if len(adjustments) != 3 {
			t.Fatalf("durable adjustments = %d, want 3", len(adjustments))
		}
		expectedQuantity := make(map[uint]int, 3)
		for i, participant := range participants {
			expectedQuantity[participant.ID] = 1 + i + 1
		}
		var policyLines []persistence.FulfillmentLine
		if err := s.gdb.Where("wave_id = ? AND generated_by = ?", s.wave.ID, "allocation_policy_driven").Find(&policyLines).Error; err != nil {
			t.Fatalf("list reconciled policy lines: %v", err)
		}
		if len(policyLines) != 3 {
			t.Fatalf("reconciled policy lines = %d, want 3", len(policyLines))
		}
		for _, line := range policyLines {
			if line.WaveParticipantSnapshotID == nil {
				t.Fatalf("policy line %d has no participant", line.ID)
			}
			if want := expectedQuantity[*line.WaveParticipantSnapshotID]; line.Quantity != want {
				t.Fatalf("policy line %d quantity = %d, want %d", line.ID, line.Quantity, want)
			}
			if line.CustomerProfileID != nil && *line.CustomerProfileID == s.xProfileID {
				s.xLineID = line.ID
			}
		}
		if s.xLineID == 0 {
			t.Fatal("did not find X's reconciled fulfillment line")
		}
	}) {
		t.FailNow()
	}

	if !t.Run("4_bind_addresses_export_and_ship_only_x", func(t *testing.T) {
		var lines []persistence.FulfillmentLine
		if err := s.gdb.Where("wave_id = ?", s.wave.ID).Order("id ASC").Find(&lines).Error; err != nil {
			t.Fatalf("list final lines before address binding: %v", err)
		}
		if len(lines) != 5 {
			t.Fatalf("final lines = %d, want 5", len(lines))
		}
		for _, line := range lines {
			if line.CustomerProfileID == nil {
				t.Fatalf("line %d has no customer profile", line.ID)
			}
			addressID := s.addresses[*line.CustomerProfileID]
			if _, err := s.addressController.BindAddressToLine(dto.BindAddressInput{
				FulfillmentLineID: line.ID,
				CustomerAddressID: addressID,
			}); err != nil {
				t.Fatalf("bind address %d to line %d: %v", addressID, line.ID, err)
			}
		}
		s.lineIDs = scenarioDUintIDs(lines)

		orders, err := s.exportController.ExportSupplierOrder(s.wave.ID)
		if err != nil {
			t.Fatalf("export draft supplier orders: %v", err)
		}
		if len(orders) == 0 {
			t.Fatal("export created no supplier orders")
		}
		for _, order := range orders {
			if order.Status != string(domain.SupplierOrderStatusDraft) {
				t.Fatalf("supplier order %d status = %q, want draft", order.ID, order.Status)
			}
		}

		var supplierLine persistence.SupplierOrderLine
		if err := s.gdb.Where("fulfillment_line_id = ?", s.xLineID).First(&supplierLine).Error; err != nil {
			t.Fatalf("find X supplier order line: %v", err)
		}
		shipmentResult, err := s.shipmentController.ImportShipments(dto.ImportShipmentInput{
			WaveID:     s.wave.ID,
			ImportMode: "reject_all",
			Entries: []dto.ImportShipmentEntry{{
				SupplierOrderLineID: supplierLine.ID,
				FulfillmentLineID:   s.xLineID,
				ExternalShipmentNo:  "SCENARIO-D-SHIP-X",
				CarrierCode:         "UPS",
				CarrierName:         "Scenario D Carrier",
				TrackingNo:          "SCENARIO-D-TRACK-X",
				Quantity:            supplierLine.SubmittedQuantity,
			}},
		})
		if err != nil {
			t.Fatalf("import X shipment: %v", err)
		}
		if shipmentResult.SuccessCount != 1 || shipmentResult.ErrorCount != 0 || len(shipmentResult.CreatedShipments) != 1 {
			t.Fatalf("shipment import success=%d errors=%d shipments=%d, want 1/0/1", shipmentResult.SuccessCount, shipmentResult.ErrorCount, len(shipmentResult.CreatedShipments))
		}

		var finalLines []persistence.FulfillmentLine
		if err := s.gdb.Where("wave_id = ?", s.wave.ID).Order("id ASC").Find(&finalLines).Error; err != nil {
			t.Fatalf("reload final fulfillment lines: %v", err)
		}
		for _, line := range finalLines {
			if line.AddressState != string(domain.AddressStateReady) {
				t.Fatalf("line %d address state = %q, want ready", line.ID, line.AddressState)
			}
			wantSupplierState := string(domain.SupplierStateNotSubmitted)
			if line.ID == s.xLineID {
				wantSupplierState = string(domain.SupplierStateShipped)
			}
			if line.SupplierState != wantSupplierState {
				t.Fatalf("line %d supplier state = %q, want %q", line.ID, line.SupplierState, wantSupplierState)
			}
		}
	}) {
		t.FailNow()
	}

	if !t.Run("5_question_d1_has_x_been_shipped", func(t *testing.T) {
		// The operator question is intentionally answered with exactly one controller call.
		history, err := s.profileController.GetCustomerFulfillmentHistory(s.xProfileID)
		if err != nil {
			t.Fatalf("get X fulfillment history: %v", err)
		}
		if len(history) != 1 {
			t.Fatalf("X fulfillment history rows = %d, want 1", len(history))
		}
		row := history[0]
		if row.FulfillmentLineID != s.xLineID || row.ShipmentID == nil || *row.ShipmentID == 0 {
			t.Fatalf("X history does not identify the shipped line: %+v", row)
		}
		if row.SupplierState != string(domain.SupplierStateShipped) || row.ShipmentStatus != string(domain.ShipmentStatusShipped) {
			t.Fatalf("X history supplier/shipment states = %q/%q, want shipped/shipped", row.SupplierState, row.ShipmentStatus)
		}
		if row.CarrierName != "Scenario D Carrier" || row.TrackingNo != "SCENARIO-D-TRACK-X" {
			t.Fatalf("X history carrier/tracking = %q/%q", row.CarrierName, row.TrackingNo)
		}
		if row.WaveID != s.wave.ID || row.WaveName != s.wave.Name || row.WaveNo != s.wave.WaveNo {
			t.Fatalf("X history wave context = id %d name %q no %q", row.WaveID, row.WaveName, row.WaveNo)
		}
		if row.ProductName != s.product.Name || row.ProductSKU != s.product.FactorySKU {
			t.Fatalf("X history product = %q/%q, want %q/%q", row.ProductName, row.ProductSKU, s.product.Name, s.product.FactorySKU)
		}
	}) {
		t.FailNow()
	}

	if !t.Run("6_question_d2_address_ready_not_submitted", func(t *testing.T) {
		expectedIDs := make([]uint, 0, len(s.lineIDs)-1)
		for _, id := range s.lineIDs {
			if id != s.xLineID {
				expectedIDs = append(expectedIDs, id)
			}
		}

		// The two dimensions are ANDed, and the operator question uses exactly one call.
		page, err := s.waveController.ListWaveFulfillmentRowsFiltered(dto.WaveFulfillmentFilterInput{
			WaveID:         s.wave.ID,
			AddressStates:  []string{string(domain.AddressStateReady)},
			SupplierStates: []string{string(domain.SupplierStateNotSubmitted)},
			Pagination:     dto.PaginationInput{Page: 1, PageSize: 100},
		})
		if err != nil {
			t.Fatalf("list address-ready not-submitted rows: %v", err)
		}
		gotIDs := make([]uint, len(page.Items))
		for i, row := range page.Items {
			gotIDs[i] = row.FulfillmentLineID
			if row.AddressState != string(domain.AddressStateReady) || row.SupplierState != string(domain.SupplierStateNotSubmitted) {
				t.Fatalf("filtered row %d leaked states address=%q supplier=%q", row.FulfillmentLineID, row.AddressState, row.SupplierState)
			}
		}
		if page.Pagination.TotalCount != len(expectedIDs) || !scenarioDEqualIDs(gotIDs, expectedIDs) {
			t.Fatalf("filtered rows total=%d ids=%v, want total=%d ids=%v", page.Pagination.TotalCount, gotIDs, len(expectedIDs), expectedIDs)
		}
		for _, id := range gotIDs {
			if id == s.xLineID {
				t.Fatalf("shipped X line %d leaked into not-submitted results", id)
			}
		}
	}) {
		t.FailNow()
	}
}
