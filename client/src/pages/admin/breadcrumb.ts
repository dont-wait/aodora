import { createContext, useContext, useEffect } from 'react'

export type AdminCrumb = { label: string; to?: string }

/** Top-level admin sections, keyed by the first path segment after /admin. */
export const ADMIN_SECTIONS: Record<string, string> = {
  catalog: 'Sản phẩm',
  categories: 'Danh mục',
  orders: 'Đơn hàng',
  returns: 'Hoàn hàng',
  tailoring: 'May đo',
  reviews: 'Đánh giá',
  import: 'Nhập dữ liệu',
}

/**
 * Builds the trail from the URL: "Tổng quan / Section / Detail".
 * Returns [] on the dashboard itself. `detail` replaces the generic
 * "Chi tiết" label of a detail route once the page knows its name.
 */
export function adminBreadcrumb(pathname: string, detail?: string): AdminCrumb[] {
  const [, admin, section, id] = pathname.split('/')
  if (admin !== 'admin' || !section || !ADMIN_SECTIONS[section]) return []
  const trail: AdminCrumb[] = [{ label: 'Tổng quan', to: '/admin' }]
  if (!id) return [...trail, { label: ADMIN_SECTIONS[section] }]
  return [...trail, { label: ADMIN_SECTIONS[section], to: `/admin/${section}` }, { label: detail || 'Chi tiết' }]
}

/** AdminLayout provides this; detail pages name the last crumb (e.g. an order number). */
export const AdminCrumbContext = createContext<{ setDetail: (label: string | null) => void }>({ setDetail: () => {} })

export function useAdminCrumb(label: string | null): void {
  const { setDetail } = useContext(AdminCrumbContext)
  useEffect(() => {
    setDetail(label)
    return () => setDetail(null)
  }, [label, setDetail])
}
