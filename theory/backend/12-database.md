# Bài 12 — Database: Postgres, repository thật, connection pool

## Mục tiêu
Thay MemoryRepo bằng repository thật nói chuyện với Postgres.

## Lý thuyết

### Chạy Postgres local bằng Docker
```bash
docker run -d --name dareka-pg \
  -e POSTGRES_PASSWORD=secret -e POSTGRES_DB=dareka \
  -p 5432:5432 postgres:16
```

### Chọn thư viện
- `database/sql` (chuẩn) + driver — nền tảng, hơi thô.
- `sqlx` — mở rộng nhẹ của `database/sql`, tự map struct ↔ row. **Khuyến nghị cho người mới** vì gần SQL gốc nhưng đỡ lặp.
- `gorm` — ORM đầy đủ (giống JPA), tiện nhưng "magic" nhiều.

Bài này dùng `sqlx`.
```bash
go get github.com/jmoiron/sqlx
go get github.com/lib/pq         # driver Postgres
```

### Kết nối + connection pool
```go
import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"          // import _ để đăng ký driver
)

func NewDB() (*sqlx.DB, error) {
	dsn := "host=localhost port=5432 user=postgres password=secret dbname=dareka sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)         // tối đa 10 kết nối đồng thời
	db.SetMaxIdleConns(5)          // giữ sẵn 5 kết nối nhàn rỗi
	db.SetConnMaxLifetime(time.Hour)
	return db, nil
}
```
**Connection pool** là gì: mở connection tới DB rất tốn kém, nên Go giữ sẵn một "hồ" connection tái dùng. `SetMaxOpenConns` giới hạn để không làm sập DB khi traffic cao. Đây là chỗ ảnh hưởng lớn tới hiệu năng.

### Tạo bảng
```sql
CREATE TABLE submissions (
	id         TEXT PRIMARY KEY,
	user_id    TEXT NOT NULL,
	language   TEXT NOT NULL,
	code       TEXT NOT NULL,
	status     TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL
);
```

### Repository với sqlx
```go
type PostgresRepo struct {
	db *sqlx.DB
}
func NewPostgresRepo(db *sqlx.DB) *PostgresRepo {
	return &PostgresRepo{db: db}
}

// struct với tag db:
type submissionRow struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	Language  string    `db:"language"`
	Code      string    `db:"code"`
	Status    string    `db:"status"`
	CreatedAt time.Time `db:"created_at"`
}

func (r *PostgresRepo) Create(s model.Submission) error {
	_, err := r.db.Exec(
		`INSERT INTO submissions (id,user_id,language,code,status,created_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		s.ID, s.UserID, s.Language, s.Code, s.Status, s.CreatedAt)
	return err
}

func (r *PostgresRepo) GetByID(id string) (model.Submission, error) {
	var row submissionRow
	err := r.db.Get(&row, `SELECT * FROM submissions WHERE id=$1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Submission{}, ErrNotFound
	}
	if err != nil {
		return model.Submission{}, err
	}
	return model.Submission(row), nil   // chuyển row -> model
}
```
- `$1, $2...` là placeholder của Postgres (chống SQL injection — KHÔNG bao giờ nối chuỗi SQL).
- `db.Get` lấy 1 dòng, `db.Select` lấy nhiều dòng.
- `sql.ErrNoRows` = không có dòng nào → map sang `ErrNotFound`.

### Lắp vào main (chỉ đổi 1 dòng so với bài 11)
```go
db, _ := repository.NewDB()
repo := repository.NewPostgresRepo(db)     // thay NewMemoryRepo
svc := service.NewSubmissionService(repo)  // service KHÔNG đổi gì
```
Vì service phụ thuộc interface, đổi từ Memory sang Postgres không động tới service/handler. Đây là phần thưởng của kiến trúc bài 11.

## Bài tập
1. Chạy Postgres bằng Docker, tạo bảng `submissions`.
2. Viết `NewDB` với connection pool, kết nối thành công (in log "DB connected").
3. Viết `PostgresRepo` với `Create`, `GetByID`, `List` dùng sqlx.
4. Đổi `main` dùng `PostgresRepo`. Chạy lại API bài 11 — phải hoạt động y hệt, dữ liệu giờ lưu trong Postgres (kiểm tra bằng `psql` hoặc tool xem DB).
5. Test `GetByID` với id không tồn tại → service trả `ErrNotFound` → handler trả 404. Chứng minh `sql.ErrNoRows` được map đúng.

## Checklist
- [ ] Kết nối Postgres bằng sqlx + cấu hình connection pool
- [ ] Dùng placeholder `$1` (không nối chuỗi SQL)
- [ ] `db.Get` / `db.Select` map row vào struct (tag `db:`)
- [ ] Map `sql.ErrNoRows` sang lỗi nghiệp vụ
- [ ] Đổi Memory → Postgres mà không sửa service/handler
