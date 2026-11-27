import { test } from 'node:test'
import assert from 'node:assert/strict'
import { landingPath, safeNextPath } from '../src/features/auth/redirect.ts'

test('chỉ cho phép next path nội bộ', () => {
  assert.equal(safeNextPath('/admin/catalog'), '/admin/catalog')
  assert.equal(safeNextPath('//example.com'), '/')
  assert.equal(safeNextPath('https://example.com'), '/')
  assert.equal(safeNextPath('/\\example.com'), '/')
  assert.equal(safeNextPath(null), '/')
})

test('landingPath: admin goes to the dashboard, customer to the shop, explicit next wins', () => {
  assert.equal(landingPath('admin', null), '/admin')
  assert.equal(landingPath('customer', null), '/')
  assert.equal(landingPath(undefined, '/'), '/')
  assert.equal(landingPath('admin', '/admin/orders'), '/admin/orders')
  assert.equal(landingPath('customer', '/gio-hang'), '/gio-hang')
  assert.equal(landingPath('admin', '//evil.example'), '/admin')
  // A customer sent to /admin still lands there; the admin guard answers with 403.
  assert.equal(landingPath('customer', '/admin'), '/admin')
})
