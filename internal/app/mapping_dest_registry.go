package app

import (
	"fmt"
	"sort"
	"strings"
)

// Known MappingRules transforms (closed set). Keep in sync with applyTransforms.
var knownTransforms = map[string]struct{}{
	"trim":                {},
	"strip_quotes":        {},
	"strip_leading_quote": {},
}

// Demand-line dest keys accepted by setDemandLineField (unprefixed v1 form).
var lineDestBare = []string{
	"line_type",
	"obligation_trigger_kind",
	"entitlement_authority",
	"recipient_input_state",
	"routing_disposition",
	"routing_reason_code",
	"eligibility_context_ref",
	"entitlement_code",
	"gift_level_snapshot",
	"recipient_input_payload",
	"external_title",
	"requested_quantity",
}

// Document / recipient namespaces used by demand import (import_mapping.go).
var documentDests = []string{
	"document.source_customer_ref",
	"document.source_document_no",
	"document.display_name",
}

var recipientDests = []string{
	"recipient.name",
	"recipient.phone",
	"recipient.country",
	"recipient.province",
	"recipient.city",
	"recipient.district",
	"recipient.address_line1",
	"recipient.address_line2",
	"recipient.postal_code",
	"recipient.label",
	"recipient.is_default",
}

// Product catalog import dests (product_catalog_import.go + frontend destFields).
var productDests = []string{
	"product.supplier_platform",
	"product.factory_sku",
	"product.name",
	"product.supplier_product_ref",
	"product.product_kind",
	"product.extra_data",
}

// Shipment reconcile dests (shipment_reconcile.go + frontend destFields).
var shipmentDests = []string{
	"shipment.third_party_order_no",
	"shipment.external_key",
	"shipment.fulfillment_line_id",
	"shipment.factory_sku",
	"shipment.sku",
	"shipment.supplier_product_ref",
	"shipment.sku_quantity",
	"shipment.spec_quantity",
	"shipment.phone",
	"shipment.recipient_phone",
	"shipment.recipient_name",
	"shipment.name",
	"shipment.tracking_no",
	"shipment.external_shipment_no",
	"shipment.carrier_code",
	"shipment.carrier_name",
	"shipment.quantity",
	"shipment.shipped_at",
}

// Carrier mapping import dests.
var carrierDests = []string{
	"carrier.internal_carrier_code",
	"carrier.external_carrier_code",
	"carrier.external_carrier_name",
	"carrier.aliases",
	"carrier.is_default",
}

// Factory-order / tracking export dests (template_payload_renderer.go + destFields.ts).
var exportDests = []string{
	"export.third_party_order_no",
	"export.tracking_no",
	"export.carrier_code",
	"export.external_document_no",
	"export.shipment_id",
	"export.recipient",
	"export.recipient_name",
	"export.phone",
	"export.address",
	"export.factory_sku",
	"export.supplier_sku",
	"export.quantity",
	"export.supplier_line_no",
	"export.product_id",
	"export.fulfillment_line_id",
}

var trackingDests = []string{
	"tracking.tracking_no",
	"tracking.carrier_code",
	"tracking.carrier_name",
	"tracking.external_shipment_no",
	// Renderer-accepted tracking export keys (also used with export.* prefix).
	"tracking.third_party_order_no",
	"tracking.fulfillment_line_id",
	"tracking.shipment_id",
	"tracking.item_id",
	"tracking.external_document_no",
	"tracking.external_line_no",
}

// destCatalogByDocType is the authoritative per-document-type dest vocabulary.
// Built once at init from the slices above.
var destCatalogByDocType map[string]map[string]struct{}

// allKnownDests is the union of every per-docType catalog (plus line.* forms).
// Used by ApplyRow when no docType is in scope.
var allKnownDests map[string]struct{}

func init() {
	linePrefixed := make([]string, len(lineDestBare))
	for i, f := range lineDestBare {
		linePrefixed[i] = "line." + f
	}
	demandDests := concatDests(lineDestBare, linePrefixed, documentDests, recipientDests)

	destCatalogByDocType = map[string]map[string]struct{}{
		"import_entitlement":            toDestSet(demandDests),
		"import_sales_order":            toDestSet(demandDests),
		"import_product_catalog":        toDestSet(productDests),
		"import_supplier_shipment":      toDestSet(shipmentDests),
		"import_carrier_mapping":        toDestSet(carrierDests),
		"export_supplier_order":         toDestSet(exportDests),
		"export_source_tracking_update": toDestSet(concatDests(trackingDests, exportDests)),
	}

	allKnownDests = make(map[string]struct{})
	for _, set := range destCatalogByDocType {
		for k := range set {
			allKnownDests[k] = struct{}{}
		}
	}
}

func toDestSet(keys []string) map[string]struct{} {
	set := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		set[NormalizeDestKey(k)] = struct{}{}
	}
	return set
}

func concatDests(parts ...[]string) []string {
	n := 0
	for _, p := range parts {
		n += len(p)
	}
	out := make([]string, 0, n)
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// NormalizeDestKey trims and lowercases a mapping dest key for catalog lookups.
func NormalizeDestKey(dest string) string {
	return strings.ToLower(strings.TrimSpace(dest))
}

// DestCatalog returns the sorted legal dest keys for a document type.
// Unknown docType yields nil.
func DestCatalog(docType string) []string {
	set := destCatalogByDocType[strings.TrimSpace(docType)]
	if set == nil {
		return nil
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// IsLegalDest reports whether dest is in the legal vocabulary for docType.
func IsLegalDest(docType, dest string) bool {
	set := destCatalogByDocType[strings.TrimSpace(docType)]
	if set == nil {
		return false
	}
	_, ok := set[NormalizeDestKey(dest)]
	return ok
}

// IsKnownDest reports whether dest appears in any document-type catalog.
func IsKnownDest(dest string) bool {
	_, ok := allKnownDests[NormalizeDestKey(dest)]
	return ok
}

// IsKnownTransform reports whether name is in the closed transform set.
func IsKnownTransform(name string) bool {
	_, ok := knownTransforms[strings.TrimSpace(name)]
	return ok
}

// ValidateMappingRulesConfig fail-fast validates MappingRules against the
// dest catalog and transform closed set for a document type.
//
// Checks:
//   - unknown transform names
//   - required dests missing from columns|positions
//   - duplicate dest keys (required list, or same dest in both columns and positions)
//   - illegal dest keys for the document type
func ValidateMappingRulesConfig(docType string, rules *TemplateMappingRules) error {
	if rules == nil {
		return fmt.Errorf("mapping rules are nil")
	}
	docType = strings.TrimSpace(docType)
	if destCatalogByDocType[docType] == nil {
		return fmt.Errorf("unknown document type %q for mapping dest validation", docType)
	}

	// Collect dest keys that participate in source mapping.
	mapped := make(map[string]struct{})
	addMapped := func(dest string) error {
		key := NormalizeDestKey(dest)
		if key == "" {
			return fmt.Errorf("empty dest key in mapping rules")
		}
		if !IsLegalDest(docType, key) {
			return fmt.Errorf("illegal dest %q for document type %q", dest, docType)
		}
		if _, exists := mapped[key]; exists {
			return fmt.Errorf("duplicate dest %q in mapping rules", dest)
		}
		mapped[key] = struct{}{}
		return nil
	}

	for dest := range rules.Columns {
		if err := addMapped(dest); err != nil {
			return err
		}
	}
	for dest := range rules.Positions {
		key := NormalizeDestKey(dest)
		if key == "" {
			return fmt.Errorf("empty dest key in positions")
		}
		// Same dest in both columns and positions is a duplicate mapping.
		if _, exists := mapped[key]; exists {
			if _, inCols := rules.Columns[dest]; inCols || hasNormalizedKey(rules.Columns, key) {
				return fmt.Errorf("duplicate dest %q appears in both columns and positions", dest)
			}
		}
		if err := addMapped(dest); err != nil {
			return err
		}
	}

	// Defaults / transforms / columnOrder / required must also be legal dests.
	for dest := range rules.Defaults {
		if !IsLegalDest(docType, dest) {
			return fmt.Errorf("illegal dest %q in defaults for document type %q", dest, docType)
		}
	}
	for dest, transforms := range rules.Transforms {
		if !IsLegalDest(docType, dest) {
			return fmt.Errorf("illegal dest %q in transforms for document type %q", dest, docType)
		}
		for _, name := range transforms {
			if !IsKnownTransform(name) {
				return fmt.Errorf("unknown transform %q for dest %q", name, dest)
			}
		}
	}
	for _, dest := range rules.ColumnOrder {
		if !IsLegalDest(docType, dest) {
			return fmt.Errorf("illegal dest %q in columnOrder for document type %q", dest, docType)
		}
	}

	seenRequired := make(map[string]struct{}, len(rules.Required))
	for _, dest := range rules.Required {
		key := NormalizeDestKey(dest)
		if key == "" {
			return fmt.Errorf("empty dest in required list")
		}
		if !IsLegalDest(docType, dest) {
			return fmt.Errorf("illegal dest %q in required for document type %q", dest, docType)
		}
		if _, dup := seenRequired[key]; dup {
			return fmt.Errorf("duplicate dest %q in required list", dest)
		}
		seenRequired[key] = struct{}{}

		// Required must be present in columns or positions (not defaults-only).
		if !hasNormalizedKey(rules.Columns, key) && !hasNormalizedKey(rules.Positions, key) {
			return fmt.Errorf("required dest %q is not present in columns or positions", dest)
		}
	}

	// ImageLayout matchField, when set, must be a legal dest for catalog imports.
	if rules.ImageLayout != nil && strings.TrimSpace(rules.ImageLayout.MatchField) != "" {
		mf := rules.ImageLayout.MatchField
		if !IsLegalDest(docType, mf) {
			return fmt.Errorf("illegal imageLayout.matchField %q for document type %q", mf, docType)
		}
	}

	return nil
}

func hasNormalizedKey[V any](m map[string]V, want string) bool {
	if m == nil {
		return false
	}
	if _, ok := m[want]; ok {
		return true
	}
	for k := range m {
		if NormalizeDestKey(k) == want {
			return true
		}
	}
	return false
}

// warnUnknownDests returns row-level warnings for mapped dests outside the
// global vocabulary. Values are intentionally kept (not dropped).
func warnUnknownDests(out map[string]string) []string {
	if len(out) == 0 {
		return nil
	}
	var warnings []string
	// Stable order for tests / logs.
	keys := make([]string, 0, len(out))
	for dest := range out {
		keys = append(keys, dest)
	}
	sort.Strings(keys)
	for _, dest := range keys {
		if !IsKnownDest(dest) {
			warnings = append(warnings, fmt.Sprintf("unknown mapping dest %q (not in legal vocabulary)", dest))
		}
	}
	return warnings
}
