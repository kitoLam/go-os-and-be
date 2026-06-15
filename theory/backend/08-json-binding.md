# Bài 08 — JSON: nhận và trả dữ liệu

## Mục tiêu
Bind JSON từ request body vào struct, và trả JSON về client đúng cách.

## Lý thuyết

### Struct tag cho JSON
Go ánh xạ field struct ↔ JSON qua **tag**:
```go
type CreateSubmissionRequest struct {
	Language string `json:"language"`
	Code     string `json:"code"`
	UserID   string `json:"user_id"`
}
```
- `json:"language"` → field `Language` (Go) ↔ key `"language"` (JSON).
- Field phải **viết hoa** (export) thì mới được serialize.

Tag hữu ích khác:
```go
type Submission struct {
	ID       string `json:"id"`
	Note     string `json:"note,omitempty"`  // bỏ qua nếu rỗng
	Internal string `json:"-"`               // không bao giờ serialize
}
```

### Bind JSON từ request (Gin)
```go
func (h *Handler) Create(c *gin.Context) {
	var req CreateSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {  // parse body -> struct
		c.JSON(400, gin.H{"error": "JSON không hợp lệ"})
		return
	}
	// dùng req.Language, req.Code...
	c.JSON(201, gin.H{"id": "new-id"})
}
```
`ShouldBindJSON` đọc body, parse JSON, đổ vào struct. Nếu JSON sai cú pháp → trả error.

### Trả JSON
```go
c.JSON(200, submission)              // serialize struct
c.JSON(201, gin.H{"id": id})         // hoặc map tùy ý
c.JSON(404, gin.H{"error": "not found"})
```

### Tách request DTO khỏi model
Thực hành tốt: dùng struct **request** riêng (chỉ chứa thứ client gửi) tách khỏi struct **model/entity** (lưu DB, có thêm ID, timestamp, status). Tránh để client set những field nhạy cảm:
```go
// nhận từ client
type CreateSubmissionRequest struct {
	Language string `json:"language"`
	Code     string `json:"code"`
}

// model nội bộ
type Submission struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Language  string    `json:"language"`
	Code      string    `json:"code"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
```
Handler chuyển request → model, gắn ID/UserID/Status từ phía server, không tin client.

### Số trong JSON
JSON không phân biệt int/float — khi unmarshal vào `any`, số thành `float64`. Nên khai báo kiểu cụ thể trong struct (`int`, `int64`) để tránh bất ngờ.

## Bài tập
1. Định nghĩa `CreateSubmissionRequest` (language, code) và model `Submission` (đầy đủ id, status, created_at).
2. Viết handler `POST /submissions`: bind request, nếu JSON lỗi trả 400.
3. Trong handler, chuyển request → model: tự sinh `ID` (dùng `time.Now().UnixNano()` tạm), set `Status="PENDING"`, `CreatedAt=time.Now()`. Trả model về client với 201.
4. Test bằng `curl -X POST localhost:8080/submissions -d '{"language":"cpp","code":"int main(){}"}'`.
5. Thêm tag `json:"-"` cho một field "secret" trong model, xác nhận nó không xuất hiện trong response.

## Checklist
- [ ] Hiểu struct tag `json:"..."`, omitempty, `-`
- [ ] Bind body bằng `ShouldBindJSON`
- [ ] Trả JSON bằng `c.JSON`
- [ ] Tách request DTO khỏi model nội bộ
