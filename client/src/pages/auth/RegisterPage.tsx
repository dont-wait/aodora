import { useState, type FormEvent } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { AuthApiError } from '@/features/auth/api'
import { useAuth } from '@/features/auth/useAuth'
import { safeNextPath } from '@/features/auth/redirect'
import { ROUTES } from '@/data/site'
import Icon from '@/components/ui/Icon'

export default function RegisterPage() {
  const { register, login } = useAuth()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const [form, setForm] = useState({ full_name: '', email: '', phone: '', password: '', confirm: '' })
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [showPassword, setShowPassword] = useState(false)
  const [showConfirm, setShowConfirm] = useState(false)
  const next = safeNextPath(params.get('next'))

  async function submit(event: FormEvent) {
    event.preventDefault(); setError('')
    if (form.password !== form.confirm) { setError('Mật khẩu nhập lại không khớp.'); return }
    setBusy(true)
    try {
      await register({ full_name: form.full_name, email: form.email, phone: form.phone, password: form.password })
    } catch (err) {
      setError(err instanceof AuthApiError || err instanceof Error ? err.message : 'Đăng ký thất bại')
      setBusy(false)
      return
    }
    try {
      await login(form.email, form.password)
      navigate(next, { replace: true })
    } catch {
      // Account exists but auto sign-in failed: hand over to the login page with a notice and the email.
      const query = new URLSearchParams({ registered: '1', email: form.email })
      if (next !== ROUTES.home) query.set('next', next)
      navigate(`/dang-nhap?${query.toString()}`, { replace: true })
    } finally { setBusy(false) }
  }

  return <main className="max-w-lg mx-auto px-5 py-16">
    <div className="bg-surface p-8 shadow-sm space-y-6">
      <div><p className="uppercase tracking-widest text-secondary text-sm">Tài khoản AODORA</p><h1 className="font-headline-lg text-headline-lg text-primary mt-2">Tạo tài khoản</h1></div>
      <form className="space-y-4" onSubmit={(event) => void submit(event)}>
        <label className="block space-y-1"><span>Họ và tên</span><input required autoComplete="name" value={form.full_name} onChange={(event) => setForm({ ...form, full_name: event.target.value })} className="w-full bg-surface-container p-3" /></label>
        <label className="block space-y-1"><span>Email</span><input required type="email" autoComplete="email" value={form.email} onChange={(event) => setForm({ ...form, email: event.target.value })} className="w-full bg-surface-container p-3" /></label>
        <label className="block space-y-1"><span>Số điện thoại <small>(không bắt buộc)</small></span><input type="tel" autoComplete="tel" value={form.phone} onChange={(event) => setForm({ ...form, phone: event.target.value })} className="w-full bg-surface-container p-3" /></label>
        <label className="block space-y-1"><span>Mật khẩu</span><span className="relative block"><input required minLength={8} type={showPassword ? 'text' : 'password'} autoComplete="new-password" value={form.password} onChange={(event) => setForm({ ...form, password: event.target.value })} className="w-full bg-surface-container p-3 pr-12" /><button type="button" aria-label={showPassword ? 'Ẩn mật khẩu' : 'Hiện mật khẩu'} onClick={() => setShowPassword((value) => !value)} className="absolute right-0 top-1/2 -translate-y-1/2 inline-flex h-10 w-11 items-center justify-center leading-none text-on-surface-variant hover:text-primary"><Icon name={showPassword ? 'visibility_off' : 'visibility'} size={20} /></button></span></label>
        <label className="block space-y-1"><span>Nhập lại mật khẩu</span><span className="relative block"><input required minLength={8} type={showConfirm ? 'text' : 'password'} autoComplete="new-password" value={form.confirm} onChange={(event) => setForm({ ...form, confirm: event.target.value })} className="w-full bg-surface-container p-3 pr-12" /><button type="button" aria-label={showConfirm ? 'Ẩn mật khẩu xác nhận' : 'Hiện mật khẩu xác nhận'} onClick={() => setShowConfirm((value) => !value)} className="absolute right-0 top-1/2 -translate-y-1/2 inline-flex h-10 w-11 items-center justify-center leading-none text-on-surface-variant hover:text-primary"><Icon name={showConfirm ? 'visibility_off' : 'visibility'} size={20} /></button></span></label>
        {error && <p role="alert" className="text-red-700">{error}</p>}
        <button disabled={busy} className="w-full bg-primary text-on-primary py-3 disabled:opacity-50" type="submit">{busy ? 'Đang tạo tài khoản...' : 'Đăng ký'}</button>
      </form>
      <p className="text-on-surface-variant">Đã có tài khoản? <Link className="underline text-primary" to={`/dang-nhap${next !== ROUTES.home ? `?next=${encodeURIComponent(next)}` : ''}`}>Đăng nhập</Link></p>
    </div>
  </main>
}
