/**
 * 安装完成页"进入后台"目标路径解析（web_navigation F3）。
 *
 * `admin_path_cache` 是同源可写的 localStorage。直接拼进
 * `window.location.replace('/<path>/login')` 时，被写成 `//evil.com` 会按 URL
 * 规范解析成外部 origin（`new URL('///evil.com/login', origin)` 在浏览器与 Node
 * 中都是 `https://evil.com/login`），浏览器真实离开站点——与 H1"localStorage 只是
 * 路径提示、不作导航权威"的原则相悖。
 *
 * 这里只接受与后端 `ValidateAdminPath` 一致的 `[a-zA-Z0-9]+` 单段路径；
 * 优先使用安装流程受控的 `props.adminPath`，localStorage 仅作兜底且必须过同一
 * 校验，非法一律回退 'admin'。纯函数，便于回归测试。
 */
export const ADMIN_PATH_SEGMENT_RE = /^[a-zA-Z0-9]+$/

const normalizeSegment = (value: string): string =>
  String(value || '')
    .trim()
    .replace(/^\/+|\/+$/g, '')

export function resolveInstallAdminPath(preferred: string, cachedHint: string): string {
  const preferredPath = normalizeSegment(preferred)
  if (preferredPath && ADMIN_PATH_SEGMENT_RE.test(preferredPath)) return preferredPath
  const hint = normalizeSegment(cachedHint)
  if (hint && ADMIN_PATH_SEGMENT_RE.test(hint)) return hint
  return 'admin'
}
