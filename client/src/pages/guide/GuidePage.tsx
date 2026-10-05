import FaqSection from './components/FaqSection'
import GuideHero from './components/GuideHero'
import HeritageSection from './components/HeritageSection'
import HomeVisitSection from './components/HomeVisitSection'
import MeasurementSection from './components/MeasurementSection'
import PreparationSection from './components/PreparationSection'

export default function GuidePage() {
  return (
    <div className="flex flex-col w-full">
      <GuideHero />
      <PreparationSection />
      <MeasurementSection />
      <HeritageSection />
      <HomeVisitSection />
      <FaqSection />
    </div>
  )
}
