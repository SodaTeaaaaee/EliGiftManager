package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"gorm.io/gorm"
)

func durableEvidenceDB(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := database.InitDB(filepath.Join(t.TempDir(), "durable.db"))
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("DB: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return gdb
}

func TestImportEvidenceInitialFailurePreventsBusinessWrites(t *testing.T) {
	gdb := durableEvidenceDB(t)
	if err := gdb.Exec(`CREATE TRIGGER fail_raw_insert BEFORE INSERT ON import_raw_records BEGIN SELECT RAISE(ABORT, 'injected raw failure'); END`).Error; err != nil {
		t.Fatalf("create trigger: %v", err)
	}
	evidence := NewDeferredImportEvidenceUseCase(infra.NewImportEvidenceRepository(gdb))
	businessInvoked := false
	err := func() error {
		if _, _, startErr := evidence.StartImportEvidence(context.Background(), "test", 0, "reject_all", "source.csv", `{}`, []any{[]string{"row"}}, []map[string]string{{}}, nil); startErr != nil {
			return startErr
		}
		businessInvoked = true
		return nil
	}()
	if err == nil {
		t.Fatal("expected initial evidence failure")
	}
	if businessInvoked {
		t.Fatal("business phase ran after initial evidence failure")
	}
	var runs, records int64
	if err := gdb.Table("import_runs").Count(&runs).Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.Table("import_raw_records").Count(&records).Error; err != nil {
		t.Fatal(err)
	}
	if runs != 0 || records != 0 {
		t.Fatalf("initial evidence transaction was not atomic: runs=%d records=%d", runs, records)
	}
}

func TestImportEvidenceSurvivesBusinessRollback(t *testing.T) {
	gdb := durableEvidenceDB(t)
	ctx := context.Background()
	if err := gdb.Exec(`CREATE TABLE business_probe (id INTEGER PRIMARY KEY)`).Error; err != nil {
		t.Fatal(err)
	}
	evidence := NewDeferredImportEvidenceUseCase(infra.NewImportEvidenceRepository(gdb))
	run, records, err := evidence.StartImportEvidence(ctx, "test", 0, "reject_all", "source.csv", `{}`, []any{[]string{"row"}}, []map[string]string{{}}, nil)
	if err != nil {
		t.Fatalf("StartImportEvidence: %v", err)
	}
	MarkImportEvidenceSuccess(records, 0, "business_probe", 1)
	rollbackErr := errors.New("injected business rollback")
	err = gdb.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`INSERT INTO business_probe(id) VALUES (1)`).Error; err != nil {
			return err
		}
		return rollbackErr
	})
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("business transaction error = %v", err)
	}
	if err := evidence.FinalizeFailure(ctx, "failed", rollbackErr); err != nil {
		t.Fatalf("FinalizeFailure: %v", err)
	}
	var probes int64
	if err := gdb.Table("business_probe").Count(&probes).Error; err != nil {
		t.Fatal(err)
	}
	if probes != 0 {
		t.Fatalf("business rollback persisted %d probe rows", probes)
	}
	detail, err := NewImportEvidenceUseCase(infra.NewImportEvidenceRepository(gdb)).GetRunDetail(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRunDetail: %v", err)
	}
	if detail.Run.Status != "failed" || len(detail.Records) != 1 || detail.Records[0].Outcome != "failed" || detail.Records[0].ResultID != nil {
		t.Fatalf("rollback evidence = %+v", detail)
	}
}

func TestImportEvidenceFinalizeFailureIsExplicitAndAtomic(t *testing.T) {
	gdb := durableEvidenceDB(t)
	ctx := context.Background()
	if err := gdb.Exec(`CREATE TABLE business_probe (id INTEGER PRIMARY KEY)`).Error; err != nil {
		t.Fatal(err)
	}
	evidence := NewDeferredImportEvidenceUseCase(infra.NewImportEvidenceRepository(gdb))
	run, records, err := evidence.StartImportEvidence(ctx, "test", 0, "skip_invalid", "source.csv", `{}`, []any{[]string{"row"}}, []map[string]string{{}}, nil)
	if err != nil {
		t.Fatalf("StartImportEvidence: %v", err)
	}
	if err := gdb.Transaction(func(tx *gorm.DB) error { return tx.Exec(`INSERT INTO business_probe(id) VALUES (1)`).Error }); err != nil {
		t.Fatalf("business commit: %v", err)
	}
	MarkImportEvidenceSuccess(records, 0, "business_probe", 1)
	if err := evidence.CompleteImportEvidence(ctx, run, records, "completed"); err != nil {
		t.Fatalf("CompleteImportEvidence: %v", err)
	}
	if err := gdb.Exec(`CREATE TRIGGER fail_run_finalize BEFORE UPDATE ON import_runs WHEN NEW.status <> 'running' BEGIN SELECT RAISE(ABORT, 'injected finalize failure'); END`).Error; err != nil {
		t.Fatal(err)
	}
	err = evidence.FinalizePending(ctx)
	var incomplete *ImportEvidenceAuditIncompleteError
	if !errors.As(err, &incomplete) || incomplete.RunID != run.ID {
		t.Fatalf("FinalizePending error = %T %v", err, err)
	}
	var probes int64
	if err := gdb.Table("business_probe").Count(&probes).Error; err != nil || probes != 1 {
		t.Fatalf("committed business row count=%d err=%v", probes, err)
	}
	detail, err := NewImportEvidenceUseCase(infra.NewImportEvidenceRepository(gdb)).GetRunDetail(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Run.Status != "running" || detail.Records[0].Outcome != "pending" {
		t.Fatalf("failed finalization was not atomic: %+v", detail)
	}
}

func seedCarrierImportDB(t *testing.T, gdb *gorm.DB) (*TemplateMappingService, domain.CarrierMappingRepository, domain.IntegrationProfileRepository) {
	t.Helper()
	ctx := context.Background()
	profileRepo := infra.NewIntegrationProfileRepository(gdb)
	profile := &domain.IntegrationProfile{ProfileKey: "carrier-p0", SourceSurface: string(domain.SourceSurfaceMembership), DemandKind: string(domain.DemandKindMembershipEntitlement), RequiresCarrierMapping: true}
	if err := profileRepo.Create(ctx, profile); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	templateRepo := infra.NewDocumentTemplateRepository(gdb)
	template := &domain.DocumentTemplate{TemplateKey: "carrier-p0", DocumentType: "import_carrier_mapping", Format: "csv", MappingRules: `{"version":2,"mode":"header","hasHeader":true,"columns":{"carrier.internal_carrier_code":"Internal","carrier.external_carrier_code":"External","carrier.external_carrier_name":"Name"}}`}
	if err := templateRepo.Create(ctx, template); err != nil {
		t.Fatalf("create template: %v", err)
	}
	bindingRepo := infra.NewProfileTemplateBindingRepository(gdb)
	if err := bindingRepo.Create(ctx, &domain.IntegrationProfileTemplateBinding{IntegrationProfileID: profile.ID, DocumentType: template.DocumentType, TemplateID: template.ID, IsDefault: true}); err != nil {
		t.Fatalf("create binding: %v", err)
	}
	return NewTemplateMappingService(templateRepo, bindingRepo, profileRepo), infra.NewCarrierMappingRepository(gdb), profileRepo
}

func newCarrierPreflightUC(gdb *gorm.DB, mapping *TemplateMappingService, mappingRepo domain.CarrierMappingRepository, profileRepo domain.IntegrationProfileRepository, evidence *ImportEvidenceUseCase) CarrierMappingUseCase {
	registry := NewExternalCarrierUseCase(infra.NewExternalCarrierRepository(gdb))
	uc := NewCarrierMappingUseCase(mappingRepo, profileRepo)
	uc = WithCarrierImportDeps(uc, mapping)
	uc = WithCarrierImportEvidence(uc, evidence)
	uc = WithExternalCarrierRegistry(uc, registry)
	return WithCarrierConflictAudit(uc, registry)
}

func TestCarrierRejectAllPreflightHasZeroBusinessDeltaAndDurableConflictAudit(t *testing.T) {
	gdb := durableEvidenceDB(t)
	mapping, mappingRepo, profileRepo := seedCarrierImportDB(t, gdb)
	evidence := NewDeferredImportEvidenceUseCase(infra.NewImportEvidenceRepository(gdb))
	uc := newCarrierPreflightUC(gdb, mapping, mappingRepo, profileRepo, evidence)
	input := dto.ImportCarrierMappingsInput{IntegrationProfileID: 1, ImportMode: "reject_all", Rows: []map[string]string{
		{"Internal": "SF", "External": "same", "Name": "Name A"},
		{"Internal": "SF", "External": "same", "Name": "Name B"},
	}}
	plan, err := uc.PreflightCarrierMappings(context.Background(), input)
	if err != nil {
		t.Fatalf("PreflightCarrierMappings: %v", err)
	}
	if !plan.RejectsBusinessWrites() {
		t.Fatal("reject_all conflict plan did not block business writes")
	}
	result, err := uc.ExecuteCarrierImportPlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecuteCarrierImportPlan: %v", err)
	}
	if err := evidence.FinalizePending(context.Background()); err != nil {
		t.Fatalf("FinalizePending: %v", err)
	}
	if result.TotalProcessed != 2 || result.SuccessCount != 0 || result.ErrorCount != 2 || result.SuccessCount+result.ErrorCount != result.TotalProcessed {
		t.Fatalf("reject_all arithmetic = %+v", result)
	}
	var registryCount, mappingCount, conflictCount, rawCount int64
	for table, target := range map[string]*int64{"external_carriers": &registryCount, "carrier_mappings": &mappingCount, "external_carrier_conflicts": &conflictCount, "import_raw_records": &rawCount} {
		if err := gdb.Table(table).Count(target).Error; err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
	}
	if registryCount != 0 || mappingCount != 0 || conflictCount != 2 || rawCount != 2 {
		t.Fatalf("deltas registry=%d mapping=%d conflicts=%d raw=%d", registryCount, mappingCount, conflictCount, rawCount)
	}
}

func TestCarrierConflictAndRegistryPersistenceFailures(t *testing.T) {
	t.Run("conflict audit failure", func(t *testing.T) {
		gdb := durableEvidenceDB(t)
		mapping, mappingRepo, profileRepo := seedCarrierImportDB(t, gdb)
		if err := gdb.Exec(`CREATE TRIGGER fail_conflict BEFORE INSERT ON external_carrier_conflicts BEGIN SELECT RAISE(ABORT, 'injected conflict failure'); END`).Error; err != nil {
			t.Fatal(err)
		}
		evidence := NewDeferredImportEvidenceUseCase(infra.NewImportEvidenceRepository(gdb))
		uc := newCarrierPreflightUC(gdb, mapping, mappingRepo, profileRepo, evidence)
		_, err := uc.PreflightCarrierMappings(context.Background(), dto.ImportCarrierMappingsInput{IntegrationProfileID: 1, ImportMode: "reject_all", Rows: []map[string]string{{"Internal": "SF", "External": "same", "Name": "A"}, {"Internal": "SF", "External": "same", "Name": "B"}}})
		if err == nil {
			t.Fatal("expected conflict persistence failure")
		}
		if finalizeErr := evidence.FinalizeFailure(context.Background(), "failed", err); finalizeErr != nil {
			t.Fatalf("FinalizeFailure: %v", finalizeErr)
		}
		var conflicts, carriers, mappings int64
		_ = gdb.Table("external_carrier_conflicts").Count(&conflicts).Error
		_ = gdb.Table("external_carriers").Count(&carriers).Error
		_ = gdb.Table("carrier_mappings").Count(&mappings).Error
		if conflicts != 0 || carriers != 0 || mappings != 0 {
			t.Fatalf("failed conflict audit leaked writes: conflicts=%d carriers=%d mappings=%d", conflicts, carriers, mappings)
		}
	})

	t.Run("registry failure rolls business back", func(t *testing.T) {
		gdb := durableEvidenceDB(t)
		mapping, mappingRepo, profileRepo := seedCarrierImportDB(t, gdb)
		if err := gdb.Exec(`CREATE TRIGGER fail_registry BEFORE INSERT ON external_carriers BEGIN SELECT RAISE(ABORT, 'injected registry failure'); END`).Error; err != nil {
			t.Fatal(err)
		}
		evidence := NewDeferredImportEvidenceUseCase(infra.NewImportEvidenceRepository(gdb))
		uc := newCarrierPreflightUC(gdb, mapping, mappingRepo, profileRepo, evidence)
		plan, err := uc.PreflightCarrierMappings(context.Background(), dto.ImportCarrierMappingsInput{IntegrationProfileID: 1, ImportMode: "skip_invalid", Rows: []map[string]string{{"Internal": "SF", "External": "sf", "Name": "Carrier"}}})
		if err != nil {
			t.Fatal(err)
		}
		err = gdb.Transaction(func(tx *gorm.DB) error {
			txUC := NewCarrierMappingUseCase(infra.NewCarrierMappingRepository(tx), infra.NewIntegrationProfileRepository(tx))
			txUC = WithCarrierImportEvidence(txUC, evidence)
			txUC = WithExternalCarrierRegistry(txUC, NewExternalCarrierUseCase(infra.NewExternalCarrierRepository(tx)))
			_, executeErr := txUC.ExecuteCarrierImportPlan(context.Background(), plan)
			return executeErr
		})
		if err == nil {
			t.Fatal("expected registry failure")
		}
		if finalizeErr := evidence.FinalizeFailure(context.Background(), "failed", err); finalizeErr != nil {
			t.Fatalf("FinalizeFailure: %v", finalizeErr)
		}
		var carriers, mappings int64
		_ = gdb.Table("external_carriers").Count(&carriers).Error
		_ = gdb.Table("carrier_mappings").Count(&mappings).Error
		if carriers != 0 || mappings != 0 {
			t.Fatalf("registry failure leaked writes: carriers=%d mappings=%d", carriers, mappings)
		}
	})

	t.Run("mapping failure rolls registry provenance back", func(t *testing.T) {
		gdb := durableEvidenceDB(t)
		mapping, mappingRepo, profileRepo := seedCarrierImportDB(t, gdb)
		if err := gdb.Exec(`CREATE TRIGGER fail_mapping BEFORE INSERT ON carrier_mappings BEGIN SELECT RAISE(ABORT, 'injected mapping failure'); END`).Error; err != nil {
			t.Fatal(err)
		}
		evidence := NewDeferredImportEvidenceUseCase(infra.NewImportEvidenceRepository(gdb))
		uc := newCarrierPreflightUC(gdb, mapping, mappingRepo, profileRepo, evidence)
		plan, err := uc.PreflightCarrierMappings(context.Background(), dto.ImportCarrierMappingsInput{IntegrationProfileID: 1, ImportMode: "skip_invalid", Rows: []map[string]string{{"Internal": "SF", "External": "sf", "Name": "Carrier"}}})
		if err != nil {
			t.Fatal(err)
		}
		if plan.evidenceRun == nil || len(plan.evidenceRecords) != 1 {
			t.Fatalf("prepared evidence = run:%+v records:%+v", plan.evidenceRun, plan.evidenceRecords)
		}
		err = gdb.Transaction(func(tx *gorm.DB) error {
			txUC := NewCarrierMappingUseCase(infra.NewCarrierMappingRepository(tx), infra.NewIntegrationProfileRepository(tx))
			txUC = WithCarrierImportEvidence(txUC, evidence)
			txUC = WithExternalCarrierRegistry(txUC, NewExternalCarrierUseCase(infra.NewExternalCarrierRepository(tx)))
			_, executeErr := txUC.ExecuteCarrierImportPlan(context.Background(), plan)
			return executeErr
		})
		if err == nil {
			t.Fatal("expected mapping failure")
		}
		if finalizeErr := evidence.FinalizeFailure(context.Background(), "failed", err); finalizeErr != nil {
			t.Fatalf("FinalizeFailure: %v", finalizeErr)
		}
		var carriers, mappings int64
		_ = gdb.Table("external_carriers").Count(&carriers).Error
		_ = gdb.Table("carrier_mappings").Count(&mappings).Error
		if carriers != 0 || mappings != 0 {
			t.Fatalf("mapping rollback leaked registry provenance: carriers=%d mappings=%d", carriers, mappings)
		}
		records, listErr := infra.NewImportEvidenceRepository(gdb).ListRecordsByRun(context.Background(), plan.evidenceRun.ID)
		if listErr != nil || len(records) != 1 || records[0].Outcome != "failed" {
			t.Fatalf("rollback evidence records=%+v err=%v", records, listErr)
		}
	})
}

func TestCarrierImportRegistryProvenanceUsesFirstValidRawAndPreservesFirstSource(t *testing.T) {
	gdb := durableEvidenceDB(t)
	mapping, mappingRepo, profileRepo := seedCarrierImportDB(t, gdb)
	ctx := context.Background()
	input := dto.ImportCarrierMappingsInput{IntegrationProfileID: 1, ImportMode: "skip_invalid", Rows: []map[string]string{
		{"Internal": "SF", "External": "sf", "Name": "Carrier"},
		{"Internal": "SF", "External": "sf", "Name": "Carrier"},
	}}

	evidence := NewDeferredImportEvidenceUseCase(infra.NewImportEvidenceRepository(gdb))
	uc := newCarrierPreflightUC(gdb, mapping, mappingRepo, profileRepo, evidence)
	plan, err := uc.PreflightCarrierMappings(ctx, input)
	if err != nil {
		t.Fatalf("preflight exact duplicate observations: %v", err)
	}
	if plan.result.ErrorCount != 0 || plan.evidenceRun == nil || len(plan.evidenceRecords) != 2 {
		t.Fatalf("preflight result=%+v run=%+v records=%+v", plan.result, plan.evidenceRun, plan.evidenceRecords)
	}
	firstRunID := plan.evidenceRun.ID
	firstRawID := plan.evidenceRecords[0].ID
	if firstRunID == 0 || firstRawID == 0 || plan.evidenceRecords[1].ID == 0 {
		t.Fatalf("evidence IDs were not persisted: run=%d records=%+v", firstRunID, plan.evidenceRecords)
	}
	var result *dto.ImportCarrierMappingsResult
	err = gdb.Transaction(func(tx *gorm.DB) error {
		txUC := NewCarrierMappingUseCase(infra.NewCarrierMappingRepository(tx), infra.NewIntegrationProfileRepository(tx))
		txUC = WithCarrierImportEvidence(txUC, evidence)
		txUC = WithExternalCarrierRegistry(txUC, NewExternalCarrierUseCase(infra.NewExternalCarrierRepository(tx)))
		var executeErr error
		result, executeErr = txUC.ExecuteCarrierImportPlan(ctx, plan)
		return executeErr
	})
	if err != nil {
		t.Fatalf("execute exact duplicate observations: %v", err)
	}
	if err := evidence.FinalizePending(ctx); err != nil {
		t.Fatalf("finalize exact duplicate observations: %v", err)
	}
	if result.SuccessCount != 2 || result.ErrorCount != 0 || len(result.ExternalCarriers) != 1 {
		t.Fatalf("exact duplicate result=%+v", result)
	}

	registryRepo := infra.NewExternalCarrierRepository(gdb)
	carriers, err := registryRepo.ListByProfile(ctx, 1)
	if err != nil || len(carriers) != 1 {
		t.Fatalf("registry rows=%+v err=%v", carriers, err)
	}
	carrier := carriers[0]
	if carrier.SourceImportRunID == nil || *carrier.SourceImportRunID != firstRunID ||
		carrier.SourceRawRecordID == nil || *carrier.SourceRawRecordID != firstRawID {
		t.Fatalf("registry provenance=%+v want run=%d firstRaw=%d", carrier, firstRunID, firstRawID)
	}
	records, err := infra.NewImportEvidenceRepository(gdb).ListRecordsByRun(ctx, firstRunID)
	if err != nil || len(records) != 2 || records[0].ID != firstRawID || records[0].ImportRunID != firstRunID {
		t.Fatalf("source RAW reference is not durable: records=%+v err=%v", records, err)
	}

	secondEvidence := NewDeferredImportEvidenceUseCase(infra.NewImportEvidenceRepository(gdb))
	secondUC := newCarrierPreflightUC(gdb, mapping, mappingRepo, profileRepo, secondEvidence)
	secondPlan, err := secondUC.PreflightCarrierMappings(ctx, dto.ImportCarrierMappingsInput{IntegrationProfileID: 1, ImportMode: "skip_invalid", Rows: input.Rows[:1]})
	if err != nil {
		t.Fatalf("second preflight: %v", err)
	}
	err = gdb.Transaction(func(tx *gorm.DB) error {
		txUC := NewCarrierMappingUseCase(infra.NewCarrierMappingRepository(tx), infra.NewIntegrationProfileRepository(tx))
		txUC = WithCarrierImportEvidence(txUC, secondEvidence)
		txUC = WithExternalCarrierRegistry(txUC, NewExternalCarrierUseCase(infra.NewExternalCarrierRepository(tx)))
		_, executeErr := txUC.ExecuteCarrierImportPlan(ctx, secondPlan)
		return executeErr
	})
	if err != nil {
		t.Fatalf("second execute: %v", err)
	}
	if err := secondEvidence.FinalizePending(ctx); err != nil {
		t.Fatalf("second finalize: %v", err)
	}
	carriers, err = registryRepo.ListByProfile(ctx, 1)
	if err != nil || len(carriers) != 1 || carriers[0].SourceImportRunID == nil || *carriers[0].SourceImportRunID != firstRunID ||
		carriers[0].SourceRawRecordID == nil || *carriers[0].SourceRawRecordID != firstRawID {
		t.Fatalf("exact repeat replaced first provenance: carriers=%+v err=%v", carriers, err)
	}
}

func TestCarrierRepeatedImportIsIdempotent(t *testing.T) {
	gdb := durableEvidenceDB(t)
	mapping, mappingRepo, profileRepo := seedCarrierImportDB(t, gdb)
	input := dto.ImportCarrierMappingsInput{IntegrationProfileID: 1, ImportMode: "skip_invalid", Rows: []map[string]string{{"Internal": "SF", "External": "sf", "Name": "Carrier"}}}
	for attempt := 0; attempt < 2; attempt++ {
		evidence := NewDeferredImportEvidenceUseCase(infra.NewImportEvidenceRepository(gdb))
		uc := newCarrierPreflightUC(gdb, mapping, mappingRepo, profileRepo, evidence)
		plan, err := uc.PreflightCarrierMappings(context.Background(), input)
		if err != nil {
			t.Fatalf("attempt %d preflight: %v", attempt, err)
		}
		if plan.RejectsBusinessWrites() {
			t.Fatalf("attempt %d clean plan unexpectedly rejected business writes", attempt)
		}
		var result *dto.ImportCarrierMappingsResult
		err = gdb.Transaction(func(tx *gorm.DB) error {
			txUC := NewCarrierMappingUseCase(infra.NewCarrierMappingRepository(tx), infra.NewIntegrationProfileRepository(tx))
			txUC = WithCarrierImportEvidence(txUC, evidence)
			txUC = WithExternalCarrierRegistry(txUC, NewExternalCarrierUseCase(infra.NewExternalCarrierRepository(tx)))
			var executeErr error
			result, executeErr = txUC.ExecuteCarrierImportPlan(context.Background(), plan)
			return executeErr
		})
		if err != nil {
			t.Fatalf("attempt %d execute: %v", attempt, err)
		}
		if err := evidence.FinalizePending(context.Background()); err != nil {
			t.Fatalf("attempt %d finalize: %v", attempt, err)
		}
		if result.SuccessCount != 1 || result.ErrorCount != 0 || result.SuccessCount+result.ErrorCount != result.TotalProcessed {
			t.Fatalf("attempt %d arithmetic: %+v", attempt, result)
		}
	}
	var carriers, mappings int64
	_ = gdb.Table("external_carriers").Count(&carriers).Error
	_ = gdb.Table("carrier_mappings").Count(&mappings).Error
	if carriers != 1 || mappings != 1 {
		t.Fatalf("repeated import counts carriers=%d mappings=%d", carriers, mappings)
	}
}
