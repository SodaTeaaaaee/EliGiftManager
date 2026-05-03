package service

import (
	"fmt"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
)

// ParseRowDynamically extracts coreData and extraData from a single CSV row
// using the DynamicTemplateRules mapping schema.
//
// For header-based templates (rules.HasHeader == true), SourceColumn entries
// are matched against the provided headers via case/separator-insensitive
// normalization.  ColumnIndex entries are used directly when SourceColumn is
// empty even for header-based templates.
//
// For headless templates (rules.HasHeader == false), ColumnIndex entries
// are always used (headers may be nil).
//
// A ColumnIndex < 0 means "column does not exist in the source" — the
// corresponding coreData entry will be "" and the caller is expected to
// apply its own fallback (e.g. template.Platform).
func ParseRowDynamically(record []string, headers []string, rules model.DynamicTemplateRules) (coreData map[string]string, extraData map[string]interface{}, err error) {
	coreData = make(map[string]string, len(rules.Mapping))

	// Track which column indices are consumed by the core mapping so that
	// catch_all ExtraData can collect everything else.
	usedIndices := make(map[int]bool, len(rules.Mapping))

	// 1. Build a normalized header index when present.
	var headerIndex map[string]int
	if rules.HasHeader && len(headers) > 0 {
		headerIndex = make(map[string]int, len(headers))
		for i, h := range headers {
			headerIndex[normalizeDynamicKey(h)] = i
		}
	}

	// 2. Walk the core mapping in map iteration order.
	for key, entry := range rules.Mapping {
		var colIdx int
		var found bool

		if rules.HasHeader && entry.SourceColumn != "" {
			normalized := normalizeDynamicKey(entry.SourceColumn)
			idx, exists := headerIndex[normalized]
			if !exists {
				if entry.Required {
					return nil, nil, fmt.Errorf("required column %q not found in CSV headers", entry.SourceColumn)
				}
				coreData[key] = ""
				continue
			}
			colIdx = idx
			found = true
		} else {
			colIdx = entry.ColumnIndex
			found = true
		}

		// ColumnIndex < 0 means column absent by design — skip extraction, caller supplies fallback.
		if !found || colIdx < 0 {
			coreData[key] = ""
			continue
		}

		value := readCSVCell(record, colIdx)

		if value == "" && entry.Required {
			return nil, nil, fmt.Errorf("required field %q (column %d) is empty in CSV row", key, colIdx)
		}
		if value == "" && entry.DefaultValue != "" {
			value = entry.DefaultValue
		}

		coreData[key] = value
		usedIndices[colIdx] = true
	}

	// 3. Extract extra data according to the configured strategy.
	extraData = make(map[string]interface{})

	switch rules.ExtraData.Strategy {
	case model.ExtraDataStrategyExplicit:
		for key, entry := range rules.ExtraData.ExplicitMapping {
			var colIdx int
			if rules.HasHeader && entry.SourceColumn != "" {
				if headerIndex == nil {
					continue
				}
				idx, exists := headerIndex[normalizeDynamicKey(entry.SourceColumn)]
				if !exists {
					continue
				}
				colIdx = idx
			} else {
				colIdx = entry.ColumnIndex
			}

			if colIdx < 0 || colIdx >= len(record) {
				continue
			}

			value := strings.TrimSpace(record[colIdx])
			if value == "" && entry.DefaultValue != "" {
				value = entry.DefaultValue
			}
			extraData[key] = value
		}

	default:
		// Default behaviour (Strategy == "" or "catch_all"): collect every
		// column that was not consumed by the core mapping.
		for i, value := range record {
			if usedIndices[i] {
				continue
			}
			v := strings.TrimSpace(value)
			if v == "" {
				continue
			}
			var headerName string
			if i < len(headers) {
				headerName = strings.TrimSpace(headers[i])
			} else {
				headerName = fmt.Sprintf("column_%d", i)
			}
			extraData[headerName] = v
		}
	}

	return coreData, extraData, nil
}

// normalizeDynamicKey normalizes a field name for case/separator-insensitive
// matching against CSV headers.  It is self-contained so that csv_transformer.go
// helpers can be cleaned up independently.
func normalizeDynamicKey(key string) string {
	normalized := strings.TrimSpace(key)
	normalized = strings.TrimPrefix(normalized, "\ufeff")
	normalized = strings.ToLower(normalized)
	normalized = strings.NewReplacer("_", "", "-", "", " ", "").Replace(normalized)
	return normalized
}
