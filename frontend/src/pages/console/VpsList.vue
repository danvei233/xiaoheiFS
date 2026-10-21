
<template>
  <div class="page vps-list-page">
    <div class="page-header-gradient">
      <div class="page-header-content">
        <div class="page-title-section">
          <div class="page-title-wrapper">
            <CloudServerOutlined class="page-title-icon" />
            <div>
              <h1 class="page-title">VPS 实例</h1>
              <p class="page-desc">管理你的云主机资源与生命周期</p>
            </div>
          </div>
        </div>
        <div class="page-header-actions">
          <a-button @click="fetchData" class="action-btn">
            <template #icon><ReloadOutlined /></template>
            刷新
          </a-button>
          <a-button type="primary" @click="goBuy" class="action-btn-primary">
            <template #icon><PlusOutlined /></template>
            购买 VPS
          </a-button>
        </div>
      </div>
    </div>

    <div class="stats-container">
      <div class="stat-card stat-running">
        <div class="stat-icon-wrapper">
          <PlayCircleOutlined class="stat-icon" />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.running }}</div>
          <div class="stat-label">运行中</div>
        </div>
        <div class="stat-decoration"></div>
      </div>
      <div class="stat-card stat-stopped">
        <div class="stat-icon-wrapper">
          <StopOutlined class="stat-icon" />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.stopped }}</div>
          <div class="stat-label">已关机</div>
        </div>
        <div class="stat-decoration"></div>
      </div>
      <div class="stat-card stat-provisioning">
        <div class="stat-icon-wrapper">
          <SyncOutlined spin class="stat-icon" />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.provisioning }}</div>
          <div class="stat-label">开通中</div>
        </div>
        <div class="stat-decoration"></div>
      </div>
      <div class="stat-card stat-expiring">
        <div class="stat-icon-wrapper">
          <ClockCircleOutlined class="stat-icon" />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.expiring }}</div>
          <div class="stat-label">即将到期</div>
        </div>
        <div class="stat-decoration"></div>
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
    >
      <template #advanced>
        <a-space direction="vertical" style="width: 260px">
          <a-input v-model:value="filters.region" placeholder="地区">
            <template #prefix><GlobalOutlined /></template>
          </a-input>
          <a-input v-model:value="filters.expire_days" placeholder="到期天数">
            <template #prefix><CalendarOutlined /></template>
          </a-input>
        </a-space>
      </template>
    </FilterBar>

    <ProTable
      :columns="columns"
      :data-source="dataSource"
      :loading="loading"
      :pagination="pagination"
      selectable
      @change="onTableChange"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'name'">
          <div class="name-cell">
            <CloudOutlined class="name-icon" />
            <div>
              <div class="name-text">{{ record.name || "-" }}</div>
              <div class="name-id">ID: {{ record.id }}</div>
            </div>
          </div>
        </template>
        <template v-else-if="column.key === 'regionLine'">
          <span>{{ record.regionLine }}</span>
        </template>
        <template v-else-if="column.key === 'spec'">
          <a-tag color="blue">{{ record.specText }}</a-tag>
        </template>
        <template v-else-if="column.key === 'ip'">
          <a-typography-text copyable>{{ record.ip }}</a-typography-text>
        </template>
        <template v-else-if="column.key === 'status'">
          <VpsStatusTag :status="record.status" />
        </template>
        <template v-else-if="column.key === 'expire_at'">
          <div>
            <span :class="{ expiring: isExpiring(record.expire_at) }">{{ formatLocalDateTime(record.expire_at) }}</span>
            <div v-if="record.destroy_in_days !== undefined && record.destroy_in_days !== null" class="destroy-hint">
              Auto delete in {{ record.destroy_in_days }} days
            </div>
          </div>
        </template>
        <template v-else-if="column.key === 'action'">
          <a-space :size="4">
            <a-tooltip title="详情">
              <a-button type="text" size="small" @click="goDetail(record)">
                <template #icon><EyeOutlined /></template>
              </a-button>
            </a-tooltip>
            <a-tooltip title="控制面板">
              <a-button type="text" size="small" @click="openPanel(record)">
                <template #icon><ControlOutlined /></template>
              </a-button>
            </a-tooltip>
            <a-tooltip v-if="emergencyRenewEligible(record)" title="紧急续费">
              <a-button type="primary" size="small" danger class="urgent-renew-btn" @click="submitEmergencyRenew(record)">
                紧急续费
              </a-button>
            </a-tooltip>
            <a-dropdown>
              <a-button type="text" size="small">
                <template #icon><MoreOutlined /></template>
              </a-button>
              <template #overlay>
                <a-menu>
                  <a-menu-item key="start" @click="start(record)">开机</a-menu-item>
                  <a-menu-item key="shutdown" @click="shutdown(record)">关机</a-menu-item>
                  <a-menu-item key="reboot" @click="reboot(record)">重启</a-menu-item>
                  <a-menu-divider />
                  <a-menu-item key="renew" @click="openRenew(record)">续费</a-menu-item>
                  <a-menu-item v-if="emergencyRenewEligible(record)" key="urgent-renew" @click="submitEmergencyRenew(record)">
                    紧急续费
                  </a-menu-item>
                  <a-menu-item key="resize" :disabled="isExpired(record)" @click="openResize(record)">升降配</a-menu-item>
                  <a-menu-item key="refresh" @click="refresh(record)">刷新状态</a-menu-item>
                  <a-menu-divider />
                  <a-menu-item key="refund" danger @click="openRefund(record)">申请退款</a-menu-item>
                </a-menu>
              </template>
            </a-dropdown>
          </a-space>
        </template>
      </template>

      <template #mobile>
        <div class="vps-mobile-list">
          <div v-for="item in dataSource" :key="item.id" class="vps-mobile-card" @click="goDetail(item)">
            <div class="vps-card-header">
              <div class="vps-card-title">
                <CloudOutlined class="vps-card-icon" />
                <span class="vps-card-name">{{ item.name || "-" }}</span>
              </div>
              <VpsStatusTag :status="item.status" />
            </div>
            <div class="vps-card-body">
              <div class="vps-info-row">
                <span class="vps-info-label">
                  <IdcardOutlined class="info-icon" />
                  ID
                </span>
                <span class="vps-info-value">{{ item.id }}</span>
              </div>
              <div class="vps-info-row">
                <span class="vps-info-label">
                  <GlobalOutlined class="info-icon" />
                  地区
                </span>
                <span class="vps-info-value">{{ item.regionLine }}</span>
              </div>
              <div class="vps-info-row">
                <span class="vps-info-label">
                  <ApartmentOutlined class="info-icon" />
                  配置
                </span>
                <a-tag size="small" color="blue" class="vps-spec-tag">{{ item.specText }}</a-tag>
              </div>
              <div class="vps-info-row">
                <span class="vps-info-label">
                  <EnvironmentOutlined class="info-icon" />
                  IP
                </span>
                <a-typography-text class="vps-info-value" copyable>{{ item.ip }}</a-typography-text>
              </div>
              <div class="vps-info-row">
                <span class="vps-info-label">
                  <CalendarOutlined class="info-icon" />
                  到期
                </span>
                <span class="vps-info-value" :class="{ expiring: isExpiring(item.expire_at) }">
                  {{ formatLocalDateTime(item.expire_at) }}
                </span>
              </div>
              <div v-if="item.destroy_in_days !== undefined && item.destroy_in_days !== null" class="vps-info-row vps-destroy-row">
                <WarningOutlined class="info-icon warning" />
                <span class="vps-info-value destroy-hint">
                  将在 {{ item.destroy_in_days }} 天后自动删除
                </span>
              </div>
            </div>
            <div class="vps-card-actions">
              <a-tooltip title="详情">
                <div class="vps-action-btn" @click.stop="goDetail(item)">
                  <EyeOutlined />
                </div>
              </a-tooltip>
              <a-tooltip title="控制面板">
                <div class="vps-action-btn" @click.stop="openPanel(item)">
                  <ControlOutlined />
                </div>
              </a-tooltip>
              <a-tooltip v-if="emergencyRenewEligible(item)" title="紧急续费">
                <div class="vps-action-btn vps-action-urgent" @click.stop="submitEmergencyRenew(item)">
                  <CalendarOutlined />
                </div>
              </a-tooltip>
              <a-tooltip title="VNC">
                <div class="vps-action-btn" @click.stop="openVnc(item)">
                  <DesktopOutlined />
                </div>
              </a-tooltip>
              <a-tooltip title="更多操作">
                <div class="vps-action-btn vps-action-more" @click.stop="showMobileActions(item)">
                  <MoreOutlined />
                </div>
              </a-tooltip>
            </div>
          </div>
        </div>
      </template>
    </ProTable>

    <a-modal v-model:open="renewOpen" title="续费 VPS" :width="500" @ok="submitRenew" :confirm-loading="renewing">
      <a-alert
        message="续费说明"
        description="续费将延长 VPS 使用期限，请在到期前完成续费以避免服务中断。"
        type="info"
        show-icon
        style="margin-bottom: 20px"
      />
      <a-form layout="vertical">
        <a-form-item label="续费时长">
          <a-input-number v-model:value="renewForm.months" :min="1" :max="120" style="width: 100%" size="large">
            <template #addonAfter>个月</template>
          </a-input-number>
        </a-form-item>
        <a-alert
          message="金额自动计算"
          :description="`系统将根据月费 ￥${activeRecord?.monthly_price || 0} × ${renewForm.months} 个月自动计算`"
          type="info"
          show-icon
        />
      </a-form>
    </a-modal>

    <a-modal v-model:open="resizeOpen" title="升降配 VPS" :width="560" @ok="submitResize" :confirm-loading="resizing" :ok-button-props="{ disabled: isExpired(activeRecord) }">
      <a-form layout="vertical">
        <a-form-item label="目标套餐">
          <a-select v-model:value="resizeForm.target_package_id" placeholder="选择套餐" size="large" :disabled="!resizeEnabled">
              <a-select-option v-for="pkg in packageOptions" :key="pkg.id" :value="pkg.id">
                {{ pkg.name }} (￥{{ Number(pkg.monthly_price || 0).toFixed(2) }}/月)
              </a-select-option>
            </a-select>
          </a-form-item>
        <a-alert
          v-if="isSameTargetSelection"
          type="warning"
          show-icon
          message="不能选择当前套餐"
          style="margin-bottom: 12px"
        />
        <a-alert
          v-if="!resizeEnabled"
          type="warning"
          show-icon
          message="升降配功能已关闭"
          style="margin-bottom: 12px"
        />
        <a-alert
          v-if="isExpired(activeRecord)"
          type="warning"
          show-icon
          message="已到期实例不支持升降配"
          style="margin-bottom: 12px"
        />
        <a-form-item label="执行时间">
          <a-radio-group v-model:value="resizeForm.schedule_mode">
            <a-radio value="now">立即执行</a-radio>
            <a-radio value="later">指定时间</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item v-if="resizeForm.schedule_mode === 'later'" label="指定时间">
          <a-date-picker v-model:value="resizeForm.scheduled_at" show-time style="width: 100%" placeholder="选择执行时间" />
        </a-form-item>
        <a-form-item>
          <a-switch v-model:checked="resizeForm.reset_addons" />
          <span style="margin-left: 8px">清零附加项（本单差额直接体现）</span>
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="CPU 附加">
              <a-input-number v-model:value="resizeForm.add_cores" :min="addonMin.add_cores" :max="addonMax.add_cores" :step="addonStep.add_cores" :disabled="resizeForm.reset_addons" style="width: 100%" size="large">
                <template #addonAfter>核</template>
              </a-input-number>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="内存 附加">
              <a-input-number v-model:value="resizeForm.add_mem_gb" :min="addonMin.add_mem_gb" :max="addonMax.add_mem_gb" :step="addonStep.add_mem_gb" :disabled="resizeForm.reset_addons" style="width: 100%" size="large">
                <template #addonAfter>GB</template>
              </a-input-number>
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="磁盘 附加">
              <a-input-number v-model:value="resizeForm.add_disk_gb" :min="addonMin.add_disk_gb" :max="addonMax.add_disk_gb" :step="addonStep.add_disk_gb" :disabled="resizeForm.reset_addons" style="width: 100%" size="large">
                <template #addonAfter>GB</template>
              </a-input-number>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="带宽 附加">
              <a-input-number v-model:value="resizeForm.add_bw_mbps" :min="addonMin.add_bw_mbps" :max="addonMax.add_bw_mbps" :step="addonStep.add_bw_mbps" :disabled="resizeForm.reset_addons" style="width: 100%" size="large">
                <template #addonAfter>Mbps</template>
              </a-input-number>
            </a-form-item>
          </a-col>
        </a-row>
        <a-alert
          v-if="resizeQuoteLoading"
          type="info"
          show-icon
          message="正在计算价格..."
          style="margin-top: 8px"
        />
        <a-alert
          v-else-if="resizeQuoteError"
          type="error"
          show-icon
          :message="resizeQuoteError"
          style="margin-top: 8px"
        />
        <a-alert
          v-else-if="resizeQuote"
          type="success"
          show-icon
          style="margin-top: 8px"
        >
          <template #message>
            <span v-if="resizeQuoteAmount >= 0">
              本周期需支付：￥{{ resizeQuoteAmount.toFixed(2) }}
            </span>
            <span v-else>
              本周期差额：-￥{{ Math.abs(resizeQuoteAmount).toFixed(2) }}
            </span>
          </template>
          <template #description>
            <span v-if="resizeQuoteAmount < 0">退款方式以支付渠道为准。</span>
            <span v-else>金额以系统最终计算为准。</span>
          </template>
        </a-alert>
      </a-form>
    </a-modal>

    <a-modal v-model:open="refundOpen" title="申请退款" :width="500" @ok="submitRefund" :confirm-loading="refunding">
      <a-alert
        message="退款说明"
        description="提交退款申请后，管理员将进行审核。审核通过后，退款将退回余额。"
        type="warning"
        show-icon
        style="margin-bottom: 20px"
      />
      <a-form layout="vertical">
        <a-form-item label="退款原因" required>
          <a-textarea
            v-model:value="refundReason"
            :rows="4"
            placeholder="请详细说明退款原因..."
            :maxlength="INPUT_LIMITS.REFUND_REASON"
            show-count
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-drawer v-model:open="mobileActionsOpen" title="操作" placement="bottom" :height="320">
      <a-grid :col="4" :gutter="[12, 12]">
        <a-grid-item><div class="action-grid-item" @click="handleMobileAction('detail')"><EyeOutlined class="action-icon" /><span>详情</span></div></a-grid-item>
        <a-grid-item><div class="action-grid-item" @click="handleMobileAction('panel')"><ControlOutlined class="action-icon" /><span>面板</span></div></a-grid-item>
        <a-grid-item><div class="action-grid-item" @click="handleMobileAction('vnc')"><DesktopOutlined class="action-icon" /><span>VNC</span></div></a-grid-item>
        <a-grid-item><div class="action-grid-item" @click="handleMobileAction('start')"><PlayCircleOutlined class="action-icon start" /><span>开机</span></div></a-grid-item>
        <a-grid-item><div class="action-grid-item" @click="handleMobileAction('shutdown')"><StopOutlined class="action-icon stop" /><span>关机</span></div></a-grid-item>
        <a-grid-item><div class="action-grid-item" @click="handleMobileAction('reboot')"><ReloadOutlined class="action-icon reboot" /><span>重启</span></div></a-grid-item>
        <a-grid-item><div class="action-grid-item" @click="handleMobileAction('renew')"><CalendarOutlined class="action-icon renew" /><span>续费</span></div></a-grid-item>
        <a-grid-item v-if="emergencyRenewEligible(mobileActionRecord)"><div class="action-grid-item danger" @click="handleMobileAction('urgent-renew')"><CalendarOutlined class="action-icon renew" /><span>紧急续费</span></div></a-grid-item>
        <a-grid-item><div class="action-grid-item" @click="handleMobileAction('resize')"><ApartmentOutlined class="action-icon resize" /><span>升降配</span></div></a-grid-item>
        <a-grid-item><div class="action-grid-item" @click="handleMobileAction('refresh')"><SyncOutlined class="action-icon" /><span>刷新</span></div></a-grid-item>
        <a-grid-item><div class="action-grid-item danger" @click="handleMobileAction('refund')"><RollbackOutlined class="action-icon" /><span>退款</span></div></a-grid-item>
      </a-grid>
    </a-drawer>
  </div>
</template>
<script setup>
import { reactive, ref, computed, watch } from "vue";
import FilterBar from "@/components/FilterBar.vue";
import ProTable from "@/components/ProTable.vue";
import VpsStatusTag from "@/components/VpsStatusTag.vue";
import { useRouter } from "vue-router";
import { useVpsStore } from "@/stores/vps";
import { useCatalogStore } from "@/stores/catalog";
import { useAuthStore } from "@/stores/auth";
import { useSiteStore } from "@/stores/site";
import {
  createVpsRenewOrder,
  emergencyRenewVps,
  createVpsResizeOrder,
  quoteVpsResizeOrder,
  rebootVps,
  shutdownVps,
  startVps,
  requestVpsRefund
} from "@/services/user";
import { INPUT_LIMITS } from "@/constants/inputLimits";
import { message, Modal } from "ant-design-vue";
import dayjs from "dayjs";
import {
  PlusOutlined,
  ReloadOutlined,
  CloudOutlined,
  CloudServerOutlined,
  EnvironmentOutlined,
  GlobalOutlined,
  CalendarOutlined,
  EyeOutlined,
  ControlOutlined,
  DesktopOutlined,
  MoreOutlined,
  PlayCircleOutlined,
  StopOutlined,
  ApartmentOutlined,
  SyncOutlined,
  RollbackOutlined,
  ClockCircleOutlined,
  WarningOutlined,
  IdcardOutlined
} from "@ant-design/icons-vue";

const router = useRouter();
const store = useVpsStore();
const auth = useAuthStore();
const catalog = useCatalogStore();
const site = useSiteStore();

const filters = reactive({
  keyword: "",
  status: undefined,
  range: [],
  region: "",
  expire_days: ""
});

const statusOptions = [
  { label: "运行中", value: "running" },
  { label: "已关机", value: "stopped" },
  { label: "锁定", value: "locked" },
  { label: "已到期", value: "expired_locked" },
  { label: "开通中", value: "provisioning" },
  { label: "重装系统中", value: "reinstalling" },
  { label: "重装系统失败", value: "reinstall_failed" },
  { label: "创建失败", value: "failed" },
  { label: "删除中", value: "deleting" }
];

const statusTabs = [
  { label: "运行中", value: "running" },
  { label: "已关机", value: "stopped" },
  { label: "锁定", value: "locked" },
  { label: "已到期", value: "expired_locked" },
  { label: "重装中", value: "reinstalling" }
];

const loading = computed(() => store.loading);
const pagination = reactive({ current: 1, pageSize: 10, total: 0, showSizeChanger: true });

const columns = [
  { title: "实例", dataIndex: "name", key: "name", width: 200 },
  { title: "地区/线路", dataIndex: "regionLine", key: "regionLine", width: 150 },
  { title: "配置", dataIndex: "spec", key: "spec", width: 200 },
  { title: "IP地址", dataIndex: "ip", key: "ip", width: 140 },
  { title: "状态", dataIndex: "status", key: "status", width: 100 },
  { title: "到期时间", dataIndex: "expire_at", key: "expire_at", width: 170 },
  { title: "操作", key: "action", width: 160, fixed: "right" }
];

const parseJson = (input) => {
  if (!input) return {};
  if (typeof input === "string") {
    try {
      return JSON.parse(input);
    } catch {
      return {};
    }
  }
  return input;
};

const getSettingBool = (key) => {
  const raw = site.settings?.[key];
  if (raw === undefined || raw === null || raw === "") return undefined;
  if (typeof raw === "boolean") return raw;
  const normalized = String(raw).trim().toLowerCase();
  if (["true", "1", "yes", "on"].includes(normalized)) return true;
  if (["false", "0", "no", "off"].includes(normalized)) return false;
  return undefined;
};

const normalizeSpec = (spec, fallback = {}) => {
  if (!spec) spec = fallback;
  if (!spec) return "-";
  if (typeof spec === "string") {
    try {
      return normalizeSpec(JSON.parse(spec), fallback);
    } catch {
      return spec;
    }
  }
  const cpu = spec.cpu ?? spec.cores ?? spec.CPU ?? spec.Cores ?? fallback.cpu ?? fallback.CPU ?? 0;
  const mem = spec.memory_gb ?? spec.mem_gb ?? spec.MemoryGB ?? fallback.memory_gb ?? fallback.MemoryGB ?? 0;
  const disk = spec.disk_gb ?? spec.DiskGB ?? fallback.disk_gb ?? fallback.DiskGB ?? 0;
  const bw = spec.bandwidth_mbps ?? spec.BandwidthMB ?? spec.bandwidth ?? fallback.bandwidth_mbps ?? fallback.BandwidthMB ?? null;
  const parts = [`CPU ${cpu}核`, `内存 ${mem}G`, `磁盘 ${disk}G`];
  if (bw != null) parts.push(`带宽 ${bw}M`);
  return parts.join(" / ");
};

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

const isExpired = (row) => {
  const expireAt = row?.expire_at ?? row?.ExpireAt;
  if (!expireAt) return false;
  const expire = new Date(expireAt).getTime();
  if (Number.isNaN(expire)) return false;
  return expire <= Date.now();
};

const shouldShowExpiredLocked = (row, status) => {
  if (!isExpired(row)) return false;
  return status === "locked" || status === "expired_locked";
};

const dataSource = computed(() => {
  let items = store.items.map((row) => {
    const access = parseJson(row.access_info ?? row.AccessInfo ?? row.access_info_json ?? row.AccessInfoJSON);
    const ip = access.remote_ip ?? access.ip ?? access.public_ip ?? access.ipv4 ?? access.Ip ?? "-";
    const region = row.region ?? row.Region ?? "-";
    const line = row.line ?? row.Line ?? row.line_name ?? row.LineName ?? access.line ?? "";
    const rawStatus = row.status ?? row.Status ?? "";
    const rawAutomationState = row.automation_state ?? row.AutomationState ?? null;
    const baseStatus =
      rawAutomationState !== null && rawAutomationState !== undefined
        ? statusFromAutomation(rawAutomationState)
        : rawStatus;
    const resolvedStatus = shouldShowExpiredLocked(row, baseStatus) ? "expired_locked" : baseStatus;
      return {
        id: row.id ?? row.ID,
        name: row.name ?? row.Name,
        regionLine: line ? `${region}/${line}` : region,
          status: resolvedStatus,
          expire_at: row.expire_at ?? row.ExpireAt,
          destroy_in_days: row.destroy_in_days ?? row.DestroyInDays,
          last_emergency_renew_at: row.last_emergency_renew_at ?? row.LastEmergencyRenewAt ?? null,
          package_id: row.package_id ?? row.PackageID ?? 0,
        spec_raw: row.spec ?? row.Spec ?? row.spec_json ?? row.SpecJSON,
        specText: normalizeSpec(row.spec ?? row.Spec ?? row.spec_json ?? row.SpecJSON, row),
        ip,
      monthly_price: row.monthly_price ?? row.MonthlyPrice ?? 0
    };
  });

  if (filters.keyword) {
    const key = filters.keyword.toLowerCase();
    items = items.filter((item) =>
      String(item.id).includes(key) ||
      String(item.name || "").toLowerCase().includes(key) ||
      String(item.ip || "").toLowerCase().includes(key)
    );
  }
  if (filters.status) {
    items = items.filter((item) => item.status === filters.status);
  }
  if (filters.region) {
    items = items.filter((item) => String(item.regionLine || "").includes(filters.region));
  }
  if (filters.expire_days) {
    const days = Number(filters.expire_days);
    if (!Number.isNaN(days)) {
      const now = Date.now();
      items = items.filter((item) => {
        const expire = new Date(item.expire_at).getTime();
        if (Number.isNaN(expire)) return false;
        const diff = Math.ceil((expire - now) / (24 * 3600 * 1000));
        return diff <= days;
      });
    }
  }

  pagination.total = items.length;
  return items;
});

const currentAddons = computed(() => {
  const spec = parseJson(activeRecord.value?.spec_raw);
  return {
    add_cores: Number(spec?.add_cores ?? spec?.AddCores ?? 0),
    add_mem_gb: Number(spec?.add_mem_gb ?? spec?.AddMemGB ?? 0),
    add_disk_gb: Number(spec?.add_disk_gb ?? spec?.AddDiskGB ?? 0),
    add_bw_mbps: Number(spec?.add_bw_mbps ?? spec?.AddBWMbps ?? 0)
  };
});

const currentPackage = computed(() => {
  const id = activeRecord.value?.package_id;
  if (!id) return null;
  return catalog.packages.find((pkg) => String(pkg.id) === String(id)) || null;
});

const currentPlanGroup = computed(() => {
  if (!currentPackage.value) return null;
  const groupId = currentPackage.value.plan_group_id ?? currentPackage.value.planGroupId ?? currentPackage.value.PlanGroupID;
  return catalog.planGroups.find((g) => String(g.id) === String(groupId)) || null;
});

const packageOptions = computed(() => {
  if (!currentPlanGroup.value) return [];
  const groupId = currentPlanGroup.value.id ?? currentPlanGroup.value.ID;
  return catalog.packages
    .filter((pkg) => {
      const pkgGroup = pkg.plan_group_id ?? pkg.planGroupId ?? pkg.PlanGroupID;
      if (String(pkgGroup) !== String(groupId)) return false;
      if (pkg.active === false || pkg.visible === false) return false;
      return true;
    })
    .sort((a, b) => Number(a.monthly_price || 0) - Number(b.monthly_price || 0));
});

const resizeEnabled = computed(() => getSettingBool("resize_enabled") !== false);

const supportsScheduledResize = computed(() => {
  const byKey =
    getSettingBool("resize_scheduled_enabled") ??
    getSettingBool("resize_schedule_enabled") ??
    getSettingBool("resize_scheduled");
  return byKey === true;
});

const normalizePackageSpec = (pkg) => ({
  cpu: Number(pkg?.cores ?? pkg?.cpu ?? pkg?.CPU ?? pkg?.Cores ?? 0),
  memory_gb: Number(pkg?.memory_gb ?? pkg?.mem_gb ?? pkg?.MemoryGB ?? 0),
  disk_gb: Number(pkg?.disk_gb ?? pkg?.DiskGB ?? 0),
  bandwidth_mbps: Number(pkg?.bandwidth_mbps ?? pkg?.BandwidthMB ?? pkg?.bandwidth ?? 0)
});

const currentSpecForCompare = computed(() => {
  const fallback = {
    cpu: Number(activeRecord.value?.cpu || 0),
    memory_gb: Number(activeRecord.value?.memory_gb || 0),
    disk_gb: Number(activeRecord.value?.disk_gb || 0),
    bandwidth_mbps: Number(activeRecord.value?.bandwidth_mbps || 0)
  };
  if (!currentPackage.value) return fallback;
  const fromPackage = normalizePackageSpec(currentPackage.value);
  const hasSpec = Object.values(fromPackage).some((val) => Number(val) > 0);
  return hasSpec ? fromPackage : fallback;
});

const isSamePackageOption = (pkg) => {
  if (!pkg) return false;
  if (currentPackage.value?.id && String(pkg.id) === String(currentPackage.value.id)) return true;
  const pkgProduct = pkg.product_id ?? pkg.ProductID;
  const currentProduct = currentPackage.value?.product_id ?? currentPackage.value?.ProductID;
  if (pkgProduct && currentProduct && String(pkgProduct) === String(currentProduct)) return true;
  const pkgSpec = normalizePackageSpec(pkg);
  const currentSpec = currentSpecForCompare.value;
  return (
    pkgSpec.cpu === currentSpec.cpu &&
    pkgSpec.memory_gb === currentSpec.memory_gb &&
    pkgSpec.disk_gb === currentSpec.disk_gb &&
    pkgSpec.bandwidth_mbps === currentSpec.bandwidth_mbps
  );
};

  const isSameAddonsSelection = computed(() => {
    const current = currentAddons.value;
    return (
      Number(resizeForm.add_cores || 0) === Number(current.add_cores || 0) &&
      Number(resizeForm.add_mem_gb || 0) === Number(current.add_mem_gb || 0) &&
      Number(resizeForm.add_disk_gb || 0) === Number(current.add_disk_gb || 0) &&
      Number(resizeForm.add_bw_mbps || 0) === Number(current.add_bw_mbps || 0)
    );
  });

  const isSameTargetSelection = computed(() => {
    if (!resizeForm.target_package_id) return false;
    const target = packageOptions.value.find((pkg) => String(pkg.id) === String(resizeForm.target_package_id));
    return target ? isSamePackageOption(target) && isSameAddonsSelection.value : false;
  });

const addonMin = computed(() => ({
  add_cores: 0,
  add_mem_gb: 0,
  add_disk_gb: 0,
  add_bw_mbps: 0
}));

const addonMax = computed(() => {
  const group = currentPlanGroup.value || {};
  return {
    add_cores: group.add_core_max ?? 64,
    add_mem_gb: group.add_mem_max ?? 256,
    add_disk_gb: group.add_disk_max ?? 2000,
    add_bw_mbps: group.add_bw_max ?? 1000
  };
});

const addonStep = computed(() => {
  const group = currentPlanGroup.value || {};
  return {
    add_cores: group.add_core_step ?? 1,
    add_mem_gb: group.add_mem_step ?? 1,
    add_disk_gb: group.add_disk_step ?? 10,
    add_bw_mbps: group.add_bw_step ?? 10
  };
});

const stats = computed(() => {
  const items = store.items || [];
  const mapStatus = (row) => {
    const rawStatus = row.status ?? row.Status ?? "";
    const rawAutomationState = row.automation_state ?? row.AutomationState ?? null;
    return rawAutomationState !== null && rawAutomationState !== undefined
      ? statusFromAutomation(rawAutomationState)
      : rawStatus;
  };
  const running = items.filter((i) => mapStatus(i) === "running").length;
  const stopped = items.filter((i) => mapStatus(i) === "stopped").length;
  const provisioning = items.filter((i) => mapStatus(i) === "provisioning").length;
  const now = Date.now();
  const expiring = items.filter((i) => {
    const expire = new Date(i.expire_at ?? i.ExpireAt).getTime();
    if (Number.isNaN(expire)) return false;
    const diff = Math.ceil((expire - now) / (24 * 3600 * 1000));
    return diff <= 7 && diff > 0;
  }).length;
  return { running, stopped, provisioning, expiring };
});

const isExpiring = (dateStr) => {
  if (!dateStr) return false;
  const expire = new Date(dateStr).getTime();
  if (Number.isNaN(expire)) return false;
  const diff = Math.ceil((expire - Date.now()) / (24 * 3600 * 1000));
  return diff <= 7 && diff > 0;
};

const formatLocalDateTime = (value) => {
  if (!value) return "-";
  const dt = new Date(value);
  if (Number.isNaN(dt.getTime())) return String(value);
  return dt.toLocaleString("zh-CN", { hour12: false });
};

const emergencyRenewPolicy = computed(() => {
  const settings = site.settings || {};
  const enabledValue = settings.emergency_renew_enabled;
  const enabled =
    enabledValue === undefined ||
    enabledValue === null ||
    (typeof enabledValue === "string"
      ? enabledValue.toLowerCase() !== "false" && enabledValue !== "0"
      : enabledValue !== false);
  let windowDays = Number.parseInt(settings.emergency_renew_window_days ?? "7", 10);
  let intervalHours = Number.parseInt(settings.emergency_renew_interval_hours ?? "720", 10);
  if (!Number.isFinite(windowDays)) windowDays = 7;
  if (!Number.isFinite(intervalHours)) intervalHours = 720;
  if (windowDays < 0) windowDays = 0;
  if (intervalHours <= 0) intervalHours = 24;
  return {
    enabled,
    windowDays,
    intervalHours
  };
});

const emergencyRenewEligible = (record) => {
  if (!record?.expire_at) return false;
  if (!emergencyRenewPolicy.value.enabled) return false;
  const now = dayjs();
  const expire = dayjs(record.expire_at);
  if (expire.isBefore(now)) return false;
  const windowDays = emergencyRenewPolicy.value.windowDays;
  if (windowDays > 0) {
    const windowStart = expire.subtract(windowDays, "day");
    if (now.isBefore(windowStart)) return false;
  }
  if (record.last_emergency_renew_at) {
    const lastAt = dayjs(record.last_emergency_renew_at);
    const intervalHours = emergencyRenewPolicy.value.intervalHours;
    if (intervalHours > 0 && now.diff(lastAt, "hour", true) < intervalHours) return false;
  }
  return true;
};

const fetchData = () => {
  store.fetchVps();
};

const onTableChange = (pager) => {
  pagination.current = pager.current;
  pagination.pageSize = pager.pageSize;
};

const exportCsv = () => {
  const csv = "id,name,status\n" + dataSource.value.map((i) => `${i.id},${i.name},${i.status}`).join("\n");
  const blob = new Blob([csv], { type: "text/csv;charset=utf-8;" });
  const link = document.createElement("a");
  link.href = URL.createObjectURL(blob);
  link.download = "vps.csv";
  link.click();
};

const goDetail = (record) => router.push(`/console/vps/${record.id}`);
const goBuy = () => router.push("/console/buy");

const base = import.meta.env.VITE_API_BASE || "";

const openPanel = (record) => {
  const token = auth.token;
  const query = token ? `?token=${encodeURIComponent(token)}` : "";
  window.open(`${base}/api/v1/vps/${record.id}/panel${query}`, "_blank");
};

const openVnc = (record) => {
  const token = auth.token;
  const query = token ? `?token=${encodeURIComponent(token)}` : "";
  window.open(`${base}/api/v1/vps/${record.id}/vnc${query}`, "_blank");
};

const refresh = async (record) => {
  await store.refresh(record.id);
  message.success("已刷新");
};

const start = async (record) => {
  await startVps(record.id);
  message.success("已触发开机");
  fetchData();
};

const shutdown = async (record) => {
  await shutdownVps(record.id);
  message.success("已触发关机");
  fetchData();
};

const reboot = async (record) => {
  await rebootVps(record.id);
  message.success("已触发重启");
  fetchData();
};

const renewOpen = ref(false);
const resizeOpen = ref(false);
const refundOpen = ref(false);
const renewing = ref(false);
const resizing = ref(false);
const refunding = ref(false);
const resizeQuote = ref(null);
const resizeQuoteLoading = ref(false);
const resizeQuoteError = ref("");
const activeRecord = ref(null);
const renewForm = reactive({ months: 1 });
const resizeForm = reactive({
  add_cores: 0,
  add_mem_gb: 0,
  add_disk_gb: 0,
  add_bw_mbps: 0,
  target_package_id: null,
  reset_addons: false,
  schedule_mode: "now",
  scheduled_at: null
});
const refundReason = ref("");

const openRenew = (record) => {
  activeRecord.value = record;
  renewForm.months = 1;
  renewOpen.value = true;
};

const submitRenew = async () => {
  if (!activeRecord.value) return;
  renewing.value = true;
  try {
    await createVpsRenewOrder(activeRecord.value.id, { duration_months: renewForm.months });
    message.success("已生成续费订单");
    renewOpen.value = false;
  } catch (e) {
    const status = e?.response?.status;
    const errorText = e?.response?.data?.error || "续费失败";
    if (status === 409) {
      Modal.confirm({
        title: "已有待处理续费订单",
        content: errorText,
        okText: "去订单列表",
        cancelText: "我知道了",
        onOk: () => router.push("/console/orders")
      });
      return;
    }
    message.error(errorText);
  } finally {
    renewing.value = false;
  }
};

const submitEmergencyRenew = (record) => {
  if (!record?.id) return;
  Modal.confirm({
    title: "紧急续费确认",
    content: "紧急续费将按系统策略续费固定天数，确认继续？",
    okText: "确认",
    cancelText: "取消",
    async onOk() {
      try {
        await emergencyRenewVps(record.id);
        message.success("紧急续费已提交");
        await store.fetchVps();
      } catch (err) {
        message.error(err.response?.data?.error || "紧急续费失败");
      }
    }
  });
};

const openResize = (record) => {
  if (isExpired(record)) {
    message.warning("已到期实例不支持升降配");
    return;
  }
  activeRecord.value = record;
  resizeForm.add_cores = currentAddons.value.add_cores;
  resizeForm.add_mem_gb = currentAddons.value.add_mem_gb;
  resizeForm.add_disk_gb = currentAddons.value.add_disk_gb;
  resizeForm.add_bw_mbps = currentAddons.value.add_bw_mbps;
  resizeForm.target_package_id = currentPackage.value?.id ?? null;
  resizeForm.reset_addons = false;
  resizeForm.schedule_mode = "now";
  resizeForm.scheduled_at = null;
  resizeQuote.value = null;
  resizeQuoteError.value = "";
  resizeOpen.value = true;
  scheduleResizeQuote();
};

const resizeQuoteAmount = computed(() => {
  const raw = resizeQuote.value?.charge_amount ?? resizeQuote.value?.chargeAmount ?? 0;
  return Number(raw || 0);
});

const buildResizePayload = () => {
  const spec = resizeForm.reset_addons
    ? { add_cores: 0, add_mem_gb: 0, add_disk_gb: 0, add_bw_mbps: 0 }
    : {
        add_cores: resizeForm.add_cores,
        add_mem_gb: resizeForm.add_mem_gb,
        add_disk_gb: resizeForm.add_disk_gb,
        add_bw_mbps: resizeForm.add_bw_mbps
      };
  const payload = {
    target_package_id: resizeForm.target_package_id,
    reset_addons: resizeForm.reset_addons,
    spec
  };
  if (resizeForm.schedule_mode === "later" && resizeForm.scheduled_at) {
    const scheduled = dayjs(resizeForm.scheduled_at);
    if (scheduled.isValid()) {
      payload.scheduled_at = scheduled.toISOString();
    }
  }
  return payload;
};

const buildResizeQuotePayload = () => {
  const spec = resizeForm.reset_addons
    ? { add_cores: 0, add_mem_gb: 0, add_disk_gb: 0, add_bw_mbps: 0 }
    : {
        add_cores: resizeForm.add_cores,
        add_mem_gb: resizeForm.add_mem_gb,
        add_disk_gb: resizeForm.add_disk_gb,
        add_bw_mbps: resizeForm.add_bw_mbps
      };
  return {
    target_package_id: resizeForm.target_package_id,
    reset_addons: resizeForm.reset_addons,
    spec
  };
};

const fetchResizeQuote = async () => {
  if (!resizeEnabled.value) {
    resizeQuote.value = null;
    resizeQuoteError.value = "升降配功能已关闭";
    return;
  }
  if (!resizeForm.target_package_id || isSameTargetSelection.value) {
    resizeQuote.value = null;
    resizeQuoteError.value = "";
    return;
  }
  resizeQuoteLoading.value = true;
  resizeQuoteError.value = "";
  try {
    const res = await quoteVpsResizeOrder(activeRecord.value.id, buildResizeQuotePayload());
    resizeQuote.value = res.data?.quote ?? res.data;
  } catch (e) {
    const status = e?.response?.status;
    const errorText = e?.response?.data?.error || "升降配失败";
    resizeQuote.value = null;
    if (status === 409) {
      resizeQuoteError.value = "已有进行中的升降配任务/订单";
      return;
    }
    resizeQuoteError.value = errorText;
  } finally {
    resizeQuoteLoading.value = false;
  }
};

let resizeQuoteTimer;
const scheduleResizeQuote = () => {
  if (resizeQuoteTimer) clearTimeout(resizeQuoteTimer);
  resizeQuoteTimer = setTimeout(() => {
    if (!activeRecord.value) return;
    void fetchResizeQuote();
  }, 300);
};

const submitResize = async () => {
  if (!activeRecord.value) return;
  if (isExpired(activeRecord.value)) {
    message.warning("已到期实例不支持升降配");
    return;
  }
  if (!resizeEnabled.value) {
    message.error("升降配功能已关闭");
    return;
  }
  if (!resizeForm.target_package_id) {
    message.warning("暂无可用套餐");
    return;
  }
  if (isSameTargetSelection.value) {
    message.warning("不能选择当前套餐");
    return;
  }
  if (resizeForm.schedule_mode === "later") {
    if (!resizeForm.scheduled_at) {
      message.warning("请选择执行时间");
      return;
    }
    const scheduled = dayjs(resizeForm.scheduled_at);
    if (!scheduled.isValid()) {
      message.warning("执行时间格式不正确");
      return;
    }
    if (scheduled.isBefore(dayjs())) {
      message.warning("执行时间需要晚于当前时间");
      return;
    }
  }
  resizing.value = true;
  try {
    const payload = buildResizePayload();
    await createVpsResizeOrder(activeRecord.value.id, payload);
    message.success("已生成改配订单");
    resizeOpen.value = false;
  } catch (e) {
    const status = e?.response?.status;
    const errorText = e?.response?.data?.error || "升降配失败";
    if (status === 409) {
      Modal.confirm({
        title: "已有进行中的升降配任务/订单",
        content: errorText,
        okText: "去订单列表",
        cancelText: "我知道了",
        onOk: () => router.push("/console/orders")
      });
    } else {
      message.error(errorText);
    }
  } finally {
    resizing.value = false;
  }
};

const openRefund = (record) => {
  activeRecord.value = record;
  refundReason.value = "";
  refundOpen.value = true;
};

const submitRefund = async () => {
  if (refunding.value) return;
  if (!refundReason.value.trim()) {
    message.warning("请填写退款原因");
    return;
  }
  if (String(refundReason.value || "").length > INPUT_LIMITS.REFUND_REASON) {
    message.warning(`退款原因长度不能超过 ${INPUT_LIMITS.REFUND_REASON} 个字符`);
    return;
  }
  refunding.value = true;
  try {
    const res = await requestVpsRefund(activeRecord.value.id, { reason: refundReason.value });
    const orderId = res?.data?.order?.id ?? res?.data?.order?.ID;
    if (orderId) {
      message.success("已提交退款申请，订单ID: " + orderId);
    } else {
      message.success("已提交退款申请");
    }
    refundOpen.value = false;
  } catch (e) {
    message.error(e?.response?.data?.error || e?.response?.data?.message || e?.message || "提交失败");
  } finally {
    refunding.value = false;
  }
};

const mobileActionsOpen = ref(false);
const mobileActionRecord = ref(null);

const showMobileActions = (record) => {
  mobileActionRecord.value = record;
  mobileActionsOpen.value = true;
};

const handleMobileAction = (action) => {
  const record = mobileActionRecord.value;
  if (!record) return;
  mobileActionsOpen.value = false;

    switch (action) {
      case "detail":
        goDetail(record);
        break;
      case "panel":
        openPanel(record);
        break;
      case "vnc":
        openVnc(record);
        break;
      case "start":
        start(record);
        break;
      case "shutdown":
        shutdown(record);
        break;
      case "reboot":
        reboot(record);
        break;
      case "renew":
        openRenew(record);
        break;
      case "urgent-renew":
        submitEmergencyRenew(record);
        break;
      case "resize":
        openResize(record);
        break;
      case "refresh":
        refresh(record);
        break;
      case "refund":
      openRefund(record);
      break;
  }
};

watch(
  () => [
    resizeForm.target_package_id,
    resizeForm.add_cores,
    resizeForm.add_mem_gb,
    resizeForm.add_disk_gb,
    resizeForm.add_bw_mbps,
    resizeForm.reset_addons
  ],
  () => {
    if (!resizeOpen.value) return;
    scheduleResizeQuote();
  }
);

watch(
  () => resizeOpen.value,
  (open) => {
    if (open) return;
    if (resizeQuoteTimer) clearTimeout(resizeQuoteTimer);
    resizeQuoteTimer = null;
    resizeQuote.value = null;
    resizeQuoteError.value = "";
  }
);

fetchData();
if (!catalog.packages.length || !catalog.planGroups.length) {
  catalog.fetchCatalog();
}
site.fetchSettings();
</script>
<style scoped>
.vps-list-page {
  padding: 0;
  min-height: 100vh;
  background: var(--bg-primary);
}

/* Page Header with Gradient */
.page-header-gradient {
  background: linear-gradient(135deg, var(--primary) 0%, #0958d9 100%);
  padding: 32px 24px;
  position: relative;
  overflow: hidden;
}

.page-header-gradient::before {
  content: "";
  position: absolute;
  top: -50%;
  right: -10%;
  width: 500px;
  height: 500px;
  background: radial-gradient(circle, rgba(255,255,255,0.1) 0%, transparent 70%);
  border-radius: 50%;
  pointer-events: none;
}

.page-header-gradient::after {
  content: "";
  position: absolute;
  bottom: -30%;
  left: -5%;
  width: 300px;
  height: 300px;
  background: radial-gradient(circle, rgba(255,255,255,0.08) 0%, transparent 70%);
  border-radius: 50%;
  pointer-events: none;
}

.page-header-content {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  flex-wrap: wrap;
  max-width: 1400px;
  margin: 0 auto;
}

.page-title-wrapper {
  display: flex;
  align-items: center;
  gap: 16px;
}

.page-title-icon {
  font-size: 42px;
  color: rgba(255, 255, 255, 0.95);
  filter: drop-shadow(0 2px 8px rgba(0,0,0,0.15));
}

.page-title {
  font-size: 28px;
  font-weight: 700;
  color: #ffffff;
  margin: 0;
  letter-spacing: -0.5px;
  text-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

.page-desc {
  color: rgba(255, 255, 255, 0.85);
  font-size: 13px;
  margin: 4px 0 0 0;
  font-weight: 400;
}

.page-header-actions {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.action-btn {
  height: 40px;
  padding: 0 20px;
  border-radius: 10px;
  font-weight: 500;
  border: 1px solid rgba(255,255,255,0.3);
  background: rgba(255,255,255,0.15);
  color: #ffffff;
  backdrop-filter: blur(10px);
  transition: all 0.3s ease;
}

.action-btn:hover {
  background: rgba(255,255,255,0.25);
  border-color: rgba(255,255,255,0.5);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0,0,0,0.15);
  color: #ffffff;
}

.action-btn-primary {
  height: 40px;
  padding: 0 20px;
  border-radius: 10px;
  font-weight: 600;
  background: #ffffff;
  color: var(--primary);
  border: none;
  box-shadow: 0 4px 12px rgba(0,0,0,0.15);
  transition: all 0.3s ease;
}

.action-btn-primary:hover {
  background: #f8f9fa;
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(0,0,0,0.2);
  color: var(--primary);
}

/* Statistics Cards */
.stats-container {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 16px;
  padding: 24px;
  max-width: 1400px;
  margin: 0 auto;
}

.stat-card {
  position: relative;
  background: var(--card);
  border-radius: 16px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0,0,0,0.04);
  transition: all 0.3s ease;
  cursor: default;
}

.stat-card::before {
  content: "";
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: currentColor;
  opacity: 0;
  transition: opacity 0.3s ease;
}

.stat-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0,0,0,0.12);
}

.stat-card:hover::before {
  opacity: 1;
}

.stat-icon-wrapper {
  width: 56px;
  height: 56px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  z-index: 1;
}

.stat-icon {
  font-size: 26px;
  color: #ffffff;
}

.stat-decoration {
  position: absolute;
  right: -10px;
  bottom: -10px;
  width: 80px;
  height: 80px;
  border-radius: 50%;
  opacity: 0.1;
  pointer-events: none;
}

.stat-content {
  flex: 1;
  position: relative;
  z-index: 1;
}

.stat-value {
  font-size: 32px;
  font-weight: 700;
  line-height: 1;
  margin-bottom: 4px;
  background: linear-gradient(135deg, currentColor 0%, currentColor 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.stat-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--text2);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

/* Stat Card Variants */
.stat-running {
  color: #52c41a;
}

.stat-running .stat-icon-wrapper {
  background: linear-gradient(135deg, #52c41a 0%, #389e0d 100%);
  box-shadow: 0 4px 12px rgba(82, 196, 26, 0.3);
}

.stat-running .stat-decoration {
  background: #52c41a;
}

.stat-stopped {
  color: #8c8c8c;
}

.stat-stopped .stat-icon-wrapper {
  background: linear-gradient(135deg, #8c8c8c 0%, #595959 100%);
  box-shadow: 0 4px 12px rgba(140, 140, 140, 0.3);
}

.stat-stopped .stat-decoration {
  background: #8c8c8c;
}

.stat-provisioning {
  color: #1677ff;
}

.stat-provisioning .stat-icon-wrapper {
  background: linear-gradient(135deg, #1677ff 0%, #0958d9 100%);
  box-shadow: 0 4px 12px rgba(22, 119, 255, 0.3);
}

.stat-provisioning .stat-decoration {
  background: #1677ff;
}

.stat-expiring {
  color: #ff4d4f;
}

.stat-expiring .stat-icon-wrapper {
  background: linear-gradient(135deg, #ff4d4f 0%, #cf1322 100%);
  box-shadow: 0 4px 12px rgba(255, 77, 79, 0.3);
}

.stat-expiring .stat-decoration {
  background: #ff4d4f;
}

/* Filter Bar Card Override */
.stats-container + :deep(.card) {
  border-radius: 16px;
  overflow: hidden;
}

.stats-container + :deep(.card .card-body) {
  padding: 16px;
}

/* Table Styles */
:deep(.ant-table) {
  background: transparent;
}

:deep(.ant-table-container) {
  border-radius: 12px;
  overflow: hidden;
}

:deep(.ant-table-thead > tr > th) {
  background: linear-gradient(180deg, #fafbfc 0%, #f5f7fa 100%);
  font-weight: 600;
  font-size: 13px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  padding: 16px;
  border-bottom: 2px solid #e8ecf1;
}

:deep(.ant-table-tbody > tr) {
  transition: all 0.2s ease;
}

:deep(.ant-table-tbody > tr:hover) {
  background: #f8fafd !important;
  transform: scale(1.002);
}

:deep(.ant-table-tbody > tr > td) {
  padding: 14px 16px;
  border-bottom: 1px solid #f0f2f5;
}

/* Name Cell */
.name-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.name-icon {
  font-size: 22px;
  color: var(--primary);
  padding: 6px;
  background: linear-gradient(135deg, #e6f4ff 0%, #bae0ff 100%);
  border-radius: 8px;
}

.name-text {
  font-weight: 600;
  color: var(--text);
  font-size: 14px;
}

.name-id {
  font-size: 11px;
  color: var(--text3);
  font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
  margin-top: 2px;
}

/* Expiring Status */
.expiring {
  color: #ff4d4f;
  font-weight: 600;
  animation: pulse-warning 2s ease-in-out infinite;
}

@keyframes pulse-warning {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.7; }
}

.destroy-hint {
  font-size: 11px;
  color: #ff4d4f;
  background: #fff1f0;
  padding: 2px 8px;
  border-radius: 4px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-weight: 500;
}

/* Mobile Cards */
.vps-mobile-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 0 24px 24px;
}

.vps-mobile-card {
  background: #ffffff;
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 2px 12px rgba(0,0,0,0.06);
  transition: all 0.3s ease;
  cursor: pointer;
  border: 1px solid #f0f2f5;
}

.vps-mobile-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(0,0,0,0.12);
  border-color: #d9d9d9;
}

.vps-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 16px 12px;
  background: linear-gradient(180deg, #fafbfc 0%, #ffffff 100%);
  border-bottom: 1px solid #f0f2f5;
}

.vps-card-title {
  display: flex;
  align-items: center;
  gap: 10px;
}

.vps-card-icon {
  font-size: 24px;
  color: var(--primary);
}

.vps-card-name {
  font-weight: 600;
  font-size: 15px;
  color: var(--text);
}

.vps-card-body {
  padding: 12px 16px;
}

.vps-info-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid #f5f7fa;
}

.vps-info-row:last-child {
  border-bottom: none;
}

.vps-info-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text3);
  font-weight: 500;
}

.info-icon {
  font-size: 14px;
  color: var(--text2);
}

.info-icon.warning {
  color: #ff4d4f;
}

.vps-info-value {
  font-size: 13px;
  color: var(--text);
  font-weight: 500;
}

.vps-spec-tag {
  font-weight: 500;
  border-radius: 6px;
}

.vps-destroy-row {
  background: #fff1f0;
  margin: 8px -16px -12px;
  padding: 10px 16px !important;
  border-radius: 0 0 12px 12px;
  border-bottom: none !important;
}

.vps-card-actions {
  display: flex;
  border-top: 1px solid #f0f2f5;
  background: #fafbfc;
}

.vps-action-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 14px;
  color: var(--text2);
  font-size: 18px;
  transition: all 0.2s ease;
  border-right: 1px solid #f0f2f5;
}

.vps-action-btn:last-child {
  border-right: none;
}

.vps-action-btn:hover {
  background: #e6f7ff;
  color: var(--primary);
}

.vps-action-urgent {
  color: #ff4d4f;
}

.vps-action-urgent:hover {
  background: #fff1f0;
  color: #cf1322;
}

.urgent-renew-btn {
  height: 28px;
  padding: 0 10px;
}

.vps-action-more:hover {
  background: #f0f2f5;
  color: var(--text);
}

/* Action Grid in Drawer */
.action-grid-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 16px;
  background: linear-gradient(135deg, #f7f8fa 0%, #f0f2f5 100%);
  border-radius: 12px;
  cursor: pointer;
  font-size: 12px;
  font-weight: 500;
  color: var(--text2);
  transition: all 0.3s ease;
  border: 1px solid transparent;
}

.action-grid-item:hover {
  background: linear-gradient(135deg, #e6f7ff 0%, #bae0ff 100%);
  color: var(--primary);
  border-color: #91caff;
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(22, 119, 255, 0.15);
}

.action-grid-item.danger {
  color: #ff4d4f;
}

.action-grid-item.danger:hover {
  background: linear-gradient(135deg, #fff1f0 0%, #ffccc7 100%);
  border-color: #ff7875;
  box-shadow: 0 4px 12px rgba(255, 77, 79, 0.15);
}

.action-icon {
  font-size: 22px;
}

.action-icon.start {
  color: #52c41a;
}

.action-icon.stop {
  color: #8c8c8c;
}

.action-icon.reboot {
  color: #faad14;
}

.action-icon.renew {
  color: #1677ff;
}

.action-icon.resize {
  color: #722ed1;
}

/* Modal Styles */
:deep(.ant-modal-content) {
  border-radius: 16px;
  overflow: hidden;
}

:deep(.ant-modal-header) {
  background: linear-gradient(135deg, #f7f8fa 0%, #f0f2f5 100%);
  border-bottom: 1px solid #e8ecf1;
  padding: 20px 24px;
}

:deep(.ant-modal-title) {
  font-weight: 600;
  font-size: 16px;
  color: var(--text);
}

:deep(.ant-modal-body) {
  padding: 24px;
}

:deep(.ant-modal-footer) {
  border-top: 1px solid #f0f2f5;
  padding: 16px 24px;
}

/* Responsive */
@media (max-width: 768px) {
  .page-header-content {
    flex-direction: column;
    align-items: flex-start;
  }

  .page-title-icon {
    font-size: 32px;
  }

  .page-title {
    font-size: 22px;
  }

  .stats-container {
    grid-template-columns: repeat(2, 1fr);
    padding: 16px;
    gap: 12px;
  }

  .stat-card {
    padding: 16px;
  }

  .stat-icon-wrapper {
    width: 48px;
    height: 48px;
  }

  .stat-icon {
    font-size: 22px;
  }

  .stat-value {
    font-size: 24px;
  }

  .vps-mobile-list {
    padding: 0 16px 16px;
  }
}

@media (max-width: 480px) {
  .stats-container {
    grid-template-columns: 1fr;
  }
}
</style>
