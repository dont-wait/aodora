import { useState } from 'react'
import { Link } from 'react-router-dom'
import { ROUTES } from '@/data/site'
import Icon from '@/components/ui/Icon'
import StarRating from '@/components/ui/StarRating'
import type { CatalogProduct } from '@/features/catalog/types'
import { getSelectedVariant, createCartSelection, addSelectedProduct } from '@/features/catalog/selection'
import { useToast } from '@/hooks/useToast'
import { formatVND } from '@/utils/format'

export default function PurchasePanel({ product, onOpenSizeGuide }: { product: CatalogProduct; onOpenSizeGuide: () => void }) {
  const { showToast } = useToast()
  const [mode, setMode] = useState<'ready' | 'custom'>(product.ready_available ? 'ready' : 'custom')
  const first = product.variants.find((variant) => variant.available) || product.variants[0]
  const [color, setColor] = useState(first?.color || '')
  const [size, setSize] = useState(first?.size || '')
  const [quantity, setQuantity] = useState(1)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const variant = getSelectedVariant(product, color, size)
  const maxStock = variant ? Math.min(variant.stock, product.stock) : (product.variants.length ? 0 : product.stock)
  const inStock = product.ready_available && maxStock >= quantity && (variant?.available || (!product.variants.length && !!product.sku))
  const colors = [...new Set(product.variants.map((v) => v.color).filter(Boolean))]
  const sizes = [...new Set(product.variants.map((v) => v.size).filter(Boolean))]
  const compare = product.compare_at_price || product.old_price
  const validColor = (name: string) => product.colors?.find((c) => c.name === name && c.hex)?.hex

  async function addToCart() {
    setError('')
    try {
      const selection = createCartSelection(product, color, size, quantity)
      setSaving(true)
      await addSelectedProduct(selection)
      showToast('Sản phẩm đã được lưu vào giỏ hàng.')
    } catch (e) { setError(e instanceof Error ? e.message : 'Không thể thêm vào giỏ hàng.') }
    finally { setSaving(false) }
  }

  const tailoringParams = new URLSearchParams({ product_id: product.id, product_slug: product.slug, mode: 'custom' })
  return <div className="lg:col-span-5 flex flex-col">
    {product.collection && <p className="uppercase text-secondary tracking-widest text-sm">{product.collection}</p>}
    <h1 className="font-headline-lg text-headline-lg text-primary leading-tight my-4">{product.name}</h1>
    {product.material && <p className="text-on-surface-variant mb-3">Chất liệu: {product.material}</p>}
    {product.description && <p className="text-on-surface-variant mb-4">{product.description}</p>}
    <div className="flex items-center gap-2 mb-5"><StarRating value={Math.round(product.reviews.average)} />
      <a href="#danh-gia-khach-hang" className="text-sm text-on-surface-variant underline">{product.reviews.count} đánh giá</a>
    </div>
    <div className="bg-surface-container-low p-5 mb-6">
      <strong className="text-primary font-display-hero-mobile text-display-hero-mobile">{formatVND(product.price)}</strong>
      {compare !== undefined && compare > product.price && <span className="ml-3 text-on-surface-variant line-through">{formatVND(compare)}</span>}
      {product.bundle && <p className="text-on-surface-variant mt-2">{product.bundle}</p>}
    </div>
    {product.ready_available && product.bespoke && <div className="grid grid-cols-2 gap-2 mb-6">
      <button type="button" aria-pressed={mode === 'ready'} onClick={() => setMode('ready')}
        className={`px-3 py-3 ${mode === 'ready' ? 'bg-primary text-on-primary' : 'bg-surface-container'}`}>May sẵn</button>
      <button type="button" aria-pressed={mode === 'custom'} onClick={() => setMode('custom')}
        className={`px-3 py-3 ${mode === 'custom' ? 'bg-primary text-on-primary' : 'bg-surface-container'}`}>May đo theo số đo</button>
    </div>}
    {mode === 'ready' && product.ready_available && <>
      {!!colors.length && <fieldset className="mb-5"><legend className="font-semibold mb-2">Màu sắc: {color || 'Chọn màu'}</legend>
        <div className="flex flex-wrap gap-2">{colors.map((value) => {
          const hex = validColor(value)
          return <button key={value} type="button" aria-pressed={color === value}
            onClick={() => { setColor(value); setSize(product.variants.find((v) => v.color === value && v.available)?.size || ''); setQuantity(1); setError('') }}
            className={`px-3 py-2 border ${color === value ? 'border-primary ring-1 ring-primary' : 'border-outline-variant'} flex items-center gap-2`}>
            {hex && <span className="w-5 h-5 rounded-full border" style={{ backgroundColor: hex }} />}{value}</button>
        })}</div>
      </fieldset>}
      {!!sizes.length && <fieldset className="mb-5"><legend className="font-semibold mb-2">Kích cỡ: {size || 'Chọn kích cỡ'}</legend>
        <div className="flex flex-wrap gap-2">{sizes.map((value) => {
          const available = product.variants.some((v) => v.color === color && v.size === value && v.available)
          return <button key={value} type="button" aria-pressed={size === value} disabled={!available}
            onClick={() => { setSize(value); setQuantity(1); setError('') }}
            className={`min-w-12 px-3 py-2 border ${size === value ? 'border-primary bg-primary text-on-primary' : 'border-outline-variant'} disabled:opacity-40 disabled:cursor-not-allowed`}>
            {value}</button>
        })}</div>
        <button type="button" onClick={onOpenSizeGuide} className="underline text-primary mt-2 text-sm">Xem bảng kích cỡ tham khảo</button>
      </fieldset>}
      <p role="status" className="mb-4 text-on-surface-variant text-sm">{variant ? `SKU: ${variant.sku} — Còn ${maxStock} sản phẩm` : product.sku ? `SKU: ${product.sku} — Còn ${maxStock} sản phẩm` : 'Không có SKU phù hợp.'}</p>
      <div className="flex items-center gap-3 mb-4">
        <div className="flex items-center bg-surface-container px-2">
          <button type="button" aria-label="Giảm số lượng" onClick={() => setQuantity((n) => Math.max(1, n - 1))} className="p-2"><Icon name="remove" size={18} /></button>
          <span className="w-9 text-center">{quantity}</span>
          <button type="button" aria-label="Tăng số lượng" disabled={quantity >= maxStock} onClick={() => setQuantity((n) => Math.min(maxStock, n + 1))} className="p-2 disabled:opacity-40"><Icon name="add" size={18} /></button>
        </div>
        <button type="button" disabled={!inStock || saving} onClick={addToCart}
          className="flex-1 bg-primary hover:bg-primary-container disabled:opacity-50 text-on-primary py-3 px-4 uppercase text-sm">
          {saving ? 'Đang xử lý...' : 'Thêm vào giỏ'}</button>
      </div>
    </>}
    {(!product.ready_available || mode === 'custom') && <p className="mb-4 text-on-surface-variant">{product.bespoke ? 'Mẫu này có hỗ trợ tư vấn may đo.' : 'Sản phẩm hiện không còn SKU may sẵn có thể mua.'}</p>}
    {error && <p role="alert" className="mb-4 text-red-700">{error}</p>}
    {product.bespoke && <Link to={`${ROUTES.tailoring}?${tailoringParams}`} className="block bg-surface-container-high text-primary py-3 px-5 text-center mb-4 font-semibold">Xem hướng dẫn may đo cho mẫu này</Link>}
    <p className="text-xs text-on-surface-variant">Giá và tồn kho sẽ được kiểm tra lại khi đặt đơn. Giỏ hàng và yêu cầu may đo chỉ được lưu sau khi các module tương ứng tích hợp.</p>
  </div>
}
