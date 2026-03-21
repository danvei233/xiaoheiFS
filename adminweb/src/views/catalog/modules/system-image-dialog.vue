<template>
  <ElDialog
    v-model="dialogVisible"
    :title="localForm.id ? '编辑镜像' : '新增镜像'"
    width="460px"
    destroy-on-close
    align-center
  >
    <ElForm label-position="top">
      <ElFormItem label="镜像 ID">
        <ElInputNumber v-model="localForm.image_id" :min="1" :precision="0" class="full-width" />
      </ElFormItem>
      <ElFormItem label="名称">
        <ElInput v-model.trim="localForm.name" maxlength="120" placeholder="请输入名称" />
      </ElFormItem>
      <ElFormItem label="类型">
        <ElSelect v-model="localForm.type" class="full-width" placeholder="请选择类型">
          <ElOption label="Linux" value="linux" />
          <ElOption label="Windows" value="windows" />
        </ElSelect>
      </ElFormItem>
      <ElFormItem label="启用">
        <ElSwitch v-model="localForm.enabled" />
      </ElFormItem>
    </ElForm>

    <template #footer>
      <div class="dialog-footer">
        <ElButton @click="dialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="submitting" @click="emit('submit', { ...localForm })">
          保存
        </ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { useSystemImageDialogBinding } from '@/components/business/system-image-dialog/model'
  import type { SystemImageDialogFormValue } from '@/components/business/system-image-dialog/model'

  defineOptions({ name: 'CatalogSystemImageDialog' })

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
