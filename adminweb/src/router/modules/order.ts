import { AppRouteRecord } from '@/types/router'

export const orderRoutes: AppRouteRecord = {
  path: '/order',
  name: 'Order',
  component: '/index/index',
  meta: {
    title: '订单管理',
    icon: 'ri:file-list-3-line',
    roles: ['R_SUPER', 'R_ADMIN']
  },
  children: [
    {
      path: 'review',
      name: 'OrderReview',
      component: '/order/review',
      meta: {
        title: '订单审核',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '列表', authMark: 'order.list' },
          { title: '查看', authMark: 'order.view' },
          { title: '更新', authMark: 'order.update' },
          { title: '删除', authMark: 'order.delete' }
        ]
      }
    }
  ]
}
