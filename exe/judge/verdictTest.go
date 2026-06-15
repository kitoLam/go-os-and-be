package judge

import (
	"fmt"
)

func TestWorse() {
	tests := []struct {
		a  Verdict
		b  Verdict
		ex Verdict
	}{
		{AC, WA, WA},
		{WA, AC, AC},
		{TLE, RE, RE},
		{RE, TLE, RE},
		{CE, AC, CE},
		{AC, CE, CE},
		{MLE, OLE, MLE},
		{OLE, MLE, MLE},
		{AC, AC, AC},
		{WA, TLE, WA},
	}

	for _, tt := range tests {
		actualValue := Worse(tt.a, tt.b)
		if actualValue != tt.ex {
			fmt.Printf("Lỗi nè: a = %s, b = %s, expected output = %s, mà ra actual output = %s\n", tt.a, tt.b, tt.ex, actualValue)
		}
		fmt.Printf("Ngon nè: a = %s, b = %s, expected output = %s, và ra actual output = %s\n", tt.a, tt.b, tt.ex, actualValue)
	}
}
