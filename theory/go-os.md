# Go OS Interaction — Xây dựng Judge Worker hoàn chỉnh

Tài liệu này dạy đủ kiến thức OS-level trong Go để xây một judge: nhận 1 file code của thí sinh → compile → chạy với từng test input → đo thời gian/bộ nhớ → so output với expected → trả về verdict (AC / WA / TLE / MLE / RE / CE).

Mỗi chương = 1 chủ đề nhỏ, gồm **Lý thuyết** (kèm code áp dụng được ngay) và **Bài tập**. Làm tuần tự, mỗi bài tập là 1 mảnh ghép, đến chương 12 sẽ ráp thành judge chạy thật.

> Môi trường: Linux (cgroup v2, kernel 5.19+ là lý tưởng), Go 1.22+ (cần cho `UseCgroupFD`). Nếu dev trên Windows/macOS, hãy chạy trong 1 VM/container Linux vì phần lớn syscall ở đây là Linux-only.

---

## Chương 0 — Bức tranh tổng thể

### Lý thuyết

Một judge worker, ở mức cốt lõi, là một chương trình **điều phối process con không tin cậy** một cách an toàn và đo lường được. Go không tự dịch/chạy C++ hay Java — nó gọi `g++`, `javac`, `java`, `python3`... (đã cài sẵn trong môi trường) như những process con, rồi kiểm soát chặt: giới hạn thời gian, giới hạn bộ nhớ, chặn quyền nguy hiểm, bắt output.

Pipeline đầy đủ cho 1 submission:

```
nhận job  ->  tạo sandbox dir  ->  ghi source file  ->  compile (nếu cần)
   |                                                          |
   |                                              CE nếu compile lỗi
   v
for mỗi test case:
    chạy binary với stdin = input, có time limit + memory limit + giới hạn output
    -> đọc exit status + signal + tài nguyên đã dùng
    -> phân loại: TLE? MLE? RE? hay chạy xong bình thường?
    -> nếu chạy xong: so output với expected -> AC hoặc WA
verdict cuối = tổng hợp verdict các test (thường lấy cái "tệ" nhất)
   |
   v
dọn sandbox  ->  publish kết quả
```

### Các verdict cần phân biệt

| Verdict | Tên | Khi nào |
|---|---|---|
| `AC` | Accepted | Tất cả test pass |
| `WA` | Wrong Answer | Chạy xong nhưng output sai |
| `TLE` | Time Limit Exceeded | Chạy quá thời gian cho phép |
| `MLE` | Memory Limit Exceeded | Dùng quá bộ nhớ cho phép |
| `RE` | Runtime Error | Crash, exit code khác 0, segfault... |
| `CE` | Compile Error | Compile thất bại |
| `OLE` | Output Limit Exceeded | In ra quá nhiều (chống output bomb) |

Độ ưu tiên khi tổng hợp (gặp cái nào "nặng" hơn thì verdict cuối là cái đó): `CE > RE > MLE > TLE > OLE > WA > AC`. (Thứ tự này tùy hệ thống, nhưng CE luôn cao nhất vì không chạy được test nào.)

### Bốn nhóm API Go sẽ dùng xuyên suốt

1. `os/exec` — spawn process con, gắn stdin/stdout, lấy exit code.
2. `context` — đặt timeout, hủy process khi quá giờ.
3. `syscall` / `golang.org/x/sys/unix` — `SysProcAttr` (process group, cgroup, drop privilege), đọc `WaitStatus` và `Rusage`.
4. `os` / `io` — tạo/xóa thư mục tạm, ghi file, giới hạn lượng byte đọc/ghi.

### Bài tập 0

Viết file `judge/verdict.go` định nghĩa:
- `type Verdict string` với các hằng `AC`, `WA`, `TLE`, `MLE`, `RE`, `CE`, `OLE`.
- Hàm `func worse(a, b Verdict) Verdict` trả về verdict "tệ hơn" theo bảng ưu tiên trên.
- `type TestResult struct { Verdict Verdict; TimeMs int64; MemoryKb int64; Message string }`.
- `type JudgeResult struct { Verdict Verdict; Tests []TestResult }`.

Viết unit test cho `worse`: `worse(AC, WA) == WA`, `worse(TLE, RE) == RE`, `worse(CE, AC) == CE`.

---

## Chương 1 — Spawn process con với `os/exec`

### Lý thuyết

`exec.Command(name, args...)` tạo một `*exec.Cmd` mô tả lệnh sẽ chạy. **Quan trọng**: các argument được truyền dưới dạng mảng riêng biệt, KHÔNG đi qua shell. Đây là điểm bảo mật then chốt — nếu bạn ghép chuỗi rồi `exec.Command("sh", "-c", userInput)` thì dính command injection ngay.

```go
// ĐÚNG - mỗi tham số tách riêng, không qua shell
cmd := exec.Command("g++", "main.cpp", "-O2", "-o", "main")

// SAI - đi qua shell, nguy hiểm với input không tin cậy
cmd := exec.Command("sh", "-c", "g++ "+filename+" -o main")
```

Hai cách chạy:
- `cmd.Run()` = `Start()` + `Wait()`, block tới khi xong. Dùng khi không cần can thiệp giữa chừng.
- `cmd.Start()` rồi `cmd.Wait()` riêng — dùng khi cần lấy PID ngay sau Start (để bỏ vào cgroup, để kill...).

Capture output bằng cách gán `io.Writer` vào `cmd.Stdout` / `cmd.Stderr`:

```go
var stdout, stderr bytes.Buffer
cmd.Stdout = &stdout
cmd.Stderr = &stderr
err := cmd.Run()
```

Lấy exit code:

```go
err := cmd.Run()
if err != nil {
    if exitErr, ok := err.(*exec.ExitError); ok {
        // chương trình chạy nhưng exit khác 0
        code := exitErr.ExitCode() // -1 nếu bị kill bằng signal
    } else {
        // không chạy được (không tìm thấy file, không có quyền exec...)
    }
}
```

Sau khi `Wait()` xong, `cmd.ProcessState` chứa mọi thông tin: `.ExitCode()`, `.Success()`, `.Sys()` (ép kiểu `syscall.WaitStatus`), `.SysUsage()` (ép kiểu `*syscall.Rusage`). Đây là nguồn dữ liệu chính để phân loại verdict ở chương 7.

### Bài tập 1

Viết hàm:
```go
func runOnce(name string, args []string, stdinData string) (stdout, stderr string, exitCode int, err error)
```
- Gắn `cmd.Stdin = strings.NewReader(stdinData)`.
- Capture stdout/stderr vào `bytes.Buffer`.
- Trả về exit code (dùng `ProcessState.ExitCode()`).

Test với:
1. Một script in `stdin * 2`: viết file `double.py` (`print(int(input())*2)`), gọi `runOnce("python3", []string{"double.py"}, "21")`, kỳ vọng stdout = `"42\n"`, exitCode = 0.
2. Một script `exit(3)`: kỳ vọng exitCode = 3.
3. Lệnh không tồn tại (`"khong_co_lenh"`): kỳ vọng `err != nil` và KHÔNG phải `*exec.ExitError`.

---

## Chương 2 — Stdin/Stdout/Stderr & chống output bomb

### Lý thuyết

**Feed input**: 3 cách gắn stdin:
- `cmd.Stdin = strings.NewReader(s)` — input nằm sẵn trong memory.
- `cmd.Stdin = file` — đọc thẳng từ file input của test case (tiết kiệm RAM khi input lớn).
- `cmd.StdinPipe()` — ghi dần (ít dùng cho judge).

**Chống output bomb**: thí sinh có thể in vô hạn (`while True: print(1)`). Nếu gán thẳng `cmd.Stdout = &bytes.Buffer{}`, buffer phình tới khi worker hết RAM. Phải giới hạn. Một `io.Writer` tự cắt:

```go
type cappedWriter struct {
    buf   bytes.Buffer
    limit int
    over  bool
}

func (w *cappedWriter) Write(p []byte) (int, error) {
    if w.buf.Len() >= w.limit {
        w.over = true
        return len(p), nil // "nuốt" phần thừa, không lưu, nhưng không báo lỗi để process không nhận EPIPE bất ngờ
    }
    remain := w.limit - w.buf.Len()
    if len(p) > remain {
        w.buf.Write(p[:remain])
        w.over = true
        return len(p), nil
    }
    return w.buf.Write(p)
}
```

Nếu `over == true` sau khi chạy → verdict `OLE`. (Cách quyết liệt hơn: khi vượt limit thì kill luôn process — sẽ làm ở chương kill.)

**Lưu ý**: đừng dùng `cmd.Output()` hay `cmd.CombinedOutput()` cho judge — chúng đọc toàn bộ output vào memory không giới hạn.

Một lớp bảo vệ ở tầng OS bổ sung: `RLIMIT_FSIZE` giới hạn dung lượng file process được ghi (nếu thí sinh ghi ra file thay vì stdout) — sẽ gặp ở chương rlimit.

### Bài tập 2

1. Viết `cappedWriter` như trên, gắn vào `cmd.Stdout`.
2. Viết script in vô hạn (`yes` command, hoặc `while True: print("x")` trong Python).
3. Chạy với limit = 64KB, kết hợp timeout 1s (tạm dùng `exec.CommandContext` với `context.WithTimeout`, sẽ học kỹ ở chương 3).
4. Kiểm tra: `over == true`, kích thước buffer ≤ 64KB, và process bị dừng (không treo worker).
5. So sánh: thử gán thẳng `bytes.Buffer` không cap, quan sát RAM của tiến trình test tăng (dùng `top`/`htop`) — để thấy vì sao bắt buộc phải cap.

---

## Chương 3 — Timeout & kill process group

### Lý thuyết

**Wall-clock time** (thời gian thực) vs **CPU time** (thời gian CPU thực sự tiêu thụ):
- Wall-clock = `time.Since(start)` quanh `cmd.Run()`. Đây là cái người dùng cảm nhận; bị ảnh hưởng bởi tải máy. Bắt được cả trường hợp chương trình `sleep` hoặc chờ I/O.
- CPU time = lấy từ `Rusage` (chương 6) hoặc rlimit. Ổn định hơn, công bằng hơn giữa các lần chạy, nhưng KHÔNG bắt được chương trình cố tình ngủ.

Judge thường đặt giới hạn **wall-clock = time_limit × hệ_số** (vd ×2 hoặc +1s) làm "tường cứng" để không treo, và dùng CPU time để báo cáo/so sánh. Lý do: chỉ dựa CPU time thì một chương trình `sleep(100)` sẽ không bao giờ bị TLE.

**Đặt timeout** bằng `context`:

```go
ctx, cancel := context.WithTimeout(context.Background(), timeLimit)
defer cancel()
cmd := exec.CommandContext(ctx, "./main")
```

Khi ctx hết hạn, Go gửi `Kill` tới process. NHƯNG mặc định chỉ kill **process leader**, không kill các process con mà thí sinh fork ra → nếu thí sinh fork rồi parent thoát, đám con vẫn sống. Phải kill cả **process group**.

**Process group + kill nhóm** (Go 1.20+):

```go
cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // child thành leader của group mới (pgid = pid)

// Ghi đè hành vi cancel của CommandContext: kill cả group bằng pid âm
cmd.Cancel = func() error {
    return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
cmd.WaitDelay = 200 * time.Millisecond // sau khi cancel, chờ tối đa 200ms rồi buông
```

`syscall.Kill(-pid, ...)` với **pid âm** nghĩa là "gửi signal tới cả process group có pgid = pid" — diệt sạch cây process con.

**Phát hiện TLE** sau khi chạy:

```go
start := time.Now()
err := cmd.Run()
wall := time.Since(start)

if ctx.Err() == context.DeadlineExceeded {
    // -> TLE
}
```

### Bài tập 3

1. Viết `runWithTimeout(ctx, name, args, stdin) (stdout string, wallMs int64, timedOut bool, ps *os.ProcessState)`.
2. Set `Setpgid: true` và `cmd.Cancel` kill group như trên.
3. Test A — chương trình ngủ quá hạn: script `sleep 5`, timeout 1s → `timedOut == true`, `wall ≈ 1s` (không phải 5s).
4. Test B — fork bomb nhẹ: script bash tạo 1 process con chạy `sleep 100` rồi parent in xong thoát. Xác nhận sau timeout, `sleep 100` con **cũng bị kill** (kiểm tra bằng `ps aux | grep sleep` sau khi hàm return). So sánh với khi KHÔNG set `Setpgid`/`Cancel` để thấy con bị "mồ côi" vẫn sống.
5. Test C — chương trình nhanh: chạy < timeout → `timedOut == false`, `wall` nhỏ.

---

## Chương 4 — Giới hạn tài nguyên bằng rlimit

### Lý thuyết

`rlimit` (resource limit) là cơ chế kernel giới hạn tài nguyên **mỗi process**, được **kế thừa** sang process con. Các limit quan trọng cho judge (`golang.org/x/sys/unix`):

| Limit | Giới hạn | Vượt thì sao |
|---|---|---|
| `RLIMIT_CPU` | CPU time (giây) | soft → `SIGXCPU`, hard → `SIGKILL` |
| `RLIMIT_AS` | address space / virtual memory (byte) | `mmap`/`malloc` fail → thường crash `SIGSEGV` hoặc `bad_alloc` |
| `RLIMIT_DATA` | vùng heap (byte) | tương tự AS nhưng hẹp hơn |
| `RLIMIT_STACK` | stack tối đa (byte) | đệ quy sâu → `SIGSEGV` |
| `RLIMIT_NOFILE` | số file descriptor | mở file thứ N+1 fail |
| `RLIMIT_NPROC` | số process của user | chống fork bomb — `fork` fail |
| `RLIMIT_FSIZE` | dung lượng file ghi ra (byte) | ghi vượt → `SIGXFSZ` |

**Cạm bẫy lớn với `RLIMIT_AS` và Java/JVM**: JVM (và một số runtime khác) **reserve** một lượng virtual memory khổng lồ ngay khi khởi động, dù không thực sự dùng (RSS). Nếu bạn set `RLIMIT_AS = 256MB`, JVM có thể fail khởi động dù bài chỉ cần 50MB RSS. Vì vậy: với C/C++/Python, `RLIMIT_AS` ổn; với Java, **đừng dùng `RLIMIT_AS`** — hãy giới hạn memory bằng cgroup (chương 5, đo theo RSS thực) thay vì virtual memory.

**Vấn đề kỹ thuật khi set rlimit trong Go**: `SysProcAttr` không có field trực tiếp để set rlimit cho riêng process con. rlimit lại được kế thừa, nên không thể "set ở parent" mà không ảnh hưởng chính Go runtime. Ba cách thực dụng:

1. **Bọc bằng `prlimit`** (đơn giản nhất): gọi tiện ích hệ thống `prlimit`, nó set limit rồi `exec` chương trình đích.
   ```go
   cmd := exec.CommandContext(ctx, "prlimit",
       "--cpu=2",            // RLIMIT_CPU = 2 giây
       "--nofile=64",        // RLIMIT_NOFILE
       "--nproc=64",         // RLIMIT_NPROC
       "--fsize=10000000",   // RLIMIT_FSIZE = 10MB
       "--", "./main")
   ```
2. **Viết 1 launcher nhỏ** (Go hoặc C): chương trình này gọi `setrlimit()` cho chính nó rồi `execve()` chương trình đích. Build 1 lần, dùng mãi. Kiểm soát tốt nhất.
3. **cgroup** (chương 5): cách robust nhất cho CPU & memory, khuyến nghị làm "tường chính".

Khuyến nghị thực tế: dùng **cgroup cho memory + CPU** (chính xác theo RSS, không dính bẫy JVM), kết hợp **rlimit cho NPROC/NOFILE/FSIZE/STACK** (những thứ cgroup không lo) qua `prlimit` hoặc launcher.

### Bài tập 4

1. Viết hàm chạy chương trình đích qua `prlimit` với `--cpu`, `--nproc`, `--fsize`.
2. Test `RLIMIT_CPU`: chương trình C/C++ vòng lặp vô hạn tính toán (busy loop). Set cpu=2 → quan sát process bị kill (signal `SIGXCPU`/`SIGKILL`) quanh mốc ~2s CPU.
3. Test `RLIMIT_NPROC`: script bash fork bomb có kiểm soát (`:(){ :|:& };:` — **CHỈ chạy trong VM/container dùng-một-lần**). Với nproc=64, fork bomb phải bị chặn, máy không sập.
4. Test `RLIMIT_FSIZE`: chương trình ghi file 100MB, set fsize=10MB → bị `SIGXFSZ`.
5. Ghi chú lại: thử set `RLIMIT_AS=256MB` rồi chạy `java` — quan sát JVM có khởi động nổi không, để tự kiểm chứng bẫy đã nêu.

---

## Chương 5 — Giới hạn tài nguyên bằng cgroups v2

### Lý thuyết

cgroup (control group) là cơ chế kernel nhóm các process và áp **giới hạn cứng** lên cả nhóm. Khác rlimit (per-process, kế thừa, dễ bị qua mặt bằng fork), cgroup giới hạn **tổng** tài nguyên của cả cây process — không thể lách bằng fork. Đây là cơ chế Docker dùng bên dưới.

cgroup v2 là một cây thư mục dưới `/sys/fs/cgroup/`. Tạo 1 cgroup = `mkdir` 1 thư mục con, rồi ghi giá trị vào các file điều khiển:

```
/sys/fs/cgroup/judge-<id>/
├── memory.max      # giới hạn RAM (byte) -> vượt thì OOM killer giết
├── memory.peak     # đỉnh RAM đã dùng (byte) [kernel 5.19+] -> dùng để báo cáo MLE
├── memory.events   # có dòng "oom_kill N" -> N>0 nghĩa là đã bị OOM
├── cpu.max         # "QUOTA PERIOD" (micro giây), vd "50000 100000" = 50% 1 core
├── cpu.stat        # usage_usec -> CPU time đã dùng
├── pids.max        # số process tối đa (chống fork bomb)
└── cgroup.procs    # ghi PID vào đây để đưa process vào cgroup
```

Format `cpu.max`: `"<quota> <period>"` micro giây. `"100000 100000"` = trọn 1 core; `"50000 100000"` = nửa core; `"200000 100000"` = 2 core.

**Đưa process vào cgroup — đúng cách (Go 1.22+)**. Cách cũ là `Start()` rồi ghi PID vào `cgroup.procs` — nhưng có **race**: giữa lúc Start và lúc ghi PID, process có thể đã fork con thoát ra ngoài cgroup. Go 1.22 thêm `UseCgroupFD`/`CgroupFD` vào `SysProcAttr`, đặt process con vào cgroup **ngay lúc tạo** (qua `clone3` + `CLONE_INTO_CGROUP`), triệt tiêu race:

```go
cgPath := "/sys/fs/cgroup/judge-" + id
os.Mkdir(cgPath, 0o755)
os.WriteFile(cgPath+"/memory.max", []byte("268435456"), 0o644) // 256MB
os.WriteFile(cgPath+"/cpu.max",    []byte("100000 100000"), 0o644) // 1 core
os.WriteFile(cgPath+"/pids.max",   []byte("64"), 0o644)

cgDir, _ := os.Open(cgPath)
defer cgDir.Close()

cmd.SysProcAttr = &syscall.SysProcAttr{
    UseCgroupFD: true,
    CgroupFD:    int(cgDir.Fd()),
    Setpgid:     true,
}
```

**Đọc kết quả sau khi chạy**:
```go
peak, _ := os.ReadFile(cgPath + "/memory.peak") // byte, đỉnh RAM -> so với memory.max để biết MLE
events, _ := os.ReadFile(cgPath + "/memory.events") // tìm "oom_kill <n>", n>0 => đã bị OOM => MLE
cpuStat, _ := os.ReadFile(cgPath + "/cpu.stat")  // dòng "usage_usec <n>" -> CPU time
```

**Dọn dẹp**: sau khi chạy xong, `os.Remove(cgPath)` (chỉ xóa được khi không còn process nào trong cgroup — đảm bảo đã kill sạch trước).

**Lưu ý quyền & môi trường**: thao tác cgroup cần quyền ghi vào `/sys/fs/cgroup` (thường cần chạy worker với quyền phù hợp, hoặc cgroup được "delegate"). Trong Docker, cần bật cgroup v2 và cấp quyền tương ứng. Nếu môi trường học không cho đụng cgroup, tạm fallback dùng rlimit (chương 4) cho phần memory với C/C++, và để dành cgroup khi deploy.

### Bài tập 5

1. Viết `prepareCgroup(id string, memBytes int64, cpuQuotaUs, pidsMax int) (cgPath string, fd *os.File, cleanup func(), err error)`.
2. Chạy chương trình "ngốn memory" (C/C++ cấp phát slice lớn dần) trong cgroup `memory.max=64MB`:
   - Quan sát process bị OOM kill khi vượt.
   - Đọc `memory.events`, xác nhận `oom_kill` > 0.
3. Đọc `memory.peak` của một chương trình "hiền" (dùng ~30MB) và in ra — đối chiếu với mức bạn biết.
4. Chạy chương trình busy-loop trong cgroup `cpu.max="50000 100000"` (nửa core) và đo wall-clock — so với khi cho trọn 1 core, thời gian thực phải dài gần gấp đôi.
5. **Quan trọng**: lặp lại bài "ngốn memory" với **Java** dùng cgroup (thay vì `RLIMIT_AS`) — xác nhận JVM khởi động bình thường và vẫn bị giới hạn RSS đúng. Đây là bằng chứng vì sao cgroup thắng rlimit cho Java.

---

## Chương 6 — Đo lường: Rusage & cgroup metrics

### Lý thuyết

Sau `cmd.Wait()`, lấy số liệu tài nguyên của process con từ `Rusage`:

```go
ru := cmd.ProcessState.SysUsage().(*syscall.Rusage)

// CPU time = user time + system time
cpuTime := time.Duration(ru.Utime.Nano() + ru.Stime.Nano())

// Peak RSS:
//   LINUX  -> Maxrss tính bằng KILOBYTE
//   macOS  -> Maxrss tính bằng BYTE   (cạm bẫy khi dev trên Mac!)
maxRSSKb := ru.Maxrss // trên Linux đã là KB
```

**Cạm bẫy `Maxrss`**: đơn vị khác nhau theo OS (Linux: KB, macOS/BSD: byte). Hãy hardcode giả định Linux cho production và ghi rõ trong comment.

So sánh hai nguồn đo memory:
- `Rusage.Maxrss` — peak RSS của **riêng process được Wait** (process leader). Nếu thí sinh fork, RSS các con không gộp vào đây.
- cgroup `memory.peak` — đỉnh RSS của **cả cgroup** (gộp mọi process con). Chính xác hơn cho việc bắt MLE. → Ưu tiên cgroup nếu có; Rusage làm phương án dự phòng cho C/C++ đơn process.

Tương tự cho CPU:
- `Rusage` Utime/Stime — CPU của process leader.
- cgroup `cpu.stat` (`usage_usec`) — CPU của cả nhóm.

Quy ước báo cáo cho judge: `TimeMs` = wall-clock (đo ở chương 3) hoặc CPU time tùy chính sách; `MemoryKb` = `memory.peak`/1024 (cgroup) hoặc `Maxrss` (rlimit fallback).

### Bài tập 6

1. Viết `collectUsage(ps *os.ProcessState) (cpuMs, maxRssKb int64)` từ `Rusage`.
2. Chạy chương trình tốn ~1s CPU và ~50MB → in cả 3 con số: wall-clock (chương 3), CPU time (Rusage), peak RSS (Rusage).
3. Nếu đã làm chương 5: in thêm `memory.peak` và `cpu.stat usage_usec` từ cgroup, so sánh với Rusage. Giải thích chênh lệch (nếu chương trình có fork).
4. Viết 1 chương trình `sleep 2` (không tốn CPU) → quan sát: wall-clock ≈ 2s nhưng CPU time ≈ 0. Đây là minh chứng vì sao không thể chỉ dựa CPU time để bắt TLE.

---

## Chương 7 — Phân loại verdict từ kết quả process

### Lý thuyết

Đây là "bộ não" của judge: từ `ProcessState` + context + cgroup, suy ra verdict. Dùng `syscall.WaitStatus`:

```go
ps := cmd.ProcessState
ws := ps.Sys().(syscall.WaitStatus)

if ws.Signaled() {
    sig := ws.Signal() // bị kill bởi signal
} else if ws.Exited() {
    code := ws.ExitStatus() // thoát bình thường với exit code
}
```

Cây quyết định (theo thứ tự kiểm tra):

```
1. compile lỗi (chương 11)                         -> CE   (không tới đây)
2. ctx.Err() == DeadlineExceeded (timeout của ta)  -> TLE
3. cgroup memory.events có oom_kill > 0            -> MLE
4. ws.Signaled():
      - SIGXCPU                  -> TLE  (vượt RLIMIT_CPU)
      - SIGKILL                  -> nếu cgroup báo OOM thì MLE, không thì coi TLE/RE
                                    (SIGKILL thường do ta kill timeout, hoặc OOM killer)
      - SIGSEGV / SIGABRT / ...  -> RE   (crash; nếu set RLIMIT_AS thì có thể là MLE)
      - SIGXFSZ                  -> OLE  (vượt RLIMIT_FSIZE)
5. ws.Exited():
      - code != 0                -> RE   (chương trình tự báo lỗi / runtime exception)
      - code == 0                -> output check (chương 8) -> AC hoặc WA
6. cờ output bom (chương 2) over == true            -> OLE
```

**Tại sao thứ tự quan trọng**: cùng một `SIGKILL` có thể đến từ "ta kill vì timeout" HOẶC "OOM killer giết vì hết RAM". Phải kiểm tra `ctx.Err()` (timeout) và `memory.events` (OOM) **trước** khi diễn giải signal, nếu không sẽ gán nhầm verdict. Ưu tiên dùng tín hiệu trực tiếp (deadline của context, oom_kill của cgroup) thay vì đoán từ signal.

Gói gọn thành hàm:

```go
func classify(
    ctxErr error,
    ps *os.ProcessState,
    oomKilled bool,
    outputOver bool,
) Verdict {
    if ctxErr == context.DeadlineExceeded {
        return TLE
    }
    if oomKilled {
        return MLE
    }
    ws := ps.Sys().(syscall.WaitStatus)
    if ws.Signaled() {
        switch ws.Signal() {
        case syscall.SIGXCPU:
            return TLE
        case syscall.SIGXFSZ:
            return OLE
        case syscall.SIGKILL:
            return TLE // đã loại OOM ở trên; còn lại coi là TLE biên
        default:
            return RE // SIGSEGV, SIGABRT...
        }
    }
    if outputOver {
        return OLE
    }
    if ws.ExitStatus() != 0 {
        return RE
    }
    return "" // "" = chạy sạch, cần so output ở chương 8
}
```

### Bài tập 7

Viết `classify` như trên + bộ test, mỗi case dựng tình huống thật rồi assert verdict:
1. `sleep` quá hạn → `TLE` (qua `ctx.Err()`).
2. Cấp phát vượt `memory.max` (cgroup) → `MLE` (qua `oomKilled`).
3. Chương trình `*(int*)0 = 1;` (segfault) → `RE` (SIGSEGV).
4. Chương trình `return 1;` → `RE` (exit code khác 0).
5. Chương trình in `42` đúng, exit 0 → trả `""` (chờ so output).
6. Vượt `RLIMIT_FSIZE` → `OLE` (SIGXFSZ).

---

## Chương 8 — So sánh output & tính verdict cuối

### Lý thuyết

Khi process chạy sạch (verdict tạm = `""`), so output thực với expected. Bốn kiểu so sánh, từ đơn giản tới phức tạp:

1. **Exact match**: byte-for-byte. Quá khắt khe (1 dấu cách thừa cuối dòng là WA) — hiếm khi dùng.
2. **Token-based (chuẩn phổ biến nhất)**: tách cả 2 output thành các "token" (chuỗi không khoảng trắng), so từng token. Bỏ qua khác biệt về số lượng dấu cách / dòng trống / khoảng trắng cuối. Đây là default của hầu hết online judge.
   ```go
   func tokensEqual(got, want string) bool {
       return strings.Fields(got) // tách theo mọi whitespace
           // so sánh slice
   }
   ```
   (Dùng `strings.Fields` cho cả hai rồi so sánh từng phần tử.)
3. **Line-based with trailing trim**: so từng dòng sau khi `TrimRight` khoảng trắng, bỏ dòng trống cuối. Dùng khi format dòng quan trọng.
4. **Float comparison**: với bài có đáp án thực, so từng số với sai số `|a-b| <= eps` hoặc sai số tương đối. Cần khi đề ghi "chấp nhận sai số 1e-6".
5. **Special judge (checker)**: bài có **nhiều đáp án đúng** (vd "in ra bất kỳ đường đi ngắn nhất nào"). Không thể so với 1 expected cố định — phải chạy 1 chương trình checker riêng nhận `(input, output_thí_sinh)` và tự phán đúng/sai. Checker cũng là 1 process con, chạy qua đúng cơ chế `os/exec`.

Khuyến nghị: implement token-based làm mặc định, thêm float comparison, để special judge là tùy chọn cho sau.

**Tổng hợp verdict cuối** cho cả submission:
```go
final := AC
for _, tc := range testCases {
    v := judgeOne(tc) // có thể là AC/WA/TLE/MLE/RE/OLE
    final = worse(final, v)
    if v != AC && stopOnFirstFail {
        break // tùy chính sách: dừng ngay hay chấm hết
    }
}
```
Chính sách `stopOnFirstFail`: ICPC thường dừng ngay test fail đầu tiên; nhiều hệ thống chấm theo subtask thì chấm hết để cho điểm từng phần. DARE-ka live battle có thể muốn dừng sớm để phản hồi nhanh.

### Bài tập 8

1. Viết `compareTokens(got, want string) bool` (token-based).
2. Viết `compareFloats(got, want string, eps float64) bool`.
3. Test các case: output đúng nhưng thừa khoảng trắng/xuống dòng → token-based vẫn AC; sai 1 token → WA; số lệch trong eps → AC, lệch ngoài eps → WA.
4. Viết `finalVerdict(results []Verdict, stopOnFirstFail bool) Verdict` dùng `worse` từ chương 0.
5. (Tùy chọn) Viết khung gọi special judge: chạy checker binary với stdin = `input + "\n" + output`, exit 0 = AC, exit khác = WA.

---

## Chương 9 — Quản lý sandbox filesystem

### Lý thuyết

Mỗi submission cần 1 thư mục tạm riêng chứa: source file, binary đã compile, (tùy chọn) file input/output. Phải đảm bảo **luôn được dọn** dù có lỗi.

```go
func prepareSandbox(id, lang, code string) (dir string, srcPath string, cleanup func(), err error) {
    dir, err = os.MkdirTemp("", "judge-"+id+"-*")
    if err != nil {
        return "", "", func() {}, err
    }
    cleanup = func() { os.RemoveAll(dir) } // ĐĂNG KÝ NGAY

    cfg := langs[lang]
    srcPath = filepath.Join(dir, cfg.SourceFile)
    if err = os.WriteFile(srcPath, []byte(code), 0o600); err != nil {
        cleanup()
        return "", "", func() {}, err
    }
    return dir, srcPath, cleanup, nil
}
```

Ở caller, dùng `defer cleanup()` ngay sau khi gọi để dù compile/run lỗi hay panic, thư mục vẫn bị xóa:

```go
dir, src, cleanup, err := prepareSandbox(id, lang, code)
if err != nil { ... }
defer cleanup()
```

`os.Chmod(binPath, 0o755)` để set bit thực thi cho binary sau compile (Go thường giữ quyền từ compiler nhưng set tường minh cho chắc).

**Lưu ý quyền & vị trí**: nếu chạy code thí sinh dưới user `nobody` (chương 10), thư mục sandbox phải để user đó đọc/ghi được. `os.MkdirTemp` mặc định `/tmp` — đảm bảo `/tmp` writable trong container. Để cách ly mạnh hơn, có thể đặt sandbox trong 1 mount riêng (tmpfs) để mỗi run hoàn toàn sạch và nhanh.

### Bài tập 9

1. Viết `prepareSandbox` như trên + `langs` map (xem chương 11).
2. Test: tạo sandbox, ghi 1 file C++, `compile`, rồi `defer cleanup()`. Sau khi hàm return, dùng `os.Stat(dir)` xác nhận thư mục đã biến mất.
3. Test panic: cố tình `panic` giữa chừng trong 1 hàm có `defer cleanup()` + `recover` ở ngoài → xác nhận thư mục vẫn được xóa.
4. Test compile lỗi: source C++ sai cú pháp → `cleanup` vẫn chạy, không để lại rác trong `/tmp`.

---

## Chương 10 — Bảo mật & cách ly

### Lý thuyết

Bạn đang chạy **code không tin cậy**. Các lớp phòng thủ, từ dễ tới khó (làm theo thứ tự, mỗi lớp giảm rủi ro thêm):

**Lớp 1 — Chạy non-root** (bắt buộc, dễ): tạo 1 user không đặc quyền (vd `nobody` hoặc 1 user `judge` riêng), chạy process con dưới UID/GID đó:
```go
cmd.SysProcAttr = &syscall.SysProcAttr{
    Credential: &syscall.Credential{Uid: nobodyUid, Gid: nobodyGid},
    Setpgid:    true,
    // ... cgroup fd
}
```
Process con không thể ghi đè file hệ thống, không kill process của user khác. Worker chính phải chạy với quyền đủ để `setuid` xuống (thường cần là root hoặc có capability phù hợp — cân nhắc đánh đổi).

**Lớp 2 — Giới hạn tài nguyên** (đã làm): rlimit + cgroup chống ngốn CPU/RAM/process/file.

**Lớp 3 — Cách ly filesystem bằng `chroot`** (trung bình): nhốt process vào 1 thư mục gốc giả, không thấy filesystem thật:
```go
cmd.SysProcAttr.Chroot = sandboxRootDir
```
Cần chuẩn bị 1 "rootfs" tối thiểu trong sandbox (chứa libc, binary cần thiết...) — khá việc. Container đã làm sẵn việc này nên nhiều judge bỏ qua chroot khi đã chạy trong container.

**Lớp 4 — Linux namespaces** (khó): tách process khỏi host về PID, mount, network, IPC, UTS:
```go
cmd.SysProcAttr.Cloneflags = syscall.CLONE_NEWPID |
    syscall.CLONE_NEWNS | syscall.CLONE_NEWNET |
    syscall.CLONE_NEWIPC | syscall.CLONE_NEWUTS
```
`CLONE_NEWNET` (không có network) đặc biệt quan trọng — chặn thí sinh gọi ra ngoài internet. Đây chính là những gì Docker dựng cho bạn; tự làm bằng tay phức tạp nhưng nhẹ và nhanh hơn spawn container mỗi lần.

**Lớp 5 — seccomp** (khó, nâng cao): lọc syscall, chỉ cho phép tập syscall an toàn (chặn `ptrace`, `mount`, các network syscall...). Dùng `libseccomp-golang`. Để dành cho giai đoạn cứng hóa sau.

**Lộ trình thực tế trong 1 tuần**: Lớp 1 + Lớp 2 là tối thiểu bắt buộc và làm được ngay. `CLONE_NEWNET` (chỉ riêng tắt network) nên thêm vì rẻ và chặn được lỗ hổng lớn nhất (exfiltration). chroot/namespace đầy đủ và seccomp để dành sau hoặc dựa vào container.

### Bài tập 10

1. Tạo user `judge` (hoặc dùng `nobody`), set `Credential` cho process con. Test: process con thử ghi file vào `$HOME` của user worker → phải `permission denied`.
2. Thêm `CLONE_NEWNET`. Test: chương trình con thử `curl`/mở socket ra ngoài → phải thất bại (không có network). (Cần chạy worker với quyền tạo namespace.)
3. (Tùy chọn, nếu có thời gian) Thử `Chroot` vào 1 thư mục có sẵn rootfs tối thiểu (vd dùng `busybox`), chạy 1 lệnh đơn giản bên trong.
4. Ghi chú: nếu môi trường không cho tạo namespace/chroot (thiếu quyền), nêu rõ trong README rằng cách ly mạnh sẽ dựa vào container runtime khi deploy.

---

## Chương 11 — Compile đa ngôn ngữ

### Lý thuyết

Go chỉ điều phối — toolchain thật (`g++`, `javac`/`java`, `python3`, `node`) phải được **cài sẵn trong image**. Mỗi ngôn ngữ có cấu hình compile/run khác nhau:

```go
type LangConfig struct {
    SourceFile string   // tên file ghi code vào
    CompileCmd []string // nil nếu là ngôn ngữ thông dịch
    RunCmd     []string // lệnh chạy (sau compile)
    TimeFactor float64  // hệ số nhân time limit (Java/Python chậm hơn C++)
}

var langs = map[string]LangConfig{
    "cpp": {
        SourceFile: "main.cpp",
        CompileCmd: []string{"g++", "main.cpp", "-O2", "-std=c++17", "-o", "main"},
        RunCmd:     []string{"./main"},
        TimeFactor: 1.0,
    },
    "java": {
        SourceFile: "Main.java",
        CompileCmd: []string{"javac", "Main.java"},
        RunCmd:     []string{"java", "Main"},
        TimeFactor: 2.0, // JVM khởi động chậm + JIT warm-up
    },
    "python": {
        SourceFile: "main.py",
        CompileCmd: nil,
        RunCmd:     []string{"python3", "main.py"},
        TimeFactor: 3.0, // thông dịch, chậm hơn nhiều
    },
}
```

**Bước compile** chạy 1 lần trước mọi test case:
- Compile thường KHÔNG tính vào time limit của bài (nhưng nên có timeout riêng, vd 10s, để chống "compile bomb" — code C++ template metaprogramming có thể làm `g++` chạy hàng phút).
- Compile chạy với quyền bình thường (không cần sandbox chặt như khi chạy code — nhưng vẫn nên giới hạn thời gian/bộ nhớ vì compiler cũng có thể bị lạm dụng).
- Nếu `CompileCmd` exit khác 0 → verdict toàn submission = `CE`, đính kèm stderr của compiler để thí sinh xem lỗi. Không chạy test nào.

**Bước run** dùng `RunCmd`, áp time limit = `problemTimeLimit × TimeFactor`, kèm toàn bộ sandbox (cgroup/rlimit/credential).

**Java và memory**: như đã nói ở chương 4 — không dùng `RLIMIT_AS` cho Java, giới hạn bằng cgroup `memory.max`. Ngoài ra Java cần đủ memory cho cả JVM (heap + metaspace + overhead), nên memory limit cho Java thường phải nới rộng hơn C++ cho cùng 1 bài.

### Bài tập 11

1. Hoàn thiện `langs` cho ít nhất `cpp`, `python`, `java`.
2. Viết `compile(ctx, dir string, cfg LangConfig) (ok bool, stderr string)`:
   - Nếu `CompileCmd == nil` → trả `ok = true` luôn (ngôn ngữ thông dịch).
   - Chạy compile trong `dir` (set `cmd.Dir = dir`), timeout 10s.
   - Trả về stderr nếu lỗi.
3. Test: submit 1 bài "đọc 2 số in tổng" bằng cả 3 ngôn ngữ, xác nhận compile/run đúng cho từng loại.
4. Test CE: source C++ sai cú pháp → `ok = false`, stderr chứa thông báo lỗi của `g++`.
5. Đo và so sánh thời gian chạy cùng 1 thuật toán giữa C++/Python/Java để tự cảm nhận vì sao cần `TimeFactor`.

---

## Chương 12 — Ghép nối: Judge function hoàn chỉnh

### Lý thuyết

Giờ ráp mọi mảnh thành 1 hàm `Judge` nhận submission, trả `JudgeResult`. Đây là xương sống của Judger Worker trong DARE-ka.

```go
type Submission struct {
    ID        string
    Language  string
    Code      string
    TimeLimit time.Duration // time limit gốc của bài (cho C++)
    MemoryKb  int64         // memory limit của bài
    Tests     []TestCase
}

type TestCase struct {
    Input    string
    Expected string
}

func Judge(sub Submission) JudgeResult {
    cfg, ok := langs[sub.Language]
    if !ok {
        return JudgeResult{Verdict: CE, Tests: nil} // ngôn ngữ không hỗ trợ
    }

    // 1. sandbox
    dir, _, cleanup, err := prepareSandbox(sub.ID, sub.Language, sub.Code)
    if err != nil {
        return JudgeResult{Verdict: RE}
    }
    defer cleanup()

    // 2. compile
    compileCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    if okc, ceMsg := compile(compileCtx, dir, cfg); !okc {
        return JudgeResult{Verdict: CE, Tests: []TestResult{{Verdict: CE, Message: ceMsg}}}
    }

    // 3. chạy từng test case
    final := AC
    var results []TestResult
    effTimeLimit := time.Duration(float64(sub.TimeLimit) * cfg.TimeFactor)

    for i, tc := range sub.Tests {
        tr := runTestCase(dir, cfg, tc, effTimeLimit, sub.MemoryKb, sub.ID, i)
        results = append(results, tr)
        final = worse(final, tr.Verdict)
        if tr.Verdict != AC {
            break // stopOnFirstFail (tùy chính sách)
        }
    }

    return JudgeResult{Verdict: final, Tests: results}
}
```

Trong đó `runTestCase` gom chương 2–8:

```go
func runTestCase(dir string, cfg LangConfig, tc TestCase,
    timeLimit time.Duration, memKb int64, subID string, idx int) TestResult {

    // cgroup riêng cho test này
    cgPath, cgFd, cgCleanup, err := prepareCgroup(
        fmt.Sprintf("%s-%d", subID, idx), memKb*1024, 100000, 64)
    if err != nil { return TestResult{Verdict: RE, Message: err.Error()} }
    defer cgCleanup()

    ctx, cancel := context.WithTimeout(context.Background(), timeLimit)
    defer cancel()

    cmd := exec.CommandContext(ctx, cfg.RunCmd[0], cfg.RunCmd[1:]...)
    cmd.Dir = dir
    cmd.Stdin = strings.NewReader(tc.Input)
    out := &cappedWriter{limit: 64 * 1024}
    cmd.Stdout = out
    cmd.Stderr = io.Discard

    cmd.SysProcAttr = &syscall.SysProcAttr{
        UseCgroupFD: true,
        CgroupFD:    int(cgFd.Fd()),
        Setpgid:     true,
        // Credential: &syscall.Credential{Uid: nobodyUid, Gid: nobodyGid}, // chương 10
        // Cloneflags: syscall.CLONE_NEWNET,                                 // chương 10
    }
    cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
    cmd.WaitDelay = 200 * time.Millisecond

    start := time.Now()
    _ = cmd.Run()
    wallMs := time.Since(start).Milliseconds()

    oom := cgroupOOMKilled(cgPath)
    peakKb := cgroupPeakKb(cgPath)

    v := classify(ctx.Err(), cmd.ProcessState, oom, out.over)
    if v == "" {
        if compareTokens(out.buf.String(), tc.Expected) {
            v = AC
        } else {
            v = WA
        }
    }
    return TestResult{Verdict: v, TimeMs: wallMs, MemoryKb: peakKb}
}
```

Ráp với worker pool + RabbitMQ (đã học ở track OS trước): mỗi worker goroutine consume job → gọi `Judge(sub)` → publish `JudgeResult` lên `judge.results`.

### Bài tập 12 (Capstone)

Ráp toàn bộ thành judge chạy thật:

1. Hoàn thiện `Judge` + `runTestCase` dùng tất cả hàm đã viết ở chương 1–11.
2. Chuẩn bị 1 bài test: "đọc 2 số nguyên, in tổng", time limit 1s, memory 256MB, 3 test case.
3. Submit và xác nhận đúng verdict cho từng loại lời giải:
   - Lời giải đúng (C++/Python/Java) → `AC`.
   - In sai (cộng thành trừ) → `WA`.
   - `while(true){}` → `TLE`.
   - Cấp phát mảng khổng lồ → `MLE`.
   - Chia cho 0 / segfault → `RE`.
   - Source sai cú pháp → `CE`.
   - In vô hạn → `OLE`.
4. Bọc bằng worker pool: gửi 20 submission đồng thời vào 1 channel, N worker (N = `runtime.NumCPU()`) xử lý song song, in tổng thời gian. So sánh N=1 vs N=NumCPU.
5. (Mở rộng) Nối với RabbitMQ: consume `judge.jobs`, publish `judge.results`, để Submission Service cập nhật DB — hoàn thành luồng end-to-end.

Khi bài capstone chạy đúng cả 7 verdict, bạn đã có một judge engine thật sự — đủ làm lõi chấm bài cho DARE-ka.

---

## Phụ lục — Checklist kiến thức trước khi code

Đánh dấu khi đã nắm chắc:

- [ ] `exec.Command` không qua shell; lấy exit code qua `*exec.ExitError` / `ProcessState`
- [ ] Gắn stdin từ string/file; cap output chống output bomb
- [ ] `context.WithTimeout` + `Setpgid` + `cmd.Cancel` kill cả process group
- [ ] Phân biệt wall-clock vs CPU time, biết khi nào dùng cái nào
- [ ] rlimit: các constant chính + bẫy `RLIMIT_AS` với Java
- [ ] cgroup v2: `memory.max`/`cpu.max`/`pids.max`, `UseCgroupFD` (Go 1.22), đọc `memory.peak`/`memory.events`
- [ ] Đọc `Rusage` (Maxrss đơn vị KB trên Linux)
- [ ] `WaitStatus`: `Signaled()`/`Signal()`/`ExitStatus()`; cây quyết định verdict
- [ ] So output token-based + float; tổng hợp verdict cuối bằng `worse`
- [ ] sandbox dir + `defer cleanup` đảm bảo dọn sạch
- [ ] non-root `Credential` + `CLONE_NEWNET` tắt mạng
- [ ] `langs` map đa ngôn ngữ + `TimeFactor` + phát hiện `CE`

Khi tất cả ô đã tick và bài capstone xanh, phần OS interaction coi như đủ vững.
