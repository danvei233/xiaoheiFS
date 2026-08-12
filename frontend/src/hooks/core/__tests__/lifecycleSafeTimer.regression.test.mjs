/**
 * 回归测试：useLifecycleSafeTimer —— keep-alive 生命周期安全的定时器
 * （web_refresh_perf F1）。
 *
 * 背景：keep-alive 页面切后台不卸载、onBeforeUnmount 不触发，裸 setTimeout
 *   轮询在缓存里无限续跑（后台请求风暴）。组合式函数在 onDeactivated 停表、
 *   onActivated 恢复，并自愈式地在 async fn settle 后排下一拍。
 *
 * 运行：`node --test src/hooks/core/__tests__/lifecycleSafeTimer.regression.test.mjs`
 */
import { bundleEntry } from '../../../../tests/_bundle.mjs'
import assert from 'node:assert/strict'

// vue 的生命周期钩子 stub 成记录器，便于手动触发 onDeactivated / onActivated。
const lifecycle = { mounted: [], deactivated: [], activated: [], unmounted: [] }
const vueStub = {
  name: 'stub-vue',
  setup(build) {
    build.onResolve({ filter: /^vue$/ }, () => ({ path: 'vue-stub', namespace: 'vue' }))
    build.onLoad({ filter: /.*/, namespace: 'vue' }, () => ({
      contents: `
        export const onMounted = (fn) => { globalThis.__lifecycle.mounted.push(fn) }
        export const onDeactivated = (fn) => { globalThis.__lifecycle.deactivated.push(fn) }
        export const onActivated = (fn) => { globalThis.__lifecycle.activated.push(fn) }
        export const onBeforeUnmount = (fn) => { globalThis.__lifecycle.unmounted.push(fn) }
      `,
      loader: 'js'
    }))
  }
}

globalThis.__lifecycle = lifecycle
globalThis.__docHidden = false
globalThis.document = {
  get hidden() {
    return globalThis.__docHidden
  }
}

const mod = await bundleEntry(
  `export { useLifecycleSafeTimer } from '@/hooks/core/useLifecycleSafeTimer'`,
  import.meta.url,
  { plugins: [vueStub] }
)

const { useLifecycleSafeTimer } = mod

// 定时器捕获：把 setTimeout 换成 push 到数组，手动驱动 tick。
let timers = []
const nativeSetTimeout = globalThis.setTimeout
const nativeClearTimeout = globalThis.clearTimeout
globalThis.setTimeout = (fn, ms) => {
  timers.push({ fn, ms })
  return timers.length
}
globalThis.clearTimeout = () => {}

const resetTimers = () => {
  timers = []
}
globalThis.__tickCalls = 0

// ── 场景 1：deactivate 停表、activate 恢复、unmount/cancel 销毁 ──────────
{
  globalThis.__tickCalls = 0
  const { schedule, cancel } = useLifecycleSafeTimer(() => {
    globalThis.__tickCalls++
  }, 1000)

  assert.equal(lifecycle.mounted.length, 1, '注册 onMounted')
  assert.equal(lifecycle.deactivated.length, 1, '注册 onDeactivated')
  assert.equal(lifecycle.activated.length, 1, '注册 onActivated')
  assert.equal(lifecycle.unmounted.length, 1, '注册 onBeforeUnmount')

  resetTimers()
  schedule()
  assert.equal(timers.length, 1, 'schedule 排定下一拍')

  // 触发 tick → 同步 fn 立即重排
  timers.shift().fn()
  assert.equal(globalThis.__tickCalls, 1, 'tick 执行 fn')
  assert.equal(timers.length, 1, '同步 fn 后重排下一拍')

  // deactivate → 停表：清定时器，之后 schedule 不再生效
  resetTimers()
  lifecycle.deactivated[0]()
  schedule()
  assert.equal(timers.length, 0, 'deactivate 后 schedule 不再排定（后台不请求）')

  // activate → 恢复运行，可再排定
  lifecycle.activated[0]()
  schedule()
  assert.equal(timers.length, 1, 'activate 后恢复可排定')

  // cancel → 立即停
  resetTimers()
  cancel()
  assert.equal(timers.length, 0, 'cancel 立即清定时器')

  // unmount → 销毁，之后 schedule 不再生效
  lifecycle.unmounted[0]()
  resetTimers()
  schedule()
  assert.equal(timers.length, 0, 'unmount 后不可再排定')
}

// ── 场景 2：异步 fn 自愈式轮询（settle 后排下一拍，天然防重叠）────────
{
  globalThis.__tickCalls = 0
  let release
  const gate = new Promise((resolve) => (release = resolve))
  const timer = useLifecycleSafeTimer(() => {
    globalThis.__tickCalls++
    return gate
  }, 1000)

  resetTimers()
  timer.schedule()
  timers.shift().fn() // 触发 tick，fn 返回 pending promise
  assert.equal(globalThis.__tickCalls, 1, 'async fn 被调用')
  assert.equal(timers.length, 0, 'async fn 在途时不重排（防重叠）')

  release() // settle
  await gate
  await Promise.resolve()
  await Promise.resolve()
  assert.ok(timers.length >= 1, 'async fn settle 后重排下一拍')
  timer.cancel()
}

// ── 场景 3：document.hidden 时暂停（兜底浏览器切走/最小化）────────────
{
  globalThis.__tickCalls = 0
  const timer = useLifecycleSafeTimer(() => {
    globalThis.__tickCalls++
  }, 1000)

  resetTimers()
  timer.schedule()
  globalThis.__docHidden = true
  timers.shift().fn() // tick 发现 hidden → 重排，不执行 fn
  assert.equal(globalThis.__tickCalls, 0, 'document.hidden 时 tick 不执行 fn')
  assert.equal(timers.length, 1, 'hidden 时重排（可见后继续）')
  globalThis.__docHidden = false
  timers.shift().fn()
  assert.equal(globalThis.__tickCalls, 1, '可见时 tick 执行 fn')
  timer.cancel()
}

globalThis.setTimeout = nativeSetTimeout
globalThis.clearTimeout = nativeClearTimeout
console.log('✅ lifecycleSafeTimer.regression.test.mjs —— keep-alive 生命周期安全定时器断言通过')
