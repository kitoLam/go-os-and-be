# Bài 18 — Messaging trong hệ thống thật

## Mục tiêu
Tích hợp RabbitMQ vào kiến trúc tầng: producer trong Submission Service, consumer kết hợp worker pool trong Judger Worker.

## Lý thuyết

### Bọc RabbitMQ sau một interface
Đừng để code RabbitMQ rải khắp service. Định nghĩa một interface "publisher" để service không phụ thuộc trực tiếp vào RabbitMQ (dễ test, dễ đổi broker):
```go
// internal/queue/publisher.go
package queue

type JobPublisher interface {
	PublishJudgeJob(ctx context.Context, job JudgeJob) error
}

type JudgeJob struct {
	SubmissionID string `json:"submission_id"`
	Language     string `json:"language"`
	Code         string `json:"code"`
}
```

Implementation RabbitMQ:
```go
type RabbitPublisher struct {
	ch    *amqp.Channel
	queue string
}
func NewRabbitPublisher(ch *amqp.Channel, queue string) *RabbitPublisher {
	return &RabbitPublisher{ch: ch, queue: queue}
}
func (p *RabbitPublisher) PublishJudgeJob(ctx context.Context, job JudgeJob) error {
	body, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return p.ch.PublishWithContext(ctx, "", p.queue, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		})
}
```

### Service dùng publisher
Service nhận `JobPublisher` qua constructor (giống nhận repository):
```go
type SubmissionService struct {
	repo      repository.SubmissionRepository
	publisher queue.JobPublisher
}
func NewSubmissionService(repo ..., pub queue.JobPublisher) *SubmissionService {
	return &SubmissionService{repo: repo, publisher: pub}
}

func (s *SubmissionService) Create(ctx context.Context, in CreateInput) (model.Submission, error) {
	sub := model.Submission{ /* ... */ Status: "PENDING" }
	if err := s.repo.Create(sub); err != nil {     // 1. lưu DB
		return model.Submission{}, err
	}
	err := s.publisher.PublishJudgeJob(ctx, queue.JudgeJob{   // 2. đẩy job
		SubmissionID: sub.ID, Language: sub.Language, Code: sub.Code,
	})
	if err != nil {
		// tùy chính sách: log lỗi, hoặc đánh dấu cần retry
		return model.Submission{}, fmt.Errorf("publish job: %w", err)
	}
	return sub, nil          // 3. trả ngay, không chờ chấm xong
}
```
Luồng: lưu DB → publish job → trả response ngay. User không phải chờ chấm.

### Consumer + worker pool (Judger Worker)
Bên judger, consume queue và đẩy vào worker pool (kết hợp bài 15 + 17):
```go
func RunConsumer(ctx context.Context, ch *amqp.Channel, numWorkers int) error {
	ch.Qos(numWorkers, 0, false)        // prefetch = số worker (không ôm quá nhiều job)
	msgs, err := ch.Consume("judge.jobs", "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case d, ok := <-msgs:
					if !ok { return }
					var job queue.JudgeJob
					if err := json.Unmarshal(d.Body, &job); err != nil {
						d.Nack(false, false)   // message hỏng -> bỏ (không requeue)
						continue
					}
					result := judge(job)       // chấm bài (phần OS interaction)
					publishResult(ch, result)   // đẩy kết quả về queue judge.results
					d.Ack(false)                // báo xong job này
				case <-ctx.Done():
					return                      // graceful shutdown
				}
			}
		}()
	}
	<-ctx.Done()
	wg.Wait()
	return nil
}
```
- `Qos(prefetch)`: giới hạn số message mỗi worker "ôm" sẵn → cân bằng tải giữa nhiều worker.
- `Ack` sau khi chấm xong: nếu worker chết giữa chừng (chưa ack), job tự được giao lại worker khác.
- `Nack(requeue=false)` cho message hỏng (JSON sai) để khỏi lặp vô hạn.

### Vòng kết quả (đóng vòng tròn)
Judger chấm xong publish kết quả vào queue `judge.results`. Submission Service chạy thêm một consumer nghe queue này để cập nhật `status` trong DB từ `PENDING` → `AC`/`WA`/... Client gọi `GET /submissions/:id` thấy kết quả mới.

Đây chính là luồng end-to-end hoàn chỉnh của hệ thống chấm bài.

## Bài tập
1. Định nghĩa interface `JobPublisher` + `JudgeJob`, viết `RabbitPublisher` implement nó.
2. Sửa `SubmissionService.Create` (bài 11/12) để: lưu DB → publish job → trả response. Tiêm publisher qua constructor.
3. Viết một `FakedPublisher` (chỉ in ra console) thỏa `JobPublisher` — dùng để chạy service mà không cần RabbitMQ thật. Chứng minh lợi ích của interface.
4. Viết consumer + worker pool như trên (phần `judge` tạm sleep 200ms rồi trả "AC"). Chạy: nộp bài qua API → thấy job xuất hiện ở consumer → consumer xử lý.
5. (Nâng cao) Thêm queue `judge.results`: worker publish kết quả, Submission Service consume và update status trong DB. Hoàn thành vòng tròn, kiểm tra `GET /submissions/:id` thấy status đổi từ PENDING.

## Checklist
- [ ] Bọc RabbitMQ sau interface `JobPublisher`
- [ ] Service publish job sau khi lưu DB, trả response ngay
- [ ] Consumer kết hợp worker pool + prefetch (Qos)
- [ ] Ack/Nack đúng để không mất job và không lặp message hỏng
- [ ] Hiểu vòng kết quả khép kín (judge.results → update DB)
