package alignment

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// Transformer rewrites one semantic value in the context of its semantic key.
// Key context lets mapEnum find its table inside the owning MappingConfig.
type Transformer func(key, value string) (string, error)

// joinAddressPart separates the source parts fed to joinAddress. The unit
// separator is effectively impossible in spreadsheet cell data.
const joinAddressPart = "\x1f"

// buildTransformers compiles the config's transform chains into callable
// transformers. Every name must come from domain.NamedTransformers, and
// splitSkuQuantity is rejected here as well because it is a row-expansion
// capability carried by MappingConfig.SplitSkuQuantity.
func buildTransformers(cfg MappingConfig) (map[string]Transformer, error) {
	for _, chains := range cfg.Transforms {
		for _, name := range chains {
			known := false
			for _, n := range domain.NamedTransformers {
				if n == name {
					known = true
					break
				}
			}
			if !known {
				return nil, fmt.Errorf("alignment: unknown transformer %q", name)
			}
			if name == "splitSkuQuantity" {
				return nil, fmt.Errorf("alignment: transformer splitSkuQuantity is a row-expansion capability; set MappingConfig.SplitSkuQuantity instead")
			}
		}
	}
	return map[string]Transformer{
		"trim":           transformTrim,
		"strip_quotes":   transformStripQuotes,
		"parseDate":      transformParseDate,
		"mapEnum":        mapEnumTransformer(cfg),
		"normalizePhone": transformNormalizePhone,
		"joinAddress":    transformJoinAddress,
	}, nil
}

func transformTrim(_, value string) (string, error) {
	return strings.TrimSpace(value), nil
}

// transformStripQuotes removes one layer of paired surrounding quotes, a
// leading single apostrophe used by spreadsheets to force text cells (for
// example '435167587794147), and stray CR/LF noise that Excel-style exports
// leave inside identifier cells. Identifiers never carry meaningful
// newlines, so dropping them everywhere is safe here.
func transformStripQuotes(_, value string) (string, error) {
	v := value
	if len(v) >= 2 {
		first, last := v[0], v[len(v)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			v = v[1 : len(v)-1]
		}
	}
	if strings.HasPrefix(v, "'") {
		v = v[1:]
	}
	v = strings.ReplaceAll(v, "\r", "")
	v = strings.ReplaceAll(v, "\n", "")
	return v, nil
}

var dateLayouts = []string{
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04",
	"2006-01-02",
	"2006/01/02 15:04:05",
	"2006/01/02 15:04",
	"2006/01/02",
	"2006/1/2 15:04:05",
	"2006/1/2 15:04",
	"2006/1/2",
	"2006-1-2 15:04:05",
	"2006-1-2 15:04",
	"2006-1-2",
	time.RFC3339,
	"20060102150405",
	"20060102",
}

// excelSerialEpoch anchors Excel's 1900 date system. Serials at or above 61
// count days from 1899-12-30; smaller serials count from 1899-12-31 to skip
// Excel's fictitious 1900-02-29.
var excelSerialEpochLate = time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
var excelSerialEpochEarly = time.Date(1899, 12, 31, 0, 0, 0, 0, time.UTC)

// transformParseDate normalizes common textual forms and Excel serial numbers
// to RFC3339, or to a bare date when the input carries no time of day.
// Textual layouts win over serials so compact forms like 20260501 parse as
// dates, not as absurd day counts.
func transformParseDate(_, value string) (string, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return "", nil
	}
	for _, layout := range dateLayouts {
		if t, err := time.ParseInLocation(layout, raw, time.UTC); err == nil {
			return formatDateLike(t), nil
		}
	}
	if serial, err := strconv.ParseFloat(raw, 64); err == nil {
		return formatExcelSerial(serial)
	}
	return "", fmt.Errorf("parseDate: cannot parse %q", value)
}

func formatExcelSerial(serial float64) (string, error) {
	if serial < 0 {
		return "", fmt.Errorf("parseDate: negative Excel serial %v", serial)
	}
	days := int(serial)
	frac := serial - float64(days)
	epoch := excelSerialEpochLate
	if days < 61 {
		epoch = excelSerialEpochEarly
	}
	t := epoch.AddDate(0, 0, days).Add(time.Duration(frac * 24 * float64(time.Hour)))
	return formatDateLike(t), nil
}

func formatDateLike(t time.Time) string {
	if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0 {
		return t.Format("2006-01-02")
	}
	return t.Format(time.RFC3339)
}

// mapEnumTransformer builds a transformer that resolves values through the
// config's EnumMaps table for the semantic key. Unknown values are errors so
// silent passthrough cannot smuggle external vocabulary into internal enums.
func mapEnumTransformer(cfg MappingConfig) Transformer {
	return func(key, value string) (string, error) {
		table, ok := cfg.EnumMaps[key]
		if !ok {
			return "", fmt.Errorf("mapEnum: no enumMaps entry for key %q", key)
		}
		mapped, ok := table[value]
		if !ok {
			return "", fmt.Errorf("mapEnum: value %q is not mapped for key %q", value, key)
		}
		return mapped, nil
	}
}

// transformNormalizePhone strips spacing and punctuation (ASCII and CJK
// fullwidth variants) and folds +86 / 86 country-code prefixes back to a bare
// 11-digit national number (keeping any other leading + intact).
//
// Known limitation: fullwidth digits（１３８…）and other non-ASCII digit
// forms are NOT folded back to ASCII digits; they pass through untouched so
// downstream format checks, not this transformer, decide whether they are
// usable phone data.
func transformNormalizePhone(_, value string) (string, error) {
	var b strings.Builder
	for _, r := range value {
		switch r {
		case ' ', '\t', '-', '(', ')':
			continue
		// CJK fullwidth/typographic separators seen in Chinese exports:
		// ideographic space, fullwidth parentheses, vertical/small parenthesis
		// presentation forms, fullwidth hyphen-minus, en/em dash.
		case '\u3000', '\uff08', '\uff09', '\ufe35', '\ufe36', '\ufe59', '\ufe5a',
			'\uff0d', '\u2013', '\u2014':
			continue
		default:
			b.WriteRune(r)
		}
	}
	out := b.String()
	for _, prefix := range []string{"+86", "86"} {
		if strings.HasPrefix(out, prefix) {
			rest := strings.TrimPrefix(out, prefix)
			if len(rest) == 11 && strings.Trim(rest, "0123456789") == "" {
				out = rest
			}
			break
		}
	}
	return out, nil
}

// transformJoinAddress merges \x1f-separated source parts into one address
// string: each part is trimmed, empty parts are dropped, and the rest is
// concatenated with no separator (Chinese address fragments concatenate
// directly). A value without the separator is returned trimmed.
func transformJoinAddress(_, value string) (string, error) {
	parts := strings.Split(value, joinAddressPart)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, ""), nil
}
