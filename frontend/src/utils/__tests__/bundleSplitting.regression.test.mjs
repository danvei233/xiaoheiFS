/**
 * 回归测试：按需加载回归防护（web_bundle_mobile F3/F4/F6）。
 *
 * 背景：
 *  - F3：v-highlight 用 `import('highlight.js')` 全量语言表（963kB/307kB gzip）；
 *  - F4：邮件设置页静态 import wangeditor（831kB/280kB gzip），进页即整块加载；
 *  - F6：安装门禁在全局 beforeEach 同步等待 install/status，阻塞每次整页加载首帧。
 *
 * 这些是源码级契约，用漂移断言锁定：
 * 运行：`node --test src/utils/__tests__/bundleSplitting.regression.test.mjs`
 */
import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const here = path.dirname(fileURLToPath(import.meta.url))
const srcRoot = path.resolve(here, '../../../src')

// F3：highlight 指令必须走 core + 白名单语言，禁止全量 import。
const highlight = fs.readFileSync(path.resolve(srcRoot, 'directives/business/highlight.ts'), 'utf8')
assert.ok(highlight.includes("import('highlight.js/lib/core')"), '必须用 highlight.js/lib/core')
assert.ok(
  !/= import\('highlight\.js'\)/.test(highlight),
  '禁止全量 import highlight.js（只允许 lib/core）'
)
assert.ok(highlight.includes('registerLanguage'), '必须按白名单注册语言')

// F4：邮件设置页不得静态 import wangeditor 组件（须 defineAsyncComponent 按需）。
const emailPage = fs.readFileSync(
  path.resolve(srcRoot, 'views/admin/settings/email/index.vue'),
  'utf8'
)
assert.ok(
  emailPage.includes('defineAsyncComponent'),
  '邮件模板编辑器必须异步加载（defineAsyncComponent）'
)
assert.ok(
  !emailPage.includes('import EmailTemplateEditor from'),
  '禁止静态 import 邮件模板编辑器（会整块拉 wangeditor）'
)

// F6：安装门禁必须持久化已安装状态（缓存命中不阻塞首帧）。
const installStore = fs.readFileSync(path.resolve(srcRoot, 'stores/install.ts'), 'utf8')
assert.ok(installStore.includes('install_status_cached'), '安装状态必须持久化缓存')
assert.ok(installStore.includes('readCachedInstalled'), '必须读取缓存判断是否已安装')

console.log('✅ bundleSplitting.regression.test.mjs —— 按需加载回归防护断言通过')
