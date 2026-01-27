import { defineStore } from "pinia";
import { getDashboard, listOrders, listVps, getWallet, getRealNameStatus } from "@/services/user";
import { normalizeWallet } from "@/utils/wallet";

const toDateKey = (d) => {
  const dt = new Date(d);
  if (Number.isNaN(dt.getTime())) return "";
  return dt.toISOString().slice(0, 10);
};

const withinDays = (dateStr, days) => {
  const dt = new Date(dateStr);
  if (Number.isNaN(dt.getTime())) return false;
  const diff = dt.getTime() - Date.now();
  return diff <= days * 24 * 3600 * 1000 && diff >= 0;
};

export const useDashboardStore = defineStore("dashboard", {
  state: () => ({
    loading: false,
    metrics: {},
    charts: {},
    wallet: null,
    realname: null
  }),
  actions: {
    async fetchUserDashboard() {
      this.loading = true;
      try {
        const [dashRes, ordersRes, vpsRes, walletRes, realnameRes] = await Promise.all([
          getDashboard(),
          listOrders({ limit: 200, offset: 0 }),
          listVps(),
          getWallet(),
          getRealNameStatus()
        ]);

        const dash = dashRes.data || {};
        const orders = ordersRes.data?.items || [];
        const vpsList = vpsRes.data?.items || [];
        const wallet = normalizeWallet(walletRes.data) || {};
        const realname = realnameRes.data || {};

        const thirtyDaysAgo = Date.now() - 30 * 24 * 3600 * 1000;
        const recentOrders = orders.filter((order) => {
          const createdAt = new Date(order.created_at ?? order.CreatedAt).getTime();
          return !Number.isNaN(createdAt) && createdAt >= thirtyDaysAgo;
        });
        const spend30 = recentOrders.reduce(
          (sum, order) => sum + Number(order.total_amount ?? order.TotalAmount ?? 0),
          0
        );

        const realnameStatus = realname.verified
          ? "verified"
          : realname.verification?.status || (realname.enabled ? "" : "disabled");

        this.metrics = {
          vps_total: dash.vps || vpsList.length,
          expiring:
            dash.expiring ||
            vpsList.filter((v) => withinDays(v.expire_at ?? v.ExpireAt, 7)).length,
          orders_total: dash.orders || orders.length,
          pending_orders: dash.pending_review || orders.filter((o) => (o.status ?? o.Status) === "pending_review").length,
          spend_30d: dash.spend_30d || spend30,
          balance: wallet.balance ?? 0,
          currency: wallet.currency || "CNY",
          realname_status: realnameStatus
        };
        this.wallet = wallet;
        this.realname = realname;

        const trendMap = new Map();
        recentOrders.forEach((order) => {
          const key = toDateKey(order.created_at ?? order.CreatedAt);
          if (!key) return;
          trendMap.set(key, (trendMap.get(key) || 0) + Number(order.total_amount ?? order.TotalAmount ?? 0));
        });
        const labels = Array.from(trendMap.keys()).sort();
        const values = labels.map((key) => trendMap.get(key));

        const statusMap = new Map();
        orders.forEach((item) => {
          const key = item.status ?? item.Status ?? "unknown";
          statusMap.set(key, (statusMap.get(key) || 0) + 1);
        });

        this.charts = {
          spendTrend: { labels, values },
          orderStatus: Array.from(statusMap.entries()).map(([name, value]) => ({ name, value })),
          expiringList: vpsList.filter((v) => v.expire_at ?? v.ExpireAt).slice(0, 5)
        };
      } finally {
        this.loading = false;
      }
    }
  }
});
