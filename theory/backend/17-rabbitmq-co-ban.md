# Bài 17 — RabbitMQ cơ bản

## Mục tiêu
Hiểu message queue dùng để làm gì, và publish/consume message bằng Go.

## Lý thuyết

### Message queue giải quyết vấn đề gì
Trong DARE-ka: khi user nộp bài, Submission Service KHÔNG nên tự chấm (chậm, nặng CPU) rồi mới trả response — user sẽ chờ lâu. Thay vào đó:
1. Submission Service nhận bài, lưu DB, **đẩy một "job" vào hàng đợi**, trả response ngay ("đã nhận, đang chấm").
2. Judger Worker **lấy job từ hàng đợi** khi rảnh, chấm, trả kết quả.

Hàng đợi (message queue) là "băng chuyền" trung gian giúp 2 service:
- **Tách rời** (decouple): Submission không cần biết Judger ở đâu, sống hay chết.
- **Chịu tải** (buffer): lúc cao điểm 5000 bài nộp cùng lúc, queue giữ lại, worker xử lý dần — không sập.
- **Co giãn**: thêm nhiều worker để chấm nhanh hơn mà không đổi Submission Service.

RabbitMQ là một message broker phổ biến làm việc này.

### Khái niệm cốt lõi
- **Producer**: bên gửi message (Submission Service).
- **Queue**: hàng đợi chứa message.
- **Consumer**: bên nhận message (Judger Worker).
- **Exchange**: bộ định tuyến message tới queue (dùng `""` — default exchange — cho trường hợp đơn giản nhất).
- **Ack**: consumer báo "đã xử lý xong" để queue xóa message; nếu consumer chết trước khi ack, message được giao lại cho worker khác → không mất job.

### Chạy RabbitMQ local
```bash
docker run -d --name dareka-mq \
  -p 5672:5672 -p 15672:15672 \
  rabbitmq:3-management
# 5672 = cổng app, 15672 = giao diện web quản lý (guest/guest)
```

### Cài thư viện
```bash
go get github.com/rabbitmq/amqp091-go
```

### Kết nối + khai báo queue
```go
import amqp "github.com/rabbitmq/amqp091-go"

conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
// xử lý err...
defer conn.Close()

ch, err := conn.Channel()         // channel của RabbitMQ (khác Go channel!)
defer ch.Close()

q, err := ch.QueueDeclare(
	"judge.jobs", // tên queue
	true,         // durable: queue tồn tại lại sau khi RabbitMQ restart
	false,        // autoDelete
	false,        // exclusive
	false,        // noWait
	nil,          // args
)
```

### Publish (producer)
```go
body := []byte(`{"submission_id":"1","language":"cpp"}`)
err = ch.PublishWithContext(ctx,
	"",          // exchange ("" = default)
	q.Name,      // routing key = tên queue
	false, false,
	amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,  // message lưu xuống đĩa, không mất khi restart
	},
)
```

### Consume (consumer)
```go
msgs, err := ch.Consume(
	q.Name, "",
	false,       // autoAck = false -> tự ack thủ công (an toàn hơn)
	false, false, false, nil,
)
for d := range msgs {
	fmt.Println("nhận:", string(d.Body))
	// ... xử lý ...
	d.Ack(false)              // báo đã xử lý xong
	// nếu lỗi: d.Nack(false, true) -> trả lại queue để thử lại
}
```

### durable + persistent + manual ack = không mất job
Ba thứ này phối hợp đảm bảo job không biến mất kể cả khi RabbitMQ restart hay worker chết giữa chừng. Quan trọng cho hệ thống chấm bài — mất job nghĩa là bài của user không được chấm.

## Bài tập
1. Chạy RabbitMQ bằng Docker, mở giao diện `localhost:15672` (guest/guest).
2. Viết chương trình **producer**: kết nối, khai báo queue `judge.jobs` (durable), publish 5 message JSON. Vào giao diện web xem 5 message trong queue.
3. Viết chương trình **consumer** riêng: consume queue đó, in từng message, `Ack`. Quan sát message biến mất khỏi queue sau khi ack.
4. Test "không mất job": chạy consumer với `autoAck=false`, nhận message nhưng **không** ack (comment dòng Ack), tắt consumer giữa chừng — vào web xem message vẫn còn (unacked → trả lại).
5. Chạy 2 consumer cùng lúc trên cùng queue, publish 10 message — quan sát chúng chia nhau xử lý (mỗi message chỉ tới 1 consumer).

## Checklist
- [ ] Hiểu vì sao cần message queue (tách rời, chịu tải, co giãn)
- [ ] Nắm producer / queue / consumer / ack
- [ ] Publish và consume bằng amqp091-go
- [ ] Hiểu durable + persistent + manual ack chống mất job
