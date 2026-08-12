import { http } from './http'

/**
 * 管理后台路径解析。
 *
 * 安全契约（独立复核 H1）：
 *
 * - **localStorage 只是路径提示**，不是验证权威。`admin_path_cache` 里存的路径
 *   可以被同源脚本任意伪造，之前把"localStorage 路径 + 时间戳"当作已验证
 *   是错误的——攻击者可同时伪造两项，24h TTL 没有任何安全意义。
 * - **每次页面启动 / 首次进入管理域都必须向后端验证一次**（`POST
 *   /check-admin-path`）。只有**本页面内由成功响应建立的内存态**才可以
 *   fast-path（同一 SPA 会话内重复导航到同一路径不再发请求）。
 * - **切换 admin_path 后旧 tab 下一次导航失败关闭**：`admin_path_epoch` 是
 *   单调递增的"路径代数"，改路径 / 清缓存时自增。旧 tab 内存里的验证代数
 *   与当前代数不一致 → 放弃 fast-path、回落后端复核 → 路径已被改掉则 404。
 *   不靠整页刷新来恢复。
 */

const HINT_KEY = 'admin_path_cache'
/** 旧版本写入的"验证时间戳"，已无安全意义，读取时忽略、清理时删除。 */
const LEGACY_VALIDATED_AT_KEY = 'admin_path_validated_at'
/** 路径代数：改 admin_path / 清缓存时自增，跨 tab 使旧验证失效。 */
const EPOCH_KEY = 'admin_path_epoch'

let cachedAdminPath: string | null = null
let isValidated = false
/** 上一次验证成功时的路径代数（内存态），与 localStorage 代数比对判断是否过期 */
let validatedEpoch = 0

function normalizePath(path: string): string {
  return String(path || '')
    .trim()
    .replace(/^\/+|\/+$/g, '')
}

/** 只读路径提示；不存在时返回 null（不兜底默认值，避免把伪造值当权威）。 */
function loadAdminPathFromStorage(): string | null {
  try {
    const path = normalizePath(localStorage.getItem(HINT_KEY) || '')
    return path || null
  } catch {
    return null
  }
}

/** 写路径提示。提示不携带任何"已验证"语义。 */
export function setAdminPathHint(path: string): void {
  const normalized = normalizePath(path)
  if (!normalized) return
  try {
    localStorage.setItem(HINT_KEY, normalized)
  } catch {
    // ignore
  }
}

function readEpoch(): number {
  try {
    return Number(localStorage.getItem(EPOCH_KEY) || '0') || 0
  } catch {
    return 0
  }
}

function advanceEpoch(): number {
  const next = readEpoch() + 1
  try {
    localStorage.setItem(EPOCH_KEY, String(next))
  } catch {
    // ignore
  }
  return next
}

/**
 * 校验路径首段是否为配置的管理后台路径。
 *
 * fast-path 只在内存态存在、路径匹配且代数一致时生效——这三个条件全部由
 * 本页面内成功的后端响应建立，localStorage 无法伪造。其余情况一律回落后端。
 */
export async function checkAdminPath(
  path: string
): Promise<{ isAdmin: boolean; adminPath: string }> {
  const normalized = normalizePath(path)
  const hint = loadAdminPathFromStorage()

  if (isValidated && cachedAdminPath && normalized === cachedAdminPath) {
    // 代数不一致 = 别处改过 admin_path / 清过缓存，内存态作废，必须复核。
    if (validatedEpoch === readEpoch()) {
      return { isAdmin: true, adminPath: cachedAdminPath }
    }
  }

  if (!normalized) {
    return { isAdmin: false, adminPath: cachedAdminPath || hint || 'admin' }
  }

  try {
    const res = await http.post<{ is_admin: boolean }>('/api/v1/check-admin-path', {
      path: normalized
    })
    if (res.data?.is_admin) {
      cachedAdminPath = normalized
      isValidated = true
      validatedEpoch = readEpoch()
      setAdminPathHint(normalized)
      return { isAdmin: true, adminPath: normalized }
    }
    // 后端明确否认该路径是管理后台：若它正是当前内存验证过的路径，立即作废
    // 内存验证态。这样即便后端改 admin_path 时没有走到 clearAdminPathCache
    // （直接改库/换实例），本会话内旧路径的 fast-path 也不会继续放行。
    if (cachedAdminPath === normalized) {
      isValidated = false
      cachedAdminPath = null
    }
  } catch (error) {
    // 只记录 message，不展开 axios 错误对象——它的 config.headers 里带着
    // Authorization: Bearer <token>（m051 F4）。网络错误保持 fail-closed。
    console.error(
      'Failed to check admin path:',
      error instanceof Error ? error.message : 'unknown error'
    )
  }

  // 失败/非管理域：不建立内存验证态（fast-path 只认成功响应）。
  return { isAdmin: false, adminPath: cachedAdminPath || hint || 'admin' }
}

export function getCachedAdminPath(): string {
  if (!cachedAdminPath) {
    cachedAdminPath = loadAdminPathFromStorage()
  }
  return cachedAdminPath || 'admin'
}

export function clearAdminPathCache(): void {
  cachedAdminPath = null
  isValidated = false
  advanceEpoch()
  try {
    localStorage.removeItem(HINT_KEY)
    localStorage.removeItem(LEGACY_VALIDATED_AT_KEY)
  } catch {
    // ignore
  }
}

export async function fetchAdminPath(): Promise<string> {
  return getCachedAdminPath()
}

export function buildAdminUrl(subPath: string = ''): string {
  const adminPath = getCachedAdminPath()
  const cleanSubPath = subPath.replace(/^\/+/, '')
  return cleanSubPath ? `/${adminPath}/${cleanSubPath}` : `/${adminPath}`
}

export async function navigateToAdminLogin(router: any, redirect?: string): Promise<void> {
  const adminPath = await fetchAdminPath()
  const query = redirect ? { redirect } : {}
  await router.push({ path: `/${adminPath}/login`, query })
}
