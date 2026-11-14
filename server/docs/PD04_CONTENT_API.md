# PD-04 — API biên tập câu chuyện sản phẩm

## Phạm vi

Nội dung biên tập nằm ngay trong ba field của collection `products`: `story`, `fabric`, `craft`. Không tạo collection mới và không cập nhật tồn kho, giá, SKU, đơn hàng hoặc đánh giá trong các endpoint nội dung.

API chi tiết công khai `GET /api/v1/products/{slug}` đã đọc riêng ba field này. Không trả nội dung mẫu nếu sản phẩm không có dữ liệu. Quy tắc ẩn sản phẩm/danh mục từ PD-01 vẫn có hiệu lực.

## API dành cho chủ cửa hàng

| Phương thức | Endpoint | Ý nghĩa |
|---|---|---|
| GET | `/api/v1/admin/products/{id}/content` | Đọc nội dung và `version` theo ObjectID |
| PUT | `/api/v1/admin/products/{id}/content` | Thay thế ba phần nội dung, có kiểm soát phiên bản |

Cả hai endpoint chỉ dành cho admin được **KR-01 xác thực và phân quyền ở backend**. Khi chưa có KR-01, hệ thống từ chối với HTTP `403 ADMIN_REQUIRED`. Cấu hình CORS với `ADMIN_ALLOWED_ORIGIN` giữ nguyên theo PD-03, không biến preflight thành quyền đăng nhập.

### GET response

```json
{
  "data": {
    "id": "670000000000000000000001",
    "name": "Tên sản phẩm do cửa hàng nhập",
    "version": 1,
    "story": null,
    "fabric": null,
    "craft": null
  }
}
```

`null` ở đây biểu thị phần nội dung chưa có; không phải nội dung được tạo tự động.

### PUT request

```json
{
  "expected_version": 1,
  "story": {
    "eyebrow": "",
    "title": "",
    "paragraphs": ["Đoạn nội dung do chủ cửa hàng nhập"],
    "image": null
  },
  "fabric": {
    "eyebrow": "",
    "title": "",
    "description": "",
    "image": null,
    "care": [{ "title": "Hướng dẫn", "description": "Nội dung do chủ cửa hàng nhập" }]
  },
  "craft": {
    "eyebrow": "",
    "title": "",
    "description": "",
    "image": null,
    "steps": [{ "title": "Tên bước", "description": "Nội dung bước" }]
  }
}
```

Ví dụ thể hiện **hình dạng JSON**, không phải dữ liệu được lưu sẵn. `story`, `fabric`, `craft` phải cùng xuất hiện, có thể gửi `null` để chủ động xóa một phần. `expected_version` bắt buộc là số nguyên không âm, lấy từ lần đọc mới nhất.

Lệnh ghi chỉ `$set` hoặc `$unset` các field nội dung và `updated_at`, đồng thời tăng `version` nguyên tử. Trường `stock`, `variants`, `images`, `price`, `rating` và `review_count` không thể được ghi qua contract này. Lưu từ một tab cũ với version không còn khớp trả `409 CONFLICT`; frontend yêu cầu tải lại trước khi lưu tiếp.

## Kiểm tra đầu vào và lỗi

- Story: tối đa 30 đoạn, mỗi đoạn có nội dung, dài tối đa 3.000 ký tự Unicode; các field văn bản, URL/alt và mảng có giới hạn.
- Fabric: mô tả, ảnh tùy chọn, `care[]` gồm `title`, `description`, `icon` tùy chọn.
- Craft: mô tả, ảnh tùy chọn, `steps[]` gồm `title`, `description` theo đúng schema.
- Ảnh tùy chọn; khi gửi ảnh cần `src` và `alt`, URL `http(s)` hoặc đường dẫn nội bộ hợp lệ.
- Không nhận HTML dưới dạng markup hoặc cho phép chạy script; phía khách hiển thị text được React escape.
- `400 INVALID_FIELD`: thiếu field bắt buộc, sai cấu trúc hoặc vượt giới hạn.
- `403 ADMIN_REQUIRED`: chưa có quyền quản trị từ KR-01.
- `404 NOT_FOUND`: ID sản phẩm không tồn tại.
- `409 CONFLICT`: `version` bị thay đổi bởi thao tác khác.

## Hệ thống dữ liệu

Schema MongoDB hiện có cho phép `story`, `fabric`, `craft`, nên **không cần migration mới**. Thử nghiệm CRUD nội dung với tài khoản admin thật phụ thuộc KR-01. Không được tắt kiểm tra quyền để thử dữ liệu thật.
