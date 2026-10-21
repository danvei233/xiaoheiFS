import { AppRouteRecord } from '@/types/router'

export const probeRoutes: AppRouteRecord = {
  path: '/probes',
  name: 'Probes',
  component: '/index/index',
  meta: {
    title: '探针监控',
    icon: 'ri:radar-line',
    roles: ['R_SUPER', 'R_ADMIN']
  },
  children: [
    {
      path: 'list',
      name: 'ProbeList',
      component: '/probe/list',
      meta: {
        title: '探针列表',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '列表', authMark: 'probe.list' },
          { title: '详情', authMark: 'probe.view' },
          { title: '创建', authMark: 'probe.create' },
          { title: '更新', authMark: 'probe.update' },
          { title: '删除', authMark: 'probe.delete' }
        ]
      }
    },
    {
      path: ':id',
      name: 'ProbeDetail',
      component: '/probe/detail',
      meta: {
        title: '探针详情',
        isHide: true,
        activePath: '/probes/list',
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '详情', authMark: 'probe.view' },
          { title: '更新', authMark: 'probe.update' }
        ]
      }
    }
  ]
}
