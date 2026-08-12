<template>
  <div class="site-settings-page pb-5">
    <ElCard v-loading="loading" class="art-card-xs">
      <template #header>
        <div class="settings-header">
          <div>
            <h2 class="settings-title">站点设置</h2>
            <p class="settings-subtitle">品牌、联系方式、备案信息与管理入口</p>
          </div>

          <ElSpace wrap>
            <ElButton
              v-if="canUpdate"
              type="primary"
              :loading="saving"
              :disabled="loading"
              @click="saveSettings"
            >
              <ArtSvgIcon icon="ri:save-line" />
              保存更改
            </ElButton>
          </ElSpace>
        </div>
      </template>

      <ArtForm
        ref="formRef"
        v-model="form"
        :items="formItems"
        :rules="formRules"
        :disabled="formDisabled"
        :show-reset="false"
        :show-submit="false"
        :span="12"
        :gutter="20"
        label-position="top"
      >
        <template #basic_section>
          <div class="section-heading">基础信息</div>
        </template>

        <template #logo_url>
          <ElInput v-model="form.logo_url" placeholder="https://example.com/logo.png" />
          <div class="logo-preview">
            <div class="logo-preview__media" aria-hidden="true">
              <img v-if="form.logo_url" :src="form.logo_url" alt="" />
              <DefaultLogoMark v-else :size="22" />
            </div>
            <span>未设置时使用默认 Logo</span>
          </div>
        </template>

        <template #contact_section>
          <div class="section-heading">联系方式</div>
        </template>

        <template #filing_section>
          <div class="section-heading">备案与版权</div>
        </template>

        <template #beian_info_list>
          <div class="filing-list">
            <div class="filing-list__header">
              <span class="field-label">备案信息</span>
              <ElButton v-if="canUpdate" plain type="primary" @click="addFiling">
                <ArtSvgIcon icon="ri:add-line" />
                添加备案
              </ElButton>
            </div>

            <div v-for="(filing, index) in filingList" :key="index" class="filing-item">
              <div class="filing-item__header">
                <span>备案 {{ index + 1 }}</span>
                <ElTooltip v-if="canUpdate && filingList.length > 1" content="删除备案">
                  <ElButton
                    circle
                    plain
                    type="danger"
                    aria-label="删除备案"
                    @click="removeFiling(index)"
                  >
                    <ArtSvgIcon icon="ri:delete-bin-line" />
                  </ElButton>
                </ElTooltip>
              </div>

              <ElRow :gutter="16">
                <ElCol :xs="24" :md="8">
                  <label class="field-label">备案号</label>
                  <ElInput v-model="filing.number" placeholder="京 ICP 备 12345678 号" />
                </ElCol>
                <ElCol :xs="24" :md="8">
                  <label class="field-label">备案图标 URL</label>
                  <ElInput v-model="filing.icon_url" placeholder="https://example.com/icon.png" />
                </ElCol>
                <ElCol :xs="24" :md="8">
                  <label class="field-label">备案跳转链接</label>
                  <ElInput v-model="filing.link_url" placeholder="https://beian.miit.gov.cn/" />
                </ElCol>
              </ElRow>
            </div>
          </div>
        </template>

        <template #security_section>
          <div class="section-heading">安全设置</div>
        </template>

        <template #admin_path>
          <div class="admin-path-control">
            <ElInput v-model="form.admin_path" placeholder="admin" />
            <ElTooltip content="随机生成管理路径">
              <ElButton
                :loading="generatingPath"
                aria-label="随机生成管理路径"
                @click="generateAdminPath"
              >
                <ArtSvgIcon icon="ri:shuffle-line" />
              </ElButton>
            </ElTooltip>
          </div>
          <div class="field-help">
            自定义管理后台访问路径，修改后将自动跳转到新路径（仅支持字母和数字）
          </div>
        </template>
      </ArtForm>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import type { FormRules } from 'element-plus'
  import DefaultLogoMark from '@/components/brand/DefaultLogoMark.vue'
  import { useAuth } from '@/hooks/core/useAuth'
  import { clearAdminPathCache, setAdminPathHint } from '@/services/adminPath'
  import { listSettings, updateSetting } from '@/services/admin'
  import type { SettingItem } from '@/services/types'
  import { useSiteStore } from '@/stores/site'

  defineOptions({ name: 'AdminSettingsSite' })

  interface SiteSettingsForm extends Record<string, string> {
    site_name: string
    site_url: string
    logo_url: string
    favicon_url: string
    site_description: string
    site_keywords: string
    company_name: string
    contact_phone: string
    contact_email: string
    contact_qq: string
    wechat_qrcode: string
    copyright_text: string
    admin_path: string
    analytics_code: string
  }

  interface FilingItem {
    number: string
    icon_url: string
    link_url: string
  }

  interface ArtFormExpose {
    validate: () => Promise<boolean> | undefined
  }

  const SITE_SETTING_KEYS: Array<keyof SiteSettingsForm> = [
    'site_name',
    'site_url',
    'logo_url',
    'favicon_url',
    'site_description',
    'site_keywords',
    'company_name',
    'contact_phone',
    'contact_email',
    'contact_qq',
    'wechat_qrcode',
    'copyright_text',
    'admin_path',
    'analytics_code'
  ]

  const loading = ref(false)
  const saving = ref(false)
  const generatingPath = ref(false)
  const originalAdminPath = ref('admin')
  const formRef = ref<ArtFormExpose>()
  const form = reactive<SiteSettingsForm>(createDefaultForm())
  const filingList = ref<FilingItem[]>([createFilingItem()])

  const siteStore = useSiteStore()
  const { hasAuth } = useAuth()
  const canUpdate = computed(() => hasAuth('settings.update'))
  const formDisabled = computed(() => loading.value || saving.value || !canUpdate.value)

  const formItems = computed(() => [
    { key: 'basic_section', label: '', span: 24 },
    {
      key: 'site_name',
      label: '站点名称',
      type: 'input',
      span: 12,
      props: { placeholder: '小黑云控制台' }
    },
    {
      key: 'site_url',
      label: '网站 URL',
      type: 'input',
      span: 12,
      props: { placeholder: 'https://example.com' }
    },
    { key: 'logo_url', label: 'Logo URL', span: 12 },
    {
      key: 'favicon_url',
      label: 'Favicon URL',
      type: 'input',
      span: 12,
      props: { placeholder: 'https://example.com/favicon.ico' }
    },
    {
      key: 'site_description',
      label: '网站描述',
      type: 'input',
      span: 24,
      props: {
        type: 'textarea',
        rows: 3,
        placeholder: '专业的云服务提供商'
      }
    },
    {
      key: 'site_keywords',
      label: '关键词',
      type: 'input',
      span: 24,
      props: { placeholder: '云服务器,VPS,云主机' }
    },
    { key: 'contact_section', label: '', span: 24 },
    {
      key: 'company_name',
      label: '公司名称',
      type: 'input',
      span: 12,
      props: {}
    },
    {
      key: 'contact_phone',
      label: '联系电话',
      type: 'input',
      span: 12,
      props: {}
    },
    {
      key: 'contact_email',
      label: '联系邮箱',
      type: 'input',
      span: 12,
      props: {}
    },
    {
      key: 'contact_qq',
      label: 'QQ 号码',
      type: 'input',
      span: 12,
      props: {}
    },
    {
      key: 'wechat_qrcode',
      label: '微信二维码',
      type: 'input',
      span: 24,
      props: { placeholder: '二维码图片 URL' }
    },
    { key: 'filing_section', label: '', span: 24 },
    {
      key: 'copyright_text',
      label: '版权信息',
      type: 'input',
      span: 24,
      props: {
        maxlength: 200,
        showWordLimit: true,
        placeholder: `${new Date().getFullYear()} xx云服务 All rights reserved.`
      }
    },
    { key: 'beian_info_list', label: '', span: 24 },
    { key: 'security_section', label: '', span: 24 },
    { key: 'admin_path', label: '管理端路径', span: 24 }
  ])

  const formRules: FormRules<SiteSettingsForm> = {
    copyright_text: [
      { required: true, message: '版权信息不能为空', trigger: 'blur' },
      { max: 200, message: '版权信息不能超过 200 个字符', trigger: 'blur' }
    ]
  }

  onMounted(fetchSettings)

  function createDefaultForm(): SiteSettingsForm {
    return {
      site_name: '',
      site_url: '',
      logo_url: '',
      favicon_url: '',
      site_description: '',
      site_keywords: '',
      company_name: '',
      contact_phone: '',
      contact_email: '',
      contact_qq: '',
      wechat_qrcode: '',
      copyright_text: '',
      admin_path: '',
      analytics_code: ''
    }
  }

  function createFilingItem(): FilingItem {
    return { number: '', icon_url: '', link_url: '' }
  }

  function parseFilingList(items: SettingItem[]): FilingItem[] {
    const value = items.find((item) => item.key === 'beian_info_list')?.value
    if (!value) return [createFilingItem()]

    try {
      const parsed = JSON.parse(value)
      if (!Array.isArray(parsed) || parsed.length === 0) return [createFilingItem()]

      return parsed.map((item) => ({
        number: String(item?.number ?? ''),
        icon_url: String(item?.icon_url ?? ''),
        link_url: String(item?.link_url ?? '')
      }))
    } catch {
      ElMessage.warning('备案信息格式无效，已显示空白表单')
      return [createFilingItem()]
    }
  }

  async function fetchSettings(): Promise<void> {
    loading.value = true
    try {
      const response = await listSettings()
      const items = response.data?.items ?? []
      const nextForm = createDefaultForm()

      items.forEach((item) => {
        const key = String(item.key ?? '') as keyof SiteSettingsForm
        if (SITE_SETTING_KEYS.includes(key)) nextForm[key] = String(item.value ?? '')
      })

      Object.assign(form, nextForm)
      filingList.value = parseFilingList(items)
      originalAdminPath.value = nextForm.admin_path || 'admin'
    } finally {
      loading.value = false
    }
  }

  function addFiling(): void {
    filingList.value.push(createFilingItem())
  }

  function removeFiling(index: number): void {
    filingList.value.splice(index, 1)
  }

  function generateAdminPath(): void {
    generatingPath.value = true
    try {
      const characters = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789'
      const values = crypto.getRandomValues(new Uint8Array(12))
      form.admin_path = Array.from(values, (value) => characters[value % characters.length]).join(
        ''
      )
      ElMessage.success('已生成新的管理路径')
    } finally {
      generatingPath.value = false
    }
  }

  function validateFilingList(): boolean {
    const invalidIndex = filingList.value.findIndex(
      (item) => (item.icon_url.trim() || item.link_url.trim()) && !item.number.trim()
    )
    if (invalidIndex < 0) return true

    ElMessage.error(`备案 ${invalidIndex + 1} 的备案号不能为空`)
    return false
  }

  function cacheAdminPath(path: string): void {
    // localStorage 只作为路径提示（H1）：改完 admin_path 后旧内存验证态已被
    // clearAdminPathCache 的代数自增作废，新页面首次进入会回落服务端复核。
    setAdminPathHint(path)
    try {
      // 清理旧版本遗留的"验证时间戳"键，避免误导后续代码。
      localStorage.removeItem('admin_path_validated_at')
    } catch {
      ElMessage.warning('管理路径已保存，但浏览器缓存更新失败')
    }
  }

  async function saveSettings(): Promise<void> {
    const isValid = await formRef.value?.validate()?.catch(() => false)
    if (!isValid || !validateFilingList()) return

    saving.value = true
    try {
      const nextAdminPath = form.admin_path || 'admin'
      const items = SITE_SETTING_KEYS.map((key) => ({
        key,
        value: key === 'admin_path' ? nextAdminPath : String(form[key] ?? '')
      }))
      items.push({
        key: 'beian_info_list',
        value: JSON.stringify(filingList.value.filter((item) => item.number.trim()))
      })

      await updateSetting({ items })
      await siteStore.fetchSettings(true)

      if (nextAdminPath !== originalAdminPath.value) {
        clearAdminPathCache()
        cacheAdminPath(nextAdminPath)
        ElMessage.success('保存成功，正在进入新的管理路径')
        window.setTimeout(() => {
          window.location.assign(`/${nextAdminPath}/settings/site`)
        }, 1200)
        return
      }

      ElMessage.success('保存成功')
      originalAdminPath.value = nextAdminPath
    } finally {
      saving.value = false
    }
  }
</script>

<style lang="scss" scoped>
  .site-settings-page {
    min-width: 0;
  }

  .settings-header,
  .filing-list__header,
  .filing-item__header,
  .admin-path-control,
  .logo-preview {
    display: flex;
    align-items: center;
  }

  .settings-header,
  .filing-list__header,
  .filing-item__header {
    gap: 16px;
    justify-content: space-between;
  }

  .settings-title {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
    line-height: 1.4;
    color: var(--art-gray-900);
  }

  .settings-subtitle {
    margin: 4px 0 0;
    font-size: 13px;
    color: var(--art-gray-600);
  }

  .section-heading {
    width: 100%;
    padding-bottom: 10px;
    font-size: 15px;
    font-weight: 600;
    color: var(--art-gray-900);
    border-bottom: 1px solid var(--default-border);
  }

  .logo-preview {
    gap: 10px;
    margin-top: 10px;
    font-size: 12px;
    color: var(--art-gray-600);
  }

  .logo-preview__media {
    display: grid;
    place-items: center;
    width: 36px;
    height: 36px;
    overflow: hidden;
    color: var(--theme-color);
    background: var(--el-color-primary-light-9);
    border: 1px solid var(--default-border);
    border-radius: calc(var(--custom-radius) / 2 + 2px);
  }

  .logo-preview__media img {
    display: block;
    width: 24px;
    height: 24px;
    object-fit: contain;
  }

  .filing-list {
    width: 100%;
  }

  .filing-item {
    padding: 16px;
    background: var(--default-bg-color);
    border: 1px solid var(--default-border);
    border-radius: calc(var(--custom-radius) / 2 + 2px);
  }

  .filing-item + .filing-item {
    margin-top: 12px;
  }

  .filing-item__header {
    margin-bottom: 14px;
    font-weight: 600;
    color: var(--art-gray-800);
  }

  .field-label {
    display: block;
    margin-bottom: 7px;
    font-size: 13px;
    color: var(--art-gray-700);
  }

  .admin-path-control {
    gap: 8px;
    width: 100%;
  }

  .field-help {
    margin-top: 8px;
    font-size: 12px;
    color: var(--art-gray-600);
  }

  @media (width <= 640px) {
    .settings-header {
      flex-direction: column;
      align-items: flex-start;
    }

    .filing-item :is(.el-col) + :is(.el-col) {
      margin-top: 12px;
    }
  }
</style>
