/**
 * VPS 访问信息（IP/凭据）的公共解析。
 *
 * P2-4 跨层回归：后端移除 AccessInfo 后，列表接口的 access_info 恒为 null，
 * 但 VPS 记录顶层仍有 ip 字段。解析 IP 时必须优先读 row.ip，再回退到
 * access_info 的 remote_ip，避免用户控制台 VPS 列表的 IP 列恒显 "-"。
 */

export interface VpsAccessInfoLike {
  remote_ip?: unknown
  ip?: unknown
  public_ip?: unknown
  ipv4?: unknown
  Ip?: unknown
  [key: string]: unknown
}

function isJsonObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

/** 从 VPS 记录提取 access_info（兼容 JSON 字符串 / 对象 / snake_case / PascalCase）。 */
export function parseVpsAccessInfo(row: {
  access_info?: unknown
  AccessInfo?: unknown
  access_info_json?: unknown
  AccessInfoJSON?: unknown
}): VpsAccessInfoLike {
  const raw = row.access_info ?? row.AccessInfo ?? row.access_info_json ?? row.AccessInfoJSON
  if (isJsonObject(raw)) return raw as VpsAccessInfoLike
  if (typeof raw === 'string') {
    try {
      const parsed: unknown = JSON.parse(raw)
      if (isJsonObject(parsed)) return parsed as VpsAccessInfoLike
    } catch {
      /* 非法 JSON 视为无 access_info */
    }
  }
  return {}
}

/** 解析 VPS 记录展示用 IP：优先顶层 row.ip，再回退 access_info 的 remote_ip。 */
export function resolveVpsIp(row: {
  ip?: unknown
  access_info?: unknown
  AccessInfo?: unknown
  access_info_json?: unknown
  AccessInfoJSON?: unknown
}): string {
  const access = parseVpsAccessInfo(row)
  return String(
    row.ip ?? access.remote_ip ?? access.ip ?? access.public_ip ?? access.ipv4 ?? access.Ip ?? '-'
  )
}
