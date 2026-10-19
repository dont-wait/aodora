# Backlog triển khai hệ thống cửa hàng áo dài Aodora

**Căn cứ nghiệp vụ:** [system.md](system.md).

**Trạng thái:** Các task đều chưa thực hiện. Đây là phân công trong repository, chưa phải issue đã tạo hoặc công việc đã giao qua nền tảng bên ngoài.

**Nguyên tắc:** Mỗi người phụ trách API, xử lý nghiệp vụ, giao diện và kiểm tra tích hợp của phần việc được giao. Các tên API/URL sẽ thống nhất khi triển khai; tài liệu này quy định hành vi và dữ liệu cần đáp ứng.

## 1. Phạm vi và trách nhiệm

| Người | Phạm vi chịu trách nhiệm | Collection chính | Kết quả cuối cùng |
| --- | --- | --- | --- |
| `pdmh18` | Danh mục, sản phẩm, câu chuyện và quản trị nội dung áo | `categories`, `products` | Khách tìm và xem đúng mẫu; chủ cửa hàng quản lý được dữ liệu áo |
| `MyNameIsKry` | Tài khoản, quyền truy cập, hồ sơ, giỏ, yêu thích và review | `customers`, `carts`, `reviews` | Khách đăng nhập hoặc mua không tài khoản; người đã nhận áo đánh giá được |
| `Dontwait` | Đặt hàng, tồn kho do giao dịch, thu tiền, may đo và quản trị đơn | `orders`, `tailoring_requests` | Khách đặt được áo; chủ cửa hàng giao áo, thu tiền và xử lý may đo |

Giữ nguyên hai nhóm quyền: người dùng và admin là chủ cửa hàng. Câu chuyện nằm trong `products.story`. Phương thức thanh toán trong giao diện chỉ gồm khi giao hàng và thu tại quầy. Không bổ sung chức năng ngoài `system.md` vào các task này.

### 1.1. Danh sách task

| Mã | Người phụ trách | Đầu việc | Phụ thuộc để nghiệm thu |
| --- | --- | --- | --- |
| PD-01 | pdmh18 | API đọc danh mục và sản phẩm | Không |
| PD-02 | pdmh18 | Giao diện tìm và xem áo dùng dữ liệu thật | PD-01 |
| PD-03 | pdmh18 | Admin quản lý danh mục và thông tin bán áo | PD-01, KR-01 |
| PD-04 | pdmh18 | Admin biên soạn và khách đọc câu chuyện của áo | PD-02, PD-03 |
| KR-01 | MyNameIsKry | Tài khoản khách và xác thực admin | Không |
| KR-02 | MyNameIsKry | Hồ sơ, địa chỉ, số đo và yêu thích | KR-01, PD-01 |
| KR-03 | MyNameIsKry | Giỏ khách vãng lai và giỏ tài khoản | PD-01, KR-01 |
| KR-04 | MyNameIsKry | Review sau nhận áo và quản lý review | KR-01, DW-02, DW-03 |
| DW-01 | Dontwait | Schema đơn hàng theo yêu cầu đã chốt | Các quyết định DB tại mục 5 |
| DW-02 | Dontwait | Tạo đơn, tính tiền, cập nhật tồn và xử lý trạng thái | DW-01; dữ liệu PD-01; xác thực KR-01 cho tài khoản |
| DW-03 | Dontwait | Checkout, xem đơn, admin giao hàng và thu tiền | DW-02, KR-03, KR-01 |
| DW-04 | Dontwait | Gửi và xử lý may đo | KR-01 cho admin; DW-02 cho đơn may đo |

“Phụ thuộc để nghiệm thu” không có nghĩa phải chờ mới bắt đầu viết code. Có thể phát triển theo dữ liệu giao tiếp đã thống nhất, nhưng chỉ hoàn thành khi chạy được với API/DB thật.

## 2. Task của pdmh18

### PD-01 — API đọc danh mục và sản phẩm

**Mục tiêu người dùng:** Tôi muốn tìm áo theo nhu cầu và xem đúng thông tin mẫu trước khi chọn mua.

**Căn cứ:** Specs 3.1, 3.2, 6; hỗ trợ AC-03, AC-06.

**Phụ thuộc:** Không; cung cấp dữ liệu cho PD-02, KR-02, KR-03 và DW-02.

**Dữ liệu đầu vào:** Từ khóa tên/mô tả; danh mục; khoảng giá; màu; kích cỡ; chất liệu; cách sắp xếp; trang và kích thước trang; slug khi xem chi tiết.

**Việc cần thực hiện:**

- Đọc danh mục có `is_active = true`, theo thứ tự hiển thị hiện có.
- Đọc danh sách sản phẩm đang bán; áp dụng đồng thời các điều kiện lọc. Tìm kiếm chỉ dựa trên field hiện có, không thêm bộ lọc thời đại riêng.
- Trả dữ liệu danh sách đủ cho card: ID, slug, tên, ảnh, giá, chất liệu, nhãn, màu/cỡ và khả năng mua/may đo.
- Trả chi tiết đúng slug, gồm SKU/biến thể/tồn kho, `story`, `fabric`, `craft`, phần đi kèm và đánh giá tổng hợp.
- Tính tổng kết quả sau lọc trước khi phân trang; sắp xếp ổn định để chuyển trang không lặp hoặc mất sản phẩm do thứ tự không xác định.
- Chuẩn hóa cách dùng `stock` tổng và `variants[].stock` với PD-03/DW-02. Nếu có biến thể, kiểm tra lựa chọn màu/cỡ trong cùng một biến thể phù hợp; không ghép màu của SKU này với cỡ của SKU khác.

**Ngoại lệ:** Slug sai hoặc mẫu tắt bán không trả chi tiết bán hàng công khai; tham số sai trả lỗi có thể hiểu được; không có kết quả trả danh sách rỗng với tổng bằng 0.

**Nghiệm thu:**

- [ ] Tìm theo tên/mô tả trả đúng các mẫu khớp; lọc kết hợp danh mục, giá, màu và cỡ trả đúng kết quả.
- [ ] Có thể mở trực tiếp URL mẫu B và nhận dữ liệu B, không phải mẫu mặc định A.
- [ ] Số dòng và tổng kết quả khớp DB; không có số trang/tổng sản phẩm ghi cố định.
- [ ] Sản phẩm hoặc danh mục tắt hiển thị được xử lý nhất quán theo quy tắc đã thống nhất.
- [ ] Response công khai không chứa thông tin khách, đơn hoặc số đo.

**Bàn giao:** API danh sách/chi tiết; ví dụ request/response và lỗi; bộ dữ liệu minh họa có nhiều màu/cỡ, hết hàng, tắt bán và nhiều trang.

### PD-02 — Giao diện tìm và xem áo dùng dữ liệu thật

**Mục tiêu người dùng:** Tôi có thể duyệt từ trang chủ đến bộ sưu tập và mẫu áo, chọn được mẫu mình muốn mua.

**Căn cứ:** Specs 3.1–3.3; hỗ trợ AC-01, AC-03, AC-06.

**Phụ thuộc:** PD-01; phối hợp KR-03 ở nút thêm giỏ.

**Màn hình liên quan:** Trang chủ, bộ sưu tập, chi tiết sản phẩm; tận dụng các trang/component hiện có trong `client/src/pages`.

**Việc cần thực hiện:**

- Thay dữ liệu catalog mẫu trên các màn hình bằng dữ liệu từ PD-01; không để thông tin list và detail thuộc hai nguồn khác nhau.
- Nối bộ lọc, tìm kiếm, sắp xếp và phân trang; thay điều kiện đưa về trang đầu.
- Hiển thị đúng ảnh, giá, mô tả, màu/cỡ, tồn kho và lựa chọn may sẵn/may đo của từng mẫu.
- Chọn màu/cỡ phải xác định được SKU thực; không cho gửi lựa chọn hết hàng hoặc tổ hợp không tồn tại.
- Bàn giao lựa chọn sản phẩm/SKU/số lượng sang KR-03; lựa chọn may đo chuyển sang form DW-04 với đúng mẫu quan tâm.
- Có trạng thái đang tải, không có kết quả, tải lỗi và mẫu không tồn tại; không báo thành công chỉ bằng toast khi chưa lưu giỏ.

**Nghiệm thu:**

- [ ] Click card nào mở đúng mẫu đó; tải lại trang chi tiết vẫn hiển thị đúng.
- [ ] Thay bộ lọc cập nhật đúng danh sách, tổng và trang; danh sách rỗng có hướng dẫn điều chỉnh lọc.
- [ ] Chọn SKU hết hàng không thêm được vào giỏ; SKU còn hàng chuyển đúng lựa chọn và lượng sang giỏ.
- [ ] Lỗi tải dữ liệu hiển thị rõ và có thể thử lại, không thay bằng nội dung mẫu sai.
- [ ] Hoàn thành các thao tác xem/lọc/chọn áo trên điện thoại.

**Bàn giao:** Giao diện đọc catalog đã nối API; danh sách điểm ghép với giỏ và may đo; kết quả kiểm tra từ trang chủ đến chi tiết.

### PD-03 — Admin quản lý danh mục và thông tin bán áo

**Mục tiêu admin:** Tôi tự cập nhật mẫu áo, giá và tồn kho để cửa hàng bán đúng hàng đang có.

**Căn cứ:** Specs 4, 6, 7; AC-10.

**Phụ thuộc:** PD-01, KR-01; phối hợp Dontwait về thay đổi tồn kho.

**Dữ liệu nhập:** Danh mục; tên/slug; ảnh; giá; chất liệu; màu/cỡ; SKU/tồn kho; phần đi kèm; bật/tắt bán. Các field tùy chọn khác chỉ theo schema hiện có.

**Việc cần thực hiện:**

- Tạo danh sách, form thêm/sửa danh mục và sản phẩm; hỗ trợ bật/tắt và sắp xếp danh mục.
- Kiểm tra trường bắt buộc theo schema; giá/tồn kho không âm; slug không trùng; biến thể có đủ SKU, màu, cỡ và tồn kho.
- Kiểm tra category tham chiếu có tồn tại; thống nhất tác động khi tắt danh mục để API công khai không xử lý khác nhau giữa list/detail.
- Quản lý ảnh theo cấu trúc hiện có; kiểm tra dữ liệu `src`, `alt`. Cách nhận ảnh thống nhất dùng chung với KR-04, không mặc định phải tích hợp dịch vụ lưu trữ ngoài.
- Tách cập nhật tồn kho khỏi lưu nội dung sản phẩm nếu cần để form đang mở không ghi đè tồn vừa bị trừ bởi đơn mới.
- Không cho form ghi đè `rating`/`review_count`; đây là dữ liệu tổng hợp từ KR-04.
- Bảo vệ cả trang và API bằng quyền admin; tài khoản khách không được thực hiện thao tác quản trị.

**Nghiệm thu:**

- [ ] Admin tạo mẫu hợp lệ, sửa giá và tắt bán; dữ liệu phía khách cập nhật tương ứng.
- [ ] Slug trùng, giá âm hoặc SKU thiếu thông tin bị từ chối và chỉ rõ lỗi.
- [ ] Tài khoản khách gọi trực tiếp API sửa sản phẩm bị từ chối.
- [ ] Đơn đã đặt không đổi giá/tên trong dòng đơn khi admin sửa catalog.
- [ ] Admin lưu mô tả trong lúc phát sinh mua hàng không phục hồi tồn kho về số cũ.

**Bàn giao:** API/giao diện quản lý danh mục và sản phẩm; quy tắc tồn kho chung với DW-02; ví dụ dữ liệu hợp lệ và lỗi.

### PD-04 — Câu chuyện, chất liệu và quy trình làm áo

**Mục tiêu:** Admin kể được câu chuyện của từng mẫu; khách hiểu ý nghĩa mẫu áo ngay tại trang chi tiết.

**Căn cứ:** Specs 1, 3.2; AC-03 và phần nội dung của AC-10.

**Phụ thuộc:** PD-02, PD-03.

**Việc cần thực hiện:**

- Bổ sung form nhập `story.eyebrow`, `title`, `paragraphs[]`, `image`; cho thêm, sửa, xóa và đổi thứ tự đoạn kể.
- Cho admin sửa `fabric` và các hướng dẫn `care[]`; sửa `craft` và các bước `steps[]` theo schema hiện tại.
- Hiển thị đúng nội dung theo sản phẩm; không dùng câu chuyện cố định cho mọi slug.
- Nội dung có thể kể về bối cảnh/thời đại/cảm hứng nhưng lưu ngay trong sản phẩm; diễn đạt đúng mức độ thông tin có căn cứ.
- Khi một phần nội dung chưa có, hiển thị hoặc ẩn phần đó hợp lý; không tự chèn câu chuyện mẫu như nội dung thật.

**Nghiệm thu:**

- [ ] Admin lưu ba đoạn câu chuyện theo thứ tự; khách đọc đúng ba đoạn sau tải lại trang.
- [ ] Sửa câu chuyện mẫu A không làm thay đổi mẫu B.
- [ ] Chất liệu, bảo quản và bước chế tác được lưu/hiển thị đúng field tương ứng.
- [ ] Nội dung do admin nhập được hiển thị an toàn, không chạy mã được chèn vào nội dung.

**Bàn giao:** Form nội dung trong quản lý sản phẩm; phần đọc trên trang chi tiết; dữ liệu kiểm tra cho hai mẫu có câu chuyện khác nhau.

## 3. Task của MyNameIsKry

### KR-01 — Tài khoản khách và xác thực chủ cửa hàng

**Mục tiêu:** Khách sử dụng tài khoản của mình; chỉ chủ cửa hàng được truy cập quản trị.

**Căn cứ:** Specs 2, 7, 8; hỗ trợ AC-02, AC-10.

**Phụ thuộc:** Không; cung cấp xác thực cho toàn bộ API có quyền.

**Việc cần thực hiện:**

- Làm API/giao diện đăng ký, đăng nhập và đăng xuất của khách với các field hiện có trong `customers`.
- Chuẩn hóa email để tránh tạo trùng chỉ do chữ hoa/thường; lưu hash mật khẩu, không trả `password_hash` trong response.
- Xác thực admin riêng theo phương án DB đã chốt ở mục 5; khách không thể tự đăng ký hoặc gửi field để nhận quyền admin.
- Cung cấp cách đọc danh tính hiện tại và kiểm tra quyền cho API khách/admin của hai thành viên còn lại.
- Khi hết phiên/đăng xuất, xóa trạng thái đăng nhập phù hợp; trang cần quyền hướng dẫn đăng nhập lại.
- Giữ đường đặt mua/gửi may đo của khách vãng lai hoạt động mà không yêu cầu đăng nhập.

**Ngoại lệ:** Email trùng; thiếu trường bắt buộc; sai thông tin đăng nhập; phiên không hợp lệ; tài khoản khách gọi API admin.

**Nghiệm thu:**

- [ ] Đăng ký hợp lệ lưu được khách, đăng nhập/đăng xuất hoạt động với API thật.
- [ ] Email chỉ khác hoa/thường không tạo được tài khoản thứ hai.
- [ ] Không thể đọc/sửa hồ sơ hoặc đơn người khác bằng cách đổi ID trong request.
- [ ] Không thể nhận quyền admin bằng cách sửa dữ liệu gửi từ giao diện.
- [ ] Khách chưa đăng nhập vẫn mở được checkout và gửi yêu cầu may đo.

**Bàn giao:** API/giao diện tài khoản; kiểm tra quyền dùng chung; cách tạo/cấu hình admin phục vụ chạy dự án; migration cần thiết và hướng dẫn gọi API có xác thực.

### KR-02 — Hồ sơ, địa chỉ, số đo và yêu thích

**Mục tiêu người dùng:** Tôi quản lý thông tin của mình để sử dụng cho lần mua tiếp theo.

**Căn cứ:** Specs 2, 6, 7; hỗ trợ AC-02.

**Phụ thuộc:** KR-01; PD-01 để hiển thị áo yêu thích.

**Việc cần thực hiện:**

- Hiển thị/sửa họ tên và số điện thoại theo `customers`.
- Thêm/sửa/xóa địa chỉ trong `addresses[]`; địa chỉ có người nhận, điện thoại và các phần địa chỉ theo DB; bảo đảm không có hai địa chỉ cùng mặc định.
- Hiển thị/sửa `measurement_profile` và `measurement_notes`; đơn vị cm cho các field `_cm`; `body` giữ dạng mô tả theo dữ liệu hiện có.
- Bật/tắt yêu thích bằng `wishlist_product_ids`; hiển thị danh sách mẫu đã lưu; thao tác lặp không tạo ID trùng.
- Không tự thay số đo, địa chỉ hoặc thông tin nhận hàng của đơn đã tạo khi khách sửa hồ sơ.

**Nghiệm thu:**

- [ ] Đổi địa chỉ mặc định chỉ còn tối đa một địa chỉ mặc định; xóa/sửa không tác động khách khác.
- [ ] Số đo âm bị từ chối; các field tùy chọn chưa có không bị điền thành dữ liệu giả.
- [ ] Lưu/bỏ yêu thích tồn tại sau đăng nhập lại; mẫu tắt bán được hiển thị tình trạng phù hợp hoặc loại khỏi danh sách mua.
- [ ] Sửa hồ sơ không đổi dữ liệu đã chốt của đơn trước đó.

**Bàn giao:** API và màn hình hồ sơ/yêu thích; cách checkout lấy địa chỉ từ tài khoản; kiểm tra quyền với hai tài khoản khác nhau.

### KR-03 — Giỏ hàng có và không có tài khoản

**Mục tiêu người dùng:** Tôi chọn nhiều áo, sửa số lượng và chuyển sang đặt hàng mà không phải đăng ký tài khoản.

**Căn cứ:** Specs 3.3, 7; hỗ trợ AC-01, AC-02, AC-06.

**Phụ thuộc:** PD-01, KR-01; bàn giao cho DW-03.

**Việc cần thực hiện:**

- Lưu giỏ khách vãng lai tại trình duyệt; giỏ tài khoản lưu `carts` theo đúng khách đã xác thực.
- Thêm, sửa số lượng, xóa dòng và đọc giỏ sau tải lại trang.
- Cùng SKU thì cộng số lượng; khác màu/cỡ thì giữ thành dòng riêng; số lượng phải là số nguyên dương.
- Hiển thị tên, ảnh, SKU, màu/cỡ, đơn giá, số lượng và tổng tạm tính; thông tin giá/tồn phải được kiểm tra lại khi checkout.
- Khi đăng nhập từ giỏ khách, hợp nhất theo SKU vào giỏ tài khoản; đọc lại giá/tồn và báo dòng cần sửa. Không xóa giỏ khách trước khi lưu giỏ tài khoản thành công. Đây là quy tắc triển khai đề xuất cần chốt ở mục 5.
- Khi đặt hàng thành công, chỉ bỏ các dòng đã mua; đặt thất bại giữ giỏ để khách sửa hoặc thử lại.

**Nghiệm thu:**

- [ ] Không đăng nhập, thêm hai SKU khác nhau, tải lại trang vẫn còn đúng lựa chọn.
- [ ] Thêm cùng SKU hai lần không sinh hai dòng giống nhau; số lượng 0, âm hoặc thập phân bị xử lý rõ.
- [ ] Giỏ tài khoản A không xuất hiện khi tài khoản B đăng nhập.
- [ ] Hết hàng/tăng giá sau khi thêm giỏ được thông báo khi kiểm tra; không âm thầm đổi tổng và tạo đơn.
- [ ] Đặt thất bại không mất giỏ; đặt thành công chỉ xóa dòng thuộc đơn thành công.

**Bàn giao:** API giỏ tài khoản; giỏ khách; màn hình giỏ; hàm/dữ liệu đưa sang checkout và quy tắc dọn giỏ sau kết quả đặt đơn.

### KR-04 — Review sau nhận áo và quản lý review

**Mục tiêu người dùng:** Sau khi nhận áo, tôi chia sẻ trải nghiệm về đúng sản phẩm đã mua.

**Căn cứ:** Specs 3.7, 4, 7, 8; AC-07, AC-08.

**Phụ thuộc:** KR-01, DW-02, DW-03; dùng `reviews` và cập nhật dữ liệu tổng hợp trong `products`.

**Việc cần thực hiện:**

- Hiển thị danh sách review theo sản phẩm; form điểm 1–5, nhận xét và ảnh theo schema.
- Kiểm tra ở server rằng sản phẩm thuộc đơn đã `delivered` của khách; không chỉ tin `verified_purchase` hoặc `customer_id` do trình duyệt gửi.
- Với tài khoản: giữ một review/khách/sản phẩm theo index hiện có; gửi lại cập nhật review của mình.
- Với khách vãng lai: nhận quyền truy cập đơn đã xác minh từ DW-03; bổ sung liên kết đơn vào review và điều chỉnh validator/index tối thiểu theo quyết định ở mục 5.
- Cung cấp màn hình admin xem và xử lý review không phù hợp. DB chưa có trạng thái duyệt/ẩn, cần chốt thao tác xử lý cụ thể trước triển khai; không tự thêm luồng duyệt.
- Tính lại `products.rating` và `review_count` sau thêm/sửa/xử lý review; không tăng số review khi chỉ sửa nội dung cũ. Không có review thì điểm/số lượng tổng hợp là 0.
- Thống nhất cách nhận ảnh với PD-03; không trả thông tin liên hệ, ID xác minh hoặc dữ liệu riêng của đơn trong review công khai.

**Nghiệm thu:**

- [ ] Đơn chưa giao, đã hủy hoặc không chứa mẫu cần review bị từ chối đánh giá mua hàng.
- [ ] Khách A không đánh giá/sửa review thay khách B dù biết ID đơn hoặc review.
- [ ] Khách vãng lai có quyền truy cập hợp lệ của đơn đã giao đánh giá được đúng mẫu trong đơn.
- [ ] Rating ngoài 1–5 bị từ chối; gửi lại review của tài khoản không tạo trùng.
- [ ] Có hai review điểm 4 và 5 thì tổng hợp là 4,5 với số lượng 2; sửa điểm không tăng số lượng.
- [ ] Admin xử lý review theo phương án đã chốt; số tổng hợp khớp danh sách review được công khai.

**Bàn giao:** API/giao diện review khách và admin; migration validator/index; dữ liệu kiểm tra tài khoản/khách vãng lai; bằng chứng đối chiếu điểm và số lượng.

## 4. Task của Dontwait

### DW-01 — Điều chỉnh schema đơn hàng

**Mục tiêu nghiệp vụ:** DB lưu được đơn không có tài khoản, COD và đơn thu tại quầy bằng dữ liệu thật.

**Căn cứ:** Specs 3.4, 3.5, 8; hỗ trợ AC-01, AC-04.

**Phụ thuộc:** Chốt cách lưu thông tin nhận tại quầy ở mục 5.

**Việc cần thực hiện:**

- Cho `orders.customer_id` thiếu hoặc null khi khách vãng lai đặt; vẫn kiểm tra đúng khách khi có tài khoản.
- Thêm `cash_at_counter` vào enum `payment_method`; giữ tương thích giá trị cũ trong DB nhưng API đặt mới chỉ nhận `cod` hoặc `cash_at_counter` theo phạm vi.
- Cho đơn nhận tại quầy thiếu các phần địa chỉ giao; vẫn có tên và điện thoại để chủ cửa hàng xử lý đơn.
- Bám cấu trúc `shipping_address` hiện có nếu đáp ứng được liên hệ; không tạo email, mật khẩu, tài khoản hoặc địa chỉ giả.
- Ghi rõ điều kiện địa chỉ theo cách nhận: giao tận nơi cần đủ địa chỉ, nhận tại quầy không cần. Validator và xử lý ứng dụng phải nhất quán.
- Viết migration up/down; kiểm tra với dữ liệu cũ. Nếu down không tương thích đơn mới có null/enum mới, ghi rõ dữ liệu nào phải xử lý trước, không xóa đơn để migration chạy được.

**Nghiệm thu:**

- [ ] Migration chấp nhận đơn tài khoản cũ và đơn khách vãng lai mới.
- [ ] Lưu được đơn thu tại quầy chỉ có tên/điện thoại liên hệ, không cần địa chỉ giả.
- [ ] Đơn giao tận nơi thiếu địa chỉ bị xử lý nghiệp vụ từ chối.
- [ ] Migration không làm mất đơn hoặc đổi ý nghĩa trạng thái cũ.

**Bàn giao:** Migration và hướng dẫn chạy/khôi phục; bảng field bắt buộc theo hai trường hợp nhận áo; dữ liệu mẫu hợp lệ/không hợp lệ.

### DW-02 — Tạo và xử lý đơn, tính tiền, cập nhật tồn kho

**Mục tiêu:** Mỗi đơn lưu đúng áo, đúng giá, đúng người đặt và không bán vượt hàng hiện có.

**Căn cứ:** Specs 3.4, 5, 7; AC-01, AC-02, AC-06.

**Phụ thuộc:** DW-01; PD-01 về dữ liệu catalog; KR-01 cho tài khoản/admin.

**Dữ liệu tạo đơn:** Các sản phẩm/SKU và số lượng; thông tin nhận; `payment_method`; ghi chú; thông tin may đo/liên kết yêu cầu nếu áp dụng. Danh tính tài khoản lấy từ xác thực, không lấy tùy ý từ request.

**Việc cần thực hiện:**

- Kiểm tra có ít nhất một dòng; sản phẩm tồn tại/đang bán; SKU thuộc sản phẩm; lượng hợp lệ và đủ tồn cho hàng sẵn.
- Đọc đơn giá từ DB; tính `subtotal = Σ(quantity × unit_price)`, `total = subtotal + shipping_fee`; nhận tại quầy phí bằng 0. Phí giao tận nơi cần nguồn tính thống nhất ở mục 5, không tin giá khách gửi.
- Lưu tên, SKU, màu/cỡ, lượng và giá vào `items[]`; tạo đơn `pending`, `payment_status = unpaid`; dùng `order_number` để khách nhận diện đơn.
- Bảo đảm tạo đơn và trừ tồn nhất quán; lỗi giữa chừng không để đơn thành công nhưng tồn chưa trừ hoặc ngược lại. Cách thực hiện phải phù hợp MongoDB thực tế của dự án.
- API xem đơn: khách tài khoản chỉ xem đơn của mình; admin xem/xử lý đơn theo quyền; khách vãng lai chỉ nhận dữ liệu qua quyền xác minh đã chốt cùng DW-03/KR-04.
- API chuyển trạng thái: giao tận nơi `pending → confirmed → shipping → delivered`; tại quầy `pending → confirmed → delivered`; chặn chuyển ngược hoặc chuyển trái luồng không hợp lệ.
- Hủy đơn chưa giao xong theo specs, hoàn lại đúng tồn đã trừ; lặp thao tác không hoàn tồn lần hai. Nếu hàng đã bàn giao vận chuyển, cách xác nhận thu hồi trước hoàn tồn cần chốt ở mục 5.
- Hỗ trợ đơn `custom` với `tailoring_request_id` và số đo hiện có; không áp dụng máy móc tồn áo may sẵn cho đơn may đo. Giá và thông tin áo phải được chủ cửa hàng xác nhận trước khi tạo đơn may đo.

**Nghiệm thu:**

- [ ] Khách không đăng nhập tạo đơn được; không phát sinh tài khoản giả.
- [ ] Sửa giá/tổng tiền trong request không làm thay đổi cách tính từ dữ liệu có thẩm quyền.
- [ ] Hai khách cùng mua SKU còn một chiếc: chỉ một đơn mua được, tồn không âm.
- [ ] Hủy cùng đơn hai lần chỉ hoàn đúng lượng đã trừ một lần.
- [ ] Sửa catalog sau đặt không thay thông tin/đơn giá đã lưu trong đơn.
- [ ] Tài khoản khác không đọc được đơn dù biết ID; khách thường không cập nhật trạng thái được.

**Bàn giao:** API tạo/đọc/xử lý đơn; quy tắc lỗi; kiểm tra cạnh tranh tồn kho và hủy lặp với DB thật; dữ liệu đơn cho KR-04 kiểm tra quyền review.

### DW-03 — Checkout, xem đơn, admin giao áo và thu tiền

**Mục tiêu:** Khách biết mình đã đặt gì và trả tiền khi nào; chủ cửa hàng xử lý giao áo và tiền thu độc lập.

**Căn cứ:** Specs 3.4, 3.5, 4, 5; AC-01, AC-02, AC-04, AC-05, AC-10.

**Phụ thuộc:** DW-02, KR-03, KR-01; cung cấp quyền xác minh đơn cho KR-04.

**Việc cần thực hiện — phía khách:**

- Màn hình checkout lấy giỏ thật; hiển thị từng dòng, thông tin nhận, tiền hàng, phí và tổng tiền trước xác nhận.
- Cho chọn COD hoặc thu tại quầy; áp dụng trường bắt buộc theo cách nhận đã thống nhất trong specs, không hiện thêm phương thức thanh toán.
- Khách đăng nhập có thể dùng địa chỉ đã lưu; khách vãng lai nhập thông tin trực tiếp.
- Nếu giá/tồn thay đổi, hiển thị và yêu cầu khách kiểm tra lại trước khi gửi đơn theo dữ liệu mới.
- Chỉ báo đặt thành công khi server đã lưu; hiển thị mã đơn và tóm tắt. Khi đang gửi, ngăn bấm liên tục; mất kết nối không tự tạo thêm đơn nếu chưa biết kết quả lần trước.
- Trang tài khoản hiển thị danh sách/chi tiết đơn của mình, gồm trạng thái nhận áo và thanh toán tách biệt.
- Cùng KR-04 triển khai cách khách vãng lai xác minh đơn đã mua để review. Phương án phải được chốt ở mục 5 trước khi làm; không dùng riêng mã đơn công khai hay số điện thoại tự khai làm quyền truy cập.

**Việc cần thực hiện — phía admin:**

- Danh sách/chi tiết đơn, lọc theo trạng thái đơn/thanh toán; xem tên/điện thoại, địa chỉ nếu giao hàng, dòng áo và tổng tiền.
- Thao tác xác nhận, giao, đã nhận và hủy chỉ hiển thị khi phù hợp trạng thái hiện tại; server vẫn kiểm tra lại.
- Nút ghi nhận thu tiền riêng với nút cập nhật giao hàng; thu đủ thì chuyển `unpaid → paid` theo thực tế.
- Nếu phát sinh hoàn tiền, chỉ chuyển `paid → refunded` sau khi admin xác nhận đã hoàn; không hiểu `refunded` là tính năng hoàn tiền tự động.

**Nghiệm thu:**

- [ ] COD yêu cầu địa chỉ; tại quầy không yêu cầu địa chỉ giao và phí bằng 0.
- [ ] Khách vãng lai hoàn tất đặt hàng mà không bị chuyển sang đăng ký.
- [ ] Bấm xác nhận đơn không tự chuyển thành `paid`; admin đánh dấu `delivered` cũng không tự chuyển thành `paid`.
- [ ] Đơn tại quầy chuyển từ đã xác nhận sang đã nhận mà không qua đang giao.
- [ ] Tải lại trang hoặc gọi API trực tiếp vẫn kiểm tra quyền đúng; khách vãng lai không xem đơn khác bằng cách đổi mã đơn.
- [ ] Đặt thất bại giữ lại dữ liệu/giỏ; không báo thành công giả hoặc trừ tồn thêm do gửi lại cùng thao tác.

**Bàn giao:** Checkout, trang đơn khách, quản trị đơn/thu tiền; luồng xác minh cho review; kịch bản COD và tại quầy từ đầu đến cuối.

### DW-04 — Gửi và xử lý yêu cầu may đo

**Mục tiêu:** Khách gửi nhu cầu thật; chủ cửa hàng theo dõi và thực hiện bằng dữ liệu có sẵn.

**Căn cứ:** Specs 3.6, 4, 5.3, 6; AC-09.

**Phụ thuộc:** KR-01 cho admin; DW-02 khi tạo đơn may đo.

**Việc cần thực hiện:**

- Nối form tại trang hướng dẫn/may đo và lựa chọn tư vấn ở chi tiết áo với API lưu `tailoring_requests`.
- Bắt buộc tên/điện thoại; mẫu quan tâm và số đo có thể chưa đủ khi gửi yêu cầu. Không yêu cầu tài khoản; có tài khoản thì liên kết danh tính đúng.
- Lưu `body`, bảy field số đo `_cm`, ghi chú và kênh liên hệ theo schema; không tự đổi `body` thành trường chiều cao/cân nặng mới.
- Admin xem danh sách/chi tiết, lọc trạng thái, cập nhật ghi chú, lịch hẹn, số đo và tình trạng theo enum hiện có.
- Chủ cửa hàng kiểm tra lại số đo với khách trước khi đánh dấu `measurement_confirmed`; không tự chuyển chỉ vì form có đủ số.
- Khi xác nhận mua may đo, tạo đơn `custom`, liên kết đúng yêu cầu, lưu số đo của đơn; giá được xác nhận, dùng cùng hai hình thức thu tiền đã chốt.
- Ghi rõ trên giao diện rằng gửi yêu cầu may đo chưa phải đặt đơn thành công. Cập nhật hồ sơ/yêu cầu về sau không tự ghi đè số đo đã chốt trong đơn.

**Nghiệm thu:**

- [ ] Khách vãng lai chỉ nhập tên/điện thoại và nhu cầu vẫn gửi được; thiếu tên/điện thoại thì chưa lưu.
- [ ] Chọn tư vấn từ mẫu A tạo yêu cầu liên quan đúng A; không còn slug mẫu mặc định.
- [ ] Admin tải lại trang vẫn thấy yêu cầu, lịch hẹn và số đo đã sửa.
- [ ] Lỗi lưu không xóa dữ liệu nhập và không thông báo đã lưu.
- [ ] Đơn may đo có `fulfillment_mode = custom`, liên kết yêu cầu và số đo đúng; sửa yêu cầu sau đó không sửa ngầm đơn.

**Bàn giao:** API/giao diện gửi và quản trị yêu cầu; cách tạo đơn may đo; dữ liệu kiểm tra khách vãng lai và khách có tài khoản.

## 5. Quyết định cần chốt trước task liên quan

Các điểm dưới đây chưa được DB/specs xác định đủ để lập trình thống nhất. Đây là việc cần hoàn tất trong task được giao, không phải lý do dừng toàn bộ nhóm. Người phụ trách ghi quyết định vào tài liệu API/DB của PR trước khi triển khai phần phụ thuộc.

| Mã | Nội dung cần chốt | Người chủ trì / phối hợp | Đầu ra bắt buộc | Task chịu ảnh hưởng |
| --- | --- | --- | --- | --- |
| Q-01 | Lưu và cấp quyền admin bằng cách nào, dựa trên khoảng trống schema hiện tại | MyNameIsKry / cả nhóm | Phương án tối thiểu, cách khởi tạo admin, kiểm tra quyền server; không cho khách tự nâng quyền | KR-01 và mọi API admin |
| Q-02 | Đơn tại quầy lưu tên/điện thoại và địa chỉ tùy chọn thế nào | Dontwait / MyNameIsKry | Field mapping, validator, hai payload mẫu COD/tại quầy; thống nhất tại quầy trong phạm vi này là nhận trực tiếp | DW-01–03 |
| Q-03 | Khách vãng lai chứng minh quyền review và lưu liên kết thế nào | Dontwait / MyNameIsKry | Cách xác minh khả thi khi vận hành, không cần dịch vụ ngoài chưa có; field/index review; quy tắc chống gửi lặp cho cùng đơn/sản phẩm | DW-03, KR-04 |
| Q-04 | Phí giao tận nơi lấy từ đâu | Dontwait / chủ cửa hàng | Quy tắc phí được chủ cửa hàng xác nhận, nguồn cấu hình và cách hiển thị trước chốt; không tự đặt số tiền | DW-02, DW-03 |
| Q-05 | Giỏ khi đăng nhập | MyNameIsKry / Dontwait | Chốt quy tắc hợp nhất đề xuất ở KR-03 và xử lý lượng vượt tồn; định dạng bàn giao checkout | KR-03, DW-03 |
| Q-06 | Admin xử lý review bằng thao tác nào khi chưa có field ẩn/duyệt | MyNameIsKry / chủ cửa hàng | Hành vi cụ thể bám DB; nếu cần đổi schema phải nêu lý do và thống nhất trước, không tự thêm duyệt | KR-04 |
| Q-07 | Đồng bộ tồn tổng/biến thể và hủy đơn đã đang giao | Dontwait / pdmh18 | Nguồn tồn chuẩn, thời điểm trừ/hoàn, điều kiện hàng thu hồi được nhập lại; tránh cộng hàng chưa về kho | PD-03, DW-02 |
| Q-08 | Cách nhập/lưu ảnh sản phẩm và review | pdmh18 / MyNameIsKry | Dữ liệu `src`/`alt` hoặc cách nhận ảnh thống nhất, giới hạn phù hợp, kiểm soát quyền; không mặc định thêm hạ tầng lưu trữ | PD-03, KR-04 |

Không gán deadline hoặc số ngày công khi chưa có năng lực/lịch làm của ba người. Sau khi chốt Q-01–08, từng người ước lượng phần của mình và báo các phụ thuộc có thể ảnh hưởng tiến độ.

## 6. Hợp đồng bàn giao giữa ba người

| Bên cung cấp → bên dùng | Dữ liệu/hành vi cần thống nhất | Điều kiện được coi là bàn giao |
| --- | --- | --- |
| pdmh18 → MyNameIsKry, Dontwait | ID sản phẩm, slug, SKU, màu/cỡ, giá, tồn kho, trạng thái bán, cờ may đo | Có ví dụ hàng còn/hết, SKU không tồn tại và API chạy được |
| MyNameIsKry → pdmh18, Dontwait | Danh tính khách/admin, cách gọi có xác thực, hành vi hết phiên/thiếu quyền | Có tài khoản thử, ví dụ khách/admin và kiểm tra server |
| MyNameIsKry → Dontwait | Dòng giỏ, cách xác định SKU, sửa lượng và dọn dòng đã mua | Checkout đọc được; lỗi đặt không xóa giỏ |
| Dontwait → MyNameIsKry | Đơn thuộc ai, trạng thái nhận áo, sản phẩm trong đơn, quyền xác minh đơn khách vãng lai | KR-04 kiểm tra được quyền review bằng dữ liệu nghiệp vụ, không qua cờ khách tự gửi |
| MyNameIsKry → pdmh18 | Quy tắc tổng hợp rating/count và dữ liệu review công khai | Trang detail dùng cùng kết quả; form admin không ghi đè |
| Dontwait ↔ pdmh18 | Cập nhật tồn do đặt/hủy và do admin chỉnh | Không mất cập nhật khi hai thao tác diễn ra đồng thời |

Mỗi API bàn giao phải mô tả: đầu vào bắt buộc/tùy chọn, response thành công, quyền truy cập, lỗi thiếu dữ liệu/không tồn tại/không đủ tồn/không đủ quyền khi áp dụng. Tên field chốt một lần và dùng thống nhất giữa giao diện, API và DB; thay đổi cần thông báo người đang dùng.

## 7. Trình tự triển khai và mốc ghép luồng

| Mốc | Việc thực hiện | Điều kiện kết thúc mốc |
| --- | --- | --- |
| M1 — Nền dữ liệu/quyền | PD-01; KR-01; DW-01; chốt các quyết định ảnh hưởng trực tiếp | Đọc catalog thật, xác thực khách/admin, schema nhận được hai loại đơn |
| M2 — Chọn áo và quản trị catalog | PD-02, PD-03, PD-04; KR-02, KR-03; DW-02 | Khách chọn đúng SKU vào giỏ; admin cập nhật được áo/câu chuyện; API tạo đơn tính đúng tiền/tồn |
| M3 — Bán và nhận áo | DW-03; DW-04 | Khách vãng lai/tài khoản đặt được COD/tại quầy; admin giao và thu tiền riêng; may đo lưu thật |
| M4 — Review và kiểm tra xuyên suốt | KR-04; cả nhóm ghép các luồng dưới đây | Đạt AC-01–10 của specs bằng API/DB/giao diện thật, ghi rõ lỗi còn lại |

Phần đọc review, form và API độc lập có thể chuẩn bị trước M4. KR-04 chỉ được nghiệm thu sau khi đã có đơn và quyền xác minh thật từ DW-02/DW-03.

## 8. Kịch bản nghiệm thu tích hợp

| Mã | Dữ liệu/tiền điều kiện | Thao tác | Kết quả bắt buộc | Người chịu trách nhiệm | Specs |
| --- | --- | --- | --- | --- | --- |
| IT-01 | Mẫu A/B có câu chuyện khác nhau, mỗi mẫu có biến thể | Mở từ danh sách → từng mẫu → chọn SKU → thêm giỏ | Đúng nội dung, đúng SKU/màu/cỡ; giỏ không dùng mẫu mặc định | pdmh18 + MyNameIsKry | AC-03 |
| IT-02 | Khách chưa đăng nhập, SKU còn ít nhất 2 | Đặt 2 áo COD với địa chỉ hợp lệ | Một đơn không tài khoản, tổng đúng, `pending/unpaid`, tồn giảm đúng; không tạo tài khoản giả | Dontwait + MyNameIsKry | AC-01, AC-04, AC-06 |
| IT-03 | Khách đăng nhập, giỏ có áo | Chọn tại quầy, không nhập địa chỉ giao → admin xác nhận → giao tại quầy → thu tiền | Phí 0; `confirmed → delivered`; chỉ thao tác thu tiền mới chuyển `paid`; khách thấy đơn của mình | Dontwait + MyNameIsKry | AC-02, AC-04, AC-05 |
| IT-04 | SKU chỉ còn một chiếc | Hai khách xác nhận mua gần đồng thời | Chỉ một khách mua được; không có tồn âm hoặc hai đơn thành công dùng cùng chiếc áo | Dontwait + pdmh18 | AC-06 |
| IT-05 | Đơn đã trừ tồn, chưa giao | Admin hủy rồi gửi lại thao tác hủy | Đơn hủy; tồn chỉ hoàn một lần | Dontwait | AC-06 |
| IT-06 | Đơn có giá/tên đã lưu | Admin sửa giá, tên áo và câu chuyện | Catalog đổi; tên/giá trong đơn cũ giữ nguyên | pdmh18 + Dontwait | AC-03, AC-06 |
| IT-07 | Tài khoản A đã nhận mẫu X; B không sở hữu đơn | A review rồi sửa; B thử review qua ID của A | A có một review được cập nhật; B bị từ chối; rating/count đúng | MyNameIsKry + Dontwait | AC-07, AC-08 |
| IT-08 | Đơn khách vãng lai đã giao, có cách xác minh đã chốt | Khách xác minh → review; người khác dùng mã đơn công khai thử truy cập | Người có quyền review được; người chỉ biết mã đơn không đọc/đánh giá thay | MyNameIsKry + Dontwait | AC-07 |
| IT-09 | Đơn pending/cancelled hoặc không chứa mẫu Y | Thử review Y bằng request trực tiếp | Bị từ chối ở server, không phát sinh mua hàng xác thực | MyNameIsKry | AC-07 |
| IT-10 | Khách chưa đăng nhập ở mẫu A | Gửi may đo → admin xem/sửa lịch và số đo → tạo đơn custom khi xác nhận mua | Đúng mẫu A, lưu thật; yêu cầu và đơn phân biệt; đơn liên kết đúng yêu cầu | Dontwait | AC-09 |
| IT-11 | Tài khoản khách thường và admin | Gọi trực tiếp API sửa áo, cập nhật đơn và tiền | Khách bị từ chối; admin được thao tác hợp lệ | Cả ba theo API phụ trách | AC-10 |
| IT-12 | Giỏ hoặc form đang nhập | Giả lập lỗi API khi gửi; thử lại sau khi xác định kết quả lần trước | Không báo thành công giả, không mất dữ liệu, không tạo đơn lặp do thử lại | MyNameIsKry + Dontwait | AC-01, AC-06, AC-09 |

Mỗi kịch bản ghi kết quả thực tế, dữ liệu thử, người kiểm tra và bằng chứng phù hợp. Không đánh dấu đạt chỉ vì build thành công hoặc toast hiện đúng chữ.

## 9. Điều kiện hoàn thành và quy ước PR

Một task chỉ hoàn thành khi:

- [ ] Đạt các mục nghiệm thu của task và phần AC trong specs liên quan.
- [ ] API/giao diện dùng DB thật, kiểm tra cả thành công và lỗi có ảnh hưởng nghiệp vụ.
- [ ] Quyền khách/admin và quyền sở hữu tài nguyên được kiểm tra ở server.
- [ ] Có request/response mẫu để người làm phần phụ thuộc tích hợp.
- [ ] Nếu đổi schema/index, có migration up/down và ghi rõ tương thích dữ liệu cũ; không sửa migration đã áp dụng để che thay đổi.
- [ ] Không làm hỏng route/giao diện hiện có thuộc luồng cần giữ; đã xử lý các file chung có người khác sử dụng.
- [ ] Chạy kiểm tra phù hợp: trong `client/`, `npm run build` và `npm run lint`; trong `server/`, `make test` cho thay đổi Go, kiểm tra migration riêng nếu có.
- [ ] PR theo template repository, có mã task, kết quả thay đổi, cách kiểm tra, giới hạn chưa kiểm chứng và quyết định Q liên quan.

**Phối hợp file và migration:** Người sửa collection chịu trách nhiệm migration tương ứng; thống nhất số thứ tự trước khi tạo. `router.go`, route client và kiểu dùng chung cần ghép có trao đổi. Phân quyền và cấu trúc request dùng chung không được mỗi người tự định nghĩa một cách.

**Tổng hợp nghiệm thu:** Mỗi người báo kết quả task và IT do mình phụ trách; Dontwait tổng hợp mức đáp ứng AC-01–10. Chỉ sửa `system.md` khi có thay đổi yêu cầu được thống nhất; lỗi triển khai không phải lý do tự đổi specs để đạt nghiệm thu.
