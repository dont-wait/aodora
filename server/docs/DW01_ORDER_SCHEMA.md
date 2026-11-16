# DW-01 — Đơn khách vãng lai, nhận tại quầy và nền transaction

Issue: [#18](https://github.com/dont-wait/aodora/issues/18). Migration mới: `order_checkout` (UUID); không sửa các migration trước đó. Tài liệu này mô tả hợp đồng DB, chưa phải API đặt hàng (DW-02).

## Field mapping và Q-02

Giữ tên field hiện có, dùng `payment_method` phân biệt giao tận nơi và nhận trực tiếp trong phạm vi hai phương thức đặt mới đã chốt tại specs.

| Field | COD (`cod`) | Nhận tại quầy (`cash_at_counter`) |
| --- | --- | --- |
| `customer_id` | ObjectId từ session nếu đăng nhập; thiếu/null khi guest | Tương tự COD |
| `shipping_address.recipient` | Bắt buộc string | Bắt buộc string |
| `shipping_address.phone` | Bắt buộc string | Bắt buộc string |
| `shipping_address.line1`, `.city` | Bắt buộc string | Có thể thiếu; nếu gửi vẫn phải string |
| `shipping_address.ward`, `.district` | Tùy chọn string/null theo schema cũ | Tương tự COD |
| `shipping_fee` | Số không âm; mức phí do Q-04 và server tính khi làm DW-02 | Bắt buộc bằng 0 |
| `status` / `payment_status` | Giữ enum cũ; DW-02 tạo `pending/unpaid` | Tương tự COD |

DB giữ các kiểu tiền `int/long/decimal`, cấu trúc dòng đơn, `fulfillment_mode`, liên kết may đo và số đo từ migration 000004. Không tạo tài khoản hoặc địa chỉ giả. Validator không xác minh customer/product có tồn tại; kiểm tra tham chiếu, chuỗi trắng, quyền, tổng tiền và chuyển trạng thái thuộc application DW-02. Recipient/phone/address giữ constraint string cũ để không làm dữ liệu legacy mất tương thích; API phải trim và từ chối nội dung rỗng khi triển khai.

`bank_transfer`, `card`, `e_wallet` vẫn hợp lệ cho dữ liệu cũ và vẫn cần đủ địa chỉ như trước. API tạo mới sẽ chỉ nhận `cod` hoặc `cash_at_counter`; chưa có endpoint mới trong DW-01. Không đổi trạng thái hay dữ liệu các đơn cũ trong migration.

## Hai document mẫu

Đây là MongoDB Extended JSON để lưu DB, không phải payload nhận từ trình duyệt. Ví dụ phí COD chỉ minh họa hình dạng document; không phải quyết định mức phí vận hành. API DW-02 phải tính lại tên/giá/tổng từ nguồn có thẩm quyền, lấy danh tính từ session và không tin các field đó do client gửi.

COD, khách có tài khoản:

```json
{
  "customer_id": {"$oid": "690000000000000000000001"},
  "items": [{
    "product_id": {"$oid": "680000000000000000000001"},
    "product_name": "Áo dài lụa", "sku": "LUA-DO-M",
    "size": "M", "color": "Đỏ", "quantity": 1,
    "unit_price": 500000, "fulfillment_mode": "ready"
  }],
  "subtotal": 500000, "shipping_fee": 30000, "total": 530000,
  "status": "pending", "payment_method": "cod", "payment_status": "unpaid",
  "shipping_address": {
    "recipient": "Nguyễn An", "phone": "0900000001",
    "line1": "12 Nguyễn Trãi", "city": "TP. Hồ Chí Minh"
  },
  "created_at": {"$date": "2026-11-16T04:00:00Z"},
  "updated_at": {"$date": "2026-11-16T04:00:00Z"}
}
```

Khách vãng lai nhận tại quầy, không có `customer_id` hoặc địa chỉ giả:

```json
{
  "items": [{
    "product_id": {"$oid": "680000000000000000000001"},
    "product_name": "Áo dài lụa", "sku": "LUA-DO-M",
    "size": "M", "color": "Đỏ", "quantity": 1,
    "unit_price": 500000, "fulfillment_mode": "ready"
  }],
  "subtotal": 500000, "shipping_fee": 0, "total": 500000,
  "status": "pending", "payment_method": "cash_at_counter", "payment_status": "unpaid",
  "shipping_address": {"recipient": "Nguyễn An", "phone": "0900000001"},
  "created_at": {"$date": "2026-11-16T04:00:00Z"},
  "updated_at": {"$date": "2026-11-16T04:00:00Z"}
}
```

Không hợp lệ ở DB: COD thiếu `line1/city`, liên hệ thiếu tên/điện thoại, `customer_id` dạng string, tại quầy có phí khác 0, lượng bằng 0, số đo âm hoặc phương thức thanh toán ngoài enum.

## MongoDB transaction riêng cho local/test

Từ thư mục `server/`, trong Nix shell (`nix develop`):

```bash
make transactions-up
export MONGO_TEST_URI='mongodb://aodora_test:aodora_test@127.0.0.1:27018/?authSource=admin&replicaSet=aodora-rs&directConnection=true'
make test-integration
```

Compose `docker-compose.transactions.yml` dùng project `aodora-transactions`, volume `aodora-transactions_mongo_data` và `aodora-transactions_mongo_config`; không mount `server_mongo_data`. Có auth và keyfile được tạo một lần, giữ quyền file 400. Init replica set lặp được, chờ primary rồi mới kết thúc; lỗi khởi tạo không bị bỏ qua. Cổng chỉ bind loopback. Đổi cổng/user/password bằng `MONGO_TEST_PORT`, `MONGO_TEST_USERNAME`, `MONGO_TEST_PASSWORD` trước lần khởi tạo đầu; thay env không đổi user đã lưu trong volume cũ.

Member được công bố là `mongo:27017`. Khi gọi từ host qua port 27018, `directConnection=true` giữ kết nối tới cổng đã publish, tránh tự khám phá hostname nội bộ Docker. Từ container cùng network dùng host `mongo:27017`. Đây là replica set một node cho phát triển/test, không phải cấu hình HA.

Test tạo database tên `aodora_dw01_test_<ObjectId>` mới, không dùng database trong URI, và chỉ dọn database đó. Chạy thiếu `MONGO_TEST_URI` hoặc nối standalone sẽ thất bại rõ, không skip để báo đạt giả. Test kiểm tra migration toàn bộ legacy → migration checkout UUID với dữ liệu cũ, document hợp lệ/không hợp lệ, rollback bị chặn, rollback/reapply thành công và commit/abort hai collection trên replica set thật. Test transaction chứng minh hạ tầng; chưa chứng minh nghiệp vụ chống bán vượt tồn của DW-02.

Runner lấy UUID áp dụng cuối từ ledger MongoDB, đọc file `down` của migration đó và lấy collection/validator ngay từ các lệnh `collMod`; không gắn version, tên collection hoặc field nghiệp vụ trong code kiểm tra rollback. Test checkout tự tìm migration theo tên `order_checkout`, lấy đúng các migration cha theo `depends_on`, không dựa vào số version hoặc thứ tự UUID. Có thêm test với collection/field/version khác để kiểm chứng cơ chế này. CLI nhận `-path`; Make hỗ trợ `MIGRATIONS_PATH` (mặc định `migrations`), còn integration test hỗ trợ `MONGO_MIGRATIONS_PATH` khi cần chọn thư mục khác.

`make transactions-down` dừng stack riêng và giữ volume. Default `make infra-up` và MongoDB đang dùng được giữ nguyên. Muốn thử API/seed trên DB riêng, cấu hình `MONGO_URI` theo URI test và chọn `MONGO_DB` riêng rồi chạy migration bằng lệnh hiện có; không chạy vào DB đang dùng khi chỉ muốn kiểm tra.

## Up/down và dữ liệu cũ

1. Chụp backup và dừng writer trước khi thay validator. Với DB local đang dùng, kiểm tra URI/database và migration version trước; chỉ áp dụng bằng `make migrate-up` khi chủ động chọn DB đó.
2. Up checkout UUID chỉ `collMod`, không rewrite/xóa đơn, không tạo khách giả, không đụng sản phẩm hoặc tồn. Index hiện tại được giữ nguyên. Field/index chống tạo lặp và guest access sẽ đi cùng hợp đồng DW-02/Q-03, trước khi có API phụ thuộc; migration này không tự quyết định schema review.
3. Down khôi phục chính xác validator orders từ 000004. Chạy qua `make migrate-down` (runner repository): trước khi đổi validator, runner đọc validator của lệnh `collMod` trong down và truy vấn document không tương thích. Có document vi phạm thì báo lỗi, giữ nguyên validator/version sạch và dữ liệu.
4. Guest thiếu/null `customer_id`, `cash_at_counter` hoặc địa chỉ thiếu phần giao hàng đều không tương thích schema cũ. Không đổi guest thành tài khoản giả, không đổi phương thức thu hoặc xóa đơn chỉ để down thành công. Nếu chưa có phương án chuyển dữ liệu hợp lệ và được thống nhất, giữ migration checkout đã áp dụng và khôi phục phiên bản ứng dụng tương thích version này.
5. Preflight và DDL không nằm trong cùng transaction; phải dừng mọi writer suốt quá trình down. Không chạy trực tiếp JSON down bằng `mongosh`/CLI khác để bỏ qua kiểm tra này. MongoDB `collMod` tự nó không quét dữ liệu đã tồn tại.

Không chuyển database standalone đang dùng thành replica set trong thay đổi này. Nếu cần chuyển dữ liệu thật sang môi trường replica set, thực hiện backup/restore và kiểm chứng riêng trước khi đổi cấu hình API; không tái sử dụng nhầm volume của repo khác.

## Ghép runner UUID

Runner UUID đã merge tại [PR #26](https://github.com/dont-wait/aodora/pull/26). Migration checkout nằm trong thư mục `ae74e773-1d52-45da-bd8d-07864eea7737_order_checkout`, phụ thuộc UUID của migration auth legacy được generator phát hiện. Thay đổi schema up/down giữ nguyên so với bản DW-01 trước khi đổi ID. DB cũ cần đối chiếu manifest và tiếp nhận lịch sử tường minh trước khi chạy; xem [UUID_MIGRATIONS.md](UUID_MIGRATIONS.md). Không chạy lại bản checkout số 000009 và UUID trên cùng DB; bản số chỉ tồn tại ở PR chưa merge và chưa được áp dụng vào DB đang dùng trong công việc này. Nếu đã tự áp dụng bản số đó ở môi trường khác, cần đối chiếu và chuyển metadata riêng, không xóa history để chạy lại.
