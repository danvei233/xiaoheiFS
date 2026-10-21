<template>
  <a-card class="card">
    <div class="table-toolbar flex space-between">
      <div class="subtle">共 {{ pagination.total }} 条</div>
      <div class="flex gap-8">
        <a-dropdown>
          <a-button>列设置</a-button>
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

const props = defineProps({
  columns: { type: Array, required: true },
  dataSource: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  pagination: { type: Object, required: true },
  rowKey: { type: [String, Function], default: "id" },
  selectable: { type: Boolean, default: false },
  rowSelection: { type: Object, default: null }
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
