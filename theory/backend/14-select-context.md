# Bài 14 — select, context, timeout

## Mục tiêu
Chờ nhiều channel cùng lúc (select), và điều khiển timeout/hủy bằng context.

## Lý thuyết

### select — chờ nhiều channel
`select` cho phép chờ trên nhiều channel, cái nào sẵn sàng trước thì xử lý:
```go
select {
case v := <-ch1:
	fmt.Println("từ ch1:", v)
case v := <-ch2:
	fmt.Println("từ ch2:", v)
}
```
Nếu nhiều case cùng sẵn sàng, Go chọn ngẫu nhiên một.

### Timeout với select + time.After
```go
select {
case res := <-resultCh:
	fmt.Println("kết quả:", res)
case <-time.After(2 * time.Second):
	fmt.Println("hết giờ!")        // sau 2s không có kết quả
}
```
`time.After` trả về một channel "kêu" sau khoảng thời gian — pattern timeout kinh điển.

### default — non-blocking
```go
select {
case v := <-ch:
	fmt.Println(v)
default:
	fmt.Println("không có gì, làm việc khác")  // không block
}
```

### context — điều khiển timeout & hủy chuẩn mực
`context` là cách chuẩn để truyền tín hiệu "dừng lại" xuyên qua các tầng/goroutine. Bạn sẽ thấy `ctx context.Context` là tham số đầu tiên của rất nhiều hàm.

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()                   // luôn gọi cancel để giải phóng

select {
case res := <-work(ctx):
	fmt.Println(res)
case <-ctx.Done():               // ctx hết hạn hoặc bị hủy
	fmt.Println("hủy:", ctx.Err())  // context.DeadlineExceeded
}
```
- `context.Background()` : context gốc.
- `WithTimeout` : tự hủy sau thời gian cho trước.
- `WithCancel` : hủy thủ công bằng cách gọi `cancel()`.
- `ctx.Done()` : channel đóng khi bị hủy/hết hạn.
- `ctx.Err()` : lý do (`DeadlineExceeded` hoặc `Canceled`).

### context trong web request
Gin tự gắn context vào request. Mọi lời gọi DB/HTTP nên nhận context để khi client ngắt kết nối, mọi việc dưới được hủy theo:
```go
func (h *Handler) Get(c *gin.Context) {
	ctx := c.Request.Context()
	sub, err := h.svc.Get(ctx, id)   // truyền ctx xuống
}
```

### context truyền giá trị (request-scoped)
```go
ctx := context.WithValue(parent, "request_id", rid)
rid := ctx.Value("request_id")
```
Dùng cho dữ liệu gắn với 1 request (request ID, user ID). Đừng lạm dụng cho mọi thứ — chỉ cho dữ liệu xuyên suốt request.

## Bài tập
1. Viết 2 goroutine gửi vào 2 channel ở 2 thời điểm khác nhau; dùng `select` nhận cái nào tới trước.
2. Viết hàm `slowWork(ch chan string)` sleep 3s rồi gửi kết quả. Dùng `select` + `time.After(2s)` để timeout — xác nhận in "hết giờ".
3. Làm lại bài 2 nhưng dùng `context.WithTimeout(2s)` thay cho `time.After`. So sánh 2 cách.
4. Viết `select` có `default` để kiểm tra channel mà không block.
5. Tạo `context.WithCancel`, chạy một goroutine lặp vô hạn nhưng dừng khi `<-ctx.Done()`. Gọi `cancel()` sau 1s và xác nhận goroutine dừng.

## Checklist
- [ ] Dùng `select` chờ nhiều channel
- [ ] Pattern timeout bằng `time.After` và bằng `context.WithTimeout`
- [ ] Hiểu `ctx.Done()` và `ctx.Err()`
- [ ] Biết truyền `context` xuống các tầng
- [ ] `WithCancel` để hủy goroutine chủ động
