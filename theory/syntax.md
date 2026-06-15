# Go Syntax & Library Reference 

Tài liệu tra cứu nhanh: cú pháp Go cốt lõi + mọi thư viện dùng trong Submission Service (API) và Judger Worker (OS). Mỗi mục có snippet áp dụng được ngay và ghi chú dùng vào đâu trong dự án.

## Mục lục

- Phần A — Cú pháp Go cốt lõi
- Phần B — Standard library
- Phần C — Thư viện cho Submission Service (API)
- Phần D — Thư viện cho Judger Worker (OS)
- Phần E — Bảng tra nhanh "cần X thì dùng gì"

---

# Phần A — Cú pháp Go cốt lõi

## A1. Module & package

```go
// go.mod — khai báo module
module github.com/dare-ka/submission-service
go 1.22
require (
    github.com/gin-gonic/gin v1.10.0
)
```

```bash
go mod init github.com/dare-ka/submission-service
go mod tidy        # dọn & tải dependency
go run ./cmd/server
go build -o app ./cmd/server
```

Mỗi file bắt đầu bằng `package`. `main` là package đặc biệt (chương trình chạy được, có `func main()`). Tên viết hoa = export ra ngoài package; viết thường = private.

```go
package judge

func Judge(s Submission) Result { ... } // export (J hoa)
func classify(...) Verdict { ... }       // private (c thường)
```

## A2. Biến, hằng, kiểu

```go
var x int = 5
y := 10                 // suy kiểu, chỉ dùng trong hàm
const MaxWorkers = 8
var (
    host string = "localhost"
    port int    = 8080
)

// kiểu cơ bản: bool, string, int/int8..int64, uint.., float32/64, byte (=uint8), rune (=int32)
// zero value: 0, "", false, nil — biến không khởi tạo tự nhận zero value
```

## A3. Struct, method, embedding

```go
type Submission struct {
    ID       string
    Language string
    Code     string
}

// method với receiver value (bản sao)
func (s Submission) IsEmpty() bool { return s.Code == "" }

// method với receiver pointer (sửa được bản gốc)
func (s *Submission) SetStatus(st string) { s.Status = st }

// embedding — "kế thừa" bằng cách nhúng struct
type AuditedSubmission struct {
    Submission        // nhúng -> truy cập trực tiếp s.ID
    CreatedAt time.Time
}
```

Quy tắc: dùng pointer receiver `*T` khi cần sửa state hoặc struct lớn (tránh copy); value receiver `T` khi nhỏ và bất biến. Trong 1 type nên thống nhất một kiểu receiver.

## A4. Interface (implicit)

Không cần `implements` — type nào có đủ method là tự động thỏa interface.

```go
type Runner interface {
    Run(ctx context.Context, job Job) (Result, error)
}

type DockerRunner struct{}
func (d DockerRunner) Run(ctx context.Context, job Job) (Result, error) { ... }
// DockerRunner tự động là Runner, không cần khai báo gì thêm

// interface rỗng / any — chứa bất kỳ kiểu nào
var v any = 42

// type assertion & type switch
if ws, ok := ps.Sys().(syscall.WaitStatus); ok { ... }
switch x := v.(type) {
case int:    ...
case string: ...
}
```

## A5. Error handling

Error là **giá trị**, không phải exception.

```go
func parse(b []byte) (*Submission, error) {
    var s Submission
    if err := json.Unmarshal(b, &s); err != nil {
        return nil, fmt.Errorf("parse submission: %w", err) // %w để wrap
    }
    return &s, nil
}

// caller
s, err := parse(data)
if err != nil {
    return err // xử lý tường minh
}

// custom error
type ValidationError struct{ Field, Msg string }
func (e *ValidationError) Error() string { return e.Field + ": " + e.Msg }

// kiểm tra loại lỗi
errors.Is(err, sql.ErrNoRows)          // so với sentinel error
var ve *ValidationError
errors.As(err, &ve)                    // ép về kiểu cụ thể
```

## A6. Pointer

```go
x := 10
p := &x        // p trỏ tới x
*p = 20        // sửa qua pointer -> x = 20
var s *Submission           // nil pointer
s = &Submission{ID: "1"}    // cấp phát
```

Go có garbage collector — không cần free thủ công. Không có con trỏ số học (pointer arithmetic) như C.

## A7. Slice & map

```go
// slice — mảng động
nums := []int{1, 2, 3}
nums = append(nums, 4)
sub := nums[1:3]            // [2,3], chia sẻ backing array
make([]int, 0, 100)        // len 0, cap 100 (cấp trước để tránh realloc)

// map
m := map[string]int{"a": 1}
m["b"] = 2
v, ok := m["a"]            // ok=false nếu không có key
delete(m, "a")
for k, v := range m { ... } // thứ tự ngẫu nhiên

// lang config map (dùng thật trong judge)
var langs = map[string]LangConfig{
    "cpp": {SourceFile: "main.cpp", RunCmd: []string{"./main"}},
}
```

## A8. Control flow

```go
if v, err := f(); err != nil { ... }   // khai báo + điều kiện

for i := 0; i < n; i++ { ... }          // for cổ điển
for i < n { ... }                       // for kiểu while
for { ... }                             // vòng lặp vô hạn
for i, x := range slice { ... }         // range

switch x {
case 1, 2: ...                          // không tự fallthrough
default: ...
}
```

## A9. defer / panic / recover

```go
func process() (err error) {
    f, _ := os.Open("x")
    defer f.Close()                  // chạy khi hàm return (LIFO)

    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("panic: %v", r) // bắt panic -> error
        }
    }()
    // ...
}
```

`defer` lý tưởng cho cleanup (đóng file, xóa sandbox, unlock mutex). `panic`/`recover` chỉ dùng cho lỗi thật sự bất thường, không thay cho error thường.

## A10. Goroutine, channel, select

```go
go worker()                            // chạy concurrent

ch := make(chan int)                   // unbuffered (đồng bộ)
ch := make(chan int, 10)               // buffered (cap 10)
ch <- 5                                // gửi
v := <-ch                              // nhận
v, ok := <-ch                          // ok=false nếu đã close & cạn
close(ch)                              // chỉ bên gửi close
for v := range ch { ... }              // lặp tới khi close

select {
case v := <-ch1:        ...
case ch2 <- x:          ...
case <-ctx.Done():      return
case <-time.After(2*time.Second): ...  // timeout
default:                ...            // non-blocking
}
```

## A11. Generics (Go 1.18+)

```go
func Map[T, U any](s []T, f func(T) U) []U {
    out := make([]U, len(s))
    for i, v := range s { out[i] = f(v) }
    return out
}

type Number interface { ~int | ~int64 | ~float64 }
func Sum[T Number](s []T) T { ... }
```

---

# Phần B — Standard library

## B1. `context` — timeout, cancel, truyền giá trị

Trung tâm của mọi thứ I/O và process trong dự án.

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()

ctx, cancel := context.WithCancel(parent)
ctx := context.WithValue(parent, key, val)  // truyền request-scoped data

<-ctx.Done()              // channel đóng khi hết hạn/bị hủy
ctx.Err()                 // context.DeadlineExceeded hoặc context.Canceled
```

## B2. `os` / `io` / `bufio`

```go
os.MkdirTemp("", "judge-*")            // thư mục tạm
os.WriteFile(path, data, 0o600)
os.ReadFile(path)
os.RemoveAll(dir)                      // xóa cây thư mục
os.Chmod(path, 0o755)
os.Getenv("PORT")
os.Exit(1)

io.Copy(dst, src)
io.LimitReader(r, 64*1024)             // giới hạn byte đọc (chống output bomb)
io.Discard                             // /dev/null
```

## B3. `os/signal` — graceful shutdown

```go
ctx, stop := signal.NotifyContext(context.Background(),
    syscall.SIGINT, syscall.SIGTERM)
defer stop()
<-ctx.Done()                           // chờ tín hiệu tắt
// -> dừng nhận job mới, chờ job đang chạy, đóng connection
```

## B4. `sync` — Mutex, WaitGroup, Once

```go
var mu sync.Mutex
mu.Lock(); count++; mu.Unlock()

var rw sync.RWMutex
rw.RLock()  // nhiều reader
rw.Lock()   // 1 writer

var wg sync.WaitGroup
wg.Add(1)
go func() { defer wg.Done(); ... }()
wg.Wait()                              // chờ tất cả xong

var once sync.Once
once.Do(func() { ... })                // chạy đúng 1 lần
```

Khi nào dùng channel vs mutex: channel để **chuyển dữ liệu/quyền sở hữu** giữa goroutine (pipeline job→result); mutex để **bảo vệ state chung tồn tại lâu** (cache, counter).

## B5. `encoding/json`

```go
type Sub struct {
    ID   string `json:"id"`
    Lang string `json:"language"`
    X    int    `json:"x,omitempty"`   // bỏ qua nếu zero
    secret string `json:"-"`           // không serialize
}
b, err := json.Marshal(sub)
err = json.Unmarshal(b, &sub)
```

## B6. `time`, `strings`, `bytes`, `strconv`, `fmt`

```go
time.Now(); time.Since(start); time.Sleep(d); 2 * time.Second
start := time.Now(); elapsed := time.Since(start).Milliseconds()

strings.Fields(s)        // tách theo whitespace (so output token-based)
strings.TrimSpace(s); strings.Split(s, ","); strings.NewReader(s)

var buf bytes.Buffer; buf.WriteString("x"); buf.String()

strconv.Atoi("42"); strconv.Itoa(42); strconv.ParseFloat(s, 64)

fmt.Sprintf("%s-%d", id, i); fmt.Errorf("...: %w", err)
```

## B7. `testing` + `httptest`

```go
func TestWorse(t *testing.T) {
    tests := []struct {           // table-driven
        a, b, want Verdict
    }{
        {AC, WA, WA},
        {TLE, RE, RE},
    }
    for _, tc := range tests {
        if got := worse(tc.a, tc.b); got != tc.want {
            t.Errorf("worse(%v,%v)=%v want %v", tc.a, tc.b, got, tc.want)
        }
    }
}

// test HTTP handler không cần chạy server thật
req := httptest.NewRequest("POST", "/submissions", body)
w := httptest.NewRecorder()
router.ServeHTTP(w, req)
// w.Code, w.Body.String()
```

```bash
go test ./...           # chạy tất cả test
go test -race ./...     # phát hiện race condition
go test -cover ./...    # độ phủ
```

---

# Phần C — Thư viện cho Submission Service (API)

## C1. Gin — HTTP framework

`github.com/gin-gonic/gin`

```go
r := gin.Default()                     // có sẵn logger + recovery

r.POST("/submissions", createHandler)
r.GET("/submissions/:id", getHandler)
r.Use(authMiddleware())                // middleware toàn cục

func createHandler(c *gin.Context) {
    var req CreateReq
    if err := c.ShouldBindJSON(&req); err != nil { // parse + validate
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    id := c.Param("id")                 // path param
    q := c.Query("status")              // query param
    c.JSON(201, gin.H{"id": "..."})
}

// middleware
func authMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if !valid(token) {
            c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
            return
        }
        c.Set("userID", uid)            // truyền xuống handler
        c.Next()
    }
}

r.Run(":8080")
```

## C2. validator — kiểm tra dữ liệu

`github.com/go-playground/validator/v10` (Gin tích hợp sẵn qua tag `binding`)

```go
type CreateReq struct {
    Language string `json:"language" binding:"required,oneof=cpp java python"`
    Code     string `json:"code"     binding:"required,max=65536"`
    Email    string `json:"email"    binding:"omitempty,email"`
}
// ShouldBindJSON tự validate theo tag. Tag hay dùng:
// required, omitempty, min, max, len, oneof, email, url, gte, lte, uuid
```

## C3. gRPC + protobuf

`google.golang.org/grpc`, `google.golang.org/protobuf` — dùng `buf` để generate.

```protobuf
// submission.proto
syntax = "proto3";
service SubmissionService {
  rpc CreateSubmission(CreateRequest) returns (CreateResponse);
  rpc GetSubmission(GetRequest) returns (Submission);
}
```

```bash
buf generate         # sinh code Go từ proto (theo buf.gen.yaml)
```

```go
// server
type server struct { pb.UnimplementedSubmissionServiceServer }
func (s *server) GetSubmission(ctx context.Context, r *pb.GetRequest) (*pb.Submission, error) {
    sub, ok := store[r.Id]
    if !ok {
        return nil, status.Error(codes.NotFound, "submission not found")
    }
    return sub, nil
}

lis, _ := net.Listen("tcp", ":50051")
gs := grpc.NewServer(grpc.UnaryInterceptor(logInterceptor))
pb.RegisterSubmissionServiceServer(gs, &server{})
gs.Serve(lis)

// client
conn, _ := grpc.NewClient("localhost:50051",
    grpc.WithTransportCredentials(insecure.NewCredentials()))
defer conn.Close()
client := pb.NewSubmissionServiceClient(conn)
resp, err := client.GetSubmission(ctx, &pb.GetRequest{Id: "1"})
if status.Code(err) == codes.NotFound { ... }

// interceptor (middleware của gRPC)
func logInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo,
    handler grpc.UnaryHandler) (any, error) {
    start := time.Now()
    resp, err := handler(ctx, req)
    log.Printf("%s took %v", info.FullMethod, time.Since(start))
    return resp, err
}
```

## C4. RabbitMQ

`github.com/rabbitmq/amqp091-go`

```go
conn, _ := amqp.Dial("amqp://guest:guest@localhost:5672/")
defer conn.Close()
ch, _ := conn.Channel()
defer ch.Close()

q, _ := ch.QueueDeclare("judge.jobs", true, false, false, false, nil) // durable

// publish
ch.PublishWithContext(ctx, "", q.Name, false, false, amqp.Publishing{
    ContentType:  "application/json",
    Body:         body,
    DeliveryMode: amqp.Persistent,
})

// consume
ch.Qos(8, 0, false)              // prefetch = 8 (giới hạn job đang xử lý)
msgs, _ := ch.Consume(q.Name, "", false, false, false, false, nil) // autoAck=false
for d := range msgs {
    if err := handle(d.Body); err != nil {
        d.Nack(false, true)      // lỗi -> trả lại queue
    } else {
        d.Ack(false)             // xong -> xác nhận
    }
}
```

## C5. Database — sqlx (nhẹ) hoặc GORM (ORM)

`github.com/jmoiron/sqlx` + driver (`github.com/lib/pq` hoặc `pgx`)

```go
db, _ := sqlx.Connect("postgres", dsn)
db.SetMaxOpenConns(10)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(time.Hour)

type Sub struct {
    ID     string `db:"id"`
    Status string `db:"status"`
}
var s Sub
db.Get(&s, "SELECT * FROM submissions WHERE id=$1", id)   // 1 dòng
var list []Sub
db.Select(&list, "SELECT * FROM submissions")             // nhiều dòng
db.NamedExec(`INSERT INTO submissions (id,status) VALUES (:id,:status)`, s)
```

`gorm.io/gorm` (nếu thích ORM kiểu JPA):

```go
db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})
db.AutoMigrate(&Sub{})
db.Create(&sub)
db.First(&sub, "id = ?", id)
db.Model(&sub).Update("status", "DONE")
db.Where("status = ?", "PENDING").Find(&list)
```

## C6. ScyllaDB / Cassandra

`github.com/gocql/gocql`

```go
cluster := gocql.NewCluster("127.0.0.1")
cluster.Keyspace = "dareka"
cluster.Consistency = gocql.Quorum
session, _ := cluster.CreateSession()
defer session.Close()

session.Query(`INSERT INTO submission_log (id, ts) VALUES (?, ?)`,
    id, time.Now()).Exec()
var ts time.Time
session.Query(`SELECT ts FROM submission_log WHERE id=?`, id).Scan(&ts)
```

## C7. JWT

`github.com/golang-jwt/jwt/v5`

```go
// tạo token
claims := jwt.MapClaims{"sub": userID, "role": "user",
    "exp": time.Now().Add(time.Hour).Unix()}
token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
signed, _ := token.SignedString([]byte(secret))

// verify
parsed, err := jwt.Parse(signed, func(t *jwt.Token) (any, error) {
    return []byte(secret), nil
})
if claims, ok := parsed.Claims.(jwt.MapClaims); ok && parsed.Valid {
    uid := claims["sub"]
}
```

## C8. Config — Viper

`github.com/spf13/viper`

```go
viper.SetConfigName("config")
viper.SetConfigType("yaml")
viper.AddConfigPath(".")
viper.AutomaticEnv()                    // env override file
viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // database.host -> DATABASE_HOST
viper.ReadInConfig()

port := viper.GetString("server.port")
viper.Unmarshal(&cfg)                   // map cả vào struct
```

## C9. Logging — zap (hoặc zerolog)

`go.uber.org/zap`

```go
logger, _ := zap.NewProduction()        // log JSON
defer logger.Sync()
logger.Info("submission created",
    zap.String("request_id", rid),
    zap.String("user_id", uid),
    zap.Int("code_len", n))
logger.Error("db failed", zap.Error(err))
```

`github.com/rs/zerolog`:

```go
log.Info().Str("request_id", rid).Int("len", n).Msg("created")
log.Error().Err(err).Msg("db failed")
```

## C10. Testing nâng cao — testify

`github.com/stretchr/testify`

```go
assert.Equal(t, want, got)
require.NoError(t, err)                  // require dừng test ngay nếu fail
assert.Len(t, list, 3)

// mock
type MockRepo struct{ mock.Mock }
func (m *MockRepo) GetByID(id string) (*Sub, error) {
    args := m.Called(id)
    return args.Get(0).(*Sub), args.Error(1)
}
repo.On("GetByID", "1").Return(&Sub{}, nil)
```

---

# Phần D — Thư viện cho Judger Worker (OS)

## D1. `os/exec` — spawn process

```go
cmd := exec.CommandContext(ctx, "./main")  // KHÔNG qua shell
cmd.Dir = sandboxDir
cmd.Stdin = strings.NewReader(input)
cmd.Stdout = &cappedBuf
cmd.Stderr = io.Discard
err := cmd.Run()                            // = Start() + Wait()

cmd.ProcessState.ExitCode()                 // -1 nếu bị signal
cmd.ProcessState.Sys().(syscall.WaitStatus) // chi tiết signal/exit
cmd.ProcessState.SysUsage().(*syscall.Rusage)
cmd.Process.Pid

cmd.Cancel = func() error {                 // Go 1.20+: tùy biến hành vi hủy
    return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
cmd.WaitDelay = 200 * time.Millisecond
```

## D2. `syscall` — SysProcAttr (cách ly & giới hạn)

```go
cmd.SysProcAttr = &syscall.SysProcAttr{
    Setpgid:    true,                       // tạo process group -> kill cả nhóm
    Credential: &syscall.Credential{        // chạy non-root
        Uid: nobodyUid, Gid: nobodyGid,
    },
    Chroot:      sandboxRoot,                // nhốt filesystem
    Cloneflags:  syscall.CLONE_NEWPID |      // namespace cách ly
                 syscall.CLONE_NEWNS  |
                 syscall.CLONE_NEWNET,       // tắt network
    UseCgroupFD: true,                       // Go 1.22+: đặt vào cgroup lúc tạo
    CgroupFD:    int(cgDir.Fd()),
}

// WaitStatus
ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
ws.Signaled(); ws.Signal()                  // SIGKILL, SIGSEGV, SIGXCPU...
ws.Exited(); ws.ExitStatus()

// kill process group
syscall.Kill(-pid, syscall.SIGKILL)
```

## D3. `golang.org/x/sys/unix` — rlimit

```go
import "golang.org/x/sys/unix"

var rl unix.Rlimit
rl = unix.Rlimit{Cur: 2, Max: 2}            // RLIMIT_CPU = 2 giây
unix.Setrlimit(unix.RLIMIT_CPU, &rl)
// các limit: RLIMIT_CPU, RLIMIT_AS, RLIMIT_NPROC,
//            RLIMIT_NOFILE, RLIMIT_FSIZE, RLIMIT_STACK
// (nhớ: KHÔNG dùng RLIMIT_AS cho Java — dùng cgroup)
```

Vì rlimit kế thừa và khó set riêng cho process con trong Go, thực tế hay bọc qua tiện ích `prlimit` hoặc dùng cgroup (xem tài liệu Judge OS deep-dive).

## D4. `syscall.Rusage` — đo tài nguyên

```go
ru := cmd.ProcessState.SysUsage().(*syscall.Rusage)
cpu := time.Duration(ru.Utime.Nano() + ru.Stime.Nano())
maxRssKb := ru.Maxrss                       // LINUX: KB (macOS: byte!)
```

## D5. cgroups v2 (qua `os` thường)

cgroup thao tác bằng đọc/ghi file, không cần lib riêng:

```go
cg := "/sys/fs/cgroup/judge-" + id
os.Mkdir(cg, 0o755)
os.WriteFile(cg+"/memory.max", []byte("268435456"), 0o644) // 256MB
os.WriteFile(cg+"/cpu.max",    []byte("100000 100000"), 0o644)
os.WriteFile(cg+"/pids.max",   []byte("64"), 0o644)
peak, _ := os.ReadFile(cg+"/memory.peak")
events, _ := os.ReadFile(cg+"/memory.events") // tìm oom_kill
```

## D6. `go.uber.org/automaxprocs` — GOMAXPROCS đúng trong container

```go
import _ "go.uber.org/automaxprocs"          // chỉ cần import, tự chạy
// đặt GOMAXPROCS theo CPU limit của cgroup thay vì CPU của host
```

## D7. Worker pool (chỉ dùng stdlib)

```go
jobCh := make(chan Job)
resCh := make(chan Result)
var wg sync.WaitGroup
for i := 0; i < runtime.NumCPU(); i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        for j := range jobCh { resCh <- Judge(j) }
    }()
}
go func() { for _, j := range jobs { jobCh <- j }; close(jobCh) }()
go func() { wg.Wait(); close(resCh) }()
for r := range resCh { ... }
```

---

# Phần E — Bảng tra nhanh "cần X thì dùng gì"

| Cần làm | Dùng |
|---|---|
| HTTP API | `gin` |
| Validate request | `validator` (tag `binding`) |
| Giao tiếp giữa service | `grpc` + `protobuf` + `buf` |
| Hàng đợi job | `amqp091-go` (RabbitMQ) |
| DB quan hệ (nhẹ) | `sqlx` + `pgx`/`lib/pq` |
| DB quan hệ (ORM) | `gorm` |
| ScyllaDB | `gocql` |
| JWT | `golang-jwt/jwt/v5` |
| Config (file + env) | `viper` |
| Log JSON có cấu trúc | `zap` hoặc `zerolog` |
| Test assertion + mock | `testify` |
| Spawn process con | `os/exec` |
| Timeout / hủy | `context` |
| Cách ly & quyền process | `syscall.SysProcAttr` |
| rlimit | `golang.org/x/sys/unix` |
| Giới hạn mem/cpu chuẩn | cgroups v2 (qua `os.WriteFile`) |
| Đo CPU/memory | `syscall.Rusage` + cgroup `memory.peak` |
| GOMAXPROCS trong container | `automaxprocs` |
| Graceful shutdown | `os/signal` + `signal.NotifyContext` |
| Đồng bộ goroutine | `sync` (Mutex/WaitGroup) + channel |
| Giới hạn output | `io.LimitReader` / custom writer |
| So output token-based | `strings.Fields` |

## Lệnh hay dùng

```bash
go mod init <module>      # khởi tạo module
go mod tidy               # dọn dependency
go get <pkg>@<version>    # thêm dependency
go run ./cmd/server       # chạy
go build -o app ./cmd/... # build
go test ./...             # test
go test -race ./...       # phát hiện race
go vet ./...              # bắt lỗi tĩnh
gofmt -w .                # format
CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o app  # build static, nhỏ gọn
```

## Cấu trúc project gợi ý

```
submission-service/
├── cmd/server/main.go          # entrypoint
├── internal/
│   ├── handler/                # HTTP/gRPC handler
│   ├── service/                # business logic
│   ├── repository/             # truy cập DB
│   └── middleware/             # auth, logging
├── pkg/                        # code tái dùng được
├── proto/                      # file .proto
├── config.yaml
├── go.mod
└── Dockerfile
```
