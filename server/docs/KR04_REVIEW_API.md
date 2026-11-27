# KR-04 — Review sau nhận hàng và quản lý review

Khách có tài khoản đánh giá sản phẩm sau khi nhận hàng. Khách vãng lai chỉ xem review. Chủ cửa hàng xử lý review không phù hợp.

## Quyết định

| Mã | Trạng thái | Nội dung |
|---|---|---|
| Q-03 | **Đã chốt** | Gửi review **bắt buộc đăng nhập**. Khách vãng lai (không tài khoản) chỉ có quyền xem review công khai. Đơn đặt không cần tài khoản không được dùng để review, kể cả khi sau đó khách đăng nhập, vì đơn đó không gắn với tài khoản. |
| Q-06 | **Đã chốt** | Admin có hai thao tác, đều phải xác nhận qua popup và được ghi log `admin_id`/`review_id`: **Ẩn/Hiện lại** (field `hidden`, đảo ngược được) và **Xóa vĩnh viễn**. Review bị ẩn không hiện công khai, không được tính vào điểm, nhưng tác giả vẫn thấy (kèm ghi chú đang bị ẩn); tác giả sửa review thì review vẫn bị ẩn. Không có luồng duyệt trước khi đăng. |
| Q-08 | Chờ xác nhận | Ảnh review nhận theo `src` (URL http/https) và `alt` bắt buộc, tối đa 5 ảnh, không upload file. Giống form ảnh sản phẩm của PD-03. |

## Public reviews (không cần đăng nhập)

```text
GET /api/v1/products/{slug}/reviews?page=1&page_size=6
```

```json
{ "data": { "items": [], "total": 0, "page": 1, "page_size": 6, "average": 0 } }
```

Response công khai không có email, số điện thoại, customer ID, order ID hay mã truy cập.

## Review của khách tài khoản (bắt buộc session)

```text
GET /api/v1/me/reviews?product_id={product_id}
GET /api/v1/me/reviews/eligible-orders?product_id={product_id}
GET /api/v1/me/reviews/status?product_id={a}&product_id={b}
PUT /api/v1/me/reviews/{product_id}
```

```json
{ "order_id": "...", "rating": 5, "content": "Nhận xét", "images": [{ "src": "https://...", "alt": "Ảnh sản phẩm" }] }
```

- Không có session hợp lệ thì trả `401 AUTH_REQUIRED`, kể cả khi gửi kèm mã truy cập của đơn guest.
- `eligible-orders` duyệt hết các trang đơn của session hiện tại (tối đa 5.000 đơn), chỉ trả đơn `delivered` có sản phẩm tương ứng.
- Server gọi `VerifiedPurchase` với customer ID lấy từ session. Đơn của người khác, đơn guest, đơn chưa giao hoặc đã hủy, hoặc đơn không có sản phẩm đều trả `403 REVIEW_NOT_ELIGIBLE`.
- `customer_id` và `verified_purchase` do client gửi lên bị từ chối (`DisallowUnknownFields`).
- Mỗi khách có một review cho một sản phẩm; gửi lại sẽ cập nhật review đó.
- `status` trả `[{ "product_id", "slug", "reviewed" }]` theo đúng thứ tự gửi lên (bỏ trùng, tối đa 200 ID). `slug` rỗng khi sản phẩm không còn hiển thị công khai. Trang `/tai-khoan/don-hang` dùng API này để hiện nút **Đánh giá** cho từng mẫu trong đơn đã nhận áo và đã thu đủ tiền; mẫu đã review thì nút bị vô hiệu hóa.

## Quản trị (chỉ admin)

```text
GET    /api/v1/admin/reviews?page=1&page_size=20&rating=1..5&hidden=true|false&product_id=...
PATCH  /api/v1/admin/reviews/{id}    body {"hidden": true|false}
DELETE /api/v1/admin/reviews/{id}
```

List trả `{ data, total, page, page_size }`. Mỗi item gồm tên/slug sản phẩm, `order_number` và `hidden`, không có email, số điện thoại hay customer ID. Màn hình quản trị ở `/admin/reviews`.

## Số tổng hợp `products.rating` / `review_count`

- Mỗi lần thêm, sửa hay xóa review, server ghi review và tính lại `rating`/`review_count` trong **cùng một transaction** MongoDB. Nếu hai lần ghi cùng sản phẩm chạy song song, MongoDB báo write conflict trên document sản phẩm và `WithTransaction` chạy lại lần thua, nên số tổng hợp không bị lệch.
- Chỉ review đang hiện (`hidden` khác `true`) được tính. Điểm trung bình làm tròn 2 chữ số. Không có review đang hiện thì cả hai bằng 0. Ẩn, hiện lại hay xóa đều tính lại trong cùng transaction.
- Sửa điểm không làm tăng `review_count`. Trang chi tiết sản phẩm và danh sách review công khai dùng cùng một cách tính.
- Form admin catalog (PD-03) không ghi hai field này.

## Migration `1c8a7827-…_review_summary_recompute`

Không đổi validator hay index của `reviews`: `customer_id` vẫn bắt buộc và `uq_reviews_customer_product` giữ nguyên.

Up tính lại `rating`/`review_count` của mọi sản phẩm từ `reviews`, để số đang lưu (seed hoặc nhập tay) khớp với review thật. Sản phẩm không có review về 0/0. Down không làm gì vì đây là dữ liệu dẫn xuất.

## Migration `8dfd63fe-…_review_hidden`

Thêm `hidden: bool` vào validator `reviews`. Review cũ không có field này được coi là đang hiện, nên không phải sửa dữ liệu. Down khôi phục validator trước đó. Validator cũ không cấm field lạ nên rollback vẫn chạy được khi đã có review bị ẩn.

## Kiểm thử và bằng chứng nghiệm thu

```bash
cd server
make transactions-up
export MONGO_TEST_URI='mongodb://<user>:<pass>@127.0.0.1:27018/?authSource=admin&directConnection=true'
go test -race ./internal/application/review
go test -race -tags=integration ./internal/infrastructure/migration -run TestReviewSummaryRecomputeMigration -count=1 -v
go test -race -tags=integration ./internal/interfaces/http -run TestReviewHTTPAcceptanceIntegration -count=1 -v
cd ../client && npm run test:reviews
```

Dữ liệu kiểm thử (tài khoản A, B, admin; đơn delivered, pending, shipping, cancelled, đơn không có sản phẩm, đơn guest đã giao) được tạo trong `review_http_integration_test.go`, trên database tạm và tự xóa sau test.

| Nghiệm thu | Subtest |
|---|---|
| Đơn chưa giao, đã hủy hoặc không có mẫu bị từ chối | `orders_that_are_not_delivered…` |
| Khách A không review hay sửa review thay khách B | `customer_B_cannot_review_or_edit_as_customer_A` |
| Khách vãng lai chỉ xem, không gửi được review (Q-03) | `guests_without_an_account_can_read_but_not_write_reviews` |
| Rating ngoài 1–5 bị từ chối; gửi lại không tạo bản trùng | `rating_outside_1-5…`, `4_and_5_aggregate…`, `concurrent_submissions…` |
| 4 và 5 cho 4,5 với số lượng 2; sửa điểm không tăng số lượng | `4_and_5_aggregate_to_4.5_with_count_2…` |
| Admin xử lý theo phương án đã chốt; số tổng hợp khớp danh sách công khai | `admin_hides_and_shows_a_review…`, `admin_lists_and_deletes_reviews…` (sau mỗi bước so `products`, list công khai và trang chi tiết) |
