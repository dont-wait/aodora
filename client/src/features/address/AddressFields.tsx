import { useEffect, useState } from 'react'
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
        <label htmlFor={idCity} className={labelClassName}>
          <span>Tỉnh / thành phố{required ? ' *' : ''}</span>
          <select
            id={idCity}
            name={`${idPrefix}-city`}
            autoComplete="address-level1"
            className={selectClassName}
            required={required}
            disabled={disabled || loading || error}
            value={province ? province.name : ''}
            aria-invalid={Boolean(errors.city)}
            onChange={(event) => { setLegacy(false); onChange({ city: event.target.value, ward: '' }) }}
          >
            <option value="">{loading ? 'Đang tải danh mục...' : 'Chọn tỉnh / thành phố'}</option>
            {provinces?.map((item) => <option key={item.code} value={item.name}>{item.name}</option>)}
          </select>
          {errors.city && <span role="alert" className="block text-red-700 text-sm">{errors.city}</span>}
        </label>
        <label htmlFor={idWard} className={labelClassName}>
          <span>Phường / xã{wardRequired ? ' *' : ''}</span>
          <select
            id={idWard}
            name={`${idPrefix}-ward`}
            autoComplete="address-level2"
            className={selectClassName}
            required={wardRequired}
            disabled={disabled || !province}
            value={province?.wards.some((item) => item.name === value.ward) ? value.ward : ''}
            aria-invalid={Boolean(errors.ward)}
            onChange={(event) => { setLegacy(false); onChange({ city: value.city, ward: event.target.value }) }}
          >
            <option value="">{province ? 'Chọn phường / xã' : 'Chọn tỉnh / thành phố trước'}</option>
            {province?.wards.map((item) => <option key={item.code} value={item.name}>{item.name}</option>)}
          </select>
          {errors.ward && <span role="alert" className="block text-red-700 text-sm">{errors.ward}</span>}
        </label>
      </div>
      {legacy && (
        <p role="status" className="text-sm text-secondary">
          Địa chỉ đã lưu theo đơn vị hành chính cũ, vui lòng chọn lại Tỉnh/Phường theo danh mục mới.
        </p>
      )}
    </div>
  )
}
