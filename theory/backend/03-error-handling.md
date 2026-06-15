# Bài 03 — Error handling

## Mục tiêu
Hiểu triết lý "error là giá trị" của Go, biết tạo/wrap/kiểm tra lỗi đúng cách.

## Lý thuyết

### Error không phải exception
Go không có try/catch. Lỗi là một **giá trị** trả về, bạn xử lý tường minh ngay tại chỗ:
```go
s, err := repo.GetByID(id)
if err != nil {
	// xử lý lỗi
	return err
}
// dùng s an toàn
```
Pattern `if err != nil` xuất hiện khắp nơi. Ban đầu thấy lặp, nhưng nó làm luồng lỗi rõ ràng — không có lỗi nào "bay" ngầm như exception.

### Tạo lỗi
```go
import "errors"

err := errors.New("submission không tồn tại")
err := fmt.Errorf("không tìm thấy submission %s", id)   // có format
```

### Wrap lỗi với `%w` — giữ chuỗi nguyên nhân
Khi lỗi đi qua nhiều tầng, bạn muốn giữ lại lỗi gốc để debug:
```go
func (s *Service) Get(id string) (Submission, error) {
	sub, err := s.repo.GetByID(id)
	if err != nil {
		return Submission{}, fmt.Errorf("service get %s: %w", id, err) // %w wrap
	}
	return sub, nil
}
```
`%w` "gói" lỗi gốc vào trong, sau này lấy ra được.

### Kiểm tra loại lỗi
```go
// so với một lỗi sentinel cụ thể
var ErrNotFound = errors.New("not found")
if errors.Is(err, ErrNotFound) {
	c.JSON(404, ...)
}

// ép về một kiểu lỗi cụ thể để lấy thông tin
type ValidationError struct {
	Field string
	Msg   string
}
func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Msg
}

var ve *ValidationError
if errors.As(err, &ve) {
	fmt.Println("Lỗi ở field:", ve.Field)
}
```
`errors.Is` = "lỗi này CÓ PHẢI là X không" (kể cả khi đã bị wrap). `errors.As` = "lỗi này có thuộc KIỂU X không, nếu có thì lấy ra".

### Sentinel error — định nghĩa sẵn để so sánh
```go
package repository

var (
	ErrNotFound      = errors.New("submission not found")
	ErrAlreadyExists = errors.New("submission already exists")
)
// repo trả về ErrNotFound, tầng trên dùng errors.Is để bắt và map sang HTTP 404
```

### Khi nào panic
`panic` chỉ dành cho lỗi **không thể tiếp tục** (bug lập trình, config thiếu lúc khởi động). Lỗi nghiệp vụ bình thường (không tìm thấy, validate fail) → luôn dùng `error`, không panic.

## Bài tập
1. Định nghĩa sentinel error `ErrNotFound` trong package `repository`.
2. Viết hàm `GetByID(id string) (Submission, error)` trả `ErrNotFound` khi không có.
3. Ở tầng service, gọi hàm trên và wrap lỗi bằng `fmt.Errorf("...: %w", err)`.
4. Ở tầng trên cùng (giả lập handler), dùng `errors.Is(err, repository.ErrNotFound)` để in "404 not found", ngược lại in "500 internal error". Chứng minh `errors.Is` vẫn nhận ra lỗi dù đã bị wrap qua tầng service.
5. Tạo `ValidationError` (Field, Msg), viết hàm validate trả về nó, và dùng `errors.As` để lấy ra tên field bị lỗi.

## Checklist
- [ ] Hiểu error là giá trị, xử lý bằng `if err != nil`
- [ ] Biết wrap lỗi bằng `%w`
- [ ] Phân biệt `errors.Is` (so sánh) và `errors.As` (ép kiểu)
- [ ] Biết khi nào dùng error, khi nào panic
