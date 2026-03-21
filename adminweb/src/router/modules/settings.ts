import { AppRouteRecord } from '@/types/router'

export const settingsRoutes: AppRouteRecord = {
  path: '/settings',
  name: 'Settings',
  component: '/index/index',
  meta: {
    title: '系统设置',
    icon: 'ri:settings-3-line',
    roles: ['R_SUPER', 'R_ADMIN']
  },
  children: [
    {
      path: 'scheduled-tasks',
      name: 'SettingsScheduledTasks',
      component: '/ops/scheduled-tasks',
      meta: {
        title: '计划任务',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '列表', authMark: 'scheduled_tasks.list' },
          { title: '更新', authMark: 'scheduled_tasks.update' },
          { title: '运行记录', authMark: 'scheduled_tasks.runs' }
        ]
      }
    },
    {
      path: 'site',
      name: 'SettingsSite',
      component: '/settings/site',
      meta: {
        title: '站点设置',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看设置', authMark: 'settings.view' },
          { title: '更新设置', authMark: 'settings.update' }
        ]
      }
    },
    {
      path: 'auth',
      name: 'SettingsAuth',
      component: '/settings/auth',
      meta: {
        title: '认证设置',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看设置', authMark: 'settings.view' },
          { title: '更新设置', authMark: 'settings.update' }
        ]
      }
    },
    {
      path: 'captcha',
      name: 'SettingsCaptcha',
      component: '/settings/captcha',
      meta: {
        title: '验证码设置',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看设置', authMark: 'settings.view' },
          { title: '更新设置', authMark: 'settings.update' }
        ]
      }
    },
    {
      path: 'apikey',
      name: 'SettingsApiKeys',
      component: '/settings/api-keys',
      meta: {
        title: 'API 密钥',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看 API 密钥', authMark: 'api_key.list' },
          { title: '创建 API 密钥', authMark: 'api_key.create' },
          { title: '更新 API 密钥', authMark: 'api_key.update' }
        ]
      }
    },
    {
      path: 'plugins',
      name: 'SettingsPlugins',
      component: '/settings/plugins',
      meta: {
        title: '插件管理',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看插件', authMark: 'plugin.list' },
          { title: '查看插件配置', authMark: 'plugin.view' },
          { title: '创建插件', authMark: 'plugin.create' },
          { title: '更新插件', authMark: 'plugin.update' },
          { title: '删除插件', authMark: 'plugin.delete' },
          { title: '上传插件', authMark: 'plugin.upload' }
        ]
      }
    },
    {
      path: 'payment-plugins',
      name: 'SettingsPaymentPlugins',
      component: '/settings/payment-plugins',
      meta: {
        title: '支付插件上传',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看支付提供方', authMark: 'payment.list' },
          { title: '更新支付提供方', authMark: 'payment.update' },
          { title: '上传插件', authMark: 'plugin.upload' }
        ]
      }
    },
    {
      path: 'lifecycle',
      name: 'SettingsLifecycle',
      component: '/settings/lifecycle',
      meta: {
        title: '生命周期设置',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看设置', authMark: 'settings.view' },
          { title: '更新设置', authMark: 'settings.update' }
        ]
      }
    },
    {
      path: 'pricing',
      name: 'SettingsPricing',
      component: '/settings/pricing',
      meta: {
        title: '定价设置',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看设置', authMark: 'settings.view' },
          { title: '更新设置', authMark: 'settings.update' }
        ]
      }
    },
    {
      path: 'email',
      name: 'SettingsEmail',
      component: '/settings/email',
      meta: {
        title: '邮件设置',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看设置', authMark: 'settings.view' },
          { title: '更新设置', authMark: 'settings.update' }
        ]
      }
    },
    {
      path: 'fcm',
      name: 'SettingsFcm',
      component: '/settings/fcm',
      meta: {
        title: 'FCM 设置',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看设置', authMark: 'settings.view' },
          { title: '更新设置', authMark: 'settings.update' }
        ]
      }
    },
    {
      path: 'payments',
      name: 'SettingsPayments',
      component: '/settings/payments',
      meta: {
        title: '支付设置',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看设置', authMark: 'settings.view' },
          { title: '更新设置', authMark: 'settings.update' }
        ]
      }
    },
    {
      path: 'webhook',
      name: 'SettingsWebhook',
      component: '/settings/webhook',
      meta: {
        title: 'Webhook 设置',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看设置', authMark: 'settings.view' },
          { title: '更新设置', authMark: 'settings.update' }
        ]
      }
    },
    {
      path: 'sms',
      name: 'SettingsSms',
      component: '/settings/sms',
      meta: {
        title: '短信设置',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看设置', authMark: 'settings.view' },
          { title: '更新设置', authMark: 'settings.update' }
        ]
      }
    },
    {
      path: 'realname-config',
      name: 'SettingsRealnameConfig',
      component: '/realname/config',
      meta: {
        title: '实名认证配置',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看配置', authMark: 'realname.view' },
          { title: '更新配置', authMark: 'realname.update' }
        ]
      }
    },
    {
      path: 'realname-records',
      name: 'SettingsRealnameRecords',
      component: '/realname/records',
      meta: {
        title: '实名认证记录',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [{ title: '查看记录', authMark: 'realname.list' }]
      }
    },
    {
      path: 'automation',
      name: 'SettingsAutomation',
      component: '/automation/index',
      meta: {
        title: '自动化对接',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [{ title: '自动化对接', authMark: 'automation.view' }]
      }
    }
  ]
}
