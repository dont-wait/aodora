/** The only sizes the shop uses, in display order (mirrors server catalog.StandardSizes). */
export const STANDARD_SIZES = ['XS', 'S', 'M', 'L', 'XL', 'XXL', 'Freesize'] as const

export type StandardSize = (typeof STANDARD_SIZES)[number]

export function isStandardSize(size: string): size is StandardSize {
  return (STANDARD_SIZES as readonly string[]).includes(size)
}

/** Standard sizes first in standard order; unknown sizes keep their original relative order at the end. */
export function sortSizes<T extends string>(sizes: readonly T[]): T[] {
  const rank = (size: string) => {
    const index = (STANDARD_SIZES as readonly string[]).indexOf(size)
    return index === -1 ? STANDARD_SIZES.length : index
  }
  return sizes.map((size, index) => ({ size, index })).sort((a, b) => rank(a.size) - rank(b.size) || a.index - b.index).map((item) => item.size)
}

/** Body measurements (cm, kg) the shop sews each ready-made áo dài size for. */
export interface SizeMeasurement {
  bust: [number, number]
  waist: [number, number]
  hip: [number, number]
  height: [number, number]
  weight: [number, number]
}

/**
 * Common Vietnamese áo dài ready-made chart (fitted bodice, measured on the body,
 * not the garment). Freesize has no fixed measurements.
 */
export const SIZE_CHART: Partial<Record<StandardSize, SizeMeasurement>> = {
  XS: { bust: [76, 79], waist: [60, 62], hip: [82, 85], height: [148, 153], weight: [38, 42] },
  S: { bust: [80, 83], waist: [63, 65], hip: [86, 89], height: [150, 155], weight: [42, 46] },
  M: { bust: [84, 87], waist: [66, 69], hip: [90, 93], height: [153, 158], weight: [46, 51] },
  L: { bust: [88, 91], waist: [70, 73], hip: [94, 97], height: [156, 162], weight: [51, 56] },
  XL: { bust: [92, 95], waist: [74, 77], hip: [98, 101], height: [158, 165], weight: [56, 61] },
  XXL: { bust: [96, 100], waist: [78, 82], hip: [102, 106], height: [160, 168], weight: [61, 67] },
}

/** The first size whose bust, waist and hip ranges all fit; when ranges disagree the largest needed size wins. */
export function suggestSize(bust: number, waist: number, hip: number): StandardSize | null {
  const order = STANDARD_SIZES.filter((size) => SIZE_CHART[size])
  const need = (value: number, key: 'bust' | 'waist' | 'hip') => order.findIndex((size) => value <= SIZE_CHART[size]![key][1])
  const indexes = [need(bust, 'bust'), need(waist, 'waist'), need(hip, 'hip')]
  if (indexes.some((index) => index === -1) || [bust, waist, hip].some((v) => !(v > 0))) return null
  return order[Math.max(...indexes)]
}

/**
 * Rough size from height and weight only, for shoppers who have not measured
 * their girths. Weight decides (it tracks girth); height only breaks ties.
 */
export function estimateSizeFromBody(height: number, weight: number): StandardSize | null {
  if (!(height > 0) || !(weight > 0)) return null
  const order = STANDARD_SIZES.filter((size) => SIZE_CHART[size])
  const distance = ([min, max]: [number, number], value: number) => (value < min ? min - value : value > max ? value - max : 0)
  let best: StandardSize | null = null
  let bestScore = Infinity
  for (const size of order) {
    const m = SIZE_CHART[size]!
    const score = distance(m.weight, weight) * 10 + distance(m.height, height)
    if (score < bestScore) { best = size; bestScore = score }
  }
  // Far outside the chart (e.g. 30 kg or 95 kg): ready-made sizes will not fit.
  const first = SIZE_CHART[order[0]]!.weight[0], last = SIZE_CHART[order[order.length - 1]]!.weight[1]
  return weight < first - 4 || weight > last + 6 ? null : best
}

export type LengthFit = 'short' | 'long' | 'ok'

/** Whether a size's length suits the wearer: taller than the size's range → tà/tay may be short. */
export function lengthFit(size: StandardSize, height: number): LengthFit | null {
  const m = SIZE_CHART[size]
  if (!m || !(height > 0)) return null
  if (height > m.height[1] + 2) return 'short'
  if (height < m.height[0] - 2) return 'long'
  return 'ok'
}
