# Kế hoạch hoàn thiện phần Dontwait

Ngày đối chiếu: 16/11/2026. Phạm vi: DW-01–04 trong [tasks.md](tasks.md), bám [system.md](system.md). Đây là kế hoạch; các phương án ghi “đề xuất” chưa phải quyết định nghiệp vụ đã chốt.

## 1. Hiện trạng đã kiểm tra từ source

| Phần | Hiện trạng | Hướng triển khai |
| --- | --- | --- |
| Catalog | Có API public, admin và mutation tồn riêng trong `server/internal/infrastructure/mongo/admin_catalog_repository.go` | Dùng chung sản phẩm/SKU; phối hợp PD-03 để không mất cập nhật tồn |
| Xác thực | Có session cookie, `AuthGuard.Identity`, `RequireAuth`, `RequireAdmin`; migration `000008_auth_sessions` | Tái sử dụng danh tính/quyền và bảo vệ request thay đổi dữ liệu theo cơ chế auth hiện có |
| Schema | Có `orders`, `tailoring_requests` từ migration cũ; còn khoảng trống guest/tại quầy đã nêu trong specs | Thêm migration mới, không sửa migration đã áp dụng |
| API nghiệp vụ | Chưa có package đơn hàng/may đo trong domain, application, repository và HTTP | Xây module theo cấu trúc server hiện tại; wire qua registration đang dùng |
| Giao diện | `App.tsx` chưa có checkout, trang đơn hoặc quản trị đơn/may đo | Thêm route và màn hình tương ứng; dùng layout/control hiện có |
| Giỏ | `features/catalog/selection.ts` có `CartSelection`, `registerCartAddHandler`; chưa có module giỏ thực | Chốt adapter đọc giỏ/dọn dòng với KR-03; chưa coi tích hợp hoàn thành khi chỉ dùng fixture |
| May đo | `PurchasePanel` truyền `product_id`, `product_slug`; form `MeasurementSection` và `HomeVisitSection` đang toast/reset mà chưa lưu API | Đọc đúng mẫu từ URL, thay submit giả bằng lưu thật; lỗi giữ dữ liệu |
| MongoDB | Compose cấu hình standalone, chưa thấy replica set | Giải quyết nền transaction trước khi nghiệm thu tạo đơn nhiều sản phẩm |

Hiện trạng trên là kiểm tra source, chưa xác minh DB/container đang chạy hoặc chạy thử luồng người dùng. Dòng “Các task đều chưa thực hiện” ở đầu backlog không phản ánh đầy đủ các module PD/KR đã xuất hiện trong checkout; không dùng dòng đó làm bằng chứng trạng thái nghiệm thu của người khác.

## 2. Quy trình issue và PR

1. Tạo issue trên `dont-wait/aodora` theo từng đợt ở mục 4; ghi scope, phụ thuộc, acceptance criteria và cách kiểm tra theo template repository.
2. Gắn link/số issue vào bảng kế hoạch rồi mới triển khai đợt tương ứng. Issue được phép mở khi còn phụ thuộc, nhưng phải ghi rõ điểm chưa chốt.
3. Tạo branch theo issue. Khi thay đổi đã review được, mở PR theo `.github/pull_request_template.md`, dẫn `Closes #...`, mã DW và quyết định Q liên quan.
4. Chỉ đóng issue khi đáp ứng nghiệm thu; ghi riêng phần chưa kiểm chứng bằng DB/browser. Không dùng PR thay cho bước tạo issue trước.

Không tự giao việc hoặc sửa phần của thành viên khác. Các hợp đồng giao tiếp được ghi vào tài liệu API của issue/PR để cùng tích hợp.

## 3. Quyết định và hợp đồng cần có

| Điểm | Phương án / việc cần chốt | Ảnh hưởng |
| --- | --- | --- |
| Q-01 | Source đã có `customers.role`, session và kiểm tra admin; tái sử dụng, xác minh với KR-01 khi ghép | Mọi API quản trị |
| Q-02 | Đề xuất giữ `shipping_address.recipient/phone` cho cả hai cách nhận; `cod` cần `line1/city`, `cash_at_counter` không cần địa chỉ, phí 0; guest thiếu/null `customer_id` | DW-01, checkout |
| Q-03 | Đề xuất cấp secret ngẫu nhiên cho đơn guest, chỉ lưu hash; mã đơn công khai không phải quyền truy cập. Chốt cách trao/lưu/khôi phục secret, phạm vi truy cập đơn và review với KR-04; không đưa secret vào URL/log | API guest, checkout, bàn giao review |
| Q-04 | Cần chủ cửa hàng xác nhận phí/quy tắc phí COD. Dự kiến cấu hình phía server, báo phí trước chốt; chưa cấu hình thì báo không thể tính phí, không tự mặc định COD miễn phí | Quote và tạo đơn COD |
| Q-05 | Chốt adapter đọc giỏ, snapshot dòng mua, nhận thay đổi giá/tồn và xóa đúng dòng đã mua với KR-03; không xóa toàn bộ giỏ sau response chậm | DW-03 |
| Q-07 | Đề xuất giữ cả `stock` và `variants[].stock` đồng bộ; sản phẩm có biến thể dùng SKU, sản phẩm legacy không biến thể dùng SKU cấp sản phẩm. Tồn lệch phải được xử lý rõ trước bán, không tự sửa dữ liệu | DW-02, PD-03 |
| Hủy đang giao | Hủy pending/confirmed hoàn tồn đúng một lần; shipping chỉ nhập lại khi admin xác nhận hàng đã thu hồi. Chốt có cho hủy trước khi hàng về không và metadata cần lưu; không thêm enum trạng thái ngoài specs | DW-02, admin |
| Giao dịch DB | Đề xuất Mongo replica set một node cho local/test để dùng transaction đơn + nhiều sản phẩm. Cần cấu hình auth/keyfile, địa chỉ member dùng được từ host/container và giữ volume hiện có; không chuyển đổi DB đang dùng khi mới lập plan | DW-01–02 |

Các field kỹ thuật mới cho chống lặp, guest access hoặc xác nhận nhập lại tồn cần mô tả schema/index/up/down trước khi code; không coi là field đã tồn tại.

## 4. Các đợt triển khai và issue đã tạo

| Đợt | Issue | Phụ thuộc | Đầu ra |
| --- | --- | --- | --- |
| 1 | [#18](https://github.com/dont-wait/aodora/issues/18) — [DW-01] Schema guest/tại quầy và nền transaction MongoDB | Q-02; phương án giao dịch | Migration, payload DB, môi trường integration test |
| 2 | [#19](https://github.com/dont-wait/aodora/issues/19) — [DW-02] API tạo đơn, tính tiền và cập nhật tồn nhất quán | Đợt 1; Q-04/Q-07; PD-01 | Quote, tạo/đọc đơn, chống lặp, transaction và quyền sở hữu |
| 3 | [#20](https://github.com/dont-wait/aodora/issues/20) — [DW-02/DW-03] API quản trị trạng thái, hủy đơn và thu tiền | Đợt 2; quy tắc thu hồi hàng | State machine, hoàn tồn, thanh toán độc lập |
| 4 | [#21](https://github.com/dont-wait/aodora/issues/21) — [DW-03] Checkout, đơn của khách và quản trị đơn | Đợt 2–3; KR-03; Q-03/Q-05 | Giao diện đầu cuối cho guest/tài khoản và admin |
| 5 | [#22](https://github.com/dont-wait/aodora/issues/22) — [DW-04] Gửi và quản trị may đo, tạo đơn custom | Đợt 2–3 cho custom; KR-01 cho admin | Form/API thật, quản trị yêu cầu, snapshot đơn custom |
| 6 | [#23](https://github.com/dont-wait/aodora/issues/23) — [DW] Nghiệm thu tích hợp và bàn giao quyền review | Đợt 1–5; các module KR/PD liên quan | Kết quả IT, request/response mẫu, tổng hợp AC-01–10 |

### Đợt 1 — Schema và nền giao dịch

- [ ] Kiểm tra validator/index thực tế từ migration 000002/000004 và dữ liệu legacy; tạo migration UUID bằng generator và khai báo phụ thuộc; giữ nguyên các file legacy đã áp dụng.
- [ ] Viết migration up/down cho guest, `cash_at_counter`, địa chỉ có điều kiện; giữ giá trị thanh toán cũ khi đọc DB, chỉ cho hai phương thức mới được yêu cầu khi tạo.
- [ ] Định nghĩa field/index chống gửi lặp và quyền truy cập guest theo hợp đồng đã chốt; index cho danh sách đơn/yêu cầu nếu index hiện có chưa đủ.
- [ ] Chuẩn bị replica set riêng cho integration test; hướng dẫn nâng local có backup/khôi phục và không đổi tên volume hiện có.
- [ ] Kiểm tra up trên DB có dữ liệu cũ; fixture guest/COD/tại quầy hợp lệ và không hợp lệ. Down phải nêu dữ liệu cần chuyển trước và từ chối an toàn khi không tương thích, không xóa đơn mới để rollback.

### Đợt 2 — Tạo và đọc đơn

- [ ] Thêm `domain/order`, `application/order`, Mongo repository, HTTP handler; giữ domain/application độc lập HTTP/Mongo.
- [ ] Tài liệu API dự kiến: quote, tạo đơn, danh sách/chi tiết đơn của mình và đọc đơn guest có secret. Tên route/DTO chốt trong issue trước tích hợp.
- [ ] Quote đọc catalog thật và cấu hình phí. Request chỉ truyền lựa chọn/lượng và thông tin nhận; server tự tính tiền, kiểm tra số nguyên dương, SKU thuộc sản phẩm, trạng thái bán và tồn. Quy tắc kiểu tiền/giới hạn phải khớp BSON hiện có, kiểm tra tràn số.
- [ ] Chốt hợp đồng xác nhận lại quote: giá/phí/tồn thay đổi phải báo conflict để khách kiểm tra, không âm thầm tạo đơn với tổng mới.
- [ ] Trong một transaction: gộp dòng trùng SKU, kiểm tra/trừ tồn có điều kiện, lưu snapshot dòng đơn, mã đơn unique và `pending/unpaid`. Một dòng thất bại rollback cả đơn.
- [ ] Chống tạo lặp bằng request key + fingerprint + unique index, gắn với đúng tài khoản hoặc phạm vi guest có secret; cùng key/cùng payload trả kết quả cũ, khác payload báo conflict. Retry transaction/commit không sinh đơn hoặc trừ tồn lần hai.
- [ ] Danh tính tài khoản lấy từ auth; session không hợp lệ không âm thầm trở thành guest. Khách chỉ đọc đơn của mình; guest phải có quyền truy cập đã chốt; response/log không lộ hash/secret hoặc dữ liệu đơn khác.
- [ ] Kiểm chứng DB thật: tranh mua chiếc cuối, đơn nhiều SKU lỗi giữa chừng, gửi lặp đồng thời, timeout sau commit, snapshot không đổi khi sửa catalog, mutation tồn đồng thời với admin.

### Đợt 3 — Xử lý đơn và tiền

- [ ] API admin list/detail có phân trang, lọc trạng thái đơn/thanh toán và sắp xếp ổn định.
- [ ] Chuyển trạng thái bằng điều kiện trạng thái hiện tại: COD `pending → confirmed → shipping → delivered`; tại quầy `pending → confirmed → delivered`; từ chối bước ngược/trái luồng.
- [ ] Hủy và hoàn tồn trong transaction, lưu dấu đã hoàn để retry/concurrency chỉ hoàn một lần; shipping tuân thủ quy tắc thu hồi đã chốt. Custom không hoàn tồn hàng sẵn.
- [ ] Mutation thanh toán riêng: `unpaid → paid`, `paid → refunded` sau xác nhận admin đã thu/hoàn đủ; cập nhật giao hàng/hủy không tự đổi trạng thái tiền. Không mở rộng thu từng phần hoặc hoàn tự động.
- [ ] Kiểm chứng request trực tiếp bằng guest/khách/admin, hủy lặp, hủy đua với giao hàng và thay đổi tiền đồng thời.

### Đợt 4 — Checkout và màn hình đơn

- [ ] Thêm checkout, kết quả đặt đơn, đơn của tôi/chi tiết đơn và trang admin đơn; dùng auth provider/guard hiện có.
- [ ] Checkout đọc giỏ thật qua hợp đồng KR-03, dùng địa chỉ tài khoản khi có, cho guest nhập trực tiếp; địa chỉ bắt buộc theo cách nhận.
- [ ] Hiển thị quote, tiền hàng/phí/tổng; khi thay đổi giá/tồn cần khách xác nhận lại. Đang gửi khóa thao tác; giữ request key qua lỗi mạng/reload và tra kết quả trước khi tạo thao tác mới.
- [ ] Chỉ hiển thị thành công sau khi server lưu; thất bại giữ form/giỏ. Thành công chỉ dọn đúng lượng/dòng đã mua theo snapshot, không làm mất thay đổi giỏ mới.
- [ ] Admin có nút trạng thái hợp lệ và nút thu/hoàn tiền riêng; lỗi conflict tải lại dữ liệu để thao tác tiếp.
- [ ] Ghép secret guest theo Q-03 và kiểm tra tải lại trang, truy cập đơn khác, phiên hết hạn; không chỉ kiểm tra ẩn nút.
- [ ] Chạy luồng trình duyệt trên desktop/mobile với API/DB thật, COD và tại quầy, guest và tài khoản.

### Đợt 5 — May đo

Phần nhận/quản lý yêu cầu có thể triển khai sớm sau hợp đồng auth; phần tạo đơn custom chỉ ghép sau lõi đơn hàng.

- [ ] Thêm domain/application/repository/handler may đo: public submit, admin list/detail/update, admin tạo đơn custom từ yêu cầu.
- [ ] Bắt buộc tên/điện thoại, cho thiếu mẫu/số đo; giữ `body` và bảy field `_cm`, không biến phần thiếu thành 0. Liên kết đúng tài khoản nếu đã xác thực.
- [ ] Đọc `product_id/product_slug` từ PurchasePanel; server xác minh mẫu tham chiếu và tránh ID/slug mâu thuẫn. Form hướng dẫn không có mẫu vẫn gửi được.
- [ ] Thay submit giả ở hai form; map lịch hẹn/ghi chú vào field schema phù hợp. Các dữ liệu UI chưa có field phải nêu mapping, không tự tạo schema ngoài phạm vi.
- [ ] Admin lọc/sửa lịch, ghi chú, số đo và status theo enum có sẵn; `measurement_confirmed` cần thao tác xác nhận chủ động.
- [ ] Tạo đơn `custom` bằng nghiệp vụ dùng chung: admin xác nhận mẫu/giá/số đo, liên kết yêu cầu, snapshot số đo, không trừ tồn áo sẵn; chống tạo trùng do retry.
- [ ] Kiểm chứng lưu lỗi không reset, reload thấy dữ liệu, sửa yêu cầu không ghi đè đơn cũ; gửi yêu cầu không bị diễn đạt thành đặt đơn thành công.

### Đợt 6 — Nghiệm thu và bàn giao

- [ ] Tài liệu API có payload COD/tại quầy/custom, response, quyền, lỗi validation/stock/conflict/not-found/auth và cách retry; ghi field required/optional/null.
- [ ] Bàn giao KR-04 cách xác minh quyền review ở server: chủ đơn hoặc guest có quyền, đơn `delivered`, sản phẩm thuộc `items`; không tin ID/cờ verified do client tự khai. Schema/index review do KR-04 phụ trách phối hợp Q-03.
- [ ] Ghi bằng chứng IT-02/03/04/05/06/08/10/11/12 theo trách nhiệm Dontwait; phối hợp KR cho IT-07 và tổng hợp AC-01–10 mà không tự đánh dấu đạt phần chưa ghép.
- [ ] `server/`: `make test`; kiểm tra integration Mongo thật có transaction riêng, không coi mock là bằng chứng không oversell/rollback.
- [ ] `client/`: `npm run build`, `npm run lint`, các test hiện có liên quan; test hành vi checkout/retry khi triển khai, rồi kiểm tra browser thật.
- [ ] Ghi phiên bản migration, fixture, thao tác, kết quả DB/UI và giới hạn chưa kiểm chứng trong PR/issue. Chỉ đóng task khi đạt điều kiện hoàn thành mục 9 của backlog.

## 5. Bước bắt đầu

Đã tạo issue #18–23 trước khi triển khai hoặc mở PR. Bắt đầu đợt 1 theo [#18](https://github.com/dont-wait/aodora/issues/18) bằng hợp đồng field COD/tại quầy, rà validator/index và thiết kế transaction/test DB. Song song ghi rõ phần chờ Q-03/04/05/07 trong issue; các quyết định đó không cản việc chuẩn bị schema và test fixture độc lập.

## 6. Tiến độ đợt 1 — 16/11/2026

Đã triển khai migration `order_checkout` (UUID), preflight rollback đọc động version/collection/validator từ migration và Compose replica set riêng; hướng dẫn/mapping ở [DW01_ORDER_SCHEMA.md](server/docs/DW01_ORDER_SCHEMA.md). Chưa merge hoặc đánh dấu hoàn thành task; chưa áp dụng migration vào DB đang dùng.

- `nix develop --command make test`: đạt toàn bộ Go test với race detector.
- Integration test trên MongoDB 8 replica set thật: đạt kiểm tra dữ liệu cũ, guest/COD/tại quầy, rollback/reapply và transaction commit/abort. Fixture collection/version khác cũng đạt, xác nhận runner không gắn riêng với orders hoặc một version cố định.
- Lint package migration gồm integration test: 0 issues. Lint toàn server còn 7 lỗi baseline ở file không thay đổi (errcheck/staticcheck); lỗi mới đã xử lý.
- Chưa có API đơn hàng, chống lặp hoặc quyền guest/review. Field/index kỹ thuật cho các phần này theo hợp đồng DW-02/Q-03 ở đợt tiếp theo; không tự chốt schema khi mới làm nền dữ liệu.

Đã ghép main chứa runner UUID sau khi PR #26 merge (8af41d7). Migration checkout chuyển sang thư mục UUID, phụ thuộc auth legacy theo metadata; test DW đọc đúng ledger UUID và các migration cha. Phần DW-01 vẫn theo PR #24.
