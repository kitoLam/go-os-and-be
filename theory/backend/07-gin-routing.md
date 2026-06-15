# Bài 07 — Gin: routing & handler

## Mục tiêu
Dùng Gin để định tuyến gọn gàng, lấy path/query param, nhóm route.

## Lý thuyết

### Cài Gin
```bash
go get github.com/gin-gonic/gin
```

### Server Gin tối giản
```go
package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()       // có sẵn middleware logger + recovery

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	r.Run(":8080")
}
```
So với net/http: gọn hơn nhiều, `c.JSON` tự set Content-Type và encode. `gin.H` là viết tắt của `map[string]any` cho tiện trả JSON.

### `*gin.Context` — trung tâm mọi thứ
Mọi handler nhận `c *gin.Context`, chứa cả request lẫn response:
```go
func handler(c *gin.Context) {
	id := c.Param("id")              // path param /submissions/:id
	status := c.Query("status")      // query ?status=pending
	page := c.DefaultQuery("page", "1") // có giá trị mặc định
	auth := c.GetHeader("Authorization")

	c.JSON(200, gin.H{"id": id})     // trả JSON + status code
	c.Status(204)                    // chỉ status, không body
}
```

### Định tuyến đầy đủ
```go
r.GET("/submissions", listHandler)
r.GET("/submissions/:id", getHandler)     // :id là path param
r.POST("/submissions", createHandler)
r.PUT("/submissions/:id", updateHandler)
r.DELETE("/submissions/:id", deleteHandler)
```

### Nhóm route (route group)
Gom các route cùng tiền tố, tiện cho versioning và gắn middleware chung:
```go
v1 := r.Group("/api/v1")
{
	v1.GET("/submissions", listHandler)
	v1.POST("/submissions", createHandler)
}
// -> /api/v1/submissions
```

### Gắn handler vào struct (chuẩn bị cho kiến trúc tầng)
```go
type SubmissionHandler struct {
	svc *service.SubmissionService
}

func NewSubmissionHandler(svc *service.SubmissionService) *SubmissionHandler {
	return &SubmissionHandler{svc: svc}
}

func (h *SubmissionHandler) Get(c *gin.Context) {
	id := c.Param("id")
	sub, err := h.svc.Get(id)
	if err != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	c.JSON(200, sub)
}

func (h *SubmissionHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/submissions/:id", h.Get)
}
```
Cách này (handler là struct giữ service) sẽ là nền cho bài kiến trúc.

## Bài tập
1. Cài Gin, dựng server có `/ping` trả `{"message":"pong"}`.
2. Tạo route group `/api/v1` chứa: `GET /submissions`, `GET /submissions/:id`, `POST /submissions`.
3. `GET /submissions/:id` lấy `id` từ path và trả lại trong JSON.
4. `GET /submissions` đọc query `?status=` và trả lại giá trị đó.
5. Tạo struct `SubmissionHandler` với method `Get`, `List` và hàm `RegisterRoutes`. Lắp vào `main`. (Service tạm để rỗng, trả dữ liệu giả.)

## Checklist
- [ ] Dựng server Gin, dùng `c.JSON` + `gin.H`
- [ ] Lấy path param (`c.Param`) và query (`c.Query`)
- [ ] Dùng route group
- [ ] Tổ chức handler dạng struct + RegisterRoutes
