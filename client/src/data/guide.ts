import type { Img } from '@/types'

export const guideImages: Record<'hero' | 'diagram' | 'heritageA' | 'heritageB' | 'artisan', Img> = {
  hero: {
    src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuCOn-rIu_1H7CHVcSsHlh4NXSJyoiuG_mSYDBYy6thPr89HO8AQI4TPzSlINMhbPSbRO3cEKGgr9y-tb3SfxKikTHiGwgq1ZQvz5PlVAseiTo_5YB4xDkPZWrUVSqfwRpGGAp6QIbOCpUsY2Ixg7g7ecZ69q6GuOWpj3qk04-8FJGiFy3ojFSVR9uqIFZY_THuWyfUDEfNRUEZCxmT3wrAUM4eMt5cIb0RJ8CzJu9SssuJTPnig8FAmSQ',
    alt: 'Nghệ nhân may đo áo dài tại xưởng AN SOIE',
  },
  diagram: {
    src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuBZMLJX6H_5jrVjRAlxFtNhucwY5rYXzlKtM-vBuWv-hZadQGHbq4IncjtMeMC_aMmbxTLiR_Fol3mgXR0r5Mv2XqjIpEEBE6TIJ6Y4Jh8QFOXsNznaOsggoCf2KzDWsYEULzGIpDZ0MTovqROPkeZi0qUW8sYLwV5IPEGmIGLIALfI7aEduU4EUjqOg9z_P42h7E6fZHzAbi9Wp7uSOjhN2_xT4tyJ1LJq6Xtlg9rZ-v5FupVrlCgvQw',
    alt: 'Hình minh hoạ 8 vị trí số đo trên áo dài',
  },
  heritageA: {
    src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuBkyJDC1td4phZh7SvYNR45nlQggIuaeh2BsJe1S2NOuUq9Un5a22HqVFVp32PUx0VnuV0SOdSyYlHutKlAWTe_rUFJHjOp3MnsFFjY5tHTj9hJbOA3xqAMsdO31dALb6T7s7FQoFq2UfK09NOeD-xdEXvTek5ESicxf-ctmuPj7BmsJJjZOlV0cFTpTWjmM769VzZm551p5XW6CWFlGoEDFCtPy14ybQdSs3cQxshTyJUu2xfh62Ctsg',
    alt: 'Nghệ nhân lão luyện bên khung cửi lụa',
  },
  heritageB: {
    src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuCgUVbaC_g2bJv-SP8Zxm8qmzyDLiR4y8VhQ25buhK5hDuSikkz9-6KpsNU3o28Hkxl2oduSb4GJHe3nl7YjJi7JuZRz1Lpt-kf8gceWUCWvjUzkjps1WPbMNc0pWrJi8P-iyiuYTCYrjz-2sPl2O6AcA-qoSH2ousMSKFiilbiDUlQFuER49WhGopQcFlCFHw2-noAyxiSvT0s44gkR3dLy_eLV24DXyE6B239MpPIO9yURVCBug8mxQ',
    alt: 'Nghệ nhân thêu tay áo dài',
  },
  artisan: {
    src: 'https://lh3.googleusercontent.com/aida-public/AB6AXuC_JY5xDZhKRLZ28vx8YyK4rd0H6-M2XRmNNH3s0ZagvMrJLXW2lUBibyHXL5xBDQS0OM3JQCZOrNadL8sBwW5_YYJU9lhkMnE3nVV0sgYREwJGPG5N_2qNEa-2DmXwlq6gOSX4XY6jABnpJvijoOo7KU0QVlGA0d7sbCI6VaEsZ8DftIixAC-zkwsOZl8QhebjoQIdLi1rBnHsFFcipeYTnCMje8NTuKyef2hzhx3Y-iMHTqGcgiuRWA',
    alt: 'Nghệ nhân Ưu tú Nguyễn Thị Thương',
  },
}

export interface Measurement {
  step: number
  title: string
  icon: string
  /** Vị trí chấm số trên hình minh hoạ (class Tailwind) */
  dotPosition: string
  indicator: string
  focusName: string
  focusDesc: string
  guide: string
}

export const measurements: Measurement[] = [
  {
    step: 1,
    title: 'Chiều cao & Cân nặng',
    icon: 'tune',
    dotPosition: 'top-[14%] left-1/2 -translate-x-1/2',
    indicator: 'Thông số 1/8',
    focusName: '1. Chiều Cao & Cân Nặng Thực Tế',
    focusDesc: 'Xác định tỉ lệ nhân trắc tổng thể để căn chỉnh độ rơi của tà áo và độ mở của quần lụa thướt tha.',
    guide: 'Cho thợ biết dáng người (cao gầy, quả lê hay đồng hồ cát). Giúp tính chính xác độ hạ eo và độ dốc của tà lụa khi đứng hay chuyển động.',
  },
  {
    step: 2,
    title: 'Vòng cổ',
    icon: 'straighten',
    dotPosition: 'top-[19%] left-1/2 -translate-x-1/2',
    indicator: 'Thông số 2/8',
    focusName: '2. Vòng Cổ (Neck)',
    focusDesc: 'Đo sát chân cổ trên nền vải mềm để tạo dáng cổ đứng truyền thống 2cm - 3.5cm êm ái, thanh thoát.',
    guide:
      'Quấn thước quanh chân cổ nơi tiếp giáp với vai, để chừa 1 ngón tay lách vừa vào trong thước. Đảm bảo cổ áo đứng êm ái, thanh thoát mà không thít khó thở.',
  },
  {
    step: 3,
    title: 'Vòng ngực (Bust)',
    icon: 'straighten',
    dotPosition: 'top-[28%] left-1/2 -translate-x-1/2',
    indicator: 'Thông số 3/8',
    focusName: '3. Vòng Ngực (Bust)',
    focusDesc: 'Đo ngang đỉnh ngực cao nhất. Thước giữ phẳng ngang lưng để áo ôm tôn đường cong mà không bức bối.',
    guide:
      'Đo vòng quanh phần nở nhất của ngực, giữ thước dây ngang bằng phẳng phía sau lưng. Không siết quá chặt, thước áp nhẹ và trượt thoải mái trên cúp áo.',
  },
  {
    step: 4,
    title: 'Hạ ngực & Dang ngực',
    icon: 'straighten',
    dotPosition: 'top-[34%] left-1/2 -translate-x-1/2',
    indicator: 'Thông số 4/8',
    focusName: '4. Hạ Ngực & Dang Ngực',
    focusDesc: 'Căn chuẩn tâm điểm bầu ngực để cắt đường ben sườn và chích eo mượt mà, không bao giờ bị nhăn nhúm.',
    guide:
      '**Hạ ngực:** Đo từ chân cổ vai đến đỉnh ngực cao nhất. **Dang ngực:** Đo khoảng cách ngang giữa 2 đầu ngực để đặt đường chiết ben chuẩn xác, tôn bầu ngực tự nhiên.',
  },
  {
    step: 5,
    title: 'Vòng eo con kiến',
    icon: 'straighten',
    dotPosition: 'top-[43%] left-1/2 -translate-x-1/2',
    indicator: 'Thông số 5/8',
    focusName: '5. Vòng Eo Thắt (Waist)',
    focusDesc: 'Đo eo trên rốn 2-3cm để tà áo dài tạo cảm giác chân dài miên man và đường cong eo con kiến quyến rũ.',
    guide:
      'Đo điểm thắt nhỏ nhất của vòng bụng (thường cách trên rốn khoảng 2-3cm). Đây là then chốt để tà áo dài lượn cong ôm lấy eo mà vẫn cho phép quý cô hít thở khoan thai.',
  },
  {
    step: 6,
    title: 'Vòng mông & Điểm xẻ tà',
    icon: 'straighten',
    dotPosition: 'top-[52%] left-1/2 -translate-x-1/2',
    indicator: 'Thông số 6/8',
    focusName: '6. Vòng Mông & Vị Trí Xẻ Tà',
    focusDesc: 'Căn cứ xẻ tà tà trước và sau chạm đúng thắt lưng quần, giữ trọn sự kín đáo trang nghiêm khi đi lại.',
    guide:
      'Đo vòng quanh điểm nở nhất của mông. Vị trí xẻ tà của AN SOIE được tính toán trùng khít đường cạp quần để không bao giờ bị hở lườn, kín đáo trang nhã.',
  },
  {
    step: 7,
    title: 'Rộng vai & Dài tay áo',
    icon: 'straighten',
    dotPosition: 'top-[22%] left-[20%]',
    indicator: 'Thông số 7/8',
    focusName: '7. Rộng Vai & Dài Tay Áo',
    focusDesc: 'Đo chuẩn từ chân cổ qua mỏm vai đến cổ tay, giúp phần nách raglan của áo dài phẳng phiu không đùn vải.',
    guide: 'Đo từ đỉnh vai trái sang đỉnh vai phải phía sau lưng. Tay áo đo từ đỉnh vai xuôi dọc cánh tay qua mắt cá cổ tay (hoặc lửng theo ý thích).',
  },
  {
    step: 8,
    title: 'Dài áo & Dài quần',
    icon: 'straighten',
    dotPosition: 'bottom-[10%] left-1/2 -translate-x-1/2',
    indicator: 'Thông số 8/8',
    focusName: '8. Dài Áo & Dài Quần',
    focusDesc: 'Đo từ gáy sau rủ thẳng qua gót chân, tính toán bù trừ hoàn hảo theo chiều cao đế giày của quý cô.',
    guide: 'Đo từ đốt sống cổ thứ 7 sau gáy kéo thẳng xuống mắt cá chân hoặc chạm mu bàn chân tùy thuộc vào độ cao gót giày dự định mang.',
  },
]

export const measurementFormFields = [
  { id: 'neck', label: 'Cổ (cm)', placeholder: '34', type: 'number' },
  { id: 'bust', label: 'Ngực (cm)', placeholder: '86', type: 'number' },
  { id: 'waist', label: 'Eo (cm)', placeholder: '65', type: 'number' },
  { id: 'hips', label: 'Mông (cm)', placeholder: '92', type: 'number' },
  { id: 'shoulder', label: 'Rộng vai (cm)', placeholder: '37', type: 'number' },
  { id: 'sleeve', label: 'Dài tay (cm)', placeholder: '56', type: 'number' },
  { id: 'length', label: 'Dài áo (cm)', placeholder: '138', type: 'number' },
  { id: 'body', label: 'Chiều cao/Cân', placeholder: '1m62 / 50kg', type: 'text' },
]

export interface Faq {
  id: number
  question: string
  answer: string
  bullets?: string[]
}

export const faqs: Faq[] = [
  {
    id: 1,
    question: '1. Nếu áo may đo xong mặc không vừa vặn thì xử lý thế nào?',
    answer:
      'AN SOIE áp dụng chính sách **Chỉnh sửa miễn phí trọn đời** cho mọi trang phục may đo. Nếu mặc chưa ưng ý dù chỉ 1cm ở nách, eo hay dài tà, quý khách chỉ cần gửi áo hoặc chuyên viên sẽ đến lấy lại và tinh chỉnh hoàn thiện trong 48h làm việc mà không phát sinh thêm bất cứ chi phí nào.',
  },
  {
    id: 2,
    question: '2. Thời gian hoàn thiện một chiếc áo dài may đo mất bao lâu?',
    answer:
      'Thời gian may thông thường từ **5 đến 7 ngày** đối với áo dài trơn lụa tơ tằm. Với các mẫu thêu tay kỳ công họa tiết dân gian từ làng Quất Động hoặc kết hạt cườm thủ công, thời gian cần từ **10 đến 15 ngày**. Trong trường hợp quý khách cần gấp cho lễ cưới hỏi hoặc sự kiện ngoại giao, AN SOIE có dịch vụ may tốc hành ưu tiên trong 72 giờ.',
  },
  {
    id: 3,
    question: '3. Người có vóc dáng đầy đặn hoặc mảnh khảnh có mặc đẹp không?',
    answer: 'Áo dài may đo chính là giải pháp lý tưởng nhất để tôn vinh nét riêng của mỗi người:',
    bullets: [
      '**Vóc dáng đầy đặn hoặc vai ngang:** Nghệ nhân sẽ tư vấn dáng cổ thuyền nhẹ, cổ tròn thấp 1.5cm, kết hợp lụa có độ rủ mềm và đường chiết nẹp khéo léo để tạo cảm giác cổ thanh dài, giấu vòng bụng tự nhiên.',
      '**Vóc dáng mảnh khảnh:** Phù hợp với áo ngũ thân tay chẽn hoặc lụa dệt vân chìm, gam màu ấm sáng, kết hợp độ hạ ngực nâng phom giúp cơ thể đầy đặn, quý phái.',
    ],
  },
  {
    id: 4,
    question: '4. Cách bảo quản và giặt ủi áo dài tơ tằm tự nhiên tại gia đình?',
    answer:
      'Vì lụa tơ tằm nguyên chất từ Vạn Phúc là sợi protein tự nhiên, quý khách nên giặt khô hoặc giặt nhẹ bằng tay với dầu gội dịu nhẹ, nước mát. Tuyệt đối không vắt xoắn mạnh, phơi trong bóng râm thoáng gió và ủi ở chế độ lụa (Silk) khi vải còn hơi ẩm nhẹ từ mặt trái.',
  },
]
