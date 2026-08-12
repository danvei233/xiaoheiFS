/**
 * 回归测试：useTable 列表端点 AbortSignal 网络层转发（web_perf F6）。
 *
 * 背景：`useTable` 把 `currentController.signal` 作为 apiFn 第二实参传入，但
 *   大量视图的 apiFn（局部包装函数）与服务层函数不接受/不透传 signal，翻页时
 *   旧请求不会在网络层被中止，浪费后端查询（客户端已丢弃旧响应，但请求照发）。
 *
 * 修复：为被点名的视图（automation/catalog/cms/marketing/scheduled-tasks/
 *   revenue-analytics/settings/system/wallet/console-api-keys 等）的服务层
 *   函数与 .vue apiFn 包装统一加 `signal?: AbortSignal` 并透传到 http。
 *
 * 这些逻辑位于 .vue SFC 与 services/*.ts 内，用**源码漂移断言**保证信号链
 *   不被回退（与 urlPolicy 测试同法）。
 *
 * 运行：`node --test src/utils/__tests__/signalForwarding.regression.test.mjs`
 */
import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const here = path.dirname(fileURLToPath(import.meta.url))
const srcRoot = path.resolve(here, '../../../src')

const read = (rel) => fs.readFileSync(path.resolve(srcRoot, rel), 'utf8')

// ── 服务层：函数接受 signal 并透传到 http（{ params, signal } / { signal }）──
const adminSvc = read('services/admin.ts')
const userSvc = read('services/user.ts')

// 关键列表端点的 signal 透传链（签名 + http config 转发）
const svcCases = [
  [
    'admin.ts',
    'listCoupons',
    "http.get<ApiList<Coupon>>('/admin/api/v1/coupons', { params, signal })"
  ],
  [
    'admin.ts',
    'listApiKeys',
    "http.get<ApiList<Record<string, unknown>>>('/admin/api/v1/api-keys', { params, signal })"
  ],
  [
    'admin.ts',
    'listAdminWalletOrders',
    "http.get<WalletOrderListResponse>('/admin/api/v1/wallet/orders', { params, signal })"
  ],
  [
    'admin.ts',
    'listAdminScheduledTasks',
    "http.get<ApiList<Record<string, unknown>>>('/admin/api/v1/scheduled-tasks', { signal })"
  ],
  [
    'admin.ts',
    'listPermissionGroups',
    "http.get<ApiList<PermissionGroup>>('/admin/api/v1/permission-groups', { signal })"
  ],
  ['admin.ts', 'listAdminPlugins', '{ signal })'],
  ['admin.ts', 'fetchAdminCoupons', 'unwrapAdmin(listCoupons(params, signal))'],
  ['admin.ts', 'fetchAdminScheduledTasks', 'unwrapAdmin(listAdminScheduledTasks(signal))'],
  ['admin.ts', 'fetchCMSPosts', 'unwrapAdmin(listCmsPosts(params, signal))'],
  ['admin.ts', 'fetchCMSBlocks', 'unwrapAdmin(listCmsBlocks(params, signal))'],
  ['admin.ts', 'fetchAdminUploads', 'unwrapAdmin(listUploads(params, signal))'],
  ['admin.ts', 'fetchAdminAccounts', 'unwrapAdmin(listAdmins(params, signal))'],
  ['admin.ts', 'fetchAdminPermissions', 'unwrapAdmin(listPermissions(signal))'],
  [
    'user.ts',
    'listUserApiKeys',
    "http.get<ApiList<UserAPIKey>>('/api/v1/open/me/api-keys', { params, signal })"
  ]
]
for (const [file, name, needle] of svcCases) {
  const src = file === 'admin.ts' ? adminSvc : userSvc
  const sig = new RegExp(`export const ${name} = \\([^)]*signal\\?: AbortSignal`)
  assert.ok(sig.test(src), `${file}::${name} 签名必须接受 signal`)
  assert.ok(src.includes(needle), `${file}::${name} 必须透传 signal（期望包含 ${needle}）`)
}

// ── .vue apiFn 包装：接受 signal 并透传到服务调用 ───────────────────────
const viewCases = [
  ['views/admin/marketing/coupons/index.vue', 'fetchCouponTable', 'signal?: AbortSignal'],
  ['views/admin/settings/api-keys/index.vue', 'fetchApiKeys', 'signal?: AbortSignal'],
  ['views/admin/wallet/orders/index.vue', 'fetchWalletOrders', 'signal?: AbortSignal'],
  ['views/admin/ops/scheduled-tasks/index.vue', 'fetchScheduledTaskTable', 'signal?: AbortSignal'],
  ['views/admin/cms/blocks/index.vue', 'fetchBlockTable', 'signal?: AbortSignal'],
  ['views/admin/cms/posts/index.vue', 'fetchPostTable', 'signal?: AbortSignal'],
  ['views/admin/cms/uploads/index.vue', 'fetchUploadTable', 'signal?: AbortSignal'],
  ['views/admin/automation/index.vue', 'fetchLogTable', 'signal?: AbortSignal'],
  ['views/admin/system/admin/index.vue', 'fetchAdminTable', 'signal?: AbortSignal'],
  [
    'views/admin/system/permission-group/index.vue',
    'fetchPermissionGroupTable',
    'signal?: AbortSignal'
  ],
  ['views/admin/revenue-analytics/index.vue', 'fetchRevenueDetailsTable', 'signal?: AbortSignal'],
  ['views/console/api-keys/index.vue', 'fetchApiKeys', 'signal?: AbortSignal']
]
for (const [file, fn, needle] of viewCases) {
  const src = read(file)
  // 支持 `async function fetchX(` 与 `const fetchX = async (` 两种包装形式。
  const defIdx =
    src.indexOf(`function ${fn}(`) !== -1
      ? src.indexOf(`function ${fn}(`)
      : src.indexOf(`${fn} = async (`)
  assert.ok(defIdx !== -1, `${file}::${fn} 包装函数定义存在`)
  const block = src.slice(defIdx, defIdx + 400)
  assert.ok(
    block.includes(needle),
    `${file}::${fn} 包装函数必须接受 signal（前 400 字符内未找到 ${needle}）`
  )
}

// useTable 始终把 signal 作为第二实参传入（既有契约不得回退）
const useTableSrc = read('hooks/core/useTable.ts')
assert.ok(
  useTableSrc.includes('apiFn(requestParams, currentController.signal)'),
  'useTable 必须把 AbortSignal 作为 apiFn 第二实参传入'
)

console.log('✅ signalForwarding.regression.test.mjs —— AbortSignal 网络层转发断言通过')
