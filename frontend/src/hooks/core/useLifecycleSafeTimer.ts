/**
 * keep-alive 生命周期安全的定时器组合式函数（web_refresh_perf F1）。
 *
 * 背景：路由 `meta.keepAlive: true` 的页面切到后台时组件不会卸载（被
 *   `<KeepAlive>` 缓存），`onBeforeUnmount` 不触发；如果页面只在 unmount
 *   清理 setInterval/setTimeout，定时器会在后台无限续跑，构成后台请求风暴
 *   （探针列表 10s 轮询、订单审核抽屉 5s 轮询叠加后 ~3600 请求/小时）。
 *
 * 这里统一在 `onDeactivated` 清定时器、`onActivated` 重排，并且默认在
 * `document.hidden`（整页隐藏）时也暂停，页面可见时恢复。
 *
 * 用法：
 *   const { schedule, cancel } = useLifecycleSafeTimer(() => refreshSoft(), 10000)
 *   onMounted(() => schedule())
 */
import { onActivated, onBeforeUnmount, onDeactivated, onMounted } from 'vue'

export interface LifecycleSafeTimer {
  /** 启动/重排下一次执行；已在运行则复位计时。 */
  schedule: () => void
  /** 停止并清空计时。 */
  cancel: () => void
  /** 当前是否处于可见且未取消状态（用于在途保护判断）。 */
  isActive: () => boolean
}

export function useLifecycleSafeTimer(
  fn: () => void,
  intervalMs: number,
  options: { pauseWhenHidden?: boolean } = {}
): LifecycleSafeTimer {
  const pauseWhenHidden = options.pauseWhenHidden ?? true
  let timer: ReturnType<typeof setTimeout> | null = null
  let disposed = false
  // 组合式函数在页面 setup 阶段创建，页面此时已挂载；默认 running=true。
  let running = true

  const clearTimer = () => {
    if (timer !== null) {
      clearTimeout(timer)
      timer = null
    }
  }

  const tick = () => {
    if (disposed || !running) return
    // 整页隐藏时暂停，避免后台标签页持续发请求（keep-alive 页面切 tab 时
    // onDeactivated 已停表，这里兜底浏览器切走/最小化）。
    if (pauseWhenHidden && typeof document !== 'undefined' && document.hidden) {
      schedule()
      return
    }
    let result: unknown
    try {
      result = fn()
    } catch {
      schedule()
      return
    }
    // 自愈式轮询：fn 返回 Promise 时等它 settle 后再排下一拍，天然防重叠；
    // 同步 fn 则立即重排。
    if (result && typeof (result as Promise<void>).then === 'function') {
      Promise.resolve(result)
        .catch(() => undefined)
        .finally(() => {
          if (!disposed) schedule()
        })
    } else {
      schedule()
    }
  }

  const schedule = () => {
    // deactivated/unmount 后 running=false：schedule 不再排定（后台不请求）。
    if (disposed || !running) return
    clearTimer()
    timer = setTimeout(tick, intervalMs)
  }

  const cancel = () => {
    running = false
    clearTimer()
  }

  onMounted(() => {
    running = true
  })
  // keep-alive 页面切后台：必须停表，否则定时器在缓存里无限续跑（F1）。
  onDeactivated(() => {
    running = false
    clearTimer()
  })
  // keep-alive 页面切回前台：恢复轮询（无自启动语义，由调用方在 onActivated
  // 里调 schedule 决定是否继续）。
  onActivated(() => {
    if (!disposed) running = true
  })
  onBeforeUnmount(() => {
    disposed = true
    running = false
    clearTimer()
  })

  return { schedule, cancel, isActive: () => running }
}
