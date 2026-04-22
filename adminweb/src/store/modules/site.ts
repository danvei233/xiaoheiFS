import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import AppConfig from '@/config'
import { fetchAdminSettings, type SettingItemRecord } from '@/api/admin'

function updateFavicon(href: string) {
  if (typeof document === 'undefined' || !href) {
    return
  }

  let link = document.querySelector<HTMLLinkElement>('link[rel*="icon"]')
  if (!link) {
    link = document.createElement('link')
    link.rel = 'icon'
    document.head.appendChild(link)
  }

  link.href = href
}

export const useSiteStore = defineStore(
  'siteStore',
  () => {
    const siteName = ref('')
    const logoUrl = ref('')
    const faviconUrl = ref('')
    const adminPath = ref('admin')
    const maintenanceMode = ref(false)
    const maintenanceMessage = ref('')
    const settings = ref<Record<string, string>>({})

    const resolvedSiteName = computed(() => siteName.value || AppConfig.systemInfo.name)

    function resolveSettingValue(key: string, legacyKey?: string) {
      return settings.value[key] || (legacyKey ? settings.value[legacyKey] : '') || ''
    }

    function applyBranding() {
      if (faviconUrl.value) {
        updateFavicon(faviconUrl.value)
      }
    }

    function applySettings(items: SettingItemRecord[]) {
      const nextSettings: Record<string, string> = {}

      items.forEach((item) => {
        const key = String(item.key || '').trim()
        if (key) {
          nextSettings[key] = String(item.value || '')
        }
      })

      settings.value = nextSettings
      siteName.value = resolveSettingValue('site_name') || AppConfig.systemInfo.name
      logoUrl.value = resolveSettingValue('logo_url', 'site_logo')
      faviconUrl.value = resolveSettingValue('favicon_url', 'site_favicon')
      adminPath.value = resolveSettingValue('admin_path') || 'admin'

      const maintenanceRaw = resolveSettingValue('maintenance_mode', 'site_maintenance_mode')
      maintenanceMode.value = maintenanceRaw === 'true'
      maintenanceMessage.value =
        resolveSettingValue('maintenance_message', 'site_maintenance_message') ||
        '系统维护中，请稍后再试'

      applyBranding()
    }

    async function fetchSettings() {
      const payload = await fetchAdminSettings()
      applySettings(payload.items || [])
    }

    return {
      siteName,
      logoUrl,
      faviconUrl,
      adminPath,
      maintenanceMode,
      maintenanceMessage,
      settings,
      resolvedSiteName,
      applyBranding,
      applySettings,
      fetchSettings
    }
  },
  {
    persist: {
      key: 'site',
      storage: localStorage
    }
  }
)
