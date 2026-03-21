<template>
  <ElDrawer
    v-model="drawerVisible"
    title="VPS 详情"
    size="900px"
    destroy-on-close
    class="vps-detail-drawer"
  >
    <div v-loading="loading" class="detail-panel">
      <div class="detail-hero">
        <div class="detail-heading">
          <h3>{{ detail?.name || `VPS #${detail?.id || '-'}` }}</h3>
          <p>{{ detail?.automation_instance_id || '未绑定自动化实例 ID' }}</p>
          <div class="hero-tags">
            <ElTag :type="getVpsStatusTagType(detail?.status)">{{
              getVpsStatusText(detail?.status)
            }}</ElTag>
            <ElTag :type="getAdminStatusTagType(detail?.admin_status)">
              {{ getAdminStatusText(detail?.admin_status) }}
            </ElTag>
          </div>
        </div>

        <div class="hero-actions">
          <ElButton v-if="canRefresh" @click="emit('refresh')">刷新</ElButton>
          <ElButton v-if="canUpdate" type="primary" @click="emit('edit')">编辑</ElButton>
        </div>
      </div>

      <div class="action-grid">
        <ElButton v-if="canUpdateStatus" @click="emit('status')">设置状态</ElButton>
        <ElButton v-if="canUpdateExpire" @click="emit('expire')">修改到期</ElButton>
        <ElButton v-if="canResize" @click="emit('resize')">改配</ElButton>
        <ElButton v-if="canEmergencyRenew" @click="emit('renew')">紧急续费</ElButton>
        <ElButton v-if="canLock" @click="emit('lock')">锁定</ElButton>
        <ElButton v-if="canUnlock" @click="emit('unlock')">解锁</ElButton>
        <ElButton v-if="canDelete" type="danger" @click="emit('delete')">删除</ElButton>
      </div>

      <ElDescriptions :column="2" border>
        <ElDescriptionsItem label="实例 ID">{{ detail?.id || '-' }}</ElDescriptionsItem>
        <ElDescriptionsItem label="用户 ID">{{ detail?.user_id || '-' }}</ElDescriptionsItem>
        <ElDescriptionsItem label="订单项 ID">{{
          detail?.order_item_id || '-'
        }}</ElDescriptionsItem>
        <ElDescriptionsItem label="商品类型 ID">{{
          detail?.goods_type_id || '-'
        }}</ElDescriptionsItem>
        <ElDescriptionsItem label="地区">{{ detail?.region || '-' }}</ElDescriptionsItem>
        <ElDescriptionsItem label="线路 ID">{{ detail?.line_id || '-' }}</ElDescriptionsItem>
        <ElDescriptionsItem label="套餐">{{ detail?.package_name || '-' }}</ElDescriptionsItem>
        <ElDescriptionsItem label="套餐 ID">{{ detail?.package_id || '-' }}</ElDescriptionsItem>
        <ElDescriptionsItem label="月费">{{
          formatPrice(detail?.monthly_price)
        }}</ElDescriptionsItem>
        <ElDescriptionsItem label="系统镜像 ID">{{ detail?.system_id || '-' }}</ElDescriptionsItem>
        <ElDescriptionsItem label="CPU">{{ detail?.cpu || 0 }} 核</ElDescriptionsItem>
        <ElDescriptionsItem label="内存">{{ detail?.memory_gb || 0 }} GB</ElDescriptionsItem>
        <ElDescriptionsItem label="磁盘">{{ detail?.disk_gb || 0 }} GB</ElDescriptionsItem>
        <ElDescriptionsItem label="带宽">{{ detail?.bandwidth_mbps || 0 }} Mbps</ElDescriptionsItem>
        <ElDescriptionsItem label="端口数">{{ detail?.port_num || 0 }}</ElDescriptionsItem>
        <ElDescriptionsItem label="到期时间">{{
          formatDateTime(detail?.expire_at)
        }}</ElDescriptionsItem>
        <ElDescriptionsItem label="销毁时间">{{
          formatDateTime(detail?.destroy_at)
        }}</ElDescriptionsItem>
        <ElDescriptionsItem label="距离销毁">{{
          formatDestroyInDays(detail?.destroy_in_days)
        }}</ElDescriptionsItem>
        <ElDescriptionsItem label="紧急续费时间">
          {{ formatDateTime(detail?.last_emergency_renew_at) }}
        </ElDescriptionsItem>
        <ElDescriptionsItem label="创建时间">{{
          formatDateTime(detail?.created_at)
        }}</ElDescriptionsItem>
        <ElDescriptionsItem label="更新时间">{{
          formatDateTime(detail?.updated_at)
        }}</ElDescriptionsItem>
        <ElDescriptionsItem label="面板缓存地址" :span="2">
          {{ detail?.panel_url_cache || '-' }}
        </ElDescriptionsItem>
      </ElDescriptions>

      <section class="detail-section">
        <div class="section-title">规格参数</div>
        <pre class="json-box">{{ formatJson(detail?.spec) }}</pre>
      </section>

      <section class="detail-section">
        <div class="section-title">访问信息</div>
        <pre class="json-box">{{ formatJson(detail?.access_info) }}</pre>
      </section>
    </div>
  </ElDrawer>
</template>

<script setup lang="ts">
  interface VpsDetailData {
    id: number | null
    user_id: number | null
    order_item_id: number | null
    goods_type_id: number | null
    automation_instance_id: string
    name: string
    region: string
    region_id: number | null
    line_id: number | null
    package_id: number | null
    package_name: string
    cpu: number
    memory_gb: number
    disk_gb: number
    bandwidth_mbps: number
    port_num: number
    monthly_price: number
    spec: Record<string, unknown>
    system_id: number | null
    status: string
    automation_state: number | null
    admin_status: string
    expire_at: string | null
    destroy_at: string | null
    destroy_in_days: number | null
    panel_url_cache: string
    access_info: Record<string, unknown>
    last_emergency_renew_at: string | null
    created_at: string
    updated_at: string
  }

  interface Props {
    visible: boolean
    detail?: VpsDetailData | null
    loading?: boolean
    canRefresh?: boolean
    canUpdate?: boolean
    canDelete?: boolean
    canLock?: boolean
    canUnlock?: boolean
    canResize?: boolean
    canEmergencyRenew?: boolean
    canUpdateStatus?: boolean
    canUpdateExpire?: boolean
  }

  interface Emits {
    (e: 'update:visible', value: boolean): void
    (e: 'edit'): void
    (e: 'status'): void
    (e: 'resize'): void
    (e: 'expire'): void
    (e: 'refresh'): void
    (e: 'renew'): void
    (e: 'lock'): void
    (e: 'unlock'): void
    (e: 'delete'): void
  }

  defineOptions({ name: 'VpsDetailDrawer' })

  const props = withDefaults(defineProps<Props>(), {
    detail: null,
    loading: false,
    canRefresh: false,
    canUpdate: false,
    canDelete: false,
    canLock: false,
    canUnlock: false,
    canResize: false,
    canEmergencyRenew: false,
    canUpdateStatus: false,
    canUpdateExpire: false
  })

  const emit = defineEmits<Emits>()

  const drawerVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })

  function getVpsStatusText(status?: string | null) {
    switch (status) {
      case 'running':
        return '运行中'
      case 'stopped':
        return '已关机'
      case 'provisioning':
        return '开通中'
      case 'reinstalling':
        return '重装中'
      case 'reinstall_failed':
        return '重装失败'
      case 'locked':
        return '已锁定'
      case 'failed':
        return '异常'
      case 'deleting':
        return '删除中'
      case 'expired_locked':
        return '已到期'
      default:
        return status || '-'
    }
  }

  function getVpsStatusTagType(status?: string | null) {
    switch (status) {
      case 'running':
        return 'success' as const
      case 'provisioning':
      case 'reinstalling':
      case 'deleting':
        return 'warning' as const
      case 'reinstall_failed':
      case 'locked':
      case 'failed':
      case 'expired_locked':
        return 'danger' as const
      default:
        return 'info' as const
    }
  }

  function getAdminStatusText(status?: string | null) {
    switch (status) {
      case 'normal':
        return 'normal'
      case 'abuse':
        return 'abuse'
      case 'fraud':
        return 'fraud'
      case 'locked':
        return 'locked'
      default:
        return status || '-'
    }
  }

  function getAdminStatusTagType(status?: string | null) {
    switch (status) {
      case 'normal':
        return 'success' as const
      case 'abuse':
        return 'warning' as const
      case 'fraud':
      case 'locked':
        return 'danger' as const
      default:
        return 'info' as const
    }
  }

  function formatPrice(value?: number | null) {
    return Number(value || 0).toFixed(2)
  }

  function formatDateTime(value?: string | null) {
    if (!value) {
      return '-'
    }

    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }

  function formatDestroyInDays(value?: number | null) {
    if (value === null || value === undefined) {
      return '-'
    }

    if (Number(value) <= 0) {
      return '今天'
    }

    return `${Number(value)} 天`
  }

  function formatJson(value?: Record<string, unknown> | null) {
    if (!value || !Object.keys(value).length) {
      return '-'
    }

    try {
      return JSON.stringify(value, null, 2)
    } catch {
      return String(value)
    }
  }
</script>

<style scoped lang="scss">
  .detail-panel {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .detail-hero {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    padding: 18px;
    border-radius: 12px;
    background: linear-gradient(135deg, rgb(14 165 233 / 10%), rgb(59 130 246 / 8%));
  }

  .detail-heading h3 {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: 22px;
    font-weight: 700;
  }

  .detail-heading p {
    margin: 6px 0 0;
    color: var(--el-text-color-secondary);
  }

  .hero-tags {
    display: flex;
    gap: 8px;
    margin-top: 12px;
    flex-wrap: wrap;
  }

  .hero-actions {
    display: flex;
    gap: 10px;
    flex-wrap: wrap;
    justify-content: flex-end;
  }

  .action-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
  }

  .detail-section {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .section-title {
    color: var(--el-text-color-primary);
    font-size: 15px;
    font-weight: 600;
  }

  .json-box {
    overflow: auto;
    margin: 0;
    padding: 14px;
    border: 1px solid var(--el-border-color-light);
    border-radius: 12px;
    background: var(--el-fill-color-lighter);
    color: var(--el-text-color-primary);
    font-size: 12px;
    line-height: 1.7;
    white-space: pre-wrap;
    word-break: break-word;
  }

  @media (max-width: 768px) {
    .detail-hero {
      flex-direction: column;
    }

    .hero-actions {
      justify-content: flex-start;
    }
  }
</style>
