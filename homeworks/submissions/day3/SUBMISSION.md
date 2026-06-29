# Homework Submission - Day 3

**Họ tên:** Nguyễn Thành Nam

## Các bài đã hoàn thành

- [x] Bài 1: Migrate sang Database (MySQL)
- [x] Bài 2: Mở rộng Scan API (ip, port, ssl, tech)
- [x] Bài 3: Viết Unit Tests
- [x] Bài 4: Tích hợp Frontend
- [x] Bài 5: CI/CD với GitHub Actions (Bonus)
- [x] Bài 6: Deploy với Docker Compose (Bonus)
- [x] Bài 7: Tính năng EASM mới (Bonus - Asset Tags & Scheduled Scans)
- [x] Bài 8: Deploy lên Cloud VM (Bonus)
- [ ] Bài 9: Domain & TLS/HTTPS (Bonus)
- [ ] Bài 10: Auto Deploy on Merge (Bonus)

---

## Cách khởi chạy

Khởi chạy toàn bộ hệ thống gồm MySQL Database, Go Backend API và Nginx Frontend:

```bash
docker compose up --build -d
```

Các cổng kết nối sau khi chạy:

- **Frontend Dashboard:** http://localhost:3000 (Giao diện Premium Dark Glassmorphism)
- **Backend API Server:** http://localhost:8080
- **MySQL Database:** localhost:3306

---

## Chi tiết và Hướng dẫn kiểm thử Bài 7 (Tính năng EASM mới)

Dự án đã triển khai thành công 2 tính năng nâng cao (Bonus) cho hệ thống EASM:

### 1. Phân loại tài sản bằng thẻ nhãn (Asset Tags)

- **Mô tả:** Người dùng có thể phân loại các tài sản theo mức độ quan trọng hoặc bộ phận sở hữu (ví dụ: `production`, `critical`, `frontend`).
- **Cách hoạt động:**
  - Thêm nhãn mới tại Form tạo Asset (ô **Tags (comma separated)**, cách nhau bằng dấu phẩy).
  - Hiển thị danh sách tags trực quan dưới dạng các nhãn nhỏ (badges) đầy màu sắc trên cột **Tags** của bảng Asset.
  - Bộ lọc thông minh **Filter by Tag...** tại trang tổng quan giúp tìm kiếm nhanh các asset có nhãn tương ứng.

### 2. Tự động quét định kỳ (Scheduled Scans)

- **Mô tả:** Tự động lên lịch và chạy ngầm các tiến trình quét cho toàn bộ tài sản đang có trạng thái `active` mà không cần người dùng thao tác thủ công.
- **Cách hoạt động:**
  - Khi khởi động Server Go, một goroutine chạy ngầm sẽ được kích hoạt tại `main.go`: `scanSvc.StartScheduledScans(5 * time.Minute)`.
  - Cứ mỗi 5 phút, hệ thống tự động tìm các Asset `active` chưa có tiến trình quét đang chạy để tạo và đẩy các lượt quét mới vào hàng đợi (quét `dns` cho Domain, quét `ip` cho IP).
  - Nhật ký quét được lưu trữ trong DB MySQL và hiển thị tại trang **Global Scan Logs** (Tổng quan -> Scan Logs).
