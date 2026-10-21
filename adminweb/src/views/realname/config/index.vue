<template>
  <div v-loading="loading" class="realname-config-page art-full-height">
    <div class="page-header">
      <div>
        <div class="page-kicker">Realname</div>
        <div class="page-title">实名认证配置</div>
        <div class="page-subtitle">
          配置实名开关、默认服务商，以及哪些关键操作必须先完成实名认证。
        </div>
      </div>

      <div class="page-actions">
        <ElButton v-if="canView" :disabled="loading || saving" @click="fetchData">刷新</ElButton>
        <ElButton v-if="canUpdate" type="primary" :loading="saving" @click="handleSave">
          保存配置
        </ElButton>
      </div>
    </div>

    <ElEmpty v-if="!canView" description="当前账号没有查看实名认证配置的权限。" />

    <template v-else>
      <ElRow :gutter="16">
        <ElCol :xs="24" :xl="14">
          <ElCard shadow="never" class="section-card">
            <template #header>
              <div class="section-title">实名开关</div>
            </template>

            <ElForm label-position="top" :disabled="formDisabled">
              <ElFormItem label="启用实名认证">
                <div class="switch-row">
                  <ElSwitch v-model="form.enabled" />
                  <span class="field-tip">
                    开启后，用户需要先完成实名认证，才能执行受限操作。
                  </span>
                </div>
              </ElFormItem>

              <ElFormItem label="认证服务商">
                <ElSelect
                  v-model="form.provider"
                  clearable
                  filterable
                  placeholder="请选择认证服务商"
                  class="full-width"
                >
                  <ElOption
                    v-for="item in providerOptions"
                    :key="item.key"
                    :label="getProviderLabel(item)"
                    :value="item.key || ''"
                  />
                </ElSelect>
                <div class="field-tip">
                  建议优先选择带完整实例标识的服务商，避免多实例时选错目标。
                </div>
              </ElFormItem>
            </ElForm>
          </ElCard>
        </ElCol>

        <ElCol :xs="24" :xl="10">
          <ElCard shadow="never" class="section-card section-card--accent">
            <template #header>
              <div class="section-title">受限操作</div>
            </template>

            <ElForm label-position="top" :disabled="formDisabled">
              <ElFormItem label="完成实名后才允许">
                <ElCheckboxGroup v-model="form.block_actions" class="checkbox-stack">
                  <ElCheckbox
                    v-for="item in blockActionOptions"
                    :key="item.value"
                    :label="item.value"
                  >
                    {{ item.label }}
                  </ElCheckbox>
                </ElCheckboxGroup>
                <div class="field-tip">
                  这里选择的操作会在前台统一拦截，要求用户先完成实名认证。
                </div>
              </ElFormItem>

              <ElAlert
                type="info"
                :closable="false"
                show-icon
                title="与用户管理联动"
                description="用户页里已经支持直接查看实名记录和修改实名状态，这里的配置会影响前台准入策略。"
              />
            </ElForm>
          </ElCard>
        </ElCol>
      </ElRow>
    </template>
  </div>
</template>

<script setup lang="ts">
  import type { RealNameConfigRecord, RealNameProviderRecord } from '@/api/admin'
  import {
    fetchRealNameConfig,
    fetchRealNameProviders,
    hasAdminPermission,
    updateRealNameConfig
  } from '@/api/admin'
  import { useUserStore } from '@/store/modules/user'
  import { ElMessage } from 'element-plus'

  defineOptions({ name: 'RealnameConfigPage' })

  interface RealnameConfigForm {
    enabled: boolean
    provider: string
    block_actions: string[]
  }

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const loading = ref(false)
  const saving = ref(false)
  const providerOptions = ref<RealNameProviderRecord[]>([])

  const form = reactive<RealnameConfigForm>({
    enabled: false,
    provider: '',
    block_actions: []
  })

  const blockActionOptions = [
    { label: '购买 VPS', value: 'purchase_vps' },
    { label: '续费 VPS', value: 'renew_vps' },
    { label: '升级/扩容 VPS', value: 'resize_vps' }
  ]

  const canView = computed(() => hasAdminPermission(info.value?.buttons, ['realname.view']))
  const canUpdate = computed(() => hasAdminPermission(info.value?.buttons, ['realname.update']))
  const formDisabled = computed(() => loading.value || saving.value || !canUpdate.value)

  onMounted(() => {
    fetchData()
  })

  function applyForm(data?: RealNameConfigRecord | null) {
    form.enabled = Boolean(data?.enabled)
    form.provider = String(data?.provider || '')
    form.block_actions = Array.isArray(data?.block_actions) ? [...data.block_actions] : []
  }

  async function fetchData() {
    if (!canView.value) {
      return
    }

    loading.value = true

    try {
      const [configPayload, providerPayload] = await Promise.all([
        fetchRealNameConfig(),
        fetchRealNameProviders()
      ])

      applyForm(configPayload)
      providerOptions.value = providerPayload.items || []
    } finally {
      loading.value = false
    }
  }

  async function handleSave() {
    saving.value = true

    try {
      await updateRealNameConfig({
        enabled: form.enabled,
        provider: String(form.provider || '').trim(),
        block_actions: form.block_actions
      })
      ElMessage.success('实名认证配置已保存')
    } finally {
      saving.value = false
    }
  }

  function getProviderLabel(provider?: RealNameProviderRecord | null) {
    const key = String(provider?.key || '')
    const name = String(provider?.name || '')

    if (!key) {
      return name || '-'
    }

    if (!name) {
      return key
    }

    return `${name} (${key})`
  }
</script>

<style scoped lang="scss">
  .page-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    margin-bottom: 16px;
  }

  .page-kicker,
  .page-subtitle,
  .field-tip {
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.7;
  }

  .page-kicker {
    margin-bottom: 6px;
    font-weight: 600;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .page-title,
  .section-title {
    color: var(--el-text-color-primary);
    font-weight: 700;
  }

  .page-title {
    font-size: 26px;
  }

  .page-subtitle {
    margin-top: 8px;
    max-width: 720px;
  }

  .page-actions {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .section-card {
    height: 100%;
  }

  .section-card--accent {
    background:
      linear-gradient(180deg, rgb(34 197 94 / 5%), rgb(34 197 94 / 0%)), var(--el-bg-color);
  }

  .section-title {
    font-size: 16px;
  }

  .switch-row {
    display: flex;
    gap: 12px;
    align-items: center;
  }

  .checkbox-stack {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .full-width {
    width: 100%;
  }

  @media (max-width: 768px) {
    .page-header,
    .page-actions,
    .switch-row {
      flex-direction: column;
      align-items: flex-start;
    }
  }
</style>
