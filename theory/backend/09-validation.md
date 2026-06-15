# Bài 09 — Validation: kiểm tra dữ liệu đầu vào

## Mục tiêu
Tự động kiểm tra request hợp lệ trước khi xử lý, trả lỗi rõ ràng cho client.

## Lý thuyết

### Vì sao cần
Không bao giờ tin dữ liệu client gửi lên. Code rỗng? Ngôn ngữ không hỗ trợ? Code quá dài (cố làm tràn bộ nhớ)? Phải chặn ngay ở cửa, trước khi đụng tới service/DB.

### Validation bằng tag (Gin tích hợp sẵn)
Gin dùng thư viện `validator` qua tag `binding`:
```go
type CreateSubmissionRequest struct {
	Language string `json:"language" binding:"required,oneof=cpp java python"`
	Code     string `json:"code"     binding:"required,max=65536"`
	UserID   string `json:"user_id"  binding:"required,uuid"`
}
```
Khi gọi `c.ShouldBindJSON(&req)`, Gin tự kiểm tra theo tag. Nếu sai → trả error.

### Các luật hay dùng
```
required          // bắt buộc có
omitempty         // bỏ qua nếu rỗng (không validate field rỗng)
min=1 / max=100   // độ dài chuỗi hoặc giá trị số
len=10            // độ dài chính xác
oneof=a b c       // phải thuộc tập giá trị
email             // định dạng email
uuid              // định dạng UUID
gte=0 / lte=100   // >= / <=
```

### Xử lý lỗi validation cho đẹp
Mặc định `err.Error()` khá khó đọc. Bạn có thể trả gọn:
```go
func (h *Handler) Create(c *gin.Context) {
	var req CreateSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"error":   "dữ liệu không hợp lệ",
			"details": err.Error(),
		})
		return
	}
	// ...
}
```

### Validation nghiệp vụ (vượt quá tag)
Tag chỉ kiểm tra được dạng. Luật nghiệp vụ phức tạp hơn (vd "user này có quyền nộp bài cho contest này không") thì viết tay trong **service**, không nhồi vào handler. Phân tầng:
- Handler: validate **dạng** (tag) — đúng kiểu, đủ field, đúng format.
- Service: validate **nghiệp vụ** — quyền, trạng thái, ràng buộc logic.

### Custom validation tag (nâng cao)
Bạn có thể đăng ký luật riêng, ví dụ kiểm tra code không chứa chuỗi cấm. Để dành khi cần — ban đầu các tag có sẵn là đủ.

## Bài tập
1. Thêm tag `binding` vào `CreateSubmissionRequest`: language phải thuộc {cpp, java, python}, code bắt buộc và tối đa 65536 ký tự.
2. Test các case sai: thiếu `code`, `language="rust"` (không trong oneof), code rỗng — xác nhận đều trả 400.
3. Trả lỗi dưới dạng JSON `{"error": "...", "details": "..."}`.
4. Thêm field `time_limit_ms int` với `binding:"gte=100,lte=10000"`. Test giá trị ngoài khoảng.
5. Viết một validation **nghiệp vụ** trong service: hàm `Create` trả lỗi nếu `language` hợp lệ nhưng `code` chỉ chứa khoảng trắng (dùng `strings.TrimSpace`). Chứng minh đây là luật không thể biểu diễn bằng tag.

## Checklist
- [ ] Dùng tag `binding` để validate dạng
- [ ] Biết các luật required/oneof/max/min/email/uuid
- [ ] Trả lỗi validation rõ ràng cho client (400)
- [ ] Phân biệt validate dạng (handler) vs nghiệp vụ (service)
