# Khóa học Go Backend — Từ 0 tới Submission Service

Khóa học dành cho người mới với Go (nhưng đã biết lập trình). Mục tiêu cuối: tự build được **Submission Service** cho DARE-ka — một API nhận bài nộp, lưu DB, đẩy job sang hàng đợi RabbitMQ để judger chấm.

Mỗi bài là 1 file. Học tuần tự. Mỗi bài có: **Mục tiêu → Lý thuyết (kèm code) → Bài tập → Checklist**. Code trong bài luôn hướng tới hệ thống thật, không phải ví dụ trừu tượng.

## Lộ trình

### Phần 0 — Khởi động
- `00-setup.md` — Cài Go, công cụ, chạy chương trình đầu tiên, hiểu `go.mod`

### Phần 1 — Nền tảng ngôn ngữ (vừa đủ cho backend)
- `01-cu-phap-co-ban.md` — Biến, kiểu, hàm, vòng lặp, điều kiện
- `02-struct-method-interface.md` — Mô hình hóa dữ liệu, interface
- `03-error-handling.md` — Error là giá trị, wrap/unwrap
- `04-slice-map-pointer.md` — Cấu trúc dữ liệu nền tảng + pointer
- `05-package-project.md` — Chia package, tổ chức thư mục

### Phần 2 — HTTP & API
- `06-http-net-http.md` — Server HTTP bằng thư viện chuẩn (hiểu gốc rễ)
- `07-gin-routing.md` — Gin: routing, handler, path/query param
- `08-json-binding.md` — Nhận/trả JSON, bind request
- `09-validation.md` — Kiểm tra dữ liệu đầu vào
- `10-middleware.md` — Logging, recovery, auth
- `11-kien-truc-api.md` — Tách handler / service / repository
- `12-database.md` — Kết nối Postgres, repository pattern, connection pool

### Phần 3 — Concurrency
- `13-goroutine-channel.md` — Goroutine và channel
- `14-select-context.md` — select, timeout, hủy bằng context
- `15-sync-worker-pool.md` — sync, WaitGroup, worker pool
- `16-graceful-shutdown.md` — Tắt server/worker an toàn

### Phần 4 — RabbitMQ & Messaging
- `17-rabbitmq-co-ban.md` — Message queue là gì, publish/consume
- `18-messaging-he-thong.md` — Producer + Consumer trong hệ thống thật

### Phần 5 — Capstone
- `19-capstone-submission-service.md` — Ráp Submission Service hoàn chỉnh

## Cách học hiệu quả

Đừng chỉ đọc — gõ lại mọi đoạn code và chạy. Làm hết bài tập trước khi sang bài sau. Mỗi phần kết thúc bạn nên có code chạy được, không phải chỉ kiến thức trên giấy.

Gợi ý nhịp độ: Phần 0+1 trong 2-3 ngày, Phần 2 trong 3-4 ngày, Phần 3 trong 2-3 ngày, Phần 4 trong 2 ngày, Capstone 1-2 ngày. Tổng ~2 tuần nếu học chậm rãi, hoặc nén lại 1 tuần nếu đã quen lập trình.

## Chuẩn bị môi trường

- Go 1.22+ (xem bài 00)
- Một editor: VS Code (+ extension Go) hoặc GoLand
- Docker (để chạy Postgres + RabbitMQ local từ Phần 2 trở đi)
- Terminal cơ bản
