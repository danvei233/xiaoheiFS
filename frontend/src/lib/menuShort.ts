/**
 * Short rail labels for collapsed sidebars.
 *
 * Truncating the first two characters produces nonsense ("云服" for 云服务器),
 * so the meaningful short form is curated first and only then falls back to a
 * suffix-stripping heuristic.
 */

const EXACT: Record<string, string> = {
  系统总览: "总览",
  运营总览: "总览",
  云服务器: "实例",
  "VPS 管理": "实例",
  待购清单: "待购",
  订单管理: "订单",
  订单审核: "订单",
  钱包充值: "钱包",
  钱包订单: "钱包",
  "API 管理": "API",
  工单中心: "工单",
  实名认证: "实名",
  个人设置: "个人",
  用户管理: "用户",
  用户等级: "等级",
  用户客服: "客服",
  售卖目录: "套餐",
  收入统计: "收入",
  交易资源: "交易",
  探测监控: "探测",
  内容运营: "内容",
  平台配置: "平台",
  站点设置: "站点",
  支付设置: "支付",
  推送设置: "推送",
};

/** Generic words that carry little meaning once the space is 52px wide. */
const SUFFIXES = [
  "管理", "中心", "设置", "统计", "审核", "监控", "列表", "服务商",
  "记录", "配置", "目录", "运营", "资源", "充值", "认证", "清单", "订单"
];

export function shortMenuLabel(label: string): string {
  const text = String(label || "").trim();
  if (!text) return "";
  if (EXACT[text]) return EXACT[text];

  const latin = text.match(/^[A-Za-z0-9][A-Za-z0-9&+._/-]*/);
  if (latin) {
    const token = latin[0].toUpperCase();
    return token.length <= 3 ? token : token.slice(0, 3);
  }

  let rest = text;
  for (const suffix of SUFFIXES) {
    if (rest.length > suffix.length + 1 && rest.endsWith(suffix)) {
      rest = rest.slice(0, -suffix.length);
      break;
    }
  }
  const cjk = rest.match(/[\u4e00-\u9fa5]{1,2}/);
  return cjk ? cjk[0] : text.slice(0, 2);
}
