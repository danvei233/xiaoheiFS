<template>
  <div class="page-content !border-0 !bg-transparent min-h-screen flex-cc">
    <div class="flex-cc max-md:!block max-md:text-center">
      <ThemeSvg :src="data.imgUrl" size="100%" class="!w-100" />
      <div class="ml-15 w-75 max-md:mx-auto max-md:mt-10 max-md:w-full max-md:text-center">
        <p class="text-xl leading-7 text-g-600 max-md:text-lg">{{ data.desc }}</p>
        <ElButton type="primary" size="large" @click="backHome" v-ripple class="mt-5">{{
          data.btnText
        }}</ElButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { useCommon } from '@/hooks/core/useCommon'

  const router = useRouter()

  interface ExceptionData {
    /** 标题 */
    title: string
    /** 描述 */
    desc: string
    /** 按钮文本 */
    btnText: string
    /** 图片地址 */
    imgUrl: string
  }

  withDefaults(
    defineProps<{
      data: ExceptionData
    }>(),
    {}
  )

  const { homePath } = useCommon()

  /**
   * 返回按钮的目标：
   *  1. 菜单 store 已算好的 homePath（控制台/后台菜单加载过时）；
   *  2. 否则当前 URL 落在 admin 前缀 → 后台首页；
   *  3. 否则回公开首页 `/`。
   *
   * P3-48：不再因 `userStore.isLogin` 恒 false 把匿名访客/管理员一律踢去登录页。
   */
  const resolveHomePath = (): string => {
    const menuHome = homePath.value
    if (menuHome && menuHome !== '/') return menuHome
    try {
      const adminPath = (localStorage.getItem('admin_path_cache') || 'admin')
        .trim()
        .replace(/^\/+|\/+$/g, '')
      const p = window.location.pathname
      if (adminPath && (p === `/${adminPath}` || p.startsWith(`/${adminPath}/`))) {
        return `/${adminPath}`
      }
    } catch {
      // ignore
    }
    return '/'
  }

  const backHome = () => {
    router.push(resolveHomePath())
  }
</script>
