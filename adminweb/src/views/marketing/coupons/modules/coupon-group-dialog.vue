<template>
  <ElDialog
    v-model="dialogVisible"
    :title="localForm.id ? 'Edit Coupon Group' : 'Create Coupon Group'"
    width="980px"
    destroy-on-close
    align-center
  >
    <ElForm label-position="top">
      <ElFormItem label="Group Name">
        <ElInput
          v-model.trim="localForm.name"
          :maxlength="INPUT_LIMITS.COUPON_GROUP_NAME"
          placeholder="Enter coupon group name"
        />
      </ElFormItem>

      <div class="rule-header">
        <span>Product Rules</span>
        <ElButton type="primary" plain @click="addRule">Add Rule</ElButton>
      </div>

      <div v-for="(rule, index) in localForm.rules" :key="`rule-${index}`" class="rule-card">
        <div class="rule-card-header">
          <span>Rule {{ index + 1 }}</span>
          <ElButton
            link
            type="danger"
            :disabled="localForm.rules.length <= 1"
            @click="removeRule(index)"
          >
            Remove
          </ElButton>
        </div>

        <ElRow :gutter="12">
          <ElCol :xs="24" :md="8">
            <ElFormItem label="Scope">
              <ElSelect v-model="rule.scope" class="full-width">
                <ElOption
                  v-for="item in scopeOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>

          <ElCol :xs="24" :md="8">
            <ElFormItem label="Goods Type">
              <ElSelect
                v-model="rule.goods_type_id"
                clearable
                filterable
                class="full-width"
                :disabled="!needGoodsType(rule.scope)"
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

          <ElCol :xs="24" :md="8">
            <ElFormItem label="Region">
              <ElSelect
                v-model="rule.region_id"
                clearable
                filterable
                class="full-width"
                :disabled="!needRegion(rule.scope)"
              >
                <ElOption
                  v-for="item in getRegionOptions(rule)"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>

          <ElCol :xs="24" :md="12">
            <ElFormItem label="Plan Group">
              <ElSelect
                v-model="rule.plan_group_id"
                clearable
                filterable
                class="full-width"
                :disabled="!needPlanGroup(rule.scope)"
              >
                <ElOption
                  v-for="item in getPlanGroupOptions(rule)"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>

          <ElCol :xs="24" :md="12">
            <ElFormItem label="Package">
              <ElSelect
                v-model="rule.package_id"
                clearable
                filterable
                class="full-width"
                :disabled="!needPackage(rule.scope)"
              >
                <ElOption
                  v-for="item in getPackageOptions(rule)"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElRow v-if="rule.scope === 'addon_config'" :gutter="12">
          <ElCol :xs="12" :md="6">
            <ElFormItem label="Addon CPU">
              <ElSwitch v-model="rule.addon_core_enabled" />
            </ElFormItem>
          </ElCol>
          <ElCol :xs="12" :md="6">
            <ElFormItem label="Addon Memory">
              <ElSwitch v-model="rule.addon_mem_enabled" />
            </ElFormItem>
          </ElCol>
          <ElCol :xs="12" :md="6">
            <ElFormItem label="Addon Disk">
              <ElSwitch v-model="rule.addon_disk_enabled" />
            </ElFormItem>
          </ElCol>
          <ElCol :xs="12" :md="6">
            <ElFormItem label="Addon Bandwidth">
              <ElSwitch v-model="rule.addon_bw_enabled" />
            </ElFormItem>
          </ElCol>
        </ElRow>
      </div>
    </ElForm>

    <template #footer>
      <div class="dialog-footer">
        <ElButton @click="dialogVisible = false">Cancel</ElButton>
        <ElButton type="primary" :loading="submitting" @click="handleSubmit">Save</ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { INPUT_LIMITS } from '@/utils/constants'

  defineOptions({ name: 'CouponGroupDialog' })

  interface CouponRuleFormValue {
    scope: string
    goods_type_id: number | null
    region_id: number | null
    plan_group_id: number | null
    package_id: number | null
    addon_core_enabled: boolean
    addon_mem_enabled: boolean
    addon_disk_enabled: boolean
    addon_bw_enabled: boolean
  }

  interface CouponGroupFormValue {
    id: number | null
    name: string
    rules: CouponRuleFormValue[]
  }

  interface OptionItem {
    id: number | null
    name: string
    goods_type_id?: number | null
    region_id?: number | null
    plan_group_id?: number | null
  }

  interface SelectOption {
    label: string
    value: number
  }

  interface Props {
    visible: boolean
    formData: CouponGroupFormValue
    goodsTypes?: OptionItem[]
    regions?: OptionItem[]
    planGroups?: OptionItem[]
    packages?: OptionItem[]
    submitting?: boolean
  }

  interface Emits {
    (e: 'update:visible', value: boolean): void
    (e: 'submit', value: CouponGroupFormValue): void
  }

  const scopeOptions = [
    { label: 'All Products', value: 'all' },
    { label: 'All Addons', value: 'all_addons' },
    { label: 'Goods Type', value: 'goods_type' },
    { label: 'Goods Type + Region', value: 'goods_type_region' },
    { label: 'Plan Group', value: 'plan_group' },
    { label: 'Package', value: 'package' },
    { label: 'Addon Config', value: 'addon_config' }
  ]

  const props = withDefaults(defineProps<Props>(), {
    goodsTypes: () => [],
    regions: () => [],
    planGroups: () => [],
    packages: () => [],
    submitting: false
  })
  const emit = defineEmits<Emits>()

  const localForm = reactive<CouponGroupFormValue>(createDefaultForm())

  const dialogVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })

  const goodsTypeOptions = computed<SelectOption[]>(() =>
    props.goodsTypes
      .filter((item) => item.id !== null)
      .map((item) => ({
        label: item.name || `#${item.id}`,
        value: Number(item.id)
      }))
  )

  watch(
    () => [props.visible, props.formData] as const,
    ([visible]) => {
      if (!visible) {
        return
      }

      Object.assign(localForm, createDefaultForm(), {
        ...props.formData,
        rules: normalizeRules(props.formData.rules)
      })
    },
    { immediate: true, deep: true }
  )

  function createEmptyRule(): CouponRuleFormValue {
    return {
      scope: 'all',
      goods_type_id: null,
      region_id: null,
      plan_group_id: null,
      package_id: null,
      addon_core_enabled: false,
      addon_mem_enabled: false,
      addon_disk_enabled: false,
      addon_bw_enabled: false
    }
  }

  function createDefaultForm(): CouponGroupFormValue {
    return {
      id: null,
      name: '',
      rules: [createEmptyRule()]
    }
  }

  function normalizeRule(rule?: Partial<CouponRuleFormValue>): CouponRuleFormValue {
    return {
      ...createEmptyRule(),
      ...rule,
      goods_type_id: toNullableNumber(rule?.goods_type_id),
      region_id: toNullableNumber(rule?.region_id),
      plan_group_id: toNullableNumber(rule?.plan_group_id),
      package_id: toNullableNumber(rule?.package_id)
    }
  }

  function normalizeRules(rules?: CouponRuleFormValue[]) {
    const items = Array.isArray(rules) ? rules.map((item) => normalizeRule(item)) : []
    return items.length ? items : [createEmptyRule()]
  }

  function toNullableNumber(value: unknown) {
    if (value === null || value === undefined || value === '' || Number(value) <= 0) {
      return null
    }

    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : null
  }

  function needGoodsType(scope?: string) {
    return ['goods_type', 'goods_type_region', 'plan_group', 'package', 'addon_config'].includes(
      String(scope || '')
    )
  }

  function needRegion(scope?: string) {
    return ['goods_type_region', 'plan_group'].includes(String(scope || ''))
  }

  function needPlanGroup(scope?: string) {
    return ['plan_group', 'package', 'addon_config'].includes(String(scope || ''))
  }

  function needPackage(scope?: string) {
    return String(scope || '') === 'package'
  }

  function getRegionOptions(rule: CouponRuleFormValue): SelectOption[] {
    const goodsTypeId = Number(rule.goods_type_id || 0)
    return props.regions
      .filter((item) => item.id !== null)
      .filter((item) => !goodsTypeId || Number(item.goods_type_id) === goodsTypeId)
      .map((item) => ({
        label: item.name || `#${item.id}`,
        value: Number(item.id)
      }))
  }

  function getPlanGroupOptions(rule: CouponRuleFormValue): SelectOption[] {
    const goodsTypeId = Number(rule.goods_type_id || 0)
    const regionId = Number(rule.region_id || 0)

    return props.planGroups
      .filter((item) => item.id !== null)
      .filter(
        (item) =>
          (!goodsTypeId || Number(item.goods_type_id) === goodsTypeId) &&
          (!regionId || Number(item.region_id) === regionId)
      )
      .map((item) => ({
        label: item.name || `#${item.id}`,
        value: Number(item.id)
      }))
  }

  function getPackageOptions(rule: CouponRuleFormValue): SelectOption[] {
    const planGroupId = Number(rule.plan_group_id || 0)

    return props.packages
      .filter((item) => item.id !== null)
      .filter((item) => !planGroupId || Number(item.plan_group_id) === planGroupId)
      .map((item) => ({
        label: item.name || `#${item.id}`,
        value: Number(item.id)
      }))
  }

  function addRule() {
    localForm.rules.push(createEmptyRule())
  }

  function removeRule(index: number) {
    if (localForm.rules.length <= 1) {
      return
    }

    localForm.rules.splice(index, 1)
  }

  function handleSubmit() {
    emit('submit', {
      id: localForm.id,
      name: localForm.name,
      rules: localForm.rules.map((item) => normalizeRule(item))
    })
  }
</script>

<style scoped lang="scss">
  .rule-header,
  .rule-card-header,
  .dialog-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .rule-header {
    margin-bottom: 12px;
    font-weight: 600;
  }

  .rule-card {
    margin-bottom: 12px;
    padding: 14px;
    border: 1px solid var(--el-border-color);
    border-radius: 12px;
    background: var(--el-fill-color-blank);
  }

  .rule-card-header {
    margin-bottom: 10px;
  }

  .dialog-footer {
    justify-content: flex-end;
    gap: 12px;
  }

  .full-width {
    width: 100%;
  }
</style>
