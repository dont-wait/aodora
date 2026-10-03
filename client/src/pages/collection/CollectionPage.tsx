import { useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { categories, products, type SortValue } from '@/data/products'
import type { CategoryId } from '@/types'
import BespokeBanner from './components/BespokeBanner'
import CategoryTabs, { type CategoryFilter } from './components/CategoryTabs'
import CollectionHero from './components/CollectionHero'
import FilterSidebar from './components/FilterSidebar'
import Pagination from './components/Pagination'
import ProductCard from './components/ProductCard'
import Toolbar from './components/Toolbar'

const TOTAL_PRODUCTS = categories.reduce((sum, c) => sum + c.count, 0)
const isCategory = (v: string | null): v is CategoryId => categories.some((c) => c.id === v)

export default function CollectionPage() {
  const [params, setParams] = useSearchParams()
  const loai = params.get('loai')
  const active: CategoryFilter = isCategory(loai) ? loai : 'all'

  const [sort, setSort] = useState<SortValue>('newest')
  const [filterOpen, setFilterOpen] = useState(true)
  const [page, setPage] = useState(1)

  const visible = useMemo(() => {
    const list = active === 'all' ? [...products] : products.filter((p) => p.category === active)
    if (sort === 'price-asc') list.sort((a, b) => a.price - b.price)
    if (sort === 'price-desc') list.sort((a, b) => b.price - a.price)
    return list
  }, [active, sort])

  const changeCategory = (id: CategoryFilter) => {
    setParams(id === 'all' ? {} : { loai: id })
    setPage(1)
  }

  return (
    <div className="flex flex-col w-full">
      <CollectionHero />

      <div className="max-w-[1440px] mx-auto w-full px-margin md:px-margin-desktop py-10 md:py-16">
        <CategoryTabs categories={categories} active={active} total={TOTAL_PRODUCTS} onChange={changeCategory} />
        <Toolbar
          shown={visible.length}
          total={active === 'all' ? TOTAL_PRODUCTS : categories.find((c) => c.id === active)!.count}
          filterOpen={filterOpen}
          onToggleFilter={() => setFilterOpen((o) => !o)}
          sort={sort}
          onSortChange={setSort}
        />

        <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
          {filterOpen && <FilterSidebar />}
          <div className={filterOpen ? 'lg:col-span-9' : 'lg:col-span-12'}>
            {visible.length === 0 ? (
              <p className="py-20 text-center text-on-surface-variant">Chưa có mẫu áo dài nào trong nhóm này.</p>
            ) : (
              <div className={`grid grid-cols-1 md:grid-cols-2 gap-6 md:gap-8 ${filterOpen ? 'xl:grid-cols-3' : 'xl:grid-cols-4'}`}>
                {visible.map((p) => (
                  <ProductCard key={p.id} product={p} />
                ))}
              </div>
            )}
            <Pagination page={page} totalPages={6} onChange={setPage} />
          </div>
        </div>

        <BespokeBanner />
      </div>
    </div>
  )
}
