<template>
  <a-card class="card">
    <div v-if="tabOptions.length" class="status-tabs">
      <a-segmented :options="tabOptions" v-model:value="tabValue" @change="onTabChange" />
    </div>
    <div class="filter-bar">
      <a-input-search v-model:value="local.keyword" placeholder="关键词搜索" style="width: 220px" @search="emitSearch" />
      <a-select v-model:value="local.status" placeholder="状态" style="width: 160px" allow-clear>
        <a-select-option v-for="item in statusOptions" :key="item.value" :value="item.value">
          {{ item.label }}
        </a-select-option>
      </a-select>
      <a-range-picker v-if="showRange" v-model:value="local.range" />
      <a-popover trigger="click" placement="bottomLeft">
        <template #content>
          <slot name="advanced">
            <div class="subtle">无高级筛选项</div>
          </slot>
        </template>
        <a-button>高级筛选</a-button>
      </a-popover>
      <a-button @click="emitReset">重置</a-button>
      <a-button @click="$emit('refresh')">刷新</a-button>
      <a-button v-if="showExport" @click="$emit('export')">导出 CSV</a-button>
      <slot name="actions" />
    </div>
  </a-card>
</template>

<script setup>
import { computed, reactive, ref, watch } from "vue";

const props = defineProps({
  filters: { type: Object, required: true },
  statusOptions: { type: Array, default: () => [] },
  statusTabs: { type: Array, default: () => [] },
  showRange: { type: Boolean, default: true },
  showExport: { type: Boolean, default: true }
});

const emit = defineEmits(["update:filters", "search", "refresh", "reset", "export"]);

const local = reactive({
  keyword: props.filters.keyword || "",
  status: props.filters.status || undefined,
  range: props.filters.range || []
});

const tabOptions = computed(() => {
  if (!props.statusTabs.length) return [];
  return [
    { label: "全部", value: "all" },
    ...props.statusTabs.map((item) => ({ label: item.label, value: item.value }))
  ];
});

const tabValue = ref(local.status ?? "all");

watch(
  () => props.filters,
  (next) => {
    local.keyword = next.keyword || "";
    local.status = next.status || undefined;
    local.range = next.range || [];
    tabValue.value = next.status || "all";
  },
  { deep: true }
);

const emitSearch = () => {
  emit("update:filters", { ...props.filters, ...local });
  emit("search", { ...local });
};

const emitReset = () => {
  local.keyword = "";
  local.status = undefined;
  local.range = [];
  tabValue.value = "all";
  emit("update:filters", { keyword: "", status: undefined, range: [] });
  emit("reset");
};

const onTabChange = (val) => {
  if (val === "all") {
    local.status = undefined;
  } else {
    local.status = val;
  }
  emitSearch();
};
</script>
