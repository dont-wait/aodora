// Mirrors POST /api/v1/tailoring-requests (server/docs/DW04_TAILORING.md mục 5).
export type PreferredContact = 'phone' | 'zalo'

export interface TailoringMeasurements {
  body?: string
  neck_cm?: number
  bust_cm?: number
  waist_cm?: number
  hips_cm?: number
  shoulder_cm?: number
  sleeve_cm?: number
  garment_length_cm?: number
}

/** Only these fields are accepted; customer_id, status and timestamps are server-owned. */
export interface TailoringRequestBody {
  full_name: string
  phone: string
  preferred_contact?: PreferredContact
  product_id?: string
  product_slug?: string
  measurements?: TailoringMeasurements
  notes?: string
}

/** A 201 only confirms the request was stored; it is not an order. */
export interface TailoringReceipt {
  id: string
  status: string
}
