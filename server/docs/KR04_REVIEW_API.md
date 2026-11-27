# KR-04 — Review sau nhận hàng và quản lý review

Hỗ trợ khách tài khoản, khách vãng lai và thao tác xử lý của chủ cửa hàng.

## Phương án đã áp dụng (chờ nhóm/chủ cửa hàng xác nhận)

| Mã | Phương án | Lý do |
|---|---|---|
| Q-03 | Khách vãng lai chứng minh quyền bằng mã truy cập riêng của đơn do DW-03 cấp lúc đặt (`X-Order-Access-Token`, server chỉ lưu hash). Review lưu `order_id`, không có `customer_id`. Mỗi đơn chỉ có một review cho một sản phẩm; gửi lại sẽ cập nhật review cũ. | Dùng lại cơ chế đã có, không cần dịch vụ ngoài. Mã đơn công khai (`order_number`) không cấp quyền. |
| Q-06 | Admin xử lý review không phù hợp bằng cách **xóa vĩnh viễn**. Không thêm trạng thái ẩn/duyệt. Mỗi lần xóa được ghi log với `admin_id` và `review_id`. | Bám đúng DB hiện có; task yêu cầu không tự thêm luồng duyệt. Nếu chủ cửa hàng muốn ẩn thay vì xóa thì phải thêm field và migration riêng. |
| Q-08 | Ảnh review nhận theo `src` (URL http/https) và `alt` bắt buộc, tối đa 5 ảnh. Không upload file. | Giống cấu trúc `images[]` của sản phẩm; không thêm hạ tầng lưu trữ. |

## Public reviews

```text
GET /api/v1/products/{slug}/reviews?page=1&page_size=6
```

```json
{ "data": { "items": [], "total": 0, "page": 1, "page_size": 6, "average": 0 } }
```

Response công khai không có email, số điện thoại, customer ID, order ID hay mã truy cập.

## Review của khách tài khoản

```text
GET /api/v1/me/reviews?product_id={product_id}
GET /api/v1/me/reviews/eligible-orders?product_id={product_id}
PUT /api/v1/me/reviews/{product_id}
```

```json
{ "order_id": "...", "rating": 5, "content": "Nhận xét", "images": [{ "src": "https://...", "alt": "Ảnh sản phẩm" }] }
```

`eligible-orders` duyệt hết các trang đơn của session hiện tại (tối đa 5.000 đơn), chỉ trả đơn `delivered` có sản phẩm tương ứng. Server gọi `VerifiedPurchase` với customer ID lấy từ session. `customer_id` và `verified_purchase` do client gửi lên bị từ chối (`DisallowUnknownFields`). Mỗi khách có một review cho một sản phẩm; gửi lại sẽ cập nhật review đó.

## Review của khách vãng lai

```text
GET /api/v1/guest-orders/{order_id}/reviews
PUT /api/v1/guest-orders/{order_id}/reviews/{product_id}
Header: X-Order-Access-Token: <mã của đơn, lưu ở trình duyệt lúc đặt>
```

```json
{ "rating": 5, "content": "Nhận xét", "images": [] }
```

| Trường hợp | Kết quả |
|---|---|
| Thiếu mã hoặc mã sai định dạng | `400 INVALID_FIELD` (`X-Order-Access-Token`) |
| Mã không khớp đơn, hoặc là đơn của tài khoản | `404 ORDER_NOT_FOUND` (không lộ đơn có tồn tại hay không) |
| Đơn chưa giao, đã hủy, hoặc không có sản phẩm này | `403 REVIEW_NOT_ELIGIBLE` |

Tên hiển thị lấy từ `shipping_address.recipient` của đơn. Phía client, form nằm ở trang `/don-hang/{id}` khi đơn đã giao và trình duyệt còn giữ mã truy cập.

## Quản trị (chỉ admin)

```text
GET    /api/v1/admin/reviews?page=1&page_size=20&rating=1..5&product_id=...
DELETE /api/v1/admin/reviews/{id}
```

List trả `{ data, total, page, page_size }`. Mỗi item gồm tên/slug sản phẩm, `source` (`account`/`guest`) và `order_number`, không có email, số điện thoại hay mã truy cập. Màn hình quản trị ở `/admin/reviews`.

## Số tổng hợp `products.rating` / `review_count`

- Mỗi lần thêm, sửa hay xóa review, server ghi review và tính lại `rating`/`review_count` trong **cùng một transaction** MongoDB. Nếu hai lần ghi cùng sản phẩm chạy song song, MongoDB báo write conflict trên document sản phẩm và `WithTransaction` chạy lại lần thua, nên số tổng hợp không bị lệch.
- Mọi review được tính, kể cả review cũ chưa có `verified_purchase`. Điểm trung bình làm tròn 2 chữ số. Không có review thì cả hai bằng 0.
- Sửa điểm không làm tăng `review_count`. Trang chi tiết sản phẩm và danh sách review công khai dùng cùng một cách tính.
- Form admin catalog (PD-03) không ghi hai field này.

## Migration `4eaaa779-…_review_guest_access`

Up:

1. Validator `reviews`: `customer_id` không còn bắt buộc, bắt buộc có `customer_id` hoặc `order_id` (`anyOf`), thêm `order_id: objectId`.
2. `uq_reviews_customer_product` đổi thành partial index (chỉ khi `customer_id` là objectId), để nhiều review guest không có `customer_id` không đụng nhau.
3. Thêm `uq_reviews_order_product` (partial khi có `order_id`).
4. Tính lại `rating`/`review_count` của mọi sản phẩm từ `reviews`. Sản phẩm không có review về 0/0.

Down khôi phục validator và index cũ. Runner từ chối rollback khi còn review guest (validator cũ bắt buộc `customer_id`) và không thay đổi gì. Down không khôi phục số tổng hợp cũ vì đó là dữ liệu dẫn xuất.

## Kiểm thử và bằng chứng nghiệm thu

```bash
cd server
make transactions-up
export MONGO_TEST_URI='mongodb://<user>:<pass>@127.0.0.1:27018/?authSource=admin&directConnection=true'
go test -race ./internal/application/review
go test -race -tags=integration ./internal/infrastructure/migration -run TestReviewGuestAccessMigration -count=1 -v
go test -race -tags=integration ./internal/interfaces/http -run TestReviewHTTPAcceptanceIntegration -count=1 -v
cd ../client && npm run test:reviews
```

Dữ liệu kiểm thử (tài khoản A, B, admin; đơn delivered, pending, shipping, cancelled, đơn không có sản phẩm, đơn guest delivered/pending) được tạo trong `review_http_integration_test.go`, trên database tạm và tự xóa sau test.

| Nghiệm thu | Subtest |
|---|---|
| Đơn chưa giao, đã hủy hoặc không có mẫu bị từ chối | `orders_that_are_not_delivered…` và phần guest |
| Khách A không review hay sửa review thay khách B | `customer_B_cannot_review_or_edit_as_customer_A` |
| Khách vãng lai có mã hợp lệ đánh giá đúng mẫu trong đơn | `guest_with_the_order_token…` |
| Rating ngoài 1–5 bị từ chối; gửi lại không tạo bản trùng | `rating_outside_1-5…`, `4_and_5_aggregate…`, `concurrent_submissions…` |
| 4 và 5 cho 4,5 với số lượng 2; sửa điểm không tăng số lượng | `4_and_5_aggregate_to_4.5_with_count_2…` |
| Admin xử lý theo phương án đã chốt; số tổng hợp khớp danh sách công khai | `admin_lists_and_deletes_reviews…` (sau mỗi bước so `products`, list công khai và trang chi tiết) |
