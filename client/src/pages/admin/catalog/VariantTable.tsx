import { useRef, useState } from 'react'
import { adminCatalog, AdminApiError, type Product, type Variant } from '@/features/admin/catalog'
import { STANDARD_SIZES, isStandardSize } from '@/features/catalog/sizes'
import { Button, Card, DataTable, StatusBadge } from '../ui'
import { inputClass, tdClass, thClass } from '../uiStyles'

interface SharedProps {
  onSaved: () => Promise<void>
  onForbidden: () => void
  onMessage: (message: string, kind: 'success' | 'error') => void
}

function SizeSelect({ value, onChange, label, disabled }: { value: string; onChange: (value: string) => void; label: string; disabled?: boolean }) {
  const legacy = value !== '' && !isStandardSize(value)
  return (
    <div className="space-y-1">
      <select className={inputClass} aria-label={label} value={value} disabled={disabled} onChange={(event) => onChange(event.target.value)}>
        <option value="">Không có cỡ</option>
        {legacy && <option value={value}>{value} (cũ)</option>}
        {STANDARD_SIZES.map((size) => <option key={size} value={size}>{size}</option>)}
      </select>
      {legacy && <p className="text-xs text-amber-800">Cỡ cũ, không thuộc danh sách chuẩn. Chọn lại cỡ chuẩn để lưu.</p>}
    </div>
  )
}

function VariantRow({ productId, variant, onSaved, onForbidden, onMessage }: SharedProps & { productId: string; variant: Variant }) {
  const [color, setColor] = useState(variant.color)
  const [size, setSize] = useState(variant.size)
  const [delta, setDelta] = useState('')
  const [busy, setBusy] = useState(false)
  const pendingStockRequest = useRef<string | null>(null)
  const soldOut = variant.stock <= 0
  async function act(action: 'details' | 'stock') {
    setBusy(true)
    try {
      if (action === 'details') {
        await adminCatalog.updateVariant(productId, variant.sku, { color, size })
        onMessage('Đã lưu màu/cỡ của ' + variant.sku, 'success')
      } else {
        const n = Number(delta)
        if (!Number.isSafeInteger(n) || n === 0) throw new Error('Nhập số nguyên khác 0, ví dụ -2 hoặc 5')
        // The same request_id is reused until it succeeds, so a retried network call never applies twice.
        const requestId = pendingStockRequest.current || crypto.randomUUID()
        pendingStockRequest.current = requestId
        await adminCatalog.adjustStock(productId, variant.sku, n, requestId)
        pendingStockRequest.current = null
        setDelta('')
        onMessage('Đã điều chỉnh tồn của ' + variant.sku, 'success')
      }
      await onSaved()
    } catch (err) {
      if (err instanceof AdminApiError && (err.status === 401 || err.status === 403)) onForbidden()
      else onMessage(err instanceof Error ? err.message : 'Không thể lưu', 'error')
    } finally {
      setBusy(false)
    }
  }
  const changed = color !== variant.color || size !== variant.size
  return (
    <tr className={soldOut ? 'bg-surface-container-low text-on-surface-variant' : ''}>
      <td className={tdClass}><span className="font-mono text-xs">{variant.sku}</span></td>
      <td className={tdClass}><input className={`${inputClass} min-w-28`} aria-label={`Màu ${variant.sku}`} value={color} onChange={(event) => setColor(event.target.value)} /></td>
      <td className={tdClass}><SizeSelect label={`Cỡ ${variant.sku}`} value={size} onChange={setSize} /></td>
      <td className={tdClass}>
        <span className="font-semibold">{variant.stock}</span>
        {soldOut && <StatusBadge tone="danger" className="ml-2">Hết hàng</StatusBadge>}
      </td>
      <td className={tdClass}>
        <div className="flex gap-2">
          <input
            className={`${inputClass} w-24`}
            aria-label={`Điều chỉnh tồn ${variant.sku}`}
            placeholder="+5 / -2"
            value={delta}
            onChange={(event) => { pendingStockRequest.current = null; setDelta(event.target.value) }}
          />
          <Button variant="secondary" disabled={busy || delta.trim() === ''} onClick={() => void act('stock')}>Cộng/trừ</Button>
        </div>
      </td>
      <td className={tdClass}><Button disabled={busy || !changed} onClick={() => void act('details')}>Lưu màu/cỡ</Button></td>
    </tr>
  )
}

export default function VariantTable({ product, onSaved, onForbidden, onMessage }: SharedProps & { product: Product }) {
  const [adding, setAdding] = useState({ sku: '', color: '', size: '' })
  const [busy, setBusy] = useState(false)
  async function addSKU() {
    setBusy(true)
    try {
      await adminCatalog.addVariant(product.id, { ...adding, stock: 0 })
      await onSaved()
      setAdding({ sku: '', color: '', size: '' })
      onMessage('Đã thêm SKU, tồn kho ban đầu bằng 0', 'success')
    } catch (err) {
      if (err instanceof AdminApiError && (err.status === 401 || err.status === 403)) onForbidden()
      else onMessage(err instanceof Error ? err.message : 'Không thể thêm SKU', 'error')
    } finally {
      setBusy(false)
    }
  }
  const variants = product.variants || []
  return (
    <Card title={'SKU & tồn kho — tổng ' + product.stock} description="Điều chỉnh tồn bằng số chênh lệch, không nhập đè số tồn cũ.">
      <DataTable label="SKU và tồn kho">
        <thead>
          <tr>
            <th className={thClass}>SKU</th><th className={thClass}>Màu</th><th className={thClass}>Cỡ</th><th className={thClass}>Tồn</th>
            <th className={thClass}>Điều chỉnh tồn</th><th className={thClass}>Hành động</th>
          </tr>
        </thead>
        <tbody>
          {variants.length === 0 && <tr><td className={tdClass} colSpan={6}>Sản phẩm chưa có SKU.</td></tr>}
          {variants.map((variant) => (
            <VariantRow key={product.id + ':' + variant.sku + ':' + variant.color + ':' + variant.size} productId={product.id} variant={variant} onSaved={onSaved} onForbidden={onForbidden} onMessage={onMessage} />
          ))}
        </tbody>
      </DataTable>
      <div className="border-t border-outline-variant/40 pt-4 space-y-3">
        <h3 className="font-bold">Thêm SKU (tồn = 0)</h3>
        <div className="grid gap-2 md:grid-cols-3">
          <input className={inputClass} aria-label="SKU mới" placeholder="SKU" value={adding.sku} onChange={(event) => setAdding({ ...adding, sku: event.target.value })} />
          <input className={inputClass} aria-label="Màu của SKU mới" placeholder="Màu" value={adding.color} onChange={(event) => setAdding({ ...adding, color: event.target.value })} />
          <SizeSelect label="Cỡ của SKU mới" value={adding.size} onChange={(size) => setAdding({ ...adding, size })} />
        </div>
        <Button disabled={busy} onClick={() => void addSKU()}>Thêm SKU</Button>
      </div>
    </Card>
  )
}
