# Bài 15 — sync & worker pool

## Mục tiêu
Đồng bộ goroutine bằng `sync`, và xây worker pool — pattern trung tâm để xử lý job song song có kiểm soát.

## Lý thuyết

### sync.WaitGroup — đợi nhiều goroutine xong
```go
var wg sync.WaitGroup
for i := 0; i < 5; i++ {
	wg.Add(1)                    // tăng đếm trước khi chạy
	go func(n int) {
		defer wg.Done()          // báo xong khi hàm kết thúc
		fmt.Println("worker", n)
	}(i)
}
wg.Wait()                        // block tới khi tất cả Done
fmt.Println("tất cả xong")
```
Đây là cách sạch nhất để "đợi N goroutine hoàn thành" — thay cho việc đếm bằng channel thủ công.

### sync.Mutex — bảo vệ dữ liệu chung
Khi nhiều goroutine cùng ghi vào một biến → race condition. Mutex (khóa) đảm bảo mỗi lúc chỉ một goroutine vào "vùng nguy hiểm":
```go
var (
	mu    sync.Mutex
	count int
)
func incr() {
	mu.Lock()
	count++              // chỉ 1 goroutine làm việc này tại 1 thời điểm
	mu.Unlock()
}
```
Phát hiện race bằng `go run -race .` — luôn chạy thử với cờ này.

### Channel vs Mutex — chọn cái nào
- **Channel**: khi cần **chuyển dữ liệu/quyền sở hữu** giữa goroutine (job → result). Triết lý Go: "đừng chia sẻ bộ nhớ rồi khóa, hãy truyền dữ liệu qua channel".
- **Mutex**: khi cần **bảo vệ state chung tồn tại lâu** (cache, counter) mà truyền qua channel sẽ rườm rà.

### Worker pool — pattern quan trọng nhất
Vấn đề: có N job, muốn xử lý song song nhưng **giới hạn** số goroutine (không nổ 1 triệu goroutine). Giải pháp: cố định số "worker", chúng cùng lấy job từ một channel.

```go
type Job struct{ ID string }
type Result struct{ ID, Status string }

func runPool(jobs []Job, numWorkers int) []Result {
	jobCh := make(chan Job)
	resCh := make(chan Result)

	// 1. khởi động N worker
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobCh {          // lấy job tới khi jobCh đóng
				resCh <- process(job)
			}
		}()
	}

	// 2. bơm job rồi đóng channel
	go func() {
		for _, j := range jobs { jobCh <- j }
		close(jobCh)                          // báo hết job
	}()

	// 3. đóng resCh khi mọi worker xong
	go func() {
		wg.Wait()
		close(resCh)
	}()

	// 4. gom kết quả
	var out []Result
	for r := range resCh {
		out = append(out, r)
	}
	return out
}
```
Để ý cách `close` lan truyền tín hiệu kết thúc: đóng `jobCh` → worker thoát → `wg.Wait()` trả về → đóng `resCh` → vòng gom thoát. Không có mutex, chỉ truyền dữ liệu — đúng tinh thần Go.

`numWorkers` thường = `runtime.NumCPU()` cho việc CPU-bound, hoặc lớn hơn cho việc I/O-bound.

### Semaphore bằng buffered channel
Cách gọn để giới hạn số việc đồng thời mà không cần pool đầy đủ:
```go
sem := make(chan struct{}, 4)    // tối đa 4 cùng lúc
for _, job := range jobs {
	sem <- struct{}{}            // xin "vé" (block nếu đủ 4)
	go func(j Job) {
		defer func() { <-sem }() // trả vé
		process(j)
	}(job)
}
```

## Bài tập
1. Dùng `WaitGroup` chạy 5 goroutine, đợi tất cả xong rồi in "done".
2. Tạo race condition: 100 goroutine cùng `count++` không khóa, chạy `go run -race .` để thấy cảnh báo. Rồi sửa bằng `sync.Mutex`.
3. Viết `runPool` xử lý 20 job với `numWorkers = 4`. `process` chỉ sleep 100ms rồi trả Result.
4. Đo thời gian `runPool` với numWorkers = 1 vs 4 vs 8, lập bảng so sánh.
5. Viết phiên bản dùng semaphore (buffered channel) giới hạn 4 việc đồng thời, so với worker pool.

## Checklist
- [ ] `WaitGroup`: Add / Done / Wait
- [ ] `Mutex` bảo vệ state chung; phát hiện race bằng `-race`
- [ ] Biết khi nào channel, khi nào mutex
- [ ] Xây được worker pool hoàn chỉnh (4 bước, close lan truyền)
- [ ] Hiểu semaphore bằng buffered channel
