import { Link } from 'react-router-dom'

export interface Crumb {
  label: string
  to?: string
}

export default function Breadcrumb({
  items,
  className = '',
  separatorClass = 'text-outline-variant font-light',
}: {
  items: Crumb[]
  className?: string
  separatorClass?: string
}) {
  return (
    <nav aria-label="Breadcrumb" className={`flex items-center flex-wrap gap-2 ${className}`}>
      {items.map((item, i) => (
        <span key={item.label} className="flex items-center gap-2">
          {i > 0 && <span className={separatorClass}>/</span>}
          {item.to ? (
            <Link className="hover:text-primary transition-colors" to={item.to}>
              {item.label}
            </Link>
          ) : (
            <span className="text-primary font-semibold">{item.label}</span>
          )}
        </span>
      ))}
    </nav>
  )
}
