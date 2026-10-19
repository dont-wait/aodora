# Phân công triển khai Aodora

Căn cứ: [system.md](system.md). Mỗi người phụ trách cả API và giao diện của phần việc được giao, từ dữ liệu đến luồng sử dụng thực tế.

## 1. Tổng quan

| Người phụ trách | Phạm vi chính | Dữ liệu chính |
| --- | --- | --- |
| `pdmh18` | Danh mục, sản phẩm, câu chuyện và màn hình quản trị tương ứng | `categories`, `products` |
| `MyNameIsKry` | Tài khoản, giỏ hàng, yêu thích và đánh giá sau mua | `customers`, `carts`, `reviews` |
| `Dontwait` | Đặt hàng, thanh toán, may đo và màn hình quản trị tương ứng | `orders`, `tailoring_requests` |

Các task bên dưới đều ở trạng thái **chưa thực hiện**. Phân công này là kế hoạch trong repository.

## 2. Task của pdmh18

| Mã | Công việc | Tiêu chí hoàn thành | Phụ thuộc |
| --- | --- | --- | --- |
| PD-01 | API danh mục và sản phẩm cho khách | Danh sách, chi tiết theo slug, tìm tên/mô tả, lọc theo các field trong specs, sắp xếp và phân trang; trả tổng kết quả thật và biến thể màu/cỡ/tồn kho | Không |
| PD-02 | Nối API vào giao diện cửa hàng | Trang chủ, bộ sưu tập và chi tiết dùng dữ liệu thật; mở đúng mẫu theo slug; hiển thị trạng thái tải, rỗng và lỗi | PD-01 |
| PD-03 | API và giao diện admin quản lý danh mục/sản phẩm | Thêm, sửa, bật/tắt; quản lý ảnh, giá, màu, cỡ, SKU và tồn kho; API kiểm tra quyền admin | PD-01, KR-01 |
| PD-04 | Nội dung câu chuyện của áo | Admin sửa trực tiếp `story`, `fabric`, `craft` trong sản phẩm; khách đọc đúng nội dung của mẫu; giữ câu chuyện trong `products.story` | PD-02, PD-03 |

**Bàn giao:** API đọc catalog cho giỏ/đặt hàng; API quản trị và giao diện sản phẩm; tài liệu request/response có ví dụ. Đối chiếu AC-03 và phần catalog của AC-10 trong specs.

## 3. Task của MyNameIsKry

| Mã | Công việc | Tiêu chí hoàn thành | Phụ thuộc |
| --- | --- | --- | --- |
| KR-01 | Tài khoản khách và xác thực admin | Đăng ký/đăng nhập khách; admin đăng nhập với quyền chủ cửa hàng; có kiểm tra quyền ở server; thống nhất cách xác thực để hai người còn lại dùng | Không |
| KR-02 | Hồ sơ khách và yêu thích | Khách cập nhật thông tin, địa chỉ, số đo và danh sách yêu thích theo field hiện có; chỉ truy cập dữ liệu của mình | KR-01, PD-01 cho chọn sản phẩm |
| KR-03 | Giỏ hàng có/không tài khoản | Giỏ tài khoản lưu `carts`; giỏ khách lưu tại trình duyệt; chọn đúng SKU, sửa lượng, xóa dòng; cung cấp dữ liệu giỏ cho màn hình đặt hàng | PD-01, KR-01 cho giỏ tài khoản |
| KR-04 | Đánh giá sau mua và quản lý đánh giá | Xem đánh giá công khai; khách đã nhận áo gửi/sửa điểm 1–5, nhận xét và ảnh; kiểm tra quyền đánh giá; hỗ trợ khách vãng lai qua đơn đã xác minh; cập nhật điểm/số đánh giá sản phẩm; admin xem và xử lý đánh giá không phù hợp | KR-01, DW-02, DW-03 |

**Bàn giao:** cơ chế xác thực dùng chung; giao diện tài khoản/giỏ/đánh giá; API tương ứng và migration của `customers`, `carts`, `reviews` nếu cần. Đối chiếu AC-07, AC-08 và phần tài khoản của AC-02.

Với KR-04, giữ unique khách/sản phẩm cho tài khoản. Cùng Dontwait chốt cách xác minh đơn của khách vãng lai và liên kết review trước khi sửa schema; không dùng tên/số điện thoại tự nhập làm bằng chứng duy nhất, không tạo tài khoản giả.

## 4. Task của Dontwait

| Mã | Công việc | Tiêu chí hoàn thành | Phụ thuộc |
| --- | --- | --- | --- |
| DW-01 | Điều chỉnh schema đơn hàng theo specs | Hỗ trợ `customer_id` thiếu/null cho khách vãng lai; thêm `cash_at_counter`; nhận tại quầy không cần địa chỉ giao nhưng vẫn có tên/số điện thoại; migration có up/down | Không |
| DW-02 | API đặt hàng và xử lý đơn | Khách có/không tài khoản đều đặt được; tính lại giá/tổng tiền, kiểm tra và cập nhật tồn kho SKU nhất quán; hủy không hoàn tồn lặp; lưu thông tin dòng đơn tại thời điểm mua; kiểm tra quyền xem đơn | DW-01, PD-01, KR-01 cho đơn tài khoản |
| DW-03 | Giao diện đặt hàng, đơn của khách và quản trị đơn | Checkout từ giỏ; chỉ hai cách thanh toán COD/thu tại quầy; trang xác nhận, danh sách/chi tiết đơn tài khoản; admin cập nhật đơn và thu tiền riêng biệt; khách vãng lai có cách xác minh đơn để đánh giá | DW-02, KR-03; KR-01 cho admin |
| DW-04 | API và giao diện yêu cầu may đo | Form hiện có lưu thật vào `tailoring_requests`, khách vãng lai gửi được; admin xem, liên hệ, cập nhật số đo/lịch hẹn và trạng thái hiện có; đơn may đo lưu liên kết và số đo theo specs | KR-01 cho admin, DW-02 cho đơn may đo |

**Bàn giao:** migration đơn hàng; API và giao diện đặt hàng/thu tiền/may đo; cách xác minh đơn cho KR-04; bằng chứng kiểm tra tồn kho và tổng tiền. Đối chiếu AC-01, AC-02, AC-04, AC-05, AC-06, AC-09 và phần đơn/thanh toán của AC-10.

## 5. Thứ tự và phối hợp

### Bước 1 — Thống nhất dữ liệu giao tiếp

Ba người thống nhất request/response trước khi nối giao diện:

- `pdmh18` cung cấp ID sản phẩm, slug, SKU, lựa chọn màu/cỡ, giá và tồn kho theo DB.
- `MyNameIsKry` cung cấp cách xác thực khách/admin và cấu trúc dữ liệu giỏ cho checkout.
- `Dontwait` cung cấp cấu trúc tạo/đọc đơn, hai cách thanh toán, nhận tại quầy và xác minh đơn khách vãng lai.

Chỉ điều chỉnh DB cho những chênh lệch đã nêu trong mục 8 của `system.md`. Những nội dung cần thống nhất ở đây là cách triển khai yêu cầu đã có.

### Bước 2 — Làm phần nền song song

`pdmh18` làm PD-01; `MyNameIsKry` làm KR-01; `Dontwait` làm DW-01 và phần xử lý đơn độc lập. Sau đó mỗi người tiếp tục các task trong phạm vi của mình theo cột phụ thuộc. Có thể làm giao diện theo hợp đồng đã thống nhất trong lúc chờ API, nhưng chỉ tính hoàn thành sau khi nối dữ liệu thật.

### Bước 3 — Ghép luồng và nghiệm thu

| Luồng kiểm tra | Người thực hiện chính | Người phối hợp |
| --- | --- | --- |
| Xem áo và câu chuyện → thêm đúng biến thể vào giỏ | `pdmh18` | `MyNameIsKry` |
| Khách vãng lai → đặt COD → admin giao và xác nhận thu tiền | `Dontwait` | `MyNameIsKry` |
| Khách tài khoản → đặt nhận tại quầy → xem đơn → admin thu tiền | `Dontwait` | `MyNameIsKry` |
| Đã nhận áo → đánh giá; chưa nhận/không xác minh → bị từ chối | `MyNameIsKry` | `Dontwait` |
| Gửi yêu cầu may đo → admin cập nhật xử lý | `Dontwait` | `MyNameIsKry` hỗ trợ quyền |
| Admin sửa câu chuyện/tồn kho → khách thấy đúng dữ liệu mới | `pdmh18` | `Dontwait` kiểm tra mua theo tồn kho mới |

Mỗi người cung cấp kết quả kiểm tra phần mình; Dontwait tổng hợp theo AC-01 đến AC-10 của specs. Kiểm tra cả hai khách cùng mua SKU còn một chiếc và hủy đơn lặp để xác nhận tồn kho đúng.

## 6. Quy ước bàn giao

- Mỗi PR ghi mã task, chức năng đã chạy và cách kiểm tra; giữ phạm vi theo `system.md`.
- Migration dùng số thứ tự riêng đã thống nhất để không trùng tên. Ai sửa collection nào chịu trách nhiệm migration tương ứng; thay đổi liên quan nhiều collection cần trao đổi trước.
- `products.rating` và `review_count` do phần đánh giá cập nhật; form sửa sản phẩm không ghi đè bằng dữ liệu tổng hợp cũ.
- Tồn kho: `pdmh18` phụ trách thao tác điều chỉnh của admin; `Dontwait` phụ trách thay đổi do đặt/hủy đơn. Hai bên thống nhất quy tắc cập nhật để không ghi đè lẫn nhau.
- Các file chung như router server, route client và kiểu dữ liệu dùng chung cần ghép thay đổi có phối hợp.
- Chạy build/lint client và test Go phù hợp với phần thay đổi. Các luồng tài khoản, đơn, thanh toán và dữ liệu riêng phải kiểm tra với API/DB thực tế; thông báo giả lập chưa đạt tiêu chí hoàn thành.
