<template>
  <ElConfigProvider
    size="default"
    :locale="locales[language]"
    :z-index="3000"
    :card="{
      shadow: 'never'
    }"
  >
    <RouterView></RouterView>
  </ElConfigProvider>
</template>

<script setup lang="ts">
  import { useUserStore } from './store/modules/user'
  import zh from 'element-plus/es/locale/lang/zh-cn'
  import en from 'element-plus/es/locale/lang/en'
  import zhTW from 'element-plus/es/locale/lang/zh-tw'
  import ja from 'element-plus/es/locale/lang/ja'
  import ko from 'element-plus/es/locale/lang/ko'
  import fr from 'element-plus/es/locale/lang/fr'
  import de from 'element-plus/es/locale/lang/de'
  import es from 'element-plus/es/locale/lang/es'
  import it from 'element-plus/es/locale/lang/it'
  import pt from 'element-plus/es/locale/lang/pt'
  import ru from 'element-plus/es/locale/lang/ru'
  import ar from 'element-plus/es/locale/lang/ar'
  import { useI18n } from 'vue-i18n'
  import { languageOptions, matchLanguage, browserLanguage } from './locales/registry'
  import { systemUpgrade } from './utils/sys'
  import { toggleTransition } from './utils/ui/animation'
  import { checkStorageCompatibility } from './utils/storage'
  import { initializeTheme } from './hooks/core/useTheme'

  const userStore = useUserStore()
  const { language } = storeToRefs(userStore)

  const locales = { zh, 'zh-TW': zhTW, en, ja, ko, fr, de, es, it, pt, ru, ar }
  const { locale } = useI18n()
  try {
    const preference = localStorage.getItem('admin-language')
    if (preference) {
      userStore.languageAuto = preference === 'auto'
      language.value = matchLanguage(preference) || browserLanguage()
    } else if (userStore.languageAuto) {
      language.value = browserLanguage()
    }
  } catch {
    language.value = browserLanguage()
  }
  const followBrowserLanguage = () => {
    if (userStore.languageAuto) language.value = browserLanguage()
  }
  window.addEventListener('languagechange', followBrowserLanguage)
  onBeforeUnmount(() => window.removeEventListener('languagechange', followBrowserLanguage))
  watch(
    language,
    (value) => {
      locale.value = value
      document.documentElement.lang =
        languageOptions.find((item) => item.value === value)?.tag || 'en'
      document.documentElement.dir = value === 'ar' ? 'rtl' : 'ltr'
    },
    { immediate: true }
  )

  onBeforeMount(() => {
    toggleTransition(true)
    initializeTheme()
  })

  onMounted(() => {
    checkStorageCompatibility()
    toggleTransition(false)
    systemUpgrade()
  })
</script>
