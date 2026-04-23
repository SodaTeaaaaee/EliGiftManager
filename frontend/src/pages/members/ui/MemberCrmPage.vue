<script setup lang="ts">
const members = [
  { name: 'Mika', level: 'VIP', city: 'Tokyo', address: '已完整', orders: 12 },
  { name: 'Haruka', level: 'Gold', city: 'Osaka', address: '待校验', orders: 8 },
  { name: 'Nana', level: 'Silver', city: 'Nagoya', address: '缺少门牌', orders: 5 },
  { name: 'Yui', level: 'New', city: 'Fukuoka', address: '已完整', orders: 2 },
]
</script>

<template>
  <section class="space-y-6">
    <header>
      <p class="text-sm font-medium uppercase tracking-[0.24em] text-orange-300">Member CRM</p>
      <h1 class="mt-2 text-3xl font-semibold tracking-tight text-neutral-50">会员地址库</h1>
      <p class="mt-2 text-neutral-400">统一管理粉丝档案、地址完整度和历史派发记录。</p>
    </header>

    <section class="rounded-3xl border border-white/10 bg-neutral-900/70 p-5 shadow-xl shadow-black/20">
      <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <input
          class="w-full rounded-2xl border border-white/10 bg-neutral-950/60 px-4 py-3 text-sm text-neutral-100 outline-none placeholder:text-neutral-500 md:max-w-md"
          placeholder="搜索昵称、手机或收件城市"
        />
        <button class="rounded-2xl bg-white/[0.06] px-4 py-3 text-sm text-neutral-100 hover:bg-white/[0.1]">批量导入</button>
      </div>

      <div class="mt-5 overflow-hidden rounded-2xl border border-white/10">
        <table class="w-full text-left text-sm">
          <thead class="bg-white/[0.04] text-neutral-400">
            <tr>
              <th class="px-4 py-3 font-medium">粉丝</th>
              <th class="px-4 py-3 font-medium">等级</th>
              <th class="px-4 py-3 font-medium">城市</th>
              <th class="px-4 py-3 font-medium">地址状态</th>
              <th class="px-4 py-3 font-medium">派发次数</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-white/10 text-neutral-200">
            <tr v-for="member in members" :key="member.name" class="hover:bg-white/[0.03]">
              <td class="px-4 py-4 font-medium text-neutral-100">{{ member.name }}</td>
              <td class="px-4 py-4">{{ member.level }}</td>
              <td class="px-4 py-4">{{ member.city }}</td>
              <td class="px-4 py-4">{{ member.address }}</td>
              <td class="px-4 py-4">{{ member.orders }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </section>
</template>
