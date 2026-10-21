<template>
  <ElDrawer
    v-model="drawerVisible"
    title="用户详情"
    size="760px"
    destroy-on-close
    class="user-detail-drawer"
  >
    <div v-loading="loading" class="detail-panel">
      <div class="detail-hero">
        <div class="detail-user">
          <ElAvatar :size="56" :src="user?.avatar">
            {{ user?.username?.slice(0, 1)?.toUpperCase() || 'U' }}
          </ElAvatar>
          <div class="detail-user-meta">
            <h3>{{ user?.username || '-' }}</h3>
            <p>{{ user?.email || '-' }}</p>
          </div>
        </div>

        <ElButton type="primary" :disabled="!canImpersonate" @click="emit('impersonate')">
          以此用户登录
        </ElButton>
      </div>

      <ElDescriptions :column="2" border>
        <ElDescriptionsItem label="用户 ID">{{ user?.id || '-' }}</ElDescriptionsItem>
        <ElDescriptionsItem label="状态">
          <ElTag :type="getUserStatusTagType(user?.status)">{{ user?.status || '-' }}</ElTag>
        </ElDescriptionsItem>
        <ElDescriptionsItem label="邮箱">{{ user?.email || '-' }}</ElDescriptionsItem>
        <ElDescriptionsItem label="QQ">{{ user?.qq || '-' }}</ElDescriptionsItem>
        <ElDescriptionsItem label="用户组">{{ resolvedTierGroupName }}</ElDescriptionsItem>
        <ElDescriptionsItem label="到期时间">
          {{ formatDateTime(user?.userTierExpireAt) }}
        </ElDescriptionsItem>
        <ElDescriptionsItem label="注册时间">{{
          formatDateTime(user?.createdAt)
        }}</ElDescriptionsItem>
        <ElDescriptionsItem label="更新时间">{{
          formatDateTime(user?.updatedAt)
        }}</ElDescriptionsItem>
      </ElDescriptions>

      <section class="detail-section">
        <div class="section-title">实名认证</div>

        <div class="realname-toolbar">
          <ElSelect v-model="realnameStatusModel" placeholder="选择实名状态" style="width: 180px">
            <ElOption label="待审核" value="pending" />
            <ElOption label="已通过" value="verified" />
            <ElOption label="未通过" value="failed" />
          </ElSelect>

          <ElInput v-model="realnameReasonModel" placeholder="审核备注（可选）" :maxlength="1000" />

          <ElButton
            type="primary"
            :loading="realnameUpdating"
            :disabled="!realnameRecord"
            @click="emit('update-realname-status')"
          >
            更新实名状态
          </ElButton>
        </div>

        <div v-if="!realnameRecord" class="detail-muted">暂无实名认证记录，无法修改状态</div>

        <ElDescriptions v-else :column="2" border size="small">
          <ElDescriptionsItem label="当前状态">
            <ElTag :type="getRealnameTagType(realnameRecord.status)">
              {{ realnameRecord.status || '-' }}
            </ElTag>
          </ElDescriptionsItem>
          <ElDescriptionsItem label="认证渠道">
            {{ realnameRecord.provider || '-' }}
          </ElDescriptionsItem>
          <ElDescriptionsItem label="提交时间">
            {{ formatDateTime(realnameRecord.created_at) }}
          </ElDescriptionsItem>
          <ElDescriptionsItem label="审核时间">
            {{ formatDateTime(realnameRecord.verified_at) }}
          </ElDescriptionsItem>
          <ElDescriptionsItem label="备注" :span="2">
            {{ realnameRecord.reason || '-' }}
          </ElDescriptionsItem>
        </ElDescriptions>
      </section>

      <section class="detail-section">
        <div class="section-title">钱包余额</div>
        <div class="wallet-card">
          <div class="wallet-amount">￥{{ formatAmount(walletInfo?.balance) }}</div>
          <div class="detail-muted">
            {{ walletInfo?.currency ? `币种：${walletInfo.currency}` : '币种：CNY' }}
          </div>
        </div>
      </section>

      <section class="detail-section">
        <div class="section-title">订单记录</div>
        <ElTable :data="orderRecords" border size="small" row-key="id">
          <ElTableColumn label="订单号" min-width="160">
            <template #default="{ row }">
              {{ row.order_no || row.id || '-' }}
            </template>
          </ElTableColumn>
          <ElTableColumn prop="status" label="状态" min-width="120" />
          <ElTableColumn label="金额" min-width="120">
            <template #default="{ row }">￥{{ formatAmount(row.total_amount) }}</template>
          </ElTableColumn>
          <ElTableColumn label="创建时间" min-width="180">
            <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
          </ElTableColumn>
        </ElTable>
      </section>

      <section class="detail-section">
        <div class="section-title">钱包记录</div>
        <ElTable :data="walletTransactions" border size="small" row-key="id">
          <ElTableColumn prop="type" label="类型" min-width="120" />
          <ElTableColumn label="金额" min-width="120">
            <template #default="{ row }">￥{{ formatAmount(row.amount) }}</template>
          </ElTableColumn>
          <ElTableColumn prop="note" label="备注" min-width="180" show-overflow-tooltip />
          <ElTableColumn label="时间" min-width="180">
            <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
          </ElTableColumn>
        </ElTable>
      </section>
    </div>
  </ElDrawer>
</template>

<script setup lang="ts">
  import type {
    OrderRecord,
    RealNameVerification,
    UserTierGroup,
    WalletInfo,
    WalletTransaction
  } from '@/api/admin'

  defineOptions({ name: 'UserDetailDrawer' })

  interface UserDetailData {
    id: number | null
    username: string
    email: string
    qq: string
    avatar: string
    status: string
    role: string
    createdAt: string
    updatedAt: string
    userTierGroupId: number | null
    userTierExpireAt: string | null
    userTierGroupName: string
  }

  interface Props {
    visible: boolean
    user?: UserDetailData | null
    loading?: boolean
    tierGroups?: UserTierGroup[]
    walletInfo?: WalletInfo | null
    orderRecords?: OrderRecord[]
    walletTransactions?: WalletTransaction[]
    realnameRecord?: RealNameVerification | null
    realnameStatus?: string
    realnameReason?: string
    realnameUpdating?: boolean
  }

  interface Emits {
    (e: 'update:visible', value: boolean): void
    (e: 'update:realnameStatus', value: string): void
    (e: 'update:realnameReason', value: string): void
    (e: 'impersonate'): void
    (e: 'update-realname-status'): void
  }

  const props = withDefaults(defineProps<Props>(), {
    user: null,
    loading: false,
    tierGroups: () => [],
    walletInfo: null,
    orderRecords: () => [],
    walletTransactions: () => [],
    realnameRecord: null,
    realnameStatus: '',
    realnameReason: '',
    realnameUpdating: false
  })
  const emit = defineEmits<Emits>()

  const drawerVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })

  const realnameStatusModel = computed({
    get: () => props.realnameStatus,
    set: (value) => emit('update:realnameStatus', value)
  })

  const realnameReasonModel = computed({
    get: () => props.realnameReason,
    set: (value) => emit('update:realnameReason', value)
  })

  const resolvedTierGroupName = computed(() => {
    if (props.user?.userTierGroupName) {
      return props.user.userTierGroupName
    }

    if (props.user?.userTierGroupId === null || props.user?.userTierGroupId === undefined) {
      return '-'
    }

    const matched = props.tierGroups.find(
      (group) => Number(group.id) === Number(props.user?.userTierGroupId)
    )
    return String(matched?.name || '-')
  })

  const canImpersonate = computed(() => {
    return (
      Boolean(props.user?.id) && props.user?.role !== 'admin' && props.user?.status === 'active'
    )
  })

  function formatDateTime(value?: string | null) {
    if (!value) {
      return '-'
    }

    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }

  function formatAmount(value?: number | null) {
    return Number(value || 0).toFixed(2)
  }

  function getUserStatusTagType(status?: string) {
    if (status === 'active') {
      return 'success' as const
    }

    if (status === 'blocked') {
      return 'danger' as const
    }

    return 'info' as const
  }

  function getRealnameTagType(status?: string) {
    if (status === 'verified') {
      return 'success' as const
    }

    if (status === 'failed') {
      return 'danger' as const
    }

    return 'warning' as const
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
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: 18px;
    border-radius: 12px;
    background: linear-gradient(135deg, rgb(34 197 94 / 12%), rgb(59 130 246 / 10%));
  }

  .detail-user {
    display: flex;
    align-items: center;
    gap: 14px;
    min-width: 0;
  }

  .detail-user-meta {
    min-width: 0;
  }

  .detail-user-meta h3 {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: 20px;
    font-weight: 700;
  }

  .detail-user-meta p {
    margin: 4px 0 0;
    color: var(--el-text-color-secondary);
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

  .realname-toolbar {
    display: grid;
    grid-template-columns: 180px minmax(0, 1fr) auto;
    gap: 12px;
    align-items: center;
  }

  .wallet-card {
    padding: 18px;
    border: 1px solid var(--el-border-color-light);
    border-radius: 12px;
    background: var(--el-bg-color-page);
  }

  .wallet-amount {
    color: var(--el-text-color-primary);
    font-size: 28px;
    font-weight: 700;
    line-height: 1;
  }

  .detail-muted {
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.6;
  }

  @media (max-width: 768px) {
    .detail-hero {
      flex-direction: column;
      align-items: flex-start;
    }

    .realname-toolbar {
      grid-template-columns: 1fr;
    }
  }
</style>
