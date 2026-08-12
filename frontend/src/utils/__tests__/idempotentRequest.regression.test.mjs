/**
 * 回归测试：幂等请求 key 生命周期（P2-71 / m054 F1+F2）。
 *
 * 背景：
 *  - 408/429 被旧实现当"已知结果"释放 key → 用户改 payload 重试时可能
 *    命中旧记录或产生重复下单。
 *  - 未知结果（网络错误/5xx）后，旧实现复用旧 key + 旧 payload → 用户改了
 *    优惠券/trade_no 后重试，后端 `(userID, key)` 去重命中旧记录，新意图被吞。
 *
 * 目标契约：
 *  - 同 payload 重试 → 复用 key；
 *  - 改 payload → 换新 key + 新 payload；
 *  - 网络错误 / 408 / 429 / 5xx → reject 保留 key；
 *  - 确定性 4xx（400/404）→ reject 释放 key；
 *  - 成功 resolve → 清 key。
 *
 * 运行：`node --test src/utils/__tests__/idempotentRequest.regression.test.mjs`
 */
import { bundleEntry } from '../../../tests/_bundle.mjs'
import assert from 'node:assert/strict'

const { IdempotentRequest } = await bundleEntry(
  `export { IdempotentRequest } from '@/utils/idempotentRequest'`,
  import.meta.url
)

const axiosError = (status) => ({
  isAxiosError: true,
  response: status == null ? undefined : { status },
  message: 'Request failed'
})

// ── 1. 同 payload 复用 key ─────────────────────────────────────────────
const r = new IdempotentRequest('order')
const a1 = r.prepare(() => ({ coupon: 'C1', qty: 1 }))
const a2 = r.prepare(() => ({ coupon: 'C1', qty: 1 }))
assert.equal(a1.key, a2.key, '同 payload 重试应复用 key')
assert.deepEqual(a1.payload, a2.payload)

// ── 2. 改 payload 换新 key ─────────────────────────────────────────────
const a3 = r.prepare(() => ({ coupon: 'C2', qty: 1 }))
assert.notEqual(a1.key, a3.key, '改 payload 应换新 key')
assert.equal(a3.payload.coupon, 'C2', '新 key 必须携带新 payload')

// ── 3. 对象键序无关（同 payload 不同键序仍复用）────────────────────────
const b = new IdempotentRequest('order')
const b1 = b.prepare(() => ({ a: 1, b: 2 }))
const b2 = b.prepare(() => ({ b: 2, a: 1 }))
assert.equal(b1.key, b2.key, '键序不同的等价对象应视为同 payload')

// ── 4. 未知结果保留 key ────────────────────────────────────────────────
const c = new IdempotentRequest('order')
const c1 = c.prepare(() => ({ x: 1 }))
c.reject(axiosError(null)) // 网络错误
assert.equal(c.prepare(() => ({ x: 1 })).key, c1.key, '网络错误后同 payload 仍复用 key')
c.reject(axiosError(408))
assert.equal(c.prepare(() => ({ x: 1 })).key, c1.key, '408 后复用 key')
c.reject(axiosError(429))
assert.equal(c.prepare(() => ({ x: 1 })).key, c1.key, '429 后复用 key')
c.reject(axiosError(503))
assert.equal(c.prepare(() => ({ x: 1 })).key, c1.key, '5xx 后复用 key')

// ── 5. 确定性 4xx 释放 key ─────────────────────────────────────────────
const d = new IdempotentRequest('order')
const d1 = d.prepare(() => ({ x: 1 }))
d.reject(axiosError(400))
assert.notEqual(d.prepare(() => ({ x: 1 })).key, d1.key, '400 后应释放 key 换新')

// ── 6. 成功 resolve 清 key ─────────────────────────────────────────────
const e = new IdempotentRequest('order')
const e1 = e.prepare(() => ({ x: 1 }))
e.resolve()
assert.notEqual(e.prepare(() => ({ x: 1 })).key, e1.key, '成功后应清 key')

console.log('✅ idempotentRequest.regression.test.mjs —— key 生命周期断言通过')
