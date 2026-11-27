import { useMemo, useState } from 'react'
import type { Product } from '@/features/admin/catalog'
import { fold } from '@/features/admin/productSearch'
import { Button, StatusBadge } from '../ui'
import { inputClass } from '../uiStyles'

interface Props {
  products: Product[]
  selectedId: string
  page: number
  totalPages: number
  onSelect: (id: string) => void
  onPage: (page: number) => void
}

export default function ProductList({ products, selectedId, page, totalPages, onSelect, onPage }: Props) {
  const [query, setQuery] = useState('')
  const visible = useMemo(() => {
    const key = fold(query)
    return key ? products.filter((row) => fold(row.name).includes(key) || fold(row.slug).includes(key)) : products
  }, [products, query])
  return (
    <div className="space-y-3">
      <div className="flex gap-2">
        <input className={inputClass} type="search" aria-label="Tìm sản phẩm trên trang này" placeholder="Tìm theo tên hoặc slug..." value={query} onChange={(event) => setQuery(event.target.value)} />
        <Button onClick={() => onSelect('')}>+ Mới</Button>
      </div>
      <ul className="space-y-2 max-h-[70vh] overflow-y-auto" aria-label="Danh sách sản phẩm">
        {visible.length === 0 && <li className="text-sm text-on-surface-variant py-4 text-center">Không có sản phẩm khớp trên trang này.</li>}
        {visible.map((row) => {
          const src = row.image?.src || row.images?.[0] || ''
          return (
            <li key={row.id}>
              <button
                type="button"
                onClick={() => onSelect(row.id)}
                aria-current={row.id === selectedId ? 'true' : undefined}
                className={`flex w-full items-center gap-3 border p-2 text-left transition-colors ${row.id === selectedId ? 'border-primary bg-surface-container' : 'border-outline-variant/40 hover:bg-surface-container-low'}`}
              >
                <span className="h-14 w-11 shrink-0 overflow-hidden bg-surface-container">
                  {src ? <img src={src} alt="" loading="lazy" className="h-full w-full object-cover" onError={(event) => { event.currentTarget.style.display = 'none' }} /> : null}
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate font-semibold text-sm">{row.name}</span>
                  <span className="block truncate text-xs text-on-surface-variant">{row.slug}</span>
                  <span className="mt-1 flex flex-wrap items-center gap-1.5">
                    <StatusBadge tone={row.is_active ? 'success' : 'neutral'}>{row.is_active ? 'Đang bán' : 'Đang ẩn'}</StatusBadge>
                    <span className="text-xs text-on-surface-variant">Tồn {row.stock}</span>
                  </span>
                </span>
              </button>
            </li>
          )
        })}
      </ul>
      <div className="flex items-center justify-between text-sm">
        <Button variant="ghost" disabled={page <= 1} onClick={() => onPage(page - 1)}>Trước</Button>
        <span>{page}/{totalPages}</span>
        <Button variant="ghost" disabled={page >= totalPages} onClick={() => onPage(page + 1)}>Sau</Button>
      </div>
    </div>
  )
}
