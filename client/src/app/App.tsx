import { Route, Routes } from 'react-router-dom'
import Layout from '@/components/layout/Layout'
import CollectionPage from '@/pages/collection/CollectionPage'
import GuidePage from '@/pages/guide/GuidePage'
import HomePage from '@/pages/home/HomePage'
import ProductDetailPage from '@/pages/product-detail/ProductDetailPage'

export default function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<HomePage />} />
        <Route path="bo-suu-tap" element={<CollectionPage />} />
        <Route path="san-pham/:slug" element={<ProductDetailPage />} />
        <Route path="dich-vu-may-do" element={<GuidePage />} />
        <Route path="*" element={<HomePage />} />
      </Route>
    </Routes>
  )
}
