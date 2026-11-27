import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, Navigate } from 'react-router-dom'
import DonutChart from './charts/DonutChart'
import HBarChart from './charts/HBarChart'
import TimeSeriesChart from './charts/TimeSeriesChart'
import { Alert, Button, Card, CardSkeleton, DataTable, EmptyState, PageHeader, Skeleton, StatusBadge } from './ui'
import { inputClass, tdClass, tdNumClass, thClass, thNumClass } from './uiStyles'
import { AdminApiError } from '@/features/admin/catalog'
import {
  defaultGranularity, fetchDashboard, formatChange, formatPercent, formatVND, formatVNDShort, presetRange, rangeDays, statusLabel,
  type DashboardReport, type Granularity, type Metric, type Preset,
} from '@/features/admin/dashboard'

const PRESETS: { id: Preset; label: string }[] = [
  { id: '7d', label: '7 ngày' }, { id: '30d', label: '30 ngày' }, { id: '90d', label: '90 ngày' }, { id: 'ytd', label: 'Năm nay' }, { id: 'custom', label: 'Tùy chọn' },
]
const GRANULARITIES: { id: Granularity; label: string }[] = [{ id: 'day', label: 'Ngày' }, { id: 'week', label: 'Tuần' }, { id: 'month', label: 'Tháng' }]

function KpiTile({ title, metric, format, invert = false }: { title: string; metric: Metric; format: (v: number) => string; invert?: boolean }) {
  const pct = metric.change_pct
  const up = pct !== null && pct > 0
  const down = pct !== null && pct < 0
  const good = invert ? down : up
  const bad = invert ? up : down
  return (
    <div className="rounded-lg border border-outline-variant/50 bg-surface-container-lowest p-5 shadow-[0_1px_2px_rgba(31,27,24,0.04)]">
      <p className="text-[13px] font-medium text-on-surface-variant">{title}</p>
      <p className="mt-2 text-2xl font-semibold tabular-nums text-primary">{format(metric.value)}</p>
      <p className={`mt-1 flex items-center gap-1 text-xs tabular-nums ${good ? 'text-green-700' : bad ? 'text-red-700' : 'text-on-surface-variant'}`}>
        <span aria-hidden="true">{up ? '▲' : down ? '▼' : '•'}</span>
        <span>{formatChange(pct)}</span>
        <span className="text-on-surface-variant">so với kỳ trước ({format(metric.previous)})</span>
      </p>
    </div>
  )
}

function DashboardSkeleton() {
  return (
    <div role="status" aria-label="Đang tải số liệu" className="space-y-6">
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-5">{Array.from({ length: 5 }, (_, i) => <Skeleton key={i} className="h-28" />)}</div>
      <CardSkeleton lines={8} />
      <div className="grid gap-6 lg:grid-cols-2"><CardSkeleton lines={6} /><CardSkeleton lines={6} /></div>
    </div>
  )
}

export default function DashboardPage() {
  const initial = presetRange('30d')
  const [preset, setPreset] = useState<Preset>('30d')
  const [range, setRange] = useState(initial)
  const [granularity, setGranularity] = useState<Granularity>('day')
  const [mode, setMode] = useState<'line' | 'bar'>('line')
  const [report, setReport] = useState<DashboardReport | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [forbidden, setForbidden] = useState(false)
  const [revision, setRevision] = useState(0)
  const sequence = useRef(0)

  const days = rangeDays(range.from, range.to)
  const rangeError = !Number.isFinite(days) || days < 1 ? 'Khoảng ngày không hợp lệ' : days > 366 ? 'Khoảng thời gian tối đa 366 ngày' : ''

  useEffect(() => {
    if (rangeError) return
    const current = ++sequence.current
    const controller = new AbortController()
    fetchDashboard({ from: range.from, to: range.to, granularity }, controller.signal)
      .then((data) => {
        if (current !== sequence.current) return
        setReport(data); setError(''); setLoading(false)
      })
      .catch((err: unknown) => {
        if (current !== sequence.current || (err instanceof DOMException && err.name === 'AbortError')) return
        if (err instanceof AdminApiError && [401, 403].includes(err.status)) setForbidden(true)
        setError(err instanceof Error ? err.message : 'Không thể tải số liệu thống kê')
        setLoading(false)
      })
    return () => { controller.abort() }
  }, [range.from, range.to, granularity, revision, rangeError])

  const choosePreset = useCallback((next: Preset) => {
    setPreset(next)
    setLoading(true)
    if (next !== 'custom') {
      setRange(presetRange(next))
      setGranularity(defaultGranularity(next))
    }
  }, [])
  const retry = () => { setError(''); setLoading(true); setRevision((n) => n + 1) }

  if (forbidden) return <Navigate to="/403" replace />
  const r = report

  return (
    <>
      <PageHeader
        title="Tổng quan"
        subtitle={r ? 'Từ ' + r.range.from + ' đến ' + r.range.to + ' (giờ Việt Nam) · so với ' + r.range.prev_from + ' – ' + r.range.prev_to : 'Số liệu bán hàng, tồn kho và may đo'}
        actions={<Button variant="secondary" onClick={retry} loading={loading && !!report}>Tải lại</Button>}
      />
      <Card>
        <div className="flex flex-wrap items-end gap-4">
          <div role="group" aria-label="Khoảng thời gian" className="flex flex-wrap gap-2">
            {PRESETS.map((p) => (
              <button
                key={p.id}
                type="button"
                aria-pressed={preset === p.id}
                onClick={() => choosePreset(p.id)}
                className={`h-10 rounded-lg border px-4 text-sm font-medium ${preset === p.id ? 'border-primary bg-primary text-on-primary' : 'border-outline-variant bg-surface-container-lowest hover:bg-surface-container'}`}
              >
                {p.label}
              </button>
            ))}
          </div>
          {preset === 'custom' && (
            <div className="flex flex-wrap items-end gap-3">
              <label className="space-y-1 text-[13px] font-medium text-on-surface-variant">Từ ngày<input type="date" className={inputClass + ' w-auto'} value={range.from} max={range.to} onChange={(e) => { setLoading(true); setRange({ ...range, from: e.target.value }) }} /></label>
              <label className="space-y-1 text-[13px] font-medium text-on-surface-variant">Đến ngày<input type="date" className={inputClass + ' w-auto'} value={range.to} min={range.from} onChange={(e) => { setLoading(true); setRange({ ...range, to: e.target.value }) }} /></label>
            </div>
          )}
          <div role="group" aria-label="Nhóm theo" className="flex gap-2 sm:ml-auto">
            {GRANULARITIES.map((g) => (
              <button
                key={g.id}
                type="button"
                aria-pressed={granularity === g.id}
                onClick={() => { setLoading(true); setGranularity(g.id) }}
                className={`h-10 rounded-lg border px-3 text-sm ${granularity === g.id ? 'border-primary bg-surface-container font-semibold text-primary' : 'border-outline-variant bg-surface-container-lowest hover:bg-surface-container'}`}
              >
                {g.label}
              </button>
            ))}
          </div>
        </div>
        {rangeError && <Alert>{rangeError}</Alert>}
      </Card>

      {error && <Alert action={<Button variant="secondary" onClick={retry}>Thử lại</Button>}>{error}</Alert>}
      {loading && !r && !error && <DashboardSkeleton />}
      {r && (
        <div className={loading ? 'space-y-6 opacity-60 transition-opacity' : 'space-y-6'} aria-busy={loading}>
          <section aria-label="Chỉ số chính" className="grid gap-4 sm:grid-cols-2 xl:grid-cols-5">
            <KpiTile title="Doanh thu" metric={r.kpis.revenue} format={formatVNDShort} />
            <KpiTile title="Số đơn" metric={r.kpis.orders} format={(v) => String(Math.round(v))} />
            <KpiTile title="Giá trị đơn trung bình" metric={r.kpis.avg_order_value} format={formatVNDShort} />
            <KpiTile title="Khách mới" metric={r.kpis.new_customers} format={(v) => String(Math.round(v))} />
            <KpiTile title="Tỉ lệ hủy" metric={r.kpis.cancel_rate} format={formatPercent} invert />
          </section>

          <Card
            title="Doanh thu theo thời gian"
            description="Doanh thu = tổng tiền của đơn đã thanh toán và không bị hủy."
            actions={
              <div role="group" aria-label="Kiểu biểu đồ" className="flex gap-2">
                {(['line', 'bar'] as const).map((m) => (
                  <button key={m} type="button" aria-pressed={mode === m} onClick={() => setMode(m)}
                    className={`h-8 rounded-lg border px-3 text-xs ${mode === m ? 'border-primary bg-surface-container font-semibold text-primary' : 'border-outline-variant hover:bg-surface-container'}`}>
                    {m === 'line' ? 'Đường' : 'Cột'}
                  </button>
                ))}
              </div>
            }
          >
            <TimeSeriesChart data={r.series} granularity={r.range.granularity} mode={mode} label={'Doanh thu theo ' + (r.range.granularity === 'day' ? 'ngày' : r.range.granularity === 'week' ? 'tuần' : 'tháng') + ' từ ' + r.range.from + ' đến ' + r.range.to} />
          </Card>

          <div className="grid gap-6 lg:grid-cols-2">
            <Card title="Đơn theo trạng thái"><DonutChart label="Số đơn theo trạng thái" items={r.orders_by_status.map((c) => ({ key: c.key, label: statusLabel(c.key), value: c.count }))} /></Card>
            <Card title="Đơn theo thanh toán">
              <DonutChart label="Số đơn theo trạng thái thanh toán" items={r.orders_by_payment_status.map((c) => ({ key: c.key, label: statusLabel(c.key), value: c.count }))} />
              {r.orders_by_payment_method.length > 0 && (
                <ul className="space-y-1 border-t border-outline-variant/40 pt-3 text-sm">
                  {r.orders_by_payment_method.map((m) => (
                    <li key={m.key} className="flex justify-between gap-3"><span>{statusLabel(m.key)} · {m.count} đơn</span><span className="tabular-nums">{formatVND(m.revenue)}</span></li>
                  ))}
                </ul>
              )}
            </Card>
          </div>

          <div className="grid gap-6 lg:grid-cols-2">
            <Card title="Top 10 sản phẩm theo doanh thu">
              <HBarChart label="Top sản phẩm theo doanh thu" format={formatVNDShort} items={r.top_products_revenue.map((p) => ({ key: p.product_id || p.name, label: p.name, value: p.revenue, detail: p.quantity + ' sản phẩm' }))} />
            </Card>
            <Card title="Top 10 sản phẩm theo số lượng">
              <HBarChart label="Top sản phẩm theo số lượng" colorIndex={1} format={(v) => v + ' sp'} items={r.top_products_quantity.map((p) => ({ key: p.product_id || p.name, label: p.name, value: p.quantity, detail: formatVND(p.revenue) }))} />
            </Card>
            <Card title="Doanh thu theo danh mục">
              <HBarChart label="Doanh thu theo danh mục" categorical format={formatVNDShort} items={r.revenue_by_category.map((c) => ({ key: c.category_id || c.name, label: c.name, value: c.revenue, detail: c.quantity + ' sản phẩm' }))} />
            </Card>
            <Card title="Doanh thu theo size">
              <HBarChart label="Doanh thu theo size" colorIndex={2} format={formatVNDShort} items={r.revenue_by_size.map((s) => ({ key: s.size || 'none', label: s.size || 'Không có cỡ', value: s.revenue, detail: s.quantity + ' sản phẩm' }))} />
            </Card>
          </div>

          <Card
            title="Tồn kho"
            description={'SKU đang bán có tồn ≤ ' + r.inventory.low_stock_threshold + '. Giá trị tồn = giá bán × tồn kho của mọi sản phẩm.'}
            actions={<Link className="text-sm font-medium text-primary underline" to="/admin/catalog">Quản lý sản phẩm</Link>}
          >
            <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
              {[
                ['Giá trị tồn kho', formatVND(r.inventory.stock_value)],
                ['SKU hết hàng', String(r.inventory.out_of_stock_skus)],
                ['SKU sắp hết', String(r.inventory.low_stock_skus - r.inventory.out_of_stock_skus)],
                ['Đang bán / đang ẩn', r.inventory.active_products + ' / ' + r.inventory.hidden_products],
              ].map(([k, v]) => (
                <div key={k} className="rounded-lg bg-surface-container-low p-4"><p className="text-[13px] text-on-surface-variant">{k}</p><p className="mt-1 text-lg font-semibold tabular-nums">{v}</p></div>
              ))}
            </div>
            {r.inventory.low_stock.length === 0 ? (
              <EmptyState title="Không có SKU nào sắp hết hàng" />
            ) : (
              <DataTable label="SKU sắp hết hàng">
                <thead><tr><th className={thClass}>Sản phẩm</th><th className={thClass}>SKU</th><th className={thClass}>Màu / cỡ</th><th className={thNumClass}>Tồn</th></tr></thead>
                <tbody>
                  {r.inventory.low_stock.map((item) => (
                    <tr key={item.sku}>
                      <td className={tdClass}><Link className="font-medium text-primary underline" to={'/admin/catalog?product=' + encodeURIComponent(item.product_id)}>{item.name}</Link></td>
                      <td className={tdClass + ' font-mono text-xs'}>{item.sku}</td>
                      <td className={tdClass}>{[item.color, item.size].filter(Boolean).join(' / ') || '—'}</td>
                      <td className={tdNumClass}>{item.stock <= 0 ? <StatusBadge tone="danger">Hết hàng</StatusBadge> : item.stock}</td>
                    </tr>
                  ))}
                </tbody>
              </DataTable>
            )}
          </Card>

          <Card title="Yêu cầu may đo trong kỳ">
            {r.tailoring_by_status.length === 0 ? <p className="text-sm text-on-surface-variant">Chưa có yêu cầu may đo nào trong kỳ.</p> : (
              <HBarChart label="Yêu cầu may đo theo trạng thái" categorical format={(v) => String(v)} items={r.tailoring_by_status.map((c) => ({ key: c.key, label: statusLabel(c.key), value: c.count }))} />
            )}
          </Card>
        </div>
      )}
    </>
  )
}
