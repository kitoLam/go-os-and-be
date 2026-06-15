# Bài 11 — Kiến trúc API: handler / service / repository

## Mục tiêu
Tổ chức code thành 3 tầng rõ ràng, dễ test và mở rộng — cách đạt sự ngăn nắp của NestJS theo lối Go.

## Lý thuyết

### Ba tầng và trách nhiệm
- **Handler** (tầng HTTP): nhận request, validate dạng, gọi service, trả response. KHÔNG chứa business logic.
- **Service** (tầng nghiệp vụ): logic chính (luật, điều phối), gọi repository. KHÔNG biết gì về HTTP.
- **Repository** (tầng dữ liệu): đọc/ghi DB. KHÔNG biết gì về business logic.

Chiều phụ thuộc một chiều: `handler → service → repository`. Tách bạch giúp: đổi DB không ảnh hưởng handler, test service không cần dựng HTTP, mock repository để test.

### Repository (định nghĩa qua interface)
```go
// internal/repository/repository.go
package repository

type SubmissionRepository interface {
	Create(s model.Submission) error
	GetByID(id string) (model.Submission, error)
}

// implementation in-memory (đổi sang Postgres ở bài 12)
type MemoryRepo struct {
	data map[string]model.Submission
}
func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{data: map[string]model.Submission{}}
}
func (r *MemoryRepo) Create(s model.Submission) error {
	r.data[s.ID] = s
	return nil
}
func (r *MemoryRepo) GetByID(id string) (model.Submission, error) {
	s, ok := r.data[id]
	if !ok {
		return model.Submission{}, ErrNotFound
	}
	return s, nil
}
```
Service phụ thuộc **interface** `SubmissionRepository`, không phải `MemoryRepo` cụ thể.

### Service
```go
// internal/service/submission.go
package service

type SubmissionService struct {
	repo repository.SubmissionRepository   // interface
}
func NewSubmissionService(repo repository.SubmissionRepository) *SubmissionService {
	return &SubmissionService{repo: repo}
}

func (s *SubmissionService) Create(req CreateInput) (model.Submission, error) {
	if strings.TrimSpace(req.Code) == "" {       // validate nghiệp vụ
		return model.Submission{}, errors.New("code rỗng")
	}
	sub := model.Submission{
		ID:        uuid.New().String(),
		Language:  req.Language,
		Code:      req.Code,
		Status:    "PENDING",
		CreatedAt: time.Now(),
	}
	if err := s.repo.Create(sub); err != nil {
		return model.Submission{}, fmt.Errorf("create: %w", err)
	}
	return sub, nil
}
```

### Handler
```go
// internal/handler/submission.go
package handler

type SubmissionHandler struct {
	svc *service.SubmissionService
}
func NewSubmissionHandler(svc *service.SubmissionService) *SubmissionHandler {
	return &SubmissionHandler{svc: svc}
}

func (h *SubmissionHandler) Create(c *gin.Context) {
	var req CreateSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	sub, err := h.svc.Create(service.CreateInput{
		Language: req.Language, Code: req.Code,
	})
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, sub)
}

func (h *SubmissionHandler) RegisterRoutes(r *gin.Engine) {
	r.POST("/submissions", h.Create)
	r.GET("/submissions/:id", h.Get)
}
```

### Lắp ráp ở main
```go
// cmd/server/main.go
func main() {
	repo := repository.NewMemoryRepo()
	svc := service.NewSubmissionService(repo)
	h := handler.NewSubmissionHandler(svc)

	r := gin.Default()
	h.RegisterRoutes(r)
	r.Run(":8080")
}
```
Đây là **dependency injection thủ công**: tạo từ tầng dưới lên, tiêm vào tầng trên qua constructor. Muốn đổi sang Postgres, chỉ sửa 1 dòng `repo := ...` ở main.

## Bài tập
1. Dựng đủ 3 tầng cho `Submission` như trên (dùng MemoryRepo).
2. Hoàn thiện `Get` ở cả service và handler; khi không tìm thấy, service trả `ErrNotFound`, handler map sang HTTP 404 (dùng `errors.Is`).
3. Thêm method `List() []model.Submission` xuyên suốt 3 tầng.
4. Viết validation nghiệp vụ trong service: từ chối nếu cùng user nộp 2 lần trong 1 giây (lưu thời điểm nộp cuối trong repo) — chứng minh logic này thuộc service, không thuộc handler.
5. Chứng minh tính tách tầng: viết một hàm gọi thẳng `svc.Create(...)` mà KHÔNG qua HTTP, chạy trong `main`. Service phải chạy độc lập được.

## Checklist
- [ ] Hiểu trách nhiệm từng tầng và chiều phụ thuộc
- [ ] Repository định nghĩa qua interface
- [ ] Service nhận repo qua constructor (DI thủ công)
- [ ] Handler chỉ lo HTTP, service lo nghiệp vụ
- [ ] Lắp ráp ở main, đổi implementation chỉ sửa 1 chỗ
