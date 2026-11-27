import { formFieldForServerField, validateTailoringForm, emptyTailoringForm } from './form.ts'
import type { TailoringRequestBody } from './types.ts'

/** Extra consultation fields without a dedicated API field; sent as "Nhãn: giá trị" lines in `notes`. */
export interface ConsultInput {
  name: string
  phone: string
  /** Ordered label → value pairs; blank values are dropped. */
  extras: [label: string, value: string][]
}

export function buildConsultNotes(extras: ConsultInput['extras']): string {
  return extras
    .map(([label, value]) => [label, value.trim()] as const)
    .filter(([, value]) => value !== '')
    .map(([label, value]) => `${label}: ${value}`)
    .join('\n')
}

export function buildConsultBody(input: ConsultInput): TailoringRequestBody {
  const body: TailoringRequestBody = { full_name: input.name.trim(), phone: input.phone.trim() }
  const notes = buildConsultNotes(input.extras)
  if (notes) body.notes = notes
  return body
}

/** Local name/phone errors, reusing the tailoring form rules. */
export function validateConsult(input: Pick<ConsultInput, 'name' | 'phone'>): Record<string, string> {
  const errors = validateTailoringForm({ ...emptyTailoringForm, name: input.name, phone: input.phone })
  const out: Record<string, string> = {}
  if (errors.name) out.name = errors.name
  if (errors.phone) out.phone = errors.phone
  return out
}

/** Server field → consult control name ("notes" has no own control here, so it is reported as a general error). */
export function consultFieldForServerField(field: string): string {
  const mapped = formFieldForServerField(field)
  return mapped === 'name' || mapped === 'phone' ? mapped : ''
}

export const consultSuccessMessage =
  'Đã nhận yêu cầu tư vấn của quý khách — đây chưa phải đặt đơn. Cửa hàng sẽ liên hệ xác nhận số đo, giá và lịch hẹn.'
