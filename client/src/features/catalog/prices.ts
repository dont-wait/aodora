// Currency inputs are display-formatted only. API parameters remain integer VND.
export function parseVNDInput(value: string): string | null {
  const input = value.trim()
  if (!input) return ''
  if (!/^\d+$/.test(input) && !/^\d{1,3}(?:[.,\s]\d{3})+$/.test(input)) return null
  const digits = input.replace(/[.,\s]/g, '')
  if (!digits || !Number.isSafeInteger(Number(digits))) return null
  return String(Number(digits))
}

export function formatVNDInput(value: string): string {
  const normalized = parseVNDInput(value)
  return normalized ? Number(normalized).toLocaleString('vi-VN') : normalized === '' ? '' : value
}

const MAX_MONEY_DIGITS = 13 // up to 9.999.999.999.999 ₫, well inside Number.MAX_SAFE_INTEGER

/** 1250000 → "1.250.000" (fixed Vietnamese grouping, independent of the browser locale). */
export function formatMoney(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return ''
  return String(Math.trunc(value)).replace(/\B(?=(\d{3})+(?!\d))/g, '.')
}

export interface MaskedMoney { text: string; caret: number; value: number | null }

/**
 * Formats what the user is typing into a money field: dots are inserted
 * automatically and the caret stays after the same digit. "800k" becomes
 * 800.000 and "2tr" / "2m" becomes 2.000.000 as soon as the suffix is typed.
 */
export function maskMoneyInput(raw: string, caret = raw.length): MaskedMoney {
  const suffix = raw.trim().toLowerCase().match(/(k|tr|triệu|m)$/)?.[1]
  const allDigits = raw.replace(/\D/g, '')
  const leadingZeros = allDigits.length - allDigits.replace(/^0+(?=\d)/, '').length
  const digits = allDigits.slice(leadingZeros, leadingZeros + MAX_MONEY_DIGITS)
  if (!digits) return { text: '', caret: 0, value: null }
  if (suffix) {
    const value = Number(digits) * (suffix === 'k' ? 1_000 : 1_000_000)
    if (Number.isSafeInteger(value)) {
      const text = formatMoney(value)
      return { text, caret: text.length, value }
    }
  }
  const value = Number(digits)
  const text = formatMoney(value)
  // Put the caret after the same digit it followed before re-formatting.
  let wanted = Math.max(0, raw.slice(0, caret).replace(/\D/g, '').length - leadingZeros)
  let next = 0
  while (next < text.length && wanted > 0) {
    if (/\d/.test(text[next])) wanted--
    next++
  }
  return { text, caret: next, value }
}
