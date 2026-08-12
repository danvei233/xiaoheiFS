/**
 * 回归测试：用户域登录后的 redirect query 白名单校验（web_auth_surface F4）。
 *
 * 背景：管理域登录已有 resolveAdminRedirect 白名单；用户登录页此前把
 *   `route.query.redirect` 裸交给 router.replace。vue-router 4 的 base 解析
 *   暂时保证同源，但为防将来改成 location.href / 升级路由变成开放重定向，
 *   这里与 admin 域一致收敛到白名单。
 *
 * 运行：`node --test src/utils/__tests__/userRedirect.regression.test.mjs`
 */
import assert from 'node:assert/strict'

// 直接导入纯函数（无依赖链，无需 esbuild）。
const { resolveUserRedirect } = await import('../../utils/auth.ts')

// ── 合法值原样返回 ───────────────────────────────────────────────────────
assert.equal(resolveUserRedirect('/console'), '/console', 'console 根允许')
assert.equal(resolveUserRedirect('/console/orders'), '/console/orders', 'console 子路径允许')
assert.equal(resolveUserRedirect('/login'), '/login', 'login 允许')
assert.equal(resolveUserRedirect('/register'), '/register', 'register 允许')
assert.equal(resolveUserRedirect('/cart'), '/cart', 'cart 允许')
assert.equal(resolveUserRedirect('/buy'), '/buy', 'buy 允许')

// ── 危险值回退 /console ──────────────────────────────────────────────────
assert.equal(resolveUserRedirect('//evil.com'), '/console', '协议相对回退')
assert.equal(resolveUserRedirect('https://evil.com/steal'), '/console', '外域 https 回退')
assert.equal(resolveUserRedirect('javascript:alert(1)'), '/console', 'javascript 回退')
assert.equal(resolveUserRedirect('/console\\..\\evil'), '/console', '反斜杠回退')
assert.equal(resolveUserRedirect('not-a-path'), '/console', '非相对路径回退')
assert.equal(resolveUserRedirect(''), '/console', '空串回退')
assert.equal(resolveUserRedirect(null), '/console', 'null 回退')
assert.equal(resolveUserRedirect(undefined), '/console', 'undefined 回退')
assert.equal(
  resolveUserRedirect('/admin/dashboard'),
  '/console',
  'admin 域路径回退（用户域不允许）'
)

console.log('✅ userRedirect.regression.test.mjs —— 用户域 redirect 白名单断言通过')
