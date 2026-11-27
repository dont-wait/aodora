import { useEffect, useMemo, useState } from 'react'
import SearchSelect from '@/components/ui/SearchSelect'
import { matchSavedAddress } from './divisions'
import { useDivisions } from './useDivisions'

export interface AddressFieldsValue { city: string; ward: string }

interface Props {
  value: AddressFieldsValue
  onChange: (next: AddressFieldsValue) => void
  /** District of an address saved before 07/2025; makes the pair "legacy" so the ward must be re-chosen. */
  legacyDistrict?: string
  required?: boolean
  /** Defaults to `required`. */
  wardRequired?: boolean
  disabled?: boolean
  idPrefix: string
  errors?: { city?: string; ward?: string }
  className?: string
  selectClassName?: string
  labelClassName?: string
}

export default function AddressFields({
  value, onChange, legacyDistrict = '', required = false, wardRequired = required, disabled = false, idPrefix, errors = {},
  className = 'grid sm:grid-cols-2 gap-4', selectClassName = 'w-full bg-surface-container px-4 py-3 border border-outline-variant/60', labelClassName = 'space-y-2 block',
}: Props) {
  const { provinces, loading, error, retry } = useDivisions()
  const [legacy, setLegacy] = useState(false)
  const match = provinces ? matchSavedAddress(provinces, { city: value.city, ward: value.ward, district: legacyDistrict }) : null

  // Adjust state during render (not in an effect): remember that the incoming data was legacy until a valid ward is chosen.
  if (match?.legacy && !legacy) setLegacy(true)
  else if (match && !match.legacy && legacy && value.ward) setLegacy(false)

  // Bring stored values onto the canonical names; legacy data keeps only a matching province and drops the ward.
  const canonicalCity = match ? (match.province?.name ?? '') : value.city
  const canonicalWard = match ? (match.legacy ? '' : (match.ward?.name ?? '')) : value.ward
  const needsCanonical = match !== null && (value.city !== canonicalCity || value.ward !== canonicalWard)
  useEffect(() => {
    if (needsCanonical) onChange({ city: canonicalCity, ward: canonicalWard })
  }, [needsCanonical, canonicalCity, canonicalWard, onChange])

  const province = provinces?.find((item) => item.name === value.city)
  const provinceOptions = useMemo(() => (provinces ?? []).map((item) => ({ value: item.name, label: item.name })), [provinces])
  const wardOptions = useMemo(() => (province?.wards ?? []).map((item) => ({ value: item.name, label: item.name })), [province])
  const idCity = `${idPrefix}-city`
  const idWard = `${idPrefix}-ward`

  return (
    <div className="space-y-2">
      {error && (
        <p role="alert" className="text-red-700 text-sm">
          Không tải được danh mục tỉnh/phường.{' '}
          <button type="button" className="underline" onClick={retry}>Thử lại</button>
        </p>
      )}
      <div className={className}>
        <div className={labelClassName}>
          <label htmlFor={idCity}>Tỉnh / thành phố{required ? ' *' : ''}</label>
          <SearchSelect
            id={idCity}
            name={`${idPrefix}-city`}
            autoComplete="off"
            className={selectClassName}
            required={required}
            disabled={disabled || loading || error}
            options={provinceOptions}
            value={province ? province.name : ''}
            placeholder={loading ? 'Đang tải danh mục...' : 'Gõ để tìm tỉnh / thành phố'}
            aria-invalid={Boolean(errors.city)}
            onChange={(next) => { setLegacy(false); onChange({ city: next, ward: '' }) }}
          />
          {errors.city && <span role="alert" className="block text-red-700 text-sm">{errors.city}</span>}
        </div>
        <div className={labelClassName}>
          <label htmlFor={idWard}>Phường / xã{wardRequired ? ' *' : ''}</label>
          <SearchSelect
            key={province?.name ?? ''}
            id={idWard}
            name={`${idPrefix}-ward`}
            autoComplete="off"
            className={selectClassName}
            required={wardRequired}
            disabled={disabled || !province}
            options={wardOptions}
            value={province?.wards.some((item) => item.name === value.ward) ? value.ward : ''}
            placeholder={province ? 'Gõ để tìm phường / xã' : 'Chọn tỉnh / thành phố trước'}
            aria-invalid={Boolean(errors.ward)}
            onChange={(next) => { setLegacy(false); onChange({ city: value.city, ward: next }) }}
          />
          {errors.ward && <span role="alert" className="block text-red-700 text-sm">{errors.ward}</span>}
        </div>
      </div>
      {legacy && (
        <p role="status" className="text-sm text-secondary">
          Địa chỉ đã lưu theo đơn vị hành chính cũ, vui lòng chọn lại Tỉnh/Phường theo danh mục mới.
        </p>
      )}
    </div>
  )
}
