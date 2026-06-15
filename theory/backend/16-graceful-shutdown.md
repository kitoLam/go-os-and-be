# Bài 16 — Graceful shutdown

## Mục tiêu
Tắt server/worker an toàn: hoàn thành việc đang dở, đóng kết nối sạch, không mất dữ liệu.

## Lý thuyết

### Vì sao cần
Khi deploy phiên bản mới, Docker/Coolify gửi tín hiệu `SIGTERM` để dừng container. Nếu bạn để chương trình chết ngay, request đang xử lý dở bị cắt, message RabbitMQ đang publish có thể mất, connection DB không đóng sạch. Graceful shutdown = "nghe tín hiệu dừng → ngừng nhận việc mới → chờ việc đang làm xong → đóng tài nguyên → mới thoát".

### Bắt tín hiệu hệ điều hành
```go
import (
	"os/signal"
	"syscall"
)

ctx, stop := signal.NotifyContext(context.Background(),
	syscall.SIGINT, syscall.SIGTERM)
defer stop()

<-ctx.Done()    // block tới khi nhận Ctrl+C (SIGINT) hoặc SIGTERM
```
`signal.NotifyContext` tạo một context tự hủy khi nhận tín hiệu — kết hợp đẹp với mọi thứ đã học về context.

### Graceful shutdown cho HTTP server
```go
func main() {
	r := gin.Default()
	// ... đăng ký routes ...

	srv := &http.Server{Addr: ":8080", Handler: r}

	// chạy server trong goroutine để không block
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// chờ tín hiệu dừng
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	log.Println("đang tắt server...")

	// cho tối đa 10s để hoàn thành request đang dở
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Println("tắt cưỡng bức:", err)
	}
	log.Println("server đã tắt sạch")
}
```
`srv.Shutdown(ctx)` ngừng nhận request mới và chờ request đang xử lý xong (tới khi hết `shutdownCtx`).

### Graceful shutdown cho worker (judger)
Worker đang chấm bài giữa chừng thì nhận SIGTERM — phải để nó chấm xong job hiện tại:
```go
func main() {
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	jobCh := make(chan Job)
	var wg sync.WaitGroup

	// worker
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case job := <-jobCh:
					process(job)            // làm xong job hiện tại
				case <-ctx.Done():
					return                  // nhận tín hiệu -> thoát vòng
				}
			}
		}()
	}

	<-ctx.Done()
	log.Println("ngừng nhận job mới, chờ job đang chạy...")
	wg.Wait()                                // chờ tất cả worker xong
	// đóng RabbitMQ connection, DB pool ở đây
	log.Println("worker đã tắt sạch")
}
```

### Thứ tự đóng tài nguyên
Đóng theo chiều ngược với chiều mở, đại loại: ngừng nhận việc mới → chờ việc đang làm → đóng consumer queue → đóng connection MQ → đóng DB pool → thoát. Mục tiêu: không bỏ rơi việc đang xử lý và không để lại kết nối "treo".

## Bài tập
1. Thêm graceful shutdown cho HTTP server bài 12: chạy server trong goroutine, bắt SIGTERM, `srv.Shutdown` với timeout 10s.
2. Test: tạo một route sleep 5s, gọi nó rồi ngay lập tức Ctrl+C — quan sát server chờ request đó xong mới tắt (không cắt ngang).
3. Viết worker đơn giản dùng `select { case job: ...; case <-ctx.Done(): return }`. Gửi vài job, rồi Ctrl+C giữa chừng — xác nhận job đang chạy hoàn thành trước khi thoát.
4. Thêm log ở các mốc: "nhận tín hiệu", "chờ job", "đã tắt sạch" để thấy rõ trình tự.
5. Thử đặt timeout shutdown rất ngắn (1s) với request sleep 5s — quan sát "tắt cưỡng bức" xảy ra.

## Checklist
- [ ] Bắt SIGINT/SIGTERM bằng `signal.NotifyContext`
- [ ] `srv.Shutdown(ctx)` cho HTTP server với timeout
- [ ] Worker thoát qua `<-ctx.Done()` + `wg.Wait()`
- [ ] Hiểu thứ tự đóng tài nguyên để không mất việc/dữ liệu
