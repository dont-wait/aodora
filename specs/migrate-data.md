# Migration dữ liệu sản phẩm áo dài (Sumire) → MongoDB + RustFS

Trạng thái: **ĐÃ TRIỂN KHAI** (chờ review/merge). Branch: `feature/migrate-sumire-catalog`.

## 0. Khác biệt so với bản draft (đã chốt khi code)

- **Không cần migration schema mới.** Validator `products` (000002/000004/000007) không đặt `additionalProperties:false` và `variants[].size/color` là string tự do, nên các field `seed_source`, `source_url`, `source_handle` và size `AA/A/B/C/D` đều hợp lệ.
- **Upload ảnh bằng service compose `rustfs-sumire`** (aws-cli `s3 sync`, cùng kiểu `rustfs-seed`), không thêm S3 SDK vào Go. `sync` bỏ qua file đã có nên chạy lại không upload lại.
- **`manifest.json` trên Drive không đủ dữ liệu** (thiếu biến thể/option/mô tả). Đã thêm `catalog.json` (1MB) vào cùng thư mục Drive: đầy đủ options, variants, `body_html`, và `images[].file` (đã giải quyết câu hỏi mở số 4). `manifest.json` giữ lại để tra `drive_file_id`.
- Quy tắc danh mục nằm trong code (`catalog.go`), không có `categories.json`/`colors.json`.
- `product_count`/`price_from` của danh mục **không lưu** (API/client tính từ `products`).
- Mặc định cho các câu hỏi mở (mục 10): ẩn sản phẩm `vd-ngung-sx`, bỏ loại "Khác", danh mục chia theo tag, giữ ảnh phụ 400×600.

## 1. Mục tiêu

Nạp một bộ catalog áo dài thật (223 sản phẩm, 1066 ảnh) vào mọi môi trường dev của nhóm, để các bạn clone repo là có dữ liệu và hình ảnh để làm UI/API. Ảnh nằm ở RustFS (S3-compatible), dữ liệu sản phẩm nằm ở MongoDB, chạy lại nhiều lần không tạo trùng.

Không nằm trong phạm vi: đồng bộ liên tục với website nguồn, nạp review/đơn/khách, sửa schema `products` ngoài mức bắt buộc (mục 6).

## 2. Nguồn dữ liệu

Nguồn gốc: https://sumirestore.com/collections/ao-dai (Shopify; `/collections/ao-dai/products.json`). Dữ liệu đã được thu thập sẵn lên Google Drive:

- Thư mục: `aodora-ao-dai-images` — https://drive.google.com/drive/folders/1qRvpcGhJJWpzN2nlxlND1SiAdYNR4X3p
- `catalog.json`: **nguồn chính cho importer** — mảng 223 phần tử `{handle, title, product_type, vendor, tags[], body_html, source_url, options[], variants[], images[{order,file,width,height,source}]}`.
- `manifest.json`: mảng 223 phần tử `{handle, title, product_type, vendor, tags, price, compare_at_price, source_url, drive_folder_id, images[]}`; mỗi ảnh `{order, width, height, source, drive_file_id}`.
- `<handle>/NN.<ext>`: `01` là ảnh chính (683×1024, ~1.2MB), các ảnh sau là 400×600 (~400KB). `ext` theo đuôi URL nguồn (`png`, `jpg`, `webp`; `jpeg` đổi thành `jpg`). 145/1066 ảnh không phải PNG, nên **không được hardcode `.png`**.

Tên file ảnh nằm ở `catalog.json` (`images[].file`), không còn phải suy từ đuôi URL.

Dữ liệu thô không chứa tồn kho và hàng hết ngưng bán (`vd-ngung-sx` là tag ngừng sản xuất, xem 5.4).

### Bản quyền

Ảnh/mô tả là của Sumire Store. Chỉ dùng cho dev/demo nội bộ. Không commit ảnh vào Git, không đưa lên môi trường công khai. Mỗi sản phẩm lưu `source_url` + `seed_source` để truy nguồn (giống seed Forher hiện có).

## 3. Kiến trúc

Theo mô hình đã có ở `server/cmd/seed` + `server/seeds/`, **không** nhét dữ liệu vào migration JSON, vì migration JSON chỉ chạy lệnh MongoDB, không upload được ảnh và phải giữ checksum bất biến.

| Thành phần | Việc |
| --- | --- |
| `server/seeds/sumire/` (gitignore) | Thư mục cục bộ chứa `catalog.json` + `<handle>/NN.<ext>` tải từ Drive |
| `server/cmd/importsumire/` | Validate `catalog.json` + file ảnh → upsert `categories`/`products` (chỉ DB, không upload) |
| `docker-compose.yml` → `rustfs-sumire` (profile `sumire`) | `aws s3 sync` ảnh lên `catalog/sumire/` |
| `Makefile` | `sumire-fetch`, `sumire-check`, `sumire-images`, `sumire-import`, `sumire-purge` |
| `server/seeds/README.md` | Hướng dẫn cho đồng đội |

### 3.1 Lấy dữ liệu từ Drive về máy

Drive cần đăng nhập nên importer **không** gọi Drive trực tiếp. Hai cách, đều kết thúc bằng thư mục `server/seeds/sumire/`:

1. Thủ công: tải thư mục `aodora-ao-dai-images` từ Drive (Download → zip), giải nén vào `server/seeds/sumire/`.
2. `rclone` (khuyến nghị cho ai dùng nhiều): `rclone copy gdrive:aodora-ao-dai-images server/seeds/sumire`. `make sumire-fetch` chỉ in hướng dẫn / gọi rclone nếu đã cấu hình remote `gdrive`, không tự lấy credential.

Thư mục Drive phải được chia sẻ cho thành viên (chủ thư mục thực hiện, ngoài phạm vi code).

### 3.2 Luồng chạy

```
make sumire-fetch          # (hoặc tải tay từ Drive vào server/seeds/sumire/)
make sumire-check          # validate catalog + file ảnh, không đụng DB
make sumire-import         # check + infra-up + migrate-up + upload ảnh + upsert DB
# hoặc từng bước: make sumire-images && go run ./cmd/importsumire
```

`importsumire` có cờ: `-check` (chỉ validate), `-dry-run` (đọc DB, in số bản ghi sẽ thêm/bỏ qua), `-limit N`, `-update`, `-default-stock N`, `-purge -yes`, `-allow-remote`.

## 4. Lưu trữ ảnh (RustFS)

- Bucket: `$RUSTFS_BUCKET` (mặc định `aodora-images`), bucket đã public read (xem `rustfs-init`).
- Key: `catalog/sumire/<handle>/NN.<ext>` (khác prefix `demo/forher/` của seed cũ để không đè nhau).
- Upload bằng service `rustfs-sumire` (`aws s3 sync`, mỗi đuôi png/jpg/webp một lượt với đúng `Content-Type`, `Cache-Control: public,max-age=86400`). `sync` bỏ qua file đã đồng bộ, không xóa object khác.
- Importer kiểm tra trước khi ghi DB: file tồn tại, ≥ 12 byte, ≤ 20MB, magic bytes khớp đuôi (PNG/JPEG/WebP), đường dẫn không thoát khỏi thư mục assets.
- URL lưu trong DB: `<RUSTFS_PUBLIC_URL>/catalog/sumire/<handle>/NN.<ext>`. `publicURL()` được viết lại cục bộ (~10 dòng) vì `cmd/seed` là `package main`.

## 5. Ánh xạ dữ liệu → MongoDB

Schema đích: `server/docs/database-schema.md` (`products`, `categories`). Validator ở chế độ `strict/error`, nên mọi document phải qua validator hiện tại; nếu cần field mới thì xử lý ở mục 6.

### 5.1 ID và idempotency

- `_id` = 12 byte đầu của SHA-256 `aodora:sumire:<kind>:<key>` → chạy lại ra cùng ObjectId.
- Ghi bằng `$setOnInsert` (không ghi đè dữ liệu đã chỉnh tay như tồn kho, giá, `is_active`). Có cờ `-update` tường minh nếu muốn làm mới các field nguồn (name, description, images) nhưng **không bao giờ** đụng `variants[].stock`, `stock`, `version`.
- `slug` = `sumire-<handle>`; `sku` cấp sản phẩm = SKU biến thể đầu tiên bỏ hậu tố size. Variant thiếu SKU → `SUMIRE-<HANDLE>-<n>`. **Nguồn dùng lại SKU giữa các sản phẩm** (đã gặp `ADTT2198A/B/C`): SKU trùng được thêm hậu tố ổn định `-<6 hex theo handle>` và ghi cảnh báo. Slug đang bị bản ghi khác chiếm → báo lỗi, không ghi đè.

### 5.2 Danh mục

Không dùng 1 danh mục chung. Suy từ `tags` và tên (hằng số trong `catalog.go`):

- `aodai4ta` / "tà đôi" → `ao-dai-ta-doi`; `AODAITAYCHEN` → `ao-dai-tay-chen`; `tet2026` → `ao-dai-tet`; còn lại → `ao-dai-sumire` (mặc định).
- Sản phẩm `product_type = "Khác"` (3 sản phẩm): **không import**, ghi vào báo cáo "bỏ qua" để người duyệt xem (có thể là phụ kiện).
- Danh mục có tiền tố slug `sumire-`, `sort_order` 100–103, `is_active=true`. Chỉ tạo danh mục có sản phẩm.

### 5.3 Sản phẩm

| Field đích | Nguồn |
| --- | --- |
| `name` | `title` |
| `slug`, `sku` | 5.1 |
| `category_id` | 5.2 |
| `collection` | Cố định `"Sumire Store"` (tag `bst-*` là mã nội bộ, không đẹp để hiển thị) |
| `description` | `body_html` chuyển sang text thuần (bỏ thẻ, `&nbsp;`, dòng `____`); giữ đoạn đầu mô tả, tối đa ~1000 ký tự |
| `material` | Chỉ lấy khi có nhãn rõ: `Chất liệu: X` hoặc mục `[Chất liệu]` + dòng kế tiếp (cắt tại `: , . ;`). Không đoán từ văn xuôi; không có nhãn thì bỏ field (≈183/220 có) |
| `price` | giá variant (VND nguyên) |
| `old_price`, `compare_at_price` | `compare_at_price` nếu > price (169/223 sản phẩm có), ngược lại bỏ |
| `image` | ảnh `order=1`; `alt` = `title` |
| `gallery[]` | các ảnh còn lại, `alt` = `title — ảnh N` |
| `images[]` | danh sách URL (field legacy mà seed đang ghi) |
| `colors[]` | giá trị option "Màu sắc" → `{name, hex}`; hex = từ màu xuất hiện **sớm nhất** trong tên ("Nền trắng hoa tím" → trắng), bảng trong `catalog.go`; tên riêng như "Nguyệt Mai" dùng `#CCCCCC` và in cảnh báo |
| `sizes[]` | giá trị option "Kích thước", chuẩn hóa về size chuẩn VN `XS/S/M/L/XL/XXL/Freesize`: bỏ tiền tố "Size ", bỏ tên mẫu như "Tú Trân / ", rồi map `AA→XS, A→S, B→M, C→L, D→XL`; sắp theo thứ tự chuẩn (SKU giữ nguyên hậu tố cũ). Dữ liệu đã import trước đó được đổi bằng migration `*_size_standard` |
| `variants[]` | `{sku, size, color, stock, price}` mỗi variant Shopify. Option size/màu lấy theo `position` của option, không theo thứ tự mảng. `price` theo từng variant; `price` sản phẩm = giá thấp nhất (chỉ 1 sản phẩm có giá khác nhau, có cảnh báo khi import) |
| `stock` | tổng `variants[].stock` |
| `is_active` | true, trừ khi theo 5.4 |
| `badge`/`tag` | `tag` "Ngừng sản xuất" nếu có `vd-ngung-sx` (5.4); không bịa badge khác |
| `source_url`, `seed_source` | `source_url` từ manifest; `seed_source="sumire-store"` |
| `created_at`, `updated_at` | thời điểm import |

Không điền `story`: mô tả sản phẩm nằm ở `description`, không nhân đôi thành tab "Câu chuyện tác phẩm". Ảnh được sắp theo `order`. Chỉ nhận đuôi `png/jpg/webp` (`.jpeg` bị từ chối vì `rustfs-sumire` không sync). Không điền `rating`, `review_count`, `story`, `fabric`, `craft`, `accessories`, `trust` bằng nội dung bịa. Nếu validator bắt buộc, dùng giá trị rỗng/mặc định trung tính (kiểm tra ở 6).

### 5.4 Tồn kho và trạng thái

Shopify chỉ cho `available` (bool), không có số lượng. Quy ước dữ liệu dev, ghi rõ là **giả lập**:

- variant `available=true` → `stock = 5`; `false` → `stock = 0` (hết hàng, để thử UI hết hàng).
- Sản phẩm có tag `vd-ngung-sx` → `is_active=false` (không hiện ở catalog công khai), vẫn import để giữ dữ liệu. Cần chủ dự án xác nhận: ngoài ra có thể chọn vẫn `is_active=true` nhưng gắn tag "Ngừng sản xuất".
- Hằng số 5 đặt ở flag `-default-stock`.

## 6. Schema / migration

**Không thêm migration.** Đã kiểm tra validator `products` (000007 là bản mới nhất): không `additionalProperties:false`; `variants[].size/color` là string tự do (cho phép `""` với sản phẩm không có size); `sizes` là mảng string. Import thật trên DB sạch sau `migrate-up` thành công 220/220 sản phẩm.

Nếu sau này validator bị siết, thêm migration **mới** theo quy trình UUID (`make migrate-create`), không sửa migration cũ.

`-purge -yes` xóa **chỉ** document có `seed_source="sumire-store"` và danh mục `sumire-*` đã hết sản phẩm; không đụng ảnh trong RustFS.

## 7. Báo cáo, lỗi, an toàn

- Kết thúc in bảng: sản phẩm thêm / bỏ qua (đã có) / bị loại (+lý do), ảnh upload / bỏ qua / lỗi, cảnh báo màu thiếu hex, sản phẩm chưa suy ra được size.
- Lỗi dữ liệu (handle sai/trùng, thiếu ảnh/file, giá sai, slug bị bản ghi khác chiếm) làm lệnh dừng với exit 1 **trước khi ghi gì**; chạy lại an toàn.
- `MONGO_URI` không phải localhost (mọi host) bị từ chối khi không có `-allow-remote`, kể cả `-purge`. `make sumire-import` kiểm tra điều này **trước** `migrate-up` (`importsumire -verify-target`). `-purge` không dùng chung được với `-dry-run`; `-default-stock` tối đa 100000.
- Giới hạn đã biết (dev tool, chạy tuần tự một người): ghi theo từng document, không transaction (chạy lại để hoàn tất nếu bị ngắt); `-update` không tăng `version` và không chống ghi chồng giữa hai snapshot catalog khác nhau.
- Không log credential. Không commit `seeds/sumire/`, `.env`.
- Các bước ảnh/DB không nằm trong một transaction; thứ tự: ảnh trước, DB sau để DB không bao giờ trỏ tới ảnh chưa có.

## 8. Kiểm thử (đã chạy)

Kết quả thực tế: `go test ./cmd/importsumire` xanh; import thật 4 danh mục + 220 sản phẩm + 1058 ảnh; lần 2 không thêm bản ghi; `-update` giữ nguyên `stock`/`price` đã sửa tay và xóa `material` cũ khi nguồn không còn; `GET /api/v1/categories` và `/products/{slug}` trả đúng, sản phẩm `is_active=false` trả 404; ảnh trả 200 đúng Content-Type. Danh sách dự kiến ban đầu:


- Unit: chuẩn hóa size/màu, strip HTML, sinh ID/slug/SKU, ánh xạ danh mục, đoán MIME/ext từ source URL, `demoID` ổn định.
- `-check` trên fixture nhỏ (3 sản phẩm, ảnh PNG/JPG/WebP tí hon) trong `server/docs/fixtures/` hoặc `testdata`.
- Integration (Mongo test DB như các test hiện có, `MONGO_TEST_URI`): import 2 lần → số document không đổi; sửa `stock` tay rồi import lại → không bị ghi đè; slug bị chiếm → lỗi.
- Smoke với RustFS thật: `make storage-up && make sumire-import -limit 5` rồi `curl -I` URL ảnh trả 200 + đúng Content-Type.
- Sau import đầy đủ: 220 sản phẩm (223 − 3 loại "Khác"), tổng ảnh khớp manifest (trừ ảnh của sản phẩm bị loại), API catalog `GET` danh sách/chi tiết trả đúng và client hiển thị được.

## 9. Tiêu chí hoàn thành

1. `make sumire-check` và `make sumire-import` chạy được trên máy sạch theo README, từ thư mục `seeds/sumire/` tải từ Drive.
2. Chạy lại lần 2 không thay đổi dữ liệu, không upload lại ảnh.
3. Ảnh mở được qua `RUSTFS_PUBLIC_URL`; client hiển thị danh sách + chi tiết sản phẩm Sumire.
4. `make test` (+ test migration nếu có migration mới) xanh; lint sạch.
5. README/docs cập nhật: cách lấy dữ liệu từ Drive, cách import, cách purge, lưu ý bản quyền.

## 10. Câu hỏi mở (mặc định đã áp dụng, đổi được bằng một dòng code)

1. Sản phẩm `vd-ngung-sx` (~51 sản phẩm): ẩn (`is_active=false`) hay vẫn hiển thị kèm nhãn?
2. 3 sản phẩm `product_type="Khác"`: bỏ hay nạp như phụ kiện?
3. Bộ danh mục theo tag (5.2) có ổn không, hay muốn 1 danh mục "Áo dài" duy nhất cho đơn giản?
4. ~~Bổ sung `file` vào manifest~~ → đã giải quyết bằng `catalog.json`.
5. Ảnh phụ 400×600 có đủ chất lượng cho UI chi tiết sản phẩm không, hay cần tải lại bản 683×1024 cho toàn bộ?
