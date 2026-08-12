import { createApp } from 'vue'
import App from './App.vue'

import { initStore } from './store'
import { initRouter } from './router'
import { setupGlobDirectives } from './directives'
import { setupErrorHandle } from './utils/sys/error-handle'
import { initializeTheme } from './hooks/core/useTheme'
import language from './locales'

import '@styles/core/tailwind.css'
import '@styles/index.scss'
import '@utils/sys/console'

// web_bundle_mobile F2：注册 ri 图标离线集合，内网/离线生产不再依赖
// api.iconify.design 运行时拉取（图标不再空白）。
import '@/utils/ui/iconify-loader'

// iOS 上阻止双击缩放需要非 passive 监听
document.addEventListener('touchstart', function () {}, { passive: false })

const app = createApp(App)

initStore(app)
initRouter(app)
setupGlobDirectives(app)
setupErrorHandle(app)

app.use(language)

// 主题必须在 store 就绪之后初始化（它要读 settingStore 里持久化的偏好）
initializeTheme()

app.mount('#app')
