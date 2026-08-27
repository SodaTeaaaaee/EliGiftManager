package alignment

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ParsedRow is one semantic row after mapping, defaults, transforms, and
// (when enabled) multi-SKU expansion. LineNo is unique across expanded rows
// (1-based over produced rows); SourceRow points back at the source data row
// so expansions of one blob stay groupable.
type ParsedRow struct {
	LineNo      int
	SourceRow   int
	Values      map[string]string
	Fingerprint string
}

// ParseIssue describes one non-fatal problem found while parsing. Rows that
// miss a Required key are dropped; the issue records why.
type ParseIssue struct {
	LineNo  int
	Key     string
	Message string
}

// TemplateSpec aggregates what Parse needs beyond the mapping: the fact kind
// the template is expected to produce (used by the app layer when converting
// rows into ingest facts; alignment itself only carries it through).
type TemplateSpec struct {
	Mapping MappingConfig
	Kind    string
}

// Semantic keys filled by splitSkuQuantity expansion.
const (
	SplitTargetSku     = "product.alias_id"
	SplitTargetTitle   = "product.alias_title"
	SplitTargetQty     = "quantity"
	defaultFingerprint = "\x00default"
)

// Parse turns raw file bytes into semantic rows. Fatal problems (unreadable
// file, unusable mapping) come back as errors; per-row problems come back as
// issues. The fingerprint is the SHA-256 of the Fingerprint keys' values
// joined with \x1f; when Fingerprint is empty every non-empty value keyed by
// semantic key participates (sorted by key) so duplicate detection still has
// something stable to chew on.
func Parse(data []byte, format string, spec TemplateSpec) ([]ParsedRow, []ParseIssue, error) {
	cfg := spec.Mapping
	if err := cfg.normalize(); err != nil {
		return nil, nil, err
	}
	records, err := ReadRows(data, format, cfg.SheetName)
	if err != nil {
		return nil, nil, err
	}

	// Split header from data records.
	headers := map[string]int{}
	dataStart := 0
	if cfg.HasHeader {
		if len(records) == 0 {
			return nil, nil, fmt.Errorf("alignment: parse: file has no header row")
		}
		for i, h := range records[0] {
			name := strings.TrimSpace(h)
			if _, dup := headers[name]; !dup {
				headers[name] = i
			}
		}
		dataStart = 1
	}

	transformers, err := buildTransformers(cfg)
	if err != nil {
		return nil, nil, err
	}

	// Resolve every referenced source column up front so missing headers are
	// reported once, not per row.
	colIndex := func(key string) (int, bool, error) {
		if cfg.Mode == ModeHeader {
			name, ok := cfg.Columns[key]
			if !ok {
				return 0, false, nil
			}
			idx, ok := headers[strings.TrimSpace(name)]
			if !ok {
				return 0, true, fmt.Errorf("alignment: parse: source column %q for %q not found in header", name, key)
			}
			return idx, true, nil
		}
		idx, ok := cfg.Positions[key]
		if !ok {
			return 0, false, nil
		}
		return idx, true, nil
	}

	mapped := cfg.mappingKeys()
	colOf := map[string]int{}
	var issues []ParseIssue
	for _, key := range mapped {
		idx, isMapped, err := colIndex(key)
		if err != nil {
			issues = append(issues, ParseIssue{LineNo: 0, Key: key, Message: err.Error()})
			continue
		}
		if isMapped {
			colOf[key] = idx
		}
	}

	var rows []ParsedRow
	lineNo := 0
	for r := dataStart; r < len(records); r++ {
		record := records[r]
		sourceRow := r - dataStart + 1

		values := map[string]string{}
		for _, key := range mapped {
			if idx, ok := colOf[key]; ok {
				values[key] = cell(record, idx)
			}
		}

		// Multi-column joins feed the joinAddress transformer with \x1f parts.
		joinKeys := make([]string, 0, len(cfg.JoinSources))
		for key := range cfg.JoinSources {
			joinKeys = append(joinKeys, key)
		}
		sort.Strings(joinKeys)
		for _, key := range joinKeys {
			parts := make([]string, 0, len(cfg.JoinSources[key]))
			for _, ref := range cfg.JoinSources[key] {
				parts = append(parts, resolveRef(record, headers, cfg.Mode, ref))
			}
			values[key] = strings.Join(parts, joinAddressPart)
		}

		// Fixed defaults override whatever the source produced.
		for key, def := range cfg.Defaults {
			values[key] = def
		}

		candidates := []map[string]string{values}
		if cfg.SplitSkuQuantity != "" {
			expanded, err := expandSplitSku(values, cfg.SplitSkuQuantity)
			if err != nil {
				issues = append(issues, ParseIssue{LineNo: sourceRow, Key: cfg.SplitSkuQuantity, Message: err.Error()})
				continue
			}
			candidates = expanded
		}

		for _, candidate := range candidates {
			row, rowIssues := finishRow(candidate, cfg, transformers, sourceRow)
			issues = append(issues, rowIssues...)
			if row != nil {
				lineNo++
				row.LineNo = lineNo
				rows = append(rows, *row)
			}
		}
	}
	return rows, issues, nil
}

// finishRow applies transform chains, required checks, and fingerprinting for
// one candidate value set. It returns a nil row when the row must be dropped.
func finishRow(values map[string]string, cfg MappingConfig, transformers map[string]Transformer, sourceRow int) (*ParsedRow, []ParseIssue) {
	var issues []ParseIssue
	// JoinSources implies joinAddress (the parts carry the \x1f separator).
	transforms := map[string][]string{}
	for key, chain := range cfg.Transforms {
		transforms[key] = append([]string(nil), chain...)
	}
	for key := range cfg.JoinSources {
		hasJoin := false
		for _, name := range transforms[key] {
			if name == "joinAddress" {
				hasJoin = true
				break
			}
		}
		if !hasJoin {
			transforms[key] = append(transforms[key], "joinAddress")
		}
	}
	for key, chain := range transforms {
		v := values[key]
		for _, name := range chain {
			tf, ok := transformers[name]
			if !ok {
				continue
			}
			out, err := tf(key, v)
			if err != nil {
				issues = append(issues, ParseIssue{LineNo: sourceRow, Key: key, Message: err.Error()})
				continue
			}
			v = out
		}
		values[key] = v
	}

	dropped := false
	for _, key := range cfg.Required {
		if strings.TrimSpace(values[key]) == "" {
			issues = append(issues, ParseIssue{LineNo: sourceRow, Key: key, Message: fmt.Sprintf("required key %q is empty", key)})
			dropped = true
		}
	}
	if dropped {
		return nil, issues
	}
	return &ParsedRow{SourceRow: sourceRow, Values: values, Fingerprint: FingerprintValues(values, cfg.Fingerprint)}, issues
}

// FingerprintValues hashes the given keys' values (or, when keys is empty,
// every non-empty value sorted by key) with SHA-256.
func FingerprintValues(values map[string]string, keys []string) string {
	parts := make([]string, 0, len(keys))
	if len(keys) == 0 {
		for _, k := range sortedKeys(values) {
			if values[k] != "" {
				parts = append(parts, k+"\x1e"+values[k])
			}
		}
		if len(parts) == 0 {
			parts = append(parts, defaultFingerprint)
		}
	} else {
		for _, k := range keys {
			parts = append(parts, values[k])
		}
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return hex.EncodeToString(sum[:])
}

// expandSplitSku turns one row carrying a pipe-concatenated multi-product
// blob ("sku_title * qty|...") into one value set per segment. The blob key
// keeps the raw segment; the conventional targets product.alias_id,
// product.alias_title, and quantity receive the parsed parts.
func expandSplitSku(values map[string]string, blobKey string) ([]map[string]string, error) {
	blob := strings.TrimSpace(values[blobKey])
	if blob == "" {
		return []map[string]string{values}, nil
	}
	var out []map[string]string
	for _, seg := range strings.Split(blob, "|") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		sku, title, qty, err := parseSkuSegment(seg)
		if err != nil {
			return nil, fmt.Errorf("splitSkuQuantity: %w", err)
		}
		clone := cloneValues(values)
		clone[blobKey] = seg
		clone[SplitTargetSku] = sku
		clone[SplitTargetTitle] = title
		clone[SplitTargetQty] = strconv.Itoa(qty)
		out = append(out, clone)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("splitSkuQuantity: blob %q has no segments", blob)
	}
	return out, nil
}

// parseSkuSegment parses "sku_title * qty". The SKU runs to the first
// underscore (matching rouzao's "<code>_<title>" convention); the quantity
// follows the last "*".
func parseSkuSegment(seg string) (sku, title string, qty int, err error) {
	rest := seg
	qty = 1
	if i := strings.LastIndex(rest, "*"); i >= 0 {
		qtyText := strings.TrimSpace(rest[i+1:])
		n, perr := strconv.Atoi(qtyText)
		if perr != nil {
			return "", "", 0, fmt.Errorf("segment %q has non-numeric quantity %q", seg, qtyText)
		}
		if n <= 0 {
			return "", "", 0, fmt.Errorf("segment %q has non-positive quantity %d", seg, n)
		}
		qty = n
		rest = strings.TrimSpace(rest[:i])
	}
	if i := strings.Index(rest, "_"); i >= 0 {
		return rest[:i], rest[i+1:], qty, nil
	}
	return rest, "", qty, nil
}

// resolveRef resolves one JoinSources reference against the current record:
// header names in header mode, decimal column indexes in positional mode.
func resolveRef(record []string, headers map[string]int, mode, ref string) string {
	ref = strings.TrimSpace(ref)
	if mode == ModeHeader {
		if idx, ok := headers[ref]; ok {
			return cell(record, idx)
		}
		return ""
	}
	idx, err := strconv.Atoi(ref)
	if err != nil {
		return ""
	}
	return cell(record, idx)
}

func cloneValues(values map[string]string) map[string]string {
	clone := make(map[string]string, len(values)+3)
	for k, v := range values {
		clone[k] = v
	}
	return clone
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
