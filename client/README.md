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
├── data/                # Dữ liệu tĩnh (mock) tách khỏi UI – thay bằng API khi có backend
├── hooks/               # useToast...
├── types/               # Kiểu dùng chung (Product, Category, Img...)
└── utils/               # formatVND...
```

- Alias `@/` trỏ tới `src/`.
- Design tokens (màu, font, spacing) nằm trong `tailwind.config.js`.
- Tương tác (tab, modal, form, toast...) dùng React state, không thao tác DOM trực tiếp.
- Các chỗ cần nối API được đánh dấu `TODO`.
