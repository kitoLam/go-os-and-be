package judge

type Verdict string

const (
	AC  Verdict = "AC"
	WA  Verdict = "WA"
	TLE Verdict = "TLE"
	MLE Verdict = "MLE" // Memory Limit Exceeded
	RE  Verdict = "RE"  // Runtime Error
	CE  Verdict = "CE"  // Compile Error
	OLE Verdict = "OLE" // Output Limit Exceeded
)

var verdictPriority = map[Verdict]int{
	AC:  1,
	WA:  2,
	OLE: 3,
	TLE: 4,
	MLE: 5,
	RE:  6,
	CE:  7,
}

func Worse(a Verdict, b Verdict) Verdict {
	priorityA := verdictPriority[a]
	priorityB := verdictPriority[b]

	if priorityA > priorityB {
		return a
	} else {
		return b
	}
}
