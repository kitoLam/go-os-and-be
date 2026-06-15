# Bài 01 — Cú pháp cơ bản

## Mục tiêu
Nắm biến, kiểu dữ liệu, hàm, điều kiện, vòng lặp — đủ để viết logic cơ bản.

## Lý thuyết

### Biến
```go
var x int = 5          // khai báo đầy đủ
var y = 10             // tự suy kiểu
z := 15                // ngắn gọn, CHỈ dùng trong hàm
const MaxCode = 65536  // hằng số

var a, b int = 1, 2    // nhiều biến cùng lúc
```
`:=` là cách bạn sẽ dùng nhiều nhất. Lưu ý: biến khai báo mà không dùng → lỗi biên dịch.

### Kiểu dữ liệu cơ bản
```go
bool                      // true / false
string                    // "xin chào"
int, int64                // số nguyên
float64                   // số thực
byte                      // = uint8 (1 byte)
```
**Zero value**: biến chưa gán giá trị tự nhận giá trị mặc định — `0` cho số, `""` cho string, `false` cho bool, `nil` cho pointer/slice/map. Không có "undefined" như JS.

### Hàm
```go
func cong(a int, b int) int {
	return a + b
}

func chia(a, b int) (int, error) {   // trả về nhiều giá trị
	if b == 0 {
		return 0, fmt.Errorf("chia cho 0")
	}
	return a / b, nil
}
```
Đặc trưng Go: hàm **trả về nhiều giá trị**, và quy ước giá trị cuối thường là `error`. Bạn sẽ thấy `value, err := f()` ở khắp nơi.

### Điều kiện
```go
if x > 10 {
	fmt.Println("lớn")
} else if x > 5 {
	fmt.Println("vừa")
} else {
	fmt.Println("nhỏ")
}

// khai báo biến ngay trong if (rất hay dùng)
if v, err := chia(10, 2); err != nil {
	fmt.Println("lỗi:", err)
} else {
	fmt.Println("kết quả:", v)
}
```

### Vòng lặp — Go chỉ có `for`
```go
for i := 0; i < 5; i++ { }          // cổ điển
for x < 10 { x++ }                  // kiểu while
for { break }                       // vô hạn (dùng break để thoát)

nums := []int{10, 20, 30}
for i, v := range nums {            // duyệt slice: i là index, v là giá trị
	fmt.Println(i, v)
}
for _, v := range nums { }          // _ = bỏ qua index
```

### switch
```go
switch lang {
case "cpp":
	fmt.Println("C++")
case "java", "python":              // nhiều giá trị
	fmt.Println("Java hoặc Python")
default:
	fmt.Println("không hỗ trợ")
}
```
Go **không tự fallthrough** — mỗi case tự dừng, không cần `break`.

## Bài tập
1. Viết hàm `tinhDiem(soTestPass, tongTest int) float64` trả về phần trăm test pass.
2. Viết hàm `phanLoai(diem float64) string`: ≥80 → "Giỏi", ≥50 → "Khá", còn lại "Cần cố gắng". Dùng if-else.
3. Viết hàm `kiemTraNgonNgu(lang string) bool` dùng switch, trả true nếu lang thuộc {cpp, java, python}.
4. Viết vòng lặp duyệt slice `[]string{"AC","WA","TLE"}` và in ra index + giá trị.
5. Viết hàm `chiaAnToan(a, b int) (int, error)` trả lỗi khi b=0, và viết đoạn `main` gọi nó với cả b=0 và b=2, xử lý error đúng cách.

## Checklist
- [ ] Phân biệt `var`, `:=`, `const`
- [ ] Hiểu zero value
- [ ] Viết được hàm trả về `(value, error)`
- [ ] Dùng được `for` ở cả 3 dạng + `range`
- [ ] Hiểu switch không fallthrough
