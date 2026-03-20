import { AppRouteRecord } from '@/types/router'

export const catalogRoutes: AppRouteRecord = {
  path: '/catalog',
  name: 'Catalog',
  component: '/catalog/index',
  meta: {
    title: '商品目录',
    icon: 'ri:store-3-line',
    isHide: true,
    keepAlive: true,
    activePath: '/product/catalog',
    roles: ['R_SUPER', 'R_ADMIN'],
    authList: [
      { title: '商品类型', authMark: 'goods_type.list' },
      { title: '地区', authMark: 'region.list' },
      { title: '线路', authMark: 'plan_group.list' },
      { title: '套餐', authMark: 'package.list' },
      { title: '系统镜像', authMark: 'system_image.list' },
      { title: '计费周期', authMark: 'billing_cycle.list' }
    ]
  }
}
