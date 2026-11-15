import { useState, type FormEvent } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { AuthApiError } from '@/features/auth/api'
import { useAuth } from '@/features/auth/useAuth'
import { safeNextPath } from '@/features/auth/redirect'
import { ROUTES } from '@/data/site'

export default function LoginPage() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const next = safeNextPath(params.get('next'))

  async function submit(event: FormEvent) {
    event.preventDefault()
    setBusy(true); setError('')
    try {
      await login(email, password)
      navigate(next, { replace: true })
    } catch (err) {
      setError(err instanceof AuthApiError || err instanceof Error ? err.message : 'Đăng nhập thất bại')
    } finally { setBusy(false) }
  }

  return <main className="max-w-lg mx-auto px-5 py-16">
    <div className="bg-surface p-8 shadow-sm space-y-6">
      <div><p className="uppercase tracking-widest text-secondary text-sm">Tài khoản AN SOIE</p><h1 className="font-headline-lg text-headline-lg text-primary mt-2">Đăng nhập</h1></div>
      <form className="space-y-4" onSubmit={(event) => void submit(event)}>
        <label className="block space-y-1"><span>Email</span><input required type="email" autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} className="w-full bg-surface-container p-3" /></label>
        <label className="block space-y-1"><span>Mật khẩu</span><input required type="password" autoComplete="current-password" value={password} onChange={(event) => setPassword(event.target.value)} className="w-full bg-surface-container p-3" /></label>
        {error && <p role="alert" className="text-red-700">{error}</p>}
        <button disabled={busy} className="w-full bg-primary text-on-primary py-3 disabled:opacity-50" type="submit">{busy ? 'Đang đăng nhập...' : 'Đăng nhập'}</button>
      </form>
      <p className="text-on-surface-variant">Chưa có tài khoản? <Link className="underline text-primary" to={`/dang-ky${next !== ROUTES.home ? `?next=${encodeURIComponent(next)}` : ''}`}>Đăng ký</Link></p>
    </div>
  </main>
}
