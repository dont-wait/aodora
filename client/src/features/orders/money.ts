// Quote/Order Money is a JSON number on the wire. Quote all numeric lexemes
// BEFORE JSON.parse; only counts and quantities are converted back to numbers.
// String tokens (including escaped quotes and numbers in names) stay untouched.
export function parseExactJSON(text: string): unknown {
  return JSON.parse(
    text.replace(/("(?:\\.|[^"\\])*")|(-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)/g, (token, stringToken: string | undefined) =>
      stringToken ? token : JSON.stringify(token),
    ),
  )
}

export function formatOrderMoney(value: string): string {
  if (!/^\d+(?:\.\d+)?$/.test(value)) return '—'
  const [integer, fraction] = value.split('.')
  return `${integer.replace(/\B(?=(\d{3})+(?!\d))/g, '.')}${fraction === undefined ? '' : `,${fraction}`} ₫`
}
