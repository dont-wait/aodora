import type { ReactNode } from 'react'
import { PageHeader } from './ui'
import { useAdminCrumb } from './breadcrumb'

/** Page title + content inside AdminLayout (navigation, chrome and toast live in the layout). */
export default function AdminOrderFrame({ title, crumb, children }: { title: string; crumb?: string; children: ReactNode }) {
  useAdminCrumb(crumb ?? null)
  return (
    <>
      <PageHeader title={title} />
      {children}
    </>
  )
}
