# Backup và khôi phục MongoDB trước migration đơn hàng

Thực hiện trong `server/`, chọn đúng stack/database. Dừng API, worker, script ghi dữ liệu và mọi migrator trước khi dump; giữ chúng dừng đến hết up/down hoặc kiểm tra restore. Advisory migration lock không chặn các application writer. Không dùng backup có dirty migration hoặc lock đang được giữ làm snapshot sạch để khôi phục tùy ý.

## Backup trước thay đổi

Chọn compose và DB bằng biến môi trường; không ghi URI/password vào command history hay file được commit. Ví dụ test dùng `docker-compose.transactions.yml`; DB đang dùng thường là `docker-compose.yml`. Container MongoDB lấy username/password từ environment sẵn có của chính container.

```bash
set -eu
umask 077
set -C  # Không ghi đè archive đã tồn tại.
export COMPOSE_FILE=docker-compose.transactions.yml
export MONGO_DB=ten_database_can_backup
export BACKUP_FILE=/tmp/aodora-before-checkout.archive.gz

docker compose exec -T mongo sh -ec '
  exec mongodump --username "$MONGO_INITDB_ROOT_USERNAME" --password "$MONGO_INITDB_ROOT_PASSWORD" \
    --authenticationDatabase admin --db "$1" --archive --gzip
' sh "$MONGO_DB" > "$BACKUP_FILE"
test -s "$BACKUP_FILE"
sha256sum "$BACKUP_FILE"
```

Kiểm tra exit code của mongodump trước khi coi backup thành công. Archive phải gồm cả dữ liệu, validator/index và metadata `schema_migrations`/`schema_migrations_uuid`; không chỉ export collection orders. Ghi lại commit ứng dụng, manifest fingerprint, UUID head, trạng thái dirty, tên DB và checksum archive bên ngoài repository nếu chứa thông tin riêng tư. Trên replica set nhiều node, việc dừng writer trong toàn bộ dump là điều kiện để snapshot nhiều collection nhất quán ở quy trình này.

## Restore vào database mới trước khi đổi ứng dụng

Không chạy restore đè DB đang dùng để thử nghiệm. Chọn tên mới chưa tồn tại và chỉ chạy khi không có writer/migrator ở DB đích; không dùng `--drop` trong quy trình này.

```bash
set -eu
export RESTORE_DB=ten_database_restore_moi
test "$RESTORE_DB" != "$MONGO_DB"

# Bảo đảm database đích chưa tồn tại, kể cả database chỉ có dữ liệu nhỏ.
docker compose exec -T -e RESTORE_DB mongo sh -ec '
  exec mongosh --quiet --username "$MONGO_INITDB_ROOT_USERNAME" --password "$MONGO_INITDB_ROOT_PASSWORD" \
    --authenticationDatabase admin --eval '\''
      const name = process.env.RESTORE_DB;
      if (!name) throw new Error("RESTORE_DB is required");
      const names = db.adminCommand({listDatabases: 1, nameOnly: true}).databases.map(x => x.name);
      if (names.includes(name)) throw new Error("Restore target already exists");
    '\''
'

docker compose exec -T mongo sh -ec '
  exec mongorestore --username "$MONGO_INITDB_ROOT_USERNAME" --password "$MONGO_INITDB_ROOT_PASSWORD" \
    --authenticationDatabase admin --archive --gzip --stopOnError \
    --nsInclude "$1.*" --nsFrom "$1.*" --nsTo "$2.*"
' sh "$MONGO_DB" "$RESTORE_DB" < "$BACKUP_FILE"
```

Exit code thành công chưa đủ để đổi DB: so sánh số document theo từng collection và các bản ghi nghiệp vụ đại diện, validator/index orders, các tham chiếu, ledger UUID head/dirty/checksum và legacy version với snapshot ghi lúc backup. Bảo đảm không có advisory lock đang hoạt động trong snapshot. Chạy thử phiên bản ứng dụng tại commit tương ứng trên DB restore riêng, rồi mới chủ động đổi `MONGO_DB` cho ứng dụng và khởi động lại writer.

Backup trước khi có lịch sử UUID phải tiếp nhận legacy bằng manifest/fingerprint đã đối chiếu ở [UUID_MIGRATIONS.md](UUID_MIGRATIONS.md); backup đã có ledger UUID thì giữ nguyên ledger. Không xóa history hoặc chạy lại seed để làm dữ liệu trông đúng. Nếu restore thất bại, dừng tại DB đích mới, giữ archive/DB gốc và điều tra; không chuyển app sang DB đích chưa kiểm chứng.

Khi down checkout bị chặn bởi đơn guest/tại quầy, có thể giữ schema mới và chạy phiên bản ứng dụng tương thích. Restore snapshot cũ sẽ không chứa đơn phát sinh sau snapshot: phải đối soát và có phương án giữ các đơn đó trước khi thay DB, không coi restore là rollback không mất dữ liệu.

Việc backup/restore không đổi tên/mount volume `server_mongo_data` hoặc volume replica set test. Các lệnh trên dùng mongodump/mongorestore có sẵn trong image MongoDB của stack. Quyền đọc archive tương đương quyền đọc toàn DB; không đưa archive vào Git hoặc public bucket ảnh.

Đã kiểm chứng quy trình bằng hai DB tạm trên replica set test: dump/restore giữ nguyên count của 11 collection (3 đơn gồm seed và fixture), validator/index orders, UUID ledger head và guest credential hash. Cả hai DB tạm được dọn sau kiểm tra; không dùng dữ liệu DB đang vận hành.
