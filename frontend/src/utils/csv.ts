/**
 * CSV 单元格转义（P3-39 / m062 L1）。
 *
 * 两个职责：
 *  1. 双引号包裹并把内部 `"` 翻倍，符合 CSV 引号规则；
 *  2. 中和公式注入：以 `= + - @ \t \r` 开头的单元格会被 Excel/Google Sheets
 *     当作公式执行，前置单引号使其按文本对待。
 */
export function escapeCsvCell(value: string | number | null | undefined): string {
  let text = String(value ?? '')
  if (/^[=+\-@\t\r]/.test(text)) {
    text = `'${text}`
  }
  return `"${text.replace(/"/g, '""')}"`
}

/** 把若干行转义后的单元格拼成一行 CSV。 */
export function toCsvLine(cells: Array<string | number | null | undefined>): string {
  return cells.map(escapeCsvCell).join(',')
}
