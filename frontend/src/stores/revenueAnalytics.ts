import { defineStore } from "pinia";
import {
  getRevenueAnalyticsDetails,
  getRevenueAnalyticsOverview,
  getRevenueAnalyticsTop,
  getRevenueAnalyticsTrend
} from "@/services/admin";
import type {
  RevenueAnalyticsDetailRecord,
  RevenueAnalyticsOverviewResponse,
  RevenueAnalyticsQuery,
  RevenueAnalyticsTopItem,
  RevenueAnalyticsTrendPoint
} from "@/services/types";

const defaultQuery = (): RevenueAnalyticsQuery => {
  const now = new Date();
  const from = new Date(now.getTime() - 30 * 24 * 3600 * 1000);
  return {
    from_at: from.toISOString(),
    to_at: now.toISOString(),
    level: "overall",
    page: 1,
    page_size: 20,
    sort_field: "paid_at",
    sort_order: "desc"
  };
};

export const useRevenueAnalyticsStore = defineStore("revenueAnalytics", {
  state: () => ({
    loading: false,
    query: defaultQuery(),
    overview: {} as RevenueAnalyticsOverviewResponse,
    trend: [] as RevenueAnalyticsTrendPoint[],
    top: [] as RevenueAnalyticsTopItem[],
    details: [] as RevenueAnalyticsDetailRecord[],
    detailTotal: 0
  }),
  actions: {
    setQuery(partial: Partial<RevenueAnalyticsQuery>) {
      this.query = { ...this.query, ...partial };
    },
    async fetchOverview() {
      this.loading = true;
      try {
        const [overviewRes, trendRes, topRes] = await Promise.all([
          getRevenueAnalyticsOverview(this.query),
          getRevenueAnalyticsTrend(this.query),
          getRevenueAnalyticsTop(this.query)
        ]);
        this.overview = overviewRes.data || {};
        this.trend = trendRes.data?.items || [];
        this.top = topRes.data?.items || [];
      } finally {
        this.loading = false;
      }
    },
    async fetchDetails() {
      const res = await getRevenueAnalyticsDetails(this.query);
      this.details = res.data?.items || [];
      this.detailTotal = Number(res.data?.total || 0);
    },
    async fetchAll() {
      await this.fetchOverview();
      await this.fetchDetails();
    }
  }
});
