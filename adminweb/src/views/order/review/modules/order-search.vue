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
  interface OrderSearchForm {
    keyword: string
    status?: string
    user_id: string
    order_no: string
  }

  interface Props {
    modelValue: OrderSearchForm
  }

  interface Emits {
    (e: 'update:modelValue', value: OrderSearchForm): void
    (e: 'search', value: OrderSearchForm): void
    (e: 'reset'): void
  }

  defineOptions({ name: 'OrderSearch' })

  const props = defineProps<Props>()
  const emit = defineEmits<Emits>()

  const searchBarRef = ref()

  const statusOptions = computed(() => [
    { label: '待支付', value: 'pending_payment' },
    { label: '待审核', value: 'pending_review' },
    { label: '已通过', value: 'approved' },
    { label: '开通中', value: 'provisioning' },
    { label: '已完成', value: 'active' },
    { label: '失败', value: 'failed' },
    { label: '已驳回', value: 'rejected' }
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
        placeholder: '按订单 ID 搜索'
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
    },
    {
      label: '用户 ID',
      key: 'user_id',
      type: 'input',
      props: {
        clearable: true,
        placeholder: '请输入用户 ID'
      }
    },
    {
      label: '订单号',
      key: 'order_no',
      type: 'input',
      props: {
        clearable: true,
        placeholder: '请输入订单号'
      }
    }
  ])

  function handleReset() {
    emit('reset')
  }

  async function handleSearch(params: Record<string, any>) {
    await searchBarRef.value?.validate?.()
    emit('search', params as OrderSearchForm)
  }
</script>
