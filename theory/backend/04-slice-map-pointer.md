# Bài 04 — Slice, map, pointer

## Mục tiêu
Thành thạo 2 cấu trúc dữ liệu dùng nhiều nhất (slice, map) và hiểu pointer.

## Lý thuyết

### Slice — mảng động
```go
var s []int                  // nil slice (rỗng)
s = []int{1, 2, 3}           // khởi tạo
s = append(s, 4)             // thêm phần tử -> [1 2 3 4]
fmt.Println(len(s))          // độ dài
fmt.Println(s[0], s[1:3])    // truy cập, cắt lát [1:3] = [2 3]

// cấp phát trước để tối ưu (biết trước số lượng)
results := make([]TestResult, 0, 100)   // len 0, cap 100
```
Slice là cách bạn lưu danh sách: danh sách submission, danh sách test case, danh sách kết quả.

### Map — key-value
```go
m := map[string]int{}            // map rỗng
m := make(map[string]int)        // tương đương
m["AC"] = 5
m["WA"] = 2

v := m["AC"]                     // 5
v, ok := m["XYZ"]                // ok=false vì key không tồn tại (v = 0)
delete(m, "AC")

for k, v := range m {            // duyệt (thứ tự NGẪU NHIÊN)
	fmt.Println(k, v)
}
```
**Quan trọng**: dùng `v, ok := m[key]` để phân biệt "key không có" với "key có giá trị zero". Map dùng nhiều cho config, đếm, lookup nhanh.

```go
// ví dụ thật: config ngôn ngữ
langs := map[string]string{
	"cpp":    "g++",
	"python": "python3",
}
compiler, ok := langs[lang]
if !ok {
	return errors.New("ngôn ngữ không hỗ trợ")
}
```

### Pointer — trỏ tới ô nhớ
```go
x := 10
p := &x          // p là pointer tới x (& = lấy địa chỉ)
fmt.Println(*p)  // 10 (* = lấy giá trị tại địa chỉ)
*p = 20          // sửa qua pointer -> x = 20
```
Vì sao cần pointer:
1. **Sửa được giá trị gốc** trong hàm (Go truyền tham số theo bản sao):
```go
func reset(s *Submission) { s.Status = "PENDING" }
reset(&sub)      // sub.Status thật sự đổi
```
2. **Tránh copy struct lớn** (truyền địa chỉ thay vì copy toàn bộ).
3. **Biểu diễn "có thể nil"**: `*Submission` có thể là nil (chưa có), `Submission` thì luôn tồn tại.

Go có garbage collector — không cần giải phóng bộ nhớ thủ công. Không có pointer arithmetic như C.

### Lưu ý slice là tham chiếu
```go
a := []int{1, 2, 3}
b := a
b[0] = 99        // a[0] CŨNG thành 99 (cùng backing array)
```
Slice chia sẻ dữ liệu nền — nếu cần bản sao độc lập phải `copy()`.

## Bài tập
1. Tạo slice `[]Submission`, thêm 3 submission bằng `append`, in ra số lượng.
2. Viết hàm `demVerdict(results []string) map[string]int` đếm số lần mỗi verdict xuất hiện (vd `["AC","WA","AC"]` → `{AC:2, WA:1}`).
3. Tạo map `langs` ánh xạ ngôn ngữ → lệnh compile. Viết hàm tra cứu trả về `(cmd string, ok bool)`.
4. Viết hàm `markAllPending(subs []Submission)` set tất cả Status = "PENDING". Để nó sửa được bản gốc, suy nghĩ: nên dùng `[]Submission` hay `[]*Submission`? Thử cả hai và quan sát.
5. Chứng minh "slice chia sẻ dữ liệu": gán `b := a`, sửa `b`, in `a` để thấy nó cũng đổi.

## Checklist
- [ ] Tạo, append, cắt slice; biết `make` với cap
- [ ] Dùng map + cú pháp `v, ok := m[key]`
- [ ] Hiểu `&` và `*`, biết khi nào cần pointer
- [ ] Biết slice là tham chiếu (chia sẻ backing array)
