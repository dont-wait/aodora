# PD-03 — Giao diện quản trị catalog Aodora

Giao diện quản trị ở `/admin/catalog` sử dụng dữ liệu MongoDB thông qua API có kiểm tra quyền admin ở server. Không có danh sách sản phẩm, giá, danh mục hoặc ảnh mẫu gán cố định trong client.

- **Danh mục:** chọn, tạo mới, chỉnh tên, slug, mô tả, danh mục cha, thứ tự, trạng thái hiển thị và ảnh (`src`/`alt`).
- **Sản phẩm:** chọn, tạo, sửa thông tin bán hàng, giá, chất liệu, trạng thái, ảnh chính, gallery, mã màu và phần đi kèm.
- **SKU:** tạo sản phẩm có nhiều SKU; với sản phẩm đã có, chỉ bổ sung SKU mới có tồn 0 và sửa tên màu/cỡ của SKU hiện hữu.
- **Tồn kho:** tăng/giảm từng SKU bằng chênh lệch, đồng thời cập nhật stock tổng trên server. Không lấy số tồn trong form cũ để ghi đè.
- **Thông báo:** hiển thị lỗi xác thực và validation từ API. Khi chưa có xác thực từ KR-01, trang sẽ không hiển thị dữ liệu quản trị.

Mã màu được nhập dưới dạng mảng `{name, hex}`; phụ kiện gồm `id`, `name`, `description`, `price`, `image: {src, alt}`. Ảnh phụ gồm các dòng `URL | mô tả ảnh`. Giá nhập bằng số nguyên VNĐ. Những dữ liệu này được kiểm tra lại tại backend; frontend không quyết định quyền admin.

**Phụ thuộc:** KR-01 xác minh phiên admin; Q-07 phối hợp Dontwait về giảm/hoàn tồn khi chốt hoặc hủy đơn; Q-08 phối hợp MyNameIsKry về nguồn đường dẫn ảnh và mô tả. Chức năng upload ảnh chưa được triển khai vì schema hiện lưu URL.
