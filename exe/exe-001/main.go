package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type TestCase struct {
	Input    string
	Expected string
}

func runOnce(name string, args []string, stdinData string) (stdout, stderr string, exitCode int, err error) {
	cmd := exec.Command(name, args...) // mô tả lệnh (chưa chạy)

	cmd.Stdin = strings.NewReader(stdinData) // bơm input vào stdin

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf // hứng stdout
	cmd.Stderr = &errBuf // hứng stderr

	err = cmd.Run()

	stdout = outBuf.String()
	stderr = errBuf.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			return stdout, stderr, exitCode, nil
		}
		return stdout, stderr, -1, err
	}

	return stdout, stderr, 0, nil
}
func judge() {
	code := `#include <iostream>
		int main() {
				int n;
				std::cin >> n;
				std::cout << n * 2 << "\n" << 5 / 0;
				return 0;
		}`

	dir, _ := os.MkdirTemp("", "judge-*")
	defer os.RemoveAll(dir)

	srcPath := filepath.Join(dir, "main.cpp")
	binPath := filepath.Join(dir, "main")
	os.WriteFile(srcPath, []byte(code), 0o600)
	//                            ^^^ ->  code mình gửi lên

	_, ceError, ceCode, err := runOnce("g++", []string{srcPath, "-O2", "-o", binPath}, "")
	if err != nil {
		fmt.Println("không chạy được g++:", err)
		return
	}
	if ceCode != 0 {
		fmt.Println("Verdict: CE")
		fmt.Println(ceError)
		return
	}

	tests := []TestCase{
		{Input: "5\n", Expected: "10\n"},
		{Input: "0\n", Expected: "0\n"},
		{Input: "-3\n", Expected: "-6\n"},
		{Input: "100\n", Expected: "200\n"},
	}

	finalVerdict := "AC"

	for i, tc := range tests {
		stdout, _, exCode, err := runOnce(binPath, nil, tc.Input)
		var verdict string
		switch {
		case err != nil || exCode != 0:
			{
				verdict = "RE"
				break
			}
		case stdout == tc.Expected:
			{
				verdict = "AC"
				break
			}
		default:
			verdict = "WA"
		}
		fmt.Printf("Test %d: %s (input=%q, got=%q, want=%q)\n",
			i+1, verdict, tc.Input, stdout, tc.Expected)
		if verdict != "AC" {
			finalVerdict = verdict
		}
	}
	fmt.Println("Final Verdict::" + finalVerdict)
}

func main() {
	judge()
}
