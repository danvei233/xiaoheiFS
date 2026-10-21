import { describe, expect, it } from "vitest";
import { setActivePinia, createPinia } from "pinia";
import { useRevenueAnalyticsStore } from "@/stores/revenueAnalytics";

describe("revenueAnalytics store", () => {
  it("updates pagination query", () => {
    setActivePinia(createPinia());
    const store = useRevenueAnalyticsStore();
    store.setQuery({ page: 2, page_size: 50 });
    expect(store.query.page).toBe(2);
    expect(store.query.page_size).toBe(50);
  });
});
