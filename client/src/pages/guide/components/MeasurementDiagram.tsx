import type { KeyboardEvent } from 'react'

// Áo dài measurement diagram drawn in its own coordinate space (viewBox 0 0 240 460),
// so every marker stays on the body part it names at any rendered size.
const C = { primary: '#5b0215', secondary: '#775a19', line: '#897172', body: '#f0e6e1', dress: '#eae1dc', trouser: '#fff8f5', soft: '#dcc0c0' }

type Marker = { step: number; dot: [number, number]; anchor: [number, number] }

// Dots sit in two side columns; a leader line joins each dot to the point it measures.
const MARKERS: Marker[] = [
  { step: 1, dot: [22, 236], anchor: [22, 236] },
  { step: 2, dot: [212, 66], anchor: [129, 66] },
  { step: 3, dot: [212, 122], anchor: [148, 122] },
  { step: 4, dot: [40, 136], anchor: [107, 130] },
  { step: 5, dot: [212, 158], anchor: [140, 158] },
  { step: 6, dot: [212, 196], anchor: [146, 196] },
  { step: 7, dot: [40, 80], anchor: [86, 80] },
  { step: 8, dot: [212, 300], anchor: [182, 300] },
]

function Overlay({ step, active }: { step: number; active: boolean }) {
  const stroke = active ? C.primary : C.soft
  const common = { stroke, strokeWidth: active ? 2 : 1, fill: 'none', strokeDasharray: active ? undefined : '3 3', vectorEffect: 'non-scaling-stroke' as const }
  switch (step) {
    case 1: return <g {...common}><line x1={22} y1={16} x2={22} y2={444} /><line x1={14} y1={16} x2={30} y2={16} /><line x1={14} y1={444} x2={30} y2={444} /></g>
    case 2: return <ellipse cx={120} cy={66} rx={10} ry={3.5} {...common} />
    case 3: return <ellipse cx={120} cy={122} rx={28} ry={6} {...common} />
    case 4: return <g {...common}><line x1={113} y1={70} x2={107} y2={130} /><line x1={107} y1={130} x2={133} y2={130} /></g>
    case 5: return <ellipse cx={120} cy={158} rx={20} ry={5} {...common} />
    case 6: return <g {...common}><ellipse cx={120} cy={196} rx={26} ry={6} /><line x1={100} y1={196} x2={95} y2={206} /><line x1={140} y1={196} x2={145} y2={206} /></g>
    case 7: return <g {...common}><line x1={86} y1={80} x2={154} y2={80} /><line x1={86} y1={80} x2={66} y2={214} /></g>
    case 8: return <g {...common}><line x1={182} y1={64} x2={182} y2={436} /><line x1={176} y1={64} x2={188} y2={64} /><line x1={176} y1={436} x2={188} y2={436} /></g>
    default: return null
  }
}

function Figure() {
  return (
    <g stroke={C.line} strokeWidth={1} strokeLinejoin="round" vectorEffect="non-scaling-stroke">
      {/* Trousers, visible below the slit */}
      <path d="M98 200 L88 438 L116 438 L120 250 L124 438 L152 438 L142 200 Z" fill={C.trouser} />
      {/* Sleeves */}
      <path d="M90 78 L64 212 L74 214 L98 108 Z" fill={C.dress} />
      <path d="M150 78 L176 212 L166 214 L142 108 Z" fill={C.dress} />
      {/* Hands */}
      <ellipse cx={68} cy={220} rx={5} ry={7} fill={C.body} />
      <ellipse cx={172} cy={220} rx={5} ry={7} fill={C.body} />
      {/* Bodice to the slit at the hip, then the front panel (tà) */}
      <path d="M110 70 L90 78 L94 106 L92 124 L100 158 L96 196 L102 206 L104 420 L136 420 L138 206 L144 196 L140 158 L148 124 L146 106 L150 78 L130 70 Z" fill={C.dress} />
      {/* Standing collar */}
      <path d="M110 60 L110 72 Q120 76 130 72 L130 60 Z" fill={C.dress} />
      {/* Neck and head */}
      <rect x={114} y={52} width={12} height={10} fill={C.body} />
      <ellipse cx={120} cy={36} rx={14} ry={17} fill={C.body} />
      <path d="M106 32 Q120 12 134 32 Q132 20 120 18 Q108 20 106 32 Z" fill={C.secondary} opacity={0.55} />
      {/* Feet */}
      <path d="M92 438 L84 446 L104 446 L110 438 Z" fill={C.body} />
      <path d="M130 438 L136 446 L156 446 L148 438 Z" fill={C.body} />
    </g>
  )
}

export default function MeasurementDiagram({ step, titles, onSelect }: { step: number; titles: Record<number, string>; onSelect: (step: number) => void }) {
  const onKey = (event: KeyboardEvent<SVGGElement>, value: number) => {
    if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); onSelect(value) }
  }
  return (
    <svg viewBox="0 0 240 460" className="w-full h-auto" role="group" aria-label="Sơ đồ 8 vị trí số đo áo dài">
      <rect x={0} y={0} width={240} height={460} fill="#f5ece7" />
      <Figure />
      {MARKERS.map((m) => <Overlay key={m.step} step={m.step} active={m.step === step} />)}
      {MARKERS.map((m) => {
        const active = m.step === step
        const [x, y] = m.dot
        return (
          <g
            key={m.step}
            role="button"
            tabIndex={0}
            aria-label={`Số đo ${m.step}: ${titles[m.step] ?? ''}`}
            aria-pressed={active}
            className="cursor-pointer focus:outline-none [&:focus-visible>circle]:stroke-[3]"
            onClick={() => onSelect(m.step)}
            onKeyDown={(event) => onKey(event, m.step)}
          >
            {m.anchor[0] !== x && <line x1={x} y1={y} x2={m.anchor[0]} y2={m.anchor[1]} stroke={active ? C.primary : C.soft} strokeWidth={1} strokeDasharray={active ? undefined : '2 2'} />}
            {m.anchor[0] !== x && <circle cx={m.anchor[0]} cy={m.anchor[1]} r={2.5} fill={active ? C.primary : C.line} />}
            <circle cx={x} cy={y} r={11} fill={active ? C.primary : '#ffffff'} stroke={C.primary} strokeWidth={active ? 2 : 1} />
            <text x={x} y={y + 4} textAnchor="middle" fontSize={11} fontWeight={700} fill={active ? '#ffffff' : C.primary} style={{ pointerEvents: 'none' }}>{m.step}</text>
          </g>
        )
      })}
    </svg>
  )
}
