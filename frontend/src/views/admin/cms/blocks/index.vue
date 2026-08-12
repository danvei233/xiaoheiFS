<template>
  <div class="art-full-height">
    <ArtSearchBar
      v-if="canView"
      v-model="filters"
      :items="searchItems"
      :span="8"
      :show-expand="false"
      @search="handleSearch"
      @reset="handleReset"
    />

    <ElCard v-loading="loading" class="art-table-card">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">区块管理</div>
            <div class="page-subtitle">
              管理可复用页面区块、JSON 数据、自定义 HTML 和显示状态。
            </div>
          </div>

          <div class="page-actions">
            <ElButton v-if="canView" :disabled="loading" @click="fetchData()">刷新</ElButton>
            <ElButton v-if="canCreate" type="primary" @click="openCreate">新建区块</ElButton>
          </div>
        </div>
      </template>

      <ElEmpty v-if="!canView" description="你没有查看区块的权限。" />

      <template v-else>
        <ArtTableHeader
          v-model:columns="columnChecks"
          :showSearchBar="false"
          :loading="loading"
          @refresh="fetchData"
        />

        <ArtTable
          row-key="id"
          :loading="loading"
          :data="tableData"
          :columns="columns"
          :pagination="pagination"
          @pagination:size-change="handlePageSizeChange"
          @pagination:current-change="handlePageCurrentChange"
        >
          <template #visible="{ row }">
            <ElSwitch
              :model-value="row.visible"
              :disabled="!canUpdate || row.switching"
              :loading="row.switching"
              @change="handleToggleVisible(row, $event)"
            />
          </template>

          <template #payload="{ row }">
            <div class="payload-summary">
              <div>{{ getPayloadSummary(row) }}</div>
              <ElTag size="small" type="info">
                {{ row.type === 'custom_html' ? 'HTML' : 'JSON' }}
              </ElTag>
            </div>
          </template>

          <template #operation="{ row }">
            <div class="table-actions">
              <ElButton v-if="canUpdate" link type="primary" @click="openEdit(row)">编辑</ElButton>
              <ElButton v-if="canDelete" link type="danger" @click="handleDelete(row)">
                删除
              </ElButton>
            </div>
          </template>
        </ArtTable>
      </template>
    </ElCard>

    <ElDialog
      v-model="dialogVisible"
      :title="dialogMode === 'create' ? '新建区块' : '编辑区块'"
      width="1180px"
      destroy-on-close
      align-center
    >
      <div class="dialog-editor-grid">
        <ElForm ref="formRef" :model="dialogForm" :rules="rules" label-position="top">
          <ElRow :gutter="16">
            <ElCol :span="8">
              <ElFormItem label="页面" prop="page">
                <ElSelect v-model="dialogForm.page" filterable placeholder="home">
                  <ElOption
                    v-for="item in pageOptions"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </ElSelect>
              </ElFormItem>
            </ElCol>

            <ElCol :span="8">
              <ElFormItem label="类型" prop="type">
                <ElSelect v-model="dialogForm.type" filterable placeholder="hero">
                  <ElOption
                    v-for="item in dialogTypeOptions"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </ElSelect>
              </ElFormItem>
            </ElCol>

            <ElCol :span="8">
              <ElFormItem label="语言" prop="lang">
                <ElSelect v-model="dialogForm.lang" placeholder="请选择语言">
                  <ElOption
                    v-for="item in languageOptions"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </ElSelect>
              </ElFormItem>
            </ElCol>
          </ElRow>

          <ElRow :gutter="16">
            <ElCol :span="12">
              <ElFormItem label="标题" prop="title">
                <ElInput v-model="dialogForm.title" placeholder="请输入区块标题" />
              </ElFormItem>
            </ElCol>

            <ElCol :span="12">
              <ElFormItem label="副标题">
                <ElInput v-model="dialogForm.subtitle" placeholder="可选副标题" />
              </ElFormItem>
            </ElCol>
          </ElRow>

          <ElRow :gutter="16">
            <ElCol :span="12">
              <ElFormItem label="排序值">
                <ElInputNumber v-model="dialogForm.sort_order" :min="0" style="width: 100%" />
              </ElFormItem>
            </ElCol>

            <ElCol :span="12">
              <ElFormItem label="是否显示">
                <ElSwitch v-model="dialogForm.visible" />
              </ElFormItem>
            </ElCol>
          </ElRow>

          <template v-if="dialogForm.type !== 'custom_html'">
            <ElRow v-if="isStructuredType" :gutter="12">
              <ElCol v-for="field in scalarFields" :key="field.key" :xs="24" :md="field.span || 12">
                <ElFormItem :label="field.label">
                  <ElInput
                    v-model="contentModel[field.key]"
                    :placeholder="field.placeholder || '可选'"
                  />
                </ElFormItem>
              </ElCol>
            </ElRow>

            <div v-for="list in structuredLists" :key="list.key" class="structured-list">
              <div class="structured-list-header">
                <span>{{ list.label }}</span>
                <ElButton link type="primary" @click="addStructuredItem(list)">新增条目</ElButton>
              </div>
              <ElRow
                v-for="(item, index) in getStructuredItems(list.key)"
                :key="index"
                :gutter="12"
                class="structured-item"
              >
                <ElCol v-if="list.primitive" :xs="20" :md="20">
                  <ElInput
                    :model-value="String(item ?? '')"
                    placeholder="请输入内容"
                    @update:model-value="setPrimitiveListValue(list.key, index, $event)"
                  />
                </ElCol>
                <template v-else>
                  <ElCol
                    v-for="field in list.fields"
                    :key="field.key"
                    :xs="24"
                    :md="field.span || 8"
                  >
                    <ElSwitch
                      v-if="field.kind === 'boolean'"
                      :model-value="Boolean(getStructuredValue(item, field.key))"
                      :active-text="field.label"
                      @update:model-value="setStructuredValue(item, field, $event)"
                    />
                    <ElInputNumber
                      v-else-if="field.kind === 'number'"
                      :model-value="Number(getStructuredValue(item, field.key) || 0)"
                      :placeholder="field.label"
                      class="full-width"
                      @update:model-value="setStructuredValue(item, field, $event)"
                    />
                    <ElInput
                      v-else
                      :model-value="formatStructuredValue(item, field)"
                      :placeholder="field.label"
                      @update:model-value="setStructuredValue(item, field, $event)"
                    />
                  </ElCol>
                </template>
                <ElCol :xs="4" :md="4">
                  <ElButton link type="danger" @click="removeStructuredItem(list.key, index)">
                    删除
                  </ElButton>
                </ElCol>
              </ElRow>
            </div>

            <ElFormItem v-if="!isStructuredType" label="内容 JSON" prop="content_json">
              <ElInput
                v-model="dialogForm.content_json"
                type="textarea"
                :rows="12"
                placeholder='{"items":[...]}'
              />
            </ElFormItem>
          </template>

          <ElFormItem
            v-if="dialogForm.type === 'custom_html'"
            label="自定义 HTML"
            prop="custom_html"
          >
            <ElInput
              v-model="dialogForm.custom_html"
              type="textarea"
              :rows="12"
              placeholder="<section>...</section>"
            />
          </ElFormItem>
        </ElForm>

        <section class="preview-panel" aria-label="区块预览">
          <div class="preview-header">
            <span>预览</span>
            <div class="preview-controls">
              <ElSwitch v-model="previewZoomEnabled" active-text="缩放" />
              <ElSlider
                v-model="previewScalePercent"
                :min="40"
                :max="100"
                :step="5"
                :disabled="!previewZoomEnabled"
              />
            </div>
          </div>
          <div ref="previewBodyRef" class="preview-body">
            <div ref="previewViewportRef" class="preview-viewport">
              <div
                class="preview-canvas"
                :class="{ 'is-zoom': previewZoomEnabled }"
                :style="previewCanvasStyle"
              >
                <div
                  v-if="dialogForm.type === 'custom_html'"
                  class="preview-html"
                  v-html="previewContent"
                ></div>
                <component
                  v-else-if="previewComponent"
                  :is="previewComponent"
                  v-bind="previewProps"
                />
                <pre v-else class="preview-json">{{ previewContent }}</pre>
              </div>
            </div>
          </div>
        </section>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <ElButton @click="dialogVisible = false">取消</ElButton>
          <ElButton type="primary" :loading="dialogSubmitting" @click="handleSubmit">
            保存
          </ElButton>
        </div>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import type { CMSBlockRecord } from '@/services/admin'
  import FooterBlock from '@/components/business/cms-blocks/FooterBlock.vue'
  import HomeCtaBlock from '@/components/business/cms-blocks/home/HomeCtaBlock.vue'
  import HomeFeaturesBlock from '@/components/business/cms-blocks/home/HomeFeaturesBlock.vue'
  import HomeHeroBlock from '@/components/business/cms-blocks/home/HomeHeroBlock.vue'
  import HomeProductsBlock from '@/components/business/cms-blocks/home/HomeProductsBlock.vue'
  import ProductsCalculatorBlock from '@/components/business/cms-blocks/products/ProductsCalculatorBlock.vue'
  import ProductsComparisonBlock from '@/components/business/cms-blocks/products/ProductsComparisonBlock.vue'
  import ProductsCtaBlock from '@/components/business/cms-blocks/products/ProductsCtaBlock.vue'
  import ProductsHeroBlock from '@/components/business/cms-blocks/products/ProductsHeroBlock.vue'
  import ProductsPricingBlock from '@/components/business/cms-blocks/products/ProductsPricingBlock.vue'
  import HelpActionsBlock from '@/components/business/cms-blocks/help/HelpActionsBlock.vue'
  import HelpContactBlock from '@/components/business/cms-blocks/help/HelpContactBlock.vue'
  import HelpFaqBlock from '@/components/business/cms-blocks/help/HelpFaqBlock.vue'
  import HelpHeroBlock from '@/components/business/cms-blocks/help/HelpHeroBlock.vue'
  import { createCMSBlock, deleteCMSBlock, fetchCMSBlocks, updateCMSBlock } from '@/services/admin'
  import { useAuth } from '@/hooks/core/useAuth'
  import { useTable } from '@/hooks/core/useTable'
  import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'

  defineOptions({ name: 'CmsBlocksPage' })

  interface BlockRow {
    id: number | null
    page: string
    type: string
    title: string
    subtitle: string
    content_json: string
    custom_html: string
    lang: string
    visible: boolean
    sort_order: number
    created_at: string
    updated_at: string
    switching?: boolean
  }

  interface BlockDialogForm {
    id: number | null
    page: string
    type: string
    title: string
    subtitle: string
    content_json: string
    custom_html: string
    lang: string
    visible: boolean
    sort_order: number
  }

  interface FilterState {
    page?: string
    type?: string
  }

  interface BlockTableParams extends Api.Common.CommonSearchParams, FilterState {}

  interface OptionItem {
    label: string
    value: string
  }

  interface StructuredField {
    key: string
    label: string
    kind?: 'text' | 'number' | 'boolean' | 'string-list'
    placeholder?: string
    span?: number
  }

  interface StructuredListDefinition {
    key: string
    label: string
    primitive?: boolean
    fields: StructuredField[]
  }

  const languageOptions: OptionItem[] = [
    { label: '简体中文', value: 'zh-CN' },
    { label: '英文', value: 'en-US' }
  ]

  const pageOptions: OptionItem[] = [
    { label: '首页', value: 'home' },
    { label: '产品页', value: 'products' },
    { label: '文档页', value: 'docs' },
    { label: '公告页', value: 'announcements' },
    { label: '活动页', value: 'activities' },
    { label: '教程页', value: 'tutorials' },
    { label: '帮助页', value: 'help' },
    { label: '页脚', value: 'footer' }
  ]

  const sharedTypeOptions: OptionItem[] = [
    { label: '首屏横幅', value: 'hero' },
    { label: '功能特色', value: 'features' },
    { label: '行动引导', value: 'cta' },
    { label: '产品列表', value: 'products' },
    { label: '计算器', value: 'calculator' },
    { label: '价格模块', value: 'pricing' },
    { label: '对比模块', value: 'comparison' },
    { label: '页脚', value: 'footer' },
    { label: '文章列表', value: 'posts' },
    { label: '资源模块', value: 'resources' },
    { label: '自定义 HTML', value: 'custom_html' }
  ]

  const helpTypeOptions: OptionItem[] = [
    { label: '帮助页首屏', value: 'help_hero' },
    { label: '帮助页快捷入口', value: 'help_actions' },
    { label: '帮助页常见问题', value: 'help_faq' },
    { label: '帮助页联系方式', value: 'help_contact' },
    { label: '自定义 HTML', value: 'custom_html' }
  ]

  const { hasAuth } = useAuth()

  const initialized = ref(false)
  const dialogVisible = ref(false)
  const dialogSubmitting = ref(false)
  const dialogMode = ref<'create' | 'edit'>('create')

  const filters = ref<FilterState>({
    page: undefined,
    type: undefined
  })
  const dialogForm = reactive<BlockDialogForm>(createDefaultDialogForm())
  const contentModel = reactive<Record<string, any>>({})
  const formRef = ref<FormInstance>()
  const previewZoomEnabled = ref(true)
  const previewScalePercent = ref(70)
  const previewBodyRef = ref<HTMLElement | null>(null)
  const previewViewportRef = ref<HTMLElement | null>(null)
  const previewBaseScale = ref(1)
  const PREVIEW_WIDTH = 720
  const PREVIEW_HEIGHT = 405
  let previewResizeObserver: ResizeObserver | null = null

  const canView = computed(() => hasAuth('cms_block.list'))
  const canCreate = computed(() => hasAuth('cms_block.create'))
  const canUpdate = computed(() => hasAuth('cms_block.update'))
  const canDelete = computed(() => hasAuth('cms_block.delete'))

  const allTypeOptions = computed(() => {
    const map = new Map<string, OptionItem>()
    ;[...sharedTypeOptions, ...helpTypeOptions].forEach((item) => {
      map.set(item.value, item)
    })
    return Array.from(map.values())
  })

  const searchItems = computed(() => [
    {
      key: 'page',
      label: '页面',
      type: 'select',
      props: { clearable: true, filterable: true, placeholder: '全部页面', options: pageOptions }
    },
    {
      key: 'type',
      label: '类型',
      type: 'select',
      props: {
        clearable: true,
        filterable: true,
        placeholder: '全部类型',
        options: allTypeOptions.value
      }
    }
  ])

  const {
    columnChecks,
    columns,
    data: tableData,
    loading,
    pagination,
    searchParams,
    getData,
    fetchData,
    resetSearchParams,
    handleSizeChange: handlePageSizeChange,
    handleCurrentChange: handlePageCurrentChange
  } = useTable({
    core: {
      apiFn: fetchBlockTable,
      apiParams: { current: 1, size: 20, ...filters.value },
      immediate: false,
      columnsFactory: () => [
        { prop: 'id', label: 'ID', width: 80 },
        { prop: 'page', label: '页面', width: 140 },
        { prop: 'type', label: '类型', width: 160 },
        { prop: 'title', label: '标题', minWidth: 220, showOverflowTooltip: true },
        { prop: 'payload', label: '内容摘要', minWidth: 220, useSlot: true },
        { prop: 'lang', label: '语言', width: 140 },
        { prop: 'visible', label: '显示', width: 110, useSlot: true },
        {
          prop: 'updated_at',
          label: '更新时间',
          minWidth: 180,
          formatter: (row: BlockRow) => formatDateTime(row.updated_at)
        },
        { prop: 'operation', label: '操作', width: 150, fixed: 'right', useSlot: true }
      ]
    }
  })

  const dialogTypeOptions = computed(() => {
    if (dialogForm.page === 'help') return helpTypeOptions
    if (dialogForm.page === 'home') {
      return sharedTypeOptions.filter((item) =>
        ['hero', 'features', 'products', 'cta', 'custom_html'].includes(item.value)
      )
    }
    if (dialogForm.page === 'products') {
      return sharedTypeOptions.filter((item) =>
        ['hero', 'calculator', 'pricing', 'comparison', 'cta', 'custom_html'].includes(item.value)
      )
    }
    if (dialogForm.page === 'footer') {
      return sharedTypeOptions.filter((item) => ['footer', 'custom_html'].includes(item.value))
    }
    return sharedTypeOptions.filter((item) =>
      ['hero', 'posts', 'resources', 'custom_html'].includes(item.value)
    )
  })

  const previewContent = computed(() => {
    if (dialogForm.type === 'custom_html') {
      return dialogForm.custom_html || '<p>暂无内容</p>'
    }

    const raw = dialogForm.content_json.trim()
    if (!raw) {
      return '{}'
    }
    try {
      return JSON.stringify(JSON.parse(raw), null, 2)
    } catch {
      return raw
    }
  })

  const isStructuredType = computed(() =>
    [
      'hero',
      'features',
      'products',
      'cta',
      'calculator',
      'pricing',
      'comparison',
      'footer',
      'help_hero',
      'help_actions',
      'help_faq',
      'help_contact'
    ].includes(dialogForm.type)
  )

  const scalarFields = computed<StructuredField[]>(() => {
    if (dialogForm.type === 'hero' && dialogForm.page === 'home') {
      return [
        { key: 'badge', label: '徽标' },
        { key: 'title1', label: '主标题' },
        { key: 'subtitle', label: '副标题' },
        { key: 'primary_button_text', label: '主按钮文字' },
        { key: 'primary_button_link', label: '主按钮链接' },
        { key: 'secondary_button_text', label: '次按钮文字' },
        { key: 'secondary_button_link', label: '次按钮链接' }
      ]
    }
    if (dialogForm.type === 'hero') {
      return [
        { key: 'badge', label: '徽标' },
        { key: 'title', label: '标题' },
        { key: 'subtitle', label: '副标题' }
      ]
    }
    if (dialogForm.type === 'features') {
      return [
        { key: 'badge', label: '徽标' },
        { key: 'title', label: '标题' },
        { key: 'desc', label: '描述' }
      ]
    }
    if (dialogForm.type === 'products') {
      return [
        { key: 'badge', label: '徽标' },
        { key: 'title', label: '标题' }
      ]
    }
    if (dialogForm.type === 'cta' && dialogForm.page === 'products') {
      return [
        { key: 'title', label: '标题' },
        { key: 'desc', label: '描述' },
        { key: 'contact_text', label: '联系按钮文字' },
        { key: 'contact_link', label: '联系链接' },
        { key: 'email', label: '联系邮箱' }
      ]
    }
    if (dialogForm.type === 'cta') {
      return [
        { key: 'title', label: '标题' },
        { key: 'desc', label: '描述' },
        { key: 'button_text', label: '按钮文字' },
        { key: 'button_link', label: '按钮链接' }
      ]
    }
    if (dialogForm.type === 'calculator') {
      return [
        { key: 'title', label: '标题' },
        { key: 'desc', label: '描述' }
      ]
    }
    if (dialogForm.type === 'comparison') return [{ key: 'title', label: '标题' }]
    if (dialogForm.type === 'pricing') return []
    if (dialogForm.type === 'help_hero') {
      return [
        { key: 'badge', label: '徽标' },
        { key: 'title_main', label: '主标题' },
        { key: 'title_gradient', label: '强调标题' },
        { key: 'subtitle', label: '副标题' },
        { key: 'search_placeholder', label: '搜索提示' }
      ]
    }
    if (dialogForm.type === 'help_faq') {
      return [
        { key: 'title', label: '标题' },
        { key: 'subtitle', label: '副标题' }
      ]
    }
    if (dialogForm.type === 'help_contact') {
      return [
        { key: 'title', label: '标题' },
        { key: 'description', label: '描述' },
        { key: 'cta_title', label: '行动标题' },
        { key: 'cta_desc', label: '行动描述' },
        { key: 'cta_button_text', label: '行动按钮文字' },
        { key: 'cta_url', label: '行动链接' }
      ]
    }
    if (dialogForm.type === 'footer') {
      return [
        { key: 'description', label: '站点描述' },
        { key: 'copyright', label: '版权信息' }
      ]
    }
    return []
  })

  const structuredLists = computed<StructuredListDefinition[]>(() => {
    const labels: Record<string, string> = {
      icon: '图标',
      tag: '标签',
      title: '标题',
      description: '描述',
      price: '价格',
      key: '标识',
      url: '链接',
      guest_url: '游客链接',
      value: '数值',
      suffix: '后缀',
      label: '标签',
      category: '分类',
      question: '问题',
      answer: '答案',
      subtitle: '副标题'
    }
    const textFields = (fields: string[]): StructuredField[] =>
      fields.map((key) => ({ key, label: labels[key] || key }))
    if (dialogForm.type === 'hero' && dialogForm.page === 'home') {
      return [
        { key: 'typewriter_words', label: '轮播文字', primitive: true, fields: [] },
        {
          key: 'cards',
          label: '浮动卡片',
          fields: [
            { key: 'title', label: '标题' },
            { key: 'desc', label: '描述' }
          ]
        },
        {
          key: 'stats',
          label: '统计数据',
          fields: [
            { key: 'value', label: '数值' },
            { key: 'suffix', label: '后缀' },
            { key: 'label', label: '标签' }
          ]
        }
      ]
    }
    if (dialogForm.type === 'features') {
      return [
        { key: 'items', label: '功能条目', fields: textFields(['icon', 'title', 'description']) }
      ]
    }
    if (dialogForm.type === 'products' && dialogForm.page === 'home') {
      return [
        {
          key: 'items',
          label: '产品条目',
          fields: textFields(['icon', 'tag', 'title', 'description', 'price'])
        }
      ]
    }
    if (dialogForm.type === 'cta' && dialogForm.page === 'home') {
      return [{ key: 'features', label: '卖点条目', primitive: true, fields: [] }]
    }
    if (dialogForm.type === 'hero' && dialogForm.page === 'products') {
      return [{ key: 'features', label: '卖点条目', primitive: true, fields: [] }]
    }
    if (dialogForm.type === 'calculator') {
      return [
        {
          key: 'scenarios',
          label: '使用场景',
          fields: [
            { key: 'icon', label: '图标' },
            { key: 'name', label: '名称' },
            { key: 'recommended', label: '推荐配置' },
            { key: 'plan', label: '套餐 ID', kind: 'number' }
          ]
        }
      ]
    }
    if (dialogForm.type === 'pricing') {
      return [
        {
          key: 'products',
          label: '价格套餐',
          fields: [
            { key: 'icon', label: '图标' },
            { key: 'name', label: '名称' },
            { key: 'description', label: '描述' },
            { key: 'price', label: '价格' },
            { key: 'recommended', label: '推荐', kind: 'boolean' },
            { key: 'cta', label: '按钮文字' }
          ]
        }
      ]
    }
    if (dialogForm.type === 'comparison') {
      return [
        {
          key: 'rows',
          label: '对比条目',
          fields: [
            { key: 'feature', label: '配置项' },
            { key: 'values', label: '各套餐值（逗号分隔）', kind: 'string-list' }
          ]
        }
      ]
    }
    if (dialogForm.type === 'help_hero') {
      return [{ key: 'quick_stats', label: '快捷统计', fields: textFields(['value', 'label']) }]
    }
    if (dialogForm.type === 'help_actions') {
      return [
        {
          key: 'cards',
          label: '快捷入口',
          fields: textFields(['key', 'title', 'description', 'url', 'guest_url'])
        }
      ]
    }
    if (dialogForm.type === 'help_faq') {
      return [
        {
          key: 'categories',
          label: '问题分类',
          fields: textFields(['key', 'label'])
        },
        {
          key: 'faqs',
          label: '常见问题',
          fields: textFields(['category', 'question', 'answer'])
        }
      ]
    }
    if (dialogForm.type === 'help_contact') {
      return [
        {
          key: 'channels',
          label: '联系方式',
          fields: textFields(['key', 'title', 'subtitle'])
        }
      ]
    }
    if (dialogForm.type === 'footer') {
      return [
        {
          key: 'social_links',
          label: '社交链接',
          fields: textFields(['key', 'url'])
        },
        { key: 'sections', label: '页脚栏目', fields: textFields(['title']) }
      ]
    }
    return []
  })

  const previewComponent = computed(() => {
    const map: Record<string, unknown> = {
      hero: dialogForm.page === 'home' ? HomeHeroBlock : ProductsHeroBlock,
      features: HomeFeaturesBlock,
      products: HomeProductsBlock,
      calculator: ProductsCalculatorBlock,
      pricing: ProductsPricingBlock,
      comparison: ProductsComparisonBlock,
      cta: dialogForm.page === 'products' ? ProductsCtaBlock : HomeCtaBlock,
      footer: FooterBlock,
      help_hero: HelpHeroBlock,
      help_actions: HelpActionsBlock,
      help_faq: HelpFaqBlock,
      help_contact: HelpContactBlock
    }
    return map[dialogForm.type] || null
  })

  const previewProps = computed(() => {
    const content = contentModel
    if (dialogForm.type === 'hero' && dialogForm.page === 'home') {
      const stats = Array.isArray(content.stats) ? content.stats : []
      return {
        heroContent: content,
        typewriterText: String(content.typewriter_words?.[0] || ''),
        stats,
        animatedStats: stats.map((item: Record<string, unknown>) => String(item?.value || '')),
        heroCards: Array.isArray(content.cards) ? content.cards : [],
        handleTilt: () => undefined,
        resetTilt: () => undefined
      }
    }
    if (dialogForm.type === 'hero') return { content }
    if (dialogForm.type === 'features') {
      return {
        content,
        features: Array.isArray(content.items) ? content.items : [],
        handleFeatureGlow: () => undefined,
        resetFeatureGlow: () => undefined
      }
    }
    if (dialogForm.type === 'products') {
      return { content, products: Array.isArray(content.items) ? content.items : [] }
    }
    if (dialogForm.type === 'calculator') {
      return {
        content,
        scenarios: Array.isArray(content.scenarios) ? content.scenarios : [],
        selectedScenario: null,
        onSelect: () => undefined
      }
    }
    if (dialogForm.type === 'pricing') {
      return {
        products: Array.isArray(content.products) ? content.products : [],
        selectedPlan: -1,
        onSelect: () => undefined,
        onHover: () => undefined
      }
    }
    if (dialogForm.type === 'comparison') {
      const rows = Array.isArray(content.rows) ? content.rows : []
      const productCount = Math.max(
        1,
        ...rows.map((row: Record<string, unknown>) =>
          Array.isArray(row?.values) ? row.values.length : 0
        )
      )
      return {
        content,
        products: Array.from({ length: productCount }, (_, index) => ({
          name: `套餐 ${index + 1}`
        })),
        rows
      }
    }
    if (dialogForm.type === 'cta' && dialogForm.page !== 'products') {
      return { content, features: Array.isArray(content.features) ? content.features : [] }
    }
    if (dialogForm.type === 'help_actions') return { content, isAuthenticated: false }
    if (dialogForm.type === 'help_hero') return { content, searchQuery: '' }
    if (dialogForm.type === 'help_faq') return { content, searchQuery: '' }
    if (dialogForm.type === 'footer') {
      return {
        siteName: '站点',
        content,
        sections: Array.isArray(content.sections) ? content.sections : [],
        badges: [],
        copyrightText: String(content.copyright || ''),
        beianInfoList: Array.isArray(content.beian_info_list) ? content.beian_info_list : []
      }
    }
    return { content }
  })

  const previewCanvasStyle = computed(() => {
    if (!previewZoomEnabled.value) {
      return {}
    }
    return { zoom: previewBaseScale.value * (previewScalePercent.value / 100) }
  })

  const rules = computed<FormRules>(() => ({
    page: [{ required: true, message: '请选择页面', trigger: 'change' }],
    type: [{ required: true, message: '请选择区块类型', trigger: 'change' }],
    lang: [{ required: true, message: '请选择语言', trigger: 'change' }],
    title: [{ required: true, message: '请输入区块标题', trigger: 'blur' }]
  }))

  watch(
    canView,
    (value) => {
      if (value && !initialized.value) {
        initialized.value = true
        fetchData()
      }
    },
    { immediate: true }
  )

  watch(dialogVisible, async (visible) => {
    if (!visible) {
      previewResizeObserver?.disconnect()
      return
    }

    await nextTick()
    updatePreviewScale()
    previewResizeObserver ||= new ResizeObserver(updatePreviewScale)
    if (previewBodyRef.value) {
      previewResizeObserver.observe(previewBodyRef.value)
    }
  })

  onBeforeUnmount(() => {
    previewResizeObserver?.disconnect()
    previewResizeObserver = null
  })

  watch(
    () => dialogForm.page,
    (page) => {
      if (!page) {
        return
      }

      if (!dialogTypeOptions.value.some((item) => item.value === dialogForm.type)) {
        dialogForm.type = dialogTypeOptions.value[0]?.value || ''
      }
      ensureStructuredLists()
    }
  )

  watch(
    () => dialogForm.type,
    () => ensureStructuredLists()
  )

  function createDefaultDialogForm(): BlockDialogForm {
    return {
      id: null,
      page: 'home',
      type: 'hero',
      title: '',
      subtitle: '',
      content_json: '',
      custom_html: '',
      lang: 'zh-CN',
      visible: true,
      sort_order: 0
    }
  }

  function updatePreviewScale() {
    const body = previewBodyRef.value
    if (!body) {
      return
    }

    const width = body.clientWidth
    const height = body.clientHeight
    if (!width || !height) {
      return
    }

    const scale = Math.min(width / PREVIEW_WIDTH, height / PREVIEW_HEIGHT)
    previewBaseScale.value = Math.max(0.1, scale)
    if (previewViewportRef.value) {
      previewViewportRef.value.style.width = `${Math.floor(PREVIEW_WIDTH * scale)}px`
      previewViewportRef.value.style.height = `${Math.floor(PREVIEW_HEIGHT * scale)}px`
    }
  }

  function resetDialogForm() {
    Object.assign(dialogForm, createDefaultDialogForm())
    Object.keys(contentModel).forEach((key) => delete contentModel[key])
    ensureStructuredLists()
  }

  function loadContentModel(value: string) {
    Object.keys(contentModel).forEach((key) => delete contentModel[key])
    try {
      const parsed = JSON.parse(value || '{}')
      if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
        Object.assign(contentModel, parsed)
      }
    } catch {
      Object.keys(contentModel).forEach((key) => delete contentModel[key])
    }
    ensureStructuredLists()
  }

  function ensureStructuredLists() {
    if (!isStructuredType.value) return
    structuredLists.value.forEach((list) => {
      if (!Array.isArray(contentModel[list.key])) contentModel[list.key] = []
    })
  }

  function getStructuredItems(key: string): unknown[] {
    return Array.isArray(contentModel[key]) ? contentModel[key] : []
  }

  function addStructuredItem(list: StructuredListDefinition) {
    if (!Array.isArray(contentModel[list.key])) contentModel[list.key] = []
    if (list.primitive) {
      contentModel[list.key].push('')
      return
    }
    const item: Record<string, unknown> = {}
    list.fields.forEach((field) => {
      if (field.kind === 'boolean') item[field.key] = false
      else if (field.kind === 'number') item[field.key] = 0
      else if (field.kind === 'string-list') item[field.key] = []
      else item[field.key] = ''
    })
    contentModel[list.key].push(item)
  }

  function removeStructuredItem(key: string, index: number) {
    if (Array.isArray(contentModel[key])) contentModel[key].splice(index, 1)
  }

  function setPrimitiveListValue(key: string, index: number, value: string) {
    if (Array.isArray(contentModel[key])) contentModel[key][index] = value
  }

  function getStructuredValue(item: unknown, key: string) {
    return item && typeof item === 'object' && !Array.isArray(item)
      ? (item as Record<string, unknown>)[key]
      : undefined
  }

  function formatStructuredValue(item: unknown, field: StructuredField) {
    const value = getStructuredValue(item, field.key)
    if (field.kind === 'string-list') return Array.isArray(value) ? value.join(', ') : ''
    return value == null ? '' : String(value)
  }

  function setStructuredValue(item: unknown, field: StructuredField, value: unknown) {
    if (!item || typeof item !== 'object' || Array.isArray(item)) return
    ;(item as Record<string, unknown>)[field.key] =
      field.kind === 'string-list'
        ? String(value || '')
            .split(',')
            .map((entry) => entry.trim())
            .filter(Boolean)
        : value
  }

  function normalizeNullableNumber(value: unknown): number | null {
    if (value === '' || value === null || value === undefined) {
      return null
    }

    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : null
  }

  function normalizeRow(item?: CMSBlockRecord): BlockRow {
    return {
      id: normalizeNullableNumber(item?.id),
      page: String(item?.page || ''),
      type: String(item?.type || ''),
      title: String(item?.title || ''),
      subtitle: String(item?.subtitle || ''),
      content_json: String(item?.content_json || ''),
      custom_html: String(item?.custom_html || ''),
      lang: String(item?.lang || 'zh-CN'),
      visible: Boolean(item?.visible),
      sort_order: Number(item?.sort_order || 0),
      created_at: String(item?.created_at || ''),
      updated_at: String(item?.updated_at || ''),
      switching: false
    }
  }

  function formatDateTime(value?: string | null) {
    if (!value) {
      return '-'
    }

    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }

  function getPayloadSummary(row: BlockRow) {
    const raw = row.type === 'custom_html' ? row.custom_html : row.content_json
    const trimmed = String(raw || '').trim()
    if (!trimmed) {
      return '暂无内容'
    }

    return trimmed.length > 80 ? `${trimmed.slice(0, 80)}...` : trimmed
  }

  async function fetchBlockTable(
    params: BlockTableParams,
    signal?: AbortSignal
  ): Promise<Api.Common.PaginatedResponse<BlockRow>> {
    if (!canView.value) {
      return { records: [], current: params.current, size: params.size, total: 0 }
    }

    const payload = await fetchCMSBlocks(
      {
        page: params.page || undefined
      },
      signal
    )
    const records = (payload.items || [])
      .map((item) => normalizeRow(item))
      .filter((row) => !params.type || row.type === params.type)
    const start = (params.current - 1) * params.size

    return {
      records: records.slice(start, start + params.size),
      current: params.current,
      size: params.size,
      total: records.length
    }
  }

  async function handleSearch(params: FilterState) {
    Object.assign(searchParams, params)
    await getData()
  }

  async function handleReset() {
    filters.value = { page: undefined, type: undefined }
    await resetSearchParams()
  }

  function openCreate() {
    dialogMode.value = 'create'
    resetDialogForm()
    dialogVisible.value = true
    nextTick(() => formRef.value?.clearValidate())
  }

  function openEdit(row: BlockRow) {
    dialogMode.value = 'edit'
    Object.assign(dialogForm, {
      id: row.id,
      page: row.page,
      type: row.type,
      title: row.title,
      subtitle: row.subtitle,
      content_json: row.content_json,
      custom_html: row.custom_html,
      lang: row.lang || 'zh-CN',
      visible: row.visible,
      sort_order: row.sort_order
    })
    loadContentModel(row.content_json)
    dialogVisible.value = true
    nextTick(() => formRef.value?.clearValidate())
  }

  async function handleSubmit() {
    if (!formRef.value) {
      return
    }

    const valid = await formRef.value.validate().catch(() => false)
    if (!valid) {
      return
    }

    dialogSubmitting.value = true

    try {
      const payload = {
        page: String(dialogForm.page || '').trim(),
        type: String(dialogForm.type || '').trim(),
        title: String(dialogForm.title || ''),
        subtitle: String(dialogForm.subtitle || ''),
        content_json:
          dialogForm.type === 'custom_html'
            ? ''
            : isStructuredType.value
              ? JSON.stringify(contentModel)
              : String(dialogForm.content_json || ''),
        custom_html: dialogForm.type === 'custom_html' ? String(dialogForm.custom_html || '') : '',
        lang: String(dialogForm.lang || 'zh-CN').trim() || 'zh-CN',
        visible: Boolean(dialogForm.visible),
        sort_order: Number(dialogForm.sort_order || 0)
      }

      if (dialogMode.value === 'create') {
        await createCMSBlock(payload)
        ElMessage.success('区块创建成功')
      } else if (dialogForm.id) {
        await updateCMSBlock(dialogForm.id, payload)
        ElMessage.success('区块更新成功')
      }

      dialogVisible.value = false
      await fetchData()
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.error || '操作失败')
    } finally {
      dialogSubmitting.value = false
    }
  }

  async function handleToggleVisible(row: BlockRow, checked: string | number | boolean) {
    if (!row.id) {
      return
    }

    row.switching = true

    try {
      await updateCMSBlock(row.id, { visible: Boolean(checked) })
      row.visible = Boolean(checked)
      ElMessage.success('显示状态已更新')
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.error || '更新显示状态失败')
    } finally {
      row.switching = false
    }
  }

  async function handleDelete(row: BlockRow) {
    if (!row.id) {
      return
    }

    try {
      await ElMessageBox.confirm(
        `确定要删除区块“${row.title || row.type}”吗？该操作不可恢复。`,
        '删除区块',
        {
          type: 'warning'
        }
      )

      await deleteCMSBlock(row.id)
      ElMessage.success('区块删除成功')
      await fetchData()
    } catch (error: any) {
      if (error === 'cancel' || error === 'close') {
        return
      }

      ElMessage.error(error?.response?.data?.error || '删除区块失败')
    }
  }
</script>

<style scoped lang="scss">
  .page-header {
    display: flex;
    gap: 16px;
    align-items: center;
    justify-content: space-between;
  }

  .page-title {
    font-size: 20px;
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  .page-subtitle {
    margin-top: 6px;
    font-size: 13px;
    line-height: 1.6;
    color: var(--el-text-color-secondary);
  }

  .page-actions,
  .toolbar,
  .table-actions,
  .dialog-footer,
  .payload-summary {
    display: flex;
    gap: 12px;
    align-items: center;
  }

  .toolbar {
    flex-wrap: wrap;
    margin-bottom: 12px;
  }

  .toolbar-select {
    width: 180px;
  }

  .payload-summary {
    justify-content: space-between;
    min-width: 0;
  }

  .payload-summary > div:first-child {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .table-actions {
    justify-content: flex-end;
  }

  .dialog-footer {
    justify-content: flex-end;
  }

  .dialog-editor-grid {
    display: grid;
    grid-template-columns: minmax(0, 3fr) minmax(320px, 2fr);
    gap: 16px;
  }

  .preview-panel {
    min-width: 0;
    overflow: hidden;
    background: var(--el-fill-color-lighter);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .preview-header,
  .preview-controls {
    display: flex;
    gap: 12px;
    align-items: center;
  }

  .preview-header {
    justify-content: space-between;
    padding: 10px 12px;
    font-weight: 600;
    color: var(--el-text-color-primary);
    background: var(--el-bg-color);
    border-bottom: 1px solid var(--el-border-color-light);
  }

  .preview-controls {
    width: 180px;
  }

  .preview-body {
    display: flex;
    align-items: flex-start;
    justify-content: center;
    height: 520px;
    padding: 12px;
    overflow: auto;
    background: var(--el-fill-color-light);
  }

  .preview-viewport {
    display: flex;
    align-items: flex-start;
    justify-content: center;
  }

  .preview-canvas {
    width: 100%;
    min-height: 100%;
    overflow: auto;
    color: var(--el-text-color-primary);
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .preview-canvas.is-zoom {
    width: 720px;
    height: 405px;
  }

  .preview-html,
  .preview-json {
    min-height: 100%;
    padding: 24px;
    margin: 0;
    overflow: auto;
    color: inherit;
    word-break: break-word;
    white-space: pre-wrap;
    background: inherit;
  }

  @media (width <= 768px) {
    .dialog-editor-grid {
      grid-template-columns: minmax(0, 1fr);
    }

    .page-header {
      flex-direction: column;
      align-items: flex-start;
    }

    .page-actions,
    .toolbar {
      flex-wrap: wrap;
      width: 100%;
    }

    .toolbar-select {
      width: 100%;
    }
  }
</style>
