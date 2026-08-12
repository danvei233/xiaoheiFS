/**
 * 离线图标加载器
 *
 * 用于在内网环境下支持 Iconify 图标的离线加载。
 * 通过预加载图标集数据，避免运行时从 CDN 获取图标。
 *
 * web_bundle_mobile F2：全部 `ri:*` 图标此前依赖 `@iconify/vue` 默认的
 * `https://api.iconify.design` 运行时拉取——离线/内网/受限网络生产图标全空白，
 * 且把图标名发给第三方 CDN。这里用**已提交的 ri 子集**（`src/assets/iconify/
 * ri-subset.json`，只含当前实际用到的 166 个图标，约 48KB）注册到 Iconify
 * 离线集合；未收录的新图标仍回退 CDN。
 *
 * 用法：在 main.ts 里 `import '@/utils/ui/iconify-loader'`。
 */
import { addCollection } from '@iconify/vue'
import riIcons from '@/assets/iconify/ri-subset.json'

// 注册离线图标集（remix icon 子集）
addCollection(riIcons as never)

export {}
