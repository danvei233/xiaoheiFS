<template>
  <ArtSearchBar
    ref="searchBarRef"
    v-model="formData"
    :items="formItems"
    :showExpand="false"
    @reset="handleReset"
    @search="handleSearch"
  />
</template>

<script setup lang="ts">
  interface AdminSearchForm {
    keyword: string
    status: 'all' | 'active' | 'disabled'
  }

  interface Props {
    modelValue: AdminSearchForm
  }

  interface Emits {
    (e: 'update:modelValue', value: AdminSearchForm): void
    (e: 'search', value: AdminSearchForm): void
    (e: 'reset'): void
  }

  defineOptions({ name: 'AdminSearch' })

  const props = defineProps<Props>()
  const emit = defineEmits<Emits>()

  const searchBarRef = ref()

  const formData = computed({
    get: () => props.modelValue,
    set: (value) => emit('update:modelValue', value)
  })

  const formItems = computed(() => [
    {
      label: '状态',
      key: 'status',
      type: 'select',
      props: {
        clearable: false,
        placeholder: '请选择状态',
        options: [
          { label: '全部', value: 'all' },
          { label: '启用', value: 'active' },
          { label: '禁用', value: 'disabled' }
        ]
      }
    },
    {
      label: '关键词',
      key: 'keyword',
      type: 'input',
      props: {
        clearable: true,
        placeholder: '按用户名、邮箱或 QQ 搜索'
      }
    }
  ])

  function handleReset() {
    emit('reset')
  }

  async function handleSearch(params: Record<string, unknown>) {
    await searchBarRef.value?.validate?.()
    const status =
      params.status === 'active' || params.status === 'disabled' || params.status === 'all'
        ? params.status
        : 'all'

    emit('search', {
      keyword: String(params.keyword || ''),
      status
    })
  }
</script>
