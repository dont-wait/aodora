# KR-02 — Hồ sơ khách hàng, địa chỉ, số đo và yêu thích

Các endpoint dưới đây yêu cầu session hợp lệ từ KR-01. `customer_id` luôn lấy từ session, không nhận từ request.

## Hồ sơ

| Method | Endpoint | Ý nghĩa |
|---|---|---|
| GET | `/api/v1/me/profile` | Đọc họ tên, email, điện thoại, địa chỉ và số đo |
| PATCH | `/api/v1/me/profile` | Sửa họ tên và điện thoại |

`email`, `role` và `password_hash` không thể sửa qua contract này.

## Địa chỉ

| Method | Endpoint | Ý nghĩa |
|---|---|---|
| GET | `/api/v1/me/addresses` | Danh sách địa chỉ |
| POST | `/api/v1/me/addresses` | Thêm địa chỉ |
| PUT | `/api/v1/me/addresses/{id}` | Sửa địa chỉ hoặc đặt mặc định |
| DELETE | `/api/v1/me/addresses/{id}` | Xóa địa chỉ |

Địa chỉ đầu tiên tự là mặc định. Tại mọi thời điểm tối đa một địa chỉ có `is_default=true`. Địa chỉ cũ chưa có ID được cấp ID `legacy-{index}` khi đọc và được chuẩn hóa khi có thao tác ghi.

## Số đo

| Method | Endpoint | Ý nghĩa |
|---|---|---|
| GET | `/api/v1/me/measurements` | Đọc hồ sơ số đo |
| PUT | `/api/v1/me/measurements` | Thay thế hồ sơ số đo và ghi chú |

Các field `_cm` là số không âm. Field bỏ trống được xóa, không tự chuyển thành `0`.

## Yêu thích

| Method | Endpoint | Ý nghĩa |
|---|---|---|
| GET | `/api/v1/me/wishlist` | Đọc danh sách mẫu đã lưu |
| PUT | `/api/v1/me/wishlist/{product_id}` | Thêm sản phẩm đang công khai |
| DELETE | `/api/v1/me/wishlist/{product_id}` | Bỏ sản phẩm |

Thêm dùng `$addToSet`, nên thao tác lặp không tạo ID trùng. Sản phẩm/danh mục không còn công khai được trả dưới dạng `unavailable` để người dùng vẫn bỏ lưu được mà không lộ thông tin sản phẩm.
