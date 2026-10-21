<template>
  <div v-loading="loading" class="payment-plugins-page art-full-height">
    <div class="page-header">
      <div>
        <div class="page-kicker">Billing</div>
        <div class="page-title">Payment Plugin Upload</div>
        <div class="page-subtitle">
          Upload payment plugin packages and control the top-level enabled state of payment
          providers.
        </div>
      </div>

      <div class="page-actions">
        <ElButton v-if="canView" :disabled="loading" @click="fetchData">Refresh</ElButton>
        <ElUpload
          v-if="canUpload"
          :auto-upload="false"
          :show-file-list="false"
          accept=".zip"
          @change="handleFileSelect"
        >
          <ElButton type="primary" :loading="uploading">Upload Plugin</ElButton>
        </ElUpload>
      </div>
    </div>

    <ElEmpty
      v-if="!canView"
      description="You do not have permission to view payment plugin settings."
    />

    <template v-else>
      <ElAlert
        type="info"
        :closable="false"
        show-icon
        title="Scope Of This Page"
        description="This page keeps the legacy payment-plugin upload workflow. Advanced plugin management now lives in Plugin Management, and scene-level switches live in Payment Settings."
      />

      <ElCard shadow="never" class="section-card">
        <template #header>
          <div class="section-title">Payment Providers</div>
        </template>

        <ElTable :data="rows" border row-key="key">
          <ElTableColumn prop="name" label="Name" min-width="220" />
          <ElTableColumn prop="key" label="Key" min-width="240" show-overflow-tooltip />

          <ElTableColumn label="Type" width="120">
            <template #default="{ row }">
              <ElTag :type="row.is_plugin ? 'warning' : 'info'">
                {{ row.is_plugin ? 'Plugin' : 'Built-in' }}
              </ElTag>
            </template>
          </ElTableColumn>

          <ElTableColumn label="Enabled" width="120">
            <template #default="{ row }">
              <ElSwitch
                :model-value="row.enabled"
                :disabled="!canUpdate"
                :loading="busyKey === row.key"
                @change="handleToggle(row, $event)"
              />
            </template>
          </ElTableColumn>

          <ElTableColumn label="Actions" width="170" fixed="right">
            <template #default="{ row }">
              <div class="table-actions">
                <ElButton link type="primary" @click="openProviderHint(row)">Manage</ElButton>
              </div>
            </template>
          </ElTableColumn>
        </ElTable>
      </ElCard>
    </template>

    <ElDialog
      v-model="uploadDialogVisible"
      title="Upload Payment Plugin"
      width="460px"
      destroy-on-close
      align-center
    >
      <ElForm label-position="top">
        <ElFormItem label="Selected File">
          <ElInput :model-value="selectedFile?.name || ''" readonly />
        </ElFormItem>

        <ElFormItem label="Upload Password">
          <ElInput
            v-model="uploadPassword"
            type="password"
            show-password
            placeholder="Enter payment plugin upload password"
          />
        </ElFormItem>
      </ElForm>

      <template #footer>
        <div class="dialog-footer">
          <ElButton @click="uploadDialogVisible = false">Cancel</ElButton>
          <ElButton type="primary" :loading="uploading" @click="submitUpload">Upload</ElButton>
        </div>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import type { PaymentProviderRecord } from '@/api/admin'
  import {
    fetchAdminPaymentProviders,
    hasAdminPermission,
    updateAdminPaymentProvider,
    uploadAdminPaymentPlugin
  } from '@/api/admin'
  import { useUserStore } from '@/store/modules/user'
  import { ElMessage, type UploadFile } from 'element-plus'

  defineOptions({ name: 'SettingsPaymentPluginsPage' })

  interface PaymentPluginRow {
    key: string
    name: string
    enabled: boolean
    is_plugin: boolean
  }

  const loading = ref(false)
  const uploading = ref(false)
  const busyKey = ref('')

  const rows = ref<PaymentPluginRow[]>([])
  const uploadDialogVisible = ref(false)
  const selectedFile = ref<File | null>(null)
  const uploadPassword = ref('')

  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const canView = computed(() => hasAdminPermission(info.value?.buttons, ['payment.list']))
  const canUpdate = computed(() => hasAdminPermission(info.value?.buttons, ['payment.update']))
  const canUpload = computed(() => hasAdminPermission(info.value?.buttons, ['plugin.upload']))

  watch(
    canView,
    (value) => {
      if (value && !rows.value.length) {
        fetchData()
      }
    },
    { immediate: true }
  )

  function normalizeRow(item?: PaymentProviderRecord): PaymentPluginRow {
    const key = String(item?.key || '')

    return {
      key,
      name: String(item?.name || key || ''),
      enabled: Boolean(item?.enabled),
      is_plugin: key.includes('.')
    }
  }

  async function fetchData() {
    if (!canView.value) {
      return
    }

    loading.value = true

    try {
      const payload = await fetchAdminPaymentProviders({
        include_disabled: true,
        include_legacy: true
      })

      rows.value = (payload.items || []).map((item) => normalizeRow(item))
    } finally {
      loading.value = false
    }
  }

  async function handleToggle(row: PaymentPluginRow, checked: boolean | string | number) {
    busyKey.value = row.key

    try {
      await updateAdminPaymentProvider(row.key, {
        enabled: Boolean(checked)
      })

      row.enabled = Boolean(checked)
      ElMessage.success('Provider state updated successfully')
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.error || 'Failed to update provider state')
    } finally {
      busyKey.value = ''
    }
  }

  function handleFileSelect(uploadFile: UploadFile) {
    if (!uploadFile.raw) {
      return
    }

    selectedFile.value = uploadFile.raw
    uploadPassword.value = ''
    uploadDialogVisible.value = true
  }

  async function submitUpload() {
    if (!selectedFile.value) {
      ElMessage.error('Please select a plugin file first')
      return
    }

    if (!String(uploadPassword.value || '').trim()) {
      ElMessage.error('Please enter the upload password')
      return
    }

    uploading.value = true

    try {
      await uploadAdminPaymentPlugin(selectedFile.value, String(uploadPassword.value || '').trim())
      uploadDialogVisible.value = false
      selectedFile.value = null
      uploadPassword.value = ''
      ElMessage.success('Payment plugin uploaded successfully')
      await fetchData()
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.error || 'Failed to upload payment plugin')
    } finally {
      uploading.value = false
    }
  }

  function openProviderHint(row: PaymentPluginRow) {
    if (row.is_plugin) {
      ElMessage.info('Use Plugin Management for instance config and plugin lifecycle actions')
      return
    }

    ElMessage.info('Use Payment Settings for detailed provider scene controls')
  }
</script>

<style scoped lang="scss">
  .payment-plugins-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .page-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
  }

  .page-kicker {
    color: var(--el-text-color-placeholder);
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.18em;
    text-transform: uppercase;
  }

  .page-title {
    margin-top: 4px;
    color: var(--el-text-color-primary);
    font-size: 22px;
    font-weight: 700;
  }

  .page-subtitle {
    margin-top: 6px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.7;
  }

  .page-actions,
  .dialog-footer,
  .table-actions {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .section-card {
    border: none;
    border-radius: 18px;
  }

  .section-title {
    color: var(--el-text-color-primary);
    font-size: 16px;
    font-weight: 700;
  }

  .dialog-footer {
    justify-content: flex-end;
  }

  .table-actions {
    justify-content: flex-end;
  }

  @media (max-width: 768px) {
    .page-header {
      flex-direction: column;
      align-items: stretch;
    }

    .page-actions {
      flex-wrap: wrap;
    }
  }
</style>
