package middleware

import "net/http"

// CORSMiddleware là middleware cấu hình Cross-Origin Resource Sharing (CORS).
// Nó cho phép các ứng dụng Frontend chạy ở các origin khác (như localhost:3000)
// có thể gửi yêu cầu HTTP đến Backend một cách hợp lệ mà không bị trình duyệt chặn.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

