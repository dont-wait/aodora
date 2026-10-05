# Aodora server

Go API base theo Clean Architecture, MongoDB và [go-migrate](https://github.com/golang-migrate/migrate).

## Cấu trúc

```text
cmd/api/                 composition root, khởi động process
internal/config/         đọc và validate environment
internal/domain/         entity, value object, repository contracts
internal/application/    use cases và application services
internal/infrastructure/ database, external adapters
internal/interfaces/     HTTP handlers, transport DTOs
migrations/              MongoDB JSON migrations up/down
```

Dependency rule: domain/application không phụ thuộc HTTP hoặc MongoDB. `cmd/api` là nơi wire các implementation cụ thể vào application service.

## Chạy local

```bash
cp .env.example .env
make infra-up
make migrate-up
make run
```

Kiểm tra:

```bash
curl localhost:8080/healthz
curl localhost:8080/readyz
make test
```

`/healthz` chỉ xác nhận process đang sống. `/readyz` ping MongoDB và trả `503` khi database chưa sẵn sàng.

Config được load từ `.env` ở project root nếu file tồn tại, sau đó đọc environment variables. Environment variables được ưu tiên; các key chính là `HTTP_ADDR`, `LOG_LEVEL`, `LOG_FORMAT`, `MONGO_URI`, `MONGO_DB`.

Logger mặc định ghi structured JSON ra stdout. Có thể dùng `LOG_FORMAT=text` khi development local; mỗi HTTP request được log với method, path, status và duration.

Migration MongoDB:

```bash
make migrate-up
make migrate-up-one
make migrate-down
```

Schema, validator, index và mapping field từ client được mô tả tại [docs/database-schema.md](docs/database-schema.md). `make migrate-up` tạo collection bán hàng, seed demo và thêm `tailoring_requests` để lưu form may đo. Database mặc định là `QuanLyBanQuanAo` theo tài liệu đề tài; có thể đổi bằng `MONGO_DB`.

Sơ đồ có thể mở/chỉnh sửa bằng draw.io tại [docs/database-schema.drawio](docs/database-schema.drawio), hoặc xem nhanh bản [SVG](docs/database-schema.svg).

Kết nối MongoDB local bằng Compass/Navicat: host `localhost`, port `27017`, authentication database `admin`, username/password mặc định `aodora` / `aodora`, database `QuanLyBanQuanAo`. Connection string: `mongodb://aodora:aodora@localhost:27017/QuanLyBanQuanAo?authSource=admin`. Nếu đã đổi `MONGO_ROOT_USERNAME` hoặc `MONGO_ROOT_PASSWORD`, dùng giá trị tương ứng trong `.env`.

## Quy ước mở rộng

- Mỗi business capability có package riêng trong `internal/domain` và `internal/application`.
- Repository interface đặt ở layer sử dụng nó; MongoDB implementation đặt trong `internal/infrastructure`.
- Handler chỉ parse request, gọi use case, map response/error; không chứa business rule.
- Mọi thay đổi schema/index/validator phải có cặp migration `.up.json` và `.down.json`, tên theo số thứ tự.
- Không chạy migration tự động khi API boot; migration chạy riêng trong deploy/release step.
