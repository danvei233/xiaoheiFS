import App from './App.vue'
import { createApp } from 'vue'
import { initStore } from './store'
import { initRouter } from './router'
import language from './locales'
import '@styles/core/tailwind.css'
import '@styles/index.scss'
import '@utils/sys/console.ts'
import { setupGlobDirectives } from './directives'
import { setupErrorHandle } from './utils/sys/error-handle'

normalizeInstallHashEntry()

document.addEventListener(
  'touchstart',
  function () {},
  { passive: false }
)

const app = createApp(App)
initStore(app)
initRouter(app)
setupGlobDirectives(app)
setupErrorHandle(app)

app.use(language)
app.mount('#app')

function normalizeInstallHashEntry() {
  if (typeof window === 'undefined' || window.location.hash) {
    return
  }

  const normalizedPath = window.location.pathname.replace(/\/+$/, '')
  if (normalizedPath !== '/install') {
    return
  }

  const pathname = window.location.pathname.endsWith('/')
    ? window.location.pathname
    : `${window.location.pathname}/`

  window.history.replaceState(
    window.history.state,
    '',
    `${pathname}${window.location.search}#/install`
  )
}
