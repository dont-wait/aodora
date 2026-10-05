import type { Img } from '@/types'

export const heroStats = [
  { value: '100%', label: 'Lụa Tơ Tằm Tự Nhiên' },
  { value: '30+', label: 'Năm Nghệ Nhân May' },
  { value: '15.000+', label: 'Khách Hàng Tin Yêu' },
]

export const heroImages: { main: Img; detail: Img } = {
  main: {
    src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuBGYzE_LsU46KmIT3BetQwQsQjCH2mZAThdnx52hWYUDiRrVmFoIW5AjEJtuxvzuaTZyPlw1y5asudQpWJAssXfq-qK18s7KYdCilmf_xchO0EgmnIsSaK4s0fUShksP2zN4kUdhGLSj4hGDrOT3bt0W4zH5B9mmEXkfPWHAGCzGCoMuVCEkjuvD2h2YL4-707lGP4naTDeI5axXwCRaQ-zM3daduVrYKuFBGpurS1dXCvUqcCM-nHPbQ',
    alt: 'Người phụ nữ Việt mặc áo dài lụa đỏ thẫm giữa sân nhà cổ Huế',
  },
  detail: {
    src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuC04of0E9gUUBhpcVQHfSblEuoH9xqGEc2ZfezwEj66-Q1UdvvLtkEiSX9o9LftIwkwzSLrokQtziXnLxUwJeieLyUbpFC6oVndkKaKIC__tsDsWRopqg2TZ4xQol7TBzDPzb7FsFsf2qqfMPKTIf5mqIvd8iyfgfXaqZjtwzJAyJwzGMWhQI0e32Y-PMQ5sRWp2loV8d1bBcpqkb3g9BpFwXd-eG9FznCrx_miTKEyim7RU7o9fgYJOw',
    alt: 'Cận cảnh họa tiết sen vàng thêu tay trên lụa tơ tằm màu ngà',
  },
}

export interface HomeCategory {
  name: string
  description: string
  priceFrom: string
  badge: string
  variant: 'primary' | 'secondary' | 'surface' | 'primaryContainer'
  image: Img
}

export const homeCategories: HomeCategory[] = [
  {
    name: 'Cổ Phục & Ngũ Thân',
    description: 'Kế thừa chuẩn mực năm tà thanh tao, cúc ngọc trai cài kín đáo tôn vinh đạo lễ gia phong.',
    priceFrom: 'Từ 2.850.000đ',
    badge: 'Di sản phục dựng',
    variant: 'primary',
    image: {
      src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuBQH_3RT2ZdD8yEgR-ydcWfpXcNDgRuUQ4tRxjgXMeBMdSfdfIDLn23nPDNfgxKC7C7-FCMGtBrZ3DOhopiSBCplb3qa-SqgeisnYH5qWqKYCsdzCmafuzg4ppN1h2r32VxINNEbcOP9w6sbtpijFkClQ8RpR8l1rZKdHGqkWyINfsfUSsK1LUuAZup64ciMfPcAqM7rdxVANNnucnUPowkBl6RrmHqSTZqlpi85d1Hj7N78hHzS2zFMQ',
      alt: 'Cổ Phục & Ngũ Thân',
    },
  },
  {
    name: 'Dân Gian & Lễ Hội',
    description: 'Hoa văn thêu tay sen ngát, cúc họa mi và họa tiết mây ngũ sắc trên nền lụa tơ sống.',
    priceFrom: 'Từ 2.200.000đ',
    badge: 'Nghệ thuật dân gian',
    variant: 'secondary',
    image: {
      src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuBz9Pl7vwueBq_JAY69OqQQK-JzEnuSvz7rMdD0NrXES_iazX98DgRryuMBv7CLQW9unl2gjFFgLSCL0LUfVXz3XZDt7HM79AKysGZi142JStkRkQLFYE9EN6ujDmJ3Wl48jzjVxYQyra2g-fy5y5QFm-PftMhwoqxE-GT7LAVcDdsBAx0gA588hzQOJpj0Vssit5nOeZOFBA-tvfR0g0zrAEx0uhqo3l0TGWjzJe9g0qFxk2BeNpFoVQ',
      alt: 'Dân Gian & Lễ Hội',
    },
  },
  {
    name: 'Cách Tân Đương Đại',
    description: 'Cắt cúp tối giản, tay bồng thanh lịch, chất liệu lụa đũi mộc nhẹ tựa sương mai thường nhật.',
    priceFrom: 'Từ 1.650.000đ',
    badge: 'Thời thượng',
    variant: 'surface',
    image: {
      src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuB3rsHX7EKKsFPFTynKRdaozAEiUdajZYYKZ-SMGBOMPJH9UzgWNuDlz0qnhc8I9OGJ3fy_OiBHLvKO1o3RP1hrTu3leS9Fb2GsITAa0HIafg4EpK2gg_xqL86I8ExesXH4XDizYNqI8Onz1fX0QxSiUNw4ezyZrRRqUknA328G3hY4jMHUKcYoBPW7_a6048ViSeLbIoVwy8uFy9-VAXQdXPARQl3041NBrfxeWPiUK86DbhM4rnOfpw',
      alt: 'Cách Tân Đương Đại',
    },
  },
  {
    name: 'Hỷ Sự & Dạ Yến',
    description: 'Gấm thượng phẩm dệt nổi hoa chìm hoàng triều, kết ngọc trai sông thủ công đài các trọn vẹn duyên lành.',
    priceFrom: 'Từ 4.500.000đ',
    badge: 'Đại hỷ & Dạ yến',
    variant: 'primaryContainer',
    image: {
      src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuCDceu-3ynS6hLYTOek7D7bq0TLDohcgy7Xpd2qZfB0UasJzdxhlqTF5j8FSPfVSCBUM_enP9795gn0QMjZgJ0PQQyW-_des3n-h_-TUbaFKKMrtkckAnEo0X4is0KXHR-l-vtPuEGzQHEiFLpGocX11lzUudQTG4Zwq6H6UW5ZCbqfJu048pPuDg2DzWBe85QfEXloHbZSufDKzIp-PsV-AS4KIjurA1YvoSF8tXKYL6B3mweUCZMz8w',
      alt: 'Hỷ Sự & Dạ Yến',
    },
  },
]

export interface FeaturedProduct {
  slug: string
  material: string
  name: string
  description: string
  price: number
  oldPrice?: number
  badge: string
  variant: 'primary' | 'secondary' | 'surface' | 'primaryContainer'
  availability?: string
  image: Img
}

export const featuredProducts: FeaturedProduct[] = [
  {
    slug: 'ngu-than-lua-to-sen-trang',
    material: 'Lụa Vạn Phúc 16 Momme',
    name: 'Ngũ Thân Lụa Tơ Sen Trắng',
    description: 'Kèm vấn lụa tơ tằm đồng màu & cúc khuy xà cừ',
    price: 3450000,
    oldPrice: 3800000,
    badge: 'Độc bản',
    variant: 'primary',
    availability: undefined,
    image: {
      src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuB_E2D89zS5_-AqOOGlU1IWgOCbpLKwwCwNZtOzr4UJ0jcz9ObMJHMmLT9bKrEXFKEjUa1tfuAxm1UJYG_FNnmBIHkMaLFQwjKFWEB60LQc1078T6l4G2isCFkkmeHAuGS2hoVPM1zrLLm6AvRpxSRnk-QXsS_HhlqMXp0gLUvhSxQYhWo9llRlSStfUo_oWHVYA96M49YchXNuosubxL_jQaWnVxPXkvcIQnA323ZgGiJbyfJJv5-_CQ',
      alt: 'Ngũ Thân Lụa Tơ Sen Trắng',
    },
  },
  {
    slug: 'dan-gian-theu-chim-hac-tram',
    material: 'Lụa Tơ Tằm Bảo Lộc',
    name: 'Dân Gian Thêu Chim Hạc Trầm',
    description: 'Kỹ thuật thêu bạt chỉ tơ bóng ngọc trai',
    price: 2890000,
    badge: 'Bán chạy',
    variant: 'secondary',
    availability: 'Có sẵn',
    image: {
      src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuBz67SLMhzwB6ZcNlSGPL3gOmYHnPufhLX901fyFQjsFynJgO3BVsAtFOoaT_7zKi7QI3VzBLDpP7Vr5_hiU-xYkxeDi8mpFrWGcfT4HKi8e5sSrEKxhr82dIoYZZsJoXdf0n7Zl3CyRgsr7D5ql5F9SDhnB001yK_vQPWy07aDSImo8CFHvBjJ39btJSWqar71hC82_JwEiesY2SmJtpKP0KLDZZdEUkkzM4j11UMbCWhXl2xaHxl3tA',
      alt: 'Dân Gian Thêu Chim Hạc Trầm',
    },
  },
  {
    slug: 'cach-tan-co-tru-mong-to',
    material: 'Lụa Hà Đông Phối Organza',
    name: 'Cách Tân Cổ Trụ Mộng Tơ',
    description: 'Gam màu hồng phấn nhã nhặn, tôn sáng làn da',
    price: 1850000,
    badge: 'Mới ra mắt',
    variant: 'surface',
    availability: 'Có sẵn',
    image: {
      src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuBsUYDeD5KTL7sEqmZvSKh2qD4eB_MW3Xq6O68HX7KGI1x0ZNByVWKJ4O852I7cOH0vKhPfRd_z15H8z4TGxRbIADLwtR28g82_gCccBBWUsBmacZ3P-3I9pJzVfiTDjbHE6yxLNFhdtckxcaJRoMEPO0AO9LhfhNXrrQede9tiiMO-W0PTvlCbCyRA4EsepsJ7b2iiy8rJqcBi_qnwiUFSt9awELavaLw5dHx9_VrItFXK6zFxOUAHpg',
      alt: 'Cách Tân Cổ Trụ Mộng Tơ',
    },
  },
  {
    slug: 'gam-hoa-chim-hoang-trieu',
    material: 'Gấm Dệt Sa Tơ Tằm Hoàng Gia',
    name: 'Gấm Hoa Chìm Hoàng Triều',
    description: 'Nút ngọc trai nước ngọt Phú Quốc dệt cẩn xà cừ',
    price: 4200000,
    badge: 'Tuyệt tác',
    variant: 'primaryContainer',
    availability: 'May theo số đo',
    image: {
      src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuC6_s9Gx_kO8Gy1Tg6Ntw67Kn50qYNQMo2WPWnkqox5O-hd56siR9fqD6MTLs8gunQ0-NYYm_kkXyf0U_RfDeiLf4pwoSg1p3SLCzXRh3rjoMGmglQwr-DL9DlGzf0MGTUZN0xo2EDVnIZNxTtMR2Pzs0_OMXmYT-7PWRMoeqPDr4UYUpQoQg5w2IkE9PCk2dy_g2ePcXg09Ossy8TB6lX8p20xqSsjcZenmRxAsRhcOeqBsQO7uXxu6g',
      alt: 'Gấm Hoa Chìm Hoàng Triều',
    },
  },
]

export const pillars = [
  {
    icon: 'nest_eco_leaf',
    title: 'Lụa Vạn Phúc & Bảo Lộc',
    description:
      'Tuyển chọn 100% từ kén tằm tự nhiên vùng cao nguyên đất đỏ và làng lụa Vạn Phúc nghìn năm tuổi. Chất lụa thoáng khí, mềm rủ tự nhiên, mát rượi vào hạ và ấm áp dịu dàng khi gió đông sang.',
    label: 'Dệt Thủ Công',
    image: {
      src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuAmHaFrV0j40kqkbOdphWzsnDomkUsSyfO6Kwm7dHkcGS5SLk3RYa-1T8I2GccPIIBxtsFSj9g7LKFky6YJ0ZhF7NXPoSTLfcWcKxz7LfP_qfUXjn78bqbNbsQAmlegQob25lNCgXsHtJzy0M188Gg15O6ZTzq1b7y9UAZZo5UXbq8lqC9SbDWyZcCEDcKHs_weDF7ooWEdyItRy8QIUod-PAGMrdPyohtBV2k7Ht7GbBH3UqC23fC5hg',
      alt: 'Lụa Vạn Phúc & Bảo Lộc',
    },
  },
  {
    icon: 'architecture',
    title: 'Đôi Tay Nghệ Nhân Đất Bắc',
    description:
      'Từng cánh sen, đóa cúc hay cánh hạc bồng bềnh đều được thêu tay hoàn toàn bởi các nghệ nhân thêu làng Quất Động với kỹ thuật đâm xô, thêu bạt và kim tuyến kéo dài từ 40 đến 80 giờ lao động tận tâm.',
    label: 'Thêu Tay Tỉ Mỉ',
    image: {
      src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuBG4fFpP4YXulYMC8RyTw8TdQt4aNIozyLntURcUr-G-NZX-tIzhSwc535DAz3k73UYGemAHK4AFdDf5CGNdCcbNYbzqv2k81wq5V7NDvmFhso2H_SM_5TGIsnEjZI4-BO_cRYDm23Bq2wMpSUktP7iNqWXCrEOOsvAUEeW9xfwCq_yoF8-5zV-43MpsWublcX0gKrySxEG9VKAVvqy7UZNdfFprQB7NsQ-Mn736JzBr6wGIQw5eUQyEg',
      alt: 'Đôi Tay Nghệ Nhân Đất Bắc',
    },
  },
  {
    icon: 'content_cut',
    title: 'Kỹ Thuật Đóng Tà Chuẩn Việt',
    description:
      'Đường cắt lượn eo tinh tế theo 18 chỉ số đo nhân trắc học cá nhân. Kỹ thuật ép keo lụa viền tà giữ cho hai tà áo luôn buông thẳng mềm mại, bay bổng trong từng bước đi mà không hề nhăn rúm hay co giật.',
    label: 'Phom Dáng Vừa Vặn',
    image: {
      src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuAuPGb737I6R_HYsTzsbT2G8rmyjEQbSHVOS-ZwzfjiXMpvLNLiuc_TWw2vUxg7npSaW_d2c9BzeTpD6nOcQjkLqWOxm63X2z33bwHLVy_bhdCJbLMbazDZOOKDhX9BfmcZb_a1suHA-bJEaJpLVE0xN26Evm1qBpg2kxqf1oOVdYQ24OY8X6BjO_qKfC98ClsYsB-ir3XXURjfBKscK1OGUiUjvh7TjfvRcQpJ2anVbYKhFcrVJ3p7GQ',
      alt: 'Kỹ Thuật Đóng Tà Chuẩn Việt',
    },
  },
]

export const heroTestimonial = {
  quote:
    'Lần đầu mặc chiếc Áo Dài Ngũ Thân của AN SOIE trong buổi thuyết trình văn hóa quốc tế tại Paris, tôi cảm nhận được sự trang nghiêm, tự hào khôn tả. Từng thớ lụa ôm lấy vóc dáng rất tự nhiên, vừa vặn tuyệt đối.',
  name: 'TS. Mai Lan Hương',
  role: 'Nhà nghiên cứu Mỹ thuật & Di sản Dân gian',
  avatar: {
    src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuCmkJ56U3IuEJE-No6j9ADnHnSRcCtzqO6UNXTLgfxKDcf54eNEDIodF4vUeC9WrHm7cOFj7g8hM6JoNJQApKJxH8TXdfipsVq-R6xBMoIfil9Lu7XpU0wVoFn02U49jYMT3_1NYe_mJa5Rtsc6rzxbgKh2im6OExRBCKqSvYNVm_uEla9QEBh4vGpUzxSh9vfx2vBkGGklzRisTvOWE4a2iSKSiv0FdlvHldl8Djr68RsxgD9xdyHqFQ',
    alt: 'Chân dung TS. Mai Lan Hương',
  },
}

export const testimonials = [
  {
    content:
      'Bộ áo dài lễ cưới bằng gấm sa thêu chim phụng khiến ngày trọng đại của tôi thêm phần linh thiêng. Dịch vụ may đo tận nơi của nhà may rất chuyên nghiệp, chăm chút từng chiếc cúc khuy.',
    name: 'Nguyễn Thu Thảo',
    role: 'Cô Dâu Tháng 10, Hà Nội',
    tag: 'Áo Dài Cưới Gấm',
  },
  {
    content:
      'Chất lụa tơ tằm Bảo Lộc của AN SOIE sờ vào như mây mát, mặc cả ngày đi dạy học vẫn phẳng phiu, không bị dính vào người. Tôi đã đặt thêm 3 bộ cách tân để diện hàng ngày.',
    name: 'Trần Ngọc Bích',
    role: 'Giảng viên Ngôn ngữ học, TP.HCM',
    tag: 'Áo Dài Cách Tân',
  },
]

export const wideTestimonial = {
  content:
    'Đặt may đo từ xa mà vừa in như in! Đội ngũ tư vấn hướng dẫn lấy số đo qua video cực kỳ chi tiết. Đóng gói hộp lụa chỉn chu, mở ra thoảng hương hoa nhài rất tinh tế và hoài niệm.',
  name: 'Phạm Khánh Linh',
  role: 'Kiều bào tại Tokyo, Nhật Bản',
  avatar: {
    src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuBPpf1S4BXHOVrnuOKhHRKLgytu8-3bMXG0nnYdizsp2Xpmtx5VFjhV9fDM4izkTFTQ8ECh7yzPqbbipNFT2i8kIGM5PpbjWSeD98-y27MlE91DzcVe5g4Gw-6FTPLFobeOAtBKxOhNF3R9wPnxORpyADUzf4uzdDfPeS4_MBDqLOHHw3lISlg7grS0zHhyl-_gLdrQk0mUaZhPXuhHjBMYX2Rg2g1kx9xsxRKJXht59gCpT1JnKsoodw',
    alt: 'Chân dung Phạm Khánh Linh',
  },
}

export const bespokePerks = [
  { icon: 'workspace_premium', title: 'Chỉnh sửa miễn phí', description: 'Đến khi quý khách hoàn toàn hài lòng về phom dáng.' },
  { icon: 'history_edu', title: 'Bảo hành chỉ thêu trọn đời', description: 'Đảm bảo vẻ đẹp trường tồn theo năm tháng.' },
]

export const bespokeCities = ['Hà Nội', 'TP. Hồ Chí Minh', 'Tỉnh thành khác']

export const bespokeInterests = ['Áo Dài Cổ Phục / Ngũ Thân', 'Áo Dài Lễ Cưới & Đại Hỷ', 'Áo Dài Dân Gian Thêu Tay', 'Áo Dài Cách Tân Đi Làm / Dạo Phố']
