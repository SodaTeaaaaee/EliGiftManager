<template>
  <div class="customer-crm-page">
    <div class="crm-header mb-6">
      <div>
        <div class="app-kicker">{{ t("nav.customers") }}</div>
        <h1 class="app-title mt-2">{{ t("customer.title") }}</h1>
        <p class="app-copy mt-2">{{ t("customer.subtitle") }}</p>
      </div>
    </div>

    <!-- Tabs -->
    <n-tabs v-model:value="activeTab" type="line" animated>
      <!-- Tab 1: Customer List -->
      <n-tab-pane name="list" :tab="t('customer.tabList')">
        <n-space vertical size="large">
          <!-- Stats Cards -->
          <div class="stats-grid">
            <n-card class="stat-card">
              <div class="stat-label">{{ t("customer.memberCount") }}</div>
              <div class="stat-val text-primary">{{ stats.customerCount }}</div>
            </n-card>
            <n-card class="stat-card">
              <div class="stat-label">{{ t("customer.addressCount") }}</div>
              <div class="stat-val text-success">{{ stats.addressCount }}</div>
            </n-card>
            <n-card class="stat-card">
              <div class="stat-label">{{ t("customer.missingAddresses") }}</div>
              <div class="stat-val text-error">{{ stats.missingAddressCount }}</div>
            </n-card>
          </div>

          <!-- Toolbar -->
          <div class="crm-toolbar card-blur">
            <n-space align="center" justify="space-between" style="width: 100%">
              <n-space align="center" size="medium">
                <n-input
                  v-model:value="filters.keyword"
                  :placeholder="t('customer.searchPlaceholder')"
                  clearable
                  style="width: 260px"
                  @update:value="loadCustomers"
                />
                <n-select
                  v-model:value="filters.platform"
                  :options="platformOptions"
                  :placeholder="t('customer.allPlatforms')"
                  clearable
                  style="width: 160px"
                  @update:value="loadCustomers"
                />
                <n-space align="center" size="small" style="margin-left: 8px">
                  <span class="text-xs text-muted">{{ t("customer.showMissingOnly") }}</span>
                  <n-switch v-model:value="filters.missingOnly" @update:value="loadCustomers" />
                </n-space>
              </n-space>
              <n-button type="primary" @click="openCreateModal">
                {{ t("customer.create") }}
              </n-button>
            </n-space>
          </div>

          <!-- Data Table -->
          <n-data-table
            :columns="columns"
            :data="customers"
            :loading="loading"
            :row-key="(row: any) => row.id"
            :row-props="rowProps"
            size="small"
            class="card-blur"
          />
        </n-space>
      </n-tab-pane>

      <!-- Tab 2: Suggested Merges -->
      <n-tab-pane name="suggestions" :tab="t('customer.tabSuggestions')">
        <n-space vertical size="large">
          <div class="suggestion-header card-blur">
            <div class="app-heading-sm">{{ t("suggestedMerges.title") }}</div>
            <p class="app-copy mt-1">{{ t("suggestedMerges.subtitle") }}</p>
          </div>

          <n-empty v-if="suggestions.length === 0" :description="t('suggestedMerges.empty')" />

          <div v-else class="suggestions-list">
            <n-card
              v-for="s in suggestions"
              :key="s.id"
              class="suggestion-item card-blur"
              size="small"
            >
              <div class="suggestion-meta mb-3">
                <n-space justify="space-between" align="center">
                  <n-tag type="warning" round>
                    {{ t("suggestedMerges.reason") }}: {{ s.reason }}
                  </n-tag>
                  <n-space>
                    <n-button size="small" type="primary" @click="executeSuggestionMerge(s)">
                      {{ t("suggestedMerges.actionMerge") }}
                    </n-button>
                    <n-button size="small" secondary @click="dismissSuggestion(s.id)">
                      {{ t("suggestedMerges.actionDismiss") }}
                    </n-button>
                  </n-space>
                </n-space>
              </div>

              <div class="profiles-comparison">
                <!-- Target Profile (Kept) -->
                <div class="profile-card primary-border">
                  <div class="profile-header mb-2">
                    <span class="profile-badge badge-keep">Keep (Target)</span>
                    <strong class="profile-name">{{ s.targetProfile.displayName }}</strong>
                    <span class="profile-id">ID: {{ s.targetProfile.id }}</span>
                  </div>
                  <div class="profile-details">
                    <div class="detail-row">
                      <span class="detail-label">Identities:</span>
                      <n-space size="mini">
                        <n-tag
                          v-for="ident in (s.targetProfile.identities || [])"
                          :key="ident.id"
                          size="tiny"
                          round
                          type="info"
                        >
                          {{ ident.identityPlatform }}: {{ ident.identityValue }}
                        </n-tag>
                      </n-space>
                    </div>
                    <div class="detail-row mt-1">
                      <span class="detail-label">Addresses:</span>
                      <span class="text-xs text-muted">{{ (s.targetProfile.addresses || []).length }} address(es)</span>
                    </div>
                  </div>
                </div>

                <div class="comparison-arrow">➔</div>

                <!-- Source Profile (Deleted) -->
                <div class="profile-card secondary-border">
                  <div class="profile-header mb-2">
                    <span class="profile-badge badge-delete">Merge & Delete (Source)</span>
                    <strong class="profile-name">{{ s.sourceProfile.displayName }}</strong>
                    <span class="profile-id">ID: {{ s.sourceProfile.id }}</span>
                  </div>
                  <div class="profile-details">
                    <div class="detail-row">
                      <span class="detail-label">Identities:</span>
                      <n-space size="mini">
                        <n-tag
                          v-for="ident in (s.sourceProfile.identities || [])"
                          :key="ident.id"
                          size="tiny"
                          round
                          type="info"
                        >
                          {{ ident.identityPlatform }}: {{ ident.identityValue }}
                        </n-tag>
                      </n-space>
                    </div>
                    <div class="detail-row mt-1">
                      <span class="detail-label">Addresses:</span>
                      <span class="text-xs text-muted">{{ (s.sourceProfile.addresses || []).length }} address(es)</span>
                    </div>
                  </div>
                </div>
              </div>
            </n-card>
          </div>
        </n-space>
      </n-tab-pane>

      <!-- Tab 3: Manual Merge -->
      <n-tab-pane name="manual-merge" :tab="t('customer.tabManualMerge')">
        <n-card class="card-blur" style="max-width: 600px; margin: 0 auto">
          <div class="app-heading-sm mb-4">{{ t("merge.title") }}</div>
          <p class="app-copy mb-6">{{ t("merge.subtitle") }}</p>

          <n-form label-placement="top">
            <n-form-item :label="t('merge.sourceProfile')" required>
              <n-select
                v-model:value="manualMergeData.sourceProfileId"
                :options="customerOptionsFiltered"
                placeholder="选择源账户（合并后删除）"
                filterable
              />
              <template #feedback>
                <span class="text-xs text-muted">{{ t("merge.sourceHint") }}</span>
              </template>
            </n-form-item>

            <n-form-item :label="t('merge.targetProfile')" required>
              <n-select
                v-model:value="manualMergeData.targetProfileId"
                :options="customerOptions"
                placeholder="选择目标账户（合并后保留）"
                filterable
              />
              <template #feedback>
                <span class="text-xs text-muted">{{ t("merge.targetHint") }}</span>
              </template>
            </n-form-item>

            <n-alert type="warning" class="mb-6">
              {{ t("merge.confirmDesc") }}
            </n-alert>

            <n-button
              type="primary"
              block
              :disabled="!manualMergeData.sourceProfileId || !manualMergeData.targetProfileId || manualMergeData.sourceProfileId === manualMergeData.targetProfileId"
              @click="confirmManualMerge"
            >
              {{ t("merge.execute") }}
            </n-button>
          </n-form>
        </n-card>
      </n-tab-pane>
    </n-tabs>

    <!-- Customer Drawer (Details & Address & Identity management) -->
    <n-drawer v-model:show="drawerVisible" :width="560" placement="right">
      <n-drawer-content :title="selectedCustomer ? selectedCustomer.displayName : ''" closable>
        <div v-if="selectedCustomer" class="drawer-inner">
          <n-tabs type="segment" animated>
            <!-- Drawer Tab 1: Profile Info -->
            <n-tab-pane name="basic" tab="基本信息">
              <n-space vertical size="large" class="mt-3">
                <n-form label-placement="top" :model="selectedCustomer">
                  <n-form-item :label="t('customer.displayName')" required>
                    <n-input v-model:value="selectedCustomer.displayName" />
                  </n-form-item>
                  <n-form-item :label="t('customer.profileType')" required>
                    <n-select
                      v-model:value="selectedCustomer.profileType"
                      :options="customerTypeOptions"
                    />
                  </n-form-item>
                  <n-form-item :label="t('customer.extraData')">
                    <n-input
                      v-model:value="selectedCustomer.extraData"
                      type="textarea"
                      placeholder='JSON 格式附加信息，例如 {"notes": "VIP"}'
                      :rows="3"
                    />
                  </n-form-item>
                </n-form>
                <n-space justify="end">
                  <n-button type="primary" @click="saveProfileDetails">保存基本信息</n-button>
                </n-space>
              </n-space>
            </n-tab-pane>

            <!-- Drawer Tab 2: Identities -->
            <n-tab-pane name="identities" :tab="t('customer.identities')">
              <n-space vertical size="large" class="mt-3">
                <div class="flex justify-between align-center">
                  <span class="text-xs text-muted">管理关联的第三方平台账号标识</span>
                  <n-button size="small" type="primary" @click="showAddIdentity = true">
                    {{ t("customer.addIdentity") }}
                  </n-button>
                </div>

                <div v-if="(selectedCustomer.identities || []).length === 0" class="empty-state">
                  <n-empty :description="t('customer.noIdentities')" />
                </div>

                <div v-else class="identities-list">
                  <div
                    v-for="ident in (selectedCustomer.identities || [])"
                    :key="ident.id"
                    class="identity-item card-blur"
                  >
                    <div class="flex justify-between align-center">
                      <div>
                        <strong class="text-sm">{{ ident.identityPlatform }}</strong>
                        <div class="text-xs text-muted mt-0-5">
                          Value: <code>{{ ident.identityValue }}</code> (Type: {{ ident.identityType }})
                        </div>
                      </div>
                      <n-space align="center">
                        <n-tag v-if="ident.isPrimary" size="small" type="success">Primary</n-tag>
                        <n-popconfirm
                          @positive-click="removeIdentity(ident.id)"
                          trigger="click"
                          positive-text="删除"
                          negative-text="取消"
                        >
                          <template #trigger>
                            <n-button size="tiny" type="error" secondary>删除</n-button>
                          </template>
                          确认删除此平台身份关联？
                        </n-popconfirm>
                      </n-space>
                    </div>
                  </div>
                </div>
              </n-space>
            </n-tab-pane>

            <!-- Drawer Tab 3: Address Book -->
            <n-tab-pane name="addresses" :tab="t('profile.tabAddresses')">
              <n-space vertical size="large" class="mt-3">
                <div class="flex justify-between align-center">
                  <span class="text-xs text-muted">管理本档案持有的历史收货地址信息</span>
                  <n-button size="small" type="primary" @click="openCreateAddressModal">
                    {{ t("address.create") }}
                  </n-button>
                </div>

                <div v-if="(selectedCustomer.addresses || []).length === 0" class="empty-state">
                  <n-empty :description="t('address.noAddresses')" />
                </div>

                <div v-else class="addresses-list">
                  <div
                    v-for="addr in (selectedCustomer.addresses || [])"
                    :key="addr.id"
                    class="address-item card-blur"
                  >
                    <div class="flex justify-between align-center">
                      <strong>{{ addr.recipientName }}</strong>
                      <n-space size="small" align="center">
                        <n-tag v-if="addr.isDefault" type="success" size="small" round>默认</n-tag>
                        <n-tag v-if="addr.isTest" type="warning" size="small" round>测试</n-tag>
                      </n-space>
                    </div>
                    <div class="address-phone mt-1 text-xs">{{ addr.phone }}</div>
                    <div class="address-details mt-1 text-xs text-muted">
                      {{ addr.province }}{{ addr.city }}{{ addr.district }}{{ addr.addressLine1 }} {{ addr.addressLine2 }}
                    </div>
                    <div class="address-actions mt-3">
                      <n-space>
                        <n-button
                          v-if="!addr.isDefault"
                          size="tiny"
                          secondary
                          @click="setAddressAsDefault(addr)"
                        >
                          设为默认
                        </n-button>
                        <n-button size="tiny" secondary @click="openEditAddressModal(addr)">
                          编辑
                        </n-button>
                        <n-popconfirm
                          @positive-click="removeAddress(addr.id)"
                          trigger="click"
                          positive-text="删除"
                          negative-text="取消"
                        >
                          <template #trigger>
                            <n-button size="tiny" type="error" secondary>删除</n-button>
                          </template>
                          确认删除此地址？
                        </n-popconfirm>
                      </n-space>
                    </div>
                  </div>
                </div>
              </n-space>
            </n-tab-pane>
          </n-tabs>
        </div>
      </n-drawer-content>
    </n-drawer>

    <!-- Create Profile Modal -->
    <n-modal v-model:show="showCreateModal" preset="card" :title="t('customer.create')" style="max-width: 480px">
      <n-form :model="createForm" label-placement="top">
        <n-form-item :label="t('customer.displayName')" required>
          <n-input v-model:value="createForm.displayName" placeholder="输入客户名称" />
        </n-form-item>
        <n-form-item :label="t('customer.profileType')" required>
          <n-select v-model:value="createForm.profileType" :options="customerTypeOptions" />
        </n-form-item>
        <n-form-item :label="t('customer.extraData')">
          <n-input
            v-model:value="createForm.extraData"
            type="textarea"
            placeholder='{"notes": ""}'
            :rows="2"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showCreateModal = false">{{ t("common.cancel") }}</n-button>
          <n-button type="primary" :disabled="!createForm.displayName" @click="submitCreateProfile">
            {{ t("common.save") }}
          </n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- Add Identity Modal -->
    <n-modal v-model:show="showAddIdentity" preset="card" :title="t('customer.addIdentity')" style="max-width: 460px">
      <n-form :model="identityForm" label-placement="top">
        <n-form-item :label="t('customer.platform')" required>
          <n-input v-model:value="identityForm.identityPlatform" placeholder="例如 patreon, bilibili, gumroad" />
        </n-form-item>
        <n-form-item :label="t('customer.identityValue')" required>
          <n-input v-model:value="identityForm.identityValue" placeholder="输入用户UID或邮箱地址" />
        </n-form-item>
        <n-form-item :label="t('customer.identityType')" required>
          <n-select v-model:value="identityForm.identityType" :options="identityTypeOptions" />
        </n-form-item>
        <n-form-item :label="t('customer.isPrimary')">
          <n-switch v-model:value="identityForm.isPrimary" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showAddIdentity = false">{{ t("common.cancel") }}</n-button>
          <n-button
            type="primary"
            :disabled="!identityForm.identityPlatform || !identityForm.identityValue"
            @click="submitAddIdentity"
          >
            {{ t("common.save") }}
          </n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- Create/Edit Address Modal -->
    <n-modal v-model:show="showAddressModal" preset="card" :title="editingAddressId ? t('address.edit') : t('address.create')" style="max-width: 520px">
      <n-form :model="addressForm" label-placement="left" label-width="110">
        <n-form-item :label="t('address.label')" required>
          <n-input v-model:value="addressForm.label" placeholder="例如 家, 公司" />
        </n-form-item>
        <n-form-item :label="t('address.recipientName')" required>
          <n-input v-model:value="addressForm.recipientName" />
        </n-form-item>
        <n-form-item :label="t('address.phone')" required>
          <n-input v-model:value="addressForm.phone" />
        </n-form-item>
        <n-form-item :label="t('address.country')">
          <n-input v-model:value="addressForm.country" />
        </n-form-item>
        <n-form-item :label="t('address.province')">
          <n-input v-model:value="addressForm.province" />
        </n-form-item>
        <n-form-item :label="t('address.city')">
          <n-input v-model:value="addressForm.city" />
        </n-form-item>
        <n-form-item :label="t('address.district')">
          <n-input v-model:value="addressForm.district" />
        </n-form-item>
        <n-form-item :label="t('address.addressLine1')" required>
          <n-input v-model:value="addressForm.addressLine1" />
        </n-form-item>
        <n-form-item :label="t('address.addressLine2')">
          <n-input v-model:value="addressForm.addressLine2" />
        </n-form-item>
        <n-form-item :label="t('address.postalCode')">
          <n-input v-model:value="addressForm.postalCode" />
        </n-form-item>
        <n-form-item :label="t('address.isDefault')">
          <n-switch v-model:value="addressForm.isDefault" />
        </n-form-item>
        <n-form-item :label="t('address.isTest')">
          <n-switch v-model:value="addressForm.isTest" />
        </n-form-item>
        <n-form-item :label="t('address.validationStatus')">
          <n-select v-model:value="addressForm.validationStatus" :options="addressValidationOptions" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showAddressModal = false; editingAddressId = null">{{ t("common.cancel") }}</n-button>
          <n-button
            type="primary"
            :disabled="!addressForm.label || !addressForm.recipientName || !addressForm.phone || !addressForm.addressLine1"
            @click="submitAddress"
          >
            {{ t("common.save") }}
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, h } from "vue";
import { useRouter } from "vue-router";
import {
  NTabs,
  NTabPane,
  NSpace,
  NButton,
  NCard,
  NInput,
  NSelect,
  NSwitch,
  NDataTable,
  NDrawer,
  NDrawerContent,
  NForm,
  NFormItem,
  NModal,
  NTag,
  NEmpty,
  NPopconfirm,
  NAvatar,
  NAlert,
  useMessage,
  useDialog,
} from "naive-ui";
import type { DataTableColumns } from "naive-ui";
import { useI18n } from "@/shared/i18n";
import {
  listCustomerProfiles,
  createCustomerProfile,
  updateCustomerProfile,
  deleteCustomerProfile,
  addCustomerIdentity,
  deleteCustomerIdentity,
  getMergeSuggestions,
  dismissMergeSuggestion,
  mergeProfiles,
  createAddress,
  updateAddress,
  deleteAddress,
  listProfiles,
} from "@/shared/lib/wails/app";
import { dto } from "@/../wailsjs/go/models";

const { t } = useI18n();
const message = useMessage();
const dialog = useDialog();
const router = useRouter();

// ── Tabs State ──
const activeTab = ref("list");

// ── CRM List State ──
const customers = ref<dto.CustomerProfileDTO[]>([]);
const loading = ref(false);
const suggestions = ref<dto.MergeSuggestionDTO[]>([]);
const uniquePlatforms = ref<string[]>([]);

const filters = reactive({
  keyword: "",
  platform: "",
  missingOnly: false,
});

const stats = computed(() => {
  const list = customers.value || [];
  let customerCount = list.length;
  let addressCount = 0;
  let missingAddressCount = 0;

  list.forEach((c) => {
    addressCount += c.activeAddressCount;
    if (c.activeAddressCount === 0) {
      missingAddressCount++;
    }
  });

  return {
    customerCount,
    addressCount,
    missingAddressCount,
  };
});

const platformOptions = computed(() => {
  const options = [{ label: t("customer.allPlatforms"), value: "" }];
  uniquePlatforms.value.forEach((p) => {
    options.push({ label: p, value: p });
  });
  return options;
});

const customerTypeOptions = [
  { label: t("customer.typeOptions.member"), value: "member" },
  { label: t("customer.typeOptions.buyer"), value: "buyer" },
  { label: t("customer.typeOptions.mixed"), value: "mixed" },
  { label: t("customer.typeOptions.manual"), value: "manual" },
];

const identityTypeOptions = [
  { label: "Platform UID", value: "platform_uid" },
  { label: "Email", value: "email" },
  { label: "Username", value: "username" },
  { label: "External Buyer ID", value: "external_buyer_id" },
];

const addressValidationOptions = [
  { label: t("address.status.unvalidated"), value: "unvalidated" },
  { label: t("address.status.valid"), value: "valid" },
  { label: t("address.status.invalid"), value: "invalid" },
];

// ── Columns ──
const columns: DataTableColumns<dto.CustomerProfileDTO> = [
  {
    title: t("customer.displayName"),
    key: "displayName",
    minWidth: 160,
    render(row) {
      return h(NSpace, { align: "center", size: "small" }, () => [
        h(
          NAvatar,
          {
            size: 30,
            round: true,
            color: "var(--accent-surface)",
            style: { color: "var(--accent)", fontWeight: "700" },
          },
          () => (row.displayName || "").slice(0, 1).toUpperCase()
        ),
        h("span", { style: { fontWeight: "500" } }, row.displayName || ""),
      ]);
    },
  },
  {
    title: t("customer.profileType"),
    key: "profileType",
    width: 100,
    render(row) {
      const typeMap: Record<string, { label: string; type: any }> = {
        member: { label: "Member", type: "info" },
        buyer: { label: "Buyer", type: "warning" },
        mixed: { label: "Mixed", type: "success" },
        manual: { label: "Manual", type: "default" },
      };
      const info = typeMap[row.profileType] || { label: row.profileType, type: "default" };
      return h(NTag, { size: "small", type: info.type, round: true }, () => info.label);
    },
  },
  {
    title: t("customer.identities"),
    key: "identities",
    minWidth: 200,
    render(row) {
      const idents = row.identities || [];
      if (idents.length === 0) return h("span", { class: "text-muted text-xs" }, "-");
      return h(
        NSpace,
        { size: "mini" },
        () =>
          idents.map((i) =>
            h(
              NTag,
              { size: "tiny", type: "info", round: true },
              () => `${i.identityPlatform}: ${i.identityValue}`
            )
          )
      );
    },
  },
  {
    title: t("address.validationStatus"),
    key: "addressStatus",
    width: 120,
    render(row) {
      const count = row.activeAddressCount;
      if (count > 0) {
        return h(NTag, { size: "small", type: "success", round: true }, () => `已完备 (${count})`);
      }
      return h(NTag, { size: "small", type: "error", round: true }, () => "缺地址");
    },
  },
  {
    title: "默认收件人",
    key: "defaultRecipient",
    width: 130,
    render(row) {
      const addrs = row.addresses || [];
      const def = addrs.find((a) => a.isDefault);
      if (def) return def.recipientName;
      if (addrs.length > 0) return addrs[0].recipientName;
      return "-";
    },
  },
  {
    title: "收货电话",
    key: "defaultPhone",
    width: 130,
    render(row) {
      const addrs = row.addresses || [];
      const def = addrs.find((a) => a.isDefault);
      if (def) return def.phone;
      if (addrs.length > 0) return addrs[0].phone;
      return "-";
    },
  },
  {
    title: t("nav.addresses"),
    key: "defaultAddress",
    minWidth: 200,
    ellipsis: { tooltip: true },
    render(row) {
      const addrs = row.addresses || [];
      const def = addrs.find((a) => a.isDefault);
      const addr = def || addrs[0];
      if (addr) {
        return `${addr.province}${addr.city}${addr.district}${addr.addressLine1}`;
      }
      return "-";
    },
  },
  {
    title: "操作",
    key: "actions",
    width: 160,
    render(row) {
      return h(NSpace, { size: 4 }, () => [
        h(
          NButton,
          {
            size: "tiny",
            secondary: true,
            onClick: (e: MouseEvent) => {
              e.stopPropagation();
              router.push(`/customers/${row.id}`);
            },
          },
          () => "详情"
        ),
        h(
          NPopconfirm,
          {
            onPositiveClick: () => confirmDeleteProfile(row.id),
            positiveText: "删除",
            negativeText: "取消",
          },
          {
            trigger: () =>
              h(
                NButton,
                { size: "tiny", type: "error", secondary: true, onClick: (e: MouseEvent) => e.stopPropagation() },
                () => "删除"
              ),
            default: () => "删除该客户档案将清除其关联数据。确认删除？",
          }
        ),
      ]);
    },
  },
];

// Row events
function rowProps(row: dto.CustomerProfileDTO) {
  return {
    style: "cursor: pointer;",
    onClick: () => {
      openDrawer(row);
    },
  };
}

// ── CRUD Actions ──
async function loadCustomers() {
  loading.value = true;
  try {
    customers.value = (await listCustomerProfiles(filters.keyword, filters.platform, filters.missingOnly)) || [];
  } catch (e: any) {
    message.error(e?.message ?? String(e));
  } finally {
    loading.value = false;
  }
}

async function loadUniquePlatforms() {
  try {
    const list = (await listProfiles()) || [];
    const set = new Set<string>();
    list.forEach((p) => {
      if (p.sourceChannel) set.add(p.sourceChannel);
    });
    uniquePlatforms.value = Array.from(set);
  } catch (e) {
    console.error("load integration profiles failed", e);
  }
}

async function confirmDeleteProfile(id: number) {
  try {
    await deleteCustomerProfile(id);
    message.success("删除成功");
    await loadCustomers();
  } catch (e: any) {
    message.error(e?.message ?? String(e));
  }
}

// ── Drawer State & Actions ──
const drawerVisible = ref(false);
const selectedCustomer = ref<dto.CustomerProfileDTO | null>(null);

async function openDrawer(row: dto.CustomerProfileDTO) {
  try {
    selectedCustomer.value = await getCustomerProfile(row.id);
    drawerVisible.value = true;
  } catch (e: any) {
    message.error(e?.message ?? String(e));
  }
}

function handleDrawerVisibility(val: boolean) {
  if (!val) {
    selectedCustomer.value = null;
  }
}

async function saveProfileDetails() {
  if (!selectedCustomer.value) return;
  try {
    await updateCustomerProfile({
      id: selectedCustomer.value.id,
      displayName: selectedCustomer.value.displayName,
      profileType: selectedCustomer.value.profileType,
      extraData: selectedCustomer.value.extraData,
    });
    message.success("档案信息已更新");
    await loadCustomers();
  } catch (e: any) {
    message.error(e?.message ?? String(e));
  }
}

// ── Identity Management ──
const showAddIdentity = ref(false);
const identityForm = reactive({
  identityPlatform: "",
  identityValue: "",
  identityType: "platform_uid",
  isPrimary: false,
  extraData: "",
});

function resetIdentityForm() {
  identityForm.identityPlatform = "";
  identityForm.identityValue = "";
  identityForm.identityType = "platform_uid";
  identityForm.isPrimary = false;
  identityForm.extraData = "";
}

async function submitAddIdentity() {
  if (!selectedCustomer.value) return;
  try {
    await addCustomerIdentity({
      customerProfileId: selectedCustomer.value.id,
      identityPlatform: identityForm.identityPlatform,
      identityValue: identityForm.identityValue,
      identityType: identityForm.identityType,
      isPrimary: identityForm.isPrimary,
      extraData: identityForm.extraData,
    });
    message.success("身份关联已添加");
    showAddIdentity.value = false;
    resetIdentityForm();

    // Reload details
    selectedCustomer.value = await getCustomerProfile(selectedCustomer.value.id);
    await loadCustomers();
  } catch (e: any) {
    message.error(e?.message ?? String(e));
  }
}

async function removeIdentity(id: number) {
  if (!selectedCustomer.value) return;
  try {
    await deleteCustomerIdentity(id);
    message.success("身份关联已移除");

    // Reload details
    selectedCustomer.value = await getCustomerProfile(selectedCustomer.value.id);
    await loadCustomers();
  } catch (e: any) {
    message.error(e?.message ?? String(e));
  }
}

// ── Address Management ──
const showAddressModal = ref(false);
const editingAddressId = ref<number | null>(null);
const addressForm = reactive({
  label: "",
  recipientName: "",
  phone: "",
  country: "CN",
  province: "",
  city: "",
  district: "",
  addressLine1: "",
  addressLine2: "",
  postalCode: "",
  isDefault: false,
  isTest: false,
  validationStatus: "unvalidated",
  validationDetail: "",
  extraData: "",
});

function resetAddressForm() {
  addressForm.label = "";
  addressForm.recipientName = "";
  addressForm.phone = "";
  addressForm.country = "CN";
  addressForm.province = "";
  addressForm.city = "";
  addressForm.district = "";
  addressForm.addressLine1 = "";
  addressForm.addressLine2 = "";
  addressForm.postalCode = "";
  addressForm.isDefault = false;
  addressForm.isTest = false;
  addressForm.validationStatus = "unvalidated";
  addressForm.validationDetail = "";
  addressForm.extraData = "";
}

function openCreateAddressModal() {
  resetAddressForm();
  editingAddressId.value = null;
  showAddressModal.value = true;
}

function openEditAddressModal(addr: dto.CustomerAddressDTO) {
  editingAddressId.value = addr.id;
  addressForm.label = addr.label;
  addressForm.recipientName = addr.recipientName;
  addressForm.phone = addr.phone;
  addressForm.country = addr.country;
  addressForm.province = addr.province;
  addressForm.city = addr.city;
  addressForm.district = addr.district;
  addressForm.addressLine1 = addr.addressLine1;
  addressForm.addressLine2 = addr.addressLine2;
  addressForm.postalCode = addr.postalCode;
  addressForm.isDefault = addr.isDefault;
  addressForm.isTest = addr.isTest;
  addressForm.validationStatus = addr.validationStatus;
  addressForm.validationDetail = addr.validationDetail;
  addressForm.extraData = addr.extraData;
  showAddressModal.value = true;
}

async function submitAddress() {
  if (!selectedCustomer.value) return;
  try {
    const payload = {
      customerProfileId: selectedCustomer.value.id,
      label: addressForm.label,
      recipientName: addressForm.recipientName,
      phone: addressForm.phone,
      country: addressForm.country,
      province: addressForm.province,
      city: addressForm.city,
      district: addressForm.district,
      addressLine1: addressForm.addressLine1,
      addressLine2: addressForm.addressLine2,
      postalCode: addressForm.postalCode,
      isDefault: addressForm.isDefault,
      isTest: addressForm.isTest,
      validationStatus: addressForm.validationStatus,
      validationDetail: addressForm.validationDetail,
      extraData: addressForm.extraData,
    };

    if (editingAddressId.value) {
      await updateAddress({ id: editingAddressId.value, ...payload });
      message.success("地址已更新");
    } else {
      await createAddress(payload);
      message.success("地址已添加");
    }

    showAddressModal.value = false;
    editingAddressId.value = null;
    resetAddressForm();

    // Reload details
    selectedCustomer.value = await getCustomerProfile(selectedCustomer.value.id);
    await loadCustomers();
  } catch (e: any) {
    message.error(e?.message ?? String(e));
  }
}

async function removeAddress(id: number) {
  if (!selectedCustomer.value) return;
  try {
    await deleteAddress(id);
    message.success("地址已删除");

    // Reload details
    selectedCustomer.value = await getCustomerProfile(selectedCustomer.value.id);
    await loadCustomers();
  } catch (e: any) {
    message.error(e?.message ?? String(e));
  }
}

async function setAddressAsDefault(addr: dto.CustomerAddressDTO) {
  if (!selectedCustomer.value) return;
  try {
    // Call updateAddress on this specific record setting isDefault to true.
    // The backend's repo implementation will automatically clear other defaults for this profile.
    await updateAddress({
      id: addr.id,
      customerProfileId: addr.customerProfileId,
      label: addr.label,
      recipientName: addr.recipientName,
      phone: addr.phone,
      country: addr.country,
      province: addr.province,
      city: addr.city,
      district: addr.district,
      addressLine1: addr.addressLine1,
      addressLine2: addr.addressLine2,
      postalCode: addr.postalCode,
      isDefault: true,
      isTest: addr.isTest,
      validationStatus: addr.validationStatus,
      validationDetail: addr.validationDetail,
      extraData: addr.extraData,
    });
    message.success("默认地址设置成功");

    // Reload details
    selectedCustomer.value = await getCustomerProfile(selectedCustomer.value.id);
    await loadCustomers();
  } catch (e: any) {
    message.error(e?.message ?? String(e));
  }
}

// ── Create Customer Profile Modal State ──
const showCreateModal = ref(false);
const createForm = reactive({
  displayName: "",
  profileType: "member",
  extraData: "",
});

function openCreateModal() {
  createForm.displayName = "";
  createForm.profileType = "member";
  createForm.extraData = "";
  showCreateModal.value = true;
}

async function submitCreateProfile() {
  try {
    const created = await createCustomerProfile({
      displayName: createForm.displayName,
      profileType: createForm.profileType,
      extraData: createForm.extraData,
    });
    message.success("客户档案已创建");
    showCreateModal.value = false;
    await loadCustomers();
    // Automatically open drawer for the newly created profile
    openDrawer(created);
  } catch (e: any) {
    message.error(e?.message ?? String(e));
  }
}

// ── Suggestions State & Actions ──
async function loadSuggestions() {
  try {
    suggestions.value = (await getMergeSuggestions()) || [];
  } catch (e) {
    console.error("load merge suggestions failed", e);
  }
}

async function dismissSuggestion(id: number) {
  try {
    await dismissMergeSuggestion(id);
    message.info(t("suggestedMerges.successDismiss"));
    await loadSuggestions();
  } catch (e: any) {
    message.error(e?.message ?? String(e));
  }
}

async function executeSuggestionMerge(s: dto.MergeSuggestionDTO) {
  dialog.warning({
    title: t("merge.confirmTitle"),
    content: t("merge.confirmDesc"),
    positiveText: t("merge.execute"),
    negativeText: t("common.cancel"),
    onPositiveClick: async () => {
      try {
        const result = await mergeProfiles({
          sourceProfileId: s.sourceProfileId,
          targetProfileId: s.targetProfileId,
        });
        message.success(
          `${t("suggestedMerges.successMerge")} (Migrated ${result.migratedIdentityCount} identities, ${result.migratedAddressCount} addresses)`
        );
        await loadSuggestions();
        await loadCustomers();
      } catch (e: any) {
        message.error(e?.message ?? String(e));
      }
    },
  });
}

// ── Manual Merge State & Actions ──
const manualMergeData = reactive({
  sourceProfileId: null as number | null,
  targetProfileId: null as number | null,
});

const customerOptions = computed(() => {
  return (customers.value || []).map((c) => ({
    label: `${c.displayName} (ID: ${c.id})`,
    value: c.id,
  }));
});

const customerOptionsFiltered = computed(() => {
  // Exclude target profile ID from source options
  return customerOptions.value.filter((o) => o.value !== manualMergeData.targetProfileId);
});

async function confirmManualMerge() {
  if (!manualMergeData.sourceProfileId || !manualMergeData.targetProfileId) return;

  dialog.warning({
    title: t("merge.confirmTitle"),
    content: t("merge.confirmDesc"),
    positiveText: t("merge.execute"),
    negativeText: t("common.cancel"),
    onPositiveClick: async () => {
      try {
        const result = await mergeProfiles({
          sourceProfileId: manualMergeData.sourceProfileId!,
          targetProfileId: manualMergeData.targetProfileId!,
        });
        message.success(
          `合并成功！已迁移 ${result.migratedIdentityCount} 个平台身份, ${result.migratedAddressCount} 个收货地址。`
        );
        manualMergeData.sourceProfileId = null;
        manualMergeData.targetProfileId = null;
        await loadCustomers();
        await loadSuggestions();
      } catch (e: any) {
        message.error(e?.message ?? String(e));
      }
    },
  });
}

// ── Lifecycle ──
onMounted(() => {
  loadCustomers();
  loadUniquePlatforms();
  loadSuggestions();
});
</script>

<style scoped>
.customer-crm-page {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 8px;
}

.stat-card {
  border-radius: 12px;
  background: linear-gradient(135deg, var(--surface-strong) 0%, var(--surface-muted) 100%);
  border: 1px solid rgba(255, 255, 255, 0.08);
}

.stat-label {
  font-size: 0.8rem;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.stat-val {
  font-size: 1.8rem;
  font-weight: 700;
  margin-top: 4px;
}

.crm-toolbar {
  padding: 14px 18px;
  border-radius: 12px;
  background: var(--surface-strong);
  border: 1px solid rgba(255, 255, 255, 0.08);
  display: flex;
  align-items: center;
}

.suggestion-header {
  padding: 16px 20px;
  border-radius: 12px;
  background: var(--surface-strong);
  border: 1px solid rgba(255, 255, 255, 0.08);
}

.suggestions-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.suggestion-item {
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  padding: 16px;
}

.profiles-comparison {
  display: flex;
  align-items: center;
  gap: 20px;
  margin-top: 12px;
}

.profile-card {
  flex: 1;
  padding: 14px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid rgba(255, 255, 255, 0.06);
}

.primary-border {
  border-left: 4px solid var(--accent);
}

.secondary-border {
  border-left: 4px solid var(--danger, #ef4444);
}

.profile-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.profile-badge {
  font-size: 0.7rem;
  padding: 2px 6px;
  border-radius: 4px;
  font-weight: 600;
}

.badge-keep {
  background: rgba(37, 99, 235, 0.15);
  color: var(--accent);
}

.badge-delete {
  background: rgba(239, 68, 68, 0.15);
  color: var(--danger, #ef4444);
}

.profile-name {
  font-size: 0.95rem;
}

.profile-id {
  font-size: 0.75rem;
  color: var(--text-muted);
  margin-left: auto;
}

.comparison-arrow {
  font-size: 1.8rem;
  color: var(--text-muted);
}

.detail-row {
  display: flex;
  font-size: 0.8rem;
  align-items: center;
  gap: 8px;
}

.detail-label {
  color: var(--text-muted);
}

.drawer-inner {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.identity-item, .address-item {
  padding: 14px;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.06);
  background: rgba(255, 255, 255, 0.01);
  margin-bottom: 12px;
}

.text-primary { color: var(--accent); }
.text-success { color: var(--success); }
.text-error { color: var(--danger, #ef4444); }
.text-xs { font-size: 0.75rem; }
.text-sm { font-size: 0.875rem; }
.text-muted { color: var(--text-muted); }
.flex { display: flex; }
.justify-between { justify-content: space-between; }
.align-center { align-items: center; }
.mt-0-5 { margin-top: 2px; }
.mt-1 { margin-top: 4px; }
.mt-2 { margin-top: 8px; }
.mt-3 { margin-top: 12px; }
.mb-2 { margin-bottom: 8px; }
.mb-3 { margin-bottom: 12px; }
.mb-4 { margin-bottom: 16px; }
.mb-6 { margin-bottom: 24px; }
</style>
