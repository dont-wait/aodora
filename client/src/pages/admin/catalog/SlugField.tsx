import { useState } from 'react'
import { fieldId } from '@/features/admin/validation'
import { Button } from '../ui'
import { inputClass } from '../uiStyles'

interface Props {
  slug: string
  onChange: (slug: string) => void
  /** Existing record: changing the slug breaks old links. */
  originalSlug?: string
  error?: string
  /** URL prefix shown to the user, e.g. "/san-pham/". */
  prefix: string
  label?: string
}

/** Shows "Đường dẫn: /san-pham/<slug>" with an "Sửa" button (advanced); the slug itself is generated from the name. */
export default function SlugField({ slug, onChange, originalSlug, error, prefix, label = 'Đường dẫn' }: Props) {
  const [editing, setEditing] = useState(false)
  const id = fieldId('slug')
  const changed = !!originalSlug && slug !== originalSlug
  const showInput = editing || !!error
  return (
    <div className="space-y-1.5">
      <div className="flex flex-wrap items-center gap-2 text-[13px]">
        <span className="font-medium text-on-surface-variant">{label}:</span>
        <code className="rounded bg-surface-container px-2 py-0.5 text-xs">{prefix}{slug || '…'}</code>
        {!showInput && <Button variant="ghost" className="!h-8 !px-2 text-xs" onClick={() => setEditing(true)}>Sửa</Button>}
      </div>
      {showInput && (
        <input
          id={id}
          className={inputClass + ' font-mono'}
          aria-label="Sửa đường dẫn"
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? id + '-error' : undefined}
          value={slug}
          onChange={(e) => onChange(e.target.value.toLowerCase())}
        />
      )}
      {error && <p id={id + '-error'} role="alert" className="text-xs text-red-700">{error}</p>}
      {changed && <p role="status" className="text-xs text-amber-800">Đổi đường dẫn sẽ làm hỏng các liên kết cũ tới mục này (đã chia sẻ hoặc được Google lưu).</p>}
    </div>
  )
}
