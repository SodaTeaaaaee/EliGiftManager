package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/alignment"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// carrierNameSuffixes are the generic carrier words dropped before matching so
// 申通快递 (as a factory file spells it) meets 申通 (as a source platform lists
// it).
var carrierNameSuffixes = []string{"快递", "速运", "物流", "速递"}

// normalizeCarrierName folds a carrier description for matching: trim, drop
// inner whitespace, case-fold, and strip the generic suffixes.
func normalizeCarrierName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.Join(strings.Fields(s), "")
	for changed := true; changed; {
		changed = false
		for _, suffix := range carrierNameSuffixes {
			if trimmed := strings.TrimSuffix(s, suffix); trimmed != s && trimmed != "" {
				s = trimmed
				changed = true
			}
		}
	}
	return s
}

// matchCarrierCode finds the source-platform carrier id for a carrier as a
// factory shipment file describes it. Exact normalized-name matches win; then
// a single containment match either way; several candidates are ambiguous and
// yield no code. The boolean reports that ambiguity so callers can log it.
func matchCarrierCode(mappings []domain.CarrierMapping, carrierName string) (string, bool) {
	want := normalizeCarrierName(carrierName)
	if want == "" {
		return "", false
	}
	var exact, partial []string
	for _, m := range mappings {
		if m.ExternalCode == "" {
			continue
		}
		have := normalizeCarrierName(m.InternalName)
		if have == "" {
			continue
		}
		switch {
		case have == want:
			exact = append(exact, m.ExternalCode)
		case strings.Contains(have, want) || strings.Contains(want, have):
			partial = append(partial, m.ExternalCode)
		}
	}
	pick := func(codes []string) (string, bool) {
		first := codes[0]
		for _, c := range codes[1:] {
			if c != first {
				return "", true
			}
		}
		return first, false
	}
	if len(exact) > 0 {
		return pick(exact)
	}
	if len(partial) > 0 {
		return pick(partial)
	}
	return "", false
}

func (ws *Workspace) CreateCarrierMapping(ctx context.Context, m *domain.CarrierMapping) error {
	m.InternalName = strings.TrimSpace(m.InternalName)
	m.ExternalCode = strings.TrimSpace(m.ExternalCode)
	if m.InternalName == "" || m.ExternalCode == "" {
		return fmt.Errorf("carrier mapping needs a carrier name and the platform's carrier code")
	}
	return ws.Store.CreateCarrierMapping(ctx, m)
}

func (ws *Workspace) ListCarrierMappings(ctx context.Context, platformID uint) ([]domain.CarrierMapping, error) {
	return ws.Store.ListCarrierMappings(ctx, platformID)
}

// UpdateCarrierMapping overwrites the name and codes of an existing mapping;
// the owning platform is fixed.
func (ws *Workspace) UpdateCarrierMapping(ctx context.Context, m *domain.CarrierMapping) (*domain.CarrierMapping, error) {
	existing, err := ws.Store.GetCarrierMapping(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	existing.InternalName = strings.TrimSpace(m.InternalName)
	existing.ExternalCode = strings.TrimSpace(m.ExternalCode)
	existing.InternalCode = strings.TrimSpace(m.InternalCode)
	if existing.InternalName == "" || existing.ExternalCode == "" {
		return nil, fmt.Errorf("carrier mapping needs a carrier name and the platform's carrier code")
	}
	if err := ws.Store.UpdateCarrierMapping(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (ws *Workspace) DeleteCarrierMapping(ctx context.Context, id uint) error {
	if _, err := ws.Store.GetCarrierMapping(ctx, id); err != nil {
		return err
	}
	return ws.Store.DeleteCarrierMapping(ctx, id)
}

// ImportCarrierMappingsResult summarizes a carrier table import.
type ImportCarrierMappingsResult struct {
	Created int
	Updated int
	Skipped int
	Issues  []alignment.ParseIssue
}

// ImportCarrierMappings loads a source platform's carrier table (for example
// bilibili's 快递编码映射关系 export) in header mode: nameHeader feeds
// InternalName, codeHeader feeds ExternalCode. Rows upsert by normalized name
// within the platform; rows missing either value are skipped with an issue.
// The whole import commits or rolls back together.
func (ws *Workspace) ImportCarrierMappings(ctx context.Context, platformID uint, filePath, nameHeader, codeHeader string) (*ImportCarrierMappingsResult, error) {
	if _, err := ws.Store.GetPlatform(ctx, platformID); err != nil {
		return nil, err
	}
	nameHeader, codeHeader = strings.TrimSpace(nameHeader), strings.TrimSpace(codeHeader)
	if nameHeader == "" || codeHeader == "" {
		return nil, fmt.Errorf("import carrier mappings: name and code header names are required")
	}
	data, format, err := readFileWithFormat(filePath)
	if err != nil {
		return nil, err
	}
	records, err := alignment.ReadRows(data, format, "")
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("import carrier mappings: file has no header row")
	}
	nameIdx, codeIdx := -1, -1
	for i, h := range records[0] {
		switch strings.TrimSpace(h) {
		case nameHeader:
			if nameIdx < 0 {
				nameIdx = i
			}
		case codeHeader:
			if codeIdx < 0 {
				codeIdx = i
			}
		}
	}
	if nameIdx < 0 {
		return nil, fmt.Errorf("import carrier mappings: header %q not found", nameHeader)
	}
	if codeIdx < 0 {
		return nil, fmt.Errorf("import carrier mappings: header %q not found", codeHeader)
	}

	result := &ImportCarrierMappingsResult{Issues: []alignment.ParseIssue{}}
	if err := ws.Store.WithTx(ctx, func(tx domain.Store) error {
		existing, err := tx.ListCarrierMappings(ctx, platformID)
		if err != nil {
			return err
		}
		byName := make(map[string]*domain.CarrierMapping, len(existing))
		for i := range existing {
			key := normalizeCarrierName(existing[i].InternalName)
			if _, dup := byName[key]; !dup {
				byName[key] = &existing[i]
			}
		}
		for r := 1; r < len(records); r++ {
			lineNo := r
			name := strings.TrimSpace(cellAt(records[r], nameIdx))
			code := strings.TrimSpace(cellAt(records[r], codeIdx))
			if name == "" && code == "" && rowIsBlank(records[r]) {
				continue
			}
			if name == "" || code == "" {
				result.Skipped++
				key, msg := "shipment.carrier_name", fmt.Sprintf("row %d has no carrier name", lineNo)
				if name != "" {
					key, msg = "shipment.carrier_code", fmt.Sprintf("carrier %q has no code on this platform", name)
				}
				result.Issues = append(result.Issues, alignment.ParseIssue{LineNo: lineNo, Key: key, Message: msg})
				continue
			}
			key := normalizeCarrierName(name)
			if current, ok := byName[key]; ok {
				current.InternalName = name
				current.ExternalCode = code
				if err := tx.UpdateCarrierMapping(ctx, current); err != nil {
					return err
				}
				result.Updated++
				continue
			}
			m := &domain.CarrierMapping{PlatformID: platformID, InternalName: name, ExternalCode: code}
			if err := tx.CreateCarrierMapping(ctx, m); err != nil {
				return err
			}
			byName[key] = m
			result.Created++
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func cellAt(record []string, i int) string {
	if i < 0 || i >= len(record) {
		return ""
	}
	return record[i]
}

func rowIsBlank(record []string) bool {
	for _, c := range record {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}
