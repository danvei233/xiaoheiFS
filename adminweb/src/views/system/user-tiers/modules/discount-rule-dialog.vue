<template>
  <ElDialog
    v-model="dialogVisible"
    :title="dialogTitle"
    width="820px"
    destroy-on-close
    align-center
  >
    <ElForm ref="formRef" :model="localForm" :rules="rules" label-position="top">
      <ElFormItem label="对象范围" prop="scope">
        <ElSelect v-model="localForm.scope" placeholder="请选择范围" class="full-width">
          <ElOption
            v-for="option in scopeOptions"
            :key="option.value"
            :label="option.label"
            :value="option.value"
          />
        </ElSelect>
      </ElFormItem>

      <ElAlert
        type="info"
        :closable="false"
        class="discount-help"
        title="折扣说明"
        description="折扣字段是减免值 ÷ 10000。计算公式：最终价 = 原价 × (1 - 折扣/10000)。例如原价 10 元，填 1000 后为 9 元。"
      />

      <ElRow v-if="needsTargetSelection" :gutter="12">
        <ElCol v-if="needGoodsType" :span="6">
          <ElFormItem label="类型" prop="goods_type_id">
            <ElSelect
              v-model="localForm.goods_type_id"
              clearable
              filterable
              placeholder="请选择类型"
              class="full-width"
              @change="handleGoodsTypeChange"
            >
              <ElOption
                v-for="item in goodsTypeOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </ElSelect>
          </ElFormItem>
        </ElCol>

        <ElCol v-if="needRegion" :span="6">
          <ElFormItem label="地区" prop="region_id">
            <ElSelect
              v-model="localForm.region_id"
              clearable
              filterable
              :disabled="!localForm.goods_type_id"
              placeholder="请选择地区"
              class="full-width"
              @change="handleRegionChange"
            >
              <ElOption
                v-for="item in regionOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </ElSelect>
          </ElFormItem>
        </ElCol>

        <ElCol v-if="needPlanGroup" :span="6">
          <ElFormItem label="线路" prop="plan_group_id">
            <ElSelect
              v-model="localForm.plan_group_id"
              clearable
              filterable
              :disabled="!localForm.goods_type_id"
              placeholder="请选择线路"
              class="full-width"
              @change="handlePlanGroupChange"
            >
              <ElOption
                v-for="item in planGroupOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </ElSelect>
          </ElFormItem>
        </ElCol>

        <ElCol v-if="needPackage" :span="6">
          <ElFormItem label="套餐" prop="package_id">
            <ElSelect
              v-model="localForm.package_id"
              clearable
              filterable
              :disabled="!localForm.plan_group_id"
              placeholder="请选择套餐"
              class="full-width"
            >
              <ElOption
                v-for="item in packageOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </ElSelect>
          </ElFormItem>
        </ElCol>
      </ElRow>

      <ElRow :gutter="12">
        <ElCol :span="8">
          <ElFormItem label="折扣(‰)" prop="discount_permille">
            <ElInputNumber
              v-model="localForm.discount_permille"
              :min="0"
              :max="10000"
              class="full-width"
            />
          </ElFormItem>
        </ElCol>

        <ElCol :span="8">
          <ElFormItem label="固定价格(分)" prop="fixed_price">
            <ElInputNumber v-model="localForm.fixed_price" :min="0" class="full-width" />
          </ElFormItem>
        </ElCol>

        <ElCol :span="8">
          <ElFormItem label="核心附加折扣(‰)" prop="add_core_permille">
            <ElInputNumber
              v-model="localForm.add_core_permille"
              :min="0"
              :max="10000"
              class="full-width"
            />
          </ElFormItem>
        </ElCol>
      </ElRow>

      <ElRow :gutter="12">
        <ElCol :span="8">
          <ElFormItem label="内存附加折扣(‰)" prop="add_mem_permille">
            <ElInputNumber
              v-model="localForm.add_mem_permille"
              :min="0"
              :max="10000"
              class="full-width"
            />
          </ElFormItem>
        </ElCol>

        <ElCol :span="8">
          <ElFormItem label="磁盘附加折扣(‰)" prop="add_disk_permille">
            <ElInputNumber
              v-model="localForm.add_disk_permille"
              :min="0"
              :max="10000"
              class="full-width"
            />
          </ElFormItem>
        </ElCol>

        <ElCol :span="8">
          <ElFormItem label="带宽附加折扣(‰)" prop="add_bw_permille">
            <ElInputNumber
              v-model="localForm.add_bw_permille"
              :min="0"
              :max="10000"
              class="full-width"
            />
          </ElFormItem>
        </ElCol>
      </ElRow>
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
  import type {
    CatalogGoodsType,
    CatalogPackage,
    CatalogPlanGroup,
    CatalogRegion
  } from '@/api/admin'
  import type { FormInstance, FormRules } from 'element-plus'

  defineOptions({ name: 'UserTierDiscountRuleDialog' })

  export interface UserTierDiscountRuleFormValue {
    id: number | null
    scope: string
    goods_type_id: number
    region_id: number
    plan_group_id: number
    package_id: number
    discount_permille: number
    fixed_price: number | null
    add_core_permille: number
    add_mem_permille: number
    add_disk_permille: number
    add_bw_permille: number
  }

  interface Props {
    visible: boolean
    mode: 'create' | 'edit'
    formData: UserTierDiscountRuleFormValue
    goodsTypes?: CatalogGoodsType[]
    regions?: CatalogRegion[]
    planGroups?: CatalogPlanGroup[]
    packages?: CatalogPackage[]
    submitting?: boolean
  }

  interface Emits {
    (e: 'update:visible', value: boolean): void
    (e: 'submit', value: UserTierDiscountRuleFormValue): void
  }

  const scopeOptions = [
    { value: 'all', label: '全部(不含附加项)' },
    { value: 'all_addons', label: '全部附加项' },
    { value: 'goods_type', label: '类型' },
    { value: 'goods_type_region', label: '类型-地区' },
    { value: 'plan_group', label: '类型-地区-线路' },
    { value: 'package', label: '套餐' },
    { value: 'addon_config', label: '附加项配置' }
  ]

  const props = withDefaults(defineProps<Props>(), {
    goodsTypes: () => [],
    regions: () => [],
    planGroups: () => [],
    packages: () => [],
    submitting: false
  })
  const emit = defineEmits<Emits>()

  const formRef = ref<FormInstance>()

  const dialogVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })

  const dialogTitle = computed(() => (props.mode === 'edit' ? '编辑优惠规则' : '新增优惠规则'))
  const localForm = reactive<UserTierDiscountRuleFormValue>(createDefaultForm())

  const needGoodsType = computed(() =>
    ['goods_type', 'goods_type_region', 'plan_group', 'addon_config', 'package'].includes(
      localForm.scope
    )
  )
  const needRegion = computed(() =>
    ['goods_type_region', 'plan_group', 'addon_config', 'package'].includes(localForm.scope)
  )
  const needPlanGroup = computed(() =>
    ['plan_group', 'addon_config', 'package'].includes(localForm.scope)
  )
  const needPackage = computed(() => localForm.scope === 'package')
  const needsTargetSelection = computed(
    () => needGoodsType.value || needRegion.value || needPlanGroup.value || needPackage.value
  )

  const goodsTypeOptions = computed(() =>
    props.goodsTypes.map((item) => ({
      label: String(item.name || `#${item.id}`),
      value: Number(item.id || 0)
    }))
  )

  const regionOptions = computed(() =>
    props.regions
      .filter(
        (item) =>
          !localForm.goods_type_id ||
          Number(item.goods_type_id || 0) === Number(localForm.goods_type_id)
      )
      .map((item) => ({
        label: String(item.name || `#${item.id}`),
        value: Number(item.id || 0)
      }))
  )

  const planGroupOptions = computed(() =>
    props.planGroups
      .filter(
        (item) =>
          !localForm.goods_type_id ||
          Number(item.goods_type_id || 0) === Number(localForm.goods_type_id)
      )
      .filter(
        (item) =>
          !localForm.region_id || Number(item.region_id || 0) === Number(localForm.region_id)
      )
      .map((item) => ({
        label: String(item.name || `#${item.id}`),
        value: Number(item.id || 0)
      }))
  )

  const packageOptions = computed(() =>
    props.packages
      .filter(
        (item) =>
          !localForm.goods_type_id ||
          Number(item.goods_type_id || 0) === Number(localForm.goods_type_id)
      )
      .filter(
        (item) =>
          !localForm.plan_group_id ||
          Number(item.plan_group_id || 0) === Number(localForm.plan_group_id)
      )
      .map((item) => ({
        label: String(item.name || `#${item.id}`),
        value: Number(item.id || 0)
      }))
  )

  const rules = computed<FormRules>(() => ({
    scope: [{ required: true, message: '请选择范围', trigger: 'change' }]
  }))

  watch(
    () => props.visible,
    (visible) => {
      if (!visible) {
        return
      }

      Object.assign(localForm, createDefaultForm(), props.formData)
      normalizeTargetByScope()
      nextTick(() => formRef.value?.clearValidate())
    },
    { immediate: true }
  )

  watch(
    () => localForm.scope,
    () => {
      normalizeTargetByScope()
    }
  )

  function createDefaultForm(): UserTierDiscountRuleFormValue {
    return {
      id: null,
      scope: 'all',
      goods_type_id: 0,
      region_id: 0,
      plan_group_id: 0,
      package_id: 0,
      discount_permille: 1000,
      fixed_price: null,
      add_core_permille: 1000,
      add_mem_permille: 1000,
      add_disk_permille: 1000,
      add_bw_permille: 1000
    }
  }

  function normalizeTargetByScope() {
    if (!needGoodsType.value) {
      localForm.goods_type_id = 0
    }
    if (!needRegion.value) {
      localForm.region_id = 0
    }
    if (!needPlanGroup.value) {
      localForm.plan_group_id = 0
    }
    if (!needPackage.value) {
      localForm.package_id = 0
    }
  }

  function handleGoodsTypeChange() {
    if (!needRegion.value) {
      return
    }

    localForm.region_id = 0
    localForm.plan_group_id = 0
    localForm.package_id = 0
  }

  function handleRegionChange() {
    if (!needPlanGroup.value) {
      return
    }

    localForm.plan_group_id = 0
    localForm.package_id = 0
  }

  function handlePlanGroupChange() {
    if (!needPackage.value) {
      return
    }

    localForm.package_id = 0
  }

  async function handleSubmit() {
    if (!formRef.value) {
      return
    }

    const valid = await formRef.value.validate().catch(() => false)
    if (!valid) {
      return
    }

    if (needGoodsType.value && !localForm.goods_type_id) {
      ElMessage.error('请选择类型')
      return
    }
    if (needRegion.value && !localForm.region_id) {
      ElMessage.error('请选择地区')
      return
    }
    if (needPlanGroup.value && !localForm.plan_group_id) {
      ElMessage.error('请选择线路')
      return
    }
    if (needPackage.value && !localForm.package_id) {
      ElMessage.error('请选择套餐')
      return
    }

    emit('submit', {
      ...localForm,
      scope: String(localForm.scope || 'all')
    })
  }
</script>

<style scoped lang="scss">
  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  .discount-help {
    margin-bottom: 16px;
  }

  .full-width {
    width: 100%;
  }
</style>
