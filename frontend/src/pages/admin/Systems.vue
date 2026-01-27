<template>
  <div class="page">
    <div class="page-header">
      <div>
        <div class="page-title">系统镜像</div>
        <div class="subtle">镜像库维护与同步</div>
      </div>
      <div class="page-header-actions">
        <a-space>
          <a-button danger :disabled="!selectedKeys.length" @click="bulkRemove">批量删除</a-button>
          <a-button @click="openSync">同步镜像</a-button>
          <a-button @click="openLineConfig">线路镜像配置</a-button>
          <a-button type="primary" @click="openCreate">新增镜像</a-button>
        </a-space>
      </div>
    </div>
    <div class="subtle" style="margin-bottom: 12px">
      同步会调用自动化 /mirror_image?line_id=... 并更新该线路启用的镜像关系
    </div>

    <FilterBar
      v-model:filters="filters"
      :status-options="statusOptions"
      @search="fetchData"
      @refresh="fetchData"
      @reset="fetchData"
      :show-export="false"
    />
    <ProTable
      :columns="columns"
      :data-source="dataSource"
      :loading="loading"
      :pagination="pagination"
      :row-selection="rowSelection"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'type'">
          <a-tag :color="typeTagColor(record.type)">
            <WindowsOutlined v-if="isWindowsType(record.type)" />
            <CodeOutlined v-else-if="isLinuxType(record.type)" />
            <span style="margin-left: 6px">{{ formatImageType(record.type) }}</span>
          </a-tag>
        </template>
        <template v-else-if="column.key === 'enabled'">
          <a-tag :color="record.enabled ? 'green' : 'red'">{{ record.enabled ? '启用' : '停用' }}</a-tag>
        </template>
        <template v-else-if="column.key === 'action'">
          <a-space>
            <a-button size="small" @click="openEdit(record)">编辑</a-button>
            <a-button size="small" danger @click="remove(record)">删除</a-button>
          </a-space>
        </template>
      </template>
    </ProTable>

    <a-modal v-model:open="formOpen" title="系统镜像" @ok="submit">
      <a-form layout="vertical">
        <a-form-item label="镜像 ID"><a-input v-model:value="form.image_id" /></a-form-item>
        <a-form-item label="名称"><a-input v-model:value="form.name" /></a-form-item>
        <a-form-item label="类型"><a-input v-model:value="form.type" /></a-form-item>
        <a-form-item label="启用"><a-switch v-model:checked="form.enabled" /></a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="configOpen" title="线路镜像配置" @ok="submitLineConfig">
      <a-form layout="vertical">
        <a-form-item label="线路">
          <a-select v-model:value="configLineId" placeholder="选择线路">
            <a-select-option v-for="line in lines" :key="line.id" :value="line.id">
              {{ `${line.name} (${line.line_id ?? line.id})` }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="启用镜像">
          <a-select
            v-model:value="configImageIds"
            mode="multiple"
            placeholder="选择启用的镜像"
            :options="imageOptions"
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="syncOpen" title="同步镜像" @ok="submitSync">
      <a-form layout="vertical">
        <a-form-item label="线路">
          <a-select v-model:value="syncLineId" placeholder="选择线路">
            <a-select-option v-for="line in lines" :key="line.id" :value="line.id">
              {{ `${line.name} (${line.line_id ?? line.id})` }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <div class="subtle">同步会调用自动化 /mirror_image?line_id=... 并更新该线路启用的镜像关系</div>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { computed, reactive, ref, watch } from "vue";
import FilterBar from "@/components/FilterBar.vue";
import ProTable from "@/components/ProTable.vue";
import { CodeOutlined, WindowsOutlined } from "@ant-design/icons-vue";
import {
  listSystemImages,
  syncSystemImages,
  createSystemImage,
  updateSystemImage,
  deleteSystemImage,
  bulkDeleteSystemImages,
  setLineSystemImages,
  listLines
} from "@/services/admin";
import { message, Modal } from "ant-design-vue";

const filters = reactive({ keyword: "", status: undefined, range: [] });
const statusOptions = [
  { label: "enabled", value: "enabled" },
  { label: "disabled", value: "disabled" }
];

const loading = ref(false);
const dataSource = ref([]);
const pagination = reactive({ current: 1, pageSize: 20, total: 0, showSizeChanger: true });
const lines = ref([]);
const selectedKeys = ref([]);
const syncLineId = ref(null);
const syncOpen = ref(false);
const rowSelection = computed(() => ({
  selectedRowKeys: selectedKeys.value,
  onChange: (keys) => {
    selectedKeys.value = keys;
  }
}));

const formOpen = ref(false);
const form = reactive({ id: null, image_id: "", name: "", type: "", enabled: true });
const configOpen = ref(false);
const configLineId = ref(null);
const configImageIds = ref([]);
const allImages = ref([]);

const imageOptions = computed(() =>
  allImages.value.map((img) => ({
    label: `${img.name} (${img.type || "-"})`,
    value: img.id
  }))
);

const columns = [
  { title: "ID", dataIndex: "id", key: "id" },
  { title: "镜像 ID", dataIndex: "image_id", key: "image_id" },
  { title: "名称", dataIndex: "name", key: "name" },
  { title: "类型", dataIndex: "type", key: "type" },
  { title: "启用", dataIndex: "enabled", key: "enabled" },
  { title: "操作", key: "action" }
];

const normalize = (row) => ({
  id: row.id ?? row.ID,
  image_id: row.image_id ?? row.ImageID,
  name: row.name ?? row.Name,
  type: row.type ?? row.Type,
  enabled: row.enabled ?? row.Enabled
});

const isWindowsType = (value) => String(value || "").toLowerCase().includes("win");
const isLinuxType = (value) => String(value || "").toLowerCase().includes("linux");
const formatImageType = (value) => (value ? String(value) : "-");
const typeTagColor = (value) => {
  if (isWindowsType(value)) return "blue";
  if (isLinuxType(value)) return "green";
  return "default";
};

const fetchData = async () => {
  loading.value = true;
  try {
    const res = await listSystemImages();
    const payload = res.data || {};
    dataSource.value = (payload.items || []).map(normalize);
    if (filters.status) {
      dataSource.value = dataSource.value.filter((item) => (filters.status === "enabled" ? item.enabled : !item.enabled));
    }
    pagination.total = dataSource.value.length;
  } finally {
    loading.value = false;
  }
};

const openSync = () => {
  syncOpen.value = true;
};

const submitSync = async () => {
  if (!syncLineId.value) {
    message.error("请先选择线路再同步镜像");
    return;
  }
  const cloudLineId = getCloudLineId(syncLineId.value);
  if (!cloudLineId) {
    message.error("无法解析线路 ID");
    return;
  }
  await syncSystemImages({ line_id: cloudLineId });
  message.success("已触发同步");
  syncOpen.value = false;
  fetchData();
};

const openCreate = () => {
  Object.assign(form, { id: null, image_id: "", name: "", type: "", enabled: true });
  formOpen.value = true;
};

const openEdit = (record) => {
  Object.assign(form, record);
  formOpen.value = true;
};

const submit = async () => {
  if (form.id) {
    await updateSystemImage(form.id, form);
  } else {
    await createSystemImage(form);
  }
  message.success("已保存镜像");
  formOpen.value = false;
  fetchData();
};

const remove = (record) => {
  Modal.confirm({
    title: "确认删除该镜像?",
    onOk: async () => {
      await deleteSystemImage(record.id);
      message.success("已删除");
      fetchData();
    }
  });
};

const bulkRemove = () => {
  Modal.confirm({
    title: `确认删除所选 ${selectedKeys.value.length} 个镜像?`,
    onOk: async () => {
      await bulkDeleteSystemImages(selectedKeys.value);
      selectedKeys.value = [];
      message.success("已批量删除");
      fetchData();
    }
  });
};

const openLineConfig = async () => {
  configOpen.value = true;
  if (syncLineId.value) configLineId.value = syncLineId.value;
  await loadAllImages();
  if (configLineId.value) {
    await loadLineImages(configLineId.value);
  }
};

const loadAllImages = async () => {
  const res = await listSystemImages();
  const payload = res.data || {};
  allImages.value = (payload.items || []).map((row) => ({
    id: row.id ?? row.ID,
    name: row.name ?? row.Name,
    type: row.type ?? row.Type
  }));
};

const loadLineImages = async (lineId) => {
  const cloudLineId = getCloudLineId(lineId);
  if (!cloudLineId) {
    configImageIds.value = [];
    return;
  }
  const res = await listSystemImages({ line_id: cloudLineId });
  const payload = res.data || {};
  configImageIds.value = (payload.items || []).map((row) => row.id ?? row.ID);
};

const submitLineConfig = async () => {
  if (!configLineId.value) {
    message.error("请选择线路");
    return;
  }
  await setLineSystemImages(configLineId.value, { image_ids: configImageIds.value });
  message.success("已保存线路镜像配置");
  configOpen.value = false;
};

watch(
  () => configLineId.value,
  async (val) => {
    if (!val) {
      configImageIds.value = [];
      return;
    }
    await loadLineImages(val);
  }
);

const loadLines = async () => {
  const res = await listLines();
  lines.value = (res.data?.items || []).map((row) => ({
    id: row.id ?? row.ID,
    name: row.name ?? row.Name ?? row.line_name ?? row.LineName,
    line_id: row.line_id ?? row.LineID
  }));
};

const getCloudLineId = (lineId) => {
  const match = lines.value.find((line) => Number(line.id) === Number(lineId));
  return match?.line_id ?? null;
};

const init = async () => {
  await loadLines();
  await fetchData();
};

init();
</script>
