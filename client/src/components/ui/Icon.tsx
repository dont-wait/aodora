import type { CSSProperties } from 'react'

interface IconProps {
  /** Tên icon trong bộ Material Symbols Outlined */
  name: string
  size?: number
  filled?: boolean
  className?: string
  style?: CSSProperties
}

export default function Icon({ name, size, filled = false, className = '', style }: IconProps) {
  return (
    <span
      aria-hidden="true"
      className={`material-symbols-outlined ${className}`}
      style={{
        ...(size ? { fontSize: size } : null),
        ...(filled ? { fontVariationSettings: "'FILL' 1" } : null),
        ...style,
      }}
    >
      {name}
    </span>
  )
}
