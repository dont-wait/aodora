import { useCallback, useEffect, useState } from 'react'
import type { Province } from './divisions'

let cache: Promise<Province[]> | null = null

/** Loads the static division list once, as its own chunk (not part of the main bundle). */
export function loadDivisions(): Promise<Province[]> {
  if (!cache) {
    cache = import('@/data/vn-divisions-2025.json').then((module) => module.default.provinces as Province[])
    cache.catch(() => { cache = null })
  }
  return cache
}

export function useDivisions() {
  const [provinces, setProvinces] = useState<Province[] | null>(null)
  const [error, setError] = useState(false)
  const [attempt, setAttempt] = useState(0)

  useEffect(() => {
    let active = true
    loadDivisions()
      .then((list) => { if (active) { setProvinces(list); setError(false) } })
      .catch(() => { if (active) setError(true) })
    return () => { active = false }
  }, [attempt])

  const retry = useCallback(() => { setError(false); setAttempt((n) => n + 1) }, [])
  return { provinces, loading: provinces === null && !error, error, retry }
}
