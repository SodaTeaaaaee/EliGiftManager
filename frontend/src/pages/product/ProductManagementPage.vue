<template>
  <div class="product-management-page">
    <h1 class="text-xl font-medium mb-4">商品管理</h1>

    <n-space vertical size="large">
      <n-space>
        <n-button type="primary" @click="openCreateDrawer">
          创建商品
        </n-button>
      </n-space>

      <n-alert v-if="error" type="error" :title="error" closable @close="error = ''" />
      <n-alert v-if="successMsg" type="success" :title="successMsg" closable @close="successMsg = ''" />

      <n-data-table
        :columns="columns"
        :data="products"
        :loading="loading"
        :row-key="(row: dto.ProductMasterDTO) => row.id"
        size="small"
      />
    </n-space>

    <!-- Create / Edit Drawer -->
    <n-drawer v-model:show="drawerVisible" :width="480" placement="right">
      <n-drawer-content :title="editingId ? '编辑商品' : '创建商品'">
        <n-form label-placement="left" label-width="140" :model="formData">
          <n-form-item label="商品名称" required>
            <n-input v-model:value="formData.name" placeholder="商品名称" />
          </n-form-item>
          <n-form-item label="供应商平台">
            <n-input v-model:value="formData.supplierPlatform" placeholder="e.g. bilibili_mall" />
          </n-form-item>
          <n-form-item label="工厂 SKU">
            <n-input v-model:value="formData.factorySku" placeholder="工厂 SKU" />
          </n-form-item>
          <n-form-item label="供应商产品引用">
            <n-input v-model:value="formData.supplierProductRef" placeholder="供应商产品引用" />
          </n-form-item>
          <n-form-item label="商品类型">
            <n-select
              v-model:value="formData.productKind"
              :options="productKindOptions"
              placeholder="选择"
            />
          </n-form-item>
          <n-form-item v-if="editingId" label="归档">
            <n-switch v-model:value="formData.archived" />
          </n-form-item>
        </n-form>

        <template #footer>
          <n-space justify="end">
            <n-button @click="drawerVisible = false">取消</n-button>
            <n-button type="primary" @click="submitForm" :loading="submitting">
              {{ editingId ? '保存' : '创建' }}
            </n-button>
          </n-space>
        </template>
      </n-drawer-content>
    </n-drawer>

    <!-- Snapshot Modal -->
    <n-modal v-model:show="snapshotModalVisible" preset="card" title="快照到波次" style="width: 400px">
      <n-space vertical size="large">
        <p v-if="snapshotTargetMaster">
          商品: <strong>{{ snapshotTargetMaster.name }}</strong> ({{ snapshotTargetMaster.factorySku }})
        </p>
        <n-select
          v-model:value="snapshotWaveId"
          :options="snapshotWaveOptions"
          :loading="snapshotWavesLoading"
          placeholder="选择目标波次"
          filterable
        />
        <n-alert v-if="snapshotMsg" type="success" :title="snapshotMsg" />
        <n-alert v-if="snapshotErr" type="error" :title="snapshotErr" />
        <n-space justify="end">
          <n-button @click="snapshotModalVisible = false">取消</n-button>
          <n-button type="primary" :loading="snapshotting" :disabled="!snapshotWaveId" @click="doSnapshot">
            确认快照
          </n-button>
        </n-space>
      </n-space>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, h } from "vue";
import {
  NButton,
  NDataTable,
  NSpace,
  NAlert,
  NDrawer,
  NDrawerContent,
  NForm,
  NFormItem,
  NInput,
  NModal,
  NSelect,
  NSwitch,
  NTag,
} from "naive-ui";
import type { DataTableColumns } from "naive-ui";
import {
  listProductMasters,
  createProductMaster,
  updateProductMaster,
  snapshotProductsForWave,
  listWaves,
} from "@/shared/lib/wails/app";
import { dto } from "@/../wailsjs/go/models";

// ── Options ──

const productKindOptions = [
  { label: "徽章 (Badge)", value: "badge" },
  { label: "立牌 (Standee)", value: "standee" },
  { label: "挂件 (Charm)", value: "charm" },
  { label: "明信片 (Postcard)", value: "postcard" },
  { label: "印刷品 (Print)", value: "print" },
  { label: "套装 (Bundle)", value: "bundle" },
  { label: "其他 (Other)", value: "other" },
];

// ── State ──

const products = ref<dto.ProductMasterDTO[]>([]);
const loading = ref(false);
const error = ref("");
const successMsg = ref("");
const drawerVisible = ref(false);
const editingId = ref<number | null>(null);
const submitting = ref(false);

// ── Form ──

const makeEmptyForm = () => ({
  name: "",
  supplierPlatform: "",
  factorySku: "",
  supplierProductRef: "",
  productKind: "other",
  archived: false,
});

const formData = reactive(makeEmptyForm());

function resetForm() {
  Object.assign(formData, makeEmptyForm());
}

// ── Snapshot state ──

const snapshotModalVisible = ref(false)
const snapshotTargetMaster = ref<any>(null)
const snapshotWaveId = ref<number | null>(null)
const snapshotWaveOptions = ref<Array<{ label: string; value: number }>>([])
const snapshotWavesLoading = ref(false)
const snapshotting = ref(false)
const snapshotMsg = ref("")
const snapshotErr = ref("")

async function openSnapshotModal(row: any) {
  snapshotTargetMaster.value = row
  snapshotWaveId.value = null
  snapshotMsg.value = ""
  snapshotErr.value = ""
  snapshotModalVisible.value = true
  snapshotWavesLoading.value = true
  try {
    const waves = await listWaves()
    snapshotWaveOptions.value = waves.map((w: any) => ({
      label: `${w.waveNo} — ${w.name}`,
      value: w.id,
    }))
  } catch (e: any) {
    snapshotErr.value = e?.message ?? String(e)
  } finally {
    snapshotWavesLoading.value = false
  }
}

async function doSnapshot() {
  if (!snapshotTargetMaster.value || !snapshotWaveId.value) return
  snapshotting.value = true
  snapshotMsg.value = ""
  snapshotErr.value = ""
  try {
    await snapshotProductsForWave({ waveId: snapshotWaveId.value, masterIds: [snapshotTargetMaster.value.id] })
    snapshotMsg.value = `已快照到波次`
  } catch (e: any) {
    snapshotErr.value = e?.message ?? String(e)
  } finally {
    snapshotting.value = false
  }
}

// ── Table columns ──

const columns: DataTableColumns<dto.ProductMasterDTO> = [
  { title: "ID", key: "id", width: 60 },
  { title: "名称", key: "name", width: 180 },
  { title: "供应商平台", key: "supplierPlatform", width: 120 },
  { title: "工厂 SKU", key: "factorySku", width: 140 },
  { title: "商品类型", key: "productKind", width: 100 },
  {
    title: "状态",
    key: "archived",
    width: 80,
    render(row) {
      return h(
        NTag,
        { type: row.archived ? "default" : "success", size: "small" },
        { default: () => (row.archived ? "已归档" : "活跃") },
      );
    },
  },
  {
    title: "操作",
    key: "actions",
    width: 160,
    render(row) {
      return h(NSpace, { size: "small" }, () => [
        h(
          NButton,
          { size: "small", onClick: () => openEditDrawer(row) },
          () => "编辑",
        ),
        h(
          NButton,
          { size: "small", type: "info", onClick: () => openSnapshotModal(row) },
          () => "快照到波次",
        ),
      ]);
    },
  },
];

// ── Actions ──

async function loadProducts() {
  loading.value = true;
  try {
    products.value = await listProductMasters();
  } catch (e: any) {
    error.value = e?.message ?? String(e);
  } finally {
    loading.value = false;
  }
}

function openCreateDrawer() {
  resetForm();
  editingId.value = null;
  drawerVisible.value = true;
}

function openEditDrawer(row: dto.ProductMasterDTO) {
  editingId.value = row.id;
  Object.assign(formData, {
    name: row.name,
    supplierPlatform: row.supplierPlatform,
    factorySku: row.factorySku,
    supplierProductRef: row.supplierProductRef,
    productKind: row.productKind,
    archived: row.archived,
  });
  drawerVisible.value = true;
}

async function submitForm() {
  if (!formData.name) {
    error.value = "商品名称不能为空";
    return;
  }
  submitting.value = true;
  error.value = "";
  try {
    if (editingId.value) {
      await updateProductMaster({ id: editingId.value, ...formData });
      successMsg.value = "商品已更新";
    } else {
      await createProductMaster({ ...formData });
      successMsg.value = "商品已创建";
    }
    drawerVisible.value = false;
    await loadProducts();
  } catch (e: any) {
    error.value = e?.message ?? String(e);
  } finally {
    submitting.value = false;
  }
}

// ── Lifecycle ──

onMounted(() => {
  loadProducts();
});
</script>
