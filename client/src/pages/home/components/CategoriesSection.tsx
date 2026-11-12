import { Link } from 'react-router-dom'
import EyebrowHeading from '@/components/ui/EyebrowHeading'
import { ROUTES } from '@/data/site'
import CatalogState from '@/features/catalog/CatalogState'
import { getCategories } from '@/features/catalog/api'
import { useCatalogRequest } from '@/features/catalog/useCatalogRequest'

export default function CategoriesSection() {
  const { data, loading, error, retry } = useCatalogRequest(getCategories, 'home:categories')
  return <section className="w-full py-16 lg:py-24 bg-surface-container-low">
    <div className="max-w-[1440px] mx-auto px-margin md:px-margin-tablet lg:px-margin-desktop">
      <EyebrowHeading eyebrow="Phân Loại Dáng Áo" title="Khám Phá Các Danh Mục" titleClassName="text-on-surface" />
      <div className="mt-10"><CatalogState loading={loading} error={error} onRetry={retry}>
        {data?.length ? <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
          {data.map((category) => <Link key={category.id} to={`${ROUTES.collection}?category=${encodeURIComponent(category.slug || category.id)}`}
            className="group bg-surface shadow-sm hover:shadow-xl transition-all block">
            {category.image?.src && <div className="aspect-[3/4] overflow-hidden bg-surface-container"><img src={category.image.src} alt={category.image.alt || category.name} loading="lazy" className="w-full h-full object-cover group-hover:scale-105 transition-transform" /></div>}
            <div className="p-6"><h3 className="font-headline-sm text-headline-sm group-hover:text-primary">{category.name}</h3>
              {category.description && <p className="mt-2 text-on-surface-variant line-clamp-3">{category.description}</p>}
              <span className="block text-primary mt-4">Khám phá →</span></div>
          </Link>)}
        </div> : <p className="text-on-surface-variant">Chưa có danh mục đang hiển thị.</p>}
      </CatalogState></div>
    </div>
  </section>
}
