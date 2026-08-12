import { isAxiosError } from 'axios'

export interface IdempotentAttempt<T> {
  readonly key: string
  readonly payload: T
}

function createKey(prefix: string): string {
  const suffix =
    globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`
  return `${prefix}-${suffix}`
}

/**
 * 结果未知的错误 = 请求可能已在服务端落库，绝不能复用/释放旧 key：
 *  - 无状态码（网络错误/超时）
 *  - 408 请求超时
 *  - 429 限流（"稍后重试"，未确认结果）
 *  - 5xx 服务端错误
 *
 * 4xx 是确定性的"没成"，可以安全释放 key（P2-71 / m054 F1+F2）。
 */
function isUnknownOutcome(error: unknown): boolean {
  if (!isAxiosError(error)) return false
  const status = error.response?.status
  return status == null || status === 408 || status === 429 || status >= 500
}

/**
 * 深比较两个 payload 是否等价（对象键序无关）。用于判断用户是否改了提交意图：
 * 意图变了 → 必须换新 key + 新 payload，否则后端 `(userID, key)` 去重会命中
 * 旧记录，把新表单当成上次的重复提交。
 */
function isSamePayload(a: unknown, b: unknown): boolean {
  if (Object.is(a, b)) return true
  if (typeof a !== typeof b) return false
  if (a === null || b === null) return a === b
  if (Array.isArray(a) && Array.isArray(b)) {
    if (a.length !== b.length) return false
    return a.every((item, index) => isSamePayload(item, b[index]))
  }
  if (typeof a === 'object' && typeof b === 'object') {
    const aKeys = Object.keys(a as object).sort()
    const bKeys = Object.keys(b as object).sort()
    if (aKeys.length !== bKeys.length) return false
    return aKeys.every((key) =>
      isSamePayload((a as Record<string, unknown>)[key], (b as Record<string, unknown>)[key])
    )
  }
  return false
}

export class IdempotentRequest<T> {
  private pending: IdempotentAttempt<T> | undefined

  constructor(private readonly prefix: string) {}

  /**
   * 每次调用都重新求值 payloadFactory：
   *  - 与上一次相同 → 复用 key（用户原样重试，服务端去重命中同一条记录）；
   *  - 与上一次不同 → 生成新 key + 新 payload（新意图，不得复用旧 key）。
   */
  prepare(payloadFactory: () => T): IdempotentAttempt<T> {
    const payload = payloadFactory()
    if (!this.pending || !isSamePayload(this.pending.payload, payload)) {
      this.pending = { key: createKey(this.prefix), payload }
    }
    return this.pending
  }

  resolve(): void {
    this.pending = undefined
  }

  reject(error: unknown): void {
    if (!isUnknownOutcome(error)) this.resolve()
  }
}
