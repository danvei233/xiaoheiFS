import { AppRouteRecord } from '@/types/router'
import { automationRoutes } from './automation'
import { auditRoutes } from './audit'
import { catalogRoutes } from './catalog'
import { cmsRoutes } from './cms-routes'
import { dashboardRoutes } from './dashboard'
import { debugRoutes } from './debug'
import { marketingRoutes } from './marketing'
import { opsRoutes } from './ops'
import { orderRoutes } from './order'
import { productRoutes } from './product'
import { probeRoutes } from './probe'
import { realnameRoutes } from './realname'
import { settingsRoutes } from './settings'
import { systemsRoutes } from './systems'
import { systemRoutes } from './system'
import { ticketRoutes } from './ticket'
import { vpsRoutes } from './vps'
import { walletRoutes } from './wallet'
import { resultRoutes } from './result'
import { exceptionRoutes } from './exception'

/**
 * 导出所有模块化路由
 */
export const routeModules: AppRouteRecord[] = [
  productRoutes,
  cmsRoutes,
  dashboardRoutes,
  marketingRoutes,
  orderRoutes,
  probeRoutes,
  settingsRoutes,
  systemRoutes,
  ticketRoutes,
  vpsRoutes,
  walletRoutes,
  auditRoutes,
  debugRoutes,
  automationRoutes,
  catalogRoutes,
  opsRoutes,
  realnameRoutes,
  systemsRoutes,
  resultRoutes,
  exceptionRoutes
]
