import { useId, useState } from 'react'
import { chartColor } from '@/features/admin/dashboard'

export interface DonutItem { key: string; label: string; value: number }
interface Props { items: DonutItem[]; label: string; unit?: string }

const R = 56
const STROKE = 26
const C = 2 * Math.PI * R

export default function DonutChart({ items, label, unit = 'đơn' }: Props) {
  const tableId = useId()
  const [active, setActive] = useState<string | null>(null)
  const total = items.reduce((sum, i) => sum + i.value, 0)
  if (total === 0) return <p className="py-8 text-center text-sm text-on-surface-variant">Chưa có dữ liệu trong kỳ.</p>
  const offsets: number[] = []
  items.reduce((acc, item) => { offsets.push(acc); return acc + (item.value / total) * C }, 0)
  const shown = active ? items.find((i) => i.key === active) : null
  return (
    <div className="flex flex-wrap items-center gap-5">
      <svg viewBox="0 0 160 160" className="h-40 w-40 shrink-0" role="group" aria-label={label} aria-describedby={tableId}>
        <g transform="rotate(-90 80 80)">
          {items.map((item, i) => {
            const length = (item.value / total) * C
            const dash = length + ' ' + (C - length)
            return (
              <circle
                key={item.key}
                cx={80} cy={80} r={R} fill="none"
                stroke={chartColor(i)} strokeWidth={active === item.key ? STROKE + 4 : STROKE}
                strokeDasharray={dash} strokeDashoffset={-offsets[i]}
                opacity={active && active !== item.key ? 0.45 : 1}
                tabIndex={0}
                role="img"
                aria-label={item.label + ': ' + item.value + ' ' + unit + ' (' + Math.round((item.value / total) * 100) + '%)'}
                onMouseEnter={() => setActive(item.key)}
                onMouseLeave={() => setActive(null)}
                onFocus={() => setActive(item.key)}
                onBlur={() => setActive(null)}
                className="outline-none"
              >
                <title>{item.label + ': ' + item.value + ' ' + unit}</title>
              </circle>
            )
          })}
        </g>
        <text x={80} y={shown ? 76 : 84} textAnchor="middle" fontSize={shown ? 20 : 24} fontWeight={600} fill="#1f1b18" className="tabular-nums">{shown ? shown.value : total}</text>
        <text x={80} y={shown ? 94 : 100} textAnchor="middle" fontSize={10} fill="#564242">{shown ? shown.label.slice(0, 16) : unit}</text>
      </svg>
      <ul className="min-w-40 flex-1 space-y-1.5 text-sm">
        {items.map((item, i) => (
          <li key={item.key} className="flex items-center justify-between gap-3" onMouseEnter={() => setActive(item.key)} onMouseLeave={() => setActive(null)}>
            <span className="flex items-center gap-2"><span className="h-3 w-3 rounded-sm" style={{ background: chartColor(i) }} />{item.label}</span>
            <span className="tabular-nums text-on-surface-variant">{item.value} · {Math.round((item.value / total) * 100)}%</span>
          </li>
        ))}
      </ul>
      <table id={tableId} className="sr-only">
        <caption>{label}</caption>
        <tbody>{items.map((i) => <tr key={i.key}><td>{i.label}</td><td>{i.value}</td></tr>)}</tbody>
      </table>
    </div>
  )
}
