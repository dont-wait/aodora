// Pure helpers for the 2-level Vietnamese administrative divisions in force since 01/07/2025
// (province/city → ward/commune; the district level no longer exists).

export interface Ward { code: number; name: string }
export interface Province { code: number; name: string; wards: Ward[] }

const PREFIX = /^(thanh pho|tinh|tp\.?|phuong|xa|dac khu)\s+/

/** Lowercase, no diacritics, no administrative prefix ("Tỉnh", "Thành phố", "TP.", "Phường", "Xã", "Đặc khu"), single spaces. */
export function normalizeName(value: string): string {
  const base = value
    .normalize('NFD')
    .replace(/[\u{300}-\u{36f}]/gu, '')
    .replace(/đ/gi, 'd')
    .toLowerCase()
    .replace(/\s+/g, ' ')
    .trim()
  return base.replace(PREFIX, '').trim()
}

/** Exact full name first; otherwise a normalized match that must be unambiguous (diacritic-stripping can collide, e.g. "Văn Lang"/"Văn Lăng"). */
function pickByName<T extends { name: string }>(items: T[], name: string): T | undefined {
  const exact = name.trim().replace(/\s+/g, ' ').toLowerCase()
  if (!exact) return undefined
  const full = items.find((item) => item.name.toLowerCase() === exact)
  if (full) return full
  const key = normalizeName(name)
  if (!key) return undefined
  const candidates = items.filter((item) => normalizeName(item.name) === key)
  return candidates.length === 1 ? candidates[0] : undefined
}

/** Lowercase, no diacritics, single spaces; the administrative prefix is kept. */
export function foldName(value: string): string {
  return value
    .normalize('NFD')
    .replace(/[\u{300}-\u{36f}]/gu, '')
    .replace(/đ/gi, 'd')
    .toLowerCase()
    .replace(/\s+/g, ' ')
    .trim()
}

/**
 * Filters options for a type-ahead box: diacritic/case-insensitive substring match against the
 * name with or without its administrative prefix. Empty query returns everything; options
 * whose name (sans prefix) starts with the query come first, then the rest in original order.
 */
export function searchOptions<T extends { label: string }>(options: T[], query: string): T[] {
  const full = foldName(query)
  if (!full) return options
  const bare = normalizeName(query)
  const ranked: { option: T; rank: number }[] = []
  for (const option of options) {
    const optFull = foldName(option.label)
    const optBare = normalizeName(option.label)
    let rank = -1
    const typedPrefix = bare !== full // e.g. "phuong an": stay close to what was typed, no loose substring on the bare part
    if ((bare && optBare.startsWith(bare)) || optFull.startsWith(full)) rank = 0
    else if (optFull.includes(full) || (!typedPrefix && bare && optBare.includes(bare))) rank = 1
    if (rank >= 0) ranked.push({ option, rank })
  }
  return ranked.filter((item) => item.rank === 0).concat(ranked.filter((item) => item.rank === 1)).map((item) => item.option)
}

export function findProvince(list: Province[], name: string): Province | undefined {
  return pickByName(list, name)
}

export function findWard(province: Province | undefined, name: string): Ward | undefined {
  return province ? pickByName(province.wards, name) : undefined
}

export interface SavedAddressParts { city?: string; ward?: string; district?: string }
export interface SavedAddressMatch { province?: Province; ward?: Ward; legacy: boolean }

/**
 * Matches a stored address against the current list. `legacy` is true when the
 * address carries a district, or when a non-empty city/ward is not in the new list.
 */
export function matchSavedAddress(list: Province[], saved: SavedAddressParts): SavedAddressMatch {
  const city = (saved.city ?? '').trim()
  const ward = (saved.ward ?? '').trim()
  const province = city ? findProvince(list, city) : undefined
  const matchedWard = ward ? findWard(province, ward) : undefined
  const legacy = Boolean((saved.district ?? '').trim()) || (city !== '' && !province) || (ward !== '' && !matchedWard)
  return { ...(province ? { province } : {}), ...(matchedWard ? { ward: matchedWard } : {}), legacy }
}

/** Joins the non-empty parts; `district` only appears for data saved before 07/2025. */
export function formatAddress(address: { line1?: string; ward?: string; district?: string; city?: string }): string {
  return [address.line1, address.ward, address.district, address.city].map((part) => (part ?? '').trim()).filter(Boolean).join(', ')
}
