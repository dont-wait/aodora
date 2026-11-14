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

Kết nối MongoDB local bằng Compass/Navicat: sử dụng host/port và authentication database theo cấu hình MongoDB của môi trường, lấy tên database từ `MONGO_DB`. Thông tin tài khoản và mật khẩu được quản lý riêng trong biến môi trường/`.env`, không ghi trực tiếp vào tài liệu chung.

## Lưu ảnh bằng RustFS local

Cấu hình tham khảo `anomaly/server/docker-compose.yml`: RustFS cung cấp S3 API,
console và volume lưu dữ liệu riêng. Compose dùng tên project `aodora` để tách
container/network/volume khỏi các repo khác. Aodora dùng cổng `9100`/`9101` để tránh
trùng với RustFS của Anomaly. Có thể đổi bằng `RUSTFS_API_PORT` và
`RUSTFS_CONSOLE_PORT` trong `.env`.
Volume MongoDB vẫn dùng tên cũ `server_mongo_data` để giữ dữ liệu đã có;
nếu trước đây dùng tên project khác, đặt `MONGO_DATA_VOLUME` theo volume hiện tại.

```bash
make storage-up
```

Lệnh này khởi động RustFS, chờ healthcheck, tạo bucket `RUSTFS_BUCKET`
(mặc định `aodora-images`) và bật đọc ảnh công khai. Chạy lại không xóa ảnh.
`make infra-up` cũng khởi tạo storage cùng MongoDB.

1. Mở console `http://localhost:9101`, đăng nhập bằng `RUSTFS_ACCESS_KEY` và
   `RUSTFS_SECRET_KEY` trong `.env`.
2. Upload ảnh vào bucket `aodora-images`, ví dụ object `products/ao-dai/front.jpg`.
3. Nhập URL `http://localhost:9100/aodora-images/products/ao-dai/front.jpg`
   vào ô ảnh chính hoặc ảnh phụ của admin. Ảnh phụ nhập mỗi dòng `URL | mô tả ảnh`.
4. Lưu sản phẩm. MongoDB lưu URL và alt trong `products.image`, `products.gallery`
   và danh sách URL `products.images`; file nằm trong volume `rustfs_data`.

Bucket này dành riêng cho ảnh storefront: người chưa đăng nhập được đọc ảnh,
không được upload/xóa ẩn danh. Không đưa tài liệu riêng tư vào đây.
API Go và giao diện admin hiện vẫn nhận URL; chưa có endpoint/nút upload file.
Không dùng URL ký tạm thời trong gallery vì ảnh sẽ hết hạn.

`make storage-down` dừng storage và giữ dữ liệu. `docker compose down` giữ volume;
`docker compose down -v` xóa cả ảnh và dữ liệu MongoDB.
Khi triển khai ngoài local, đổi credentials, dùng domain HTTPS công khai trỏ đến
S3 API và lưu URL domain đó trong sản phẩm; trình duyệt khách phải truy cập được
URL này. Các cổng Compose hiện chỉ bind loopback.

## Seed catalog có ảnh RustFS

```bash
make seed-demo
```

Seeder tải chín ảnh tham khảo từ Forher, upload vào RustFS và thêm ba danh mục,
ba sản phẩm cùng SKU/tồn kho mẫu. Chạy lại giữ nguyên dữ liệu đã chỉnh.
Chi tiết nguồn ảnh, cấu hình URL và hành vi chạy lại: [seeds/README.md](seeds/README.md).

## Navicat qua trình duyệt

```bash
make navicat-up
```

Build image từ AppImage Navicat 18 for MongoDB đã tải trong `~/Downloads`, rồi mở
`https://localhost:9443` (tài khoản web mặc định `aodora` / `navicat-local`).
Navicat và cấu hình chạy trong Docker; không cần cài trực tiếp trên host.
Chi tiết build, kết nối MongoDB và volume: [docker/navicat/README.md](docker/navicat/README.md).

## Quy ước mở rộng

- Mỗi business capability có package riêng trong `internal/domain` và `internal/application`.
- Repository interface đặt ở layer sử dụng nó; MongoDB implementation đặt trong `internal/infrastructure`.
- Handler chỉ parse request, gọi use case, map response/error; không chứa business rule.
- Mọi thay đổi schema/index/validator phải có cặp migration `.up.json` và `.down.json`, tên theo số thứ tự.
- Không chạy migration tự động khi API boot; migration chạy riêng trong deploy/release step.

## Product catalog (PD-01)

Public read-only endpoints: `GET /api/v1/categories`, `GET /api/v1/products`, and `GET /api/v1/products/{slug}`. Filters, pagination, detail fields and consistency rules: [docs/PD01_CATALOG_API.md](docs/PD01_CATALOG_API.md).
