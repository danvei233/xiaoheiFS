import { defineStore } from 'pinia'
import { listVps, getVpsDetail, getVpsPanelUrl, refreshVps } from '@/services/user'
import { useAuthStore } from '@/stores/auth'

export const useVpsStore = defineStore('vps', {
  state: () => ({
    items: [],
    loading: false,
    current: null,
    panelUrl: '',
    detailReqId: 0
  }),
  actions: {
    async fetchVps() {
      const auth = useAuthStore()
      const tokenAtStart = auth.token
      this.loading = true
      try {
        const res = await listVps()
        // 会话切换（登出/模拟登录/换账号）后丢弃旧账号响应，避免跨账号残留
        // 写回 store（web_data_races F7）。
        if (auth.token === tokenAtStart) {
          this.items = res.data?.items || []
        }
      } finally {
        this.loading = false
      }
    },
    async fetchDetail(id) {
      const auth = useAuthStore()
      const tokenAtStart = auth.token
      const targetId = String(id ?? '')
      // 单槽 current：请求发出前必须清空，否则从 /console/vps/1 切到
      // /console/vps/2 时，在 VPS2 响应落地前模板会渲染 VPS1 的明文
      // access_info（系统密码/面板密码）在 VPS2 的 URL 下（web_state_isolation F1）。
      if (this.current && String(this.current.id ?? this.current.ID ?? '') !== targetId) {
        this.current = null
      }
      const requestId = ++this.detailReqId
      const res = await getVpsDetail(id)
      // 会话切换（登出/模拟登录/换账号）后丢弃旧账号响应，避免跨账号残留写回。
      if (auth.token !== tokenAtStart) return
      // 请求代次：慢响应下旧 fetchDetail 后到时不得覆盖已加载的新详情。
      if (requestId !== this.detailReqId) return
      const data = res.data || null
      // 只接受与请求 id 匹配的响应：后端 404/错位返回不得把其他 VPS 的凭据
      // 覆盖进单槽 current。
      if (data && String(data.id ?? data.ID ?? '') !== targetId) return
      this.current = data
    },
    async fetchPanel(id) {
      const res = await getVpsPanelUrl(id)
      this.panelUrl = String(res.data?.url || '')
    },
    async refresh(id) {
      await refreshVps(id)
      await this.fetchDetail(id)
    },
    reset() {
      this.items = []
      this.loading = false
      this.current = null
      this.panelUrl = ''
      this.detailReqId = 0
    }
  }
})
