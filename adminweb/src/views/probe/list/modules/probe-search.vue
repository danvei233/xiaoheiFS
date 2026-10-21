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
  interface ProbeSearchForm {
    keyword: string
    status?: string
  }

  interface Props {
    modelValue: ProbeSearchForm
  }

  interface Emits {
    (e: 'update:modelValue', value: ProbeSearchForm): void
    (e: 'search', value: ProbeSearchForm): void
    (e: 'reset'): void
  }

  defineOptions({ name: 'ProbeSearch' })

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
        clearable: true,
        placeholder: '请选择状态',
        options: [
          { label: '在线', value: 'online' },
          { label: '离线', value: 'offline' }
        ]
      }
    },
    {
      label: '关键词',
      key: 'keyword',
      type: 'input',
      props: {
        clearable: true,
        placeholder: '按名称或 Agent ID 搜索'
      }
    }
  ])

  function handleReset() {
    emit('reset')
  }

  async function handleSearch(params: Record<string, unknown>) {
    await searchBarRef.value?.validate?.()
    emit('search', {
      keyword: String(params.keyword || ''),
      status: params.status ? String(params.status) : undefined
    })
  }
</script>
