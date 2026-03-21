import { AppRouteRecord } from '@/types/router'

export const dashboardRoutes: AppRouteRecord = {
  name: 'Dashboard',
  path: '/dashboard',
  component: '/index/index',
  meta: {
    title: 'menus.dashboard.title',
    icon: 'ri:pie-chart-line',
    roles: ['R_SUPER', 'R_ADMIN']
  },
  children: [
    {
      path: 'console',
      name: 'Console',
      component: '/dashboard/overview',
      meta: {
        title: 'menus.dashboard.console',
        keepAlive: false,
        fixedTab: true,
        authList: [
          { title: 'Dashboard Overview', authMark: 'dashboard.overview' },
          { title: 'Dashboard Revenue', authMark: 'dashboard.revenue' },
          { title: 'Dashboard VPS Status', authMark: 'dashboard.vps_status' },
          { title: 'Server Status', authMark: 'server.status' }
        ]
      }
    },
    {
      path: 'revenue-analytics',
      name: 'RevenueAnalytics',
      component: '/dashboard/revenue-analytics',
      meta: {
        title: '收入统计',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: 'Revenue Overview', authMark: 'dashboard.revenue_analytics_overview' },
          { title: 'Revenue Trend', authMark: 'dashboard.revenue_analytics_trend' },
          { title: 'Revenue Top', authMark: 'dashboard.revenue_analytics_top' },
          { title: 'Revenue Details', authMark: 'dashboard.revenue_analytics_details' },
          { title: 'Revenue Dashboard', authMark: 'dashboard.revenue' }
        ]
      }
    }
  ]
}
