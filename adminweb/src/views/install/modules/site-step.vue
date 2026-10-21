<template>
  <div class="step-shell">
    <div class="step-head">
      <div>
        <h2>站点信息</h2>
        <p>写入站点名称和访问地址，后续还可以在后台继续调整。</p>
      </div>
      <ElTag type="info" round>第 2 步 / 共 4 步</ElTag>
    </div>

    <ElCard shadow="never" class="step-card">
      <ElForm ref="formRef" :model="form" :rules="rules" label-position="top">
        <ElFormItem label="站点名称" prop="siteName">
          <ElInput v-model="form.siteName" placeholder="例如：小黑云" />
        </ElFormItem>
        <ElFormItem label="站点 URL" prop="siteUrl">
          <ElInput v-model="form.siteUrl" placeholder="例如：https://example.com" />
          <div class="field-hint">用于邮件链接、回调地址和公开展示，留空也可以继续。</div>
        </ElFormItem>
      </ElForm>
    </ElCard>

    <div class="bottom-bar">
      <ElAlert
        type="info"
        :closable="false"
        title="站点名称会显示在页面标题、邮件通知和部分后台信息中。"
        show-icon
      />

      <div class="actions">
        <ElButton @click="emit('back')">上一步</ElButton>
        <ElButton type="primary" @click="handleNext">下一步</ElButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import type { FormInstance, FormRules } from 'element-plus'
  import { useInstallWizardStore } from '@/store/modules/install-wizard'

  defineOptions({ name: 'InstallSiteStep' })

  const emit = defineEmits<{
    next: []
    back: []
  }>()

  const wizard = useInstallWizardStore()
  const formRef = ref<FormInstance>()
  const form = reactive({
    siteName: wizard.siteName,
    siteUrl: wizard.siteUrl
  })

  const rules: FormRules = {
    siteName: [{ required: true, message: '请输入站点名称', trigger: 'blur' }]
  }

  watch(
    form,
    (value) => {
      wizard.siteName = value.siteName
      wizard.siteUrl = value.siteUrl
      wizard.persist()
    },
    { deep: true }
  )

  async function handleNext() {
    if (!formRef.value) {
      return
    }

    const valid = await formRef.value.validate().catch(() => false)
    if (!valid) {
      return
    }

    emit('next')
  }
</script>

<style scoped lang="scss">
  .step-shell {
    display: flex;
    flex-direction: column;
    gap: 22px;
  }

  .step-head {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
  }

  .step-head h2 {
    margin: 0 0 8px;
    color: var(--el-text-color-primary);
    font-size: 28px;
    font-weight: 800;
  }

  .step-head p,
  .field-hint {
    margin: 0;
    color: var(--el-text-color-secondary);
    line-height: 1.6;
  }

  .step-card {
    border-radius: 22px;
  }

  .bottom-bar {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: 20px;
  }

  .bottom-bar :deep(.el-alert) {
    flex: 1;
  }

  .actions {
    display: flex;
    gap: 12px;
    flex-shrink: 0;
  }

  @media (max-width: 768px) {
    .step-head,
    .bottom-bar {
      flex-direction: column;
      align-items: stretch;
    }

    .actions {
      width: 100%;
    }

    .actions :deep(.el-button) {
      flex: 1;
    }
  }
</style>
