# Bài 02 — Struct, method, interface

## Mục tiêu
Mô hình hóa dữ liệu bằng struct, gắn hành vi bằng method, trừu tượng hóa bằng interface.

## Lý thuyết

### Struct — gom dữ liệu liên quan
```go
type Submission struct {
	ID       string
	UserID   string
	Language string
	Code     string
	Status   string
}

// tạo instance
s := Submission{ID: "1", Language: "cpp", Code: "..."}
s2 := Submission{}              // mọi field = zero value
fmt.Println(s.Language)         // truy cập field
```
Struct giống "class chỉ có field" trong các ngôn ngữ khác. Đây là cách bạn biểu diễn một bản ghi submission, user, request...

### Method — gắn hành vi vào struct
```go
// receiver value (s là bản sao)
func (s Submission) IsEmpty() bool {
	return s.Code == ""
}

// receiver pointer (sửa được bản gốc)
func (s *Submission) SetStatus(st string) {
	s.Status = st
}

s.SetStatus("PENDING")          // gọi method
```
Quy tắc: dùng `*Submission` (pointer) khi cần **sửa** struct hoặc struct lớn; dùng `Submission` (value) khi chỉ đọc và struct nhỏ. Trong một type nên thống nhất.

### Interface — định nghĩa "khả năng", không quan tâm kiểu cụ thể
Đây là khái niệm quan trọng nhất cho việc viết code dễ test và dễ mở rộng.
```go
type SubmissionRepository interface {
	Create(s Submission) error
	GetByID(id string) (Submission, error)
}
```
Interface nói "bất cứ thứ gì có 2 method này đều là một SubmissionRepository". Đặc biệt: trong Go bạn **không cần khai báo `implements`** — chỉ cần có đủ method là tự động thỏa interface:
```go
type PostgresRepo struct{ /* ... */ }
func (r PostgresRepo) Create(s Submission) error          { /* ... */ return nil }
func (r PostgresRepo) GetByID(id string) (Submission, error) { /* ... */ return Submission{}, nil }
// PostgresRepo tự động LÀ SubmissionRepository, không cần khai báo thêm
```

### Vì sao interface quan trọng
Code của bạn phụ thuộc vào **interface** thay vì kiểu cụ thể → đổi implementation dễ dàng, và test dễ (thay repo thật bằng repo giả):
```go
type Service struct {
	repo SubmissionRepository    // phụ thuộc interface, không phải PostgresRepo
}
// lúc chạy thật: truyền PostgresRepo
// lúc test: truyền FakeRepo (cũng thỏa interface)
```

### Embedding — tái sử dụng bằng cách nhúng
```go
type Timestamps struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}
type Submission struct {
	ID string
	Timestamps        // nhúng -> truy cập s.CreatedAt trực tiếp
}
```

## Bài tập
1. Định nghĩa struct `User` (ID, Email, Role) và struct `Submission` như trên.
2. Viết method `(s Submission) ShortCode() string` trả về 20 ký tự đầu của code (cẩn thận khi code ngắn hơn 20).
3. Viết method pointer `(s *Submission) MarkAccepted()` set Status = "AC".
4. Định nghĩa interface `Notifier` có method `Notify(msg string) error`. Viết 2 type thỏa nó: `EmailNotifier` và `LogNotifier` (chỉ in ra màn hình).
5. Viết hàm `func sendAll(n Notifier, msgs []string)` nhận bất kỳ Notifier nào và gửi từng msg — chứng minh bạn truyền được cả 2 type vào cùng một hàm.

## Checklist
- [ ] Tạo struct và truy cập field
- [ ] Phân biệt method receiver value vs pointer
- [ ] Hiểu interface là "tập method", thỏa ngầm (không cần implements)
- [ ] Hiểu vì sao phụ thuộc interface giúp dễ test
