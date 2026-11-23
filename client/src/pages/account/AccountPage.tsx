import { useEffect, useState, type FormEvent, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import Breadcrumb from '@/components/ui/Breadcrumb'
import Icon from '@/components/ui/Icon'
import { accountApi, type Address, type AddressInput, type Measurements, type Profile } from '@/features/account/api'
import { useAuth } from '@/features/auth/useAuth'
import { ROUTES } from '@/data/site'

type Section = 'profile' | 'addresses' | 'measurements'
const inputClass = 'w-full bg-surface-container px-4 py-3 border border-outline-variant/60 focus:border-primary focus:outline-none'
const emptyAddress: AddressInput = { recipient: '', phone: '', line1: '', ward: '', district: '', city: '', is_default: false }
const measurementFields: { key: keyof Measurements; label: string }[] = [
  { key: 'neck_cm', label: 'Vòng cổ' }, { key: 'bust_cm', label: 'Vòng ngực' }, { key: 'waist_cm', label: 'Vòng eo' },
  { key: 'hips_cm', label: 'Vòng mông' }, { key: 'shoulder_cm', label: 'Ngang vai' }, { key: 'sleeve_cm', label: 'Dài tay' },
  { key: 'garment_length_cm', label: 'Dài áo' },
]

export default function AccountPage() {
  const [section, setSection] = useState<Section>('profile')
  const [profile, setProfile] = useState<Profile | null>(null)
  const [profileForm, setProfileForm] = useState({ full_name: '', phone: '' })
  const [addressForm, setAddressForm] = useState<AddressInput>(emptyAddress)
  const [editingAddress, setEditingAddress] = useState<string | null>(null)
  const [measurements, setMeasurements] = useState<Measurements>({})
  const [measurementNotes, setMeasurementNotes] = useState('')
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [reload, setReload] = useState(0)
  const { refresh: refreshAuth } = useAuth()

  useEffect(() => {
    let active = true
    accountApi.profile()
      .then((next) => {
        if (!active) return
        setProfile(next)
        setProfileForm({ full_name: next.full_name, phone: next.phone || '' })
        setMeasurements(next.measurement_profile || {})
        setMeasurementNotes(next.measurement_notes || '')
      })
      .catch((err: unknown) => { if (active) setError(err instanceof Error ? err.message : 'Không thể tải tài khoản') })
      .finally(() => { if (active) setLoading(false) })
    return () => { active = false }
  }, [reload])

  function notify(message: string) { setSuccess(message); setError('') }

  async function saveProfile(event: FormEvent) {
    event.preventDefault(); setBusy(true); setError(''); setSuccess('')
    try { setProfile(await accountApi.updateProfile(profileForm)); await refreshAuth(); notify('Đã lưu thông tin cá nhân.') }
    catch (err) { setError(err instanceof Error ? err.message : 'Không thể lưu thông tin') }
    finally { setBusy(false) }
  }

  function editAddress(address: Address) {
    setEditingAddress(address.id)
    setAddressForm({ recipient: address.recipient, phone: address.phone, line1: address.line1, ward: address.ward || '', district: address.district || '', city: address.city, is_default: address.is_default })
  }

  async function saveAddress(event: FormEvent) {
    event.preventDefault(); setBusy(true); setError(''); setSuccess('')
    try {
      const next = editingAddress ? await accountApi.updateAddress(editingAddress, addressForm) : await accountApi.createAddress(addressForm)
      setProfile(next); setAddressForm(emptyAddress); setEditingAddress(null); notify(editingAddress ? 'Đã cập nhật địa chỉ.' : 'Đã thêm địa chỉ.')
    } catch (err) { setError(err instanceof Error ? err.message : 'Không thể lưu địa chỉ') }
    finally { setBusy(false) }
  }

  async function deleteAddress(id: string) {
    if (!window.confirm('Xóa địa chỉ này?')) return
    setBusy(true); setError(''); setSuccess('')
    try { setProfile(await accountApi.deleteAddress(id)); notify('Đã xóa địa chỉ.') }
    catch (err) { setError(err instanceof Error ? err.message : 'Không thể xóa địa chỉ') }
    finally { setBusy(false) }
  }

  async function selectAddress(address: Address) {
    if (address.is_default || busy) return
    setBusy(true); setError(''); setSuccess('')
    try {
      const next = await accountApi.updateAddress(address.id, {
        recipient: address.recipient, phone: address.phone, line1: address.line1,
        ward: address.ward || '', district: address.district || '', city: address.city, is_default: true,
      })
      setProfile(next)
      notify('Đã chọn địa chỉ mặc định.')
    } catch (err) { setError(err instanceof Error ? err.message : 'Không thể chọn địa chỉ mặc định') }
    finally { setBusy(false) }
  }

  async function saveMeasurements(event: FormEvent) {
    event.preventDefault(); setBusy(true); setError(''); setSuccess('')
    try {
      const profileData = Object.fromEntries(Object.entries(measurements).filter(([, value]) => value !== undefined && value !== null && value !== '')) as Measurements
      const next = await accountApi.updateMeasurements({ measurement_profile: Object.keys(profileData).length ? profileData : null, measurement_notes: measurementNotes })
      setMeasurements(next.measurement_profile || {}); setMeasurementNotes(next.measurement_notes || ''); notify('Đã lưu hồ sơ số đo.')
    } catch (err) { setError(err instanceof Error ? err.message : 'Không thể lưu số đo') }
    finally { setBusy(false) }
  }

  if (loading) return <main className="max-w-[1440px] mx-auto px-margin-desktop py-20"><p role="status">Đang tải tài khoản...</p></main>
  if (!profile && error) return <main className="max-w-lg mx-auto px-5 py-20 text-center space-y-4"><p role="alert" className="text-red-700">{error}</p><button type="button" className="bg-primary text-on-primary px-5 py-3" onClick={() => { setError(''); setLoading(true); setReload((value) => value + 1) }}>Thử tải lại</button></main>
  return <main className="bg-surface min-h-screen">
    <div className="max-w-[1440px] mx-auto px-margin md:px-margin-tablet lg:px-margin-desktop py-4"><Breadcrumb items={[{ label: 'Trang chủ', to: ROUTES.home }, { label: 'Tài khoản' }]} /></div>
    <section className="bg-surface-container-lowest py-10 lg:py-16">
      <div className="max-w-[1440px] mx-auto px-margin md:px-margin-tablet lg:px-margin-desktop">
        <p className="font-label-uppercase text-label-uppercase text-secondary tracking-widest">Không gian riêng</p>
        <h1 className="font-headline-lg text-headline-lg text-primary mt-2">Tài Khoản Của Bạn</h1>
        <p className="text-on-surface-variant mt-3 max-w-2xl">Lưu lại thông tin vừa vặn, địa chỉ nhận hàng và những dấu ấn riêng cho hành trình áo dài của bạn.</p>
      </div>
    </section>
    <div className="max-w-[1440px] mx-auto px-margin md:px-margin-tablet lg:px-margin-desktop py-10 lg:py-16 grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
      <aside className="lg:col-span-3 bg-surface-container-low p-3">
        {([['profile', 'Thông tin cá nhân', 'person'], ['addresses', 'Địa chỉ nhận hàng', 'location_on'], ['measurements', 'Hồ sơ số đo', 'straighten']] as const).map(([key, label, icon]) => <button key={key} type="button" onClick={() => { setSection(key); setError(''); setSuccess('') }} className={`w-full flex items-center gap-3 text-left px-4 py-3 ${section === key ? 'bg-primary text-on-primary' : 'text-on-surface-variant hover:bg-surface-container'}`}><Icon name={icon} size={19} />{label}</button>)}
        <Link to="/tai-khoan/don-hang" className="w-full flex items-center gap-3 text-left px-4 py-3 text-on-surface-variant hover:bg-surface-container"><Icon name="receipt_long" size={19} />Đơn hàng của tôi</Link>
        <Link to={ROUTES.wishlist} className="w-full flex items-center gap-3 text-left px-4 py-3 text-on-surface-variant hover:bg-surface-container"><Icon name="favorite" size={19} />Danh sách yêu thích</Link>
      </aside>
      <section className="lg:col-span-9 bg-surface-container-lowest p-6 sm:p-8">
        {section === 'profile' && <form onSubmit={(event) => void saveProfile(event)} className="space-y-6"><SectionTitle eyebrow="Thông tin tài khoản" title="Thông tin cá nhân" description="Email đăng nhập được giữ nguyên để bảo vệ tài khoản." /><div className="grid sm:grid-cols-2 gap-5"><Field label="Họ và tên"><input className={inputClass} required value={profileForm.full_name} onChange={(event) => setProfileForm({ ...profileForm, full_name: event.target.value })} /></Field><Field label="Email"><input className={`${inputClass} opacity-70`} readOnly value={profile?.email || ''} /></Field></div><Field label="Số điện thoại"><input className={inputClass} type="tel" value={profileForm.phone} onChange={(event) => setProfileForm({ ...profileForm, phone: event.target.value })} /></Field><SaveButton busy={busy} label="Lưu thay đổi" /></form>}
        {section === 'addresses' && <div className="space-y-8"><SectionTitle eyebrow="Thông tin nhận hàng" title="Địa chỉ nhận hàng" description="Chọn trực tiếp vào card để đặt địa chỉ mặc định cho checkout." /><div className="grid md:grid-cols-2 gap-4">{(profile?.addresses || []).map((address) => <article key={address.id} role="button" tabIndex={address.is_default ? -1 : 0} aria-pressed={address.is_default} onClick={() => void selectAddress(address)} onKeyDown={(event) => { if ((event.key === 'Enter' || event.key === ' ') && !address.is_default) { event.preventDefault(); void selectAddress(address) } }} className={`border p-5 bg-surface cursor-pointer transition-colors focus:outline-none focus:ring-2 focus:ring-primary ${address.is_default ? 'border-primary' : 'border-outline-variant/60 hover:border-primary/60'}`}><div className="flex items-start justify-between gap-3"><div><h3 className="font-headline-sm text-primary">{address.recipient}</h3><p className="text-sm text-on-surface-variant mt-1">{address.phone}</p></div>{address.is_default && <span className="bg-secondary-fixed px-2 py-1 text-xs text-on-secondary-fixed">Mặc định</span>}</div><p className="mt-4 text-sm leading-6">{[address.line1, address.ward, address.district, address.city].filter(Boolean).join(', ')}</p><div className="flex gap-4 mt-5 text-sm"><button type="button" className="underline text-primary" onClick={(event) => { event.stopPropagation(); editAddress(address) }}>Sửa</button><button type="button" className="underline text-red-700" onClick={(event) => { event.stopPropagation(); void deleteAddress(address.id) }}>Xóa</button></div></article>)}</div><form onSubmit={(event) => void saveAddress(event)} className="border-t border-outline-variant/60 pt-8 space-y-5"><h2 className="font-headline-md text-primary">{editingAddress ? 'Chỉnh sửa địa chỉ' : 'Thêm địa chỉ mới'}</h2><div className="grid sm:grid-cols-2 gap-5"><Field label="Người nhận"><input className={inputClass} required value={addressForm.recipient} onChange={(event) => setAddressForm({ ...addressForm, recipient: event.target.value })} /></Field><Field label="Số điện thoại"><input className={inputClass} required value={addressForm.phone} onChange={(event) => setAddressForm({ ...addressForm, phone: event.target.value })} /></Field></div><Field label="Địa chỉ"><input className={inputClass} required value={addressForm.line1} onChange={(event) => setAddressForm({ ...addressForm, line1: event.target.value })} /></Field><div className="grid sm:grid-cols-3 gap-5"><Field label="Phường/Xã"><input className={inputClass} value={addressForm.ward} onChange={(event) => setAddressForm({ ...addressForm, ward: event.target.value })} /></Field><Field label="Quận/Huyện"><input className={inputClass} value={addressForm.district} onChange={(event) => setAddressForm({ ...addressForm, district: event.target.value })} /></Field><Field label="Tỉnh/Thành phố"><input className={inputClass} required value={addressForm.city} onChange={(event) => setAddressForm({ ...addressForm, city: event.target.value })} /></Field></div><label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={addressForm.is_default} onChange={(event) => setAddressForm({ ...addressForm, is_default: event.target.checked })} />Đặt làm địa chỉ mặc định</label><div className="flex gap-3"><SaveButton busy={busy} label={editingAddress ? 'Cập nhật địa chỉ' : 'Thêm địa chỉ'} />{editingAddress && <button type="button" className="border border-outline-variant px-5 py-3" onClick={() => { setEditingAddress(null); setAddressForm(emptyAddress) }}>Hủy</button>}</div></form></div>}
        {section === 'measurements' && <form onSubmit={(event) => void saveMeasurements(event)} className="space-y-6"><SectionTitle eyebrow="Vừa vặn theo cách riêng" title="Hồ sơ số đo" description="Các số đo được lưu riêng cho tài khoản và không thay đổi dữ liệu của những đơn hàng cũ." /><Field label="Mô tả vóc dáng"><textarea className={inputClass} rows={3} value={measurements.body || ''} onChange={(event) => setMeasurements({ ...measurements, body: event.target.value })} placeholder="Ví dụ: Vai xuôi, lưng hơi cong nhẹ..." /></Field><div className="grid grid-cols-2 sm:grid-cols-3 gap-5">{measurementFields.map(({ key, label }) => <Field key={key} label={`${label} (cm)`}><div className="relative"><input className={`${inputClass} pr-12`} type="number" min="0" step="0.1" value={measurements[key] ?? ''} onChange={(event) => setMeasurements({ ...measurements, [key]: event.target.value === '' ? undefined : Number(event.target.value) })} /><span className="absolute right-3 top-1/2 -translate-y-1/2 text-sm text-on-surface-variant">cm</span></div></Field>)}</div><Field label="Ghi chú"><textarea className={inputClass} rows={4} maxLength={2000} value={measurementNotes} onChange={(event) => setMeasurementNotes(event.target.value)} placeholder="Yêu cầu riêng về phom dáng hoặc tà áo..." /></Field><SaveButton busy={busy} label="Lưu hồ sơ số đo" /></form>}
        {success && <p role="status" className="mt-6 text-green-700">{success}</p>}{error && <p role="alert" className="mt-6 text-red-700">{error}</p>}
      </section>
    </div>
  </main>
}

function SectionTitle({ eyebrow, title, description }: { eyebrow: string; title: string; description: string }) { return <div><p className="font-label-uppercase text-label-uppercase text-secondary tracking-widest">{eyebrow}</p><h2 className="font-headline-md text-primary mt-2">{title}</h2><p className="text-on-surface-variant mt-2">{description}</p></div> }
function Field({ label, children }: { label: string; children: ReactNode }) { return <label className="block space-y-1.5"><span className="text-sm text-on-surface-variant">{label}</span>{children}</label> }
function SaveButton({ busy, label }: { busy: boolean; label: string }) { return <button disabled={busy} className="bg-primary text-on-primary px-6 py-3 disabled:opacity-50" type="submit">{busy ? 'Đang lưu...' : label}</button> }
