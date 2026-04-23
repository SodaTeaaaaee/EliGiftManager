<script setup lang="ts">
const settingGroups = [
  {
    title: '导入默认项',
    tag: 'Import',
    description: '控制成员、商品与派发记录导入时的标准化策略。',
    items: [
      { label: 'CSV 编码探测', value: 'UTF-8 / Shift-JIS 自动识别' },
      { label: '重复会员处理', value: '按平台 UID 合并主档案' },
      { label: '默认平台要求', value: '缺失平台标识时阻止导入' },
    ],
  },
  {
    title: '地址校验',
    tag: 'Validate',
    description: '决定批量派发前的地址完整度和异常拦截规则。',
    items: [
      { label: '手机号规则', value: '缺失或长度异常时标记待校验' },
      { label: '门牌缺失策略', value: '允许保存，但阻止进入导出批次' },
      { label: '导出前预校验', value: '始终执行 ValidateBatch' },
    ],
  },
  {
    title: '导出偏好',
    tag: 'Export',
    description: '统一发货单、日报和快递输出时的默认格式。',
    items: [
      { label: '发货单命名', value: '批次名 + 日期 + 序号' },
      { label: '日报时间基准', value: 'Asia/Tokyo' },
      { label: '快递模板回退', value: '未命中时使用标准字段顺序' },
    ],
  },
  {
    title: '桌面工作区',
    tag: 'Workspace',
    description: '保留桌面端特有的数据库和工作区默认行为。',
    items: [
      { label: '数据库位置', value: '用户配置目录 / data / eligiftmanager.db' },
      { label: '启动首页', value: '工作台' },
      { label: '批次结果缓存', value: '会话级保留，重启后刷新' },
    ],
  },
]
</script>

<template>
  <section class="space-y-6">
    <header>
      <p class="text-sm font-medium uppercase tracking-[0.24em] text-orange-300">Settings</p>
      <h1 class="mt-2 text-3xl font-semibold tracking-tight text-neutral-50">设置</h1>
      <p class="mt-2 max-w-3xl text-neutral-400">
        这里放全局运行规则，不处理具体模板内容。模板本身已经拆到单独页面，设置页只负责默认策略和工作区行为。
      </p>
    </header>

    <div class="grid gap-5 xl:grid-cols-2">
      <article
        v-for="group in settingGroups"
        :key="group.title"
        class="rounded-3xl border border-white/10 bg-neutral-900/70 p-6 shadow-xl shadow-black/20"
      >
        <div class="flex items-start justify-between gap-4">
          <div>
            <span class="rounded-full bg-orange-300/10 px-3 py-1 text-xs uppercase tracking-[0.2em] text-orange-100">
              {{ group.tag }}
            </span>
            <h2 class="mt-4 text-xl font-semibold text-neutral-50">{{ group.title }}</h2>
          </div>
          <div class="flex h-14 w-14 items-center justify-center rounded-2xl bg-white/[0.05] text-xl font-semibold text-neutral-100">
            {{ group.title.slice(0, 1) }}
          </div>
        </div>

        <p class="mt-4 leading-7 text-neutral-400">{{ group.description }}</p>

        <div class="mt-6 space-y-3">
          <div
            v-for="item in group.items"
            :key="item.label"
            class="rounded-2xl border border-white/10 bg-white/[0.04] px-4 py-4"
          >
            <p class="text-sm text-neutral-400">{{ item.label }}</p>
            <p class="mt-2 text-sm font-medium leading-6 text-neutral-100">{{ item.value }}</p>
          </div>
        </div>
      </article>
    </div>
  </section>
</template>
