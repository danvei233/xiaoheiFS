/**
 * 回归测试：支付结果 URL 安全策略（H2）。
 *
 * 背景：后端支付插件返回的 code_url/pay_url/urlscheme 是外部可控数据，
 *   旧实现直接 `window.location.href = javascript:...` / `window.open(...)`，
 *   `javascript:` 跳转在目标窗口能读到同源 token。
 *
 * 目标契约（classifyPaymentUrl 纯函数）：
 *  - web 只允许 https（生产）；http 仅开发模式 + localhost 放行；
 *  - 自定义 scheme 只允许支付白名单；
 *  - 拒绝 javascript/data/file/blob/vbscript/协议相对/userinfo/控制字符/空；
 *  - openPaymentUrl 对合法 web 用 noopener,noreferrer 新窗口、scheme 用
 *    location.href，非法值不导航且回调 onInvalid。
 *
 * 运行：`node --test src/utils/__tests__/paymentUrl.regression.test.mjs`
 */
import { bundleEntry } from '../../../tests/_bundle.mjs'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const { classifyPaymentUrl, openPaymentUrl } = await bundleEntry(
  `export { classifyPaymentUrl, openPaymentUrl } from '@/utils/paymentUrl'`,
  import.meta.url
)

const ok = (value, kind, label) => {
  const d = classifyPaymentUrl(value)
  assert.equal(d.ok, true, label)
  assert.equal(d.kind, kind, `${label}: kind=${kind}`)
}
const rejected = (value, label) => {
  const d = classifyPaymentUrl(value)
  assert.equal(d.ok, false, label)
  assert.ok(d.reason, `${label}: 应给出拒绝原因`)
}

// ── 合法 web（https）──────────────────────────────────────────────────
ok('https://pay.example.com/gateway?order=1', 'web', 'https 允许')
ok('https://example.com/', 'web', 'https 根路径允许')

// ── http：生产环境一律拒绝 ─────────────────────────────────────────────
rejected('http://pay.example.com/gateway', '生产 http 拒绝')
rejected('http://localhost:8080/pay', '非开发模式 localhost http 也拒绝')

// ── 危险 scheme 全部拒绝 ───────────────────────────────────────────────
rejected('javascript:alert(document.cookie)', 'javascript 拒绝')
rejected('JaVaScRiPt:alert(1)', '大小写混淆 javascript 拒绝')
rejected('data:text/html,<script>alert(1)</script>', 'data 拒绝')
rejected('file:///etc/passwd', 'file 拒绝')
rejected('blob:https://example.com/uuid', 'blob 拒绝')
rejected('vbscript:msgbox(1)', 'vbscript 拒绝')

// ── 协议相对 / 缺失 scheme / userinfo / 控制字符 / 空 ───────────────────
rejected('//evil.example.com/steal', '协议相对拒绝')
rejected('not-a-url', '缺失 scheme 拒绝')
rejected('https://user:pass@evil.example.com/', 'https userinfo 拒绝')
rejected('https://evil.example.com/\u0000javascript:alert(1)', '控制字符拒绝')
rejected('', '空串拒绝')
rejected(null, 'null 拒绝')
rejected(undefined, 'undefined 拒绝')
rejected(12345, '非字符串拒绝')
rejected('   ', '纯空白拒绝')

// ── 支付 scheme 白名单 ─────────────────────────────────────────────────
ok('weixin://wxpay/bizpayurl?pr=abc', 'scheme', 'weixin scheme 允许')
ok('alipays://platformapi/startapp?appId=20000123', 'scheme', 'alipays scheme 允许')
ok('wechat://wxa/?appid=x', 'scheme', 'wechat scheme 允许')
ok('WXPAY://order/1', 'scheme', '大写 scheme 归一化后允许')
ok('alipay://app/pay', 'scheme', 'alipay scheme 允许')

// 非支付 scheme 拒绝
rejected('mailto:admin@example.com', 'mailto 拒绝')
rejected('tel:10086', 'tel 拒绝')
rejected('intent://pay/#Intent;scheme=https;end', 'intent 拒绝')
rejected('unknownscheme://x', '未知 scheme 拒绝')

// ── openPaymentUrl 导航行为 ────────────────────────────────────────────
const opened = []
globalThis.window = {
  location: { href: 'initial' },
  open: (url, target, features) => {
    opened.push({ url, target, features })
    return {}
  }
}

// 合法 web → noopener,noreferrer 新窗口
assert.equal(
  openPaymentUrl('https://pay.example.com/gateway', { onInvalid: () => assert.fail('不应拒绝') }),
  true,
  '合法 web 应导航'
)
assert.equal(opened.length, 1)
assert.equal(opened[0].url, 'https://pay.example.com/gateway')
assert.equal(opened[0].target, '_blank')
assert.ok(opened[0].features.includes('noopener'), '新窗口必须带 noopener')
assert.ok(opened[0].features.includes('noreferrer'), '新窗口必须带 noreferrer')

// 合法 scheme → location.href
assert.equal(openPaymentUrl('weixin://wxpay/bizpayurl?pr=abc'), true, '合法 scheme 应导航')
assert.equal(window.location.href, 'weixin://wxpay/bizpayurl?pr=abc', 'scheme 走 location.href')

// 非法 → 不导航 + onInvalid 回调
let invalidReason = ''
const navigated = openPaymentUrl('javascript:alert(1)', {
  onInvalid: (r) => {
    invalidReason = r
  }
})
assert.equal(navigated, false, '非法值不得导航')
assert.equal(invalidReason, 'scheme-not-allowed', '非法值回调拒绝原因')
assert.equal(opened.length, 1, '非法值不得触发 window.open')
assert.equal(window.location.href, 'weixin://wxpay/bizpayurl?pr=abc', '非法值不得改 location')

// ── H8：二维码展示只要求 decision.ok，不看 kind ────────────────────────
// 微信 native 支付的 code_url 通常是白名单 scheme `weixin://wxpay/bizpayurl`，
// 二维码展示（QRCode.toDataURL）只是把内容渲染成图片，用户扫码后由微信 App
// 处理，不应被 `kind !== 'web'` 错杀；javascript:/data: 等危险内容仍拒绝。
assert.equal(
  classifyPaymentUrl('weixin://wxpay/bizpayurl?pr=abc').ok,
  true,
  '合法微信 code_url（scheme）QR 展示应放行'
)
assert.equal(
  classifyPaymentUrl('alipays://platformapi/startapp?appId=20000123').ok,
  true,
  '合法支付宝 scheme QR 展示应放行'
)
assert.equal(classifyPaymentUrl('javascript:alert(1)').ok, false, 'javascript 内容不得生成二维码')
assert.equal(
  classifyPaymentUrl('data:image/svg+xml,<svg onload=alert(1)>').ok,
  false,
  'data 内容不得生成二维码'
)
assert.equal(
  classifyPaymentUrl('weixin://user:pass@wxpay/x').ok,
  false,
  '带 userinfo 的 scheme 不得生成二维码'
)

// ── 源码漂移防护：openWeChatQr 必须只看 decision.ok ─────────────────────
const here = path.dirname(fileURLToPath(import.meta.url))
const detailPath = path.resolve(here, '../../../src/views/console/orders/detail.vue')
const detailSource = fs.readFileSync(detailPath, 'utf8')
// 抽出 openWeChatQr 函数体（约到下一个顶层 const 为止），只对该段断言，
// 避免匹配到其它函数或注释文本。
const qrMatch = /const openWeChatQr[\s\S]*?\n {2}\}/.exec(detailSource)
assert.ok(qrMatch, '应能找到 openWeChatQr 函数体')
const qrBody = qrMatch[0]
assert.ok(
  qrBody.includes('if (!decision.ok)'),
  "openWeChatQr 应只校验 decision.ok（不得用 kind !== 'web' 错杀微信 scheme 二维码）"
)
assert.ok(
  !/if\s*\(\s*decision\.kind\s*!==\s*'web'/.test(qrBody),
  "openWeChatQr 函数体内不得出现 decision.kind !== 'web' 判断"
)

console.log('✅ paymentUrl.regression.test.mjs —— 支付 URL 策略断言通过')
