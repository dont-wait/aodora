# Hợp đồng API Yêu Cầu May Đo và Quản Trị Custom (DW-04)

Tài liệu kỹ thuật và hợp đồng API cho chức năng may đo theo Issue [#22](https://github.com/dont-wait/aodora/issues/22), căn cứ [system.md](../../system.md) (mục 3.6, 5.3, 6) và [tasks.md](../../tasks.md) (mục DW-04).

---

## 1. Mục đích và phạm vi

Issue #22 (DW-04) gồm 5 bước triển khai tuần tự:
1. **Bước 1 (Xong)**: API gửi yêu cầu may đo công khai (`POST /api/v1/tailoring-requests`), hỗ trợ khách vãng lai và khách có tài khoản; hợp đồng API và mapping schema.
2. **Bước 2 (Hiện tại)**: API quản trị may đo (admin list, filter status, paging, detail, update appointment/notes/measurements/status; xác nhận số đo chủ động `measurement_confirmed`).
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
