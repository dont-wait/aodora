import AddressFields from '@/features/address/AddressFields'
import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import Breadcrumb from '@/components/ui/Breadcrumb'
import { accountApi, type Address, type Profile } from '@/features/account/api'
import { useAuth } from '@/features/auth/useAuth'
import { useCart } from '@/features/cart/useCart'
import { cartApi } from '@/features/cart/api'
import { ordersApi, OrderApiError } from '@/features/orders/api'
import {
  attemptQuoteExpired,
  clearAttempt,
  CheckoutIdentityError,
  isDefinitiveOrderError,
  lookupAttempt,
  makeAttempt,
  matchesIdentity,
  quoteDifferences,
  readAttempt,
  saveAttempt,
  sendAttempt,
} from '@/features/orders/checkout'
import { completeCheckout } from '@/features/orders/completion'
import { formatOrderMoney } from '@/features/orders/money'
import type { CheckoutAttempt, CheckoutBody, Order, PaymentMethod, Quote, ShippingAddress } from '@/features/orders/types'
import { SkeletonBlock, SkeletonStatus } from '@/components/ui/Skeleton'

const inputClass = 'w-full border border-outline-variant bg-surface-container-lowest px-4 py-3 focus:outline-primary'
const buttonClass = 'bg-primary px-6 py-3 text-on-primary disabled:opacity-50 disabled:cursor-not-allowed'
const emptyAddress: ShippingAddress = { recipient: '', phone: '', line1: '', ward: '', city: '' }
function initialCheckout() {
  try {
    const attempt = readAttempt()
    return { attempt, body: attempt ? (JSON.parse(attempt.body) as CheckoutBody) : null, error: '' }
  } catch (error) {
    return { attempt: null, body: null, error: error instanceof Error ? error.message : 'Không đọc được dữ liệu đặt hàng.' }
  }
}
function messageFor(error: unknown) {
  if (error instanceof CheckoutIdentityError) return error.message
  if (error instanceof OrderApiError) {
    if (error.code === 'QUOTE_EXPIRED')
      return 'Đơn chưa được gửi vì giá chỉ được giữ 15 phút và đã hết hiệu lực. Thông tin nhận áo vẫn được giữ: bấm “Tính tổng tiền” bên dưới rồi xác nhận để đặt lại.'
    if (error.code === 'QUOTE_CHANGED')
      return 'Đơn chưa được gửi vì giá hoặc giỏ hàng đã thay đổi. Bấm “Tính tổng tiền” bên dưới để xem giá mới rồi xác nhận lại.'
    if (error.code === 'SHIPPING_UNAVAILABLE')
      return 'COD tạm không khả dụng vì chưa có phí giao hàng. Bạn có thể chọn nhận và thanh toán tại quầy khi không còn đơn nào đang gửi.'
    return error.message
  }
  return 'Chưa nhận được phản hồi rõ ràng. Vui lòng kiểm tra kết nối và tra cứu lại; không đặt đơn mới ở cửa sổ khác.'
}

export default function CheckoutPage() {
  const [initial] = useState(initialCheckout)
  const [attempt, setAttempt] = useState<CheckoutAttempt | null>(initial.attempt)
  const [address, setAddress] = useState<ShippingAddress>(initial.body?.shipping_address ?? emptyAddress)
  const [payment, setPayment] = useState<PaymentMethod>(initial.body?.payment_method ?? 'cod')
  const [note, setNote] = useState(initial.body?.note ?? '')
  // District of a saved pre-07/2025 address; only used to flag that the ward must be chosen again.
  const [legacyDistrict, setLegacyDistrict] = useState('')
  const addressRef = useRef(address)
  useEffect(() => {
    addressRef.current = address
  }, [address])
  const [profile, setProfile] = useState<Profile | null>(null)
  const [profileError, setProfileError] = useState('')
  const [quote, setQuote] = useState<Quote | null>(null)
  const [quoteFor, setQuoteFor] = useState('')
  const [changes, setChanges] = useState<string[]>([])
  const [accepted, setAccepted] = useState(false)
  const [expired, setExpired] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(initial.error)
  const [notice, setNotice] = useState('')
  // Lookup ran and the server has no order for the saved request: nothing was placed.
  const [notPlaced, setNotPlaced] = useState(false)
  const { user, loading: authLoading } = useAuth()
  const { cart, loading: cartLoading, error: cartError, refresh, consumePurchased, selectedItems: items, selectedSubtotal } = useCart()
  const customerId = user?.id ?? null
  const navigate = useNavigate()
  const work = useRef(false)
  const mounted = useRef(true)
  const recovered = useRef('')
  const previousQuote = useRef<Quote | null>(null)
  useEffect(() => {
    mounted.current = true
    return () => {
      mounted.current = false
    }
  }, [])

  const finish = useCallback(
    async (order: Order, pending: CheckoutAttempt) => {
      await completeCheckout(order, pending, consumePurchased)
      if (mounted.current) navigate(`/dat-hang-thanh-cong/${encodeURIComponent(order.id)}`, { replace: true })
    },
    [consumePurchased, navigate],
  )

  // On reload, lookup ALWAYS precedes any retry. Never attach a guest token
  // while logged into an account, or query a different account's scope.
  useEffect(() => {
    if (authLoading || !attempt || !matchesIdentity(attempt.credential, customerId) || work.current) return
    const key = `${attempt.request_id}:${customerId ?? 'guest'}`
    if (recovered.current === key) return
    recovered.current = key
    work.current = true
    setBusy(true)
    setError('')
    setNotice('Đang kiểm tra đơn bạn vừa gửi...')
    lookupAttempt(attempt, customerId)
      .then(async (order) => {
        if (order) await finish(order, attempt)
        else {
          setNotice('')
          setNotPlaced(true)
        }
      })
      .catch((error: unknown) => setError(messageFor(error)))
      .finally(() => {
        work.current = false
        if (mounted.current) setBusy(false)
      })
  }, [attempt, authLoading, customerId, finish])

  useEffect(() => {
    if (!user || attempt) return
    let active = true
    accountApi
      .profile()
      .then((next) => {
        if (!active) return
        setProfile(next)
        setProfileError('')
        const saved = next.addresses.find((item) => item.is_default) ?? next.addresses[0]
        setAddress((current) => ({
          recipient: current.recipient || saved?.recipient || next.full_name,
          phone: current.phone || saved?.phone || next.phone || '',
          line1: current.line1 || saved?.line1 || '',
          ward: current.ward || saved?.ward || '',
          city: current.city || saved?.city || '',
        }))
        if (!addressRef.current.city && !addressRef.current.ward) setLegacyDistrict(saved?.district || '')
      })
      .catch(() => {
        if (active) setProfileError('Chưa tải được địa chỉ đã lưu. Bạn vẫn có thể nhập thông tin nhận áo bên dưới.')
      })
    return () => {
      active = false
    }
  }, [user, attempt])

  useEffect(() => {
    if (!quote) return
    const delay = Date.parse(quote.expires_at) - Date.now()
    const timer = window.setTimeout(
      () => {
        setExpired(true)
        setAccepted(false)
      },
      Math.max(0, Math.min(delay, 2147483647)),
    )
    return () => window.clearTimeout(timer)
  }, [quote])

  const body: CheckoutBody = {
    payment_method: payment,
    shipping_address: payment === 'cod' ? { ...address } : { recipient: address.recipient, phone: address.phone },
    items: items.map((item) => ({ product_id: item.product_id, sku: item.sku, quantity: item.quantity })),
    ...(note.trim() ? { note } : {}),
  }
  const bodyKey = JSON.stringify(body)
  const quoteCurrent = quote !== null && quoteFor === bodyKey && !expired
  const cartLineOf = (item: { product_id: string; sku: string }) => cart.items.find((line) => line.product_id === item.product_id && line.sku === item.sku)
  const blocked = busy || authLoading || cartLoading || !!initial.error
  const identityMatches = !attempt || matchesIdentity(attempt.credential, customerId)
  // Only once lookup proved nothing was placed: resending an expired quote would just be refused.
  const staleQuote = notPlaced && !!attempt && attemptQuoteExpired(attempt)

  function selectAddress(saved: Address) {
    setAddress({ recipient: saved.recipient, phone: saved.phone, line1: saved.line1, ward: saved.ward ?? '', city: saved.city })
    setLegacyDistrict(saved.district ?? '')
    setAccepted(false)
  }
  function changeAddress(field: keyof ShippingAddress, value: string) {
    setAddress((current) => ({ ...current, [field]: value }))
    setAccepted(false)
  }
  const changeRegion = useCallback((next: { city: string; ward: string }) => {
    setAddress((current) => (current.city === next.city && current.ward === next.ward ? current : { ...current, ...next }))
    setLegacyDistrict('')
    setAccepted(false)
  }, [])
  async function getQuote(event: FormEvent) {
    event.preventDefault()
    if (work.current || attempt || blocked || !items.length || cartError) return
    work.current = true
    setBusy(true)
    setError('')
    setNotice('')
    setAccepted(false)
    try {
      const checked = await cartApi.validate(
        items.map((item) => ({
          product_id: item.product_id,
          product_slug: item.product_slug,
          sku: item.sku,
          quantity: item.quantity,
          color: item.color,
          size: item.size,
          fulfillment_mode: item.fulfillment_mode,
        })),
      )
      if (checked.issues.length) throw new OrderApiError(409, 'OUT_OF_STOCK', 'items', 'Một số dòng áo không còn đủ tồn kho. Vui lòng kiểm tra lại giỏ hàng.')
      const stockChanges = checked.items.flatMap((item) => {
        const old = items.find((line) => line.product_id === item.product_id && line.sku === item.sku)
        return old && old.stock !== item.stock ? [`${item.name || 'Sản phẩm áo dài'}: chỉ còn ${item.stock} sản phẩm.`] : []
      })
      const next = await ordersApi.quote(body)
      const baseline = previousQuote.current ?? {
        ...next,
        subtotal: String(selectedSubtotal),
        items: items.map((item) => ({ product_id: item.product_id, sku: item.sku, quantity: item.quantity, unit_price: String(item.unit_price) })),
      }
      setChanges([...quoteDifferences(baseline, next), ...stockChanges])
      previousQuote.current = next
      setQuote(next)
      setQuoteFor(bodyKey)
      setExpired(false)
    } catch (error) {
      setQuote(null)
      setError(messageFor(error))
      if (error instanceof OrderApiError && ['OUT_OF_STOCK', 'QUOTE_CHANGED', 'QUOTE_EXPIRED'].includes(error.code)) {
        setChanges(['Giá hoặc số lượng còn lại đã thay đổi. Vui lòng kiểm tra lại giỏ hàng và bấm “Tính tổng tiền” lần nữa.'])
        await refresh()
      }
    } finally {
      work.current = false
      setBusy(false)
    }
  }
  async function sendOrder() {
    if (work.current || blocked || !identityMatches) return
    if (!attempt && (!quoteCurrent || !accepted || !items.length || cartError)) return
    if (!attempt && quote && Date.parse(quote.expires_at) <= Date.now()) {
      setExpired(true)
      setAccepted(false)
      return
    }
    work.current = true
    setBusy(true)
    setError('')
    setNotice('')
    let pending = attempt
    try {
      if (!pending) {
        pending = makeAttempt(
          { ...body, quote_token: quote!.quote_token },
          items.map((item) => ({ product_id: item.product_id, sku: item.sku, quantity: item.quantity, name: item.name })),
          customerId,
          quote!.expires_at,
        )
        try {
          saveAttempt(pending)
        } catch {
          pending = null
          throw new Error('Không lưu được dữ liệu đặt hàng trên trình duyệt. Chưa gửi đơn; vui lòng kiểm tra dung lượng/quyền lưu trữ rồi thử lại.')
        }
        setAttempt(pending)
      } else {
        const found = await lookupAttempt(pending, customerId)
        if (found) {
          await finish(found, pending)
          return
        }
      }
      let order: Order
      try {
        order = await sendAttempt(pending, customerId)
      } catch (error) {
        if (isDefinitiveOrderError(error)) {
          clearAttempt(pending.request_id)
          setAttempt(null)
          setQuote(null)
          setAccepted(false)
          setChanges(['Đơn này chưa gửi thành công. Vui lòng xem lại thông tin, bấm “Tính tổng tiền” và xác nhận lại.'])
          if (error instanceof OrderApiError && ['OUT_OF_STOCK', 'QUOTE_CHANGED'].includes(error.code)) await refresh()
        }
        throw error
      }
      // Storage/cart failure after this point is never a failed create.
      await finish(order, pending)
    } catch (error) {
      setError(error instanceof Error && !(error instanceof OrderApiError) && !pending ? error.message : messageFor(error))
      if (pending && !isDefinitiveOrderError(error)) setNotice('Nếu chưa có kết quả rõ ràng, đơn vẫn được giữ lại để bạn kiểm tra và gửi lại an toàn.')
    } finally {
      work.current = false
      if (mounted.current) setBusy(false)
    }
  }

  return (
    <main className="max-w-[1180px] mx-auto px-5 py-8 lg:py-14 space-y-8">
      <Breadcrumb items={[{ label: 'Trang chủ', to: '/' }, { label: 'Giỏ hàng', to: '/gio-hang' }, { label: 'Đặt mua áo' }]} />
      <header>
        <p className="text-secondary text-sm tracking-widest uppercase">Aodora · Đặt mua áo</p>
        <h1 className="font-headline-lg text-4xl text-primary mt-2">Thông tin nhận áo</h1>
        <p className="mt-3 text-on-surface-variant">
          {user ? <>Đặt hàng với tài khoản <strong>{user.full_name || user.email}</strong>; địa chỉ đã lưu được điền sẵn.</> : 'Đặt hàng bằng thông tin liên hệ của bạn, không cần tạo tài khoản.'}
        </p>
      </header>
      {error && (
        <p role="alert" className="border border-error/30 bg-error-container p-4 text-on-error-container">
          {error}
        </p>
      )}
      {notice && (
        <p role="status" className="border border-outline-variant p-4">
          {notice}
        </p>
      )}
      {authLoading ? (
        <SkeletonStatus label="Đang kiểm tra đăng nhập" className="grid lg:grid-cols-[1.1fr_1fr] gap-8 items-start">
          <div className="space-y-4">
            <SkeletonBlock className="h-16 w-full" />
            <SkeletonBlock className="h-16 w-full" />
            <SkeletonBlock className="h-12 w-full" />
            <SkeletonBlock className="h-12 w-full" />
            <SkeletonBlock className="h-24 w-full" />
          </div>
          <div className="bg-surface-container-low p-6 sm:p-8 space-y-4">
            <SkeletonBlock className="h-7 w-1/2" />
            <SkeletonBlock className="h-4 w-full" />
            <SkeletonBlock className="h-4 w-3/4" />
            <SkeletonBlock className="h-12 w-full" />
          </div>
        </SkeletonStatus>
      ) : attempt ? (
        <section className="bg-surface-container-low p-6 space-y-5">
          <h2 className="font-headline-md text-2xl text-primary">{notPlaced ? 'Đơn trước chưa gửi thành công' : 'Đơn đang gửi, chờ kết quả'}</h2>
          <p>
            {notPlaced && staleQuote
              ? 'Lần đặt hàng trước bị gián đoạn nên cửa hàng chưa nhận được đơn này (chưa có đơn nào được tạo, bạn chưa bị tính tiền). Giá trong đơn chỉ được giữ 15 phút và đã hết hiệu lực, nên cần tính lại tổng tiền. Thông tin nhận áo vẫn được giữ; đơn mới gồm các áo đang chọn trong giỏ.'
              : notPlaced
              ? 'Lần đặt hàng trước bị gián đoạn nên cửa hàng chưa nhận được đơn này (chưa có đơn nào được tạo, bạn chưa bị tính tiền). Bạn có thể gửi lại đúng đơn bên dưới, hoặc bỏ đơn này để đặt lại từ giỏ hàng.'
              : 'Nội dung đã gửi được giữ nguyên để tránh tạo hai đơn. Bạn có thể xem hoặc thêm áo trong giỏ; đơn đang gửi này vẫn chỉ gồm các dòng bên dưới.'}
          </p>
          <ul className="space-y-2">
            {attempt.snapshot.map((line) => (
              <li key={`${line.product_id}:${line.sku}`}>
                {line.name || 'Sản phẩm áo dài'} · {line.quantity} áo
              </li>
            ))}
          </ul>
          {!identityMatches && (
            <p role="alert">
              {attempt.credential.kind === 'account'
                ? 'Vui lòng đăng nhập lại đúng tài khoản đã đặt đơn.'
                : 'Đơn này được gửi khi bạn chưa đăng nhập. Vui lòng đăng xuất khỏi tài khoản hiện tại để kiểm tra lại đơn trên trình duyệt này.'}{' '}
              <Link to="/dang-nhap?next=%2Fthanh-toan" className="underline">
                Đăng nhập
              </Link>
            </p>
          )}
          {staleQuote ? (
            <button type="button" className={buttonClass} disabled={busy} onClick={() => {
                clearAttempt(attempt.request_id)
                setAttempt(null)
                setNotPlaced(false)
                setNotice('')
                setError('')
              }}>
              Tính lại tổng tiền
            </button>
          ) : (
          <button type="button" className={buttonClass} disabled={busy || !identityMatches || !!initial.error} onClick={() => void sendOrder()}>
            {busy ? 'Đang tra cứu / gửi đơn...' : notPlaced ? 'Gửi lại đơn này' : 'Tra cứu và gửi lại đơn đã lưu'}
          </button>
          )}
          {notPlaced && !staleQuote && (
            <button
              type="button"
              className="ml-3 border border-outline-variant px-6 py-3 text-on-surface disabled:opacity-50"
              disabled={busy}
              onClick={() => {
                clearAttempt(attempt.request_id)
                setAttempt(null)
                setNotPlaced(false)
                setNotice('')
                setError('')
              }}
            >
              Bỏ đơn này, đặt đơn mới
            </button>
          )}
          <Link to="/gio-hang" className="block underline text-primary">
            Xem giỏ hàng
          </Link>
        </section>
      ) : (
        <div className="grid lg:grid-cols-[1.1fr_1fr] gap-8 items-start">
          <form onSubmit={getQuote} className="space-y-6">
            <fieldset disabled={blocked} className="space-y-5 disabled:opacity-60">
              <legend className="font-headline-md text-2xl text-primary mb-4">Cách nhận và thanh toán</legend>
              <label className="flex gap-3 items-start border border-outline-variant p-4">
                <input
                  type="radio"
                  name="payment"
                  checked={payment === 'cod'}
                  onChange={() => {
                    setPayment('cod')
                    setAccepted(false)
                  }}
                />
                <span>Giao tận nơi · Thanh toán khi nhận áo (COD)</span>
              </label>
              <label className="flex gap-3 items-start border border-outline-variant p-4">
                <input
                  type="radio"
                  name="payment"
                  checked={payment === 'cash_at_counter'}
                  onChange={() => {
                    setPayment('cash_at_counter')
                    setAccepted(false)
                  }}
                />
                <span>Nhận và thanh toán tại quầy · Không có phí giao hàng</span>
              </label>
              {profileError && <p role="status">{profileError}</p>}
              {!!profile?.addresses.length && payment === 'cod' && (
                <label className="block space-y-2">
                  <span>Địa chỉ đã lưu</span>
                  <select
                    className={inputClass}
                    defaultValue=""
                    onChange={(event) => {
                      const saved = profile.addresses.find((item) => item.id === event.target.value)
                      if (saved) selectAddress(saved)
                    }}
                  >
                    <option value="">Chọn địa chỉ để điền thông tin</option>
                    {profile.addresses.map((saved) => (
                      <option key={saved.id} value={saved.id}>
                        {saved.recipient} · {saved.line1}, {saved.city}
                      </option>
                    ))}
                  </select>
                </label>
              )}
              <label className="block space-y-2">
                <span>Họ tên người nhận *</span>
                <input
                  autoComplete="name"
                  className={inputClass}
                  required
                  maxLength={120}
                  value={address.recipient}
                  onChange={(event) => changeAddress('recipient', event.target.value)}
                />
              </label>
              <label className="block space-y-2">
                <span>Số điện thoại *</span>
                <input
                  autoComplete="tel"
                  type="tel"
                  className={inputClass}
                  required
                  maxLength={30}
                  value={address.phone}
                  onChange={(event) => changeAddress('phone', event.target.value)}
                />
              </label>
              {payment === 'cod' && (
                <div className="space-y-5">
                  <label className="block space-y-2">
                    <span>Địa chỉ nhận áo *</span>
                    <input
                      autoComplete="street-address"
                      className={inputClass}
                      required
                      maxLength={200}
                      value={address.line1 || ''}
                      onChange={(event) => changeAddress('line1', event.target.value)}
                    />
                  </label>
                  <AddressFields
                    idPrefix="checkout"
                    required
                    value={{ city: address.city || '', ward: address.ward || '' }}
                    legacyDistrict={legacyDistrict}
                    disabled={busy}
                    labelClassName="space-y-2 block"
                    selectClassName={inputClass}
                    onChange={changeRegion}
                  />
                </div>
              )}
              <label className="block space-y-2">
                <span>Ghi chú</span>
                <textarea
                  className={inputClass}
                  rows={3}
                  maxLength={1000}
                  value={note}
                  onChange={(event) => {
                    setNote(event.target.value)
                    setAccepted(false)
                  }}
                />
              </label>
              {cartLoading && <SkeletonStatus label="Đang kiểm tra giỏ hàng" className="space-y-2"><SkeletonBlock className="h-4 w-2/3" /><SkeletonBlock className="h-4 w-1/2" /></SkeletonStatus>}
              {cartError && (
                <div role="alert" className="space-y-2 text-error">
                  <p>{cartError}</p>
                  <Link to="/gio-hang" className="underline">
                    Kiểm tra giỏ hàng
                  </Link>
                </div>
              )}
              {!cartError && cart.issues.length > 0 && (
                <div className="space-y-2 text-on-surface-variant border border-outline-variant p-4">
                  <p>Các dòng sau cần kiểm tra lại và không được đưa vào đơn này:</p>
                  {cart.issues.map((item) => (
                    <p key={`${item.product_id}:${item.sku}`}>
                      {item.name || 'Sản phẩm trong giỏ'}: {item.issue_detail || 'Giá hoặc tồn kho đã thay đổi.'}
                    </p>
                  ))}
                  <Link to="/gio-hang" className="underline">
                    Kiểm tra giỏ hàng
                  </Link>
                </div>
              )}
              {!items.length && !cartLoading && (
                <p>
                  {cart.items.length ? 'Bạn chưa chọn món nào để đặt mua.' : 'Giỏ hàng chưa có áo sẵn sàng đặt.'}{' '}
                  <Link className="underline" to="/gio-hang">
                    Xem giỏ hàng
                  </Link>
                </p>
              )}
              <button type="submit" className={buttonClass} disabled={blocked || !items.length || !!cartError}>
                {busy ? 'Đang tính tổng tiền...' : 'Tính tổng tiền'}
              </button>
            </fieldset>
          </form>
          <aside className="bg-surface-container-low p-6 sm:p-8 space-y-5 lg:sticky lg:top-32">
            <h2 className="font-headline-md text-2xl text-primary">Kiểm tra đơn hàng</h2>
            {quote ? (
              <>
                <ul className="divide-y divide-outline-variant">
                  {quote.items.map((item) => (
                    <li className="py-3 space-y-1" key={`${item.product_id}:${item.sku}`}>
                      <p className="font-semibold">
                        {cartLineOf(item)?.name || 'Sản phẩm áo dài'}
                      </p>
                      <p className="text-sm text-on-surface-variant">
                        {[cartLineOf(item)?.color, cartLineOf(item)?.size].filter(Boolean).join(' · ') || 'Áo dài'} · {item.quantity} áo · Đơn giá {formatOrderMoney(item.unit_price)}
                      </p>
                    </li>
                  ))}
                </ul>
                <dl className="space-y-3 border-t border-outline-variant pt-4">
                  <div className="flex justify-between gap-4">
                    <dt>Tiền hàng</dt>
                    <dd>{formatOrderMoney(quote.subtotal)}</dd>
                  </div>
                  <div className="flex justify-between gap-4">
                    <dt>Phí giao hàng</dt>
                    <dd>{formatOrderMoney(quote.shipping_fee)}</dd>
                  </div>
                  <div className="flex justify-between gap-4 text-xl text-primary font-semibold">
                    <dt>Tổng tiền</dt>
                    <dd>{formatOrderMoney(quote.total)}</dd>
                  </div>
                </dl>
                {!quoteCurrent && (
                  <p role="alert" className="text-error">
                    {expired
                      ? 'Tổng tiền đã hết hạn. Vui lòng bấm “Tính tổng tiền” lần nữa và xác nhận lại.'
                      : 'Thông tin hoặc giỏ hàng đã thay đổi. Vui lòng bấm “Tính tổng tiền” lần nữa.'}
                  </p>
                )}
                {!!changes.length && (
                  <div role="alert" className="border border-secondary p-3 space-y-2">
                    <p className="font-semibold">Vui lòng kiểm tra lại thay đổi</p>
                    {changes.map((change, index) => (
                      <p key={index}>{change}</p>
                    ))}
                  </div>
                )}
                <label className="flex gap-3 items-start">
                  <input
                    type="checkbox"
                    checked={accepted && quoteCurrent}
                    disabled={!quoteCurrent || busy}
                    onChange={(event) => setAccepted(event.target.checked)}
                  />
                  <span>Tôi đã kiểm tra thông tin, các dòng áo và tổng tiền mới nhất.</span>
                </label>
                <button
                  type="button"
                  className={`${buttonClass} w-full`}
                  disabled={blocked || !quoteCurrent || !accepted || !!cartError}
                  onClick={() => void sendOrder()}
                >
                  {busy ? 'Đang gửi đơn...' : 'Xác nhận đặt mua'}
                </button>
              </>
            ) : (
              <>
                <ul className="divide-y divide-outline-variant">
                  {items.map((item) => (
                    <li className="py-3 flex justify-between gap-4" key={`${item.product_id}:${item.sku}`}>
                      <div className="min-w-0">
                        <p className="font-semibold">{item.name || 'Sản phẩm áo dài'}</p>
                        <p className="text-sm text-on-surface-variant">
                          {[item.color, item.size].filter(Boolean).join(' · ')}{item.color || item.size ? ' · ' : ''}{item.quantity} áo
                        </p>
                      </div>
                      <span>{formatOrderMoney(String(item.line_total))}</span>
                    </li>
                  ))}
                </ul>
                <div className="flex justify-between gap-4 border-t border-outline-variant pt-4">
                  <span>Tạm tính</span>
                  <strong className="text-primary">{formatOrderMoney(String(selectedSubtotal))}</strong>
                </div>
                {items.length > 0 && items.length < cart.items.length && (
                  <p className="text-sm text-on-surface-variant">
                    Đặt {items.length}/{cart.items.length} dòng trong giỏ; các món còn lại vẫn được giữ.{' '}
                    <Link to="/gio-hang" className="underline">
                      Đổi lựa chọn
                    </Link>
                  </p>
                )}
                <p>Phí giao và tổng tiền được chốt ở bước “Tính tổng tiền”.</p>
              </>
            )}
            <p className="text-sm text-on-surface-variant">Đơn mới chưa được thanh toán. Cửa hàng chỉ ghi nhận đã thu tiền khi thực tế nhận đủ tiền.</p>
          </aside>
        </div>
      )}
    </main>
  )
}
