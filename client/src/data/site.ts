export const SITE = {
  name: 'AN SOIE',
  tagline: 'Áo Dài Di Sản Việt',
  logo: 'https://lh3.googleusercontent.com/aida/AEtjO1X-9KiMTRhVLlzblCutvzRwTbzWOewmGAtJ1Zc0vCBtBNp5ji7K6JZovhmRKm9FyRstKHBx2ZmVqPsGLOkUTZTGlMPOJeHBmC1Ra2FOM3tczDNO6vprpX09CuoSnxmj__Auy9YZJDfc59KWsuWdP8QTcVkDOPW5cbOj-QJkAs55nCzz3DDmdDMjY30Mn17EL8Afm5jaX1hGt39c1DplyWJR_NgQInYA3D70RdxAZN3q689gk5w4gaom8dXp',
  announcement: 'Miễn phí vận chuyển toàn quốc cho đơn hàng từ 1.500.000đ | Dịch vụ may đo chuẩn dáng tận nơi',
  hotline: '0988 888 888',
}

export const ROUTES = {
  home: '/',
  collection: '/bo-suu-tap',
  product: (slug: string) => `/san-pham/${slug}`,
  tailoring: '/dich-vu-may-do',
} as const

export const NAV_ITEMS = [
  { to: ROUTES.home, label: 'Trang chủ', end: true },
  { to: ROUTES.collection, label: 'Bộ sưu tập' },
  { to: ROUTES.tailoring, label: 'Dịch vụ May đo' },
]

export const FOOTER = {
  about: 'Gìn giữ và tôn vinh cốt cách Áo Dài Việt Nam qua từng tấc lụa tơ tằm thượng hạng dệt thủ công từ làng nghề nghìn năm tuổi.',
  badge: 'Lụa Tơ Tằm Tự Nhiên',
  support: ['Hướng dẫn lấy số đo', 'Chính sách may đo bespoke', 'Bảo hành chỉ may trọn đời', 'Đổi trả & Giao hàng tận nơi', 'Câu hỏi thường gặp'],
  showrooms: [
    { label: 'Hà Nội', value: 'Làng Lụa Vạn Phúc, Hà Đông & 18 Tràng Tiền, Hoàn Kiếm' },
    { label: 'TP. Hồ Chí Minh', value: '68 Đồng Khởi, Quận 1' },
    { label: 'Hotline', value: '0988 888 888' },
    { label: 'Giờ mở cửa', value: '09:00 - 21:00 hàng ngày' },
  ],
  socials: ['Instagram', 'Facebook', 'Pinterest', 'Youtube'],
  copyright: '© 2025 AN SOIE. Bản quyền thuộc về Công ty TNHH Áo Dài Di Sản Việt. Tinh hoa tơ tằm Việt Nam.',
}
