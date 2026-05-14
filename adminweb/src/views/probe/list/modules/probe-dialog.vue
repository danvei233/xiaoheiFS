<template>
  <ElDialog
    v-model="dialogVisible"
    :title="dialogTitle"
    width="640px"
    destroy-on-close
    align-center
  >
    <ElForm ref="formRef" :model="localForm" :rules="rules" label-width="110px">
      <ElFormItem label="探针名称" prop="name">
        <ElInput v-model="localForm.name" placeholder="例如：香港节点 A" :maxlength="128" />
      </ElFormItem>

      <ElFormItem label="Agent ID" prop="agent_id">
        <ElInput
          v-model="localForm.agent_id"
          :disabled="isEditMode"
          placeholder="例如：hk-node-a"
          :maxlength="128"
        />
      </ElFormItem>

      <ElFormItem label="系统类型" prop="os_type">
        <ElSelect v-model="localForm.os_type" placeholder="请选择系统类型">
          <ElOption label="Linux" value="linux" />
          <ElOption label="Windows" value="windows" />
        </ElSelect>
      </ElFormItem>

      <ElFormItem label="标签">
        <ElSelect
          v-model="localForm.tags"
          multiple
          filterable
          allow-create
          default-first-option
          clearable
          placeholder="回车创建标签，例如 region:hkg"
        />
      </ElFormItem>
    </ElForm>

    <template #footer>
      <div class="dialog-footer">
        <ElButton @click="dialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="submitting" @click="handleSubmit">提交</ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import type { FormInstance, FormRules } from 'element-plus'

  defineOptions({ name: 'ProbeDialog' })

  interface ProbeDialogFormValue {
    id: number | null
    name: string
    agent_id: string
    os_type: string
    tags: string[]
  }

  interface Props {
    visible: boolean
    mode: 'create' | 'edit'
    formData: ProbeDialogFormValue
    submitting?: boolean
  }

  interface Emits {
    (e: 'update:visible', value: boolean): void
    (e: 'submit', value: ProbeDialogFormValue): void
  }

  const props = withDefaults(defineProps<Props>(), {
    submitting: false
  })
  const emit = defineEmits<Emits>()

  const formRef = ref<FormInstance>()

  const dialogVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })

  const isEditMode = computed(() => props.mode === 'edit')
  const dialogTitle = computed(() => (isEditMode.value ? '编辑探针' : '创建探针'))

  const localForm = reactive<ProbeDialogFormValue>(createDefaultForm())

  const rules = computed<FormRules>(() => ({
    name: [
      { required: true, message: '请输入探针名称', trigger: 'blur' },
      { max: 128, message: '探针名称长度不能超过 128 个字符', trigger: 'blur' }
    ],
    agent_id: [
      { required: true, message: '请输入 Agent ID', trigger: 'blur' },
      { max: 128, message: 'Agent ID 长度不能超过 128 个字符', trigger: 'blur' }
    ],
    os_type: [{ required: true, message: '请选择系统类型', trigger: 'change' }]
  }))

  watch(
    () => [props.visible, props.mode, props.formData] as const,
    ([visible]) => {
      if (!visible) {
        return
      }

      applyFormData(props.formData)
      nextTick(() => formRef.value?.clearValidate())
    },
    { immediate: true, deep: true }
  )

  function createDefaultForm(): ProbeDialogFormValue {
    return {
      id: null,
      name: '',
      agent_id: '',
      os_type: 'linux',
      tags: []
    }
  }

  function applyFormData(value: ProbeDialogFormValue) {
    Object.assign(localForm, createDefaultForm(), {
      ...value,
      tags: Array.isArray(value.tags) ? [...value.tags] : []
    })
  }

  async function handleSubmit() {
    if (!formRef.value) {
      return
    }

    const valid = await formRef.value.validate().catch(() => false)
    if (!valid) {
      return
    }

    emit('submit', {
      ...localForm,
      name: localForm.name.trim(),
      agent_id: localForm.agent_id.trim(),
      os_type: localForm.os_type.trim(),
      tags: localForm.tags.map((item) => String(item || '').trim()).filter(Boolean)
    })
  }
</script>

<style scoped lang="scss">
  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }
</style>
