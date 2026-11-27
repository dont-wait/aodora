/** "Áo dài Lụa Đỏ!" -> "ao-dai-lua-do": no diacritics (đ -> d), lower-case, single dashes. */
export function slugify(input: string, maxLength = 120): string {
  return input
    .normalize('NFD')
    .replace(/[\u{300}-\u{36f}]/gu, '')
    .replace(/đ/gi, 'd')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, maxLength)
    .replace(/-+$/g, '')
}
