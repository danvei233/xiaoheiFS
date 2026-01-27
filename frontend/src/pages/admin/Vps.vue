<template>
  <div class="page">
    <div class="page-header">
      <div>
        <div class="page-title">VPS 管理</div>
        <div class="subtle">管理实例状态与生命周期</div>
      </div>
    </div>

    <FilterBar
      v-model:filters="filters"
      :status-options="statusOptions"
      :status-tabs="statusTabs"
      @search="fetchData"
      @refresh="fetchData"
      @reset="fetchData"
      @export="exportCsv"
    />
    <ProTable
      :columns="columns"
      :data-source="dataSource"
      :loading="loading"
      :pagination="pagination"
      selectable
      @change="onTableChange"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'status'">
          <VpsStatusTag :status="record.status" />
        </template>
        <template v-else-if="column.key === 'admin_status'">
          <StatusTag :status="record.admin_status" />
        </template>
        <template v-else-if="column.key === 'action'">
          <a-dropdown>
            <a class="subtle">操作</a>
            <template #overlay>
              <a-menu>
                <a-menu-item @click="openEdit(record)">编辑</a-menu-item>
                <a-menu-item @click="openStatus(record)">设置状态</a-menu-item>
                <a-menu-item @click="openExpire(record)">修改到期</a-menu-item>
                <a-menu-item @click="refresh(record)">刷新</a-menu-item>
                <a-menu-item @click="emergencyRenew(record)">紧急续费</a-menu-item>
                <a-menu-item @click="confirmAction('锁定该实例?', () => lock(record))">锁定</a-menu-item>
                <a-menu-item @click="confirmAction('解锁该实例?', () => unlock(record))">解锁</a-menu-item>
                <a-menu-item @click="openResize(record)">改配</a-menu-item>
                <a-menu-item danger @click="openDelete(record)">删除</a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
        </template>
      </template>
    </ProTable>

    <a-modal v-model:open="statusOpen" title="设置实例状态" @ok="submitStatus">
      <a-form layout="vertical">
        <a-form-item label="状态">
          <a-select v-model:value="statusForm.status">
            <a-select-option value="normal">normal</a-select-option>
            <a-select-option value="abuse">abuse</a-select-option>
            <a-select-option value="fraud">fraud</a-select-option>
            <a-select-option value="locked">locked</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="原因">
          <a-textarea v-model:value="statusForm.reason" rows="3" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="renewOpen" title="紧急续费" @ok="submitRenew">
      <a-form layout="vertical">
        <a-form-item label="执行策略">
          <div class="subtle">Use lifecycle settings for days, window, and cooldown.</div>
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="resizeOpen" title="改配" @ok="submitResize">
      <a-form layout="vertical">
        <a-row :gutter="12">
          <a-col :span="12"><a-form-item label="CPU">
              <a-input-number v-model:value="resizeForm.cpu" :min="0" style="width: 100%" />
            </a-form-item></a-col>
          <a-col :span="12"><a-form-item label="内存(GB)">
              <a-input-number v-model:value="resizeForm.memory_gb" :min="0" style="width: 100%" />
            </a-form-item></a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="12"><a-form-item label="磁盘(GB)">
              <a-input-number v-model:value="resizeForm.disk_gb" :min="0" style="width: 100%" />
            </a-form-item></a-col>
          <a-col :span="12"><a-form-item label="带宽(Mbps)">
              <a-input-number v-model:value="resizeForm.bandwidth_mbps" :min="0" style="width: 100%" />
            </a-form-item></a-col>
        </a-row>
      </a-form>
    </a-modal>

    <a-modal v-model:open="expireOpen" title="修改到期时间" @ok="submitExpire">
      <a-form layout="vertical">
        <a-form-item label="到期时间">
          <a-date-picker v-model:value="expireForm.expire_at" show-time style="width: 100%" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="deleteOpen" title="删除实例" @ok="submitDelete">
      <a-form layout="vertical">
        <a-form-item label="删除原因">
          <a-textarea v-model:value="deleteReason" rows="3" placeholder="可选，便于审计与自动退款" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="editOpen" title="编辑 VPS" width="640px" @ok="submitEdit">
      <a-form layout="vertical">
        <a-row :gutter="12">
          <a-col :span="12">
            <a-form-item label="同步模式">
              <a-select v-model:value="editForm.sync_mode">
                <a-select-option value="local">只修改本地</a-select-option>
                <a-select-option value="automation">同步到自动化</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="套餐 ID">
              <a-input-number v-model:value="editForm.package_id" :min="0" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="12">
            <a-form-item label="月费">
              <a-input-number v-model:value="editForm.monthly_price" :min="0" :precision="2" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="套餐名称">
              <a-input v-model:value="editForm.package_name" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="8">
            <a-form-item label="CPU (核)">
              <a-input-number v-model:value="editForm.cpu" :min="0" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="内存 (GB)">
              <a-input-number v-model:value="editForm.memory_gb" :min="0" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="磁盘 (GB)">
              <a-input-number v-model:value="editForm.disk_gb" :min="0" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="12">
            <a-form-item label="带宽 (Mbps)">
              <a-input-number v-model:value="editForm.bandwidth_mbps" :min="0" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="端口数">
              <a-input-number v-model:value="editForm.port_num" :min="0" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="12">
            <a-form-item label="地区">
              <a-input v-model:value="editForm.region" disabled />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="线路 ID">
              <a-input-number v-model:value="editForm.line_id" disabled style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="12">
            <a-form-item label="实例名称">
              <a-input v-model:value="editForm.name" disabled />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="自动化实例 ID">
              <a-input v-model:value="editForm.automation_instance_id" disabled />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="12">
            <a-form-item label="状态">
              <a-select v-model:value="editForm.status">
                <a-select-option value="running">运行中</a-select-option>
                <a-select-option value="stopped">已关机</a-select-option>
                <a-select-option value="provisioning">开通中</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="管理状态">
              <a-select v-model:value="editForm.admin_status">
                <a-select-option value="normal">normal</a-select-option>
                <a-select-option value="abuse">abuse</a-select-option>
                <a-select-option value="fraud">fraud</a-select-option>
                <a-select-option value="locked">locked</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="系统镜像 ID">
          <a-input-number v-model:value="editForm.system_id" :min="0" style="width: 100%" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { reactive, ref } from "vue";
import FilterBar from "@/components/FilterBar.vue";
import ProTable from "@/components/ProTable.vue";
import StatusTag from "@/components/StatusTag.vue";
import VpsStatusTag from "@/components/VpsStatusTag.vue";
import {
  listAdminVps,
  lockAdminVps,
  unlockAdminVps,
  deleteAdminVps,
  resizeAdminVps,
  refreshAdminVps,
  updateAdminVps,
  updateAdminVpsStatus,
  emergencyRenewAdminVps,
  updateAdminVpsExpire
} from "@/services/admin";
import { message, Modal } from "ant-design-vue";

const filters = reactive({ keyword: "", status: undefined, range: [] });
const statusOptions = [
  { label: "normal", value: "normal" },
  { label: "abuse", value: "abuse" },
  { label: "fraud", value: "fraud" },
  { label: "locked", value: "locked" }
];

const statusTabs = [
  { label: "normal", value: "normal" },
  { label: "abuse", value: "abuse" },
  { label: "fraud", value: "fraud" }
];

const loading = ref(false);
const dataSource = ref([]);
const pagination = reactive({ current: 1, pageSize: 20, total: 0, showSizeChanger: true });

const statusOpen = ref(false);
const renewOpen = ref(false);
const resizeOpen = ref(false);
const expireOpen = ref(false);
const deleteOpen = ref(false);
const editOpen = ref(false);
const activeRecord = ref(null);
const statusForm = reactive({ status: "normal", reason: "" });
const resizeForm = reactive({ cpu: 0, memory_gb: 0, disk_gb: 0, bandwidth_mbps: 0 });
const expireForm = reactive({ expire_at: null });
const deleteReason = ref("");
const editForm = reactive({
  sync_mode: "local",
  package_id: 0,
  monthly_price: 0,
  package_name: "",
  cpu: 0,
  memory_gb: 0,
  disk_gb: 0,
  bandwidth_mbps: 0,
  port_num: 0,
  status: "running",
  admin_status: "normal",
  system_id: 0,
  region: "",
  line_id: 0,
  name: "",
  automation_instance_id: ""
});

const columns = [
  { title: "实例 ID", dataIndex: "id", key: "id" },
  { title: "用户", dataIndex: "user_id", key: "user_id" },
  { title: "地区", dataIndex: "region", key: "region" },
  { title: "套餐", dataIndex: "package_name", key: "package_name" },
  { title: "月费", dataIndex: "monthly_price", key: "monthly_price" },
  { title: "状态", dataIndex: "status", key: "status" },
  { title: "管理状态", dataIndex: "admin_status", key: "admin_status" },
  { title: "到期时间", dataIndex: "expire_at", key: "expire_at" },
  { title: "操作", key: "action" }
];

const statusFromAutomation = (state) => {
  switch (Number(state)) {
    case 1:
    case 13:
      return "provisioning";
    case 2:
      return "running";
    case 3:
      return "stopped";
    case 4:
      return "reinstalling";
    case 5:
      return "reinstall_failed";
    case 10:
      return "locked";
    case 11:
      return "failed";
    case 12:
      return "deleting";
    default:
      return "";
  }
};

const normalize = (row) => {
  const rawStatus = row.status ?? row.Status ?? "";
  const rawAutomationState = row.automation_state ?? row.AutomationState ?? null;
  const resolvedStatus =
    rawAutomationState !== null && rawAutomationState !== undefined
      ? statusFromAutomation(rawAutomationState)
      : rawStatus;
  return {
    id: row.id ?? row.ID,
    user_id: row.user_id ?? row.UserID,
    region: row.region ?? row.Region,
    region_id: row.region_id ?? row.RegionID ?? 0,
    line_id: row.line_id ?? row.LineID ?? 0,
    package_id: row.package_id ?? row.PackageID ?? 0,
    status: resolvedStatus,
    admin_status: row.admin_status ?? row.AdminStatus,
    expire_at: row.expire_at ?? row.ExpireAt,
    monthly_price: row.monthly_price ?? row.MonthlyPrice ?? 0,
    package_name: row.package_name ?? row.PackageName ?? "",
    cpu: row.cpu ?? row.CPU ?? 0,
    memory_gb: row.memory_gb ?? row.MemoryGB ?? 0,
    disk_gb: row.disk_gb ?? row.DiskGB ?? 0,
    bandwidth_mbps: row.bandwidth_mbps ?? row.BandwidthMB ?? 0,
    port_num: row.port_num ?? row.PortNum ?? 0,
    system_id: row.system_id ?? row.SystemID ?? 0,
    name: row.name ?? row.Name ?? "",
    automation_instance_id: row.automation_instance_id ?? row.AutomationInstanceID ?? ""
  };
};

const fetchData = async () => {
  loading.value = true;
  try {
    const res = await listAdminVps({
      limit: pagination.pageSize,
      offset: (pagination.current - 1) * pagination.pageSize
    });
    const payload = res.data || {};
    let items = (payload.items || []).map(normalize);
    if (filters.status) {
      items = items.filter((item) => item.admin_status === filters.status);
    }
    dataSource.value = items;
    pagination.total = payload.total || dataSource.value.length;
  } finally {
    loading.value = false;
  }
};

const onTableChange = (pager) => {
  pagination.current = pager.current;
  pagination.pageSize = pager.pageSize;
  fetchData();
};

const exportCsv = () => {
  const csv = "id,status\n" + dataSource.value.map((i) => `${i.id},${i.status}`).join("\n");
  const blob = new Blob([csv], { type: "text/csv;charset=utf-8;" });
  const link = document.createElement("a");
  link.href = URL.createObjectURL(blob);
  link.download = "admin-vps.csv";
  link.click();
};

const confirmAction = (title, action) => {
  Modal.confirm({
    title,
    onOk: action
  });
};

const lock = async (record) => {
  await lockAdminVps(record.id);
  message.success("已锁定");
  fetchData();
};

const unlock = async (record) => {
  await unlockAdminVps(record.id);
  message.success("已解锁");
  fetchData();
};

const remove = async (record) => {
  await deleteAdminVps(record.id, { reason: deleteReason.value });
  message.success("已删除");
  fetchData();
};

const refresh = async (record) => {
  await refreshAdminVps(record.id);
  message.success("已刷新");
  fetchData();
};

const openResize = (record) => {
  activeRecord.value = record;
  resizeForm.cpu = 0;
  resizeForm.memory_gb = 0;
  resizeForm.disk_gb = 0;
  resizeForm.bandwidth_mbps = 0;
  resizeOpen.value = true;
};

const submitResize = async () => {
  if (!activeRecord.value) return;
  await resizeAdminVps(activeRecord.value.id, {
    cpu: resizeForm.cpu,
    memory_gb: resizeForm.memory_gb,
    disk_gb: resizeForm.disk_gb,
    bandwidth_mbps: resizeForm.bandwidth_mbps
  });
  resizeOpen.value = false;
  message.success("已提交改配");
};

const openStatus = (record) => {
  activeRecord.value = record;
  statusForm.status = record.admin_status || "normal";
  statusForm.reason = "";
  statusOpen.value = true;
};

const submitStatus = async () => {
  if (!activeRecord.value) return;
  await updateAdminVpsStatus(activeRecord.value.id, {
    status: statusForm.status,
    reason: statusForm.reason
  });
  try {
    await refreshAdminVps(activeRecord.value.id);
  } catch (error) {
    message.warning(error.response?.data?.error || "同步状态失败");
  }
  statusOpen.value = false;
  message.success("已更新状态");
  fetchData();
};

const emergencyRenew = (record) => {
  activeRecord.value = record;
  renewOpen.value = true;
};

const submitRenew = async () => {
  if (!activeRecord.value) return;
  await emergencyRenewAdminVps(activeRecord.value.id, {});
  renewOpen.value = false;
  message.success("已触发紧急续费");
  fetchData();
};

const openExpire = (record) => {
  activeRecord.value = record;
  expireForm.expire_at = null;
  expireOpen.value = true;
};

const submitExpire = async () => {
  if (!activeRecord.value || !expireForm.expire_at) return;
  const value = expireForm.expire_at?.toISOString ? expireForm.expire_at.toISOString() : expireForm.expire_at;
  await updateAdminVpsExpire(activeRecord.value.id, { expire_at: value });
  expireOpen.value = false;
  message.success("已修改到期时间");
  fetchData();
};

const openDelete = (record) => {
  activeRecord.value = record;
  deleteReason.value = "";
  deleteOpen.value = true;
};

const submitDelete = async () => {
  if (!activeRecord.value) return;
  await remove(activeRecord.value);
  deleteOpen.value = false;
};

const openEdit = (record) => {
  activeRecord.value = record;
  editForm.sync_mode = "local";
  editForm.package_id = record.package_id ?? record.PackageID ?? 0;
  editForm.monthly_price = record.monthly_price ?? record.MonthlyPrice ?? 0;
  editForm.package_name = record.package_name ?? record.PackageName ?? "";
  editForm.cpu = record.cpu ?? record.CPU ?? 0;
  editForm.memory_gb = record.memory_gb ?? record.MemoryGB ?? 0;
  editForm.disk_gb = record.disk_gb ?? record.DiskGB ?? 0;
  editForm.bandwidth_mbps = record.bandwidth_mbps ?? record.BandwidthMB ?? 0;
  editForm.port_num = record.port_num ?? record.PortNum ?? 0;
  editForm.status = record.status ?? record.Status ?? "running";
  editForm.admin_status = record.admin_status ?? record.AdminStatus ?? "normal";
  editForm.system_id = record.system_id ?? record.SystemID ?? 0;
  editForm.region = record.region ?? record.Region ?? "";
  editForm.line_id = record.line_id ?? record.LineID ?? 0;
  editForm.name = record.name ?? record.Name ?? "";
  editForm.automation_instance_id = record.automation_instance_id ?? record.AutomationInstanceID ?? "";
  editOpen.value = true;
};

const submitEdit = async () => {
  if (!activeRecord.value) return;
  const payload = {
    sync_mode: editForm.sync_mode,
    package_id: editForm.package_id || undefined,
    monthly_price: editForm.monthly_price,
    package_name: editForm.package_name || undefined,
    cpu: editForm.cpu,
    memory_gb: editForm.memory_gb,
    disk_gb: editForm.disk_gb,
    bandwidth_mbps: editForm.bandwidth_mbps,
    port_num: editForm.port_num,
    status: editForm.status,
    admin_status: editForm.admin_status,
    system_id: editForm.system_id || undefined
  };
  await updateAdminVps(activeRecord.value.id, payload);
  editOpen.value = false;
  message.success("已更新 VPS");
  fetchData();
};

fetchData();
</script>
