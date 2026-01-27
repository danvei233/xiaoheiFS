import { defineStore } from "pinia";
import {
  getAdminDashboardOverview,
  getAdminDashboardRevenue,
  getAdminDashboardVpsStatus,
  listAdminOrders,
  listAdminVps
} from "@/services/admin";

export const useAdminStore = defineStore("admin", {
  state: () => ({
    overview: {
      total_revenue: 0,
      today_revenue: 0,
      pending_orders: 0,
      provisioning: 0,
      failed: 0,
      vps_total: 0,
      expiring: 0
    },
    revenue: [] as any[],
    vpsStatus: [] as any[],
    loading: false
  }),

  actions: {
    async fetchOverview() {
      this.loading = true;
      try {
        const res = await getAdminDashboardOverview();
        this.overview = res.data || this.overview;
      } finally {
        this.loading = false;
      }
    },

    async fetchRevenue(params?: any) {
      try {
        const res = await getAdminDashboardRevenue(params);
        this.revenue = res.data?.points || [];
      } catch (error) {
        console.error("Failed to fetch revenue data:", error);
      }
    },

    async fetchVpsStatus() {
      try {
        const res = await getAdminDashboardVpsStatus();
        this.vpsStatus = res.data?.points || [];
      } catch (error) {
        console.error("Failed to fetch VPS status:", error);
      }
    },

    async fetchDashboardData(params?: any) {
      await Promise.all([
        this.fetchOverview(),
        this.fetchRevenue(params),
        this.fetchVpsStatus()
      ]);
    }
  }
});
