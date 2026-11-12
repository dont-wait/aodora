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
