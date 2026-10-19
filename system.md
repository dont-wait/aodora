# ĐẶC TẢ HỆ THỐNG CỬA HÀNG ÁO DÀI VIỆT NAM

**Dự án:** Aodora  
**Phạm vi:** Cửa hàng bán áo dài, bám theo cấu trúc dữ liệu hiện có.

## 1. Mục tiêu

Xây dựng website để khách xem và đặt mua áo dài Việt Nam. Bên cạnh hình ảnh, giá và chất liệu, mỗi mẫu áo được giới thiệu qua câu chuyện về cảm hứng thiết kế, nét văn hóa hoặc một thời đại đã qua.

Câu chuyện được trình bày ngay trong chi tiết sản phẩm, sử dụng `products.story`. Chủ cửa hàng trực tiếp cập nhật nội dung này cùng thông tin áo.

## 2. Đối tượng sử dụng

Hệ thống có người dùng và admin. Người dùng có thể có hoặc không có tài khoản.

| Đối tượng | Chức năng |
| --- | --- |
| Người dùng không có tài khoản | Xem danh mục, sản phẩm, câu chuyện, đánh giá; chọn áo và đặt mua; gửi yêu cầu may đo |
| Người dùng có tài khoản | Các chức năng trên; quản lý thông tin cá nhân, địa chỉ, giỏ hàng, yêu thích và xem đơn hàng của mình |
| Admin — chủ cửa hàng | Quản lý danh mục, sản phẩm, câu chuyện, tồn kho, đơn hàng, thanh toán, yêu cầu may đo và đánh giá |

Khách hàng đã mua áo có thể đánh giá sản phẩm sau khi nhận áo. Việc không có tài khoản không cản trở đặt mua.

## 3. Chức năng người dùng

### 3.1. Xem và tìm áo

- Xem danh mục áo dài, danh sách sản phẩm và bộ sưu tập.
- Tìm theo tên hoặc mô tả sản phẩm; lọc theo danh mục, giá, màu, kích cỡ và chất liệu có trong dữ liệu sản phẩm.
- Xem ảnh, tên, giá, chất liệu, màu, kích cỡ, tình trạng hàng và phần đi kèm.
- Xem câu chuyện, hướng dẫn bảo quản, quy trình làm áo và đánh giá của người mua.

### 3.2. Câu chuyện của áo

Nội dung câu chuyện giải thích ý tưởng của mẫu áo: gợi nhớ thời đại nào, nét sinh hoạt nào, hoặc ý nghĩa của phom dáng và họa tiết. Nếu là thiết kế lấy cảm hứng, nội dung cần diễn đạt đúng là lấy cảm hứng, tránh khẳng định đây là trang phục nguyên gốc của một thời kỳ khi chưa có căn cứ.

| Nội dung | Field hiện có |
| --- | --- |
| Dòng giới thiệu | `products.story.eyebrow` |
| Tiêu đề câu chuyện | `products.story.title` |
| Các đoạn kể chuyện | `products.story.paragraphs[]` |
| Ảnh minh họa | `products.story.image` |
| Chất liệu và bảo quản | `products.fabric` |
| Quy trình làm áo | `products.craft` |

Câu chuyện là nội dung thuộc sản phẩm, do admin nhập và sửa trực tiếp.

### 3.3. Chọn áo và giỏ hàng

- Khách chọn màu, kích cỡ, số lượng và thêm áo vào giỏ.
- Khách có thể sửa số lượng hoặc bỏ sản phẩm trước khi đặt.
- Hệ thống kiểm tra giá, tình trạng bán và tồn kho trước khi tạo đơn.
- Giỏ hàng của tài khoản được lưu bằng `carts`. Giỏ của khách chưa đăng nhập được giữ tại trình duyệt; DB hiện bắt buộc `carts.customer_id`.

### 3.4. Đặt mua áo

1. Khách chọn áo và kiểm tra giỏ hàng.
2. Khách nhập họ tên, số điện thoại, địa chỉ nhận hàng nếu giao tận nơi và ghi chú nếu cần.
3. Khách chọn thanh toán khi giao hàng hoặc thanh toán tại quầy.
4. Hệ thống hiển thị danh sách áo, số lượng, tiền hàng, phí giao và tổng tiền.
5. Khách xác nhận đặt mua; hệ thống lưu đơn và trả thông tin xác nhận.
6. Admin kiểm tra, xác nhận đơn và xử lý giao áo hoặc giao trực tiếp tại cửa hàng.

Khách có tài khoản được gắn đơn với tài khoản. Khách không có tài khoản vẫn đặt được bằng thông tin liên hệ đã nhập; không yêu cầu tạo tài khoản mới để hoàn tất mua hàng.

### 3.5. Thanh toán

| Hình thức | Cách xử lý |
| --- | --- |
| Thanh toán khi giao hàng | Khách trả tiền khi nhận áo; admin cập nhật đã thanh toán sau khi xác nhận thu được tiền |
| Thu tại quầy | Khách thanh toán trực tiếp tại cửa hàng; admin ghi nhận đã thanh toán khi thực tế nhận tiền |

Đơn mới có `payment_status = unpaid`. Chọn hình thức thanh toán hoặc xác nhận đặt hàng không đồng nghĩa với đã thanh toán. Khi thu đủ tiền, admin cập nhật `payment_status = paid`.

Đơn thu tại quầy và giao trực tiếp không cần đi qua trạng thái đang giao hàng. Thông tin địa chỉ bắt buộc hiện tại của DB cần được điều chỉnh cho trường hợp này, không nhập địa chỉ giả để vượt kiểm tra.

### 3.6. Yêu cầu may đo

Chức năng này dùng collection `tailoring_requests` đã có:

- Khách nhập họ tên, số điện thoại, mẫu áo quan tâm, số đo nếu có và ghi chú.
- Khách chưa đăng nhập vẫn gửi được yêu cầu.
- Admin tiếp nhận, liên hệ, ghi lịch hẹn nếu cần và cập nhật tình trạng xử lý.
- Đơn may đo có thể dùng `orders.fulfillment_mode = custom`, liên kết `tailoring_request_id` và lưu số đo bằng các field hiện có.

Các số đo hiện có gồm chiều cao/cân nặng dạng mô tả (`body`), vòng cổ, ngực, eo, mông, vai, tay và chiều dài áo. Admin kiểm tra lại thông tin với khách trước khi thực hiện.

### 3.7. Đánh giá sản phẩm

- Khách đã nhận áo có thể đánh giá từ 1 đến 5 sao, viết nhận xét và đính kèm ảnh.
- Hệ thống kiểm tra khách đã mua đúng sản phẩm trước khi ghi nhận đánh giá mua hàng xác thực.
- Một khách có một đánh giá cho một sản phẩm, theo unique index hiện có trên `customer_id` và `product_id`; gửi lại sẽ cập nhật đánh giá của mình.
- Người chưa mua vẫn xem được đánh giá nhưng không được ghi nhận là người mua xác thực.
- Với khách mua không có tài khoản, cần xác minh đúng đơn mua trước khi cho đánh giá; không tự gắn đánh giá với tài khoản khác chỉ vì trùng tên hoặc số điện thoại nhập vào.

## 4. Chức năng admin — chủ cửa hàng

| Nhóm | Công việc |
| --- | --- |
| Danh mục | Thêm, sửa, sắp xếp và bật/tắt danh mục |
| Sản phẩm | Quản lý tên, giá, hình ảnh, chất liệu, màu, cỡ, SKU, tồn kho và tình trạng bán |
| Nội dung áo | Cập nhật câu chuyện, chất liệu, bảo quản và quy trình làm áo ngay trong sản phẩm |
| Đơn hàng | Xem chi tiết, kiểm tra thông tin khách, xác nhận, cập nhật giao/nhận và hủy đơn |
| Thanh toán | Xem phương thức thu tiền, cập nhật chưa thanh toán/đã thanh toán; ghi nhận hoàn tiền nếu phát sinh |
| May đo | Xem yêu cầu, số đo, ghi chú, lịch hẹn và cập nhật trạng thái |
| Khách hàng | Xem thông tin khách và đơn liên quan phục vụ xử lý mua hàng |
| Đánh giá | Xem và xử lý đánh giá không phù hợp; dữ liệu hiện chưa có trạng thái duyệt/ẩn riêng |

Admin là chủ cửa hàng; các công việc trên không chia thành vai trò nhân viên riêng.

## 5. Trạng thái theo DB hiện có

### 5.1. Đơn hàng — `orders.status`

| Giá trị | Ý nghĩa |
| --- | --- |
| `pending` | Đã đặt, chờ chủ cửa hàng xác nhận |
| `confirmed` | Chủ cửa hàng đã xác nhận |
| `shipping` | Đang giao áo cho khách |
| `delivered` | Khách đã nhận áo, gồm cả nhận trực tiếp tại cửa hàng |
| `cancelled` | Đơn bị hủy |

Luồng giao hàng: `pending → confirmed → shipping → delivered`.

Luồng nhận trực tiếp: `pending → confirmed → delivered`.

Admin có thể hủy đơn chưa giao xong và xử lý lại tồn kho tương ứng. Trạng thái nhận hàng và trạng thái thanh toán được cập nhật riêng, theo thực tế.

### 5.2. Thanh toán — `orders.payment_status`

| Giá trị | Ý nghĩa |
| --- | --- |
| `unpaid` | Chưa thu tiền |
| `paid` | Đã thu tiền |
| `refunded` | Đã hoàn tiền |

### 5.3. May đo — `tailoring_requests.status`

Giữ các giá trị hiện có: `submitted` (đã gửi), `contacted` (đã liên hệ), `consultation_scheduled` (đã hẹn tư vấn), `measurement_confirmed` (đã xác nhận số đo), `in_progress` (đang thực hiện), `completed` (hoàn tất), `cancelled` (đã hủy).

## 6. Dữ liệu sử dụng

| Collection | Nội dung phục vụ hệ thống |
| --- | --- |
| `categories` | Danh mục áo, mô tả, hình ảnh, thứ tự và trạng thái hiển thị |
| `products` | Áo, biến thể, giá, tồn kho, câu chuyện, chất liệu, quy trình, phụ kiện và thông tin đánh giá tổng hợp |
| `customers` | Tài khoản khách, thông tin liên hệ, địa chỉ, số đo và sản phẩm yêu thích |
| `carts` | Giỏ hàng của khách có tài khoản |
| `orders` | Thông tin khách đặt, các áo đã chọn, giá tại thời điểm mua, tổng tiền, trạng thái và phương thức thanh toán |
| `reviews` | Điểm, nội dung, ảnh và thông tin người đánh giá sản phẩm |
| `tailoring_requests` | Yêu cầu may đo, liên hệ, số đo, ghi chú, lịch hẹn và trạng thái xử lý |

## 7. Quy tắc nghiệp vụ

1. Khách có hoặc không có tài khoản đều được đặt áo.
2. Chỉ sản phẩm đang bán và còn đủ tồn kho mới được đặt theo hình thức may sẵn; kiểm tra theo SKU màu/cỡ đã chọn.
3. Số lượng mua là số nguyên dương. Hệ thống tính lại tiền từ dữ liệu sản phẩm, không tin tổng tiền do trình duyệt gửi.
4. Tiền hàng (`subtotal`) bằng tổng số lượng nhân đơn giá từng dòng; tổng đơn (`total`) bằng tiền hàng cộng phí giao (`shipping_fee`). Nhận tại quầy có phí giao bằng 0.
5. Đơn giữ tên áo, SKU, lựa chọn và đơn giá trong `orders.items`; thay đổi thông tin sản phẩm về sau không sửa giá đã đặt.
6. Việc tạo đơn và cập nhật tồn kho phải nhất quán; hủy đơn không hoàn tồn nhiều lần cho cùng một lần trừ.
7. Khách chỉ được xem dữ liệu riêng của mình; thông tin đơn và số đo không công khai cùng trang sản phẩm.
8. Admin chỉ đánh dấu đã thanh toán khi đã thu tiền. Đã nhận áo không tự động có nghĩa đã thu tiền.
9. Khách đã nhận sản phẩm mới được đánh giá trải nghiệm mua; hệ thống kiểm tra đơn tương ứng trước khi xác nhận mua hàng.
10. Điểm và số lượt đánh giá hiển thị trên sản phẩm được tổng hợp từ `reviews`.

## 8. Các điểm DB chưa khớp yêu cầu

Các mục dưới đây là chênh lệch có thật trong migration hiện tại. Tài liệu ghi nhận để triển khai đúng yêu cầu; lượt sửa này chỉ thay specs, chưa thay schema hoặc code.

| Yêu cầu | DB hiện tại | Điều chỉnh tối thiểu cần có khi triển khai |
| --- | --- | --- |
| Khách không có tài khoản đặt hàng | `orders.customer_id` bắt buộc là ObjectId; `customers` bắt buộc có `email` và `password_hash` | Cho đơn không có tài khoản được thiếu/null `customer_id`; giữ thông tin người nhận/liên hệ của đơn, không tạo email hoặc mật khẩu giả |
| Thu tiền tại quầy | `payment_method` chỉ nhận `cod`, `bank_transfer`, `card`, `e_wallet` | Giữ `cod` cho thu khi giao và bổ sung giá trị `cash_at_counter` cho thu tại quầy; giao diện chỉ cung cấp hai lựa chọn đã yêu cầu |
| Nhận áo tại quầy | `shipping_address` bắt buộc có `recipient`, `phone`, `line1`, `city` | Giữ tên/số điện thoại liên hệ; cho phép thiếu địa chỉ giao trong đơn nhận trực tiếp tại quầy |
| Khách không có tài khoản đánh giá sau mua | `reviews.customer_id` bắt buộc và unique theo khách/sản phẩm; chưa có liên kết đơn trên review | Cần liên kết đánh giá với đơn đã xác minh cho khách vãng lai; giữ nguyên cơ chế đánh giá theo tài khoản cho khách đăng nhập |
| Admin đăng nhập | Chưa có collection admin hoặc field phân quyền trong schema khách hàng | Cần xác thực riêng cho chủ cửa hàng khi triển khai; không coi mọi tài khoản trong `customers` là admin |

Câu chuyện dùng trực tiếp `products.story`; yêu cầu may đo dùng `tailoring_requests`. Các giá trị trạng thái đơn, thanh toán và may đo giữ theo DB như mục 5.

## 9. Tiêu chí nghiệm thu

| Mã | Kết quả cần đạt |
| --- | --- |
| AC-01 | Người chưa đăng nhập xem áo, chọn mua và tạo đơn bằng thông tin liên hệ, không bị buộc đăng ký |
| AC-02 | Người có tài khoản đặt được áo và xem đúng đơn của mình |
| AC-03 | Chi tiết áo hiển thị câu chuyện, chất liệu và quy trình từ các field của sản phẩm |
| AC-04 | Khách chọn được thanh toán khi giao hoặc thu tại quầy; đơn mới vẫn là chưa thanh toán |
| AC-05 | Admin xác nhận thu tiền thì đơn chuyển sang đã thanh toán; thay đổi trạng thái giao hàng không tự ghi nhận tiền |
| AC-06 | Không đặt vượt tồn kho SKU; tổng tiền đúng số lượng, đơn giá và phí giao |
| AC-07 | Khách đã nhận áo đánh giá được đúng sản phẩm; người khác không đánh giá thay đơn chưa xác minh |
| AC-08 | Một tài khoản gửi lại đánh giá cùng sản phẩm không tạo thêm đánh giá trùng |
| AC-09 | Khách chưa đăng nhập gửi được yêu cầu may đo; admin xem và cập nhật đúng trạng thái hiện có |
| AC-10 | Admin quản lý được danh mục, áo, nội dung câu chuyện, đơn và thanh toán trong cùng phạm vi chủ cửa hàng |

## 10. Căn cứ đối chiếu

- [Schema gốc và index](server/migrations/000002_clothing_store.up.json).
- [Schema mở rộng cho client và may đo](server/migrations/000004_client_fields.up.json).
- [Routes client hiện có](client/src/app/App.tsx).
- [Router server hiện có](server/internal/interfaces/http/router.go).

Client hiện sử dụng dữ liệu mẫu; router server hiện chỉ có health/readiness. Tài liệu này đặc tả chức năng cần thực hiện dựa trên DB, không xác nhận các luồng bán hàng đã chạy hoàn chỉnh.
