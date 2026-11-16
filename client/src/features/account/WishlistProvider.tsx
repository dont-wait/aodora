import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { AccountApiError, accountApi, type WishlistItem } from './api'
import { WishlistContext } from './context'
import { useAuth } from '@/features/auth/useAuth'

export function WishlistProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const [items, setItems] = useState<WishlistItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const refresh = useCallback(async () => {
    if (!user) return
    setLoading(true)
    try {
      setItems((await accountApi.wishlist()).items || [])
      setError('')
    } catch (err) {
      if (err instanceof AccountApiError && err.status === 401) setItems([])
      else setError(err instanceof Error ? err.message : 'Không thể tải danh sách yêu thích')
    } finally { setLoading(false) }
  }, [user])

  useEffect(() => {
    if (!user) return
    let active = true
    accountApi.wishlist()
      .then((result) => { if (active) { setItems(result.items || []); setError('') } })
      .catch((err: unknown) => {
        if (!active) return
        if (err instanceof AccountApiError && err.status === 401) setItems([])
        else setError(err instanceof Error ? err.message : 'Không thể tải danh sách yêu thích')
      })
      .finally(() => { if (active) setLoading(false) })
    return () => { active = false }
  }, [user])

  const isSaved = useCallback((productId: string) => items.some((item) => item.id === productId), [items])

  const toggle = useCallback(async (productId: string) => {
    if (isSaved(productId)) await accountApi.removeWishlist(productId)
    else await accountApi.addWishlist(productId)
    await refresh()
  }, [isSaved, refresh])

  const value = useMemo(() => ({ items, loading, error, refresh, isSaved, toggle }), [items, loading, error, refresh, isSaved, toggle])
  return <WishlistContext.Provider value={value}>{children}</WishlistContext.Provider>
}
