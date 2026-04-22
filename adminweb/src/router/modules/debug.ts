import { AppRouteRecord } from '@/types/router'

export const debugRoutes: AppRouteRecord = {
  path: '/debug',
  name: 'Debug',
  component: '/debug/index',
  meta: {
    title: '调试中心',
    icon: 'ri:bug-line',
    keepAlive: true,
    roles: ['R_SUPER', 'R_ADMIN'],
    authList: [
      { title: '查看状态', authMark: 'debug.view' },
      { title: '查看日志', authMark: 'debug.list' },
      { title: '更新调试', authMark: 'debug.update' },
      { title: '查看设置', authMark: 'settings.view' },
      { title: '更新设置', authMark: 'settings.update' }
    ]
  }
}
