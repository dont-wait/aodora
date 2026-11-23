# Hợp đồng API Yêu Cầu May Đo và Quản Trị Custom (DW-04)

Tài liệu kỹ thuật và hợp đồng API cho chức năng may đo theo Issue [#22](https://github.com/dont-wait/aodora/issues/22), căn cứ [system.md](../../system.md) (mục 3.6, 5.3, 6) và [tasks.md](../../tasks.md) (mục DW-04).

---

## 1. Mục đích và phạm vi

Issue #22 (DW-04) gồm 5 bước triển khai tuần tự:
1. **Bước 1 (Hiện tại)**: API gửi yêu cầu may đo công khai (`POST /api/v1/tailoring-requests`), hỗ trợ khách vãng lai và khách có tài khoản; hợp đồng API và mapping schema.
2. **Bước 2**: API quản trị may đo (admin list, filter status, paging, detail, update appointment/notes/measurements/status; xác nhận số đo chủ động `measurement_confirmed`).
3. **Bước 3**: Admin tạo đơn custom (`orders.fulfillment_mode = custom`, liên kết `tailoring_request_id`, snapshot số đo, chốt giá, không trừ tồn kho sẵn, chống tạo trùng khi retry).
4. **Bước 4**: Tích hợp client (form GuidePage, nút tư vấn ở chi tiết áo, trang admin quản trị yêu cầu may đo).
5. **Bước 5**: Nghiệm thu xuyên suốt, bằng chứng DB test, kiểm tra trình duyệt.

---

## 2. Schema `tailoring_requests` và Mapping Trường Dữ Liệu

Schema của collection `tailoring_requests` được định nghĩa trong migration `server/migrations/000004_client_fields.up.json` và tài liệu [database-schema.md](database-schema.md).

### 2.1. Bảng mapping chi tiết

| Tên trường DB | Kiểu BSON | Bắt buộc trong DB | Bắt buộc ở API | Ý nghĩa / Ghi chú |
| --- | --- | --- | --- | --- |
| `_id` | `objectId` | Có | Không (Server tạo) | ID định danh của yêu cầu may đo |
| `customer_id` | `objectId`, `null` | Không | Không | ID tài khoản nếu khách đã đăng nhập; `null`/bỏ qua nếu là khách vãng lai (guest). **Validator không cấm guest.** |
| `full_name` | `string` (minLength: 1) | Có | Có | Họ và tên khách hàng (trim chuỗi, từ chối chuỗi trắng, max 200 ký tự). DTO chấp nhận alias `name`; nếu gửi cả hai mà khác nhau (sau trim) ⇒ 400 `INVALID_FIELD` (field `name`). |
| `phone` | `string` (minLength: 1) | Có | Có | Số điện thoại liên hệ (trim chuỗi, từ chối chuỗi trắng, max 50 ký tự, định dạng số hợp lệ). |
| `preferred_contact` | `string` (enum: `phone`, `zalo`) | Không | Không | Kênh liên hệ ưu tiên của khách hàng. Chỉ nhận giá trị `phone` hoặc `zalo`. |
| `category_id` | `string`, `null` | Không | Không | Mã danh mục sản phẩm quan tâm (ví dụ: `co-phuc`, `dan-gian`). |
| `product_id` | `objectId`, `null` | Không | Không | ID sản phẩm tham chiếu trong catalog. Server xác minh tồn tại; ID và slug mâu thuẫn sẽ báo lỗi. |
| `product_slug` | `string`, `null` | Không | Không | Slug sản phẩm tham chiếu. Có thể thiếu khi khách chỉ gửi nhu cầu chung từ trang may đo. |
| `measurements` | `object` | Không | Không | Object chứa 8 thông số số đo của khách (chi tiết mục 2.2). |
| `notes` | `string`, `null` | Không | Không | Ghi chú yêu cầu đặc biệt của khách (max 2000 ký tự). DTO chấp nhận alias `note`; nếu gửi cả hai mà khác nhau (sau trim) ⇒ 400 `INVALID_FIELD` (field `note`). |
| `status` | `string` (enum) | Có | Không (Server gán) | Trạng thái xử lý. Giá trị mặc định khi tạo mới là `submitted`. |
| `appointment_at` | `date`, `null` | Không | Không | Lịch hẹn tư vấn/lấy số đo (admin cập nhật ở Bước 2). |
| `assigned_staff` | `string`, `null` | Không | Không | Nhân viên phụ trách yêu cầu (admin cập nhật ở Bước 2). |
| `created_at` | `date` | Có | Không (Server gán) | Thời điểm gửi yêu cầu (UTC timestamp). |
| `updated_at` | `date` | Có | Không (Server gán) | Thời điểm cập nhật cuối (UTC timestamp). |

### 2.2. Chi tiết trường `measurements` (8 thông số vàng)

| Tên trường | Kiểu BSON | Ràng buộc API | Ý nghĩa |
| --- | --- | --- | --- |
| `body` | `string`, `null` | Chuỗi mô tả vóc dáng, max 500 ký tự | Mô tả chiều cao/cân nặng (ví dụ: `1m62 / 52kg`). **Tuyệt đối không tự tách thành trường số float.** |
| `neck_cm` | `int`, `long`, `double`, `decimal` | Số dương `> 0`, tối đa `500` cm | Vòng cổ (cm) |
| `bust_cm` | `int`, `long`, `double`, `decimal` | Số dương `> 0`, tối đa `500` cm | Vòng ngực (cm) |
| `waist_cm` | `int`, `long`, `double`, `decimal` | Số dương `> 0`, tối đa `500` cm | Vòng eo thắt (cm) |
| `hips_cm` | `int`, `long`, `double`, `decimal` | Số dương `> 0`, tối đa `500` cm | Vòng mông (cm) |
| `shoulder_cm` | `int`, `long`, `double`, `decimal` | Số dương `> 0`, tối đa `500` cm | Rộng vai (cm) |
| `sleeve_cm` | `int`, `long`, `double`, `decimal` | Số dương `> 0`, tối đa `500` cm | Dài tay (cm) |
| `garment_length_cm` | `int`, `long`, `double`, `decimal` | Số dương `> 0`, tối đa `500` cm | Chiều dài áo (cm) |

**Quy tắc số đo:**
- Số đo có thể thiếu khi gửi yêu cầu.
- **Không bao giờ biến số đo thiếu thành `0`**: trường nào client không nhập sẽ giữ `nil` (không lưu trong BSON document hoặc lưu `null`).
- Các trường số đo cm phải là số dương hợp lệ (`> 0` và `<= 500`). Số âm, số 0 hoặc chữ bị từ chối với lỗi `400 INVALID_FIELD`.
- Số thập phân (như `34.5`, `138.5`) được giữ nguyên giá trị chính xác (`exact`), không làm tròn thành số nguyên.

### 2.3. Danh sách trạng thái (`tailoring_requests.status`)

Theo schema hiện có:
- `submitted`: Đã tiếp nhận yêu cầu từ khách (mặc định cho yêu cầu mới).
- `contacted`: Đã liên hệ với khách.
- `consultation_scheduled`: Đã lên lịch hẹn tư vấn.
- `measurement_confirmed`: Đã xác nhận chính thức bộ số đo với khách (thao tác chủ động của admin, không tự chuyển).
- `in_progress`: Đang thực hiện may trang phục.
- `completed`: Đã hoàn tất yêu cầu may đo.
- `cancelled`: Yêu cầu đã hủy.

### 2.4. Các index đã tạo

- `ix_tailoring_requests_phone_created`: `{ phone: 1, created_at: -1 }` (tra cứu theo SĐT).
- `ix_tailoring_requests_status_created`: `{ status: 1, created_at: -1 }` (lọc trạng thái cho admin).
- `ix_tailoring_requests_customer_created`: `{ customer_id: 1, created_at: -1 }`, `sparse: true` (tra cứu theo tài khoản khách).

---

## 3. Bảng Route và Quyền Truy Cập (Bước 1)

| Phương thức | Đường dẫn | Quyền truy cập | Thành công | Ý nghĩa |
| --- | --- | --- | --- | --- |
| `OPTIONS` | `/api/v1/tailoring-requests` | Công khai | `204 No Content` | CORS preflight |
| `OPTIONS` | `/api/v1/tailoring-requests/{path...}` | Công khai | `204 No Content` | CORS preflight |
| `POST` | `/api/v1/tailoring-requests` | Khách vãng lai hoặc Tài khoản | `201 Created` | Tiếp nhận yêu cầu may đo |

Phản hồi luôn kèm header `Cache-Control: no-store`.

---

## 4. Xác Thực và Phân Quyền

1. **Khách có cookie phiên (`aodora_session`)**:
   - Nếu cookie hợp lệ: Trích xuất `customer.ID` và gán vào `tailoring_requests.customer_id`.
   - Nếu cookie gửi lên nhưng phiên không hợp lệ hoặc đã hết hạn: Trả về ngay `401 AUTH_REQUIRED`, **không hạ quyền xuống guest** (nhất quán với nghiệp vụ Orders).
2. **Khách vãng lai (không gửi cookie)**:
   - Cho phép gửi yêu cầu bình thường, lưu `customer_id: null` (hoặc bỏ qua trong BSON).
3. **Bảo mật**:
   - Client không được gửi các trường quản trị hoặc hệ thống: `customer_id`, `status`, `appointment_at`, `assigned_staff`, `created_at`, `updated_at`. Nếu gửi sẽ bị từ chối `400 INVALID_BODY` do cơ chế `DisallowUnknownFields()`.
   - Phản hồi chỉ trả về `id` và `status`, **tuyệt đối không để lộ dữ liệu liên hệ cá nhân** (`full_name`, `phone`, `notes`).
   - Guest không nhận credential đọc lại ở bước này (không có API đọc public).

---

## 5. Chi Tiết Endpoint `POST /api/v1/tailoring-requests`

### 5.1. Headers
- `Content-Type: application/json` (bắt buộc)
- `Cookie: aodora_session=<token>` (tùy chọn)

### 5.2. Request Payload mẫu

#### Trường hợp 1: Tối thiểu (chỉ họ tên và số điện thoại)
```json
{
  "full_name": "Nguyễn Minh An",
  "phone": "0900000001"
}
```

#### Trường hợp 2: Đầy đủ kèm mẫu sản phẩm và số đo
```json
{
  "full_name": "Nguyễn Minh An",
  "phone": "0900000001",
  "preferred_contact": "zalo",
  "category_id": "co-phuc",
  "product_id": "680000000000000000000001",
  "product_slug": "ao-dai-lua-ha-dong",
  "measurements": {
    "body": "1m62 / 52kg",
    "neck_cm": 34.5,
    "bust_cm": 86,
    "waist_cm": 65,
    "hips_cm": 92,
    "shoulder_cm": 37,
    "sleeve_cm": 56,
    "garment_length_cm": 138.5
  },
  "notes": "Khách muốn được tư vấn phom áo và độ dài tà."
}
```

### 5.3. Response Thành Công (`201 Created`)
```json
{
  "data": {
    "id": "6d0000000000000000000001",
    "status": "submitted"
  }
}
```

### 5.4. Lỗi và Mã Lỗi Thường Gặp (`400`, `401`, `403`, `503`)

| HTTP Code | Error Code | Field | Lý do / Thông điệp |
| --- | --- | --- | --- |
| 400 | `INVALID_FIELD` | `full_name` | Họ và tên để trống hoặc chỉ có khoảng trắng |
| 400 | `INVALID_FIELD` | `phone` | Số điện thoại để trống hoặc định dạng không hợp lệ |
| 400 | `INVALID_FIELD` | `preferred_contact` | Kênh liên hệ khác `phone` hoặc `zalo` |
| 400 | `INVALID_FIELD` | `product_id` | Mã sản phẩm sai định dạng hex hoặc sản phẩm không tồn tại / đã ẩn (`is_active=false`, không lộ trạng thái ẩn) |
| 400 | `INVALID_FIELD` | `product_slug` | Slug không tồn tại hoặc mâu thuẫn với `product_id` |
| 400 | `INVALID_FIELD` | `neck_cm` (hoặc các trường `_cm` khác) | Số đo âm, bằng 0, quá lớn (> 500 cm) hoặc chứa chữ |
| 400 | `INVALID_BODY` | `Content-Type` | Header không bắt đầu bằng `application/json` |
| 400 | `INVALID_BODY` | `body` | JSON không hợp lệ, chứa unknown fields hoặc vượt quá 64 KiB |
| 401 | `AUTH_REQUIRED` | *(trống)* | Cookie phiên được gửi lên nhưng không hợp lệ hoặc đã hết hạn |
| 403 | `ORIGIN_FORBIDDEN` | *(trống)* | Nguồn truy cập (Origin) không nằm trong danh sách được cấp phép |
| 503 | `TAILORING_UNAVAILABLE` | *(trống)* | Lỗi hạ tầng (mất kết nối Mongo khi tra sản phẩm hoặc ghi yêu cầu); không rò chi tiết, client có thể thử lại. Không phải lỗi 400 |

---

## 6. Quyết Định Kỹ Thuật (Chống spam và Retry)

1. **Giới hạn kích thước payload**: Sử dụng `http.MaxBytesReader` giới hạn 64 KiB để chống tấn công làm tràn bộ nhớ.
2. **Giới hạn độ dài chuỗi**:
   - `full_name`: Tối đa 200 ký tự.
   - `phone`: Tối đa 50 ký tự.
   - `body`: Tối đa 500 ký tự.
   - `notes`: Tối đa 2000 ký tự.
3. **Cơ chế chống gửi lặp (Retry)**:
   - Ở Bước 1 (tiếp nhận yêu cầu tư vấn may đo công khai), mỗi lần gửi sẽ tạo một document yêu cầu mới với ID riêng. Không bắt buộc `Idempotency-Key` hay token guest ở bước này để giữ luồng tư vấn đơn giản và thân thiện với khách hàng.
   - Cơ chế idempotency và chống tạo trùng đơn sẽ được áp dụng chặt chẽ ở Bước 3 khi admin tạo đơn custom từ yêu cầu may đo.
