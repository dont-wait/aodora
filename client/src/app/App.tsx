import { Navigate, Route, Routes } from 'react-router-dom'
import Layout from '@/components/layout/Layout'
import CollectionPage from '@/pages/collection/CollectionPage'
import GuidePage from '@/pages/guide/GuidePage'
import HomePage from '@/pages/home/HomePage'
import NotFoundPage from '@/pages/NotFoundPage'
import ProductDetailPage from '@/pages/product-detail/ProductDetailPage'
import AdminCatalogPage from '@/pages/admin/AdminCatalogPage'
import AdminAccessGuard from '@/pages/admin/AdminAccessGuard'
import AdminOrdersPage from '@/pages/admin/AdminOrdersPage'
import AdminOrderDetailPage from '@/pages/admin/AdminOrderDetailPage'
import AdminTailoringPage from '@/pages/admin/AdminTailoringPage'
import AdminTailoringDetailPage from '@/pages/admin/AdminTailoringDetailPage'
import ForbiddenPage from '@/pages/admin/ForbiddenPage'
import LoginPage from '@/pages/auth/LoginPage'
import RegisterPage from '@/pages/auth/RegisterPage'
import AccountAccessGuard from '@/pages/account/AccountAccessGuard'
import AccountPage from '@/pages/account/AccountPage'
import WishlistPage from '@/pages/wishlist/WishlistPage'
import CartPage from '@/pages/cart/CartPage'
import CheckoutPage from '@/pages/checkout/CheckoutPage'
import CheckoutResultPage from '@/pages/checkout/CheckoutResultPage'
import MyOrdersPage from '@/pages/orders/MyOrdersPage'
import OrderDetailPage from '@/pages/orders/OrderDetailPage'

export default function App() {
  return (
    <Routes>
      <Route path="403" element={<ForbiddenPage />} />
      <Route path="admin" element={<AdminAccessGuard />}>
        <Route index element={<Navigate to="/admin/catalog" replace />} />
        <Route path="catalog" element={<AdminCatalogPage />} />
        <Route path="orders" element={<AdminOrdersPage />} />
        <Route path="orders/:id" element={<AdminOrderDetailPage />} />
        <Route path="tailoring" element={<AdminTailoringPage />} />
        <Route path="tailoring/:id" element={<AdminTailoringDetailPage />} />
        <Route path="*" element={<Navigate to="/403" replace />} />
      </Route>
      <Route element={<Layout />}>
        <Route path="dang-nhap" element={<LoginPage />} />
        <Route path="dang-ky" element={<RegisterPage />} />
        <Route path="gio-hang" element={<CartPage />} />
        <Route path="thanh-toan" element={<CheckoutPage />} />
        <Route path="dat-hang-thanh-cong/:id" element={<CheckoutResultPage />} />
        <Route path="don-hang/:id" element={<OrderDetailPage />} />
        <Route element={<AccountAccessGuard />}>
          <Route path="tai-khoan" element={<AccountPage />} />
          <Route path="tai-khoan/don-hang" element={<MyOrdersPage />} />
          <Route path="yeu-thich" element={<WishlistPage />} />
        </Route>
        <Route index element={<HomePage />} />
        <Route path="bo-suu-tap" element={<CollectionPage />} />
        <Route path="san-pham/:slug" element={<ProductDetailPage />} />
        <Route path="dich-vu-may-do" element={<GuidePage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  )
}
