/**
 * 支付结果 URL 安全策略（H2）。
 *
 * 后端支付插件返回的 `code_url` / `pay_url` / `urlscheme` 是外部可控数据。
 * 直接交给 `window.location.href = 'javascript:...'` / `window.open(...)` 等价于
 * 同源脚本执行：`javascript:` 跳转在目标窗口里能读到本应用 token。因此所有
 * 支付出口必须统一经过 `classifyPaymentUrl` 校验后再导航。
 *
 * 规则（内核在 `utils/urlPolicy.ts`，H9 普通外链复用同一套）：
 *  - Web 跳转只允许 HTTPS（开发环境按明确规则允许 localhost HTTP）；
 *  - 自定义 scheme 只允许支付 scheme 白名单（`PAYMENT_SCHEME_ALLOWLIST`）；
 *  - 拒绝 `javascript:` / `data:` / `file:` / `blob:` / `vbscript:` / 协议相对
 *    （`//`）/ userinfo（`user:pass@`）/ 控制字符；
 *  - 新窗口一律 `noopener,noreferrer`；
 *  - 空/非法值显示错误，不导航。
 */
import { classifyExternalUrl, openExternalUrl } from '@/utils/urlPolicy'
import type { UrlDecision } from '@/utils/urlPolicy'

/** 支付自定义 scheme 白名单（微信/支付宝/银联/云闪付等）。 */
export const PAYMENT_SCHEME_ALLOWLIST = new Set([
  'weixin',
  'wechat',
  'wxpay',
  'wxx',
  'alipay',
  'alipays',
  'unionpay',
  'uppay',
  'cloudpay',
  'wappay'
])

export type PaymentUrlDecision = UrlDecision

/**
 * 纯函数：把任意支付 URL 分类为可安全导航的 web/scheme，或给出拒绝原因。
 * 任何非 ok 结果都不得进入 `location.href` / `window.open`。
 */
export function classifyPaymentUrl(value: unknown): PaymentUrlDecision {
  return classifyExternalUrl(value, { allowSchemes: PAYMENT_SCHEME_ALLOWLIST })
}

export interface OpenPaymentUrlOptions {
  /** 校验失败时的回调（由调用方弹 toast），不传则只 console.warn。 */
  onInvalid?: (reason: string) => void
}

/**
 * 校验后导航：scheme 用 `location.href` 拉起支付 App，web 用
 * `noopener,noreferrer` 新窗口。返回是否真的导航。
 */
export function openPaymentUrl(value: unknown, options: OpenPaymentUrlOptions = {}): boolean {
  const decision = classifyPaymentUrl(value)
  if (!decision.ok) {
    const reason = decision.reason || 'invalid'
    options.onInvalid?.(reason)
    console.warn('[PaymentUrl] blocked navigation:', reason)
    return false
  }
  const raw = String(value).trim()
  if (decision.kind === 'scheme') {
    window.location.href = raw
  } else {
    openExternalUrl(raw, { allowSchemes: PAYMENT_SCHEME_ALLOWLIST, onInvalid: options.onInvalid })
  }
  return true
}
