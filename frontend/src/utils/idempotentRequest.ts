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

function isUnknownOutcome(error: unknown): boolean {
  if (!isAxiosError(error)) return false
  const status = error.response?.status
  return status == null || status >= 500
}

export class IdempotentRequest<T> {
  private pending: IdempotentAttempt<T> | undefined

  constructor(private readonly prefix: string) {}

  prepare(payloadFactory: () => T): IdempotentAttempt<T> {
    this.pending ??= {
      key: createKey(this.prefix),
      payload: payloadFactory()
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
