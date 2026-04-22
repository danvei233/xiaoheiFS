<template>
  <div class="site-settings-page art-full-height">
    <ElCard v-loading="loading" shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">站点设置</div>
            <div class="page-subtitle">配置品牌信息、备案资料与管理入口。</div>
          </div>

          <div class="page-actions">
            <ElButton :disabled="loading" @click="fetchData">刷新</ElButton>
            <ElButton v-if="canUpdate" type="primary" :loading="saving" @click="handleSave">
              保存更改
            </ElButton>
          </div>
        </div>
      </template>

      <template v-if="canView">
        <ElForm ref="formRef" :model="form" :rules="rules" label-position="top">
          <section class="form-section">
            <div class="section-title">基础信息</div>

            <ElRow :gutter="16">
              <ElCol :xs="24" :md="12">
                <ElFormItem label="站点名称" prop="site_name">
                  <ElInput
                    v-model="form.site_name"
                    :disabled="formDisabled"
                    maxlength="120"
                    placeholder="小黑云控制台"
                  />
                </ElFormItem>
              </ElCol>

              <ElCol :xs="24" :md="12">
                <ElFormItem label="网站 URL" prop="site_url">
                  <ElInput
                    v-model="form.site_url"
                    :disabled="formDisabled"
                    maxlength="1024"
                    placeholder="https://example.com"
                  />
                </ElFormItem>
              </ElCol>
            </ElRow>

            <ElRow :gutter="16">
              <ElCol :xs="24" :md="12">
                <ElFormItem label="Logo URL" prop="logo_url">
                  <ElInput
                    v-model="form.logo_url"
                    :disabled="formDisabled"
                    maxlength="1024"
                    placeholder="https://example.com/logo.png"
                  />
                  <div class="logo-preview">
                    <div class="logo-preview-badge">
                      <img v-if="form.logo_url" :src="form.logo_url" alt="logo-preview" />
                      <span v-else>{{ logoFallbackText }}</span>
                    </div>
                    <span class="logo-preview-tip">留空则继续使用默认 Logo。</span>
                  </div>
                </ElFormItem>
              </ElCol>

              <ElCol :xs="24" :md="12">
                <ElFormItem label="Favicon URL" prop="favicon_url">
                  <ElInput
                    v-model="form.favicon_url"
                    :disabled="formDisabled"
                    maxlength="1024"
                    placeholder="https://example.com/favicon.ico"
                  />
                </ElFormItem>
              </ElCol>
            </ElRow>

            <ElFormItem label="网站描述" prop="site_description">
              <ElInput
                v-model="form.site_description"
                type="textarea"
                :rows="3"
                :disabled="formDisabled"
                maxlength="500"
                show-word-limit
                placeholder="专业的云服务提供商"
              />
            </ElFormItem>

            <ElFormItem label="关键词" prop="site_keywords">
              <ElInput
                v-model="form.site_keywords"
                :disabled="formDisabled"
                maxlength="500"
                placeholder="云服务器,VPS,云主机"
              />
            </ElFormItem>
          </section>

          <ElDivider />

          <section class="form-section">
            <div class="section-title">联系方式</div>

            <ElRow :gutter="16">
              <ElCol :xs="24" :md="12">
                <ElFormItem label="公司名称" prop="company_name">
                  <ElInput v-model="form.company_name" :disabled="formDisabled" maxlength="120" />
                </ElFormItem>
              </ElCol>

              <ElCol :xs="24" :md="12">
                <ElFormItem label="联系电话" prop="contact_phone">
                  <ElInput v-model="form.contact_phone" :disabled="formDisabled" maxlength="64" />
                </ElFormItem>
              </ElCol>
            </ElRow>

            <ElRow :gutter="16">
              <ElCol :xs="24" :md="12">
                <ElFormItem label="联系邮箱" prop="contact_email">
                  <ElInput v-model="form.contact_email" :disabled="formDisabled" maxlength="254" />
                </ElFormItem>
              </ElCol>

              <ElCol :xs="24" :md="12">
                <ElFormItem label="QQ 号码" prop="contact_qq">
                  <ElInput v-model="form.contact_qq" :disabled="formDisabled" maxlength="64" />
                </ElFormItem>
              </ElCol>
            </ElRow>

            <ElFormItem label="微信二维码" prop="wechat_qrcode">
              <ElInput
                v-model="form.wechat_qrcode"
                :disabled="formDisabled"
                maxlength="1024"
                placeholder="二维码图片 URL"
              />
            </ElFormItem>
          </section>

          <ElDivider />

          <section class="form-section">
            <div class="section-title">备案与版权</div>

            <ElFormItem label="版权信息" prop="copyright_text">
              <ElInput
                v-model="form.copyright_text"
                :disabled="formDisabled"
                maxlength="200"
                show-word-limit
                placeholder="2026 xx云服务 All rights reserved."
              />
              <div class="field-tip">用于设置网站底部版权文案，最多 200 个字符。</div>
            </ElFormItem>

            <div class="beian-header">
              <div class="section-subtitle">备案信息</div>
              <ElButton
                v-if="canUpdate"
                type="primary"
                plain
                :disabled="formDisabled"
                @click="handleAddBeianInfo"
              >
                添加备案信息
              </ElButton>
            </div>

            <div class="beian-list">
              <div v-for="(beian, index) in beianInfoList" :key="index" class="beian-card">
                <div class="beian-card-header">
                  <div class="beian-card-title">备案信息 {{ index + 1 }}</div>
                  <ElButton
                    v-if="canUpdate && beianInfoList.length > 1"
                    type="danger"
                    link
                    :disabled="formDisabled"
                    @click="handleRemoveBeianInfo(index)"
                  >
                    删除
                  </ElButton>
                </div>

                <ElRow :gutter="12">
                  <ElCol :xs="24" :md="24">
                    <label class="inline-label">备案号<span class="required-mark">*</span></label>
                    <ElInput
                      v-model="beian.number"
                      :disabled="formDisabled"
                      placeholder="例如：京 ICP 备 12345678 号"
                    />
                  </ElCol>
                </ElRow>

                <ElRow :gutter="12" class="beian-row">
                  <ElCol :xs="24" :md="12">
                    <label class="inline-label">备案图标 URL</label>
                    <ElInput
                      v-model="beian.icon_url"
                      :disabled="formDisabled"
                      maxlength="1024"
                      placeholder="https://example.com/icon.png"
                    />
                  </ElCol>

                  <ElCol :xs="24" :md="12">
                    <label class="inline-label">备案跳转链接</label>
                    <ElInput
                      v-model="beian.link_url"
                      :disabled="formDisabled"
                      maxlength="1024"
                      placeholder="https://beian.miit.gov.cn/"
                    />
                  </ElCol>
                </ElRow>
              </div>
            </div>
          </section>

          <ElDivider />

          <section class="form-section">
            <div class="section-title">安全设置</div>

            <ElFormItem label="管理端路径" prop="admin_path">
              <div class="admin-path-row">
                <ElInput
                  v-model="form.admin_path"
                  :disabled="formDisabled"
                  maxlength="64"
                  placeholder="admin"
                />
                <ElButton
                  :disabled="formDisabled"
                  :loading="refreshing"
                  @click="handleRefreshAdminPath"
                >
                  随机生成
                </ElButton>
              </div>
              <div class="field-tip">
                仅支持字母和数字。保存后会更新本地缓存；如果当前已经通过动态管理入口访问，会自动跳转到新地址。
              </div>
            </ElFormItem>

            <ElFormItem label="统计代码" prop="analytics_code">
              <ElInput
                v-model="form.analytics_code"
                type="textarea"
                :rows="4"
                :disabled="formDisabled"
                maxlength="5000"
                placeholder="<script>...</script>"
              />
            </ElFormItem>
          </section>
        </ElForm>
      </template>

      <ElEmpty v-else description="当前账号没有查看站点设置的权限" />
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { ElMessage, ElNotification, type FormInstance, type FormRules } from 'element-plus'
  import { fetchAdminSettings, hasAdminPermission, updateAdminSettings } from '@/api/admin'
  import { useSiteStore } from '@/store/modules/site'
  import { useUserStore } from '@/store/modules/user'

  defineOptions({ name: 'SettingsSitePage' })

  interface SiteSettingForm {
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
    analytics_code: string
    admin_path: string
    copyright_text: string
  }

  interface BeianInfoItem {
    number: string
    icon_url: string
    link_url: string
  }

  const loading = ref(false)
  const saving = ref(false)
  const refreshing = ref(false)
  const originalAdminPath = ref('admin')

  const formRef = ref<FormInstance>()
  const siteStore = useSiteStore()
  const userStore = useUserStore()
  const { info } = storeToRefs(userStore)

  const form = reactive<SiteSettingForm>(createDefaultForm())
  const beianInfoList = ref<BeianInfoItem[]>([createBeianInfoItem()])

  const canView = computed(() => hasAdminPermission(info.value?.buttons, ['settings.view']))
  const canUpdate = computed(() => hasAdminPermission(info.value?.buttons, ['settings.update']))
  const formDisabled = computed(() => !canUpdate.value || loading.value || saving.value)

  const logoFallbackText = computed(() => {
    const text = String(form.site_name || siteStore.resolvedSiteName || 'A').trim()
    return (text[0] || 'A').toUpperCase()
  })

  const rules = computed<FormRules>(() => ({
    site_name: [{ required: true, message: '请输入站点名称', trigger: 'blur' }],
    contact_email: [
      {
        type: 'email',
        message: '请输入正确的邮箱地址',
        trigger: ['blur', 'change']
      }
    ],
    copyright_text: [
      { required: true, message: '版权信息不能为空', trigger: 'blur' },
      { max: 200, message: '版权信息不能超过 200 个字符', trigger: 'blur' }
    ],
    admin_path: [
      { required: true, message: '管理端路径不能为空', trigger: 'blur' },
      {
        validator: (_rule, value, callback) => {
          const normalized = normalizeAdminPath(String(value || ''))
          if (!/^[a-zA-Z0-9]+$/.test(normalized)) {
            callback(new Error('管理端路径仅支持字母和数字'))
            return
          }
          callback()
        },
        trigger: 'blur'
      }
    ]
  }))

  onMounted(() => {
    if (canView.value) {
      fetchData()
    }
  })

  function createDefaultForm(): SiteSettingForm {
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
      analytics_code: '',
      admin_path: 'admin',
      copyright_text: ''
    }
  }

  function createBeianInfoItem(): BeianInfoItem {
    return {
      number: '',
      icon_url: '',
      link_url: ''
    }
  }

  function normalizeAdminPath(path: string) {
    return String(path || '')
      .trim()
      .replace(/^\/+|\/+$/g, '')
  }

  function getCachedAdminPath() {
    try {
      return normalizeAdminPath(localStorage.getItem('admin_path_cache') || '') || 'admin'
    } catch {
      return 'admin'
    }
  }

  function clearAdminPathCache() {
    try {
      localStorage.removeItem('admin_path_cache')
      localStorage.removeItem('admin_path_validated')
    } catch {
      // ignore
    }
  }

  function cacheAdminPath(path: string) {
    try {
      localStorage.setItem('admin_path_cache', normalizeAdminPath(path))
      localStorage.setItem('admin_path_validated', 'true')
    } catch {
      // ignore
    }
  }

  function buildAdminPathRedirectUrl(path: string) {
    return `${window.location.origin}/${normalizeAdminPath(path)}/#/settings/site`
  }

  function generateRandomAdminPath() {
    const charset = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789'
    const randomValues = new Uint8Array(12)
    crypto.getRandomValues(randomValues)

    return Array.from(randomValues)
      .map((value) => charset[value % charset.length])
      .join('')
  }

  function applySettingsMap(items: Array<{ key?: string; value?: string }>) {
    Object.assign(form, createDefaultForm())

    items.forEach((item) => {
      const key = String(item.key || '')
      if (key in form) {
        form[key as keyof SiteSettingForm] = String(item.value || '')
      }
    })

    const beianSetting = items.find((item) => item.key === 'beian_info_list')
    if (beianSetting?.value) {
      try {
        const parsed = JSON.parse(beianSetting.value)
        if (Array.isArray(parsed) && parsed.length > 0) {
          beianInfoList.value = parsed.map((item) => ({
            number: String(item?.number || ''),
            icon_url: String(item?.icon_url || ''),
            link_url: String(item?.link_url || '')
          }))
        } else {
          beianInfoList.value = [createBeianInfoItem()]
        }
      } catch {
        beianInfoList.value = [createBeianInfoItem()]
      }
    } else {
      beianInfoList.value = [createBeianInfoItem()]
    }

    form.admin_path = normalizeAdminPath(form.admin_path || getCachedAdminPath()) || 'admin'
    originalAdminPath.value = form.admin_path
  }

  async function fetchData() {
    loading.value = true

    try {
      const payload = await fetchAdminSettings()
      applySettingsMap(payload.items || [])
    } finally {
      loading.value = false
    }
  }

  function handleAddBeianInfo() {
    beianInfoList.value.push(createBeianInfoItem())
  }

  function handleRemoveBeianInfo(index: number) {
    if (beianInfoList.value.length <= 1) {
      beianInfoList.value = [createBeianInfoItem()]
      return
    }

    beianInfoList.value.splice(index, 1)
  }

  function handleRefreshAdminPath() {
    refreshing.value = true

    try {
      form.admin_path = generateRandomAdminPath()
      ElMessage.success('已生成新的管理路径')
    } finally {
      refreshing.value = false
    }
  }

  function validateBeianInfoList() {
    for (let index = 0; index < beianInfoList.value.length; index += 1) {
      const item = beianInfoList.value[index]
      if ((item.icon_url || item.link_url) && !String(item.number || '').trim()) {
        ElMessage.error(`备案信息 ${index + 1} 的备案号不能为空`)
        return false
      }
    }

    return true
  }

  async function handleSave() {
    if (!formRef.value) {
      return
    }

    const valid = await formRef.value.validate().catch(() => false)
    if (!valid || !validateBeianInfoList()) {
      return
    }

    saving.value = true

    try {
      const nextAdminPath = normalizeAdminPath(form.admin_path) || 'admin'
      const previousAdminPath = originalAdminPath.value || getCachedAdminPath()

      await updateAdminSettings({
        items: [
          ...Object.entries(form).map(([key, value]) => ({
            key,
            value: key === 'admin_path' ? nextAdminPath : String(value ?? '')
          })),
          {
            key: 'beian_info_list',
            value: JSON.stringify(
              beianInfoList.value.filter((item) => String(item.number || '').trim())
            )
          }
        ]
      })

      form.admin_path = nextAdminPath
      originalAdminPath.value = nextAdminPath
      await siteStore.fetchSettings()

      if (nextAdminPath !== previousAdminPath) {
        clearAdminPathCache()
        cacheAdminPath(nextAdminPath)

        if (window.location.pathname !== '/') {
          ElMessage.success('保存成功，正在跳转到新的管理入口')
          window.setTimeout(() => {
            window.location.assign(buildAdminPathRedirectUrl(nextAdminPath))
          }, 1500)
          return
        }

        ElNotification.success({
          title: '保存成功',
          message: `管理路径已更新为 /${nextAdminPath}。当前开发入口不会自动切换，但本地缓存已经同步。`,
          duration: 3500
        })
        return
      }

      ElMessage.success('保存成功')
    } finally {
      saving.value = false
    }
  }
</script>

<style scoped lang="scss">
  .site-settings-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .page-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
  }

  .page-title {
    color: var(--el-text-color-primary);
    font-size: 18px;
    font-weight: 700;
  }

  .page-subtitle,
  .field-tip,
  .logo-preview-tip {
    margin-top: 4px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.7;
  }

  .page-actions {
    display: flex;
    gap: 12px;
  }

  .form-section {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .section-title,
  .section-subtitle {
    color: var(--el-text-color-primary);
    font-size: 16px;
    font-weight: 700;
  }

  .logo-preview {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-top: 10px;
  }

  .logo-preview-badge {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 38px;
    height: 38px;
    overflow: hidden;
    border-radius: 12px;
    background: linear-gradient(135deg, #0ea5e9 0%, #0284c7 50%, #10b981 100%);
    color: #fff;
    font-size: 15px;
    font-weight: 700;
    box-shadow: 0 8px 18px rgb(14 165 233 / 24%);
  }

  .logo-preview-badge img {
    width: 100%;
    height: 100%;
    object-fit: contain;
    background: #fff;
  }

  .beian-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    margin-bottom: 8px;
  }

  .beian-list {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .beian-card {
    padding: 16px;
    border: 1px solid var(--el-border-color-light);
    border-radius: 14px;
    background: linear-gradient(180deg, rgb(148 163 184 / 8%), rgb(255 255 255 / 96%));
  }

  .beian-card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;
  }

  .beian-card-title {
    font-weight: 700;
  }

  .beian-row {
    margin-top: 12px;
  }

  .inline-label {
    display: inline-flex;
    margin-bottom: 8px;
    color: var(--el-text-color-regular);
    font-size: 13px;
    font-weight: 600;
  }

  .required-mark {
    margin-left: 4px;
    color: var(--el-color-danger);
  }

  .admin-path-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 12px;
    width: 100%;
  }

  @media (max-width: 768px) {
    .page-header,
    .beian-header {
      flex-direction: column;
      align-items: flex-start;
    }

    .page-actions,
    .admin-path-row {
      width: 100%;
    }

    .page-actions :deep(.el-button) {
      flex: 1;
    }

    .admin-path-row {
      grid-template-columns: 1fr;
    }
  }
</style>
