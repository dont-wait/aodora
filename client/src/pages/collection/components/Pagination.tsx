import Icon from '@/components/ui/Icon'

interface Props {
  page: number
  totalPages: number
  onChange: (page: number) => void
}

export default function Pagination({ page, totalPages, onChange }: Props) {
  const pages = Array.from({ length: Math.min(totalPages, 4) }, (_, i) => i + 1)
  return (
    <div className="mt-14 pt-8 flex flex-col sm:flex-row items-center justify-between gap-6">
      <p className="font-body-sm text-body-sm text-on-surface-variant">
        Trang <strong className="text-on-surface">{page}</strong> trên <strong className="text-on-surface">{totalPages}</strong> trang
      </p>
      <nav aria-label="Phân trang" className="flex items-center gap-2">
        <button
          type="button"
          aria-label="Trang trước"
          disabled={page === 1}
          onClick={() => onChange(page - 1)}
          className="w-10 h-10 flex items-center justify-center bg-surface-container text-on-surface-variant hover:bg-surface-container-high transition-colors disabled:opacity-50"
        >
          <Icon name="chevron_left" size={18} />
        </button>
        {pages.map((p) => (
          <button
            key={p}
            type="button"
            aria-current={p === page ? 'page' : undefined}
            onClick={() => onChange(p)}
            className={`w-10 h-10 flex items-center justify-center font-label-uppercase text-[12px] transition-colors shadow-sm ${p === page ? 'bg-primary text-on-primary' : 'bg-surface text-on-surface hover:bg-surface-container'}`}
          >
            {p}
          </button>
        ))}
        {totalPages > 4 && <span className="px-2 text-outline">...</span>}
        <button
          type="button"
          disabled={page === totalPages}
          onClick={() => onChange(page + 1)}
          className="px-4 h-10 flex items-center gap-1.5 bg-surface text-on-surface font-label-uppercase text-[11px] hover:bg-surface-container transition-colors shadow-sm disabled:opacity-50"
        >
          <span>Tiếp theo</span>
          <Icon name="chevron_right" size={16} />
        </button>
      </nav>
    </div>
  )
}
