import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react'
import { Navigate } from 'react-router-dom'
import ProductContentEditor from './ProductContentEditor'
import CategoryForm from './catalog/CategoryForm'
import ProductForm from './catalog/ProductForm'
import ProductList from './catalog/ProductList'
import VariantBuilder from './catalog/VariantBuilder'
import { builderRows, emptyBuilder, type BuilderState } from '@/features/admin/variantBuilder'
import VariantTable from './catalog/VariantTable'
import { Alert, Button, Card, PageHeader } from './ui'
import type { MetadataWrite } from '@/features/admin/content'
import { useToast } from '@/hooks/useToast'
import {
  adminCatalog, AdminApiError, cleanColors, draftsToVariants, editableGallery, parseAccessoriesJson, parseImageLines,
  type Category, type CategoryInput, type Product, type ProductInput,
} from '@/features/admin/catalog'

const emptyCategory: CategoryInput = { name: '', slug: '', description: '', parent_id: '', is_active: true, sort_order: 0, image: null }
const emptyProduct: ProductInput = {
  name: '', slug: '', category_id: '', description: '', material: '', price: 0,
  compare_at_price: null, is_active: true, images: [], image: null, gallery: [], colors: [], accessories: [],
}

export default function AdminCatalogPage({ view }: { view: 'product' | 'category' }) {
  const { showToast } = useToast()
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
  const [galleryText, setGalleryText] = useState('')
  const [metadataWrites, setMetadataWrites] = useState<(MetadataWrite & { productId: string })[]>([])
  const [accessoriesText, setAccessoriesText] = useState('')

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
    setLoading(true)
    void load()
  }
  const onMessage = useCallback((message: string, kind: 'success' | 'error') => showToast(message, kind), [showToast])
  const onForbidden = useCallback(() => setAccessDenied(true), [])
  function fail(err: unknown, fallback: string) {
    if (err instanceof AdminApiError && [401, 403].includes(err.status)) setAccessDenied(true)
    else showToast(err instanceof Error ? err.message : fallback, 'error')
  }

  function selectCategory(id: string) {
    setCategoryId(id)
    const value = categories.find((row) => row.id === id)
    setCategory(value ? { name: value.name, slug: value.slug, description: value.description || '', parent_id: value.parent_id || '', is_active: value.is_active, sort_order: value.sort_order || 0, image: value.image || null, expected_version: value.version ?? 0 } : { ...emptyCategory })
  }
  function selectProduct(id: string) {
    if (id !== productId) setMetadataWrites([])
    setProductId(id)
    const value = products.find((row) => row.id === id)
    setProduct(value ? {
      name: value.name, slug: value.slug, category_id: value.category_id, description: value.description || '', material: value.material || '',
      price: Number(value.price), compare_at_price: value.compare_at_price == null ? (value.old_price == null ? null : Number(value.old_price)) : Number(value.compare_at_price),
      is_active: value.is_active, images: value.images || [], image: value.image || (value.images?.[0] ? { src: value.images[0], alt: value.name } : null), expected_version: value.version ?? 0,
      gallery: value.gallery || [], colors: (value.colors || []).map((color) => ({ ...color })), accessories: value.accessories || [],
    } : { ...emptyProduct })
    setBuilder(emptyBuilder)
    setGalleryText(value ? editableGallery(value).map((item) => item.src + ' | ' + item.alt).join('\n') : '')
    setAccessoriesText(JSON.stringify(value?.accessories || [], null, 2))
  }
  async function submitCategory(event: FormEvent) {
    event.preventDefault(); setBusy(true)
    try {
      if (categoryId) {
        const updated = await adminCatalog.updateCategory(categoryId, category)
        setCategory((current) => ({ ...current, expected_version: updated.version ?? 0 }))
      } else await adminCatalog.createCategory(category)
      await load()
      showToast('Đã lưu danh mục', 'success')
    } catch (err) { fail(err, 'Không thể lưu danh mục') }
    finally { setBusy(false) }
  }
  async function submitProduct(event: FormEvent) {
    event.preventDefault(); setBusy(true)
    try {
      const gallery = parseImageLines(galleryText)
      const colors = cleanColors(product.colors)
      const accessories = parseAccessoriesJson(accessoriesText)
      const images = [...new Set([...(product.image?.src ? [product.image.src] : []), ...gallery.map((item) => item.src)])]
      const data = { ...product, images, gallery, colors, accessories }
      if (productId) {
        const updated = await adminCatalog.updateProduct(productId, data)
        setMetadataWrites((writes) => [...writes, { productId, from: data.expected_version ?? 0, to: updated.version ?? 0 }])
        setProduct((current) => ({ ...current, expected_version: updated.version ?? 0 }))
      } else {
        await adminCatalog.createProduct({ ...data, variants: draftsToVariants(builderRows(product.slug, builder)) })
      }
      await load()
      showToast(productId ? 'Đã lưu sản phẩm' : 'Đã tạo sản phẩm', 'success')
    } catch (err) { fail(err, 'Không thể lưu sản phẩm') }
    finally { setBusy(false) }
  }

  const chosen = products.find((row) => row.id === productId)
  if (accessDenied) return <Navigate to="/403" replace />
  const isProducts = view === 'product'
  return (
    <>
      <PageHeader
        title={isProducts ? 'Sản phẩm' : 'Danh mục'}
        subtitle={isProducts ? 'Thông tin bán hàng, SKU và tồn kho' : 'Nhóm sản phẩm hiển thị trên cửa hàng'}
        actions={<Button variant="secondary" onClick={refresh}>Tải lại</Button>}
      />
      {loading && <p role="status">Đang kiểm tra quyền admin và tải dữ liệu...</p>}
      {!loading && loadError && <Alert>{loadError}</Alert>}
      {!loading && !isProducts && (
        <CategoryForm categories={categories} categoryId={categoryId} value={category} busy={busy} onSelect={selectCategory} onChange={setCategory} onSubmit={(e) => void submitCategory(e)} />
      )}
      {!loading && isProducts && (
        <div className="grid gap-6 lg:grid-cols-[20rem_1fr] items-start">
          <Card title={'Sản phẩm (' + products.length + ')'}>
            <ProductList
              products={products}
              selectedId={productId}
              page={productPage}
              totalPages={totalPages}
              onSelect={selectProduct}
              onPage={(next) => { requestSequence.current++; setLoading(true); setProductPage(next) }}
            />
          </Card>
          <div className="space-y-5 min-w-0">
            <ProductForm
              productId={productId}
              product={product}
              categories={categories}
              galleryText={galleryText}
              accessoriesText={accessoriesText}
              busy={busy}
              onChange={setProduct}
              onGalleryText={setGalleryText}
              onAccessoriesText={setAccessoriesText}
              onSubmit={(e) => void submitProduct(e)}
            >
              {!productId && <VariantBuilder slug={product.slug} value={builder} onChange={setBuilder} />}
            </ProductForm>
            {chosen && <VariantTable product={chosen} onSaved={load} onForbidden={onForbidden} onMessage={onMessage} />}
            {chosen && <ProductContentEditor key={chosen.id} productId={chosen.id} metadataWrites={metadataWrites.filter((write) => write.productId === chosen.id)} onForbidden={onForbidden}
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
