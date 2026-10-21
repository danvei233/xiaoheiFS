import { AppRouteRecord } from '@/types/router'

export const productRoutes: AppRouteRecord = {
  path: '/product',
  name: 'ProductSettings',
  component: '/index/index',
  meta: {
    title: '商品设置',
    icon: 'ri:store-3-line',
    roles: ['R_SUPER', 'R_ADMIN']
  },
  children: [
    {
      path: 'catalog',
      name: 'ProductCatalog',
      component: '/catalog/index',
      meta: {
        title: '商品目录',
        keepAlive: true,
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
    },
    {
      path: 'systems',
      name: 'ProductSystems',
      component: '/systems/index',
      meta: {
        title: '系统镜像',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '系统镜像', authMark: 'system_image.list' },
          { title: '线路', authMark: 'line.list' }
        ]
      }
    }
  ]
}
