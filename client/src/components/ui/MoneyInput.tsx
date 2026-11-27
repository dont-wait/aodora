import { useLayoutEffect, useRef, type InputHTMLAttributes } from 'react'
import { formatMoney, maskMoneyInput } from '@/features/catalog/prices'

type Props = Omit<InputHTMLAttributes<HTMLInputElement>, 'value' | 'onChange' | 'type' | 'defaultValue'> & {
  value: number | null
  onChange: (value: number | null) => void
}

/**
 * Whole-VND input shared by the storefront and admin: the user types digits only,
 * thousands dots appear as they type, "800k" / "2tr" are expanded, and the caret
 * does not jump to the end when editing in the middle.
 */
export default function MoneyInput({ value, onChange, ...rest }: Props) {
  const ref = useRef<HTMLInputElement>(null)
  const caret = useRef<number | null>(null)
  const text = formatMoney(value)
  useLayoutEffect(() => {
    const el = ref.current
    if (el && caret.current !== null && document.activeElement === el) el.setSelectionRange(caret.current, caret.current)
    caret.current = null
  })
  return (
    <input
      {...rest}
      ref={ref}
      type="text"
      inputMode="numeric"
      autoComplete="off"
      value={text}
      onChange={(event) => {
        const masked = maskMoneyInput(event.target.value, event.target.selectionStart ?? event.target.value.length)
        caret.current = masked.caret
        if (masked.value === value) {
          // Rejected keystroke (a letter, a typed dot): React keeps the old text; restore the caret.
          event.target.value = masked.text
          event.target.setSelectionRange(masked.caret, masked.caret)
        }
        onChange(masked.value)
      }}
    />
  )
}
