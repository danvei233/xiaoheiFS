import { AppRouteRecord } from '@/types/router'

export const opsRoutes: AppRouteRecord = {
  path: '/scheduled-tasks',
  name: 'ScheduledTasks',
  component: '/ops/scheduled-tasks',
  meta: {
    title: '计划任务',
    icon: 'ri:time-line',
    isHide: true,
    keepAlive: true,
    activePath: '/settings/scheduled-tasks',
    roles: ['R_SUPER', 'R_ADMIN'],
    authList: [
      { title: '列表', authMark: 'scheduled_tasks.list' },
      { title: '更新', authMark: 'scheduled_tasks.update' },
      { title: '运行记录', authMark: 'scheduled_tasks.runs' }
    ]
  }
}
