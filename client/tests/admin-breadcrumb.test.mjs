import { test } from 'node:test'
import assert from 'node:assert/strict'
import { adminBreadcrumb } from '../src/pages/admin/breadcrumb.ts'

test('dashboard and unknown routes have no breadcrumb', () => {
  assert.deepEqual(adminBreadcrumb('/admin'), [])
  assert.deepEqual(adminBreadcrumb('/admin/'), [])
  assert.deepEqual(adminBreadcrumb('/admin/unknown'), [])
  assert.deepEqual(adminBreadcrumb('/san-pham/ao'), [])
})

test('section pages link back to the dashboard', () => {
  assert.deepEqual(adminBreadcrumb('/admin/orders'), [{ label: 'Tổng quan', to: '/admin' }, { label: 'Đơn hàng' }])
  assert.deepEqual(adminBreadcrumb('/admin/reviews').map((c) => c.label), ['Tổng quan', 'Đánh giá'])
})

test('detail pages link to their list and use the page label when known', () => {
  assert.deepEqual(adminBreadcrumb('/admin/orders/abc'), [
    { label: 'Tổng quan', to: '/admin' },
    { label: 'Đơn hàng', to: '/admin/orders' },
    { label: 'Chi tiết' },
  ])
  assert.equal(adminBreadcrumb('/admin/tailoring/x', 'Nguyễn An').at(-1).label, 'Nguyễn An')
  assert.equal(adminBreadcrumb('/admin/returns/x', 'Đơn AD-1').at(1).to, '/admin/returns')
})
