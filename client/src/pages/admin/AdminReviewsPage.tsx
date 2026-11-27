import { useCallback, useState } from 'react'
import { Link, Navigate, useLocation, useSearchParams } from 'react-router-dom'
import Modal from '@/components/ui/Modal'
import StarRating from '@/components/ui/StarRating'
import { useAuth } from '@/features/auth/useAuth'
import {
  adminReviewActions,
  adminReviewSearch,
  loadAdminReviews,
  performAdminReviewAction,
  readAdminReviewFilters,
  type AdminReview,
  type AdminReviewAction,
  type AdminReviewVisibility,
} from '@/features/admin/reviews'
import { formatOrderDate, orderLoginPath } from '@/features/orders/presentation'
import { orderPageCount } from '@/features/orders/read'
import { useOrderRead } from '@/features/orders/useOrderRead'
import AdminOrderFrame from './AdminOrderFrame'
import AdminOrderNotice from './AdminOrderNotice'
import { Button } from './ui'
import { inputClass } from './uiStyles'

const buttonClass = 'border border-primary text-primary px-5 py-3 disabled:opacity-40'
const title = 'Quản lý đánh giá'

export default function AdminReviewsPage() {
  const { user, loading, error } = useAuth()
  const [params, setParams] = useSearchParams()
  const location = useLocation()
  const filter = readAdminReviewFilters(params)
  const { rating, visibility, page, page_size } = filter
  const [revision, setRevision] = useState(0)
  const [dialog, setDialog] = useState<{ review: AdminReview; action: AdminReviewAction } | null>(null)
  const [busy, setBusy] = useState(false)
  const [notice, setNotice] = useState<{ kind: 'ok' | 'error'; text: string } | null>(null)
  const [redirect, setRedirect] = useState<'unauthenticated' | 'forbidden' | null>(null)
  const load = useCallback(() => loadAdminReviews({ rating, visibility, page, page_size }), [rating, visibility, page, page_size])
  const scope = JSON.stringify([user?.id, rating, visibility, page, page_size, revision])
  const result = useOrderRead(scope, !!user && !loading && !error, load)
  const retry = () => setRevision((value) => value + 1)
  const next = location.pathname + location.search
  if (redirect === 'unauthenticated') return <Navigate to={orderLoginPath(next)} replace />
  if (redirect === 'forbidden') return <Navigate to="/403" replace />
  if (!result || result.kind !== 'ready') return <AdminOrderNotice title={title} result={result} next={next} retry={retry} />
  const data = result.page
  const pages = orderPageCount(data)
  const picked = dialog ? adminReviewActions[dialog.action] : null

  async function confirm() {
    if (!dialog || busy) return
    setBusy(true)
    setNotice(null)
    const outcome = await performAdminReviewAction(dialog.review.id, dialog.action)
    setBusy(false)
    const action = adminReviewActions[dialog.action]
    setDialog(null)
    if (outcome.kind === 'unauthenticated' || outcome.kind === 'forbidden') return setRedirect(outcome.kind)
    if (outcome.kind === 'done') setNotice({ kind: 'ok', text: action.done })
    else if (outcome.kind === 'not-found') setNotice({ kind: 'ok', text: outcome.message })
    else return setNotice({ kind: 'error', text: outcome.message })
    retry()
  }

  return (
    <AdminOrderFrame title={title}>
      <section className="bg-surface p-5 space-y-3">
        <p className="text-sm text-on-surface-variant">
          Đánh giá không phù hợp có thể được <strong>ẩn</strong> (gỡ khỏi trang sản phẩm, hiện lại được) hoặc <strong>xóa vĩnh viễn</strong>. Điểm trung bình và số lượt đánh giá chỉ tính các đánh giá đang hiện và được tính lại ngay.
        </p>
        <div className="flex flex-wrap items-end gap-3">
          <label className="space-y-1 text-sm">
            <span className="block font-medium">Điểm</span>
            <select className={`${inputClass} w-auto min-w-40`} value={rating} onChange={(event) => setParams(adminReviewSearch({ ...filter, rating: Number(event.target.value), page: 1 }))}>
              <option value={0}>Tất cả</option>
              {[1, 2, 3, 4, 5].map((value) => <option key={value} value={value}>{value} sao</option>)}
            </select>
          </label>
          <label className="space-y-1 text-sm">
            <span className="block font-medium">Hiển thị</span>
            <select className={`${inputClass} w-auto min-w-40`} value={visibility} onChange={(event) => setParams(adminReviewSearch({ ...filter, visibility: event.target.value as AdminReviewVisibility, page: 1 }))}>
              <option value="">Tất cả</option>
              <option value="visible">Đang hiện</option>
              <option value="hidden">Đang ẩn</option>
            </select>
          </label>
          <label className="space-y-1 text-sm">
            <span className="block font-medium">Đánh giá/trang</span>
            <select className={`${inputClass} w-auto`} value={page_size} onChange={(event) => setParams(adminReviewSearch({ ...filter, page_size: Number(event.target.value), page: 1 }))}>
              {[...new Set([20, 50, 100, page_size])].sort((a, b) => a - b).map((size) => <option key={size} value={size}>{size}</option>)}
            </select>
          </label>
          <Button variant="secondary" onClick={retry}>Tải lại</Button>
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
            <li key={item.id} className={`bg-surface border p-5 sm:p-6 space-y-4 min-w-0 ${item.hidden ? 'border-dashed border-outline' : 'border-outline-variant'}`}>
              <div className="flex flex-wrap justify-between gap-3">
                <div className="min-w-0">
                  <h2 className="font-title-editorial text-xl text-primary break-words">
                    {item.product_slug ? <Link to={`/san-pham/${encodeURIComponent(item.product_slug)}`} className="underline">{item.product_name || item.product_slug}</Link> : item.product_name || 'Sản phẩm không còn tồn tại'}
                  </h2>
                  <p className="text-on-surface-variant break-words">{item.display_name}</p>
                </div>
                <div className="flex flex-col items-end gap-2">
                  <StarRating value={item.rating} />
                  {item.hidden && <span className="bg-surface-container-high px-3 py-1 text-sm">Đang ẩn</span>}
                </div>
              </div>
              <div className={item.hidden ? 'opacity-60' : ''}>
                {item.content && <p className="whitespace-pre-line break-words">{item.content}</p>}
                {item.images.length > 0 && (
                  <ul className="text-sm space-y-1 mt-3">
                    {item.images.map((image, index) => (
                      <li key={`${image.src}-${index}`} className="break-all">
                        <a href={image.src} target="_blank" rel="noopener noreferrer nofollow" className="underline">{image.alt || image.src}</a>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
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
              <div className="flex flex-wrap gap-3">
                <button type="button" className="border border-primary text-primary px-5 py-2" onClick={() => setDialog({ review: item, action: item.hidden ? 'show' : 'hide' })}>
                  {item.hidden ? 'Hiện lại' : 'Ẩn đánh giá'}
                </button>
                <button type="button" className="border border-error text-error px-5 py-2" onClick={() => setDialog({ review: item, action: 'delete' })}>
                  Xóa đánh giá
                </button>
              </div>
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
      <Modal
        open={!!dialog}
        onClose={() => {
          if (!busy) setDialog(null)
        }}
      >
        {dialog && picked && (
          <div className="space-y-5">
            <h2 className="font-headline-md text-2xl text-primary">
              {picked.label} · {dialog.review.display_name}
            </h2>
            <p className="text-sm text-on-surface-variant">
              {dialog.review.product_name || dialog.review.product_slug} · {dialog.review.rating} sao
            </p>
            <p>{picked.confirmation}</p>
            <div className="flex flex-wrap gap-3">
              <button
                type="button"
                disabled={busy}
                className={`${dialog.action === 'delete' ? 'bg-error text-on-error' : 'bg-primary text-on-primary'} px-5 py-3 disabled:opacity-40`}
                onClick={() => void confirm()}
              >
                {busy ? 'Đang gửi...' : 'Xác nhận thao tác'}
              </button>
              <button type="button" disabled={busy} className={buttonClass} onClick={() => setDialog(null)}>
                Quay lại
              </button>
            </div>
          </div>
        )}
      </Modal>
    </AdminOrderFrame>
  )
}
