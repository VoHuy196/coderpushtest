# Video Demo Script (3–5 Minutes)
## Supplier Catalog Updates Assessment

Tài liệu này hướng dẫn chi tiết từng bước để quay video demo (3–5 phút) nộp bài đánh giá. Bao gồm: **chuẩn bị trước khi quay**, **thời gian bấm giờ**, **thao tác trên màn hình**, và **lời thoại (Voiceover script bằng tiếng Anh & ghi chú tiếng Việt)**.

---

## 1. Chuẩn bị trước khi bấm REC (Bật quay)

1. **Reset Database sạch (để bắt đầu từ đầu):**
   Mở PowerShell tại thư mục dự án và chạy:
   ```powershell
   Remove-Item -Force -ErrorAction SilentlyContinue catalog.db, catalog.db-shm, catalog.db-wal
   ```
2. **Khởi động 2 service:**
   - **Terminal 1:**
     ```powershell
     .\bin\fakesupplier.exe -addr :8081 -delay 500ms
     ```
   - **Terminal 2:**
     ```powershell
     .\bin\server.exe -addr :8080 -supplier http://localhost:8081 -static web/dist
     ```
3. **Mở trình duyệt:**
   - Mở sẵn tab `http://localhost:8080` (Giao diện chính).
   - Mở sẵn 1 cửa sổ Terminal thứ 3 để sẵn sàng gõ lệnh `go test` ở phần cuối.
4. **Phần mềm quay màn hình:** Dùng OBS Studio, Loom, hoặc phím tắt Windows `Win + Alt + R` (Xbox Game Bar).

---

## 2. Kịch bản chi tiết theo thời gian (Timeline)

### ⏱️ Phần 1: Giới thiệu tổng quan (0:00 – 0:40)
* **Màn hình:** Quay tab giao diện `http://localhost:8080`.
* **Thao tác:** Rê chuột nhẹ qua 3 Card thống kê (Active Products: 0, Withdrawn: 0, Problems: 0) và thanh Header.
* **Lời thoại (Voiceover):**
  > "Hello everyone. Today I'm demonstrating my solution for the Supplier Catalog Updates assessment built in Go, SQLite, and a React Ant Design dashboard.
  >
  > The system maintains a local copy of a supplier's catalog. It solves 3 key challenges:
  > 1. Resilient batch imports that survive outages and application restarts without losing completed work.
  > 2. Idempotent versioning based on `updated_at` timestamps, ensuring out-of-order or duplicate records never overwrite newer data.
  > 3. Tombstone preservation so withdrawn products are never accidentally brought back to life."

---

### ⏱️ Phần 2: Import bản cập nhật đầu tiên & Xem Thay đổi / Problems (0:40 – 1:30)
* **Màn hình:** Box "Start an update".
* **Thao tác:**
  1. Trong dropdown, chọn: `NW-2025-09-01 — Initial full catalog (3 batches)`.
  2. Bấm nút **Start update**.
  3. Chỉ vào thanh progress bar đang chạy: Batch 1/3, 2/3, 3/3 -> Chuyển sang tag xanh **Completed**.
  4. Bấm nút **Details** ở dòng Update #1.
  5. Trong Drawer hiện ra, bấm qua các tab filter `created (11)`, `skipped (1)`, `invalid (1)`.
  6. Đóng Drawer, bấm sang Tab **Problems (1)** ở màn hình chính.
* **Lời thoại (Voiceover):**
  > "First, let's start the initial catalog update: `NW-2025-09-01`.
  > As you can see, batches are processed asynchronously with live progress.
  >
  > When it completes, we can click **Details** to see the full change log:
  > - 11 products were created.
  > - 1 repeated record was safely skipped.
  > - And 1 malformed record without a name was caught.
  >
  > Under the **Problems** tab, notice this invalid record is quarantined for staff attention rather than breaking the entire batch or crashing the import."

---

### ⏱️ Phần 3: Chống trùng lặp (Idempotency) & Bảo vệ dữ liệu mới (1:30 – 2:30)
* **Màn hình:** Form Start update & Bảng Runs.
* **Thao tác:**
  1. Trong dropdown, chọn lại chính `NW-2025-09-01` một lần nữa -> Bấm **Start update**.
  2. Chờ hoàn thành: Chỉ vào số liệu `+0 created, ~0 updated, skip 12`.
  3. Tiếp theo, chọn bản update tiếp theo `NW-2025-09-08 — Weekly changes (3 batches)` -> Bấm **Start update**.
  4. Chờ hoàn thành: Chỉ vào số liệu có cả `updated`, `withdrawn` và sản phẩm chair được sửa tên (tự động resolve problem cũ).
  5. **QUAN TRỌNG:** Chọn bản cập nhật cũ đến muộn `NW-2025-09-05 — Mid-week corrections (delivered late)` -> Bấm **Start update**.
  6. Chờ xong, mở **Details**: Chỉ vào các dòng ghi chú `older than stored data; skipped`.
  7. Bấm sang Tab **Local Catalog**, tìm kiếm `NW-STOVE-MINI` -> Vẫn là `withdrawn`, không bị hồi sinh.
* **Lời thoại (Voiceover):**
  > "Now, let's test idempotency. If we import `NW-2025-09-01` a second time: zero duplicates are created, and all records are skipped because their timestamp matches stored data.
  >
  > Next, we run `NW-2025-09-08` which applies weekly price changes and marks `NW-STOVE-MINI` as withdrawn.
  >
  > Now, what if an older update, `NW-2025-09-05`, arrives late and out of order?
  > Let's import it. Notice in the change log: the late update cannot overwrite newer prices, and it CANNOT restore the withdrawn stove. The stove remains withdrawn because our tombstone timestamp guards against older data."

---

### ⏱️ Phần 4: Mô phỏng sự cố mạng (Outage), Không mất dữ liệu & Resume (2:30 – 3:45)
* **Màn hình:** Header & Start update.
* **Thao tác:**
  1. Chọn bản update lớn `NW-2025-09-15 — Autumn collection (6 batches)`.
  2. Bấm **Start update**.
  3. Khi thanh tiến trình đang chạy được khoảng batch 2/6:
     - Gạt ngay công tắc trên Header: **"Simulate Supplier Outage (503)" sang OFFLINE**.
  4. Thanh tiến trình dừng lại, chuyển sang tag vàng **Interrupted** với thông báo lỗi HTTP 503.
  5. Bấm tab **Local Catalog**: Cho người xem thấy các sản phẩm từ batch 1 và 2 vẫn còn nguyên vẹn trong database, không bị mất mát.
  6. Gạt công tắc lại sang **ONLINE**.
  7. Bấm nút xanh **Resume** ở dòng Update bị Interrupted.
  8. Thanh tiến trình chạy tiếp tục từ batch 3, 4, 5, 6 -> **Completed**.
* **Lời thoại (Voiceover):**
  > "Now let's test fault tolerance during an outage or application interruption.
  > I'll start `NW-2025-09-15` which has 6 batches.
  >
  > While it's running, I'll toggle our simulated supplier outage to OFFLINE.
  > The update halts cleanly with the status **Interrupted**. 
  > Notice that:
  > - The partial update is NEVER reported as completed.
  > - The catalog is NOT erased: all products committed up to batch 2 are completely safe in SQLite because each batch is its own transaction.
  >
  > Once the supplier is back ONLINE, staff simply clicks **Resume**.
  > The worker picks up right from batch 3 without repeating prior work, and completes successfully."

---

### ⏱️ Phần 5: CLI & Automated Tests (3:45 – 4:30)
* **Màn hình:** Chuyển sang màn hình Terminal.
* **Thao tác:**
  1. Gõ lệnh CLI xem catalog:
     ```bash
     .\bin\catalog.exe runs
     ```
  2. Gõ lệnh chạy toàn bộ test tự động:
     ```bash
     go test -v ./internal/catalog ./internal/importer
     ```
  3. Đợi vài giây cho toàn bộ test hiện chữ `PASS` màu xanh.
* **Lời thoại (Voiceover):**
  > "In addition to the Web UI, staff can also use the `catalog` CLI tool to inspect runs, changes, problems, or trigger updates directly in the terminal.
  >
  > Finally, let's run our automated test suite.
  > All 18 unit and integration tests pass, covering:
  > - Pure timestamp merge rules and diffing.
  > - Idempotent re-imports.
  > - Out-of-order deliveries.
  > - Mid-run outage recovery.
  > - And application crash recovery after abrupt restarts.
  >
  > Thank you for watching!"

---

## 3. Mẹo khi quay video

- **Âm thanh:** Dùng mic tai nghe rõ tiếng, không bị rè. Nói với tốc độ vừa phải, tự tin.
- **Độ phân giải:** 1080p (1920x1080), phóng to trình duyệt tầm 110% (Ctrl + lăn chuột) để chữ và các tag trạng thái hiển thị rõ nét trên video.
- **Upload:** Tải video lên YouTube (chế độ **Unlisted** - Không công khai) hoặc Google Drive (nhớ bật quyền *"Bất kỳ ai có đường liên kết đều có thể xem"*), sau đó chèn link video vào file `README.md` hoặc email nộp bài.
