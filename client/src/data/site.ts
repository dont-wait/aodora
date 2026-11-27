export const SITE = {
  name: 'AODORA',
  tagline: 'Áo Dài Di Sản Việt',
  announcement: 'Miễn phí vận chuyển toàn quốc cho đơn hàng từ 1.500.000đ | Dịch vụ may đo chuẩn dáng tận nơi',
  hotline: '0988 888 888',
}

export const ROUTES = {
  home: '/',
  collection: '/bo-suu-tap',
  product: (slug: string) => `/san-pham/${slug}`,
  tailoring: '/dich-vu-may-do',
  login: '/dang-nhap',
  register: '/dang-ky',
  account: '/tai-khoan',
  wishlist: '/yeu-thich',
  cart: '/gio-hang',
} as const

export const NAV_ITEMS = [
  { to: ROUTES.home, label: 'Trang chủ', end: true },
  { to: ROUTES.collection, label: 'Bộ sưu tập' },
  { to: ROUTES.tailoring, label: 'Dịch vụ May đo' },
]

export const FOOTER = {
  about: 'Gìn giữ và tôn vinh cốt cách Áo Dài Việt Nam qua từng tấc lụa tơ tằm thượng hạng dệt thủ công từ làng nghề nghìn năm tuổi.',
  badge: 'Lụa Tơ Tằm Tự Nhiên',
  support: [
    { label: 'Hướng dẫn lấy số đo', to: `${ROUTES.tailoring}#bang-so-do` },
    { label: 'Đặt lịch may đo tận nơi', to: `${ROUTES.tailoring}#dat-lich-tan-noi` },
    { label: 'Câu hỏi thường gặp', to: `${ROUTES.tailoring}#cau-hoi-thuong-gap` },
  ],
  showrooms: [
    { label: 'Địa chỉ', value: '140 Lê Trọng Tấn' },
    { label: 'Giờ mở cửa', value: '09:00 - 21:00 hàng ngày' },
  ],
  /** Chỉ hiển thị khi có href thật. */
  socials: [
    { label: 'Instagram' },
    { label: 'Facebook' },
    { label: 'Pinterest' },
    { label: 'Youtube' },
  ] as { label: string; href?: string }[],
  copyright: '© 2025 AODORA. Bản quyền thuộc về Công ty TNHH Áo Dài Di Sản Việt. Tinh hoa tơ tằm Việt Nam.',
}
