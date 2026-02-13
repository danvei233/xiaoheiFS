<template>
  <div class="page">
    <div class="page-header">
      <div>
        <div class="page-title">用户管理</div>
        <div class="subtle">管理用户账号与状态</div>
      </div>
      <div class="page-header-actions">
        <a-button type="primary" @click="openCreate">创建用户</a-button>
      </div>
    </div>

    <FilterBar
      v-model:filters="filters"
      :status-options="statusOptions"
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
        <template v-if="column.key === 'avatar'">
          <a-avatar :src="record.avatar" :size="32">
            <template #icon><UserOutlined /></template>
          </a-avatar>
        </template>
        <template v-else-if="column.key === 'status'">
          <StatusTag :status="record.status" />
        </template>
        <template v-else-if="column.key === 'action'">
          <a-space>
            <a-button type="link" @click="openDetail(record)">详情</a-button>
            <a-button type="link" @click="openEdit(record)">编辑</a-button>
            <a-button type="link" @click="loginAsUser(record)">以此用户登录</a-button>
            <a-button type="link" @click="toggle(record)">禁用/启用</a-button>
            <a-button type="link" danger @click="openReset(record)">重置密码</a-button>
          </a-space>
        </template>
      </template>
    </ProTable>

    <a-modal v-model:open="createOpen" title="创建用户" @ok="create">
      <a-form layout="vertical">
        <a-form-item label="用户名">
          <a-input v-model:value="createForm.username" :maxlength="INPUT_LIMITS.USERNAME" />
        </a-form-item>
        <a-form-item label="邮箱">
          <a-input v-model:value="createForm.email" :maxlength="INPUT_LIMITS.EMAIL" />
        </a-form-item>
        <a-form-item label="密码">
          <a-input-password v-model:value="createForm.password" :maxlength="INPUT_LIMITS.PASSWORD" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="resetOpen" title="重置密码" @ok="submitReset">
      <a-input-password v-model:value="resetPassword" placeholder="输入新密码" :maxlength="INPUT_LIMITS.PASSWORD" />
    </a-modal>

    <a-modal v-model:open="editOpen" title="编辑用户" @ok="submitEdit">
      <a-form layout="vertical">
        <a-form-item label="用户名">
          <a-input v-model:value="editForm.username" :maxlength="INPUT_LIMITS.USERNAME" />
        </a-form-item>
        <a-form-item label="邮箱">
          <a-input v-model:value="editForm.email" :maxlength="INPUT_LIMITS.EMAIL" />
        </a-form-item>
        <a-form-item label="QQ">
          <a-input v-model:value="editForm.qq" :maxlength="INPUT_LIMITS.QQ" />
        </a-form-item>
        <a-form-item label="头像 URL">
          <a-input v-model:value="editForm.avatar" placeholder="可选" :maxlength="INPUT_LIMITS.URL" />
        </a-form-item>
        <a-form-item label="状态">
          <a-select v-model:value="editForm.status">
            <a-select-option value="active">active</a-select-option>
            <a-select-option value="blocked">blocked</a-select-option>
          </a-select>
        </a-form-item>
        <a-divider />
        <a-form-item label="实名认证状态">
          <a-space align="start" :size="12">
            <a-select v-model:value="realnameStatus" placeholder="选择实名状态" style="width: 160px">
              <a-select-option value="pending">待审核</a-select-option>
              <a-select-option value="verified">已通过</a-select-option>
              <a-select-option value="failed">未通过</a-select-option>
            </a-select>
            <a-input v-model:value="realnameReason" placeholder="审核备注（可选）" style="width: 220px" :maxlength="INPUT_LIMITS.REVIEW_REASON" />
            <a-button
              type="primary"
              :loading="realnameUpdating"
              :disabled="!realnameRecord"
              @click="submitRealnameStatus(editForm.id)"
            >
              更新实名状态
            </a-button>
          </a-space>
        </a-form-item>
      </a-form>
    </a-modal>

    <a-drawer v-model:open="detailOpen" width="720" title="用户详情">
      <a-space style="margin-bottom: 12px">
        <a-button type="primary" @click="loginAsUser(detail)" :disabled="detail?.status !== 'active'">以此用户登录</a-button>
      </a-space>
      <a-descriptions :column="2" bordered>
        <a-descriptions-item label="用户 ID">{{ detail?.id || '-' }}</a-descriptions-item>
        <a-descriptions-item label="用户名">{{ detail?.username || '-' }}</a-descriptions-item>
        <a-descriptions-item label="邮箱">{{ detail?.email || '-' }}</a-descriptions-item>
        <a-descriptions-item label="状态">{{ detail?.status || '-' }}</a-descriptions-item>
      </a-descriptions>

      <a-divider />

      <div class="section-title">实名认证</div>
      <a-space align="start" :size="12" style="margin-bottom: 12px">
        <a-select v-model:value="realnameStatus" placeholder="选择实名状态" style="width: 160px">
          <a-select-option value="pending">待审核</a-select-option>
          <a-select-option value="verified">已通过</a-select-option>
          <a-select-option value="failed">未通过</a-select-option>
        </a-select>
        <a-input
          v-model:value="realnameReason"
          placeholder="审核备注（可选）"
          style="width: 260px"
          :maxlength="INPUT_LIMITS.REVIEW_REASON"
        />
        <a-button type="primary" :loading="realnameUpdating" :disabled="!realnameRecord" @click="submitRealnameStatus(detail?.id)">
          更新实名状态
        </a-button>
      </a-space>
      <div class="subtle" v-if="!realnameRecord">暂无实名认证记录，无法修改状态</div>

      <a-divider />

      <div class="section-title">钱包余额</div>
      <div class="subtle">￥{{ formatAmount(walletInfo?.balance) }}</div>

      <a-divider />

      <div class="section-title">订单记录</div>
      <a-table
        :columns="orderColumns"
        :data-source="orderRecords"
        :pagination="false"
        size="small"
        row-key="id"
        :loading="detailLoading"
      />

      <a-divider />

      <div class="section-title">钱包记录</div>
      <a-table
        :columns="walletTxColumns"
        :data-source="walletTransactions"
        :pagination="false"
        size="small"
        row-key="id"
        :loading="detailLoading"
      />
    </a-drawer>
  </div>
</template>

<script setup>
import { reactive, ref } from "vue";
import { UserOutlined } from "@ant-design/icons-vue";
import FilterBar from "@/components/FilterBar.vue";
import ProTable from "@/components/ProTable.vue";
import StatusTag from "@/components/StatusTag.vue";
import {
  listAdminUsers,
  updateUserStatus,
  createAdminUser,
  resetUserPassword,
  getAdminUserDetail,
  updateAdminUser,
  updateAdminUserRealNameStatus,
  adminImpersonateUser,
  listAdminOrders,
  listRealNameRecords,
  getAdminWalletInfo,
  listAdminWalletTransactions
} from "@/services/admin";
import { message } from "ant-design-vue";
import { INPUT_LIMITS } from "@/constants/inputLimits";

const filters = reactive({ keyword: "", status: undefined, range: [] });
const statusOptions = [
  { label: "active", value: "active" },
  { label: "blocked", value: "blocked" }
];

const loading = ref(false);
const dataSource = ref([]);
const pagination = reactive({ current: 1, pageSize: 20, total: 0, showSizeChanger: true });

const createOpen = ref(false);
const resetOpen = ref(false);
const editOpen = ref(false);
const detailOpen = ref(false);
const createForm = reactive({ username: "", email: "", password: "" });
const resetPassword = ref("");
const activeRecord = ref(null);
const detail = ref(null);
const detailLoading = ref(false);
const walletInfo = ref(null);
const orderRecords = ref([]);
const walletTransactions = ref([]);
const realnameRecord = ref(null);
const realnameStatus = ref("");
const realnameReason = ref("");
const realnameUpdating = ref(false);
const editForm = reactive({
  id: null,
  username: "",
  email: "",
  qq: "",
  avatar: "",
  status: "active"
});

const columns = [
  { title: "用户 ID", dataIndex: "id", key: "id", width: 80 },
  { title: "头像", key: "avatar", width: 60 },
  { title: "用户名", dataIndex: "username", key: "username" },
  { title: "邮箱", dataIndex: "email", key: "email" },
  { title: "状态", dataIndex: "status", key: "status", width: 90 },
  { title: "注册时间", dataIndex: "created_at", key: "created_at", width: 170 },
  { title: "操作", key: "action", width: 320, fixed: "right" }
];

const normalize = (row) => ({
  id: row.id ?? row.ID,
  username: row.username ?? row.Username,
  email: row.email ?? row.Email,
  avatar: row.avatar ?? row.avatar_url ?? row.AvatarURL,
  status: row.status ?? row.Status,
  role: row.role ?? row.Role,
  created_at: row.created_at ?? row.CreatedAt
});

const formatAmount = (value) => Number(value || 0).toFixed(2);

const formatDate = (value) => {
  if (!value) return "-";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString("zh-CN");
};

const orderColumns = [
  {
    title: "订单号",
    dataIndex: "order_no",
    key: "order_no",
    width: 140,
    customRender: ({ record }) => record.order_no || record.id || "-"
  },
  { title: "状态", dataIndex: "status", key: "status", width: 120 },
  {
    title: "金额",
    dataIndex: "total_amount",
    key: "total_amount",
    width: 120,
    customRender: ({ text }) => `￥${formatAmount(text)}`
  },
  {
    title: "创建时间",
    dataIndex: "created_at",
    key: "created_at",
    width: 180,
    customRender: ({ text }) => formatDate(text)
  }
];

const walletTxColumns = [
  { title: "类型", dataIndex: "type", key: "type", width: 120 },
  {
    title: "金额",
    dataIndex: "amount",
    key: "amount",
    width: 120,
    customRender: ({ text }) => `￥${formatAmount(text)}`
  },
  { title: "备注", dataIndex: "note", key: "note" },
  {
    title: "时间",
    dataIndex: "created_at",
    key: "created_at",
    width: 180,
    customRender: ({ text }) => formatDate(text)
  }
];

const fetchData = async () => {
  loading.value = true;
  try {
    const res = await listAdminUsers({
      limit: pagination.pageSize,
      offset: (pagination.current - 1) * pagination.pageSize
    });
    const payload = res.data || {};
    dataSource.value = (payload.items || []).map(normalize).filter((item) => item.role !== "admin");
    if (filters.status) {
      dataSource.value = dataSource.value.filter((item) => item.status === filters.status);
    }
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
  const csv = "id,email,status\n" + dataSource.value.map((i) => `${i.id},${i.email},${i.status}`).join("\n");
  const blob = new Blob([csv], { type: "text/csv;charset=utf-8;" });
  const link = document.createElement("a");
  link.href = URL.createObjectURL(blob);
  link.download = "users.csv";
  link.click();
};

const toggle = async (record) => {
  if (record.role === "admin") {
    message.warning("管理员账号不支持在此处修改");
    return;
  }
  const status = record.status === "active" ? "blocked" : "active";
  await updateUserStatus(record.id, { status });
  message.success("已更新状态");
  fetchData();
};

const openCreate = () => {
  createForm.username = "";
  createForm.email = "";
  createForm.password = "";
  createOpen.value = true;
};

const create = async () => {
  if (String(createForm.username || "").length > INPUT_LIMITS.USERNAME) {
    message.error(`用户名长度不能超过 ${INPUT_LIMITS.USERNAME} 个字符`);
    return;
  }
  if (String(createForm.email || "").length > INPUT_LIMITS.EMAIL) {
    message.error(`邮箱长度不能超过 ${INPUT_LIMITS.EMAIL} 个字符`);
    return;
  }
  if (String(createForm.password || "").length > INPUT_LIMITS.PASSWORD) {
    message.error(`密码长度不能超过 ${INPUT_LIMITS.PASSWORD} 个字符`);
    return;
  }
  await createAdminUser({ ...createForm });
  message.success("用户已创建");
  createOpen.value = false;
  fetchData();
};

const openReset = (record) => {
  activeRecord.value = record;
  resetPassword.value = "";
  resetOpen.value = true;
};

const submitReset = async () => {
  if (!activeRecord.value) return;
  if (String(resetPassword.value || "").length > INPUT_LIMITS.PASSWORD) {
    message.error(`密码长度不能超过 ${INPUT_LIMITS.PASSWORD} 个字符`);
    return;
  }
  await resetUserPassword(activeRecord.value.id, { password: resetPassword.value });
  message.success("已重置密码");
  resetOpen.value = false;
};

const fetchUserExtras = async (userId) => {
  detailLoading.value = true;
  walletInfo.value = null;
  orderRecords.value = [];
  walletTransactions.value = [];
  realnameRecord.value = null;
  try {
    const [walletRes, ordersRes, walletTxRes, realnameRes] = await Promise.all([
      getAdminWalletInfo(userId),
      listAdminOrders({ user_id: userId, limit: 20, offset: 0 }),
      listAdminWalletTransactions(userId, { limit: 20, offset: 0 }),
      listRealNameRecords({ user_id: userId, limit: 1, offset: 0 })
    ]);
    walletInfo.value = walletRes.data?.wallet || null;
    orderRecords.value = ordersRes.data?.items || [];
    walletTransactions.value = walletTxRes.data?.items || [];
    realnameRecord.value = realnameRes.data?.items?.[0] || null;
    realnameStatus.value = realnameRecord.value?.status || "";
    realnameReason.value = realnameRecord.value?.reason || "";
  } catch (e) {
    message.error(e.response?.data?.error || "获取用户信息失败");
  } finally {
    detailLoading.value = false;
  }
};

const openDetail = async (record) => {
  if (record.role === "admin") {
    message.warning("管理员账号不支持在此处查看");
    return;
  }
  detailOpen.value = true;
  const res = await getAdminUserDetail(record.id);
  detail.value = res.data || null;
  await fetchUserExtras(record.id);
};

const openEdit = async (record) => {
  activeRecord.value = record;
  if (record.role === "admin") {
    message.warning("管理员账号不支持在此处修改");
    return;
  }
  editForm.id = record.id;
  editForm.username = record.username || "";
  editForm.email = record.email || "";
  editForm.qq = record.qq || "";
  editForm.avatar = record.avatar || "";
  editForm.status = record.status || "active";
  editOpen.value = true;
  await fetchUserExtras(record.id);
};

const loginAsUser = async (record) => {
  if (!record?.id) return;
  if (record.role === "admin") {
    message.warning("管理员账号不支持模拟登录");
    return;
  }
  try {
    const res = await adminImpersonateUser(record.id);
    const token = res.data?.access_token;
    if (!token) {
      message.error("未获取到用户令牌");
      return;
    }
    localStorage.setItem("user_token", token);
    window.open("/console", "_blank");
  } catch (e) {
    message.error(e.response?.data?.error || "模拟登录失败");
  }
};

const submitRealnameStatus = async (userId) => {
  const targetId = userId || detail.value?.id || editForm.id;
  if (!targetId) return;
  realnameUpdating.value = true;
  try {
    if (String(realnameReason.value || "").length > INPUT_LIMITS.REVIEW_REASON) {
      message.error(`审核备注长度不能超过 ${INPUT_LIMITS.REVIEW_REASON} 个字符`);
      return;
    }
    await updateAdminUserRealNameStatus(targetId, {
      status: realnameStatus.value,
      reason: realnameReason.value
    });
    message.success("实名状态已更新");
    await fetchUserExtras(targetId);
  } catch (e) {
    message.error(e.response?.data?.error || "更新实名状态失败");
  } finally {
    realnameUpdating.value = false;
  }
};

const submitEdit = async () => {
  if (!editForm.id) return;
  if (String(editForm.username || "").length > INPUT_LIMITS.USERNAME) {
    message.error(`用户名长度不能超过 ${INPUT_LIMITS.USERNAME} 个字符`);
    return;
  }
  if (String(editForm.email || "").length > INPUT_LIMITS.EMAIL) {
    message.error(`邮箱长度不能超过 ${INPUT_LIMITS.EMAIL} 个字符`);
    return;
  }
  if (String(editForm.qq || "").length > INPUT_LIMITS.QQ) {
    message.error(`QQ 长度不能超过 ${INPUT_LIMITS.QQ} 个字符`);
    return;
  }
  if (String(editForm.avatar || "").length > INPUT_LIMITS.URL) {
    message.error(`头像 URL 长度不能超过 ${INPUT_LIMITS.URL} 个字符`);
    return;
  }
  await updateAdminUser(editForm.id, {
    username: editForm.username,
    email: editForm.email,
    qq: editForm.qq,
    avatar: editForm.avatar,
    status: editForm.status
  });
  message.success("已更新用户资料");
  editOpen.value = false;
  fetchData();
};

fetchData();
</script>

