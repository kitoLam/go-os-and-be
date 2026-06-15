# Bài 13 — Goroutine & channel

## Mục tiêu
Hiểu đơn vị concurrency của Go (goroutine) và cách chúng giao tiếp (channel).

## Lý thuyết

### Goroutine — "việc" chạy đồng thời
Thêm `go` trước một lời gọi hàm để chạy nó concurrent:
```go
func main() {
	go saySomething("hello")   // chạy đồng thời
	saySomething("world")      // chạy ở goroutine chính
}
```
Goroutine **rất rẻ** (~2KB), tạo hàng nghìn cái thoải mái — khác hẳn thread OS (~1MB). Đây là lý do Go xử lý concurrency tốt.

**Cạm bẫy đầu tiên**: nếu `main` kết thúc, mọi goroutine bị giết ngay, kể cả chưa chạy xong.
```go
func main() {
	go fmt.Println("có thể không kịp in")
	// main thoát ngay -> goroutine trên có thể chưa chạy
}
```
Cần cách "đợi" goroutine — đó là việc của channel (và WaitGroup ở bài 15).

### Goroutine không trả giá trị trực tiếp
```go
result := go compute()   // SAI — không có cú pháp này
```
Goroutine không trả về gì. Muốn lấy kết quả ra, phải dùng channel.

### Channel — đường ống giao tiếp
```go
ch := make(chan int)     // channel truyền int
go func() {
	ch <- 42             // gửi vào channel
}()
v := <-ch                // nhận từ channel (block tới khi có)
fmt.Println(v)           // 42
```
- `ch <- x` : gửi.
- `<-ch` : nhận.
- Channel vừa truyền dữ liệu, vừa **đồng bộ**: `<-ch` chờ tới khi có giá trị, nên đây cũng là cách "đợi" goroutine xong.

### Unbuffered vs buffered
```go
ch := make(chan int)        // unbuffered: gửi block tới khi có người nhận
ch := make(chan int, 5)     // buffered cap 5: gửi chỉ block khi đầy
```
Unbuffered = hẹn gặp trực tiếp (đồng bộ chặt). Buffered = đệm tới N phần tử, sender không cần chờ receiver ngay.

### Đóng channel & range
```go
ch := make(chan int, 3)
go func() {
	for i := 0; i < 3; i++ { ch <- i }
	close(ch)                // báo hết dữ liệu (chỉ bên GỬI đóng)
}()
for v := range ch {          // lặp tới khi channel đóng
	fmt.Println(v)
}
```
Quy tắc: chỉ bên gửi đóng channel; gửi vào channel đã đóng → panic.

### Ví dụ thực tế: xử lý nhiều submission đồng thời
```go
func processAll(ids []string) []string {
	ch := make(chan string, len(ids))
	for _, id := range ids {
		go func(id string) {
			// giả lập xử lý
			time.Sleep(100 * time.Millisecond)
			ch <- "done-" + id
		}(id)                          // LƯU Ý truyền id vào tham số
	}
	var results []string
	for range ids {
		results = append(results, <-ch)
	}
	return results
}
```
**Cạm bẫy closure**: phải truyền `id` vào hàm (`func(id string)`) thay vì dùng biến vòng lặp trực tiếp, nếu không mọi goroutine có thể thấy cùng một giá trị. (Từ Go 1.22 biến vòng lặp đã an toàn hơn, nhưng truyền tham số vẫn là thói quen tốt.)

## Bài tập
1. Chạy 3 goroutine in 3 chuỗi khác nhau. Quan sát thứ tự không cố định.
2. Tạo channel, dùng 1 goroutine gửi 1 số, main nhận và in. Hiểu vì sao main "đợi" được.
3. Viết `processAll` như trên xử lý 10 id. Đo thời gian, so với làm tuần tự (for + sleep) — bản goroutine phải nhanh hơn nhiều.
4. Thử bỏ phần nhận (`<-ch`) đi và để main thoát ngay — quan sát goroutine chưa kịp chạy.
5. Tạo channel buffered cap 3, gửi 3 giá trị rồi `close`, dùng `for range` nhận hết. Thử gửi cái thứ 4 sau khi close → quan sát panic.

## Checklist
- [ ] Chạy goroutine bằng `go f()`
- [ ] Hiểu main thoát thì goroutine chết theo
- [ ] Gửi/nhận qua channel, hiểu nó vừa truyền vừa đồng bộ
- [ ] Phân biệt unbuffered vs buffered
- [ ] Biết quy tắc đóng channel + cạm bẫy closure trong vòng lặp
