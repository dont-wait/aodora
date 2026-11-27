import { useState } from 'react'
import { tailoringApi, TailoringApiError } from './api'
import { buildConsultBody, consultFieldForServerField, consultSuccessMessage, validateConsult, type ConsultInput } from './consult'

/** Shared submit state for the lightweight consultation forms (home / home-visit). */
export function useConsultSubmit() {
  const [submitting, setSubmitting] = useState(false)
  const [success, setSuccess] = useState('')
  const [error, setError] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})

  /** Resolves true when the request was stored (caller resets the form). */
  async function submit(input: ConsultInput): Promise<boolean> {
    if (submitting) return false
    setSuccess(''); setError('')
    const local = validateConsult(input)
    setFieldErrors(local)
    if (Object.keys(local).length > 0) return false
    setSubmitting(true)
    try {
      await tailoringApi.submit(buildConsultBody(input))
      setSuccess(consultSuccessMessage)
      return true
    } catch (err) {
      if (err instanceof TailoringApiError) {
        const field = err.field ? consultFieldForServerField(err.field) : ''
        if (field) setFieldErrors({ [field]: err.message })
        else setError(err.message)
      } else {
        setError('Không thể gửi yêu cầu. Vui lòng kiểm tra kết nối và thử lại.')
      }
      return false
    } finally {
      setSubmitting(false)
    }
  }

  return { submitting, success, error, fieldErrors, submit }
}
