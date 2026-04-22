<template>
  <ElDialog
    v-model="dialogVisible"
    :title="dialogTitle"
    width="520px"
    destroy-on-close
    align-center
  >
    <ElForm ref="formRef" :model="localForm" :rules="rules" label-position="top">
      <ElFormItem label="用户名" prop="username">
        <ElInput
          v-model="localForm.username"
          placeholder="请输入用户名"
          :maxlength="INPUT_LIMITS.USERNAME"
        />
      </ElFormItem>

      <ElFormItem label="邮箱" prop="email">
        <ElInput
          v-model="localForm.email"
          placeholder="请输入邮箱"
          :maxlength="INPUT_LIMITS.EMAIL"
        />
      </ElFormItem>

      <ElFormItem label="QQ" prop="qq">
        <ElInput v-model="localForm.qq" placeholder="请输入 QQ 号" :maxlength="INPUT_LIMITS.QQ" />
      </ElFormItem>

      <ElFormItem v-if="!isEditMode" label="密码" prop="password">
        <ElInput
          v-model="localForm.password"
          type="password"
          show-password
          placeholder="请输入密码"
          :maxlength="INPUT_LIMITS.PASSWORD"
        />
      </ElFormItem>

      <ElFormItem label="权限组" prop="permission_group_id">
        <ElSelect
          v-model="localForm.permission_group_id"
          clearable
          filterable
          :disabled="disablePermissionGroup"
          placeholder="请选择权限组"
          class="full-width"
        >
          <ElOption
            v-for="item in permissionGroupOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
      </ElFormItem>

      <div v-if="disablePermissionGroup" class="dialog-tip">
        当前管理员不能在这里修改自己的权限组。
      </div>
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
  import type { PermissionGroupRecord } from '@/api/admin'
  import type { FormInstance, FormRules } from 'element-plus'
  import { INPUT_LIMITS } from '@/utils/constants'

  defineOptions({ name: 'AdminDialog' })

  interface AdminDialogFormValue {
    id: number | null
    username: string
    email: string
    qq: string
    password: string
    permission_group_id: number | null
  }

  interface Props {
    visible: boolean
    mode: 'create' | 'edit'
    formData: AdminDialogFormValue
    permissionGroups?: PermissionGroupRecord[]
    currentAdminId?: number | null
    submitting?: boolean
  }

  interface Emits {
    (e: 'update:visible', value: boolean): void
    (e: 'submit', value: AdminDialogFormValue): void
  }

  const props = withDefaults(defineProps<Props>(), {
    permissionGroups: () => [],
    currentAdminId: null,
    submitting: false
  })
  const emit = defineEmits<Emits>()

  const formRef = ref<FormInstance>()

  const dialogVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })

  const isEditMode = computed(() => props.mode === 'edit')
  const dialogTitle = computed(() => (isEditMode.value ? '编辑管理员' : '创建管理员'))
  const disablePermissionGroup = computed(
    () =>
      isEditMode.value &&
      props.currentAdminId !== null &&
      props.formData.id === props.currentAdminId
  )

  const localForm = reactive<AdminDialogFormValue>(createDefaultForm())

  const permissionGroupOptions = computed(
    () =>
      props.permissionGroups
        .map((group) => {
          const value = normalizeNullableNumber(group.id ?? group.ID)
          return {
            label: String(group.name ?? group.Name ?? ''),
            value
          }
        })
        .filter((item) => item.value !== null) as Array<{ label: string; value: number }>
  )

  const rules = computed<FormRules>(() => ({
    username: [
      { required: true, message: '请输入用户名', trigger: 'blur' },
      {
        max: INPUT_LIMITS.USERNAME,
        message: `用户名长度不能超过 ${INPUT_LIMITS.USERNAME} 个字符`,
        trigger: 'blur'
      }
    ],
    email: [
      { required: true, message: '请输入邮箱', trigger: 'blur' },
      { type: 'email', message: '请输入正确的邮箱地址', trigger: ['blur', 'change'] },
      {
        max: INPUT_LIMITS.EMAIL,
        message: `邮箱长度不能超过 ${INPUT_LIMITS.EMAIL} 个字符`,
        trigger: 'blur'
      }
    ],
    qq: [
      {
        validator: (_rule, value, callback) => {
          const text = String(value || '').trim()
          if (text && !/^\d+$/.test(text)) {
            callback(new Error('QQ 号必须是数字'))
            return
          }
          callback()
        },
        trigger: 'blur'
      },
      {
        max: INPUT_LIMITS.QQ,
        message: `QQ 长度不能超过 ${INPUT_LIMITS.QQ} 个字符`,
        trigger: 'blur'
      }
    ],
    password: isEditMode.value
      ? []
      : [
          { required: true, message: '请输入密码', trigger: 'blur' },
          {
            max: INPUT_LIMITS.PASSWORD,
            message: `密码长度不能超过 ${INPUT_LIMITS.PASSWORD} 个字符`,
            trigger: 'blur'
          }
        ],
    permission_group_id: [{ required: true, message: '请选择权限组', trigger: 'change' }]
  }))

  watch(
    () => [props.visible, props.formData] as const,
    ([visible]) => {
      if (!visible) {
        return
      }

      Object.assign(localForm, createDefaultForm(), {
        ...props.formData,
        permission_group_id: props.formData.permission_group_id ?? null
      })

      nextTick(() => formRef.value?.clearValidate())
    },
    { immediate: true, deep: true }
  )

  function createDefaultForm(): AdminDialogFormValue {
    return {
      id: null,
      username: '',
      email: '',
      qq: '',
      password: '',
      permission_group_id: null
    }
  }

  function normalizeNullableNumber(value: unknown): number | null {
    if (value === '' || value === null || value === undefined) {
      return null
    }

    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : null
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
      username: localForm.username.trim(),
      email: localForm.email.trim(),
      qq: localForm.qq.trim()
    })
  }
</script>

<style scoped lang="scss">
  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  .dialog-tip {
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.6;
  }

  .full-width {
    width: 100%;
  }
</style>
