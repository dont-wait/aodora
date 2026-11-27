import { useCallback, useState } from 'react'
import { Link, Navigate, useLocation, useSearchParams } from 'react-router-dom'
import StarRating from '@/components/ui/StarRating'
import { useAuth } from '@/features/auth/useAuth'
import { adminReviewSearch, deleteAdminReview, deleteReviewConfirmation, loadAdminReviews, readAdminReviewFilters } from '@/features/admin/reviews'
import { formatOrderDate, orderLoginPath } from '@/features/orders/presentation'
import { orderPageCount } from '@/features/orders/read'
import { useOrderRead } from '@/features/orders/useOrderRead'
import AdminOrderFrame from './AdminOrderFrame'
import AdminOrderNotice from './AdminOrderNotice'

const inputClass = 'w-full border border-outline-variant bg-surface p-3'
const title = 'Quản lý đánh giá'

export default function AdminReviewsPage() {
  const { user, loading, error } = useAuth()
  const [params, setParams] = useSearchParams()
  const location = useLocation()
  const filter = readAdminReviewFilters(params)
  const { rating, page, page_size } = filter
  const [revision, setRevision] = useState(0)
  const [deleting, setDeleting] = useState('')
  const [notice, setNotice] = useState<{ kind: 'ok' | 'error'; text: string } | null>(null)
  const [redirect, setRedirect] = useState<'unauthenticated' | 'forbidden' | null>(null)
  const load = useCallback(() => loadAdminReviews({ rating, page, page_size }), [rating, page, page_size])
  const scope = JSON.stringify([user?.id, rating, page, page_size, revision])
  const result = useOrderRead(scope, !!user && !loading && !error, load)
  const retry = () => setRevision((value) => value + 1)
  const next = location.pathname + location.search
  if (redirect === 'unauthenticated') return <Navigate to={orderLoginPath(next)} replace />
  if (redirect === 'forbidden') return <Navigate to="/403" replace />
  if (!result || result.kind !== 'ready') return <AdminOrderNotice title={title} result={result} next={next} retry={retry} />
  const data = result.page
  const pages = orderPageCount(data)

  async function remove(id: string) {
    if (!window.confirm(deleteReviewConfirmation)) return
    setDeleting(id); setNotice(null)
    const outcome = await deleteAdminReview(id)
    setDeleting('')
    if (outcome.kind === 'unauthenticated' || outcome.kind === 'forbidden') return setRedirect(outcome.kind)
    if (outcome.kind === 'deleted') setNotice({ kind: 'ok', text: 'Đã xóa đánh giá và tính lại điểm của sản phẩm.' })
    else if (outcome.kind === 'not-found') setNotice({ kind: 'ok', text: outcome.message })
    else return setNotice({ kind: 'error', text: outcome.message })
    retry()
  }

  return (
    <AdminOrderFrame title={title}>
      <section className="bg-surface p-5 space-y-3">
        <p className="text-sm text-on-surface-variant">
          Đánh giá không phù hợp được xử lý bằng cách xóa vĩnh viễn. Điểm trung bình và số lượt đánh giá trên sản phẩm được tính lại ngay sau khi xóa.
        </p>
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4 items-end">
          <label className="space-y-2">
            <span>Điểm</span>
            <select className={inputClass} value={rating} onChange={(event) => setParams(adminReviewSearch({ ...filter, rating: Number(event.target.value), page: 1 }))}>
              <option value={0}>Tất cả</option>
              {[1, 2, 3, 4, 5].map((value) => <option key={value} value={value}>{value} sao</option>)}
            </select>
          </label>
          <label className="space-y-2">
            <span>Số đánh giá mỗi trang</span>
            <select className={inputClass} value={page_size} onChange={(event) => setParams(adminReviewSearch({ ...filter, page_size: Number(event.target.value), page: 1 }))}>
              {[...new Set([20, 50, 100, page_size])].sort((a, b) => a - b).map((size) => <option key={size} value={size}>{size}</option>)}
            </select>
          </label>
          <button type="button" className="border border-primary text-primary px-5 py-3" onClick={retry}>Tải lại danh sách</button>
        </div>
      </section>
      {notice && <p role={notice.kind === 'error' ? 'alert' : 'status'} className={notice.kind === 'error' ? 'text-error' : 'text-secondary'}>{notice.text}</p>}
      {data.data.length === 0 ? (
        <section className="bg-surface p-8 text-center space-y-4">
          <h2 className="font-headline-md text-primary text-2xl">Không có đánh giá ở bộ lọc/trang này</h2>
          {data.page > 1 && <button type="button" className="text-primary underline" onClick={() => setParams(adminReviewSearch({ ...filter, page: 1 }))}>Về trang đầu</button>}
        </section>
      ) : (
        <ul className="grid xl:grid-cols-2 gap-5">
          {data.data.map((item) => (
            <li key={item.id} className="bg-surface border border-outline-variant p-5 sm:p-6 space-y-4 min-w-0">
              <div className="flex flex-wrap justify-between gap-3">
                <div className="min-w-0">
                  <h2 className="font-title-editorial text-xl text-primary break-words">
                    {item.product_slug ? <Link to={`/san-pham/${encodeURIComponent(item.product_slug)}`} className="underline">{item.product_name || item.product_slug}</Link> : item.product_name || 'Sản phẩm không còn tồn tại'}
                  </h2>
                  <p className="text-on-surface-variant break-words">{item.display_name}</p>
                </div>
                <StarRating value={item.rating} />
              </div>
              {item.content && <p className="whitespace-pre-line break-words">{item.content}</p>}
              {item.images.length > 0 && (
                <ul className="text-sm space-y-1">
                  {item.images.map((image, index) => (
                    <li key={`${image.src}-${index}`} className="break-all">
                      <a href={image.src} target="_blank" rel="noopener noreferrer nofollow" className="underline">{image.alt || image.src}</a>
                    </li>
                  ))}
                </ul>
              )}
              <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
                <dt className="text-on-surface-variant">Đơn hàng</dt>
                <dd>{item.order_number || 'Không có'}</dd>
                <dt className="text-on-surface-variant">Ngày gửi</dt>
                <dd><time dateTime={item.created_at}>{formatOrderDate(item.created_at)}</time></dd>
                {item.updated_at !== item.created_at && (
                  <>
                    <dt className="text-on-surface-variant">Cập nhật</dt>
                    <dd><time dateTime={item.updated_at}>{formatOrderDate(item.updated_at)}</time></dd>
                  </>
                )}
              </dl>
              <button type="button" disabled={deleting === item.id} className="border border-error text-error px-5 py-2 disabled:opacity-50" onClick={() => void remove(item.id)}>
                {deleting === item.id ? 'Đang xóa...' : 'Xóa đánh giá'}
              </button>
            </li>
          ))}
        </ul>
      )}
      {data.total > 0 && (
        <nav aria-label="Phân trang đánh giá" className="flex flex-wrap justify-between items-center gap-4 border-t border-outline-variant pt-5">
          <p>{data.total} đánh giá · Trang {data.page}/{pages} · {data.page_size}/trang</p>
          <div className="flex gap-3">
            <button type="button" className="border border-primary px-4 py-2 disabled:opacity-40" disabled={data.page <= 1} onClick={() => setParams(adminReviewSearch({ ...filter, page: data.page - 1 }))}>Trang trước</button>
            <button type="button" className="border border-primary px-4 py-2 disabled:opacity-40" disabled={data.page >= pages || data.page >= 100000} onClick={() => setParams(adminReviewSearch({ ...filter, page: data.page + 1 }))}>Trang sau</button>
          </div>
        </nav>
      )}
    </AdminOrderFrame>
  )
}
