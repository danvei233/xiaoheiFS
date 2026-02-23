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
      <a-card :bordered="false" class="filter-card">
        <a-row :gutter="[12, 12]" align="middle">
          <a-col flex="auto">
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

              <a-input-number
                v-model:value="query.user_id"
                style="width: 160px"
                :min="1"
                :precision="0"
                placeholder="用户ID"
                @change="onUserFilterChange"
              />

              <a-range-picker v-model:value="rangeValue" @change="onRangeChange" />
            </a-space>
          </a-col>
          <a-col>
            <a-space>
              <a-button type="primary" :loading="store.loading" @click="reloadAll">查询</a-button>
              <a-button @click="onResetFilters">重置</a-button>
            </a-space>
          </a-col>
        </a-row>

        <a-divider style="margin: 12px 0" />

        <a-row justify="space-between" align="middle" :gutter="[12, 8]">
          <a-col :xs="24" :md="14" :lg="12">
            <a-space wrap>
              <span class="subtle">快捷时间：</span>
              <a-segmented v-model:value="quickRangeKey" :options="quickRangeOptions" @change="onQuickRangeChange" />
            </a-space>
          </a-col>
          <a-col v-if="hasActiveFilters" :xs="24" :md="10" :lg="12">
            <a-space wrap class="active-filters">
              <span class="subtle">当前筛选：</span>
              <a-tag v-if="query.goods_type_id" color="blue">类型: {{ lookupLabel(goodsTypeOptions, query.goods_type_id) }}</a-tag>
              <a-tag v-if="query.region_id" color="geekblue">地区: {{ lookupLabel(regionOptions, query.region_id) }}</a-tag>
              <a-tag v-if="query.line_id" color="cyan">线路: {{ lookupLabel(lineOptions, query.line_id) }}</a-tag>
              <a-tag v-if="query.package_id" color="purple">套餐: {{ lookupLabel(packageOptions, query.package_id) }}</a-tag>
              <a-tag v-if="query.user_id" color="green">用户: {{ query.user_id }}</a-tag>
            </a-space>
          </a-col>
        </a-row>
      </a-card>

      <a-row :gutter="16" class="analytics-row analytics-row-kpi">
        <a-col :xs="24" :md="8">
          <a-card :bordered="false" title="总收入" :loading="store.loading" class="kpi-card">
            <div class="kpi">¥{{ centsToYuan(overview.summary?.total_revenue_cents) }}</div>
            <div class="kpi-sub-row">
              <span class="subtle">订单数：{{ overview.summary?.order_count || 0 }}</span>
              <a-tag :color="periodCompare.color">{{ periodCompare.tag }}</a-tag>
            </div>
            <div class="compare-line" :class="periodCompare.className">
              相比上个周期：{{ periodCompare.text }}
              <template v-if="periodCompare.percentText">
                （{{ periodCompare.percentText }}）
              </template>
            </div>
          </a-card>
        </a-col>
        <a-col :xs="24" :md="8">
          <a-card :bordered="false" title="同比" :loading="store.loading" class="kpi-card">
            <div class="kpi">{{ formatRatio(overview.summary?.yoy_ratio, overview.summary?.yoy_comparable) }}</div>
          </a-card>
        </a-col>
        <a-col :xs="24" :md="8">
          <a-card :bordered="false" title="环比" :loading="store.loading" class="kpi-card">
            <div class="kpi">{{ formatRatio(overview.summary?.mom_ratio, overview.summary?.mom_comparable) }}</div>
          </a-card>
        </a-col>
      </a-row>

      <a-row :gutter="16" class="analytics-row">
        <a-col :xs="24" :lg="12">
          <a-card :bordered="false" title="收入占比（可点下钻）" class="chart-card">
            <PieChart :data="shareChartData" @slice-click="onShareSliceClick" />
          </a-card>
        </a-col>
        <a-col :xs="24" :lg="12">
          <a-card :bordered="false" title="收入趋势" class="chart-card">
            <LineChart
              :data="trendChartData"
              :y-axis-value-formatter="formatTrendYValue"
              :tooltip-value-formatter="formatTrendTooltipValue"
            />
          </a-card>
        </a-col>
      </a-row>

      <a-row :gutter="16" class="analytics-row data-row">
        <a-col :xs="24" :lg="8">
          <a-card :bordered="false" :loading="store.loading || userRankLoading" class="top-card">
            <a-tabs v-model:activeKey="leftPanelTab" size="small">
              <a-tab-pane key="dimension" tab="维度Top5">
                <a-list :data-source="topList" size="small" :locale="{ emptyText: '暂无数据' }">
                  <template #renderItem="{ item }">
                    <a-list-item class="drill-item" @click="onDrillDown(item)">
                      <div class="top-item-left">
                        <a-tag :color="item.rank <= 3 ? 'gold' : 'default'">#{{ item.rank }}</a-tag>
                        <span>{{ item.dimension_name }}</span>
                      </div>
                      <div class="top-item-right">
                        <strong>¥{{ centsToYuan(item.revenue_cents) }}</strong>
                        <span class="subtle">{{ ((Number(item.ratio || 0) * 100) || 0).toFixed(2) }}%</span>
                      </div>
                    </a-list-item>
                  </template>
                </a-list>
              </a-tab-pane>
              <a-tab-pane key="user" tab="用户消费榜">
                <a-list :data-source="userRankList" size="small" :locale="{ emptyText: '当前筛选下无用户消费数据' }">
                  <template #renderItem="{ item }">
                    <a-list-item class="drill-item" @click="openUserFinance(item.user_id)">
                      <div class="top-item-left">
                        <a-tag :color="item.rank <= 3 ? 'gold' : 'default'">#{{ item.rank }}</a-tag>
                        <a-avatar :size="22" :src="userAvatar(item.user_id)">{{ String(item.user_id || "").slice(-2) }}</a-avatar>
                        <span>{{ formatUserLabel(item.user_id) }}</span>
                      </div>
                      <div class="top-item-right">
                        <strong>¥{{ centsToYuan(item.revenue_cents) }}</strong>
                        <span class="subtle">{{ item.order_count }} 单</span>
                      </div>
                    </a-list-item>
                  </template>
                </a-list>
              </a-tab-pane>
            </a-tabs>
          </a-card>
        </a-col>
        <a-col :xs="24" :lg="16">
          <a-card :bordered="false" title="明细表" :loading="store.loading" class="detail-card">
            <a-table
              :data-source="details"
              :columns="columns"
              :pagination="pagination"
              :scroll="{ x: 980 }"
              size="middle"
              row-key="payment_id"
              @change="onTableChange"
            >
              <template #bodyCell="{ column, text, record }">
                <template v-if="column.key === 'order_no'">
                  <a-typography-text class="order-no-text" :copyable="!!text">{{ text || "-" }}</a-typography-text>
                </template>
                <template v-else-if="column.key === 'user_id'">
                  <a-space size="small">
                    <a-avatar :size="22" :src="userAvatar(Number(text || 0))">{{ String(text || "").slice(-2) }}</a-avatar>
                    <a-button type="link" size="small" @click="openUserFinance(Number(text || 0))">{{ formatUserLabel(Number(text || 0)) }}</a-button>
                  </a-space>
                </template>
                <template v-else-if="column.key === 'amount_cents'">
                  <span :class="amountClass(Number(text || 0))">
                    {{ amountPrefix(Number(text || 0)) }}¥{{ centsToYuan(Math.abs(Number(text || 0))) }}
                  </span>
                </template>
                <template v-else-if="column.key === 'status'">
                  <a-badge :status="statusMeta(record.status).badge" :text="statusMeta(record.status).text" />
                </template>
                <template v-else-if="column.key === 'paid_at'">
                  <span class="subtle">{{ formatDateTime(String(text || '')) }}</span>
                </template>
              </template>
            </a-table>
          </a-card>
        </a-col>
      </a-row>

      <a-drawer
        v-model:open="userFinanceOpen"
        width="760"
        :title="`用户财务详情 #${selectedUserId || '-'}`"
        @close="onCloseUserFinance"
      >
        <a-spin :spinning="userFinanceLoading">
          <a-descriptions :column="2" bordered size="small">
            <a-descriptions-item label="用户">
              <a-space size="small">
                <a-avatar :size="24" :src="userAvatar(selectedUserId)">{{ String(selectedUserId || "").slice(-2) }}</a-avatar>
                <span>{{ formatUserLabel(selectedUserId) }}</span>
              </a-space>
            </a-descriptions-item>
            <a-descriptions-item label="邮箱">{{ userFinanceProfile?.email || "-" }}</a-descriptions-item>
            <a-descriptions-item label="总收入贡献">¥{{ centsToYuan(userFinanceSummary.total_revenue_cents) }}</a-descriptions-item>
            <a-descriptions-item label="订单数">{{ userFinanceSummary.order_count }}</a-descriptions-item>
            <a-descriptions-item label="净收入订单">{{ userFinanceSummary.positive_order_count }}</a-descriptions-item>
            <a-descriptions-item label="退款/负数订单">{{ userFinanceSummary.negative_order_count }}</a-descriptions-item>
            <a-descriptions-item label="客单价">¥{{ centsToYuan(userFinanceSummary.avg_order_cents) }}</a-descriptions-item>
            <a-descriptions-item label="最近支付时间">{{ userFinanceSummary.last_paid_at || "-" }}</a-descriptions-item>
          </a-descriptions>

          <a-divider />

          <a-space style="margin-bottom: 12px">
            <a-button type="primary" @click="applyUserFilterFromDrawer">按此用户筛选主视图</a-button>
          </a-space>

          <a-table
            :data-source="userFinanceRows"
            :columns="columns"
            :pagination="false"
            :scroll="{ x: 980 }"
            size="small"
            row-key="payment_id"
          >
            <template #bodyCell="{ column, text, record }">
              <template v-if="column.key === 'order_no'">
                <a-typography-text class="order-no-text" :copyable="!!text">{{ text || "-" }}</a-typography-text>
              </template>
              <template v-else-if="column.key === 'user_id'">
                <a-space size="small">
                  <a-avatar :size="20" :src="userAvatar(Number(text || 0))">{{ String(text || "").slice(-2) }}</a-avatar>
                  <span>{{ formatUserLabel(Number(text || 0)) }}</span>
                </a-space>
              </template>
              <template v-else-if="column.key === 'amount_cents'">
                <span :class="amountClass(Number(text || 0))">
                  {{ amountPrefix(Number(text || 0)) }}¥{{ centsToYuan(Math.abs(Number(text || 0))) }}
                </span>
              </template>
              <template v-else-if="column.key === 'status'">
                <a-badge :status="statusMeta(record.status).badge" :text="statusMeta(record.status).text" />
              </template>
              <template v-else-if="column.key === 'paid_at'">
                <span class="subtle">{{ formatDateTime(String(text || '')) }}</span>
              </template>
            </template>
          </a-table>
        </a-spin>
      </a-drawer>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import dayjs, { Dayjs } from "dayjs";
import { message, type TablePaginationConfig } from "ant-design-vue";
import PieChart from "@/components/Charts/PieChart.vue";
import LineChart from "@/components/Charts/LineChart.vue";
import { useRevenueAnalyticsStore } from "@/stores/revenueAnalytics";
import { useAdminAuthStore } from "@/stores/adminAuth";
import { getAdminUserDetail, getRevenueAnalyticsDetails, listGoodsTypes, listPackages, listPlanGroups, listRegions } from "@/services/admin";
import type { Line, Package, RevenueAnalyticsDetailRecord, RevenueAnalyticsLevel, RevenueAnalyticsQuery } from "@/services/types";

const store = useRevenueAnalyticsStore();
const adminAuth = useAdminAuthStore();

const defaultRange = (): [Dayjs, Dayjs] => [dayjs().subtract(1, "month").startOf("day"), dayjs().endOf("day")];
const rangeValue = ref<[Dayjs, Dayjs]>(defaultRange());
const quickRangeOptions = [
  { label: "今天", value: "today" },
  { label: "近7天", value: "7d" },
  { label: "近30天", value: "30d" },
  { label: "本月", value: "month" }
];
const quickRangeKey = ref<string>("30d");
const regionRows = ref<any[]>([]);
const lineRows = ref<Line[]>([]);
const packageRows = ref<Package[]>([]);
const leftPanelTab = ref<"dimension" | "user">("dimension");
const userRankLoading = ref(false);
const userRankList = ref<Array<{ rank: number; user_id: number; revenue_cents: number; order_count: number }>>([]);
const userFinanceOpen = ref(false);
const userFinanceLoading = ref(false);
const selectedUserId = ref<number>();
const userFinanceProfile = ref<any>(null);
const userFinanceRows = ref<RevenueAnalyticsDetailRecord[]>([]);
const userMetaMap = ref<Record<number, { username?: string; avatar?: string; email?: string }>>({});
const userFinanceSummary = ref({
  total_revenue_cents: 0,
  order_count: 0,
  positive_order_count: 0,
  negative_order_count: 0,
  avg_order_cents: 0,
  last_paid_at: ""
});

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
const hasActiveFilters = computed(() => !!(query.value.goods_type_id || query.value.region_id || query.value.line_id || query.value.package_id || query.value.user_id));
const periodCompare = computed(() => {
  const current = Number(overview.value.summary?.total_revenue_cents || 0);
  const ratio = overview.value.summary?.mom_ratio;
  const comparable = !!overview.value.summary?.mom_comparable && ratio != null;
  if (!comparable || ratio == null || ratio <= -1) {
    return { color: "default", tag: "不可比", className: "compare-neutral", text: "暂无可比数据", percentText: "" };
  }
  const previous = current / (1 + ratio);
  const delta = current - previous;
  const up = delta >= 0;
  return {
    color: up ? "success" : "error",
    tag: up ? "增长" : "下降",
    className: up ? "compare-up" : "compare-down",
    text: `${up ? "增长" : "下降"} ¥${(Math.abs(delta) / 100).toFixed(2)}`,
    percentText: `${up ? "+" : "-"}${Math.abs(ratio * 100).toFixed(2)}%`
  };
});

const shareChartData = computed(() => {
  const items = overview.value.share_items || [];
  return items.map((it: any) => ({
    name: it.dimension_name,
    value: Number(it.revenue_cents || 0),
    dimension_id: Number(it.dimension_id || 0)
  }));
});

const trendChartData = computed(() => {
  const items = store.trend || [];
  return {
    labels: items.map((it: any) => it.bucket || ""),
    values: items.map((it: any) => Number(it.revenue_cents || 0) / 100)
  };
});

const columns = [
  { title: "支付ID", dataIndex: "payment_id", key: "payment_id", width: 110 },
  { title: "订单号", dataIndex: "order_no", key: "order_no", width: 220, ellipsis: true },
  { title: "用户", dataIndex: "user_id", key: "user_id", width: 220 },
  { title: "金额", dataIndex: "amount_cents", key: "amount_cents", width: 140 },
  { title: "状态", dataIndex: "status", key: "status", width: 120 },
  { title: "支付时间", dataIndex: "paid_at", key: "paid_at", width: 210 }
];

const pagination = computed<TablePaginationConfig>(() => ({
  current: Number(store.query.page || 1),
  pageSize: Number(store.query.page_size || 20),
  total: Number(store.detailTotal || 0),
  showSizeChanger: true
}));

const centsToYuan = (val?: number) => (Number(val || 0) / 100).toFixed(2);
const formatTrendYValue = (value?: number) => `¥${Number(value || 0).toFixed(2)}`;
const formatTrendTooltipValue = (value?: number) => `¥${Number(value || 0).toFixed(2)}`;
const formatDateTime = (value: string) => {
  if (!value) return "-";
  const d = dayjs(value);
  return d.isValid() ? d.format("YYYY-MM-DD HH:mm:ss") : value;
};
const amountClass = (value: number) => {
  if (value > 0) return "amount-up";
  if (value < 0) return "amount-down";
  return "amount-neutral";
};
const amountPrefix = (value: number) => {
  if (value > 0) return "+";
  if (value < 0) return "-";
  return "";
};
const setUserMeta = (id: number, payload: any) => {
  if (!id) return;
  const existing = userMetaMap.value[id] || {};
  userMetaMap.value[id] = {
    username: String(payload?.username || payload?.Username || existing.username || "").trim(),
    avatar: String(payload?.avatar || payload?.avatar_url || payload?.AvatarURL || existing.avatar || "").trim(),
    email: String(payload?.email || payload?.Email || existing.email || "").trim()
  };
};
const ensureUserMeta = async (ids: number[]) => {
  const unique = Array.from(new Set(ids.map((v) => Number(v || 0)).filter((v) => v > 0)));
  const missing = unique.filter((id) => !userMetaMap.value[id]?.username && !userMetaMap.value[id]?.avatar);
  if (!missing.length) return;
  await Promise.all(
    missing.map(async (id) => {
      try {
        const res = await getAdminUserDetail(id);
        setUserMeta(id, res.data || {});
      } catch {
        // ignore single user fetch failure
      }
    })
  );
};
const formatUserLabel = (id?: number) => {
  const uid = Number(id || 0);
  if (!uid) return "未知用户";
  const username = userMetaMap.value[uid]?.username;
  return username || "未知用户";
};
const userAvatar = (id?: number) => {
  const uid = Number(id || 0);
  if (!uid) return "";
  return userMetaMap.value[uid]?.avatar || "";
};
const statusMeta = (status: string) => {
  const s = String(status || "").toLowerCase();
  if (s === "approved" || s === "active") return { badge: "success" as const, text: "已完成" };
  if (s === "pending_payment") return { badge: "processing" as const, text: "待支付" };
  if (s === "pending_review") return { badge: "warning" as const, text: "待审核" };
  if (s === "canceled") return { badge: "default" as const, text: "已取消" };
  if (s === "failed" || s === "rejected") return { badge: "error" as const, text: "失败" };
  return { badge: "default" as const, text: s || "-" };
};
const lookupLabel = (source: Array<{ label: string; value: number }>, value: number | undefined) => {
  const id = Number(value || 0);
  if (!id) return "-";
  return source.find((it) => Number(it.value) === id)?.label || String(id);
};
const detectQuickRangeKey = (from: Dayjs, to: Dayjs) => {
  const now = dayjs();
  if (from.isSame(now.startOf("day")) && to.isSame(now.endOf("day"))) return "today";
  if (from.isSame(now.subtract(6, "day").startOf("day")) && to.isSame(now.endOf("day"))) return "7d";
  if (from.isSame(now.subtract(1, "month").startOf("day")) && to.isSame(now.endOf("day"))) return "30d";
  if (from.isSame(now.startOf("month")) && to.isSame(now.endOf("day"))) return "month";
  return "30d";
};

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

const onUserFilterChange = (value?: number | null) => {
  store.setQuery({ user_id: Number(value || 0) || undefined, page: 1 });
};

const fetchAllDetailRows = async (extra: Partial<RevenueAnalyticsQuery> = {}) => {
  const base = {
    ...store.query,
    page: 1,
    page_size: 200,
    sort_field: "paid_at" as const,
    sort_order: "desc" as const,
    ...extra
  };
  const rows: RevenueAnalyticsDetailRecord[] = [];
  let page = 1;
  let total = 0;
  const pageSize = Number(base.page_size || 200);
  for (;;) {
    const res = await getRevenueAnalyticsDetails({ ...base, page });
    const items = (res.data?.items || []) as RevenueAnalyticsDetailRecord[];
    total = Number(res.data?.total || 0);
    rows.push(...items);
    if (items.length < pageSize || rows.length >= total || page >= 50) {
      break;
    }
    page += 1;
  }
  return rows;
};

const refreshUserRanking = async () => {
  userRankLoading.value = true;
  try {
    const rows = await fetchAllDetailRows({ user_id: undefined });
    const agg = new Map<number, { revenue_cents: number; orders: Set<number> }>();
    for (const row of rows) {
      const uid = Number(row.user_id || 0);
      if (!uid) continue;
      if (!agg.has(uid)) {
        agg.set(uid, { revenue_cents: 0, orders: new Set<number>() });
      }
      const item = agg.get(uid)!;
      item.revenue_cents += Number(row.amount_cents || 0);
      if (row.order_id) item.orders.add(Number(row.order_id));
    }
    const ranked = Array.from(agg.entries())
      .map(([userID, item]) => ({
        user_id: userID,
        revenue_cents: item.revenue_cents,
        order_count: item.orders.size
      }))
      .sort((a, b) => b.revenue_cents - a.revenue_cents)
      .slice(0, 20)
      .map((it, idx) => ({ ...it, rank: idx + 1 }));
    userRankList.value = ranked;
    await ensureUserMeta(ranked.map((it) => it.user_id));
  } finally {
    userRankLoading.value = false;
  }
};

const onRangeChange = (vals: [Dayjs, Dayjs] | null) => {
  if (!vals || vals.length !== 2) return;
  quickRangeKey.value = detectQuickRangeKey(vals[0].startOf("day"), vals[1].endOf("day"));
  store.setQuery({
    from_at: vals[0].startOf("day").toISOString(),
    to_at: vals[1].endOf("day").toISOString()
  });
};

const applyQuickRange = (key: string) => {
  const now = dayjs();
  let from = now.startOf("day");
  let to = now.endOf("day");
  if (key === "7d") {
    from = now.subtract(6, "day").startOf("day");
  } else if (key === "30d") {
    from = now.subtract(1, "month").startOf("day");
  } else if (key === "month") {
    from = now.startOf("month");
  }
  rangeValue.value = [from, to];
  store.setQuery({ from_at: from.toISOString(), to_at: to.toISOString(), page: 1 });
};

const onQuickRangeChange = async (key: string | number) => {
  applyQuickRange(String(key));
  await reloadAll();
};

const onTableChange = (pager: TablePaginationConfig) => {
  store.setQuery({ page: Number(pager.current || 1), page_size: Number(pager.pageSize || 20) });
  void store.fetchDetails();
};

const reloadAll = async () => {
  if (!ensureQueryValid(false)) return;
  syncInferredLevel();
  store.setQuery({ page: 1 });
  await Promise.all([store.fetchAll(), refreshUserRanking()]);
};

const onDrillDown = async (item: any) => {
  const id = Number(item?.dimension_id || 0);
  if (!id) return;
  const current = inferLevelFromSelection();
  if (current === "overall") {
    await onGoodsTypeChange(id);
  } else if (current === "goods_type") {
    await onRegionChange(id);
  } else if (current === "region") {
    await onLineChange(id);
  } else if (current === "line") {
    onPackageChange(id);
  } else {
    return;
  }
  syncInferredLevel();

  if (ensureQueryValid(true)) {
    await store.fetchAll();
  }
};

const onShareSliceClick = async (item: any) => {
  await onDrillDown(item);
};

const summarizeUserFinance = (rows: RevenueAnalyticsDetailRecord[]) => {
  const orderMap = new Map<number, number>();
  let lastPaid = "";
  for (const row of rows) {
    const orderID = Number(row.order_id || 0);
    const amount = Number(row.amount_cents || 0);
    if (orderID > 0) {
      orderMap.set(orderID, (orderMap.get(orderID) || 0) + amount);
    }
    const paidAt = String(row.paid_at || "");
    if (paidAt && (!lastPaid || dayjs(paidAt).isAfter(lastPaid))) {
      lastPaid = paidAt;
    }
  }
  const orderTotals = Array.from(orderMap.values());
  const total = orderTotals.reduce((sum, v) => sum + v, 0);
  const positive = orderTotals.filter((v) => v > 0).length;
  const negative = orderTotals.filter((v) => v < 0).length;
  return {
    total_revenue_cents: total,
    order_count: orderTotals.length,
    positive_order_count: positive,
    negative_order_count: negative,
    avg_order_cents: orderTotals.length > 0 ? Math.trunc(total / orderTotals.length) : 0,
    last_paid_at: lastPaid ? formatDateTime(lastPaid) : ""
  };
};

const openUserFinance = async (userID: number) => {
  const id = Number(userID || 0);
  if (!id) return;
  userFinanceOpen.value = true;
  userFinanceLoading.value = true;
  selectedUserId.value = id;
  try {
    const [userRes, rows] = await Promise.all([
      getAdminUserDetail(id).catch(() => ({ data: null })),
      fetchAllDetailRows({ user_id: id })
    ]);
    userFinanceProfile.value = userRes.data || null;
    setUserMeta(id, userFinanceProfile.value || {});
    userFinanceRows.value = rows.slice(0, 100);
    userFinanceSummary.value = summarizeUserFinance(rows);
    await ensureUserMeta(rows.map((it) => Number(it.user_id || 0)));
  } finally {
    userFinanceLoading.value = false;
  }
};

const onCloseUserFinance = () => {
  userFinanceProfile.value = null;
  userFinanceRows.value = [];
  selectedUserId.value = undefined;
};

const applyUserFilterFromDrawer = async () => {
  if (!selectedUserId.value) return;
  store.setQuery({ user_id: selectedUserId.value, page: 1 });
  userFinanceOpen.value = false;
  await reloadAll();
};

const onResetFilters = async () => {
  const [from, to] = defaultRange();
  rangeValue.value = [from, to];
  quickRangeKey.value = "30d";
  store.setQuery({
    goods_type_id: undefined,
    region_id: undefined,
    line_id: undefined,
    package_id: undefined,
    user_id: undefined,
    from_at: from.toISOString(),
    to_at: to.toISOString(),
    page: 1
  });
  regionRows.value = [];
  lineRows.value = [];
  packageRows.value = [];
  await reloadAll();
};

onMounted(async () => {
  const from = dayjs(query.value.from_at || defaultRange()[0]).startOf("day");
  const to = dayjs(query.value.to_at || defaultRange()[1]).endOf("day");
  rangeValue.value = [from, to];
  quickRangeKey.value = detectQuickRangeKey(from, to);
  store.setQuery({ from_at: from.toISOString(), to_at: to.toISOString() });

  await loadGoodsTypes();
  await loadRegions();
  if (query.value.region_id) {
    await loadLines();
    await loadPackages();
  }
  syncInferredLevel();
  if (ensureQueryValid(true)) {
    await Promise.all([store.fetchAll(), refreshUserRanking()]);
  }
});

watch(
  details,
  async (rows) => {
    await ensureUserMeta((rows || []).map((it) => Number(it.user_id || 0)));
  },
  { immediate: true }
);
</script>

<style scoped>
.analytics-row {
  margin-top: 16px;
}

.kpi {
  font-size: 28px;
  font-weight: 700;
  line-height: 1.2;
}

.kpi-sub-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 8px;
}

.filter-card :deep(.ant-segmented) {
  background: #f7f9fc;
}

.filter-card :deep(.ant-card-body) {
  padding: 16px 20px;
}

.active-filters {
  justify-content: flex-end;
}

.compare-line {
  margin-top: 8px;
  font-size: 13px;
}

.compare-up {
  color: #389e0d;
}

.compare-down {
  color: #cf1322;
}

.compare-neutral {
  color: #6b7280;
}

.top-item-left {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.top-item-right {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  line-height: 1.3;
}

.amount-up {
  color: #389e0d;
  font-weight: 600;
}

.amount-down {
  color: #cf1322;
  font-weight: 600;
}

.amount-neutral {
  color: #4b5563;
  font-weight: 600;
}

.drill-item {
  cursor: pointer;
  transition: all 0.2s ease;
}

.drill-item:hover {
  background: #f5f8ff;
  transform: translateX(2px);
}

.analytics-row-kpi .ant-col,
.analytics-row .ant-col,
.data-row .ant-col {
  display: flex;
}

.kpi-card,
.chart-card,
.top-card,
.detail-card {
  width: 100%;
}

.kpi-card {
  min-height: 192px;
}

.chart-card {
  min-height: 388px;
}

.top-card,
.detail-card {
  min-height: 420px;
}

.kpi-card :deep(.ant-card-body) {
  padding-top: 18px;
}

.top-card :deep(.ant-card-body) {
  padding-bottom: 8px;
}

.detail-card :deep(.ant-table-wrapper) {
  min-height: 320px;
}

.detail-card :deep(.ant-table-thead > tr > th),
.detail-card :deep(.ant-table-tbody > tr > td) {
  white-space: nowrap;
}

.order-no-text {
  font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace;
}

@media (max-width: 768px) {
  .kpi {
    font-size: 24px;
  }

  .active-filters {
    justify-content: flex-start;
  }

  .kpi-card,
  .chart-card,
  .top-card,
  .detail-card {
    min-height: 0;
  }
}
</style>
