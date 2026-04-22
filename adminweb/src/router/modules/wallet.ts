import { AppRouteRecord } from '@/types/router'

export const walletRoutes: AppRouteRecord = {
  path: '/wallet',
  name: 'Wallet',
  component: '/index/index',
  meta: {
    title: '钱包管理',
    icon: 'ri:wallet-3-line',
    roles: ['R_SUPER', 'R_ADMIN']
  },
  children: [
    {
      path: 'orders',
      name: 'WalletOrders',
      component: '/wallet/orders',
      meta: {
        title: '钱包订单',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看', authMark: 'wallet_order.list' },
          { title: '通过', authMark: 'wallet_order.approve' },
          { title: '驳回', authMark: 'wallet_order.reject' }
        ]
      }
    }
  ]
}
