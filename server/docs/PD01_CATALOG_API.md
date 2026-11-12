# PD-01 — API đọc danh mục và sản phẩm (pdmh18)

Phạm vi theo `tasks.md` / `system.md`: **chỉ public read**, không có admin, giỏ hàng, xác thực, chỉnh tồn kho hoặc thay đổi migration. DB: `QuanLyBanQuanAo`. Các response là JSON và chỉ chứa field catalog được cho phép.

## Khởi động

Yêu cầu: Go theo `server/go.mod` (1.25.11), Docker + Docker Compose cho MongoDB. Chạy **trong thư mục `server/`**:

```bash
# Linux/macOS; Windows PowerShell: Copy-Item .env.example .env
cp .env.example .env

docker compose up -d mongo
go run ./cmd/migrate -command up -path migrations
go test ./...
go run ./cmd/api
```

Nếu migrations đã được áp dụng thì chạy `go run ./cmd/migrate -command up -path migrations` không thêm dữ liệu mới. Không có migration mới trong PD-01.

## Endpoints (không cần token)

| Method | Path | Mô tả |
| --- | --- | --- |
| GET | `/api/v1/categories` | Danh mục `is_active=true`, thứ tự `sort_order`, `name`, `_id` |
| GET | `/api/v1/products` | Danh sách công khai, kết hợp filter, total và phân trang |
| GET | `/api/v1/products/{slug}` | Nội dung mẫu theo slug, biến thể và tổng hợp review |

### GET `/api/v1/categories`

```json
{"data":[{"id":"7f0000000000000000000001","key":"co-phuc","slug":"pd01-co-phuc","name":"Cổ Phục Ngũ Thân","sort_order":1}]}
```

Ví dụ có thể có các trường tùy chọn `image`, `badge`, `description`, `parent_id`. Danh mục tắt, bị thiếu/ẩn cha hoặc tạo vòng lặp cha-con không được hiển thị; sản phẩm thuộc các danh mục này cũng không được trả về công khai.

### GET `/api/v1/products`

Tất cả bộ lọc được áp dụng đồng thời:

| Query | Ý nghĩa / điều kiện |
| --- | --- |
| `search` hoặc `q` | Từ khóa khớp một phần tên/mô tả, không phân biệt hoa-thường; dùng một trong hai |
| `category` | ID MongoDB, `categories.slug` hoặc `categories.id` (ví dụ `co-phuc`) |
| `min_price`, `max_price` | Số nguyên VND không âm, có thể dùng riêng hoặc kết hợp |
| `color` | Tên màu khớp `variants[].color` (không phân biệt hoa-thường), không phải mã hex khi có SKU |
| `size` | Size khớp `variants[].size` |
| `material` | Từ khóa khớp một phần `products.material` |
| `sort` | `newest` (mặc định), `price_asc`, `price_desc`, `name_asc`, `name_desc`; hỗ trợ `price-asc` và `price-desc` như UI cũ |
| `page` | Mặc định 1; từ 1 đến 100000 |
| `page_size` | Mặc định 12; từ 1 đến 50 |

**Lưu ý chọn SKU:** Nếu cùng lọc `color` và `size`, sản phẩm có biến thể chỉ khớp khi **một phần tử trong `variants[]`** có cả hai giá trị. Không ghép chéo từ hai SKU. Sản phẩm không có biến thể có thể lọc theo `colors[].name`/`colors[].hex` và `sizes[]` để xem nội dung may đo, nhưng không suy diễn SKU may sẵn.

Response ví dụ minh họa:

```json
{
  "data": [{
    "id": "7e0000000000000000000001",
    "category_id": "7f0000000000000000000001",
    "slug": "pd01-ao-dai-01",
    "name": "Áo Dài Lụa Ngà Lập Lĩnh",
    "price": 2950000,
    "material": "Lụa Tơ Tằm",
    "colors": [{"name":"Ngà","hex":"#FAF5EE"}],
    "sizes": ["M", "L"],
    "stock": 5,
    "ready_available": true,
    "bespoke": true
  }],
  "total": 16,
  "page": 1,
  "page_size": 1,
  "total_pages": 16
}
```

`total` đếm sau lọc và trước phân trang (con số ví dụ trên chỉ là ví dụ khi **không có sản phẩm nào khác**). `data` là mảng rỗng khi không có kết quả hoặc trang vượt quá tổng, `total_pages=0` nếu total=0. Các mảng trong response không trả `null` để client dễ xử lý.

Thứ tự luôn có `_id` làm khóa phụ để ổn định khi giá/tên/ngày giống nhau. Phân trang offset không đảm bảo snapshot cố định nếu dữ liệu thay đổi ngay giữa hai request.

### GET `/api/v1/products/{slug}`

Trả `{ "data": { ... } }` với thông tin catalog, `variants` (gồm `sku`, `color`, `size`, `stock`, `available`), `story`, `fabric`, `craft`, `gallery`, `accessories`, `trust`, `reviews: {"count":0,"average":0}`. Số liệu review được tổng hợp từ collection `reviews` theo `product_id`; **không dựa trên số `rating`/`review_count` lưu sẵn**. Các phần nội dung không có dữ liệu sẽ bị bỏ qua hoặc trả mảng rỗng; không sinh chuyện mẫu.

`ready_available` chỉ đúng khi `stock > 0` **và** ít nhất một SKU có tồn > 0 (hoặc có `sku` gốc và không có variants). `bespoke` độc lập, thể hiện khả năng yêu cầu may đo. Lưu ý: **giỏ hàng/checkout phải kiểm tra lại cùng SKU và tồn kho tại thời điểm tạo đơn**; API đọc không giữ hàng.

### Mã lỗi

| HTTP | code | Khi nào |
| --- | --- | --- |
| 400 | `INVALID_QUERY` | Tham số lạ, lặp, không hợp lệ; giá âm hoặc min > max; page/page_size sai |
| 400 | `INVALID_SLUG` | Slug sai định dạng |
| 404 | `PRODUCT_NOT_FOUND` | Không có slug, sản phẩm tắt hoặc danh mục/cha tắt |
| 500 | `CATALOG_UNAVAILABLE` | Truy vấn MongoDB lỗi |

Ví dụ:

```json
{"error":{"code":"INVALID_QUERY","message":"page_size phải là số nguyên từ 1 đến 50","field":"page_size"}}
```

## Dữ liệu mẫu tự chọn (không chạy tự động)

Migration ban đầu `000003_demo_seed.up.json` có sẵn đồ thời trang thông thường; **PD-01 không ghi đè hoặc xóa dữ liệu đó**. Để thử đủ case của Aodora, chạy Mongo shell (ở `server/`):

```bash
mongosh "mongodb://aodora:aodora@localhost:27017/QuanLyBanQuanAo?authSource=admin" --file scripts/pd01_demo_seed.js
```

Nếu máy không có `mongosh` nhưng đã chạy Docker Compose:

```bash
docker compose cp scripts/pd01_demo_seed.js mongo:/tmp/pd01_demo_seed.js
docker compose exec mongo mongosh -u aodora -p aodora --authenticationDatabase admin --file /tmp/pd01_demo_seed.js
```

Dữ liệu này tạo 5 danh mục (4 hiện, 1 ẩn); 18 mẫu (16 hiện, 1 tắt, 1 thuộc danh mục ẩn). Một số SKU hết hàng, trường hợp `Ngà/M` và `Đỏ/L` tồn tại nhưng `Ngà/L` không tồn tại; có story/fabric/craft riêng cho từng mẫu. IDs và slug cố định nên chạy lại không nhân bản, không ghi đè mẫu đã sửa. Ảnh demo mượn từ file dữ liệu tĩnh sẵn trong client; **không phải ảnh sản phẩm thật**. Dùng trên database test thay vì database thật.

Xóa đúng dữ liệu mẫu (không xóa seed gốc):

```bash
mongosh "mongodb://aodora:aodora@localhost:27017/QuanLyBanQuanAo?authSource=admin" --file scripts/pd01_demo_clear.js
```

Script mẫu cố định database `QuanLyBanQuanAo`; nếu đổi `MONGO_DB`, đổi tên DB trong script tương ứng.

## Test thủ công / nghiệm thu

Sau khi `go run ./cmd/api` đã lên (PowerShell dùng `curl.exe` hoặc `Invoke-RestMethod`):

```bash
curl "http://localhost:8080/api/v1/categories"
curl "http://localhost:8080/api/v1/products?page=1&page_size=5"
curl "http://localhost:8080/api/v1/products?page=2&page_size=5&sort=price-asc"
curl "http://localhost:8080/api/v1/products?search=Lụa&category=co-phuc&min_price=1000000&max_price=4000000"
curl "http://localhost:8080/api/v1/products?color=Ngà&size=M"
curl "http://localhost:8080/api/v1/products?color=Ngà&size=L"
curl "http://localhost:8080/api/v1/products/pd01-ao-dai-01"
curl -i "http://localhost:8080/api/v1/products/pd01-ao-dai-17"
curl -i "http://localhost:8080/api/v1/products/pd01-ao-dai-18"
curl -i "http://localhost:8080/api/v1/products?page=0"
```

- Đối chiếu số `total` với `db.products.countDocuments(...)` theo filter và category active (bao gồm cha).
- Hai request `Ngà+M` và `Ngà+L` **không được** cho cùng một bộ SKU nếu trong một sản phẩm chỉ có cặp `Ngà/M`, `Đỏ/L`.
- Check `page=1` / `page=2`: không trùng ID khi DB không thay đổi.
- `pd01-ao-dai-17` tắt bán và `pd01-ao-dai-18` thuộc danh mục ẩn đều trả 404.
- Detail từng slug có story theo đúng tên, `stock`, SKU và mảng review tổng hợp; không có email/password_hash/đơn/số đo.
- Chạy `go test -race ./...` ở `server/` (đây là test code; test MongoDB thật qua các lệnh bên trên).

## Bàn giao sang PD-02/KR-03/DW-02

- Dùng `id` (ObjectId hex), `slug`, `category_id`, `price`, `image`/`images`, `colors`, `sizes`, `variants[]` từ API; không sử dụng mảng giả `client/src/data/products.ts` làm nguồn chính khi triển khai PD-02.
- Với checkout, chỉ lưu lựa chọn hợp lệ theo **`product.id + variant.sku + quantity`**; server checkout phải lấy giá/tồn từ MongoDB, cập nhật nguyên tử (do DW-02 phụ trách).
- Cần thống nhất Q-07: `products.stock` là tổng tồn may sẵn, `variants[].stock` tồn từng SKU; API công khai chỉ thông báo còn hàng khi cả tổng và SKU > 0. Khi tạo/hủy đơn phải cập nhật nhất quán các số này.
- API này không chỉnh sửa index. Tìm kiếm tên/mô tả hiện dùng regex khớp một phần và không lợi dụng text index; nếu DB lớn cần tối ưu qua chiến lược tìm kiếm và index phù hợp sau khi thống nhất.
