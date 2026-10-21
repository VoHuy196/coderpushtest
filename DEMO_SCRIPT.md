# Kịch Bản Quay Video Demo (3–5 Phút)
## Bài Đánh Giá: Cập Nhật Danh Mục Nhà Cung Cấp (Supplier Catalog Updates)

Tài liệu này là kịch bản thoại **100% bằng tiếng Việt**, được viết theo văn phong tự nhiên, rõ ràng, dễ đọc khi đang quay màn hình. Bạn chỉ cần bật phần mềm quay màn hình, làm theo từng thao tác và đọc theo lời thoại tương ứng.

---

## 1. Chuẩn bị trước khi bấm Quay (REC)

1. **Làm sạch database (để bắt đầu từ trạng thái mới tinh):**
   Mở terminal tại thư mục dự án và chạy:
   ```powershell
   Remove-Item -Force -ErrorAction SilentlyContinue catalog.db, catalog.db-shm, catalog.db-wal
   ```
2. **Khởi động 2 service:**
   - **Terminal 1 (Fake Supplier):**
     ```powershell
     .\bin\fakesupplier.exe -addr :8081 -delay 500ms
     ```
   - **Terminal 2 (Server chính):**
     ```powershell
     .\bin\server.exe -addr :8080 -supplier http://localhost:8081 -static web/dist
     ```
3. **Mở trình duyệt:**
   - Mở tab: `http://localhost:8080` (Phóng to cỡ chữ khoảng 110% cho rõ nét).
   - Mở sẵn 1 cửa sổ Terminal thứ 3 để cuối video gõ lệnh test.

---

## 2. Kịch bản chi tiết & Lời thoại tiếng Việt

### ⏱️ Phần 1: Giới thiệu dự án (0:00 – 0:40)
* **Thao tác màn hình:** Quay toàn cảnh trang `http://localhost:8080`. Rê chuột qua 3 thẻ thống kê ở trên cùng và công tắc Outage ở thanh header.
* **Lời thoại nói:**
  > "Xin chào mọi người. Hôm nay mình xin demo dự án giải quyết bài toán đồng bộ danh mục sản phẩm từ nhà cung cấp, được xây dựng bằng ngôn ngữ Go, cơ sở dữ liệu SQLite và giao diện web React Ant Design.
  >
  > Hệ thống này giải quyết trọn vẹn 3 yêu cầu cốt lõi của đề bài:
  > Thứ nhất là khả năng chống chịu sự cố: import theo từng batch an toàn, nếu mất mạng hay sập ứng dụng giữa chừng thì không làm mất dữ liệu đã lưu và có thể tiếp tục chạy lại.
  > Thứ hai là cơ chế versioning dựa trên thời gian `updated_at`: dữ liệu đến trễ hoặc đến sai thứ tự sẽ không thể ghi đè dữ liệu mới hơn, và việc import lặp lại nhiều lần sẽ không bao giờ sinh ra sản phẩm trùng lặp.
  > Thứ ba là cơ chế tombstone: sản phẩm đã ngừng bán sẽ được lưu vết để không bao giờ bị các bản cập nhật cũ hồi sinh lại."

---

### ⏱️ Phần 2: Chạy bản cập nhật đầu tiên & Triage lỗi (0:40 – 1:35)
* **Thao tác màn hình:**
  1. Trong hộp chọn "Start an update", chọn: `NW-2025-09-01 — Initial full catalog (3 batches)`.
  2. Bấm nút màu xanh **Start update**.
  3. Thanh tiến trình chạy qua từng batch: 1/3, 2/3, 3/3 và chuyển thành tag xanh **Completed**.
  4. Bấm nút **Details** ở dòng Update #1 để mở thanh chi tiết bên phải.
  5. Bấm vào các nút lọc: `created (11)`, `skipped (1)`, `invalid (1)`.
  6. Đóng thanh chi tiết, bấm sang tab **Problems (1)** ở màn hình chính.
* **Lời thoại nói:**
  > "Bây giờ, mình sẽ bắt đầu với bản cập nhật đầu tiên: `NW-2025-09-01`.
  > Quá trình import diễn ra bất đồng bộ và thanh tiến trình hiển thị trực tiếp theo từng batch.
  >
  > Khi hoàn thành, mình bấm vào nút **Details** để xem toàn bộ lịch sử thay đổi:
  > - Đã có 11 sản phẩm mới được tạo thành công.
  > - Có 1 bản ghi bị trùng lặp trong đợt gửi này được hệ thống tự động phát hiện và bỏ qua (skip).
  > - Đặc biệt, có 1 sản phẩm bị thiếu tên đã được phát hiện là không hợp lệ.
  >
  > Ở tab **Problems**, bản ghi lỗi này được giữ lại thành một sự cố riêng biệt để nhân viên có thể kiểm tra và xử lý, thay vì làm sập toàn bộ đợt cập nhật hay làm gián đoạn các sản phẩm khác."

---

### ⏱️ Phần 3: Chống trùng lặp & Bảo vệ dữ liệu mới (1:35 – 2:40)
* **Thao tác màn hình:**
  1. Ở ô chọn update, chọn lại chính `NW-2025-09-01` một lần nữa -> Bấm **Start update**.
  2. Khi chạy xong, chỉ vào kết quả: `+0 created, ~0 updated, skip 12`.
  3. Tiếp theo, chọn bản `NW-2025-09-08 — Weekly changes (3 batches)` -> Bấm **Start update**.
  4. Chờ chạy xong, mở Details cho thấy: sản phẩm ghế xếp đã được nhà cung cấp bổ sung tên và tự động đóng problem cũ, kèm theo lệnh ngừng bán sản phẩm bếp mini `NW-STOVE-MINI`.
  5. **BƯỚC QUAN TRỌNG:** Chọn bản cập nhật cũ gửi muộn: `NW-2025-09-05 — Mid-week corrections (delivered late)` -> Bấm **Start update**.
  6. Mở **Details** của đợt này: Chỉ vào các dòng ghi chú `older than stored data; skipped`.
  7. Bấm sang tab **Local Catalog**, gõ tìm kiếm `NW-STOVE-MINI`: Sản phẩm vẫn giữ trạng thái `withdrawn`, không hề bị chuyển lại thành active.
* **Lời thoại nói:**
  > "Tiếp theo, mình sẽ kiểm tra tính toàn vẹn dữ liệu. Nếu nhân viên lỡ bấm import lại đúng bản cập nhật `09-01` này lần thứ hai: hệ thống hoàn toàn không tạo thêm sản phẩm trùng lặp nào, mà toàn bộ các dòng đều được skip vì thời gian không mới hơn dữ liệu đang có.
  >
  > Giờ mình sẽ chạy bản cập nhật tuần tiếp theo, ngày `09-08`. Bản này cập nhật giá mới và thực hiện ngừng bán sản phẩm bếp mini `NW-STOVE-MINI`.
  >
  > Tình huống phức tạp đặt ra là: điều gì xảy ra nếu một bản cập nhật cũ hơn, ví dụ ngày `09-05`, lại bị gửi đến muộn sau đó?
  > Mình sẽ tiến hành import bản `09-05` này. Nhìn vào lịch sử thay đổi: hệ thống từ chối các mức giá cũ và tuyệt đối KHÔNG hồi sinh chiếc bếp mini đã bị ngừng bán trước đó. Chiếc bếp vẫn ở trạng thái withdrawn vì cơ chế tombstone đã ghi nhớ mốc thời gian rút bán mới hơn."

---

### ⏱️ Phần 4: Sự cố mất mạng & Tiếp tục cập nhật (Resume) (2:40 – 3:50)
* **Thao tác màn hình:**
  1. Chọn bản cập nhật lớn: `NW-2025-09-15 — Autumn collection (6 batches)`.
  2. Bấm **Start update**.
  3. Khi thanh tiến trình đang chạy đến khoảng batch 2/6:
     - Gạt công tắc ở thanh Header: **"Simulate Supplier Outage (503)" sang chế độ OFFLINE**.
  4. Thanh tiến trình dừng lại, chuyển sang tag màu vàng cam: **Interrupted** kèm thông báo lỗi HTTP 503.
  5. Bấm vào tab **Local Catalog**: Cho thấy các sản phẩm thuộc batch 1 và batch 2 đã commit vẫn nằm nguyên vẹn trong danh mục, không hề bị mất mát hay rollback toàn bộ catalog.
  6. Gạt công tắc Outage trở lại **ONLINE**.
  7. Bấm vào nút màu xanh **Resume** ở dòng update đang bị Interrupted.
  8. Tiến trình tự động chạy tiếp từ batch 3, 4, 5, 6 cho đến khi thành công chuyển sang **Completed**.
* **Lời thoại nói:**
  > "Bây giờ là tính năng quan trọng nhất: khả năng chịu lỗi khi mất mạng hoặc tắt ứng dụng.
  > Mình sẽ bắt đầu import bản `09-15`, bản này có tổng cộng 6 batches.
  >
  > Trong lúc hệ thống đang chạy batch 2, mình sẽ giả lập sự cố mất kết nối tới nhà cung cấp bằng công tắc Outage này.
  > Ngay lập tức, đợt cập nhật dừng lại một cách an toàn và chuyển trạng thái thành **Interrupted**. Đề bài yêu cầu bản cập nhật dở dang tuyệt đối không được báo là hoàn thành, và hệ thống đã tuân thủ đúng yêu cầu này.
  > Đồng thời, toàn bộ sản phẩm của các batch 1 và 2 đã lưu trước đó vẫn còn nguyên vẹn trong cơ sở dữ liệu SQLite vì mỗi batch được cam kết trong một transaction độc lập.
  >
  > Khi nhà cung cấp có mạng trở lại, nhân viên chỉ việc bấm vào nút **Resume**.
  > Hệ thống tự động bắt đầu lại chính xác từ batch thứ 3 mà không cần làm lại từ đầu hay làm trùng dữ liệu, và kết thúc trọn vẹn ở trạng thái Completed."

---

### ⏱️ Phần 5: Công cụ dòng lệnh (CLI) & Chạy Test tự động (3:50 – 4:30)
* **Thao tác màn hình:**
  1. Chuyển sang cửa sổ Terminal thứ 3.
  2. Gõ lệnh xem danh sách runs:
     ```powershell
     .\bin\catalog.exe runs
     ```
  3. Gõ lệnh chạy toàn bộ test tự động:
     ```powershell
     go test -v ./internal/catalog ./internal/importer
     ```
  4. Đợi 1-2 giây cho tất cả các bài test in ra kết quả màu xanh `PASS`.
* **Lời thoại nói:**
  > "Bên cạnh giao diện web, nhân viên cũng có thể sử dụng công cụ dòng lệnh `catalog` để tra cứu lịch sử, giải quyết sự cố hoặc kích hoạt cập nhật trực tiếp trong terminal.
  >
  > Cuối cùng, mình sẽ chạy toàn bộ bộ kiểm thử tự động của dự án.
  > Tất cả 18 test case đều vượt qua tuyệt đối (PASS), kiểm tra chặt chẽ:
  > - Các quy tắc merge thời gian và diff trường dữ liệu.
  > - Tính bất biến khi import nhiều lần.
  > - Xử lý dữ liệu đến sai thứ tự và bảo vệ sản phẩm đã ngừng bán.
  > - Phục hồi sau khi mất mạng và phục hồi sau khi ứng dụng bị tắt đột ngột.
  >
  > Cảm ơn mọi người đã theo dõi buổi demo của mình!"

---

## 3. Một số lưu ý nhỏ khi nói

- Giọng đọc rõ ràng, tự tin, không cần đọc quá nhanh.
- Khi bấm nút trên màn hình, dừng chuột lại 1 giây ở kết quả để người xem video kịp nhìn thấy các con số thống kê hoặc các tag màu sắc (`Completed`, `Interrupted`, `Resume`).
- Sau khi quay xong, bạn có thể upload video lên Google Drive (nhớ mở quyền *"Bất kỳ ai có liên kết đều xem được"*) hoặc YouTube (chế độ *"Không công khai" - Unlisted*) rồi gắn link vào email nộp bài.
