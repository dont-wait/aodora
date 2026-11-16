# Migration JSON dùng UUID

Issue [#25](https://github.com/dont-wait/aodora/issues/25). UUID là định danh migration, còn thứ tự thực thi dựa trên phụ thuộc và lịch sử thực tế, không dựa trên UUID lớn nhất. Hai nhánh cùng xuất phát từ một schema có thể tạo migration độc lập mà không cùng chọn số tiếp theo.

## Tạo và chạy

Trong `server/`, dùng Nix shell của repo:

```bash
nix develop
make migrate-create name=add_order_checkout
# Sửa hai file JSON up/down vừa sinh, giữ metadata trong cùng commit.
make migrate-up
make migrate-up-one
make migrate-down
```

`migrate-create` chạy được khi không có DB hoặc `.env`. `MIGRATIONS_PATH` cấu hình thư mục cho các Make target; CLI cũng hỗ trợ `-path`.

Một migration mới có ba file:

```text
<UUID>_add_order_checkout.up.json
<UUID>_add_order_checkout.down.json
<UUID>_add_order_checkout.meta.json
```

Up/down giữ định dạng mảng lệnh MongoDB Extended JSON hiện có. Generator tạo UUIDv4 bằng nguồn ngẫu nhiên của hệ điều hành và metadata `{"depends_on":["UUID-cha", ...]}`. Nó tự chọn các migration cuối mỗi nhánh phụ thuộc đang có trong thư mục làm cha. Mỗi migration có metadata riêng; không sửa manifest chung hoặc tăng counter.

Runner xác minh đủ cặp file, ID không trùng, mọi phụ thuộc tồn tại và không có chu trình trước khi kết nối DB. Chạy theo thứ tự topo; các migration độc lập có thứ tự xác định theo ID để có thể tái lập. Nếu migration B được merge sau khi A đã chạy, B vẫn được áp dụng dù UUID của B nhỏ hơn A. Nếu B thực sự phụ thuộc A thì ghi ID của A trong `depends_on` trước lần áp dụng đầu.

UUID tránh trùng định danh, không tự hợp nhất thay đổi nghiệp vụ: nếu hai nhánh cùng thay validator của một collection, vẫn cần đối chiếu/ghép validator và phụ thuộc phù hợp. Không đổi `depends_on`, tên hoặc nội dung file đã áp dụng; runner kiểm tra checksum và từ chối file bị sửa/mất.

## Tương thích DB và migration số cũ

- Giữ nguyên file `000001_...` đến các migration legacy hiện có. Loader ánh xạ version số sang UUIDv5 ổn định và xâu chuỗi theo thứ tự số; đây chỉ là lớp tương thích, migration mới dùng generator UUID.
- Lần chạy đầu tiên giữ khóa migration rồi đọc `schema_migrations` của go-migrate. Nếu DB có version sạch, runner ghi nhận các migration legacy đã chạy vào ledger mới, không chạy lại các lệnh schema/seed của chúng. Version dirty hoặc thiếu file tại mốc đã áp dụng sẽ bị từ chối.
- Lịch sử mới nằm ở `schema_migrations_uuid`: document `_id=state` giữ `head`, `dirty`, `pending`, `direction`, `legacy_version`; mỗi UUID đã áp dụng có document `_id=UUID`, `previous`, `checksum`. Chuỗi `previous` lưu thứ tự thực tế, kể cả nhánh được merge muộn.
- Metadata số cũ được giữ như snapshot tại lúc tiếp nhận, không còn là version hoạt động sau đó. Chỉ dùng CLI repository từ đây; không trộn với go-migrate CLI cũ. Nếu version legacy bị thay đổi sau tiếp nhận, runner từ chối chạy.
- DB mới cũng chạy được toàn bộ file legacy qua ledger UUID. Up chỉ chạy ID chưa áp dụng; `up-by-one` chạy một ID còn thiếu; `down` bỏ migration áp dụng cuối cùng trong chuỗi lịch sử, không chọn UUID lớn nhất.

## Khóa, rollback và lỗi

Runner giữ advisory lock của MongoDB driver go-migrate từ trước khi đọc version/lịch sử đến khi hoàn tất preflight và ghi schema/lịch sử. Lock tương thích với runner số cũ. Không cho tắt `x-advisory-locking`; đọc version, ledger và preflight luôn từ primary kể cả URI cấu hình secondary. Hai lệnh `down` riêng biệt được tuần tự hóa và mỗi lệnh hạ một migration theo lịch sử tại lúc giữ khóa.

Trước down, runner đọc các validator trong lệnh `collMod` của file down và từ chối nếu có document không tương thích; không hardcode collection/field/version. Tuy nhiên advisory lock chỉ chặn migrator, không chặn API hoặc script ghi dữ liệu. **Dừng mọi writer suốt quá trình migration, đặc biệt từ trước preflight đến hết down.** DDL và preflight không nằm trong cùng transaction; `x-transaction-mode=true` bị từ chối thay vì bỏ qua. Không cho URI `w=0`.

Trước mỗi migration, ledger ghi `dirty=true`, ID và hướng chạy. Chỉ xóa dirty sau khi tất cả lệnh thành công và ledger được cập nhật. Lỗi command, `writeErrors` hoặc `writeConcernError` đều làm thao tác thất bại. Migration nhiều lệnh có thể đã ghi một phần; runner không retry mù hoặc tự động rollback dữ liệu.

Khi dirty: giữ writer dừng, xem ID/hướng và các lệnh đã chạy, khôi phục backup hoặc hoàn tất/đảo thao tác còn dở sau khi kiểm tra DB. Không xóa dirty chỉ để tiếp tục khi schema/lịch sử chưa khớp. Nếu lỗi trong lúc import metadata, khôi phục metadata từ backup và chạy lại với snapshot legacy gốc; không xóa dữ liệu nghiệp vụ. Nếu process bị kill khi giữ lock, chỉ gỡ advisory lock sau khi xác minh process đó đã dừng. Chưa có lệnh `force` tự sửa lịch sử.

## Kiểm tra

```bash
make test
# MONGO_TEST_URI trỏ DB test riêng; tests tự tạo và dọn database có tên ngẫu nhiên.
make test-migrations
```

Tests kiểm tra generator hai nhánh, phụ thuộc đảo thứ tự UUID, thiếu file/cha/chu trình/ID trùng, tiếp nhận DB từ runner số cũ, đối chiếu dữ liệu và validator của DB mới giữa runner cũ/mới, migration merge muộn, rollback theo lịch sử, checksum, lỗi ghi/dirty và khóa chung. Integration test dùng MongoDB thật, không thay thế bằng mock hoặc skip khi thiếu URI. Chưa tự áp dụng thay đổi vào database đang dùng.
