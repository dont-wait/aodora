# KR-03 — Giỏ hàng

## Guest cart

Khách chưa đăng nhập lưu lựa chọn trong trình duyệt và có thể kiểm tra lại bằng:

```text
POST /api/v1/cart/validate
```

Request:

```json
{
  "items": [
    {
      "product_id": "670000000000000000000001",
      "product_slug": "ao-dai-mau",
      "sku": "AO-DO-M",
      "color": "Đỏ",
      "size": "M",
      "quantity": 1,
      "fulfillment_mode": "ready"
    }
  ]
}
```

Giá, tồn kho, SKU và visibility của sản phẩm đều được backend đọc lại.

## Account cart

Các endpoint yêu cầu session KR-01:

| Method | Endpoint | Ý nghĩa |
|---|---|---|
| GET | `/api/v1/me/cart` | Đọc giỏ |
| POST | `/api/v1/me/cart/items` | Thêm hoặc cộng dòng cùng SKU |
| PUT | `/api/v1/me/cart/items/{product_id}/{sku}` | Đổi số lượng |
| DELETE | `/api/v1/me/cart/items/{product_id}/{sku}` | Xóa dòng |
| POST | `/api/v1/me/cart/merge` | Hợp nhất guest cart sau đăng nhập |

`product_id + sku` là khóa của một dòng. Request merge có `request_id`; gửi lại cùng mã không cộng số lượng lần nữa. Các thao tác ghi dùng `version` của document cart để phát hiện cập nhật đồng thời.

Response có `items`, `issues`, `item_count` và `subtotal`. Một item có thể có issue `PRODUCT_UNAVAILABLE`, `SKU_NOT_FOUND`, `OUT_OF_STOCK`, `QUANTITY_EXCEEDS_STOCK` hoặc `PRICE_CHANGED`.

KR-03 không tạo đơn, không trừ tồn và không xử lý thanh toán. Checkout phải revalidate lại dữ liệu trước khi tạo order.
