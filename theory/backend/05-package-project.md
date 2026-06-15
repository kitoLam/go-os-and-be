# Bài 05 — Package & tổ chức project

## Mục tiêu
Hiểu package, import, export, và cách tổ chức thư mục cho một service backend.

## Lý thuyết

### Package
Mỗi thư mục là một package. File đầu tiên khai báo tên package:
```go
package repository
```
Quy ước: tên package = tên thư mục, viết thường, ngắn gọn (`repository`, `handler`, `service`).

### Export — viết hoa hay thường
Đây là cơ chế "public/private" của Go, dựa vào **chữ cái đầu**:
```go
func CreateSubmission() { }   // viết HOA -> export, package khác dùng được
func validateCode() { }       // viết thường -> private, chỉ trong package này

type Submission struct {       // export
	ID     string              // field viết hoa -> export
	secret string              // field viết thường -> private
}
```

### Import
Dùng đường dẫn đầy đủ tính từ module path (trong `go.mod`):
```go
import (
	"fmt"                                              // standard library
	"github.com/ban/submission-service/internal/repository"  // package nội bộ
	"github.com/gin-gonic/gin"                         // thư viện ngoài
)

// dùng
repository.CreateSubmission()
```
Lỗi hay gặp của người mới: import `"repository"` (tên cụt) thay vì đường dẫn đầy đủ → `cannot find package`. Luôn dùng module-path + đường-dẫn-thư-mục.

### Cấu trúc thư mục chuẩn cho service
```
submission-service/
├── cmd/
│   └── server/
│       └── main.go          # điểm khởi chạy, lắp ráp mọi thứ
├── internal/                # code riêng tư của service này
│   ├── handler/             # nhận HTTP request, gọi service
│   ├── service/             # business logic
│   ├── repository/          # truy cập DB
│   └── model/               # struct dữ liệu (Submission, User...)
├── config.yaml
├── go.mod
└── Dockerfile
```
Quy tắc phụ thuộc (rất quan trọng): `handler → service → repository`. Tầng trên gọi tầng dưới, không ngược lại. `model` được dùng chung bởi mọi tầng.

### `internal/` có gì đặc biệt
Thư mục tên `internal` được Go bảo vệ: chỉ code trong cùng module mới import được. Đặt code nghiệp vụ vào đây để không bị project khác lỡ import nhầm.

### `cmd/` — nơi lắp ráp
`main.go` là nơi "ghép" mọi mảnh: tạo repository → tiêm vào service → tiêm vào handler → khởi động server. Đây gọi là **dependency injection thủ công** (truyền qua constructor), cách Go-idiomatic thay cho DI tự động của NestJS.

```go
// cmd/server/main.go
func main() {
	repo := repository.NewPostgresRepo(db)
	svc := service.NewSubmissionService(repo)
	h := handler.NewHandler(svc)
	h.RegisterRoutes(router)
	router.Run(":8080")
}
```

## Bài tập
1. Tạo cấu trúc thư mục như trên cho `submission-service`.
2. Trong `internal/model`, tạo file `submission.go` định nghĩa struct `Submission`.
3. Trong `internal/repository`, viết hàm export `NewMemoryRepo()` trả về một repo lưu in-memory (dùng map). Thêm method `Create` và `GetByID`.
4. Trong `internal/service`, viết struct `SubmissionService` nhận repo qua constructor `NewSubmissionService(repo ...)`.
5. Trong `cmd/server/main.go`, lắp ráp repo → service và gọi thử `Create` + `GetByID`, in kết quả. Chứng minh chuỗi phụ thuộc hoạt động.

## Checklist
- [ ] Hiểu mỗi thư mục = 1 package
- [ ] Hiểu export bằng chữ hoa/thường
- [ ] Import bằng đường dẫn đầy đủ từ module path
- [ ] Nắm cấu trúc cmd / internal / handler / service / repository / model
- [ ] Hiểu chiều phụ thuộc handler → service → repository
