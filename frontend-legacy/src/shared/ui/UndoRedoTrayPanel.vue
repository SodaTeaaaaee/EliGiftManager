<template>
  <NPopover
    placement="bottom-end"
    trigger="click"
    :show-arrow="false"
    :width="320"
  >
    <template #trigger>
      <NButton
        size="small"
        secondary
        :disabled="!props.waveId"
        class="undo-tray-trigger"
      >
        <template #icon>
          <NIcon><TimeOutline /></NIcon>
        </template>
        {{ t("undoTray.title") }}
        <NTag
          v-if="recentEntries.length"
          size="tiny"
          round
          :bordered="false"
          type="info"
          class="ml-2"
        >
          {{ recentEntries.length }}
        </NTag>
      </NButton>
    </template>

    <div class="undo-tray-panel">
      <div class="tray-header">{{ t("undoTray.title") }}</div>

      <div v-if="!recentEntries.length" class="tray-empty">
        {{ t("undoTray.empty") }}
      </div>

      <ul v-else class="tray-list">
        <li
          v-for="entry in recentEntries"
          :key="`${entry.action}-${entry.id}`"
          class="tray-item"
          :class="entry.action === 'undo' ? 'is-undo' : 'is-redo'"
        >
          <NTag size="tiny" :type="entry.action === 'undo' ? 'warning' : 'info'" :bordered="false" round>
            {{ entry.action === "undo" ? t("undoTray.undoLabel") : t("undoTray.redoLabel") }}
          </NTag>
          <span class="tray-summary" :title="entry.summary">{{ entry.summary }}</span>
          <span class="tray-time">{{ formatTime(entry.timestamp) }}</span>
        </li>
      </ul>
    </div>
  </NPopover>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import { NButton, NIcon, NPopover, NTag } from "naive-ui";
import { TimeOutline } from "@vicons/ionicons5";
import { useI18n } from "@/shared/i18n";

interface TrayEntry {
  id: number;
  action: "undo" | "redo";
  summary: string;
  timestamp: number;
}

const props = defineProps<{
  waveId: number | null;
}>();

const { t } = useI18n();

const recentEntries = ref<TrayEntry[]>([]);
let nextId = 0;

// Reset tray when switching to a different wave.
watch(
  () => props.waveId,
  () => {
    recentEntries.value = [];
  },
);

/** Public API: parent components push an entry after a successful undo/redo. */
function pushEntry(action: "undo" | "redo", summary: string) {
  const entry: TrayEntry = {
    id: ++nextId,
    action,
    summary,
    timestamp: Date.now(),
  };
  recentEntries.value = [entry, ...recentEntries.value].slice(0, 12);
}

defineExpose({ pushEntry });

function formatTime(ts: number): string {
  const diff = Date.now() - ts;
  const sec = Math.floor(diff / 1000);
  if (sec < 5) return "just now";
  if (sec < 60) return `${sec}s`;
  const min = Math.floor(sec / 60);
  if (min < 60) return `${min}m`;
  const hr = Math.floor(min / 60);
  return `${hr}h`;
}
</script>

<style scoped>
.undo-tray-trigger {
  display: inline-flex;
  align-items: center;
}

.undo-tray-panel {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.tray-header {
  font-size: 0.85rem;
  font-weight: 700;
  color: var(--text);
  padding-bottom: 8px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.14);
}

.tray-empty {
  padding: 16px 4px;
  font-size: 0.8rem;
  color: var(--muted);
  text-align: center;
}

.tray-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 320px;
  overflow-y: auto;
}

.tray-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: 6px;
  font-size: 0.78rem;
}

.tray-item:hover {
  background: rgba(148, 163, 184, 0.08);
}

.tray-item.is-undo {
  border-left: 2px solid rgba(234, 179, 8, 0.6);
}

.tray-item.is-redo {
  border-left: 2px solid rgba(59, 130, 246, 0.6);
}

.tray-summary {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--text);
}

.tray-time {
  font-size: 0.7rem;
  color: var(--muted);
  white-space: nowrap;
}

.ml-2 {
  margin-left: 6px;
}
</style>
