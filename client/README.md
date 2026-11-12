# AN SOIE – Client

React 19 + Vite + TypeScript + Tailwind CSS 3 + React Router.

```bash
npm install
npm run dev      # http://localhost:5173
npm run build    # type-check + build
npm run lint
```

## Cấu trúc

```
src/
├── app/                 # App + định nghĩa route
├── components/
│   ├── layout/          # Header, Footer, Layout (Outlet + ToastProvider)
│   └── ui/              # Thành phần dùng chung: Icon, Badge, Modal, Breadcrumb, StarRating, RichText...
├── pages/               # Mỗi trang một thư mục (feature-based)
│   ├── home/            #   HomePage.tsx + components/ riêng của trang
│   ├── collection/
│   ├── product-detail/
│   └── guide/
├── data/                # Nội dung giới thiệu và bảng tham khảo dùng chung
├── features/catalog/   # HTTP client, kiểu dữ liệu catalog và hợp đồng SKU/giỏ hàng
├── hooks/               # useToast...
├── types/               # Kiểu dùng chung (Product, Category, Img...)
└── utils/               # formatVND...
```

- Alias `@/` trỏ tới `src/`.
- Design tokens (màu, font, spacing) nằm trong `tailwind.config.js`.
- Tương tác (tab, modal, form, toast...) dùng React state, không thao tác DOM trực tiếp.
- Các trang sản phẩm sử dụng API PD-01 thay cho danh sách mẫu; hợp đồng ghép giỏ hàng nằm ở `features/catalog/selection.ts`.
- Hướng dẫn dữ liệu và tích hợp: `docs/PD02_CATALOG_UI.md`.
- Test nghiệp vụ catalog: `npm run test:catalog` (Node.js 22+).
- Vite chuyển tiếp `/api` đến Go qua `API_PROXY_TARGET` (mặc định `http://localhost:8081`).
