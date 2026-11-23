# Hợp đồng API Yêu Cầu May Đo và Quản Trị Custom (DW-04)

Tài liệu kỹ thuật và hợp đồng API cho chức năng may đo theo Issue [#22](https://github.com/dont-wait/aodora/issues/22), căn cứ [system.md](../../system.md) (mục 3.6, 5.3, 6) và [tasks.md](../../tasks.md) (mục DW-04).

---

## 1. Mục đích và phạm vi

Issue #22 (DW-04) gồm 5 bước triển khai tuần tự:
1. **Bước 1 (Xong)**: API gửi yêu cầu may đo công khai (`POST /api/v1/tailoring-requests`), hỗ trợ khách vãng lai và khách có tài khoản; hợp đồng API và mapping schema.
2. **Bước 2 (Xong)**: API quản trị may đo (admin list, filter status, paging, detail, update appointment/notes/measurements/status; xác nhận số đo chủ động `measurement_confirmed`).
3. **Bước 3 (Hiện tại)**: Admin tạo đơn custom (`orders.fulfillment_mode = custom`, liên kết `tailoring_request_id`, snapshot số đo, chốt giá, không trừ tồn kho sẵn, chống tạo trùng khi retry).
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

---

# Bước 2 — API Quản Trị Yêu Cầu May Đo

## 7. Route, Quyền Truy Cập, Phản Hồi Chung

Dùng lại guard của admin catalog/order (#20): cookie phiên hợp lệ + vai trò `admin`. Preflight do `OPTIONS /api/v1/admin/{path...}` đã có, không đăng ký trùng.

| Method | Path | Mục đích |
| --- | --- | --- |
| GET | `/api/v1/admin/tailoring-requests` | Danh sách, lọc `status`, phân trang |
| GET | `/api/v1/admin/tailoring-requests/{id}` | Chi tiết |
| PATCH | `/api/v1/admin/tailoring-requests/{id}` | Sửa một phần ghi chú/lịch hẹn/phụ trách/kênh liên hệ/số đo |
| PATCH | `/api/v1/admin/tailoring-requests/{id}/status` | Chuyển trạng thái (bảng 7.4) |
| POST | `/api/v1/admin/tailoring-requests/{id}/confirm-measurements` | Chốt số đo (thao tác chủ động) |

- Guest hoặc khách thường ⇒ `403 ADMIN_REQUIRED`; Origin không hợp lệ ⇒ `403 ORIGIN_FORBIDDEN`. Mọi phản hồi có `Cache-Control: no-store`.
- Thành công trả `{"data": <DTO>}` (list: `{"data":[...], "total", "page", "page_size"}`).
- Body JSON object, tối đa 16 KiB, `Content-Type: application/json`, **từ chối field lạ** (`DisallowUnknownFields`, kể cả trong `measurements`) và dữ liệu thừa.

### 7.1. DTO admin

`id`, `customer_id` (null nếu khách vãng lai), `is_guest`, `full_name`, `phone`, `preferred_contact`, `category_id`, `product_id`, `product_slug`, `measurements` (`body` + bảy `_cm`, chỉ có key đã nhập), `notes`, `status`, `appointment_at`, `assigned_staff`, `created_at`, `updated_at`, `allowed_actions`.

`allowed_actions` do server tính **chỉ để UI hiển thị**; mọi thao tác vẫn được kiểm tra lại khi thực thi:

| Action | Điều kiện hiển thị |
| --- | --- |
| `update` | luôn có |
| `update_measurements` | status chưa khóa số đo (`submitted`, `contacted`, `consultation_scheduled`) |
| `contact` | `submitted` (→ `contacted`) |
| `schedule` | `contacted` (→ `consultation_scheduled`) |
| `confirm_measurements` | `contacted`/`consultation_scheduled` **và** đủ bảy `_cm` |
| `start` | `measurement_confirmed` (→ `in_progress`) |
| `complete` | `in_progress` (→ `completed`) |
| `cancel` | mọi status trừ `completed`/`cancelled` |

## 7.2. `GET /api/v1/admin/tailoring-requests`

Query: `status` (đúng 7 enum, rỗng/lạ ⇒ 400 `INVALID_FIELD`), `page` (≥1, tối đa 100000, mặc định 1), `page_size` (≥1, tối đa 100, mặc định 20). Tham số lặp hoặc không hỗ trợ ⇒ 400. Sắp xếp `created_at` giảm dần rồi `_id` giảm dần để trang ổn định khi trùng `created_at`.

## 7.3. `PATCH /api/v1/admin/tailoring-requests/{id}`

Chỉ gửi các key cần đổi; key vắng = giữ nguyên; `null` = xóa. Body rỗng `{}` ⇒ 400. **Không nhận** `status`, `customer_id`, `created_at`, `updated_at` (field lạ ⇒ 400).

| Field | Kiểu | Quy tắc |
| --- | --- | --- |
| `notes` | string/null | trim, tối đa 2000; chuỗi trắng = xóa |
| `appointment_at` | string RFC3339/null | lưu UTC; `null` xóa lịch hẹn |
| `assigned_staff` | string/null | trim, tối đa 200; chuỗi trắng = xóa |
| `preferred_contact` | `phone`/`zalo`/null | null xóa (schema không cho null nên server `$unset`) |
| `measurements` | object | gộp từng key vào số đo hiện có: `body` (string/null, ≤500) và bảy `_cm` (số > 0, ≤ 500, giữ nguyên số thập phân, `null` xóa riêng số đó). Thiếu không bao giờ thành 0; còn lại rỗng ⇒ xóa cả `measurements` |

Sửa `measurements` khi status là `measurement_confirmed`, `in_progress`, `completed`, `cancelled` ⇒ **409 `MEASUREMENT_LOCKED`**. Đây là giới hạn đã chấp nhận: chưa có luồng mở lại số đo. Các field khác (ghi chú, lịch hẹn...) vẫn sửa được ở mọi status.

## 7.4. `PATCH .../{id}/status` — bảng chuyển trạng thái

Body `{"status": "<enum>"}`.

| Từ \ Đến | `contacted` | `consultation_scheduled` | `in_progress` | `completed` | `cancelled` |
| --- | --- | --- | --- | --- | --- |
| `submitted` | ✔ | ✘ | ✘ | ✘ | ✔ |
| `contacted` | = | ✔ | ✘ | ✘ | ✔ |
| `consultation_scheduled` | ✘ | = | ✘ | ✘ | ✔ |
| `measurement_confirmed` | ✘ | ✘ | ✔ | ✘ | ✔ |
| `in_progress` | ✘ | ✘ | = | ✔ | ✔ |
| `completed` | ✘ | ✘ | ✘ | = | ✘ |
| `cancelled` | ✘ | ✘ | ✘ | ✘ | = |

- `=`: đích trùng hiện tại ⇒ no-op, `200`, không ghi.
- ✘ (lùi, nhảy cóc, rời trạng thái cuối) ⇒ `409 TAILORING_STATE_CONFLICT`.
- `measurement_confirmed` làm đích ⇒ `400 INVALID_FIELD` (`status`), chỉ đặt được qua `confirm-measurements`.
- `in_progress` **chỉ** từ `measurement_confirmed` (quyết định: chỉ thực hiện may sau khi chốt số đo).
- Giá trị ngoài 7 enum ⇒ 400.

## 7.5. `POST .../{id}/confirm-measurements`

Body bắt buộc `{"confirmed": true}` (thiếu/false/không phải boolean ⇒ 400). Không bao giờ tự chạy khi form đủ số.
- Status phải là `contacted` hoặc `consultation_scheduled`, ngược lại `409 TAILORING_STATE_CONFLICT`.
- Phải đủ bảy `_cm` dương, thiếu ⇒ `409 MEASUREMENT_INCOMPLETE` (không ghi gì). `body` không bắt buộc.
- Thành công ghi `measurement_confirmed`. Gọi lại khi đã `measurement_confirmed` ⇒ no-op `200` (retry an toàn).

## 7.6. Lỗi

| HTTP | Code | Khi nào |
| --- | --- | --- |
| 400 | `INVALID_FIELD` / `INVALID_BODY` | query/body/enum/số đo/RFC3339 không hợp lệ, field lạ |
| 403 | `ADMIN_REQUIRED` / `ORIGIN_FORBIDDEN` | không phải admin / Origin sai |
| 404 | `NOT_FOUND` | id sai định dạng hoặc không tồn tại |
| 409 | `TAILORING_STATE_CONFLICT` | chuyển trạng thái không hợp lệ, hoặc tranh chấp ghi kéo dài |
| 409 | `MEASUREMENT_LOCKED` | sửa số đo khi đã khóa |
| 409 | `MEASUREMENT_INCOMPLETE` | chốt số đo khi chưa đủ bảy `_cm` |
| 503 | `TAILORING_UNAVAILABLE` | lỗi hạ tầng (không rò chi tiết; log chỉ ghi kiểu lỗi) |

## 7.7. Ghi đồng thời

Mọi ghi dùng điều kiện trên `status` và `updated_at` đã đọc (`UpdateOne` với filter `{_id, status, updated_at}`). Nếu không khớp (admin khác vừa sửa), service đọc lại, tính lại trên dữ liệu mới và thử lại tối đa 3 lần; hết lượt ⇒ 409 `TAILORING_STATE_CONFLICT`. Nhờ đó hai admin sửa hai field khác nhau cùng lúc đều được giữ, còn hai chuyển trạng thái xung đột thì chỉ một thắng. `updated_at` luôn tăng (làm tròn mili-giây, tối thiểu +1 ms).

## 7.8. Index

`tailoring_requests` có `ix_tailoring_requests_status_created {status:1, created_at:-1}` phục vụ lọc theo status. Danh sách **không lọc** sắp theo `created_at, _id` không có index riêng (quét + sắp xếp trong bộ nhớ); với quy mô cửa hàng hiện tại chấp nhận được, nếu cần index `{created_at:-1,_id:-1}` thì phải thêm migration mới (chưa làm, xem result.md "Chưa chắc").

---

# Bước 3 — Admin Tạo Đơn Custom Từ Yêu Cầu May Đo

## 8. Đơn custom: thiết kế (khảo sát validator thật)

Nguồn: `000004_client_fields`, `order_checkout`, `order_checkout_access`, `order_number_unique`, `order_repository.go`. **Không cần migration mới, không đổi enum.**

### 8.1. Field của đơn custom

| Field | Giá trị |
| --- | --- |
| `fulfillment_mode` (đơn) | `custom` |
| `tailoring_request_id` | ObjectId của yêu cầu (đã có trong validator, nullable) |
| `measurement_profile` | snapshot `body` + bảy `_cm` của yêu cầu tại lúc tạo; chỉ key đang có |
| `items[0]` | `fulfillment_mode=custom`, `product_id`, `product_name` (tên sản phẩm lúc tạo), `sku="CUSTOM"` (validator bắt buộc chuỗi, dòng custom không phải biến thể kho), `quantity`, `unit_price` (admin chốt), `measurements` (cùng snapshot); **không** `size/color/variant_id` |
| `customer_id` | lấy từ yêu cầu; yêu cầu guest ⇒ **bỏ field** (validator cho phép; không tạo khách giả) |
| `status` / `payment_status` | `pending` / `unpaid` |
| `shipping_address` | COD: đủ `recipient/phone/line1/city`; tại quầy: chỉ `recipient/phone`, phí 0 (validator `$or`). `recipient`/`phone` mặc định lấy `full_name`/`phone` của yêu cầu; **địa chỉ giao không bao giờ tự bịa** |
| `subtotal/shipping_fee/total` | `unit_price × quantity` và phí, decimal exact (`Money`, lưu int64 hoặc Decimal128) |
| `order_number` | cùng bộ sinh với checkout; trùng ⇒ sinh lại tối đa 3 lần (unique `uq_orders_order_number`) |

Không đụng `products.stock` hay `variants[].stock` (kiểm chứng bằng test integration).

### 8.2. Có cần `checkout` marker không?

**Không.** Validator chỉ ràng buộc marker khi nó *tồn tại*: khi có marker thì bắt buộc `payment_method ∈ {cod, cash_at_counter}`, và `scope` phải khớp `customer:<customer_id>` hoặc `guest:<guest_access_hash>`. Thiếu marker thì enum cũ vẫn gồm `cod` và `cash_at_counter` và các ràng buộc phí/địa chỉ vẫn áp dụng. Vì vậy đơn custom **không có** `checkout`/`guest_access_hash`. Hệ quả đã kiểm tra trong code:
- Hủy đơn custom: `restockItems` trả rỗng (không marker, và dòng custom luôn bị loại) ⇒ `inventory_restock=not_applicable`, tồn không đổi.
- Chuyển trạng thái/thu tiền dùng nguyên `AdminService` của #20 (`cod`: pending→confirmed→shipping→delivered; `cash_at_counter`: pending→confirmed→delivered).
- `decodeOrder` nay đọc thêm `fulfillment_mode`, `tailoring_request_id`, `measurement_profile`, `items[].measurements` và DTO admin trả `fulfillment_mode`, `tailoring_request_id`, `measurement_profile`, `items[].measurements`. Đơn checkout cũ không có field này nên không đổi hình dạng.

Phương án thay thế đã loại: dùng marker giả `(scope, request_id)` tất định để mượn unique index `uq_orders_checkout_scope_request`. Nhược điểm: lạm dụng ngữ nghĩa marker (scope = danh tính người đặt), với yêu cầu có tài khoản scope bắt buộc là `customer:<id>` nên chính khách đó có thể chiếm trước request_id bằng một checkout, và guest phải bịa `guest_access_hash`.

### 8.3. Chống tạo trùng (không dùng Idempotency-Key)

Khóa tự nhiên là chính yêu cầu: một yêu cầu có tối đa **một** đơn custom. Cơ chế (`tailoring_order_repository.go`), trong **một** MongoDB transaction snapshot/majority:
1. đọc lại yêu cầu; 2. tìm đơn có `tailoring_request_id` đó, có thì trả về (`created=false`); 3. kiểm tra lại status (`measurement_confirmed`/`in_progress`) và đủ bảy `_cm`; 4. chèn đơn; 5. `UpdateOne` yêu cầu với filter `{_id, status, updated_at}` vừa đọc, đặt `updated_at` mới.

Hai request đồng thời cùng ghi **cùng một document yêu cầu** ⇒ MongoDB abort một transaction bằng `WriteConflict`; driver chạy lại callback, lần sau thấy đơn đã commit và trả đơn đó. Cũng chống tạo đơn khi admin khác vừa sửa/hủy yêu cầu. Retry khi chưa rõ kết quả cũng an toàn.

Quy ước retry:
- Gọi lại **cùng nội dung chính** (`product_id`, `unit_price`, `quantity`, `payment_method`) ⇒ `200` trả đơn đã có (kể cả khi yêu cầu đã sang `in_progress`/`completed`). Ghi chú/địa chỉ gửi lại không sửa đơn.
- Nội dung khác ⇒ `409 CUSTOM_ORDER_EXISTS`, không tạo đơn thứ hai.
- Đơn đã bị hủy **không** mở cho phép tạo đơn mới (một yêu cầu = một đơn); cần đơn khác phải xử lý thủ công, đây là giới hạn đã chấp nhận.

Giới hạn: chỉ code path này được bảo vệ; DB không có unique index theo `tailoring_request_id`. Hardening tùy chọn (cần **migration mới**, chưa làm, chờ reviewer): unique partial index `{tailoring_request_id:1}` với `partialFilterExpression: {tailoring_request_id: {$type: "objectId"}}`; đồng thời index này cũng thay phép tìm đơn theo `tailoring_request_id` (hiện là quét `orders`, chấp nhận ở quy mô cửa hàng).

### 8.4. Khách guest không có credential đọc đơn

Đơn của yêu cầu guest không có `customer_id` lẫn `guest_access_hash` ⇒ **chỉ admin đọc được** qua API admin; `GET /api/v1/orders/{id}` trả 404 cho mọi người khác (kể cả tài khoản bất kỳ và không đăng nhập), và đơn không xuất hiện trong `GET /api/v1/orders`. Khách guest theo dõi qua liên hệ trực tiếp với cửa hàng; không phát token ở bước này. Yêu cầu có tài khoản: đơn mang `customer_id` nên chủ tài khoản thấy trong `GET /api/v1/orders` và đọc được `GET /api/v1/orders/{id}`; tài khoản khác nhận 404.

## 9. `POST /api/v1/admin/tailoring-requests/{id}/order`

Guard admin như Bước 2 (guest/khách thường ⇒ 403 `ADMIN_REQUIRED`, Origin sai ⇒ 403 `ORIGIN_FORBIDDEN`, `no-store`). Body JSON ≤ 16 KiB, từ chối field lạ (`customer_id`, `status`, `total`, `shipping_fee`... đều ⇒ 400 `INVALID_BODY`).

| Field | Bắt buộc | Quy tắc |
| --- | --- | --- |
| `unit_price` | Có | số thập phân dương (number hoặc chuỗi), giữ nguyên chữ số, không float; 0/âm/chữ/số mũ ⇒ 400. **Không lấy giá từ sản phẩm** |
| `payment_method` | Có | `cod` hoặc `cash_at_counter` |
| `product_id` | Chỉ khi yêu cầu chưa có mẫu | yêu cầu đã có mẫu mà gửi khác ⇒ 400 `product_id`; sản phẩm không tồn tại/`is_active=false` ⇒ 400 |
| `quantity` | Không | integer 1..1.000.000, mặc định 1 |
| `shipping_address` | COD: `line1`, `city` bắt buộc | `recipient`/`phone` mặc định từ yêu cầu; `ward`, `district` tùy chọn |
| `note` | Không | ≤ 1000 ký tự |

Phí: tại quầy = 0; COD = `ORDER_COD_SHIPPING_FEE` (chưa cấu hình ⇒ `503 SHIPPING_UNAVAILABLE`). Không có phí mặc định (Q-04).

Điều kiện yêu cầu (kiểm tra lại trong transaction):

| Điều kiện | Kết quả nếu sai |
| --- | --- |
| id hợp lệ và tồn tại | 404 `NOT_FOUND` |
| status `measurement_confirmed` hoặc `in_progress` | 409 `TAILORING_STATE_CONFLICT` |
| đủ bảy `_cm` (`body` không bắt buộc, có thì được snapshot) | 409 `MEASUREMENT_INCOMPLETE` |
| chưa có đơn khác nội dung | 409 `CUSTOM_ORDER_EXISTS` |

Response: `201` đơn mới hoặc `200` replay, dạng `{"data": <DTO đơn admin>, "created": true|false}`; DTO như `DW03_ADMIN_ORDER_API.md` cộng `fulfillment_mode`, `tailoring_request_id`, `measurement_profile`, `items[].measurements`. Lỗi hạ tầng ⇒ 503 `TAILORING_UNAVAILABLE`.

Tạo đơn chỉ cập nhật `updated_at` của yêu cầu (không đổi status, ghi chú, số đo). Sửa yêu cầu sau đó không đổi đơn vì snapshot được sao chép theo giá trị.
