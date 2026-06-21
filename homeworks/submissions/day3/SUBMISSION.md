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
- [ ] Bài 8: Deploy lên Cloud VM (Bonus)
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
