import { defineStore } from "pinia";
import { listOrders, getOrderDetail, refreshOrder } from "@/services/user";

export const useOrdersStore = defineStore("orders", {
  state: () => ({
    items: [],
    loading: false,
    total: 0,
    currentOrder: null,
    orderItems: [],
    orderPayments: []
  }),
  actions: {
    async fetchOrders(params) {
      this.loading = true;
      try {
        const res = await listOrders(params);
        const payload = res.data || {};
        this.items = payload.items || [];
        this.total = payload.total || this.items.length;
      } finally {
        this.loading = false;
      }
    },
    async fetchOrderDetail(id) {
      const res = await getOrderDetail(id);
      this.currentOrder = res.data?.order || null;
      this.orderItems = res.data?.items || [];
      this.orderPayments = res.data?.payments || [];
    },
    async refreshOrder(id) {
      await refreshOrder(id);
      await this.fetchOrderDetail(id);
    }
  }
});
