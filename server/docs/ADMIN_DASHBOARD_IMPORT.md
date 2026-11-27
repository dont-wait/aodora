# Admin: Dashboard thống kê & Import/Export sản phẩm

Cả hai nhóm endpoint nằm dưới `/api/v1/admin/*`, dùng chung guard của catalog admin (`RequireAdmin`, kiểm tra Origin, `Cache-Control: no-store`). Preflight do `OPTIONS /api/v1/admin/{path...}` xử lý.

## 1. Dashboard

`GET /api/v1/admin/dashboard?from=YYYY-MM-DD&to=YYYY-MM-DD&granularity=day|week|month&low_stock=N`

- Mọi tham số tùy chọn. Mặc định: 30 ngày gần nhất kết thúc hôm nay, `granularity=day`, `low_stock=3`.
- Ngày tính theo **giờ Việt Nam (UTC+7)**; `from`/`to` đều là ngày bao gồm. Khoảng tối đa **366 ngày** (vượt ⇒ 400 `INVALID_QUERY`).
- **Kỳ trước** là khoảng liền ngay trước, cùng độ dài (để tính % thay đổi).
- Tuần bắt đầu thứ Hai; tháng bắt đầu ngày 1 (giờ VN). Các mốc không có đơn được điền `0`.
- Tiền trả về là **số nguyên VND**.

### Định nghĩa chỉ số

| Chỉ số | Định nghĩa |
|---|---|
| **Doanh thu** | Tổng `orders.total` của đơn có `payment_status = "paid"` **và** `status != "cancelled"`, tính theo `created_at` |
| **Số đơn** | Mọi đơn tạo trong kỳ (kể cả hủy/chưa thanh toán) |
| **Giá trị đơn trung bình** | Doanh thu ÷ số đơn tính doanh thu (đơn đã thanh toán, không hủy) |
| **Khách mới** | `customers.created_at` trong kỳ |
| **Tỉ lệ hủy** | Số đơn `cancelled` ÷ số đơn (0–1) |
| `change_pct` | (kỳ này − kỳ trước) ÷ kỳ trước × 100; `null` khi kỳ trước = 0 |
| Đơn theo trạng thái / thanh toán | Đếm theo `status` / `payment_status` của mọi đơn trong kỳ |
| Theo phương thức | `payment_method`: số đơn và doanh thu (định nghĩa như trên) |
| Top sản phẩm / theo danh mục / theo size | Từ `items` của **đơn tính doanh thu**: doanh thu dòng = `unit_price × quantity`; tên/ảnh lấy bằng `$lookup products`, danh mục `products.category_id → categories`, size = `items.size` ("" ⇒ không có cỡ) |
| **Tồn kho** | Giá trị tồn = Σ `price × stock` của **mọi** sản phẩm; SKU hết hàng (`stock ≤ 0`) / sắp hết (`stock ≤ low_stock`) và danh sách 20 SKU thấp nhất chỉ tính sản phẩm **đang bán**; số sản phẩm đang bán/ẩn |
| May đo | Số `tailoring_requests` tạo trong kỳ, theo `status` |

### Kỹ thuật
- `internal/application/dashboard` (logic thuần: khoảng thời gian, lấp mốc, % thay đổi) + `internal/infrastructure/mongo/dashboard_repository.go` (aggregation: `$match`, `$facet`, `$group`, `$dateTrunc` với `timezone: "+07:00"`, `$unwind`, `$lookup`).
- Tiền có thể là Decimal128 (seed/checkout) hoặc int/long (importer): mọi biểu thức đi qua `$toDecimal`, kết quả làm tròn bằng `$toLong($round)`.
- Migration `*_dashboard_indexes` thêm `orders.created_at`, `customers.created_at`, `tailoring_requests.created_at` vì các truy vấn lọc theo khoảng ngày (index cũ chỉ có `status + created_at`, `customer_id + created_at`, ... nên không dùng được cho truy vấn thuần theo ngày). Đã kiểm tra `explain` dùng `ix_orders_created_at`.

## 2. Import / Export

### Định dạng
CSV UTF-8 (BOM được chấp nhận; Excel: "CSV UTF-8") hoặc JSON (mảng object phẳng, khóa trùng tên cột). Tên cột không phân biệt hoa thường. Cột thiếu/dư ⇒ lỗi theo dòng 1.

**Danh mục** (`name, slug` bắt buộc): `name, slug, description, parent_slug, sort_order, is_active, image_url, image_alt`

**Sản phẩm** — mỗi dòng **một SKU**, các dòng cùng `product_slug` gộp thành một sản phẩm (`product_slug, name, category_slug, price, sku` bắt buộc): `product_slug, name, category_slug, description, material, price, compare_at_price, is_active, image_url, gallery_urls (ngăn bởi |), color_name, color_hex, size, sku, stock`

- Thông tin cấp sản phẩm lấy ở dòng đầu; dòng sau có thể để trống, nhưng nếu điền phải **giống** dòng đầu.
- `price`/`compare_at_price`/`stock`: số nguyên không âm. `is_active`: true/false/1/0 (mặc định true). `color_hex`: `#RRGGBB`.
- `size` phải là size chuẩn (XS, S, M, L, XL, XXL, Freesize) hoặc rỗng; không phân biệt hoa thường, lưu dạng chuẩn.
- Giới hạn: file ≤ 5 MB, ≤ 5.000 dòng dữ liệu.
- **Ảnh (`image_url`, `gallery_urls`)**: URL **mới** phải nằm trong bucket ảnh của hệ thống (bắt đầu bằng `RUSTFS_PUBLIC_URL`, tức URL do trang quản trị tải lên). URL đã có sẵn trên bản ghi (dữ liệu cũ) được giữ nếu không đổi. Link ảnh ngoài bị báo lỗi tại dòng/cột. File mẫu và file xuất dùng URL của bucket.
- Các luật kiểm tra (tên 3–150, giá 1.000–1.000.000.000, giá gốc > giá bán, SKU `^[A-Z0-9][A-Z0-9-]{2,39}# Admin: Dashboard thống kê & Import/Export sản phẩm

Cả hai nhóm endpoint nằm dưới `/api/v1/admin/*`, dùng chung guard của catalog admin (`RequireAdmin`, kiểm tra Origin, `Cache-Control: no-store`). Preflight do `OPTIONS /api/v1/admin/{path...}` xử lý.

## 1. Dashboard

`GET /api/v1/admin/dashboard?from=YYYY-MM-DD&to=YYYY-MM-DD&granularity=day|week|month&low_stock=N`

- Mọi tham số tùy chọn. Mặc định: 30 ngày gần nhất kết thúc hôm nay, `granularity=day`, `low_stock=3`.
- Ngày tính theo **giờ Việt Nam (UTC+7)**; `from`/`to` đều là ngày bao gồm. Khoảng tối đa **366 ngày** (vượt ⇒ 400 `INVALID_QUERY`).
- **Kỳ trước** là khoảng liền ngay trước, cùng độ dài (để tính % thay đổi).
- Tuần bắt đầu thứ Hai; tháng bắt đầu ngày 1 (giờ VN). Các mốc không có đơn được điền `0`.
- Tiền trả về là **số nguyên VND**.

### Định nghĩa chỉ số

| Chỉ số | Định nghĩa |
|---|---|
| **Doanh thu** | Tổng `orders.total` của đơn có `payment_status = "paid"` **và** `status != "cancelled"`, tính theo `created_at` |
| **Số đơn** | Mọi đơn tạo trong kỳ (kể cả hủy/chưa thanh toán) |
| **Giá trị đơn trung bình** | Doanh thu ÷ số đơn tính doanh thu (đơn đã thanh toán, không hủy) |
| **Khách mới** | `customers.created_at` trong kỳ |
| **Tỉ lệ hủy** | Số đơn `cancelled` ÷ số đơn (0–1) |
| `change_pct` | (kỳ này − kỳ trước) ÷ kỳ trước × 100; `null` khi kỳ trước = 0 |
| Đơn theo trạng thái / thanh toán | Đếm theo `status` / `payment_status` của mọi đơn trong kỳ |
| Theo phương thức | `payment_method`: số đơn và doanh thu (định nghĩa như trên) |
| Top sản phẩm / theo danh mục / theo size | Từ `items` của **đơn tính doanh thu**: doanh thu dòng = `unit_price × quantity`; tên/ảnh lấy bằng `$lookup products`, danh mục `products.category_id → categories`, size = `items.size` ("" ⇒ không có cỡ) |
| **Tồn kho** | Giá trị tồn = Σ `price × stock` của **mọi** sản phẩm; SKU hết hàng (`stock ≤ 0`) / sắp hết (`stock ≤ low_stock`) và danh sách 20 SKU thấp nhất chỉ tính sản phẩm **đang bán**; số sản phẩm đang bán/ẩn |
| May đo | Số `tailoring_requests` tạo trong kỳ, theo `status` |

### Kỹ thuật
- `internal/application/dashboard` (logic thuần: khoảng thời gian, lấp mốc, % thay đổi) + `internal/infrastructure/mongo/dashboard_repository.go` (aggregation: `$match`, `$facet`, `$group`, `$dateTrunc` với `timezone: "+07:00"`, `$unwind`, `$lookup`).
- Tiền có thể là Decimal128 (seed/checkout) hoặc int/long (importer): mọi biểu thức đi qua `$toDecimal`, kết quả làm tròn bằng `$toLong($round)`.
- Migration `*_dashboard_indexes` thêm `orders.created_at`, `customers.created_at`, `tailoring_requests.created_at` vì các truy vấn lọc theo khoảng ngày (index cũ chỉ có `status + created_at`, `customer_id + created_at`, ... nên không dùng được cho truy vấn thuần theo ngày). Đã kiểm tra `explain` dùng `ix_orders_created_at`.

## 2. Import / Export

### Định dạng
CSV UTF-8 (BOM được chấp nhận; Excel: "CSV UTF-8") hoặc JSON (mảng object phẳng, khóa trùng tên cột). Tên cột không phân biệt hoa thường. Cột thiếu/dư ⇒ lỗi theo dòng 1.

**Danh mục** (`name, slug` bắt buộc): `name, slug, description, parent_slug, sort_order, is_active, image_url, image_alt`

**Sản phẩm** — mỗi dòng **một SKU**, các dòng cùng `product_slug` gộp thành một sản phẩm (`product_slug, name, category_slug, price, sku` bắt buộc): `product_slug, name, category_slug, description, material, price, compare_at_price, is_active, image_url, gallery_urls (ngăn bởi |), color_name, color_hex, size, sku, stock`

- Thông tin cấp sản phẩm lấy ở dòng đầu; dòng sau có thể để trống, nhưng nếu điền phải **giống** dòng đầu.
- `price`/`compare_at_price`/`stock`: số nguyên không âm. `is_active`: true/false/1/0 (mặc định true). `color_hex`: `#RRGGBB`.
- `size` phải là size chuẩn (XS, S, M, L, XL, XXL, Freesize) hoặc rỗng; không phân biệt hoa thường, lưu dạng chuẩn.
, màu SKU phải nằm trong danh sách màu của sản phẩm...) là cùng một bộ luật với form quản trị, xem mục 3.

### Endpoint
- `POST /api/v1/admin/import/{categories|products}?mode=dry_run|commit` — `multipart/form-data` (trường `file`), hoặc body `text/csv` / `application/json`. Mặc định `dry_run`.
- `GET /api/v1/admin/export/{categories|products}` — CSV (BOM) đúng định dạng nhập; sản phẩm xuất một dòng mỗi SKU, lặp lại cột cấp sản phẩm (sản phẩm chưa có SKU không được xuất).

Kết quả: `{ mode, rows, created, updated, skipped, ignored_stock, committed, errors:[{row,field,message}], notes, items:[{row,slug,action,detail}] }`. `row` là số dòng trong file (dòng 1 = tiêu đề CSV). Lỗi mức file (sai định dạng, quá lớn...) trả `row = 0`/`1` trong `errors` với HTTP 200.

### Quy tắc
- Mỗi dòng được kiểm tra bằng **chính các hàm validate của `catalogadmin`** (slug, giá, ảnh, size, ...) và ghi qua chính các hàm ghi của service (cùng luật, cùng `version`).
- `dry_run` không ghi gì. `commit` **chỉ chạy khi không còn lỗi**; có lỗi ⇒ `committed:false`, không ghi gì.
- **Upsert** theo `slug` (danh mục, sản phẩm) và `sku` (variant). Mục không có thay đổi ⇒ `skipped` (nhập lại cùng một file là no-op).
- **Không bao giờ ghi đè tồn kho của SKU đã có**: giá trị `stock` của SKU cũ bị bỏ qua (`ignored_stock`, kèm ghi chú). SKU mới: tạo với tồn 0 rồi cộng tồn đầu bằng điều chỉnh delta (`request_id` ngẫu nhiên); sản phẩm mới được tạo với tồn trong file.
- Sản phẩm có sẵn: cập nhật qua `UpdateProduct` với `expected_version` hiện tại (tăng `version`). Cột để trống **không xóa** dữ liệu cũ (mô tả, chất liệu, ảnh, màu giữ nguyên); phụ kiện không bao giờ bị đổi. SKU không có trong file không bị xóa. Màu/cỡ khác ⇒ `UpdateVariant`.
- Danh mục cha theo `parent_slug` (có thể nằm cùng file, thứ tự dòng tùy ý; phát hiện cha không tồn tại/vòng lặp). Đổi cha của danh mục đã có cần replica set (như thao tác sửa thường).
- SKU đã thuộc sản phẩm khác ⇒ lỗi dòng. Trùng SKU hoặc trùng (màu, cỡ) trong file ⇒ lỗi dòng.
- **Không dùng transaction**: ghi tuần tự theo thứ tự dòng. Lỗi hạ tầng/xung đột giữa chừng ⇒ `committed:false`, `notes` ghi số mục đã ghi; chạy lại file sau khi sửa (mục đã ghi sẽ thành `skipped`).
- Lệnh commit được kéo dài write deadline lên 2 phút (server mặc định 10 giây).

## 3. Upload ảnh & luật kiểm tra

### Upload ảnh
`POST /api/v1/admin/uploads/images` — `multipart/form-data`: `file` + `purpose` (`product` | `category` | `content` | `accessory`). Cùng guard admin/Origin như các route admin.

- Giới hạn: ≤ **10 MB**; chỉ **JPEG / PNG / WebP**, nhận diện theo **magic bytes** (bỏ qua đuôi file và Content-Type; SVG, GIF, HEIC bị từ chối kèm gợi ý); cạnh ngắn ≥ **400 px**, cạnh dài ≤ **8000 px**.
- Lưu vào RustFS (S3 API, `github.com/minio/minio-go/v7`): key `uploads/<purpose>/<yyyy>/<mm>/<uuid>.<ext>`, `Content-Type` đúng, `Cache-Control: public, max-age=31536000, immutable`.
- Trả `201 { data: { url, key, width, height, size, content_type } }`. Lỗi: 400 `INVALID_FIELD` (field `file`/`purpose`), 413 `FILE_TOO_LARGE`, 503 `STORAGE_UNAVAILABLE` (thiếu cấu hình hoặc kho lỗi; API vẫn khởi động).
- Biến môi trường: `RUSTFS_ACCESS_KEY`, `RUSTFS_SECRET_KEY`, `RUSTFS_BUCKET`, `RUSTFS_PUBLIC_URL` (rỗng ⇒ `http://localhost:$RUSTFS_API_PORT/$RUSTFS_BUCKET`) và **mới** `RUSTFS_ENDPOINT` (rỗng ⇒ `http://localhost:$RUSTFS_API_PORT`).
- Server chỉ nhận ảnh của mình: URL ảnh **mới** khi tạo/sửa danh mục, sản phẩm (ảnh chính, gallery, phụ kiện) và nội dung sản phẩm phải bắt đầu bằng public URL của bucket; URL cũ đã có trên bản ghi được giữ nguyên nếu không đổi.
- **Việc sau**: ảnh đã tải lên nhưng không được lưu vào bản ghi nào (ảnh mồ côi) chưa được dọn tự động.

### Luật kiểm tra (server là nguồn sự thật; `client/src/features/admin/validation.ts` giống hệt)
Lỗi trả `400 INVALID_FIELD` với `field` là đường dẫn trường (`name`, `price`, `colors[1].hex`, `variants[2].sku`, `accessories[0].price`, `gallery[3].src`...).

| Đối tượng | Luật |
|---|---|
| Sản phẩm | tên 3–150 (trim); slug `^[a-z0-9]+(-[a-z0-9]+)*$` ≤ 120, **trùng ⇒ lỗi tại `slug`**; danh mục bắt buộc và phải tồn tại; mô tả ≤ 5.000; chất liệu ≤ 200; giá 1.000–1.000.000.000 (nguyên); giá gốc nếu có phải > giá bán; gallery ≤ 12, không trùng; alt 1–200 |
| Bật "Đang bán" | cần ảnh chính (và ≥ 1 SKU với sản phẩm đã có) khi chuyển từ ẩn sang bán |
| Màu | ≤ 20; tên 1–50, không trùng (không phân biệt hoa thường/dấu); hex `#RRGGBB` (chuẩn hóa chữ hoa). Màu đang được SKU dùng không xóa/đổi tên được |
| SKU | `^[A-Z0-9][A-Z0-9-]{2,39}$` sau khi tự viết hoa + bỏ dấu; trùng trong sản phẩm hoặc với sản phẩm khác ⇒ lỗi tại `variants[i].sku` (hoặc `sku`); size chuẩn; **màu phải thuộc danh sách màu của sản phẩm**; cặp (màu, size) không trùng; tồn ban đầu 0–100.000 |
| Điều chỉnh tồn | số nguyên khác 0, `|delta| ≤ 100.000`; không làm tồn âm (lỗi `delta`: "kho chỉ còn N") |
| Danh mục | tên 2–100; slug như trên, trùng ⇒ lỗi; cha không được là chính nó hoặc con cháu; thứ tự 0–9.999; mô tả ≤ 2.000 |
| Phụ kiện | ≤ 20; tên 1–100; mô tả ≤ 500; giá 0–100.000.000; id không trùng |

Lưu ý: DB hiện **không có unique index cho `variants.sku`** (chỉ `slug`); tính duy nhất của SKU được đảm bảo bằng kiểm tra ở service (có thể thêm index sau khi dọn dữ liệu trùng).
