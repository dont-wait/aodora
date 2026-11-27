import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react'
import { Navigate, useSearchParams } from 'react-router-dom'
import ProductContentEditor from './ProductContentEditor'
import CategoryForm from './catalog/CategoryForm'
import ProductForm from './catalog/ProductForm'
import ProductList from './catalog/ProductList'
import VariantBuilder from './catalog/VariantBuilder'
import VariantTable from './catalog/VariantTable'
import { Alert, Button, Card, CardSkeleton, PageHeader } from './ui'
import { UNSAVED_MESSAGE, useUnsavedChanges } from './unsaved'
import { builderRows, emptyBuilder, type BuilderState } from '@/features/admin/variantBuilder'
import type { MetadataWrite } from '@/features/admin/content'
import { slugify } from '@/features/admin/slug'
import { errorCount, mapServerField, validateCategory, validateProduct, validateSkuRows, type Errors } from '@/features/admin/validation'
import { useToast } from '@/hooks/useToast'
import { adminCatalog, AdminApiError, draftsToVariants, type Category, type CategoryInput, type Product, type ProductInput } from '@/features/admin/catalog'

const emptyCategory: CategoryInput = { name: '', slug: '', description: '', parent_id: '', is_active: true, sort_order: 0, image: null }
const emptyProduct: ProductInput = {
  name: '', slug: '', category_id: '', description: '', material: '', price: 0,
  compare_at_price: null, is_active: false, images: [], image: null, gallery: [], colors: [], accessories: [],
}
const snapshotOf = (value: unknown) => JSON.stringify(value)

export default function AdminCatalogPage({ view }: { view: 'product' | 'category' }) {
  const { showToast } = useToast()
  const [params, setParams] = useSearchParams()
  const focusId = params.get('product') || ''
  const [handledFocus, setHandledFocus] = useState('')
  const [categories, setCategories] = useState<Category[]>([])
  const [products, setProducts] = useState<Product[]>([])
  const [productPage, setProductPage] = useState(1)
  const [totalPages, setTotalPages] = useState(1)
  const [accessDenied, setAccessDenied] = useState(false)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [loadError, setLoadError] = useState('')
  const [categoryId, setCategoryId] = useState('')
  const [productId, setProductId] = useState('')
  const [category, setCategory] = useState<CategoryInput>(emptyCategory)
  const [product, setProduct] = useState<ProductInput>(emptyProduct)
  const [builder, setBuilder] = useState<BuilderState>(emptyBuilder)
  const [metadataWrites, setMetadataWrites] = useState<(MetadataWrite & { productId: string })[]>([])
  const [productErrors, setProductErrors] = useState<Errors>({})
  const [categoryErrors, setCategoryErrors] = useState<Errors>({})
  const [originalSlug, setOriginalSlug] = useState('')
  const [slugManual, setSlugManual] = useState(false)
  const [categorySlugManual, setCategorySlugManual] = useState(false)
  const [productBaseline, setProductBaseline] = useState(() => snapshotOf({ product: emptyProduct, builder: emptyBuilder, productId: '' }))
  const [categoryBaseline, setCategoryBaseline] = useState(() => snapshotOf({ category: emptyCategory, categoryId: '' }))
  const [focusTick, setFocusTick] = useState(0)

  const productDirty = snapshotOf({ product, builder, productId }) !== productBaseline
  const categoryDirty = snapshotOf({ category, categoryId }) !== categoryBaseline
  const dirty = view === 'product' ? productDirty : categoryDirty
  useUnsavedChanges(dirty)

  // After errors are rendered, jump to the first invalid control.
  useEffect(() => {
    if (focusTick === 0) return
    const el = document.querySelector<HTMLElement>('.admin-ui [aria-invalid="true"]')
    if (el) { el.scrollIntoView({ block: 'center', behavior: 'smooth' }); el.focus({ preventScroll: true }) }
  }, [focusTick])

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
        setLoadError('')
      })
      .catch((err: unknown) => {
        if (current !== requestSequence.current) return
        setLoadError(err instanceof Error ? err.message : 'Không thể tải dữ liệu')
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
    if (dirty && !window.confirm(UNSAVED_MESSAGE)) return
    setLoading(true)
    void load()
  }
  useEffect(() => {
    if (handledFocus) setParams({}, { replace: true })
  }, [handledFocus, setParams])
  const onMessage = useCallback((message: string, kind: 'success' | 'error') => showToast(message, kind), [showToast])
  const onForbidden = useCallback(() => setAccessDenied(true), [])
  function fail(err: unknown, fallback: string, setErrors: (e: Errors) => void) {
    if (err instanceof AdminApiError && [401, 403].includes(err.status)) { setAccessDenied(true); return }
    if (err instanceof AdminApiError && err.field) {
      setErrors({ [mapServerField(err.field)]: err.message })
      setFocusTick((n) => n + 1)
      return
    }
    showToast(err instanceof Error ? err.message : fallback, 'error')
  }

  function categoryFrom(value: Category | undefined): CategoryInput {
    return value
      ? { name: value.name, slug: value.slug, description: value.description || '', parent_id: value.parent_id || '', is_active: value.is_active, sort_order: value.sort_order || 0, image: value.image || null, expected_version: value.version ?? 0 }
      : { ...emptyCategory }
  }
  function productFrom(value: Product | undefined): ProductInput {
    return value
      ? {
          name: value.name, slug: value.slug, category_id: value.category_id, description: value.description || '', material: value.material || '',
          price: Number(value.price), compare_at_price: value.compare_at_price == null ? (value.old_price == null ? null : Number(value.old_price)) : Number(value.compare_at_price),
          is_active: value.is_active, images: value.images || [], image: value.image || (value.images?.[0] ? { src: value.images[0], alt: value.name } : null), expected_version: value.version ?? 0,
          gallery: value.gallery || [], colors: (value.colors || []).map((color) => ({ ...color })), accessories: (value.accessories || []).map((a) => ({ ...a })),
        }
      : { ...emptyProduct }
  }
  function selectCategory(id: string) {
    if (categoryDirty && !window.confirm(UNSAVED_MESSAGE)) return
    const next = categoryFrom(categories.find((row) => row.id === id))
    setCategoryId(id); setCategory(next); setCategoryErrors({}); setCategorySlugManual(false)
    setCategoryBaseline(snapshotOf({ category: next, categoryId: id }))
  }
  function selectProduct(id: string, force = false) {
    if (!force && productDirty && !window.confirm(UNSAVED_MESSAGE)) return
    if (id !== productId) setMetadataWrites([])
    const next = productFrom(products.find((row) => row.id === id))
    setProductId(id); setProduct(next); setBuilder(emptyBuilder); setProductErrors({}); setSlugManual(false); setOriginalSlug(next.slug)
    setProductBaseline(snapshotOf({ product: next, builder: emptyBuilder, productId: id }))
  }

  function changeProductName(name: string) {
    setProduct((current) => ({ ...current, name, slug: !productId && !slugManual ? slugify(name) : current.slug }))
  }
  function changeCategoryName(name: string) {
    setCategory((current) => ({ ...current, name, slug: !categoryId && !categorySlugManual ? slugify(name) : current.slug }))
  }

  async function submitCategory(event: FormEvent) {
    event.preventDefault()
    const local = validateCategory(category)
    setCategoryErrors(local)
    if (errorCount(local) > 0) { setFocusTick((n) => n + 1); return }
    setBusy(true)
    try {
      let saved = category
      let id = categoryId
      if (categoryId) {
        const updated = await adminCatalog.updateCategory(categoryId, category)
        saved = { ...category, expected_version: updated.version ?? 0 }
      } else {
        const created = await adminCatalog.createCategory(category)
        id = created.id
        saved = { ...category, expected_version: created.version ?? 0 }
        setCategoryId(id)
      }
      setCategory(saved)
      setOriginalSlug(saved.slug)
      setCategoryBaseline(snapshotOf({ category: saved, categoryId: id }))
      await load()
      showToast('Đã lưu danh mục', 'success')
    } catch (err) { fail(err, 'Không thể lưu danh mục', setCategoryErrors) }
    finally { setBusy(false) }
  }

  const chosen = products.find((row) => row.id === productId)
  const colorNames = product.colors.map((c) => c.name.trim()).filter(Boolean)
  const savedColorNames = (chosen?.colors || []).map((c) => c.name.trim()).filter(Boolean)
  const lockedColors: Record<string, string> = {}
  for (const variant of chosen?.variants || []) {
    const key = variant.color.trim().toLowerCase()
    if (!lockedColors[key]) lockedColors[key] = 'Đang dùng bởi SKU ' + variant.sku + ', không đổi tên/xóa được'
  }

  async function submitProduct(event: FormEvent) {
    event.preventDefault()
    const isNew = !productId
    const rows = isNew ? builderRows(product.slug, builder, colorNames).map((r) => ({ color: r.color, size: r.size, sku: r.sku, stock: r.stock })) : []
    const local: Errors = { ...validateProduct(product, { skuCount: chosen?.variants.length ?? 0, isNew, wasActive: chosen?.is_active }), ...(isNew ? validateSkuRows(rows, colorNames) : {}) }
    setProductErrors(local)
    if (errorCount(local) > 0) { setFocusTick((n) => n + 1); return }
    setBusy(true)
    try {
      const images = [...new Set([...(product.image?.src ? [product.image.src] : []), ...product.gallery.map((item) => item.src)])]
      const data: ProductInput = { ...product, images, name: product.name.trim() }
      if (productId) {
        const updated = await adminCatalog.updateProduct(productId, data)
        setMetadataWrites((writes) => [...writes, { productId, from: data.expected_version ?? 0, to: updated.version ?? 0 }])
        const saved = { ...product, expected_version: updated.version ?? 0 }
        setProduct(saved)
        setOriginalSlug(saved.slug)
        setProductBaseline(snapshotOf({ product: saved, builder, productId }))
      } else {
        const created = await adminCatalog.createProduct({ ...data, variants: draftsToVariants(builderRows(product.slug, builder, colorNames)) })
        const saved = { ...product, expected_version: created.version ?? 0 }
        setProductId(created.id)
        setProduct(saved)
        setBuilder(emptyBuilder)
        setOriginalSlug(saved.slug)
        setProductBaseline(snapshotOf({ product: saved, builder: emptyBuilder, productId: created.id }))
      }
      await load()
      showToast(productId ? 'Đã lưu sản phẩm' : 'Đã tạo sản phẩm', 'success')
    } catch (err) { fail(err, 'Không thể lưu sản phẩm', setProductErrors) }
    finally { setBusy(false) }
  }

  // Deep link from the dashboard (?product=<id>): walk the pages until the product is found, select it, then clear the param.
  if (view === 'product' && focusId && !loading && handledFocus !== focusId) {
    if (products.some((row) => row.id === focusId)) {
      setHandledFocus(focusId)
      selectProduct(focusId, true)
    } else if (productPage < totalPages) {
      setLoading(true)
      setProductPage(productPage + 1)
    } else {
      setHandledFocus(focusId)
    }
  }
  if (accessDenied) return <Navigate to="/403" replace />
  const isProducts = view === 'product'
  return (
    <>
      <PageHeader
        title={isProducts ? 'Sản phẩm' : 'Danh mục'}
        subtitle={isProducts ? 'Thông tin bán hàng, SKU và tồn kho' : 'Nhóm sản phẩm hiển thị trên cửa hàng'}
        actions={<Button variant="secondary" onClick={refresh}>Tải lại</Button>}
      />
      {loading && <div className="grid gap-6 lg:grid-cols-[20rem_1fr]"><CardSkeleton lines={8} label="Đang tải dữ liệu" /><CardSkeleton lines={10} label="Đang tải dữ liệu" /></div>}
      {!loading && loadError && <Alert action={<Button variant="secondary" onClick={refresh}>Thử lại</Button>}>{loadError}</Alert>}
      {!loading && !isProducts && (
        <CategoryForm
          categories={categories}
          categoryId={categoryId}
          originalSlug={originalSlug}
          value={category}
          errors={categoryErrors}
          busy={busy}
          onSelect={selectCategory}
          onChange={setCategory}
          onName={changeCategoryName}
          onSlug={(slug) => { setCategorySlugManual(true); setCategory((c) => ({ ...c, slug })) }}
          onSubmit={(e) => void submitCategory(e)}
        />
      )}
      {!loading && isProducts && (
        <div className="grid items-start gap-6 lg:grid-cols-[20rem_1fr]">
          <Card title={'Sản phẩm (' + products.length + ')'}>
            <ProductList
              products={products}
              selectedId={productId}
              page={productPage}
              totalPages={totalPages}
              onSelect={(id) => selectProduct(id)}
              onPage={(next) => { if (productDirty && !window.confirm(UNSAVED_MESSAGE)) return; setLoading(true); setProductPage(next) }}
            />
          </Card>
          <div className="min-w-0 space-y-5">
            <ProductForm
              productId={productId}
              product={product}
              originalSlug={originalSlug}
              categories={categories}
              errors={productErrors}
              busy={busy}
              skuCount={chosen?.variants.length ?? 0}
              lockedColors={lockedColors}
              onChange={setProduct}
              onName={changeProductName}
              onSlug={(slug) => { setSlugManual(true); setProduct((p) => ({ ...p, slug })) }}
              onSubmit={(e) => void submitProduct(e)}
            >
              {!productId && <VariantBuilder slug={product.slug} colorOptions={colorNames} value={builder} errors={productErrors} onChange={setBuilder} />}
            </ProductForm>
            {chosen && <VariantTable product={chosen} colorOptions={savedColorNames} onSaved={load} onForbidden={onForbidden} onMessage={onMessage} />}
            {chosen && <ProductContentEditor key={chosen.id} productId={chosen.id} productName={chosen.name} metadataWrites={metadataWrites.filter((write) => write.productId === chosen.id)} onForbidden={onForbidden}
              onVersion={(version) => {
                setProduct((current) => current.expected_version == null ? current : { ...current, expected_version: version })
                setProducts((current) => current.map((row) => row.id === chosen.id ? { ...row, version } : row))
              }} />}
          </div>
        </div>
      )}
    </>
  )
}
