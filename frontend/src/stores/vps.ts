import { defineStore } from "pinia";
import { listVps, getVpsDetail, refreshVps } from "@/services/user";

export const useVpsStore = defineStore("vps", {
  state: () => ({
    items: [],
    loading: false,
    current: null,
    panelUrl: ""
  }),
  actions: {
    async fetchVps() {
      this.loading = true;
      try {
        const res = await listVps();
        this.items = res.data?.items || [];
      } finally {
        this.loading = false;
      }
    },
    async fetchDetail(id) {
      const res = await getVpsDetail(id);
      this.current = res.data || null;
    },
    async fetchPanel(id) {
      const base = import.meta.env.VITE_API_BASE || "";
      this.panelUrl = `${base}/api/v1/vps/${id}/panel`;
    },
    async refresh(id) {
      await refreshVps(id);
      await this.fetchDetail(id);
      await this.fetchVps();
    }
  }
});
