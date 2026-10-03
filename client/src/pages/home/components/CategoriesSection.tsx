import { Link } from 'react-router-dom'
import Badge from '@/components/ui/Badge'
import EyebrowHeading from '@/components/ui/EyebrowHeading'
import Icon from '@/components/ui/Icon'
import { ROUTES } from '@/data/site'
import { homeCategories } from '@/data/home'

export default function CategoriesSection() {
  return (
    <section className="w-full py-16 lg:py-24 bg-surface-container-low">
      <div className="max-w-[1440px] mx-auto px-margin md:px-margin-tablet lg:px-margin-desktop">
        <div className="flex flex-col md:flex-row md:items-end justify-between mb-12 lg:mb-16 gap-4">
          <EyebrowHeading eyebrow="Phân Loại Dáng Áo" title="Khám Phá Di Sản Áo Dài" titleClassName="text-on-surface" />
          <p className="font-body-md text-body-md text-on-surface-variant max-w-md">
            Mỗi tà áo mang trong mình một khúc ca văn hóa, hòa quyện giữa dáng xưa đài các và nhịp sống thanh tân hiện đại.
          </p>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
          {homeCategories.map((cat) => (
            <Link
              key={cat.name}
              to={ROUTES.collection}
              className="group relative flex flex-col bg-surface shadow-sm hover:shadow-xl transition-all duration-300"
            >
              <div className="relative aspect-[3/4] overflow-hidden bg-surface-container">
                <img
                  src={cat.image.src}
                  alt={cat.image.alt}
                  loading="lazy"
                  className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
                />
                <Badge variant={cat.variant} className="absolute top-4 left-4 px-3 py-1 shadow-sm">
                  {cat.badge}
                </Badge>
              </div>
              <div className="p-6 flex flex-col flex-1 justify-between bg-surface">
                <div>
                  <h3 className="font-headline-sm text-headline-sm text-on-surface group-hover:text-primary transition-colors">{cat.name}</h3>
                  <p className="font-body-sm text-body-sm text-on-surface-variant mt-2 line-clamp-2">{cat.description}</p>
                </div>
                <div className="mt-6 pt-4 flex items-center justify-between bg-surface-container-low p-3 rounded">
                  <span className="font-label-uppercase text-label-uppercase text-secondary">{cat.priceFrom}</span>
                  <Icon name="arrow_forward" size={20} className="text-primary group-hover:translate-x-1 transition-transform" />
                </div>
              </div>
            </Link>
          ))}
        </div>
      </div>
    </section>
  )
}
