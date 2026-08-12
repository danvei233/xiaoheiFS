/**
 * 支付表单 iframe 的 sandbox 合同（H3 / P2-74）。
 *
 * 移除无条件 `allow-top-navigation`：被攻陷/恶意的支付网关脚本可以在
 * 无用户交互的情况下把顶层窗口导航到钓鱼页。改为
 * `allow-top-navigation-by-user-activation`，只允许**用户激活后**的顶层
 * 导航（点击提交 / 表单提交）。绝大多数支付网关的收尾跳转都发生在用户
 * 点击"去支付"之后，满足该条件。
 *
 * 兼容边界：要求脚本在无用户激活时程序化改写顶层 URL 的网关会失败，
 * 支付失败提示错误、不跳转——这是可接受的安全取舍。
 *
 * 另外：无 `allow-same-origin`，iframe 内的脚本在孤立 origin 运行，
 * 读不到本应用 localStorage / token。
 */
export const PAYMENT_FORM_SANDBOX =
  'allow-forms allow-scripts allow-popups allow-top-navigation-by-user-activation'

/** 不允许出现的令牌：允许出现即违反安全合同。 */
export const PAYMENT_SANDBOX_FORBIDDEN = ['allow-top-navigation', 'allow-same-origin']

/** 验证 sandbox 合同：必需能力齐全且不含被禁令牌。 */
export function assertPaymentSandboxContract(tokens: string): boolean {
  const set = new Set(tokens.split(/\s+/).filter(Boolean))
  const required = ['allow-forms', 'allow-scripts']
  return (
    required.every((token) => set.has(token)) && PAYMENT_SANDBOX_FORBIDDEN.every((t) => !set.has(t))
  )
}
