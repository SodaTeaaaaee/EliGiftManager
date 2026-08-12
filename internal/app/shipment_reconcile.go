package app

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// shipmentReconcileDeps holds optional collaborators for MapAndReconcileShipments.
type shipmentReconcileDeps struct {
	templateMapping *TemplateMappingService
	productRepo     domain.ProductRepository
	masterRepo      domain.ProductMasterRepository // optional; enables supplier_product_ref index
	addressRepo     domain.CustomerAddressRepository
	carrierUC       CarrierMappingUseCase
}

// WithShipmentReconcileDeps attaches template/product/address/carrier collaborators.
// masterRepo may be nil; supplier_product_ref match tiers then stay empty
// (factory_sku tiers still work).
func WithShipmentReconcileDeps(
	uc ShipmentImportUseCase,
	mapping *TemplateMappingService,
	productRepo domain.ProductRepository,
	masterRepo domain.ProductMasterRepository,
	addressRepo domain.CustomerAddressRepository,
	carrierUC CarrierMappingUseCase,
) ShipmentImportUseCase {
	s, ok := uc.(*shipmentImportUseCase)
	if !ok {
		return uc
	}
	s.reconcile = &shipmentReconcileDeps{
		templateMapping: mapping,
		productRepo:     productRepo,
		masterRepo:      masterRepo,
		addressRepo:     addressRepo,
		carrierUC:       carrierUC,
	}
	return s
}

// MapAndReconcileShipments maps external factory-return rows via the profile's
// import_supplier_shipment template, reconciles each row to an internal
// FulfillmentLine (and its SupplierOrderLine), then delegates to ImportShipments.
//
// Match priority (first unique hit wins; ambiguity / zero hits → row error):
//  1. third_party_order_no / external_key == FulfillmentLine.ID (decimal)
//  2. factory_sku + phone unique within the wave
//  3. factory_sku + recipient_name unique within the wave
//  4. normalized factory_sku ref (product.factory_sku with an optional generic
//     alphabetic namespace stripped, e.g. "VENDOR_1001" -> "1001") + phone
//  5. normalized factory_sku ref + recipient_name unique
//  6. supplier_product_ref + phone unique within the wave (fallback tier;
//     kept for profiles whose token genuinely carries a supplier_product_ref)
//  7. supplier_product_ref + recipient_name unique within the wave (fallback)
//
// Multi-SKU rows (shipment.sku_quantity with pipe-separated tokens) expand one
// physical source row into N ImportShipmentEntry candidates. Any token that is
// ambiguous or unmatched fails the whole physical row (no guessing).
//
// Token interpretation and product matching are independent configuration
// contracts. This use case never infers a relationship between a shipment file
// and a catalog file. The normalized factory-SKU tier is a platform-neutral
// compatibility convention; supplier_product_ref remains an independent
// fallback. Ambiguous matches always fail instead of guessing.
func (uc *shipmentImportUseCase) MapAndReconcileShipments(ctx context.Context, input dto.MapAndReconcileShipmentsInput) (*dto.ImportShipmentResult, error) {
	if uc.reconcile == nil || uc.reconcile.templateMapping == nil {
		return nil, fmt.Errorf("map and reconcile shipments: reconcile deps not configured")
	}
	mode := input.ImportMode
	if mode == "" {
		mode = "skip_invalid"
	}
	if mode != "reject_all" && mode != "skip_invalid" {
		return nil, fmt.Errorf("invalid importMode %q", mode)
	}
	if input.WaveID == 0 {
		return nil, fmt.Errorf("waveId is required")
	}
	if input.IntegrationProfileID == 0 {
		return nil, fmt.Errorf("integrationProfileId is required")
	}

	_, rules, err := uc.reconcile.templateMapping.ResolveTemplateAndRules(ctx, input.IntegrationProfileID, "import_supplier_shipment")
	if err != nil {
		return nil, fmt.Errorf("template pipeline: %w", err)
	}

	orderedRows, headers, headerRows, total, _, cleanup, err := loadImportRows(input.FilePath, input.Rows, rules, nil)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return nil, err
	}
	var evidenceRun *domain.ImportRun
	var evidenceRecords []domain.ImportRawRecord
	if uc.evidence != nil {
		evidenceRows, unmapped := importEvidenceRows(orderedRows, headers, headerRows)
		parserMetadata := fmt.Sprintf(`{"hasHeader":%t,"sheetName":%q}`, rules.HasHeader, rules.SheetName)
		evidenceRun, evidenceRecords, err = uc.evidence.StartImportEvidence(ctx, "supplier_shipment", input.IntegrationProfileID, mode, input.FilePath, parserMetadata, evidenceRows, unmapped, nil)
		if err != nil {
			return nil, fmt.Errorf("start shipment import evidence: %w", err)
		}
	}

	index, err := uc.buildReconcileIndex(ctx, input.WaveID)
	if err != nil {
		return nil, err
	}

	var entries []dto.ImportShipmentEntry
	var compactSourceRows []int
	type rowPlan struct {
		sourceRow           int
		entries             []dto.ImportShipmentEntry
		compactEntryIndices []int
		externalCarrierCode string
		externalCarrierName string
	}
	var rowPlans []rowPlan
	var mapErrors []dto.ImportShipmentError
	var rowWarnings rowWarningCollector

	for i := 0; i < total; i++ {
		var applied map[string]string
		var mapErr error
		var warnings []string
		if len(orderedRows) > 0 {
			applied, warnings, mapErr = ApplyRow(orderedRows[i], headers, rules)
		} else {
			applied, warnings, mapErr = applyHeaderMap(headerRows[i], rules)
		}
		rowWarnings.add(i, warnings)
		if mapErr != nil {
			markImportEvidenceFailure(evidenceRecords, i, "mapping_error", mapErr.Error(), warnings)
			mapErrors = append(mapErrors, dto.ImportShipmentError{EntryIndex: i, Reason: mapErr.Error()})
			continue
		}

		rowEntries, recErr := reconcileShipmentRow(applied, index, uc.reconcile.carrierUC, ctx, input.IntegrationProfileID)
		if recErr != nil {
			markImportEvidenceFailure(evidenceRecords, i, "reconcile_error", recErr.Error(), warnings)
			mapErrors = append(mapErrors, dto.ImportShipmentError{EntryIndex: i, Reason: recErr.Error()})
			continue
		}
		plan := rowPlan{
			sourceRow:           i,
			entries:             rowEntries,
			compactEntryIndices: make([]int, len(rowEntries)),
			externalCarrierCode: shipmentMappedValue(applied, "shipment.carrier_code", "shipment.carrier"),
			externalCarrierName: shipmentMappedValue(applied, "shipment.carrier_name"),
		}
		for entryIndex := range rowEntries {
			plan.compactEntryIndices[entryIndex] = len(entries) + entryIndex
		}
		rowPlans = append(rowPlans, plan)
		entries = append(entries, rowEntries...)
		for range rowEntries {
			compactSourceRows = append(compactSourceRows, i)
		}
	}

	if mode == "reject_all" && len(mapErrors) > 0 {
		for _, plan := range rowPlans {
			markImportEvidenceFailure(evidenceRecords, plan.sourceRow, "import_rejected", "file rejected because one or more rows failed mapping or reconciliation", nil)
		}
		result := &dto.ImportShipmentResult{
			ImportRunID:      importEvidenceRunID(evidenceRun),
			EvidenceDisabled: uc.evidence != nil && evidenceRun == nil,
			TotalProcessed:   total,
			SuccessCount:     0,
			ErrorCount:       len(mapErrors),
			Errors:           mapErrors,
			Warnings:         rowWarnings.warnings(),
		}
		if evidenceRun != nil {
			if err := uc.evidence.CompleteImportEvidence(ctx, evidenceRun, evidenceRecords, "rejected"); err != nil {
				return nil, err
			}
		}
		return result, nil
	}
	if len(entries) == 0 {
		result := &dto.ImportShipmentResult{
			ImportRunID:      importEvidenceRunID(evidenceRun),
			EvidenceDisabled: uc.evidence != nil && evidenceRun == nil,
			TotalProcessed:   total,
			SuccessCount:     0,
			ErrorCount:       len(mapErrors),
			Errors:           mapErrors,
			Warnings:         rowWarnings.warnings(),
		}
		if evidenceRun != nil {
			if err := uc.evidence.CompleteImportEvidence(ctx, evidenceRun, evidenceRecords, "failed"); err != nil {
				return nil, err
			}
		}
		return result, nil
	}

	outcome, err := uc.importShipmentsCoreWithOutcome(ctx, dto.ImportShipmentInput{
		WaveID:               input.WaveID,
		IntegrationProfileID: input.IntegrationProfileID,
		ImportMode:           mode,
		Entries:              entries,
	})
	if err != nil {
		return nil, err
	}
	result := outcome.result

	// The scan/reconcile pass above is deliberately write-free. Observe a
	// carrier only after the whole-file reject_all gate and the shipment core
	// have accepted the physical row. The controller supplies transaction-bound
	// repositories, so observation and all shipment business writes roll back
	// together on this or any later error.
	for _, plan := range rowPlans {
		rowSucceeded := len(plan.compactEntryIndices) > 0
		for _, entryIndex := range plan.compactEntryIndices {
			if _, ok := outcome.successfulEntryIndices[entryIndex]; !ok {
				rowSucceeded = false
				break
			}
		}
		if !rowSucceeded {
			markShipmentPlanEvidenceFailure(evidenceRecords, plan.sourceRow, plan.compactEntryIndices, result.Errors)
			continue
		}
		if uc.externalRegistry != nil && len(plan.entries) > 0 && (plan.externalCarrierCode != "" || plan.externalCarrierName != "") {
			observation := ExternalCarrierObservationInput{
				IntegrationProfileID: input.IntegrationProfileID,
				ExternalCarrierCode:  plan.externalCarrierCode,
				ExternalCarrierName:  plan.externalCarrierName,
			}
			if evidenceRun != nil && plan.sourceRow >= 0 && plan.sourceRow < len(evidenceRecords) {
				runID := evidenceRun.ID
				rawRecordID := evidenceRecords[plan.sourceRow].ID
				if runID != 0 {
					observation.SourceImportRunID = &runID
				}
				if rawRecordID != 0 {
					observation.SourceRawRecordID = &rawRecordID
				}
			}
			if _, observeErr := uc.externalRegistry.ObserveExternalCarrierWithProvenance(ctx, observation); observeErr != nil {
				return nil, fmt.Errorf("register shipment carrier row %d: %w", plan.sourceRow, observeErr)
			}
		}
		markImportEvidenceSuccess(evidenceRecords, plan.sourceRow, "shipment", 0)
	}
	result.EvidenceDisabled = uc.evidence != nil && evidenceRun == nil
	// Merge pre-import mapping errors into the result (row indices refer to source rows;
	// ImportShipments errors refer to the compacted entries slice — re-base them).
	for i := range result.Errors {
		if result.Errors[i].EntryIndex >= 0 && result.Errors[i].EntryIndex < len(compactSourceRows) {
			result.Errors[i].EntryIndex = compactSourceRows[result.Errors[i].EntryIndex]
		}
	}
	if len(mapErrors) > 0 {
		result.Errors = append(mapErrors, result.Errors...)
		result.ErrorCount = len(result.Errors)
		result.TotalProcessed = total
	} else {
		result.TotalProcessed = total
	}
	// Merge pre-import mapping warnings (ImportShipments itself never maps rows,
	// so result.Warnings is currently always empty, but merge defensively in
	// case that changes). Always assign a non-nil slice so the JSON field is
	// `[]` rather than `null` when there is nothing to report.
	result.Warnings = append(rowWarnings.warnings(), result.Warnings...)
	if evidenceRun != nil {
		result.ImportRunID = evidenceRun.ID
		status := "completed"
		if result.ErrorCount > 0 {
			status = "partial_success"
		}
		if mode == "reject_all" && result.ErrorCount > 0 {
			status = "rejected"
		}
		if err := uc.evidence.CompleteImportEvidence(ctx, evidenceRun, evidenceRecords, status); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func markShipmentPlanEvidenceFailure(records []domain.ImportRawRecord, sourceRow int, compactEntryIndices []int, importErrors []dto.ImportShipmentError) {
	reason := "shipment row was not persisted"
	for _, itemErr := range importErrors {
		for _, compactEntryIndex := range compactEntryIndices {
			if itemErr.EntryIndex == compactEntryIndex {
				reason = itemErr.Reason
				break
			}
		}
		if reason != "shipment row was not persisted" {
			break
		}
	}
	markImportEvidenceFailure(records, sourceRow, "shipment_import_error", reason, nil)
}

func shipmentMappedValue(applied map[string]string, keys ...string) string {
	for _, key := range keys {
		if value, ok := applied[key]; ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
		if strings.HasPrefix(key, "shipment.") {
			bare := strings.TrimPrefix(key, "shipment.")
			if value, ok := applied[bare]; ok && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

// reconcileIndex is a wave-scoped lookup structure for external-key matching.
type reconcileIndex struct {
	byFLID             map[uint]*reconcileCandidate
	bySKUPhone         map[string][]*reconcileCandidate
	bySKURecipient     map[string][]*reconcileCandidate
	byDerivedPhone     map[string][]*reconcileCandidate   // derived factory_sku ref (see deriveFactorySKURef) + phone
	byDerivedRecipient map[string][]*reconcileCandidate   // derived factory_sku ref + recipient_name
	byRefPhone         map[string][]*reconcileCandidate   // supplier_product_ref + phone (fallback tier, kept for compat)
	byRefRecipient     map[string][]*reconcileCandidate   // supplier_product_ref + recipient_name (fallback tier)
	solByFulfillment   map[uint]*domain.SupplierOrderLine // FL ID → first SOL
}

type reconcileCandidate struct {
	line               *domain.FulfillmentLine
	factorySKU         string
	derivedRef         string // factorySKU with its platform prefix (e.g. "ROUZAO_") stripped
	supplierProductRef string
	phone              string
	recipientName      string
}

func (uc *shipmentImportUseCase) buildReconcileIndex(ctx context.Context, waveID uint) (*reconcileIndex, error) {
	lines, err := uc.fulfillRepo.ListByWave(ctx, waveID)
	if err != nil {
		return nil, fmt.Errorf("list fulfillment lines for wave %d: %w", waveID, err)
	}

	// Product cache: ProductID → FactorySKU / SupplierProductRef
	type productInfo struct {
		factorySKU         string
		supplierProductRef string
	}
	infoByProduct := map[uint]productInfo{}
	if uc.reconcile.productRepo != nil {
		products, pErr := uc.reconcile.productRepo.ListByWave(ctx, waveID)
		if pErr != nil {
			return nil, fmt.Errorf("list products for wave %d: %w", waveID, pErr)
		}

		// Optional master lookup: Product.ProductMasterID → ProductMaster.SupplierProductRef.
		refByMasterID := map[uint]string{}
		if uc.reconcile.masterRepo != nil {
			masters, mErr := uc.reconcile.masterRepo.List(ctx)
			if mErr != nil {
				return nil, fmt.Errorf("list product masters for reconcile: %w", mErr)
			}
			for i := range masters {
				if masters[i].SupplierProductRef != "" {
					refByMasterID[masters[i].ID] = masters[i].SupplierProductRef
				}
			}
		}

		for i := range products {
			p := &products[i]
			info := productInfo{factorySKU: p.FactorySKU}
			if p.ProductMasterID != nil {
				info.supplierProductRef = refByMasterID[*p.ProductMasterID]
			}
			infoByProduct[p.ID] = info
		}
	}

	// Address cache: AddressID → phone/name
	type addrInfo struct{ phone, name string }
	addrByID := map[uint]addrInfo{}

	idx := &reconcileIndex{
		byFLID:             make(map[uint]*reconcileCandidate, len(lines)),
		bySKUPhone:         make(map[string][]*reconcileCandidate),
		bySKURecipient:     make(map[string][]*reconcileCandidate),
		byDerivedPhone:     make(map[string][]*reconcileCandidate),
		byDerivedRecipient: make(map[string][]*reconcileCandidate),
		byRefPhone:         make(map[string][]*reconcileCandidate),
		byRefRecipient:     make(map[string][]*reconcileCandidate),
		solByFulfillment:   make(map[uint]*domain.SupplierOrderLine),
	}

	for i := range lines {
		fl := &lines[i]
		cand := &reconcileCandidate{line: fl}
		if fl.ProductID != nil {
			if info, ok := infoByProduct[*fl.ProductID]; ok {
				cand.factorySKU = info.factorySKU
				cand.derivedRef = deriveFactorySKURef(info.factorySKU)
				cand.supplierProductRef = info.supplierProductRef
			}
		}
		if fl.CustomerAddressID != nil && uc.reconcile.addressRepo != nil {
			aid := *fl.CustomerAddressID
			if info, ok := addrByID[aid]; ok {
				cand.phone = info.phone
				cand.recipientName = info.name
			} else if addr, aErr := uc.reconcile.addressRepo.FindByID(ctx, aid); aErr == nil && addr != nil {
				addrByID[aid] = addrInfo{phone: addr.Phone, name: addr.RecipientName}
				cand.phone = addr.Phone
				cand.recipientName = addr.RecipientName
			}
		}
		idx.byFLID[fl.ID] = cand
		if cand.factorySKU != "" && cand.phone != "" {
			key := skuPhoneKey(cand.factorySKU, cand.phone)
			idx.bySKUPhone[key] = append(idx.bySKUPhone[key], cand)
		}
		if cand.factorySKU != "" && cand.recipientName != "" {
			key := skuRecipientKey(cand.factorySKU, cand.recipientName)
			idx.bySKURecipient[key] = append(idx.bySKURecipient[key], cand)
		}
		if cand.derivedRef != "" && cand.phone != "" {
			key := refPhoneKey(cand.derivedRef, cand.phone)
			idx.byDerivedPhone[key] = append(idx.byDerivedPhone[key], cand)
		}
		if cand.derivedRef != "" && cand.recipientName != "" {
			key := refRecipientKey(cand.derivedRef, cand.recipientName)
			idx.byDerivedRecipient[key] = append(idx.byDerivedRecipient[key], cand)
		}
		if cand.supplierProductRef != "" && cand.phone != "" {
			key := refPhoneKey(cand.supplierProductRef, cand.phone)
			idx.byRefPhone[key] = append(idx.byRefPhone[key], cand)
		}
		if cand.supplierProductRef != "" && cand.recipientName != "" {
			key := refRecipientKey(cand.supplierProductRef, cand.recipientName)
			idx.byRefRecipient[key] = append(idx.byRefRecipient[key], cand)
		}
	}

	// Build FL → SOL map from supplier orders on the wave.
	orders, err := uc.supplierRepo.ListByWave(ctx, waveID)
	if err != nil {
		return nil, fmt.Errorf("list supplier orders for wave %d: %w", waveID, err)
	}
	for i := range orders {
		sols, lErr := uc.supplierRepo.ListLinesByOrder(ctx, orders[i].ID)
		if lErr != nil {
			return nil, fmt.Errorf("list SOLs for order %d: %w", orders[i].ID, lErr)
		}
		for j := range sols {
			sol := sols[j]
			// First SOL wins if multiple (unusual); ImportShipments validates membership.
			if _, exists := idx.solByFulfillment[sol.FulfillmentLineID]; !exists {
				cp := sol
				idx.solByFulfillment[sol.FulfillmentLineID] = &cp
			}
		}
	}

	return idx, nil
}

// SKUQuantityToken is one segment of a multi-SKU factory-return cell
// (e.g. shipment.sku_quantity ← 规格&数量).
type SKUQuantityToken struct {
	// SupplierProductRef is the numeric prefix of the segment before the first '_'.
	SupplierProductRef string
	Quantity           int
}

// ParseSKUQuantityTokens parses a pipe-separated multi-SKU cell of the form:
//
//	<token> * <qty>|<token> * <qty>|...
//
// where token's supplier_product_ref is the digit run from the start up to the
// first '_' (the rest of the token is a human label and is ignored).
//
// Failures never guess: empty segments, missing '*', non-integer qty, qty<=0,
// or a token without a leading digit-ref all return an error.
func ParseSKUQuantityTokens(raw string) ([]SKUQuantityToken, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("sku_quantity is empty")
	}
	parts := strings.Split(raw, "|")
	out := make([]SKUQuantityToken, 0, len(parts))
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("sku_quantity segment %d is empty", i)
		}
		// Split on " * " (space-star-space) first; fall back to last " *".
		tokenPart, qtyPart, ok := splitTokenQty(part)
		if !ok {
			return nil, fmt.Errorf("sku_quantity segment %d %q: expected \"<token> * <qty>\"", i, part)
		}
		ref := supplierProductRefFromToken(tokenPart)
		if ref == "" {
			return nil, fmt.Errorf("sku_quantity segment %d %q: token has no numeric supplier_product_ref prefix before '_'", i, part)
		}
		qty, err := strconv.Atoi(strings.TrimSpace(qtyPart))
		if err != nil {
			return nil, fmt.Errorf("sku_quantity segment %d %q: invalid quantity %q", i, part, qtyPart)
		}
		if qty <= 0 {
			return nil, fmt.Errorf("sku_quantity segment %d %q: quantity must be positive, got %d", i, part, qty)
		}
		out = append(out, SKUQuantityToken{SupplierProductRef: ref, Quantity: qty})
	}
	return out, nil
}

// splitTokenQty splits "TOKEN * QTY". Accepts " * " or a trailing " *N" form.
func splitTokenQty(part string) (token, qty string, ok bool) {
	// Prefer the last occurrence of " * " so labels containing "*" don't break us.
	if idx := strings.LastIndex(part, " * "); idx >= 0 {
		return strings.TrimSpace(part[:idx]), strings.TrimSpace(part[idx+3:]), true
	}
	// Also accept "* " with optional surrounding spaces: "TOKEN* 1" / "TOKEN *1".
	if idx := strings.LastIndex(part, "*"); idx > 0 {
		left := strings.TrimSpace(part[:idx])
		right := strings.TrimSpace(part[idx+1:])
		if left != "" && right != "" {
			return left, right, true
		}
	}
	return "", "", false
}

// factorySKUPlatformPrefix matches a leading letters+underscore platform prefix
// on a FactorySKU value, e.g. "ROUZAO_" in "ROUZAO_206068021". It is
// intentionally generic (any ASCII-letter run followed by '_') rather than
// hard-coding "ROUZAO_" so other supplier platforms with a similar
// "<PLATFORM>_<digits>" FactorySKU shape are handled the same way.
var factorySKUPlatformPrefix = regexp.MustCompile(`^[A-Za-z]+_`)

// deriveFactorySKURef applies the platform-neutral compatibility convention of
// stripping a leading "<letters>_" namespace from FactorySKU. It does not infer
// or assert any relationship with a separate product-catalog input.
func deriveFactorySKURef(factorySKU string) string {
	s := strings.TrimSpace(factorySKU)
	if s == "" {
		return ""
	}
	return factorySKUPlatformPrefix.ReplaceAllString(s, "")
}

// supplierProductRefFromToken extracts the digit run from the start of a token
// up to (but not including) the first '_'. Returns "" if the prefix is empty
// or not purely numeric (no guessing).
func supplierProductRefFromToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	// Digit prefix through first '_', or the whole token if it is pure digits.
	prefix := token
	if i := strings.IndexByte(token, '_'); i >= 0 {
		prefix = token[:i]
	}
	if prefix == "" {
		return ""
	}
	for _, r := range prefix {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return prefix
}

func reconcileShipmentRow(
	applied map[string]string,
	idx *reconcileIndex,
	carrierUC CarrierMappingUseCase,
	ctx context.Context,
	profileID uint,
) ([]dto.ImportShipmentEntry, error) {
	get := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := applied[k]; ok && strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
			// Also try without shipment. prefix for unprefixed templates.
			if strings.HasPrefix(k, "shipment.") {
				bare := strings.TrimPrefix(k, "shipment.")
				if v, ok := applied[bare]; ok && strings.TrimSpace(v) != "" {
					return strings.TrimSpace(v)
				}
			}
		}
		return ""
	}

	externalKey := get("shipment.third_party_order_no", "shipment.external_key", "shipment.fulfillment_line_id")
	factorySKU := get("shipment.factory_sku", "shipment.sku")
	supplierProductRef := get("shipment.supplier_product_ref")
	phone := get("shipment.phone", "shipment.recipient_phone")
	recipientName := get("shipment.recipient_name", "shipment.name")
	trackingNo := get("shipment.tracking_no", "shipment.tracking")
	carrierCode := get("shipment.carrier_code", "shipment.carrier")
	carrierName := get("shipment.carrier_name")
	externalShipmentNo := get("shipment.external_shipment_no")
	qtyStr := get("shipment.quantity", "shipment.qty")
	skuQuantityRaw := get("shipment.sku_quantity", "shipment.spec_quantity")
	shippedAtRaw := get("shipment.shipped_at")

	shippedAt, err := parseTimeLoose(shippedAtRaw)
	if err != nil {
		return nil, fmt.Errorf("invalid shipped_at %q: %w", shippedAtRaw, err)
	}

	// Resolve carrier: factory returns may carry either an external code, an
	// alias, or only the external display name. Persist the internal code while
	// retaining the canonical external name when a mapping is found.
	carrierLookup := carrierCode
	if carrierLookup == "" {
		carrierLookup = carrierName
	}
	if carrierUC != nil && carrierLookup != "" {
		if internal, extName, rErr := carrierUC.ResolveByExternalOrAlias(ctx, profileID, carrierLookup); rErr == nil {
			carrierCode = internal
			if extName != "" {
				carrierName = extName
			}
		}
		// else: pass through unmapped codes as-is (ImportShipments does not require mapping).
	}

	// Shared entry shell (tracking / carrier / shipped_at / external shipment no).
	base := dto.ImportShipmentEntry{
		ExternalShipmentNo: externalShipmentNo,
		CarrierCode:        carrierCode,
		CarrierName:        carrierName,
		TrackingNo:         trackingNo,
		ShippedAt:          shippedAt,
	}

	// Multi-SKU expand path: shipment.sku_quantity present → N candidates.
	// External-key single match still takes precedence (FL-level identity).
	if skuQuantityRaw != "" && externalKey == "" {
		tokens, pErr := ParseSKUQuantityTokens(skuQuantityRaw)
		if pErr != nil {
			return nil, pErr
		}
		entries := make([]dto.ImportShipmentEntry, 0, len(tokens))
		for i, tok := range tokens {
			cand, matchErr := matchReconcileCandidate(idx, "", "", tok.SupplierProductRef, phone, recipientName)
			if matchErr != nil {
				return nil, fmt.Errorf("sku_quantity token %d ref=%q: %w", i, tok.SupplierProductRef, matchErr)
			}
			sol, ok := idx.solByFulfillment[cand.line.ID]
			if !ok || sol == nil {
				return nil, fmt.Errorf("sku_quantity token %d ref=%q: fulfillment line %d has no supplier order line on this wave", i, tok.SupplierProductRef, cand.line.ID)
			}
			e := base
			e.SupplierOrderLineID = sol.ID
			e.FulfillmentLineID = cand.line.ID
			e.Quantity = tok.Quantity
			entries = append(entries, e)
		}
		return entries, nil
	}

	// Single-row path (legacy + external key).
	qty, err := parseIntLoose(qtyStr)
	if err != nil {
		return nil, fmt.Errorf("invalid quantity %q: %w", qtyStr, err)
	}
	if qty <= 0 {
		qty = 1 // factory returns typically ship the full line qty; default 1
	}

	// If sku_quantity is present alongside external_key, still honor external_key
	// as the single-row identity (do not expand). qty stays from shipment.quantity.

	cand, matchErr := matchReconcileCandidate(idx, externalKey, factorySKU, supplierProductRef, phone, recipientName)
	if matchErr != nil {
		return nil, matchErr
	}

	sol, ok := idx.solByFulfillment[cand.line.ID]
	if !ok || sol == nil {
		return nil, fmt.Errorf("fulfillment line %d has no supplier order line on this wave", cand.line.ID)
	}

	e := base
	e.SupplierOrderLineID = sol.ID
	e.FulfillmentLineID = cand.line.ID
	e.Quantity = qty
	return []dto.ImportShipmentEntry{e}, nil
}

// matchReconcileCandidate resolves one shipment row/token to a unique
// reconcileCandidate. refToken is the raw REF value carried by the source row
// (either an explicit shipment.supplier_product_ref field, or the digit prefix
// parsed from a shipment.sku_quantity token by ParseSKUQuantityTokens) — see
// the tier ordering below for how it is actually resolved against product data.
func matchReconcileCandidate(idx *reconcileIndex, externalKey, factorySKU, refToken, phone, recipientName string) (*reconcileCandidate, error) {
	// Priority 1: external key == FulfillmentLine.ID decimal
	if externalKey != "" {
		if id, err := strconv.ParseUint(externalKey, 10, 64); err == nil {
			if cand, ok := idx.byFLID[uint(id)]; ok {
				return cand, nil
			}
			return nil, fmt.Errorf("no fulfillment line with id %s", externalKey)
		}
		// Non-numeric external keys fall through to SKU-based match.
	}

	// Priority 2: factory_sku + phone unique
	if factorySKU != "" && phone != "" {
		hits := idx.bySKUPhone[skuPhoneKey(factorySKU, phone)]
		switch len(hits) {
		case 1:
			return hits[0], nil
		case 0:
			// fall through
		default:
			return nil, fmt.Errorf("ambiguous match for factory_sku=%q phone=%q (%d hits: %s)", factorySKU, phone, len(hits), formatHitIDs(hits))
		}
	}

	// Priority 3: factory_sku + recipient_name unique
	if factorySKU != "" && recipientName != "" {
		hits := idx.bySKURecipient[skuRecipientKey(factorySKU, recipientName)]
		switch len(hits) {
		case 1:
			return hits[0], nil
		case 0:
			// fall through to derived-ref / supplier_product_ref tiers
		default:
			return nil, fmt.Errorf("ambiguous match for factory_sku=%q recipient_name=%q (%d hits: %s)", factorySKU, recipientName, len(hits), formatHitIDs(hits))
		}
	}

	// Priority 4: derived factory_sku ref (product.factory_sku with its
	// platform prefix stripped, see deriveFactorySKURef) + phone unique.
	// Primary tier for shipment.sku_quantity multi-SKU tokens.
	if refToken != "" && phone != "" {
		hits := idx.byDerivedPhone[refPhoneKey(refToken, phone)]
		switch len(hits) {
		case 1:
			return hits[0], nil
		case 0:
			// fall through
		default:
			return nil, fmt.Errorf("ambiguous match for factory_sku_ref=%q phone=%q (%d hits: %s)", refToken, phone, len(hits), formatHitIDs(hits))
		}
	}

	// Priority 5: derived factory_sku ref + recipient_name unique
	if refToken != "" && recipientName != "" {
		hits := idx.byDerivedRecipient[refRecipientKey(refToken, recipientName)]
		switch len(hits) {
		case 1:
			return hits[0], nil
		case 0:
			// fall through to supplier_product_ref fallback tiers
		default:
			return nil, fmt.Errorf("ambiguous match for factory_sku_ref=%q recipient_name=%q (%d hits: %s)", refToken, recipientName, len(hits), formatHitIDs(hits))
		}
	}

	// Priority 6: supplier_product_ref + phone unique (independent fallback tier
	// for profiles whose REF token is a supplier_product_ref value).
	if refToken != "" && phone != "" {
		hits := idx.byRefPhone[refPhoneKey(refToken, phone)]
		switch len(hits) {
		case 1:
			return hits[0], nil
		case 0:
			// fall through
		default:
			return nil, fmt.Errorf("ambiguous match for supplier_product_ref=%q phone=%q (%d hits: %s)", refToken, phone, len(hits), formatHitIDs(hits))
		}
	}

	// Priority 7: supplier_product_ref + recipient_name unique (fallback tier)
	if refToken != "" && recipientName != "" {
		hits := idx.byRefRecipient[refRecipientKey(refToken, recipientName)]
		switch len(hits) {
		case 1:
			return hits[0], nil
		case 0:
			return nil, fmt.Errorf("no match for supplier_product_ref=%q recipient_name=%q", refToken, recipientName)
		default:
			return nil, fmt.Errorf("ambiguous match for supplier_product_ref=%q recipient_name=%q (%d hits: %s)", refToken, recipientName, len(hits), formatHitIDs(hits))
		}
	}

	// Exhausted fall-through zero-hit messages (most specific first).
	if factorySKU != "" && phone != "" {
		return nil, fmt.Errorf("no match for factory_sku=%q phone=%q", factorySKU, phone)
	}
	if factorySKU != "" && recipientName != "" {
		return nil, fmt.Errorf("no match for factory_sku=%q recipient_name=%q", factorySKU, recipientName)
	}
	if refToken != "" && phone != "" {
		return nil, fmt.Errorf("no match for supplier_product_ref=%q phone=%q", refToken, phone)
	}
	return nil, fmt.Errorf("insufficient match keys (need third_party_order_no or factory_sku/supplier_product_ref + phone/recipient_name)")
}

func formatHitIDs(hits []*reconcileCandidate) string {
	if len(hits) == 0 {
		return "fl_ids=[]"
	}
	parts := make([]string, 0, len(hits))
	for _, h := range hits {
		if h != nil && h.line != nil {
			parts = append(parts, strconv.FormatUint(uint64(h.line.ID), 10))
		}
	}
	return "fl_ids=[" + strings.Join(parts, ",") + "]"
}

func skuPhoneKey(sku, phone string) string {
	return strings.ToLower(strings.TrimSpace(sku)) + "\x00" + strings.TrimSpace(phone)
}

func skuRecipientKey(sku, name string) string {
	return strings.ToLower(strings.TrimSpace(sku)) + "\x00" + strings.ToLower(strings.TrimSpace(name))
}

func refPhoneKey(ref, phone string) string {
	return strings.TrimSpace(ref) + "\x00" + strings.TrimSpace(phone)
}

func refRecipientKey(ref, name string) string {
	return strings.TrimSpace(ref) + "\x00" + strings.ToLower(strings.TrimSpace(name))
}

// parseTimeLoose parses common factory-return timestamps. Blank → (nil, nil).
func parseTimeLoose(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"2006/01/02 15:04:05",
		"2006/01/02",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			tt := t
			return &tt, nil
		}
	}
	return nil, fmt.Errorf("unrecognized time format")
}
