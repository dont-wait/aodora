# MongoDB schema: QuanLyBanQuanAo

Schema được đối chiếu với `client/src/types/index.ts`, `client/src/data/products.ts`, `client/src/data/productDetail.ts`, `client/src/data/home.ts` và form trong `client/src/pages/guide/components/MeasurementSection.tsx`. Client hiện giữ dữ liệu catalog/review tĩnh trong TypeScript; form may đo mới hiển thị luồng submit giả lập. Các collection dưới đây chuẩn hóa những field đó để API có thể đọc/ghi về sau.

## Sơ đồ

- [Mở và chỉnh sửa bằng draw.io](database-schema.drawio)
- [Xem SVG](database-schema.svg)

## Collections

### `categories`

Phân nhóm giao diện collection tabs và category cards.

| Field | Type | Ý nghĩa / mapping client |
| --- | --- | --- |
| `_id` | ObjectId | ID MongoDB |
| `id`, `slug` | string | ID FE như `co-phuc`, `dan-gian`, `cach-tan`, `hy-su`; slug truy vấn URL |
| `name` | string | Tên danh mục hiển thị |
| `description` | string? | Mô tả category card |
| `product_count` | int | Số sản phẩm; nên tính từ `products`, chỉ lưu nếu cần cache |
| `price_from` | int/long? | Giá thấp nhất VND, thay cho `priceFrom` đã format sẵn |
| `badge` | `{label, variant}`? | Nhãn category; variant theo các giá trị của UI badge |
| `image` | `{src, alt}`? | Ảnh category card |
| `parent_id` | ObjectId? | Category cha nếu phân cấp |
| `sort_order`, `is_active` | int, bool | Thứ tự và trạng thái hiển thị |
| `created_at`, `updated_at` | date | Thời gian tạo/cập nhật |

### `products`

Một document gom dữ liệu card, chi tiết sản phẩm, lựa chọn mua và nội dung biên tập.

| Field | Type | Ý nghĩa / mapping client |
| --- | --- | --- |
| `_id`, `category_id` | ObjectId | ID và tham chiếu tới `categories._id` |
| `slug`, `sku` | string | URL sản phẩm và mã sản phẩm |
| `name`, `collection`, `material`, `description` | string | Tên, bộ sưu tập, chất liệu, mô tả |
| `price`, `old_price`, `compare_at_price` | int/long? | Giá VND; `old_price` map với `oldPrice`, `compare_at_price` giữ tương thích schema cũ |
| `image` | `{src, alt}` | Ảnh chính của card/detail |
| `gallery[]` | `{src, alt}` | Ảnh detail gallery |
| `badge` | `{label, variant}` | Nhãn chính trên card/detail |
| `tag` | `{label, variant}`? | Nhãn phụ như `Giới Hạn 10 Áo` |
| `colors[]` | `{name?, hex}` | Màu chọn ở product card/detail; chuẩn hóa chuỗi hex của list thành object |
| `color_note` | string? | Map `colorNote`, ví dụ `3 Tùy chọn lụa` |
| `sizes[]` | string | Cỡ có sẵn như S, M, L, XL, XXL |
| `variants[]` | `{sku, size, color, stock}` | SKU/tồn kho theo tổ hợp; sản phẩm may đo vẫn có thể có `sizes: []` |
| `stock`, `is_active` | int, bool | Tồn kho tổng và trạng thái bán |
| `bespoke`, `cta`, `bundle`, `availability` | bool/string? | Chế độ may đo, CTA, mô tả trọn bộ và tình trạng hàng |
| `rating`, `review_count`, `discount` | number/int | Điểm, số đánh giá và phần trăm giảm hiển thị; rating/count nên tính từ reviews, discount có thể tính từ hai giá |
| `accessories[]` | object[] | Phụ kiện gợi ý trong detail: `id`, `name`, `description`, `price`, `tag`, `tag_variant`, `image` |
| `trust[]` | `{icon, title, description}` | Cam kết như chất liệu, bảo hành, đồng kiểm |
| `story` | `{eyebrow, title, paragraphs[], image}` | Tab câu chuyện tác phẩm |
| `fabric` | `{eyebrow, title, description, image, care[]}` | Tab chất liệu và hướng dẫn bảo quản; mỗi care có `icon`, `title`, `description` |
| `craft` | `{eyebrow, title, description, image, steps[]}` | Tab quy trình; mỗi step có `title`, `description` |
| `created_at`, `updated_at` | date | Thời gian tạo/cập nhật |

`price`, `old_price`, và `accessories[].price` lưu số nguyên VND. `rating`, `review_count`, `product_count` là dữ liệu tổng hợp; khi có API nên aggregate từ reviews/products thay vì để chúng lệch khỏi dữ liệu gốc.

### `customers`

Tài khoản khách, liên hệ, địa chỉ, số đo gần nhất và danh sách yêu thích.

| Field | Type | Ý nghĩa / mapping client |
| --- | --- | --- |
| `_id` | ObjectId | ID khách hàng |
| `full_name`, `email`, `password_hash` | string | Danh tính đăng nhập; email lưu lowercase, chỉ lưu hash mật khẩu |
| `phone` | string? | Điện thoại/Zalo từ form may đo |
| `addresses[]` | object[] | `recipient`, `phone`, `line1`, `ward`, `district`, `city`, `is_default` |
| `measurement_profile` | object? | Số đo mới nhất: `body` (chuỗi cao/cân theo form), `neck_cm`, `bust_cm`, `waist_cm`, `hips_cm`, `shoulder_cm`, `sleeve_cm`, `garment_length_cm` |
| `measurement_notes` | string? | Ghi chú vóc dáng/yêu cầu tà áo |
| `wishlist_product_ids[]` | ObjectId[] | Các sản phẩm đã lưu; UI hiện toggle wishlist tại client |
| `created_at`, `updated_at` | date | Thời gian tạo/cập nhật |

### `tailoring_requests`

Yêu cầu may đo được tạo từ form khách có thể gửi mà không cần tài khoản. Form hiện yêu cầu họ tên/số điện thoại và thu thập tám trường: `body` (chiều cao/cân nặng dạng text), bảy số đo cm và ghi chú.

| Field | Type | Ý nghĩa |
| --- | --- | --- |
| `_id` | ObjectId | ID yêu cầu |
| `customer_id` | ObjectId? | Link tài khoản nếu khách đã đăng nhập |
| `full_name`, `phone` | string | Liên hệ bắt buộc của form |
| `preferred_contact` | `phone \| zalo`? | Kênh liên hệ |
| `category_id`, `product_id`, `product_slug` | string/ObjectId? | Category/sản phẩm khách quan tâm |
| `measurements` | object? | `body`, `neck_cm`, `bust_cm`, `waist_cm`, `hips_cm`, `shoulder_cm`, `sleeve_cm`, `garment_length_cm` |
| `notes` | string? | Ghi chú may đo |
| `status` | enum | `submitted`, `contacted`, `consultation_scheduled`, `measurement_confirmed`, `in_progress`, `completed`, `cancelled` |
| `appointment_at`, `assigned_staff` | date/string? | Lịch hẹn và nhân viên phụ trách |
| `created_at`, `updated_at` | date | Thời gian tạo/cập nhật |

### `carts`

Một giỏ đang hoạt động trên mỗi khách; item lưu lựa chọn đúng như purchase panel.

`customer_id` (unique), `items[]` (`product_id`, `sku`, `quantity`, `size`, `color`, `color_name`, `color_hex`, `fulfillment_mode`, `unit_price`, `measurements`, `added_at`), `fulfillment_mode` (`ready`/`custom`), `measurement_profile`, `created_at`, `updated_at`.

### `orders`

Lưu snapshot khi chốt đơn, để sửa catalog sau này không thay đổi lịch sử mua.

`order_number`, `customer_id`, `items[]` (`product_id`, `product_name`, `sku`, `size`, `color`, `color_hex`, `quantity`, `unit_price`, `fulfillment_mode`, `measurements`), `fulfillment_mode` (`ready`/`custom`), `tailoring_request_id`, `measurement_profile`, `subtotal`, `shipping_fee`, `total`, `status`, `payment_method`, `payment_status`, `shipping_address`, `note`, `created_at`, `updated_at`.

Các field fulfillment và đo may là schema mở rộng để hỗ trợ luồng mua sẵn/may đo của product detail; client hiện chưa gửi đơn hàng lên API.

### `reviews`

Map review card: `customer_id`, `product_id`, `rating`, `content`, `display_name`, `initials`, `location`, `meta`, `purchase_mode` (`ready`/`custom`), `purchase_size`, `verified_purchase`, `images[]` (`src`, `alt`), `created_at`, `updated_at`. Mỗi khách chỉ có một review trên một sản phẩm theo unique index.

## Index và quan hệ

- Unique: category slug, product slug, customer email, một cart/khách, một review/khách/sản phẩm.
- Tìm kiếm/sắp xếp: product theo category và trạng thái, product text search, category theo parent, orders theo khách/trạng thái và thời gian, reviews theo sản phẩm, tailoring requests theo số điện thoại/trạng thái/thời gian.
- Quan hệ tham chiếu: `products.category_id → categories._id`; `orders.customer_id`, `carts.customer_id`, `reviews.customer_id`, `tailoring_requests.customer_id → customers._id`; `reviews.product_id`, `orders.items[].product_id`, `carts.items[].product_id`, `tailoring_requests.product_id → products._id`.
- MongoDB không enforce foreign keys. Application cần kiểm tra tham chiếu, tồn kho theo variant và tổng tiền trước khi ghi đơn.

## Kết nối local

Host `localhost`, port `27017`, auth database `admin`, database `QuanLyBanQuanAo`. Tài khoản Docker Compose mặc định: `aodora` / `aodora` (có thể đổi bằng `MONGO_ROOT_USERNAME` và `MONGO_ROOT_PASSWORD`). URI mặc định: `mongodb://aodora:aodora@localhost:27017/QuanLyBanQuanAo?authSource=admin`.

## Migration

Chạy `make migrate-up` trong `server/` để tạo schema và seed demo; `000004_client_fields` mở rộng validator của các collection hiện có và tạo `tailoring_requests`. `make migrate-down` gỡ migration mới nhất theo thứ tự. Migration `000001_init` giữ collection `health_checks`.
