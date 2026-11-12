# Giao diện catalog Aodora — PD-02

## 1. Phạm vi

Trang chủ, bộ sưu tập và chi tiết sản phẩm sử dụng các endpoint đọc công khai của backend Go (PD-01):

| API | Màn hình sử dụng | Dữ liệu |
| --- | --- | --- |
| `GET /api/v1/categories` | Trang chủ, tab danh mục, chân trang | Danh mục đang hoạt động |
| `GET /api/v1/products` | Trang chủ, bộ sưu tập | Sản phẩm, tổng kết quả, phân trang |
| `GET /api/v1/products/{slug}` | Chi tiết sản phẩm | SKU, tồn kho, ảnh, nội dung, đánh giá tổng hợp |

Giá bán, tên, ảnh, danh mục, vật liệu, số lượng tồn, biến thể, nhãn và nội dung sản phẩm đều xuất phát từ MongoDB thông qua API; không dùng bản ghi sản phẩm mẫu trong giao diện.

## 2. Duyệt, tìm kiếm và lọc

Trang bộ sưu tập đồng bộ các tham số vào URL:

- `category`: slug hoặc ID danh mục từ API.
- `search`: từ khóa cho MongoDB text search trên tên và mô tả (không tìm một đoạn ký tự bất kỳ).
- `material`, `color`, `size`: các trường lọc theo tên chính xác; màu và size phải nằm cùng biến thể SKU.
- `min_price`, `max_price`: khoảng giá không âm, đơn vị VND.
- `sort`: `newest`, `price_asc`, `price_desc`, `name_asc`, `name_desc`.
- `page`: trang đang xem; kích cỡ trang là 12.

Bộ lọc được áp dụng một lần sau khi nhấn nút áp dụng để giảm số truy vấn không cần thiết. Khi đổi danh mục/bộ lọc/thứ tự sắp xếp, trang được đưa về 1. Tổng và số trang sử dụng kết quả API, không có số lượng cố định. Nếu không có kết quả, giao diện cho phép xóa lọc.

## 3. Chi tiết và lựa chọn SKU

Đường dẫn `/san-pham/:slug` xác định sản phẩm. API trả 404 khi không tồn tại hoặc sản phẩm/danh mục không hoạt động. Trang chi tiết hiển thị lỗi và đường dẫn quay lại bộ sưu tập, không thay bằng sản phẩm mặc định.

- Ảnh chính và thư viện lấy từ `image`, `gallery` hoặc `images` trong response.
- Nội dung `story`, `fabric`, `craft`, phụ kiện, badge, trust chỉ hiển thị khi có dữ liệu tương ứng.
- Mã màu chỉ hiển thị dạng swatch khi response thực sự có `hex`; nếu không có thì dùng tên màu.
- `variants[]` quyết định SKU. Chọn một màu chỉ cho phép chọn các kích cỡ có cùng màu và còn hàng. Khi thay màu, lựa chọn kích cỡ được đặt lại để tránh ghép thuộc tính từ hai SKU.
- Số lượng phải nguyên dương và không vượt quá tồn kho của SKU cũng như tồn tổng sản phẩm. Giá/tồn kho cần được xác minh lại tại thời điểm tạo đơn hàng.
- Chỉ hiển thị tổng số và điểm đánh giá từ `reviews` của PD-01. Nội dung từng review và quy trình viết đánh giá phụ thuộc module KR-04.

## 4. Điểm ghép module khác

### KR-03 — Giỏ hàng

`client/src/features/catalog/selection.ts` định nghĩa `CartSelection`:

```ts
{
  product_id: string,
  product_slug: string,
  sku: string,
  color: string,
  size: string,
  quantity: number,
  fulfillment_mode: 'ready'
}
```

`createCartSelection` kiểm tra lựa chọn dựa trên SKU và tồn kho đã đọc. Module KR-03 đăng ký callback `registerCartAddHandler(async (selection) => { ... })` để ghi vào giỏ vãng lai hoặc giỏ tài khoản. Khi callback hoàn thành thành công giao diện mới hiển thị thông báo đã thêm. Khi chưa tích hợp, thao tác trả lỗi rõ ràng, không giả lập trạng thái giỏ. Khi checkout phải xác nhận lại giá và tồn kho tại backend.

### DW-04 — May đo

Liên kết hướng dẫn may đo mang theo `product_id`, `product_slug` và `mode=custom` trên query string. Form yêu cầu may đo của DW-04 sẽ nhận các thông tin này, cho khách nhập số đo và lưu yêu cầu qua backend. Chuyển trang không được coi là đã đặt lịch hoặc tạo yêu cầu thành công.

## 5. Kết nối môi trường

Khi chạy Vite ở chế độ phát triển, các yêu cầu `/api` được proxy tới backend Go theo biến `API_PROXY_TARGET` của môi trường chạy Vite; mặc định là `http://localhost:8081`. Cổng backend được quản lý độc lập qua cấu hình Go `HTTP_ADDR`.

Trong môi trường triển khai thực tế, web và API có thể chung origin. Nếu ở origin khác, cấu hình `VITE_API_BASE_URL` của frontend và CORS phù hợp trên backend.

## 6. Trạng thái và kiểm thử

Các màn hình hỗ trợ đang tải, rỗng, lỗi tải, thử lại; request cũ bị hủy khi thay từ khóa, bộ lọc hoặc slug. Dữ liệu cũ không được hiển thị nhầm khi chuyển từ mẫu A sang mẫu B.

Kiểm thử nghiệp vụ tại `client/tests/catalog.test.mjs` tập trung vào hành vi lấy API, sản phẩm không tồn tại, chọn SKU đúng/màu/cỡ hợp lệ, tồn kho và hợp đồng tích hợp KR-03. Kiểm thử tương tác trên trình duyệt và độ tương thích điện thoại cần chạy trên frontend đã kết nối backend MongoDB thực tế.
