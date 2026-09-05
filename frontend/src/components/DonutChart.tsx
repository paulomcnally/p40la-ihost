interface DonutSegment {
  label: string
  value: number
  color: string
}

interface DonutChartProps {
  segments: DonutSegment[]
  size?: number
  thickness?: number
  centerValue?: string
}

function polarToCartesian(cx: number, cy: number, r: number, angleDeg: number) {
  const rad = ((angleDeg - 90) * Math.PI) / 180
  return { x: cx + r * Math.cos(rad), y: cy + r * Math.sin(rad) }
}

function donutWedge(cx: number, cy: number, outerR: number, innerR: number, startAngle: number, endAngle: number) {
  const startOuter = polarToCartesian(cx, cy, outerR, endAngle)
  const endOuter = polarToCartesian(cx, cy, outerR, startAngle)
  const startInner = polarToCartesian(cx, cy, innerR, endAngle)
  const endInner = polarToCartesian(cx, cy, innerR, startAngle)
  const largeArc = endAngle - startAngle <= 180 ? '0' : '1'
  return [
    `M ${startOuter.x} ${startOuter.y}`,
    `A ${outerR} ${outerR} 0 ${largeArc} 0 ${endOuter.x} ${endOuter.y}`,
    `L ${endInner.x} ${endInner.y}`,
    `A ${innerR} ${innerR} 0 ${largeArc} 1 ${startInner.x} ${startInner.y}`,
    'Z',
  ].join(' ')
}

export default function DonutChart({ segments, size = 96, thickness = 12, centerValue }: DonutChartProps) {
  const total = segments.reduce((s, seg) => s + seg.value, 0)
  const r = (size - thickness) / 2
  const cx = size / 2
  const cy = size / 2
  const innerR = r - thickness / 2
  const outerR = r + thickness / 2

  let acc = 0
  const wedges =
    total > 0
      ? segments.map((seg) => {
          const start = (acc / total) * 360
          acc += seg.value
          const end = (acc / total) * 360
          return { ...seg, startAngle: start, endAngle: end }
        })
      : []

  return (
    <div className="relative shrink-0" style={{ width: size, height: size }}>
      <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} className="block">
        <circle cx={cx} cy={cy} r={r} fill="none" stroke="var(--color-border, #e5e5ea)" strokeWidth={thickness} />
        {wedges.map((seg) => (
          <path
            key={seg.label}
            d={donutWedge(cx, cy, outerR, innerR, seg.startAngle, seg.endAngle)}
            fill={seg.color}
          />
        ))}
      </svg>
      {centerValue !== undefined && (
        <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
          <span className="text-sm font-bold">{centerValue}</span>
        </div>
      )}
    </div>
  )
}