package app

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"gorm.io/gorm"
)

type shipmentReconcileDurableFixture struct {
	gdb     *gorm.DB
	profile *domain.IntegrationProfile
	wave    *domain.Wave
	lines   []domain.FulfillmentLine
	mapping *TemplateMappingService
}

func seedShipmentReconcileDurableFixture(t *testing.T, withCarrierMapping bool) *shipmentReconcileDurableFixture {
	t.Helper()
	ctx := context.Background()
	gdb := durableEvidenceDB(t)
	profileRepo := infra.NewIntegrationProfileRepository(gdb)
	profile := &domain.IntegrationProfile{
		ProfileKey:                     "shipment-reconcile-durable",
		SourceSurface:                  string(domain.SourceSurfaceFactory),
		SupportsImportSupplierShipment: true,
		RequiresCarrierMapping:         withCarrierMapping,
		FactorySupplierPlatform:        "factory",
	}
	if err := profileRepo.Create(ctx, profile); err != nil {
		t.Fatalf("create integration profile: %v", err)
	}
	templateRepo := infra.NewDocumentTemplateRepository(gdb)
	template := &domain.DocumentTemplate{
		TemplateKey:  "shipment-reconcile-durable",
		DocumentType: "import_supplier_shipment",
		Format:       "csv",
		MappingRules: `{"version":2,"mode":"header","hasHeader":true,"columns":{"shipment.third_party_order_no":"FL","shipment.tracking_no":"Tracking","shipment.external_shipment_no":"ExtNo","shipment.carrier_code":"Carrier","shipment.carrier_name":"CarrierName","shipment.quantity":"Qty"}}`,
	}
	if err := templateRepo.Create(ctx, template); err != nil {
		t.Fatalf("create shipment template: %v", err)
	}
	bindingRepo := infra.NewProfileTemplateBindingRepository(gdb)
	if err := bindingRepo.Create(ctx, &domain.IntegrationProfileTemplateBinding{IntegrationProfileID: profile.ID, DocumentType: template.DocumentType, TemplateID: template.ID, IsDefault: true}); err != nil {
		t.Fatalf("create shipment template binding: %v", err)
	}

	wave := &domain.Wave{WaveNo: "shipment-reconcile-durable", Name: "shipment-reconcile-durable", WaveType: "manual", LifecycleStage: "draft"}
	if err := infra.NewWaveRepository(gdb).Create(ctx, wave); err != nil {
		t.Fatalf("create wave: %v", err)
	}
	supplierRepo := infra.NewSupplierOrderRepository(gdb)
	order := &domain.SupplierOrder{WaveID: wave.ID, FactoryIntegrationProfileID: &profile.ID, SupplierPlatform: profile.FactorySupplierPlatform, BatchNo: "durable", Status: "submitted"}
	if err := supplierRepo.Create(ctx, order); err != nil {
		t.Fatalf("create supplier order: %v", err)
	}
	fulfillRepo := infra.NewFulfillmentRepository(gdb)
	lines := make([]domain.FulfillmentLine, 4)
	for i := range lines {
		line := domain.FulfillmentLine{WaveID: wave.ID, Quantity: 5, AllocationState: "allocated", SupplierState: "submitted", ChannelSyncState: "pending"}
		if err := fulfillRepo.Create(ctx, &line); err != nil {
			t.Fatalf("create fulfillment line %d: %v", i, err)
		}
		orderLine := &domain.SupplierOrderLine{SupplierOrderID: order.ID, FulfillmentLineID: line.ID, SupplierLineNo: i + 1, SubmittedQuantity: 5, Status: "submitted"}
		if err := supplierRepo.CreateLine(ctx, orderLine); err != nil {
			t.Fatalf("create supplier order line %d: %v", i, err)
		}
		lines[i] = line
	}
	if withCarrierMapping {
		mapping := &domain.CarrierMapping{IntegrationProfileID: profile.ID, InternalCarrierCode: "INTERNAL", ExternalCarrierCode: "EXTERNAL", ExternalCarrierName: "Mapped Carrier Name", Aliases: "[]"}
		if err := infra.NewCarrierMappingRepository(gdb).Create(ctx, mapping); err != nil {
			t.Fatalf("create carrier mapping: %v", err)
		}
	}

	mapping := NewTemplateMappingService(templateRepo, bindingRepo, profileRepo)
	return &shipmentReconcileDurableFixture{gdb: gdb, profile: profile, wave: wave, lines: lines, mapping: mapping}
}

func shipmentReconcileRow(lineID uint, externalShipmentNo, carrierCode, carrierName string) map[string]string {
	return map[string]string{
		"FL":          strconv.FormatUint(uint64(lineID), 10),
		"Tracking":    "TRACK-" + externalShipmentNo,
		"ExtNo":       externalShipmentNo,
		"Carrier":     carrierCode,
		"CarrierName": carrierName,
		"Qty":         "1",
	}
}

func executeShipmentReconcileDurably(
	t *testing.T,
	fixture *shipmentReconcileDurableFixture,
	input dto.MapAndReconcileShipmentsInput,
	afterMap func(*gorm.DB, *dto.ImportShipmentResult) error,
) (*dto.ImportShipmentResult, *ImportEvidenceUseCase, error) {
	t.Helper()
	ctx := context.Background()
	evidence := NewDeferredImportEvidenceUseCase(infra.NewImportEvidenceRepository(fixture.gdb))
	if err := PrepareTemplateImportEvidence(ctx, evidence, fixture.mapping, PrepareTemplateImportEvidenceInput{
		ImportKind:           "supplier_shipment",
		DocumentType:         "import_supplier_shipment",
		IntegrationProfileID: input.IntegrationProfileID,
		ImportMode:           input.ImportMode,
		FilePath:             input.FilePath,
		Rows:                 input.Rows,
	}); err != nil {
		t.Fatalf("prepare shipment evidence: %v", err)
	}

	var result *dto.ImportShipmentResult
	err := fixture.gdb.Transaction(func(tx *gorm.DB) error {
		supplierRepo := infra.NewSupplierOrderRepository(tx)
		fulfillRepo := infra.NewFulfillmentRepository(tx)
		mapping := NewTemplateMappingService(infra.NewDocumentTemplateRepository(tx), infra.NewProfileTemplateBindingRepository(tx), infra.NewIntegrationProfileRepository(tx))
		carrierMapping := NewCarrierMappingUseCase(infra.NewCarrierMappingRepository(tx), infra.NewIntegrationProfileRepository(tx))
		uc := NewShipmentImportUseCase(infra.NewShipmentRepository(tx), supplierRepo, fulfillRepo, nil)
		uc = WithShipmentReconcileDeps(uc, mapping, nil, nil, nil, carrierMapping)
		uc = WithShipmentImportEvidence(uc, evidence)
		uc = WithShipmentExternalCarrierRegistry(uc, NewExternalCarrierUseCase(infra.NewExternalCarrierRepository(tx)))
		var mapErr error
		result, mapErr = uc.MapAndReconcileShipments(ctx, input)
		if mapErr != nil {
			return mapErr
		}
		if afterMap != nil {
			return afterMap(tx, result)
		}
		return nil
	})
	if err != nil {
		if finalizeErr := evidence.FinalizeFailure(ctx, "failed", err); finalizeErr != nil {
			t.Fatalf("finalize failed shipment evidence: %v", finalizeErr)
		}
		return result, evidence, err
	}
	if err := evidence.FinalizePending(ctx); err != nil {
		t.Fatalf("finalize shipment evidence: %v", err)
	}
	return result, evidence, nil
}

func countShipmentBusinessRows(t *testing.T, gdb *gorm.DB) (carriers, mappings, shipments, shipmentLines int64) {
	t.Helper()
	for table, target := range map[string]*int64{
		"external_carriers": &carriers,
		"carrier_mappings":  &mappings,
		"shipments":         &shipments,
		"shipment_lines":    &shipmentLines,
	} {
		if err := gdb.Table(table).Count(target).Error; err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
	}
	return carriers, mappings, shipments, shipmentLines
}

func TestShipmentReconcileRejectAllStagesCarrierUntilFullFileGate(t *testing.T) {
	fixture := seedShipmentReconcileDurableFixture(t, false)
	input := dto.MapAndReconcileShipmentsInput{
		WaveID:               fixture.wave.ID,
		IntegrationProfileID: fixture.profile.ID,
		ImportMode:           "reject_all",
		Rows: []map[string]string{
			shipmentReconcileRow(fixture.lines[0].ID, "GOOD", "FIRST", "First Carrier"),
			shipmentReconcileRow(999999, "BAD", "SECOND", "Second Carrier"),
		},
	}
	result, _, err := executeShipmentReconcileDurably(t, fixture, input, nil)
	if err != nil {
		t.Fatalf("reject_all reconcile: %v", err)
	}
	if result.SuccessCount != 0 || result.ErrorCount != 1 || result.TotalProcessed != 2 {
		t.Fatalf("reject_all result = %+v", result)
	}
	carriers, mappings, shipments, shipmentLines := countShipmentBusinessRows(t, fixture.gdb)
	if carriers != 0 || mappings != 0 || shipments != 0 || shipmentLines != 0 {
		t.Fatalf("reject_all leaked business rows: carriers=%d mappings=%d shipments=%d shipmentLines=%d", carriers, mappings, shipments, shipmentLines)
	}
	detail, err := NewImportEvidenceUseCase(infra.NewImportEvidenceRepository(fixture.gdb)).GetRunDetail(context.Background(), result.ImportRunID)
	if err != nil {
		t.Fatalf("get rejected evidence: %v", err)
	}
	if detail.Run.Status != "rejected" || len(detail.Records) != 2 || detail.Records[0].Outcome != "failed" || detail.Records[1].Outcome != "failed" {
		t.Fatalf("rejected evidence = %+v", detail)
	}
}

func TestShipmentReconcileCarrierProvenanceAndSourceIdentity(t *testing.T) {
	t.Run("clean mapped carrier", func(t *testing.T) {
		fixture := seedShipmentReconcileDurableFixture(t, true)
		input := dto.MapAndReconcileShipmentsInput{WaveID: fixture.wave.ID, IntegrationProfileID: fixture.profile.ID, ImportMode: "reject_all", Rows: []map[string]string{
			shipmentReconcileRow(fixture.lines[0].ID, "CLEAN", "EXTERNAL", "Source Carrier Name"),
		}}
		result, _, err := executeShipmentReconcileDurably(t, fixture, input, nil)
		if err != nil || result.SuccessCount != 1 || len(result.CreatedShipments) != 1 {
			t.Fatalf("clean reconcile result=%+v err=%v", result, err)
		}
		if result.CreatedShipments[0].CarrierCode != "INTERNAL" || result.CreatedShipments[0].CarrierName != "Mapped Carrier Name" {
			t.Fatalf("shipment carrier was not mapped internally: %+v", result.CreatedShipments[0])
		}
		carriers, err := infra.NewExternalCarrierRepository(fixture.gdb).ListByProfile(context.Background(), fixture.profile.ID)
		if err != nil || len(carriers) != 1 {
			t.Fatalf("external carriers=%+v err=%v", carriers, err)
		}
		carrier := carriers[0]
		if carrier.ExternalCarrierCode != "EXTERNAL" || carrier.ExternalCarrierName != "Source Carrier Name" {
			t.Fatalf("registry did not preserve source carrier identity: %+v", carrier)
		}
		if carrier.SourceImportRunID == nil || *carrier.SourceImportRunID != result.ImportRunID || carrier.SourceRawRecordID == nil {
			t.Fatalf("carrier provenance = %+v; run=%d", carrier, result.ImportRunID)
		}
		records, err := infra.NewImportEvidenceRepository(fixture.gdb).ListRecordsByRun(context.Background(), result.ImportRunID)
		if err != nil || len(records) != 1 || records[0].ID != *carrier.SourceRawRecordID || records[0].ImportRunID != *carrier.SourceImportRunID {
			t.Fatalf("carrier RAW provenance is not durable: records=%+v carrier=%+v err=%v", records, carrier, err)
		}
	})

	t.Run("skip invalid observes successful source rows only", func(t *testing.T) {
		fixture := seedShipmentReconcileDurableFixture(t, false)
		input := dto.MapAndReconcileShipmentsInput{WaveID: fixture.wave.ID, IntegrationProfileID: fixture.profile.ID, ImportMode: "skip_invalid", Rows: []map[string]string{
			shipmentReconcileRow(fixture.lines[0].ID, "GOOD", "GOOD-CARRIER", "Good Carrier"),
			shipmentReconcileRow(999999, "BAD", "BAD-CARRIER", "Bad Carrier"),
		}}
		result, _, err := executeShipmentReconcileDurably(t, fixture, input, nil)
		if err != nil || result.SuccessCount != 1 || result.ErrorCount != 1 {
			t.Fatalf("skip_invalid result=%+v err=%v", result, err)
		}
		carriers, err := infra.NewExternalCarrierRepository(fixture.gdb).ListByProfile(context.Background(), fixture.profile.ID)
		if err != nil || len(carriers) != 1 || carriers[0].ExternalCarrierCode != "GOOD-CARRIER" || carriers[0].SourceImportRunID == nil || carriers[0].SourceRawRecordID == nil {
			t.Fatalf("skip_invalid registry=%+v err=%v", carriers, err)
		}
	})

	t.Run("blank carrier does not block shipment", func(t *testing.T) {
		fixture := seedShipmentReconcileDurableFixture(t, false)
		input := dto.MapAndReconcileShipmentsInput{WaveID: fixture.wave.ID, IntegrationProfileID: fixture.profile.ID, ImportMode: "reject_all", Rows: []map[string]string{
			shipmentReconcileRow(fixture.lines[0].ID, "NO-CARRIER", "", ""),
		}}
		result, _, err := executeShipmentReconcileDurably(t, fixture, input, nil)
		if err != nil || result.SuccessCount != 1 {
			t.Fatalf("blank carrier result=%+v err=%v", result, err)
		}
		carriers, _, shipments, _ := countShipmentBusinessRows(t, fixture.gdb)
		if carriers != 0 || shipments != 1 {
			t.Fatalf("blank carrier deltas: carriers=%d shipments=%d", carriers, shipments)
		}
	})
}

func TestShipmentReconcileExactCarrierRepeatPreservesFirstProvenance(t *testing.T) {
	fixture := seedShipmentReconcileDurableFixture(t, false)
	var firstRunID, firstRawID uint
	for attempt := 0; attempt < 2; attempt++ {
		input := dto.MapAndReconcileShipmentsInput{WaveID: fixture.wave.ID, IntegrationProfileID: fixture.profile.ID, ImportMode: "reject_all", Rows: []map[string]string{
			shipmentReconcileRow(fixture.lines[attempt].ID, fmt.Sprintf("REPEAT-%d", attempt), "REPEAT", "Repeat Carrier"),
		}}
		result, _, err := executeShipmentReconcileDurably(t, fixture, input, nil)
		if err != nil || result.SuccessCount != 1 {
			t.Fatalf("attempt %d result=%+v err=%v", attempt, result, err)
		}
		carriers, listErr := infra.NewExternalCarrierRepository(fixture.gdb).ListByProfile(context.Background(), fixture.profile.ID)
		if listErr != nil || len(carriers) != 1 || carriers[0].SourceImportRunID == nil || carriers[0].SourceRawRecordID == nil {
			t.Fatalf("attempt %d carriers=%+v err=%v", attempt, carriers, listErr)
		}
		if attempt == 0 {
			firstRunID = *carriers[0].SourceImportRunID
			firstRawID = *carriers[0].SourceRawRecordID
		} else if *carriers[0].SourceImportRunID != firstRunID || *carriers[0].SourceRawRecordID != firstRawID {
			t.Fatalf("exact repeat replaced first provenance: carrier=%+v firstRun=%d firstRaw=%d", carriers[0], firstRunID, firstRawID)
		}
	}
}

func TestShipmentReconcileCarrierAndBusinessFailuresRollBackWithoutLeak(t *testing.T) {
	t.Run("observation failure", func(t *testing.T) {
		fixture := seedShipmentReconcileDurableFixture(t, false)
		if err := fixture.gdb.Exec(`CREATE TRIGGER fail_shipment_carrier BEFORE INSERT ON external_carriers BEGIN SELECT RAISE(ABORT, 'injected shipment carrier failure'); END`).Error; err != nil {
			t.Fatalf("create carrier trigger: %v", err)
		}
		input := dto.MapAndReconcileShipmentsInput{WaveID: fixture.wave.ID, IntegrationProfileID: fixture.profile.ID, ImportMode: "reject_all", Rows: []map[string]string{
			shipmentReconcileRow(fixture.lines[0].ID, "OBSERVE-FAIL", "FAIL", "Fail Carrier"),
		}}
		_, _, err := executeShipmentReconcileDurably(t, fixture, input, nil)
		if err == nil {
			t.Fatal("expected carrier observation failure")
		}
		carriers, mappings, shipments, shipmentLines := countShipmentBusinessRows(t, fixture.gdb)
		if carriers != 0 || mappings != 0 || shipments != 0 || shipmentLines != 0 {
			t.Fatalf("observation failure leaked business rows: carriers=%d mappings=%d shipments=%d shipmentLines=%d", carriers, mappings, shipments, shipmentLines)
		}
	})

	t.Run("later business failure", func(t *testing.T) {
		fixture := seedShipmentReconcileDurableFixture(t, false)
		rollbackErr := errors.New("injected later shipment business failure")
		input := dto.MapAndReconcileShipmentsInput{WaveID: fixture.wave.ID, IntegrationProfileID: fixture.profile.ID, ImportMode: "reject_all", Rows: []map[string]string{
			shipmentReconcileRow(fixture.lines[0].ID, "LATER-FAIL", "LATER", "Later Carrier"),
		}}
		_, _, err := executeShipmentReconcileDurably(t, fixture, input, func(*gorm.DB, *dto.ImportShipmentResult) error { return rollbackErr })
		if !errors.Is(err, rollbackErr) {
			t.Fatalf("later failure = %v", err)
		}
		carriers, mappings, shipments, shipmentLines := countShipmentBusinessRows(t, fixture.gdb)
		if carriers != 0 || mappings != 0 || shipments != 0 || shipmentLines != 0 {
			t.Fatalf("later failure leaked business rows: carriers=%d mappings=%d shipments=%d shipmentLines=%d", carriers, mappings, shipments, shipmentLines)
		}
	})
}
