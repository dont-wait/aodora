import BespokeSection from './components/BespokeSection'
import CategoriesSection from './components/CategoriesSection'
import HeroSection from './components/HeroSection'
import NewCollectionSection from './components/NewCollectionSection'
import PillarsSection from './components/PillarsSection'

export default function HomePage() {
  return (
    <div className="flex flex-col w-full">
      <HeroSection />
      <CategoriesSection />
      <NewCollectionSection />
      <PillarsSection />
      <BespokeSection />
    </div>
  )
}
