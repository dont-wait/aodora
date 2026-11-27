import { useEffect, useState, type FormEvent } from 'react'
import { Link, useLocation } from 'react-router-dom'
import StarRating from '@/components/ui/StarRating'
import CatalogImage from '@/features/catalog/CatalogImage'
import Icon from '@/components/ui/Icon'
import type { CatalogProduct, CatalogReviewPage } from '@/features/catalog/types'
import { formatReviewImages, parseReviewImages, reviewApi, reviewErrorMessage, type EligibleOrder, type OwnReview } from '@/features/review/api'
import { useAuth } from '@/features/auth/useAuth'
import { useToast } from '@/hooks/useToast'
import { ROUTES } from '@/data/site'

const input = 'w-full bg-surface-container px-4 py-3 border border-outline-variant/60 focus:border-primary focus:outline-none'
type ReviewDraft = { order_id: string; rating: number; content: string }
const emptyForm = (orderId = ''): ReviewDraft => ({ order_id: orderId, rating: 5, content: '' })

export default function ReviewsSection({ product }: { product: CatalogProduct }) {
  const { user } = useAuth()
  const { showToast } = useToast()
  const { hash } = useLocation()
  const [page, setPage] = useState<CatalogReviewPage>({ items: [], total: 0, page: 1, page_size: 6, average: product.reviews.average })
  const [orders, setOrders] = useState<EligibleOrder[]>([])
  const [ownReview, setOwnReview] = useState<OwnReview | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [formError, setFormError] = useState('')
  const [form, setForm] = useState<ReviewDraft>(emptyForm())
  const [imagesText, setImagesText] = useState('')
  // The form stays closed while the customer's saved review is shown; it is
  // reopened (prefilled) only through "Sửa đánh giá".
  const [editing, setEditing] = useState(false)

  async function loadReviews(nextPage = 1) {
    setLoading(true); setError('')
    try { setPage(await reviewApi.list(product.slug, nextPage)) }
    catch (err) { setError(err instanceof Error ? err.message : 'Không thể tải đánh giá') }
    finally { setLoading(false) }
  }

  useEffect(() => {
    let active = true
    reviewApi.list(product.slug)
      .then((reviews) => { if (active) setPage(reviews) })
      .catch((err: unknown) => { if (active) setError(err instanceof Error ? err.message : 'Không thể tải đánh giá') })
      .finally(() => { if (active) setLoading(false) })
    if (user) {
      reviewApi.own(product.id)
        .then((own) => { if (active) setOwnReview(own) })
        .catch(() => { if (active) setOwnReview(null) })
      reviewApi.eligibleOrders(product.id)
        .then((eligible) => {
          if (!active) return
          setOrders(eligible)
          if (eligible[0]) setForm((current) => ({ ...current, order_id: current.order_id || eligible[0].id }))
        })
        .catch(() => { if (active) setFormError('Không thể tải các đơn đủ điều kiện đánh giá.') })
    }
    return () => { active = false }
  }, [product.id, product.slug, user])

  // Links from "Đơn hàng của tôi" open the page at the review section once
  // the reviews have loaded (the layout scrolls to top on navigation first).
  useEffect(() => {
    if (!loading && hash === '#danh-gia-khach-hang') document.getElementById('danh-gia-khach-hang')?.scrollIntoView({ behavior: 'smooth' })
  }, [loading, hash])

  async function submit(event: FormEvent) {
    event.preventDefault(); setSaving(true); setFormError('')
    try {
      const images = parseReviewImages(imagesText)
      const saved = await reviewApi.submit(product.id, { ...form, images })
      setOwnReview(saved); showToast('Đã lưu đánh giá của bạn.')
      setForm(emptyForm(orders[0]?.id)); setImagesText(''); setEditing(false)
      await loadReviews(page.page)
    } catch (err) {
      setFormError(reviewErrorMessage(err))
    } finally { setSaving(false) }
  }

  function startEdit(own: OwnReview) {
    setForm({ order_id: own.order_id, rating: own.rating, content: own.content || '' })
    setImagesText(formatReviewImages(own.images)); setFormError(''); setEditing(true)
  }
  function cancelEdit() {
    setForm(emptyForm(orders[0]?.id)); setImagesText(''); setFormError(''); setEditing(false)
  }

  const average = page.total ? page.average : product.reviews.average
  const count = page.total || product.reviews.count
  return <section id="danh-gia-khach-hang" className="w-full bg-surface-container-lowest py-16">
    <div className="max-w-[1440px] mx-auto px-4 sm:px-8 lg:px-margin-desktop">
      <div className="flex flex-col md:flex-row md:items-end md:justify-between gap-5 mb-8">
        <div><p className="uppercase tracking-widest text-secondary text-sm">Khách hàng đã nhận xét</p><h2 className="font-headline-lg text-headline-lg text-primary mt-2">Đánh Giá Sản Phẩm</h2></div>
        <div className="flex items-center gap-3 bg-surface-container-low px-5 py-3"><StarRating value={Math.round(average)} /><span className="text-sm text-on-surface-variant">{average.toFixed(1)}/5 · {count} đánh giá</span></div>
      </div>
      {error && <div className="border border-red-200 bg-red-50 p-4 flex items-center justify-between gap-4"><p role="alert" className="text-red-700">{error}</p><button type="button" className="underline text-primary" onClick={() => void loadReviews(page.page)}>Thử lại</button></div>}
      {!error && loading && <p role="status" className="py-10 text-on-surface-variant">Đang tải đánh giá...</p>}
      {!error && !loading && <>
        {page.items.length ? <div className="grid grid-cols-1 md:grid-cols-3 gap-5">{page.items.map((review) => <article key={review.id} className="bg-surface p-5 border border-outline-variant/50"><div className="flex items-center justify-between gap-3"><StarRating value={review.rating} /><time className="text-xs text-on-surface-variant" dateTime={review.created_at}>{formatDate(review.created_at)}</time></div><p className="font-semibold text-primary mt-4">{review.display_name}</p>{review.verified_purchase && <span className="inline-flex items-center gap-1 text-xs text-secondary mt-1"><span aria-hidden="true">✓</span> Đã mua hàng</span>}{review.content && <p className="text-sm leading-6 text-on-surface-variant mt-3 whitespace-pre-line">{review.content}</p>}{review.images.length > 0 && <div className="grid grid-cols-3 gap-2 mt-4">{review.images.map((image, index) => <CatalogImage key={`${image.src}-${index}`} src={image.src} alt={image.alt} className="aspect-square" />)}</div>}</article>)}</div> : <p className="py-8 text-on-surface-variant">Chưa có nhận xét cho mẫu áo này.</p>}
        {page.total > page.page_size && <div className="flex items-center justify-center gap-4 mt-8"><button type="button" disabled={page.page <= 1} onClick={() => void loadReviews(page.page - 1)} className="border border-outline-variant px-4 py-2 disabled:opacity-40">Trước</button><span className="text-sm text-on-surface-variant">Trang {page.page}</span><button type="button" disabled={page.page * page.page_size >= page.total} onClick={() => void loadReviews(page.page + 1)} className="border border-outline-variant px-4 py-2 disabled:opacity-40">Sau</button></div>}
      </>}
      <ReviewForm user={user} orders={orders} ownReview={ownReview} editing={editing} onEdit={startEdit} onCancel={cancelEdit} form={form} setForm={setForm} imagesText={imagesText} setImagesText={setImagesText} formError={formError} saving={saving} onSubmit={submit} />
    </div>
  </section>
}

function ReviewForm({ user, orders, ownReview, editing, onEdit, onCancel, form, setForm, imagesText, setImagesText, formError, saving, onSubmit }: { user: ReturnType<typeof useAuth>['user']; orders: EligibleOrder[]; ownReview: OwnReview | null; editing: boolean; onEdit: (own: OwnReview) => void; onCancel: () => void; form: ReviewDraft; setForm: (value: ReviewDraft) => void; imagesText: string; setImagesText: (value: string) => void; formError: string; saving: boolean; onSubmit: (event: FormEvent) => void }) {
  if (!user) return <div className="mt-10 border-t border-outline-variant/50 pt-8"><p className="text-on-surface-variant">Đăng nhập để đánh giá sau khi bạn đã nhận sản phẩm. Chỉ tài khoản có đơn đã nhận mẫu áo này mới gửi được đánh giá.</p><Link to={`${ROUTES.login}?next=${encodeURIComponent(window.location.pathname)}`} className="inline-block mt-3 text-primary underline">Đăng nhập</Link></div>
  if (!orders.length && !ownReview) return <div className="mt-10 border-t border-outline-variant/50 pt-8"><p className="text-on-surface-variant">Chỉ khách hàng đã nhận mẫu áo này mới có thể gửi đánh giá.</p></div>
  if (ownReview && !editing) return <div className="mt-10 border-t border-outline-variant/50 pt-8 max-w-3xl"><p className="font-label-uppercase text-label-uppercase text-secondary tracking-widest">Trải nghiệm của bạn</p><h3 className="font-headline-md text-primary mt-2">Bạn đã đánh giá mẫu áo này</h3><div className="mt-4"><StarRating value={ownReview.rating} /></div>{ownReview.content && <p className="text-sm leading-6 text-on-surface-variant mt-3 whitespace-pre-line">{ownReview.content}</p>}<button type="button" onClick={() => onEdit(ownReview)} className="mt-5 border border-primary text-primary px-6 py-3">Sửa đánh giá</button></div>
  return <form onSubmit={onSubmit} className="mt-10 border-t border-outline-variant/50 pt-8 max-w-3xl"><p className="font-label-uppercase text-label-uppercase text-secondary tracking-widest">Trải nghiệm của bạn</p><h3 className="font-headline-md text-primary mt-2">{ownReview ? 'Cập nhật đánh giá' : 'Viết đánh giá của bạn'}</h3><div className="grid sm:grid-cols-2 gap-5 mt-5"><label className="block space-y-2"><span className="text-sm text-on-surface-variant">Đơn hàng</span><select required value={form.order_id} onChange={(event) => setForm({ ...form, order_id: event.target.value })} className={input}><option value="">Chọn đơn đã nhận</option>{orders.map((order) => <option key={order.id} value={order.id}>{order.order_number} · {order.created_at}</option>)}</select></label><div><span className="block text-sm text-on-surface-variant mb-2">Điểm đánh giá</span><div className="flex gap-1">{[1, 2, 3, 4, 5].map((rating) => <button key={rating} type="button" aria-label={`${rating} sao`} aria-pressed={form.rating === rating} onClick={() => setForm({ ...form, rating })} className={`w-10 h-10 inline-flex items-center justify-center ${form.rating >= rating ? 'text-secondary bg-secondary-fixed' : 'text-outline bg-surface-container'}`}><Icon name="star" filled={form.rating >= rating} size={18} /></button>)}</div></div></div><label className="block space-y-2 mt-5"><span className="text-sm text-on-surface-variant">Nhận xét</span><textarea rows={5} maxLength={3000} value={form.content} onChange={(event) => setForm({ ...form, content: event.target.value })} className={input} placeholder="Chia sẻ cảm nhận của bạn về mẫu áo..." /></label><label className="block space-y-2 mt-5"><span className="text-sm text-on-surface-variant">Ảnh (mỗi dòng URL | mô tả, không bắt buộc)</span><textarea rows={3} value={imagesText} onChange={(event) => setImagesText(event.target.value)} className={input} placeholder="https://... | Ảnh áo sau khi nhận" /></label>{formError && <p role="alert" className="text-red-700 mt-4">{formError}</p>}<button disabled={saving} type="submit" className="mt-5 bg-primary text-on-primary px-6 py-3 disabled:opacity-50">{saving ? 'Đang lưu...' : ownReview ? 'Cập nhật đánh giá' : 'Gửi đánh giá'}</button>{ownReview && <button type="button" disabled={saving} onClick={onCancel} className="mt-5 ml-3 border border-outline-variant px-6 py-3 disabled:opacity-50">Hủy</button>}</form>
}

function formatDate(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? '' : date.toLocaleDateString('vi-VN') }
