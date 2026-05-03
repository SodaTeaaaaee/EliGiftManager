package model

// Tag type constants — used in ProductTag.TagType and related filtering/upsert logic.
const TagTypeLevel = "level"
const TagTypeUser = "user"

// Wave status constants — used in Wave.Status for lifecycle tracking.
const WaveStatusDraft = "draft"
const WaveStatusExported = "exported"

// Extra data strategies for template mapping rules.
const ExtraDataStrategyCatchAll = "catch_all"
const ExtraDataStrategyExplicit = "explicit"

// Template format constants — used in template configs.
const TemplateFormatCSV = "csv"
const TemplateFormatZIP = "zip"
