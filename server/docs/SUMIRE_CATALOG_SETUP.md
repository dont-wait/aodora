# Hướng dẫn nạp dữ liệu áo dài Sumire (cho đồng đội)

Sau khi làm xong, máy bạn có **220 sản phẩm áo dài + hơn 1000 ảnh** để làm UI/API. Mất khoảng 10–15 phút, chủ yếu là chờ tải ảnh từ Drive.

> Ảnh/mô tả là của Sumire Store. Chỉ dùng để dev/demo nội bộ, đừng đưa lên môi trường công khai.
> Chi tiết kỹ thuật: [specs/migrate-data.md](../../specs/migrate-data.md).

## 0. Cần có sẵn

- Docker (đang chạy) và Go, hoặc `nix develop` trong thư mục `server/` như các phần khác của dự án.
- Quyền xem thư mục Google Drive **aodora-ao-dai-images** (nhờ chủ dự án chia sẻ nếu chưa có).
- ~1GB trống ổ đĩa.

## 1. Chuẩn bị (làm một lần)

```bash
cd server
cp .env.example .env        # bỏ qua nếu đã có .env
git checkout feature/migrate-sumire-catalog   # hoặc main nếu đã merge
```

## 2. Tải dữ liệu từ Drive

1. Mở thư mục Drive **aodora-ao-dai-images** → chuột phải → **Tải xuống** (Drive sẽ nén thành vài file zip vì thư mục lớn).
2. Giải nén tất cả vào `server/seeds/sumire/`.
3. Kiểm tra cấu trúc phải đúng thế này (không có thư mục lồng thêm `aodora-ao-dai-images/`):

```
server/seeds/sumire/
├── catalog.json
├── manifest.json            (không bắt buộc)
├── ao-dai-ta-doi-van-hoa-do/
│   ├── 01.png
│   └── 02.png ...
└── ... (223 thư mục)
```

Nếu bị lồng thư mục, di chuyển nội dung ra một cấp:

```bash
mv seeds/sumire/aodora-ao-dai-images/* seeds/sumire/ && rmdir seeds/sumire/aodora-ao-dai-images
```

*Cách nhanh hơn nếu bạn có rclone:* cấu hình remote tên `gdrive` rồi chạy `make sumire-fetch`.

Thư mục `seeds/sumire/` đã được gitignore, sẽ không bị commit nhầm.

## 3. Kiểm tra dữ liệu (không đụng DB)

```bash
make sumire-check
```

Kết quả đúng: `catalog: 4 categories, 220 products, 1058 images; skipped 3` và dòng cuối `check OK`.
Có vài dòng `warning` về màu hoặc SKU là bình thường.

## 4. Nạp vào MongoDB và RustFS

```bash
make sumire-import
```

Lệnh này tự động: kiểm tra lại → bật MongoDB + RustFS → chạy migration → upload ảnh → ghi sản phẩm vào DB. Cuối cùng in:

```
categories: inserted 4, preserved 0, refreshed 0
products: inserted 220, preserved 0, refreshed 0
```

**Chạy lại bao nhiêu lần cũng an toàn**: không tạo trùng, không ghi đè tồn kho/giá bạn đã sửa.

## 5. Kiểm tra kết quả

```bash
make run     # bật API ở cổng trong .env (mặc định :8080)
curl -s localhost:8080/api/v1/categories
curl -s localhost:8080/api/v1/products/sumire-ao-dai-ta-doi-van-hoa-do
```

Hoặc mở client và vào trang danh mục áo dài. Ảnh được phục vụ từ RustFS, ví dụ:
`http://localhost:9100/aodora-images/catalog/sumire/ao-dai-ta-doi-van-hoa-do/01.png`
(xem file trong giao diện RustFS tại `http://localhost:9101`, tài khoản trong `.env`).

## Những điều cần biết về dữ liệu

| | |
| --- | --- |
| Tồn kho | **Giả lập**: biến thể còn hàng = 5, hết hàng = 0 (nguồn không có số lượng thật) |
| Sản phẩm ngừng sản xuất | 69 sản phẩm bị ẩn (`is_active=false`), API công khai trả 404 cho chúng |
| Bị bỏ qua | 3 sản phẩm loại "Khác" |
| Ảnh | Ảnh chính 683×1024, ảnh phụ 400×600 (đã thu nhỏ so với bản gốc) |
| Slug / danh mục | Slug có tiền tố `sumire-`, 4 danh mục `sumire-*` |

## Lệnh hữu ích

| Việc | Lệnh |
| --- | --- |
| Xem sẽ thêm gì, chưa ghi | `go run ./cmd/importsumire -dry-run` |
| Thử nhanh 5 sản phẩm | `go run ./cmd/importsumire -limit 5` |
| Làm mới tên/mô tả/ảnh/màu | `go run ./cmd/importsumire -update` |
| Xóa dữ liệu Sumire khỏi DB | `make sumire-purge` (không xóa ảnh trong RustFS) |
| Chỉ upload lại ảnh | `make sumire-images` |

## Gặp lỗi?

| Lỗi | Cách xử lý |
| --- | --- |
| `open seeds/sumire/catalog.json: no such file` | Chưa tải/giải nén Drive đúng chỗ (bước 2) |
| `image .../01.png missing` | Thiếu thư mục sản phẩm: tải lại các file zip còn thiếu, giải nén vào đúng `seeds/sumire/` |
| `content does not match its extension` | File ảnh bị hỏng khi tải: tải lại thư mục sản phẩm đó |
| `Command createIndexes requires authentication` (lúc `migrate-up`) | Chưa có `server/.env` nên code dùng URI không có tài khoản. Chạy `cp .env.example .env` rồi `make sumire-import` lại (`make sumire-import` giờ tự tạo `.env` nếu thiếu) |
| Cảnh báo `volume "server_mongo_data" already exists but was created for project "server"` | Bình thường, chỉ là cảnh báo: dự án cố ý dùng lại volume cũ để giữ dữ liệu MongoDB đang có |
| `MONGO_URI is not localhost: refusing…` | `.env` đang trỏ DB không phải máy bạn. Đổi về `localhost` (cố ý trỏ DB khác mới dùng `-allow-remote`) |
| `missing products collection: run make migrate-up` | Chạy `make migrate-up` rồi import lại (`make sumire-import` đã tự làm bước này) |
| `slug ... is already used by another record` | Có sản phẩm khác đang dùng slug `sumire-...`. Báo lại, đừng ép ghi đè |
| Port 27017 / 9100 đã bị chiếm | Dừng container khác đang dùng cổng, hoặc đổi `RUSTFS_API_PORT` trong `.env` |
| Ảnh không hiện trên trình duyệt | Kiểm tra `RUSTFS_PUBLIC_URL` trong `.env` (để trống thì dùng `http://localhost:9100/aodora-images`) và bucket có chạy `make sumire-images` |
