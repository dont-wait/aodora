# Checkout, đơn khách và quản trị đơn — DW-03, issue #21

Đối chiếu [API khách DW02](DW02_ORDER_API.md) và [API quản trị DW03](DW03_ADMIN_ORDER_API.md). Bằng chứng dưới đây chạy ngày 17/11/2026 trên commit `ecf3008`, branch `feature/dw-03-checkout-ui`. Bước này chỉ thêm tài liệu; chưa nghiệm thu trình duyệt desktop/mobile.

## Môi trường bàn giao đang chạy

| Thành phần | Giá trị |
| --- | --- |
| Vite / origin HTTP thử | http://127.0.0.1:15173 |
| API | http://127.0.0.1:18080 |
| MongoDB test | `127.0.0.1:27018`, replica set của stack `aodora-transactions` |
| Database ngẫu nhiên | `aodora_dw03ui_20261117T024542_ca3eda5c` |
| API PID / process group | `469082` |
| Vite npm PID / process group | `469083` |
| Tài khoản admin và khách thử | `/tmp/aodora-dw03-ui-creds.txt`, quyền `600`; chỉ đọc tại máy review |
| Fixture | `/san-pham/dw03-ui-checkout-fixture`; product `6afbbfd88ed904d9c6f47a15`, SKU `DW03-TEST-M`, đơn giá 500000 |

Docker đã có container `aodora-transactions-mongo-1` healthy, mapping host `127.0.0.1:27018`. Không khởi động/hủy infra và không kết nối host cổng 27017. Cổng 27017 bên trong container xuất hiện trong output Docker là cổng nội bộ được publish ra **27018**.

Migration chạy bằng binary `cmd/migrate`, gọi cùng `migration.Run(up)` như integration tests, bao gồm seed demo. In host và database trước khi migrate; helper chỉ nhận host `127.0.0.1:27018` và tên DB test đúng mẫu ngẫu nhiên. Thêm fixture tồn 100 để đo biến động độc lập với demo. Hai tài khoản được register qua HTTP thật (201), sau đó chỉ nâng role admin trực tiếp trong DB test rồi login qua HTTP (200).

API dùng `ORDER_QUOTE_SECRET` ngẫu nhiên trong bộ nhớ tiến trình, không ghi file; `ORDER_COD_SHIPPING_FEE=17000` **chỉ là số thử, không phải phí vận hành** (Q-04 chưa chốt). Quote TTL 15 phút. Cookie test `aodora_session`, secure=false cho HTTP local. `AUTH_ALLOWED_ORIGIN` và `ADMIN_ALLOWED_ORIGIN` cùng origin Vite. Vite dùng `API_PROXY_TARGET=http://127.0.0.1:18080`, `VITE_API_BASE_URL=''`.

Mọi request nghiệm thu đi qua proxy Vite `/api/...`, gửi Origin Vite; cookie jar tách riêng admin/khách/guest. Không ghi giá trị cookie, token, hash hoặc mật khẩu vào tài liệu/log. Ngoại lệ duy nhất được brief cho phép là mật khẩu thử trong file riêng quyền 600 nêu trên. Script/helper/log tạm chỉ nằm `/tmp`, không commit.

## Route và quyền

| Route client | Người dùng / credential |
| --- | --- |
| `/thanh-toan` | Guest hoặc tài khoản. Tài khoản dùng cookie phiên; guest dùng token ngẫu nhiên của attempt qua `X-Order-Access-Token`. |
| `/dat-hang-thanh-cong/:id` | Luôn GET đơn từ server; account dùng cookie, guest dùng token riêng của đúng order id. Receipt không cấp quyền. |
| `/don-hang/:id` | Chi tiết tài khoản hoặc guest, cùng cách GET và chọn credential như trang kết quả. |
| `/tai-khoan/don-hang` | Guard đăng nhập; GET `/api/v1/orders` bằng cookie, không guest header. Phân trang theo metadata server. |
| `/admin/orders` | Guard admin; GET list bằng cookie admin, lọc status/payment_status/page/page_size. |
| `/admin/orders/:id` | Guard admin; GET chi tiết và mutation bằng cookie admin. Server kiểm tra quyền lại ở mỗi request. |

Client giữ `next` là pathname + search khi chuyển đăng nhập; admin sai role đến `/403`. Không đưa secret vào URL. Guest không phải đăng ký để tạo hoặc đọc đơn. Tài khoản thường đang đăng nhập không đọc đơn guest qua client trong cùng phiên: client chọn cookie và không gửi token guest; cần đăng xuất để dùng quyền guest đã lưu. Đây là chính sách client; không suy rộng thành quy tắc từ chối credential hỗn hợp cho mọi GET của backend. POST tạo đơn có cả cookie và guest header bị từ chối 400.

## Cơ chế checkout và đọc đơn

- Q-03: token guest sinh từ 32 byte ngẫu nhiên, base64url 43 ký tự; lưu per-order ở `aodora.orders.guest.v1.<id>`. Mã đơn, số điện thoại và receipt không thay token.
- Attempt ở `aodora.orders.attempt.v1` lưu UUID dùng làm Idempotency-Key, body gốc, identity/credential và snapshot giỏ trước khi POST. Reload tra `/orders/attempts/{request_id}` trước; 404 chưa biết đơn thì retry cùng key/body/credential. Mất mạng/5xx/auth không tạo attempt mới; thành công hoặc lỗi nghiệp vụ 4xx xác định mới kết thúc attempt theo policy.
- Receipt chỉ có metadata `order_id`, `order_number`, `request_id`, `owner`, `cleanup`; không có PII, DTO đơn hoặc credential. Receipt cũ được migrate khi đọc. Trang kết quả và chi tiết luôn gọi `ordersApi.get`; GET lỗi không fallback snapshot PII cũ. Response đọc/mutation về trễ không ghi đè đơn hoặc phiên mới.
- Dọn giỏ theo snapshot SKU và lượng đã mua. Guest đối chiếu giỏ localStorage mới nhất; account GET giỏ mới nhất rồi PUT/DELETE phần lượng tương ứng. Không xóa toàn bộ giỏ hay lượng thêm sau lúc gửi. Receipt đánh dấu cleanup để tránh trừ lặp; lỗi cleanup chỉ cảnh báo, không biến đơn thành thất bại.
- Tiền từ API giữ lexeme decimal trước JSON.parse, hiển thị bằng formatter exact; không cộng tổng đơn bằng float ở client. Phí lấy từ quote, không có phí COD mặc định ở frontend. Đổi quote yêu cầu xem lại chênh lệch trước khi gửi.

## Admin và trạng thái độc lập

Nút lấy duy nhất từ `allowed_actions` server, chia Giao nhận và Thanh toán. Xác nhận modal nêu hệ quả; đang gửi khóa thao tác. Thành công dùng DTO response, không tự suy trạng thái hoặc tồn. Mutation 409 tự GET lại và thay cả dữ liệu/nút; GET lại lỗi không dùng dữ liệu cũ. Network/503 yêu cầu tải lại để biết kết quả, không tự gửi lại mutation. 401 chuyển login, 403 đến `/403`, 404 báo không thấy đơn.

COD đi confirmed → shipping → delivered. Tại quầy đi confirmed → delivered; không tự thêm `ship`. Giao nhận không đổi `payment_status`. `mark_paid`/`mark_refunded` là ghi nhận **đã thực tế thu/hoàn đủ tiền**, không tích hợp chuyển tiền. Q-07: hủy khi shipping tạo `awaiting_return`, chỉ restock khi hàng thực sự về; hủy không tự hoàn tiền.

## Bằng chứng HTTP thật

Evidence đã loại credential và thông tin liên hệ: `/tmp/aodora-dw03-ui-http.json`; output assert: `/tmp/aodora-dw03-ui-http-output.log`. Các response dưới đây chỉ trích field đã trả, không phải toàn bộ DTO. Header xác thực và quote token được bỏ khỏi ví dụ; không dùng các đoạn rút gọn này làm request create đầy đủ.

| Nhóm | Request / kết quả thực tế |
| --- | --- |
| B1 guest tại quầy | Quote 200, shipping_fee=0, không line1/city; create 201, `AD-261117-N0D07YTB` (`6afbc81a395cce1f860cbe46`). GET token đúng 200; sai 404; không credential 404. Replay cùng key/body 200 cùng id; lookup attempt 200 cùng id. |
| B2 guest COD | Thiếu line1 → 400 `INVALID_FIELD`, field `shipping_address.line1`; thiếu city → 400 field `shipping_address.city`. Quote hợp lệ 200 fee=17000; create 201 `AD-261117-M2APMAY3` (`6afbc81a395cce1f860cbe47`), total=1517000. |
| B3 account | COD và tại quầy quote 200/create 201; lần lượt `6afbc81a395cce1f860cbe48` và `6afbc81a395cce1f860cbe49`. List 200 total=2, đúng hai đơn account, không có hai đơn guest. GET guest chỉ bằng cookie khách 404. POST cookie + guest header 400 field `X-Order-Access-Token`. |
| B4 giao nhận/tiền | Admin filtered list 200 thấy bốn đơn thử. COD confirmed/shipping/delivered đều 200 và unpaid; sau PATCH payment paid mới paid. Tại quầy confirmed có `[deliver,cancel,mark_paid]`, không ship; thử shipping 409 `ORDER_STATE_CONFLICT`; delivered 200 vẫn unpaid; mark_paid 200. Account GET lại COD thấy delivered/paid. |
| B4 hoàn hàng | Guest COD đã shipping/paid; cancel 200 → cancelled/awaiting_return/paid; restock 200 → restocked có timestamp. Retry restock/cancel đều 200, không tăng tồn thêm; mark_refunded 200 → refunded. |
| B5 quyền | Customer GET admin 403. Anonymous GET `/orders` 401 `AUTH_REQUIRED`. Anonymous GET **admin** 403 `ADMIN_REQUIRED`, khác 401 ghi trong brief; xem phân định bên dưới. |

Ví dụ quote tại quầy (rút gọn, thông tin liên hệ thử đã ẩn):

```http
POST /api/v1/orders/quote
Origin: http://127.0.0.1:15173
Content-Type: application/json

{"payment_method":"cash_at_counter","shipping_address":{"recipient":"[ẩn]","phone":"[ẩn]"},"items":[{"product_id":"6afbbfd88ed904d9c6f47a15","sku":"DW03-TEST-M","quantity":2}],"note":"HTTP acceptance DW03 test"}
```

```json
{"data":{"subtotal":1000000,"shipping_fee":0,"total":1000000}}
```

Sau POST `/api/v1/orders` cùng payload và quote đã ký, HTTP 201:

```json
{"data":{"id":"6afbc81a395cce1f860cbe46","order_number":"AD-261117-N0D07YTB","status":"pending","payment_status":"unpaid","subtotal":1000000,"shipping_fee":0,"total":1000000}}
```

Ví dụ COD thiếu line1, POST quote → HTTP 400:

```json
{"error":{"code":"INVALID_FIELD","message":"Địa chỉ nhận hàng không được để trống","field":"shipping_address.line1"}}
```

Ví dụ admin hủy khi đang giao, POST `/api/v1/admin/orders/6afbc81a395cce1f860cbe47/cancel`, không body → HTTP 200:

```json
{"data":{"status":"cancelled","payment_status":"paid","inventory_restock":"awaiting_return","allowed_actions":["restock","mark_refunded"]}}
```

Tiếp POST cùng prefix `/restock`, không body → HTTP 200:

```json
{"data":{"status":"cancelled","payment_status":"paid","inventory_restock":"restocked","inventory_restocked_at":"2026-11-17T10:20:59.092+07:00","allowed_actions":["mark_refunded"]}}
```

Đo trực tiếp DB test cho product và variant `DW03-TEST-M` (cả hai cùng số tồn):

| Mốc | Stock |
| --- | ---: |
| Trước đặt | 100 |
| Guest quầy quantity=2, sau replay/lookup | 98 |
| Guest COD quantity=3 | 95 |
| Account COD quantity=2 + quầy quantity=1 | 92 |
| Sau deliver và thu tiền, trước cancel shipping | 92 |
| Sau cancel shipping | 92 |
| Sau restock quantity=3 | 95 |
| Sau retry restock | 95 |
| Sau retry cancel | 95 |

**B5 chưa đạt nguyên văn brief**: anonymous admin thực trả `403 {"error":{"code":"ADMIN_REQUIRED","message":"Chỉ chủ cửa hàng được phép quản trị"}}`. Hợp đồng DW03_ADMIN_ORDER_API.md và guard admin catalog đã quy định 403 cho thiếu/sai phiên từ trước bước này; không coi là bug mới và không sửa code/contract trong bước tài liệu. Reviewer cần chốt lại kỳ vọng 401 trong brief. Không đánh dấu toàn bộ B5 pass.

## Đối chiếu nghiệm thu và phần reviewer bổ sung

| Tiêu chí | Bằng chứng hiện có | Trình duyệt desktop/mobile |
| --- | --- | --- |
| COD cần địa chỉ; tại quầy không cần địa chỉ giao, phí 0 | HTTP B1/B2 | Chờ reviewer |
| Guest không bị buộc đăng ký | HTTP guest create/read thành công không cookie; route/source client | Chờ reviewer kiểm tra điều hướng |
| delivered/confirmed không tự paid; thu tiền riêng | HTTP B4 | Chờ reviewer kiểm tra nút/confirm |
| Quầy không qua shipping | HTTP B4, allowed_actions và 409 | Chờ reviewer |
| Reload/direct API kiểm tra quyền; không đọc đơn khác | HTTP B1/B3/B5; unit đổi id/phiên/server GET | Chờ reviewer reload UI |
| Mất mạng không tạo thêm đơn | HTTP replay và lookup cùng id; unit network + reload cùng key/body | Chưa ngắt mạng trình duyệt thật |
| Chỉ dọn đúng dòng/lượng đã mua | Unit snapshot/cleanup đã pass | Chưa chạy cleanup giỏ qua UI thật |
| Build/lint/tests | Kết quả bên dưới | Không thay thế browser |
| Luồng browser thật | Chưa thực hiện ở lượt Codex này | Reviewer bổ sung URL, viewport, bước và kết quả |

## Giới hạn đã chấp nhận / chưa kiểm chứng

- Đổi thiết bị/xóa storage mất quyền xem đơn guest; chưa có khôi phục bằng mã đơn/phone (Q-03).
- Account đang đăng nhập không xem guest cùng phiên qua client; phải đăng xuất. Admin đọc guest theo quyền admin riêng.
- Attempt hỏng khóa checkout để tránh tạo đơn lặp; storage đầy có thể chặn gửi trước khi lưu attempt.
- Dọn giỏ account có race giữa GET và PUT nếu tab khác cùng sửa. Dọn giỏ gián đoạn chỉ cảnh báo, không tự trừ lại khi không biết đã trừ hay chưa.
- Baseline so sánh quote đầu dùng số của giỏ hiện tại; tổng xác nhận/cuối cùng lấy exact server. Không nhận định unit tests là bằng chứng UI trên thiết bị.
- Q-04 phí COD chưa chốt; 17000 chỉ dùng test. Q-07 không hoàn tồn khi shipping trước khi nhận hàng trả về; hoàn tiền là ghi nhận thủ công.
- Không phát hiện bug mới trong các luồng đã chạy. B5 có lệch brief/contract cần review như trên; browser, modal/busy/layout/mobile và reload thực vẫn chờ reviewer.

## Lệnh đã chạy và kết quả

Từ `server/`, build API, official migrate và helper DB ngoài repo:

```sh
nix develop -c bash -c 'set -e; go build -o /tmp/aodora-dw03-ui-api ./cmd/api; go build -o /tmp/aodora-dw03-ui-migrate ./cmd/migrate; go build -o /tmp/aodora-dw03-ui-db /tmp/aodora-dw03-ui-db.go'
nix develop -c python3 /tmp/aodora-dw03-ui-ctl.py start
nix develop -c python3 /tmp/aodora-dw03-ui-http.py > /tmp/aodora-dw03-ui-http-output.log 2>&1
nix develop -c python3 /tmp/aodora-dw03-ui-ctl.py status
```

Các lệnh exit 0. Start đã in `TARGET host=127.0.0.1:27018 database=aodora_dw03ui_20261117T024542_ca3eda5c; directConnection=true; isolated migration+demo seed`, migration PASS; API `/readyz`, Vite `/` và catalog qua proxy đều ready. Script HTTP assert B1–B4 pass, tách riêng B5 contract mismatch, không che thành pass nguyên văn brief. Script start chỉ chạy một lần lúc dựng; không chạy lại khi review môi trường hiện tại.

Từ `client/`, Node v24.22.2 qua Nix shell repo:

```sh
nix develop ../server -c bash -c 'set -e; node --version; npm run test:orders; npm run test:admin-orders; node --experimental-strip-types --test tests/*.test.mjs; npm run build; npm run lint'
```

Output thật rút gọn, exit 0; đầy đủ `/tmp/aodora-issue21-step4-checks.log`:

```text
v24.22.2
test:orders:       tests 36, pass 36, fail 0
test:admin-orders: tests 13, pass 13, fail 0
tests/*.test.mjs:  tests 67, pass 67, fail 0
vite v8.3.2 building client environment for production...
✓ 125 modules transformed.
dist/index.html                   0.90 kB │ gzip:   0.49 kB
dist/assets/index-DIltqZK2.css   33.89 kB │ gzip:   6.83 kB
dist/assets/index-BjmoUSIv.js   487.10 kB │ gzip: 139.21 kB
✓ built in 911ms
> client@0.0.0 lint
> oxlint
```

## Dừng và xóa sau review

API/Vite được giữ chạy để review. Các lệnh dưới **chưa thực hiện**, chỉ chạy sau khi không còn cần môi trường:

```sh
# Dừng cả process group API và Vite (gồm các tiến trình con của npm).
kill -TERM -- -469082 -469083

# Helper chỉ cho drop đúng DB test trong state, host 27018; không có credential trên argv.
nix develop /home/dontwait/Documents/git/aodora/server -c python3 /tmp/aodora-dw03-ui-ctl.py drop aodora_dw03ui_20261117T024542_ca3eda5c
```

Helper đọc cấu hình test sẵn có tại máy; `/tmp/aodora-dw03-ui-state.json` ghi metadata công khai. File credentials chỉ phục vụ đăng nhập review; sau review có thể xóa cùng các artifact `/tmp/aodora-dw03-ui-*`. Không copy nội dung credentials vào issue/PR/log.
