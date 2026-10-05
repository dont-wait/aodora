import Icon from './Icon'

export default function StarRating({ value = 5, size = 18 }: { value?: number; size?: number }) {
  return (
    <div className="flex items-center text-secondary" role="img" aria-label={`${value} trên 5 sao`}>
      {Array.from({ length: value }, (_, i) => (
        <Icon key={i} name="star" size={size} filled />
      ))}
    </div>
  )
}
