/**
 * 回归测试：模拟登录（/console#impersonate_token=xxx）的受理判定。
 *
 * 背景（第二遍审查 m052 / F2）：
 *   handleImpersonation 原先对任何 /console 导航，只要 hash 里出现
 *   impersonate_token= 就无条件写入 localStorage.user_token 并覆盖当前会话。
 *   该 token 没有服务端签名，任何攻击者注册一个账号拿到自己的 token 就能
 *   构造链接，受害者点击后浏览器被切换进攻击者账户（会话固定/登录 CSRF）。
 *
 * 修复：把受理判定抽成纯函数 parseImpersonation，并要求浏览器里已经存在
 *   管理后台会话（模拟登录的合法发起方是管理员）才写入 token；否则只把
 *   token 从地址栏抹掉、按正常导航处理。
 *
 * 运行：`node src/router/__tests__/impersonation.test.mjs`
 *   （前端暂无 vitest/jest，见 AGENTS.md；用 esbuild 打包真实源码执行。）
 */

import { build } from 'esbuild'
import assert from 'node:assert/strict'
import path from 'node:path'
import { writeFileSync, mkdtempSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { pathToFileURL } from 'node:url'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const routerDir = path.resolve(__dirname, '../')

async function loadImpersonationModule() {
  const result = await build({
    stdin: {
      contents: `export { parseImpersonation } from './guards/impersonation'`,
      resolveDir: routerDir,
      sourcefile: 'bundle.ts'
    },
    bundle: true,
    platform: 'node',
    format: 'esm',
    write: false
  })
  const dir = mkdtempSync(path.join(tmpdir(), 'm052-'))
  const file = path.join(dir, 'bundle.mjs')
  writeFileSync(file, result.outputFiles[0].text)
  return import(pathToFileURL(file).href)
}

const { parseImpersonation } = await loadImpersonationModule()

// ── 核心安全契约：没有管理会话时不得写入 token ──────────────────────────
const noAdmin = parseImpersonation('#impersonate_token=attacker-token', false)
assert.ok(noAdmin, 'hash 含 impersonate_token 时应返回解析结果（用于清掉地址栏 token）')
assert.equal(noAdmin.token, 'attacker-token')
assert.equal(noAdmin.shouldWrite, false, '无管理会话时禁止把 token 写入用户会话')

// 管理会话存在（合法管理员发起）时才允许写入
const withAdmin = parseImpersonation('#impersonate_token=attacker-token', true)
assert.equal(withAdmin.shouldWrite, true, '存在管理会话时允许写入')

// ── 无关 hash 不受影响 ──────────────────────────────────────────────────
assert.equal(
  parseImpersonation('#some-other=1', false),
  null,
  '没有 impersonate_token 的 hash 返回 null，守卫继续正常流程'
)

// ── 抹掉 token 后保留其余 hash，避免 token 留在地址栏/历史记录 ───────────
const rest = parseImpersonation('#impersonate_token=abc&foo=1&bar=2', false)
assert.equal(rest.restHash, '#foo=1&bar=2', '应删除 impersonate_token 并保留其他参数')

const rest2 = parseImpersonation('#impersonate_token=abc', false)
assert.equal(rest2.restHash, '', 'hash 里只剩 token 时抹成空字符串')

// ── 空 token 边界：不清写，但仍要把参数抹掉 ─────────────────────────────
const empty = parseImpersonation('#impersonate_token=&foo=1', true)
assert.equal(empty.token, '')
assert.equal(empty.shouldWrite, false, '空 token 不能写入')
assert.equal(empty.restHash, '#foo=1')

console.log('✅ impersonation.test.mjs —— F2 全部断言通过')
