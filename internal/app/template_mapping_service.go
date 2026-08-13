package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// TemplateMappingService applies a template's MappingRules to convert raw rows
// into structured DemandLines.
type TemplateMappingService struct {
	templateRepo domain.DocumentTemplateRepository
	bindingRepo  domain.ProfileTemplateBindingRepository
	profileRepo  domain.IntegrationProfileRepository
}

func NewTemplateMappingService(
	templateRepo domain.DocumentTemplateRepository,
	bindingRepo domain.ProfileTemplateBindingRepository,
	profileRepo domain.IntegrationProfileRepository,
) *TemplateMappingService {
	return &TemplateMappingService{
		templateRepo: templateRepo,
		bindingRepo:  bindingRepo,
		profileRepo:  profileRepo,
	}
}

// TemplateMappingRules defines the column mapping contract stored in DocumentTemplate.MappingRules.
//
// v1 (legacy): {"columns":{...},"defaults":{...}} — treated as version=1, mode=header, hasHeader=true.
// Unprefixed dest keys are demand-line fields (semantic default namespace "line.").
//
// v2:
//
//	{
//	  "version": 2,
//	  "mode": "header"|"positional",
//	  "hasHeader": true,
//	  "columns": {"line.gift_level_snapshot": "大航海等级"},
//	  "positions": {"line.gift_level_snapshot": 0},
//	  "defaults": {},
//	  "transforms": {"shipment.tracking_no": ["trim","strip_leading_quote"]},
//	  "columnOrder": ["export.third_party_order_no"],
//	  "required": []
//	}
type TemplateMappingRules struct {
	Version   int    `json:"version"`
	Mode      string `json:"mode"`
	HasHeader bool   `json:"hasHeader"`
	// SheetName is the exact worksheet name used by xlsx renderers/readers when
	// the external document contract names a sheet. It has no effect on CSV.
	SheetName   string              `json:"sheetName,omitempty"`
	Columns     map[string]string   `json:"columns"`
	Positions   map[string]int      `json:"positions"`
	Defaults    map[string]string   `json:"defaults"`
	Transforms  map[string][]string `json:"transforms"`
	ColumnOrder []string            `json:"columnOrder"`
	Required    []string            `json:"required"`
	// ImageLayout is optional catalog-zip image association config.
	// Omitted or null is fine; ParseMappingRules fills defaults when present.
	ImageLayout *CatalogImageLayout `json:"imageLayout,omitempty"`
}

// CatalogImageLayout describes how ImportProductCatalog associates images from a
// zip (or directory root) with ProductMaster rows after tabular upsert.
//
// Paths (CoverDir/DetailDir) come from template JSON — never hard-coded per platform.
type CatalogImageLayout struct {
	Enabled     bool     `json:"enabled"`
	MatchField  string   `json:"matchField"`  // default "product.name"
	CoverDir    string   `json:"coverDir"`    // relative to extract/root
	DetailDir   string   `json:"detailDir"`   // relative to extract/root
	NamePattern string   `json:"namePattern"` // default "{match}#{nn}"
	CoverPick   string   `json:"coverPick"`   // currently only "lowest_nn"
	TabularGlob string   `json:"tabularGlob"` // default "*.csv"
	ImageExts   []string `json:"imageExts"`   // default common image extensions
}

// ParseMappingRules unmarshals a template's MappingRules JSON and normalises v1 → v2 defaults.
func ParseMappingRules(raw string) (*TemplateMappingRules, error) {
	if raw == "" {
		return nil, fmt.Errorf("template has no mapping rules")
	}
	var rules TemplateMappingRules
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return nil, fmt.Errorf("invalid mapping rules: %w", err)
	}

	switch rules.Version {
	case 0:
		// v1 compat: no version field → version=1, mode=header, hasHeader=true.
		rules.Version = 1
		if rules.Mode == "" {
			rules.Mode = "header"
		}
		rules.HasHeader = true
	case 1:
		if rules.Mode == "" {
			rules.Mode = "header"
		}
		if !rules.HasHeader {
			// v1 always assumes a header row.
			rules.HasHeader = true
		}
	case 2:
		if rules.Mode == "" {
			rules.Mode = "header"
		}
	default:
		return nil, fmt.Errorf("unsupported mapping rules version %d", rules.Version)
	}

	rules.Mode = strings.ToLower(strings.TrimSpace(rules.Mode))
	switch rules.Mode {
	case "header":
		if len(rules.Columns) == 0 {
			return nil, fmt.Errorf("template mapping rules must define at least one column")
		}
	case "positional":
		if len(rules.Positions) == 0 {
			return nil, fmt.Errorf("positional mapping rules must define at least one position")
		}
	default:
		return nil, fmt.Errorf("unsupported mapping mode %q", rules.Mode)
	}

	normalizeCatalogImageLayout(rules.ImageLayout)

	return &rules, nil
}

// normalizeCatalogImageLayout fills defaults for optional ImageLayout fields.
// No-op when layout is nil (field omitted from JSON).
func normalizeCatalogImageLayout(layout *CatalogImageLayout) {
	if layout == nil {
		return
	}
	if strings.TrimSpace(layout.MatchField) == "" {
		layout.MatchField = "product.name"
	}
	if strings.TrimSpace(layout.NamePattern) == "" {
		layout.NamePattern = "{match}#{nn}"
	}
	if strings.TrimSpace(layout.CoverPick) == "" {
		layout.CoverPick = "lowest_nn"
	}
	if strings.TrimSpace(layout.TabularGlob) == "" {
		layout.TabularGlob = "*.csv"
	}
	if len(layout.ImageExts) == 0 {
		layout.ImageExts = []string{".jpg", ".jpeg", ".png", ".webp", ".gif"}
	} else {
		normalized := make([]string, 0, len(layout.ImageExts))
		for _, ext := range layout.ImageExts {
			ext = strings.ToLower(strings.TrimSpace(ext))
			if ext == "" {
				continue
			}
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			normalized = append(normalized, ext)
		}
		layout.ImageExts = normalized
	}
}

// ApplyRow maps a single ordered data row to dest→value using rules.
//
//   - mode=header: columns[dest] is a header name looked up in headers
//   - mode=positional: positions[dest] is a zero-based cell index into row
//
// Transforms run on source values (and on defaults when a transform list is declared
// for that dest). Defaults apply after source mapping and only fill empty or missing
// dests; a non-blank mapped value is never overwritten. Required dests that are
// missing or blank after that step yield an error.
//
// Dest keys outside the global legal vocabulary produce row-level warnings but are
// still kept in values (never silently dropped). Config-time rejection of illegal
// dests for a document type is ValidateMappingRulesConfig.
func ApplyRow(row []string, headers []string, rules *TemplateMappingRules) (map[string]string, []string, error) {
	if rules == nil {
		return nil, nil, fmt.Errorf("mapping rules are nil")
	}
	out := make(map[string]string)

	switch rules.Mode {
	case "positional":
		for dest, idx := range rules.Positions {
			if idx < 0 || idx >= len(row) {
				continue
			}
			val, err := applyTransforms(row[idx], rules.Transforms[dest])
			if err != nil {
				return nil, nil, fmt.Errorf("field %q: %w", dest, err)
			}
			out[dest] = val
		}
	default: // header
		indexByHeader := make(map[string]int, len(headers))
		for i, h := range headers {
			h = strings.TrimSpace(h)
			// First occurrence wins — matches typical CSV "duplicate header" behaviour.
			if _, exists := indexByHeader[h]; !exists {
				indexByHeader[h] = i
			}
		}
		for dest, srcCol := range rules.Columns {
			idx, ok := indexByHeader[strings.TrimSpace(srcCol)]
			if !ok || idx < 0 || idx >= len(row) {
				continue
			}
			val, err := applyTransforms(row[idx], rules.Transforms[dest])
			if err != nil {
				return nil, nil, fmt.Errorf("field %q (column %q): %w", dest, srcCol, err)
			}
			out[dest] = val
		}
	}

	// Defaults only fill empty dests; a non-blank mapped value wins.
	for dest, val := range rules.Defaults {
		if existing, ok := out[dest]; ok && strings.TrimSpace(existing) != "" {
			continue
		}
		if ts := rules.Transforms[dest]; len(ts) > 0 {
			transformed, err := applyTransforms(val, ts)
			if err != nil {
				return nil, nil, fmt.Errorf("default %q: %w", dest, err)
			}
			out[dest] = transformed
			continue
		}
		out[dest] = val
	}

	for _, dest := range rules.Required {
		v, ok := out[dest]
		if !ok || strings.TrimSpace(v) == "" {
			return nil, nil, fmt.Errorf("required field %q is missing or empty", dest)
		}
	}
	return out, warnUnknownDests(out), nil
}

func applyTransforms(val string, names []string) (string, error) {
	for _, name := range names {
		switch name {
		case "trim":
			val = strings.TrimSpace(val)
		case "strip_quotes":
			val = stripSurroundingQuotes(val)
		case "strip_leading_quote":
			val = strings.TrimPrefix(val, "'")
		default:
			return "", fmt.Errorf("unknown transform %q", name)
		}
	}
	return val, nil
}

func stripSurroundingQuotes(val string) string {
	if len(val) < 2 {
		return val
	}
	if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
		return val[1 : len(val)-1]
	}
	return val
}

// demandLineFieldName resolves a mapping dest key to a DemandLine field name.
// Accepts unprefixed names (v1 semantic default namespace "line.") and "line.*" keys.
// Other namespaces (shipment.*, export.*, …) are not demand-line fields.
func demandLineFieldName(dest string) (string, bool) {
	if dest == "" {
		return "", false
	}
	if strings.HasPrefix(dest, "line.") {
		field := strings.TrimPrefix(dest, "line.")
		if field == "" || strings.Contains(field, ".") {
			return "", false
		}
		return field, true
	}
	if strings.Contains(dest, ".") {
		return "", false
	}
	return dest, true
}

// ResolveImportTemplate finds the default template binding for a profile and document type.
func (s *TemplateMappingService) ResolveImportTemplate(ctx context.Context, profileID uint, documentType string) (*domain.DocumentTemplate, error) {
	binding, err := s.bindingRepo.FindDefaultByProfileAndType(ctx, profileID, documentType)
	if err != nil {
		return nil, fmt.Errorf("resolve template binding for profile %d / type %s: %w", profileID, documentType, err)
	}
	if binding == nil {
		return nil, fmt.Errorf("no default template binding for profile %d / type %s", profileID, documentType)
	}
	t, err := s.templateRepo.FindByID(ctx, binding.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("template %d not found: %w", binding.TemplateID, err)
	}
	return t, nil
}

// MapCSVRowToDemandLine converts a single CSV row (column → value map) into a
// DemandLine using the template's MappingRules. Header mode only — positional
// mode requires ordered cells via ApplyRow.
func MapCSVRowToDemandLine(row map[string]string, rules *TemplateMappingRules) (*domain.DemandLine, error) {
	if rules == nil {
		return nil, fmt.Errorf("mapping rules are nil")
	}
	if rules.Mode == "positional" {
		return nil, fmt.Errorf("positional mapping requires ordered row cells; use ApplyRow")
	}

	// Rebuild ordered headers/cells from the header-keyed map. Order is irrelevant
	// for header-mode lookup inside ApplyRow.
	headers := make([]string, 0, len(row))
	cells := make([]string, 0, len(row))
	for k, v := range row {
		headers = append(headers, k)
		cells = append(cells, v)
	}

	// Row-level mapping warnings are intentionally not surfaced here: this function
	// backs the legacy, all-or-nothing BuildImportPipeline/ImportDemandFromCSV path,
	// which returns a bare *domain.DemandLine / dto.DemandDocumentDTO with no
	// warnings-carrying result shape, and which no frontend call site invokes
	// (frontend exclusively uses the dual-mode ImportDemandCSV pipeline, which
	// surfaces warnings via MapDemandImportRow / BuildDemandImportPipelineWithMode).
	applied, _, err := ApplyRow(cells, headers, rules)
	if err != nil {
		return nil, err
	}

	line := &domain.DemandLine{}
	for dest, val := range applied {
		field, ok := demandLineFieldName(dest)
		if !ok {
			// Non-line namespaces are ignored on the demand-line path.
			continue
		}
		if err := setDemandLineField(line, field, val); err != nil {
			return nil, fmt.Errorf("field %q: %w", dest, err)
		}
	}
	return line, nil
}

// setDemandLineField sets a single field on a DemandLine by field name.
func setDemandLineField(line *domain.DemandLine, field, value string) error {
	switch field {
	case "line_type":
		line.LineType = value
	case "obligation_trigger_kind":
		line.ObligationTriggerKind = value
	case "entitlement_authority":
		line.EntitlementAuthority = value
	case "recipient_input_state":
		line.RecipientInputState = value
	case "routing_disposition":
		line.RoutingDisposition = value
	case "routing_reason_code":
		line.RoutingReasonCode = value
	case "eligibility_context_ref":
		line.EligibilityContextRef = value
	case "entitlement_code":
		line.EntitlementCode = value
	case "gift_level_snapshot":
		line.GiftLevelSnapshot = value
	case "recipient_input_payload":
		line.RecipientInputPayload = value
	case "external_title":
		line.ExternalTitle = value
	case "requested_quantity":
		value = strings.TrimSpace(value)
		qty, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid quantity %q", value)
		}
		if qty <= 0 {
			return fmt.Errorf("invalid quantity %q", value)
		}
		line.RequestedQuantity = qty
	default:
		return fmt.Errorf("unknown field %q", field)
	}
	return nil
}

// BuildImportPipeline resolves the template for a profile + document type,
// then maps all CSV rows to DemandLines. Returns parsed lines and the resolved template.
func (s *TemplateMappingService) BuildImportPipeline(ctx context.Context, profileID uint, documentType string, rows []map[string]string) (*domain.DocumentTemplate, []*domain.DemandLine, error) {
	t, err := s.ResolveImportTemplate(ctx, profileID, documentType)
	if err != nil {
		return nil, nil, err
	}

	rules, err := ParseMappingRules(t.MappingRules)
	if err != nil {
		return nil, nil, fmt.Errorf("template %s: %w", t.TemplateKey, err)
	}

	now := time.Now()
	lines := make([]*domain.DemandLine, 0, len(rows))
	for i, row := range rows {
		line, err := MapCSVRowToDemandLine(row, rules)
		if err != nil {
			return nil, nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		if line.CreatedAt.IsZero() {
			line.CreatedAt = now
		}
		if line.UpdatedAt.IsZero() {
			line.UpdatedAt = now
		}
		if line.SourceLineNo == 0 {
			line.SourceLineNo = i + 1
		}
		lines = append(lines, line)
	}

	return t, lines, nil
}
