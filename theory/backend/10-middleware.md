# Bài 10 — Middleware

## Mục tiêu
Hiểu và viết middleware: logging, recovery, auth, request ID.

## Lý thuyết

### Middleware là gì
Một lớp xử lý **chạy trước/sau** handler chính, áp cho nhiều route. Dùng cho những việc lặp lại ở mọi request: ghi log, kiểm tra token, gắn request ID, bắt panic. Giống `app.use()` trong Express.

### Cấu trúc một middleware Gin
```go
func MyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// --- code chạy TRƯỚC handler ---
		c.Next()                   // gọi handler tiếp theo
		// --- code chạy SAU handler ---
	}
}
```
`c.Next()` chuyển quyền cho middleware/handler kế tiếp. `c.Abort()` chặn lại, không cho chạy tiếp.

### Gắn middleware
```go
r := gin.New()                       // không có middleware mặc định
r.Use(LoggingMiddleware())           // áp toàn cục
r.Use(RecoveryMiddleware())

v1 := r.Group("/api/v1")
v1.Use(AuthMiddleware())             // áp cho riêng nhóm này
```
(`gin.Default()` đã tự gắn sẵn Logger + Recovery; dùng `gin.New()` khi muốn tự kiểm soát.)

### Middleware logging (đo thời gian)
```go
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()                                  // chạy handler
		log.Printf("%s %s -> %d (%v)",
			c.Request.Method, c.Request.URL.Path,
			c.Writer.Status(), time.Since(start))
	}
}
```

### Middleware recovery (bắt panic)
```go
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic: %v", r)
				c.JSON(500, gin.H{"error": "internal server error"})
				c.Abort()
			}
		}()
		c.Next()
	}
}
```
Nếu handler panic, recovery bắt được, server không sập, client nhận 500 thay vì rớt kết nối.

### Middleware auth
```go
func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		userID, err := verifyToken(token, secret)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
			return                              // c.Abort + không Next
		}
		c.Set("userID", userID)                 // truyền xuống handler
		c.Next()
	}
}

// trong handler lấy ra:
func (h *Handler) Create(c *gin.Context) {
	userID := c.GetString("userID")
}
```

### Request ID (truy vết)
Gắn 1 ID duy nhất cho mỗi request để log dễ lần theo:
```go
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := uuid.New().String()
		c.Set("request_id", rid)
		c.Header("X-Request-ID", rid)
		c.Next()
	}
}
```

## Bài tập
1. Viết `LoggingMiddleware` in ra method, path, status, thời gian xử lý. Gắn toàn cục.
2. Viết `RecoveryMiddleware`. Tạo một route cố tình `panic("test")` và xác nhận server vẫn sống, client nhận 500.
3. Viết `AuthMiddleware` đơn giản: kiểm tra header `Authorization` có giá trị `"secret123"` không; nếu không → 401. Gắn cho group `/api/v1`.
4. Trong middleware auth, `c.Set("userID", "user-1")`, rồi trong handler lấy ra bằng `c.GetString("userID")` và trả về.
5. Viết `RequestIDMiddleware`, xác nhận response có header `X-Request-ID` khác nhau mỗi lần gọi.

## Checklist
- [ ] Hiểu middleware chạy trước/sau handler, vai trò `c.Next()` / `c.Abort()`
- [ ] Gắn middleware toàn cục và theo group
- [ ] Viết được logging, recovery, auth
- [ ] Truyền dữ liệu từ middleware xuống handler bằng `c.Set`/`c.Get`
