/**
 * 系统管理相关工具函数统一导出
 *
 * P3-49：不再 re-export './upgrade'。systemUpgrade() 全仓零调用方，却静态
 * import `src/mock/upgrade/changeLog.ts` 的模板演示升级日志；移除后该 mock
 * 不再进入生产包。mittBus 仍在被多个布局组件消费。
 */

export { default as mittBus } from './mittBus'
