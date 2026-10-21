import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchInstallStatus } from '@/api/install'

export const useInstallStore = defineStore('installStore', () => {
  const loaded = ref(false)
  const installed = ref(true)

  async function fetchStatus(force: boolean = false) {
    if (loaded.value && !force) {
      return installed.value
    }

    try {
      const payload = await fetchInstallStatus()
      installed.value = Boolean(payload.installed)
    } catch {
      installed.value = true
    } finally {
      loaded.value = true
    }

    return installed.value
  }

  function setInstalled(value: boolean) {
    installed.value = value
    loaded.value = true
  }

  return {
    loaded,
    installed,
    fetchStatus,
    setInstalled
  }
})
