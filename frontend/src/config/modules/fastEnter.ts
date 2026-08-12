/**
 * 快速入口配置
 * 应用列表 + 快速链接。
 *
 * P2-81：模板自带的 routeName 大多不存在（Console/Analysis/Fireworks/Chat/
 * ChangeLog/...），`router.push({ name })` 会 reject 并静默失败，快速入口
 * 形同虚设。这里全部换成真实注册的路由名（见 src/router/routes/**），
 * 外链也改成站内路由，不再指向模板作者的站点。
 */
import type { FastEnterConfig } from '@/types/config'

const fastEnterConfig: FastEnterConfig = {
  // 显示条件（屏幕宽度）
  minWidth: 1200,
  // 应用列表
  applications: [
    {
      name: '工作台',
      description: '控制台概览与数据统计',
      icon: 'ri:pie-chart-line',
      iconColor: '#377dff',
      enabled: true,
      order: 1,
      routeName: 'ConsoleDashboard'
    },
    {
      name: '我的 VPS',
      description: '管理云服务器实例',
      icon: 'ri:server-line',
      iconColor: '#13DEB9',
      enabled: true,
      order: 2,
      routeName: 'ConsoleVps'
    },
    {
      name: '账单与钱包',
      description: '充值、提现与流水',
      icon: 'ri:wallet-3-line',
      iconColor: '#ff6b6b',
      enabled: true,
      order: 3,
      routeName: 'ConsoleBilling'
    },
    {
      name: '产品',
      description: '浏览套餐与价格',
      icon: 'ri:shopping-bag-line',
      iconColor: '#7A7FFF',
      enabled: true,
      order: 4,
      routeName: 'PublicProducts'
    },
    {
      name: '帮助中心',
      description: '使用指南与常见问题',
      icon: 'ri:question-line',
      iconColor: '#ffb100',
      enabled: true,
      order: 5,
      routeName: 'PublicHelp'
    },
    {
      name: '文档',
      description: '产品文档与公告',
      icon: 'ri:book-line',
      iconColor: '#38C0FC',
      enabled: true,
      order: 6,
      routeName: 'PublicDocs'
    }
  ],
  // 快速链接
  quickLinks: [
    {
      name: '登录',
      enabled: true,
      order: 1,
      routeName: 'Login'
    },
    {
      name: '注册',
      enabled: true,
      order: 2,
      routeName: 'Register'
    },
    {
      name: '忘记密码',
      enabled: true,
      order: 3,
      routeName: 'ForgotPassword'
    },
    {
      name: '个人中心',
      enabled: true,
      order: 4,
      routeName: 'ConsoleProfile'
    },
    {
      name: '购物车',
      enabled: true,
      order: 5,
      routeName: 'PublicCart'
    }
  ]
}

export default Object.freeze(fastEnterConfig)
