import type { ReactNode } from 'react'

interface Props {
  eyebrow: string
  title: ReactNode
  description?: ReactNode
  align?: 'left' | 'center'
  titleClassName?: string
  className?: string
}

/** Cụm "nhãn nhỏ + tiêu đề + mô tả" lặp lại ở nhiều section. */
export default function EyebrowHeading({ eyebrow, title, description, align = 'left', titleClassName = 'text-primary', className = '' }: Props) {
  return (
    <div className={`space-y-2 ${align === 'center' ? 'text-center mx-auto' : ''} ${className}`}>
      <span className="font-label-uppercase text-label-uppercase text-secondary tracking-widest uppercase">{eyebrow}</span>
      <h2 className={`font-headline-lg text-headline-lg ${titleClassName}`}>{title}</h2>
      {description && <p className="font-body-md text-body-md text-on-surface-variant">{description}</p>}
    </div>
  )
}
