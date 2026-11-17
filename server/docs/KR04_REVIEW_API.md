# KR-04 giai đoạn 1 — Review sau nhận hàng

Giai đoạn này hỗ trợ khách đã đăng nhập. Guest review và admin moderation chưa mở trong contract này.

## Public reviews

```text
GET /api/v1/products/{slug}/reviews?page=1&page_size=6
```

Response:

```json
{
  "data": {
    "items": [],
    "total": 0,
    "page": 1,
    "page_size": 6,
    "average": 0
  }
}
```

Response công khai không có email, số điện thoại, customer ID, order ID hoặc access token.

## Review của khách đăng nhập

```text
GET /api/v1/me/reviews?product_id={product_id}
GET /api/v1/me/reviews/eligible-orders?product_id={product_id}
PUT /api/v1/me/reviews/{product_id}
```

`eligible-orders` chỉ trả các đơn thuộc session hiện tại, trạng thái `delivered` và có sản phẩm tương ứng.

Request ghi:

```json
{
  "order_id": "...",
  "rating": 5,
  "content": "Nhận xét của khách hàng",
  "images": [{ "src": "https://...", "alt": "Ảnh sản phẩm" }]
}
```

Server gọi `VerifiedPurchase` để kiểm tra đơn đã giao và sản phẩm nằm trong đơn. Không tin `customer_id` hoặc `verified_purchase` từ client. Một customer chỉ có một review cho một product; gửi lại sẽ cập nhật review cũ.

Review dùng collection/index hiện có. `order_id`, `content`, `display_name` và `verified_purchase` là field bổ sung được lưu cho review mới; guest review và moderation sẽ có migration/contract riêng ở giai đoạn sau.
