import Icon from '@/components/ui/Icon'
import { sortOptions, type SortValue } from '@/data/products'

interface Props {
  shown: number
  total: number
  filterOpen: boolean
  onToggleFilter: () => void
  sort: SortValue
  onSortChange: (v: SortValue) => void
}

export default function Toolbar({ shown, total, filterOpen, onToggleFilter, sort, onSortChange }: Props) {
  return (
    <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4 p-5 bg-surface-container-low mb-10 shadow-sm">
      <div className="flex items-center gap-4">
        <button
          type="button"
          aria-expanded={filterOpen}
          aria-controls="filterSidebar"
          onClick={onToggleFilter}
          className="flex items-center gap-2 px-4 py-2 bg-surface text-on-surface font-label-regular text-label-regular hover:bg-surface-container transition-colors shadow-sm"
        >
          <Icon name="tune" size={18} className="text-primary" />
          <span className="font-semibold">Bộ lọc tinh tuyển</span>
        </button>
        <span className="font-body-sm text-body-sm text-on-surface-variant hidden sm:inline">
          Hiển thị <strong className="text-on-surface font-medium">{shown}</strong> trong <strong className="text-on-surface font-medium">{total}</strong> mẫu
          áo dài
        </span>
      </div>

      <label className="flex flex-wrap items-center gap-3">
        <span className="font-label-regular text-label-regular text-on-surface-variant">Sắp xếp theo:</span>
        <span className="relative inline-block">
          <select
            value={sort}
            onChange={(e) => onSortChange(e.target.value as SortValue)}
            className="appearance-none bg-surface text-on-surface px-4 py-2 pr-9 font-label-regular text-label-regular shadow-sm focus:outline-none focus:bg-surface-bright cursor-pointer"
          >
            {sortOptions.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
          <Icon name="expand_more" size={18} className="absolute right-2.5 top-1/2 -translate-y-1/2 pointer-events-none text-on-surface-variant" />
        </span>
      </label>
    </div>
  )
}
