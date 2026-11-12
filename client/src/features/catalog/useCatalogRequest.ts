import { useEffect, useState } from 'react'

interface State<T> { key: string; data: T | null; loading: boolean; error: string | null }
// A request is scoped to its filter/slug key. Old data is hidden immediately
// when navigating, even before the effect for the next request has run.
export function useCatalogRequest<T>(load: (signal: AbortSignal) => Promise<T>, key: string) {
  const [attempt, setAttempt] = useState(0)
  const [state, setState] = useState<State<T>>({ key, data: null, loading: true, error: null })
  useEffect(() => {
    const controller = new AbortController()
    let active = true
    setState({ key, data: null, loading: true, error: null })
    load(controller.signal).then(
      (data) => { if (active) setState({ key, data, loading: false, error: null }) },
      (error: unknown) => {
        if (!active) return
        setState({ key, data: null, loading: false, error: error instanceof Error ? error.message : 'Không thể tải dữ liệu.' })
      },
    )
    return () => { active = false; controller.abort() }
    // Key is the canonical identity of the request; load is built from it.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, attempt])
  const current = state.key === key ? state : { key, data: null, loading: true, error: null }
  return { data: current.data, loading: current.loading, error: current.error, retry: () => setAttempt((n) => n + 1) }
}
