import { AppRouteRecord } from '@/types/router'

export const vpsRoutes: AppRouteRecord = {
  path: '/vps',
  name: 'Vps',
  component: '/vps/index',
  meta: {
    title: 'VPS 管理',
    icon: 'ri:server-line',
    keepAlive: true,
    roles: ['R_SUPER', 'R_ADMIN'],
    authList: [
      { title: '列表', authMark: 'vps.list' },
      { title: '详情', authMark: 'vps.view' },
      { title: '创建', authMark: 'vps.create' },
      { title: '更新', authMark: 'vps.update' },
      { title: '删除', authMark: 'vps.delete' },
      { title: '锁定', authMark: 'vps.lock' },
      { title: '解锁', authMark: 'vps.unlock' },
      { title: '改配', authMark: 'vps.resize' },
      { title: '状态', authMark: 'vps.admin_status' },
      { title: '到期时间', authMark: 'vps.update_expire' },
      { title: '刷新', authMark: 'vps.refresh' },
      { title: '紧急续费', authMark: 'vps.emergency_renew' }
    ]
  }
}
