# Catalog demo tham khảo Forher

Nguồn danh mục: https://forhershop.vn/product-category/ao-dai/

`forher.json` chứa ba sản phẩm được chọn từ trang nguồn, mỗi sản phẩm có ba URL
ảnh WebP gốc và đường dẫn trang chi tiết. Tên, giá, chất liệu, cỡ và màu được
tham khảo ngày 2026-11-14; tồn kho/SKU là dữ liệu giả phục vụ thử nghiệm Aodora.
Danh mục được nhóm lại cho demo, không phải bản sao phân loại của Forher.
Không sao chép chính sách bán hàng hoặc đánh giá khách hàng.

## Chạy

Trong `server/`, chuẩn bị `.env` theo `.env.example`, rồi chạy trong Nix shell:

```bash
nix develop
make seed-demo
```

Lệnh này:

1. Tải chín ảnh từ các URL cố định trong manifest về `seeds/assets/`.
2. Kiểm tra manifest và định dạng ảnh trước khi ghi dữ liệu.
3. Bật MongoDB và RustFS, tạo bucket ảnh và chạy migrations.
4. Upload ảnh lên `s3://$RUSTFS_BUCKET/demo/forher/`.
5. Thêm ba danh mục và ba sản phẩm có ảnh chính, gallery, thông tin giới thiệu,
   chất liệu và chín SKU (S/M/L), gồm một mẫu hết hàng để thử giao diện.

MongoDB dùng `MONGO_URI`/`MONGO_DB`. Trình duyệt đọc ảnh qua
`RUSTFS_PUBLIC_URL` (URL bucket công khai, không phải endpoint nội bộ Docker).
Nếu để trống, URL được suy ra từ `http://localhost:$RUSTFS_API_PORT/$RUSTFS_BUCKET`.

Ví dụ trang chi tiết sau khi seed:

- `/san-pham/demo-ao-dai-moc-tam-xanh-reu`
- `/san-pham/demo-ao-dai-ngoc-an-bi`
- `/san-pham/demo-ao-dai-thanh-xuan-do`

Kiểm tra đường dẫn router của frontend nếu chạy phiên bản giao diện khác.

## Chạy lại và bảo toàn dữ liệu

ID được suy ra ổn định từ loại bản ghi và mã sản phẩm. Seeder dùng `$setOnInsert`:
chạy lại không tạo trùng, không ghi đè thông tin, version hoặc tồn kho đã chỉnh.
Slug có tiền tố `demo-`, SKU có tiền tố `DEMO-`. Không xóa sản phẩm có sẵn.
Nếu slug đã bị bản ghi khác chiếm, lệnh báo lỗi thay vì ghi đè bản ghi đó.
Các bước không nằm trong một transaction; nếu thất bại giữa chừng có thể chạy lại.

File ảnh đã tải hợp lệ được dùng lại, ảnh không được commit vào Git.
Upload ảnh đồng bộ không xóa object khác trong bucket.
`make seed-fetch` chỉ tải ảnh; `go run ./cmd/seed -check` kiểm tra mà không ghi DB.
Nếu đã upload ảnh, `go run ./cmd/seed` chỉ thêm dữ liệu catalog.
Đổi domain ảnh sau khi đã seed không tự thay URL của các bản ghi cũ.

Ảnh giữ nguyên nguồn Forher và dùng làm tài nguyên tham khảo cho demo.
Trường `source_url` và `seed_source` được lưu cùng sản phẩm để đối chiếu.
