<script setup lang="ts">
import { computed } from 'vue'
import { NDrawer, NDrawerContent } from 'naive-ui'
import { useI18n } from 'vue-i18n'

/**
 * DetailDrawer — the right-side panel standard (plan 3.3.2's row detail
 * side-panel, and every future "inspect one thing without leaving the
 * grid" surface). Wraps NDrawer/NDrawerContent (native focus trap + Esc-to-
 * close + scrim) with token-styled header/footer zones and the app's two
 * standard widths.
 */
const props = withDefaults(
  defineProps<{
    show: boolean
    title?: string
    /** `md` for a compact inspector (~420px), `lg` for a full row-detail panel (~640px). */
    size?: 'md' | 'lg'
    closable?: boolean
  }>(),
  {
    title: undefined,
    size: 'md',
    closable: true,
  },
)

const emit = defineEmits<{ 'update:show': [boolean] }>()

const { t } = useI18n({ useScope: 'global' })

const widthPx = computed(() => (props.size === 'lg' ? 640 : 420))

function handleUpdateShow(value: boolean) {
  emit('update:show', value)
}
</script>

<template>
  <NDrawer
    :show="show"
    :width="widthPx"
    placement="right"
    :close-on-esc="closable"
    :mask-closable="closable"
    :trap-focus="true"
    @update:show="handleUpdateShow"
  >
    <NDrawerContent :native-scrollbar="false" class="detail-drawer__content">
      <template #header>
        <div class="detail-drawer__header">
          <h2 class="detail-drawer__title">
            <slot name="title">{{ title }}</slot>
          </h2>
          <button
            v-if="closable"
            type="button"
            class="detail-drawer__close"
            :aria-label="t('common.close')"
            @click="handleUpdateShow(false)"
          >
            <svg viewBox="0 0 16 16" width="16" height="16" aria-hidden="true">
              <path
                d="M3 3 L13 13 M13 3 L3 13"
                stroke="currentColor"
                stroke-width="1.5"
                stroke-linecap="round"
                fill="none"
              />
            </svg>
          </button>
        </div>
      </template>
      <div class="detail-drawer__body">
        <slot />
      </div>
      <template v-if="$slots.footer" #footer>
        <div class="detail-drawer__footer">
          <slot name="footer" />
        </div>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>

<style scoped>
.detail-drawer__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.detail-drawer__title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-lg);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.detail-drawer__close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: var(--radius-sm);
  border: none;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition:
    background var(--duration-fast) var(--ease-out),
    color var(--duration-fast) var(--ease-out);
}

.detail-drawer__close:hover {
  background: var(--color-inset);
  color: var(--color-text-primary);
}

.detail-drawer__close:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}

.detail-drawer__body {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  font-family: var(--font-body);
  color: var(--color-text-primary);
}

.detail-drawer__footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>

<style>
/* NDrawerContent renders its header/body/footer as un-scoped internal DOM
   nodes we cannot reach with `scoped` attribute selectors from inside our
   own template (the header/footer slot content IS scoped correctly; this
   block only re-tokenizes the surrounding chrome naive-ui itself renders). */
.detail-drawer__content.n-drawer-content {
  background: var(--color-surface-raised);
  color: var(--color-text-primary);
}

.detail-drawer__content .n-drawer-header,
.detail-drawer__content .n-drawer-footer {
  border-color: var(--color-border);
}

.detail-drawer__content .n-drawer-body-content-wrapper {
  padding: var(--card-padding);
}
</style>
