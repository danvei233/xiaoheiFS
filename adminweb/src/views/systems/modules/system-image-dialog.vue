<template>
  <ElDialog
    v-model="dialogVisible"
    :title="localForm.id ? t('systemImage.dialog.editTitle') : t('systemImage.dialog.createTitle')"
    width="480px"
    destroy-on-close
    align-center
  >
    <ElForm label-position="top">
      <ElFormItem :label="t('systemImage.dialog.imageId')">
        <ElInputNumber v-model="localForm.image_id" :min="1" :precision="0" class="full-width" />
      </ElFormItem>

      <ElFormItem :label="t('systemImage.dialog.name')">
        <ElInput
          v-model.trim="localForm.name"
          :maxlength="120"
          :placeholder="t('systemImage.dialog.namePlaceholder')"
        />
      </ElFormItem>

      <ElFormItem :label="t('systemImage.dialog.type')">
        <ElSelect
          v-model="localForm.type"
          class="full-width"
          :placeholder="t('systemImage.dialog.typePlaceholder')"
        >
          <ElOption label="Linux" value="linux" />
          <ElOption label="Windows" value="windows" />
        </ElSelect>
      </ElFormItem>

      <ElFormItem :label="t('systemImage.dialog.enabled')">
        <ElSwitch v-model="localForm.enabled" />
      </ElFormItem>
    </ElForm>

    <template #footer>
      <div class="dialog-footer">
        <ElButton @click="dialogVisible = false">{{ t('common.cancel') }}</ElButton>
        <ElButton type="primary" :loading="submitting" @click="emit('submit', { ...localForm })">
          {{ t('systemImage.dialog.save') }}
        </ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { useSystemImageDialogBinding } from '@/components/business/system-image-dialog/model'
  import type { SystemImageDialogFormValue } from '@/components/business/system-image-dialog/model'
  import { useI18n } from 'vue-i18n'

  defineOptions({ name: 'SystemsImageDialog' })

  interface Props {
    visible: boolean
    formData: SystemImageDialogFormValue
    submitting?: boolean
  }

  interface Emits {
    (e: 'update:visible', value: boolean): void
    (e: 'submit', value: SystemImageDialogFormValue): void
  }

  const props = withDefaults(defineProps<Props>(), {
    submitting: false
  })
  const emit = defineEmits<Emits>()
  const { t } = useI18n()

  const { localForm, dialogVisible } = useSystemImageDialogBinding(props, (value) =>
    emit('update:visible', value)
  )
</script>

<style scoped lang="scss">
  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  .full-width {
    width: 100%;
  }
</style>
