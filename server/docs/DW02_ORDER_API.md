# API đơn hàng may sẵn, tính tiền và cập nhật tồn nhất quán (DW-02)

Tài liệu kỹ thuật và hướng dẫn tích hợp API đơn hàng may sẵn cho hệ thống Aodora theo Issue [#19](https://github.com/dont-wait/aodora/issues/19), tuân thủ hợp đồng [DW_ORDER_CHECKOUT_CONTRACT.md](DW_ORDER_CHECKOUT_CONTRACT.md).

API quản trị issue #20: xem [DW03_ADMIN_ORDER_API.md](DW03_ADMIN_ORDER_API.md); DTO khách hiện có giữ nguyên.

---

## 1. Mục đích và phạm vi

Nhóm API Orders cung cấp các khả năng:
- Lấy báo giá đơn hàng (`quote`) trước khi đặt mua.
- Tạo đơn hàng may sẵn với cơ chế chống gửi lặp (`Idempotency-Key`) và xác thực danh tính nhất quán (tài khoản hoặc khách vãng lai).
- Tra cứu đơn hàng theo ID (`GET /api/v1/orders/{id}`).
- Tra cứu kết quả attempt thanh toán khi bị mất kết nối (`GET /api/v1/orders/attempts/{request_id}`).
- Liệt kê danh sách đơn hàng đã mua của tài khoản đăng nhập (`GET /api/v1/orders`).

### Ngoài phạm vi (chưa có trong issue #19)
- Giao diện người dùng checkout (thuộc Issue #21).
- Quản trị đơn hàng, máy trạng thái đơn hàng, hủy đơn/hoàn tồn, xác nhận thu tiền COD/tại quầy (thuộc Issue #20).
- Luồng đặt may đo áo dài / sản phẩm tùy chỉnh số đo (thuộc Issue #22).
- API đánh giá sản phẩm sau mua (thuộc KR-04).

---

## 2. Bảng route và quyền truy cập

| Phương thức | Đường dẫn | Quyền truy cập | Thành công | Ý nghĩa |
| --- | --- | --- | --- | --- |
| `OPTIONS` | `/api/v1/orders`, `/api/v1/orders/{path...}` | Công khai | `204 No Content` | CORS preflight cho các header checkout |
| `POST` | `/api/v1/orders/quote` | Guest hoặc Tài khoản | `200 OK` | Lấy báo giá, tính tiền và sinh quote token |
| `POST` | `/api/v1/orders` | Guest hoặc Tài khoản | `201 Created` (mới) / `200 OK` (replay) | Đặt hàng may sẵn, trừ tồn kho trong transaction |
| `GET` | `/api/v1/orders` | Chỉ tài khoản đăng nhập | `200 OK` | Danh sách đơn hàng của tài khoản |
| `GET` | `/api/v1/orders/attempts/{request_id}` | Tài khoản hoặc guest token của chính attempt đó | `200 OK` | Tra cứu trạng thái attempt theo request ID |
| `GET` | `/api/v1/orders/{id}` | Chủ đơn hoặc Quản trị viên (Admin) | `200 OK` | Chi tiết đơn hàng theo Order ID |

Mọi phản hồi của nhóm API orders (kể cả phản hồi lỗi) luôn kèm header `Cache-Control: no-store` để bảo vệ thông tin riêng tư của khách hàng.

---

## 3. Header và quy tắc xác thực danh tính

### 3.1. Header bắt buộc và tùy chọn

1. **`Idempotency-Key`**: Bắt buộc khi gọi `POST /api/v1/orders`.
   - Định dạng: Chuỗi UUID canonical lowercase (phiên bản 1 đến 8) khớp biểu thức chính quy:
     `^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`
   - Thiếu hoặc sai định dạng ⇒ trả về lỗi `400 INVALID_FIELD` với `field: "Idempotency-Key"`.

2. **`X-Order-Access-Token`**: Bắt buộc với khách vãng lai (guest) khi tạo đơn hoặc đọc đơn guest.
   - Định dạng: Chuỗi Base64URL không padding (`rawURLEncoding`) có độ dài đúng 43 ký tự, giải mã thành đúng 32 byte nhị phân ngẫu nhiên (sinh từ Web Crypto phía client).
   - Server kiểm tra tính canonical (decode đúng 32 byte và re-encode khớp 100% chuỗi gửi lên).
   - Server tính SHA-256 trên 32 byte đã giải mã và lưu dưới dạng chuỗi hex 64 ký tự thường (`GuestHash`). Tuyệt đối không lưu hoặc ghi log token gốc.
   - Sai định dạng ⇒ `400 INVALID_FIELD` với `field: "X-Order-Access-Token"`.

3. **Cookie phiên**: Bắt buộc với khách hàng có tài khoản.
   - Tên cookie lấy từ biến cấu hình `AUTH_COOKIE_NAME` (mặc định là `aodora_session`), do server cấp khi đăng nhập (`/api/v1/auth/login`).

### 3.2. Thứ tự xử lý danh tính

1. **Kiểm tra cookie phiên**: Gọi `guard.Session(r)`. Nếu cookie được gửi lên nhưng phiên không hợp lệ hoặc đã hết hạn (`presented && !ok`), server lập tức trả `401 AUTH_REQUIRED`, không bao giờ hạ quyền xuống guest (kể cả ở endpoint `/orders/quote`).
2. **Kiểm tra token guest**: Nếu có header `X-Order-Access-Token`, server kiểm tra tính hợp lệ qua `ValidateGuestAccessToken`. Sai định dạng trả `400 INVALID_FIELD`.
3. **POST tạo đơn (`POST /api/v1/orders`)**:
   - Nếu có phiên hợp lệ VÀ gửi kèm `X-Order-Access-Token` ⇒ trả `400 INVALID_FIELD` (không cho phép trộn hai danh tính).
   - Nếu không có phiên hợp lệ VÀ không có `X-Order-Access-Token` ⇒ trả `400 INVALID_FIELD` (yêu cầu đăng nhập hoặc token guest).
   - Principal: Nhận `CustomerID` từ phiên tài khoản hoặc `GuestHash` từ token guest.
4. **GET đọc đơn (`GET /api/v1/orders/{id}` và `/attempts/{request_id}`)**:
   - Với `GET /api/v1/orders/{id}`: Viewer tập hợp toàn bộ credential hợp lệ hiện có (`CustomerID`, `IsAdmin` nếu `role == "admin"`, và `GuestHash`). Nếu không có bất kỳ credential nào hợp lệ ⇒ trả `404 NOT_FOUND` ngay lập tức (không tiết lộ sự tồn tại của đơn hàng).
   - Với `GET /api/v1/orders/attempts/{request_id}`: `Service.Attempt` chỉ tra cứu theo scope của chính viewer (`customer:<CustomerID>` hoặc `guest:<GuestHash>`). Admin không tra được attempt của người khác.
5. **GET danh sách (`GET /api/v1/orders`)**: Bắt buộc tài khoản đăng nhập; không có phiên trả `401 AUTH_REQUIRED` (token guest không thay thế được).
6. **Báo giá (`POST /api/v1/orders/quote`)**: Cho phép mọi khách hàng (không yêu cầu credential, nhưng nếu có gửi cookie phiên không hợp lệ thì vẫn bị từ chối 401).

---

## 4. Request và Response mẫu

### 4.1. Báo giá đơn hàng (`POST /api/v1/orders/quote`)

**Request:**
```http
POST /api/v1/orders/quote HTTP/1.1
Content-Type: application/json

{
  "payment_method": "cod",
  "shipping_address": {
    "recipient": "Nguyễn An",
    "phone": "0900000001",
    "line1": "12 Nguyễn Trãi",
    "city": "TP. Hồ Chí Minh"
  },
  "items": [
    {
      "product_id": "680000000000000000000001",
      "sku": "LUA-DO-M",
      "quantity": 1
    }
  ],
  "note": "Giao giờ hành chính"
}
```

**Response (`200 OK`):**
```json
{
  "data": {
    "quote_token": "<quote_token từ bước quote>",
    "expires_at": "2026-11-16T12:15:00Z",
    "subtotal": 500000,
    "shipping_fee": 17000,
    "total": 517000,
    "items": [
      {
        "product_id": "680000000000000000000001",
        "sku": "LUA-DO-M",
        "quantity": 1,
        "unit_price": 500000
      }
    ]
  }
}
```
*(Lưu ý: Mức phí vận chuyển `17000` ở trên là giá trị minh họa trong môi trường thử nghiệm, không đại diện cho mức phí cố định vận hành).*

### 4.2. Tạo đơn hàng (`POST /api/v1/orders`)

#### Ca đặt hàng tại quầy (Khách vãng lai):
**Request:**
```http
POST /api/v1/orders HTTP/1.1
Content-Type: application/json
Idempotency-Key: ad577a65-cd42-46eb-ae8a-98b5f1186571
X-Order-Access-Token: <43 ký tự base64url của 32 byte ngẫu nhiên>

{
  "payment_method": "cash_at_counter",
  "shipping_address": {
    "recipient": "Nguyễn An",
    "phone": "0900000001"
  },
  "items": [
    {
      "product_id": "680000000000000000000001",
      "sku": "LUA-DO-M",
      "quantity": 1
    }
  ],
  "quote_token": "<quote_token từ bước quote>"
}
```

**Response (`201 Created` lần đầu, hoặc `200 OK` khi gửi lại cùng payload):**
```json
{
  "data": {
    "id": "690000000000000000000010",
    "order_number": "AD-261116-8H3K9N2M",
    "status": "pending",
    "payment_method": "cash_at_counter",
    "payment_status": "unpaid",
    "subtotal": 500000,
    "shipping_fee": 0,
    "total": 500000,
    "shipping_address": {
      "recipient": "Nguyễn An",
      "phone": "0900000001"
    },
    "items": [
      {
        "product_id": "680000000000000000000001",
        "product_name": "Áo Dài Lụa Gấm",
        "sku": "LUA-DO-M",
        "color": "Đỏ",
        "size": "M",
        "quantity": 1,
        "unit_price": 500000
      }
    ],
    "created_at": "2026-11-16T12:00:00Z"
  }
}
```

### 4.3. Đọc chi tiết đơn hàng (`GET /api/v1/orders/{id}`)
- Yêu cầu header `X-Order-Access-Token` (đối với đơn guest) hoặc cookie phiên `AUTH_COOKIE_NAME` (mặc định `aodora_session`, đối với đơn tài khoản hoặc tài khoản admin).
- Trả về `200 OK` kèm cấu trúc đơn hàng như trên. Nếu không tìm thấy hoặc viewer không có quyền, luôn trả về `404 NOT_FOUND`.

### 4.4. Tra cứu attempt thanh toán (`GET /api/v1/orders/attempts/{request_id}`)
- Dùng khi client gửi request tạo đơn nhưng bị rớt mạng / timeout không nhận được response.
- Tra cứu theo UUID `request_id` (chính là `Idempotency-Key` đã gửi).
- Yêu cầu đúng credential của chính attempt đó (`CustomerID` từ phiên tài khoản hoặc `X-Order-Access-Token` của guest).
- Thành công trả về `200 OK` kèm `{"data": <order DTO>}`. Nếu chưa có đơn hoặc sai quyền trả về `404 NOT_FOUND`.

### 4.5. Danh sách đơn hàng của tôi (`GET /api/v1/orders`)
- Yêu cầu cookie phiên tài khoản.
- Tham số query:
  - `page`: Số nguyên trong khoảng `1..100000` (mặc định `1`). Nếu không phải số nguyên hoặc ngoài khoảng `1..100000` ⇒ trả `400 INVALID_FIELD` với `field: "page"`.
  - `page_size`: Số nguyên `≥ 1` (mặc định `20`). Nếu không phải số nguyên `≥ 1` ⇒ trả `400 INVALID_FIELD` với `field: "page_size"`. Nếu `page_size > 100`, server không từ chối mà tự động kẹp (clamp) về `100`.
  - `status` (tùy chọn): `pending`, `confirmed`, `shipping`, `delivered` hoặc `cancelled`; bỏ trống là tất cả. Giá trị khác ⇒ `400 INVALID_FIELD` với `field: "status"`. `total` tính sau khi lọc.
  - Tham số query lạ ngoài `page`, `page_size`, `status`, hoặc lặp một tham số ⇒ trả `400 INVALID_FIELD` với `field: "query"`.

**Response (`200 OK`):**
```json
{
  "data": [
    {
      "id": "690000000000000000000010",
      "order_number": "AD-261116-8H3K9N2M",
      "status": "pending",
      "payment_method": "cash_at_counter",
      "payment_status": "unpaid",
      "subtotal": 500000,
      "shipping_fee": 0,
      "total": 500000,
      "shipping_address": {
        "recipient": "Nguyễn An",
        "phone": "0900000001"
      },
      "items": [
        {
          "product_id": "680000000000000000000001",
          "product_name": "Áo Dài Lụa Gấm",
          "sku": "LUA-DO-M",
          "color": "Đỏ",
          "size": "M",
          "quantity": 1,
          "unit_price": 500000
        }
      ],
      "created_at": "2026-11-16T12:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

### 4.6. Kiểm soát body và trường cấm
- Server bật chế độ từ chối trường không khai báo (`DisallowUnknownFields`).
- Nếu request body gửi kèm bất kỳ trường nào ngoài allowlist (như `customer_id`, `price`, `unit_price`, `total`, `subtotal`, `status`, `payment_status`, `scope`, `hash`...) ⇒ server lập tức trả `400 INVALID_BODY`.
- Trường `quote_token` gửi vào `POST /api/v1/orders/quote` bị coi là trường lạ ⇒ trả `400 INVALID_BODY`.
- `quantity` phải là số nguyên JSON chuẩn; các giá trị số thực `1.5`, `1.0`, số mũ `1e2`, chuỗi `"1"`, số `0` hoặc số âm đều bị từ chối với lỗi `400 INVALID_FIELD` trên trường `items[i].quantity`.

---

## 5. Bảng ánh xạ mã lỗi HTTP

Khớp chính xác với hàm `respondError` trong `server/internal/interfaces/http/order_handler.go`:

| HTTP Status | Error Code | Field (nếu có) | Nguyên nhân và xử lý |
| --- | --- | --- | --- |
| `400` | `INVALID_BODY` | `"Content-Type"`, `"body"` | Sai Content-Type, JSON hỏng, có dữ liệu thừa sau JSON, payload > 64 KiB, hoặc chứa trường lạ. |
| `400` | `INVALID_FIELD` | Tên trường vi phạm | Dữ liệu trường không đúng quy cách (UUID sai, token sai, phương thức thanh toán sai, thiếu địa chỉ bắt buộc, số lượng sai...). |
| `401` | `AUTH_REQUIRED` | *(trống)* | Cookie phiên không hợp lệ, hết hạn, hoặc endpoint yêu cầu đăng nhập tài khoản. |
| `403` | `ORIGIN_FORBIDDEN` | *(trống)* | Header `Origin` gửi lên không nằm trong danh sách origin được cấu hình phép gọi. |
| `404` | `NOT_FOUND` | *(trống)* | Đơn hàng không tồn tại, hoặc người dùng hiện tại không có quyền truy cập đơn hàng này. |
| `409` | `IDEMPOTENCY_CONFLICT`| *(trống)* | `Idempotency-Key` đã được sử dụng trước đó trong cùng scope với nội dung (payload) khác. |
| `409` | `OUT_OF_STOCK` | *(trống)* | Một hoặc nhiều sản phẩm trong đơn đã hết hàng hoặc không đủ tồn kho để đáp ứng. |
| `409` | `QUOTE_CHANGED` | *(trống)* | Đơn giá sản phẩm hoặc danh mục sản phẩm đã bị thay đổi sau khi báo giá được tạo. |
| `409` | `QUOTE_EXPIRED` | *(trống)* | Báo giá (`quote_token`) đã hết hạn hiệu lực; client cần gọi lại `/orders/quote` để lấy giá mới. |
| `503` | `SHIPPING_UNAVAILABLE`| *(trống)* | Phương thức vận chuyển chưa được cấu hình phí phía máy chủ (thiếu `ORDER_COD_SHIPPING_FEE`). |
| `503` | `ORDER_UNAVAILABLE` | *(trống)* | Dịch vụ tạm thời không khả dụng: cơ sở dữ liệu không hỗ trợ transaction, lỗi tồn kho lệch, hoặc lỗi hạ tầng nội bộ. |

Định dạng JSON lỗi trả về:
```json
{
  "error": {
    "code": "INVALID_FIELD",
    "field": "payment_method",
    "message": "Chỉ hỗ trợ thanh toán khi giao hàng hoặc tại quầy"
  }
}
```

---

## 6. Xử lý tiền tệ chính xác (Money Exact)

1. **Không dùng `float64`**: Mọi phép tính tiền tệ trong hệ thống sử dụng kiểu `Money` (dựa trên `math/big.Int` với hệ số nguyên và số mũ cơ số 10).
2. **Độ chính xác và giới hạn**: Giới hạn tối đa 34 chữ số có nghĩa (tương thích tuyệt đối với chuẩn IEEE 754-2008 `Decimal128` của MongoDB). Nếu chuỗi tiền vượt quá 34 chữ số có nghĩa sẽ bị từ chối với lỗi tràn số `ErrMoneyOverflow`.
3. **Lưu trữ BSON trong MongoDB**:
   - Tiền là đại lượng không âm: khi giá trị là số nguyên vừa vặn trong khoảng `0..math.MaxInt64`, repository lưu dưới dạng BSON `int64` (BSON Type `\x12`).
   - Khi giá trị tiền có phần thập phân hoặc vượt quá `MaxInt64`, repository lưu dưới dạng `bson.Decimal128` (BSON Type `\x13`).
   - Đọc dữ liệu: Repository chấp nhận cả BSON `int32`, `int64`, và `Decimal128` (kể cả số mũ dương dạng `1E+3`). Từ chối số thực double (`\x01`), số âm, NaN, +Inf, -Inf.
4. **Hiển thị JSON**: `Money.MarshalJSON()` serialize tiền thành số dạng JSON number (ví dụ `500000`, `199999.5`), không bao giờ đặt trong dấu ngoặc kép.

---

## 7. Quote Token

1. **Thuật toán sinh và ký**:
   `base64url(payload JSON) + "." + base64url(HMAC-SHA256(secret, payload JSON))`
2. **Nội dung payload**:
   - `v`: Phiên bản định dạng payload (`1`).
   - `exp`: Thời điểm hết hạn (Unix timestamp tính bằng giây).
   - `payment_method`: Phương thức thanh toán được báo giá (`cod` hoặc `cash_at_counter`).
   - `items`: Danh sách dòng sản phẩm đã gộp và sắp xếp (`product_id`, `sku`, `quantity`, `unit_price`).
   - `shipping_fee`: Phí vận chuyển đã tính.
   - `subtotal`: Tổng tiền hàng.
   - `total`: Tổng tiền đơn hàng cần thanh toán.
3. **Phạm vi bảo vệ**: Quote token niêm phong phương thức, đơn giá, số lượng, phí vận chuyển và tổng tiền tại thời điểm báo giá.
   - Token **không** niêm phong địa chỉ giao hàng vì phí COD hiện tại là mức cố định toàn hệ thống. Trong tương lai, nếu phí vận chuyển phụ thuộc vào địa chỉ (tỉnh/thành, khoảng cách), trường địa chỉ sẽ được bổ sung vào payload và phiên bản `v` sẽ được nâng lên.
4. **Thời gian hiệu lực (TTL)**: Được cấu hình qua biến môi trường `ORDER_QUOTE_TTL` (mặc định 15 phút). Hết hạn token sẽ bị từ chối với lỗi 409 `QUOTE_EXPIRED`.

---

## 8. Cơ chế Idempotency và Canonical Request

1. **Scope bảo vệ**:
   - Tài khoản đăng nhập: `customer:<ObjectId hex 24 ký tự>`
   - Khách vãng lai: `guest:<SHA-256 hex 64 ký tự của 32 byte secret>`
   - Đơn hàng của khách này không bao giờ có thể can thiệp hoặc nhận diện trùng lặp với đơn hàng của khách khác.
2. **Canonical Request Hash**:
   - Chuỗi JSON UTF-8 thu gọn, không khoảng trắng thừa, không escape ký tự HTML (`<>&`), các trường sắp xếp theo thứ tự cố định:
     `payment_method`, `shipping_address`, `items`, `note`, `quote_token`.
   - Các dòng sản phẩm trùng `(product_id, sku)` được gộp lại bằng phép cộng số lượng, sau đó sắp xếp tăng dần theo `product_id` rồi đến `sku`.
   - Hash: SHA-256 lowercase hex (64 ký tự). Fixture kiểm thử đối chiếu: `server/docs/fixtures/DW_CHECKOUT_CANONICAL.json`.
   - *Ghi chú kỹ thuật*: Go xuất chuỗi escape `\u2028`/`\u2029` còn `JSON.stringify` của JavaScript xuất ký tự UTF-8 thô; vì thế chuỗi canonical request hash do server tự tính từ payload chuẩn hóa, client không cần và không nên tự tính hash.
3. **Thứ tự xử lý trong API tạo đơn**:
   - **Bước 1**: Xác thực danh tính và tra cứu đơn theo `(scope, request_id)` trong cơ sở dữ liệu **trước** khi kiểm tra giá và tồn kho.
   - **Bước 2**: Nếu đã có đơn với cùng `request_hash` ⇒ trả về đơn đã tạo trước đó (`200 OK`, không trừ tồn lại). Nếu có đơn nhưng khác `request_hash` ⇒ trả `409 IDEMPOTENCY_CONFLICT`.
   - **Bước 3**: Nếu chưa có đơn ⇒ tiến hành mở session transaction, kiểm tra quote token, khóa tồn kho và ghi đơn mới (`201 Created`).
4. **Xử lý xung đột đua cùng key (Concurrency)**:
   - Index unique `uq_orders_checkout_scope_request` ngăn chặn việc ghi 2 đơn hàng có cùng scope và request ID.
   - Giao dịch nào đến sau bị lỗi duplicate key sẽ tự động đọc lại đơn hàng đã commit thành công của giao dịch trước và so sánh hash để phản hồi tương ứng.
5. **Mất kết nối sau khi commit (Lost Response)**:
   - Khi client bị rớt mạng và không rõ đơn hàng đã được tạo thành công hay chưa, client gửi lại `GET /api/v1/orders/attempts/{request_id}` với cùng credential ban đầu.
   - Phản hồi `404 NOT_FOUND` từ endpoint attempt khi request gốc vẫn đang xử lý dở (in-flight) không có nghĩa là thất bại; client nên tiếp tục giữ nguyên key và retry lại sau vài giây.

---

## 9. Quản lý tồn kho và giao dịch MongoDB

1. **Transaction trong Session**:
   - Sử dụng `mongo.Client.StartSession` kết hợp `WithTransaction` (Read Concern: `snapshot`, Write Concern: `majority`, timeout toàn cục: 8 giây).
   - Mọi thao tác kiểm tra giá/tồn, trừ tồn kho và tạo đơn được thực hiện trong cùng transaction.
2. **Cập nhật tồn kho có điều kiện (Single UpdateOne per Product)**:
   - Trừ tồn kho bằng một lệnh `UpdateOne` có điều kiện cho mỗi sản phẩm, tuyệt đối không dùng thao tác read-then-set hoặc ghi đè toàn bộ mảng `variants`:
     - **Sản phẩm có variants**: Filter `_id`, `is_active: true`, `stock: {$gte: tổng_cần_trừ}`, `$all` elemMatch `{sku, stock: {$gte: q}}`, và `$expr` tổng `variants.stock == stock`. Cập nhật `$inc: {stock: -tổng_cần_trừ, "variants.$[vN].stock": -q}`, `$set: {updated_at: now}` với `arrayFilters: [{"vN.sku": sku}]`.
     - **Sản phẩm legacy không variants**: Filter `_id`, `is_active: true`, `sku: sku_gốc`, `stock: {$gte: q}`, variants rỗng; cập nhật `$inc: {stock: -q}`.
   - Không can thiệp vào bảng `inventory_operations`, không ghi đè `version` metadata của admin.
3. **Quy tắc tồn kho lệch (Inconsistent Stock - PD-03/Q-07)**:
   - Nếu lệnh `UpdateOne` không match tài liệu, repository đọc lại sản phẩm ngay trong transaction để phân loại nguyên nhân:
     - Nếu phát hiện tổng tồn kho của các biến thể `variants[].stock` không khớp với tồn kho tổng `stock` của sản phẩm ⇒ trả về lỗi `ErrStockInconsistent` (được bọc thành `503 ORDER_UNAVAILABLE`).
     - Server **không tự ý sửa tồn kho**.
     - **Cách đối soát và xử lý**: Định kỳ chạy aggregation pipeline để quét tìm các sản phẩm có tổng variant lệch với stock cấp product:
       ```javascript
       db.products.aggregate([
         { $match: { variants: { $exists: true, $ne: [] } } },
         { $addFields: { variantSum: { $sum: "$variants.stock" } } },
         { $match: { $expr: { $ne: ["$stock", "$variantSum"] } } },
         { $project: { _id: 1, name: 1, stock: 1, variantSum: 1 } }
       ])
       ```
       Người quản trị (Admin) sử dụng API catalogadmin (`adjust-inventory`) để hiệu chỉnh lại số lượng tồn kho cho khớp.
4. **Điều kiện mở bán**: Sản phẩm chỉ bán được khi thỏa mãn điều kiện hiển thị công khai của catalog: `product.is_active == true`, đồng thời danh mục trực tiếp và tất cả các danh mục tổ tiên (ancestors) đều đang `is_active == true`.
5. **Máy chủ MongoDB Standalone**: MongoDB standalone (không phải replica set hoặc mongos) không hỗ trợ transaction ⇒ `503 ORDER_UNAVAILABLE`, không thực hiện bất kỳ thao tác ghi rời rạc nào vào cơ sở dữ liệu.

---

## 10. Cấu hình hệ thống (Configuration)

Các biến môi trường liên quan đến đơn hàng:

| Tên biến | Kiểu dữ liệu | Mặc định | Ý nghĩa và hành vi |
| --- | --- | --- | --- |
| `ORDER_QUOTE_SECRET` | Chuỗi ký tự | *(Không có)* | Khóa bí mật dùng để ký HMAC cho quote token. Bắt buộc có độ dài tối thiểu 32 byte. Nếu thiếu: server ghi log cảnh báo `warn` khi khởi động; mọi request gọi quote và tạo đơn sẽ trả `503 ORDER_UNAVAILABLE`. |
| `ORDER_QUOTE_TTL` | Duration | `15m` | Thời gian sống của báo giá (ví dụ: `15m`, `30m`). Phải là khoảng thời gian dương hợp lệ. |
| `ORDER_COD_SHIPPING_FEE` | Chuỗi Decimal | *(Không có)* | Mức phí vận chuyển áp dụng cho đơn hàng COD (số tiền không âm). Nếu thiếu: server ghi log cảnh báo `warn` khi khởi động; request quote hoặc tạo đơn COD sẽ trả `503 SHIPPING_UNAVAILABLE`. Q-04 chưa chốt mức phí cố định vận hành. |

---

## 11. Migration tạo Index duy nhất cho mã đơn hàng

- **Tên migration**: `274a441a-5afb-4394-9384-9f440f7424db_order_number_unique`
- **Tập tin**: `server/migrations/274a441a-5afb-4394-9384-9f440f7424db_order_number_unique/`
- **Mục đích**: Tạo partial unique index `uq_orders_order_number` trên collection `orders` với key `{order_number: 1}` và điều kiện lọc `partialFilterExpression: {order_number: {$type: "string"}}`.
- **Đặc tính tương thích**: Sử dụng partial index cho phép cơ sở dữ liệu có các đơn hàng cũ (legacy) chưa có trường `order_number` mà không bị vi phạm ràng buộc duplicate key `null`.
- **Rollback (`down.json`)**: Thực hiện `dropIndexes` xóa index `uq_orders_order_number`.
- **Kiểm tra trước khi áp dụng lên dữ liệu có sẵn**:
  Trước khi chạy migration trên cơ sở dữ liệu vận hành có dữ liệu cũ, cần thực hiện truy vấn sau bằng MongoDB shell để đảm bảo không có mã đơn hàng dạng chuỗi bị trùng lặp:
  ```javascript
  db.orders.aggregate([
    { $match: { order_number: { $type: "string" } } },
    { $group: { _id: "$order_number", count: { $sum: 1 } } },
    { $match: { count: { $gt: 1 } } }
  ])
  ```
  *(Ghi chú: Migration này hiện mới áp dụng trong stack test tích hợp, chưa chạy trên database vận hành production).*

---

## 12. Hướng dẫn kiểm thử và Danh mục Test Case

### 12.1. Các lệnh kiểm thử chính

Từ thư mục `server/`:

```bash
# 1. Chạy toàn bộ unit test với race detector
make test

# 2. Khởi động môi trường MongoDB test có hỗ trợ Transaction (Replica Set cổng 27018)
make transactions-up

# 3. Khởi động môi trường MongoDB test Standalone (Cổng 27019)
make standalone-up

# 4. Chạy Integration Test trên DB thật (yêu cầu MONGO_TEST_URI, không in credential ra file)
export MONGO_TEST_URI='<URI replica set test, xem biến mặc định trong docker-compose.transactions.yml; thêm directConnection=true>'
make test-integration

# 5. Chạy Integration Test kiểm tra từ chối Standalone (yêu cầu MONGO_STANDALONE_TEST_URI)
export MONGO_STANDALONE_TEST_URI='<URI standalone test, xem biến mặc định trong docker-compose.transactions.yml; thêm directConnection=true>'
make test-orders-standalone
```

*(Lưu ý: Stack Docker Compose test yêu cầu cấu hình `ulimits: nofile: 64000` trong `docker-compose.transactions.yml` để mongod không bị sự cố tràn file descriptors khi tạo và drop database kiểm thử liên tục).*

### 12.2. Bảng ánh xạ các kịch bản kiểm thử (Handoff Mục 6 ⇔ Test Case thật)

| STT | Kịch bản kiểm thử (Handoff mục 6) | Tên Test Case thật trong Source Code | Tập tin nguồn |
| --- | --- | --- | --- |
| 1 | Guest tại quầy chỉ tên/điện thoại | `TestOrderCheckoutScenarios/GuestCashAtCounterOnlyNamePhone`<br>`TestOrderHTTPEndToEndIntegration` | `internal/infrastructure/mongo/order_integration_test.go`<br>`internal/interfaces/http/order_http_integration_test.go` |
| 2 | Guest/account COD | `TestOrderCheckoutScenarios/CustomerCODDecimalAndIntegerPrice`<br>`TestOrderHTTPEndToEndIntegration` | `internal/infrastructure/mongo/order_integration_test.go`<br>`internal/interfaces/http/order_http_integration_test.go` |
| 3 | Bank/card/e_wallet, gửi customer_id/price/status | `TestNormalizeValidation`<br>`TestNormalizePaymentMethodBankTransferCardEWallet`<br>`TestInvalidBodyScenarios`<br>`TestOrderHTTPEndToEndIntegration` | `internal/domain/order/order_test.go`<br>`internal/interfaces/http/order_handler_test.go`<br>`internal/interfaces/http/order_handler_test.go`<br>`internal/interfaces/http/order_http_integration_test.go` |
| 4 | Cookie sai và guest token sai/không canonical | `TestGuestAccessTokenRejections`<br>`TestAuthCookieInvalidRejection`<br>`TestGuestTokenValidationAndPOSTCredentials` | `internal/domain/order/identity_test.go`<br>`internal/interfaces/http/order_handler_test.go`<br>`internal/interfaces/http/order_handler_test.go` |
| 5 | Cùng payload khác thứ tự field/dòng trùng SKU | `TestNormalizeValidation`<br>`TestCanonicalJSONAndFixture`<br>`TestCanonicalEquivalenceAndDifference` | `internal/domain/order/order_test.go` |
| 6 | Cùng scope/key, cùng payload gửi tuần tự/đồng thời | `TestOrderCheckoutScenarios/SequentialReplayWithSameKeyAndPayload`<br>`TestOrderCheckoutScenarios/ConcurrentIdempotentRequests`<br>`TestCreateSuccess201AndReplay200`<br>`TestOrderHTTPEndToEndIntegration` | `internal/infrastructure/mongo/order_integration_test.go`<br>`internal/interfaces/http/order_handler_test.go`<br>`internal/interfaces/http/order_http_integration_test.go` |
| 7 | Cùng scope/key khác payload | `TestOrderCheckoutScenarios/IdempotencyConflictAndDifferentScopes`<br>`TestOrderHTTPEndToEndIntegration` | `internal/infrastructure/mongo/order_integration_test.go`<br>`internal/interfaces/http/order_http_integration_test.go` |
| 8 | Cùng key ở hai scope khác nhau | `TestOrderCheckoutScenarios/IdempotencyConflictAndDifferentScopes` | `internal/infrastructure/mongo/order_integration_test.go` |
| 9 | SKU còn 1, hai khách đặt đồng thời | `TestOrderCheckoutScenarios/OutOfStockRace` | `internal/infrastructure/mongo/order_integration_test.go` |
| 10 | Một đơn nhiều product/SKU, dòng cuối lỗi | `TestOrderCheckoutScenarios/OrderThreeProductsLastOOS` | `internal/infrastructure/mongo/order_integration_test.go` |
| 11 | Giá decimal, quantity lớn hoặc tiền vượt giới hạn | `TestMoneyParseAndFormat`<br>`TestMoneyOverflow`<br>`TestMoneyRejections`<br>`TestMoneyMultiplicationByQuantity`<br>`TestDecodeMoneyRejectsDouble`<br>`TestDecodeMoneyPositiveExponent`<br>`TestDecodeMoneyRejectsNegativeNaNAndInf`<br>`TestRoundTripDecimalAndIntegerMoney`<br>`TestInvalidQuantityScenarios` | `internal/domain/order/money_test.go`<br>`internal/infrastructure/mongo/order_money_test.go`<br>`internal/interfaces/http/order_handler_test.go` |
| 12 | Sửa giá/tồn sau quote | `TestOrderCheckoutScenarios/PriceOrStockChangedAfterQuote` | `internal/infrastructure/mongo/order_integration_test.go` |
| 13 | Mất response sau commit/commit outcome chưa rõ | `TestOrderQueriesAndRecoveryScenarios/LostResponseAfterCommit`<br>`TestOrderHTTPEndToEndIntegration` | `internal/infrastructure/mongo/order_integration_test.go`<br>`internal/interfaces/http/order_http_integration_test.go` |
| 14 | Admin sửa metadata/adjust inventory đồng thời mua | `TestOrderCatalogAndAdminScenarios/ConcurrentOrdersWithAdminInventoryAdjustmentAndUpdateProduct` | `internal/infrastructure/mongo/order_integration_test.go` |
| 15 | Thay catalog/profile sau đặt | `TestOrderCatalogAndAdminScenarios/ProductModifiedAfterOrderRetainsSnapshot` | `internal/infrastructure/mongo/order_integration_test.go` |
| 16 | Account B đổi ID đơn A / guest token khác | `TestOrderQueriesAndRecoveryScenarios/ListPaginationStableAndAuthorization`<br>`TestOrderHTTPEndToEndIntegration`<br>`TestViewerAuthorizationAndQueryConstraints` | `internal/infrastructure/mongo/order_integration_test.go`<br>`internal/interfaces/http/order_http_integration_test.go`<br>`internal/interfaces/http/order_handler_test.go` |
| 17 | MongoDB không hỗ trợ transaction | `TestStandaloneTransactionUnavailable` | `internal/infrastructure/mongo/order_standalone_integration_test.go` |
| 18 | Log/response và CORS | `TestCORSAndPreflight`<br>`TestNoSensitiveDataInLogs`<br>`TestCreateSuccess201AndReplay200` | `internal/interfaces/http/order_handler_test.go` |
| - | Trùng mã đơn hàng `order_number` ngẫu nhiên (Retry) | `TestOrderCheckoutScenarios/DuplicateOrderNumberRollbackAndRetry` | `internal/infrastructure/mongo/order_integration_test.go` |
| - | Migration index `order_number` duy nhất | `TestMigrationOrderNumberUnique` | `internal/infrastructure/mongo/order_integration_test.go` |
| - | **Ca chưa có test tự động** | Kiểm thử giao diện checkout trên trình duyệt E2E | Chưa có UI (sẽ thực hiện tại Issue #21 / #23) |

---

## 13. Giao diện mở rộng cho KR-04 (Verified Purchase)

Để phục vụ tính năng đánh giá sản phẩm sau khi mua hàng ở hợp đồng KR-04, tầng ứng dụng cung cấp phương thức xác thực:

```go
func (s *Service) VerifiedPurchase(ctx context.Context, viewer Viewer, orderID, productID string) (bool, error)
```

**Quy tắc nghiệp vụ đối chiếu `server/internal/application/order/service.go`**:
- Trả về `true` khi và chỉ khi:
  1. `viewer` là chủ đơn hàng: `viewer.CustomerID == order.CustomerID` đối với đơn tài khoản, hoặc `viewer.GuestHash == order.GuestAccessHash` đối với đơn guest (quản trị viên Admin **không** tự động là chủ đơn).
  2. Đơn hàng có trạng thái `orders.status == "delivered"`. Trạng thái `paid` hoặc `pending` không được coi là đã mua hoàn tất.
  3. `productID` nằm trong danh sách `items[].product_id` của đơn hàng.
- Nếu không tìm thấy đơn hoặc không đúng quyền chủ đơn, trả về `(false, nil)`. Nếu có lỗi cơ sở dữ liệu, trả về `(false, err)`.
- *Ghi chú*: Phương thức này hiện chưa gắn vào route HTTP công khai nào trong DW-02; KR-04 sẽ gọi trực tiếp phương thức này khi tiếp nhận tính năng tạo review.

---

## 14. Giới hạn và các điểm cần lưu ý

1. **Q-04**: Mức phí giao hàng COD vận hành thực tế chưa được quyết định. Biến `ORDER_COD_SHIPPING_FEE` cần được cấu hình rõ ràng trong môi trường triển khai.
2. **Chưa áp dụng DB vận hành**: Migration `order_number_unique` chưa được chạy trên cơ sở dữ liệu production cổng 27017.
3. **Kiểm tra trùng mã đơn**: Cần chạy lệnh aggregate kiểm tra tính duy nhất của `order_number` trên dữ liệu thật trước khi áp dụng migration.
4. **Giao diện người dùng**: Chưa có giao diện checkout trên trình duyệt; Issue #21 sẽ thực hiện phần kết nối UI.
5. **Đọc danh mục trong snapshot**: Danh mục bị tắt công khai ngay tại thời điểm commit transaction không gây ra lỗi `WriteConflict` vì thao tác đọc danh mục nằm trong snapshot session của MongoDB.
6. **Biến thể trùng SKU**: Trường hợp dữ liệu sản phẩm bị lỗi dẫn đến có 2 biến thể trùng SKU trong cùng một sản phẩm chưa bị chặn ở tầng schema của MongoDB; nếu có, `arrayFilters` của MongoDB sẽ trừ tồn kho của cả hai biến thể.
7. **Chuyển đổi từ Guest sang Tài khoản**: Nếu khách hàng tạo đơn với tư cách guest, sau đó đăng nhập tài khoản ngay trong lúc kết quả giao dịch chưa xác định, việc gửi lại request POST tạo đơn có cookie tài khoản cùng header `X-Order-Access-Token` sẽ bị từ chối `400 INVALID_FIELD` do trộn phiên tài khoản với guest token trong POST tạo đơn. Trong trường hợp này, client cần tra cứu kết quả attempt bằng token guest ban đầu tại `/api/v1/orders/attempts/{request_id}` (hoặc không gửi guest token khi đã đăng nhập).
