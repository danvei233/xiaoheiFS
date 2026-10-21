<template>
  <ElDialog
    v-model="dialogVisible"
    :title="dialogTitle"
    width="760px"
    destroy-on-close
    align-center
  >
    <ElForm ref="formRef" :model="localForm" :rules="rules" label-position="top">
      <ElFormItem label="名称" prop="name">
        <ElInput v-model="localForm.name" maxlength="120" placeholder="请输入权限组名称" />
      </ElFormItem>

      <ElFormItem label="描述" prop="description">
        <ElInput
          v-model="localForm.description"
          type="textarea"
          :rows="3"
          maxlength="500"
          show-word-limit
          placeholder="请输入权限组描述"
        />
      </ElFormItem>

      <ElFormItem label="权限" prop="permissions">
        <div class="permission-toolbar">
          <ElInput v-model="permissionSearch" clearable placeholder="搜索权限代码、名称或分类" />

          <ElSpace wrap>
            <ElButton @click="selectAll">全选</ElButton>
            <ElButton @click="clearAll">清空</ElButton>
          </ElSpace>
        </div>

        <div class="permission-summary"> 已选 {{ localForm.permissions.length }} 项权限 </div>

        <div class="permission-list">
          <ElCollapse v-model="activeGroups">
            <ElCollapseItem
              v-for="(items, category) in groupedPermissions"
              :key="category"
              :name="category"
            >
              <template #title>
                <div class="group-title">
                  <span>{{ category }}</span>
                  <ElTag size="small" type="info">{{ items.length }}</ElTag>
                </div>
              </template>

              <ElCheckboxGroup v-model="localForm.permissions" class="permission-grid">
                <ElCheckbox
                  v-for="permission in items"
                  :key="permission.code"
                  :value="permission.code"
                  class="permission-item"
                >
                  <div class="permission-text">
                    <div class="permission-name">{{ getPermissionLabel(permission) }}</div>
                    <div class="permission-code">{{ permission.code }}</div>
                  </div>
                </ElCheckbox>
              </ElCheckboxGroup>
            </ElCollapseItem>
          </ElCollapse>
        </div>
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
  import type { PermissionRecord } from '@/api/admin'
  import type { FormInstance, FormRules } from 'element-plus'

  defineOptions({ name: 'PermissionGroupDialog' })

  interface PermissionGroupDialogFormValue {
    id: number | null
    name: string
    description: string
    permissions: string[]
  }

  interface Props {
    visible: boolean
    mode: 'create' | 'edit'
    formData: PermissionGroupDialogFormValue
    permissions?: PermissionRecord[]
    submitting?: boolean
  }

  interface Emits {
    (e: 'update:visible', value: boolean): void
    (e: 'submit', value: PermissionGroupDialogFormValue): void
  }

  const props = withDefaults(defineProps<Props>(), {
    permissions: () => [],
    submitting: false
  })
  const emit = defineEmits<Emits>()

  const formRef = ref<FormInstance>()
  const permissionSearch = ref('')
  const activeGroups = ref<string[]>([])

  const dialogVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })

  const dialogTitle = computed(() => (props.mode === 'edit' ? '编辑权限组' : '创建权限组'))

  const localForm = reactive<PermissionGroupDialogFormValue>(createDefaultForm())

  const normalizedPermissions = computed(() =>
    [...props.permissions]
      .map((item) => ({
        code: String(item.code || ''),
        name: String(item.name || ''),
        friendly_name: String(item.friendly_name || ''),
        category: String(item.category || '其他'),
        parent_code: String(item.parent_code || ''),
        sort_order: Number(item.sort_order || 0)
      }))
      .filter((item) => item.code)
      .sort((a, b) => {
        if (a.category !== b.category) {
          return a.category.localeCompare(b.category, 'zh-CN')
        }
        if (a.sort_order !== b.sort_order) {
          return a.sort_order - b.sort_order
        }
        return a.code.localeCompare(b.code, 'zh-CN')
      })
  )

  const filteredPermissions = computed(() => {
    const keyword = permissionSearch.value.trim().toLowerCase()
    if (!keyword) {
      return normalizedPermissions.value
    }

    return normalizedPermissions.value.filter((item) =>
      [item.code, item.name, item.friendly_name, item.category]
        .filter(Boolean)
        .some((value) => value.toLowerCase().includes(keyword))
    )
  })

  const groupedPermissions = computed(() => {
    const groups: Record<string, typeof filteredPermissions.value> = {}

    filteredPermissions.value.forEach((item) => {
      const category = item.category || '其他'
      if (!groups[category]) {
        groups[category] = []
      }
      groups[category].push(item)
    })

    return groups
  })

  const rules = computed<FormRules>(() => ({
    name: [{ required: true, message: '请输入权限组名称', trigger: 'blur' }],
    permissions: [
      {
        validator: (_rule, value, callback) => {
          if (!Array.isArray(value) || value.length === 0) {
            callback(new Error('请至少选择一个权限'))
            return
          }
          callback()
        },
        trigger: 'change'
      }
    ]
  }))

  watch(
    () => props.visible,
    (visible) => {
      if (!visible) {
        return
      }

      Object.assign(localForm, createDefaultForm(), {
        ...props.formData,
        permissions: Array.isArray(props.formData.permissions)
          ? [...props.formData.permissions]
          : []
      })

      permissionSearch.value = ''
      syncActiveGroups()
      nextTick(() => formRef.value?.clearValidate())
    },
    { immediate: true }
  )

  watch(
    () => groupedPermissions.value,
    () => {
      if (props.visible) {
        syncActiveGroups()
      }
    }
  )

  function createDefaultForm(): PermissionGroupDialogFormValue {
    return {
      id: null,
      name: '',
      description: '',
      permissions: []
    }
  }

  function syncActiveGroups() {
    activeGroups.value = Object.keys(groupedPermissions.value)
  }

  function getPermissionLabel(permission: PermissionRecord) {
    return String(permission.friendly_name || permission.name || permission.code || '-')
  }

  function selectAll() {
    localForm.permissions = Array.from(
      new Set(normalizedPermissions.value.map((item) => item.code).filter(Boolean))
    )
  }

  function clearAll() {
    localForm.permissions = []
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
      description: localForm.description.trim(),
      permissions: Array.from(new Set(localForm.permissions))
    })
  }
</script>

<style scoped lang="scss">
  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  .permission-toolbar {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 12px;
    align-items: center;
    margin-bottom: 12px;
  }

  .permission-summary {
    margin-bottom: 12px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .permission-list {
    max-height: 420px;
    overflow: auto;
    padding: 8px 12px;
    border: 1px solid var(--el-border-color-light);
    border-radius: 12px;
    background: var(--el-bg-color-page);
  }

  .group-title {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }

  .permission-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px;
    width: 100%;
  }

  .permission-item {
    min-width: 0;
    margin-right: 0;
    padding: 10px 12px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 10px;
    background: var(--el-bg-color);
  }

  .permission-text {
    min-width: 0;
  }

  .permission-name {
    color: var(--el-text-color-primary);
    font-size: 13px;
    font-weight: 600;
    line-height: 1.4;
    word-break: break-word;
  }

  .permission-code {
    margin-top: 4px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.4;
    word-break: break-all;
  }

  @media (max-width: 768px) {
    .permission-toolbar {
      grid-template-columns: 1fr;
    }

    .permission-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
