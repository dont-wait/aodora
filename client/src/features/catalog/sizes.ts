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
