import { useRef, useState } from 'react'
import { adminCatalog, AdminApiError, type Product, type Variant } from '@/features/admin/catalog'
import { normalizeSku, mapServerField, validateAdjustment, validateNewSku, type Errors } from '@/features/admin/validation'
import { STANDARD_SIZES, isStandardSize } from '@/features/catalog/sizes'
import { Button, Card, DataTable, StatusBadge } from '../ui'
import { inputClass, tdClass, tdNumClass, thClass, thNumClass } from '../uiStyles'

interface SharedProps {
  onSaved: () => Promise<void>
  onForbidden: () => void
  onMessage: (message: string, kind: 'success' | 'error') => void
}

function SizeSelect({ value, onChange, label, disabled, id }: { value: string; onChange: (value: string) => void; label: string; disabled?: boolean; id?: string }) {
  const legacy = value !== '' && !isStandardSize(value)
  return (
    <div className="space-y-1">
      <select id={id} className={inputClass} aria-label={label} value={value} disabled={disabled} onChange={(event) => onChange(event.target.value)}>
        <option value="">Không có cỡ</option>
        {legacy && <option value={value}>{value} (cũ)</option>}
        {STANDARD_SIZES.map((size) => <option key={size} value={size}>{size}</option>)}
      </select>
      {legacy && <p className="text-xs text-amber-800">Cỡ cũ, không thuộc danh sách chuẩn. Chọn lại cỡ chuẩn để lưu.</p>}
    </div>
  )
}

/** Colour of a SKU: always one of the product's colours (a colour outside the list is shown flagged). */
function ColorSelect({ value, options, onChange, label, id, invalid }: { value: string; options: string[]; onChange: (v: string) => void; label: string; id?: string; invalid?: boolean }) {
  const known = options.some((o) => o.toLowerCase() === value.toLowerCase())
  return (
    <select id={id} className={inputClass + ' min-w-28'} aria-label={label} aria-invalid={invalid || undefined} value={value} onChange={(event) => onChange(event.target.value)}>
      {value === '' && <option value="">Chọn màu</option>}
      {!known && value !== '' && <option value={value}>{value} (ngoài danh sách)</option>}
      {options.map((o) => <option key={o} value={o}>{o}</option>)}
    </select>
  )
}

function fieldErrors(err: unknown, fallback: string): { errors: Errors; general: string } {
  if (err instanceof AdminApiError && err.field) return { errors: { [mapServerField(err.field)]: err.message }, general: '' }
  return { errors: {}, general: err instanceof Error ? err.message : fallback }
}

function VariantRow({ productId, variant, colorOptions, onSaved, onForbidden, onMessage }: SharedProps & { productId: string; variant: Variant; colorOptions: string[] }) {
  const [color, setColor] = useState(variant.color)
  const [size, setSize] = useState(variant.size)
  const [delta, setDelta] = useState('')
  const [errors, setErrors] = useState<Errors>({})
  const [busy, setBusy] = useState(false)
  const pendingStockRequest = useRef<string | null>(null)
  const soldOut = variant.stock <= 0
  async function act(action: 'details' | 'stock') {
    setErrors({})
    if (action === 'stock') {
      const problem = validateAdjustment(delta, variant.stock)
      if (problem) { setErrors({ delta: problem }); return }
    }
    setBusy(true)
    try {
      if (action === 'details') {
        await adminCatalog.updateVariant(productId, variant.sku, { color, size })
        onMessage('Đã lưu màu/cỡ của ' + variant.sku, 'success')
      } else {
        // The same request_id is reused until it succeeds, so a retried network call never applies twice.
        const requestId = pendingStockRequest.current || crypto.randomUUID()
        pendingStockRequest.current = requestId
        await adminCatalog.adjustStock(productId, variant.sku, Number(delta), requestId)
        pendingStockRequest.current = null
        setDelta('')
        onMessage('Đã điều chỉnh tồn của ' + variant.sku, 'success')
      }
      await onSaved()
    } catch (err) {
      if (err instanceof AdminApiError && (err.status === 401 || err.status === 403)) onForbidden()
      else {
        const { errors: mapped, general } = fieldErrors(err, 'Không thể lưu')
        if (err instanceof AdminApiError && err.field === 'delta') pendingStockRequest.current = null
        setErrors(mapped)
        if (general) onMessage(general, 'error')
      }
    } finally {
      setBusy(false)
    }
  }
  const changed = color !== variant.color || size !== variant.size
  const idBase = 'sku-' + variant.sku
  return (
    <tr className={soldOut ? 'bg-surface-container-low text-on-surface-variant' : ''}>
      <td className={tdClass}><span className="font-mono text-xs">{variant.sku}</span></td>
      <td className={tdClass}>
        <ColorSelect id={idBase + '-color'} label={'Màu ' + variant.sku} value={color} options={colorOptions} invalid={!!errors.color} onChange={setColor} />
        {errors.color && <p role="alert" className="mt-1 text-xs text-red-700">{errors.color}</p>}
      </td>
      <td className={tdClass}><SizeSelect label={'Cỡ ' + variant.sku} value={size} onChange={setSize} />{errors.size && <p role="alert" className="mt-1 text-xs text-red-700">{errors.size}</p>}</td>
      <td className={tdNumClass}>
        {soldOut && <StatusBadge tone="danger" className="mr-2">Hết hàng</StatusBadge>}
        <span className="font-semibold">{variant.stock}</span>
      </td>
      <td className={tdClass}>
        <div className="flex gap-2">
          <input
            id={idBase + '-delta'}
            className={inputClass + ' w-24'}
            aria-label={'Điều chỉnh tồn ' + variant.sku}
            aria-invalid={errors.delta ? true : undefined}
            aria-describedby={errors.delta ? idBase + '-delta-error' : undefined}
            placeholder="+5 / -2"
            value={delta}
            onChange={(event) => { pendingStockRequest.current = null; setDelta(event.target.value); setErrors({}) }}
          />
          <Button variant="secondary" disabled={delta.trim() === ''} loading={busy} onClick={() => void act('stock')}>Cộng/trừ</Button>
        </div>
        {errors.delta && <p id={idBase + '-delta-error'} role="alert" className="mt-1 text-xs text-red-700">{errors.delta}</p>}
      </td>
      <td className={tdClass}><Button disabled={!changed} loading={busy} onClick={() => void act('details')}>Lưu màu/cỡ</Button></td>
    </tr>
  )
}

export default function VariantTable({ product, colorOptions, onSaved, onForbidden, onMessage }: SharedProps & { product: Product; colorOptions: string[] }) {
  const [adding, setAdding] = useState({ sku: '', color: '', size: '' })
  const [errors, setErrors] = useState<Errors>({})
  const [busy, setBusy] = useState(false)
  const variants = product.variants || []
  async function addSKU() {
    const local = validateNewSku({ ...adding, sku: normalizeSku(adding.sku) }, colorOptions, variants)
    setErrors(local)
    if (Object.keys(local).length > 0) return
    setBusy(true)
    try {
      await adminCatalog.addVariant(product.id, { sku: normalizeSku(adding.sku), color: adding.color, size: adding.size, stock: 0 })
      await onSaved()
      setAdding({ sku: '', color: '', size: '' })
      onMessage('Đã thêm SKU, tồn kho ban đầu bằng 0', 'success')
    } catch (err) {
      if (err instanceof AdminApiError && (err.status === 401 || err.status === 403)) onForbidden()
      else {
        const { errors: mapped, general } = fieldErrors(err, 'Không thể thêm SKU')
        setErrors(mapped)
        if (general) onMessage(general, 'error')
      }
    } finally {
      setBusy(false)
    }
  }
  return (
    <Card title={'SKU & tồn kho — tổng ' + product.stock} description="Điều chỉnh tồn bằng số chênh lệch, không nhập đè số tồn cũ. Không thể trừ quá số đang có.">
      <DataTable label="SKU và tồn kho">
        <thead>
          <tr>
            <th className={thClass}>SKU</th><th className={thClass}>Màu</th><th className={thClass}>Cỡ</th><th className={thNumClass}>Tồn</th>
            <th className={thClass}>Điều chỉnh tồn</th><th className={thClass}>Hành động</th>
          </tr>
        </thead>
        <tbody>
          {variants.length === 0 && <tr><td className={tdClass} colSpan={6}>Sản phẩm chưa có SKU.</td></tr>}
          {variants.map((variant) => (
            <VariantRow key={product.id + ':' + variant.sku + ':' + variant.color + ':' + variant.size} productId={product.id} variant={variant} colorOptions={colorOptions} onSaved={onSaved} onForbidden={onForbidden} onMessage={onMessage} />
          ))}
        </tbody>
      </DataTable>
      <div className="space-y-3 border-t border-outline-variant/40 pt-4">
        <h3 className="font-bold">Thêm SKU (tồn = 0)</h3>
        <div className="grid items-start gap-3 md:grid-cols-3">
          <div>
            <input id="new-sku" className={inputClass + ' font-mono'} aria-label="SKU mới" aria-invalid={errors.sku ? true : undefined} placeholder="Ví dụ: AO-DO-M" value={adding.sku} onChange={(event) => setAdding({ ...adding, sku: event.target.value.toUpperCase() })} />
            {errors.sku && <p role="alert" className="mt-1 text-xs text-red-700">{errors.sku}</p>}
          </div>
          <div>
            <ColorSelect id="new-sku-color" label="Màu của SKU mới" value={adding.color} options={colorOptions} invalid={!!errors.color} onChange={(color) => setAdding({ ...adding, color })} />
            {errors.color && <p role="alert" className="mt-1 text-xs text-red-700">{errors.color}</p>}
            {colorOptions.length === 0 && <p className="mt-1 text-xs text-amber-800">Thêm màu cho sản phẩm trước khi tạo SKU.</p>}
          </div>
          <div>
            <SizeSelect label="Cỡ của SKU mới" value={adding.size} onChange={(size) => setAdding({ ...adding, size })} />
            {errors.size && <p role="alert" className="mt-1 text-xs text-red-700">{errors.size}</p>}
          </div>
        </div>
        <Button loading={busy} disabled={colorOptions.length === 0} onClick={() => void addSKU()}>Thêm SKU</Button>
      </div>
    </Card>
  )
}
