# PD-03 — Quản trị danh mục và thông tin bán áo

## Phạm vi

Chủ cửa hàng quản lý danh mục, thông tin bán áo, giá, ảnh, phụ kiện, SKU và tồn kho trên hai collection MongoDB hiện có: `categories` và `products`. Không tạo collection mới. Không ghi vào `orders`, `reviews`, `carts` hoặc dữ liệu khách hàng.

**Xác thực:** Mọi URL `/api/v1/admin/...` phải được server xác minh là admin theo KR-01. Nếu chưa cung cấp `AdminAuthorizer` đáng tin cậy, hệ thống từ chối HTTP 403, kể cả GET. Không chấp nhận một header `X-Role`, ID tài khoản do client tự gửi hoặc mã truy cập gán cứng để suy ra quyền admin. Phần tích hợp xác thực còn phụ thuộc KR-01.

## Danh mục

| Method | URL | Ý nghĩa |
| --- | --- | --- |
| GET | `/api/v1/admin/categories` | Danh sách tất cả danh mục, kể cả đang ẩn |
| POST | `/api/v1/admin/categories` | Tạo danh mục |
| PUT | `/api/v1/admin/categories/{id}` | Thay toàn bộ metadata danh mục |

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

Danh mục ẩn không được hiển thị trên API công khai. Kiểm tra danh mục cha tồn tại và từ chối tham chiếu tạo vòng lặp; không xóa cứng danh mục để tránh mồ côi sản phẩm.

## Sản phẩm

Danh sách sản phẩm quản trị có phân trang, mặc định 50 và tối đa 100 sản phẩm/trang. Response dạng `{"data":{"items":[...],"total":0,"page":1,"page_size":50,"total_pages":0}}`.

| Method | URL | Ý nghĩa |
| --- | --- | --- |
| GET | `/api/v1/admin/products?page=1&page_size=50` | Xem sản phẩm, kể cả tắt bán (1–100 dòng/trang) |
| POST | `/api/v1/admin/products` | Tạo sản phẩm, bao gồm danh sách SKU khởi tạo |
| PUT | `/api/v1/admin/products/{id}` | Thay metadata sản phẩm; **không thay đổi tồn kho và danh sách SKU** |
| POST | `/api/v1/admin/products/{id}/variants` | Thêm SKU mới với `stock=0` |
| PATCH | `/api/v1/admin/products/{id}/variants/{sku}` | Sửa màu/cỡ của SKU, không thay tồn |
| POST | `/api/v1/admin/products/{id}/inventory/{sku}` | Điều chỉnh tồn kho theo chênh lệch có điều kiện |

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

Giá, giá gốc, số lượng là số nguyên VND hoặc số nguyên sản phẩm; không được âm. `slug` phải hợp lệ và duy nhất theo index gốc. Mỗi SKU phải có mã, màu, kích cỡ và tồn kho; SKU hoặc tổ hợp màu/cỡ không được trùng trong cùng một sản phẩm. Có tối thiểu một SKU khi tạo. Ảnh quản lý theo đường dẫn URL và mô tả `alt`; chưa tích hợp tải file lên server ở bước này. `colors[].hex` chỉ nhận `#RRGGBB` khi được cung cấp. Mọi field ngoài hợp đồng request bị từ chối, bao gồm `rating`, `review_count`, `stock` và `variants` trong PUT metadata.

### Tồn kho và phối hợp DW-02

`products.stock` là tổng tồn SKU và `products.variants[].stock` là tồn của từng SKU. Lúc tạo sản phẩm, tổng được tính từ SKU. Sau đó, chỉ cập nhật tồn qua endpoint delta:

```json
{ "delta": -2 }
```

Điều chỉnh được thực hiện bằng **một MongoDB atomic conditional `$inc`** lên cả `products.stock` và `variants.$.stock`, có điều kiện tìm đúng SKU và đảm bảo cả hai số đủ lớn khi giảm. Nếu không khớp hoặc không đủ tồn, trả `409 CONFLICT`. Lưu form thông tin sản phẩm không được `$set` vào `stock`, `variants` hoặc rating; điều này ngăn một bản form mở lâu ghi đè số vừa được trừ khi tạo đơn.

**Thỏa thuận Q-07 cần được Dontwait xác nhận:** Khi giữ hàng/hủy đơn, DW-02 phải dùng phép cập nhật nguyên tử tương đương cho cùng document và SKU, không thay toàn bộ `variants` dựa trên bản dữ liệu cũ. Giá và tên sản phẩm trong `orders.items` phải là snapshot tại thời điểm đặt, không đọc lại từ sản phẩm mỗi lần hiển thị đơn.

### Phạm vi tính nhất quán

MongoDB bảo đảm thao tác cập nhật tồn trong một document là nguyên tử. Kiểm tra cấu trúc danh mục cha trước PUT và đọc trạng thái danh mục với sản phẩm trong API công khai không phải giao dịch serializable cho mọi cập nhật đồng thời. Nếu phát sinh thay đổi quan hệ danh mục song song, cần thống nhất cơ chế transaction hoặc optimistic concurrency trong giai đoạn tích hợp nhiều người quản trị.

### Lỗi và phân quyền

- `400 INVALID_FIELD`: thiếu/sai trường, giá âm, URL ảnh không hợp lệ, nội dung không thuộc schema.
- `403 ADMIN_REQUIRED`: chưa được xác minh quyền admin server-side.
- `404 NOT_FOUND`: ID không tồn tại.
- `409 CONFLICT`: slug trùng, SKU trùng, không đủ tồn, thao tác đồng thời không khớp điều kiện.
- `500 ADMIN_CATALOG_UNAVAILABLE`: lỗi cơ sở dữ liệu; thông tin nội bộ chỉ ghi ở log đã che URI có mật khẩu.

PD-04 tiếp tục phụ trách `products.story`, `fabric` và `craft`, không chỉnh nội dung này trong form PD-03. KR-04 phụ trách dữ liệu tổng hợp đánh giá; PD-03 không cập nhật các field liên quan.

Điều chỉnh tồn bằng delta là thao tác **không idempotent**: client không tự động gửi lại request khi chưa biết kết quả; cần đọc lại tồn trước khi thao tác tiếp.
