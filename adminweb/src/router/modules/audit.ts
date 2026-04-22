import { AppRouteRecord } from '@/types/router'

export const auditRoutes: AppRouteRecord = {
  path: '/audit',
  name: 'AuditLogs',
  component: '/audit/index',
  meta: {
    title: '审计日志',
    icon: 'ri:shield-keyhole-line',
    keepAlive: true,
    roles: ['R_SUPER', 'R_ADMIN'],
    authList: [{ title: '查看审计日志', authMark: 'audit_log.view' }]
  }
}
