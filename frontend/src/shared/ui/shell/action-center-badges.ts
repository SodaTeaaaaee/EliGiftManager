import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getHomeBuckets } from '@/shared/api/bridge'

export const useActionCenterBadgesStore = defineStore('shell-action-center-badges', () => {
  const countsByNavKey = ref<Record<string, number>>({})
  const loaded = ref(false)

  async function refresh(): Promise<void> {
    try {
      const buckets = await getHomeBuckets()
      countsByNavKey.value = {
        home:
          buckets.BlockedResults +
          buckets.AlignmentConflict +
          buckets.IdentityUnattached +
          buckets.WritebackFailed,
        inbox: buckets.Unassigned,
      }
      loaded.value = true
    } catch {
      // Defensive
    }
  }

  return { countsByNavKey, loaded, refresh }
})
