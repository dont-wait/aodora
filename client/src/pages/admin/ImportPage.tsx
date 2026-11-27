import { useRef, useState, type DragEvent } from 'react'
import { Link, Navigate } from 'react-router-dom'
import { Alert, Button, ButtonLink, Card, DataTable, EmptyState, PageHeader, StatusBadge } from './ui'
import { tdClass, thClass } from './uiStyles'
import { AdminApiError } from '@/features/admin/catalog'
import {
  canCommit, commitLabel, downloadExport, errorRows, fileProblem, runImport, saveBlob, templateCsv, templateJson,
  type ImportKind, type ImportReport,
} from '@/features/admin/import'
import { useToast } from '@/hooks/useToast'

const KINDS: { id: ImportKind; label: string; hint: string }[] = [
  { id: 'categories', label: 'Danh mục', hint: 'Mỗi dòng một danh mục. Danh mục cha có thể nằm cùng file.' },
  { id: 'products', label: 'Sản phẩm', hint: 'Mỗi dòng một SKU; các dòng cùng product_slug gộp thành một sản phẩm. Nhập danh mục trước.' },
]
const actionTone = { create: 'success', update: 'info', skip: 'neutral' } as const
const actionLabel = { create: 'Tạo mới', update: 'Cập nhật', skip: 'Bỏ qua (không đổi)' } as const

export default function ImportPage() {
  const { showToast } = useToast()
  const [kind, setKind] = useState<ImportKind>('categories')
  const [file, setFile] = useState<File | null>(null)
  const [report, setReport] = useState<ImportReport | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState<'check' | 'commit' | 'export' | ''>('')
  const [dragging, setDragging] = useState(false)
  const [forbidden, setForbidden] = useState(false)
  const input = useRef<HTMLInputElement>(null)

  const fail = (err: unknown, fallback: string) => {
    if (err instanceof AdminApiError && [401, 403].includes(err.status)) setForbidden(true)
    else setError(err instanceof Error ? err.message : fallback)
  }
  const reset = (next: ImportKind = kind) => { setKind(next); setFile(null); setReport(null); setError(''); if (input.current) input.current.value = '' }

  async function check(selected: File) {
    const problem = fileProblem(selected)
    setFile(selected); setReport(null); setError('')
    if (problem) { setError(problem); return }
    setBusy('check')
    try { setReport(await runImport(kind, selected, 'dry_run')) } catch (err) { fail(err, 'Không thể kiểm tra file') } finally { setBusy('') }
  }
  async function commit() {
    if (!file || !canCommit(report)) return
    setBusy('commit'); setError('')
    try {
      const done = await runImport(kind, file, 'commit')
      setReport(done)
      if (done.committed) showToast('Đã nhập dữ liệu thành công', 'success')
    } catch (err) { fail(err, 'Không thể nhập dữ liệu') } finally { setBusy('') }
  }
  async function exportCsv(which: ImportKind) {
    setBusy('export'); setError('')
    try { await downloadExport(which) } catch (err) { fail(err, 'Không thể xuất dữ liệu') } finally { setBusy('') }
  }
  const onDrop = (event: DragEvent) => {
    event.preventDefault(); setDragging(false)
    const dropped = event.dataTransfer.files?.[0]
    if (dropped) void check(dropped)
  }

  if (forbidden) return <Navigate to="/403" replace />
  const current = KINDS.find((k) => k.id === kind)!
  const rowsWithErrors = report ? errorRows(report) : new Set<number>()
  const finished = report?.mode === 'commit' && report.committed

  return (
    <>
      <PageHeader title="Nhập dữ liệu" subtitle="Nhập hàng loạt danh mục và sản phẩm từ file CSV (UTF-8) hoặc JSON" />
      <Card title="1. Chọn loại dữ liệu và file" description={current.hint}>
        <div role="group" aria-label="Loại dữ liệu" className="flex flex-wrap gap-2">
          {KINDS.map((k) => (
            <button key={k.id} type="button" aria-pressed={kind === k.id} onClick={() => reset(k.id)}
              className={`h-10 rounded-lg border px-4 text-sm font-medium ${kind === k.id ? 'border-primary bg-primary text-on-primary' : 'border-outline-variant bg-surface-container-lowest hover:bg-surface-container'}`}>
              {k.label}
            </button>
          ))}
        </div>
        <div
          onDragOver={(e) => { e.preventDefault(); setDragging(true) }}
          onDragLeave={() => setDragging(false)}
          onDrop={onDrop}
          className={`flex flex-col items-center gap-3 rounded-lg border-2 border-dashed p-8 text-center ${dragging ? 'border-primary bg-surface-container' : 'border-outline-variant bg-surface-container-low'}`}
        >
          <p className="text-sm">Kéo thả file vào đây, hoặc</p>
          <input ref={input} id="import-file" type="file" accept=".csv,.json,text/csv,application/json" className="sr-only" onChange={(e) => { const f = e.target.files?.[0]; if (f) void check(f) }} />
          <label htmlFor="import-file" className="inline-flex h-10 cursor-pointer items-center rounded-lg border border-primary bg-surface-container-lowest px-4 text-sm font-medium text-primary hover:bg-surface-container focus-within:outline focus-within:outline-2">Chọn file</label>
          {file && <p className="text-[13px] text-on-surface-variant">{file.name} · {(file.size / 1024).toFixed(1)} KB</p>}
          <p className="text-xs text-on-surface-variant">Excel: Lưu thành &quot;CSV UTF-8&quot;. Tối đa 5 MB và 5.000 dòng.</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button variant="ghost" onClick={() => saveBlob(new Blob([templateCsv(kind)], { type: 'text/csv;charset=utf-8' }), 'aodora-mau-' + kind + '.csv')}>Tải file mẫu CSV</Button>
          <Button variant="ghost" onClick={() => saveBlob(new Blob([templateJson(kind)], { type: 'application/json' }), 'aodora-mau-' + kind + '.json')}>Tải file mẫu JSON</Button>
          <Button variant="secondary" loading={busy === 'export'} onClick={() => void exportCsv(kind)}>Xuất CSV {current.label.toLowerCase()} hiện có</Button>
        </div>
      </Card>

      {error && <Alert>{error}</Alert>}
      {busy === 'check' && <Alert kind="success">Đang kiểm tra file...</Alert>}

      {report && (
        <Card title={finished ? '3. Kết quả nhập' : '2. Xem trước kết quả kiểm tra'}>
          {report.errors.length > 0 && <Alert>Có {report.errors.length} lỗi. Sửa file rồi chọn lại; chưa có dữ liệu nào được ghi.</Alert>}
          {finished && <Alert kind="success">Đã nhập xong. <Link className="font-medium underline" to={kind === 'categories' ? '/admin/categories' : '/admin/catalog'}>Xem danh sách {current.label.toLowerCase()}</Link></Alert>}
          {report.mode === 'commit' && !report.committed && report.errors.length > 0 && <Alert>Việc nhập bị dừng giữa chừng. Xem ghi chú bên dưới để biết phần đã ghi.</Alert>}
          <dl className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            {[['Tạo mới', report.created], ['Cập nhật', report.updated], ['Không đổi', report.skipped], ['Tồn bỏ qua (SKU đã có)', report.ignored_stock]].map(([k, v]) => (
              <div key={k} className="rounded-lg bg-surface-container-low p-4"><dt className="text-[13px] text-on-surface-variant">{k}</dt><dd className="mt-1 text-xl font-semibold tabular-nums">{v}</dd></div>
            ))}
          </dl>
          {report.notes.length > 0 && <ul className="list-disc space-y-1 pl-5 text-[13px] text-on-surface-variant">{report.notes.map((n) => <li key={n}>{n}</li>)}</ul>}

          {report.errors.length > 0 && (
            <DataTable label="Lỗi theo dòng">
              <thead><tr><th className={thClass}>Dòng</th><th className={thClass}>Cột</th><th className={thClass}>Lỗi</th></tr></thead>
              <tbody>
                {report.errors.map((e, i) => (
                  <tr key={i} className="!bg-red-50 text-red-900">
                    <td className={tdClass + ' tabular-nums'}>{e.row || '—'}</td>
                    <td className={tdClass + ' font-mono text-xs'}>{e.field}</td>
                    <td className={tdClass}>{e.message}</td>
                  </tr>
                ))}
              </tbody>
            </DataTable>
          )}
          {report.items.length > 0 ? (
            <DataTable label="Các mục sẽ được xử lý">
              <thead><tr><th className={thClass}>Dòng</th><th className={thClass}>Slug</th><th className={thClass}>Hành động</th><th className={thClass}>Chi tiết</th></tr></thead>
              <tbody>
                {report.items.map((item) => (
                  <tr key={item.slug + item.row} className={rowsWithErrors.has(item.row) ? '!bg-red-50' : ''}>
                    <td className={tdClass + ' tabular-nums'}>{item.row}</td>
                    <td className={tdClass + ' font-mono text-xs'}>{item.slug}</td>
                    <td className={tdClass}><StatusBadge tone={actionTone[item.action]}>{actionLabel[item.action]}</StatusBadge></td>
                    <td className={tdClass + ' text-on-surface-variant'}>{item.detail || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </DataTable>
          ) : report.errors.length === 0 && <EmptyState title="Không có gì để nhập" />}

          {!finished && (
            <div className="flex flex-wrap items-center justify-end gap-3">
              {canCommit(report) ? <span className="text-[13px] text-on-surface-variant">Tồn kho của SKU đã có sẽ không bị ghi đè.</span> : null}
              <Button loading={busy === 'commit'} disabled={!canCommit(report)} onClick={() => void commit()}>{commitLabel(report)}</Button>
            </div>
          )}
          {finished && <div className="flex justify-end"><ButtonLink to="/admin/import" onClick={() => reset()}>Nhập file khác</ButtonLink></div>}
        </Card>
      )}
    </>
  )
}
