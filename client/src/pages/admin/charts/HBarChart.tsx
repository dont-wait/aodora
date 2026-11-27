import { useId, useState } from 'react'
import { chartColor } from '@/features/admin/dashboard'

export interface BarItem { key: string; label: string; value: number; detail?: string }
interface Props {
  items: BarItem[]
  format: (value: number) => string
  label: string
  colorIndex?: number
  /** One colour per bar (categorical) instead of a single series colour. */
  categorical?: boolean
}

const W = 480
const LABEL_W = 150
const ROW = 30
const clip = (text: string, max = 22) => (text.length > max ? text.slice(0, max - 1) + '…' : text)

export default function HBarChart({ items, format, label, colorIndex = 0, categorical = false }: Props) {
  const tableId = useId()
  const [active, setActive] = useState<string | null>(null)
  if (items.length === 0) return <p className="py-8 text-center text-sm text-on-surface-variant">Chưa có dữ liệu trong kỳ.</p>
  const max = Math.max(...items.map((i) => i.value), 1)
  const barMax = W - LABEL_W - 76
  const height = items.length * ROW + 8
  return (
    <div>
      <svg viewBox={'0 0 ' + W + ' ' + height} className="h-auto w-full" role="group" aria-label={label} aria-describedby={tableId}>
        {items.map((item, i) => {
          const w = Math.max(item.value > 0 ? 2 : 0, (item.value / max) * barMax)
          const y = 4 + i * ROW
          const on = active === item.key
          const fill = categorical ? chartColor(i) : chartColor(colorIndex)
          return (
            <g
              key={item.key}
              tabIndex={0}
              role="img"
              aria-label={item.label + ': ' + format(item.value) + (item.detail ? ', ' + item.detail : '')}
              onMouseEnter={() => setActive(item.key)}
              onMouseLeave={() => setActive(null)}
              onFocus={() => setActive(item.key)}
              onBlur={() => setActive(null)}
              className="outline-none"
            >
              <title>{item.label + ': ' + format(item.value) + (item.detail ? ' · ' + item.detail : '')}</title>
              <rect x={0} y={y - 2} width={W} height={ROW - 2} rx={4} fill={on ? '#f5ece7' : 'transparent'} />
              <text x={4} y={y + 14} fontSize={12} fill="#1f1b18">{clip(item.label)}</text>
              <rect x={LABEL_W} y={y + 3} width={w} height={ROW - 12} rx={3} fill={fill} opacity={active && !on ? 0.5 : 1} />
              <text x={LABEL_W + w + 6} y={y + 14} fontSize={12} fill="#564242" className="tabular-nums">{format(item.value)}</text>
            </g>
          )
        })}
      </svg>
      <table id={tableId} className="sr-only">
        <caption>{label}</caption>
        <tbody>{items.map((i) => <tr key={i.key}><td>{i.label}</td><td>{format(i.value)}</td></tr>)}</tbody>
      </table>
    </div>
  )
}
