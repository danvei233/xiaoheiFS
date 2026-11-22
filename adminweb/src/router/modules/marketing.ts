import { AppRouteRecord } from '@/types/router'

export const marketingRoutes: AppRouteRecord = {
  path: '/marketing',
  name: 'Marketing',
  component: '/index/index',
  meta: {
    title: '营销中心',
    icon: 'ri:coupon-3-line',
    roles: ['R_SUPER', 'R_ADMIN']
  },
  children: [
    {
      path: 'coupons',
      name: 'MarketingCoupons',
      component: '/marketing/coupons',
      meta: {
        title: '优惠券',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看优惠组', authMark: 'coupon_groups.list' },
          { title: '创建优惠组', authMark: 'coupon_groups.create' },
          { title: '更新优惠组', authMark: 'coupon_groups.update' },
          { title: '删除优惠组', authMark: 'coupon_groups.delete' },
          { title: '查看优惠券', authMark: 'coupons.list' },
          { title: '创建优惠券', authMark: 'coupons.create' },
          { title: '更新优惠券', authMark: 'coupons.update' },
          { title: '删除优惠券', authMark: 'coupons.delete' },
          { title: '批量生成优惠券', authMark: 'coupons.batch_generate' }
        ]
      }
    }
  ]
}
