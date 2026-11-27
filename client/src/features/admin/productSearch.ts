/** Lower-case, diacritic-free text for the product list search box. */
export function fold(value: string): string {
  return value
    .normalize('NFD')
    .replace(/[\u{300}-\u{36f}]/gu, '')
    .replace(/đ/gi, 'd')
    .toLowerCase()
    .trim()
}
