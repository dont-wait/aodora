import { useId, useRef, useState, type KeyboardEvent, type MouseEvent } from 'react'
import { CHART_COLORS, formatBucketLabel, formatVND, formatVNDShort, niceScale, scaleLinear, type Granularity } from '@/features/admin/dashboard'

type Point = { date: string; revenue: number; orders: number }
interface Props {
  data: Point[]
  granularity: Granularity
  mode: 'line' | 'bar'
  /** Accessible description, e.g. "Doanh thu theo ngày từ 01/03 đến 30/03". */
  label: string
}

const W = 800
const H = 300
const M = { l: 56, r: 16, t: 16, b: 32 }
const GRID = '#e6dcd6'
const AXIS_TEXT = '#564242'

export default function TimeSeriesChart({ data, granularity, mode, label }: Props) {
  const tableId = useId()
  const svgRef = useRef<SVGSVGElement>(null)
  const [active, setActive] = useState<number | null>(null)
  if (data.length === 0) return <p className="py-10 text-center text-sm text-on-surface-variant">Chưa có dữ liệu trong kỳ.</p>

  const max = Math.max(...data.map((p) => p.revenue), 0)
  const { max: yMax, ticks } = niceScale(max)
  const y = scaleLinear(0, yMax, H - M.b, M.t)
  const band = (W - M.l - M.r) / data.length
  const cx = (i: number) => M.l + band * i + band / 2
  const labelEvery = Math.max(1, Math.ceil(data.length / 8))
  const color = CHART_COLORS[0]

  const line = data.map((p, i) => (i === 0 ? 'M' : 'L') + cx(i).toFixed(1) + ' ' + y(p.revenue).toFixed(1)).join(' ')
  const area = line + ' L' + cx(data.length - 1).toFixed(1) + ' ' + (H - M.b) + ' L' + cx(0).toFixed(1) + ' ' + (H - M.b) + ' Z'

  const indexFromEvent = (event: MouseEvent<SVGSVGElement>) => {
    const rect = svgRef.current?.getBoundingClientRect()
    if (!rect || rect.width === 0) return null
    const x = ((event.clientX - rect.left) / rect.width) * W
    return Math.min(data.length - 1, Math.max(0, Math.floor((x - M.l) / band)))
  }
  const onKeyDown = (event: KeyboardEvent<SVGSVGElement>) => {
    const last = data.length - 1
    let next: number | null = null
    if (event.key === 'ArrowRight') next = active === null ? 0 : Math.min(last, active + 1)
    else if (event.key === 'ArrowLeft') next = active === null ? last : Math.max(0, active - 1)
    else if (event.key === 'Home') next = 0
    else if (event.key === 'End') next = last
    else if (event.key === 'Escape') { setActive(null); return }
    if (next !== null) { event.preventDefault(); setActive(next) }
  }
  const point = active === null ? null : data[active]

  return (
    <div className="relative">
      <svg
        ref={svgRef}
        viewBox={'0 0 ' + W + ' ' + H}
        className="h-auto w-full select-none"
        role="group"
        aria-label={label + '. Dùng phím mũi tên để xem từng mốc.'}
        aria-describedby={tableId}
        tabIndex={0}
        onMouseMove={(event) => setActive(indexFromEvent(event))}
        onMouseLeave={() => setActive(null)}
        onKeyDown={onKeyDown}
        onBlur={() => setActive(null)}
      >
        {ticks.map((t) => (
          <g key={t}>
            <line x1={M.l} x2={W - M.r} y1={y(t)} y2={y(t)} stroke={GRID} strokeWidth={1} />
            <text x={M.l - 8} y={y(t) + 4} textAnchor="end" fontSize={11} fill={AXIS_TEXT}>{formatVNDShort(t)}</text>
          </g>
        ))}
        {data.map((p, i) => (i % labelEvery === 0 ? <text key={p.date} x={cx(i)} y={H - 10} textAnchor="middle" fontSize={11} fill={AXIS_TEXT}>{formatBucketLabel(p.date, granularity)}</text> : null))}
        {mode === 'bar'
          ? data.map((p, i) => {
              const h = Math.max(0, H - M.b - y(p.revenue))
              const w = Math.max(1, band * 0.7)
              return <rect key={p.date} x={cx(i) - w / 2} y={H - M.b - h} width={w} height={h} rx={2} fill={color} opacity={active === null || active === i ? 1 : 0.45} />
            })
          : (
            <>
              <path d={area} fill={color} opacity={0.1} />
              <path d={line} fill="none" stroke={color} strokeWidth={2} strokeLinejoin="round" strokeLinecap="round" />
              {data.length <= 45 && data.map((p, i) => <circle key={p.date} cx={cx(i)} cy={y(p.revenue)} r={active === i ? 5 : 2.5} fill={color} />)}
            </>
          )}
        {active !== null && <line x1={cx(active)} x2={cx(active)} y1={M.t} y2={H - M.b} stroke={color} strokeWidth={1} strokeDasharray="3 3" opacity={0.6} />}
        {active !== null && mode === 'line' && <circle cx={cx(active)} cy={y(data[active].revenue)} r={5} fill="#fff" stroke={color} strokeWidth={2} />}
      </svg>
      {point && active !== null && (
        <div
          role="status"
          className="pointer-events-none absolute top-2 z-10 min-w-40 rounded-lg border border-outline-variant bg-surface-container-lowest px-3 py-2 text-xs shadow-md"
          style={{ left: Math.min(Math.max((cx(active) / W) * 100, 12), 88) + '%', transform: 'translateX(-50%)' }}
        >
          <p className="font-semibold">{formatBucketLabel(point.date, granularity)}{granularity === 'week' ? ' (tuần)' : ''}</p>
          <p className="tabular-nums">Doanh thu: <strong>{formatVND(point.revenue)}</strong></p>
          <p className="tabular-nums">Số đơn: <strong>{point.orders}</strong></p>
        </div>
      )}
      <table id={tableId} className="sr-only">
        <caption>{label}</caption>
        <thead><tr><th>Mốc</th><th>Doanh thu</th><th>Số đơn</th></tr></thead>
        <tbody>{data.map((p) => <tr key={p.date}><td>{p.date}</td><td>{formatVND(p.revenue)}</td><td>{p.orders}</td></tr>)}</tbody>
      </table>
    </div>
  )
}
