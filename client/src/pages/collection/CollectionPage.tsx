import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { getCategories, listProducts } from '@/features/catalog/api'
import { useCatalogRequest } from '@/features/catalog/useCatalogRequest'
import CatalogState from '@/features/catalog/CatalogState'
import type { CatalogFilters, CatalogSort } from '@/features/catalog/types'
import CollectionHero from './components/CollectionHero'
import CategoryTabs from './components/CategoryTabs'
import FilterSidebar from './components/FilterSidebar'
import { sidebarFilterKeys, type SidebarFilters } from './sidebarFilters'
import Pagination from './components/Pagination'
import ProductCard from './components/ProductCard'
import Toolbar from './components/Toolbar'
import BespokeBanner from './components/BespokeBanner'

const filterKeys = ['search', 'category', 'color', 'size', 'material', 'min_price', 'max_price'] as const

export default function CollectionPage() {
  const [params, setParams] = useSearchParams()
  const [filterOpen, setFilterOpen] = useState(true)
  const key = params.toString()
  // Reset only when an applied sidebar filter changes (including browser Back/Forward).
  // Sorting and pagination must not discard unsaved filter edits.
  const sidebarKey = JSON.stringify(sidebarFilterKeys.map((name) => params.get(name) || ''))
  const page = Math.min(200, Math.max(1, Number(params.get('page')) || 1))
  const sort = (['newest', 'price_asc', 'price_desc', 'name_asc', 'name_desc'].includes(params.get('sort') || '')
    ? params.get('sort') : 'newest') as CatalogSort
  const filters: CatalogFilters = { page, page_size: 12, sort }
  filterKeys.forEach((name) => { const value = params.get(name); if (value) filters[name] = value })

  const categories = useCatalogRequest(getCategories, 'catalog:categories')
  const products = useCatalogRequest((signal) => listProducts(filters, signal), `catalog:list:${key}`)

  function update(name: string, value: string) {
    setParams((previous) => {
      const next = new URLSearchParams(previous)
      if (value) next.set(name, value)
      else next.delete(name)
      if (name !== 'page') next.delete('page')
      return next
    })
  }

  function applyFilters(values: SidebarFilters) {
    setParams((previous) => {
      const next = new URLSearchParams(previous)
      for (const [key, value] of Object.entries(values)) {
        if (value) next.set(key, value)
        else next.delete(key)
      }
      next.delete('page')
      return next
    })
  }

  const clearFilters = () => setParams((previous) => {
    const next = new URLSearchParams(previous)
    filterKeys.forEach((key) => next.delete(key))
    next.delete('page')
    return next
  })

  return (
    <div className="flex flex-col w-full">
      <CollectionHero total={products.data?.total} />
      <div className="max-w-[1440px] mx-auto w-full px-margin md:px-margin-desktop py-10 md:py-16">
        <CatalogState loading={categories.loading} error={categories.error} onRetry={categories.retry}>
          <CategoryTabs categories={categories.data || []} active={params.get('category') || ''} onChange={(value) => update('category', value)} />
        </CatalogState>
        <Toolbar shown={products.data?.data.length || 0} total={products.data?.total || 0} filterOpen={filterOpen}
          onToggleFilter={() => setFilterOpen((v) => !v)} sort={sort} onSortChange={(v) => update('sort', v)} />
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
          {filterOpen && <FilterSidebar key={sidebarKey} params={params} onApply={applyFilters} onClear={clearFilters} />}
          <div className={filterOpen ? 'lg:col-span-9' : 'lg:col-span-12'}>
            <CatalogState loading={products.loading} error={products.error} onRetry={products.retry}>
              {products.data?.data.length ? (
                <div className={`grid grid-cols-1 md:grid-cols-2 gap-6 md:gap-8 ${filterOpen ? 'xl:grid-cols-3' : 'xl:grid-cols-4'}`}>
                  {products.data.data.map((product) => <ProductCard key={product.id} product={product} />)}
                </div>
              ) : <div className="py-16 text-center space-y-3"><p>Không có sản phẩm khớp bộ lọc.</p><button type="button" onClick={clearFilters} className="underline text-primary">Xóa bộ lọc</button></div>}
              {!!products.data && products.data.total_pages > 1 && <Pagination page={page} totalPages={products.data.total_pages} onChange={(next) => update('page', String(next))} />}
            </CatalogState>
          </div>
        </div>
        <BespokeBanner />
      </div>
    </div>
  )
}
