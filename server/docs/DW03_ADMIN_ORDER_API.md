# API quản trị đơn hàng và thu tiền (DW-03, issue #20)

UI checkout/đơn/admin và bằng chứng chạy thử issue #21: [DW03_CHECKOUT_UI.md](DW03_CHECKOUT_UI.md).

Phạm vi: backend quản trị đơn, giao nhận, hủy/hoàn tồn và ghi nhận thu/hoàn đủ tiền. Chưa có UI admin (issue #21). API khách và DTO của [DW02_ORDER_API.md](DW02_ORDER_API.md) giữ nguyên.

Nguồn: `internal/interfaces/http/admin_order_handler.go`, `internal/application/order/admin.go`, `internal/domain/order/admin.go`, `internal/infrastructure/mongo/order_repository.go`; wiring tại `cmd/api/main.go`.

## Quyền và header

Tất cả route dưới đây yêu cầu cookie phiên hợp lệ của tài khoản có `role: "admin"`, xác minh server bằng `authGuard.RequireAdmin`. Guest, khách thường, cookie sai/hết hạn hoặc không có authorizer đều nhận `403 ADMIN_REQUIRED`. Guest token không cấp quyền admin; không có vai trò nhân viên.

Handler tái dùng guard của admin catalog: đặt `Cache-Control: no-store`, kiểm tra origin trước rồi mới kiểm tra quyền. Không có Origin thì tiếp tục kiểm tra phiên. Origin cùng origin API được xử lý như admin catalog; frontend khác origin cần cấu hình `ADMIN_ALLOWED_ORIGIN` đúng origin (scheme + host + port), không wildcard. Origin sai trả `403 ORIGIN_FORBIDDEN`.

Preflight dùng OPTIONS `/api/v1/admin/{path...}` sẵn có của admin catalog, không đăng ký lại: từ `ADMIN_ALLOWED_ORIGIN`, phương thức GET/POST/PUT/PATCH và header Content-Type/Accept hợp lệ → `204`. Preflight không cần cookie. Ví dụ PATCH `/api/v1/admin/orders/{id}/status` được hỗ trợ.

Body có nội dung phải là `Content-Type: application/json` (cho phép tham số charset), tối đa **16 KiB**. Chỉ một JSON object, không field lạ hoặc JSON thừa. Sai Content-Type/JSON/kiểu dữ liệu, field lạ, root null/array hoặc vượt cỡ → `400 INVALID_BODY`. Cancel/restock cho phép không có body (không cần Content-Type) hoặc `{}`; không nhận reason, số lượng hay số tiền.

## Routes

Tiền tố: `/api/v1/admin/orders`. Mỗi mutation thành công trả `200 {"data": <admin order DTO>}`, gồm cả retry không thay đổi dữ liệu.

| Method | Route | Query/body | Thành công |
| --- | --- | --- | --- |
| GET | `/api/v1/admin/orders` | `status`, `payment_status`, `page`, `page_size` | `200 {"data": [...], "total": n, "page": n, "page_size": n}` |
| GET | `/api/v1/admin/orders/{id}` | Không body | `200 {"data": {...}}` |
| PATCH | `/api/v1/admin/orders/{id}/status` | `{"status":"confirmed"}` hoặc `shipping`, `delivered` | `200 {"data": {...}}` |
| POST | `/api/v1/admin/orders/{id}/cancel` | Không body hoặc `{}` | `200 {"data": {...}}` |
| POST | `/api/v1/admin/orders/{id}/restock` | Không body hoặc `{}` | `200 {"data": {...}}` |
| PATCH | `/api/v1/admin/orders/{id}/payment` | `{"payment_status":"paid"}` hoặc `refunded` | `200 {"data": {...}}` |

List nhận status: `pending`, `confirmed`, `shipping`, `delivered`, `cancelled`; payment_status: `unpaid`, `paid`, `refunded`. Bỏ hoặc để chuỗi rỗng nghĩa là không lọc field đó. Hai filter kết hợp bằng AND. Có cả đơn guest và tài khoản. Sort `created_at` giảm dần rồi `_id` giảm dần để phân trang ổn định với timestamp bằng nhau; `total` dùng cùng filter, không chỉ đếm trang hiện tại. Trang rỗng trả `data: []`.

`page` mặc định 1, số nguyên 1..100000. `page_size` mặc định 20, số nguyên >= 1, service kẹp tối đa 100. Giá trị không biểu diễn được bằng int64 bị từ chối. Query lạ, lặp tham số hoặc query encoding hỏng → `400 INVALID_FIELD`, field `query`; page/page_size sai → đúng tên field. Enum sai → `400 INVALID_FIELD`, field `status`/`payment_status`.

Ví dụ đọc trang:

```http
GET /api/v1/admin/orders?status=shipping&payment_status=unpaid&page=1&page_size=20 HTTP/1.1
Cookie: aodora_session=<cookie do route auth cấp; tên theo AUTH_COOKIE_NAME>
Origin: https://shop.example
```

Ví dụ ghi nhận đã thu đủ tiền:

```http
PATCH /api/v1/admin/orders/690000000000000000000010/payment HTTP/1.1
Content-Type: application/json
Cookie: aodora_session=<cookie admin>

{"payment_status":"paid"}
```

Không nhận `amount`, `payment_method`, `status` trong route payment; không thu từng phần và không tích hợp cổng tự động chuyển tiền.

## Admin order DTO

Struct riêng chỉ xuất các field được phép. `subtotal`, `shipping_fee`, `total`, `items[].unit_price` là số JSON chính xác từ domain Money, không dùng float. Field nền giống DTO khách: `id`, `order_number`, `status`, `payment_method`, `payment_status`, `subtotal`, `shipping_fee`, `total`, `shipping_address`, `items`, `note` (bỏ khi rỗng), `created_at`. Address gồm recipient/phone và line1/ward/district/city nếu có. Items gồm product_id/product_name/sku/color/size/quantity/unit_price và thêm `fulfillment_mode` (`ready`, `custom`; chuỗi rỗng nếu đơn legacy không có field).

| Field thêm | Kiểu và ý nghĩa |
| --- | --- |
| `updated_at` | Thời gian cập nhật đơn |
| `customer_id` | ID tài khoản; bỏ khi đơn guest |
| `is_guest` | Boolean, true khi customer_id rỗng |
| `cancelled_at` | Thời gian hủy; bỏ khi chưa có |
| `inventory_restock` | restocked / awaiting_return / not_applicable; bỏ khi chưa có |
| `inventory_restocked_at` | Thời gian hoàn tồn; bỏ khi chưa có |
| `paid_at` | Thời gian ghi nhận thu đủ; bỏ khi chưa có |
| `refunded_at` | Thời gian ghi nhận hoàn đủ; bỏ khi chưa có |
| `allowed_actions` | Mảng action được tính phía server, mảng rỗng khi không có action |

Timestamp được serialize dạng RFC3339; timestamp thêm có thể có phần giây lẻ. Admin DTO **không xuất** guest_access_hash, checkout, scope, request_id hay request_hash.

Ví dụ đơn COD bị hủy khi đang giao và chờ hàng về:

```json
{
  "data": {
    "id": "690000000000000000000010",
    "order_number": "AD-261117-8H3K9N2M",
    "status": "cancelled",
    "payment_method": "cod",
    "payment_status": "unpaid",
    "subtotal": 500000,
    "shipping_fee": 17000,
    "total": 517000,
    "shipping_address": {
      "recipient": "Khách mẫu",
      "phone": "0900000001",
      "line1": "12 Đường A",
      "city": "TP.HCM"
    },
    "items": [{
      "product_id": "680000000000000000000001",
      "product_name": "Áo Dài Lụa",
      "sku": "LUA-DO-M",
      "quantity": 1,
      "unit_price": 500000,
      "fulfillment_mode": "ready"
    }],
    "created_at": "2026-11-17T03:00:00Z",
    "updated_at": "2026-11-17T04:00:00Z",
    "is_guest": true,
    "cancelled_at": "2026-11-17T04:00:00Z",
    "inventory_restock": "awaiting_return",
    "allowed_actions": ["restock", "mark_paid"]
  }
}
```

Phí 17000 chỉ minh họa/test; phí vận hành lấy cấu hình ORDER_COD_SHIPPING_FEE theo quyết định chủ cửa hàng.

`allowed_actions` chỉ giúp UI hiển thị; không cấp quyền và không bảo đảm dữ liệu chưa thay đổi. Server luôn kiểm tra lại khi mutation:

| Action | Điều kiện |
| --- | --- |
| confirm / ship / deliver | NextStatusAllowed theo phương thức; không đưa action target == hiện tại |
| cancel | pending / confirmed / shipping |
| restock | cancelled + awaiting_return |
| mark_paid | unpaid |
| mark_refunded | paid |

Ví dụ confirmed COD → ship/cancel; confirmed tại quầy → deliver/cancel. Các action tiền được thêm độc lập, kể cả delivered/cancelled.

## Luồng giao nhận, hủy và hoàn tồn

| Phương thức | Luồng hợp lệ |
| --- | --- |
| cod | pending → confirmed → shipping → delivered |
| cash_at_counter | pending → confirmed → delivered; shipping bị từ chối |
| Legacy bank_transfer/card/e_wallet và phương thức khác đã lưu | Như COD; API không thêm phương thức tạo đơn mới |

Route status chỉ nhận target confirmed/shipping/delivered. Target hiện tại là no-op thành công, không đổi updated_at. Bước ngược, nhảy cóc hoặc rời delivered/cancelled → `409 ORDER_STATE_CONFLICT`. Target khác allowlist HTTP → `400 INVALID_FIELD`.

| Tình trạng khi hủy | Kết quả |
| --- | --- |
| pending / confirmed, có checkout và dòng ready cần hoàn | Một transaction: conditional status=cancelled + cộng tồn + restocked + timestamp |
| shipping, có dòng cần hoàn | cancelled + awaiting_return; **không cộng tồn khi hàng chưa về kho** |
| cancelled | 200 no-op, không hoàn lần hai |
| delivered | 409 ORDER_STATE_CONFLICT |
| Legacy không checkout marker hoặc toàn custom | cancelled + not_applicable, không đổi tồn |
| Mixed custom/ready | Chỉ các dòng không custom có hoàn tồn |

POST restock chỉ hợp lệ với cancelled + awaiting_return, trong một transaction đổi dấu restocked + cộng tồn. Đã cancelled + restocked → 200 no-op; chưa hủy hoặc not_applicable → 409 ORDER_STATE_CONFLICT.

Cơ chế retry/concurrency: update theo `_id` và trạng thái đã đọc, không ghi đè mù; CAS không match thì đọc lại/quyết định lại tối đa 3 vòng. Transaction đọc lại document, giữ nguyên lỗi transient cho driver retry. Chỉ một lần đổi dấu thành công mới cộng tồn; hủy đua với shipping quyết định restocked/awaiting_return theo trạng thái thực tế trong transaction.

Mỗi product một `$inc` stock tổng, cùng các variant stock theo arrayFilters SKU; filter yêu cầu mọi SKU còn tồn tại. Product legacy không variants yêu cầu SKU cấp product khớp. Không yêu cầu is_active; không ghi đè mảng variants, inventory_operations hay version.

Product/SKU đã bị xóa → `409 RESTOCK_UNAVAILABLE`, transaction abort, đơn giữ nguyên và không SKU nào được hoàn dở. Admin cần khôi phục product/SKU để hoàn tất hủy có hoàn tồn hoặc nhập hàng thu hồi.

## Thanh toán và lưu trữ

`unpaid → paid` ghi paid_at; `paid → refunded` ghi refunded_at, sau khi admin xác nhận đã thu/hoàn **đủ**. Target hiện tại là 200 no-op, timestamp không ghi lại. unpaid → refunded hoặc refunded → paid → `409 PAYMENT_STATE_CONFLICT`. Route HTTP chỉ nhận paid/refunded; target unpaid hoặc ngoài allowlist → `400 INVALID_FIELD`.

Theo **AC-05**, đổi giao nhận/hủy không đổi payment_status; đổi tiền không đổi status. Hủy đơn paid vẫn paid, không tự hoàn tiền. Có thể ghi nhận tiền cho cả đơn đã giao hoặc đã hủy. Ghi tiền/giao nhận đồng thời chỉ cập nhật field của thao tác tương ứng.

Document orders thêm field `cancelled_at` (date), `inventory_restock` (string), `inventory_restocked_at` (date), `paid_at` (date), `refunded_at` (date). Không tạo migration hoặc sửa validator: validator hiện cho phép field thêm. Mutation thực sự cập nhật updated_at, no-op không cập nhật. Đơn legacy thiếu các field này vẫn đọc được, DTO bỏ các field tùy chọn.

MongoDB standalone: hủy cần hoàn tồn và restock cần transaction → `503 ORDER_UNAVAILABLE`, không ghi một phần. Status/payment dùng update một document, không cần transaction. Hủy không cần hoàn tồn cũng không cần transaction; retry đơn đã restocked là no-op.

## Lỗi

Dạng chung: `{"error":{"code":"...","field":"...","message":"Thông báo tiếng Việt"}}`; field bỏ khi rỗng.

| HTTP | Code | Ý nghĩa |
| --- | --- | --- |
| 400 | INVALID_BODY | Content-Type/body/JSON/field lạ/dữ liệu thừa/quá 16 KiB |
| 400 | INVALID_FIELD | Query hoặc giá trị field sai; tên field trong error.field |
| 403 | ORIGIN_FORBIDDEN | Origin không được phép, kiểm tra trước quyền |
| 403 | ADMIN_REQUIRED | Không có phiên admin hợp lệ |
| 404 | NOT_FOUND | ID sai dạng hoặc không tồn tại |
| 409 | ORDER_STATE_CONFLICT | Thao tác không đúng trạng thái giao nhận/hoàn tồn |
| 409 | PAYMENT_STATE_CONFLICT | Thao tác không đúng trạng thái tiền |
| 409 | RESTOCK_UNAVAILABLE | Product/SKU không còn để hoàn tồn |
| 503 | ORDER_UNAVAILABLE | ErrOrderUnavailable và lỗi hạ tầng/khác, kể cả thiếu transaction |

Không log request body/header, token/hash/số điện thoại. Log lỗi hạ tầng chỉ ghi error_type và thông điệp cố định, tránh đưa dữ liệu document trong lỗi driver vào log.

## Kiểm thử và giới hạn

Tên test dưới đây có thật trong source, có thể tra bằng `rg '^func Test' internal/{interfaces/http,application/order,domain/order,infrastructure/mongo}` từ server/.

| Ca | Test |
| --- | --- |
| Guest/customer bị chặn, admin, origin, preflight PATCH, không trùng OPTIONS | TestAdminOrderAuthorizationAndPreflight |
| Route gọi đúng service, filter/envelope, cancel/restock rỗng hoặc {} | TestAdminOrderRoutesCallService |
| Body JSON, giới hạn 16 KiB, enum/missing/null, Content-Type | TestAdminOrderInvalidBodies |
| Query lạ/lặp/sai dạng/trang, mapping lỗi enum từ service | TestAdminOrderListQueryValidation |
| Đủ lỗi HTTP và log không lộ dữ liệu nhạy cảm | TestAdminOrderErrorMappingAndSafeLogs |
| DTO allowlist, timestamp/guest/customer và allowed_actions | TestAdminOrderDTOAndAllowedActions |
| Auth/register thật, seed role admin, guest tại quầy/COD, list, giao/thu tiền, shipping→cancel→restock, paid→refunded và guest đọc lại | TestAdminOrderHTTPEndToEndIntegration (integration) |
| Domain/service bảng luồng/no-op/độc lập trạng thái và lỗi | TestAdminTransitionTables; TestAdminServiceTransitionTables; TestAdminServicePortAndErrors; TestAdminMutationStoreErrors; TestAdminDisallowedTargets |
| DB thật: lifecycle/payment, hủy/hoàn lặp/đồng thời | TestAdminOrderLifecycle (integration) |
| 20 vòng COD và 20 vòng tại quầy đua hủy/giao | TestAdminOrderRaces (integration) |
| Legacy/custom/mixed/inactive, mất product/SKU, rollback | TestAdminOrderRestockFailuresAndLegacy (integration) |
| List ổn định khi cùng timestamp, guest/account/filter/total | TestAdminOrderList (integration) |
| Standalone không ghi một phần | TestAdminStandaloneAtomicity (integration_standalone) |
| Regression checkout/HTTP #19 | TestOrderHTTPEndToEndIntegration và suite cũ không đổi assert |

HTTP integration dùng router + AuthGuard + service + repository thật và replica set test cổng 27018; đăng ký hai tài khoản qua API, **seed role admin trực tiếp chỉ trong DB test ngẫu nhiên**. Suite tự drop DB. Đây không phải API tự nâng quyền. Standalone test dùng cổng 27019. Không chạy DB vận hành cổng 27017.

Giới hạn: chưa UI admin (#21), chưa audit log người thao tác, chưa chạy DB vận hành. Không thu/hoàn từng phần, không tự chuyển tiền, không mở rộng vai trò. Product/SKU mất có thể chặn hủy tới khi khôi phục. Q-07 được hiện thực theo quyết định **không cộng hàng chưa về kho**; quy trình thực tế cần chủ cửa hàng xác nhận.
