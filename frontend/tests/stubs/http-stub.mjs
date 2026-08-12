/**
 * 测试桩：替代 src/services/http.ts。
 * adminPath.ts 只 import `./http`，打包测试时用别名指向这里，
 * 避免拉进 element-plus / pinia / 全部 store 的级联。
 */
export const http = {
  post: async () => ({ data: {} }),
  get: async () => ({ data: {} }),
  request: async () => ({ data: {} })
}
