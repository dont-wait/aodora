# Hợp đồng checkout, guest access và chống gửi lặp

DW-01 bàn giao schema/index và hợp đồng này cho [DW-02 #19](https://github.com/dont-wait/aodora/issues/19), DW-03 và KR-04. **Các endpoint dưới đây đã được triển khai và đăng ký trong router theo tài liệu [DW02_ORDER_API.md](DW02_ORDER_API.md).** Không dùng document DB mẫu làm request tạo đơn; server phải lấy giá, tên/SKU và danh tính từ nguồn có thẩm quyền.

## Phương thức nhận và tiền

Đặt mới chỉ nhận `cod` hoặc `cash_at_counter`. COD bắt buộc `shipping_address.recipient`, `phone`, `line1`, `city`; tại quầy chỉ bắt buộc recipient/phone và phí 0. Field tùy chọn không có thì bỏ khỏi payload; không chèn địa chỉ giả/null cho field có kiểu string. Server trim/từ chối chuỗi trắng. `customer_id`, `status`, `payment_status`, giá, tổng và metadata kỹ thuật không được nhận tùy ý từ JSON client.

Phí COD do cấu hình phía server theo Q-04, không tự mặc định miễn phí. Chưa cấu hình thì trả `SHIPPING_UNAVAILABLE`. Quote trả tiền hàng/phí/tổng trước xác nhận. Quote token là dữ liệu opaque được server xác thực; server kiểm tra lại giá/tồn/phí khi tạo đơn. Quote hết hạn hoặc giá/phí đổi trả conflict, yêu cầu khách kiểm tra và xác nhận lại.

## Danh tính và quyền guest — Q-03

- Tài khoản dùng session cookie hiện có; server lấy ID qua auth. Có cookie nhưng session không hợp lệ phải trả 401, không hạ xuống guest. Tài khoản không gửi guest token; server không nhận scope/customer_id từ client.
- Guest tạo 32 byte ngẫu nhiên bằng Web Crypto trước lần gửi đầu, mã hóa base64url không padding (43 ký tự), giữ cùng attempt qua reload/retry. Truyền qua header `X-Order-Access-Token`, không qua URL, query string, mã đơn hoặc số điện thoại. Server kiểm tra đúng base64url canonical/32 byte và chỉ lưu SHA-256 lowercase hex trên 32 byte đã decode (không băm chuỗi base64url), không lưu/echo token gốc. [Fixture đối chiếu](fixtures/DW_CHECKOUT_ACCESS.json) dùng bytes 0..31 chỉ dành cho test, không dùng làm credential thật.
- Secret này là credential cho đơn, không chứng minh ai biết số điện thoại. Mỗi thao tác đặt mới tạo secret mới. Client giữ attempt/key/secret trong bộ nhớ và vùng lưu trữ của phiên trình duyệt để retry; trang xác nhận phải cho khách lưu mã truy cập riêng trước khi mất phiên. Người biết mã đơn công khai không có quyền đọc/review. Mất secret thì không khôi phục qua số điện thoại tự khai; chủ cửa hàng xác minh ngoài luồng trước khi cấp lại quyền ở DW-03. Không giả lập OTP/email chưa có dịch vụ.
- Khi đọc đơn guest, server yêu cầu `customer_id` thiếu/null và hash credential khớp đúng đơn; đơn tài khoản chỉ được đọc bởi chủ đơn/admin. Trả 404 chung khi không có quyền vào ID đó, tránh xác nhận đơn người khác tồn tại. Không trả/log hash, checkout scope, request hash hoặc token trong response công khai.
- GET/POST dùng cùng quy tắc origin/credentials hiện có; CORS phải cho hai header checkout đã chốt. Handler không ghi body/header credential vào log và response riêng tư dùng `Cache-Control: no-store`.

## Metadata DB và idempotency

Migration UUID `order_checkout_access` thêm:

| Field | Quy tắc |
| --- | --- |
| `checkout.request_id` | UUID canonical lowercase từ header `Idempotency-Key`; cùng thao tác retry giữ nguyên |
| `checkout.scope` | Server tính `customer:<ObjectId hex>` hoặc `guest:<SHA-256 secret>` |
| `checkout.request_hash` | SHA-256 lowercase hex của payload nghiệp vụ đã normalize/canonicalize |
| `guest_access_hash` | Chỉ guest có checkout marker: SHA-256 secret; tài khoản không có field này |

DB ràng buộc scope khớp chính customer_id hoặc guest hash. Document có checkout marker chỉ nhận hai phương thức đặt mới. Document legacy không có marker giữ enum cũ và không bị rewrite. API tạo mới ở DW-02 luôn lưu marker, không sử dụng đường legacy để bypass giới hạn.

Index `uq_orders_checkout_scope_request` unique trên `(checkout.scope, checkout.request_id)`, partial chỉ các document có hai field này kiểu string. Legacy thiếu marker không bị coi là một key null trùng nhau. Không thêm TTL xóa history chống lặp: retry muộn không được tạo đơn thứ hai. Index lịch sử khách/trạng thái hiện có được giữ; chỉ bổ sung index danh sách khác sau khi có query/explain thực tế ở DW-02.

Canonical request là JSON UTF-8 không khoảng trắng, field có thứ tự cố định: `payment_method`, `shipping_address` (recipient, phone, line1, ward, district, city), `items` (gộp dòng trùng product_id/SKU rồi sort product_id/SKU; mỗi dòng product_id, sku, quantity), `note`, `quote_token`. Trim chuỗi, ID lowercase, optional vắng mặt giữ vắng mặt; số lượng phải là integer dương; không biến missing thành null. Giữ nguyên quote token trong cùng attempt. Bộ fixture canonicalization sẽ chạy qua Go/client khi triển khai DW-02/DW-03; không dùng việc JSON stringify tùy thứ tự làm bằng chứng tương thích.

Server xác thực danh tính/credential trước, rồi tìm `(scope, request_id)` **trước khi kiểm tra lại quote/giá/tồn có thể đã đổi**. Có đơn cùng request hash thì trả đơn cũ, không trừ tồn lại; hash khác trả `IDEMPOTENCY_CONFLICT`. Chưa có thì xác thực quote và tạo đơn/trừ tồn trong một MongoDB transaction. Unique conflict khi hai attempt đồng thời phải đọc lại đơn đã commit và so hash; commit timeout/unknown outcome phải tra cùng key, không đổi key để đoán tạo lại. Attempt gắn với danh tính/credential lúc gửi; nếu đăng nhập hoặc đổi tài khoản khi kết quả chưa rõ, phải tra attempt bằng danh tính gốc trước, không chuyển cùng thao tác sang scope mới để tạo lại. Không tạo reservation/đơn giả bên ngoài transaction để báo thành công. 404 từ tra attempt trong lúc request cũ còn in-flight không chứng minh thất bại: vẫn retry cùng key/body/credential, không đổi key hoặc secret chỉ vì chưa nhìn thấy commit.

## API tạo và đọc đơn

| Endpoint | Quyền | Hành vi |
| --- | --- | --- |
| `POST /api/v1/orders/quote` | Guest/tài khoản | Kiểm tra lựa chọn/địa chỉ, trả giá/phí/tổng và quote token; chưa lưu đơn/trừ tồn |
| `POST /api/v1/orders` | Guest/tài khoản | Bắt buộc Idempotency-Key; guest thêm X-Order-Access-Token; tạo pending/unpaid |
| `GET /api/v1/orders/{id}` | Chủ đơn/admin hoặc guest có credential khớp | Đọc chi tiết; không nhận customer_id để đổi chủ thể |
| `GET /api/v1/orders` | Tài khoản | Danh sách chỉ đơn của chính mình, có phân trang ổn định |
| `GET /api/v1/orders/attempts/{request_id}` | Cùng tài khoản hoặc guest token của attempt | Tra kết quả khi mất response; không tạo đơn và không cần quote còn hạn |

Request quote COD:

```json
{
  "payment_method": "cod",
  "shipping_address": {"recipient": "Nguyễn An", "phone": "0900000001", "line1": "12 Nguyễn Trãi", "city": "TP. Hồ Chí Minh"},
  "items": [{"product_id": "680000000000000000000001", "sku": "LUA-DO-M", "quantity": 1}]
}
```

Quote response minh họa cấu trúc (phí dưới đây không phải quyết định mức phí vận hành):

```json
{"data":{"quote_token":"opaque-server-quote","expires_at":"2026-11-16T11:15:00Z","subtotal":500000,"shipping_fee":30000,"total":530000,"items":[{"product_id":"680000000000000000000001","sku":"LUA-DO-M","quantity":1,"unit_price":500000}]}}
```

Request tạo đơn tại quầy: `Content-Type: application/json`, `Idempotency-Key: ad577a65-cd42-46eb-ae8a-98b5f1186571`. Tài khoản gửi cookie phiên; guest thêm `X-Order-Access-Token` được tạo như trên.

```json
{
  "payment_method": "cash_at_counter",
  "shipping_address": {"recipient": "Nguyễn An", "phone": "0900000001"},
  "items": [{"product_id": "680000000000000000000001", "sku": "LUA-DO-M", "quantity": 1}],
  "quote_token": "opaque-server-quote"
}
```

COD tạo đơn dùng payload COD cộng quote_token; không gửi subtotal/shipping_fee/total hoặc giá từ quote về để server tin. Thành công mới trả 201; replay cùng payload trả 200 với cùng data.id/order_number. Cả hai có cùng cấu trúc:

```json
{"data":{"id":"690000000000000000000010","order_number":"AD-EXAMPLE","status":"pending","payment_method":"cash_at_counter","payment_status":"unpaid","subtotal":500000,"shipping_fee":0,"total":500000,"shipping_address":{"recipient":"Nguyễn An","phone":"0900000001"},"items":[{"product_id":"680000000000000000000001","product_name":"Áo dài lụa","sku":"LUA-DO-M","quantity":1,"unit_price":500000}]}}
```

Response danh sách: `{"data":[...],"total":1,"page":1,"page_size":20}`. Đơn/số đo đã snapshot không đổi theo hồ sơ/catalog về sau. Không trả password/session hay metadata chống lặp. Khi pending chưa có kết quả commit xác định, API không báo thành công giả; client giữ nguyên giỏ và attempt để tra lại.

| HTTP | Code | Trường hợp |
| --- | --- | --- |
| 400 | `INVALID_BODY` / `INVALID_FIELD` | JSON/field, phương thức ngoài hai lựa chọn, địa chỉ, quantity, UUID key hoặc token sai |
| 401 | `AUTH_REQUIRED` | Phiên tài khoản không hợp lệ hoặc endpoint bắt buộc tài khoản |
| 403 | `ORIGIN_FORBIDDEN` / `ADMIN_REQUIRED` | Origin hoặc quyền quản trị sai |
| 404 | `NOT_FOUND` | Không tìm thấy hoặc không có quyền đọc đơn/attempt |
| 409 | `IDEMPOTENCY_CONFLICT` | Cùng scope/key với nội dung khác |
| 409 | `OUT_OF_STOCK` / `QUOTE_CHANGED` / `QUOTE_EXPIRED` | Khách cần kiểm tra và xác nhận lại |
| 503 | `SHIPPING_UNAVAILABLE` / `ORDER_UNAVAILABLE` | Thiếu cấu hình phí hoặc DB/commit chưa xác định; giữ attempt để tra lại |

Error envelope theo server hiện có: `{"error":{"code":"INVALID_FIELD","field":"payment_method","message":"Chỉ hỗ trợ thanh toán khi giao hàng hoặc tại quầy"}}`.

## Bàn giao review cho KR-04

KR-04 gọi kiểm tra quyền ở server trong mỗi lần thêm/sửa review: đúng chủ tài khoản hoặc guest credential khớp đơn, `orders.status=delivered` và product_id thuộc items. Không tin `verified_purchase` hoặc kết quả check do trình duyệt gửi lại. Trạng thái paid không thay thế delivered.

Guest cần `reviews.order_id`, customer_id thiếu/null và unique partial `(order_id, product_id)` cho review guest. Index `(customer_id, product_id)` của tài khoản cần partial chỉ customer_id ObjectId để không ép mọi guest thành cùng một khách null. **Migration reviews do KR-04 thực hiện cùng hợp đồng này**, chưa thay collection/index reviews trong PR DW-01. API review công khai không lộ order credential, scope hay thông tin liên hệ.
