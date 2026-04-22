<template>
  <ArtSearchBar
    ref="searchBarRef"
    v-model="formData"
    :items="formItems"
    :rules="rules"
    :showExpand="false"
    @reset="handleReset"
    @search="handleSearch"
  />
</template>

<script setup lang="ts">
  interface UserSearchForm {
    keyword: string
    status?: string
  }

  interface Props {
    modelValue: UserSearchForm
  }

  interface Emits {
    (e: 'update:modelValue', value: UserSearchForm): void
    (e: 'search', params: UserSearchForm): void
    (e: 'reset'): void
  }

  defineOptions({ name: 'UserSearch' })

  const props = defineProps<Props>()
  const emit = defineEmits<Emits>()

  const searchBarRef = ref()
  const rules = {}
  const statusOptions = ref([
    { label: 'active', value: 'active' },
    { label: 'blocked', value: 'blocked' }
  ])

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
        placeholder: '按用户名或邮箱搜索'
      }
    },
    {
      label: '状态',
      key: 'status',
      type: 'select',
      props: {
        clearable: true,
        placeholder: '请选择状态',
        options: statusOptions.value
      }
    }
  ])

  function handleReset() {
    emit('reset')
  }

  async function handleSearch(params: Record<string, any>) {
    await searchBarRef.value.validate()
    emit('search', params as UserSearchForm)
  }
</script>
