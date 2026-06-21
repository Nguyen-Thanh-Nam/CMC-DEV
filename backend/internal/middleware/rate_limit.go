package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// rateLimiter quản lý danh sách và trạng thái giới hạn tần suất của tất cả các IP máy khách truy cập hệ thống.
type rateLimiter struct {
	// mu dùng để đồng bộ hóa (Mutex) để đảm bảo an toàn đa luồng (thread-safety) khi truy cập map visitors từ nhiều goroutines đồng thời.
	mu       sync.Mutex
	// visitors lưu trữ thông tin về số lượng yêu cầu của từng địa chỉ IP máy khách. Key là IP dạng chuỗi.
	visitors map[string]*visitor
}

// visitor chứa dữ liệu chi tiết của từng máy khách để tính toán rate limit.
type visitor struct {
	// lastSeen lưu mốc thời gian cuối cùng mà máy khách gửi yêu cầu lên hệ thống.
	lastSeen time.Time
	// count đếm số lượng yêu cầu (requests) mà máy khách đã thực hiện trong chu kỳ hiện tại.
	count    int
}

// limiter là thực thể rateLimiter duy nhất (Singleton) được khởi tạo toàn cục để kiểm soát tần suất truy cập toàn bộ hệ thống.
var limiter = &rateLimiter{
	visitors: make(map[string]*visitor),
}

const (
	// maxReqPerMin là tần suất giới hạn tối đa: 60 yêu cầu mỗi phút cho mỗi địa chỉ IP.
	maxReqPerMin = 60
)

// Hàm init tự động chạy khi package được nạp.
// Hàm này kích hoạt một goroutine chạy ngầm (background worker) để dọn dẹp tài nguyên.
func init() {
	go cleanupVisitors()
}

// cleanupVisitors là một tiến trình chạy ngầm vô hạn, định kỳ dọn dẹp các máy khách không hoạt động
// nhằm tránh rò rỉ bộ nhớ (memory leak) khi bản đồ visitors ngày càng phình to.
func cleanupVisitors() {
	for {
		time.Sleep(1 * time.Minute)
		
		limiter.mu.Lock()
		for ip, v := range limiter.visitors {
			if time.Since(v.lastSeen) > 1*time.Minute {
				delete(limiter.visitors, ip)
			}
		}
		limiter.mu.Unlock()
	}
}

// RateLimit là một middleware đóng vai trò lọc (interceptor) giới hạn tần suất yêu cầu.
// Nó nhận vào một handler và trả về một handler mới đã được bọc logic kiểm tra tần suất.
func RateLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := strings.Split(r.RemoteAddr, ":")[0]

		limiter.mu.Lock()
		v, exists := limiter.visitors[ip]
		
		if !exists {
			limiter.visitors[ip] = &visitor{lastSeen: time.Now(), count: 1}
			limiter.mu.Unlock()
			next.ServeHTTP(w, r)
			return
		}

		if time.Since(v.lastSeen) > 1*time.Minute {
			v.count = 0
		}
		
		v.count++
		v.lastSeen = time.Now()

		if v.count > maxReqPerMin {
			limiter.mu.Unlock()
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		
		limiter.mu.Unlock()

		next.ServeHTTP(w, r)
	}
}

