import { AppRouteRecord } from '@/types/router'

export const ticketRoutes: AppRouteRecord = {
  path: '/tickets',
  name: 'Tickets',
  component: '/index/index',
  meta: {
    title: '工单管理',
    icon: 'ri:customer-service-2-line',
    roles: ['R_SUPER', 'R_ADMIN']
  },
  children: [
    {
      path: 'list',
      name: 'TicketList',
      component: '/ticket/list',
      meta: {
        title: '工单列表',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '列表', authMark: 'tickets.list' },
          { title: '详情', authMark: 'tickets.view' },
          { title: '更新', authMark: 'tickets.update' },
          { title: '删除', authMark: 'tickets.delete' }
        ]
      }
    },
    {
      path: ':id',
      name: 'TicketDetail',
      component: '/ticket/detail',
      meta: {
        title: '工单详情',
        isHide: true,
        activePath: '/tickets/list',
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '详情', authMark: 'tickets.view' },
          { title: '更新', authMark: 'tickets.update' },
          { title: '消息', authMark: 'tickets.messages' },
          { title: '删除', authMark: 'tickets.delete' }
        ]
      }
    }
  ]
}
