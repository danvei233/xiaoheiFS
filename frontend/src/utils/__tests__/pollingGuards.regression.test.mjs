/**
 * 回归测试：详情/审核抽屉/探针页的轮询与请求代次守卫（源码漂移防护）。
 *
 * 背景（web_data_races F1/F2/F3/F4 + web_perf F4/F5）：
 *  - 控制台订单详情 `fetchDetail` 无在途保护 + 无代次，慢响应下轮询重叠、
 *    旧响应覆盖新状态；SSE 事件数组无界增长。
 *  - 管理订单审核抽屉 `loadDetail` 无代次，打开订单 B 时 A 的慢响应覆盖
 *    详情；每次轮询还重复拉取用户名（N+1）。
 *  - 探针详情 `requestInFlight` 全局布尔在切换 probeId 时吞掉新请求，
 *    展示旧探针数据；SLA 每 5s 全窗口重算造成请求风暴。
 *
 * 这些逻辑位于 .vue SFC 内，项目测试栈（node --test + esbuild）不打包 SFC，
 * 因此用**源码漂移断言**保证守卫不被回退（与 urlPolicy 测试同法）：
 * 断言关键守卫标识符必须存在于对应视图源码。
 *
 * 运行：`node --test src/utils/__tests__/pollingGuards.regression.test.mjs`
 */
import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const here = path.dirname(fileURLToPath(import.meta.url))
const srcRoot = path.resolve(here, '../../../src')

const view = (relPath) => fs.readFileSync(path.resolve(srcRoot, relPath), 'utf8')

// ── 控制台订单详情（F4）──────────────────────────────────────────────
const ordersDetail = view('views/console/orders/detail.vue')
assert.ok(ordersDetail.includes('if (fetchingDetail) return'), 'fetchDetail 必须有在途保护')
assert.ok(ordersDetail.includes('detailReqId'), 'fetchDetail 必须有请求代次')
assert.ok(ordersDetail.includes('requestId !== detailReqId'), '旧响应不得覆盖新状态')
assert.ok(ordersDetail.includes('MAX_SSE_EVENTS'), 'SSE 事件必须设上限')
assert.ok(ordersDetail.includes('events.value.splice(MAX_SSE_EVENTS)'), 'SSE 事件超限裁剪')
assert.ok(ordersDetail.includes('detailController?.abort()'), '离开页面中止在途详情请求')

// ── 管理订单审核抽屉（F1/F5）─────────────────────────────────────────
const review = view('views/admin/order/review/index.vue')
assert.ok(review.includes('if (loadingDetail) return'), 'loadDetail 必须有在途保护')
assert.ok(review.includes('requestId !== detailReqId'), 'loadDetail 必须有请求代次')
assert.ok(review.includes('lastLoadedUserId'), '用户名拉取必须按 user_id 去重（N+1）')
assert.ok(review.includes('detailController?.abort()'), '切换/关闭抽屉中止在途详情请求')
assert.ok(
  review.includes('useLifecycleSafeTimer'),
  '抽屉轮询必须用 keep-alive 生命周期安全定时器（F1 后台请求风暴）'
)
assert.ok(review.includes('{ silent: true }'), '抽屉轮询 tick 必须静默（不闪整面板 loading，F2）')
assert.ok(review.includes('onDeactivated'), '切后台必须停表')
assert.ok(review.includes('onActivated'), '切回前台必须恢复轮询')

// ── 探针列表（F1）────────────────────────────────────────────────────
const probeList = view('views/admin/probe/list/index.vue')
assert.ok(
  probeList.includes('useLifecycleSafeTimer'),
  '探针列表轮询必须用 keep-alive 生命周期安全定时器（F1 后台请求风暴）'
)
assert.ok(probeList.includes('onActivated'), '探针列表切回前台必须恢复轮询')

// ── 实名记录页用户 ID 输入（F8）：逐键触发必须防抖 ────────────────────
const realnameRecords = view('views/admin/realname/records/index.vue')
assert.ok(
  realnameRecords.includes('triggerFilterDebounced'),
  '用户 ID 输入必须走防抖（不得每键发一次请求）'
)
assert.ok(
  realnameRecords.includes('getDataDebounced'),
  '防抖路径必须用 useTable 的 getDataDebounced'
)

// ── 探针详情（F2/F3）─────────────────────────────────────────────────
const probe = view('views/admin/probe/detail.vue')
assert.ok(probe.includes('inFlightProbeId'), '探针详情必须按 probeId 跟踪在途请求')
assert.ok(
  probe.includes('if (controller.signal.aborted || probeId.value !== targetId) return'),
  '切换探针后旧响应不得写回'
)
assert.ok(probe.includes('fetchSla'), 'SLA 必须独立拉取')
assert.ok(probe.includes('30000'), 'SLA 慢轮询间隔（30s）')
assert.ok(probe.includes('startSlaPolling'), 'SLA 慢轮询已启动')

// ── 控制台订单详情 SSE 终结态关闭（F9）───────────────────────────────
assert.ok(
  ordersDetail.includes("'active', 'failed', 'rejected', 'cancelled', 'expired'"),
  '终结态列表必须存在'
)
assert.ok(ordersDetail.includes('stopSse()'), '状态推进到终结态必须关闭 SSE')
assert.ok(
  ordersDetail.includes(
    "const activeStates = ['pending_payment', 'pending_review', 'approved', 'provisioning']"
  ),
  'SSE 只为活跃推进态建连'
)

// ── 服务层：详情端点接受 AbortSignal（可被页面用于中止）────────────────
const adminSvc = fs.readFileSync(path.resolve(srcRoot, 'services/admin.ts'), 'utf8')
const userSvc = fs.readFileSync(path.resolve(srcRoot, 'services/user.ts'), 'utf8')
assert.ok(
  adminSvc.includes('getAdminOrderDetail = (id: number | string, signal?: AbortSignal)'),
  'getAdminOrderDetail 接受 signal'
)
assert.ok(
  adminSvc.includes('getAdminUserDetail = (id: number | string, signal?: AbortSignal)'),
  'getAdminUserDetail 接受 signal'
)
assert.ok(
  userSvc.includes('getOrderDetail = (id: number | string, signal?: AbortSignal)'),
  'getOrderDetail 接受 signal'
)
assert.ok(
  adminSvc.includes(
    'getAdminProbeDetail = (\n  id: number | string,\n  params?: Record<string, unknown>,\n  signal?: AbortSignal\n)'
  ),
  'getAdminProbeDetail 接受 signal'
)

console.log('✅ pollingGuards.regression.test.mjs —— 轮询/代次守卫源码漂移断言通过')
