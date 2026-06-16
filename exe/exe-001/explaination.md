# Giải thích `runOnce` và `judge` — OS Interaction trong Go

---

## Phần 1: Hệ điều hành "chạy một chương trình" nghĩa là gì

Trước khi đọc code, cần hiểu vài khái niệm OS nền tảng.

Khi bạn chạy một chương trình (vd `./main`), hệ điều hành tạo ra một **process** — hãy hình dung nó như một "nhân viên" được OS thuê để làm một việc. Nhân viên này có:

### Ba "đường ống" giao tiếp với thế giới bên ngoài (standard streams)

| Stream | Tên | Vai trò | Ẩn dụ |
|---|---|---|---|
| `stdin` | Standard Input | Nhận dữ liệu vào | Khay "hồ sơ cần xử lý" trên bàn |
| `stdout` | Standard Output | Gửi kết quả ra | Khay "hồ sơ đã xong" |
| `stderr` | Standard Error | Báo lỗi riêng | Tờ giấy nhớ "có vấn đề ở đây" |

Tách stdout và stderr ra làm hai đường riêng là có chủ đích: kết quả thật đi một đường, thông báo lỗi đi đường khác — để không bị lẫn lộn "đâu là đáp án, đâu là lời than phiền".

### Exit code — "mã thoát" khi kết thúc

Khi chương trình kết thúc, nó để lại một con số:
- **0** = thành công
- **khác 0** = có lỗi

Đây là cách một chương trình báo cho chương trình gọi nó biết "tôi làm được hay không".

> **Toàn bộ `runOnce` chỉ làm một việc**: thuê một process, nối ba đường ống vào, đợi nó xong, rồi đọc kết quả + mã thoát.

---

## Phần 2: Mổ xẻ `runOnce`

```go
func runOnce(name string, args []string, stdinData string) (stdout, stderr string, exitCode int, err error) {
```

**Đầu vào:**
- `name` — tên chương trình muốn chạy (vd `"g++"`, `"./main"`)
- `args` — các tham số cho nó (vd `[]string{"-O2", "-o", "main"}`)
- `stdinData` — dữ liệu đẩy vào stdin (vd `"5\n"`)

**Đầu ra:** stdout, stderr, exit code, và lỗi hệ thống (nếu có).

---

### Bước 1: Mô tả lệnh (chưa chạy)

```go
cmd := exec.Command(name, args...)
```

Dòng này **chưa chạy gì cả** — nó chỉ tạo ra một "bản kế hoạch" mô tả "tôi định thuê nhân viên tên `name`, giao cho anh ta các tham số `args`". Giống như viết phiếu giao việc nhưng chưa đưa cho ai.

Vì sao tách "mô tả" và "chạy"? Để bạn có cơ hội **chuẩn bị** mọi thứ (nối đường ống, đặt giới hạn) *trước khi* nhân viên bắt đầu làm.

---

### Bước 2: Nối stdin

```go
cmd.Stdin = strings.NewReader(stdinData)
```

Nối đường ống stdin. `strings.NewReader(stdinData)` biến chuỗi của bạn thành một "nguồn đọc được" — đặt sẵn xấp hồ sơ vào khay đầu vào của nhân viên.

Khi chương trình C++ gọi `std::cin >> n`, nó đọc từ chính đường ống này. Bạn đưa `"5\n"` thì `std::cin >> n` đọc được số 5.

---

### Bước 3: Hứng stdout và stderr

```go
var outBuf, errBuf bytes.Buffer
cmd.Stdout = &outBuf
cmd.Stderr = &errBuf
```

`bytes.Buffer` là một "cái thùng chứa" trong bộ nhớ — ban đầu rỗng.

Bình thường khi chương trình `std::cout << ...`, kết quả đi ra màn hình terminal. Nhưng bạn **không muốn nó ra màn hình** — bạn muốn *hứng lấy* để so sánh với đáp án. Nên bạn nói với OS:

> "Đường ống stdout của nhân viên này, thay vì đổ ra màn hình, hãy đổ vào cái thùng `outBuf` của tôi."

Tương tự stderr đổ vào `errBuf`.

Dùng `&outBuf` (có dấu `&`) vì cần đưa **địa chỉ** cái thùng để OS đổ dữ liệu vào đúng thùng gốc — nếu truyền bản sao thì Go ghi vào bản sao, `outBuf` gốc vẫn rỗng.

Sau khi nhân viên làm xong, hai cái thùng này chứa toàn bộ thứ chương trình đã in ra.

---

### Bước 4: Thật sự chạy

```go
err = cmd.Run()
```

**Bây giờ mới thật sự chạy.** `Run()` nói với OS "thuê nhân viên đi, theo đúng tờ phiếu `cmd` đã chuẩn bị". OS tạo process, chạy chương trình, và `Run()` **đứng đợi** tới khi chương trình kết thúc hoàn toàn.

Trong lúc đợi, mọi thứ chương trình in ra stdout/stderr được đổ vào hai cái thùng. `Run()` trả về `err` báo "chạy có suôn sẻ không".

```go
stdout = outBuf.String()
stderr = errBuf.String()
```

Đổ hai cái thùng ra thành chuỗi để trả về.

---

### Bước 5: Phân loại kết quả — ba khả năng

```go
if err != nil {
    if exitErr, ok := err.(*exec.ExitError); ok {
        exitCode = exitErr.ExitCode()
        return stdout, stderr, exitCode, nil
    }
    return stdout, stderr, -1, err
}
return stdout, stderr, 0, nil
```

Đây là phần quan trọng nhất. Có **ba khả năng** khác nhau hoàn toàn:

**Khả năng 1 — `err == nil`** (dòng cuối):
Chương trình chạy ngon, kết thúc với exit code 0. Mọi thứ ổn.

**Khả năng 2 — `err` là kiểu `*exec.ExitError`**:
Chương trình **có chạy được**, nhưng kết thúc với exit code **khác 0** (báo lỗi). Cái `err.(*exec.ExitError)` là phép thử "lỗi này có phải loại 'chương trình thoát với mã lỗi' không?". Ta lấy ra exit code và trả `nil` ở vị trí err — ngụ ý "đây không phải lỗi của `runOnce`, mà là chương trình con tự báo lỗi của nó".

**Khả năng 3 — `err` là loại khác**:
Chương trình **không chạy được ngay từ đầu** — không tìm thấy file, không có quyền thực thi, đường dẫn sai... Đây mới là lỗi thật sự của việc gọi, nên trả `-1` và trả luôn `err`.

**Vì sao phải tách 3 khả năng?**

| Khả năng | Ý nghĩa | Verdict |
|---|---|---|
| 1 — exit 0 | Chạy xong bình thường | AC hoặc WA (cần so output) |
| 2 — exit khác 0 | Thí sinh code lỗi, crash, return lỗi | RE |
| 3 — không chạy được | Lỗi môi trường của bạn (judge sai) | Không phải lỗi thí sinh |

Gộp chung lại là gán nhầm verdict — điều tệ nhất có thể xảy ra với một judge.

---

## Phần 3: Mổ xẻ `judge`

`judge` dùng `runOnce` để chấm một bài hoàn chỉnh. Đọc theo trình tự thời gian.

### Bước 1: Code thí sinh đến dưới dạng chuỗi

```go
code := `#include <iostream>
    int main() { ... }`
```

Đây là **code của thí sinh** dưới dạng chuỗi — mô phỏng việc thí sinh gửi code lên. Trong hệ thống thật, code đến qua RabbitMQ message dưới dạng `string`, không phải file vật lý.

---

### Bước 2: Tạo sandbox riêng cho mỗi lần chấm

```go
dir, _ := os.MkdirTemp("", "judge-*")
defer os.RemoveAll(dir)
```

`os.MkdirTemp` tạo một **thư mục tạm** ở `/tmp` (Windows: `Temp`), tên kiểu `judge-2356345426`.

Vì sao cần thư mục riêng cho mỗi lần chấm?
- **Cô lập**: file của bài này không lẫn với bài khác đang chạy song song.
- **Dọn sạch**: `defer os.RemoveAll(dir)` đặt lịch "xóa thư mục này khi hàm kết thúc" — dù kết thúc bình thường hay giữa chừng. Không để rác lại sau mỗi lần chấm.

---

### Bước 3: Ghi code ra file + compile

```go
srcPath := filepath.Join(dir, "main.cpp")
binPath := filepath.Join(dir, "main")
os.WriteFile(srcPath, []byte(code), 0o600)
```

`srcPath` và `binPath` chỉ là **chuỗi đường dẫn** — chưa có file nào tồn tại. `os.WriteFile` mới thật sự **ghi chuỗi code ra đĩa** thành file `main.cpp`.

```go
_, ceError, ceCode, err := runOnce("g++", []string{srcPath, "-O2", "-o", binPath}, "")
```

Dùng chính `runOnce` để thuê nhân viên `g++` (trình biên dịch C++):
- Đầu vào: file `srcPath` (main.cpp)
- Đầu ra: file `binPath` (main — binary thực thi)
- Stdin rỗng `""` vì compiler không cần input

```go
if err != nil {
    fmt.Println("không chạy được g++:", err)  // máy chưa cài g++
    return
}
if ceCode != 0 {
    fmt.Println("Verdict: CE")   // code thí sinh sai cú pháp
    fmt.Println(ceError)         // in lỗi biên dịch cho thí sinh
    return
}
```

Hai trường hợp khác nhau:
- `err != nil` → không chạy được g++ → lỗi môi trường của bạn.
- `ceCode != 0` → g++ chạy nhưng báo lỗi → code thí sinh sai → **CE**.

---

### Bước 4: Chạy binary với từng test case

```go
for i, tc := range tests {
    stdout, _, exCode, err := runOnce(binPath, nil, tc.Input)
```

Với mỗi test case, lại dùng `runOnce` — nhưng lần này thuê chính **binary thí sinh** (`binPath`), đẩy `tc.Input` vào stdin.

```go
    switch {
    case err != nil || exCode != 0:
        verdict = "RE"
    case stdout == tc.Expected:
        verdict = "AC"
    default:
        verdict = "WA"
    }
```

Phân loại verdict đơn giản:
- `err != nil || exCode != 0` → chương trình không chạy hoặc crash → **RE**
- `stdout == tc.Expected` → output khớp → **AC**
- còn lại → output không khớp → **WA**

> **Lưu ý**: `break` trong mỗi case của `switch` là thừa trong Go — Go tự dừng mỗi case, không cần `break` như C/Java. Nó không gây hại nhưng không cần thiết.

---

## Phần 4: Dòng thời gian tổng thể

```
code (string)
    |
    v
os.WriteFile -> main.cpp (file trên đĩa)
    |
    v
runOnce("g++", ...) -> compile -> main (binary)
    |
    | compile lỗi? -> CE, dừng
    v
for mỗi test case:
    runOnce("./main", nil, input)
        |
        +-- err != nil hoặc exCode != 0  -> RE
        +-- stdout == expected            -> AC
        +-- stdout != expected            -> WA
    |
    v
finalVerdict
    |
    v
defer os.RemoveAll(dir) -> xóa sạch thư mục tạm
```

---

## Phần 5: `5 / 0` trong code — điều bất ngờ

Bạn sửa code thí sinh thành:
```cpp
std::cout << n * 2 << "\n" << 5 / 0;
```

Tưởng sẽ tạo RE lúc chạy, nhưng có điều bất ngờ: **`5 / 0` với hằng số có thể bị bắt ngay lúc compile**. `g++` thấy "chia cho hằng số 0" là điều chắc chắn sai, nên tùy phiên bản:
- Compile báo lỗi → bạn nhận **CE**, không phải RE.
- Compile chỉ cảnh báo, vẫn tạo binary → lúc chạy crash (SIGFPE) → **RE**.

Nếu muốn chắc chắn tạo RE *lúc chạy*, hãy chia cho một **biến** bằng 0 để g++ không đoán trước được:

```cpp
int n;
std::cin >> n;
int d = n - n;        // = 0, nhưng g++ không biết trước
std::cout << 5 / d;   // chia 0 lúc chạy -> SIGFPE -> RE thật
```

---

## Phần 6: Còn thiếu gì để phân biệt đủ 7 verdict

Code hiện tại phân biệt được **4 loại**: CE, AC, WA, RE. Còn thiếu:

| Verdict | Thiếu gì | Chương |
|---|---|---|
| **TLE** | Không có timeout → `runOnce` treo vĩnh viễn nếu code vòng lặp vô hạn | Chương 3: `context.WithTimeout` |
| **MLE** | Không giới hạn RAM → chương trình ngốn thoải mái | Chương 4-5: rlimit / cgroup |
| **OLE** | `bytes.Buffer` không giới hạn → in vô hạn làm worker hết RAM | Chương 2: `cappedWriter` |

Ngoài ra, khi có TLE và MLE, cần **đọc `WaitStatus`** (thay vì chỉ nhìn exit code) để phân biệt đúng: cả RE/TLE/MLE đều có thể có exit code `-1` nếu bị signal giết — phải xem *signal nào* mới ra verdict đúng (chương 7).

---

## Tóm tắt một câu cho mỗi hàm

**`runOnce`**: thuê một process, nối stdin (đưa input vào) + hứng stdout/stderr (lấy output ra) + đợi xong + đọc exit code, rồi phân biệt ba tình huống: chạy ổn / chương trình tự báo lỗi / không chạy được.

**`judge`**: ghi code thí sinh ra file tạm → gọi `runOnce("g++", ...)` để compile (lỗi → CE) → với mỗi test case, gọi `runOnce(binary, ...)` đẩy input vào → so output với đáp án để ra AC/WA/RE → dọn thư mục tạm khi xong.
