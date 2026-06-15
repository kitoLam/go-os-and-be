# Bài 00 — Khởi động: cài đặt và chương trình đầu tien

## Mục tiêu
Cài Go, hiểu `go.mod`, chạy được chương trình đầu tiên, biết các lệnh `go` cơ bản.

## Lý thuyết

### Go là gì và vì sao dùng cho backend
Go là ngôn ngữ biên dịch (compiled), tạo ra một file binary chạy được, không cần runtime như JVM (Java) hay Node. Nó nổi tiếng vì: concurrency rẻ (goroutine), biên dịch nhanh, binary nhẹ dễ deploy, standard library mạnh về network. Đây là combo lý tưởng cho API và microservice.

### Cài đặt
Tải từ trang chủ go.dev/dl, cài theo OS. Kiểm tra:
```bash
go version
# go version go1.22.x ...
```

### Module — go.mod
Mọi project Go là một **module**. Khởi tạo:
```bash
mkdir submission-service && cd submission-service
go mod init github.com/ban/submission-service
```
Lệnh này tạo file `go.mod`:
```
module github.com/ban/submission-service
go 1.22
```
`module` là "tên" project, dùng làm gốc cho mọi import. `go.mod` cũng liệt kê dependency (thư viện ngoài) — `go` tự quản lý.

### Chương trình đầu tiên
Tạo file `main.go`:
```go
package main

import "fmt"

func main() {
	fmt.Println("Submission Service khởi động!")
}
```
- `package main` + `func main()` = điểm khởi chạy của chương trình.
- `import "fmt"` = dùng package in ấn của standard library.

Chạy:
```bash
go run .          # biên dịch + chạy ngay (không tạo file)
go build .        # tạo file binary tên submission-service
./submission-service
```

### Các lệnh phải nhớ
```bash
go run .          # chạy nhanh khi dev
go build .        # build ra binary
go mod tidy       # tải/dọn dependency theo import trong code
go get <pkg>      # thêm 1 thư viện
go test ./...     # chạy test toàn project
gofmt -w .        # tự format code (Go có style chuẩn duy nhất)
go vet ./...      # bắt lỗi tĩnh thường gặp
```

### Một điều khác Node/Python
Go **bắt buộc** format chuẩn và không cho phép import thừa hay biến khai báo mà không dùng — sẽ lỗi biên dịch. Điều này lạ lúc đầu nhưng giúp codebase team luôn sạch và đồng nhất.

## Bài tập
1. Cài Go, chạy `go version` thành công.
2. Tạo module `submission-service`, viết `main.go` in ra tên bạn + "DARE-ka".
3. Chạy bằng cả `go run .` và `go build .` + chạy binary. Quan sát khác biệt (build tạo ra file).
4. Cố tình thêm `import "strings"` mà không dùng → chạy thử → đọc thông báo lỗi `imported and not used`. Đây là lỗi bạn sẽ gặp rất nhiều lúc đầu.
5. Chạy `gofmt -w main.go`, xem nó tự sắp lại code.

## Checklist
- [ ] `go version` chạy được
- [ ] Hiểu `go.mod` là gì, `module` dùng làm gì
- [ ] Phân biệt `go run` và `go build`
- [ ] Biết vì sao import thừa gây lỗi
