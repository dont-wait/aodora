import type { Category, CategoryId } from '@/types'

export type CategoryFilter = CategoryId | 'all'

interface Props {
  categories: Category[]
  active: CategoryFilter
  total: number
  onChange: (id: CategoryFilter) => void
}

export default function CategoryTabs({ categories, active, total, onChange }: Props) {
  const tabs: { id: CategoryFilter; label: string; count: number }[] = [
    { id: 'all', label: 'Tất cả', count: total },
    ...categories.map((c) => ({ id: c.id, label: c.name, count: c.count })),
  ]
  return (
    <div role="tablist" className="flex items-center gap-3 overflow-x-auto pb-4 mb-10">
      {tabs.map((t) => (
        <button
          key={t.id}
          role="tab"
          aria-selected={active === t.id}
          type="button"
          onClick={() => onChange(t.id)}
          className={`flex-shrink-0 px-5 py-2.5 font-label-uppercase text-label-uppercase tracking-wider transition-colors ${
            active === t.id ? 'bg-primary text-on-primary shadow-sm' : 'bg-surface-container text-on-surface hover:bg-surface-container-high'
          }`}
        >
          {t.label} ({t.count})
        </button>
      ))}
    </div>
  )
}
