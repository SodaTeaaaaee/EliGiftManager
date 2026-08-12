<script setup lang="ts">
/**
 * PullDemandsDialog — 波内导入页面的「拉取需求」弹窗：浏览未分派池（业务面三态 +
 * FilterBar + server 分页），批量勾选后调 batchAssignDemandToWave 拉入当前波次。
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NModal, NRadioButton, NRadioGroup } from 'naive-ui'
import { DataGrid, createColumns } from '@/shared/ui/data-grid'
import { FilterBar } from '@/shared/ui/filter-bar'
import { useFeedback } from '@/shared/ui/feedback'
import { batchAssignDemandToWave } from '@/shared/api/bridge'
import { useInboxGrid } from '@/pages/inbox/inbox-grid/useInboxGrid'
import { buildInboxColumns } from '@/pages/inbox/inbox-grid/columns'
import { kindsFromSurface, surfaceFromKinds, type BusinessSurface } from '@/pages/inbox/inbox-grid/businessSurface'

const props = defineProps<{ show: boolean; waveId: number }>()
const emit = defineEmits<{ 'update:show': [boolean]; pulled: [count: number] }>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()

// 未分派池：unscoped 实例，assignment 固定为 'unassigned'。syncFiltersToUrl
// 关掉 URL 同步——弹窗与波内 intake 网格同路由同 schema，双向同步会把弹窗的
// 筛选/assignment 写进 wave 路由 query（反之亦然），互相串扰。
const grid = useInboxGrid({ syncFiltersToUrl: false })
grid.assignment.value = 'unassigned'

// 弹窗常驻挂载（NModal 只隐藏不销毁），拉走后池内容会变旧——每次打开时
// 重取一页，保证刚被拉走的需求立即消失。
watch(
  () => props.show,
  (show) => {
    if (show) void grid.fetchPage()
  },
)

const columns = computed(() => createColumns(buildInboxColumns(t)))

const surface = computed<BusinessSurface>(() => surfaceFromKinds(grid.filters.state.demandKind))

function handleSurfaceChange(next: BusinessSurface): void {
  grid.filters.setEnumValues('demandKind', kindsFromSurface(next))
}

function handleSelectedKeysChange(keys: Array<string | number>): void {
  grid.selectedKeys.value = keys as number[]
}

function handleUpdateShow(value: boolean): void {
  emit('update:show', value)
}

const pulling = ref(false)

async function handlePull(): Promise<void> {
  if (grid.selectedKeys.value.length === 0) return
  pulling.value = true
  try {
    const result = await batchAssignDemandToWave({ waveId: props.waveId, docIds: grid.selectedKeys.value })
    if (result.failureCount > 0) feedback.error(t('waveWorkspace.intake.pullSomeFailed', { count: result.failureCount }))
    else feedback.success(t('feedback.success'))
    grid.selectedKeys.value = []
    // 拉取成功（含部分成功）后立刻刷新未分派池，避免刚拉走的需求残留显示。
    void grid.fetchPage()
    emit('pulled', result.successCount)
    emit('update:show', false)
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    pulling.value = false
  }
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    :title="t('waveWorkspace.intake.pullDemands')"
    style="width: min(960px, 94vw)"
    @update:show="handleUpdateShow"
  >
    <div class="pull-demands-dialog">
      <div class="pull-demands-dialog__surface">
        <span class="pull-demands-dialog__label">{{ t('inbox.filters.businessSurface') }}</span>
        <NRadioGroup :value="surface" @update:value="handleSurfaceChange">
          <NRadioButton value="all">{{ t('inbox.surface.all') }}</NRadioButton>
          <NRadioButton value="membership_entitlement">{{ t('inbox.surface.membership') }}</NRadioButton>
          <NRadioButton value="retail_order">{{ t('inbox.surface.retail') }}</NRadioButton>
        </NRadioGroup>
      </div>

      <FilterBar :filters="grid.filters" />

      <DataGrid
        :columns="columns"
        :rows="grid.rows.value"
        row-key="demandDocumentId"
        selectable
        :selected-keys="grid.selectedKeys"
        :loading="grid.loading.value"
        :pagination="{
          server: {
            total: grid.totalCount.value,
            page: grid.page.value,
            pageSize: grid.pageSize.value,
            onChange: grid.onPageChange,
            onSort: grid.onSort,
          },
        }"
        :empty="{ title: t('inbox.empty.noneUnassigned') }"
        @update:selected-keys="handleSelectedKeysChange"
      />
    </div>

    <template #footer>
      <div class="pull-demands-dialog__footer">
        <span class="pull-demands-dialog__selected">
          {{ t('inbox.batch.selected', { n: grid.selectedKeys.value.length }) }}
        </span>
        <div class="pull-demands-dialog__footer-actions">
          <NButton @click="handleUpdateShow(false)">{{ t('common.cancel') }}</NButton>
          <NButton
            type="primary"
            :loading="pulling"
            :disabled="grid.selectedKeys.value.length === 0"
            @click="handlePull"
          >
            {{ t('waveWorkspace.intake.pullDemands') }}
          </NButton>
        </div>
      </div>
    </template>
  </NModal>
</template>

<style scoped>
.pull-demands-dialog {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.pull-demands-dialog__surface {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.pull-demands-dialog__label {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-secondary);
}

.pull-demands-dialog__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.pull-demands-dialog__selected {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-secondary);
}

.pull-demands-dialog__footer-actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
</style>
