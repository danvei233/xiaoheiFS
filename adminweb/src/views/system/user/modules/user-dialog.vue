<template>
  <ElDialog
    v-model="dialogVisible"
    :title="dialogTitle"
    width="680px"
    destroy-on-close
    align-center
  >
    <ElForm ref="formRef" :model="localForm" :rules="rules" label-width="110px">
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

      <ElFormItem v-if="!isEditMode" label="密码" prop="password">
        <ElInput
          v-model="localForm.password"
          type="password"
          show-password
          placeholder="请输入密码"
          :maxlength="INPUT_LIMITS.PASSWORD"
        />
      </ElFormItem>

      <template v-if="isEditMode">
        <ElFormItem label="QQ" prop="qq">
          <ElInput v-model="localForm.qq" placeholder="可选" :maxlength="INPUT_LIMITS.QQ" />
        </ElFormItem>

        <ElFormItem label="头像 URL" prop="avatar">
          <ElInput v-model="localForm.avatar" placeholder="可选" :maxlength="INPUT_LIMITS.URL" />
        </ElFormItem>

        <ElFormItem label="状态" prop="status">
          <ElSelect v-model="localForm.status" placeholder="请选择状态">
            <ElOption label="active" value="active" />
            <ElOption label="blocked" value="blocked" />
          </ElSelect>
        </ElFormItem>

        <ElFormItem label="用户组">
          <ElSelect v-model="localForm.userTierGroupId" clearable placeholder="请选择用户组">
            <ElOption
              v-for="group in tierGroups"
              :key="group.id"
              :label="group.name || '-'"
              :value="Number(group.id)"
            />
          </ElSelect>
        </ElFormItem>

        <ElFormItem label="用户组到期时间">
          <ElDatePicker
            v-model="localForm.userTierExpireAt"
            type="datetime"
            clearable
            placeholder="为空表示无限"
            style="width: 100%"
          />
        </ElFormItem>

        <ElDivider content-position="left">实名认证</ElDivider>

        <ElFormItem label="认证状态">
          <div class="realname-actions">
            <ElSelect v-model="realnameStatusModel" placeholder="选择实名状态" style="width: 180px">
              <ElOption label="待审核" value="pending" />
              <ElOption label="已通过" value="verified" />
              <ElOption label="未通过" value="failed" />
            </ElSelect>

            <ElInput
              v-model="realnameReasonModel"
              placeholder="审核备注（可选）"
              :maxlength="INPUT_LIMITS.REVIEW_REASON"
            />

            <ElButton
              type="primary"
              :loading="realnameUpdating"
              :disabled="!realnameRecord"
              @click="emit('update-realname-status')"
            >
              更新实名状态
            </ElButton>
          </div>
        </ElFormItem>

        <div v-if="realnameRecord" class="realname-summary">
          <ElTag :type="getRealnameTagType(realnameRecord.status)">
            {{ realnameRecord.status || '-' }}
          </ElTag>
          <span>审核备注：{{ realnameRecord.reason || '-' }}</span>
        </div>
        <div v-else class="dialog-tip">暂无实名认证记录，无法修改状态</div>
      </template>
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
  import type { RealNameVerification, UserTierGroup } from '@/api/admin'
  import type { FormInstance, FormRules } from 'element-plus'
  import { INPUT_LIMITS } from '@/utils/constants'

  defineOptions({ name: 'UserDialog' })

  interface UserDialogFormValue {
    id: number | null
    username: string
    email: string
    password: string
    qq: string
    avatar: string
    status: string
    userTierGroupId: number | null
    userTierExpireAt: Date | null
  }

  interface Props {
    visible: boolean
    mode: 'create' | 'edit'
    formData: UserDialogFormValue
    tierGroups?: UserTierGroup[]
    submitting?: boolean
    realnameRecord?: RealNameVerification | null
    realnameStatus?: string
    realnameReason?: string
    realnameUpdating?: boolean
  }

  interface Emits {
    (e: 'update:visible', value: boolean): void
    (e: 'submit', value: UserDialogFormValue): void
    (e: 'update:realnameStatus', value: string): void
    (e: 'update:realnameReason', value: string): void
    (e: 'update-realname-status'): void
  }

  const props = withDefaults(defineProps<Props>(), {
    tierGroups: () => [],
    submitting: false,
    realnameRecord: null,
    realnameStatus: '',
    realnameReason: '',
    realnameUpdating: false
  })
  const emit = defineEmits<Emits>()

  const formRef = ref<FormInstance>()

  const dialogVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })

  const isEditMode = computed(() => props.mode === 'edit')
  const dialogTitle = computed(() => (isEditMode.value ? '编辑用户' : '创建用户'))

  const localForm = reactive<UserDialogFormValue>(createDefaultForm())

  const realnameStatusModel = computed({
    get: () => props.realnameStatus,
    set: (value) => emit('update:realnameStatus', value)
  })

  const realnameReasonModel = computed({
    get: () => props.realnameReason,
    set: (value) => emit('update:realnameReason', value)
  })

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
    password: isEditMode.value
      ? [
          {
            max: INPUT_LIMITS.PASSWORD,
            message: `密码长度不能超过 ${INPUT_LIMITS.PASSWORD} 个字符`,
            trigger: 'blur'
          }
        ]
      : [
          { required: true, message: '请输入密码', trigger: 'blur' },
          {
            max: INPUT_LIMITS.PASSWORD,
            message: `密码长度不能超过 ${INPUT_LIMITS.PASSWORD} 个字符`,
            trigger: 'blur'
          }
        ],
    qq: [
      {
        max: INPUT_LIMITS.QQ,
        message: `QQ 长度不能超过 ${INPUT_LIMITS.QQ} 个字符`,
        trigger: 'blur'
      }
    ],
    avatar: [
      {
        max: INPUT_LIMITS.URL,
        message: `头像 URL 长度不能超过 ${INPUT_LIMITS.URL} 个字符`,
        trigger: 'blur'
      }
    ]
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

  function createDefaultForm(): UserDialogFormValue {
    return {
      id: null,
      username: '',
      email: '',
      password: '',
      qq: '',
      avatar: '',
      status: 'active',
      userTierGroupId: null,
      userTierExpireAt: null
    }
  }

  function cloneDate(value: Date | null) {
    if (!value) {
      return null
    }

    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? null : date
  }

  function applyFormData(value: UserDialogFormValue) {
    Object.assign(localForm, createDefaultForm(), {
      ...value,
      userTierGroupId: value.userTierGroupId ?? null,
      userTierExpireAt: cloneDate(value.userTierExpireAt)
    })
  }

  function getRealnameTagType(status?: string) {
    if (status === 'verified') {
      return 'success' as const
    }

    if (status === 'failed') {
      return 'danger' as const
    }

    return 'warning' as const
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
      qq: localForm.qq.trim(),
      avatar: localForm.avatar.trim(),
      userTierGroupId: localForm.userTierGroupId ?? null,
      userTierExpireAt: cloneDate(localForm.userTierExpireAt)
    })
  }
</script>

<style scoped lang="scss">
  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  .realname-actions {
    display: grid;
    width: 100%;
    grid-template-columns: 180px minmax(0, 1fr) auto;
    gap: 12px;
    align-items: center;
  }

  .realname-summary {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-top: -6px;
    padding-left: 110px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .dialog-tip {
    margin-top: -6px;
    padding-left: 110px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  @media (max-width: 768px) {
    .realname-actions {
      grid-template-columns: 1fr;
    }

    .realname-summary,
    .dialog-tip {
      padding-left: 0;
    }
  }
</style>
