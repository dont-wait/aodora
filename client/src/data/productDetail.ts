import type { Img } from '@/types'

export interface Review {
  id: string
  initials: string
  author: string
  meta: string
  content: string
  image: Img
}

export interface Accessory {
  id: string
  name: string
  description: string
  price: number
  tag: string
  tagVariant: 'surface' | 'secondaryFixed'
  image: Img
}

export interface InfoTab {
  id: 'story' | 'fabric' | 'craft'
  label: string
}

export const productDetail = {
  slug: 'ao-dai-ngu-than-sen-tram',
  name: 'Áo Dài Ngũ Thân Cổ Lập Lĩnh Lụa Tơ Tằm Sắc Sen Trầm',
  shortName: 'Áo Dài Cổ Phục Ngũ Thân Lụa Tơ Tằm',
  collection: 'Cổ Phục Ngũ Thân Triều Nguyễn',
  sku: 'AS-CP2024-08',
  rating: 5.0,
  reviewCount: 42,
  price: 3450000,
  oldPrice: 4200000,
  discount: 18,
  bundle: 'Áo Dài ngũ thân + Quần Lụa Trắng Ngà + Khăn Vấn Đồng Sắc',
  badge: 'Di Sản Thủ Công',
  mainImage: {
    src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuAMnhJBOiYKQmBlOydjvVsRn6t673R4ocK83TW6KLrSs8yBgH78XTPW-Lo5wBcYl-u-zdFVVKZhceP-4EMSJ49OWmS-ONHKxyJjDUfVlaB98DYbB29FgUoFQZN2QSp257mmKhbrjHG2VAjTZ7GW94vRzfoCQLe5lmqqfNS-SYxF7r_9tiP89SS-paVz9FlaNU_1RThBTVQnGOnQBDoNB7oSZsWzSYqe1jJgOG5wThSUqlSRIKSbChq2Sw',
    alt: 'Áo Dài Cổ Phục Ngũ Thân Lụa Tơ Tằm Sắc Sen Trầm - Mặt trước toàn thân',
  },
  gallery: [
    {
      src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuCmRncaSAx3N_8CdpOPW90aH-gawZpwWzA5IWM7GgWdeQc0UkVV52nhCOMR--zHFWzcO-3NjC5HiJ9JXdAsJ4YP3226algvipPE0w0LVV9WDg0OJgysjo5MS5zI1ou5lE74LEkTdgglwKN1bhgxgFLqcuwBDqzWGt-nDCBXZeXlQHixyfcWtqlswDxxhuE25Bitq8QB896i5Uh-wLt9GgsobgzdDNbZPLTPq1z0CR3oyQgGxIKUBNHLnw',
      alt: 'Toàn thân trước',
    },
    {
      src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuCc9lqcU3jqRDJyEiganYgMtcFUK-edmEegODowgrKZfz9FV4_UJ0TJS1XZ9UBbCv3hEZTCmnnPDfv972h4qfRdt4cQsJQNGJj65eI94AlmJkXbmknpEOgdz91Hge9X8QYMcmyukFVa-w3y1lxk5rXFQn8EPfBuByd4V5_IbUIwqjVPb5sX_sFRzef66zsGiJAjkFrb50AIBeh2kp0Lt19SozKY8faRoFfaw2t9Obhn6evaUYWbJVP9Ig',
      alt: 'Góc nghiêng tà áo thướt tha',
    },
    {
      src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuBX7Yto1_aVSzlyLLiNE7k909SERal7CNnOOjb51UkHpI8sXppREq_nS7zG6v50G1Nlbr5TatGwhrahouflTkP-37sCRYt2ushqBIo9CsGAjwnBSfqBRLJI_a_lrqX7ZpMZiHBy_1na4gAoGKkXTcb4xpjnvPXUlcoRGwenzUR4imnFDocYHHxwZ3Y5vijyF2iBQwZMTFZux-S6bGGDlJp_h5cMU2_L3Fa_cKIUWLB5OChlku7_toTK4Q',
      alt: 'Chi tiết đường may & cúc ngọc trai',
    },
    {
      src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuB9NXLOaXEHiRvb_u5fi3-CKR46ko_3E8DBRVtqqRCgYKF6DEgvH-fGz8RHOgzTpNvtjt0TCXPkbancPXMFLprzmYzdsTUyxE5h53AkNG9YKK9zZAfLhGW19IvMj8WHQCFriCbLvjqL4bsDEYvWxDj-VKNHG_Sg1GU2cek2iF20v8O23SrNYoaIwU60dC1xlE74gdT0Xv9156lQTm6PP9S2Zwg38CjbMlTR6KznOFzwvUmQuvzd6AA_XA',
      alt: 'Cận cảnh thớ lụa dệt hoa văn chìm',
    },
  ] as Img[],
  colors: [
    { name: 'Sắc Sen Trầm (Nguyên Bản)', hex: '#7A1C29' },
    { name: 'Trắng Ngà Tự Nhiên', hex: '#F5ECE7' },
    { name: 'Xanh Thủy Lục', hex: '#2C4A43' },
    { name: 'Vàng Hoàng Yến', hex: '#FED488' },
  ],
  sizes: ['S', 'M', 'L', 'XL', 'XXL'],
  trust: [
    { icon: 'workspace_premium', title: 'Lụa Vạn Phúc Thủ Công', description: '100% Tơ tằm làng nghề dệt tay truyền thống.' },
    { icon: 'verified', title: 'Bảo Hành Chỉ May Trọn Đời', description: 'Miễn phí sửa đổi form dáng tại showroom.' },
    { icon: 'inventory_2', title: 'Đồng Kiểm & Thử Tận Nơi', description: 'Thử đồ ưng ý trước khi hoàn tất thanh toán.' },
  ],
  measurementFields: [
    { id: 'height', label: 'Chiều cao (cm)', placeholder: '162' },
    { id: 'weight', label: 'Cân nặng (kg)', placeholder: '52' },
    { id: 'bust', label: 'Vòng ngực (cm)', placeholder: '86' },
    { id: 'waist', label: 'Vòng eo (cm)', placeholder: '68' },
    { id: 'hips', label: 'Vòng mông (cm)', placeholder: '92' },
    { id: 'waistline', label: 'Hạ eo (cm)', placeholder: '36' },
  ],
  storyImage: {
    src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuBWH16hVOlfbzY77jjNQmXKpHycgQjf5AqJ0pjsPq_jv48rHzpLYGAPepJmCHZK0Df7B7ijmF7SVkWpEfcL7yY2iO7yjr3kAHsvj4AOLoF1wKLjrL3Ty3SnqdZrCkiLRXBHKtlgAIbx_EYMRJkz2ynuRKXRGM7m6G4kXgSnRTGRp3kN2tO7TncCXNaJTetBvrRODQ6Ib00vI_l-FLO6j-LGv94V9d7z5gcz43J8BeDe0du0IVQJxip9Ew',
    alt: 'Bối cảnh di sản văn hoá Áo Dài ngũ thân',
  },
  fabricImage: {
    src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuDja_HHrak2khNb1wGV51PkPUKTMmFOUxnqk_Hxy1-G2yxg64E-IhitjQOPT9oa7at7HUum7BCDhDGOFcp09kgucHq7Ni0bCY_KepNW9JMprj8I4nZXxBaiDILCuVL6PRsOS9h1j7X6Z_EFzvoigqEH3SVtSEyn-W3iVms9Ng2kyXHysoj6OZa2Wm6tuLWt55yMXSDe6TtGUjqOCWN3W60mxM0ulvZQ0M8vpqEODw55lodrJGbRLMHA4Q',
    alt: 'Quy trình ươm tơ dệt lụa thủ công Vạn Phúc',
  },
  craftImage: {
    src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuAaIatTW2Wl9AznxOs-W0h17G0_Rvt50DymSnbRfThcuTHpgCH229MDSYPiMmU0DPRfiWkOHts0_Qqn1GHMBnoJSO0d3-4_IX152mvIk5wSB8jWDAjPcCdmXIoeBfsh46cAtlTSQxe3grFnKkYU7N7qh2fktk90dR8i1Y8A9fpn8TVZ1HwaINCf71isfu8yPTLmJfiMLf3K1Grp4hkI879tY_1yK3MV3ZH25vZYKQdgGIntRJ6zfKn4Gg',
    alt: 'Nghệ nhân may đo thủ công Áo Dài AN SOIE',
  },
}

export const infoTabs: InfoTab[] = [
  { id: 'story', label: 'Câu Chuyện Tác Phẩm' },
  { id: 'fabric', label: 'Chất Liệu & Bảo Quản Lụa' },
  { id: 'craft', label: 'Quy Trình May Đo Thủ Công' },
]

export const fabricCare = [
  {
    icon: 'water_drop',
    title: 'Giặt tay dịu nhẹ',
    description: 'Sử dụng nước lạnh kết hợp dầu gội đầu hoặc dung dịch giặt tơ tằm chuyên dụng. Không vò vắt mạnh.',
  },
  { icon: 'dry_cleaning', title: 'Phơi trong bóng râm mát', description: 'Tránh ánh nắng mặt trời trực tiếp để giữ sắc lụa luôn bền màu như thuở ban đầu.' },
  { icon: 'iron', title: 'Ủi hơi nước nhiệt độ lụa', description: 'Nên ủi khi vải còn ẩm nhẹ ở mặt trái hoặc sử dụng bàn ủi hơi nước đứng.' },
]

export const craftSteps = [
  { title: '1. Lấy Số Đo & Cắt Rập', description: 'Lập bản rập cá nhân hoá dựa trên 8 số đo cơ thể để khử sạch nếp nhăn vùng nách và ngực.' },
  { title: '2. Đường May Giấu Chỉ', description: 'Kỹ thuật cuộn mép tà siêu nhỏ (0.5mm) khâu tay lặn chỉ hoàn toàn vào trong thớ vải tơ.' },
  { title: '3. Cổ Lập Lĩnh Đứng', description: 'Cổ áo dựng form bằng cốt keo lụa cao cấp, êm ái cho làn da cổ, không gây cảm giác cọ rát.' },
  { title: '4. Thử Phom & Hoàn Thiện', description: 'Trang điểm cúc ngọc trai tự nhiên và ủi định hình bằng gối ủi chuyên dụng trước khi đóng hộp.' },
]

export const reviews: Review[] = [
  {
    id: 'r1',
    initials: 'PT',
    author: 'Phương Thảo (Hà Nội)',
    meta: 'Đã mua may đo riêng • 2 tuần trước',
    content:
      'Mình may bộ này để dự lễ tốt nghiệp thạc sĩ và chụp ảnh kỷ niệm tại Hoàng Thành Thăng Long. Chất lụa mát rượi, lên ảnh màu sen trầm rất tôn da và sang trọng. Bạn thợ lấy số đo chuẩn đến từng centimet!',
    image: {
      src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuDVQM3RThuVGpCS_rhq80bV4wy70VfUtHM3Tvkjf0yW_on9X2JaHjgpyskSMFjXW9s9xkxmtKkwVonMzHfAAWBxq-mT3XS5BkAavLgKAvvuU2kFqMfmokb7uDkG8pIokJaVoXU1VNN4Q9kCa9Po8jbnR8y4IiwfpOAwcRJsI7eKkTFS03m8irAPJyQaoKY2Tp8fqMZtWIjaKbzJmGkci6G53q3YCl1f_CAAhQB5nOokcfQzqRWXEaJS3Q',
      alt: 'Ảnh chụp thực tế từ chị Phương Thảo',
    },
  },
  {
    id: 'r2',
    initials: 'MC',
    author: 'Mai Chi (TP. Hồ Chí Minh)',
    meta: 'Đã mua size M • 1 tháng trước',
    content:
      'Từng đường tà may giấu chỉ rất tinh tế, không hề lộ gân may. Mình rất ưng ý chiếc khăn vấn tiệp màu đi kèm, đội lên cực kỳ vừa đầu và không bị đau tai dù diện cả ngày.',
    image: {
      src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuCeFi-xTZzJvCzUvytI6Slt4Thfp4tPM6BCH6IzKw3FWNEJLKsW6d_mxpNI8HwsfFRApOJyWaj3QVyE5hnYLm7zGZQixNsINZ5qkkLTjyGRWVB9Wudlk4ZfU8rpDO13qoZr0OrLxwvf9yWVzWx70MrJZjWK3yMHvkREPWBI8N7t0qOcR_RLpteEg8LMlJEjgq3pRdthX_vjhNZjbVMeD5323Ek29P_cHU28ogDIsrh70dScKHvyod1UIQ',
      alt: 'Ảnh thực tế từ chị Mai Chi',
    },
  },
  {
    id: 'r3',
    initials: 'TT',
    author: 'Thùy Trang (Đà Nẵng)',
    meta: 'Đã mua may đo riêng • 2 tháng trước',
    content:
      'Dịch vụ may đo tận nhà rất chu đáo. Mình ở Đà Nẵng gửi số đo online mà sản phẩm nhận về vừa khít. Cổ áo lập lĩnh đứng phom mà vẫn dễ chịu khi quay đầu cử động.',
    image: {
      src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuBGzXPdXEFj1Wi_XS6goaF_aV0La3LQ4Cp-KTRjM3U-Uqvy2ij4lZN897R_Ak3bsm8KuB9CWBYkRdXGSSFrDZMBBn4ywASGFUtWHVnA--2trfs5g3oOYxPrL6swk7xGpTu8s9rOgncwG_HLIJfuCo9jP0m3DHCd2GYxGLi6ZzrG9V1oXGJgxiAahb1sgDBbzaR6-Mgpp4gys9OCfwVL3h0_dkpPgBB5tM5ZQNohEhXmxEch4SohFzLV_Q',
      alt: 'Ảnh thực tế từ chị Thuỳ Trang',
    },
  },
]

export const accessories: Accessory[] = [
  {
    id: 'a1',
    name: 'Quạt Lụa Tơ Tằm Thêu Tay Cố Đô',
    description: 'Nan trúc uốn tay thủ công, mặt quạt lụa thêu hoa sen chỉ tơ bóng óng ả.',
    price: 350000,
    tag: 'Thêu Tay',
    tagVariant: 'surface',
    image: {
      src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuBMDfj78-t9bpkY2iVoI2xS3nf4MGDlNtOWzDLtlDjy-09tZqif7i6bV6Wi-eBCcVg5E0H_JQInA2lpaqrutH30j0dSWxvhy8Hc5vSgaJDK3TMPFC6ClUzCxDAUlqDOGMNIRuEgzto_NLUxFQmHmqz50ciqevYHdZOQw0ex61H_v8JL-nKNtIdl30EzqhoDBpthw0fIYe_1DbKSR60DFYJWhBHKV-LL7RyQopotiTitJfZJCpy05xQcxQ',
      alt: 'Quạt Lụa Tròn Thêu Tay Hoa Sen',
    },
  },
  {
    id: 'a2',
    name: 'Guốc Mộc Yên Ngựa Quai Nhung',
    description: 'Gỗ thông tuyển chọn đẽo thủ công theo dáng yên ngựa, lót êm chân không gây tiếng vang giòn.',
    price: 420000,
    tag: 'Gỗ Mộc Sơn Son',
    tagVariant: 'surface',
    image: {
      src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuAUbAoIOEwO8xNB4_DDJST8bpSC5PIjlrIiGyOMhvr904ZoFGYlen5ORBQfl660ZC80Fax2677wQk16sgnbyfGVu7pHWv6Sgb4EoK1RP6WRWUZ9ze2KkvXpG5qdKrbk5COmr0zCWq4Fqj53Rzkr4dulz0pPCbFYcP2tdmST8lk212z0ba0z9Yx-YX2-xPEMOb8ID0yMlPLcF5YjDZtv1DJK4sKPUW5mNqhqEB9_uKI4Y97wcp-BZiDhbA',
      alt: 'Guốc Mộc Quai Nhung Sắc Đỏ Rượu',
    },
  },
  {
    id: 'a3',
    name: 'Kiềng Bạc Mạ Vàng Cổ Tròn Quý Phái',
    description: 'Kiềng dẹt chạm khắc chìm hoa văn mây hóa rồng, khóa bấm êm ái ôm sát vòng cổ kiêu sa.',
    price: 1200000,
    tag: 'Bạc 925 Mạ Vàng',
    tagVariant: 'secondaryFixed',
    image: {
      src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuC0k2_FFMfhQhJH5YKLlR_WRmgMi-28EhNhzM4gBHnvYkojMgkiB4FKuWfI7R92D_CBSqBJmY6dakmsTUNLdo9i0rBry-TOH3Bwzd4oG7iITvQgbKsQK3YQgldoXZCew2yLtctaFq3fIfkrCKCtXjjo4YkGuIRd7TaaEqyWqf7eLKMzSVPxBnHcqqfPkHKN1E8ofY2pGian4gUPzHcrUF0XjKdsZzn2iDQs67DuxLMqv_nBSHriYWA99Q',
      alt: 'Vòng Kiềng Bạc Mạ Vàng Cổ Điển',
    },
  },
]

export const sizeTable = {
  headers: ['Size', 'Ngực (cm)', 'Eo (cm)', 'Mông (cm)', 'Dài Áo (cm)', 'Cân Nặng (kg)'],
  rows: [
    ['S', '80 - 84', '64 - 66', '88 - 90', '128', '43 - 48'],
    ['M', '85 - 88', '68 - 72', '91 - 94', '130', '49 - 54'],
    ['L', '89 - 92', '73 - 76', '95 - 98', '132', '55 - 59'],
    ['XL', '93 - 96', '77 - 82', '99 - 104', '134', '60 - 65'],
    ['XXL', '97 - 102', '83 - 88', '105 - 110', '135', '66 - 72'],
  ],
}
