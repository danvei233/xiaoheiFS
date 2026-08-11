/**
 * 模拟登录（/console#impersonate_token=xxx）的解析与受理判定。
 *
 * 独立成模块是为了脱离浏览器环境做单元测试（见 __tests__/impersonation.test.mjs）。
 * 守卫里的 handleImpersonation 只负责拿到判定结果后的副作用：
 * 写 store / localStorage、把 token 从地址栏抹掉、重新导航。
 *
 * @module router/guards/impersonation
 */

export interface ImpersonationParseResult {
  /** hash 里 impersonate_token 的原始值 */
  token: string
  /** 抹掉 impersonate_token 后剩余的 hash（保留其他参数），无则空字符串 */
  restHash: string
  /** 是否应该把 token 写入用户会话 */
  shouldWrite: boolean
}

/**
 * 解析 hash 里的模拟登录载荷并判定是否受理。
 *
 * URL 里的 token 没有服务端签名，任何链接都能携带任意 token；模拟登录的合法
 * 发起方是管理员（Flutter 移动管理端或 web 后台），所以要求浏览器里已经存在
 * 管理会话（hasAdminSession）才写入 token；否则只把 token 从地址栏抹掉、
 * 按正常导航处理（未登录用户会自然走到登录页），避免会话被任意链接覆盖。
 *
 * @param hash           例如 '#impersonate_token=abc&foo=1'
 * @param hasAdminSession 当前浏览器是否已登录管理后台
 * @returns hash 里没有 impersonate_token 时返回 null，守卫继续正常流程
 */
export function parseImpersonation(
  hash: string,
  hasAdminSession: boolean
): ImpersonationParseResult | null {
  if (!hash.includes('impersonate_token=')) return null

  const params = new URLSearchParams(hash.replace(/^#/, ''))
  const token = params.get('impersonate_token') || ''

  params.delete('impersonate_token')
  const rest = params.toString()

  return {
    token,
    restHash: rest ? `#${rest}` : '',
    shouldWrite: Boolean(token) && hasAdminSession
  }
}
