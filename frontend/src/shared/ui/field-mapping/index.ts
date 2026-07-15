export { default as FieldMappingEditor } from './FieldMappingEditor.vue'
export type { FieldMappingDestField, FieldMappingMode, FieldMappingValue } from './types'
export {
  emptyFieldMapping,
  ensureNamespacedDestKey,
  parseMappingRules,
  serializeMappingRules,
} from './types'
export { applyMapping, bareDestFieldLeaf, validateDestFieldValue } from './previewTransform'
export type { MappedPreviewRow } from './previewTransform'
