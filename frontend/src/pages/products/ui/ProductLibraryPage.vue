<script setup lang="ts">
const products = [
  { name: '限定徽章套装', stock: 186, tag: '热门' },
  { name: '签名明信片', stock: 42, tag: '低库存' },
  { name: '应援手幅', stock: 96, tag: '常规' },
  { name: '周年纪念盒', stock: 28, tag: '限量' },
  { name: '透明卡套', stock: 210, tag: '新品' },
  { name: '活动贴纸包', stock: 164, tag: '组合' },
]
</script>

<template>
  <section class="space-y-6">
    <header class="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
      <div>
        <p class="text-sm font-medium uppercase tracking-[0.24em] text-orange-300">Product Library</p>
        <h1 class="mt-2 text-3xl font-semibold tracking-tight text-neutral-50">礼物商品库</h1>
        <p class="mt-2 text-neutral-400">用卡片网格预览实物周边、库存和状态。</p>
      </div>
      <button class="rounded-2xl bg-orange-400 px-4 py-3 text-sm font-semibold text-neutral-950">新增商品</button>
    </header>

    <div class="grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
      <article
        v-for="product in products"
        :key="product.name"
        class="overflow-hidden rounded-3xl border border-white/10 bg-neutral-900/70 shadow-xl shadow-black/20"
      >
        <div class="flex h-44 items-center justify-center bg-gradient-to-br from-neutral-700 to-neutral-950">
          <div class="h-24 w-24 rounded-3xl border border-white/15 bg-white/[0.06] shadow-inner" />
        </div>
        <div class="p-5">
          <div class="flex items-start justify-between gap-3">
            <h2 class="font-semibold text-neutral-50">{{ product.name }}</h2>
            <span class="shrink-0 rounded-full bg-orange-300/10 px-3 py-1 text-xs text-orange-100">{{ product.tag }}</span>
          </div>
          <div class="mt-5 flex items-center justify-between rounded-2xl bg-white/[0.04] px-4 py-3">
            <span class="text-sm text-neutral-400">当前库存</span>
            <span class="text-lg font-semibold text-neutral-50">{{ product.stock }}</span>
          </div>
        </div>
      </article>
    </div>
  </section>
</template>
