import { useCallback, useEffect, useMemo, useState } from 'react'
import { useAuth } from '@/features/auth/useAuth'
import { startOrderRead } from './read'
import { loadOrderDetail, type CheckoutResult } from './result'

export function useOrderRead<T>(scope: string, enabled: boolean, load: () => Promise<T>): T | null {
  const request = useMemo(() => ({ scope, enabled }), [scope, enabled])
  const [loaded, setLoaded] = useState<{ request: typeof request; result: T } | null>(null)
  useEffect(() => {
    if (!enabled) return
    return startOrderRead(load, (result) => setLoaded({ request, result }))
  }, [request, enabled, load])
  return enabled && loaded?.request === request ? loaded.result : null
}

export function useOrderDetail(id: string) {
  const { user, loading, error, refresh } = useAuth()
  const [revision, setRevision] = useState(0)
  const customerId = user?.id ?? null
  const scope = JSON.stringify([id, customerId, revision])
  const load = useCallback(() => loadOrderDetail(id, customerId), [id, customerId])
  const response = useOrderRead(scope, !loading && !error, load)
  const retry = () => {
    setRevision((value) => value + 1)
    if (error) void refresh()
  }
  const result: CheckoutResult | null = loading ? null : error ? { kind: 'error' } : response
  return { result, retry, isAccount: customerId !== null }
}
