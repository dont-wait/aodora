# PD-03 — Quản trị danh mục và thông tin bán áo

## Phạm vi

Chủ cửa hàng quản lý danh mục, thông tin bán áo, giá, ảnh, phụ kiện, SKU và tồn kho trên hai collection MongoDB hiện có: `categories` và `products`. Sử dụng thêm collection kỹ thuật `catalog_admin_state` (một document điều phối thay đổi cây danh mục). Không ghi vào `orders`, `reviews`, `carts` hoặc dữ liệu khách hàng.

**Xác thực:** Mọi URL `/api/v1/admin/...` phải được server xác minh là admin theo KR-01. Nếu chưa cung cấp `AdminAuthorizer` đáng tin cậy, hệ thống từ chối HTTP 403, kể cả GET. Không chấp nhận một header `X-Role`, ID tài khoản do client tự gửi hoặc mã truy cập gán cứng để suy ra quyền admin. Phần tích hợp xác thực còn phụ thuộc KR-01.

## Danh mục

| Method | URL | Ý nghĩa |
| --- | --- | --- |
| GET | `/api/v1/admin/categories` | Danh sách tất cả danh mục, kể cả đang ẩn |
| POST | `/api/v1/admin/categories` | Tạo danh mục |
| PUT | `/api/v1/admin/categories/{id}` | Thay toàn bộ metadata danh mục, có `expected_version` |

Ví dụ POST:

```json
{
  "name": "Áo dài truyền thống",
  "slug": "ao-dai-truyen-thong",
  "description": "Các mẫu áo dài truyền thống",
  "parent_id": "",
  "is_active": true,
  "sort_order": 1,
  "image": {"src": "/images/categories/truyen-thong.jpg", "alt": "Danh mục áo dài truyền thống"}
}
```

Danh mục ẩn không được hiển thị trên API công khai; danh sách công khai sắp xếp theo `sort_order`, `name`, `_id`. Kiểm tra danh mục cha tồn tại và từ chối tham chiếu tạo vòng lặp; không xóa cứng danh mục để tránh mồ côi sản phẩm.

## Sản phẩm

Danh sách sản phẩm quản trị có phân trang, mặc định 50 và tối đa 100 sản phẩm/trang. Response dạng `{"data":{"items":[...],"total":0,"page":1,"page_size":50,"total_pages":0}}`.

| Method | URL | Ý nghĩa |
| --- | --- | --- |
| GET | `/api/v1/admin/products?page=1&page_size=50` | Xem sản phẩm, kể cả tắt bán (1–100 dòng/trang) |
| POST | `/api/v1/admin/products` | Tạo sản phẩm, bao gồm danh sách SKU khởi tạo |
| PUT | `/api/v1/admin/products/{id}` | Thay metadata sản phẩm với `expected_version`; **không thay đổi tồn kho và danh sách SKU** |
| POST | `/api/v1/admin/products/{id}/variants` | Thêm SKU mới với `stock=0` |
| PATCH | `/api/v1/admin/products/{id}/variants/{sku}` | Sửa màu/cỡ của SKU, không thay tồn |
| POST | `/api/v1/admin/products/{id}/inventory/{sku}` | Điều chỉnh tồn kho theo chênh lệch có điều kiện và `request_id` idempotent |

Ví dụ POST sản phẩm (chỉ là cấu trúc payload, không tạo dữ liệu khi chạy ứng dụng):

```json
{
  "name": "Áo dài dáng cổ điển",
  "slug": "ao-dai-dang-co-dien",
  "category_id": "670000000000000000000001",
  "description": "Mô tả mẫu áo",
  "material": "Lụa",
  "price": 890000,
  "compare_at_price": null,
  "is_active": true,
  "images": ["/images/products/mau.jpg"],
  "image": {"src": "/images/products/mau.jpg", "alt": "Mẫu áo dài"},
  "gallery": [],
  "colors": [],
  "accessories": [],
  "variants": [{"sku": "AD-DO-M", "color": "Đỏ", "size": "M", "stock": 5}]
}
```

Giá, giá gốc, số lượng là số nguyên VND hoặc số nguyên sản phẩm; không được âm. `slug` phải hợp lệ và duy nhất theo index gốc. Mỗi SKU phải có mã, màu, kích cỡ và tồn kho; SKU hoặc tổ hợp màu/cỡ không được trùng trong cùng một sản phẩm. Có tối thiểu một SKU khi tạo. Ảnh quản lý theo đường dẫn URL và mô tả `alt`; chưa tích hợp tải file lên server ở bước này. Migration `000007_catalog_admin_legacy_metadata` điều chỉnh validator sản phẩm để chấp nhận màu thiếu `hex` và phụ kiện thiếu `image` mà không động đến dữ liệu hiện có. `colors[].hex` là tùy chọn và chỉ nhận `#RRGGBB` khi được cung cấp. `accessories[].image` là tùy chọn; `tag` và `tag_variant` được giữ nguyên khi chỉnh sửa. Mọi field ngoài hợp đồng request bị từ chối, bao gồm `rating`, `review_count`, `stock` và `variants` trong PUT metadata.

### Tồn kho và phối hợp DW-02

`products.stock` là tổng tồn SKU và `products.variants[].stock` là tồn của từng SKU. Lúc tạo sản phẩm, tổng được tính từ SKU. Sau đó, chỉ cập nhật tồn qua endpoint delta:

```json
{ "delta": -2, "request_id": "b89928c4-7c4b-4bc9-9c0e-f3941b288080" }
```

Điều chỉnh được thực hiện bằng **một MongoDB atomic conditional `$inc`** lên cả `products.stock` và `variants.$.stock`, đồng thời `$push` bản ghi `inventory_operations` chứa `request_id`, SKU và chênh lệch. Một lần gửi lại đúng `request_id` với cùng payload trả về trạng thái hiện hành **không áp dụng delta lần nữa**; dùng lại ID cho payload khác trả 409. Khi đổi chênh lệch cần phát sinh ID mới; khi kết nối bị gián đoạn phải gửi lại cùng ID. Thiếu hoặc sai UUID trả 400. Nếu không tìm thấy sản phẩm/SKU trả 404, không đủ tồn hay SKU trùng trả 409. Lịch sử chống lặp tối đa 10000 thao tác trên mỗi sản phẩm; khi đạt giới hạn phải có quy trình lưu trữ/kiểm soát trước khi tiếp tục, không tự xóa request ID cũ rồi vô tình cộng/trừ lặp. Lưu form thông tin sản phẩm không được `$set` vào `stock`, `variants` hoặc rating; điều này ngăn một bản form mở lâu ghi đè số vừa được trừ khi tạo đơn.

**Thỏa thuận Q-07 cần được Dontwait xác nhận:** Khi giữ hàng/hủy đơn, DW-02 phải dùng phép cập nhật nguyên tử tương đương cho cùng document và SKU, không thay toàn bộ `variants` dựa trên bản dữ liệu cũ. Giá và tên sản phẩm trong `orders.items` phải là snapshot tại thời điểm đặt, không đọc lại từ sản phẩm mỗi lần hiển thị đơn.

### Đồng thời và tương thích dữ liệu cũ

Các endpoint `PUT` là **full metadata replacement**: request cần chứa các trường metadata bắt buộc kể cả `is_active`, `images`, `gallery`, `colors`, `accessories`, `image` (có thể là `null`), `compare_at_price` (có thể là `null`) cùng `expected_version`. `GET /admin/...` trả `version` (với document cũ thiếu version, client gửi `expected_version: 0`). Ghi thành công tăng version; hai tab cùng sửa một snapshot sẽ nhận `409 CONFLICT`, phải tải lại trước khi lưu.

Khi cập nhật giá so sánh, trường `old_price` legacy được xóa để không còn giá cũ trên storefront. Danh sách ảnh legacy chỉ có `images[]` được khôi phục vào gallery chỉnh sửa; admin có thể chủ động xóa ảnh bằng cách bỏ dòng tương ứng. SKU mới/tên cỡ mới cập nhật `sizes` trực tiếp từ các SKU mà không ghi đè tồn kho. Sản phẩm có tồn tổng không khớp tổng tồn SKU bị chặn khi bổ sung SKU, cần xử lý đối soát theo Q-07 thay vì tự chia tồn tùy ý.

**Đổi quan hệ cha của danh mục cần transaction MongoDB (replica set)**. Repository ghi một document điều phối chung `catalog_admin_state` trong cùng transaction, rồi mới kiểm tra ancestry và cập nhật. Hai transaction cạnh tranh sẽ bị write-conflict/retry, không thể đồng thời tạo vòng lặp. MongoDB standalone **không hỗ trợ** transaction: đổi `parent_id` trả `503 MONGO_TRANSACTION_REQUIRED`; không có fallback read-then-write thiếu an toàn. Các sửa metadata không đổi cha và tạo danh mục không bị giới hạn này. API công khai lọc danh mục tắt và sắp xếp theo thứ tự đã khai báo.

### CORS và xác thực

Nếu web và API có origin khác nhau, đặt `ADMIN_ALLOWED_ORIGIN` bằng **một origin frontend chính xác** (ví dụ `https://shop.example.test`); middleware chỉ phản hồi `Access-Control-Allow-Credentials: true` cho origin đó và xử lý preflight `OPTIONS` cho API admin. Không dùng wildcard khi gửi cookie. Khi chạy cùng origin/proxy không cần CORS. CORS không thay thế KR-01: mọi endpoint admin thực tế vẫn từ chối `403` cho phiên chưa được kiểm tra quyền ở server, bất kể preflight được phép.

### Lỗi và phân quyền

- `400 INVALID_FIELD`: thiếu/sai trường, giá âm, URL ảnh không hợp lệ, nội dung không thuộc schema.
- `403 ADMIN_REQUIRED`: chưa được xác minh quyền admin server-side.
- `404 NOT_FOUND`: sản phẩm hoặc SKU không tồn tại.
- `409 CONFLICT`: slug trùng, SKU trùng, không đủ tồn, thao tác đồng thời không khớp điều kiện.
- `500 ADMIN_CATALOG_UNAVAILABLE`: lỗi cơ sở dữ liệu; thông tin nội bộ chỉ ghi ở log đã che URI có mật khẩu.

PD-04 tiếp tục phụ trách `products.story`, `fabric` và `craft`, không chỉnh nội dung này trong form PD-03. KR-04 phụ trách dữ liệu tổng hợp đánh giá; PD-03 không cập nhật các field liên quan.

Delta có `request_id` là idempotent trên cùng sản phẩm: client giữ ID khi retry cùng payload, đổi ID khi phát sinh thao tác mới. Sau khi mất phản hồi cần đọc lại tồn và có thể retry với cùng ID. Chưa có KR-01 thì CRUD admin không thể nghiệm thu bằng phiên đăng nhập thật.
