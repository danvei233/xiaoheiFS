/**
 * 回归测试：ri 图标离线注册（web_bundle_mobile F2）。
 *
 * 背景：`ArtSvgIcon` 用 `@iconify/vue` 渲染 `ri:*` 图标，未注册离线集合时
 *   运行时从 `api.iconify.design` 拉取——内网/离线生产图标全空白。修复：把
 *   实际用到的 ri 图标子集提交为 `src/assets/iconify/ri-subset.json` 并注册。
 *
 * 运行：`node --test src/utils/__tests__/iconifyOffline.regression.test.mjs`
 */
import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { execSync } from 'node:child_process'

const here = path.dirname(fileURLToPath(import.meta.url))
const srcRoot = path.resolve(here, '../../../src')

// 1. main.ts 必须引入离线加载器
const mainSource = fs.readFileSync(path.resolve(srcRoot, 'main.ts'), 'utf8')
assert.ok(
  mainSource.includes("import '@/utils/ui/iconify-loader'"),
  'main.ts 必须引入 iconify-loader（否则离线集合不注册）'
)

// 2. 加载器必须注册子集
const loader = fs.readFileSync(path.resolve(srcRoot, 'utils/ui/iconify-loader.ts'), 'utf8')
assert.ok(loader.includes('addCollection'), '加载器必须调用 addCollection 注册')
assert.ok(loader.includes('ri-subset.json'), '加载器必须引用已提交的 ri 子集')

// 3. 子集必须覆盖源码里实际用到的 ri:* 图标。
//    注意：个别引用在 ri 集合中本就不存在（layers-line / folder-search-line），
//    它们即使走 CDN 也渲染空白，属于既有命名问题；子集覆盖的是 ri 集合里
//    真实存在的图标。
const subset = JSON.parse(
  fs.readFileSync(path.resolve(srcRoot, 'assets/iconify/ri-subset.json'), 'utf8')
)
const used = new Set()
for (const line of execSync(
  `grep -rhoP 'ri:[a-z0-9-]+' ${srcRoot} --include='*.vue' --include='*.ts'`,
  { maxBuffer: 10 * 1024 * 1024 }
)
  .toString()
  .trim()
  .split('\n')) {
  const m = line.trim().match(/^ri:(.+)$/)
  if (m) used.add(m[1])
}
const KNOWN_NOT_IN_RI = new Set(['layers-line', 'folder-search-line'])
const missing = [...used].filter((name) => !KNOWN_NOT_IN_RI.has(name) && !subset.icons[name])
assert.deepEqual(missing, [], `ri 子集必须覆盖全部 ri 集合内已使用图标，缺: ${missing.join(', ')}`)

console.log(`✅ iconifyOffline.regression.test.mjs —— ${used.size} 个 ri 图标离线覆盖断言通过`)
