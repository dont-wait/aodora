# PD-04 — Giao diện nội dung áo và trang 403 chung

## Biên tập nội dung tại trang quản trị

Trong trang `/admin/catalog`, khi chọn một sản phẩm đã lưu, form **Câu chuyện, chất liệu & chế tác** được tải qua `GET /api/v1/admin/products/{id}/content`:

- `story`: dòng dẫn, tiêu đề, URL/alt ảnh, các đoạn văn. Admin thêm, sửa, xóa và di chuyển đoạn lên hoặc xuống để quyết định thứ tự xuất hiện.
- `fabric`: dòng dẫn, tiêu đề, mô tả, ảnh và danh sách `care[]`. Admin sửa, thêm, xóa và sắp xếp hướng dẫn bảo quản.
- `craft`: dòng dẫn, tiêu đề, mô tả, ảnh và các bước `steps[]`. Admin sửa, thêm, xóa và sắp xếp từng bước.
- Có thể xóa hẳn một phần bằng nút xóa phần tương ứng; request PUT gửi `null` cho phần bị xóa.
- Trạng thái trống không tự điền dữ liệu câu chuyện mẫu. Form yêu cầu chủ cửa hàng nhập nội dung hợp lệ.

Dữ liệu lưu theo ID sản phẩm và `expected_version` để tránh ghi đè khi nhiều cửa sổ cùng sửa. Form nội dung độc lập với form giá/SKU/tồn kho. Sau khi lưu thành công, phiên bản metadata trên màn hình được đồng bộ.

## Hiển thị công khai

Trên `/san-pham/:slug`, thành phần `InfoTabs` đọc ba phần từ phản hồi chi tiết của chính slug đó. Không có giá trị thì không tạo tab trống hoặc nội dung thay thế. Story giữ nguyên thứ tự các đoạn; bảo quản và các bước chế tác theo thứ tự mảng. Mọi chuỗi do admin nhập được hiển thị như text thông thường, không sử dụng HTML nội dung thô.

## Trang 403 chung

- `/403`: thông báo **“403 — Bạn không có quyền ở đây”** và nút về trang chủ.
- Route `/admin/*` được bảo vệ ở lớp route. Khi chưa xác minh quyền admin từ backend, không render dữ liệu/form quản trị; nếu API trả 401/403, điều hướng tới `/403`.
- Nếu backend không liên lạc được, hiển thị trạng thái lỗi với nút thử lại; không nhầm lỗi mạng thành quyền admin.
- Các endpoint API vẫn trả JSON `403` chứ không redirect HTML. Khi KR-01 hoàn thiện, lớp guard cần dùng API xác thực danh tính/quyền admin của KR-01; tuyệt đối không dựa vào localStorage hoặc role do trình duyệt tự gửi.

## Phạm vi kiểm thử

Test UI thuần kiểm tra thứ tự văn bản, dữ liệu hai sản phẩm khác nhau, yêu cầu nhập bắt buộc và body PUT chỉ gồm nội dung/phiên bản. Test tích hợp cần tài khoản KR-01 và MongoDB thật, đối chiếu sản phẩm A và B sau tải lại. Không hardcode nội dung sản phẩm trong giao diện chạy thật.
