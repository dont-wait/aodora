import type { CatalogCategory } from '@/features/catalog/types'
interface Props { categories: CatalogCategory[]; active: string; onChange: (value: string) => void }
export default function CategoryTabs({ categories, active, onChange }: Props) {
  return <div role="tablist" aria-label="Danh mục" className="flex items-center gap-3 overflow-x-auto pb-4 mb-10">
    {[{ id: '', slug: '', name: 'Tất cả' }, ...categories].map((category) => (
      <button key={category.id} role="tab" aria-selected={active === (category.slug || category.id)} type="button"
        onClick={() => onChange(category.slug || category.id)}
        className={`flex-shrink-0 px-5 py-2.5 font-label-uppercase text-label-uppercase tracking-wider transition-colors ${active === (category.slug || category.id) ? 'bg-primary text-on-primary' : 'bg-surface-container hover:bg-surface-container-high text-on-surface'}`}>
        {category.name}
      </button>
    ))}
  </div>
}
