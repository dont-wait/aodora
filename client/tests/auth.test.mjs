import { test } from 'node:test'
import assert from 'node:assert/strict'
import { safeNextPath } from '../src/features/auth/redirect.ts'

test('chỉ cho phép next path nội bộ', () => {
  assert.equal(safeNextPath('/admin/catalog'), '/admin/catalog')
  assert.equal(safeNextPath('//example.com'), '/')
  assert.equal(safeNextPath('https://example.com'), '/')
  assert.equal(safeNextPath('/\\example.com'), '/')
  assert.equal(safeNextPath(null), '/')
})
