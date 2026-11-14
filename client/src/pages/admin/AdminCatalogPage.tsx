import { useCallback, useEffect, useRef, useState, type FormEvent, type ReactNode } from 'react'
import { Link, Navigate } from 'react-router-dom'
import ProductContentEditor from './ProductContentEditor'
import type { MetadataWrite } from '@/features/admin/content'
import { adminCatalog, AdminApiError, editableGallery, parseImageLines, parseVariantLines,
  type Category, type CategoryInput, type Product, type ProductInput, type Variant } from '@/features/admin/catalog'

const emptyCategory: CategoryInput = { name: '', slug: '', description: '', parent_id: '', is_active: true, sort_order: 0, image: null }
const emptyProduct: ProductInput = {
  name: '', slug: '', category_id: '', description: '', material: '', price: 0,
  compare_at_price: null, is_active: true, images: [], image: null, gallery: [], colors: [], accessories: [],
}
const inputClass = 'w-full p-2.5 border border-outline/40 bg-white text-on-surface rounded-sm'
const buttonClass = 'bg-primary px-4 py-2 text-on-primary disabled:opacity-50 rounded-sm'

function Entry({ title, children }: { title: string; children: ReactNode }) {
  return <label className="flex flex-col gap-1.5 text-sm font-medium">{title}{children}</label>
}
function VariantRow({ productId, variant, onSaved, onForbidden }: { productId: string; variant: Variant; onSaved: () => Promise<void>; onForbidden: () => void }) {
  const [color, setColor] = useState(variant.color)
  const [size, setSize] = useState(variant.size)
  const [delta, setDelta] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const pendingStockRequest = useRef<string | null>(null)
  async function act(action: 'details' | 'stock') {
    setBusy(true); setError('')
    try {
      if (action === 'details') await adminCatalog.updateVariant(productId, variant.sku, { color, size })
      else {
        const n = Number(delta)
        if (!Number.isSafeInteger(n) || n === 0) throw new Error('Nhập số nguyên khác 0, ví dụ -2 hoặc 5')
        const requestId = pendingStockRequest.current || crypto.randomUUID()
        pendingStockRequest.current = requestId
        await adminCatalog.adjustStock(productId, variant.sku, n, requestId)
        pendingStockRequest.current = null
        setDelta('')
      }
      await onSaved()
    } catch (err) {
      if (err instanceof AdminApiError && (err.status === 401 || err.status === 403)) onForbidden()
      else setError(err instanceof Error ? err.message : 'Không thể lưu')
    }
    finally { setBusy(false) }
  }
  return <div className="border border-outline/30 p-3 flex flex-col gap-2 text-sm">
    <strong>{variant.sku} — còn {variant.stock}</strong>
    <div className="flex gap-2 flex-wrap">
      <input className={`${inputClass} max-w-36`} aria-label={`Màu ${variant.sku}`} value={color} onChange={(event) => setColor(event.target.value)} />
      <input className={`${inputClass} max-w-24`} aria-label={`Cỡ ${variant.sku}`} value={size} onChange={(event) => setSize(event.target.value)} />
      <button className={buttonClass} disabled={busy} onClick={() => void act('details')} type="button">Lưu màu/cỡ</button>
    </div>
    <div className="flex gap-2 flex-wrap items-center">
      <input className={`${inputClass} max-w-36`} aria-label={`Điều chỉnh tồn ${variant.sku}`} placeholder="+5 hoặc -2" value={delta} onChange={(event) => { pendingStockRequest.current = null; setDelta(event.target.value) }} />
      <button className={buttonClass} disabled={busy} onClick={() => void act('stock')} type="button">Cộng/trừ tồn</button>
    </div>
    {error && <p className="text-red-700">{error}</p>}
  </div>
}

export default function AdminCatalogPage() {
  const [categories, setCategories] = useState<Category[]>([])
  const [products, setProducts] = useState<Product[]>([])
  const [productPage, setProductPage] = useState(1)
  const [totalPages, setTotalPages] = useState(1)
  const [accessDenied, setAccessDenied] = useState(false)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [tab, setTab] = useState<'category' | 'product'>('product')
  const [categoryId, setCategoryId] = useState('')
  const [productId, setProductId] = useState('')
  const [category, setCategory] = useState<CategoryInput>(emptyCategory)
  const [product, setProduct] = useState<ProductInput>(emptyProduct)
  const [variantsText, setVariantsText] = useState('')
  const [galleryText, setGalleryText] = useState('')
  const [metadataWrites, setMetadataWrites] = useState<(MetadataWrite & { productId: string })[]>([])
  const [colorsText, setColorsText] = useState('')
  const [accessoriesText, setAccessoriesText] = useState('')
  const [newSKU, setNewSKU] = useState({ sku: '', color: '', size: '', stock: 0 })

  // One sequence for effects, manual refreshes and post-write reloads.
  // A response from a previous page is never allowed to overwrite newer data.
  const requestSequence = useRef(0)
  const load = useCallback((page = productPage): Promise<void> => {
    const current = ++requestSequence.current
    return Promise.all([adminCatalog.categories(), adminCatalog.products(page)])
      .then(([nextCategories, nextProducts]) => {
        if (current !== requestSequence.current) return
        setCategories(nextCategories)
        setProducts(nextProducts.items)
        setTotalPages(nextProducts.total_pages || 1)
        setAccessDenied(false)
        setError('')
      })
      .catch((err: unknown) => {
        if (current !== requestSequence.current) return
        setError(err instanceof Error ? err.message : 'Không thể tải dữ liệu')
        if (err instanceof AdminApiError && [401, 403].includes(err.status)) setAccessDenied(true)
      })
      .finally(() => {
        if (current === requestSequence.current) setLoading(false)
      })
  }, [productPage])
  useEffect(() => {
    void load(productPage)
    return () => { requestSequence.current += 1 }
  }, [load, productPage])
  function refresh() {
    setLoading(true)
    void load()
  }

  function selectCategory(id: string) {
    setError(''); setSuccess(''); setCategoryId(id)
    const value = categories.find((row) => row.id === id)
    setCategory(value ? { name: value.name, slug: value.slug, description: value.description || '', parent_id: value.parent_id || '', is_active: value.is_active, sort_order: value.sort_order || 0, image: value.image || null, expected_version: value.version ?? 0 } : { ...emptyCategory })
  }
  function selectProduct(id: string) {
    if (id !== productId) setMetadataWrites([])
    setError(''); setSuccess(''); setProductId(id)
    const value = products.find((row) => row.id === id)
    setProduct(value ? {
      name: value.name, slug: value.slug, category_id: value.category_id, description: value.description || '', material: value.material || '',
      price: Number(value.price), compare_at_price: value.compare_at_price == null ? (value.old_price == null ? null : Number(value.old_price)) : Number(value.compare_at_price),
      is_active: value.is_active, images: value.images || [], image: value.image || (value.images?.[0] ? { src: value.images[0], alt: value.name } : null), expected_version: value.version ?? 0,
      gallery: value.gallery || [], colors: value.colors || [], accessories: value.accessories || [],
    } : { ...emptyProduct })
    setVariantsText('')
    setGalleryText(value ? editableGallery(value).map((item) => `${item.src} | ${item.alt}`).join('\n') : '')
    setColorsText(JSON.stringify(value?.colors || [], null, 2))
    setAccessoriesText(JSON.stringify(value?.accessories || [], null, 2))
  }
  async function submitCategory(event: FormEvent) {
    event.preventDefault();setBusy(true);setError('');setSuccess('')
    try {
      if (categoryId) {
        const updated = await adminCatalog.updateCategory(categoryId, category)
        setCategory((current) => ({ ...current, expected_version: updated.version ?? 0 }))
      } else await adminCatalog.createCategory(category)
      await load();setSuccess('Đã lưu danh mục vào MongoDB')
    } catch (err) { if (err instanceof AdminApiError && [401, 403].includes(err.status)) setAccessDenied(true); else setError(err instanceof Error ? err.message : 'Không thể lưu danh mục') }
    finally {setBusy(false)}
  }
  async function submitProduct(event: FormEvent) {
    event.preventDefault();setBusy(true);setError('');setSuccess('')
    try {
      const gallery = parseImageLines(galleryText)
      const colors = JSON.parse(colorsText || '[]') as ProductInput['colors']
      const accessories = JSON.parse(accessoriesText || '[]') as ProductInput['accessories']
      if (!Array.isArray(colors) || !Array.isArray(accessories)) throw new Error('Màu/phụ kiện phải là mảng JSON')
      const images = [...new Set([...(product.image?.src ? [product.image.src] : []), ...gallery.map((item) => item.src)])]
      const data = { ...product, images, gallery, colors, accessories }
      if (productId) {
        const updated = await adminCatalog.updateProduct(productId, data)
        setMetadataWrites((writes) => [...writes, { productId, from: data.expected_version ?? 0, to: updated.version ?? 0 }])
        setProduct((current) => ({ ...current, expected_version: updated.version ?? 0 }))
      } else await adminCatalog.createProduct({ ...data, variants: parseVariantLines(variantsText) })
      await load();setSuccess('Đã lưu sản phẩm vào MongoDB')
    } catch (err) { if (err instanceof AdminApiError && [401, 403].includes(err.status)) setAccessDenied(true); else setError(err instanceof Error ? err.message : 'Không thể lưu sản phẩm') }
    finally {setBusy(false)}
  }
  async function addSKU() {
    if (!productId) return
    setBusy(true);setError('')
    try { await adminCatalog.addVariant(productId, newSKU);await load();setNewSKU({ sku: '', color: '', size: '', stock: 0 });setSuccess('Đã thêm SKU, tồn kho ban đầu bằng 0') }
    catch (err) { if (err instanceof AdminApiError && [401, 403].includes(err.status)) setAccessDenied(true); else setError(err instanceof Error ? err.message : 'Không thể thêm SKU') }
    finally {setBusy(false)}
  }
  const chosen = products.find((row) => row.id === productId)
  const onForbidden = useCallback(() => setAccessDenied(true), [])
  if (accessDenied) return <Navigate to="/403" replace />
  return <main className="min-h-screen bg-background px-5 py-10 text-on-surface">
    <div className="max-w-6xl mx-auto space-y-7">
      <div className="flex items-center justify-between gap-4"><div><p className="text-secondary uppercase tracking-wider text-sm">Aodora · Chủ cửa hàng</p><h1 className="text-3xl font-title-editorial text-primary">Quản trị danh mục & sản phẩm</h1></div><Link className="underline" to="/">Trang khách</Link></div>
      {loading && <p role="status">Đang kiểm tra quyền admin và tải dữ liệu...</p>}
      {!loading && <>
        <div className="flex gap-3"><button type="button" className={buttonClass} onClick={() => setTab('product')}>Sản phẩm</button><button type="button" className={buttonClass} onClick={() => setTab('category')}>Danh mục</button><button type="button" className="underline" onClick={refresh}>Tải lại</button></div>
        {tab === 'category' ? <div className="grid md:grid-cols-[17rem_1fr] gap-6">
          <aside className="bg-surface p-4 space-y-2"><h2 className="font-bold">Danh mục ({categories.length})</h2><button onClick={() => selectCategory('')} className="underline" type="button">+ Tạo mới</button>{categories.map((row) => <button key={row.id} type="button" onClick={() => selectCategory(row.id)} className={`block w-full text-left border p-2 ${row.id === categoryId ? 'border-primary' : 'border-outline/20'}`}>{row.name} {row.is_active ? '' : '· Đang ẩn'}</button>)}</aside>
          <form className="bg-surface p-5 space-y-4" onSubmit={(e) => void submitCategory(e)}><h2 className="text-xl font-bold">{categoryId ? 'Cập nhật danh mục' : 'Tạo danh mục'}</h2>
            <div className="grid md:grid-cols-2 gap-4"><Entry title="Tên"><input className={inputClass} required value={category.name} onChange={(e) => setCategory({ ...category, name: e.target.value })}/></Entry><Entry title="Slug"><input className={inputClass} required value={category.slug} onChange={(e) => setCategory({ ...category, slug: e.target.value })}/></Entry></div>
            <Entry title="Mô tả"><textarea className={inputClass} value={category.description} onChange={(e) => setCategory({ ...category, description: e.target.value })}/></Entry>
            <div className="grid md:grid-cols-2 gap-4"><Entry title="Danh mục cha"><select className={inputClass} value={category.parent_id} onChange={(e) => setCategory({ ...category, parent_id: e.target.value })}><option value="">Không có</option>{categories.filter((v) => v.id !== categoryId).map((v) => <option key={v.id} value={v.id}>{v.name}</option>)}</select></Entry><Entry title="Thứ tự"><input className={inputClass} type="number" min="0" value={category.sort_order} onChange={(e) => setCategory({ ...category, sort_order: Number(e.target.value) })}/></Entry></div>
            <div className="grid md:grid-cols-2 gap-4"><Entry title="URL hình danh mục"><input className={inputClass} value={category.image?.src || ''} onChange={(e) => setCategory({ ...category, image: e.target.value || category.image?.alt ? { src: e.target.value, alt: category.image?.alt || '' } : null })}/></Entry><Entry title="Alt hình danh mục"><input className={inputClass} value={category.image?.alt || ''} onChange={(e) => setCategory({ ...category, image: e.target.value || category.image?.src ? { src: category.image?.src || '', alt: e.target.value } : null })}/></Entry></div>
            <label className="flex gap-2"><input type="checkbox" checked={category.is_active} onChange={(e) => setCategory({ ...category, is_active: e.target.checked })}/>Hiển thị danh mục</label><button type="submit" disabled={busy} className={buttonClass}>Lưu danh mục</button>
          </form></div> : <div className="grid md:grid-cols-[17rem_1fr] gap-6">
          <aside className="bg-surface p-4 space-y-2"><h2 className="font-bold">Sản phẩm ({products.length})</h2><button onClick={() => selectProduct('')} className="underline" type="button">+ Tạo mới</button>{products.map((row) => <button key={row.id} onClick={() => selectProduct(row.id)} type="button" className={`block w-full text-left border p-2 ${row.id === productId ? 'border-primary' : 'border-outline/20'}`}>{row.name} {row.is_active ? '' : '· Đang ẩn'}</button>)}
            <div className="flex items-center gap-3 pt-3"><button disabled={productPage <= 1} type="button" className="underline disabled:opacity-40" onClick={() => { requestSequence.current++; setLoading(true); setProductPage((v) => v - 1) }}>Trước</button><span>{productPage}/{totalPages}</span><button disabled={productPage>=totalPages} type="button" className="underline disabled:opacity-40" onClick={() => { requestSequence.current++; setLoading(true); setProductPage((v) => v + 1) }}>Sau</button></div></aside>
          <div className="space-y-5"><form className="bg-surface p-5 space-y-4" onSubmit={(e) => void submitProduct(e)}><h2 className="text-xl font-bold">{productId ? 'Thông tin bán hàng' : 'Tạo sản phẩm'}</h2>
            <div className="grid md:grid-cols-2 gap-4"><Entry title="Tên sản phẩm"><input required className={inputClass} value={product.name} onChange={(e) => setProduct({ ...product, name: e.target.value })}/></Entry><Entry title="Slug"><input required className={inputClass} value={product.slug} onChange={(e) => setProduct({ ...product, slug: e.target.value })}/></Entry></div>
            <Entry title="Danh mục"><select required className={inputClass} value={product.category_id} onChange={(e) => setProduct({ ...product, category_id: e.target.value })}><option value="">Chọn danh mục</option>{categories.map((v) => <option key={v.id} value={v.id}>{v.name}</option>)}</select></Entry>
            <Entry title="Mô tả"><textarea className={inputClass} value={product.description} onChange={(e) => setProduct({ ...product, description: e.target.value })}/></Entry>
            <div className="grid md:grid-cols-3 gap-4"><Entry title="Chất liệu"><input className={inputClass} value={product.material} onChange={(e) => setProduct({ ...product, material: e.target.value })}/></Entry><Entry title="Giá bán (VND)"><input className={inputClass} type="number" min="0" step="1" required value={product.price} onChange={(e) => setProduct({ ...product, price: Number(e.target.value) })}/></Entry><Entry title="Giá gốc (để trống nếu không có)"><input className={inputClass} type="number" min="0" step="1" value={product.compare_at_price ?? ''} onChange={(e) => setProduct({ ...product, compare_at_price: e.target.value === '' ? null : Number(e.target.value) })}/></Entry></div>
            <div className="grid md:grid-cols-2 gap-4"><Entry title="URL hình chính"><input className={inputClass} value={product.image?.src || ''} onChange={(e) => setProduct({ ...product, image: e.target.value || product.image?.alt ? { src: e.target.value, alt: product.image?.alt || '' } : null })}/></Entry><Entry title="Alt hình chính"><input className={inputClass} value={product.image?.alt || ''} onChange={(e) => setProduct({ ...product, image: e.target.value || product.image?.src ? { src: product.image?.src || '', alt: e.target.value } : null })}/></Entry></div>
            <Entry title="Ảnh phụ (mỗi dòng URL | mô tả ảnh)"><textarea className={inputClass} rows={3} value={galleryText} onChange={(e) => setGalleryText(e.target.value)}/></Entry>
            <Entry title="Mã màu JSON (name và hex)"><textarea className={inputClass} rows={3} value={colorsText} onChange={(e) => setColorsText(e.target.value)}/></Entry>
            <Entry title="Phụ kiện JSON (id, name, description, price, image tùy chọn, tag, tag_variant)"><textarea className={inputClass} rows={4} value={accessoriesText} onChange={(e) => setAccessoriesText(e.target.value)}/></Entry>
            {!productId && <Entry title="SKU ban đầu (mỗi dòng SKU | màu | cỡ | tồn kho)"><textarea className={inputClass} rows={4} value={variantsText} onChange={(e) => setVariantsText(e.target.value)} placeholder="AO-DO-M | Đỏ | M | 10" /></Entry>}
            <label className="flex gap-2"><input type="checkbox" checked={product.is_active} onChange={(e) => setProduct({ ...product, is_active: e.target.checked })}/>Đang bán</label>
            {productId && <p className="text-sm text-secondary">Lưu thông tin sẽ không thay đổi tồn kho, danh sách SKU, rating hay review_count.</p>}
            <button type="submit" disabled={busy} className={buttonClass}>Lưu sản phẩm</button></form>
            {chosen && <section className="bg-surface p-5 space-y-3"><h2 className="text-xl font-bold">SKU & tồn kho — tổng {chosen.stock}</h2><p className="text-sm text-secondary">Điều chỉnh tồn bằng số chênh lệch, không nhập đè số tồn cũ.</p>
              {(chosen.variants || []).map((v) => <VariantRow key={`${chosen.id}:${v.sku}`} productId={chosen.id} variant={v} onSaved={load} onForbidden={onForbidden}/>)}
              <div className="border-t pt-3 space-y-2"><h3 className="font-bold">Thêm SKU (tồn = 0)</h3><div className="grid md:grid-cols-3 gap-2"><input className={inputClass} placeholder="SKU" value={newSKU.sku} onChange={(e) => setNewSKU({ ...newSKU, sku: e.target.value })}/><input className={inputClass} placeholder="Màu" value={newSKU.color} onChange={(e) => setNewSKU({ ...newSKU, color: e.target.value })}/><input className={inputClass} placeholder="Cỡ" value={newSKU.size} onChange={(e) => setNewSKU({ ...newSKU, size: e.target.value })}/></div><button className={buttonClass} disabled={busy} type="button" onClick={() => void addSKU()}>Thêm SKU</button></div>
            </section>}
            {chosen && <ProductContentEditor key={chosen.id} productId={chosen.id} metadataWrites={metadataWrites.filter((write) => write.productId === chosen.id)} onForbidden={onForbidden}
              onVersion={(version) => {
                setProduct((current) => current.expected_version == null ? current : { ...current, expected_version: version })
                setProducts((current) => current.map((row) => row.id === chosen.id ? { ...row, version } : row))
              }}/>}
          </div></div>}
        {success && <p role="status" className="text-green-700">{success}</p>}
        {error && <p role="alert" className="text-red-700">{error}</p>}
      </>}
    </div>
  </main>
}
