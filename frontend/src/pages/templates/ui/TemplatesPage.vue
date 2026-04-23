<script setup lang="ts">
const templateCards = [
  {
    title: '会员导入模板',
    type: 'import_member',
    stage: '导入',
    status: '已启用',
    summary: '将平台会员 CSV 映射到标准会员、昵称与地址模型。',
    fields: ['平台 UID', '昵称', '手机号 / 地址'],
  },
  {
    title: '商品导入模板',
    type: 'import_product',
    stage: '导入',
    status: '待补字段',
    summary: '用于工厂商品主数据入库，保留 SKU、名称和扩展元数据。',
    fields: ['工厂 SKU', '商品名', '图片路径'],
  },
  {
    title: '发货记录导入模板',
    type: 'import_dispatch_record',
    stage: '导入',
    status: '已校验',
    summary: '承接批次结果和发货明细，为地址预校验和导出做准备。',
    fields: ['批次名', '会员 ID', '商品 ID'],
  },
  {
    title: '订单导出模板',
    type: 'export_order',
    stage: '导出',
    status: '草稿',
    summary: '把派发结果整理成面向仓库或快递渠道的导出结构。',
    fields: ['收件信息', '商品清单', '导出状态'],
  },
]

const templateRules = [
  '模板类型直接对应后端 TemplateConfig.type，避免前后端枚举再次翻译。',
  '导入模板负责字段映射，设置页只保存导入默认规则，不再混放。',
  '模板页面保留版本化空间，后续接数据库时可以直接映射到 TemplateConfig 列表。',
]
</script>

<template>
  <section class="space-y-6">
    <header class="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
      <div>
        <p class="text-sm font-medium uppercase tracking-[0.24em] text-orange-300">Templates</p>
        <h1 class="mt-2 text-3xl font-semibold tracking-tight text-neutral-50">模板</h1>
        <p class="mt-2 max-w-3xl text-neutral-400">
          这里集中管理导入导出模板本身。页面内容已经按后端真实模板类型拆开，后续接数据库列表时可以直接落到同一结构。
        </p>
      </div>
      <div class="rounded-3xl border border-orange-300/15 bg-orange-300/10 px-5 py-4 text-sm text-orange-100">
        当前模板类型：4
      </div>
    </header>

    <div class="grid gap-5 xl:grid-cols-[1.45fr_1fr]">
      <section class="grid gap-5 md:grid-cols-2">
        <article
          v-for="card in templateCards"
          :key="card.type"
          class="rounded-3xl border border-white/10 bg-neutral-900/70 p-6 shadow-xl shadow-black/20"
        >
          <div class="flex items-start justify-between gap-4">
            <div>
              <span class="rounded-full bg-white/[0.06] px-3 py-1 text-xs uppercase tracking-[0.2em] text-neutral-300">
                {{ card.stage }}
              </span>
              <h2 class="mt-4 text-xl font-semibold text-neutral-50">{{ card.title }}</h2>
            </div>
            <span class="rounded-full bg-orange-300/10 px-3 py-1 text-xs text-orange-100">{{ card.status }}</span>
          </div>

          <p class="mt-4 text-sm leading-7 text-neutral-400">{{ card.summary }}</p>

          <div class="mt-5 rounded-2xl border border-white/10 bg-white/[0.04] px-4 py-4">
            <p class="text-xs uppercase tracking-[0.18em] text-neutral-500">Template Type</p>
            <p class="mt-2 font-mono text-sm text-neutral-100">{{ card.type }}</p>
          </div>

          <div class="mt-5 space-y-3">
            <div
              v-for="field in card.fields"
              :key="field"
              class="flex items-center justify-between rounded-2xl border border-white/10 bg-white/[0.04] px-4 py-3"
            >
              <span class="text-sm text-neutral-200">{{ field }}</span>
              <span class="rounded-full bg-neutral-950/60 px-3 py-1 text-xs text-neutral-400">映射项</span>
            </div>
          </div>
        </article>
      </section>

      <aside class="rounded-3xl border border-white/10 bg-neutral-900/70 p-6 shadow-xl shadow-black/20">
        <h2 class="text-xl font-semibold text-neutral-50">模板约束</h2>
        <p class="mt-3 text-sm leading-7 text-neutral-400">
          模板页现在先展示结构约束和后端类型映射，等模板列表接口接进来后，这个布局可以直接替换成真实数据。
        </p>

        <div class="mt-6 space-y-3">
          <div
            v-for="rule in templateRules"
            :key="rule"
            class="rounded-2xl border border-white/10 bg-white/[0.04] px-4 py-4 text-sm leading-7 text-neutral-200"
          >
            {{ rule }}
          </div>
        </div>
      </aside>
    </div>
  </section>
</template>
