# Bài 06 — HTTP server bằng net/http

## Mục tiêu
Dựng HTTP server bằng thư viện chuẩn để hiểu gốc rễ, trước khi dùng framework.

## Lý thuyết

### Vì sao học net/http trước
Khác với Node (module `http` quá thô, gần như buộc dùng Express), standard library `net/http` của Go đủ mạnh để làm API thật. Hiểu nó giúp bạn biết Gin/Echo che giấu cái gì bên dưới — và nhiều service nhỏ dùng thẳng net/http luôn.

### Server tối giản
```go
package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "pong")
	})
	http.ListenAndServe(":8080", nil)
}
```
- `w http.ResponseWriter` = nơi bạn ghi response.
- `r *http.Request` = thông tin request đến (method, header, body, URL).
- `HandleFunc` đăng ký một hàm xử lý cho một path.
- `ListenAndServe(":8080", nil)` khởi động server ở cổng 8080.

Chạy rồi mở trình duyệt `localhost:8080/ping` hoặc `curl localhost:8080/ping`.

### Đọc thông tin request
```go
func handler(w http.ResponseWriter, r *http.Request) {
	method := r.Method                  // "GET", "POST"...
	q := r.URL.Query().Get("status")    // ?status=pending
	body, _ := io.ReadAll(r.Body)       // đọc body thô
	auth := r.Header.Get("Authorization")
}
```

### Trả JSON
```go
func getHandler(w http.ResponseWriter, r *http.Request) {
	sub := Submission{ID: "1", Status: "AC"}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)        // 200
	json.NewEncoder(w).Encode(sub)      // ghi JSON ra response
}
```

### Phân biệt method
```go
func subHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// trả về
	case http.MethodPost:
		// tạo mới
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
```

### Vì sao người ta vẫn dùng framework
Với net/http thuần, bạn phải tự xử lý: routing theo method + path param (`/submissions/:id`), parse JSON lặp đi lặp lại, gom middleware... Framework như Gin làm sẵn những việc đó cho gọn. Bài sau ta chuyển sang Gin — nhưng giờ bạn đã hiểu nó dựng trên cái gì.

## Bài tập
1. Viết server có route `/health` trả về `{"status":"ok"}` dạng JSON.
2. Viết route `/echo` đọc query param `?msg=hello` và trả lại `{"echo":"hello"}`.
3. Viết route `/submissions` phân biệt GET (trả danh sách giả) và POST (đọc JSON body in ra console), dùng `switch r.Method`.
4. Thử `curl` cả GET và POST tới `/submissions`, quan sát.
5. Thêm xử lý: nếu POST mà body rỗng → trả 400 với `http.Error`.

## Checklist
- [ ] Dựng được server net/http, đăng ký route
- [ ] Hiểu `ResponseWriter` và `*Request`
- [ ] Đọc query/header/body từ request
- [ ] Trả JSON đúng cách (Content-Type + Encode)
- [ ] Hiểu framework giúp gì so với net/http thuần
