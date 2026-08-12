/**
 * 回归测试：CSV 转义（P3-39 / m062 L1）。
 *
 * 背景：旧 escapeCsvCell 只做双引号包裹，`=1+1` / `+cmd` / `@SUM(A1)` 这类
 * 以公式字符开头的单元格会被 Excel/Sheets 当作公式执行（CSV 公式注入）。
 * 修复：前置单引号中和 + 维持标准引号规则。
 *
 * 运行：`node --test src/utils/__tests__/csv.regression.test.mjs`
 */
import assert from 'node:assert/strict'

// 直接导入纯函数（无依赖链，无需 esbuild）。
// 运行方式：node --experimental-strip-types --test（Node 22 直接加载 .ts）。
const { escapeCsvCell: esc, toCsvLine: line } = await import('../../utils/csv.ts')

// ── 公式注入中和 ───────────────────────────────────────────────────────
assert.equal(esc('=1+1'), `"'=1+1"`, '以 = 开头的单元格必须中和')
assert.equal(esc('+cmd'), `"'+cmd"`, '以 + 开头必须中和')
assert.equal(esc('-1+2'), `"'-1+2"`, '以 - 开头必须中和')
assert.equal(esc('@SUM(A1)'), `"'@SUM(A1)"`, '以 @ 开头必须中和')
assert.equal(esc('\tTAB'), `"'\tTAB"`, '以 tab 开头必须中和')
assert.equal(esc('\rCR'), `"'\rCR"`, '以 CR 开头必须中和')

// ── 正常值不受影响（只加双引号）────────────────────────────────────────
assert.equal(esc('hello'), `"hello"`, '普通字符串只加引号')
assert.equal(esc("O'Brien"), `"O'Brien"`, '单引号不翻倍（CSV 只翻倍双引号）')
assert.equal(esc('say "hi"'), `"say ""hi"""`, '内部双引号按标准翻倍')
assert.equal(esc(42), `"42"`, '数字转字符串')
assert.equal(esc(null), `""`, 'null 转空单元格')
assert.equal(esc(undefined), `""`, 'undefined 转空单元格')

// ── toCsvLine 组合 ────────────────────────────────────────────────────
assert.equal(line(['a', '=x', 1]), `"a","'=x","1"`, '整行组合并逐格中和')

// ── VPS 导出调用点漂移防护（web_url_content F2）────────────────────────
// 控制台 VPS 列表的 name 来自自动化插件返回的 hostname，第三方插件可构造
// 恶意主机名（如 "=HYPERLINK(...)"），导出必须经共享 toCsvLine 中和，不得
// 再用 JSON.stringify 裸拼。
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
const __dirname = path.dirname(fileURLToPath(import.meta.url))
const vpsPage = readFileSync(path.resolve(__dirname, '../../views/console/vps/index.vue'), 'utf8')
assert.ok(vpsPage.includes('toCsvLine('), 'VPS 导出必须引用共享 toCsvLine 中和公式注入')
assert.ok(!/JSON\.stringify\(value/.test(vpsPage), 'VPS 导出不得再用 JSON.stringify 裸拼 CSV')
assert.equal(
  line(['id', '=HYPERLINK("http://evil","x")', 'active', '2026-12-01']),
  `"id","'=HYPERLINK(""http://evil"",""x"")","active","2026-12-01"`,
  'VPS 行含 = 前缀 name 时整行被中和'
)

console.log('✅ csv.regression.test.mjs —— 公式注入中和断言通过')
