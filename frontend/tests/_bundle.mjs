/**
 * 测试共享打包助手。
 *
 * 项目没有 vitest/jest（AGENTS.md 已说明），回归测试统一用
 * `node --test` + esbuild 把**真实生产源码**打包成 node ESM 后执行，
 * 避免静态假断言。这个文件只是把 esbuild 的公共配置抽出来，防止每个
 * 测试文件重复 20 行样板。
 *
 * 用法：
 *   import { bundleEntry } from '../../tests/_bundle.mjs'
 *   const mod = await bundleEntry(`export { foo } from './src/...'`, import.meta.url)
 */
import { build } from 'esbuild'
import path from 'node:path'
import { writeFileSync, mkdtempSync, mkdirSync } from 'node:fs'
import { pathToFileURL, fileURLToPath } from 'node:url'

/**
 * 把一段入口源码（通常是一个 re-export）打包并动态 import。
 *
 * @param {string} entry      入口 TS/JS 源码，例如 `export { x } from './src/utils/foo'`
 * @param {string} importMetaUrl 调用方测试文件的 import.meta.url，用于定位 resolveDir
 * @param {object} [extra]    覆盖项：extra.alias / extra.external / extra.loader
 */
export async function bundleEntry(entry, importMetaUrl, extra = {}) {
  const callerDir = path.dirname(fileURLToPath(importMetaUrl))
  // 本 helper 位于 frontend/tests/_bundle.mjs → frontend 根就是其上一级目录。
  // `@` 别名必须以 frontend 根为基准解析（等价 vite.config.ts 的 alias）。
  const frontendRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
  const result = await build({
    stdin: {
      contents: entry,
      resolveDir: callerDir,
      sourcefile: 'bundle.ts'
    },
    bundle: true,
    platform: 'node',
    format: 'esm',
    write: false,
    alias: {
      '@': path.resolve(frontendRoot, 'src'),
      '@utils': path.resolve(frontendRoot, 'src/utils'),
      ...(extra.alias ?? {})
    },
    loader: {
      '.png': 'dataurl',
      '.jpg': 'dataurl',
      '.webp': 'dataurl',
      '.gif': 'dataurl',
      '.svg': 'dataurl',
      '.css': 'empty',
      ...(extra.loader ?? {})
    },
    external: ['axios', ...(extra.external ?? [])],
    plugins: extra.plugins ?? [],
    define: {
      'import.meta.env': '{}',
      ...(extra.define ?? {})
    }
  })
  // 临时产物写到 frontend/.vite-cache/tests/（已被 .gitignore 忽略），
  // 这样打包后的 bundle 从该目录向上能找到 frontend/node_modules，
  // 外部化的 axios 等依赖在运行时才能被 Node 解析到。
  const cacheDir = path.resolve(frontendRoot, '.vite-cache/tests')
  mkdirSync(cacheDir, { recursive: true })
  const dir = mkdtempSync(path.join(cacheDir, 'xh-test-'))
  const file = path.join(dir, 'bundle.mjs')
  writeFileSync(file, result.outputFiles[0].text)
  return import(pathToFileURL(file).href)
}
