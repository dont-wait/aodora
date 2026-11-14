import { Navigate, Route, Routes } from 'react-router-dom'
import Layout from '@/components/layout/Layout'
import CollectionPage from '@/pages/collection/CollectionPage'
import GuidePage from '@/pages/guide/GuidePage'
import HomePage from '@/pages/home/HomePage'
import ProductDetailPage from '@/pages/product-detail/ProductDetailPage'
import AdminCatalogPage from '@/pages/admin/AdminCatalogPage'
import AdminAccessGuard from '@/pages/admin/AdminAccessGuard'
import ForbiddenPage from '@/pages/admin/ForbiddenPage'

export default function App() {
  return (
    <Routes>
      <Route path="403" element={<ForbiddenPage />} />
      <Route path="admin" element={<AdminAccessGuard />}>
        <Route index element={<Navigate to="/admin/catalog" replace />} />
        <Route path="catalog" element={<AdminCatalogPage />} />
        <Route path="*" element={<Navigate to="/403" replace />} />
      </Route>
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
