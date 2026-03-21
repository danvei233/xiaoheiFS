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
  defineOptions({ name: 'VpsSearch' })

  interface VpsSearchForm {
    keyword: string
    status?: string
  }

  interface Props {
    modelValue: VpsSearchForm
  }

  interface Emits {
    (e: 'update:modelValue', value: VpsSearchForm): void
    (e: 'search', params: VpsSearchForm): void
    (e: 'reset'): void
  }

  const props = defineProps<Props>()
  const emit = defineEmits<Emits>()

  const searchBarRef = ref()

  const statusOptions = [
    { label: 'normal', value: 'normal' },
    { label: 'abuse', value: 'abuse' },
    { label: 'fraud', value: 'fraud' },
    { label: 'locked', value: 'locked' }
  ]

  const formData = computed({
    get: () => props.modelValue,
    set: (value) => emit('update:modelValue', value)
  })

  const formItems = computed(() => [
    {
      label: '关键词',
      key: 'keyword',
      type: 'input',
      props: {
        clearable: true,
        placeholder: '按实例名、用户 ID、地区、套餐搜索'
      }
    },
    {
      label: '管理状态',
      key: 'status',
      type: 'select',
      props: {
        clearable: true,
        placeholder: '请选择状态',
        options: statusOptions
      }
    }
  ])

  function handleReset() {
    emit('reset')
  }

  async function handleSearch(params: Record<string, any>) {
    await searchBarRef.value?.validate?.()
    emit('search', params as VpsSearchForm)
  }
</script>
