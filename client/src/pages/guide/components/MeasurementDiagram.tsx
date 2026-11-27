import type { KeyboardEvent } from 'react'

// Áo dài measurement diagram drawn in its own coordinate space (viewBox 0 0 240 460),
// so every marker stays on the body part it names at any rendered size.
const C = { primary: '#5b0215', secondary: '#a4753a', line: '#897172', outline: '#8d6e6f', skin: '#f2ddd0', hair: '#3a2723', shoe: '#5b0215', trouser: '#fffaf7', soft: '#c9a9a7' }

type Marker = { step: number; dot: [number, number]; anchor: [number, number] }

// Dots sit in two side columns; a leader line joins each dot to the point it measures.
const MARKERS: Marker[] = [
  { step: 1, dot: [22, 236], anchor: [22, 236] },
  { step: 2, dot: [212, 62], anchor: [130, 62] },
  { step: 3, dot: [212, 122], anchor: [148, 122] },
  { step: 4, dot: [40, 140], anchor: [108, 128] },
  { step: 5, dot: [212, 158], anchor: [137, 158] },
  { step: 6, dot: [212, 192], anchor: [141, 192] },
  { step: 7, dot: [40, 80], anchor: [92, 80] },
  { step: 8, dot: [212, 300], anchor: [186, 300] },
]

function Overlay({ step, active }: { step: number; active: boolean }) {
  const stroke = active ? C.primary : C.soft
  const common = { stroke, strokeWidth: active ? 2 : 1, fill: 'none', strokeDasharray: active ? undefined : '3 3', opacity: active ? 1 : 0.45, vectorEffect: 'non-scaling-stroke' as const }
  switch (step) {
    case 1: return <g {...common}><line x1={22} y1={21} x2={22} y2={449} /><line x1={14} y1={21} x2={30} y2={21} /><line x1={14} y1={449} x2={30} y2={449} /></g>
    case 2: return <ellipse cx={120} cy={62} rx={10} ry={3} {...common} />
    case 3: return <ellipse cx={120} cy={122} rx={28} ry={6} {...common} />
    case 4: return <g {...common}><line x1={113} y1={66} x2={108} y2={128} /><line x1={108} y1={128} x2={132} y2={128} /></g>
    case 5: return <ellipse cx={120} cy={158} rx={17.5} ry={4.5} {...common} />
    case 6: return <g {...common}><ellipse cx={120} cy={192} rx={22} ry={5.5} /><line x1={98} y1={196} x2={101} y2={204} /><line x1={142} y1={196} x2={139} y2={204} /></g>
    case 7: return <g {...common}><line x1={92} y1={80} x2={148} y2={80} /><line x1={92} y1={80} x2={76} y2={212} /></g>
    case 8: return <g {...common}><line x1={186} y1={60} x2={186} y2={432} /><line x1={180} y1={60} x2={192} y2={60} /><line x1={180} y1={432} x2={192} y2={432} /></g>
    default: return null
  }
}

function Flower({ x, y, r }: { x: number; y: number; r: number }) {
  return (
    <g fill={C.secondary} opacity={0.28}>
      {[0, 72, 144, 216, 288].map((a) => <circle key={a} cx={x + Math.cos((a * Math.PI) / 180) * r} cy={y + Math.sin((a * Math.PI) / 180) * r} r={r * 0.75} />)}
      <circle cx={x} cy={y} r={r * 0.55} fill={C.primary} opacity={0.5} />
    </g>
  )
}

function Figure() {
  const outline = { stroke: C.outline, strokeWidth: 0.9, strokeLinejoin: 'round' as const, strokeLinecap: 'round' as const }
  return (
    <g>
      <defs>
        <linearGradient id="md-dress" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#f2dcd7" />
          <stop offset="1" stopColor="#e3c3bd" />
        </linearGradient>
        <linearGradient id="md-back" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#dcbab4" />
          <stop offset="1" stopColor="#cfa9a2" />
        </linearGradient>
      </defs>
      {/* Long hair falling behind the shoulders */}
      <path d="M106 36 C101 60 100 100 104 132 L136 132 C140 100 139 60 134 36 Z" fill={C.hair} />
      {/* Back panel (tà sau), peeking out past the trousers */}
      <path d="M98 196 C93 262 85 352 78 432 Q120 443 162 432 C155 352 147 262 142 196 Z" fill="url(#md-back)" {...outline} />
      {/* Wide silk trousers */}
      <path d="M100 200 C97 300 92 380 85 443 L117 443 C118 380 119 300 120 252 C121 300 122 380 123 443 L155 443 C148 380 143 300 140 200 Z" fill={C.trouser} stroke="#c9b6b3" strokeWidth={0.9} strokeLinejoin="round" />
      <path d="M105 300 C103 360 100 402 97 441 M135 300 C137 360 140 402 143 441" stroke="#e2d4d1" strokeWidth={0.8} fill="none" />
      <ellipse cx={102} cy={447} rx={9} ry={2.4} fill={C.shoe} />
      <ellipse cx={138} cy={447} rx={9} ry={2.4} fill={C.shoe} />
      {/* Neck */}
      <path d="M115 50 L115 62 L125 62 L125 50 Z" fill={C.skin} />
      {/* Sleeves and hands */}
      <path d="M92 80 C86 104 82 150 77 210 L86 212 C89 170 92 130 96 106 Z" fill="url(#md-dress)" {...outline} />
      <path d="M148 80 C154 104 158 150 163 210 L154 212 C151 170 148 130 144 106 Z" fill="url(#md-dress)" {...outline} />
      <path d="M78 210 C76 216 76 224 79 230 C80 233 83 233 83 230 L83 224 C85 222 86 218 85 212 Z" fill={C.skin} {...outline} />
      <path d="M162 210 C164 216 164 224 161 230 C160 233 157 233 157 230 L157 224 C155 222 154 218 155 212 Z" fill={C.skin} {...outline} />
      {/* Bodice flowing into the front panel (tà trước) */}
      <path
        d="M111 64 C103 68 96 72 92 80 C94 90 95 98 95 106 C92 114 92 124 96 132 C100 142 103 150 103 160 C102 172 99 184 98 196 L101 204 C100 270 97 350 93 428 Q120 436 147 428 C143 350 140 270 139 204 L142 196 C141 184 138 172 137 160 C137 150 140 142 144 132 C148 124 148 114 145 106 C145 98 146 90 148 80 C144 72 137 68 129 64 Z"
        fill="url(#md-dress)" {...outline}
      />
      {/* Diagonal opening from collar to underarm, with fabric buttons */}
      <path d="M127 67 C134 75 141 89 145 104" stroke={C.outline} strokeWidth={0.8} fill="none" />
      {[[131, 72], [136, 80], [140, 89]].map(([x, y]) => <circle key={y} cx={x} cy={y} r={1.1} fill={C.outline} />)}
      <Flower x={112} y={118} r={2.2} />
      <Flower x={124} y={300} r={3} />
      <Flower x={112} y={340} r={2.4} />
      <Flower x={128} y={378} r={3.2} />
      <Flower x={115} y={408} r={2.6} />
      {/* Mandarin collar */}
      <path d="M110 58 C110 55 130 55 130 58 L130 66 C124 69 116 69 110 66 Z" fill="url(#md-dress)" {...outline} />
      {/* Head with side-swept hair */}
      <ellipse cx={120} cy={38} rx={12} ry={15} fill={C.skin} {...outline} />
      <path d="M107 40 C104 25 112 21 120 21 C129 21 136 26 133 42 C131 34 128 30 123 28 C119 31 112 32 107 40 Z" fill={C.hair} />
      {/* Soft facial features */}
      <path d="M112.5 39 Q115 37.8 117.5 39 M122.5 39 Q125 37.8 127.5 39" stroke={C.hair} strokeWidth={0.7} fill="none" strokeLinecap="round" />
      <path d="M116.5 46.5 Q120 48.6 123.5 46.5" stroke="#b5575f" strokeWidth={1.1} fill="none" strokeLinecap="round" />
      <circle cx={113.5} cy={43} r={1.8} fill="#e8a9a4" opacity={0.35} />
      <circle cx={126.5} cy={43} r={1.8} fill="#e8a9a4" opacity={0.35} />
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
