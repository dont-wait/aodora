import type { KeyboardEvent } from 'react'
import { DIAGRAM_PHOTO } from '@/data/guide'

// Real photo with the eight measurement points drawn in the photo's own pixel space
// (viewBox = the source crop, 1080 × 2012), so markers stay on the body at any size.

const W = 1080
const H = 2012
const ACTIVE = '#fed488'
const PRIMARY = '#5b0215'

type Marker = { step: number; dot: [number, number]; anchor: [number, number] }

const MARKERS: Marker[] = [
  { step: 1, dot: [70, 1150], anchor: [70, 1150] },
  { step: 2, dot: [1030, 330], anchor: [730, 355] },
  { step: 3, dot: [1030, 565], anchor: [785, 565] },
  { step: 4, dot: [170, 640], anchor: [580, 580] },
  { step: 5, dot: [1030, 770], anchor: [760, 770] },
  { step: 6, dot: [1030, 890], anchor: [790, 890] },
  { step: 7, dot: [170, 420], anchor: [470, 435] },
  { step: 8, dot: [1030, 1350], anchor: [965, 1350] },
]

/** Shapes for one measurement: ellipses for girths, bracketed lines for lengths. */
function shapes(step: number) {
  switch (step) {
    case 1: return <><line x1={70} y1={75} x2={70} y2={1995} /><line x1={40} y1={75} x2={100} y2={75} /><line x1={40} y1={1995} x2={100} y2={1995} /></>
    case 2: return <ellipse cx={655} cy={355} rx={75} ry={18} />
    case 3: return <ellipse cx={645} cy={565} rx={140} ry={26} />
    case 4: return <><polyline points="600,385 580,580" /><line x1={580} y1={580} x2={720} y2={580} /></>
    case 5: return <ellipse cx={650} cy={770} rx={110} ry={20} />
    case 6: return <ellipse cx={650} cy={890} rx={140} ry={24} />
    case 7: return <><line x1={470} y1={435} x2={795} y2={440} /><polyline points="470,435 300,650 585,705" /></>
    case 8: return <><line x1={965} y1={330} x2={965} y2={1960} /><line x1={935} y1={330} x2={995} y2={330} /><line x1={935} y1={1960} x2={995} y2={1960} /></>
    default: return null
  }
}

function Overlay({ step, active }: { step: number; active: boolean }) {
  return (
    <g fill="none" strokeLinecap="round" strokeLinejoin="round" opacity={active ? 1 : 0.55}>
      {/* Dark halo keeps the line readable on both the bright stage lights and the dark background. */}
      <g stroke="rgba(0,0,0,0.55)" strokeWidth={active ? 11 : 6}>{shapes(step)}</g>
      <g stroke={active ? ACTIVE : '#ffffff'} strokeWidth={active ? 6 : 3} strokeDasharray={active ? undefined : '14 10'}>{shapes(step)}</g>
    </g>
  )
}

export default function MeasurementDiagram({ step, titles, onSelect }: { step: number; titles: Record<number, string>; onSelect: (step: number) => void }) {
  const onKey = (event: KeyboardEvent<SVGGElement>, value: number) => {
    if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); onSelect(value) }
  }
  return (
    <figure className="w-full">
      <svg viewBox={`0 0 ${W} ${H}`} className="w-full h-auto block" role="group" aria-label="Sơ đồ 8 vị trí số đo áo dài">
        <title>{DIAGRAM_PHOTO.alt}</title>
        <image href={DIAGRAM_PHOTO.src} x={0} y={0} width={W} height={H} preserveAspectRatio="xMidYMid slice" />
        <Overlay step={step} active />
        {MARKERS.map((m) => {
          const active = m.step === step
          const [x, y] = m.dot
          const leader = m.anchor[0] !== x || m.anchor[1] !== y
          return (
            <g
              key={m.step}
              role="button"
              tabIndex={0}
              aria-label={`Số đo ${m.step}: ${titles[m.step] ?? ''}`}
              aria-pressed={active}
              className="cursor-pointer focus:outline-none [&:focus-visible>circle:last-of-type]:stroke-[#fed488]"
              onClick={() => onSelect(m.step)}
              onKeyDown={(event) => onKey(event, m.step)}
            >
              {leader && <line x1={x} y1={y} x2={m.anchor[0]} y2={m.anchor[1]} stroke="rgba(255,255,255,0.85)" strokeWidth={active ? 4 : 2.5} />}
              {leader && <circle cx={m.anchor[0]} cy={m.anchor[1]} r={9} fill={active ? ACTIVE : '#ffffff'} stroke={PRIMARY} strokeWidth={3} />}
              <circle cx={x} cy={y} r={active ? 38 : 32} fill={active ? PRIMARY : '#ffffff'} stroke={active ? ACTIVE : PRIMARY} strokeWidth={active ? 6 : 4} />
              <text x={x} y={y + 11} textAnchor="middle" fontSize={32} fontWeight={700} fontFamily="Plus Jakarta Sans, sans-serif" fill={active ? '#ffffff' : PRIMARY} style={{ pointerEvents: 'none' }}>{m.step}</text>
            </g>
          )
        })}
      </svg>
      <figcaption className="mt-2 text-[11px] text-on-surface-variant text-right">
        <a href={DIAGRAM_PHOTO.sourceUrl} target="_blank" rel="noreferrer" className="underline">{DIAGRAM_PHOTO.credit}</a>
        {' · '}
        <a href={DIAGRAM_PHOTO.licenseUrl} target="_blank" rel="noreferrer" className="underline">{DIAGRAM_PHOTO.license}</a>
        {' · đã cắt khung'}
      </figcaption>
    </figure>
  )
}
