import { computed, type ComputedRef } from 'vue'
import { useI18n } from 'vue-i18n'
import type { FieldMappingDestField, FieldMappingGroup } from '@/shared/ui/field-mapping'
import { SEMANTIC_GROUPS, semanticGroupOf } from './templateCatalog'

/**
 * Localized labels shared by every template screen: semantic dictionary
 * keys, the five dictionary groups, and the closed document-type set. Keys
 * missing from the message bundle fall back to the raw key so a dictionary
 * extension on the backend never renders blank.
 */
export function useTemplateLabels(): {
  semanticLabel: (key: string) => string
  documentTypeLabel: (type: string) => string
  groups: ComputedRef<FieldMappingGroup[]>
  destFieldsFor: (dictionary: string[]) => FieldMappingDestField[]
} {
  const { t, te } = useI18n()

  function semanticLabel(key: string): string {
    const labelKey = `templateEditor.semanticKeys.${key}`
    return te(labelKey) ? t(labelKey) : key
  }

  function documentTypeLabel(type: string): string {
    const key = `templates.documentTypeOptions.${type}`
    return te(key) ? t(key) : type
  }

  const groups = computed<FieldMappingGroup[]>(() =>
    SEMANTIC_GROUPS.map((group) => ({ key: group, label: t(`templateEditor.groups.${group}`) })),
  )

  function destFieldsFor(dictionary: string[]): FieldMappingDestField[] {
    return dictionary.map((key) => ({
      key,
      label: semanticLabel(key),
      tooltip: key,
      group: semanticGroupOf(key),
    }))
  }

  return { semanticLabel, documentTypeLabel, groups, destFieldsFor }
}
