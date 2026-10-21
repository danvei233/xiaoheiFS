import { AppRouteRecord } from '@/types/router'

export const systemRoutes: AppRouteRecord = {
  path: '/system',
  name: 'System',
  component: '/index/index',
  meta: {
    title: 'menus.system.title',
    icon: 'ri:user-3-line',
    roles: ['R_SUPER', 'R_ADMIN']
  },
  children: [
    {
      path: 'user',
      name: 'User',
      component: '/system/user',
      meta: {
        title: 'menus.system.user',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看用户列表', authMark: 'user.list' },
          { title: '查看用户详情', authMark: 'user.view' }
        ]
      }
    },
    {
      path: 'admins',
      name: 'AdminManage',
      component: '/system/admin',
      meta: {
        title: '管理员管理',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看管理员列表', authMark: 'admin.list' },
          { title: '查看管理员详情', authMark: 'admin.view' }
        ]
      }
    },
    {
      path: 'permission-groups',
      name: 'PermissionGroups',
      component: '/system/permission-group',
      meta: {
        title: '权限组管理',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看权限组列表', authMark: 'permission_group.list' },
          { title: '查看权限组详情', authMark: 'permission_group.view' }
        ]
      }
    },
    {
      path: 'user-tiers',
      name: 'UserTiers',
      component: '/system/user-tiers',
      meta: {
        title: '用户等级',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看用户等级', authMark: 'user_tiers.list' },
          { title: '查看用户列表', authMark: 'user.list' }
        ]
      }
    },
    {
      path: 'user-center',
      name: 'UserCenter',
      component: '/system/profile',
      meta: {
        title: 'menus.system.userCenter',
        isHide: true,
        keepAlive: true,
        isHideTab: true
      }
    }
  ]
}
