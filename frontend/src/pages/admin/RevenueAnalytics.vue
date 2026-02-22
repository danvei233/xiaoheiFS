<template>
  <div class="page admin-page">
    <div class="page-header">
      <div>
        <div class="page-title">收入统计</div>
        <div class="subtle">下拉联动筛选（类型 -> 地区 -> 线路 -> 套餐）</div>
      </div>
    </div>

    <a-alert
      v-if="!canView"
      type="warning"
      show-icon
      message="当前账号缺少收入统计权限"
      style="margin-bottom: 12px"
    />

    <template v-else>
      <a-card :bordered="false">
        <a-space wrap>
          <a-select
            v-model:value="query.goods_type_id"
            style="width: 180px"
            placeholder="选择类型"
            :options="goodsTypeOptions"
            show-search
            option-filter-prop="label"
            allow-clear
            @change="onGoodsTypeChange"
          />

          <a-select
            v-model:value="query.region_id"
            style="width: 180px"
            placeholder="选择地区"
            :options="regionOptions"
            :disabled="!query.goods_type_id"
            show-search
            option-filter-prop="label"
            allow-clear
            @change="onRegionChange"
          />

          <a-select
            v-model:value="query.line_id"
            style="width: 180px"
            placeholder="选择线路"
            :options="lineOptions"
            :disabled="!query.region_id"
            show-search
            option-filter-prop="label"
            allow-clear
            @change="onLineChange"
          />

          <a-select
            v-model:value="query.package_id"
            style="width: 220px"
            placeholder="选择套餐"
            :options="packageOptions"
            :disabled="!query.line_id"
            show-search
            option-filter-prop="label"
            allow-clear
            @change="onPackageChange"
          />

          <a-range-picker v-model:value="rangeValue" @change="onRangeChange" />
          <a-button type="primary" :loading="store.loading" @click="reloadAll">查询</a-button>
        </a-space>
      </a-card>

      <a-row :gutter="16" style="margin-top: 16px">
        <a-col :xs="24" :md="8">
          <a-card :bordered="false" title="总收入">
            <div class="kpi">¥{{ centsToYuan(overview.summary?.total_revenue_cents) }}</div>
            <div class="subtle">订单数：{{ overview.summary?.order_count || 0 }}</div>
          </a-card>
        </a-col>
        <a-col :xs="24" :md="8">
          <a-card :bordered="false" title="同比">
            <div class="kpi">{{ formatRatio(overview.summary?.yoy_ratio, overview.summary?.yoy_comparable) }}</div>
          </a-card>
        </a-col>
        <a-col :xs="24" :md="8">
          <a-card :bordered="false" title="环比">
            <div class="kpi">{{ formatRatio(overview.summary?.mom_ratio, overview.summary?.mom_comparable) }}</div>
          </a-card>
        </a-col>
      </a-row>

      <a-row :gutter="16" style="margin-top: 16px">
        <a-col :xs="24" :lg="12">
          <a-card :bordered="false" title="收入占比（可点下钻）">
            <PieChart :data="shareChartData" />
          </a-card>
        </a-col>
        <a-col :xs="24" :lg="12">
          <a-card :bordered="false" title="收入趋势">
            <LineChart :data="trendChartData" />
          </a-card>
        </a-col>
      </a-row>

      <a-row :gutter="16" style="margin-top: 16px">
        <a-col :xs="24" :lg="8">
          <a-card :bordered="false" title="Top5（点击自动进入下一层）">
            <a-list :data-source="topList" size="small">
              <template #renderItem="{ item }">
                <a-list-item class="drill-item" @click="onDrillDown(item)">
                  <span>#{{ item.rank }} {{ item.dimension_name }}</span>
                  <strong>¥{{ centsToYuan(item.revenue_cents) }}</strong>
                </a-list-item>
              </template>
            </a-list>
          </a-card>
        </a-col>
        <a-col :xs="24" :lg="16">
          <a-card :bordered="false" title="明细表">
            <a-table
              :data-source="details"
              :columns="columns"
              :pagination="pagination"
              row-key="payment_id"
              @change="onTableChange"
            />
          </a-card>
        </a-col>
      </a-row>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import dayjs, { Dayjs } from "dayjs";
import { message, type TablePaginationConfig } from "ant-design-vue";
import PieChart from "@/components/Charts/PieChart.vue";
import LineChart from "@/components/Charts/LineChart.vue";
import { useRevenueAnalyticsStore } from "@/stores/revenueAnalytics";
import { useAdminAuthStore } from "@/stores/adminAuth";
import { listGoodsTypes, listPackages, listPlanGroups, listRegions } from "@/services/admin";
import type { Line, Package, RevenueAnalyticsLevel } from "@/services/types";

const store = useRevenueAnalyticsStore();
const adminAuth = useAdminAuthStore();

const rangeValue = ref<[Dayjs, Dayjs]>([dayjs().subtract(30, "day"), dayjs()]);
const regionRows = ref<any[]>([]);
const lineRows = ref<Line[]>([]);
const packageRows = ref<Package[]>([]);

const query = computed({
  get: () => store.query,
  set: (v) => store.setQuery(v)
});

const inferLevelFromSelection = (): RevenueAnalyticsLevel => {
  if (!query.value.goods_type_id) return "overall";
  if (query.value.package_id) return "package";
  if (query.value.line_id) return "line";
  if (query.value.region_id) return "region";
  return "goods_type";
};

const syncInferredLevel = () => {
  store.setQuery({ level: inferLevelFromSelection() });
};

const hasPermission = (required: string): boolean => {
  const requiredPerm = String(required || "").trim();
  if (!requiredPerm) return true;
  const perms = Array.isArray(adminAuth.profile?.permissions) ? adminAuth.profile.permissions : [];
  if (!perms.length) return false;
  for (const p of perms) {
    if (p === "*" || p === requiredPerm) return true;
    if (typeof p === "string" && p.endsWith("*")) {
      const prefix = p.slice(0, -1);
      if (requiredPerm.startsWith(prefix)) return true;
    }
  }
  return false;
};

const canView = computed(() => hasPermission("dashboard.revenue") || hasPermission("dashboard.revenue_analytics_overview"));

const goodsTypeOptions = ref<Array<{ label: string; value: number }>>([]);
const regionOptions = computed(() =>
  regionRows.value.map((it: any) => ({ label: String(it.name || it.Name || `地区-${it.id || it.ID}`), value: Number(it.id || it.ID) }))
);
const lineOptions = computed(() => {
  const uniq = new Map<number, { label: string; value: number }>();
  for (const row of lineRows.value) {
    const lineID = Number(row.line_id || 0);
    if (!lineID || uniq.has(lineID)) continue;
    uniq.set(lineID, { label: `${row.name || "线路"} (${lineID})`, value: lineID });
  }
  return Array.from(uniq.values());
});
const packageOptions = computed(() => {
  const lineId = Number(query.value.line_id || 0);
  if (!lineId) return [];
  return packageRows.value
    .filter((pkg: Package) => {
      const plan = lineRows.value.find((ln: Line) => Number(ln.id) === Number(pkg.plan_group_id));
      return Number(plan?.line_id || 0) === lineId;
    })
    .map((pkg: Package) => ({ label: String(pkg.name || `套餐-${pkg.id}`), value: Number(pkg.id) }));
});

const overview = computed(() => store.overview || {});
const details = computed(() => store.details || []);
const topList = computed(() => store.top || []);

const shareChartData = computed(() => {
  const items = overview.value.share_items || [];
  return items.map((it: any) => ({ name: it.dimension_name, value: Number(it.revenue_cents || 0) }));
});

const trendChartData = computed(() => {
  const items = store.trend || [];
  return {
    labels: items.map((it: any) => it.bucket || ""),
    values: items.map((it: any) => Number(it.revenue_cents || 0))
  };
});

const columns = [
  { title: "支付ID", dataIndex: "payment_id", key: "payment_id", width: 100 },
  { title: "订单号", dataIndex: "order_no", key: "order_no" },
  { title: "用户ID", dataIndex: "user_id", key: "user_id", width: 90 },
  { title: "金额", dataIndex: "amount_cents", key: "amount_cents", customRender: ({ text }: any) => `¥${centsToYuan(text)}` },
  { title: "支付时间", dataIndex: "paid_at", key: "paid_at" }
];

const pagination = computed<TablePaginationConfig>(() => ({
  current: Number(store.query.page || 1),
  pageSize: Number(store.query.page_size || 20),
  total: Number(store.detailTotal || 0),
  showSizeChanger: true
}));

const centsToYuan = (val?: number) => (Number(val || 0) / 100).toFixed(2);

const formatRatio = (ratio?: number | null, comparable?: boolean) => {
  if (!comparable || ratio == null) return "不可比";
  return `${(ratio * 100).toFixed(2)}%`;
};

const ensureQueryValid = (silent = false) => {
  const level = inferLevelFromSelection();
  if (level === "overall") return true;
  if (level === "region" && !query.value.region_id) {
    if (!silent) message.warning("当前层级为地区，请选择地区");
    return false;
  }
  if (level === "line" && (!query.value.region_id || !query.value.line_id)) {
    if (!silent) message.warning("当前层级为线路，请先选择地区和线路");
    return false;
  }
  if (level === "package" && (!query.value.region_id || !query.value.line_id || !query.value.package_id)) {
    if (!silent) message.warning("当前层级为套餐，请先选择地区、线路和套餐");
    return false;
  }
  return true;
};

const loadGoodsTypes = async () => {
  const res = await listGoodsTypes();
  const items = res.data?.items || [];
  goodsTypeOptions.value = items.map((it: any) => ({ label: String(it.name || it.Name || "类型"), value: Number(it.id || it.ID) }));
};

const loadRegions = async () => {
  if (!query.value.goods_type_id) {
    regionRows.value = [];
    return;
  }
  const res = await listRegions({ goods_type_id: query.value.goods_type_id });
  regionRows.value = res.data?.items || [];
};

const loadLines = async () => {
  if (!query.value.goods_type_id || !query.value.region_id) {
    lineRows.value = [];
    return;
  }
  const res = await listPlanGroups({
    goods_type_id: query.value.goods_type_id,
    region_id: query.value.region_id
  });
  lineRows.value = (res.data?.items || []) as Line[];
};

const loadPackages = async () => {
  if (!query.value.goods_type_id || !query.value.region_id) {
    packageRows.value = [];
    return;
  }
  const res = await listPackages({ goods_type_id: query.value.goods_type_id });
  packageRows.value = (res.data?.items || []) as Package[];
};

const onGoodsTypeChange = async (value?: number) => {
  store.setQuery({
    goods_type_id: Number(value || 0) || undefined,
    region_id: undefined,
    line_id: undefined,
    package_id: undefined
  });
  syncInferredLevel();
  await loadRegions();
  lineRows.value = [];
  packageRows.value = [];
};

const onRegionChange = async (value?: number) => {
  store.setQuery({
    region_id: Number(value || 0) || undefined,
    line_id: undefined,
    package_id: undefined
  });
  syncInferredLevel();
  await loadLines();
  await loadPackages();
};

const onLineChange = async (value?: number) => {
  store.setQuery({
    line_id: Number(value || 0) || undefined,
    package_id: undefined
  });
  syncInferredLevel();
  await loadPackages();
};

const onPackageChange = (value?: number) => {
  store.setQuery({ package_id: Number(value || 0) || undefined });
  syncInferredLevel();
};

const onRangeChange = (vals: [Dayjs, Dayjs] | null) => {
  if (!vals || vals.length !== 2) return;
  store.setQuery({
    from_at: vals[0].startOf("day").toISOString(),
    to_at: vals[1].endOf("day").toISOString()
  });
};

const onTableChange = (pager: TablePaginationConfig) => {
  store.setQuery({ page: Number(pager.current || 1), page_size: Number(pager.pageSize || 20) });
  void store.fetchDetails();
};

const reloadAll = async () => {
  if (!ensureQueryValid(false)) return;
  syncInferredLevel();
  store.setQuery({ page: 1 });
  await store.fetchAll();
};

const onDrillDown = async (item: any) => {
  const id = Number(item?.dimension_id || 0);
  if (!id) return;
  const current = inferLevelFromSelection();
  if (current === "overall") {
    await onGoodsTypeChange(id);
  } else if (current === "goods_type") {
    await onGoodsTypeChange(id);
  } else if (current === "region") {
    await onRegionChange(id);
  } else if (current === "line") {
    await onLineChange(id);
  } else {
    onPackageChange(id);
  }
  syncInferredLevel();

  if (ensureQueryValid(true)) {
    await store.fetchAll();
  }
};

onMounted(async () => {
  await loadGoodsTypes();
  await loadRegions();
  if (query.value.region_id) {
    await loadLines();
    await loadPackages();
  }
  syncInferredLevel();
  if (ensureQueryValid(true)) {
    await store.fetchAll();
  }
});
</script>

<style scoped>
.kpi {
  font-size: 28px;
  font-weight: 700;
}

.drill-item {
  cursor: pointer;
  transition: background-color 0.2s ease;
}

.drill-item:hover {
  background: #f5f8ff;
}
</style>
