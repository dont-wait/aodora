import { useState, type FormEvent } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { AuthApiError } from '@/features/auth/api'
import { useAuth } from '@/features/auth/useAuth'
import { safeNextPath } from '@/features/auth/redirect'
import { ROUTES } from '@/data/site'
import Icon from '@/components/ui/Icon'

export default function LoginPage() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
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

  return <main className="max-w-[1180px] mx-auto px-5 py-10 lg:py-16">
    <div className="grid grid-cols-1 lg:grid-cols-2 bg-surface shadow-sm overflow-hidden">
      <div className="relative min-h-[320px] lg:min-h-[650px]">
        <img
          src="https://cdn2.tuoitre.vn/nld/291774122806476800/2025/4/27/dsc04640-17457343585081493284828.jpg"
          alt="Áo dài Việt Nam trong không gian di sản"
          className="absolute inset-0 w-full h-full object-cover"
        />
        <div className="absolute inset-0 bg-gradient-to-t from-primary/80 via-primary/10 to-transparent" />
        <div className="absolute bottom-0 left-0 right-0 p-8 text-on-primary">
          <p className="uppercase tracking-[0.25em] text-xs text-secondary-fixed">AODORA · Áo Dài Di Sản Việt</p>
          <p className="font-title-editorial text-3xl mt-3">Tinh hoa Việt trong từng nếp lụa.</p>
        </div>
      </div>
      <div className="p-8 sm:p-12 lg:p-16 flex items-center">
        <div className="w-full max-w-md mx-auto space-y-6">
          <div><p className="uppercase tracking-widest text-secondary text-sm">Tài khoản AODORA</p><h1 className="font-headline-lg text-headline-lg text-primary mt-2">Đăng nhập</h1><p className="text-on-surface-variant mt-3">Tiếp tục hành trình khám phá những thiết kế áo dài mang hồn Việt.</p></div>
          <form className="space-y-4" onSubmit={(event) => void submit(event)}>
            <label className="block space-y-1"><span>Email</span><input required type="email" autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} className="w-full bg-surface-container p-3" /></label>
            <label className="block space-y-1"><span>Mật khẩu</span><span className="relative block"><input required type={showPassword ? 'text' : 'password'} autoComplete="current-password" value={password} onChange={(event) => setPassword(event.target.value)} className="w-full bg-surface-container p-3 pr-12" /><button type="button" aria-label={showPassword ? 'Ẩn mật khẩu' : 'Hiện mật khẩu'} onClick={() => setShowPassword((value) => !value)} className="absolute right-0 top-1/2 -translate-y-1/2 inline-flex h-10 w-11 items-center justify-center leading-none text-on-surface-variant hover:text-primary"><Icon name={showPassword ? 'visibility_off' : 'visibility'} size={20} /></button></span></label>
            {error && <p role="alert" className="text-red-700">{error}</p>}
            <button disabled={busy} className="w-full bg-primary text-on-primary py-3 disabled:opacity-50" type="submit">{busy ? 'Đang đăng nhập...' : 'Đăng nhập'}</button>
          </form>
          <p className="text-on-surface-variant">Chưa có tài khoản? <Link className="underline text-primary" to={`/dang-ky${next !== ROUTES.home ? `?next=${encodeURIComponent(next)}` : ''}`}>Đăng ký</Link></p>
        </div>
      </div>
    </div>
  </main>
}
