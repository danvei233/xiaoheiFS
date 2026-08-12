<template>
  <div class="api-keys-page art-full-height">
    <div class="page-heading">
      <div>
        <h2>API 密钥管理</h2>
        <p>管理用于开放接口鉴权的 AKID/Key 凭证</p>
      </div>
      <ElButton type="primary" v-ripple :disabled="loading" @click="openCreateDialog">
        <ArtSvgIcon icon="ri:add-line" class="button-icon" />
        创建新密钥
      </ElButton>
    </div>

    <ElAlert title="签名鉴权规范" type="info" :closable="false" show-icon>
      <div class="header-list">
        <code>X-AKID</code>
        <code>X-Timestamp</code>
        <code>X-Nonce</code>
        <code>X-Signature</code>
        <ElTag type="success" size="small">时间窗: ±300 秒</ElTag>
      </div>
    </ElAlert>

    <ElCard class="art-table-card">
      <ArtTableHeader v-model:columns="columnChecks" :loading="loading" layout="">
        <template #left>
          <div class="table-heading">
            <div>
              <ArtSvgIcon icon="ri:key-2-line" />
              <span>密钥列表</span>
            </div>
            <span v-if="data.length">{{ data.length }} 个密钥</span>
          </div>
        </template>
      </ArtTableHeader>

      <ArtTable
        v-if="loading || data.length"
        :row-key="getKeyRowKey"
        :loading="loading"
        :data="data"
        :columns="columns"
        :show-table-header="true"
        empty-text="暂无 API 密钥"
      >
        <template #id="{ row }">
          <span>{{ getKeyId(row) }}</span>
        </template>

        <template #name="{ row }">
          <div class="name-cell">
            <span class="name-text">{{ getKeyName(row) }}</span>
            <span class="name-subtitle">{{ getKeyAkid(row) || '-' }}</span>
          </div>
        </template>

        <template #akid="{ row }">
          <div class="copy-cell">
            <code>{{ getKeyAkid(row) || '-' }}</code>
            <ElTooltip content="复制 AKID" placement="top">
              <ArtIconButton
                icon="ri:file-copy-line"
                @click="copyText(getKeyAkid(row), '已复制到剪贴板')"
              />
            </ElTooltip>
          </div>
        </template>

        <template #status="{ row }">
          <ElTag :type="isKeyActive(row) ? 'success' : 'info'" effect="light">
            {{ isKeyActive(row) ? '已启用' : '已停用' }}
          </ElTag>
        </template>

        <template #lastUsedAt="{ row }">
          <span class="muted-text">{{
            getLastUsedAt(row) ? formatTime(getLastUsedAt(row)) : '—'
          }}</span>
        </template>

        <template #operation="{ row }">
          <div class="operation-cell">
            <ElTooltip :content="isKeyActive(row) ? '停用密钥' : '启用密钥'" placement="top">
              <ArtButtonTable
                type="edit"
                :icon="isKeyActive(row) ? 'ri:stop-circle-line' : 'ri:play-circle-line'"
                @click="toggleStatus(row)"
              />
            </ElTooltip>
            <ElTooltip content="删除密钥" placement="top">
              <ArtButtonTable type="delete" @click="removeKey(row)" />
            </ElTooltip>
          </div>
        </template>
      </ArtTable>
      <ElEmpty v-else description="暂无 API 密钥" :image-size="120" class="empty-state">
        <ElButton link type="primary" :disabled="loading" @click="openCreateDialog">
          创建第一个密钥
        </ElButton>
      </ElEmpty>
    </ElCard>

    <ElCard class="art-card-xs code-card">
      <template #header>
        <div class="code-heading">
          <div>
            <ArtSvgIcon icon="ri:code-s-slash-line" />
            <span>签名算法示例</span>
          </div>
          <ElTag type="primary" effect="plain">Node.js</ElTag>
        </div>
      </template>
      <pre class="code-block">{{ signSnippet }}</pre>
    </ElCard>

    <ElDialog
      v-model="createDialogVisible"
      title="创建新 API 密钥"
      width="min(480px, 92vw)"
      align-center
      destroy-on-close
      @closed="resetCreateForm"
    >
      <p class="dialog-description">生成新的 AKID/Key 凭证对，Key 仅在创建时展示一次</p>
      <ElForm ref="createFormRef" :model="createForm" :rules="createRules" label-position="top">
        <ElFormItem label="密钥名称" prop="name">
          <ElInput
            v-model.trim="createForm.name"
            placeholder="例如：生产环境-订单服务"
            :maxlength="64"
            show-word-limit
            @keyup.enter="createKey"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="createDialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="creating" :disabled="loading" @click="createKey">
          创建密钥
        </ElButton>
      </template>
    </ElDialog>

    <ElDialog
      v-model="secretDialogVisible"
      width="min(560px, 92vw)"
      align-center
      :close-on-click-modal="false"
      destroy-on-close
      @closed="clearCreatedSecret"
    >
      <ElAlert
        title="重要提示"
        description="窗口关闭后将无法再次查看 Key，请务必立即复制保存"
        type="warning"
        show-icon
        :closable="false"
        class="dialog-alert"
      />
      <div class="secret-fields">
        <div class="secret-field">
          <span class="field-label">AKID (访问密钥ID)</span>
          <div class="field-row">
            <ElInput :model-value="createdSecret.akid" readonly />
            <ElButton @click="copyText(createdSecret.akid, '已复制到剪贴板')">
              <ArtSvgIcon icon="ri:file-copy-line" />
            </ElButton>
          </div>
        </div>
        <div class="secret-field secret-field-key">
          <span class="field-label">Key (访问密钥)</span>
          <div class="field-row">
            <ElInput :model-value="createdSecret.key" readonly />
            <ElButton type="primary" @click="copyText(createdSecret.key, 'Key 已复制到剪贴板')">
              <ArtSvgIcon icon="ri:file-copy-line" />
            </ElButton>
          </div>
        </div>
      </div>
      <template #footer>
        <ElButton @click="secretDialogVisible = false">关闭窗口</ElButton>
        <ElButton type="primary" @click="copyText(createdSecret.key, 'Key 已复制到剪贴板')"
          >复制 Key</ElButton
        >
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import type { FormInstance, FormRules } from 'element-plus'
  import { useTable } from '@/hooks/core/useTable'
  import {
    createUserApiKey,
    deleteUserApiKey,
    listUserApiKeys,
    updateUserApiKeyStatus
  } from '@/services/user'
  import type { UserAPIKey } from '@/services/types'

  defineOptions({ name: 'ConsoleApiKeys' })

  interface ApiKeyRecord extends UserAPIKey {
    ID?: number
    Name?: string
    AKID?: string
    Status?: string
    LastUsedAt?: string
    CreatedAt?: string
  }

  interface ApiKeyListPayload {
    items?: ApiKeyRecord[]
    Items?: ApiKeyRecord[]
  }

  const createFormRef = ref<FormInstance>()
  const createDialogVisible = ref(false)
  const secretDialogVisible = ref(false)
  const creating = ref(false)

  const createForm = reactive({ name: '' })
  const createdSecret = reactive({ akid: '', key: '' })
  const createRules: FormRules<typeof createForm> = {
    name: [
      { required: true, message: '请输入密钥名称', trigger: 'blur' },
      { max: 64, message: '密钥名称不能超过 64 个字符', trigger: 'blur' }
    ]
  }

  const getKeyId = (record: ApiKeyRecord): number | string => record.id ?? record.ID ?? '-'
  const getKeyRowKey = (record: Record<string, unknown>): string =>
    String(record.id ?? record.ID ?? '')
  const getKeyName = (record: ApiKeyRecord): string => record.name ?? record.Name ?? '-'
  const getKeyAkid = (record: ApiKeyRecord): string => record.akid ?? record.AKID ?? ''
  const getKeyStatus = (record: ApiKeyRecord): string => record.status ?? record.Status ?? ''
  const getLastUsedAt = (record: ApiKeyRecord): string =>
    record.last_used_at ?? record.LastUsedAt ?? ''
  const isKeyActive = (record: ApiKeyRecord): boolean => getKeyStatus(record) === 'active'

  const fetchApiKeys = async (_params: { current: number; size: number }, signal?: AbortSignal) => {
    void _params
    const response = await listUserApiKeys({ limit: 100, offset: 0 }, signal)
    const payload = response.data as ApiKeyListPayload | undefined
    const records = payload?.items ?? payload?.Items ?? []
    return { records, current: 1, size: records.length || 1, total: records.length }
  }

  const { columns, columnChecks, data, loading, refreshCreate } = useTable({
    core: {
      apiFn: fetchApiKeys,
      apiParams: { current: 1, size: 100 },
      columnsFactory: () => [
        { prop: 'id', label: 'ID', width: 70, useSlot: true },
        { prop: 'name', label: '名称', minWidth: 190, useSlot: true },
        { prop: 'akid', label: 'AKID', minWidth: 260, useSlot: true },
        { prop: 'status', label: '状态', width: 110, useSlot: true },
        { prop: 'lastUsedAt', label: '最近使用', width: 180, useSlot: true },
        { prop: 'operation', label: '操作', width: 110, fixed: 'right', useSlot: true }
      ]
    }
  })

  const signSnippet = `import crypto from "crypto";

const method = "POST";
const path = "/api/v1/open/orders/instant/create";
const query = "";
const ts = new Date().toISOString(); // RFC3339
const nonce = crypto.randomUUID().replace(/-/g, "");
const body = JSON.stringify({
  items: [{ package_id: 1, system_id: 1, qty: 1 }]
});

const bodyHash = crypto.createHash("sha256").update(body).digest("hex");
const canonical = [method.toUpperCase(), path, query, ts, nonce, bodyHash].join("\\n");
const sig = crypto.createHmac("sha256", process.env.OPEN_KEY)
                .update(canonical)
                .digest("hex");`

  const formatTime = (value: string): string => {
    if (!value) return '-'
    const date = new Date(value)
    if (Number.isNaN(date.getTime())) return value
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    })
  }

  const openCreateDialog = (): void => {
    if (loading.value) return
    createDialogVisible.value = true
  }

  const resetCreateForm = (): void => {
    createForm.name = ''
    createFormRef.value?.resetFields()
  }

  const createKey = async (): Promise<void> => {
    if (!createFormRef.value || loading.value || creating.value) return
    const valid = await createFormRef.value.validate().catch(() => false)
    if (!valid) return

    creating.value = true
    try {
      const response = await createUserApiKey({ name: createForm.name.trim(), scopes: [] })
      const payload = response.data as
        | {
            item?: ApiKeyRecord
            Item?: ApiKeyRecord
            key?: string
            Key?: string
            secret?: string
            Secret?: string
          }
        | undefined
      const item = payload?.item ?? payload?.Item
      createdSecret.akid = item ? getKeyAkid(item) : ''
      createdSecret.key = payload?.key ?? payload?.Key ?? payload?.secret ?? payload?.Secret ?? ''
      createDialogVisible.value = false
      secretDialogVisible.value = true
      if (item) {
        data.value = [item, ...data.value].slice(0, 100)
      } else {
        await refreshCreate()
      }
    } finally {
      creating.value = false
    }
  }

  const clearCreatedSecret = (): void => {
    createdSecret.akid = ''
    createdSecret.key = ''
  }

  const toggleStatus = async (record: ApiKeyRecord): Promise<void> => {
    const id = getKeyId(record)
    if (id === '-') return
    const nextStatus = isKeyActive(record) ? 'disabled' : 'active'
    await updateUserApiKeyStatus(id, { status: nextStatus })
    record.status = nextStatus
    ElMessage.success(nextStatus === 'active' ? '密钥已启用' : '密钥已停用')
  }

  const removeKey = async (record: ApiKeyRecord): Promise<void> => {
    const id = getKeyId(record)
    if (id === '-') return
    const confirmed = await ElMessageBox.confirm('确认删除该密钥？删除后无法恢复', '', {
      confirmButtonText: '确认删除',
      cancelButtonText: '取消',
      type: 'warning'
    }).catch(() => false)
    if (!confirmed) return
    await deleteUserApiKey(id)
    data.value = data.value.filter((item) => String(getKeyId(item)) !== String(id))
    ElMessage.success('密钥已删除')
  }

  const copyText = async (text: string, successMessage: string): Promise<void> => {
    if (!text) return
    await navigator.clipboard.writeText(text)
    ElMessage.success(successMessage)
  }
</script>

<style lang="scss" scoped>
  .page-heading {
    display: flex;
    flex: none;
    gap: 16px;
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: 16px;

    h2 {
      margin: 0;
      font-size: 22px;
      color: var(--art-gray-900);
      letter-spacing: 0;
    }

    p {
      margin: 6px 0 0;
      font-size: 13px;
      color: var(--art-gray-600);
    }
  }

  .button-icon {
    margin-right: 6px;
  }

  .table-heading,
  .code-heading {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    color: var(--art-gray-600);

    > div {
      display: flex;
      gap: 8px;
      align-items: center;
      font-weight: 600;
      color: var(--art-gray-900);

      > :first-child {
        color: var(--theme-color);
      }
    }
  }

  .table-heading > span {
    font-size: 12px;
  }

  .code-card {
    flex: none;
    margin-top: 16px;
  }

  .empty-state {
    flex: 1;
  }

  .dialog-description {
    margin: 0 0 18px;
    color: var(--art-gray-600);
  }

  .name-cell {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
  }

  .name-text {
    overflow: hidden;
    font-weight: 600;
    color: var(--art-gray-900);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .name-subtitle,
  .muted-text {
    font-size: 12px;
    color: var(--art-gray-600);
  }

  .name-subtitle {
    overflow: hidden;
    font-family: monospace;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .copy-cell {
    display: flex;
    gap: 6px;
    align-items: center;
    min-width: 0;

    code {
      overflow: hidden;
      color: var(--theme-color);
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  .operation-cell {
    display: flex;
    gap: 6px;
    align-items: center;
  }

  .dialog-alert {
    margin-bottom: 18px;
  }

  .secret-fields {
    display: flex;
    flex-direction: column;
    gap: 18px;
  }

  .secret-field {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .secret-field-key {
    padding-left: 12px;
    border-left: 3px solid var(--el-color-warning);
  }

  .field-label {
    font-size: 13px;
    font-weight: 600;
    color: var(--art-gray-700);
  }

  .field-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 8px;
  }

  .header-list {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    align-items: center;

    code {
      padding: 3px 7px;
      color: var(--theme-color);
      background: var(--art-active-color);
      border: 1px solid var(--default-border);
      border-radius: calc(var(--custom-radius) / 3 + 2px);
    }
  }

  .code-block {
    max-width: 100%;
    padding: 18px;
    margin: 0;
    overflow: auto;
    font-family: monospace;
    font-size: 13px;
    line-height: 1.65;
    color: var(--art-gray-800);
    white-space: pre;
    background: var(--default-bg-color);
    border: 1px solid var(--default-border);
    border-radius: calc(var(--custom-radius) / 2 + 2px);
  }

  @media (width <= 640px) {
    .page-heading {
      flex-direction: column;
    }

    .field-row {
      grid-template-columns: 1fr;
    }
  }
</style>
