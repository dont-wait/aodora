# API danh mục và sản phẩm — Aodora

## 1. Mục đích và phạm vi

Nhóm API catalog cung cấp dữ liệu **đọc công khai** cho trang chủ, bộ sưu tập và trang chi tiết áo dài của Aodora. Nguồn dữ liệu là các collection MongoDB `categories`, `products`, `reviews` của database được cấu hình qua `MONGO_DB`. API không tạo sản phẩm, thay đổi giá, đặt giữ hàng hoặc thay đổi tồn kho.

| Phương thức | Đường dẫn | Ý nghĩa |
| --- | --- | --- |
| `GET` | `/api/v1/categories` | Danh mục đang hiển thị |
| `GET` | `/api/v1/products` | Tìm, lọc, sắp xếp và phân trang sản phẩm |
| `GET` | `/api/v1/products/{slug}` | Chi tiết một mẫu áo, biến thể và đánh giá tổng hợp |

Các endpoint này không yêu cầu đăng nhập; phản hồi sử dụng `application/json` và không bao gồm dữ liệu tài khoản, đơn hàng hoặc thông tin nội bộ.

## 2. Danh mục

`GET /api/v1/categories` trả về:

```json
{
  "data": [
    { "id": "<ObjectId dạng hex>", "slug": "co-phuc", "name": "Cổ phục", "sort_order": 1 }
  ]
}
```

Các trường `key`, `description`, `parent_id`, `image` và `badge` có thể xuất hiện nếu dữ liệu tương ứng tồn tại. Danh mục chỉ được công khai khi `is_active = true`, đồng thời toàn bộ danh mục cha cũng đang hoạt động và tồn tại. Danh mục có quan hệ cha-con bị lỗi hoặc tạo chu trình sẽ được ẩn. Thứ tự sắp xếp theo `sort_order`, `name`, `id`.

## 3. Danh sách sản phẩm

`GET /api/v1/products` hỗ trợ kết hợp các tham số sau:

| Tham số | Quy tắc |
| --- | --- |
| `search` hoặc `q` | Tìm theo **từ khóa** trên `name` và `description` bằng MongoDB text index. Không phải tìm khớp chuỗi con bất kỳ. Chỉ dùng một tham số. |
| `category` | Khớp `_id`, `slug` hoặc `id` của danh mục đang hiển thị |
| `min_price`, `max_price` | Số nguyên VND không âm; `min_price` không vượt `max_price` |
| `color` | Tên màu trong cùng phần tử `variants[]` với size; không phân biệt hoa/thường |
| `size` | Size trong cùng phần tử `variants[]` với màu; không phân biệt hoa/thường |
| `material` | Tên chất liệu khớp nguyên trường (không phân biệt hoa/thường) |
| `sort` | `newest` (mặc định), `price_asc`, `price_desc`, `name_asc`, `name_desc`; có alias `price-asc`, `price-desc` |
| `page` | Trang bắt đầu từ 1, mặc định 1, tối đa 200 |
| `page_size` | Số sản phẩm/trang, mặc định 12, tối đa 50 |

Độ sâu phân trang (`(page-1) * page_size`) tối đa 10.000 mục. Các truy vấn MongoDB có ngân sách tối đa 8 giây và tôn trọng việc client hủy request. Những giới hạn này bảo vệ tài nguyên khi catalog lớn; giao diện nên đưa người dùng về trang đầu khi thay đổi bộ lọc.

**Chọn SKU:** Khi có cả màu và cỡ, một sản phẩm có biến thể chỉ khớp nếu *một* `variants[]` thỏa mãn đồng thời. Không được ghép màu của một SKU với size của SKU khác. Sản phẩm không có `variants[]` có thể đối chiếu `colors` và `sizes` dành cho hiển thị/may đo; API không tự tạo SKU mới.

Response có cấu trúc:

```json
{
  "data": [],
  "total": 0,
  "page": 1,
  "page_size": 12,
  "total_pages": 0
}
```

Danh sách dữ liệu được truy vấn từ MongoDB với `_id` làm khóa phụ khi sắp xếp, để bảo đảm thứ tự ổn định khi các sản phẩm có cùng giá, tên hoặc ngày tạo. Tổng kết quả và trang dữ liệu được tính trong cùng một pipeline aggregation (`$facet`), sau khi áp dụng điều kiện hiển thị danh mục. Tính nhất quán giữa *hai HTTP request độc lập* không được bảo đảm nếu admin cập nhật catalog trong lúc duyệt trang.

Dữ liệu card có `id`, `category_id`, `slug`, `name`, `description`, `material`, `price`, giá so sánh nếu có, `image`, `colors`, `sizes`, `stock`, `ready_available`, `bespoke`, nhãn và thời điểm tạo. Các giá trị đều đọc từ MongoDB; API không gán giá hoặc nội dung mẫu cố định.

## 4. Chi tiết sản phẩm

`GET /api/v1/products/{slug}` chỉ hiển thị sản phẩm `is_active = true` thuộc danh mục hợp lệ đang hoạt động. Trả về `{"data": { ... }}` gồm:

- Thông tin nhận dạng, tên/slug, mô tả, chất liệu, giá và hình ảnh.
- `variants[]` với `sku`, `color`, `size`, `stock`, `available`.
- `colors[]`, `sizes[]`, `ready_available` và `bespoke`.
- Nội dung riêng của sản phẩm nếu có: `story`, `fabric`, `craft`, `gallery`, `trust`, `accessories`.
- `reviews: {"count": ..., "average": ...}` thống kê trực tiếp từ collection `reviews` qua `product_id`.

Mã màu `hex` chỉ xuất hiện khi được lưu trong MongoDB. Nếu chỉ có tên màu từ biến thể, API giữ `name` và bỏ qua `hex`; giao diện không nên dựng swatch khi không có mã HEX hợp lệ. Không được tự suy đoán màu, SKU, giá hoặc nội dung câu chuyện.

`available` của một SKU chỉ đúng khi `products.stock > 0` và `variants[].stock > 0`; `ready_available` đúng khi có ít nhất một SKU còn hàng hoặc có SKU cấp sản phẩm và tồn tổng dương. Thông tin tồn kho chỉ để tham khảo: checkout cần xác minh lại giá và tồn kho bằng thao tác an toàn trên database tại thời điểm tạo đơn.

## 5. Kết quả lỗi

| HTTP | Mã | Ý nghĩa |
| --- | --- | --- |
| 400 | `INVALID_QUERY` | Tham số không hợp lệ, trùng, vượt giới hạn hoặc mâu thuẫn |
| 400 | `INVALID_SLUG` | Slug không hợp lệ |
| 404 | `PRODUCT_NOT_FOUND` | Không có sản phẩm, sản phẩm tắt hoặc danh mục không hiển thị |
| 500 | `CATALOG_UNAVAILABLE` | Lỗi đọc/giải mã dữ liệu từ MongoDB |
| 504 | `CATALOG_TIMEOUT` | Truy vấn vượt ngân sách thời gian |

Lỗi trả theo định dạng `{"error":{"code":"...","message":"...","field":"..."}}`; `field` chỉ xuất hiện khi cần chỉ rõ tham số. Các thông tin lỗi kỹ thuật ghi vào log backend với thao tác tương ứng; response không tiết lộ thông tin kết nối MongoDB.

## 6. MongoDB và hiệu năng

- Text index hiện có trên `products.name` và `products.description` phục vụ tham số `search`/`q`.
- Index trên `is_active`, `category_id`, `created_at`, `price`, `name` và `_id` hỗ trợ các dạng sắp xếp/lọc phổ biến; hiệu quả thực tế cần đánh giá bằng `explain()` trên dữ liệu có quy mô đại diện.
- Một aggregation sử dụng kiểm tra danh mục (`$lookup`, `$graphLookup`) trước khi đếm và phân trang, nhằm tránh lộ sản phẩm thuộc danh mục không còn hiển thị. Quan hệ danh mục cha được xác định theo ObjectId.
- Do MongoDB standalone không tự cung cấp snapshot transaction cho nhiều HTTP request, kết quả các lần truy vấn khác nhau có thể thay đổi khi admin chỉnh catalog. Trường hợp yêu cầu snapshot tuyệt đối xuyên nhiều collection cần cấu hình deployment và read concern phù hợp.
- Không dùng bộ dữ liệu tĩnh ở frontend làm nguồn giá, tồn kho hoặc danh mục; phần quản trị và checkout phải đọc lại trạng thái database trước khi ghi.
