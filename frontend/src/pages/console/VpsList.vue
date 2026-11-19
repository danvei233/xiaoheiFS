
<template>
  <div class="page vps-list-page">
    <div class="page-header-gradient">
      <div class="page-header-content">
        <div class="page-title-section">
          <div class="page-title-wrapper">
            <CloudServerOutlined class="page-title-icon" />
            <div>
              <h1 class="page-title">{{ t('console.vps.title') }}</h1>
              <p class="page-desc">{{ t('console.vps.subtitle') }}</p>
            </div>
          </div>
        </div>
        <div class="page-header-actions">
          <a-button @click="fetchData" class="action-btn">
            <template #icon><ReloadOutlined /></template>
            {{ t('common.refresh') }}
          </a-button>
          <a-button type="primary" @click="goBuy" class="action-btn-primary">
            <template #icon><PlusOutlined /></template>
            {{ t('console.vps.buy') }}
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
          <div class="stat-label">{{ t('console.vps.running') }}</div>
        </div>
      </div>
      <div class="stat-card stat-stopped">
        <div class="stat-icon-wrapper">
          <StopOutlined class="stat-icon" />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.stopped }}</div>
          <div class="stat-label">{{ t('console.vps.stopped') }}</div>
        </div>
      </div>
      <div class="stat-card stat-provisioning">
        <div class="stat-icon-wrapper">
          <SyncOutlined spin class="stat-icon" />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.provisioning }}</div>
          <div class="stat-label">{{ t('console.vps.provisioning') }}</div>
        </div>
      </div>
      <div class="stat-card stat-expiring">
        <div class="stat-icon-wrapper">
          <ClockCircleOutlined class="stat-icon" />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.expiring }}</div>
          <div class="stat-label">{{ t('console.dashboard.expiringSoon') }}</div>
        </div>
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
          <a-input v-model:value="filters.region" :placeholder="t('console.vps.region')">
            <template #prefix><GlobalOutlined /></template>
          </a-input>
          <a-input v-model:value="filters.expire_days" :placeholder="t('console.vps.expireDays')">
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
            <a-tooltip :title="t('common.detail')">
              <a-button type="text" size="small" @click="goDetail(record)">
                <template #icon><EyeOutlined /></template>
              </a-button>
            </a-tooltip>
            <a-tooltip :title="t('console.vps.controlPanel')">
              <a-button type="text" size="small" @click="openPanel(record)">
                <template #icon><ControlOutlined /></template>
              </a-button>
            </a-tooltip>
            <a-tooltip v-if="emergencyRenewEligible(record)" :title="t('console.vps.urgentRenew')">
              <a-button type="primary" size="small" danger class="urgent-renew-btn" @click="submitEmergencyRenew(record)">
                {{ t('console.vps.urgentRenew') }}
              </a-button>
            </a-tooltip>
            <a-dropdown>
              <a-button type="text" size="small">
                <template #icon><MoreOutlined /></template>
              </a-button>
              <template #overlay>
                <a-menu>
                  <a-menu-item key="start" @click="start(record)">{{ t('console.vps.start') }}</a-menu-item>
                  <a-menu-item key="shutdown" @click="shutdown(record)">{{ t('console.vps.shutdown') }}</a-menu-item>
                  <a-menu-item key="reboot" @click="reboot(record)">{{ t('console.vps.reboot') }}</a-menu-item>
                  <a-menu-divider />
                  <a-menu-item key="renew" @click="openRenew(record)">{{ t('console.vps.renew') }}</a-menu-item>
                  <a-menu-item v-if="emergencyRenewEligible(record)" key="urgent-renew" @click="submitEmergencyRenew(record)">
                    {{ t('console.vps.urgentRenew') }}
                  </a-menu-item>
                  <a-menu-item key="resize" :disabled="isExpired(record)" @click="openResize(record)">{{ t('console.vps.resize') }}</a-menu-item>
                  <a-menu-item key="refresh" @click="refresh(record)">{{ t('console.vps.refreshStatus') }}</a-menu-item>
                  <a-menu-divider />
                  <a-menu-item key="refund" danger @click="openRefund(record)">{{ t('console.vps.refund') }}</a-menu-item>
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
                  {{ t('console.vps.region') }}
                </span>
                <span class="vps-info-value">{{ item.regionLine }}</span>
              </div>
              <div class="vps-info-row">
                <span class="vps-info-label">
                  <ApartmentOutlined class="info-icon" />
                  {{ t('console.vps.spec') }}
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
                  {{ t('common.expireAt') }}
                </span>
                <span class="vps-info-value" :class="{ expiring: isExpiring(item.expire_at) }">
                  {{ formatLocalDateTime(item.expire_at) }}
                </span>
              </div>
              <div v-if="item.destroy_in_days !== undefined && item.destroy_in_days !== null" class="vps-info-row vps-destroy-row">
                <WarningOutlined class="info-icon warning" />
                <span class="vps-info-value destroy-hint">
                  {{ t('console.vps.autoDelete', { n: item.destroy_in_days }) }}
                </span>
              </div>
            </div>
            <div class="vps-card-actions">
              <a-tooltip :title="t('common.detail')">
                <div class="vps-action-btn" @click.stop="goDetail(item)">
                  <EyeOutlined />
                </div>
              </a-tooltip>
              <a-tooltip :title="t('console.vps.controlPanel')">
                <div class="vps-action-btn" @click.stop="openPanel(item)">
                  <ControlOutlined />
                </div>
              </a-tooltip>
              <a-tooltip v-if="emergencyRenewEligible(item)" :title="t('console.vps.urgentRenew')">
                <div class="vps-action-btn vps-action-urgent" @click.stop="submitEmergencyRenew(item)">
                  <CalendarOutlined />
                </div>
              </a-tooltip>
              <a-tooltip title="VNC">
                <div class="vps-action-btn" @click.stop="openVnc(item)">
                  <DesktopOutlined />
                </div>
              </a-tooltip>
              <a-tooltip :title="t('console.vps.moreActions')">
                <div class="vps-action-btn vps-action-more" @click.stop="showMobileActions(item)">
                  <MoreOutlined />
                </div>
              </a-tooltip>
            </div>
          </div>
        </div>
      </template>
    </ProTable>

    <a-modal v-model:open="renewOpen" :title="t('console.vps.renewTitle')" :width="500" @ok="submitRenew" :confirm-loading="renewing">
      <a-alert
        :message="t('console.vps.renewNotice')"
        :description="t('console.vps.renewDesc')"
        type="info"
        show-icon
        style="margin-bottom: 20px"
      />
      <a-form layout="vertical">
        <a-form-item :label="t('console.vps.renewMonths')">
          <a-input-number v-model:value="renewForm.months" :min="1" :max="120" style="width: 100%" size="large">
            <template #addonAfter>{{ t('console.vps.months') }}</template>
          </a-input-number>
        </a-form-item>
        <a-alert
          :message="t('console.vps.autoCalc')"
          :description="t('console.vps.autoCalcDesc', { price: activeRecord?.monthly_price || 0, months: renewForm.months })"
          type="info"
          show-icon
        />
      </a-form>
    </a-modal>

    <a-modal v-model:open="resizeOpen" :title="t('console.vps.resizeTitle')" :width="560" @ok="submitResize" :confirm-loading="resizing" :ok-button-props="{ disabled: isExpired(activeRecord) }">
      <a-form layout="vertical">
        <a-form-item :label="t('console.vps.targetPackage')">
          <a-select v-model:value="resizeForm.target_package_id" :placeholder="t('console.vps.selectPackage')" size="large" :disabled="!resizeEnabled">
              <a-select-option v-for="pkg in packageOptions" :key="pkg.id" :value="pkg.id">
                {{ pkg.name }} (￥{{ Number(pkg.monthly_price || 0).toFixed(2) }}/{{ t('console.vps.perMonth') }})
              </a-select-option>
            </a-select>
          </a-form-item>
        <a-alert
          v-if="isSameTargetSelection"
          type="warning"
          show-icon
          :message="t('console.vps.samePlan')"
          style="margin-bottom: 12px"
        />
        <a-alert
          v-if="!resizeEnabled"
          type="warning"
          show-icon
          :message="t('console.vps.resizeDisabled')"
          style="margin-bottom: 12px"
        />
        <a-alert
          v-if="isExpired(activeRecord)"
          type="warning"
          show-icon
          :message="t('console.vps.resizeExpired')"
          style="margin-bottom: 12px"
        />
        <a-form-item :label="t('console.vps.executionTime')">
          <a-radio-group v-model:value="resizeForm.schedule_mode">
            <a-radio value="now">{{ t('console.vps.immediate') }}</a-radio>
            <a-radio value="later">{{ t('console.vps.scheduled') }}</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item v-if="resizeForm.schedule_mode === 'later'" :label="t('console.vps.scheduled')">
          <a-date-picker v-model:value="resizeForm.scheduled_at" show-time style="width: 100%" :placeholder="t('console.vps.selectTime')" />
        </a-form-item>
        <a-form-item>
          <a-switch v-model:checked="resizeForm.reset_addons" />
          <span style="margin-left: 8px">{{ t('console.vps.resetAddons') }}</span>
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item :label="t('console.vps.cpuAddon')">
              <a-input-number v-model:value="resizeForm.add_cores" :min="addonMin.add_cores" :max="addonMax.add_cores" :step="addonStep.add_cores" :disabled="resizeForm.reset_addons" style="width: 100%" size="large">
                <template #addonAfter>{{ t('console.vps.cores') }}</template>
              </a-input-number>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item :label="t('console.vps.memAddon')">
              <a-input-number v-model:value="resizeForm.add_mem_gb" :min="addonMin.add_mem_gb" :max="addonMax.add_mem_gb" :step="addonStep.add_mem_gb" :disabled="resizeForm.reset_addons" style="width: 100%" size="large">
                <template #addonAfter>GB</template>
              </a-input-number>
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item :label="t('console.vps.diskAddon')">
              <a-input-number v-model:value="resizeForm.add_disk_gb" :min="addonMin.add_disk_gb" :max="addonMax.add_disk_gb" :step="addonStep.add_disk_gb" :disabled="resizeForm.reset_addons" style="width: 100%" size="large">
                <template #addonAfter>GB</template>
              </a-input-number>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item :label="t('console.vps.bwAddon')">
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
          :message="t('console.vps.calcPrice')"
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
              {{ t('console.vps.payThisCycle', { amount: resizeQuoteAmount.toFixed(2) }) }}
            </span>
            <span v-else>
              {{ t('console.vps.cycleDiff', { amount: Math.abs(resizeQuoteAmount).toFixed(2) }) }}
            </span>
          </template>
          <template #description>
            <span v-if="resizeQuoteAmount < 0">{{ t('console.vps.refundChannel') }}</span>
            <span v-else>{{ t('console.vps.finalAmount') }}</span>
          </template>
        </a-alert>
      </a-form>
    </a-modal>

    <a-modal v-model:open="refundOpen" :title="t('console.vps.refundTitle')" :width="500" @ok="submitRefund" :confirm-loading="refunding">
      <a-alert
        :message="t('console.vps.refundNotice')"
        :description="t('console.vps.refundDesc')"
        type="warning"
        show-icon
        style="margin-bottom: 20px"
      />
      <a-form layout="vertical">
        <a-form-item :label="t('console.vps.refundReason')" required>
          <a-textarea
            v-model:value="refundReason"
            :rows="4"
            :placeholder="t('console.vps.refundReasonPlaceholder')"
            :maxlength="INPUT_LIMITS.REFUND_REASON"
            show-count
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-drawer v-model:open="mobileActionsOpen" :title="t('common.actions')" placement="bottom" :height="320">
      <a-grid :col="4" :gutter="[12, 12]">
        <a-grid-item><div class="action-grid-item" @click="handleMobileAction('detail')"><EyeOutlined class="action-icon" /><span>{{ t('common.detail') }}</span></div></a-grid-item>
        <a-grid-item><div class="action-grid-item" @click="handleMobileAction('panel')"><ControlOutlined class="action-icon" /><span>{{ t('console.vps.controlPanel') }}</span></div></a-grid-item>
        <a-grid-item><div class="action-grid-item" @click="handleMobileAction('vnc')"><DesktopOutlined class="action-icon" /><span>VNC</span></div></a-grid-item>
        <a-grid-item><div class="action-grid-item" @click="handleMobileAction('start')"><PlayCircleOutlined class="action-icon start" /><span>{{ t('console.vps.start') }}</span></div></a-grid-item>
        <a-grid-item><div class="action-grid-item" @click="handleMobileAction('shutdown')"><StopOutlined class="action-icon stop" /><span>{{ t('console.vps.shutdown') }}</span></div></a-grid-item>
        <a-grid-item><div class="action-grid-item" @click="handleMobileAction('reboot')"><ReloadOutlined class="action-icon reboot" /><span>{{ t('console.vps.reboot') }}</span></div></a-grid-item>
        <a-grid-item><div class="action-grid-item" @click="handleMobileAction('renew')"><CalendarOutlined class="action-icon renew" /><span>{{ t('console.vps.renew') }}</span></div></a-grid-item>
        <a-grid-item v-if="emergencyRenewEligible(mobileActionRecord)"><div class="action-grid-item danger" @click="handleMobileAction('urgent-renew')"><CalendarOutlined class="action-icon renew" /><span>{{ t('console.vps.urgentRenew') }}</span></div></a-grid-item>
        <a-grid-item><div class="action-grid-item" @click="handleMobileAction('resize')"><ApartmentOutlined class="action-icon resize" /><span>{{ t('console.vps.resize') }}</span></div></a-grid-item>
        <a-grid-item><div class="action-grid-item" @click="handleMobileAction('refresh')"><SyncOutlined class="action-icon" /><span>{{ t('common.refresh') }}</span></div></a-grid-item>
        <a-grid-item><div class="action-grid-item danger" @click="handleMobileAction('refund')"><RollbackOutlined class="action-icon" /><span>{{ t('console.vps.refund') }}</span></div></a-grid-item>
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
import { t, locale } from "@/locales";
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

const statusOptions = computed(() => [
  { label: t('console.vps.running'), value: "running" },
  { label: t('console.vps.stopped'), value: "stopped" },
  { label: t('console.vps.locked'), value: "locked" },
  { label: t('expire.expired'), value: "expired_locked" },
  { label: t('console.vps.provisioning'), value: "provisioning" },
  { label: t('console.vps.reinstalling'), value: "reinstalling" },
  { label: t('console.vps.reinstallFailed'), value: "reinstall_failed" },
  { label: t('console.vps.failed'), value: "failed" },
  { label: t('console.vps.deleting'), value: "deleting" }
]);

const statusTabs = computed(() => [
  { label: t('console.vps.running'), value: "running" },
  { label: t('console.vps.stopped'), value: "stopped" },
  { label: t('console.vps.locked'), value: "locked" },
  { label: t('expire.expired'), value: "expired_locked" },
  { label: t('console.vps.reinstalling'), value: "reinstalling" }
]);

const loading = computed(() => store.loading);
const pagination = reactive({ current: 1, pageSize: 10, total: 0, showSizeChanger: true });

const columns = computed(() => [
  { title: t('console.vps.instance'), dataIndex: "name", key: "name", width: 200 },
  { title: t('console.vps.regionLine'), dataIndex: "regionLine", key: "regionLine", width: 150 },
  { title: t('console.vps.spec'), dataIndex: "spec", key: "spec", width: 200 },
  { title: t('console.vps.ip'), dataIndex: "ip", key: "ip", width: 140 },
  { title: t('common.status'), dataIndex: "status", key: "status", width: 100 },
  { title: t('common.expireAt'), dataIndex: "expire_at", key: "expire_at", width: 170 },
  { title: t('console.vps.actions'), key: "action", width: 160, fixed: "right" }
]);

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
  const parts = [
    t('console.vps.cpuSpec', { n: cpu }),
    t('console.vps.memSpec', { n: mem }),
    t('console.vps.diskSpec', { n: disk })
  ];
  if (bw != null) parts.push(t('console.vps.bwSpec', { n: bw }));
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
  return dt.toLocaleString(locale.value, { hour12: false });
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
  message.success(t('console.vps.refreshed'));
};

const start = async (record) => {
  await startVps(record.id);
  message.success(t('console.vps.triggeredStart'));
  fetchData();
};

const shutdown = async (record) => {
  await shutdownVps(record.id);
  message.success(t('console.vps.triggeredShutdown'));
  fetchData();
};

const reboot = async (record) => {
  await rebootVps(record.id);
  message.success(t('console.vps.triggeredReboot'));
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
    message.success(t('console.vps.renewCreated'));
    renewOpen.value = false;
  } catch (e) {
    const status = e?.response?.status;
    const errorText = e?.response?.data?.error || t('console.vps.renewFailed');
    if (status === 409) {
      Modal.confirm({
        title: t('console.vps.pendingRenew'),
        content: errorText,
        okText: t('console.vps.goOrders'),
        cancelText: t('console.vps.gotIt'),
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
    title: t('console.vps.urgentConfirmTitle'),
    content: t('console.vps.urgentConfirmDesc'),
    okText: t('common.confirm'),
    cancelText: t('common.cancel'),
    async onOk() {
      try {
        await emergencyRenewVps(record.id);
        message.success(t('console.vps.urgentSubmitted'));
        await store.fetchVps();
      } catch (err) {
        message.error(err.response?.data?.error || t('console.vps.urgentFailed'));
      }
    }
  });
};

const openResize = (record) => {
  if (isExpired(record)) {
    message.warning(t('console.vps.resizeExpired'));
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
    resizeQuoteError.value = t('console.vps.resizeDisabled');
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
    const errorText = e?.response?.data?.error || t('console.vps.resizeFailed');
    resizeQuote.value = null;
    if (status === 409) {
      resizeQuoteError.value = t('console.vps.resizeInProgress');
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
    message.warning(t('console.vps.resizeExpired'));
    return;
  }
  if (!resizeEnabled.value) {
    message.error(t('console.vps.resizeDisabled'));
    return;
  }
  if (!resizeForm.target_package_id) {
    message.warning(t('console.vps.noPackages'));
    return;
  }
  if (isSameTargetSelection.value) {
    message.warning(t('console.vps.samePlan'));
    return;
  }
  if (resizeForm.schedule_mode === "later") {
    if (!resizeForm.scheduled_at) {
      message.warning(t('console.vps.selectExecTime'));
      return;
    }
    const scheduled = dayjs(resizeForm.scheduled_at);
    if (!scheduled.isValid()) {
      message.warning(t('console.vps.execTimeInvalid'));
      return;
    }
    if (scheduled.isBefore(dayjs())) {
      message.warning(t('console.vps.execTimeFuture'));
      return;
    }
  }
  resizing.value = true;
  try {
    const payload = buildResizePayload();
    await createVpsResizeOrder(activeRecord.value.id, payload);
    message.success(t('console.vps.resizeCreated'));
    resizeOpen.value = false;
  } catch (e) {
    const status = e?.response?.status;
    const errorText = e?.response?.data?.error || t('console.vps.resizeFailed');
    if (status === 409) {
      Modal.confirm({
        title: t('console.vps.resizeInProgress'),
        content: errorText,
        okText: t('console.vps.goOrders'),
        cancelText: t('console.vps.gotIt'),
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
    message.warning(t('console.vps.refundReasonRequired'));
    return;
  }
  if (String(refundReason.value || "").length > INPUT_LIMITS.REFUND_REASON) {
    message.warning(t('console.vps.refundReasonTooLong', { n: INPUT_LIMITS.REFUND_REASON }));
    return;
  }
  refunding.value = true;
  try {
    const res = await requestVpsRefund(activeRecord.value.id, { reason: refundReason.value });
    const orderId = res?.data?.order?.id ?? res?.data?.order?.ID;
    if (orderId) {
      message.success(t('console.vps.refundSubmittedId', { id: orderId }));
    } else {
      message.success(t('console.vps.refundSubmitted'));
    }
    refundOpen.value = false;
  } catch (e) {
    message.error(e?.response?.data?.error || e?.response?.data?.message || e?.message || t('console.vps.submitFailed'));
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

/* Page header: themed surface instead of a fixed blue banner. */
.page-header-gradient {
  position: relative;
  padding: 22px 24px;
  background: var(--theme-surface, var(--card));
  border-bottom: 1px solid var(--border);
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
  gap: 14px;
}

.page-title-icon {
  display: inline-flex;
  width: 42px;
  height: 42px;
  box-sizing: border-box;
  padding: 9px;
  border: 1px solid color-mix(in srgb, var(--primary) 18%, transparent);
  border-radius: var(--theme-control-radius, 10px);
  background: color-mix(in srgb, var(--primary) 10%, transparent);
  color: var(--primary);
}

.page-title {
  margin: 0;
  color: var(--text-primary);
  font-size: 22px;
  font-weight: 650;
  letter-spacing: -.02em;
}

.page-desc {
  margin: 3px 0 0;
  color: var(--text-secondary);
  font-size: 13px;
}

.page-header-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.action-btn,
.action-btn-primary {
  height: 36px;
  padding: 0 16px;
  border-radius: var(--theme-btn-radius, var(--theme-control-radius, 8px));
  font-size: 13px;
  font-weight: 550;
  transition: color 140ms ease, background 140ms ease, border-color 140ms ease;
}

.action-btn {
  border: 1px solid var(--border);
  background: var(--theme-control-bg, var(--card));
  color: var(--text-primary);
}

.action-btn:hover {
  border-color: var(--primary);
  background: var(--theme-control-hover, var(--card));
  color: var(--primary);
}

.action-btn-primary {
  border: 1px solid var(--primary);
  background: var(--primary);
  color: var(--theme-primary-foreground, #fff);
}

.action-btn-primary:hover {
  border-color: var(--theme-primary-hover, var(--primary));
  background: var(--theme-primary-hover, var(--primary));
  color: var(--theme-primary-foreground, #fff);
}

/* Statistics cards: token surfaces, semantic accent per state. */
.stats-container {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 14px;
  padding: 20px 24px;
  max-width: 1400px;
  margin: 0 auto;
}

.stat-card {
  position: relative;
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px 18px;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--theme-card-radius, 12px);
  background: var(--card);
  transition: border-color 160ms ease, background 160ms ease;
}

.stat-card:hover {
  border-color: color-mix(in srgb, var(--stat-accent) 45%, var(--border));
  background: color-mix(in srgb, var(--stat-accent) 5%, var(--card));
}

.stat-icon-wrapper {
  display: flex;
  width: 40px;
  height: 40px;
  align-items: center;
  justify-content: center;
  flex: 0 0 40px;
  border: 1px solid color-mix(in srgb, var(--stat-accent) 24%, transparent);
  border-radius: var(--theme-control-radius, 10px);
  background: color-mix(in srgb, var(--stat-accent) 12%, transparent);
}

.stat-icon {
  font-size: 18px;
  color: var(--stat-accent);
}

.stat-content {
  min-width: 0;
}

.stat-value {
  margin-bottom: 2px;
  color: var(--text-primary);
  font-size: 26px;
  font-weight: 650;
  line-height: 1.1;
  font-variant-numeric: tabular-nums;
  letter-spacing: -.02em;
}

.stat-label {
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 500;
  letter-spacing: 0;
  text-transform: none;
}

.stat-running { --stat-accent: #3f9d43; }
.stat-stopped { --stat-accent: #8a94a6; }
.stat-provisioning { --stat-accent: var(--primary); }
.stat-expiring { --stat-accent: #d94a4a; }

/* Filter Bar Card Override */
.stats-container + :deep(.card) {
  border-radius: var(--theme-card-radius, 16px);
  overflow: hidden;
}

.stats-container + :deep(.card .card-body) {
  padding: 16px;
}

/* Table: the table inherits the active theme's density and surfaces. */
:deep(.ant-table) {
  background: transparent;
}

:deep(.ant-table-container) {
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--theme-card-radius, 12px);
}

:deep(.ant-table-thead > tr > th) {
  padding: 12px 16px;
  background: var(--theme-table-head-bg, var(--bg-secondary));
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0;
  text-transform: none;
  border-bottom: 1px solid var(--border);
}

:deep(.ant-table-tbody > tr) {
  transition: background-color 140ms ease;
}

:deep(.ant-table-tbody > tr:hover > td) {
  background: var(--theme-table-row-hover, var(--bg-secondary)) !important;
}

:deep(.ant-table-tbody > tr > td) {
  padding: 13px 16px;
  color: var(--text-primary);
  border-bottom: 1px solid var(--border);
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
  background: color-mix(in srgb, var(--primary) 12%, transparent);
  border-radius: 8px;
}

.name-text {
  font-weight: 600;
  color: var(--text-primary);
  font-size: 14px;
}

.name-id {
  font-size: 11px;
  color: var(--text-secondary);
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
  color: #d94a4a;
  background: color-mix(in srgb, #d94a4a 12%, transparent);
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
  background: var(--card);
  border-radius: var(--theme-card-radius, 14px);
  overflow: hidden;
  box-shadow: var(--theme-shadow, none);
  transition: all 0.3s ease;
  cursor: pointer;
  border: 1px solid var(--border);
}

.vps-mobile-card:hover {
  border-color: color-mix(in srgb, var(--primary) 40%, var(--border));
  box-shadow: var(--theme-shadow, none);
}

.vps-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 16px 12px;
  background: var(--theme-surface-soft, var(--bg-secondary));
  border-bottom: 1px solid var(--border);
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
  color: var(--text-primary);
}

.vps-card-body {
  padding: 12px 16px;
}

.vps-info-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid var(--border);
}

.vps-info-row:last-child {
  border-bottom: none;
}

.vps-info-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-secondary);
  font-weight: 500;
}

.info-icon {
  font-size: 14px;
  color: var(--text-secondary);
}

.info-icon.warning {
  color: #ff4d4f;
}

.vps-info-value {
  font-size: 13px;
  color: var(--text-primary);
  font-weight: 500;
}

.vps-spec-tag {
  font-weight: 500;
  border-radius: 6px;
}

.vps-destroy-row {
  background: color-mix(in srgb, #d94a4a 10%, transparent);
  margin: 8px -16px -12px;
  padding: 10px 16px !important;
  border-radius: 0 0 12px 12px;
  border-bottom: none !important;
}

.vps-card-actions {
  display: flex;
  border-top: 1px solid var(--border);
  background: var(--theme-surface-soft, var(--bg-secondary));
}

.vps-action-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 14px;
  color: var(--text-secondary);
  font-size: 18px;
  transition: color 140ms ease, background 140ms ease;
  border-right: 1px solid var(--border);
}

.vps-action-btn:last-child {
  border-right: none;
}

.vps-action-btn:hover {
  background: color-mix(in srgb, var(--primary) 10%, transparent);
  color: var(--primary);
}

.vps-action-urgent {
  color: #ff4d4f;
}

.vps-action-urgent:hover {
  background: color-mix(in srgb, #d94a4a 10%, transparent);
  color: #b93a3a;
}

.urgent-renew-btn {
  height: 28px;
  padding: 0 10px;
}

.vps-action-more:hover {
  background: var(--bg-secondary);
  color: var(--primary);
}

/* Action Grid in Drawer */
.action-grid-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 16px;
  background: var(--bg-secondary);
  border-radius: var(--theme-control-radius, 10px);
  cursor: pointer;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
  transition: color 140ms ease, background 140ms ease, border-color 140ms ease;
  border: 1px solid transparent;
}

.action-grid-item:hover {
  background: color-mix(in srgb, var(--primary) 10%, var(--card));
  color: var(--primary);
  border-color: color-mix(in srgb, var(--primary) 35%, var(--border));
}

.action-grid-item.danger {
  color: #ff4d4f;
}

.action-grid-item.danger:hover {
  background: color-mix(in srgb, #d94a4a 10%, var(--card));
  border-color: color-mix(in srgb, #d94a4a 35%, var(--border));
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
  border-radius: var(--theme-card-radius, 16px);
  overflow: hidden;
}

:deep(.ant-modal-header) {
  background: var(--theme-surface-soft, var(--bg-secondary));
  border-bottom: 1px solid var(--border);
  padding: 20px 24px;
}

:deep(.ant-modal-title) {
  font-weight: 600;
  font-size: 16px;
  color: var(--text-primary);
}

:deep(.ant-modal-body) {
  padding: 24px;
}

:deep(.ant-modal-footer) {
  border-top: 1px solid var(--border);
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
