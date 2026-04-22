<script setup lang="ts">
import { dashboardCommands } from '@/features/dashboard/model/commands'
import CommandList from '@/features/dashboard/ui/CommandList.vue'
import StatusPanel from '@/features/dashboard/ui/StatusPanel.vue'
import { useBootstrapState } from '@/shared/lib/wails/bootstrap'
import InfoCard from '@/shared/ui/InfoCard.vue'

const { payload, status } = useBootstrapState()
</script>

<template>
  <div class="app-shell">
    <div class="app-shell__content">
      <section class="hero">
        <div class="hero__panel">
          <div class="hero__eyebrow">
            Desktop Gift Workspace
          </div>
          <h1 class="hero__title">
            {{ payload.name }}
          </h1>
          <p class="hero__subtitle">
            {{ payload.description }}
          </p>
          <ul class="hero__highlights">
            <li
              v-for="highlight in payload.highlights"
              :key="highlight"
            >
              {{ highlight }}
            </li>
          </ul>
        </div>

        <StatusPanel
          :payload="payload"
          :status="status"
        />
      </section>

      <section
        class="dashboard-grid"
        aria-label="Project overview"
      >
        <InfoCard
          title="Go Runtime"
          :value="payload.runtime"
          detail="Backend entrypoint stays thin; reusable logic belongs in internal packages."
        />
        <InfoCard
          title="Frontend Runtime"
          :value="payload.frontend"
          detail="Vue 3 SFCs are compiled by Vite under Deno, without requiring a local Node.js installation."
        />
        <InfoCard
          title="Wails Shell"
          value="v2.12.0"
          detail="Wails binds the Go backend, manages the desktop window lifecycle, and packages the final binary."
        />
      </section>

      <CommandList :commands="dashboardCommands" />
    </div>
  </div>
</template>
