import { defineStore } from 'pinia'
import { listVps, getVpsDetail, getVpsPanelUrl, refreshVps } from '@/services/user'

export const useVpsStore = defineStore('vps', {
  state: () => ({
    items: [],
    loading: false,
    current: null,
    panelUrl: ''
  }),
  actions: {
    async fetchVps() {
      this.loading = true
      try {
        const res = await listVps()
        this.items = res.data?.items || []
      } finally {
        this.loading = false
      }
    },
    async fetchDetail(id) {
      const res = await getVpsDetail(id)
      this.current = res.data || null
    },
    async fetchPanel(id) {
      const res = await getVpsPanelUrl(id)
      this.panelUrl = String(res.data?.url || '')
    },
    async refresh(id) {
      await refreshVps(id)
      await this.fetchDetail(id)
      await this.fetchVps()
    }
  }
})
