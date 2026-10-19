# Báo cáo cấu trúc cơ sở dữ liệu MongoDB

- **Cơ sở dữ liệu:** `QuanLyBanQuanAo`
- **Phiên bản schema:** migration `4`
- **Nguồn cấu trúc:** các migration `000001_init`, `000002_clothing_store`, `000004_client_fields` trong `server/migrations`.
- **Quy ước:** BSON type mô tả kiểu được validator cho phép. `bắt buộc` nghĩa là nằm trong `required` của validator; các field còn lại là tùy chọn. Với MongoDB, `null` chỉ được phép khi ghi rõ `| null`.

## Tổng quan collections

| Collection | Mục đích |
|---|---|
| `health_checks` | Dữ liệu kiểm tra trạng thái ứng dụng. |
| `categories` | Danh mục sản phẩm và metadata hiển thị trên client. |
| `products` | Sản phẩm, biến thể, nội dung chi tiết và thông tin đặt may. |
| `customers` | Tài khoản khách hàng, địa chỉ, số đo và wishlist. |
| `orders` | Đơn hàng, dòng sản phẩm, thanh toán và địa chỉ giao hàng snapshot. |
| `carts` | Giỏ hàng của khách và tùy chọn sản phẩm. |
| `reviews` | Đánh giá sản phẩm, thông tin người đánh giá và ảnh. |
| `tailoring_requests` | Yêu cầu tư vấn/đặt may. |

MongoDB tự tạo `_id: ObjectId` cho document nếu ứng dụng không truyền `_id`; `_id` không được khai báo là trường bắt buộc trong các validator nghiệp vụ này. Các validator không đặt `additionalProperties: false`, do đó những field chưa liệt kê không bị cấm tuyệt đối. Validator kiểm tra kiểu/ràng buộc được nêu, không thay thế kiểm tra nghiệp vụ trong ứng dụng.

## `health_checks`

| Đường dẫn field | BSON type | Bắt buộc | Ràng buộc / ghi chú |
|---|---|---:|---|
| `name` | string | Có | Tên bản ghi kiểm tra. |
| `created_at` | date | Có | Thời điểm tạo. |

## `categories`

| Đường dẫn field | BSON type | Bắt buộc | Ràng buộc / ghi chú |
|---|---|---:|---|
| `name` | string | Có | `minLength: 1`. |
| `slug` | string | Có | `minLength: 1`; có unique index. |
| `is_active` | bool | Có | Trạng thái hiển thị. |
| `created_at` | date | Có | Thời điểm tạo. |
| `updated_at` | date | Có | Thời điểm cập nhật. |
| `description` | string \| null | Không | Mô tả danh mục. |
| `parent_id` | ObjectId \| null | Không | Tham chiếu logic đến `categories._id` (danh mục cha). |
| `id` | string | Không | Nếu có, thuộc enum `co-phuc`, `dan-gian`, `cach-tan`, `hy-su`. |
| `product_count` | int \| long | Không | Tối thiểu 0; số lượng phục vụ hiển thị. |
| `price_from` | int \| long \| decimal | Không | Tối thiểu 0; mức giá bắt đầu. |
| `badge` | object | Không | Metadata nhãn hiển thị. |
| `badge.label` | string | Trong object | Nội dung nhãn. |
| `badge.variant` | string | Trong object | Enum: `primary`, `secondary`, `surface`, `tertiary`, `primaryContainer`, `secondaryContainer`. |
| `image` | object | Không | Ảnh danh mục. |
| `image.src` | string | Trong object | Đường dẫn ảnh. |
| `image.alt` | string | Trong object | Văn bản thay thế ảnh. |
| `sort_order` | int \| long | Không | Tối thiểu 0. |

## `products`

| Đường dẫn field | BSON type | Bắt buộc | Ràng buộc / ghi chú |
|---|---|---:|---|
| `name` | string | Có | `minLength: 1`. |
| `slug` | string | Có | `minLength: 1`; có unique index. |
| `category_id` | ObjectId | Có | Tham chiếu logic đến `categories._id`. |
| `price` | int \| long \| decimal | Có | Tối thiểu 0; quy ước nghiệp vụ là VND. |
| `stock` | int \| long | Có | Tối thiểu 0. |
| `is_active` | bool | Có | Trạng thái kinh doanh/hiển thị. |
| `created_at` | date | Có | Thời điểm tạo. |
| `updated_at` | date | Có | Thời điểm cập nhật. |
| `description` | string \| null | Không | Mô tả sản phẩm. |
| `compare_at_price` | int \| long \| decimal \| null | Không | Nếu là số thì tối thiểu 0. |
| `images[]` | string | Không | Mảng đường dẫn ảnh dạng chuỗi. |
| `variants[]` | object | Không | Biến thể SKU/màu/size và tồn kho được nhúng. |
| `variants[].sku` | string | Trong mỗi phần tử | `minLength: 1`. |
| `variants[].size` | string | Trong mỗi phần tử | Kích cỡ. |
| `variants[].color` | string | Trong mỗi phần tử | Màu sắc. |
| `variants[].stock` | int \| long | Trong mỗi phần tử | Tối thiểu 0. |
| `sku` | string | Không | SKU cấp sản phẩm (field bổ sung cho client). |
| `material` | string | Không | Chất liệu. |
| `old_price` | int \| long \| decimal | Không | Tối thiểu 0. |
| `collection` | string \| null | Không | Bộ sưu tập. |
| `image` | object | Không | Ảnh đại diện; `src`, `alt` là string bắt buộc trong object. |
| `gallery[]` | object | Không | Mỗi ảnh có `src: string` và `alt: string` bắt buộc. |
| `badge` | object | Không | `label: string` và `variant` bắt buộc; enum variant giống `categories.badge.variant`. |
| `tag` | object | Không | `label: string` và `variant` bắt buộc; enum variant giống badge. |
| `colors[]` | object | Không | Mỗi phần tử bắt buộc `hex: string`; `name: string \| null` tùy chọn. |
| `color_note` | string \| null | Không | Ghi chú màu. |
| `sizes[]` | string | Không | Danh sách kích cỡ. |
| `bespoke` | bool | Không | Sản phẩm có hỗ trợ đặt may theo số đo hay không. |
| `cta` | string \| null | Không | Nhãn hành động trên client. |
| `bundle` | string \| null | Không | Thông tin nhóm/bộ sản phẩm. |
| `rating` | int \| long \| double \| decimal | Không | Trong khoảng 0–5. |
| `review_count` | int \| long | Không | Tối thiểu 0. |
| `discount` | int \| long | Không | Trong khoảng 0–100. |
| `availability` | string \| null | Không | Thông tin khả dụng. |
| `accessories[]` | object | Không | Sản phẩm/phụ kiện liên quan được nhúng. |
| `accessories[].id` | string | Trong mỗi phần tử | Định danh hiển thị. |
| `accessories[].name` | string | Trong mỗi phần tử | Tên. |
| `accessories[].description` | string | Trong mỗi phần tử | Mô tả. |
| `accessories[].price` | int \| long \| decimal | Trong mỗi phần tử | Tối thiểu 0. |
| `accessories[].image` | object | Trong mỗi phần tử | Có `src: string`, `alt: string` bắt buộc. |
| `accessories[].tag` | string \| null | Không | Nhãn phụ kiện. |
| `accessories[].tag_variant` | string \| null | Không | Kiểu nhãn. |
| `trust[]` | object | Không | Mỗi phần tử bắt buộc `title`, `description` kiểu string; `icon: string` tùy chọn. |
| `story` | object | Không | Nội dung câu chuyện sản phẩm; các field bên trong đều tùy chọn. |
| `story.eyebrow`, `story.title` | string \| null | Không | Tiêu đề phụ/tiêu đề. |
| `story.paragraphs[]` | string | Không | Các đoạn nội dung. |
| `story.image` | object | Không | Nếu có, `src`, `alt` kiểu string. |
| `fabric` | object | Không | Nội dung chất liệu/chăm sóc; các field bên trong đều tùy chọn. |
| `fabric.eyebrow`, `fabric.title`, `fabric.description` | string \| null | Không | Nội dung giới thiệu. |
| `fabric.image` | object | Không | Nếu có, `src`, `alt` kiểu string. |
| `fabric.care[]` | object | Không | Mỗi phần tử bắt buộc `title`, `description` kiểu string; `icon: string \| null` tùy chọn. |
| `craft` | object | Không | Nội dung quy trình chế tác; các field bên trong đều tùy chọn. |
| `craft.eyebrow`, `craft.title`, `craft.description` | string \| null | Không | Nội dung giới thiệu. |
| `craft.image` | object | Không | Nếu có, `src`, `alt` kiểu string. |
| `craft.steps[]` | object | Không | Mỗi phần tử bắt buộc `title`, `description` kiểu string. |

## Các field số đo dùng chung

Các object `measurement_profile` và `measurements` (trong đơn hàng/giỏ hàng) cho phép các field dưới đây; từng field đều tùy chọn và có thể vắng mặt. Field số đo cho phép BSON `int`, `long`, `double`, `decimal`, giá trị tối thiểu 0.

| Đường dẫn field | BSON type | Bắt buộc | Ràng buộc / ghi chú |
|---|---|---:|---|
| `body` | string \| null | Không | Dáng người/ghi chú dạng chuỗi. |
| `neck_cm` | int \| long \| double \| decimal | Không | Số đo cm, tối thiểu 0. |
| `bust_cm` | int \| long \| double \| decimal | Không | Số đo cm, tối thiểu 0. |
| `waist_cm` | int \| long \| double \| decimal | Không | Số đo cm, tối thiểu 0. |
| `hips_cm` | int \| long \| double \| decimal | Không | Số đo cm, tối thiểu 0. |
| `shoulder_cm` | int \| long \| double \| decimal | Không | Số đo cm, tối thiểu 0. |
| `sleeve_cm` | int \| long \| double \| decimal | Không | Số đo cm, tối thiểu 0. |
| `garment_length_cm` | int \| long \| double \| decimal | Không | Số đo cm, tối thiểu 0. |

## `customers`

| Đường dẫn field | BSON type | Bắt buộc | Ràng buộc / ghi chú |
|---|---|---:|---|
| `full_name` | string | Có | `minLength: 1`. |
| `email` | string | Có | `minLength: 3`; unique index. Validator không khai báo kiểm tra định dạng email. |
| `password_hash` | string | Có | `minLength: 1`; lưu hash, không lưu mật khẩu thô. |
| `created_at` | date | Có | Thời điểm tạo. |
| `updated_at` | date | Có | Thời điểm cập nhật. |
| `phone` | string \| null | Không | Có sparse index. |
| `addresses[]` | object | Không | Địa chỉ nhúng; mỗi phần tử yêu cầu `recipient`, `phone`, `line1`, `city` kiểu string. |
| `addresses[].ward` | string \| null | Không | Phường/xã. |
| `addresses[].district` | string \| null | Không | Quận/huyện. |
| `addresses[].is_default` | bool | Không | Đánh dấu địa chỉ mặc định. |
| `measurement_profile` | object | Không | Object số đo dùng chung ở phần trên. |
| `measurement_notes` | string \| null | Không | Ghi chú số đo. |
| `wishlist_product_ids[]` | ObjectId | Không | Tham chiếu logic đến `products._id`. |

## `orders`

| Đường dẫn field | BSON type | Bắt buộc | Ràng buộc / ghi chú |
|---|---|---:|---|
| `customer_id` | ObjectId | Có | Tham chiếu logic đến `customers._id`. |
| `items[]` | object | Có | Mảng phải có ít nhất 1 phần tử. Mỗi item bắt buộc `product_id:ObjectId`, `product_name:string`, `sku:string`, `quantity:int\|long` (tối thiểu 1), `unit_price:int\|long\|decimal` (tối thiểu 0). |
| `items[].size`, `items[].color` | string \| null | Không | Thuộc tính sản phẩm được chụp tại thời điểm mua. |
| `items[].color_hex` | string \| null | Không | Mã màu. |
| `items[].variant_id` | string \| null | Không | Định danh biến thể. |
| `items[].fulfillment_mode` | string | Không | Enum `ready`, `custom`. |
| `items[].measurements` | object | Không | Có thể nhúng số đo dùng chung. |
| `subtotal` | int \| long \| decimal | Có | Tối thiểu 0. |
| `shipping_fee` | int \| long \| decimal | Có | Tối thiểu 0. |
| `total` | int \| long \| decimal | Có | Tối thiểu 0. |
| `status` | string | Có | Enum `pending`, `confirmed`, `shipping`, `delivered`, `cancelled`. |
| `payment_method` | string | Có | Enum `cod`, `bank_transfer`, `card`, `e_wallet`. |
| `payment_status` | string | Có | Enum `unpaid`, `paid`, `refunded`. |
| `shipping_address` | object | Có | Bắt buộc `recipient`, `phone`, `line1`, `city` kiểu string; `ward`, `district` kiểu string hoặc null tùy chọn. Snapshot địa chỉ tại thời điểm đặt hàng. |
| `created_at` | date | Có | Thời điểm tạo. |
| `updated_at` | date | Có | Thời điểm cập nhật. |
| `note` | string \| null | Không | Ghi chú đơn hàng. |
| `order_number` | string | Không | Mã đơn hiển thị. |
| `fulfillment_mode` | string | Không | Enum `ready`, `custom`. |
| `tailoring_request_id` | ObjectId \| null | Không | Tham chiếu logic đến `tailoring_requests._id`. |
| `measurement_profile` | object | Không | Có thể nhúng số đo dùng chung. |

## `carts`

| Đường dẫn field | BSON type | Bắt buộc | Ràng buộc / ghi chú |
|---|---|---:|---|
| `customer_id` | ObjectId | Có | Tham chiếu logic đến `customers._id`; mỗi customer tối đa một cart theo unique index. |
| `items[]` | object | Có | Mảng có thể rỗng. Mỗi item bắt buộc `product_id:ObjectId`, `sku:string`, `quantity:int\|long` (tối thiểu 1). |
| `items[].added_at` | date | Không | Thời điểm thêm vào giỏ. |
| `items[].color_name`, `items[].color_hex` | string \| null | Không | Tên màu và mã màu. |
| `items[].fulfillment_mode` | string | Không | Enum `ready`, `custom`. |
| `items[].unit_price` | int \| long \| decimal | Không | Tối thiểu 0. |
| `items[].measurements` | object | Không | Có thể nhúng số đo dùng chung. |
| `fulfillment_mode` | string | Không | Enum `ready`, `custom`. |
| `measurement_profile` | object | Không | Có thể nhúng số đo dùng chung. |
| `created_at` | date | Có | Thời điểm tạo. |
| `updated_at` | date | Có | Thời điểm cập nhật. |

## `reviews`

| Đường dẫn field | BSON type | Bắt buộc | Ràng buộc / ghi chú |
|---|---|---:|---|
| `product_id` | ObjectId | Có | Tham chiếu logic đến `products._id`. |
| `customer_id` | ObjectId | Có | Tham chiếu logic đến `customers._id`. |
| `rating` | int \| long | Có | Trong khoảng 1–5. |
| `created_at` | date | Có | Thời điểm tạo. |
| `updated_at` | date | Có | Thời điểm cập nhật. |
| `comment` | string \| null | Không | Nội dung đánh giá kiểu cũ. |
| `content` | string | Không | Nội dung đánh giá. |
| `display_name` | string | Không | Tên hiển thị. |
| `initials` | string | Không | Chữ viết tắt hiển thị. |
| `location` | string \| null | Không | Địa điểm hiển thị. |
| `meta` | string \| null | Không | Metadata hiển thị. |
| `purchase_mode` | string | Không | Enum `ready`, `custom`. |
| `purchase_size` | string \| null | Không | Size đã mua. |
| `verified_purchase` | bool | Không | Đánh dấu đã xác minh mua hàng. |
| `images[]` | object | Không | Mỗi phần tử yêu cầu `src:string`, `alt:string`. |

## `tailoring_requests`

| Đường dẫn field | BSON type | Bắt buộc | Ràng buộc / ghi chú |
|---|---|---:|---|
| `full_name` | string | Có | `minLength: 1`. |
| `phone` | string | Có | `minLength: 1`. |
| `status` | string | Có | Enum `submitted`, `contacted`, `consultation_scheduled`, `measurement_confirmed`, `in_progress`, `completed`, `cancelled`. |
| `created_at` | date | Có | Thời điểm tạo. |
| `updated_at` | date | Có | Thời điểm cập nhật. |
| `customer_id` | ObjectId \| null | Không | Tham chiếu logic đến `customers._id`. |
| `preferred_contact` | string | Không | Enum `phone`, `zalo`. |
| `category_id` | string \| null | Không | ID/slug danh mục do client gửi; schema không bắt buộc là ObjectId. |
| `product_id` | ObjectId \| null | Không | Tham chiếu logic đến `products._id`. |
| `product_slug` | string \| null | Không | Slug sản phẩm. |
| `measurements` | object | Không | Có thể nhúng số đo dùng chung. |
| `notes` | string \| null | Không | Ghi chú yêu cầu. |
| `appointment_at` | date \| null | Không | Thời gian hẹn. |
| `assigned_staff` | string \| null | Không | Nhân viên phụ trách. |

## Index và ràng buộc duy nhất

| Collection | Tên/mô tả index | Field | Tính chất |
|---|---|---|---|
| `categories` | `uq_categories_slug` | `slug` | Unique. |
| `categories` | `ix_categories_parent_active` | `parent_id`, `is_active` | Tra danh mục con/trạng thái. |
| `products` | `uq_products_slug` | `slug` | Unique. |
| `products` | `ix_products_category_active` | `category_id`, `is_active` | Lọc sản phẩm theo danh mục và trạng thái. |
| `products` | `ix_products_search` | `name`, `description` | Text index tìm kiếm. |
| `customers` | unique email | `email` | Unique. |
| `customers` | phone index | `phone` | Sparse. |
| `orders` | customer timeline | `customer_id`, `created_at` giảm dần | Lịch sử đơn theo khách. |
| `orders` | status timeline | `status`, `created_at` giảm dần | Lọc đơn theo trạng thái/thời gian. |
| `carts` | unique customer | `customer_id` | Mỗi khách một cart. |
| `reviews` | product timeline | `product_id`, `created_at` giảm dần | Lấy đánh giá theo sản phẩm/thời gian. |
| `reviews` | unique customer/product | `customer_id`, `product_id` | Một khách tối đa một review cho mỗi sản phẩm. |
| `tailoring_requests` | phone timeline | `phone`, `created_at` giảm dần | Tra yêu cầu theo số điện thoại. |
| `tailoring_requests` | status timeline | `status`, `created_at` giảm dần | Lọc yêu cầu theo trạng thái/thời gian. |
| `tailoring_requests` | customer timeline | `customer_id`, `created_at` giảm dần | Sparse; tra yêu cầu của khách đã đăng nhập. |

## Thiết kế nhúng và tham chiếu

- **Nhúng (embedded documents):** địa chỉ khách trong `customers.addresses`; dòng hàng trong `orders.items` và `carts.items`; badge/image/gallery/colors/story/fabric/craft/accessories/trust trong `products`; ảnh trong `reviews`; số đo trong hồ sơ khách, cart, order và yêu cầu may. Các dữ liệu này thường được đọc cùng document cha; địa chỉ giao hàng và thông tin item trong order là snapshot để giữ lịch sử giao dịch.
- **Tham chiếu logic bằng ObjectId:** `products.category_id` → `categories`; `orders.customer_id`, `carts.customer_id`, `reviews.customer_id`, `tailoring_requests.customer_id` → `customers`; `orders.items[].product_id`, `carts.items[].product_id`, `reviews.product_id`, `tailoring_requests.product_id`, `customers.wishlist_product_ids[]` → `products`; `orders.tailoring_request_id` → `tailoring_requests`.
- MongoDB không tạo foreign key cho các tham chiếu trên. Ứng dụng cần đảm bảo document đích tồn tại và xử lý cập nhật/xóa phù hợp. Index hỗ trợ truy vấn nhưng không tự xác thực quan hệ.
- Kiểu `decimal` được validator chấp nhận cho một số tiền; ứng dụng nên thống nhất cách ghi/đọc tiền tệ và làm tròn. Schema hiện tại chỉ đặt giới hạn tối thiểu, không buộc giá trị tiền phải là số nguyên.

## Collection nội bộ của công cụ migration

`schema_migrations` và `migrate_advisory_lock` là collection phục vụ quản lý version/khóa migration của công cụ; chúng không thuộc schema nghiệp vụ được mô tả bởi các validator ở trên. Không nên tự sửa hoặc xóa dữ liệu migration khi chưa hiểu tác động tới trạng thái triển khai.
