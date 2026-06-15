# Bài 19 — Capstone: Submission Service hoàn chỉnh

## Mục tiêu
Ráp tất cả kiến thức thành một Submission Service chạy thật, end-to-end.

## Yêu cầu hệ thống

Xây Submission Service làm được:
1. `POST /api/v1/submissions` — nhận bài nộp (language, code), validate, lưu Postgres với status `PENDING`, publish job vào RabbitMQ `judge.jobs`, trả 201 với submission.
2. `GET /api/v1/submissions/:id` — trả submission (kèm status hiện tại).
3. `GET /api/v1/submissions` — liệt kê (lọc theo `?status=`).
4. Một consumer nghe `judge.results`, cập nhật status submission trong DB.
5. Có middleware logging + recovery + request ID + auth.
6. Graceful shutdown cho cả HTTP server lẫn consumer.

Kèm một **Judger Worker giả** (mock) để test end-to-end: consume `judge.jobs`, sleep ngẫu nhiên, random verdict, publish vào `judge.results`. (Judger thật là phần OS interaction ở tài liệu riêng.)

## Cấu trúc thư mục đề xuất
```
submission-service/
├── cmd/
│   ├── server/main.go         # HTTP server + result consumer
│   └── mockjudger/main.go     # judger giả để test
├── internal/
│   ├── model/submission.go
│   ├── repository/            # interface + PostgresRepo
│   ├── service/               # SubmissionService
│   ├── handler/               # SubmissionHandler + RegisterRoutes
│   ├── queue/                 # JobPublisher, ResultConsumer
│   └── middleware/            # logging, recovery, auth, requestID
├── config.yaml
├── docker-compose.yml         # Postgres + RabbitMQ
├── go.mod
└── Dockerfile
```

## Các bước thực hiện

### Bước 1 — Hạ tầng
Viết `docker-compose.yml` chạy Postgres + RabbitMQ. Tạo bảng `submissions`. (Bài 12, 17)

### Bước 2 — Tầng dữ liệu
`model.Submission`, interface `SubmissionRepository`, `PostgresRepo` với `Create`, `GetByID`, `List`, `UpdateStatus`. (Bài 11, 12)

### Bước 3 — Queue
Interface `JobPublisher` + `RabbitPublisher`. Struct `JudgeJob`, `JudgeResult`. (Bài 18)

### Bước 4 — Tầng nghiệp vụ
`SubmissionService` nhận repo + publisher. `Create` lưu DB → publish job. `Get`, `List`. (Bài 11, 18)

### Bước 5 — Tầng HTTP
`SubmissionHandler` với các route, validation bằng tag. (Bài 7, 8, 9, 11)

### Bước 6 — Middleware
Logging, recovery, request ID, auth đơn giản. Gắn vào router. (Bài 10)

### Bước 7 — Result consumer
Goroutine consume `judge.results`, parse, gọi `repo.UpdateStatus`. (Bài 18)

### Bước 8 — Mock judger
`cmd/mockjudger/main.go`: consumer `judge.jobs` + worker pool, sleep random, random verdict trong {AC, WA, TLE}, publish `judge.results`. (Bài 15, 18)

### Bước 9 — Lắp ráp + graceful shutdown
`cmd/server/main.go`: kết nối DB + MQ, dựng repo → publisher → service → handler, chạy HTTP server + result consumer trong goroutine, bắt SIGTERM tắt sạch. (Bài 16)

## Kịch bản test end-to-end

1. `docker compose up` chạy Postgres + RabbitMQ.
2. Chạy `cmd/server` và `cmd/mockjudger` (2 terminal).
3. Nộp bài:
```bash
curl -X POST localhost:8080/api/v1/submissions \
  -H "Authorization: secret123" \
  -d '{"language":"cpp","code":"int main(){return 0;}"}'
# -> 201, status "PENDING"
```
4. Ngay lập tức `GET /api/v1/submissions/:id` → status vẫn `PENDING`.
5. Đợi mock judger xử lý (vài trăm ms) → `GET` lại → status đã đổi thành AC/WA/TLE.
6. Quan sát log: request ID xuyên suốt, thời gian xử lý, message qua 2 queue.
7. Ctrl+C server giữa lúc có request → xác nhận tắt sạch.

## Tiêu chí hoàn thành (Definition of Done)
- [ ] Nộp bài trả về ngay với PENDING (không chờ chấm)
- [ ] Job xuất hiện trong `judge.jobs`, mock judger xử lý
- [ ] Kết quả về `judge.results`, status trong DB cập nhật
- [ ] `GET` phản ánh đúng trạng thái mới
- [ ] Validation chặn input sai (language lạ, code rỗng)
- [ ] Auth chặn request thiếu token (401)
- [ ] Log có request ID, đo thời gian
- [ ] Graceful shutdown hoạt động (không cắt ngang request)
- [ ] Job không mất khi mock judger restart giữa chừng (nhờ ack thủ công)

## Mở rộng (nếu còn sức)
- Viết unit test cho service dùng `FakeRepo` + `FakePublisher` (mock 2 dependency).
- Thêm endpoint `GET /healthz` cho healthcheck.
- Dockerize bằng multi-stage build (`CGO_ENABLED=0`), đo size image.
- Thay mock judger bằng judger thật (phần OS interaction — tài liệu riêng).
- Thêm gRPC cho giao tiếp nội bộ giữa các service.

Hoàn thành capstone này nghĩa là bạn đã có một backend service production-shape thật sự: phân tầng sạch, validate, auth, log, hàng đợi bất đồng bộ, xử lý song song, và tắt an toàn — đúng nền tảng để xây tiếp toàn bộ DARE-ka.
