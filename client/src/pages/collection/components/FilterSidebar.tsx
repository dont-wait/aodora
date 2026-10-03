import { useState } from 'react'
import { Link } from 'react-router-dom'
import { filterOptions } from '@/data/products'
import { ROUTES } from '@/data/site'

const heading = 'font-label-uppercase text-label-uppercase text-on-surface mb-3.5 tracking-wider'

interface FilterState {
  materials: string[]
  colors: string[]
  necks: string[]
  price: string
  mode: string
}

const INITIAL: FilterState = { materials: [], colors: [], necks: [], price: '', mode: filterOptions.modes[0] }

const toggle = (list: string[], v: string) => (list.includes(v) ? list.filter((x) => x !== v) : [...list, v])

/** Bộ lọc chi tiết. Hiện mới giữ state cục bộ – nối với API/lọc dữ liệu khi backend có thuộc tính tương ứng. */
export default function FilterSidebar() {
  const [state, setState] = useState<FilterState>(INITIAL)

  return (
    <aside id="filterSidebar" className="lg:col-span-3 space-y-8 bg-surface-container-low p-6 shadow-sm">
      <div className="flex items-center justify-between pb-3">
        <h3 className="font-title-editorial text-title-editorial text-primary font-bold">Lọc Chi Tiết</h3>
        <button
          type="button"
          onClick={() => setState(INITIAL)}
          className="font-label-uppercase text-label-uppercase text-secondary hover:text-primary transition-colors underline"
        >
          Thiết lập lại
        </button>
      </div>

      <fieldset>
        <legend className={heading}>Chất Liệu Truyền Thống</legend>
        <div className="space-y-2.5">
          {filterOptions.materials.map((m) => (
            <label key={m.label} className="flex items-center gap-3 cursor-pointer group">
              <input
                type="checkbox"
                className="w-4 h-4 accent-primary cursor-pointer"
                checked={state.materials.includes(m.label)}
                onChange={() => setState((s) => ({ ...s, materials: toggle(s.materials, m.label) }))}
              />
              <span className="font-body-sm text-body-sm text-on-surface-variant group-hover:text-primary transition-colors flex-1">{m.label}</span>
              <span className="font-label-regular text-[11px] text-on-surface-variant">{m.count}</span>
            </label>
          ))}
        </div>
      </fieldset>

      <fieldset>
        <legend className={heading}>Sắc Màu Thổ Cẩm & Tơ Lụa</legend>
        <div className="grid grid-cols-3 gap-2.5">
          {filterOptions.colors.map((c) => {
            const active = state.colors.includes(c.name)
            return (
              <button
                key={c.name}
                type="button"
                title={c.title}
                aria-pressed={active}
                onClick={() => setState((s) => ({ ...s, colors: toggle(s.colors, c.name) }))}
                className={`flex items-center gap-2 p-2 bg-surface transition-colors shadow-sm ${active ? 'ring-1 ring-primary' : 'hover:bg-surface-container'}`}
              >
                <span className="w-4 h-4 rounded-full shadow-inner" style={{ backgroundColor: c.hex }} />
                <span className={`font-label-regular text-[12px] truncate ${active ? 'text-primary font-semibold' : ''}`}>{c.name}</span>
              </button>
            )
          })}
        </div>
      </fieldset>

      <fieldset>
        <legend className={heading}>Dáng Cổ & Cấu Trúc Khuyết</legend>
        <div className="space-y-2">
          {filterOptions.necks.map((n) => (
            <label key={n} className="flex items-center gap-3 cursor-pointer group">
              <input
                type="checkbox"
                className="w-4 h-4 accent-primary cursor-pointer"
                checked={state.necks.includes(n)}
                onChange={() => setState((s) => ({ ...s, necks: toggle(s.necks, n) }))}
              />
              <span className="font-body-sm text-body-sm text-on-surface-variant group-hover:text-primary transition-colors">{n}</span>
            </label>
          ))}
        </div>
      </fieldset>

      <fieldset>
        <legend className={heading}>Mức Ngân Sách</legend>
        <div className="space-y-2 font-body-sm text-body-sm text-on-surface-variant">
          {filterOptions.prices.map((p) => (
            <label key={p} className="flex items-center gap-3 cursor-pointer hover:text-primary transition-colors">
              <input
                type="radio"
                name="price"
                className="accent-primary cursor-pointer"
                checked={state.price === p}
                onChange={() => setState((s) => ({ ...s, price: p }))}
              />
              <span className={state.price === p ? 'text-primary font-medium' : ''}>{p}</span>
            </label>
          ))}
        </div>
      </fieldset>

      <fieldset className="p-4 bg-surface shadow-inner">
        <legend className="font-label-uppercase text-label-uppercase text-secondary tracking-wider px-1">Hình Thức Sở Hữu</legend>
        <div className="space-y-2 mt-2">
          {filterOptions.modes.map((m) => (
            <label key={m} className="flex items-center gap-2 cursor-pointer">
              <input
                type="radio"
                name="mode"
                className="accent-secondary cursor-pointer"
                checked={state.mode === m}
                onChange={() => setState((s) => ({ ...s, mode: m }))}
              />
              <span className="font-label-regular text-label-regular text-on-surface">{m}</span>
            </label>
          ))}
        </div>
      </fieldset>

      <div className="relative overflow-hidden bg-primary text-on-primary p-6 text-left">
        <div className="absolute -right-6 -bottom-6 w-24 h-24 rounded-full bg-white/10 blur-xl" />
        <span className="font-label-uppercase text-[10px] text-secondary-fixed tracking-widest block mb-2">ĐẶC QUYỀN AN SOIE</span>
        <h5 className="font-headline-sm text-headline-sm mb-2">Hỗ Trợ Đo Tận Nơi</h5>
        <p className="font-body-sm text-[12px] leading-relaxed text-on-primary/80 mb-4">
          Nghệ nhân may đo hỗ trợ lấy số đo miễn phí tại nội thành Hà Nội & TP.HCM.
        </p>
        <Link
          to={`${ROUTES.tailoring}#dat-lich-tan-noi`}
          className="inline-block px-4 py-2 bg-on-primary text-primary font-label-uppercase text-[10px] tracking-wider hover:bg-surface-bright transition-colors font-bold"
        >
          Đặt Lịch Ngay
        </Link>
      </div>
    </aside>
  )
}
