# Art Design Pro v3.0.2 — 规范圣经 (Conventions Bible)

> 本文件由模板源码提取，是本次前端改造的**唯一风格标准**。
> 所有页面必须按此规范编写。写代码前先读完对应章节。
> 模板参考副本位置见 `docs/frontend/adp-migration-plan.md` 第 0 节。

---

# Art Design Pro v3.0.2 — Conventions Bible

Template root is `frontend/` (this repository's SPA is built directly on the template).
Path aliases (from `frontend/vite.config.ts`): `@`→`src`, `@views`→`src/views`, `@imgs`→`src/assets/images`, `@icons`→`src/assets/icons`, `@utils`→`src/utils`, `@plugins`→`src/plugins`, `@styles`→`src/assets/styles`. **注意：不定义 `@stores`**——本项目有两个 store 目录（`src/store/` 模板基建、`src/stores/` 业务），一律用 `@/store/...` 与 `@/stores/...` 显式书写。
Auto-imported globally (no import needed): all of `vue`, `vue-router`, `pinia`, `@vueuse/core`, plus Element Plus components/APIs via `ElementPlusResolver` (so `ElMessage`, `ElCard`, `ElButton`, `ref`, `computed`, `h`, `storeToRefs` are all ambient). All `src/components/**` `Art*` components are auto-registered by `unplugin-vue-components`.

---

## 1. Page skeleton — canonical list page

### 1a. The reference implementation: `TPL/src/views/system/user/index.vue`

```vue
<!-- art-full-height 自动计算出页面剩余高度 -->
<!-- art-table-card 一个符合系统样式的 class，同时自动撑满剩余高度 -->
<template>
  <div class="user-page art-full-height">
    <!-- 搜索栏 -->
    <UserSearch v-model="searchForm" @search="handleSearch" @reset="resetSearchParams"></UserSearch>

    <ElCard class="art-table-card">
      <!-- 表格头部 -->
      <ArtTableHeader v-model:columns="columnChecks" :loading="loading" @refresh="refreshData">
        <template #left>
          <ElSpace wrap>
            <ElButton @click="showDialog('add')" v-ripple>新增用户</ElButton>
          </ElSpace>
        </template>
      </ArtTableHeader>

      <!-- 表格 -->
      <ArtTable
        :loading="loading"
        :data="data"
        :columns="columns"
        :pagination="pagination"
        @selection-change="handleSelectionChange"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      >
      </ArtTable>

      <!-- 用户弹窗 -->
      <UserDialog
        v-model:visible="dialogVisible"
        :type="dialogType"
        :user-data="currentUserData"
        @submit="handleDialogSubmit"
      />
    </ElCard>
  </div>
</template>
```

Script skeleton (verbatim shape):

```ts
<script setup lang="ts">
  import ArtButtonTable from '@/components/core/forms/art-button-table/index.vue'
  import { useTable } from '@/hooks/core/useTable'
  import { fetchGetUserList } from '@/api/system-manage'
  import UserSearch from './modules/user-search.vue'
  import UserDialog from './modules/user-dialog.vue'
  import { ElTag, ElMessageBox, ElImage } from 'element-plus'
  import { DialogType } from '@/types'

  defineOptions({ name: 'User' })   // MUST match route `name` for keep-alive/worktab

  type UserListItem = Api.SystemManage.UserListItem

  const searchForm = ref({ userName: undefined, userGender: undefined, status: '1' })

  const {
    columns, columnChecks, data, loading, pagination,
    getData, searchParams, resetSearchParams,
    handleSizeChange, handleCurrentChange, refreshData
  } = useTable({ /* see §2 */ })

  const handleSearch = (params: Api.SystemManage.UserSearchParams) => {
    Object.assign(searchParams, params)   // mutate reactive searchParams
    getData()                             // getData() == getDataByPage() -> resets to page 1
  }
</script>
```

### 1b. Variant with toggleable search bar: `TPL/src/views/system/role/index.vue`

```vue
<template>
  <div class="art-full-height">
    <RoleSearch v-show="showSearchBar" v-model="searchForm" @search="handleSearch" @reset="resetSearchParams"></RoleSearch>

    <ElCard class="art-table-card" :style="{ 'margin-top': showSearchBar ? '12px' : '0' }">
      <ArtTableHeader
        v-model:columns="columnChecks"
        v-model:showSearchBar="showSearchBar"
        :loading="loading"
        @refresh="refreshData"
      >
        <template #left><ElSpace wrap><ElButton @click="showDialog('add')" v-ripple>新增角色</ElButton></ElSpace></template>
      </ArtTableHeader>
      <ArtTable :loading="loading" :data="data" :columns="columns" :pagination="pagination"
        @pagination:size-change="handleSizeChange" @pagination:current-change="handleCurrentChange" />
    </ElCard>
  </div>
</template>
```
Passing `v-model:showSearchBar` is what makes the magnifier button appear in ArtTableHeader (`v-if="showSearchBar != null"`).

### 1c. Minimal page (no header, no search): `TPL/src/views/examples/tables/basic.vue`

```vue
<div class="user-page art-full-height">
  <ElCard class="art-table-card" style="margin-top: 0">
    <ArtTable rowKey="id" :show-table-header="false" :loading="loading" :data="data"
      :columns="columns" :pagination="pagination"
      @pagination:size-change="handleSizeChange" @pagination:current-change="handleCurrentChange" />
  </ElCard>
</div>
```
**Rule:** if you omit `ArtTableHeader`, you MUST pass `:show-table-header="false"` or ArtTable subtracts a phantom 44px header from its height.

### 1d. Non-full-height content page: `TPL/src/views/examples/forms/index.vue`
```vue
<div class="pb-5">
  <ElCard class="art-card-xs"> ... </ElCard>
</div>
```

### Sub-module convention
Page folder layout is `views/<domain>/<page>/index.vue` + `views/<domain>/<page>/modules/*.vue` (e.g. `user-search.vue`, `user-dialog.vue`, `role-edit-dialog.vue`, `role-permission-dialog.vue`). Search components take `modelValue` and emit `update:modelValue | search | reset`; dialogs take `visible`/`type`/row-data and emit `update:visible | submit`.

---

## 2. `useTable` — `TPL/src/hooks/core/useTable.ts`

### Options object (full type, verbatim)

```ts
export interface UseTableConfig<TApiFn, TRecord, TParams, TResponse> {
  core: {
    /** API 请求函数 */                apiFn: TApiFn
    /** 默认请求参数 */                apiParams?: Partial<TParams>
    /** 排除 apiParams 中的属性 */     excludeParams?: string[]
    /** 是否立即加载数据 */            immediate?: boolean            // default true
    /** 列配置工厂函数 */              columnsFactory?: () => ColumnOption<TRecord>[]
    /** 自定义分页字段映射 */          paginationKey?: { current?: string; size?: string }
  }
  transform?: {
    dataTransformer?: (data: TRecord[]) => TRecord[]
    responseAdapter?: (response: TResponse) => ApiResponse<TRecord>   // default defaultResponseAdapter
  }
  performance?: {
    enableCache?: boolean      // default false
    cacheTime?: number         // default 5 * 60 * 1000
    debounceTime?: number      // default 300
    maxCacheSize?: number      // default 50
  }
  hooks?: {
    onSuccess?: (data: TRecord[], response: ApiResponse<TRecord>) => void
    onError?: (error: TableError) => void
    onCacheHit?: (data: TRecord[], response: ApiResponse<TRecord>) => void
    onLoading?: (loading: boolean) => void
    resetFormCallback?: () => void
  }
  debug?: { enableLog?: boolean; logLevel?: 'info' | 'warn' | 'error' }
}
```
Defaults are destructured as: `apiParams = {}`, `excludeParams = []`, `immediate = true`, `enableCache = false`, `cacheTime = 300000`, `debounceTime = 300`, `maxCacheSize = 50`, `enableLog = false`.

Pagination param names default to `tableConfig.paginationKey` = `{ current: 'current', size: 'size' }` (`TPL/src/utils/table/tableConfig.ts`).

### Return value (every field)

| Name | Type / note |
|---|---|
| `data` | `Ref<TRecord[]>` |
| `loading` | `readonly` computed boolean (`loadingState === 'loading'`) |
| `error` | `readonly Ref<TableError \| null>` |
| `isEmpty` | computed boolean |
| `hasData` | computed boolean |
| `pagination` | `readonly reactive<{current,size,total}>` — pass straight to `ArtTable :pagination` |
| `paginationMobile` | computed, adds `small: width < 768` |
| `handleSizeChange(newSize)` | resets to page 1, refetches |
| `handleCurrentChange(newCurrent)` | guards duplicates, refetches |
| `searchParams` | `reactive(TParams)` — **mutate with `Object.assign`** |
| `resetSearchParams()` | wipes params, restores `apiParams`, clears all cache, refetches, then runs `resetFormCallback` |
| `fetchData(params?)` | **keeps current page** (alias of internal `getData`) |
| `getData(params?)` | **resets to page 1 + bypasses cache** (alias of internal `getDataByPage`) — this is the search entry point |
| `getDataDebounced` | debounced `getData`, has `.cancel()` / `.flush()` |
| `clearData()` | empties data + clears cache |
| `refreshData()` | full refresh, clears ALL cache (wire to `ArtTableHeader @refresh`) |
| `refreshSoft()` | clears current-search cache only (timer refresh) |
| `refreshCreate()` | back to page 1 + clear pagination cache (after create) |
| `refreshUpdate()` | keep page, clear current-search cache (after edit) |
| `refreshRemove()` | keep page; if page became empty and >1, steps back one page |
| `cacheInfo` | computed `{ total, size, hitRate }` |
| `clearCache(strategy, context?)` | `CacheInvalidationStrategy.CLEAR_ALL / CLEAR_CURRENT / CLEAR_PAGINATION / KEEP_ALL` |
| `clearExpiredCache()` | returns cleaned count |
| `cancelRequest()` | aborts in-flight request + cancels debounce |
| **only if `columnsFactory` given:** `columns`, `columnChecks`, `addColumn`, `removeColumn`, `toggleColumn`, `updateColumn`, `batchUpdateColumns`, `reorderColumns`, `getColumnConfig`, `getAllColumns`, `resetColumns` | see §3 |

Re-exports: `CacheInvalidationStrategy`, types `ApiResponse`, `CacheItem`, `BaseRequestParams`, `TableError`.

### Complete real usage (from `views/system/user/index.vue`)

```ts
const {
  columns, columnChecks, data, loading, pagination,
  getData, searchParams, resetSearchParams,
  handleSizeChange, handleCurrentChange, refreshData
} = useTable({
  core: {
    apiFn: fetchGetUserList,
    apiParams: { current: 1, size: 20, ...searchForm.value },
    // paginationKey: { current: 'pageNum', size: 'pageSize' },
    columnsFactory: () => [
      { type: 'selection' },
      { type: 'index', width: 60, label: '序号' },
      {
        prop: 'userInfo', label: '用户名', width: 280,
        // visible: false,
        formatter: (row) => h('div', { class: 'user flex-c' }, [
          h(ElImage, { class: 'size-9.5 rounded-md', src: row.avatar,
                       previewSrcList: [row.avatar], previewTeleported: true }),
          h('div', { class: 'ml-2' }, [
            h('p', { class: 'user-name' }, row.userName),
            h('p', { class: 'email' }, row.userEmail)
          ])
        ])
      },
      { prop: 'userGender', label: '性别', sortable: true, formatter: (row) => row.userGender },
      { prop: 'userPhone', label: '手机号' },
      {
        prop: 'status', label: '状态',
        formatter: (row) => {
          const c = getUserStatusConfig(row.status)
          return h(ElTag, { type: c.type }, () => c.text)
        }
      },
      { prop: 'createTime', label: '创建日期', sortable: true },
      {
        prop: 'operation', label: '操作', width: 120, fixed: 'right',
        formatter: (row) => h('div', [
          h(ArtButtonTable, { type: 'edit',   onClick: () => showDialog('edit', row) }),
          h(ArtButtonTable, { type: 'delete', onClick: () => deleteUser(row) })
        ])
      }
    ]
  },
  transform: {
    dataTransformer: (records) => {
      if (!Array.isArray(records)) { console.warn(...); return [] }
      return records.map((item, index) => ({ ...item, avatar: ACCOUNT_TABLE_DATA[index % ACCOUNT_TABLE_DATA.length].avatar }))
    }
  }
})
```

Date-range pattern (role page): declare `daterange` in the search form, add `excludeParams: ['daterange']` in `core`, then split in `handleSearch`:
```ts
const handleSearch = (params: RoleSearchFormParams) => {
  const { daterange, ...filtersParams } = params
  const [startTime, endTime] = Array.isArray(daterange) ? daterange : [null, null]
  Object.assign(searchParams, { ...filtersParams, startTime, endTime })
  getData()
}
```

### Response-shape adaptation (`TPL/src/utils/table/tableUtils.ts` + `tableConfig.ts`)
`defaultResponseAdapter` auto-detects the record array from field names `['list','data','records','items','result','rows']`, total from `['total','count']`, current from `['current','page','pageNum']`, size from `['size','pageSize','limit']`, and also drills into a nested `data` object. Add your backend's field names to `tableConfig.ts` rather than writing per-page adapters.

---

## 3. `useTableColumns` — `TPL/src/hooks/core/useTableColumns.ts`

```ts
export function useTableColumns<T = any>(
  columnsFactory: () => ColumnOption<T>[]
): { columns: any; columnChecks: any } & DynamicColumnConfig<T>
```
`DynamicColumnConfig<T>`:
```ts
addColumn(column: ColumnOption<T> | ColumnOption<T>[], index?: number): void
removeColumn(prop: string | string[]): void
toggleColumn(prop: string | string[], visible?: boolean): void
updateColumn(prop: string | Array<{prop: string; updates: Partial<ColumnOption<T>>}>, updates?): void
batchUpdateColumns(updates): void   // @deprecated
reorderColumns(fromIndex: number, toIndex: number): void
getColumnConfig(prop: string): ColumnOption<T> | undefined
getAllColumns(): ColumnOption<T>[]
resetColumns(): void
```
Relationship to the UI:
- `columnChecks` is the **full** list including hidden columns, each entry carrying `checked`/`visible`. Bind it with `v-model:columns="columnChecks"` on `ArtTableHeader` — the popover checkbox list and the drag-to-reorder list mutate it in place.
- `columns` is a `computed` derived from `columnChecks.filter(getColumnVisibility)` mapped back to the real column configs. Bind it to `ArtTable :columns`.
- Visibility precedence: `visible` wins over `checked`, default `true` (`getColumnVisibility`).
- Special types get synthetic keys/labels and are always visible: `selection→__selection__`, `expand→__expand__`, `index→__index__` (`SPECIAL_COLUMNS`, labels from i18n `table.column.*`).
- `getColumnKey(col)` = special prop, else `col.prop`. Use that string in `toggleColumn`/`updateColumn`.
- Columns with `fixed` set get class `fixed-column` in the settings popover and cannot be dragged.

Standalone usage (from the file's own doc):
```ts
const { columns, columnChecks, toggleColumn, reorderColumns } = useTableColumns(() => [
  { prop: 'name', label: '姓名', visible: true },
  { prop: 'status', label: '状态', visible: false }
])
toggleColumn('email', false)
reorderColumns(0, 2)
```

---

## 4. `ArtTable` — `TPL/src/components/core/tables/art-table/index.vue`

`defineOptions({ name: 'ArtTable' })`. Props interface **extends `TableProps<Record<string, any>>` from Element Plus** — every native `el-table` prop, event and slot works, plus:

```ts
interface ArtTableProps extends TableProps<Record<string, any>> {
  loading?: boolean
  columns?: ColumnOption[]            // default () => []
  pagination?: { current: number; size: number; total: number }
  paginationOptions?: PaginationOptions
  emptyHeight?: string                // default '100%'
  emptyText?: string                  // default '暂无数据'
  showTableHeader?: boolean           // default true
}
interface PaginationOptions {
  pageSizes?: number[]                // default [10,20,30,50,100]
  align?: 'left' | 'center' | 'right' // default 'center'
  layout?: string                     // responsive default, see below
  background?: boolean                // default true
  hideOnSinglePage?: boolean          // default false
  size?: 'small' | 'default' | 'large'// default 'default'
  pagerCount?: number                 // default width>1200 ? 7 : 5
}
```
Other explicit defaults: `fit: true`, `showHeader: true`, `stripe/border/size: undefined` (fall back to `useTableStore`).

**Events**
```ts
(e: 'pagination:size-change', val: number): void
(e: 'pagination:current-change', val: number): void
```
Plus all native el-table events pass through `useAttrs()` (`@selection-change`, `@sort-change`, `@row-click`, …).

**Exposed**: `{ scrollToTop, elTableRef }` → `tableRef.value.elTableRef.toggleRowSelection(...)`.

**Slots**: default slot (`<template v-if="$slots.default" #default><slot /></template>`) for hand-written `<ElTableColumn>`; plus per-column dynamic slots — see below. The `#empty` slot is owned by ArtTable (renders `<ElEmpty :description="emptyText" :image-size="120" />`).

### Column object schema — `ColumnOption<T>` (`TPL/src/types/component/index.ts`)
```ts
export interface ColumnOption<T = any> {
  type?: 'selection' | 'expand' | 'index' | 'globalIndex'
  prop?: string
  label?: string
  width?: string | number
  minWidth?: string | number
  fixed?: boolean | 'left' | 'right'
  sortable?: boolean | 'custom'
  filters?: any[]
  filterMethod?: (value: any, row: any) => boolean
  filterPlacement?: string
  disabled?: boolean          // disables the checkbox in the column-setting popover
  visible?: boolean           // initial visibility (wins over checked)
  checked?: boolean
  formatter?: (row: T) => any // ← render function; return VNode via h()
  useSlot?: boolean
  slotName?: string           // default = prop
  useHeaderSlot?: boolean
  headerSlotName?: string     // default = `${prop}-header`
  [key: string]: any          // anything else is forwarded to ElTableColumn (showOverflowTooltip, align, …)
}
```
Notes:
- `formatter` here is **not** the el-table signature — it receives only `row` and its return value is used as the cell content (a VNode from `h()`, or a plain string).
- `type: 'globalIndex'` renders `(current - 1) * size + $index + 1` using `props.pagination`. `type: 'index'` is native el-table.
- `type: 'expand'` renders `<component :is="col.formatter(row)" />` inside the expanded row.
- Slot mode: `{ prop: 'status', label: '状态', useSlot: true }` then in the page `<ArtTable ...><template #status="{ row, value, prop, $index }">…</template></ArtTable>`. `useHeaderSlot` gives `#<prop>-header` with `{ prop, label, ...headerScope }`.
- `cleanColumnProps()` strips `useHeaderSlot/headerSlotName/useSlot/slotName` before spreading onto `ElTableColumn`; every other key is forwarded verbatim.

### Pagination wiring
```vue
<ArtTable :pagination="pagination"
  @pagination:size-change="handleSizeChange"
  @pagination:current-change="handleCurrentChange" />
```
Pagination is only rendered when `props.pagination` is truthy **and** data is non-empty. `layout` auto-switches: `<768px` → `'prev, pager, next, sizes, jumper, total'`; `<1024px` → `'prev, pager, next, jumper, total'`; else `'total, prev, pager, next, sizes, jumper'`. Changing the page auto-calls `elTableRef.setScrollTop(0)` and `useCommon().scrollToTop()`.

### Height
Height is `100%` normally, `props.emptyHeight` when empty+not loading, `100%` in fullscreen. Container height comes from `useTableHeight` (`TPL/src/hooks/core/useTableHeight.ts`): `calc(100% - offset)` where offset = tableHeaderHeight (measured from `#art-table-header`, fallback 44) + 12 spacing + paginationHeight + spacing (6 when `showTableHeader`, else 15).

### Global table appearance
`useTableStore` (`TPL/src/store/modules/table.ts`) supplies `isBorder`, `isZebra`, `tableSize`, `isFullScreen`, `isHeaderBackground`. Props override store per-instance.

---

## 5. `ArtTableHeader` — `TPL/src/components/core/tables/art-table-header/index.vue`

Root element carries `id="art-table-header"` — ArtTable measures this element by ID, so **do not render two ArtTableHeaders on one page**.

```ts
interface Props {
  showZebra?: boolean            // default true
  showBorder?: boolean           // default true
  showHeaderBackground?: boolean // default true
  fullClass?: string             // default 'art-page-view'  (element to fullscreen)
  layout?: string                // default 'search,refresh,size,fullscreen,columns,settings'
  loading?: boolean
  showSearchBar?: boolean        // default undefined -> search button hidden
}
const columns = defineModel<ColumnOption[]>('columns', { required: false, default: () => [] })
const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search'): void
  (e: 'update:showSearchBar', value: boolean): void
}>()
```

**Slots**: `#left` (left cluster, typically `<ElSpace wrap>` of action buttons) and `#right` (appended after the built-in icon buttons).

**Behaviors**
- `search` — only rendered when `showSearchBar != null`; toggles `update:showSearchBar` then emits `search`.
- `refresh` — sets an internal `isManualRefresh` flag (drives the spin animation while `loading`) and emits `refresh`. Wire to `useTable().refreshData`.
- `size` — dropdown of `TableSizeEnum.SMALL/DEFAULT/LARGE` → `useTableStore().setTableSize`.
- `fullscreen` — `document.querySelector('.' + fullClass)`, adds/removes class `el-full-screen`, locks `body.overflow`, sets `tableStore.setIsFullScreen`, ESC exits, cleans up on unmount. Default target `.art-page-view` is the class `ArtPageContent` puts on every routed page component.
- `columns` — popover with `VueDraggable` reordering (filter `.fixed-column`) + `ElCheckbox` per column, mutating `col.checked`/`col.visible` on the bound `columnChecks`.
- `settings` — zebra / border / header-background checkboxes bound to `useTableStore`.
Hide any of these by editing the `layout` string, e.g. `layout="refresh,columns"`.

---

## 6. `ArtSearchBar` — `TPL/src/components/core/forms/art-search-bar/index.vue`

### `items` schema
```ts
export interface SearchFormItem {
  key: string                                   // form field name (v-model key)
  label: string | (() => VNode) | Component     // string OR render fn / component
  labelWidth?: string | number
  type?: keyof typeof componentMap | string
  render?: (() => VNode) | Component            // custom component; HIGHER priority than `type`
  hidden?: boolean
  span?: number                                 // 24-grid
  options?: Record<string, any>
  props?: Record<string, any>                   // forwarded to the inner component
  slots?: Record<string, (() => any) | undefined>
  placeholder?: string
}
```

### Supported `type` values (`componentMap`, exact keys)
`input`(ElInput) · `inputTag`(ElInputTag) · `number`(ElInputNumber) · `select`(ElSelect) · `switch`(ElSwitch) · `checkbox`(ElCheckbox) · `checkboxgroup`(ElCheckboxGroup) · `radiogroup`(ElRadioGroup) · `date`/`daterange`/`datetime`/`datetimerange`(all ElDatePicker) · `rate`(ElRate) · `slider`(ElSlider) · `cascader`(ElCascader) · `timepicker`(ElTimePicker) · `timeselect`(ElTimeSelect) · `treeselect`(ElTreeSelect).
Unknown type falls back to `input`. For `select`/`checkboxgroup`/`radiogroup`, put the array in `props.options` — the component auto-renders `<ElOption>`/`<ElCheckbox>`/`<ElRadio>` with `v-bind="option"`.
There is **no** `daterange` distinct behaviour — you set `props: { type: 'daterange', valueFormat: 'YYYY-MM-DD', ... }` on a `datetime`/`date` item (see example below).

`props` resolution: if `item.props` exists it is used verbatim; otherwise the item object minus `['label','labelWidth','key','type','hidden','span','slots']` is spread as props (that's why `placeholder`/`clearable` can be written at the top level).

### Component props
```ts
interface SearchBarProps {
  items: SearchFormItem[]                 // default []
  span?: number                           // default 6
  gutter?: number                         // default 12
  isExpand?: boolean                      // default false — true = always show all, no toggle
  defaultExpanded?: boolean               // default false
  labelPosition?: 'left'|'right'|'top'    // default 'right'
  labelWidth?: string | number            // default '70px'
  showExpand?: boolean                    // default true
  buttonLeftLimit?: number                // default 2
  showReset?: boolean                     // default true
  showSearch?: boolean                    // default true
  disabledSearch?: boolean                // default false
  sanitizeOutput?: Partial<SanitizeOutputOptions>   // default {}
}
interface SanitizeOutputOptions {
  removeEmptyString: boolean; removeEmptyArray: boolean; removeEmptyObject: boolean
  removeEmptyRichText: boolean; keepZero: boolean; keepFalse: boolean
}   // effective defaults: all removes=true, keepZero=true, keepFalse=true
```
`$attrs` are spread onto the inner `<ElForm>`, so `:rules="rules"` / `:disabled` etc. go straight through.

### v-model, events, exposed
- `const modelValue = defineModel<Record<string, any>>({ default: {} })` → `v-model="formData"` on the object. Setting a field to `''` **deletes the key** (`normalizeFieldValue`).
- Events: `search: [Record<string, any>]` (payload is the *sanitized* clone, not the raw model) and `reset: []`.
- `reset` restores the **initial snapshot** taken at mount (defaults are preserved, not blanked) and calls `formInstance.resetFields()`.
- Exposed: `{ ref, validate(...args), reset, getOutput }`.
- Collapse rule: when collapsed it shows `Math.floor(24 / span) - 1` items; the expand toggle only appears if `!isExpand && showExpand && visibleItems > Math.floor(24/span) - 1`.
- Grid: each item is an `ElCol` with responsive spans from `calculateResponsiveSpan` (`TPL/src/utils/form/responsive.ts`): xs `<12 → 24`, sm `<12 → 12`, md `<8 → 8`, lg/xl use the raw span.
- Slots: named per item key — `<template #someKey="{ item, modelValue }">` overrides that item's control entirely.

### Full real example — `TPL/src/views/system/user/modules/user-search.vue`
```vue
<template>
  <ArtSearchBar ref="searchBarRef" v-model="formData" :items="formItems" :rules="rules"
    @reset="handleReset" @search="handleSearch" />
</template>
<script setup lang="ts">
  const props = defineProps<{ modelValue: Api.SystemManage.UserSearchParams }>()
  const emit = defineEmits<{ (e:'update:modelValue', v: any): void; (e:'search', p:any): void; (e:'reset'): void }>()
  const searchBarRef = ref()
  const formData = computed({ get: () => props.modelValue, set: (val) => emit('update:modelValue', val) })
  const rules = {}
  const statusOptions = ref<{label:string;value:string}[]>([])
  onMounted(async () => { statusOptions.value = await fetchStatusOptions() })

  const formItems = computed(() => [
    { label: '用户名', key: 'userName', type: 'input', placeholder: '请输入用户名', clearable: true },
    { label: '手机号', key: 'userPhone', type: 'input', props: { placeholder: '请输入手机号', maxlength: '11' } },
    { label: '邮箱',   key: 'userEmail', type: 'input', props: { placeholder: '请输入邮箱' } },
    { label: '状态',   key: 'status',    type: 'select', props: { placeholder: '请选择状态', options: statusOptions.value } },
    { label: '性别',   key: 'userGender',type: 'radiogroup', props: { options: [{label:'男',value:'1'},{label:'女',value:'2'}] } }
  ])

  function handleReset() { emit('reset') }
  async function handleSearch(params) { await searchBarRef.value.validate(); emit('search', params) }
</script>
```

### Richer item cookbook (from `TPL/src/views/examples/forms/search-bar.vue`)
```ts
{ label:'日期', key:'date', type:'datetime',
  props:{ style:{width:'100%'}, placeholder:'请选择日期', type:'date', valueFormat:'YYYY-MM-DD',
          shortcuts:[{text:'今日',value:new Date()},{text:'昨日',value:()=>new Date(Date.now()-86400000)}] } },
{ label:'日期范围', key:'daterange', type:'datetime',
  props:{ type:'daterange', valueFormat:'YYYY-MM-DD', rangeSeparator:'至',
          startPlaceholder:'开始日期', endPlaceholder:'结束日期' } },
{ label:'日期时间范围', key:'datetimerange', type:'datetime',
  props:{ type:'datetimerange', valueFormat:'YYYY-MM-DD HH:mm:ss' } },
{ label:'时间选择', key:'timeselect', type:'timeselect', props:{ placeholder:'请选择时间', valueFormat:'HH:mm:ss' } },
{ label:'时间选择器', key:'timepicker', type:'timepicker', props:{ style:{width:'100%'}, valueFormat:'HH:mm:ss' } },
{ label:'级联选择', key:'cascader', type:'cascader',
  props:{ placeholder:'请选择', clearable:true, style:{width:'100%'}, collapseTags:true,
          maxCollapseTags:1, props:{ multiple:true }, options:cascaderOptions } },
{ label:'树型选择器', key:'treeSelect', type:'treeselect',
  props:{ showCheckbox:true, multiple:true, clearable:true, data:treeSelectData } },
{ label:'复选框', key:'checkboxgroup', type:'checkboxgroup', span:12, props:{ options:checkboxOptions } },
{ label:'插槽', key:'slots', type:'input' },              // then <template #slots><ElInput .../></template>
{ label:'渲染组件', key:'iconSelector', render: () => h(ElInput, { placeholder:'渲染自定义 input' }) },
{ label:'自定义组件', key:'customComponent',
  render: () => h('div', { style:'color: var(--art-gray-600); border:1px solid var(--default-border-dashed); padding:0 15px; border-radius:6px' }, '我是一个自定义组件') }
```

---

## 7. `ArtForm` — `TPL/src/components/core/forms/art-form/index.vue`

Same engine as ArtSearchBar; differences:

- `FormItem` schema is **identical** to `SearchFormItem` (exported as `FormItem`).
- `componentMap` keys differ in one place: it's `inputtag` (lowercase) here vs `inputTag` in the search bar. Everything else is the same list.
- Props: `items`, `span` (6), `gutter` (12), `labelPosition` ('right'), `labelWidth` ('70px'), `buttonLeftLimit` (2), `showReset` (true), `showSubmit` (true), `disabledSubmit` (false), `sanitizeOutput`. **No** `isExpand`/`showExpand`/`defaultExpanded` — all non-hidden items always render.
- Events: `reset: []`, `submit: [Record<string, any>]` (sanitized payload).
- Exposed: `{ ref, validate, reset, getOutput }`.
- **Nested paths supported**: `key` may be `'a.b'` or `'a.0.b'` — `parsePath`/`setFieldValue` auto-create intermediate objects/arrays (numeric segments create arrays). ArtSearchBar does *not* do this (flat keys only).
- Validation: pass `:rules` — it lands on the inner `ElForm` via `$attrs`; `ElFormItem :prop="item.key"`.
- Wrapper is `<section class="px-4 pb-0 pt-4 md:px-4 md:pt-4">` (no card border of its own — wrap it in `<ElCard class="art-card-xs">`).

Real example (`TPL/src/views/examples/forms/index.vue`):
```vue
<ElCard class="art-card-xs">
  <ArtForm ref="formRef" v-model="formData" :items="formItems" :rules="formRules"
    :labelWidth="labelWidth" :labelPosition="labelPosition" :span="span" :gutter="gutter"
    @reset="handleReset" @submit="handleSubmit">
    <template #slots><ElInput v-model="formData.slots" placeholder="我是插槽渲染出来的组件" /></template>
  </ArtForm>
</ElCard>
```
```ts
const formRules = {
  name:  [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  phone: [{ required: true, message: '请输入手机号', trigger: 'blur' },
          { min: 11, max: 11, message: '请输入11位手机号', trigger: 'blur' },
          { pattern: /^1[3456789]\d{9}$/, message: '请输入正确的手机号', trigger: 'blur' }],
  level: [{ required: true, message: '请选择等级', trigger: 'change' }]
}
```
For classic dialog forms the template does NOT use ArtForm — see `TPL/src/views/system/user/modules/user-dialog.vue`, which uses plain `<ElDialog><ElForm :model :rules label-width="80px">` with `FormInstance`/`FormRules` from element-plus, `watch(() => [props.visible, ...])` to seed data and `formRef.value?.clearValidate()`.

---

## 8. Component catalogue (`TPL/src/components/core/...`)

### Buttons / actions — `forms/`, `widget/`, `base/`
| Component | Path | Props | Purpose |
|---|---|---|---|
| `ArtButtonTable` | `forms/art-button-table/index.vue` | `type?: 'add'\|'edit'\|'delete'\|'more'\|'view'`, `icon?`, `iconClass?`, `iconColor?`, `buttonBgColor?`; emits `click` | Square icon action button for table operation columns; presets: add=`ri:add-fill`/`bg-theme/12 text-theme`, edit=`ri:pencil-line`/secondary, delete=`ri:delete-bin-5-line`/error, view=`ri:eye-line`/info |
| `ArtButtonMore` | `forms/art-button-more/index.vue` | `list: ButtonMoreItem[]`, `auth?: string`; emits `click(item)`. `ButtonMoreItem = { key, label, disabled?, auth?, icon?, color?, iconColor? }` | "…" dropdown of row actions; per-item `auth` filtered through `useAuth().hasAuth`; whole dropdown hides if no item is permitted |
| `ArtIconButton` | `widget/art-icon-button/index.vue` | `icon: string`, `circle?: boolean` | 34px hover-highlight icon button (used in header bar) |
| `ArtSvgIcon` | `base/art-svg-icon/index.vue` | `icon?: string` (Iconify name, e.g. `'ri:refresh-line'`), `inheritAttrs:false`, forwards `class`/`style` | The icon primitive everywhere |
| `ArtLogo` | `base/art-logo/index.vue` | `size?: number\|string` (default 36) | System logo image (`@imgs/common/logo.webp`) |
| `ArtBackToTop` | `base/art-back-to-top/index.vue` | none | Floating back-to-top |

### Text effects — `text-effect/`
| `ArtCountTo` | `text-effect/art-count-to/index.vue` | `target: number`, `duration?`(2000), `autoStart?`(true), `decimals?`(0), `decimal?`('.'), `separator?`(''), `prefix?`(''), `suffix?`(''), `easing?`(`'easeOutExpo'`, key of VueUse `TransitionPresets`), `disabled?`(false); emits `started/finished/paused/reset`; exposes `start/pause/reset/stop/setTarget/isRunning/isPaused/currentValue/targetValue/progress` | Animated number rollup |
| `ArtTextScroll` | `text-effect/art-text-scroll/index.vue` | `TextScrollProps`: text, theme, direction, speed, width, height, pauseOnHover, showClose, alwaysScroll | Marquee |
| `ArtFestivalTextScroll` | `text-effect/art-festival-text-scroll/index.vue` | — | Holiday banner (rendered by `ArtPageContent` automatically) |

### Cards — `cards/`
| Component | Key props |
|---|---|
| `ArtStatsCard` | `boxStyle?`, `icon?`, `iconStyle?`, `title?`, `count?`, `decimals?`(0), `separator?`(','), `description` (required), `textColor?`, `showArrow?` |
| `ArtProgressCard` | `percentage`, `title`, `color?`('#67C23A'), `icon?`, `iconStyle?`, `strokeWidth?`(5) |
| `ArtLineChartCard` | `value`, `label`, `percentage`, `date?`, `height?`(11rem), `color?`, `showAreaColor?`, `chartData: number[]`, `isMiniChart?` |
| `ArtBarChartCard` | `value`, `label`, `percentage`, `date?`, `height?`(11), `color?`, `chartData: number[]`, `barWidth?`('26%'), `isMiniChart?` |
| `ArtDonutChartCard` | `value`, `title`, `percentage`, `percentageLabel?`, `currentValue?`, `previousValue?`, `height?`(9), `color?`, `radius?`(`['70%','90%']`), `data: [number, number]` |
| `ArtDataListCard` | `list: Activity[]` (`{title,status,time,class,icon}`), `title`, `subtitle?`, `maxCount?`(5), `showMoreButton?`; emits `more` |
| `ArtTimelineListCard` | `list: TimelineItem[]` (`{time,status,content,code?}`), `title`, `subtitle?`, `maxCount?`(5) |
| `ArtImageCard` | `imageUrl`, `title`, `category?`, `readTime?`, `views?`, `comments?`, `date?`; emits `click(card)` |

### Banners — `banners/`
`ArtBasicBanner` (`height?`, `title?`, `subtitle?`, `boxStyle?`, `decoration?`, `buttonConfig?{show,text,color,textColor,radius}`, `meteorConfig?{enabled,count}`, `imageConfig?{src,width,bottom,right}`, `titleColor?`, `subtitleColor?`) and `ArtCardBanner` (`height?`('24rem'), `image?`, `title`, `description`, `button?{show,text,color,textColor}`, `cancelButton?{...}`; emits `click`/`cancel`).

### Charts — `charts/` (all typed in `TPL/src/types/component/chart.ts`)
All extend `BaseChartProps { height?: string; loading?: boolean; isEmpty?: boolean; colors?: string[] }`, most also `AxisDisplayProps { showAxisLabel?, showAxisLine?, showSplitLine? }` and `InteractionProps { showTooltip?, showLegend?, legendPosition?: 'bottom'|'top'|'left'|'right' }`.

| Component | Props type | Distinct fields |
|---|---|---|
| `ArtLineChart` | `LineChartProps` | `data: number[] \| LineDataItem[]`, `xAxisData?`, `lineWidth?`, `showAreaColor?`, `smooth?`, `symbol?`, `symbolSize?`, `animationDelay?` |
| `ArtBarChart` / `ArtHBarChart` | `BarChartProps` | `data: number[] \| BarDataItem[]`, `xAxisData?`, `barWidth?`, `stack?`, `borderRadius?` |
| `ArtRingChart` | `RingChartProps` | `data: PieDataItem[]`, `radius?`, `borderRadius?`, `centerText?`, `showLabel?` |
| `ArtRadarChart` | `RadarChartProps` | `indicator?: {name,max}[]`, `data?: RadarDataItem[]` |
| `ArtScatterChart` | `ScatterChartProps` | `data?: ScatterDataItem[]`, `symbolSize?` |
| `ArtKLineChart` | `KLineChartProps` | `data?: KLineDataItem[]`, `showDataZoom?`, `dataZoomStart?`, `dataZoomEnd?` |
| `ArtDualBarCompareChart` | `BidirectionalBarChartProps` | `positiveData`, `negativeData`, `xAxisData?`, `positiveName?`, `negativeName?`, `barWidth?`, `yAxisMin/Max?`, `showDataLabel?`, `positive/negativeBorderRadius?` |
| `ArtMapChart` | `MapChartProps` | `mapData?`, `selectedRegion?`, `showLabels?`, `showScatter?` (deleted by `clean:dev`) |

**Theme-aware colors — `useChart` / `useChartOps` (`TPL/src/hooks/core/useChart.ts`)**
```ts
export const useChartOps = (): ChartThemeConfig => ({
  chartHeight: '16rem',
  fontSize: 13,
  fontColor: '#999',
  themeColor: getCssVar('--el-color-primary-light-1'),
  colors: [ getCssVar('--el-color-primary-light-1'), '#4ABEFF', '#EDF2FF', '#14DEBA', '#FFAF20', '#FA8A6C', '#FFAF20' ]
})
```
`useChart(options?: UseChartOptions { initOptions?, initDelay? = 0, threshold? = 0.1, autoTheme? = true })` returns:
`{ isDark, chartRef, initChart(options, isEmpty?), updateChart, handleResize, destroyChart, getChartInstance, isChartInitialized, emptyStateManager, getAxisLineStyle(show?), getSplitLineStyle(show?), getAxisLabelStyle(show?), getAxisTickStyle(), getAnimationConfig(delay?, duration?), getTooltipStyle(trigger?, custom?), getLegendStyle(position?, custom?), useChartOps, getGridWithLegend(showLegend, position?, baseGrid?) }`.
Theme-aware values: axis/split lines `#444` dark / `#EDEDED` light; tooltip bg `rgba(0,0,0,.8)` / `rgba(255,255,255,.9)`, border `#333`/`#ddd`, text `#fff`/`#333`; legend text `#fff`/`#333`. It auto-re-renders on `isDark` change, resizes on window resize + `menuOpen`/`menuType` changes, lazily inits via `IntersectionObserver` when off-screen, and injects an "暂无数据" overlay when empty.

Preferred authoring pattern is `useChartComponent`:
```ts
const { chartRef } = useChartComponent({
  props: { height: `${props.height}rem`, loading: false, isEmpty: !props.chartData?.length },
  checkEmpty: () => !props.chartData?.length || props.chartData.every(v => v === 0),
  watchSources: [() => props.chartData, () => props.color],
  generateOptions: (): EChartsOption => ({ /* ... */ itemStyle: { color: props.color || useChartOps().themeColor } })
})
```
ECharts is tree-shaken in `TPL/src/plugins/echarts.ts` (Bar/Line/Pie/Scatter/Radar/Map/Candlestick + Title/Tooltip/Grid/Legend/DataZoom/MarkPoint/MarkLine/Toolbox/Brush/Geo/VisualMap + CanvasRenderer). Register anything else there before using it.

### Data / media / misc
| Component | Path | Props (selection) | Purpose |
|---|---|---|---|
| `ArtExcelExport` | `forms/art-excel-export/index.vue` | `data: ExportData[]`, `filename?`(`export_YYYY-MM-DD`), `sheetName?`('Sheet1'), `type?`('primary'), `size?`('default'), `disabled?`, `buttonText?`('导出 Excel'), `loadingText?`, `autoIndex?`(false), `indexColumnTitle?`('序号'), `columns?: Record<string,{title,width?,formatter?}>`, `headers?: Record<string,string>`, `maxRows?`(100000), `showSuccessMessage?`(true), `showErrorMessage?`(true), `workbookOptions?{creator,lastModifiedBy,created,modified}`; emits `before-export`, `export-success(filename,rowCount)`, `export-error`, `export-progress` | One-click XLSX export button (xlsx + file-saver) |
| `ArtExcelImport` | `forms/art-excel-import/index.vue` | see file; emits import events | XLSX import |
| `ArtWangEditor` | `forms/art-wang-editor/index.vue` | `v-model` **required** (`defineModel<string>({required:true})`), `height?`('500px'), `toolbarKeys?`, `insertKeys?{index,keys}`, `excludeKeys?`(`['fontFamily']`), `mode?`('default'\|'simple'), `placeholder?`('请输入内容...'), `uploadConfig?{maxFileSize,maxNumberOfFiles,server,isCustomUpload}`; exposes `getEditor/setContent/getContent/clear/focus` | wangEditor 5 rich text; default upload URL `${VITE_API_URL}/api/common/upload/wangeditor` |
| `ArtCutterImg` | `media/art-cutter-img/index.vue` | `isModal?`(false), `tool?`(true), `toolBgc?`('#fff'), `title?`, `previewTitle?`, `showPreview?`(true), `boxWidth?`(700), `boxHeight?`(458), `cutWidth?`(470), `cutHeight?`(270), `sizeChange?`(true), `moveAble?`/`imgMove?`/`scaleAble?`(true), `originalGraph?`(true), `crossOrigin?`(true), `fileType?`('png'), `quality?`(0.9), `watermarkText?`, `watermarkFontSize?`(20), `watermarkColor?`('#ffffff'), `saveCutPosition?`(true), `previewMode?`(true), `imgUrl?`; emits `update:imgUrl`, `error`, `imageLoadComplete`, `imageLoadError` | Image cropper (`vue-img-cutter`) |
| `ArtWatermark` | `others/art-watermark/index.vue` | `content?`(=`AppConfig.systemInfo.name`), `visible?`(false), `fontSize?`(16), `fontColor?`('rgba(128,128,128,0.2)'), `rotate?`(-22), `gapX/gapY?`(100), `offsetX/offsetY?`(50), `zIndex?`(3100) | Full-screen watermark; **already mounted globally** and gated on `settingStore.watermarkVisible` |
| `ArtDragVerify` | `forms/art-drag-verify/index.vue` | `value: boolean` (use `v-model:value`), `width?`('100%'), `height?`(40), `text?`('按住滑块拖动'), `successText?`('success'), `background?`('#eee'), `progressBarBg?`('#1385FF'), `completedBg?`('#57D187'), `circle?`(false), `radius?`(`calc(var(--custom-radius)/3 + 2px)`), `handlerIcon?`('solar:double-alt-arrow-right-linear'), `successIcon?`('ri:check-fill'), `handlerBg?`('#fff'), `textSize?`('13px'), `textColor?`('#333'); emits `handlerMove`, `update:value`, `passCallback`; exposes `reset()` | Slide-to-verify |
| `ArtVideoPlayer` | `media/art-video-player/index.vue` | `id`, `url`, `poster?`, `autoplay?`, `volume?`, `playbackRate?`, `loop?`, `muted?` | xgplayer wrapper |
| `ArtMenuRight` | `others/art-menu-right/index.vue` | `menuItems`, `width?`, `submenuWidth?`, `itemHeight?`, `borderRadius?`, `duration?` … ; exposes `show/hide` | Right-click context menu |
| `ThemeSvg` | `theme/theme-svg/index.vue` | `src`, `size` | Theme-adaptive inline SVG |
| `ArtException`, `ArtResultPage`, `LoginLeftView`, `AuthTopBar` | `views/exception|result|login/` | — | Full-page scaffolds for 403/404/500, result pages, auth pages |

Layout components (`layouts/`) are wired by the shell and normally not used directly: `ArtHeaderBar`, `ArtSidebarMenu`/`ArtHorizontalMenu`/`ArtMixedMenu`, `ArtBreadcrumb`, `ArtWorkTab`, `ArtPageContent`, `ArtGlobalComponent`, `ArtSettingsPanel`, `ArtGlobalSearch`, `ArtScreenLock`, `ArtChatWindow`, `ArtNotification`, `ArtFastEnter`, `ArtFireworksEffect`.

---

## 9. Theme system

### Every `--art-*` (and sibling) custom property — `TPL/src/assets/styles/core/tailwind.css`

`:root` (light):
```
--art-color: #ffffff
--theme-color: var(--main-color)          /* --main-color: var(--el-color-primary) from el-ui.scss */
--art-primary:   oklch(0.7  0.23 260)
--art-secondary: oklch(0.72 0.19 231.6)
--art-error:     oklch(0.73 0.15 25.3)
--art-info:      oklch(0.58 0.03 254.1)
--art-success:   oklch(0.78 0.17 166.1)
--art-warning:   oklch(0.78 0.14 75.5)
--art-danger:    oklch(0.68 0.22 25.3)
--art-gray-100: #f9fafb   --art-gray-200: #f2f4f5   --art-gray-300: #e6eaeb
--art-gray-400: #dbdfe1   --art-gray-500: #949eb7   --art-gray-600: #7987a1
--art-gray-700: #4d5875   --art-gray-800: #383853   --art-gray-900: #323251
--art-card-border: rgba(0,0,0,0.08)
--default-border: #e2e8ee
--default-border-dashed: #dbdfe9
--default-bg-color: #fafbfc
--default-box-color: #ffffff
--art-hover-color: #edeff0
--art-active-color: #f2f4f5
--art-el-active-color: #f2f4f5
```
`.dark` (overrides only these — the semantic `--art-primary/secondary/...` are NOT overridden):
```
--art-color: #000000
--art-gray-100: #110f0f   --art-gray-200: #17171c   --art-gray-300: #393946
--art-gray-400: #505062   --art-gray-500: #73738c   --art-gray-600: #8f8fa3
--art-gray-700: #ababba   --art-gray-800: #c7c7d1   --art-gray-900: #e3e3e8
--art-card-border: rgba(255,255,255,0.08)
--default-border: rgba(255,255,255,0.1)
--default-border-dashed: #363843
--default-bg-color: #070707
--default-box-color: #161618
--art-hover-color: #252530
--art-active-color: #202226
--art-el-active-color: #2e2e38
```
Runtime-injected vars (not in CSS): `--custom-radius` (`${settingStore.customRadius}rem`, set in `initializeTheme`), `--art-full-height` (set by `useAutoLayoutHeight`), `--el-color-primary`, `--el-color-primary-light-1..9`, `--el-color-primary-dark-1..9`, `--el-color-primary-custom-1..15`.

### Tailwind bridge (`@theme` block) — use these utility names, never raw hex
```
bg-box / text-box            ← --default-box-color
text-theme / bg-theme        ← --theme-color (Element primary)
bg-hover-color               ← --art-hover-color
bg-active-color              ← --art-active-color
bg-el-active-color           ← --art-active-color
{text,bg,border}-primary|secondary|error|info|success|warning|danger
{text,bg,border}-g-100 … g-900   ← --art-gray-*  (auto dark-mode)
```
Custom utilities defined in the same file: `.flex-c` (flex items-center), `.flex-b` (flex justify-between), `.flex-cc`, `.flex-cb`, `.tad-200`, `.tad-300`, `.border-full-d`, `.border-b-d`, `.border-t-d`, `.border-l-d`, `.border-r-d`, `.c-p`; radius utilities `rounded-custom-xs` / `rounded-custom-sm`; component class `.art-card-header` (with `.title h4` / `.title p` typography). Dark variant is `@custom-variant dark (&:where(.dark, .dark *))` — i.e. `dark:` works off the `.dark` class on `<html>`.

### Layout classes from `TPL/src/assets/styles/core/app.scss`
- `.art-full-height` → `height: var(--art-full-height); display:flex; flex-direction:column;` (auto `height:auto` under 640px).
- `.art-table-card` → `flex:1; display:flex; flex-direction:column; margin-top:12px; border-radius: calc(var(--custom-radius)/2 + 2px)`, with `.el-card__body { height:100%; overflow:hidden }`.
- `.art-card` / `.art-card-sm` / `.art-card-xs` → theme-aware bordered/shadowed boxes; the actual look is switched by `[data-box-mode='border-mode' | 'shadow-mode']` via `@mixin art-card-base($border-color, $shadow, $radius-diff)` (radius-diff 4px / 0px / -4px).
- `.page-content` → default page box (padding 20px, `--default-box-color`, rounded).
- `.el-full-screen` → the class ArtTableHeader toggles on `.art-page-view`.
- `.art-badge` / `.art-text-badge` → menu badges driven by `meta.showBadge` / `meta.showTextBadge`.

### SCSS mixins auto-injected into every `<style lang="scss">` block
Vite injects `@use "@styles/core/el-light.scss" as *; @use "@styles/core/mixin.scss" as *;` into **every** SCSS block. Available from `TPL/src/assets/styles/core/mixin.scss` **without importing**:
```
@include ellipsis($rowCount: 1)          // 1 = nowrap ellipsis, >1 = -webkit-line-clamp
@include userSelect($value: none)
@include absoluteCenter()
@include animation($from, $to, $name, $animate)
@include circle($size: 11px, $bg: #fff)
@include placeholder($color: #bbb)
@include betterTransparentize($color, $alpha)
@include browserPrefix($propertyName, $value)
@include border($color: red)
@include backdropBlur()
```

### Rules a page author MUST follow to be dark-mode-correct
1. **Never hardcode a hex/rgb color** for text, background, border, or hover state in page CSS.
2. Text → `text-g-900` (headline), `text-g-700` (body), `text-g-600` (muted), `text-g-500` (dimmer). In raw CSS: `color: var(--art-gray-900)` etc.
3. Surface/background → `bg-box` or `var(--default-box-color)`; the page canvas is `var(--default-bg-color)`. Use `.art-card` / `.art-card-xs` / `.art-table-card` instead of hand-rolled boxes so `data-box-mode` keeps working.
4. Borders → `border-full-d`/`border-b-d`… or `var(--default-border)`; card borders `var(--art-card-border)`; dashed `var(--default-border-dashed)`.
5. Hover/active → `hover:bg-hover-color` / `bg-active-color` (or `var(--art-hover-color)` / `var(--art-active-color)`); Element-Plus-selected rows use `var(--art-el-active-color)`.
6. Brand color → `text-theme` / `bg-theme` / `var(--theme-color)`; per-shade needs use `var(--el-color-primary-light-N)` (N 1..9). For chart series call `useChartOps().themeColor` / `.colors`, never a literal.
7. Radius → `calc(var(--custom-radius) …)` or `rounded-custom-xs|sm`, so the user's radius setting applies.
8. In `<style scoped>` (plain CSS, not SCSS) you must add `@reference '@styles/core/tailwind.css';` at the top before using `@apply` — see `art-table-header/index.vue` and `views/auth/login/style.css`.

### Reading / switching theme
```ts
const settingStore = useSettingStore()                       // src/store/modules/setting.ts
const { isDark, systemThemeColor, menuOpen, menuType,
        customRadius, containerWidth, pageTransition,
        watermarkVisible, boxBorderMode, tabStyle } = storeToRefs(settingStore)
settingStore.setElementTheme('#5D87FF')   // sets systemThemeColor + calls setElementThemeColor
settingStore.reload()                     // toggles `refresh` -> ArtPageContent remounts the view

import { useTheme } from '@/hooks/core/useTheme'
const { switchThemeStyles, setSystemTheme, setSystemAutoTheme, prefersDark } = useTheme()
switchThemeStyles(SystemThemeEnum.DARK)   // 'light' | 'dark' | 'auto'
```
`isDark` is `computed(() => systemThemeType.value === SystemThemeEnum.DARK)`. `setSystemTheme` sets `<html class="dark">` (from `AppConfig.systemThemeStyles`), temporarily injects `* { transition: none !important }` (`#disable-transitions`) to avoid flashing, and regenerates the primary shades:
```ts
for (let i = 1; i <= 9; i++)
  document.documentElement.style.setProperty(`--el-color-primary-light-${i}`,
    isDark ? getDarkColor(primary, i/10) : getLightColor(primary, i/10))
```
`setElementThemeColor(color)` (`TPL/src/utils/ui/colors.ts`) sets `--el-color-primary`, light-1..9, dark-1..9 and 15 `--el-color-primary-custom-N` blends. `initializeTheme()` runs in `App.vue` `onBeforeMount` and also sets `--custom-radius`. `getCssVar(name)` reads a computed custom property.

---

## 10. Route module schema

### Exact type — `TPL/src/types/router/index.ts`
```ts
export interface RouteMeta extends Record<string | number | symbol, unknown> {
  /** 路由标题 */                title: string
  /** 路由图标 */                icon?: string
  /** 是否显示徽章 */            showBadge?: boolean
  /** 文本徽章 */                showTextBadge?: string
  /** 是否在菜单中隐藏 */        isHide?: boolean
  /** 是否在标签页中隐藏 */      isHideTab?: boolean
  /** 外部链接 */                link?: string
  /** 是否为iframe */            isIframe?: boolean
  /** 是否缓存 */                keepAlive?: boolean
  /** 操作权限 */                authList?: Array<{ title: string; authMark: string }>
  /** 是否为一级菜单 */          isFirstLevel?: boolean       // set by RouteTransformer, don't author
  /** 角色权限 */                roles?: string[]
  /** 是否固定标签页 */          fixedTab?: boolean
  /** 激活菜单路径 */            activePath?: string
  /** 是否为全屏页面 */          isFullPage?: boolean
  /** 是否为权限按钮行 */        isAuthButton?: boolean
  /** 权限标识 */                authMark?: string
  /** 父级路径 */                parentPath?: string
}

export interface AppRouteRecord extends Omit<RouteRecordRaw, 'meta' | 'children' | 'component'> {
  id?: number
  meta: RouteMeta
  children?: AppRouteRecord[]
  component?: string | (() => Promise<any>)   // ← STRING path relative to src/views, no extension
}
```
Meaning of each meta field:
- `title` — menu label + document title; if it starts with `menus.` it is treated as an i18n key (`formatMenuTitle`, `TPL/src/utils/router.ts`); if the key is missing it falls back to the last dot segment.
- `icon` — Iconify name (`'ri:user-3-line'`), rendered by `ArtSvgIcon`.
- `roles` — frontend-mode menu filter (`MenuProcessor.filterMenuByRoles`); parent without matching role removes the whole subtree.
- `authList` — backend-mode button permissions; consumed by `v-auth` and `useAuth().hasAuth` in backend mode.
- `keepAlive` — `ArtPageContent` puts the component inside `<KeepAlive :max="10" :exclude="keepAliveExclude">` only when true. **Requires `defineOptions({ name })` matching the route `name`** or cache eviction on tab close silently fails.
- `isHide` — hide from sidebar menu (still routable).
- `isHideTab` — never opens a worktab.
- `isFullPage` — page renders fixed/fullscreen, no header/sidebar chrome (`ArtPageContent` `containerStyle`), used by 403/404/500.
- `isIframe` + `link` — renders `views/outside/Iframe.vue` under the layout (`RouteTransformer.handleIframeRoute`); `link` with `isIframe:false` opens an external URL from the menu.
- `showBadge` — red dot (`.art-badge`); `showTextBadge: 'v3.0.2'` — text pill (`.art-text-badge`).
- `fixedTab` — worktab that can't be closed (Console uses it).
- `activePath` — which menu entry highlights when this (usually `isHide`) route is active, e.g. detail page pointing at its list.
- `isFirstLevel` — internal, set when a depth-0 leaf route gets wrapped in Layout.

### Full example — `TPL/src/router/modules/system.ts`
```ts
import { AppRouteRecord } from '@/types/router'

export const systemRoutes: AppRouteRecord = {
  path: '/system',
  name: 'System',
  component: '/index/index',                 // RoutesAlias.Layout
  meta: { title: 'menus.system.title', icon: 'ri:user-3-line', roles: ['R_SUPER', 'R_ADMIN'] },
  children: [
    { path: 'user', name: 'User', component: '/system/user',
      meta: { title: 'menus.system.user', icon: 'ri:user-line', keepAlive: true, roles: ['R_SUPER','R_ADMIN'] } },
    { path: 'role', name: 'Role', component: '/system/role',
      meta: { title: 'menus.system.role', icon: 'ri:user-settings-line', keepAlive: true, roles: ['R_SUPER'] } },
    { path: 'user-center', name: 'UserCenter', component: '/system/user-center',
      meta: { title: 'menus.system.userCenter', icon: 'ri:user-line', isHide: true, keepAlive: true, isHideTab: true } },
    { path: 'menu', name: 'Menus', component: '/system/menu',
      meta: { title: 'menus.system.menu', icon: 'ri:menu-line', keepAlive: true, roles: ['R_SUPER'],
        authList: [ { title: '新增', authMark: 'add' }, { title: '编辑', authMark: 'edit' }, { title: '删除', authMark: 'delete' } ] } },
    { path: 'nested', name: 'Nested', component: '',      // '' = pure grouping node
      meta: { title: 'menus.system.nested', icon: 'ri:menu-unfold-3-line', keepAlive: true },
      children: [ /* … arbitrarily deep … */ ] }
  ]
}
```
Detail-page pattern (`TPL/src/router/modules/article.ts`): `meta: { isHide: true, activePath: '/article/article-list' }`.
External-link / iframe pattern (`TPL/src/router/modules/help.ts`):
```ts
{ name: 'Document', path: '', component: '',
  meta: { title: 'menus.help.document', icon: 'ri:bill-line', link: WEB_LINKS.DOCS, isIframe: false, keepAlive: false } }
```

### End-to-end: registering a new page
1. Create `TPL/src/views/<domain>/<page>/index.vue` with `defineOptions({ name: 'XxxPage' })`.
2. Add an entry to (or create) `src/router/modules/<domain>.ts`, exporting `const xxxRoutes: AppRouteRecord`. Top-level node must use `component: '/index/index'` (the Layout) and children use string paths like `'/<domain>/<page>'` (resolved against `src/views`, `.vue` or `/index.vue` both work).
3. Register it in `src/router/modules/index.ts` → push into `routeModules`.
4. `src/router/routes/asyncRoutes.ts` re-exports `routeModules` as `asyncRoutes`; the `beforeEach` guard (`src/router/guards/beforeEach.ts`) asks `MenuProcessor` for the menu (frontend mode: `asyncRoutes` filtered by `userStore.info.roles`; backend mode: `fetchGetMenuList()` → `GET /api/v3/system/menus`), then `RouteRegistry.register()` transforms + `router.addRoute()`s them. The sidebar renders from the same menu data, so the menu appears automatically.
5. Add i18n keys in `src/locales/langs/zh.json` and `en.json` under `menus.<domain>.<page>` matching `meta.title`.
Static, always-public routes go in `src/router/routes/staticRoutes.ts` (real `component: () => import('@views/...')` there, not strings).

---

## 11. Auth / permission primitives

### `useAuth()` — `TPL/src/hooks/core/useAuth.ts`
```ts
const { hasAuth } = useAuth()
hasAuth('add')   // boolean
```
- Frontend mode (`VITE_ACCESS_MODE=frontend`): checks `userStore.info.buttons` (a `string[]`, e.g. `['add','edit','delete']`).
- Backend mode: checks `route.meta.authList.some(i => i.authMark === auth)`.
Mode comes from `useAppMode()` (`isFrontendMode`, `isBackendMode`, `currentMode`).

### Directives — registered in `TPL/src/directives/index.ts` via `setupGlobDirectives(app)`
- `v-auth="'edit'"` (`directives/core/auth.ts`) — reads `router.currentRoute.value.meta.authList` and **removes the element from the DOM** if `authMark` isn't present. `mounted` + `updated`. Note it consults `meta.authList` only, so it is the backend-mode directive; use `v-if="hasAuth('edit')"` for frontend-mode button permissions.
- `v-roles="'R_SUPER'"` or `v-roles="['R_SUPER','R_ADMIN']"` (`directives/core/roles.ts`) — OR semantics against `useUserStore().getUserInfo.roles`; removes the element otherwise.
- Also registered: `v-highlight` (highlight.js + line numbers + copy button, `directives/business/highlight.ts`) and `v-ripple` (`directives/business/ripple.ts`, used on every `ElButton` in this template).

### User store — `TPL/src/store/modules/user.ts` (`useUserStore`, persisted to `sys-v{version}-user`)
State: `language`, `isLogin`, `isLock`, `lockPassword`, `info: Partial<Api.Auth.UserInfo>` (contains `roles: string[]` and `buttons: string[]`), `searchHistory`, `accessToken`, `refreshToken`.
Getters/actions: `getUserInfo`, `getSettingState`, `getWorktabState`, `setUserInfo`, `setLoginStatus`, `setLanguage`, `setSearchHistory`, `setLockStatus`, `setLockPassword`, `setToken(access, refresh?)`, `logOut()` (clears everything, removes `iframeRoutes` from sessionStorage, resets router state, pushes to `Login` with a `redirect` query).
Role constants used across the template: `R_SUPER`, `R_ADMIN`, `R_USER`.

---

## 12. Layout shell

`TPL/src/views/index/index.vue` (route alias `RoutesAlias.Layout = '/index/index'`):
```vue
<div class="app-layout">
  <aside id="app-sidebar"><ArtSidebarMenu /></aside>
  <main id="app-main">
    <div id="app-header"><ArtHeaderBar /></div>
    <div id="app-content"><ArtPageContent /></div>
  </main>
  <div id="app-global"><ArtGlobalComponent /></div>
</div>
```
`defineOptions({ name: 'AppLayout' })`. Styles in `src/views/index/style.scss`: `#app-main` is the scroll container (`height:100vh; overflow:auto`), `#app-header` is `position: sticky`, `.layout-content` is `width: calc(100% - 40px); margin:auto`.

**What a page can assume exists**
- `ArtHeaderBar` (logo, menu collapse, refresh, fast-enter, breadcrumb or top menu, user menu, notifications, settings) and, inside it, `ArtWorkTab` when `showWorkTab`.
- `ArtBreadcrumb` — built from `route.matched` + `meta.title` via `formatMenuTitle`; nothing to do per page.
- `ArtPageContent` (`components/core/layouts/art-page-content/index.vue`) — renders `<RouterView>` wrapped in `<Transition :name="pageTransition" mode="out-in" appear>` and, when `route.meta.keepAlive`, `<KeepAlive :max="10" :exclude="keepAliveExclude">`. It puts `class="art-page-view"` on every routed component (this is the fullscreen target). It also renders `ArtFestivalTextScroll` and, when `VITE_OPEN_ROUTE_INFO === 'true'`, a `route.meta` debug strip. `#app-content-header` is the header block it measures.
- `ArtGlobalComponent` — mounts every enabled entry of `TPL/src/config/modules/component.ts`: settings panel, global search, screen lock, chat window, fireworks, **watermark**. Toggle with `enabled: false` there.
- The scroll container has `id="app-main"` — use `useCommon().scrollToTop()/smoothScrollToTop()/scrollTo(top, smooth)` rather than `window.scrollTo`.

**Full-viewport page convention**
- `useAutoLayoutHeight(headerIds = ['app-header','app-content-header'], { extraSpacing = 15, updateCssVar = true, cssVarName = '--art-full-height' })` (`TPL/src/hooks/core/useLayoutHeight.ts`) is already called once by `ArtPageContent`. It computes `calc(100vh - (headerH + contentHeaderH + 15)px)` and writes it to `--art-full-height` on `document.documentElement`.
- A page therefore only needs `class="art-full-height"` on its root div (which also makes it a column flexbox), then `class="art-table-card"` on the `ElCard` to flex-grow into the remaining space.
- Inside that card, `ArtTable` computes its own height with `useTableHeight` (see §4). Do not set explicit table heights.
- Sibling hook `useLayoutHeight(options)` returns `{ containerMinHeight, headerRef, contentHeaderRef, headerHeight, contentHeaderHeight }` for the manual-ref variant.

---

## 13. Login page family — `TPL/src/views/auth/`

Three siblings, identical skeleton: `login/index.vue`, `register/index.vue`, `forget-password/index.vue`. Routes are static (`staticRoutes`): `/auth/login` (`Login`), `/auth/register` (`Register`), `/auth/forget-password` (`ForgetPassword`), all with `meta.isHideTab: true`.

Shared shell (copy this for a new auth page):
```vue
<template>
  <div class="flex w-full h-screen">
    <LoginLeftView />

    <div class="relative flex-1">
      <AuthTopBar />

      <div class="auth-right-wrap">
        <div class="form">
          <h3 class="title">{{ $t('login.title') }}</h3>
          <p class="sub-title">{{ $t('login.subTitle') }}</p>
          <ElForm ref="formRef" :model="formData" :rules="rules" :key="formKey" @keyup.enter="handleSubmit" style="margin-top: 25px">
            <ElFormItem prop="username">
              <ElInput class="custom-height" :placeholder="$t('login.placeholder.username')" v-model.trim="formData.username" />
            </ElFormItem>
            ...
            <ElButton class="w-full custom-height" type="primary" @click="handleSubmit" :loading="loading" v-ripple>
              {{ $t('login.btnText') }}
            </ElButton>
            <div class="mt-5 text-sm text-g-600">
              <span>{{ $t('login.noAccount') }}</span>
              <RouterLink class="text-theme" :to="{ name: 'Register' }">{{ $t('login.register') }}</RouterLink>
            </div>
          </ElForm>
        </div>
      </div>
    </div>
  </div>
</template>
<style scoped>@import './style.css';</style>
```
- `LoginLeftView` (`components/core/views/login/LoginLeftView.vue`) — 65vw decorative panel (`ArtLogo`, `ThemeSvg` with `@imgs/svg/login_icon.svg`, i18n `login.leftView.title/subTitle`, geometric decorations, sun/moon toggle calling `themeAnimation`). Prop: `hideContent?: boolean`. Collapses to nothing below 1180px. Colors derive from `--el-color-primary-light-7/8/9` mixed with `--default-box-color`, with a `.dark .login-left-view` block.
- `AuthTopBar` (`components/core/views/login/AuthTopBar.vue`) — absolute top-right cluster: theme-color picker (`AppConfig.systemMainColor` → `settingStore.setElementTheme` + `reload`), language dropdown (`languageOptions` from `@/locales`, `userStore.setLanguage`), dark/light toggle (`themeAnimation` from `@/utils/ui/animation`). Visibility controlled by `useHeaderBar()`'s `shouldShowThemeToggle` / `shouldShowLanguage`.
- SCSS/CSS: `TPL/src/views/auth/login/style.css` is plain CSS with `@reference '@styles/core/tailwind.css';` at the top and `@apply` inside. It defines `.auth-right-wrap` (440×650 centered, `slideInRight` animation), `.form`, `.title` (`text-g-900 text-4xl font-semibold`), `.sub-title` (`text-g-600 text-sm`), `.custom-height` (`!h-[40px]`). Register/forget-password `@import` this same file — new auth pages should too.
- Drag verify block (login only):
```vue
<div class="relative pb-5 mt-6">
  <div class="relative z-[2] overflow-hidden select-none rounded-lg border border-transparent tad-300"
       :class="{ '!border-[#FF4E4F]': !isPassing && isClickPass }">
    <ArtDragVerify ref="dragVerify" v-model:value="isPassing" :text="$t('login.sliderText')"
      textColor="var(--art-gray-700)" :successText="$t('login.sliderSuccessText')"
      progressBarBg="var(--main-color)" :background="isDark ? '#26272F' : '#F1F1F4'"
      handlerBg="var(--default-box-color)" />
  </div>
  <p class="absolute top-0 z-[1] px-px mt-2 text-xs text-[#f56c6c] tad-300"
     :class="{ 'translate-y-10': !isPassing && isClickPass }">{{ $t('login.placeholder.slider') }}</p>
</div>
```
- i18n reset trick present in both login and register: `const formKey = ref(0); watch(locale, () => formKey.value++)` with `:key="formKey"` on the form, so validation messages re-render on language change.
- Submit flow: `formRef.value.validate()` → drag check → `fetchLogin({userName, password})` → `userStore.setToken(token, refreshToken)` + `setLoginStatus(true)` → `router.push(route.query.redirect || '/')` → `ElNotification` success, `dragVerify.value.reset()` in `finally`.
- i18n namespaces already present in `zh.json`/`en.json`: `login`, `register`, `forgetPassword`, `lockScreen`.

---

## 14. `scripts/clean-dev.ts` (`pnpm clean:dev`)

Interactive (requires typing `yes`), **destructive and irreversible**. Seven steps:

**1. `fs.rm(..., { recursive: true, force: true })` on this exact list:**
```
README.md, README.zh-CN.md, CHANGELOG.md, CHANGELOG.zh-CN.md
src/views/change, src/views/safeguard, src/views/article, src/views/examples,
src/views/system/nested, src/views/widgets, src/views/template,
src/views/dashboard/analysis, src/views/dashboard/ecommerce
src/mock/json, src/mock/temp/articleList.ts, src/mock/temp/commentDetail.ts, src/mock/temp/commentList.ts
src/assets/images/cover, src/assets/images/safeguard, src/assets/images/3d
src/components/core/charts/art-map-chart
src/components/business/comment-widget
```
**2. Route modules:** deletes `src/router/modules/{template,widgets,examples,article,safeguard,help}.ts`; **overwrites** `dashboard.ts` (Console only, icons stripped), `system.ts` (user/role/user-center/menu only, no `nested`, icons stripped), and `index.ts` (imports only dashboard/system/result/exception).
**3.** Overwrites `src/router/routesAlias.ts` down to `Layout` + `Login`.
**4.** Overwrites `src/mock/upgrade/changeLog.ts` with an empty `upgradeLogList`.
**5.** Rewrites `src/locales/langs/zh.json` and `en.json`, deleting `menus.{widgets,template,article,examples,safeguard,plan,help}`, `menus.dashboard.{analysis,ecommerce}`, `menus.system.{nested,menu1,menu2,menu21,menu3,menu31,menu32,menu321}` (re-serialized with 2-space indent).
**6.** Overwrites `src/config/fastEnter.ts` with a 4-app / 4-quick-link config. **⚠️ Note the real file lives at `src/config/modules/fastEnter.ts` in v3.0.2 — this step writes to a stale path and leaves the actual config untouched while creating a stray `src/config/fastEnter.ts`.**
**7.** In `src/api/system-manage.ts`, replaces `url: '/api/v3/system/menus'` with `'/api/v3/system/menus/simple'`.

Preserved: Dashboard(Console), System, Result, Exception, Auth pages, all of `src/components/core` except art-map-chart, all hooks/stores/styles.

**Is it a safe way to strip demo pages?** Yes for a fresh, unmodified template checkout, and it is the intended workflow — but: (a) it is irreversible with no backup, run it on a clean git commit; (b) it *overwrites* `dashboard.ts`, `system.ts`, `modules/index.ts`, `routesAlias.ts`, `changeLog.ts` and both locale JSONs, so any edits you already made to those files are lost — **run it before you start writing code, not after**; (c) it also deletes the READMEs/CHANGELOGs; (d) the fastEnter step is a no-op-plus-litter (see above), so afterwards you must manually prune `src/config/modules/fastEnter.ts` and delete the stray `src/config/fastEnter.ts`; (e) `src/views/examples` disappearing means your team loses the living documentation for `useTable`/`ArtSearchBar`/`ArtForm` — archive it or this document first.

---

## 15. Non-obvious gotchas

1. **SCSS auto-injection.** `vite.config.ts` prepends `@use "@styles/core/el-light.scss" as *; @use "@styles/core/mixin.scss" as *;` to every SCSS block. Adding your own `@use '@styles/core/mixin.scss'` causes a duplicate/conflict; just call the mixins. This only applies to `lang="scss"` — a plain `<style scoped>` gets neither mixins nor Tailwind context.
2. **`@reference` for Tailwind in plain CSS.** In `<style scoped>` (no `lang="scss"`) you must put `@reference '@styles/core/tailwind.css';` first or `@apply` silently produces nothing (see `art-table-header/index.vue`, `views/auth/login/style.css`).
3. **`unplugin-element-plus` with `useSource: true`.** The build compiles Element Plus theme SCSS from source so `el-light.scss`/`el-dark.scss` overrides apply. If you add an EP component that you import manually rather than via the resolver, its styles may not be themed. Keep relying on auto-import; and keep `element-plus/es/components/*/style/{css,index}` in `optimizeDeps.include` or dev-server cold starts thrash.
4. **`import.meta.glob` path strings in `ComponentLoader`.** `TPL/src/router/core/ComponentLoader.ts` globs `'../../views/**/*.vue'` and looks up exactly `../../views${componentPath}.vue` or `../../views${componentPath}/index.vue`. So `meta`-less details: the `component` string must start with `/`, must not include `.vue`, must be under `src/views`, and must be *statically* matchable — a component outside `src/views` renders a red `组件未找到: ...` div. Also `build.dynamicImportVarsOptions.include: ['src/views/**/*.vue']`.
5. **Hash router.** `createWebHashHistory()` in `TPL/src/router/index.ts`. All URLs are `/#/system/user`. Switching to `createWebHistory` requires server rewrite rules **and** a matching `VITE_BASE_URL`. `HOME_PAGE_PATH = ''` means "first valid menu path"; set it to force a landing route.
6. **`base` comes from `VITE_BASE_URL`, not `'./'`.** `vite.config.ts` has `base: VITE_BASE_URL` (default `/`). Deploying to a subdirectory means editing `.env.production` → `VITE_BASE_URL = /admin/`; a bare `./` will break the hash router's asset URLs.
7. **keep-alive requires `defineOptions({ name })`.** `KeepAlive :exclude="keepAliveExclude"` in `ArtPageContent` excludes by **component name**, and `worktab.ts` pushes `tab.name` (the *route* name) into that list on close. If the SFC's `name` ≠ the route `name`, closing a tab won't evict its cache and stale state reappears. Every page in the template does `defineOptions({ name: 'User' })` matching `name: 'User'` in the route.
8. **`getData` vs `fetchData` are swapped relative to intuition.** `useTable().getData` is `getDataByPage` (resets to page 1, skips cache) and `useTable().fetchData` is the keep-current-page loader. Use `getData()` after a search, `fetchData()` for a silent reload.
9. **`pagination` is `readonly`.** Don't assign `pagination.current = 2`; call `handleCurrentChange(2)`.
10. **`searchParams` must be mutated, not replaced.** It's a `reactive` object closed over by the fetcher: `Object.assign(searchParams, params)`. Reassigning the destructured variable does nothing.
11. **Params you don't want on the wire.** UI-only fields (e.g. `daterange`) must be listed in `core.excludeParams`, otherwise they are sent verbatim.
12. **ArtSearchBar deletes empty keys.** `normalizeFieldValue` turns `''` into `undefined` and `delete`s the key; the `search` payload is additionally sanitized (empty strings/arrays/objects and `<p><br></p>` removed, `0`/`false` kept). If your backend needs an explicit empty value, pass `:sanitize-output="{ removeEmptyString: false }"`.
13. **`reset` restores the initial snapshot, not blanks.** ArtSearchBar/ArtForm clone `modelValue` at setup and re-apply it on reset — so `status: '1'` defaults survive a reset. Combine with `useTable().resetSearchParams` (which re-applies `core.apiParams`) so the two stay consistent.
14. **Only one `#art-table-header` per page.** ArtTable finds it with `document.getElementById`; two headers ⇒ wrong height math. And with no header you must pass `:show-table-header="false"`.
15. **`ColumnOption.formatter` is not el-table's `formatter`.** Signature is `(row) => VNode|string`. If you need `(row, column, cellValue, index)`, use `useSlot: true` + a template slot instead.
16. **Column `type: 'index'` restarts at 1 on every page** — use `type: 'globalIndex'` for continuous numbering across pages.
17. **`ArtTable` inherits `TableProps`, so Boolean props are dangerous.** The component explicitly guards `selectOnIndeterminate` via `hasExplicitTableProp` because inherited Boolean props would otherwise stomp EP defaults with `false`. If you add new boolean pass-throughs, mimic that guard.
18. **`v-auth` is backend-mode only.** It reads `route.meta.authList`; in `VITE_ACCESS_MODE=frontend` that list may not exist, so nothing is ever permitted. Use `useAuth().hasAuth()` with `v-if` for mode-agnostic code. Both `v-auth` and `v-roles` **remove the DOM node** (`el.parentNode.removeChild`) — they cannot be re-added reactively, so don't rely on them for state that changes after mount, and never put them on a transition/keyed root.
19. **i18n `menus.*` prefix is load-bearing.** `formatMenuTitle` only translates titles starting with `menus.`; anything else is shown literally. A missing key falls back to the last dot segment (so a typo shows `console` instead of an error). Keys must exist in **both** `zh.json` and `en.json`. Element Plus locale comes separately from `App.vue`'s `ElConfigProvider :locale="locales[language]"`.
20. **`clean:dev` overwrites route/locale files** — see §14; run it first or lose edits.
21. **`--art-full-height` is a single global var.** Two `art-full-height` elements on one page both get the same computed height; nest content instead.
22. **Fullscreen target is `.art-page-view`.** That class is applied by `ArtPageContent` to the routed component's root. If your page's root element is a `<template>` fragment or you override the class, `ArtTableHeader`'s fullscreen button silently does nothing (`document.querySelector` returns null) — pass a custom `fullClass`.
23. **`.dark` does not redefine `--art-primary/--art-secondary/--art-error/...`.** Those OKLCH values are the same in both modes; only the gray ramp, borders, backgrounds and hover/active flip. Don't assume `bg-primary` inverts.
24. **Element Plus primary shades are JS-generated.** `--el-color-primary-light-N` is recomputed by `setSystemTheme`/`setElementThemeColor` from a **hex** string; `getLightColor`/`getDarkColor` throw (and `ElMessage.warning`) on a non-hex input, so `settingStore.setElementTheme()` must be given `#RRGGBB`.
25. **`useUserStore` is instantiated at module scope in `useAuth.ts`** (`const userStore = useUserStore()` outside the composable). Importing `@/hooks/core/useAuth` before Pinia is installed will throw — it's fine inside components, avoid it in early bootstrap code.
26. **Env flags that change behavior:** `VITE_ACCESS_MODE` (frontend|backend) switches whole menu+permission pipeline; `VITE_OPEN_ROUTE_INFO='true'` renders a route-meta debug strip on every page; `VITE_WITH_CREDENTIALS`; `VITE_LOCK_ENCRYPT_KEY`. Production build sets `drop_console: true` / `drop_debugger: true` in terser.
27. **Persisted store keys are version-scoped** (`sys-v{VITE_VERSION}-user`, `-setting`, `-worktab`). Bumping `VITE_VERSION` in `.env` invalidates user settings and worktabs (there's a migration/compat check in `checkStorageCompatibility` + `systemUpgrade`).
28. **`ArtWatermark` is already global** via `config/modules/component.ts` and keyed off `settingStore.watermarkVisible` — don't mount a second one per page.
29. **Node/pnpm floor:** `node >= 20.19.0`, `pnpm >= 8.8.0`. Build script is `vue-tsc --noEmit && vite build`, so type errors block the build.
30. **`AppRouteRecord.component` is a string in route modules but a real import in `staticRoutes.ts`.** Mixing them up gives you either "组件未找到" (string in staticRoutes context) or a broken addRoute.
---

## 16. 硬编码颜色规则的唯一例外：邮件模板

`views/admin/settings/email/index.vue` 里的 `defaultTemplates` 是**邮件正文的 HTML 字符串**，
不是组件样式。邮件客户端会剥掉 `<style>` 标签、也不支持 CSS 自定义属性，
所以那里**必须**用内联 style + 十六进制颜色。

合规扫描时排除它：

```bash
grep -rniE '#[0-9a-f]{3,8}\b|rgba?\(' src/views src/components/business \
  | grep -viE 'art-|el-|url\(|data:image' \
  | grep -v 'views/admin/settings/email/index.vue'
```

**不要把这些颜色改成 CSS 变量**，改了邮件在客户端里就没样式了。
