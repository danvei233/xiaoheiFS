import { AppRouteRecord } from '@/types/router'

export const automationRoutes: AppRouteRecord = {
  path: '/automation',
  name: 'Automation',
  component: '/automation/index',
  meta: {
    title: '自动化对接',
    icon: 'ri:links-line',
    isHide: true,
    keepAlive: true,
    activePath: '/settings/automation',
    roles: ['R_SUPER', 'R_ADMIN'],
    authList: [{ title: '自动化对接', authMark: 'automation.view' }]
  }
}
