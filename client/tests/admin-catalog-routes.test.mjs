import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

// Route contract regressions for the shared 403 page (no browser runtime).
test('admin URLs use a centralized access guard and shared 403 page', () => {
  const app = readFileSync(new URL('../src/app/App.tsx', import.meta.url), 'utf8')
  const gate = readFileSync(new URL('../src/pages/admin/AdminAccessGuard.tsx', import.meta.url), 'utf8')
  const page = readFileSync(new URL('../src/pages/admin/ForbiddenPage.tsx', import.meta.url), 'utf8')
  assert.match(app, /path="admin" element=\{<AdminAccessGuard \/>\}/)
  assert.match(app, /path="403" element=\{<ForbiddenPage \/>\}/)
  assert.match(gate, /adminCatalog\.categories\(\)/)
  assert.match(gate, /status === 'denied'/)
  assert.match(page, /Bạn không có quyền ở đây/)
})
