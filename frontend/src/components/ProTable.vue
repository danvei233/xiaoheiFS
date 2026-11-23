<template>
  <!-- plain：页面已经提供卡片容器时使用，避免卡片套卡片出现双层边框 -->
  <a-card
    class="pro-table-card"
    :class="{ 'is-plain': plain }"
    :body-style="plain ? { padding: 0 } : undefined"
  >
    <div class="table-toolbar flex space-between">
      <div class="subtle">{{ t('common.total', { n: pagination.total }) }}</div>
      <div class="flex gap-8">
        <a-dropdown>
          <a-button>{{ t('common.columnSettings') }}</a-button>
          <template #overlay>
            <a-menu>
              <a-menu-item v-for="col in columns" :key="col.key" @click="toggleColumn(col.key)">
                <a-checkbox :checked="visibleKeys.includes(col.key)">{{ col.title }}</a-checkbox>
              </a-menu-item>
            </a-menu>
          </template>
        </a-dropdown>
        <slot name="toolbar" />
      </div>
    </div>
    <template v-if="isMobile && $slots.mobile">
      <slot name="mobile" />
    </template>
    <a-table
      v-else
      :columns="visibleColumns"
      :data-source="dataSource"
      :loading="loading"
      :row-selection="computedRowSelection"
      :pagination="pagination"
      :row-key="rowKey"
      @change="$emit('change', $event)"
    >
      <template #bodyCell="slotProps">
        <slot name="bodyCell" v-bind="slotProps" />
      </template>
    </a-table>
  </a-card>
</template>

<script setup>
import { computed, ref } from "vue";
import { Grid } from "ant-design-vue";
import { t } from "@/locales";

const props = defineProps({
  columns: { type: Array, required: true },
  dataSource: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  pagination: { type: Object, required: true },
  rowKey: { type: [String, Function], default: "id" },
  selectable: { type: Boolean, default: false },
  rowSelection: { type: Object, default: null },
  /** 由页面外层卡片提供容器时设为 true：表格只渲染内容，不再画一层卡片边框 */
  plain: { type: Boolean, default: false }
});

const emit = defineEmits(["change", "selectionChange"]);

const visibleKeys = ref(props.columns.map((col) => col.key));
const screens = Grid.useBreakpoint();

const isMobile = computed(() => !screens.value?.md);

const visibleColumns = computed(() => props.columns.filter((col) => visibleKeys.value.includes(col.key)));

const computedRowSelection = computed(() => {
  if (props.rowSelection) return props.rowSelection;
  if (!props.selectable) return null;
  return {
    selectedRowKeys: [],
    onChange: (keys, rows) => emit("selectionChange", { keys, rows })
  };
});

const toggleColumn = (key) => {
  if (visibleKeys.value.includes(key)) {
    visibleKeys.value = visibleKeys.value.filter((k) => k !== key);
  } else {
    visibleKeys.value.push(key);
  }
};
</script>

<style scoped>
/* 数据面板不参与「卡片悬停上浮」，否则整张表格会跟着鼠标跳动 */
.pro-table-card:hover {
  transform: none !important;
}

/* 嵌入页面已有卡片时，表格不再提供第二层边框/底色/阴影 */
.pro-table-card.is-plain,
.pro-table-card.is-plain:hover {
  border: 0 !important;
  background: transparent !important;
  box-shadow: none !important;
}

/* 部分主题会在 .ant-card 上叠加装饰性伪元素，嵌入模式下同样隐藏 */
.pro-table-card.is-plain::before,
.pro-table-card.is-plain::after {
  content: none !important;
}

/* 外层卡片已经有内边距，工具栏不再额外缩进，与表格左边缘对齐 */
.pro-table-card.is-plain :deep(.table-toolbar) {
  padding: 0 0 var(--spacing-md);
}
</style>
