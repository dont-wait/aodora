# PD-03 — Giao diện quản trị catalog Aodora

Giao diện quản trị ở `/admin/catalog` sử dụng dữ liệu MongoDB thông qua API có kiểm tra quyền admin ở server. Không có danh sách sản phẩm, giá, danh mục hoặc ảnh mẫu gán cố định trong client.

- **Danh mục:** chọn, tạo mới, chỉnh tên, slug, mô tả, danh mục cha, thứ tự, trạng thái hiển thị và ảnh (`src`/`alt`).
- **Sản phẩm:** chọn, tạo, sửa thông tin bán hàng, giá, chất liệu, trạng thái, ảnh chính, gallery, mã màu và phần đi kèm.
- **SKU:** tạo sản phẩm có nhiều SKU; với sản phẩm đã có, chỉ bổ sung SKU mới có tồn 0 và sửa tên màu/cỡ của SKU hiện hữu.
- **Tồn kho:** tăng/giảm từng SKU bằng chênh lệch, đồng thời cập nhật stock tổng trên server. Không lấy số tồn trong form cũ để ghi đè.
- **Thông báo:** hiển thị lỗi xác thực và validation từ API. Khi chưa có xác thực từ KR-01, trang sẽ không hiển thị dữ liệu quản trị.

Mã màu là mảng `{name, hex?}` (`hex` có thể trống); phụ kiện gồm `id`, `name`, `description`, `price`, `image?`, `tag?`, `tag_variant?`. Ảnh phụ gồm các dòng `URL | mô tả ảnh`. Giá nhập bằng số nguyên VNĐ. Những dữ liệu này được kiểm tra lại tại backend; frontend không quyết định quyền admin.

**Phụ thuộc:** KR-01 xác minh phiên admin; Q-07 phối hợp Dontwait về giảm/hoàn tồn khi chốt hoặc hủy đơn; Q-08 phối hợp MyNameIsKry về nguồn đường dẫn ảnh và mô tả. Chức năng upload ảnh chưa được triển khai vì schema hiện lưu URL.

### Bảo toàn metadata và đồng thời

Dữ liệu legacy trong `images[]` được liệt kê vào gallery khi mở sửa, kể cả URL chưa có trong `gallery` hoặc `image`. Sửa giá/tên không tự xóa ảnh; admin chủ động bỏ dòng URL khi muốn xóa. Giá so sánh hiển thị fallback từ `old_price` cũ và sau khi lưu server xóa `old_price` để tránh còn giảm giá không đúng.

Form `PUT` gửi `expected_version` nhận từ server để phát hiện hai tab thay đổi cùng sản phẩm/danh mục (`409 CONFLICT` yêu cầu tải lại). `is_active` và các mảng metadata được gửi đầy đủ, không mặc định xóa khi body thiếu field. Mọi request tải trang và làm mới được kiểm soát thứ tự, phản hồi cũ không ghi đè kết quả trang mới.

Tồn kho yêu cầu `request_id` UUID mới cho thao tác mới; nếu gửi thất bại hoặc mất phản hồi, nút thử lại giữ nguyên UUID cho cùng giá trị delta, không làm thay đổi tồn hai lần. Khi sửa delta sẽ tạo UUID mới. Chưa hỗ trợ khôi phục ID qua reload trang; khi không rõ kết quả cần xem lại tồn trước khi bấm thao tác khác.

Nếu frontend khác origin với API, phía server cần `ADMIN_ALLOWED_ORIGIN` trùng origin của web và cấp CORS có credentials. Với Vite proxy same-origin, cấu hình proxy target tới đúng backend Go. Tất cả phụ thuộc KR-01 xác thực admin thật để sử dụng CRUD; kết quả build/unit test không thay thế kiểm thử quyền truy cập end-to-end.
