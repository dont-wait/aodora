import { useEffect, useId, useMemo, useRef, useState, type KeyboardEvent } from 'react'
import { foldName, searchOptions } from '@/features/address/divisions'

export interface SearchSelectOption { value: string; label: string }

interface Props {
  id: string
  options: SearchSelectOption[]
  value: string
  onChange: (value: string) => void
  placeholder?: string
  required?: boolean
  disabled?: boolean
  emptyText?: string
  name?: string
  autoComplete?: string
  className?: string
  'aria-invalid'?: boolean
}

/** Type-ahead combobox that only accepts values from `options`. */
export default function SearchSelect({
  id, options, value, onChange, placeholder = '', required = false, disabled = false, emptyText = 'Không tìm thấy kết quả',
  name, autoComplete = 'off', className = '', 'aria-invalid': ariaInvalid,
}: Props) {
  const listId = `${useId()}-listbox`
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLUListElement>(null)
  const [query, setQuery] = useState<string | null>(null) // null: show the selected label
  const [open, setOpen] = useState(false)
  const [active, setActive] = useState(0)

  const selected = options.find((option) => option.value === value)
  const text = query ?? selected?.label ?? ''
  const filtered = useMemo(() => searchOptions(options, query ?? ''), [options, query])
  const activeIndex = Math.min(active, Math.max(0, filtered.length - 1))
  const isOpen = open && !disabled
  const pendingText = query !== null && query.trim() !== ''

  // A half-typed query is not a value: keep the form from submitting it.
  useEffect(() => {
    inputRef.current?.setCustomValidity(pendingText ? 'Vui lòng chọn một mục trong danh sách.' : '')
  }, [pendingText])

  useEffect(() => {
    if (!isOpen) return
    listRef.current?.querySelector('[aria-selected="true"]')?.scrollIntoView({ block: 'nearest' })
  }, [isOpen, activeIndex, filtered])

  function choose(next: string) {
    onChange(next)
    setQuery(null)
    setOpen(false)
  }

  function commitOnLeave() {
    setOpen(false)
    if (query === null) return
    const typed = foldName(query)
    if (!typed) {
      if (value) onChange('')
    } else {
      const exact = options.find((option) => foldName(option.label) === typed)
      if (exact) onChange(exact.value)
    }
    setQuery(null)
  }

  function onKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      event.preventDefault()
      if (!isOpen) { setOpen(true); return }
      if (!filtered.length) return
      const step = event.key === 'ArrowDown' ? 1 : -1
      setActive((activeIndex + step + filtered.length) % filtered.length)
    } else if (event.key === 'Enter' && isOpen) {
      event.preventDefault()
      const option = filtered[activeIndex]
      if (option) choose(option.value)
    } else if (event.key === 'Escape' && isOpen) {
      event.preventDefault()
      event.stopPropagation()
      setOpen(false)
      setQuery(null)
    }
  }

  return (
    <div
      className="relative"
      onBlur={(event) => { if (!event.currentTarget.contains(event.relatedTarget as Node | null)) commitOnLeave() }}
    >
      <input
        ref={inputRef}
        id={id}
        name={name}
        type="text"
        role="combobox"
        aria-expanded={isOpen}
        aria-controls={listId}
        aria-autocomplete="list"
        aria-activedescendant={isOpen && filtered[activeIndex] ? `${listId}-${activeIndex}` : undefined}
        aria-invalid={ariaInvalid}
        autoComplete={autoComplete}
        inputMode="search"
        placeholder={placeholder}
        required={required}
        disabled={disabled}
        value={text}
        className={`${className} ${selected && !disabled ? 'pr-16' : 'pr-10'}`}
        onFocus={() => setOpen(true)}
        onClick={() => setOpen(true)}
        onChange={(event) => { setQuery(event.target.value); setActive(0); setOpen(true) }}
        onKeyDown={onKeyDown}
      />
      {selected && !disabled && (
        <button
          type="button"
          tabIndex={-1}
          aria-label="Xóa lựa chọn"
          className="absolute right-8 top-1/2 -translate-y-1/2 px-1 text-on-surface-variant hover:text-primary"
          onMouseDown={(event) => event.preventDefault()}
          onClick={() => { choose(''); inputRef.current?.focus() }}
        >
          ×
        </button>
      )}
      <span aria-hidden="true" className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-on-surface-variant text-xs">▾</span>
      {isOpen && (
        <ul
          id={listId}
          ref={listRef}
          role="listbox"
          className="absolute left-0 right-0 top-full z-30 mt-1 max-h-72 overflow-y-auto border border-outline-variant bg-surface shadow-lg"
        >
          {filtered.length === 0 && <li role="presentation" className="px-4 py-3 text-sm text-on-surface-variant">{emptyText}</li>}
          {filtered.map((option, index) => (
            <li
              key={option.value}
              id={`${listId}-${index}`}
              role="option"
              aria-selected={index === activeIndex}
              className={`cursor-pointer px-4 py-2.5 ${index === activeIndex ? 'bg-primary text-on-primary' : option.value === value ? 'bg-surface-container' : 'hover:bg-surface-container'}`}
              onMouseDown={(event) => event.preventDefault()}
              onMouseEnter={() => setActive(index)}
              onClick={() => choose(option.value)}
            >
              {option.label}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
