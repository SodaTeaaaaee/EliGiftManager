import { onMounted, ref } from 'vue'

import { Bootstrap } from '../../../../wailsjs/go/main/App'
import type { BootstrapPayload } from '@/shared/types/app'

const fallbackBootstrapPayload: BootstrapPayload = {
  name: 'EliGiftManager',
  version: '0.1.0',
  module: 'github.com/SodaTeaaaaee/EliGiftManager',
  description: 'A desktop gift planning workspace built with Go, Wails, Vue, and Deno.',
  runtime: 'go1.26.2',
  frontend: 'Vue 3.5.33 + Vite 8 + Deno 2',
  highlights: [
    'Go backend keeps reusable logic inside internal packages.',
    'Vue 3 SFCs are compiled by Vite while Deno owns dependency installation and task execution.',
    'Wails remains responsible for desktop bindings and native packaging.',
  ],
}

export function useBootstrapState() {
  const payload = ref<BootstrapPayload>(fallbackBootstrapPayload)
  const status = ref('Loading desktop bridge...')

  onMounted(async () => {
    const wailsWindow = window as Window & {
      go?: unknown
    }

    if (!wailsWindow.go) {
      status.value = 'Running in browser preview mode'
      return
    }

    try {
      payload.value = await Bootstrap()
      status.value = 'Connected to Wails backend'
    } catch {
      status.value = 'Running in browser preview mode'
    }
  })

  return {
    payload,
    status,
  }
}
