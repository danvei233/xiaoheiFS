import { AppRouteRecord } from '@/types/router'

export const realnameRoutes: AppRouteRecord = {
  path: '/realname',
  name: 'Realname',
  component: '/index/index',
  meta: {
    title: '实名认证',
    isHide: true,
    icon: 'ri:shield-user-line',
    roles: ['R_SUPER', 'R_ADMIN']
  },
  children: [
    {
      path: 'providers',
      name: 'RealnameProviders',
      component: '/realname/providers',
      meta: {
        title: '实名供应商',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [{ title: '查看供应商', authMark: 'realname.list' }]
      }
    },
    {
      path: 'config',
      name: 'RealnameConfig',
      component: '/realname/config',
      meta: {
        title: '实名认证配置',
        activePath: '/settings/realname-config',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看配置', authMark: 'realname.view' },
          { title: '更新配置', authMark: 'realname.update' }
        ]
      }
    },
    {
      path: 'records',
      name: 'RealnameRecords',
      component: '/realname/records',
      meta: {
        title: '实名认证记录',
        activePath: '/settings/realname-records',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [{ title: '查看记录', authMark: 'realname.list' }]
      }
    }
  ]
}
