import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { addSelectedProduct, registerCartAddHandler, type CartSelection } from '@/features/catalog/selection'
import { useAuth } from '@/features/auth/useAuth'
import { cartApi, type CartData } from './api'
import { CartContext } from './context'

const STORAGE_KEY = 'aodora.cart.v1'
const emptyCart: CartData = { items: [], issues: [], item_count: 0, subtotal: 0 }
const normalizeCart = (cart: CartData): CartData => ({ ...emptyCart, ...cart, items: cart.items || [], issues: cart.issues || [] })

function readGuestSelections(): CartSelection[] {
  try {
    const raw = JSON.parse(localStorage.getItem(STORAGE_KEY) || '[]') as CartSelection[]
    return Array.isArray(raw) ? raw.filter((item) => item && typeof item.product_id === 'string' && typeof item.sku === 'string' && Number.isSafeInteger(item.quantity) && item.quantity > 0) : []
  } catch { return [] }
}

function writeGuestSelections(items: CartSelection[]) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(items))
}

function mergeLocal(items: CartSelection[], selection: CartSelection): CartSelection[] {
  const copy = items.map((item) => ({ ...item }))
  const existing = copy.find((item) => item.product_id === selection.product_id && item.sku === selection.sku)
  if (existing) existing.quantity += selection.quantity
  else copy.push({ ...selection })
  return copy
}

function changeLocal(items: CartSelection[], selection: CartSelection): CartSelection[] {
  return items.map((item) => item.product_id === selection.product_id && item.sku === selection.sku ? { ...item, quantity: selection.quantity } : item)
}

function removeLocal(items: CartSelection[], productId: string, sku: string) {
  return items.filter((item) => item.product_id !== productId || item.sku !== sku)
}

export function CartProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const [guestSelections, setGuestSelections] = useState<CartSelection[]>(readGuestSelections)
  const [guestCart, setGuestCart] = useState<CartData>(emptyCart)
  const [accountCart, setAccountCart] = useState<CartData>(emptyCart)
  const [loading, setLoading] = useState(() => readGuestSelections().length > 0)
  const [error, setError] = useState('')

  const refreshGuest = useCallback(async (items = guestSelections) => {
    if (!items.length) { setGuestCart(emptyCart); setLoading(false); return }
    setLoading(true)
    try { setGuestCart(normalizeCart(await cartApi.validate(items))); setError('') }
    catch (err) { setError(err instanceof Error ? err.message : 'Không thể kiểm tra giỏ hàng') }
    finally { setLoading(false) }
  }, [guestSelections])

  const refreshAccount = useCallback(async () => {
    if (!user) return
    setLoading(true)
    try { setAccountCart(normalizeCart(await cartApi.get())); setError('') }
    catch (err) { setError(err instanceof Error ? err.message : 'Không thể tải giỏ hàng') }
    finally { setLoading(false) }
  }, [user])

  useEffect(() => {
    if (!user) return
    let active = true
    const local = readGuestSelections()
    async function load() {
      setLoading(true)
      try {
        let next: CartData
        if (local.length) {
          const checked = await cartApi.validate(local)
          const issueKeys = new Set((checked.issues || []).map((item) => `${item.product_id}:${item.sku}`))
          const valid = local.filter((item) => !issueKeys.has(`${item.product_id}:${item.sku}`))
          next = valid.length ? await cartApi.merge(valid, crypto.randomUUID()) : await cartApi.get()
          next = { ...next, issues: checked.issues || [] }
          const remaining = local.filter((item) => issueKeys.has(`${item.product_id}:${item.sku}`))
          if (remaining.length) { localStorage.setItem(STORAGE_KEY, JSON.stringify(remaining)); setGuestSelections(remaining) }
          else localStorage.removeItem(STORAGE_KEY)
        } else next = await cartApi.get()
        if (!active) return
        setAccountCart(normalizeCart(next))
        if (local.length && !(next.issues || []).length) setGuestSelections([])
        setError('')
      } catch (err) {
        if (active) setError(err instanceof Error ? err.message : 'Không thể đồng bộ giỏ hàng')
      } finally { if (active) setLoading(false) }
    }
    void load()
    return () => { active = false }
	}, [user])

	useEffect(() => {
		if (user || !guestSelections.length) return
		let active = true
		cartApi.validate(guestSelections)
			.then((next) => { if (active) { setGuestCart(normalizeCart(next)); setError('') } })
			.catch((err: unknown) => { if (active) setError(err instanceof Error ? err.message : 'Không thể kiểm tra giỏ hàng') })
			.finally(() => { if (active) setLoading(false) })
		return () => { active = false }
	}, [user, guestSelections])

  useEffect(() => registerCartAddHandler(async (selection) => {
    if (user) setAccountCart(normalizeCart(await cartApi.add(selection)))
    else {
      const next = mergeLocal(guestSelections, selection)
      setGuestSelections(next); writeGuestSelections(next)
    }
  }), [user, guestSelections])

  const add = useCallback(async (selection: CartSelection) => { await addSelectedProduct(selection) }, [])

  const update = useCallback(async (selection: CartSelection) => {
    try {
      if (user) setAccountCart(normalizeCart(await cartApi.update(selection)))
      else {
        const next = changeLocal(guestSelections, selection)
        setGuestSelections(next); writeGuestSelections(next)
      }
    } catch (err) { setError(err instanceof Error ? err.message : 'Không thể cập nhật giỏ hàng') }
  }, [user, guestSelections])

  const remove = useCallback(async (productId: string, sku: string) => {
    const guestItem = guestSelections.some((item) => item.product_id === productId && item.sku === sku)
    try {
      if (user && guestItem) {
        const next = removeLocal(guestSelections, productId, sku)
        setGuestSelections(next); writeGuestSelections(next)
        setAccountCart((current) => ({ ...current, issues: current.issues.filter((item) => item.product_id !== productId || item.sku !== sku) }))
      } else if (user) setAccountCart(normalizeCart(await cartApi.remove(productId, sku)))
      else {
        const next = removeLocal(guestSelections, productId, sku)
        setGuestSelections(next); writeGuestSelections(next)
      }
    } catch (err) { setError(err instanceof Error ? err.message : 'Không thể xóa sản phẩm khỏi giỏ') }
  }, [user, guestSelections])

  const cart = user ? accountCart : guestCart
  const itemCount = cart.items.reduce((sum, item) => sum + item.quantity, 0)
  const value = useMemo(() => ({ cart, loading, error, itemCount, add, update, remove, refresh: user ? refreshAccount : () => refreshGuest() }), [cart, loading, error, itemCount, add, update, remove, user, refreshAccount, refreshGuest])
  return <CartContext.Provider value={value}>{children}</CartContext.Provider>
}
