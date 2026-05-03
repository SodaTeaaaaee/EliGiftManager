package model

// DynamicFieldMapping defines how a single column from a CSV/ZIP source maps to a
// logical field during import.  ColumnIndex is zero-based, SourceColumn is the
// original header text, and Required/DefaultValue control validation behaviour.
type DynamicFieldMapping struct {
	ColumnIndex  int    `json:"columnIndex"`
	SourceColumn string `json:"sourceColumn"`
	Required     bool   `json:"required"`
	DefaultValue string `json:"defaultValue"`
}

// ExtraDataConfig controls how arbitrary extra fields (not part of the core mapping)
// are handled during import.
//
//   - StrategyCatchAll ("catch_all"): capture every unmapped column as key–value pairs.
//   - StrategyExplicit ("explicit"): only capture columns listed in ExplicitMapping.
type ExtraDataConfig struct {
	Strategy        string                         `json:"strategy"`
	ExplicitMapping map[string]DynamicFieldMapping `json:"explicitMapping"`
}

// DynamicTemplateRules is the new (v3) template-mapping schema that replaces the
// ad-hoc JSON formats previously parsed by parseTemplateMappingRules and its V2
// counterpart.  It uses explicit column-index mapping instead of the old flat
// column-name map, avoids ambiguous number-of-fields detection, and bundles
// ExtraData handling in a single self-describing struct.
type DynamicTemplateRules struct {
	Format    string                         `json:"format"`
	HasHeader bool                           `json:"hasHeader"`
	Mapping   map[string]DynamicFieldMapping `json:"mapping"`
	ExtraData ExtraDataConfig                `json:"extraData"`
}
