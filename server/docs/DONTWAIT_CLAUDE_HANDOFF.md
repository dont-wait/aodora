# Bàn giao cho Claude — phần Dontwait / Aodora

Ngày bàn giao: 16/11/2026. Bắt đầu triển khai từ **DW-02, issue #19**. Đây là spec thực hiện; không phải báo cáo rằng API đơn hàng đã có.

## 1. Mục tiêu và thứ tự nguồn

Hoàn thiện phần Dontwait: khách vãng lai/tài khoản đặt áo, server tính đúng tiền và tồn, chủ cửa hàng xử lý đơn/thu tiền, khách gửi yêu cầu may đo và chủ cửa hàng tạo đơn custom. Làm theo từng issue/PR, bắt đầu bằng API tạo/đọc đơn; không gom toàn bộ roadmap vào một PR.

Đọc theo thứ tự:

1. [system.md](../../system.md): yêu cầu nghiệp vụ và AC-01–10.
2. [tasks.md](../../tasks.md): phạm vi DW-01–04, trách nhiệm và IT-01–12.
3. [DW_ORDER_CHECKOUT_CONTRACT.md](DW_ORDER_CHECKOUT_CONTRACT.md): **hợp đồng field/header/route/error phải giữ nguyên**.
4. [DW01_ORDER_SCHEMA.md](DW01_ORDER_SCHEMA.md), [database-schema.md](database-schema.md): schema đã triển khai.
5. [UUID_MIGRATIONS.md](UUID_MIGRATIONS.md), [DW_BACKUP_RESTORE.md](DW_BACKUP_RESTORE.md): migration, tiếp nhận lịch sử và khôi phục.
6. [dontwait-plan.md](../../dontwait-plan.md): thứ tự triển khai; phần hiện trạng ban đầu có thể cũ, phải đối chiếu source.

Không đổi yêu cầu trong system.md để làm implementation trông như đã đạt. Nếu nhận ra hợp đồng mâu thuẫn, ghi rõ điểm đó và sửa trong phạm vi có bằng chứng; không tự đổi tên field/enum/null semantics.

## 2. Trạng thái bàn giao đã xác minh

| Phần | Trạng thái |
| --- | --- |
| Runner UUID | PR [#26](https://github.com/dont-wait/aodora/pull/26) đã merge main, commit `8af41d7` |
| DW-01/schema và hợp đồng | PR [#24](https://github.com/dont-wait/aodora/pull/24) đã merge main, commit `cea02c7`; commit nội dung cuối `7fa00ea` |
| Catalog/admin catalog | Có API, dữ liệu SKU/giá/tồn, mutation tồn riêng |
| Auth/customer | Có session cookie, quyền admin/customer, hồ sơ/địa chỉ/số đo/yêu thích |
| Quote/tạo/đọc/tra attempt đơn hàng | **Chưa có endpoint hoặc module order trong router/composition root** |
| Xử lý trạng thái/thu tiền | Chưa triển khai |
| Checkout UI/quản trị đơn | Chưa triển khai trong phần Dontwait |
| Yêu cầu may đo và đơn custom | Chưa triển khai API nghiệp vụ Dontwait |
| DB đang dùng | Agent chưa áp dụng migration mới vào DB vận hành; chỉ kiểm thử database tạm |

Schema đã có hai migration, không sửa lại chúng:

- `server/migrations/ae74e773-1d52-45da-bd8d-07864eea7737_order_checkout/`: guest customer_id thiếu/null, cash_at_counter phí 0, địa chỉ theo phương thức nhận; giữ enum cũ cho legacy.
- `server/migrations/c8689b2b-f25a-49ee-9959-cd755fa0c999_order_checkout_access/`: checkout metadata, binding scope/credential và unique partial index chống gửi lặp.

Đã qua Go race tests, integration MongoDB thật, lint package migration, Sourcery và GitGuardian trên bản merge. Backup/restore đã kiểm chứng bằng hai DB tạm: count 11 collection, 3 đơn, validator/index, UUID head và guest hash khớp. Đây là bằng chứng nền dữ liệu, **chưa chứng minh API tạo đơn hoặc chống oversell**.

## 3. Bắt đầu phiên làm việc

- Đọc AGENTS.md/hướng dẫn áp dụng nếu có; kiểm tra git status trước khi đổi branch.
- Fetch main mới nhất và tạo branch riêng từ main cho issue #19, ví dụ `feature/dw-02-orders`. Không tiếp tục commit lên branch DW-01 đã merge.
- Có thư mục `server/docker/` chưa được theo dõi từ công việc Navicat khác; không add/xóa nó hoặc đưa vào PR đơn hàng.
- Issue đã được tạo trước. Không tạo issue trùng và không mở PR trước khi có issue tương ứng.
- Không áp dụng migration hoặc đổi topology/volume của DB đang dùng chỉ để chạy test. Chạy test trên stack `aodora-transactions` và DB tên ngẫu nhiên.

## 4. Phạm vi PR đầu tiên — issue #19

Issue: [DW-02 / #19](https://github.com/dont-wait/aodora/issues/19).

Hoàn thiện API **may sẵn**: quote, tạo đơn, đọc đơn và tra kết quả attempt. API danh sách tài khoản dùng pagination ổn định. Trả dữ liệu thật từ MongoDB; không mock endpoint hoặc báo thành công khi chưa commit.

Trong PR này không làm UI checkout, state machine giao/hủy/thu tiền, review hoặc form may đo. Giữ khả năng dùng chung nghiệp vụ khi triển khai custom ở #22, nhưng không cho public client tự gửi custom để bỏ qua trừ tồn hoặc chọn giá tùy ý.

### 4.1. API bắt buộc

| Route | Quyền và kết quả |
| --- | --- |
| `POST /api/v1/orders/quote` | Guest/customer; kiểm tra lựa chọn, lấy giá/phí và trả quote token + expires_at; chưa lưu đơn/trừ tồn |
| `POST /api/v1/orders` | Guest/customer; bắt buộc Idempotency-Key, guest thêm X-Order-Access-Token; 201 khi mới tạo, 200 khi replay cùng thao tác |
| `GET /api/v1/orders/{id}` | Chủ tài khoản/admin hoặc guest credential đúng đơn; sai quyền trả 404 chung |
| `GET /api/v1/orders` | Chỉ danh sách đơn của tài khoản hiện tại; data/total/page/page_size |
| `GET /api/v1/orders/attempts/{request_id}` | Tra theo scope hiện tại + key, không cần quote còn hạn; không tạo đơn |

Payload/response/error mẫu và required/optional đã ở hợp đồng checkout. Handler phải dùng DTO allowlist; từ chối JSON sai, unknown fields, quantity không phải integer dương và field nội bộ do client gửi. Không nhận customer_id, scope, giá, total, status, payment_status hoặc hash từ trình duyệt.

Mọi đơn mới lưu `pending/unpaid`, order_number không trùng và snapshot product_name/SKU/màu/cỡ/quantity/unit_price. Trạng thái giao hàng và thanh toán giữ riêng. Không thêm phương thức thanh toán ngoài `cod`, `cash_at_counter` cho đặt mới.

### 4.2. Danh tính và credential

- Dùng auth hiện có; customer_id lấy từ session, không từ body.
- Phân biệt không có cookie và có cookie nhưng session sai/hết hạn. Trường hợp thứ hai trả 401, không âm thầm đặt guest. `AuthGuard.Identity` hiện trả bool nên handler/use case phải xử lý khác biệt này rõ ràng.
- Guest token canonical base64url không padding, decode chính xác 32 byte. Hash SHA-256 trên **bytes đã decode**, không trên chuỗi base64url. Dùng [fixture](fixtures/DW_CHECKOUT_ACCESS.json) để đối chiếu; bytes test phải tái tạo token lúc chạy, không commit token base64 tĩnh gây false positive scanner.
- Account có scope `customer:<ObjectId hex>` và không có guest hash; guest có scope `guest:<SHA256>` và guest_access_hash tương ứng. Đơn guest customer_id thiếu/null, không tạo khách giả.
- Từ chối trộn session account và guest token trong thao tác tạo. Không đổi scope của attempt đang chưa biết kết quả khi người dùng đăng nhập/đổi tài khoản.
- Response dùng DTO riêng, không serialize nguyên BSON; không lộ token/hash/checkout metadata/session/password hoặc đơn người khác. Response riêng tư có Cache-Control: no-store.
- CORS/origin policy theo server hiện có, cho hai header checkout; có OPTIONS phù hợp. Không log body/header credential và không đưa credential vào URL.

### 4.3. Quote và tiền

- Kiểm tra product/SKU, điều kiện bán, lượng và địa chỉ từ nguồn DB/config. Giữ cùng điều kiện công khai của catalog, kể cả category/ancestor bị tắt.
- Giá/tiền BSON đang cho `int/long/decimal`. Không âm thầm đổi thành float64 hoặc làm tròn decimal về integer. Chọn representation exact, kiểm tra overflow và test giá decimal hiện được schema cho phép.
- `subtotal = Σ(quantity × unit_price)`, `total = subtotal + shipping_fee`; tại quầy luôn phí 0, không cần line1/city.
- COD dùng nguồn phí phía server. **Q-04 chưa có mức phí/quy tắc được chủ cửa hàng xác nhận.** Không lấy 30.000 trong ví dụ làm default. Khi chưa cấu hình, trả SHIPPING_UNAVAILABLE; vẫn hoàn thiện/test tại quầy và COD bằng cấu hình test rõ ràng.
- Quote token phải được xác thực ở server, ràng buộc đúng lựa chọn/quantity/phương thức/thông tin ảnh hưởng phí và giá/phí được xác nhận. Có expires_at; chống client sửa nội dung.
- Có thể dùng HMAC từ key cấu hình riêng cho quote; không hardcode key hoặc dùng token chỉ base64 mà không xác thực. Tên config/key/TTL là chi tiết triển khai phải ghi rõ trong PR/.env.example; không mô tả như config đã tồn tại.
- Tạo đơn phải so lại dữ liệu hiện tại với quote; thay giá/phí hoặc hết hạn trả QUOTE_CHANGED/QUOTE_EXPIRED để khách xác nhận lại. Không âm thầm tạo với tổng khác.

### 4.4. Chống gửi lặp và transaction

Luồng bắt buộc:

1. Parse/normalize và xác thực principal/guest credential; tính scope, request_id và canonical request_hash đúng hợp đồng.
2. Tìm đơn `(checkout.scope, checkout.request_id)` trước khi kiểm tra lại giá/tồn/quote có thể đã đổi. Cùng hash trả đơn cũ; khác hash trả IDEMPOTENCY_CONFLICT.
3. Nếu chưa có, xác thực quote và tạo đơn/trừ tồn trong một session transaction MongoDB. Mọi repository write thuộc thao tác phải nhận cùng transaction context.
4. Gộp dòng trùng product_id/SKU, kiểm tra tổng quantity không overflow; cập nhật tồn có điều kiện, không read rồi set tồn dựa trên bản chụp cũ.
5. Sản phẩm có variants: trừ đúng variants[].stock và stock tổng trong cùng mutation; không ghi đè cả mảng variants. Nhiều SKU cùng một product vẫn phải kiểm tra tổng đủ tồn.
6. Legacy không variants dùng SKU cấp product và stock tổng. Dữ liệu tổng/biến thể lệch phải trả lỗi rõ và ghi cách reconciliation; không tự sửa tồn trong thao tác bán hàng. Chốt quy tắc dùng chung với PD-03/Q-07 trong PR.
7. Insert đơn với checkout marker và unique index đã có. Một dòng sản phẩm/insert/commit lỗi không để lại trừ tồn hoặc đơn thành công một phần.
8. Hai request cùng key đua nhau: xử lý unique conflict bằng đọc lại đơn đã commit và so hash; không trả raw database error hoặc trừ thêm lần nữa.
9. Commit timeout/unknown outcome: dùng driver transaction/commit semantics và tra lại cùng scope/key. Không đổi key hoặc credential để tạo lại; không báo thành công khi chưa biết kết quả. 404 tra attempt khi request cũ đang in-flight không chứng minh thất bại.

Không gọi API admin adjust-stock bên ngoài transaction như một bước đặt đơn. Admin inventory journal có cơ chế riêng, không thay thế idempotency order. `products.version` hiện dùng cho nội dung; inventory mutation không bump version này nên không được coi nó là version tồn kho.

Nếu MongoDB standalone không hỗ trợ transaction, fail rõ với ORDER_UNAVAILABLE và không ghi một phần; không fallback sang nhiều write rời. Không tự chuyển DB đang dùng sang replica set.

### 4.5. Tạo/đọc đơn và extension cho bước tiếp theo

- order_number là mã nhận diện, không phải credential. Nếu cần thêm unique index/schema cho mã đơn, tạo migration UUID mới, partial phù hợp legacy và kiểm tra dữ liệu cũ; không sửa migration đã áp dụng.
- Read/list phải kiểm tra quyền ở server; ID hợp lệ nhưng không sở hữu trả 404. Danh sách sort có tie-breaker `_id`; total đếm cùng filter trước pagination.
- Snapshot đơn độc lập với sửa catalog/profile về sau. Không trả field kỹ thuật quyền guest trong public DTO.
- Chuẩn bị đầu ra nội bộ để KR-04 kiểm tra owner/credential + delivered + product thuộc items; không nhận verified_purchase do client tự khai. Schema/index reviews thuộc KR-04, không tự sửa trong #19.

## 5. Các điểm ghép source cụ thể

| Vị trí | Hướng dùng |
| --- | --- |
| `server/internal/domain/catalog/model.go` | Đối chiếu product/SKU và kiểu tiền đọc; không dùng public availability thay cho kiểm tra transaction |
| `server/internal/infrastructure/mongo/admin_catalog_repository.go` | Đọc UpdateProduct/AdjustInventory để giữ cùng quy tắc stock; tránh mất update khi admin đang sửa áo |
| `server/internal/interfaces/http/auth_handler.go` | AuthGuard.Identity/RequireAdmin, origin policy và error envelope |
| `server/internal/interfaces/http/customer_handler.go` | Mẫu DTO/validation/error hiện có |
| `server/internal/interfaces/http/router.go` | Dùng registration point RouterOption hiện có, không viết router/options abstraction mới |
| `server/cmd/api/main.go` | Wire service/repository/handler order vào composition root |
| `server/internal/config/config.go`, `server/.env.example` | Thêm config có validate và documented defaults, không tạo default business fee |
| `server/internal/infrastructure/database/mongo.go` | Mongo client sẵn có để mở transaction; không tạo client riêng cho mỗi dòng hàng |
| `client/src/features/catalog/selection.ts` | Bàn giao product_id/SKU/quantity; giỏ thực và dọn giỏ phối hợp KR-03 |

Tổ chức package theo architecture hiện có: domain/order, application/order, Mongo repository và HTTP handler. Domain/application không phụ thuộc HTTP hoặc MongoDB. Reuse control/layout/auth ở các bước UI; không mở rộng thành refactor toàn repo.

## 6. Kiểm thử bắt buộc cho #19

Unit/HTTP tests cho parse, normalization/canonical hash, money/quote, guest/account auth và error mapping. Integration tests dùng MongoDB thật cho transaction/index/concurrency; không coi mock hoặc build thành công là bằng chứng không oversell.

| Ca | Kết quả bắt buộc |
| --- | --- |
| Guest tại quầy chỉ tên/điện thoại | Một đơn pending/unpaid, phí 0, không có khách giả |
| Guest/account COD | Đủ địa chỉ; phí từ cấu hình test, không tin client |
| Bank/card/e_wallet, gửi customer_id/price/status | Bị từ chối ở API tạo mới |
| Cookie sai và guest token sai/không canonical | Không hạ quyền hoặc nhận nhầm credential |
| Cùng payload khác thứ tự field/dòng trùng SKU | Canonicalization đúng fixture và quantity tổng đúng |
| Cùng scope/key, cùng payload gửi tuần tự/đồng thời | Một đơn, trừ tồn một lần, replay cùng ID |
| Cùng scope/key khác payload | 409 IDEMPOTENCY_CONFLICT |
| Cùng key ở hai scope khác nhau | Không đọc/trả đơn của nhau; hoạt động độc lập |
| SKU còn 1, hai khách đặt đồng thời | Chỉ một đơn thành công; tổng/SKU không âm |
| Một đơn nhiều product/SKU, dòng cuối lỗi | Không có đơn hoặc trừ tồn một phần |
| Giá decimal, quantity lớn hoặc tiền vượt giới hạn | Tính exact; không lỗi im lặng/overflow |
| Sửa giá/tồn sau quote | Conflict đúng; replay đơn cũ vẫn trả kết quả dù quote đã hết hạn |
| Mất response sau commit/commit outcome chưa rõ | Tra cùng attempt, không tạo thêm đơn |
| Admin sửa metadata/adjust inventory đồng thời mua | Không phục hồi tồn cũ hoặc mất cập nhật |
| Thay catalog/profile sau đặt | Snapshot đơn giữ nguyên |
| Account B đổi ID đơn A / guest token khác | 404, không lộ thông tin riêng |
| MongoDB không hỗ trợ transaction | Lỗi rõ; không ghi một phần |
| Log/response và CORS | Không lộ credential/hash; đúng origin và hai header mới |

Từ `server/` dùng `nix develop` theo repo:

```bash
make transactions-up
# Đặt MONGO_TEST_URI riêng trong môi trường, không commit URI có credentials.
make test
make test-integration
```

Nếu thêm package test ngoài migration, cấu hình lệnh/target integration để chạy cả chúng: target test-integration hiện chỉ chạy package migration. Không để test mới bị bỏ qua trong báo cáo pass. Chạy lint cho phần thay đổi; phân biệt lỗi mới với baseline, không sửa cảnh báo toàn repo ngoài scope.

Tests phải tạo/dọn database tạm của riêng mình. Giữ nguyên project/volume của stack khác và `server_mongo_data`. Nếu cần thay schema/index, dùng `make migrate-create name=...` với UUID/phụ thuộc; kiểm tra up/down/reapply. Backup dữ liệu vận hành không thuộc test fixture.

## 7. Roadmap sau PR #19

| Issue | Phần tiếp theo | Điều kiện quan trọng |
| --- | --- | --- |
| [#20](https://github.com/dont-wait/aodora/issues/20) | Admin list/detail, chuyển trạng thái, hủy/hoàn tồn, thu/hoàn tiền | COD pending→confirmed→shipping→delivered; tại quầy bỏ shipping; hủy lặp hoàn một lần; thu tiền độc lập giao áo |
| [#21](https://github.com/dont-wait/aodora/issues/21) | Checkout, đơn khách, quản trị đơn | Giỏ KR-03 thật; giữ form/attempt khi lỗi; thành công chỉ dọn đúng dòng/lượng đã mua; dùng UI hiện có |
| [#22](https://github.com/dont-wait/aodora/issues/22) | Gửi/quản trị may đo, tạo đơn custom | Guest gửi tên/phone; body + bảy số đo cm theo schema; xác nhận số đo chủ động; admin chốt giá; custom không trừ tồn áo sẵn |
| [#23](https://github.com/dont-wait/aodora/issues/23) | Nghiệm thu xuyên suốt, bàn giao review | Ghi kết quả API/DB/browser và AC/IT thực tế, không chỉ screenshot toast/build |

Q-04 phí COD vẫn cần chủ cửa hàng chốt. Q-07 hủy khi shipping: không cộng lại hàng chưa thu hồi; cần chốt xác nhận nhập lại trước #20. Q-05 merge/dọn giỏ phối hợp MyNameIsKry. Giữ hai nhóm quyền customer/admin; không thêm payment gateway, vai trò nhân viên hoặc trạng thái review mới.

## 8. Quy trình PR và đầu ra Claude phải báo

- Issue trước → branch từ main → triển khai/test → PR liên kết issue. #19–23 đã có; không tạo bản trùng.
- PR theo template `.github/pull_request_template.md`, base đúng main, title theo scope cuối cùng. Không merge hoặc deploy chỉ vì thấy phần nền đã merge; làm theo quyền user cấp cho phiên Claude hiện tại.
- Đọc toàn bộ comment bot sau khi mở/push PR. Phân loại đúng/sai/có điều kiện theo source; sửa bug thật, không đổi spec để né review. Không chỉ đọc Reviewer's Guide rồi nói đã review.
- Không báo pass toàn bộ nếu bỏ qua DB/browser; ghi rõ test nào chạy, dữ liệu nào dùng và giới hạn chưa kiểm chứng. Không kiểm thử trực tiếp trên DB vận hành.
- Khi báo hoàn thành, nêu: hành vi đã chạy được, API/payload, config cần đặt, migration mới nếu có, test thực tế, commit/PR và phần còn lại. Không đánh dấu frontend/custom/state machine đã xong nếu mới hoàn thành #19.

## Prompt khởi động có thể gửi nguyên cho Claude

> Đọc server/docs/DONTWAIT_CLAUDE_HANDOFF.md trong repo Aodora và các tài liệu nguồn nó liên kết. Tiếp tục phần Dontwait từ issue #19: triển khai API quote/tạo/đọc/tra attempt đơn may sẵn theo contract đã chốt, bảo toàn guest access, idempotency, tiền exact và transaction tồn kho. Bắt đầu bằng kiểm tra checkout/main và source hiện tại; tạo branch riêng từ main. Hoàn thiện code và test MongoDB thật, cập nhật tài liệu rồi mở PR liên kết #19 cho bot review. Không tự đặt phí COD, sửa migration đã áp dụng hoặc chạy thay đổi vào DB vận hành. Báo rõ phần đã chứng minh chạy thật và phần còn lại theo roadmap.
