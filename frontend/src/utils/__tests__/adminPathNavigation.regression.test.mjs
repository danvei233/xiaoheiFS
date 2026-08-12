/**
 * 回归测试：安装完成页"进入后台"的路径解析（web_navigation F3）。
 *
 * 背景：`goAdmin` 原来把 `localStorage.getItem('admin_path_cache')` 直接拼进
 *   `window.location.replace('/<path>/login')`。admin_path_cache 是同源可写的
 *   localStorage，被写成 `//evil.com` 时 `new URL('///evil.com/login', origin)`
 *   解析为 `https://evil.com/login`——浏览器真实离开站点。
 *
 * 修复：只接受与后端 ValidateAdminPath 一致的 `[a-zA-Z0-9]+` 单段路径；
 *   优先安装流程受控的 props.adminPath，localStorage 仅兜底且过同一校验。
 *
 * 运行：`node --test src/utils/__tests__/adminPathNavigation.regression.test.mjs`
 */
import { bundleEntry } from '../../../tests/_bundle.mjs'
import assert from 'node:assert/strict'

const { resolveInstallAdminPath, ADMIN_PATH_SEGMENT_RE } = await bundleEntry(
  `export { resolveInstallAdminPath, ADMIN_PATH_SEGMENT_RE } from '@/utils/navigation/adminPath'`,
  import.meta.url
)

// ── 优先 props.adminPath（安装流程受控值）──────────────────────────────
assert.equal(resolveInstallAdminPath('myadmin', '//evil.com'), 'myadmin', '受控值优先')
assert.equal(resolveInstallAdminPath('  ops  ', '//evil.com'), 'ops', 'props 值归一化去空白')
assert.equal(resolveInstallAdminPath('/panel/', '//evil.com'), 'panel', 'props 值去首尾斜杠')

// ── props 缺失时用 localStorage 兜底，但必须过同一校验 ─────────────────
assert.equal(resolveInstallAdminPath('', 'control'), 'control', '合法 hint 兜底')
assert.equal(resolveInstallAdminPath('', '/manage/'), 'manage', 'hint 去斜杠')

// ── 危险 localStorage 值不得被导航（核心：不跳外部 origin）──────────────
assert.equal(resolveInstallAdminPath('', '//evil.com'), 'admin', '协议相对 hint 拒绝→回退 admin')
assert.equal(resolveInstallAdminPath('', 'https://evil.com'), 'admin', '绝对 URL hint 拒绝')
assert.equal(resolveInstallAdminPath('', 'javascript:alert(1)'), 'admin', 'javascript: hint 拒绝')
assert.equal(resolveInstallAdminPath('', 'admin/login'), 'admin', '多段路径拒绝')
assert.equal(resolveInstallAdminPath('', 'adm in'), 'admin', '含空白拒绝')
assert.equal(resolveInstallAdminPath('', 'admin$x'), 'admin', '特殊字符拒绝')
assert.equal(resolveInstallAdminPath('', ''), 'admin', '空 hint 回退默认 admin')

// ── 校验结果的导航目标本身必须是站内相对单段（不会整页离开）───────────
// 模拟 goAdmin 的拼接：目标必须是 `/<segment>/login`，segment 只含 [a-zA-Z0-9]。
for (const hint of ['//evil.com', 'https://evil.com', 'javascript:alert(1)', 'myadmin']) {
  const target = resolveInstallAdminPath('', hint)
  assert.ok(ADMIN_PATH_SEGMENT_RE.test(target), `目标段 ${target} 必须匹配 [a-zA-Z0-9]+`)
  const href = `/${target}/login`
  assert.ok(href.startsWith('/'), '导航目标必须是站内相对路径')
  assert.equal(
    new URL(href, 'https://cloud.example.com').origin,
    'https://cloud.example.com',
    '导航不离开站点'
  )
}

console.log('✅ adminPathNavigation.regression.test.mjs —— 安装完成页后台导航安全断言通过')
