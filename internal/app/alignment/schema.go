package alignment

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Schema version of MappingConfig. Version 3 is the first released shape of
// the wave-model alignment engine (older demand/fulfillment concepts were
// never persisted as mapping JSON, so no migration is needed).
const MappingSchemaVersion = 3

// Schema version of LayoutConfig.
const LayoutSchemaVersion = 1

const (
	ModeHeader     = "header"
	ModePositional = "positional"
)

// MappingConfig is the parse-side half of a template: how an external file
// becomes semantic key/value rows. It serializes into
// TemplateConfig.MappingJSON.
//
// Column addressing differs by mode. In header mode Columns maps a semantic
// key to a source header name; in positional mode Positions maps a semantic
// key to a 0-based column number. JoinSources entries follow the same
// convention: header names in header mode, decimal column indexes as strings
// in positional mode.
//
// Transformer carriers:
//   - trim, strip_quotes, parseDate, mapEnum, normalizePhone are plain value
//     transformers listed in Transforms chains.
//   - mapEnum reads its mapping table from EnumMaps[semantic key]; a value
//     missing from the table is an error, not a passthrough.
//   - joinAddress merges several source columns: JoinSources feeds the
//     raw parts joined with the \x1f unit separator and joinAddress splits,
//     trims, drops empties, and concatenates them. Defining JoinSources for a
//     key implies joinAddress on that key.
//   - splitSkuQuantity is a row-expansion capability, not a value rewrite, so
//     it has its own field (SplitSkuQuantity) naming the semantic key of the
//     pipe-concatenated multi-product blob. Listing it in a Transforms chain
//     is a configuration error.
type MappingConfig struct {
	Version          int                          `json:"version"`
	Mode             string                       `json:"mode,omitempty"` // "header" | "positional"
	HasHeader        bool                         `json:"hasHeader"`
	SheetName        string                       `json:"sheetName,omitempty"`
	Columns          map[string]string            `json:"columns,omitempty"`
	Positions        map[string]int               `json:"positions,omitempty"`
	Defaults         map[string]string            `json:"defaults,omitempty"`
	Transforms       map[string][]string          `json:"transforms,omitempty"`
	EnumMaps         map[string]map[string]string `json:"enumMaps,omitempty"`
	JoinSources      map[string][]string          `json:"joinSources,omitempty"`
	SplitSkuQuantity string                       `json:"splitSkuQuantity,omitempty"`
	Required         []string                     `json:"required,omitempty"`
	Fingerprint      []string                     `json:"fingerprint,omitempty"`
}

// LayoutConfig is the render-side half of a template: how semantic rows go
// back out to a platform file. It serializes into TemplateConfig.LayoutJSON.
type LayoutConfig struct {
	Version     int               `json:"version"`
	Format      string            `json:"format"` // csv | xlsx
	ColumnOrder []string          `json:"columnOrder"`
	HeaderNames map[string]string `json:"headerNames,omitempty"`
}

// ParseMappingConfig decodes TemplateConfig.MappingJSON. Empty input, invalid
// JSON, or a foreign schema version are hard errors.
func ParseMappingConfig(raw string) (MappingConfig, error) {
	if strings.TrimSpace(raw) == "" {
		return MappingConfig{}, fmt.Errorf("alignment: mapping config is empty")
	}
	var cfg MappingConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return MappingConfig{}, fmt.Errorf("alignment: mapping config: %w", err)
	}
	if cfg.Version != MappingSchemaVersion {
		return MappingConfig{}, fmt.Errorf("alignment: mapping config version %d is not supported (want %d)", cfg.Version, MappingSchemaVersion)
	}
	if err := cfg.normalize(); err != nil {
		return MappingConfig{}, err
	}
	return cfg, nil
}

// ParseLayoutConfig decodes TemplateConfig.LayoutJSON. An empty string yields
// a zero config with no error: output templates without a layout fall back to
// built-in defaults in the app layer.
func ParseLayoutConfig(raw string) (LayoutConfig, error) {
	if strings.TrimSpace(raw) == "" {
		return LayoutConfig{}, nil
	}
	var cfg LayoutConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return LayoutConfig{}, fmt.Errorf("alignment: layout config: %w", err)
	}
	if cfg.Version != LayoutSchemaVersion {
		return LayoutConfig{}, fmt.Errorf("alignment: layout config version %d is not supported (want %d)", cfg.Version, LayoutSchemaVersion)
	}
	switch cfg.Format {
	case FormatCSV, FormatXLSX:
	case "":
		return LayoutConfig{}, fmt.Errorf("alignment: layout config format is empty")
	default:
		return LayoutConfig{}, fmt.Errorf("alignment: layout config format %q is not supported", cfg.Format)
	}
	return cfg, nil
}

// SerializeMappingConfig encodes a mapping config for storage.
func SerializeMappingConfig(cfg MappingConfig) (string, error) {
	cfg.Version = MappingSchemaVersion
	b, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("alignment: mapping config: %w", err)
	}
	return string(b), nil
}

// SerializeLayoutConfig encodes a layout config for storage.
func SerializeLayoutConfig(cfg LayoutConfig) (string, error) {
	cfg.Version = LayoutSchemaVersion
	b, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("alignment: layout config: %w", err)
	}
	return string(b), nil
}

// normalize reconciles Mode and HasHeader (Mode wins when set) and validates
// the shape so Parse can assume a consistent config.
func (cfg *MappingConfig) normalize() error {
	switch cfg.Mode {
	case "":
		if cfg.HasHeader {
			cfg.Mode = ModeHeader
		} else {
			cfg.Mode = ModePositional
		}
	case ModeHeader:
		cfg.HasHeader = true
	case ModePositional:
		cfg.HasHeader = false
	default:
		return fmt.Errorf("alignment: mapping config mode %q is not supported", cfg.Mode)
	}
	if cfg.Mode == ModeHeader && len(cfg.Columns) == 0 && len(cfg.Positions) == 0 && len(cfg.JoinSources) == 0 {
		return fmt.Errorf("alignment: mapping config has no column mappings")
	}
	for key, chain := range cfg.Transforms {
		for _, name := range chain {
			if name == SplitSkuQuantityName {
				return fmt.Errorf("alignment: transform splitSkuQuantity on %q belongs in the splitSkuQuantity field, not a transform chain", key)
			}
			if _, ok := namedTransformerFactories[name]; !ok {
				return fmt.Errorf("alignment: mapping config references unregistered transformer %q on key %q", name, key)
			}
		}
	}
	return nil
}

// mappingKeys returns the semantic keys that can produce values, in a stable
// order (Columns keys first, then Positions keys).
func (cfg *MappingConfig) mappingKeys() []string {
	keys := make([]string, 0, len(cfg.Columns)+len(cfg.Positions))
	for k := range cfg.Columns {
		keys = append(keys, k)
	}
	for k := range cfg.Positions {
		if _, dup := cfg.Columns[k]; !dup {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}
