import type { ReactNode } from 'react'
import { PageHeader } from './ui'

/** Page title + content inside AdminLayout (navigation, chrome and toast live in the layout). */
export default function AdminOrderFrame({ title, children }: { title: string; children: ReactNode }) {
  return (
    <>
      <PageHeader title={title} />
      {children}
    </>
  )
}
